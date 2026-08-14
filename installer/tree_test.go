package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sandboxHome points both the Claude home and the OS home at a temp dir.
// Overriding CLAUDE_HOME alone is not enough: userBinDir reads the real
// os.UserHomeDir, so a half-sandboxed test would write into the developer's
// ~/.local/bin.
func sandboxHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("CLAUDE_HOME", filepath.Join(dir, ".claude"))
	return dir
}

type tarEntry struct {
	name string
	body string
	dir  bool
}

func makeTarGz(t *testing.T, entries []tarEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		hdr := &tar.Header{Name: e.name, Mode: 0o644, Size: int64(len(e.body))}
		if e.dir {
			hdr.Typeflag = tar.TypeDir
			hdr.Mode = 0o755
			hdr.Size = 0
		} else {
			hdr.Typeflag = tar.TypeReg
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if !e.dir {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// serveTarball redirects the installer's HTTP client at a test server, so the
// real download → gunzip → untar → strip-prefix path runs end to end.
func serveTarball(t *testing.T, payload []byte) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(payload)
	}))
	t.Cleanup(srv.Close)

	prev := httpClient
	t.Cleanup(func() { httpClient = prev })
	httpClient = &http.Client{Transport: rewriteTo(srv.URL)}
}

type rewriteTransport struct{ host string }

func rewriteTo(serverURL string) http.RoundTripper {
	return rewriteTransport{host: strings.TrimPrefix(serverURL, "http://")}
}

func (rt rewriteTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	clone := r.Clone(r.Context())
	clone.URL.Scheme = "http"
	clone.URL.Host = rt.host
	return http.DefaultTransport.RoundTrip(clone)
}

// GitHub wraps the tree in a single <repo>-<ref> directory and mangles the ref
// (a leading "v" is dropped), so the prefix is discovered, never guessed.
func TestRemoteTreeStripsTheCodeloadRootPrefix(t *testing.T) {
	payload := makeTarGz(t, []tarEntry{
		{name: "harness-0.1.0/", dir: true},
		{name: "harness-0.1.0/skills/", dir: true},
		{name: "harness-0.1.0/skills/dev-loop/", dir: true},
		{name: "harness-0.1.0/skills/dev-loop/SKILL.md", body: "---\nname: dev-loop\n---\nbody\n"},
	})
	serveTarball(t, payload)

	tree := &remoteTree{repo: "diegoclair/harness", ref: "v0.1.0", tmp: t.TempDir()}
	root, err := tree.root()
	if err != nil {
		t.Fatalf("root(): %v", err)
	}
	if filepath.Base(root) != "harness-0.1.0" {
		t.Errorf("root = %s, want the archive's single top-level directory", root)
	}
	if _, err := os.Stat(filepath.Join(root, "skills", "dev-loop", "SKILL.md")); err != nil {
		t.Errorf("prefix strip did not expose skills/: %v", err)
	}
}

func TestRemoteTreeDownloadsOnlyOncePerRun(t *testing.T) {
	payload := makeTarGz(t, []tarEntry{
		{name: "harness-main/", dir: true},
		{name: "harness-main/skills/", dir: true},
	})

	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Write(payload)
	}))
	defer srv.Close()
	prev := httpClient
	defer func() { httpClient = prev }()
	httpClient = &http.Client{Transport: rewriteTo(srv.URL)}

	tree := &remoteTree{repo: "diegoclair/harness", ref: "main", tmp: t.TempDir()}
	for i := 0; i < 3; i++ {
		if _, err := tree.root(); err != nil {
			t.Fatalf("root() #%d: %v", i, err)
		}
	}
	if hits != 1 {
		t.Errorf("downloaded %d times, want 1 for the whole run", hits)
	}
}

func TestUntarRejectsPathTraversal(t *testing.T) {
	// "harness-main/../evil.txt" would normalise back inside the destination
	// and is therefore harmless; only a path that climbs above it escapes.
	payload := makeTarGz(t, []tarEntry{
		{name: "harness-main/", dir: true},
		{name: "../evil.txt", body: "pwned"},
	})

	dir := t.TempDir()
	archive := filepath.Join(dir, "a.tar.gz")
	if err := os.WriteFile(archive, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "out")

	err := untar(archive, dest)
	if err == nil {
		t.Fatal("want an error for an entry escaping the destination")
	}
	if !strings.Contains(err.Error(), "escapes destination") {
		t.Errorf("error = %v, want it to name the traversal", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "evil.txt")); statErr == nil {
		t.Fatal("traversal entry was written outside the destination")
	}
}

