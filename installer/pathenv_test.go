package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Installing several skills must not append the same export once per skill.
func TestEnsureOnPath_AppendsOnlyOnce(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	t.Setenv("PATH", "/usr/bin")
	userBin := filepath.Join(home, ".local", "bin")

	for _, skill := range []string{"confluence-docs", "jira-tickets", "social-carousel"} {
		var out bytes.Buffer
		if err := ensureOnPath(userBin, skill, defaultRepo, &out); err != nil {
			t.Fatalf("%s: %v", skill, err)
		}
	}

	content, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(content), "export PATH"); got != 1 {
		t.Errorf("wrote %d PATH exports, want 1:\n%s", got, content)
	}
}

func TestEnsureOnPath_AlreadyLive(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	userBin := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", userBin+":/usr/bin")

	var out bytes.Buffer
	if err := ensureOnPath(userBin, "jira-tickets", defaultRepo, &out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".zshrc")); err == nil {
		t.Error("profile must not be touched when the directory is already on PATH")
	}
	if !strings.Contains(out.String(), "Ready to use") {
		t.Errorf("output = %q", out.String())
	}
}

// A commented-out or conditional entry means the export exists but is not
// active; rewriting it would not help, so say so instead.
func TestEnsureOnPath_MentionedButNotActive(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	t.Setenv("PATH", "/usr/bin")
	userBin := filepath.Join(home, ".local", "bin")

	profile := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(profile, []byte("# export PATH=\""+userBin+":$PATH\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := ensureOnPath(userBin, "jira-tickets", defaultRepo, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "not on your live PATH") {
		t.Errorf("output = %q, want it to flag the inactive entry", out.String())
	}
	content, _ := os.ReadFile(profile)
	if strings.Count(string(content), "export PATH") != 1 {
		t.Errorf("profile should be left alone:\n%s", content)
	}
}

func TestProfileFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	tests := []struct {
		shell string
		want  string
	}{
		{"/bin/zsh", ".zshrc"},
		{"/usr/bin/bash", ".bashrc"},
		{"/bin/fish", ".profile"},
		{"", ".profile"},
	}
	for _, tc := range tests {
		t.Setenv("SHELL", tc.shell)
		got, err := profileFile()
		if err != nil {
			t.Fatal(err)
		}
		if filepath.Base(got) != tc.want {
			t.Errorf("SHELL=%q → %q, want %q", tc.shell, filepath.Base(got), tc.want)
		}
	}
}

func TestOnPath(t *testing.T) {
	t.Setenv("PATH", strings.Join([]string{"/usr/bin", "/home/u/.local/bin"}, string(os.PathListSeparator)))
	if !onPath("/home/u/.local/bin") {
		t.Error("want the directory to be found on PATH")
	}
	// Substring matches must not count: /home/u/.local/bin2 is a different dir.
	if onPath("/home/u/.local") {
		t.Error("a parent directory must not match")
	}
}
