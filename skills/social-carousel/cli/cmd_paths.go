// cmd_paths implements `social-carousel paths` — a single command that
// prints every filesystem location the skill cares about, in a format
// agents can grep / parse trivially.
//
// Why this command exists (and why it's the ONLY command added for the
// taste / config / theme system instead of CRUD wrappers):
//
//	The CLI's consumer is the AI agent, which already has Read/Write
//	tools. Wrapping file IO in `taste show`, `taste add`, `config init`
//	etc. is surface area that pays for nothing — every wrapper is just
//	a `cat` or `Write append`. What the agent CANNOT compute reliably,
//	however, is the cross-OS user-config dir (varies between Linux,
//	macOS, Windows) and the project-local walk-up (.git/ boundary + 10
//	level cap). So we expose just the path resolver and let the agent
//	read/write directly.
//
// Output format: stable `key=value` lines, one per path. Empty value
// means "no project-local file found near CWD" (the agent reads the
// global one only). Trailing newline. No JSON for now — the agent can
// `grep` lines without parsing.

package main

import (
	"fmt"
	"io"
	"os"
)

// runPaths implements `social-carousel paths [--cwd DIR]`.
//
// Without flags: walks up from the current working directory.
// With `--cwd DIR`: walks up from DIR instead (useful when the agent
// has the carousel YAML path and wants paths resolved relative to it,
// even when the CLI was invoked from elsewhere).
func runPaths(args []string, stdout, stderr io.Writer) (int, error) {
	startDir := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--cwd":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "paths: --cwd requires a directory argument")
				return exitInputErr, errInvalidUsage
			}
			startDir = args[i+1]
			i++
		case "-h", "--help":
			fmt.Fprintln(stdout, "usage: social-carousel paths [--cwd DIR]")
			fmt.Fprintln(stdout, "")
			fmt.Fprintln(stdout, "Prints filesystem locations relevant to this skill so the agent")
			fmt.Fprintln(stdout, "can Read/Write them directly. Output is stable key=value lines.")
			fmt.Fprintln(stdout, "Empty values mean 'no project-local file found near the start dir'.")
			return exitOK, nil
		default:
			fmt.Fprintln(stderr, "paths: unknown flag:", args[i])
			return exitInputErr, errInvalidUsage
		}
	}

	// Resolve startDir defaulting to CWD so the project walk-up has
	// something to work with.
	if startDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return exitUnknownErr, fmt.Errorf("get cwd: %w", err)
		}
		startDir = cwd
	}

	// Global taste — fixed location inside the skill install dir.
	// Stable across OSes because ~/.claude/skills/ is the install
	// convention regardless of platform.
	globalTaste, err := GlobalTastePath()
	if err != nil {
		return exitUnknownErr, fmt.Errorf("resolve global taste path: %w", err)
	}

	// Global config dir / themes dir — OS-dependent via os.UserConfigDir().
	globalConfigDir, err := GlobalConfigDir()
	if err != nil {
		return exitUnknownErr, fmt.Errorf("resolve global config dir: %w", err)
	}
	themesDir, err := GlobalThemesDir()
	if err != nil {
		return exitUnknownErr, fmt.Errorf("resolve themes dir: %w", err)
	}

	// Project walk-up results — empty string when not found.
	projectConfig, _ := FindProjectConfig(startDir)
	projectTaste, _ := FindProjectTaste(startDir)

	fmt.Fprintln(stdout, "# Paths resolved relative to:")
	fmt.Fprintln(stdout, "cwd="+startDir)
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "# Taste memory (markdown with YAML frontmatter; agent reads BEFORE generating, appends rules when user gives corrective imperatives)")
	fmt.Fprintln(stdout, "global_taste="+globalTaste)
	fmt.Fprintln(stdout, "project_taste="+projectTaste)
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "# Project config (carousel.config.yml — theme/handle/platform defaults)")
	fmt.Fprintln(stdout, "project_config="+projectConfig)
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "# User themes (custom themes saved via `theme create`)")
	fmt.Fprintln(stdout, "global_config_dir="+globalConfigDir)
	fmt.Fprintln(stdout, "themes_dir="+themesDir)

	return exitOK, nil
}