func TestUntarSkipsNonRegularEntries(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	// A symlink in an untrusted archive can redirect a later write.
	if err := tw.WriteHeader(&tar.Header{
		Name: "harness-main/link", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd", Mode: 0o777,
	}); err != nil {
		t.Fatal(err)
	}
	tw.Close()
	gz.Close()

	dir := t.TempDir()
	archive := filepath.Join(dir, "a.tar.gz")
	if err := os.WriteFile(archive, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "out")
	if err := untar(archive, dest); err != nil {
		t.Fatalf("untar: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(dest, "harness-main", "link")); err == nil {
		t.Error("symlink entry was extracted; it should be skipped")
	}
}

// A repo tree on disk, matching the layout install reads.
func fixtureTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "skills", "dev-loop", "SKILL.md"), "---\nname: dev-loop\n---\nloop body\n")
	mustWrite(t, filepath.Join(root, "skills", "dev-loop", "reference", "notes.md"), "notes\n")
	mustWrite(t, filepath.Join(root, "skills", "implementation-plan", "SKILL.md"), "---\nname: implementation-plan\n---\nplan body\n")
	mustWrite(t, filepath.Join(root, "agents", "unbiased-reviewer.md"), "---\nname: unbiased-reviewer\n---\nreviewer body\n")
	return root
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestInstallPlacesSkillAndAgent(t *testing.T) {
	home := sandboxHome(t)
	tree := localTree{path: fixtureTree(t)}

	skill, _ := findArtifact("dev-loop")
	agent, _ := findArtifact("unbiased-reviewer")
	for _, a := range []Artifact{skill, agent} {
		if err := installOne(t, a, tree, io.Discard); err != nil {
			t.Fatalf("install %s: %v", a.Name, err)
		}
	}

	assertFile(t, filepath.Join(home, ".claude", "skills", "dev-loop", "SKILL.md"), "loop body")
	assertFile(t, filepath.Join(home, ".claude", "skills", "dev-loop", "reference", "notes.md"), "notes")
	assertFile(t, filepath.Join(home, ".claude", "agents", "unbiased-reviewer.md"), "reviewer body")
}

// A skill with no binary in its payload must not keep one from a previous
// install — see TestBinaryIsDroppedWhenASkillStopsShippingOne.
func TestPlainSkillInstallClearsStaleFiles(t *testing.T) {
	home := sandboxHome(t)
	tree := localTree{path: fixtureTree(t)}
	skill, _ := findArtifact("dev-loop")

	dst := filepath.Join(home, ".claude", "skills", "dev-loop")
	mustWrite(t, filepath.Join(dst, markerFile), "dev-loop\n")
	mustWrite(t, filepath.Join(dst, "stale-reference.md"), "old")

	if err := installOne(t, skill, tree, io.Discard); err != nil {
		t.Fatalf("install: %v", err)
	}
	assertFile(t, filepath.Join(dst, "SKILL.md"), "loop body")
	if _, err := os.Stat(filepath.Join(dst, "stale-reference.md")); err == nil {
		t.Error("a renamed/removed file lingered; the skill directory must be clean-slated")
	}
}

// Wiping a directory the harness did not create would destroy a user's own
// skill of the same name.
func TestInstallRefusesToWipeAForeignDirectory(t *testing.T) {
	home := sandboxHome(t)
	tree := localTree{path: fixtureTree(t)}
	skill, _ := findArtifact("dev-loop")

	dst := filepath.Join(home, ".claude", "skills", "dev-loop")
	mustWrite(t, filepath.Join(dst, "SKILL.md"), "hand-written, not ours")

	err := installOne(t, skill, tree, io.Discard)
	if err == nil {
		t.Fatal("want a refusal when the target was not installed by harness")
	}
	assertFile(t, filepath.Join(dst, "SKILL.md"), "hand-written, not ours")
}

func TestLocalTreeRejectsANonHarnessDirectory(t *testing.T) {
	if _, err := (localTree{path: t.TempDir()}).root(); err == nil {
		t.Fatal("want an error for a directory with no skills/")
	}
}

func TestInstallReportsAMissingArtifact(t *testing.T) {
	sandboxHome(t)
	tree := localTree{path: fixtureTree(t)}
	missing := Artifact{Name: "not-there", Kind: KindSkill}

	if err := installOne(t, missing, tree, io.Discard); err == nil {
		t.Fatal("want an error when the artifact is absent from the tree")
	}
}

