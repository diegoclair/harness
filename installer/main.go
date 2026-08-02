// Command skills installs the Claude skills published by this monorepo.
//
// It replaces the per-skill shell installers, which duplicated the same
// pipeline in POSIX sh and PowerShell. One binary covers every platform the
// release workflow builds for.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// version is stamped at build time via -ldflags.
var version = "dev"

const (
	exitOK       = 0
	exitErr      = 1
	exitInputErr = 2
)

const helpText = `skills — install Claude skills from github.com/diegoclair/skills

USAGE:
  skills list                       Show every installable skill
  skills install <name>...          Install one or more skills
  skills install --all              Install everything
  skills --version

FLAGS:
  --repo OWNER/REPO   Install from a fork (default: diegoclair/skills)
  --version TAG       Pin a release tag; only valid with a single skill

Names come from ` + "`skills list`" + `. Each skill is installed to
~/.claude/skills/<name>/ and linked onto your PATH.
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
		fmt.Fprintln(stdout, "skills", version)
		return exitOK
	case "list", "--list":
		printCatalog(stdout)
		return exitOK
	case "install":
		return runInstall(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		fmt.Fprint(stderr, helpText)
		return exitInputErr
	}
}

func printCatalog(out io.Writer) {
	width := 0
	for _, s := range catalog {
		if len(s.Name) > width {
			width = len(s.Name)
		}
	}
	for _, s := range catalog {
		fmt.Fprintf(out, "%-*s  %s\n", width, s.Name, s.Summary)
	}
}

func runInstall(args []string, stdout, stderr io.Writer) int {
	opts := installOptions{Repo: defaultRepo, Out: stdout}
	var names []string
	all := false

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
			all = true
		case arg == "--repo":
			if opts.Repo, ok = next(); !ok {
				return exitInputErr
			}
		case strings.HasPrefix(arg, "--repo="):
			opts.Repo = strings.TrimPrefix(arg, "--repo=")
		case arg == "--version":
			if opts.Version, ok = next(); !ok {
				return exitInputErr
			}
		case strings.HasPrefix(arg, "--version="):
			opts.Version = strings.TrimPrefix(arg, "--version=")
		case strings.HasPrefix(arg, "-"):
			fmt.Fprintf(stderr, "unknown flag: %s\n", arg)
			return exitInputErr
		default:
			names = append(names, arg)
		}
	}

	selected, code := selectSkills(names, all, stderr)
	if code != exitOK {
		return code
	}
	// A pinned tag belongs to one skill's release series.
	if opts.Version != "" && len(selected) > 1 {
		fmt.Fprintln(stderr, "--version applies to a single skill; install them one at a time")
		return exitInputErr
	}

	failed := 0
	for i, s := range selected {
		if i > 0 {
			fmt.Fprintln(stdout)
		}
		if err := install(s, opts); err != nil {
			fmt.Fprintf(stderr, "error installing %s: %v\n", s.Name, err)
			failed++
		}
	}
	if failed > 0 {
		fmt.Fprintf(stderr, "\n%d of %d skill(s) failed to install\n", failed, len(selected))
		return exitErr
	}
	return exitOK
}

// selectSkills resolves the requested names, rejecting unknown ones before any
// installation starts so a typo never leaves a half-applied selection.
func selectSkills(names []string, all bool, stderr io.Writer) ([]Skill, int) {
	if all {
		if len(names) > 0 {
			fmt.Fprintln(stderr, "--all takes no skill names")
			return nil, exitInputErr
		}
		return catalog, exitOK
	}
	if len(names) == 0 {
		fmt.Fprintf(stderr, "no skill given. Available: %s\n", strings.Join(skillNames(), ", "))
		return nil, exitInputErr
	}

	var selected []Skill
	seen := map[string]bool{}
	for _, n := range names {
		s, ok := findSkill(n)
		if !ok {
			fmt.Fprintf(stderr, "unknown skill %q. Available: %s\n", n, strings.Join(skillNames(), ", "))
			return nil, exitInputErr
		}
		if seen[s.Name] {
			continue
		}
		seen[s.Name] = true
		selected = append(selected, s)
	}
	return selected, exitOK
}
