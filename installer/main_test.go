package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runCLI(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestListShowsBothKinds(t *testing.T) {
	code, stdout, _ := runCLI(t, "list")
	if code != exitOK {
		t.Fatalf("exit = %d", code)
	}
	for _, want := range []string{"Skills:", "Agents:", "dev-loop", "implementation-plan", "unbiased-reviewer"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("list output is missing %q\n%s", want, stdout)
		}
	}
}

func TestListCanFilterByKind(t *testing.T) {
	code, stdout, _ := runCLI(t, "list", "--agents")
	if code != exitOK {
		t.Fatalf("exit = %d", code)
	}
	if strings.Contains(stdout, "Skills:") {
		t.Errorf("--agents should not list skills\n%s", stdout)
	}
	if !strings.Contains(stdout, "unbiased-reviewer") {
		t.Errorf("--agents should list the agent\n%s", stdout)
	}
}

// Every rejection below must happen before anything is written, so a typo
// never leaves a half-applied selection.
func TestInputErrorsAreRejectedWithASpecificMessage(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		wantHint string
	}{
		{"unknown artifact", []string{"install", "dev-loop", "bogus"}, "unknown artifact"},
		{"no name", []string{"install"}, "no artifact given"},
		{"names with --all", []string{"install", "--all", "dev-loop"}, "take no artifact names"},
		{"unknown flag", []string{"install", "--nope"}, "unknown flag"},
		{"unknown command", []string{"frobnicate"}, "unknown command"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sandboxHome(t)
			code, _, stderr := runCLI(t, tc.args...)
			if code != exitInputErr {
				t.Errorf("exit = %d, want %d", code, exitInputErr)
			}
			if !strings.Contains(stderr, tc.wantHint) {
				t.Errorf("stderr = %q, want it to mention %q", stderr, tc.wantHint)
			}
		})
	}
}

// The two skills are useless without the agent they dispatch, so asking for
// one must land the other.
func TestInstallingASkillPullsInItsRequiredAgent(t *testing.T) {
	home := sandboxHome(t)
	tree := fixtureTree(t)

	code, stdout, stderr := runCLI(t, "install", "--from", tree, "dev-loop")
	if code != exitOK {
		t.Fatalf("exit = %d\nstdout:%s\nstderr:%s", code, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "agents", "unbiased-reviewer.md")); err != nil {
		t.Fatalf("required agent was not installed: %v", err)
	}
	if !strings.Contains(stdout, "Also installing required") {
		t.Errorf("the pulled-in dependency should be reported\n%s", stdout)
	}
}

func TestAllAgentsInstallsOnlyAgents(t *testing.T) {
	home := sandboxHome(t)
	tree := fixtureTree(t)

	code, _, stderr := runCLI(t, "install", "--from", tree, "--all-agents")
	if code != exitOK {
		t.Fatalf("exit = %d\nstderr:%s", code, stderr)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "agents", "unbiased-reviewer.md")); err != nil {
		t.Errorf("agent missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills")); err == nil {
		t.Error("--all-agents must not create skills/")
	}
}

func TestHelpAndVersion(t *testing.T) {
	for _, args := range [][]string{{}, {"--help"}, {"--version"}} {
		code, stdout, _ := runCLI(t, args...)
		if code != exitOK || stdout == "" {
			t.Errorf("%v: exit = %d, stdout empty = %v", args, code, stdout == "")
		}
	}
}

func TestValidateAcceptsTheRepoAndRejectsABrokenFrontmatter(t *testing.T) {
	repo := ".."

	code, stdout, stderr := runCLI(t, "validate", repo)
	if code != exitOK {
		t.Fatalf("validate exit = %d\nstdout:%s\nstderr:%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "All artifacts valid") {
		t.Errorf("stdout = %q", stdout)
	}

	// The defect this guards against: an unquoted description containing ": "
	// parses as a nested mapping.
	broken := t.TempDir()
	mustWrite(t, filepath.Join(broken, "skills", "x-skill", "SKILL.md"),
		"---\nname: x-skill\ndescription: Do a thing: and another\n---\nbody\n")

	code, _, stderr = runCLI(t, "validate", broken)
	if code == exitOK {
		t.Fatal("want a non-zero exit for invalid YAML frontmatter")
	}
	if !strings.Contains(stderr, "SKILL.md") || !strings.Contains(stderr, "YAML") {
		t.Errorf("stderr should name the file and the problem, got %q", stderr)
	}
}

func TestValidateCatchesANameLocationMismatch(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "agents", "reviewer.md"), "---\nname: something-else\ndescription: ok\n---\n")

	code, _, stderr := runCLI(t, "validate", dir)
	if code == exitOK {
		t.Fatal("want a non-zero exit when name and location disagree")
	}
	if !strings.Contains(stderr, "does not match its location") {
		t.Errorf("stderr = %q", stderr)
	}
}

// Without the bounds guard in the flag parser this panics instead of
// reporting a usage error.
func TestFlagWithoutAValueIsAUsageError(t *testing.T) {
	for _, args := range [][]string{
		{"install", "--ref"}, {"install", "--from"}, {"install", "--version"}, {"install", "--repo"},
		{"install", "--ref=", "dev-loop"}, {"install", "--from=", "dev-loop"},
		{"install", "--repo=", "dev-loop"}, {"install", "--version=", "dev-loop"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			sandboxHome(t)
			code, _, stderr := runCLI(t, args...)
			if code != exitInputErr {
				t.Errorf("exit = %d, want %d", code, exitInputErr)
			}
			if !strings.Contains(stderr, "requires a value") {
				t.Errorf("stderr = %q", stderr)
			}
		})
	}
}