func assertFile(t *testing.T, path, wantSubstring string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected %s: %v", path, err)
	}
	if !strings.Contains(string(b), wantSubstring) {
		t.Errorf("%s = %q, want it to contain %q", path, b, wantSubstring)
	}
}

// Installing twice must work: the first install has to leave the marker the
// second one checks for. Without this the marker write can regress unnoticed
// and every upgrade fails.
func TestInstallIsIdempotent(t *testing.T) {
	home := sandboxHome(t)
	tree := localTree{path: fixtureTree(t)}
	skill, _ := findArtifact("dev-loop")

	for i := range 2 {
		if err := installOne(t, skill, tree, io.Discard); err != nil {
			t.Fatalf("install #%d: %v", i+1, err)
		}
	}
	assertFile(t, filepath.Join(home, ".claude", "skills", "dev-loop", "SKILL.md"), "loop body")
	assertFile(t, filepath.Join(home, ".claude", "skills", "dev-loop", markerFile), "dev-loop")
}

// An agent is a bare file with nowhere to keep a marker, and a skill can pull
// one in without the user naming it — so a hand-written one is backed up.
func TestAgentInstallBacksUpDifferingContent(t *testing.T) {
	home := sandboxHome(t)
	tree := localTree{path: fixtureTree(t)}
	agent, _ := findArtifact("unbiased-reviewer")

	dst := filepath.Join(home, ".claude", "agents", "unbiased-reviewer.md")
	mustWrite(t, dst, "MY OWN HAND-WRITTEN REVIEWER")

	var out strings.Builder
	if err := installOne(t, agent, tree, &out); err != nil {
		t.Fatalf("install: %v", err)
	}
	assertFile(t, dst, "reviewer body")
	assertFile(t, dst+".bak", "MY OWN HAND-WRITTEN REVIEWER")
	if !strings.Contains(out.String(), "previous version saved") {
		t.Errorf("the replacement should be reported, got %q", out.String())
	}
}

// Reinstalling identical content is not a replacement and must not litter.
func TestAgentInstallDoesNotBackUpIdenticalContent(t *testing.T) {
	home := sandboxHome(t)
	tree := localTree{path: fixtureTree(t)}
	agent, _ := findArtifact("unbiased-reviewer")

	for range 2 {
		if err := installOne(t, agent, tree, io.Discard); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "agents", "unbiased-reviewer.md.bak")); err == nil {
		t.Error("an unchanged reinstall should not create a backup")
	}
}

func TestCopyTreePreservesTheExecutableBit(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	script := filepath.Join(src, "run.sh")
	mustWrite(t, script, "#!/bin/sh\n")
	if err := os.Chmod(script, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := copyTree(src, dst); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dst, "run.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Errorf("mode = %v, want the executable bit preserved", info.Mode().Perm())
	}
}

func TestRemoteTreeDoesNotRetryAFailedFetch(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	prev := httpClient
	defer func() { httpClient = prev }()
	httpClient = &http.Client{Transport: rewriteTo(srv.URL)}

	tree := &remoteTree{repo: "diegoclair/harness", ref: "main", tmp: t.TempDir()}
	for range 3 {
		if _, err := tree.root(); err == nil {
			t.Fatal("want an error")
		}
	}
	if hits != 1 {
		t.Errorf("retried %d times; a failed fetch should be remembered", hits)
	}
}

func TestUntarRejectsAnUnexpectedArchiveLayout(t *testing.T) {
	for _, tc := range []struct {
		name    string
		entries []tarEntry
	}{
		{"two top-level dirs", []tarEntry{{name: "a/", dir: true}, {name: "b/", dir: true}}},
		{"no top-level dir", []tarEntry{{name: "loose.txt", body: "x"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			serveTarball(t, makeTarGz(t, tc.entries))
			tree := &remoteTree{repo: "r/r", ref: "main", tmp: t.TempDir()}
			if _, err := tree.root(); err == nil {
				t.Fatal("want an error for an unexpected archive layout")
			}
		})
	}
}

// installOne drives the real entry points the CLI uses, so the tests exercise
// the same routing rather than a parallel one.
func installOne(t *testing.T, a Artifact, tree treeProvider, out io.Writer) error {
	t.Helper()
	root, err := tree.root()
	if err != nil {
		return err
	}
	if a.Kind == KindAgent {
		return installAgent(a, root, "test", out)
	}
	return installSkill(a, root, installOptions{Repo: defaultRepo, Ref: "test", Out: out})
}
