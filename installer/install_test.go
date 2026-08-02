package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// writeZip builds an archive from name→content pairs.
func writeZip(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	w := zip.NewWriter(f)
	for name, content := range entries {
		e, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := e.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestUnzip(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "a.zip")
	writeZip(t, archive, map[string]string{
		"bin/confluence-docs": "binary",
		"SKILL.md":            "# skill",
		"reference/config.md": "# config",
	})

	dest := filepath.Join(dir, "out")
	if err := unzip(archive, dest); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"bin/confluence-docs", "SKILL.md", "reference/config.md"} {
		if _, err := os.Stat(filepath.Join(dest, filepath.FromSlash(rel))); err != nil {
			t.Errorf("%s not extracted: %v", rel, err)
		}
	}
}

// A crafted archive must not be able to write outside the destination.
func TestUnzip_RejectsPathTraversal(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "evil.zip")
	writeZip(t, archive, map[string]string{"../escaped.txt": "pwned"})

	err := unzip(archive, filepath.Join(dir, "out"))
	if err == nil {
		t.Fatal("want an error for an entry escaping the destination")
	}
	if !strings.Contains(err.Error(), "escapes destination") {
		t.Errorf("error = %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "escaped.txt")); statErr == nil {
		t.Error("the traversal entry was written to disk")
	}
}

// Replacing a running binary is the `<skill> update` path; a plain copy would
// fail with ETXTBSY, so the install must go through a rename.
func TestInstallBinary_ReplacesExisting(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "new")
	dst := filepath.Join(dir, "installed")
	if err := os.WriteFile(src, []byte("v2"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("v1"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := installBinary(src, dst); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "v2" {
		t.Errorf("content = %q, want the new binary", got)
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Errorf("installed binary is not executable: %v", info.Mode())
	}
	// The staging file must not survive.
	if _, err := os.Stat(filepath.Join(dir, ".installed.new")); err == nil {
		t.Error("temp file left behind")
	}
}

func TestInstallPayload(t *testing.T) {
	dir := t.TempDir()
	extract := filepath.Join(dir, "extract")
	target := filepath.Join(dir, "skill")
	binDir := filepath.Join(target, "bin")
	for _, d := range []string{filepath.Join(extract, "bin"), filepath.Join(extract, "reference"), binDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(p, c string) {
		if err := os.WriteFile(p, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(extract, "bin", "jira-tickets"), "bin")
	write(filepath.Join(extract, "SKILL.md"), "# skill")
	write(filepath.Join(extract, "reference", "a.md"), "a")
	write(filepath.Join(extract, "reference", "b.md"), "b")

	// A file from a previous install that no longer ships must not survive.
	if err := os.MkdirAll(filepath.Join(target, "reference"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(filepath.Join(target, "reference", "stale.md"), "old")

	count, err := installPayload(Skill{Name: "jira-tickets"}, extract, target, binDir)
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 { // SKILL.md + two reference files
		t.Errorf("installed %d files, want 3", count)
	}
	if _, err := os.Stat(filepath.Join(target, "reference", "stale.md")); err == nil {
		t.Error("stale reference file survived the install")
	}
}

func TestInstallPayload_MissingBinary(t *testing.T) {
	dir := t.TempDir()
	extract := filepath.Join(dir, "extract")
	binDir := filepath.Join(dir, "skill", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(extract, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := installPayload(Skill{Name: "jira-tickets"}, extract, filepath.Join(dir, "skill"), binDir); err == nil {
		t.Fatal("want an error when the archive carries no binary")
	}
}

func TestResolveVersion_PinWins(t *testing.T) {
	s := Skill{Name: "jira-tickets", TagPrefix: "jira-v", VersionEnv: "JIRA_TICKETS_VERSION"}

	got, err := resolveVersion(s, installOptions{Version: "jira-v9.9.9"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "jira-v9.9.9" {
		t.Errorf("version = %q, want the explicit pin", got)
	}

	t.Setenv("JIRA_TICKETS_VERSION", "jira-v1.2.3")
	got, err = resolveVersion(s, installOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got != "jira-v1.2.3" {
		t.Errorf("version = %q, want the legacy env var", got)
	}

	// SKILL_VERSION is what the old shell installer honoured.
	t.Setenv("SKILL_VERSION", "jira-v2.0.0")
	got, err = resolveVersion(s, installOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got != "jira-v2.0.0" {
		t.Errorf("version = %q, want SKILL_VERSION to win over the per-skill var", got)
	}
}

// The installer must relay the skill's own diagnosis: collapsing every failure
// into "not configured" is what made a valid grant look like missing
// credentials.
func TestReportUnconfigured(t *testing.T) {
	skill := Skill{Name: "jira-tickets"}

	stderrErr := func(msg string) error {
		err := exec.Command("sh", "-c", "echo '"+msg+"' >&2; exit 1").Run()
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			ee.Stderr = []byte(msg + "\n")
			return ee
		}
		t.Fatalf("could not build an ExitError")
		return nil
	}

	t.Run("relays the reason", func(t *testing.T) {
		var out bytes.Buffer
		reportUnconfigured(skill, "", stderrErr("this Atlassian site has no Jira"), &out)
		if !strings.Contains(out.String(), "no Jira") {
			t.Errorf("output = %q, want the skill's own message", out.String())
		}
		if strings.Contains(out.String(), "Not yet configured") {
			t.Errorf("generic message should not replace a real diagnosis: %q", out.String())
		}
	})

	t.Run("does not repeat the setup hint", func(t *testing.T) {
		var out bytes.Buffer
		reportUnconfigured(skill, "", stderrErr("no credentials configured — run `jira-tickets setup`"), &out)
		if strings.Count(out.String(), "jira-tickets setup") != 1 {
			t.Errorf("hint printed more than once: %q", out.String())
		}
	})

	t.Run("falls back when the skill says nothing", func(t *testing.T) {
		var out bytes.Buffer
		reportUnconfigured(skill, "", errors.New("boom"), &out)
		if !strings.Contains(out.String(), "Not yet configured") {
			t.Errorf("output = %q, want the generic fallback", out.String())
		}
	})
}