// Asking for skills and agents at once must select both, not their empty
// intersection.
func TestWildcardsCombineAsAUnion(t *testing.T) {
	home := sandboxHome(t)
	tree := fixtureTree(t)
	serveReleases(t)

	code, _, stderr := runCLI(t, "install", "--from", tree, "--all-skills", "--all-agents")
	if code != exitOK {
		t.Fatalf("exit = %d\nstderr:%s", code, stderr)
	}
	for _, p := range []string{
		filepath.Join(home, ".claude", "skills", "dev-loop", "SKILL.md"),
		filepath.Join(home, ".claude", "agents", "unbiased-reviewer.md"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("missing %s: %v", p, err)
		}
	}

	// The same batch carries skills that drive a binary: one pipeline has to
	// serve both without the caller distinguishing them.
	for _, name := range []string{"confluence-docs", "jira-tickets", "social-carousel"} {
		if _, err := os.Stat(filepath.Join(home, ".claude", "skills", name, "bin", name)); err != nil {
			t.Errorf("%s should have been installed with its binary: %v", name, err)
		}
		if _, err := os.Lstat(filepath.Join(home, ".local", "bin", name)); err != nil {
			t.Errorf("%s should be on PATH: %v", name, err)
		}
	}
	// ...while the plain skills get neither.
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "dev-loop", "bin")); err == nil {
		t.Error("dev-loop ships no cli/ and must not get a bin/")
	}
}

func TestValidateRules(t *testing.T) {
	cases := []struct {
		name, frontmatter, wantHint string
	}{
		{"missing description", "name: a-skill\n", "missing 'description'"},
		{"missing name", "description: something\n", "missing 'name'"},
		{"uppercase name", "name: A-Skill\ndescription: ok\n", "lowercase with hyphens"},
		{"underscore in name", "name: a_skill\ndescription: ok\n", "lowercase with hyphens"},
		{"name too long", "name: " + strings.Repeat("a", 65) + "\ndescription: ok\n", "max 64"},
		{"description too long", "name: a-skill\ndescription: " + strings.Repeat("x", 1025) + "\n", "max 1024"},
		{"angle brackets", "name: a-skill\ndescription: use <tag> here\n", "< or >"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			mustWrite(t, filepath.Join(dir, "skills", "a-skill", "SKILL.md"), "---\n"+tc.frontmatter+"---\nbody\n")
			code, _, stderr := runCLI(t, "validate", dir)
			if code == exitOK {
				t.Fatalf("want a non-zero exit; stderr = %q", stderr)
			}
			if !strings.Contains(stderr, tc.wantHint) {
				t.Errorf("stderr = %q, want it to mention %q", stderr, tc.wantHint)
			}
		})
	}
}

// Multi-byte punctuation is common in descriptions; the limit counts
// characters, so a byte-based check would reject a valid artifact.
func TestDescriptionLimitCountsCharactersNotBytes(t *testing.T) {
	dir := t.TempDir()
	desc := strings.Repeat("—", 1024) // 3 bytes each, 1024 characters
	mustWrite(t, filepath.Join(dir, "agents", "a-agent.md"), "---\nname: a-agent\ndescription: \""+desc+"\"\n---\n")

	code, _, stderr := runCLI(t, "validate", dir)
	if code != exitOK {
		t.Errorf("exit = %d, want 0; stderr = %q", code, stderr)
	}
}

// A failed artifact must be visible in the exit code and the summary, not just
// in a line of output.
func TestAFailedArtifactIsReportedAndChangesTheExitCode(t *testing.T) {
	home := sandboxHome(t)
	tree := fixtureTree(t)
	serveReleases(t)
	// A directory harness did not install makes exactly one artifact fail.
	mustWrite(t, filepath.Join(home, ".claude", "skills", "dev-loop", "SKILL.md"), "hand-written")

	code, _, stderr := runCLI(t, "install", "--from", tree, "--all")
	if code != exitErr {
		t.Errorf("exit = %d, want %d", code, exitErr)
	}
	if !strings.Contains(stderr, "1 of 6 artifact(s) failed") {
		t.Errorf("stderr should summarise the failures, got %q", stderr)
	}
	// The others still install: one bad artifact does not abort the batch.
	if _, err := os.Stat(filepath.Join(home, ".claude", "agents", "unbiased-reviewer.md")); err != nil {
		t.Errorf("unrelated artifacts should still install: %v", err)
	}
}

// A missing SKILL.md must be named, not surfaced as a raw lstat error, and
// must not leave a stray directory behind.
func TestMissingSkillFileIsReportedClearly(t *testing.T) {
	home := sandboxHome(t)
	root := fixtureTree(t)
	if err := os.Remove(filepath.Join(root, "skills", "dev-loop", "SKILL.md")); err != nil {
		t.Fatal(err)
	}

	code, _, stderr := runCLI(t, "install", "--from", root, "dev-loop")
	if code != exitErr {
		t.Errorf("exit = %d, want %d", code, exitErr)
	}
	if !strings.Contains(stderr, "SKILL.md") {
		t.Errorf("stderr = %q, want it to name the missing file", stderr)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "dev-loop")); err == nil {
		t.Error("a failed install must not leave a stray skill directory")
	}
}
