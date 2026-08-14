package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// onPath reports whether dir is already in the live PATH.
func onPath(dir string) bool {
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if p == dir {
			return true
		}
	}
	return false
}

// profileFile picks the shell profile to persist a PATH export into, matching
// the shell installer's choices.
func profileFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	shell := os.Getenv("SHELL")
	switch {
	case strings.HasSuffix(shell, "/zsh"):
		return filepath.Join(home, ".zshrc"), nil
	case strings.HasSuffix(shell, "/bash"):
		if runtime.GOOS == "darwin" {
			if p := filepath.Join(home, ".bash_profile"); fileExists(p) {
				return p, nil
			}
		}
		return filepath.Join(home, ".bashrc"), nil
	default:
		return filepath.Join(home, ".profile"), nil
	}
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// ensureOnPath appends a PATH export to the user's shell profile when the bin
// directory is not reachable, and explains what to do next either way.
func ensureOnPath(userBin, skill, repo string, out io.Writer) error {
	if onPath(userBin) {
		fmt.Fprintf(out, "\nReady to use: `%s --version` from any directory.\n", skill)
		return nil
	}

	profile, err := profileFile()
	if err != nil {
		return err
	}
	// One marker for the whole installer, not per skill: the export is
	// identical either way, and a per-skill marker appends a duplicate line
	// for every skill installed.
	marker := fmt.Sprintf("# Added by the skills installer (https://github.com/%s)", repo)

	existing, _ := os.ReadFile(profile)
	switch {
	case strings.Contains(string(existing), marker):
		fmt.Fprintf(out, "\n%s already has the PATH entry from a previous install.\n", profile)
		fmt.Fprintf(out, "Open a new terminal, or run: source %s\n", profile)
		return nil
	case strings.Contains(string(existing), userBin):
		fmt.Fprintf(out, "\nNote: %s is referenced in %s but not on your live PATH.\n", userBin, profile)
		fmt.Fprintf(out, "It may be commented out — uncomment it, or run:\n  export PATH=\"%s:$PATH\"\n", userBin)
		return nil
	}

	f, err := os.OpenFile(profile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(out, "\nNote: %s is NOT on your PATH. Add this to your shell profile:\n", userBin)
		fmt.Fprintf(out, "  export PATH=\"%s:$PATH\"\n", userBin)
		return nil
	}
	defer f.Close()
	if _, err := fmt.Fprintf(f, "\n%s\nexport PATH=\"$HOME/.local/bin:$PATH\"\n", marker); err != nil {
		return fmt.Errorf("write %s: %w", profile, err)
	}
	fmt.Fprintf(out, "\nAdded to %s:\n  export PATH=\"$HOME/.local/bin:$PATH\"\n", profile)
	fmt.Fprintf(out, "Open a new terminal — or run `source %s` — to use `%s` directly.\n", profile, skill)
	return nil
}

// registerWindowsPath adds the bin directory to the persistent user PATH.
// Delegated to PowerShell because that is the supported way to touch the user
// environment; symlinking would require elevation.
func registerWindowsPath(binDir string, out io.Writer) error {
	if onPath(binDir) {
		fmt.Fprintln(out, "  (Already on PATH for this shell.)")
		return nil
	}
	script := fmt.Sprintf(
		`$p=[Environment]::GetEnvironmentVariable('PATH','User'); `+
			`if ($p -notlike '*%[1]s*') { [Environment]::SetEnvironmentVariable('PATH', '%[1]s;' + $p, 'User') }`,
		binDir)
	if err := exec.Command("powershell", "-NoProfile", "-Command", script).Run(); err != nil {
		fmt.Fprintf(out, "  Could not register PATH automatically. Add manually:\n    %s\n", binDir)
		return nil
	}
	fmt.Fprintf(out, "  Registered on user PATH: %s\n", binDir)
	fmt.Fprintln(out, "  Open a new terminal for it to take effect.")
	return nil
}
