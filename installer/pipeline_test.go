package main

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeCLI stands in for a released binary: enough of the contract for the
// installer's verification and credential probe to run for real.
const fakeCLI = `#!/bin/sh
case "$1" in
  --version) echo "cli-skill v9.9.9" ;;
  setup)     exit 0 ;;
  *)         exit 1 ;;
esac
`

func makeReleaseZip(t *testing.T, skill, binBody string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	add := func(name, body string, mode os.FileMode) {
		hdr := &zip.FileHeader{Name: name, Method: zip.Deflate}
		hdr.SetMode(mode)
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	add("SKILL.md", "---\nname: "+skill+"\n---\nreleased body\n", 0o644)
	add("reference/workflows.md", "released reference\n", 0o644)
	// The real release workflow has shipped cli/ before; the installer, not the
	// fixture, has to be what keeps it out of ~/.claude.
	add("cli/main.go", "package main\n", 0o644)
	add("cli/go.mod", "module x\n", 0o644)
	if binBody != "" {
		add("bin/"+skill, binBody, 0o755)
	}

	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// serveRelease answers both the releases API (tag resolution) and the asset
// download, so the release path runs end to end.
func serveRelease(t *testing.T, tag string, zipBody []byte) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/releases") && !strings.Contains(r.URL.Path, "/download/") {
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `[{"tag_name":"`+tag+`","draft":false,"prerelease":false}]`)
			return
		}
		w.Write(zipBody)
	}))
	t.Cleanup(srv.Close)

	prev := httpClient
	t.Cleanup(func() { httpClient = prev })
	httpClient = &http.Client{Transport: rewriteTo(srv.URL)}
}

// The whole point of the single pipeline: the same command installs a skill
// with a binary and one without, and only the first gets a binary on PATH.
func TestOnePipelineHandlesBothWithAndWithoutABinary(t *testing.T) {
	t.Run("no cli directory: files only", func(t *testing.T) {
		home := sandboxHome(t)
		tree := localTree{path: fixtureTree(t)}
		skill, _ := findArtifact("dev-loop")

		if err := installOne(t, skill, tree, io.Discard); err != nil {
			t.Fatalf("install: %v", err)
		}
		assertFile(t, filepath.Join(home, ".claude", "skills", "dev-loop", "SKILL.md"), "loop body")
		if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "dev-loop", "bin")); err == nil {
			t.Error("a skill with no cli/ must not get a bin/ directory")
		}
		if _, err := os.Lstat(filepath.Join(home, ".local", "bin", "dev-loop")); err == nil {
			t.Error("a skill with no binary must not be linked onto PATH")
		}
	})

	t.Run("cli directory present: binary installed and linked", func(t *testing.T) {
		home := sandboxHome(t)
		root := fixtureTree(t)
		// The artifact declares it drives a binary by shipping cli/ — no
		// catalog flag involved.
		mustWrite(t, filepath.Join(root, "skills", "cli-skill", "SKILL.md"), "---\nname: cli-skill\n---\ntree body\n")
		mustWrite(t, filepath.Join(root, "skills", "cli-skill", "cli", "main.go"), "package main\n")
		serveRelease(t, "cli-v9.9.9", makeReleaseZip(t, "cli-skill", fakeCLI))

		skill := Artifact{Name: "cli-skill", Kind: KindSkill, TagPrefix: "cli-v"}
		var out strings.Builder
		if err := installOne(t, skill, localTree{path: root}, &out); err != nil {
			t.Fatalf("install: %v\n%s", err, out.String())
		}

		dir := filepath.Join(home, ".claude", "skills", "cli-skill")
		// Payload came from the release, not the tree — one consistent snapshot.
		assertFile(t, filepath.Join(dir, "SKILL.md"), "released body")
		assertFile(t, filepath.Join(dir, "reference", "workflows.md"), "released reference")

		info, err := os.Stat(filepath.Join(dir, "bin", "cli-skill"))
		if err != nil {
			t.Fatalf("binary was not installed: %v", err)
		}
		if info.Mode().Perm()&0o111 == 0 {
			t.Errorf("binary mode = %v, want it executable", info.Mode().Perm())
		}
		link, err := os.Readlink(filepath.Join(home, ".local", "bin", "cli-skill"))
		if err != nil {
			t.Fatalf("binary was not linked onto PATH: %v", err)
		}
		if link != filepath.Join(dir, "bin", "cli-skill") {
			t.Errorf("symlink -> %s", link)
		}
		if !strings.Contains(out.String(), "cli-skill v9.9.9") {
			t.Errorf("the installed binary should be verified by running it, got:\n%s", out.String())
		}
	})
}

