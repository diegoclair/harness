// Command harness installs the Claude Code skills and agents published by
// this repo.
//
// Two payload kinds share one selection mechanism: markdown artifacts are
// copied straight out of the repo tree (no CI, no release), while skills that
// ship a Go CLI come from a per-platform release archive.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// version is stamped at build time via -ldflags.
var version = "dev"

const (
	exitOK       = 0
	exitErr      = 1
	exitInputErr = 2
)

const helpText = `harness — install Claude Code skills and agents from github.com/diegoclair/harness

USAGE:
  harness list [--skills|--agents]     Show every installable artifact
  harness install <name>...            Install skills and/or agents
  harness install --all                Install everything
  harness install --all-skills         Install every skill
  harness install --all-agents         Install every agent
  harness validate [PATH]              Check the artifacts in a repo tree
  harness --version

FLAGS:
  --repo OWNER/REPO   Install from a fork (default: diegoclair/harness)
  --ref REF           Branch, tag or SHA of the repo tree (default: main)
  --from PATH         Install from a local clone instead of downloading
  --version TAG       Pin a release tag; only for a skill that ships a binary

Skills are installed to ~/.claude/skills/<name>/ and agents to
~/.claude/agents/<name>.md. Dependencies are pulled in automatically.
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, helpText)
		return exitOK
	}

	switch args[0] {
	case "-h", "--help", "help":
		fmt.Fprint(stdout, helpText)
		return exitOK
	case "-v", "--version":
		fmt.Fprintln(stdout, "harness", version)
		return exitOK
	case "list", "--list":
		return runList(args[1:], stdout, stderr)
	case "validate":
		return runValidate(args[1:], stdout, stderr)
	case "install":
		return runInstall(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		fmt.Fprint(stderr, helpText)
		return exitInputErr
	}
}

func runList(args []string, stdout, stderr io.Writer) int {
	var onlyKind *Kind
	for _, a := range args {
		switch a {
		case "--skills":
			k := KindSkill
			onlyKind = &k
		case "--agents":
			k := KindAgent
			onlyKind = &k
		default:
			fmt.Fprintf(stderr, "unknown flag: %s\n", a)
			return exitInputErr
		}
	}

	width := 0
	for _, a := range catalog {
		if len(a.Name) > width {
			width = len(a.Name)
		}
	}
	for _, kind := range []Kind{KindSkill, KindAgent} {
		if onlyKind != nil && *onlyKind != kind {
			continue
		}
		fmt.Fprintf(stdout, "%ss:\n", strings.ToUpper(kind.String()[:1])+kind.String()[1:])
		for _, a := range catalog {
			if a.Kind != kind {
				continue
			}
			fmt.Fprintf(stdout, "  %-*s  %s\n", width, a.Name, a.Summary)
		}
	}
	return exitOK
}

// installFlags is what the command line asked for, before any policy applies.
type installFlags struct {
	repo    string
	ref     string
	from    string
	version string

	refSet     bool
	fromSet    bool
	versionSet bool

	all       bool
	allSkills bool
	allAgents bool
	names     []string
}

func runInstall(args []string, stdout, stderr io.Writer) int {
	f := installFlags{repo: defaultRepo, ref: "main"}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		next := func() (string, bool) {
			if i+1 >= len(args) {
				fmt.Fprintf(stderr, "flag %s requires a value\n", arg)
				return "", false
			}
			i++
			return args[i], true
		}
		var ok bool
		switch {
		case arg == "--all":
			f.all = true
		case arg == "--all-skills":
			f.allSkills = true
		case arg == "--all-agents":
			f.allAgents = true
		case arg == "--repo":
			if f.repo, ok = next(); !ok {
				return exitInputErr
			}
		case strings.HasPrefix(arg, "--repo="):
			f.repo = strings.TrimPrefix(arg, "--repo=")
		case arg == "--ref":
			if f.ref, ok = next(); !ok {
				return exitInputErr
			}
			f.refSet = true
		case strings.HasPrefix(arg, "--ref="):
			f.ref, f.refSet = strings.TrimPrefix(arg, "--ref="), true
		case arg == "--from":
			if f.from, ok = next(); !ok {
				return exitInputErr
			}
			f.fromSet = true
		case strings.HasPrefix(arg, "--from="):
			f.from, f.fromSet = strings.TrimPrefix(arg, "--from="), true
		case arg == "--version":
			if f.version, ok = next(); !ok {
				return exitInputErr
			}
			f.versionSet = true
		case strings.HasPrefix(arg, "--version="):
			f.version, f.versionSet = strings.TrimPrefix(arg, "--version="), true
		case strings.HasPrefix(arg, "-"):
			fmt.Fprintf(stderr, "unknown flag: %s\n", arg)
			return exitInputErr
		default:
			f.names = append(f.names, arg)
		}
	}

	// An empty value would otherwise read as "flag not given" and silently
	// fall back to the default (--from= would install from the network).
	for _, e := range []struct {
		name  string
		set   bool
		value string
	}{{"--ref", f.refSet, f.ref}, {"--from", f.fromSet, f.from}, {"--version", f.versionSet, f.version}, {"--repo", true, f.repo}} {
		if e.set && e.value == "" {
			fmt.Fprintf(stderr, "flag %s requires a value\n", e.name)
			return exitInputErr
		}
	}

	selected, code := selectArtifacts(f, stdout, stderr)
	if code != exitOK {
		return code
	}
	return installAll(selected, f, stdout, stderr)
}

// selectArtifacts resolves the request into the artifacts to install,
// rejecting every input error before any installation starts so a typo never
// leaves a half-applied selection.
func selectArtifacts(f installFlags, stdout, stderr io.Writer) ([]Artifact, int) {
	wildcard := f.all || f.allSkills || f.allAgents
	if wildcard && len(f.names) > 0 {
		fmt.Fprintln(stderr, "--all/--all-skills/--all-agents take no artifact names")
		return nil, exitInputErr
	}
	if !wildcard && len(f.names) == 0 {
		fmt.Fprintf(stderr, "no artifact given. Available: %s\n", strings.Join(artifactNames(), ", "))
		return nil, exitInputErr
	}

	var selected []Artifact
	if wildcard {
		for _, a := range catalog {
			if f.all ||
				(f.allSkills && a.Kind == KindSkill) ||
				(f.allAgents && a.Kind == KindAgent) {
				selected = append(selected, a)
			}
		}
	} else {
		seen := map[string]bool{}
		for _, n := range f.names {
			a, ok := findArtifact(n)
			if !ok {
				fmt.Fprintf(stderr, "unknown artifact %q. Available: %s\n", n, strings.Join(artifactNames(), ", "))
				return nil, exitInputErr
			}
			if seen[a.Name] {
				continue
			}
			seen[a.Name] = true
			selected = append(selected, a)
		}
	}

	if len(selected) == 0 {
		fmt.Fprintln(stderr, "nothing to install for that selection")
		return nil, exitInputErr
	}

	selected, added, err := resolveRequires(selected)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return nil, exitErr
	}
	if len(added) > 0 {
		fmt.Fprintf(stdout, "Also installing required: %s\n\n", strings.Join(added, ", "))
	}

	return selected, exitOK
}

// rejectFlagMismatch refuses a flag that cannot apply to a named artifact.
// A version pin only means something for a skill that ships a binary, and an
// agent is always plain files.
func rejectFlagMismatch(selected []Artifact, root string, f installFlags, stderr io.Writer) int {
	if !f.versionSet {
		return exitOK
	}
	var pinnable []Artifact
	for _, a := range selected {
		if a.Kind == KindSkill && shipsCLI(filepath.Join(root, "skills", a.Name)) {
			pinnable = append(pinnable, a)
		}
	}
	if len(pinnable) == 0 {
		fmt.Fprintln(stderr, "--version pins a release tag, and none of the selected artifacts "+
			"ship a binary; use --ref to pick a different version of their files")
		return exitInputErr
	}
	if len(pinnable) > 1 {
		fmt.Fprintln(stderr, "--version applies to a single artifact; install them one at a time")
		return exitInputErr
	}
	return exitOK
}

func installAll(selected []Artifact, f installFlags, stdout, stderr io.Writer) int {
	tmp, err := os.MkdirTemp("", "harness-install-")
	if err != nil {
		fmt.Fprintf(stderr, "create temp dir: %v\n", err)
		return exitErr
	}
	defer os.RemoveAll(tmp)

	// One tree for the whole run: installing five markdown artifacts is one
	// download, not five.
	var tree treeProvider
	if f.from != "" {
		abs, err := filepath.Abs(f.from)
		if err != nil {
			fmt.Fprintf(stderr, "resolve --from path: %v\n", err)
			return exitInputErr
		}
		tree = localTree{path: abs}
	} else {
		tree = &remoteTree{repo: f.repo, ref: f.ref, tmp: tmp}
	}

	root, err := tree.root()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitErr
	}
	// Flag applicability depends on what an artifact actually ships, which is
	// only knowable once the tree is in hand. Nothing has been written yet.
	if code := rejectFlagMismatch(selected, root, f, stderr); code != exitOK {
		return code
	}

	opts := installOptions{Repo: f.repo, Version: f.version, Ref: tree.describe(), Out: stdout}

	failed := 0
	for i, a := range selected {
		if i > 0 {
			fmt.Fprintln(stdout)
		}
		var err error
		if a.Kind == KindAgent {
			err = installAgent(a, root, opts.Ref, stdout)
		} else {
			err = installSkill(a, root, opts)
		}
		if err != nil {
			fmt.Fprintf(stderr, "error installing %s: %v\n", a.Name, err)
			failed++
		}
	}
	if failed > 0 {
		fmt.Fprintf(stderr, "\n%d of %d artifact(s) failed to install\n", failed, len(selected))
		return exitErr
	}
	return exitOK
}
