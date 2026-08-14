package main

import (
	"os"
	"strings"
	"testing"
)

// The bootstrap scripts, the Makefile and the release workflow independently
// spell the asset name and tag prefix. A mismatch is a 404 that only shows up
// after publishing, so it is checked statically here.
func TestBootstrapArtifactNamingAgrees(t *testing.T) {
	files := map[string]string{
		"install.sh":  "../install.sh",
		"install.ps1": "../install.ps1",
		"Makefile":    "Makefile",
		".github/workflows/release-installer.yml": "../.github/workflows/release-installer.yml",
	}
	contents := map[string]string{}
	for label, path := range files {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		contents[label] = string(b)
	}

	mustContain := func(label, want, why string) {
		t.Helper()
		if !strings.Contains(contents[label], want) {
			t.Errorf("%s does not contain %q (%s)", label, want, why)
		}
	}

	// The binary the Makefile emits is the one the bootstraps download.
	mustContain("Makefile", "BIN     := harness", "asset base name")
	mustContain("install.sh", `ASSET="harness"`, "asset base name")
	mustContain("install.ps1", `$asset     = "harness"`, "asset base name")

	// Release tags carry the same prefix everywhere.
	mustContain("install.sh", `TAG_PREFIX="harness-v"`, "release tag prefix")
	mustContain("install.ps1", `$tagPrefix = "harness-v"`, "release tag prefix")
	mustContain(".github/workflows/release-installer.yml", "'harness-v*'", "release trigger")

	// The Makefile names assets through $(BIN); expand it so the literals the
	// bootstraps download are what cross-compilation actually emits.
	binValue := makeVariable(t, contents["Makefile"], "BIN")
	if binValue != "harness" {
		t.Fatalf("Makefile BIN = %q, want harness", binValue)
	}
	expanded := strings.ReplaceAll(contents["Makefile"], "$(BIN)", binValue)
	for _, asset := range []string{
		"dist/harness-darwin-amd64", "dist/harness-darwin-arm64",
		"dist/harness-linux-amd64", "dist/harness-linux-arm64",
		"dist/harness-windows-amd64.exe",
	} {
		if !strings.Contains(expanded, asset) {
			t.Errorf("Makefile does not cross-compile %s, which a bootstrap would download", asset)
		}
	}
	mustContain("install.ps1", "$asset-windows-amd64.exe", "windows asset suffix")
	mustContain("install.sh", "${ASSET}-${os}-${arch}", "unix asset suffix")

	// A fork must be reachable through both bootstraps under the same name.
	mustContain("install.sh", "HARNESS_REPO", "fork override")
	mustContain("install.ps1", "HARNESS_REPO", "fork override")
	mustContain("install.sh", "HARNESS_INSTALLER_VERSION", "version pin override")
	mustContain("install.ps1", "HARNESS_INSTALLER_VERSION", "version pin override")
}

// A bare `curl ... | sh` must not write into ~/.claude for someone who asked
// for nothing.
func TestBareBootstrapDoesNotInstallEverything(t *testing.T) {
	b, err := os.ReadFile("../install.sh")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if strings.Contains(s, "install --all") {
		t.Error("install.sh installs everything with no arguments; it should list and exit")
	}
	if !strings.Contains(s, "exit 2") {
		t.Error("install.sh should exit 2 when given no arguments")
	}
}

func TestDefaultRepoMatchesTheBootstraps(t *testing.T) {
	if defaultRepo != "diegoclair/harness" {
		t.Errorf("defaultRepo = %q", defaultRepo)
	}
	for _, path := range []string{"../install.sh", "../install.ps1"} {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), defaultRepo) {
			t.Errorf("%s does not reference %s", path, defaultRepo)
		}
	}
}

// makeVariable reads a simple `NAME := value` assignment out of a Makefile.
func makeVariable(t *testing.T, makefile, name string) string {
	t.Helper()
	for _, line := range strings.Split(makefile, "\n") {
		key, value, found := strings.Cut(line, ":=")
		if !found || strings.TrimSpace(key) != name {
			continue
		}
		return strings.TrimSpace(value)
	}
	t.Fatalf("Makefile has no %s assignment", name)
	return ""
}