// The cli/ source tree is a build input, not something a user needs installed.
func TestReleasePayloadDoesNotShipTheSourceTree(t *testing.T) {
	home := sandboxHome(t)
	root := fixtureTree(t)
	mustWrite(t, filepath.Join(root, "skills", "cli-skill", "SKILL.md"), "---\nname: cli-skill\n---\n")
	mustWrite(t, filepath.Join(root, "skills", "cli-skill", "cli", "main.go"), "package main\n")
	serveRelease(t, "cli-v9.9.9", makeReleaseZip(t, "cli-skill", fakeCLI))

	skill := Artifact{Name: "cli-skill", Kind: KindSkill, TagPrefix: "cli-v"}
	if err := installOne(t, skill, localTree{path: root}, io.Discard); err != nil {
		t.Fatalf("install: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "cli-skill", "cli")); err == nil {
		t.Error("cli/ is a build input and must not be installed")
	}
}

// Upgrading in place must replace a running executable, which a plain copy
// cannot do (ETXTBSY) — this is the `<skill> update` path.
func TestBinarySkillReinstallsOverItself(t *testing.T) {
	home := sandboxHome(t)
	root := fixtureTree(t)
	mustWrite(t, filepath.Join(root, "skills", "cli-skill", "SKILL.md"), "---\nname: cli-skill\n---\n")
	mustWrite(t, filepath.Join(root, "skills", "cli-skill", "cli", "main.go"), "package main\n")
	serveRelease(t, "cli-v9.9.9", makeReleaseZip(t, "cli-skill", fakeCLI))

	skill := Artifact{Name: "cli-skill", Kind: KindSkill, TagPrefix: "cli-v"}
	for i := range 2 {
		if err := installOne(t, skill, localTree{path: root}, io.Discard); err != nil {
			t.Fatalf("install #%d: %v", i+1, err)
		}
	}
	assertFile(t, filepath.Join(home, ".claude", "skills", "cli-skill", "bin", "cli-skill"), "cli-skill v9.9.9")
}

// A version pin is meaningless for an artifact that ships no binary; saying so
// beats silently ignoring the flag.
func TestVersionPinIsRejectedForAnArtifactWithoutABinary(t *testing.T) {
	sandboxHome(t)
	tree := fixtureTree(t)

	code, _, stderr := runCLI(t, "install", "--from", tree, "--version", "v1.2.3", "dev-loop")
	if code != exitInputErr {
		t.Fatalf("exit = %d, want %d", code, exitInputErr)
	}
	if !strings.Contains(stderr, "ship a binary") {
		t.Errorf("stderr = %q", stderr)
	}
	// Flag applicability is discovered from the tree, so the check runs late —
	// but still before a single file is written.
	home := os.Getenv("HOME")
	for _, p := range []string{
		filepath.Join(home, ".claude", "skills", "dev-loop"),
		filepath.Join(home, ".claude", "agents", "unbiased-reviewer.md"),
	} {
		if _, err := os.Stat(p); err == nil {
			t.Errorf("%s was written despite the input error", p)
		}
	}
}

// A missing binary in an archive that should have one is a broken release, not
// a skill to install quietly without it.
func TestReleaseWithoutTheBinaryFails(t *testing.T) {
	sandboxHome(t)
	root := fixtureTree(t)
	mustWrite(t, filepath.Join(root, "skills", "cli-skill", "SKILL.md"), "---\nname: cli-skill\n---\n")
	mustWrite(t, filepath.Join(root, "skills", "cli-skill", "cli", "main.go"), "package main\n")

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("SKILL.md")
	w.Write([]byte("---\nname: cli-skill\n---\n"))
	zw.Close()
	serveRelease(t, "cli-v9.9.9", buf.Bytes())

	skill := Artifact{Name: "cli-skill", Kind: KindSkill, TagPrefix: "cli-v"}
	err := installOne(t, skill, localTree{path: root}, io.Discard)
	if err == nil {
		t.Fatal("want an error: the skill ships cli/ but the release carried no binary")
	}
	if !strings.Contains(err.Error(), "binary") {
		t.Errorf("error should name the missing binary, got: %v", err)
	}
}

// A broken release must not leave a loadable skill behind: Claude Code would
// pick it up with every command missing.
func TestBrokenReleaseWritesNothing(t *testing.T) {
	home := sandboxHome(t)
	root := fixtureTree(t)
	mustWrite(t, filepath.Join(root, "skills", "cli-skill", "SKILL.md"), "---\nname: cli-skill\n---\n")
	mustWrite(t, filepath.Join(root, "skills", "cli-skill", "cli", "main.go"), "package main\n")
	serveRelease(t, "cli-v9.9.9", makeReleaseZip(t, "cli-skill", "")) // no bin/ entry

	skill := Artifact{Name: "cli-skill", Kind: KindSkill, TagPrefix: "cli-v"}
	if err := installOne(t, skill, localTree{path: root}, io.Discard); err == nil {
		t.Fatal("want an error: the release carried no binary")
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "cli-skill")); err == nil {
		t.Error("a failed install must leave nothing behind")
	}
}

