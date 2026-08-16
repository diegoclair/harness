// quality-gate runs the checks a reviewer would otherwise run by hand: is this
// comment explaining behavior instead of purpose, does this block already exist
// somewhere in the repo, is a domain rule leaking into a handler or a query.
//
// It is meant to run at the end of a delivery, next to the project's build, and
// again on the PR from the same binary.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var version = "v0.1.0"

const (
	exitOK       = 0
	exitFindings = 1
	exitUsage    = 2
)

const helpText = `quality-gate — the review pass a linter can actually do.

USAGE:
  quality-gate check     [--all] [--since REF] [--format text|json] [--quiet]
  quality-gate baseline  [--prune] [--dir PATH]
  quality-gate explain   RULE-ID
  quality-gate rules
  quality-gate init      [--ruleset go|web|auto]
  quality-gate --version | --help

CHECK:
  Scans the delivery — everything different from the base branch plus
  everything still uncommitted — and reports findings. Duplication is always
  measured against the whole repo, so a block that already exists elsewhere is
  caught even when the delivery only touched one file.

  --all      scan every file instead of the delivery
  --since    base ref to diff against (default: origin/main, then main)
  --format   text (default) or json
  --quiet    print errors only

  Exit 0 clean, 1 new errors, 2 configuration or usage failure. Warnings never
  change the exit code.

BASELINE:
  Records today's debt in .quality-gate-baseline.json so the gate blocks new
  violations without demanding the repo be clean first. Entries are matched by
  content, so editing the code loses the pass. Commit the file.

  --prune drops the entries whose code is gone and freezes nothing new. That is
  the safe move after upgrading the binary: new rules retire old findings, and a
  full regenerate would silently re-freeze debt introduced since.

EXPLAIN / RULES:
  rules lists the catalog; explain gives one rule's reasoning and how to fix it.

INIT:
  Scaffolds .quality-gate.yml for the stack detected in the current directory.
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, helpText)
		return exitUsage
	}
	switch args[0] {
	case "--help", "-h", "help":
		fmt.Fprint(stdout, helpText)
		return exitOK
	case "--version", "-v":
		fmt.Fprintln(stdout, "quality-gate "+version)
		return exitOK
	case "check":
		return cmdCheck(args[1:], stdout, stderr)
	case "baseline":
		return cmdBaseline(args[1:], stdout, stderr)
	case "explain":
		return cmdExplain(args[1:], stdout, stderr)
	case "rules":
		reportRules(stdout)
		return exitOK
	case "init":
		return cmdInit(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n%s", args[0], helpText)
		return exitUsage
	}
}

func cmdCheck(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	all := fs.Bool("all", false, "scan every file instead of the delivery")
	since := fs.String("since", "", "base ref to diff against")
	format := fs.String("format", "text", "text or json")
	quiet := fs.Bool("quiet", false, "print errors only")
	dir := fs.String("dir", ".", "directory to resolve the config from")
	skipPrereq := fs.Bool("skip-prerequisites", false, "do not run the project's own gates first")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	cfg, err := loadConfig(*dir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitUsage
	}
	if !*skipPrereq {
		if err := cfg.runPrerequisites(); err != nil {
			fmt.Fprintln(stderr, err)
			fmt.Fprintln(stderr, "\nthe project's own gate is red — quality-gate reports on a tree its formatter already accepts")
			return exitUsage
		}
	}

	res, err := runCheck(cfg, checkOptions{all: *all, since: *since})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitUsage
	}
	switch *format {
	case "json":
		if err := reportJSON(stdout, res); err != nil {
			fmt.Fprintln(stderr, err)
			return exitUsage
		}
	case "text":
		reportText(stdout, res, *quiet)
	default:
		fmt.Fprintf(stderr, "unknown format %q\n", *format)
		return exitUsage
	}
	if res.Errors > 0 {
		return exitFindings
	}
	return exitOK
}

func cmdBaseline(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("baseline", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "directory to resolve the config from")
	prune := fs.Bool("prune", false, "drop entries whose code is gone, add nothing")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	cfg, err := loadConfig(*dir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitUsage
	}
	if *prune {
		return cmdPrune(cfg, stdout, stderr)
	}
	// The baseline records what exists, so it is taken without one applied.
	if err := os.Remove(cfg.Root + "/" + baselineName); err != nil && !os.IsNotExist(err) {
		fmt.Fprintln(stderr, err)
		return exitUsage
	}
	res, err := runCheck(cfg, checkOptions{all: true})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitUsage
	}
	n, err := writeBaseline(cfg.Root, res.Findings)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitUsage
	}
	fmt.Fprintf(stdout, "wrote %s with %d entries (%d errors, %d warnings frozen)\n",
		baselineName, n, res.Errors, res.Warnings)
	fmt.Fprintln(stdout, "commit it: it is the honest record of what the repo owes.")
	return exitOK
}

// cmdPrune keeps the baseline shrinking across an engine upgrade: entries the
// rules no longer produce are dropped, and nothing new is ever frozen.
func cmdPrune(cfg *Config, stdout, stderr io.Writer) int {
	before, err := loadBaseline(cfg.Root)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitUsage
	}
	res, err := runCheck(cfg, checkOptions{all: true})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitUsage
	}
	orphan := map[string]bool{}
	for _, e := range res.StaleEntries {
		orphan[e.Signature] = true
	}
	kept := Baseline{Version: before.Version}
	for _, e := range before.Entries {
		if !orphan[e.Signature] {
			kept.Entries = append(kept.Entries, e)
		}
	}
	dropped := len(before.Entries) - len(kept.Entries)
	raw, err := encodeBaseline(kept)
	if err == nil {
		err = os.WriteFile(filepath.Join(cfg.Root, baselineName), raw, 0o644)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitUsage
	}
	fmt.Fprintf(stdout, "pruned %d stale entr(ies); %d remain. Nothing new was frozen.\n", dropped, len(kept.Entries))
	return exitOK
}

func cmdExplain(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: quality-gate explain RULE-ID")
		return exitUsage
	}
	text, err := explain(args[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitUsage
	}
	fmt.Fprint(stdout, text)
	return exitOK
}

func cmdInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	ruleset := fs.String("ruleset", "", "go, web or auto (default: detected)")
	dir := fs.String("dir", ".", "where to write the config")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	path := *dir + "/" + configName
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(stderr, "%s already exists — edit it instead\n", path)
		return exitUsage
	}
	detected := *ruleset
	if detected == "" {
		detected = detectRuleset(*dir)
	}
	if err := os.WriteFile(path, []byte(starterConfig(*dir, detected)), 0o644); err != nil {
		fmt.Fprintln(stderr, err)
		return exitUsage
	}
	fmt.Fprintf(stdout, "wrote %s (ruleset: %s)\n", path, detected)
	fmt.Fprintln(stdout, "declare your layers, then run `quality-gate baseline`.")
	return exitOK
}
