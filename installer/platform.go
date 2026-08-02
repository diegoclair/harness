package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// platform is the release-asset suffix for the running machine, matching the
// names the release workflow produces (darwin/linux/windows × amd64/arm64).
func platform() (string, error) {
	var os_ string
	switch runtime.GOOS {
	case "linux", "darwin", "windows":
		os_ = runtime.GOOS
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	switch runtime.GOARCH {
	case "amd64", "arm64":
	default:
		return "", fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
	}

	// The workflow only cross-compiles windows/amd64.
	if runtime.GOOS == "windows" && runtime.GOARCH != "amd64" {
		return "", fmt.Errorf("no windows/%s build is published", runtime.GOARCH)
	}
	return os_ + "-" + runtime.GOARCH, nil
}

// binaryName is the on-disk executable name for a skill.
func binaryName(skill string) string {
	if runtime.GOOS == "windows" {
		return skill + ".exe"
	}
	return skill
}

// claudeHome is where Claude looks for skills. CLAUDE_HOME overrides it, same
// as the shell installer.
func claudeHome() (string, error) {
	if v := os.Getenv("CLAUDE_HOME"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".claude"), nil
}

// skillDir is the install root for one skill.
func skillDir(skill string) (string, error) {
	home, err := claudeHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "skills", skill), nil
}

// userBinDir is the directory symlinked onto PATH on Unix.
func userBinDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".local", "bin"), nil
}