// A skill that stops shipping cli/ must not leave its old binary runnable.
func TestBinaryIsDroppedWhenASkillStopsShippingOne(t *testing.T) {
	home := sandboxHome(t)
	root := fixtureTree(t)
	skillDir := filepath.Join(root, "skills", "cli-skill")
	mustWrite(t, filepath.Join(skillDir, "SKILL.md"), "---\nname: cli-skill\n---\ntree body\n")
	mustWrite(t, filepath.Join(skillDir, "cli", "main.go"), "package main\n")
	serveRelease(t, "cli-v9.9.9", makeReleaseZip(t, "cli-skill", fakeCLI))

	skill := Artifact{Name: "cli-skill", Kind: KindSkill, TagPrefix: "cli-v"}
	if err := installOne(t, skill, localTree{path: root}, io.Discard); err != nil {
		t.Fatalf("first install: %v", err)
	}
	installed := filepath.Join(home, ".claude", "skills", "cli-skill")
	link := filepath.Join(home, ".local", "bin", "cli-skill")
	if _, err := os.Lstat(link); err != nil {
		t.Fatalf("setup: the binary should be on PATH: %v", err)
	}

	// The skill drops its CLI and becomes plain files.
	if err := os.RemoveAll(filepath.Join(skillDir, "cli")); err != nil {
		t.Fatal(err)
	}
	if err := installOne(t, skill, localTree{path: root}, io.Discard); err != nil {
		t.Fatalf("second install: %v", err)
	}
	if _, err := os.Stat(filepath.Join(installed, "bin")); err == nil {
		t.Error("the stale binary is still installed")
	}
	if _, err := os.Lstat(link); err == nil {
		t.Error("the PATH link still points at a binary the skill no longer ships")
	}
	assertFile(t, filepath.Join(installed, "SKILL.md"), "tree body")
}

// The binary is replaced by rename, not by writing over it: a plain copy
// cannot overwrite a file it may not open for writing — the same failure a
// running executable produces (ETXTBSY) on the `<skill> update` path.
func TestBinaryInstallReplacesAnUnwritableFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "new")
	dst := filepath.Join(dir, "installed")
	mustWrite(t, src, "new binary")
	mustWrite(t, dst, "old binary")
	if err := os.Chmod(dst, 0o500); err != nil {
		t.Fatal(err)
	}

	if err := installBinary(src, dst); err != nil {
		t.Fatalf("installBinary: %v", err)
	}
	assertFile(t, dst, "new binary")
}

// The pin has to reach the download URL, not just be accepted.
func TestVersionPinSelectsTheRequestedRelease(t *testing.T) {
	home := sandboxHome(t)
	root := fixtureTree(t)
	mustWrite(t, filepath.Join(root, "skills", "cli-skill", "SKILL.md"), "---\nname: cli-skill\n---\n")
	mustWrite(t, filepath.Join(root, "skills", "cli-skill", "cli", "main.go"), "package main\n")

	zipBody := makeReleaseZip(t, "cli-skill", fakeCLI)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only the pinned tag exists; resolving "latest" would 404.
		if !strings.Contains(r.URL.Path, "cli-v1.0.0") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write(zipBody)
	}))
	defer srv.Close()
	prev := httpClient
	defer func() { httpClient = prev }()
	httpClient = &http.Client{Transport: rewriteTo(srv.URL)}

	skill := Artifact{Name: "cli-skill", Kind: KindSkill, TagPrefix: "cli-v"}
	root2, _ := localTree{path: root}.root()
	err := installSkill(skill, root2, installOptions{Repo: defaultRepo, Version: "cli-v1.0.0", Out: io.Discard})
	if err != nil {
		t.Fatalf("install with a pinned version: %v", err)
	}
	assertFile(t, filepath.Join(home, ".claude", "skills", "cli-skill", "SKILL.md"), "released body")
}

