package main

import (
	"bytes"
	"io"
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
	for _, want := range []string{"Skills:", "Agents:", "dev-loop", "implementation-plan", "unbiased-reviewer", "confluence-docs"} {
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
		{"version with two release skills", []string{"install", "--version", "jira-v0.4.1", "jira-tickets", "confluence-docs"}, "single artifact"},
		{"version on a markdown artifact", []string{"install", "--version", "v1", "dev-loop"}, "no release tags"},
		{"ref on a release skill", []string{"install", "--ref", "main2", "confluence-docs"}, "does not apply"},
		{"from on a release skill", []string{"install", "--from", ".", "confluence-docs"}, "cannot be installed from a local tree"},
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

// A wildcard narrows to what a repo tree can provide instead of failing, and
// says what it skipped.
func TestAllSkillsFromALocalTreeSkipsReleaseBackedOnes(t *testing.T) {
	home := sandboxHome(t)
	tree := fixtureTree(t)

	code, stdout, stderr := runCLI(t, "install", "--from", tree, "--all-skills")
	if code != exitOK {
		t.Fatalf("exit = %d\nstderr:%s", code, stderr)
	}
	if !strings.Contains(stdout, "Skipping release-backed") {
		t.Errorf("skipped artifacts must be reported, not silently dropped\n%s", stdout)
	}
	for _, name := range []string{"confluence-docs", "jira-tickets", "social-carousel"} {
		if _, err := os.Stat(filepath.Join(home, ".claude", "skills", name)); err == nil {
			t.Errorf("%s is release-backed and should not have been installed from a tree", name)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "dev-loop", "SKILL.md")); err != nil {
		t.Errorf("dev-loop should have been installed: %v", err)
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

// Until the repo is published, release-kind resolution has to fail with
// something a user can act on.
func TestReleaseResolutionFailureIsActionable(t *testing.T) {
	sandboxHome(t)
	a, _ := findArtifact("confluence-docs")
	err := installFromRelease(a, installOptions{Repo: "diegoclair/definitely-not-a-real-repo", Out: io.Discard})
	if err == nil {
		t.Fatal("want an error when the repo has no releases")
	}
	if !strings.Contains(err.Error(), "--version") {
		t.Errorf("error should tell the user how to proceed, got: %v", err)
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
		{"install", "--ref=", "dev-loop"}, {"install", "--from=", "dev-loop"}, {"install", "--repo=", "dev-loop"},
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
}

// A catalogued artifact that has not been published yet must say so, not fail
// with a bare 404.
func TestPendingArtifactFailsWithAnExplanation(t *testing.T) {
	sandboxHome(t)
	code, _, stderr := runCLI(t, "install", "confluence-docs")
	if code != exitErr {
		t.Errorf("exit = %d, want %d", code, exitErr)
	}
	if !strings.Contains(stderr, "migration-from-skills") {
		t.Errorf("stderr should point at the migration doc, got %q", stderr)
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