// The skill's own credential probe and post-install hook are part of the
// binary stage; silently skipping them would hide a misconfigured install.
func TestBinaryStageRunsTheSkillsOwnChecks(t *testing.T) {
	sandboxHome(t)
	root := fixtureTree(t)
	mustWrite(t, filepath.Join(root, "skills", "cli-skill", "SKILL.md"), "---\nname: cli-skill\n---\n")
	mustWrite(t, filepath.Join(root, "skills", "cli-skill", "cli", "main.go"), "package main\n")

	hookCLI := `#!/bin/sh
case "$1" in
  --version)   echo "cli-skill v9.9.9" ;;
  setup)       exit 0 ;;
  postinstall) [ "$2" = "--check" ] && exit 0; echo "HOOK RAN" ;;
  *)           exit 1 ;;
esac
`
	serveRelease(t, "cli-v9.9.9", makeReleaseZip(t, "cli-skill", hookCLI))

	skill := Artifact{Name: "cli-skill", Kind: KindSkill, TagPrefix: "cli-v"}
	var out strings.Builder
	if err := installOne(t, skill, localTree{path: root}, &out); err != nil {
		t.Fatalf("install: %v\n%s", err, out.String())
	}
	for _, want := range []string{"HOOK RAN", "Already configured."} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output is missing %q:\n%s", want, out.String())
		}
	}
}

// A directory named cli is the declaration; a regular file called cli is not.
func TestShipsCLIRequiresADirectory(t *testing.T) {
	dir := t.TempDir()
	if shipsCLI(dir) {
		t.Error("no cli entry at all")
	}
	mustWrite(t, filepath.Join(dir, "cli"), "not a directory")
	if shipsCLI(dir) {
		t.Error("a regular file named cli must not count as a CLI")
	}
	if err := os.Remove(filepath.Join(dir, "cli")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "cli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !shipsCLI(dir) {
		t.Error("a cli directory declares a CLI")
	}
}

func TestUnzipRejectsPathTraversal(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("../evil.txt")
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]byte("pwned"))
	zw.Close()

	archive := filepath.Join(dir, "a.zip")
	if err := os.WriteFile(archive, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := unzip(archive, filepath.Join(dir, "out")); err == nil {
		t.Fatal("want an error for an entry escaping the destination")
	}
	if _, err := os.Stat(filepath.Join(dir, "evil.txt")); err == nil {
		t.Fatal("traversal entry was written outside the destination")
	}
}

// bin/ must be excluded from the file copy so the binary only ever arrives by
// atomic rename: copying would fail against a binary that cannot be opened for
// writing, which is what a running executable looks like.
func TestReinstallSucceedsOverAnUnwritableInstalledBinary(t *testing.T) {
	home := sandboxHome(t)
	root := fixtureTree(t)
	mustWrite(t, filepath.Join(root, "skills", "cli-skill", "SKILL.md"), "---\nname: cli-skill\n---\n")
	mustWrite(t, filepath.Join(root, "skills", "cli-skill", "cli", "main.go"), "package main\n")
	serveRelease(t, "cli-v9.9.9", makeReleaseZip(t, "cli-skill", fakeCLI))

	skill := Artifact{Name: "cli-skill", Kind: KindSkill, TagPrefix: "cli-v"}
	if err := installOne(t, skill, localTree{path: root}, io.Discard); err != nil {
		t.Fatalf("first install: %v", err)
	}
	installedBin := filepath.Join(home, ".claude", "skills", "cli-skill", "bin", "cli-skill")
	if err := os.Chmod(installedBin, 0o500); err != nil {
		t.Fatal(err)
	}
	if err := installOne(t, skill, localTree{path: root}, io.Discard); err != nil {
		t.Fatalf("reinstall over an unwritable binary: %v", err)
	}
}

// The skill's own diagnosis has to reach the user: collapsing it into a
// generic "not configured" is what the credential probe exists to avoid.
func TestUnconfiguredSkillReportsItsOwnReason(t *testing.T) {
	sandboxHome(t)
	root := fixtureTree(t)
	mustWrite(t, filepath.Join(root, "skills", "cli-skill", "SKILL.md"), "---\nname: cli-skill\n---\n")
	mustWrite(t, filepath.Join(root, "skills", "cli-skill", "cli", "main.go"), "package main\n")

	pickyCLI := `#!/bin/sh
case "$1" in
  --version) echo "cli-skill v9.9.9" ;;
  setup)     echo "no API token found for this workspace" >&2; exit 1 ;;
  *)         exit 1 ;;
esac
`
	serveRelease(t, "cli-v9.9.9", makeReleaseZip(t, "cli-skill", pickyCLI))

	skill := Artifact{Name: "cli-skill", Kind: KindSkill, TagPrefix: "cli-v"}
	var out strings.Builder
	if err := installOne(t, skill, localTree{path: root}, &out); err != nil {
		t.Fatalf("install: %v", err)
	}
	if !strings.Contains(out.String(), "no API token found for this workspace") {
		t.Errorf("the skill's own reason should be surfaced, got:\n%s", out.String())
	}
}
