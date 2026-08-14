package main

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// FindProjectConfig — walk-up semantics
// ---------------------------------------------------------------------------

// makeTree creates a temporary directory tree under t.TempDir() with the
// given files (path → contents) and returns the temp root. Directories
// implied by file paths are created automatically. An empty content
// string means "create a directory at this path" (handy for .git/).
func makeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		abs := filepath.Join(root, rel)
		if content == "" {
			if err := os.MkdirAll(abs, 0o755); err != nil {
				t.Fatalf("mkdirall %q: %v", abs, err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("mkdirall parent of %q: %v", abs, err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatalf("write %q: %v", abs, err)
		}
	}
	return root
}

func TestFindProjectConfig_FileAtCWD(t *testing.T) {
	root := makeTree(t, map[string]string{
		ProjectConfigFilename: "theme: example-dark-tech\n",
	})
	got, err := FindProjectConfig(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(root, ProjectConfigFilename)
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestFindProjectConfig_WalksUpToParent(t *testing.T) {
	root := makeTree(t, map[string]string{
		ProjectConfigFilename: "theme: example-light-editorial\n",
		"src/feature/foo.go":  "package foo\n",
	})
	startDir := filepath.Join(root, "src", "feature")
	got, err := FindProjectConfig(startDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(root, ProjectConfigFilename)
	if got != want {
		t.Errorf("expected walk-up to %q, got %q", want, got)
	}
}

func TestFindProjectConfig_StopsAtGitBoundary(t *testing.T) {
	// Parent dir has the config; nested dir has a .git/ between them.
	// The walk-up MUST stop at .git/ and not return the parent's config.
	root := makeTree(t, map[string]string{
		ProjectConfigFilename:    "theme: parent-theme\n",
		"inner-repo/.git":        "",
		"inner-repo/src/main.go": "package main\n",
	})
	startDir := filepath.Join(root, "inner-repo", "src")
	got, err := FindProjectConfig(startDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty (walk stopped at .git boundary), got %q", got)
	}
}

func TestFindProjectConfig_NotFoundIsCleanEmpty(t *testing.T) {
	root := makeTree(t, map[string]string{
		"src/main.go": "package main\n",
	})
	got, err := FindProjectConfig(filepath.Join(root, "src"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty for missing config, got %q", got)
	}
}

func TestFindProjectConfig_DefaultsToCWD(t *testing.T) {
	root := makeTree(t, map[string]string{
		ProjectConfigFilename: "theme: example-minimal-mono\n",
	})
	prev, _ := os.Getwd()
	defer os.Chdir(prev) //nolint:errcheck
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	got, err := FindProjectConfig("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// On macOS the temp dir's real path may differ (e.g. /private/var/...).
	// Match by basename rather than full path equality.
	if filepath.Base(got) != ProjectConfigFilename {
		t.Errorf("expected basename %q, got %q", ProjectConfigFilename, got)
	}
}

// ---------------------------------------------------------------------------
// LoadProjectConfig — parsing + nil-when-absent
// ---------------------------------------------------------------------------

func TestLoadProjectConfig_NilWhenAbsent(t *testing.T) {
	root := makeTree(t, map[string]string{
		"src/main.go": "package main\n",
	})
	pc, err := LoadProjectConfig(filepath.Join(root, "src"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pc != nil {
		t.Errorf("expected nil when no config found, got %+v", pc)
	}
}

func TestLoadProjectConfig_ParsesFields(t *testing.T) {
	root := makeTree(t, map[string]string{
		ProjectConfigFilename: `theme: my-brand
handle: "@mybrand"
platform: linkedin-4x5
`,
	})
	pc, err := LoadProjectConfig(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pc == nil {
		t.Fatalf("expected populated config, got nil")
	}
	if pc.Theme != "my-brand" {
		t.Errorf("expected theme=my-brand, got %q", pc.Theme)
	}
	if pc.Handle != "@mybrand" {
		t.Errorf("expected handle=@mybrand, got %q", pc.Handle)
	}
	if pc.Platform != "linkedin-4x5" {
		t.Errorf("expected platform=linkedin-4x5, got %q", pc.Platform)
	}
	if pc.Path() == "" {
		t.Error("expected Path() to be set after loading")
	}
}

// ---------------------------------------------------------------------------
// ApplyToCarousel — gap-fill semantics (carousel YAML always wins)
// ---------------------------------------------------------------------------

func TestApplyToCarousel_FillsBlanks(t *testing.T) {
	pc := &ProjectConfig{
		Theme:    "project-theme",
		Handle:   "@project",
		Platform: "linkedin-4x5",
	}
	c := &Carousel{} // all blank
	pc.ApplyToCarousel(c)
	if c.Theme != "project-theme" {
		t.Errorf("expected theme filled from project, got %q", c.Theme)
	}
	if c.Handle != "@project" {
		t.Errorf("expected handle filled, got %q", c.Handle)
	}
	if c.Platform != "linkedin-4x5" {
		t.Errorf("expected platform filled, got %q", c.Platform)
	}
}

func TestApplyToCarousel_CarouselWinsWhenSet(t *testing.T) {
	pc := &ProjectConfig{
		Theme:    "project-theme",
		Handle:   "@project",
		Platform: "linkedin-4x5",
	}
	c := &Carousel{
		Theme:    "carousel-theme",
		Handle:   "@carousel",
		Platform: "instagram-4x5",
	}
	pc.ApplyToCarousel(c)
	if c.Theme != "carousel-theme" {
		t.Error("carousel theme must win over project")
	}
	if c.Handle != "@carousel" {
		t.Error("carousel handle must win over project")
	}
	if c.Platform != "instagram-4x5" {
		t.Error("carousel platform must win over project")
	}
}

func TestApplyToCarousel_NilSafe(t *testing.T) {
	var pc *ProjectConfig
	c := &Carousel{Theme: "x"}
	if got := pc.ApplyToCarousel(c); got != c {
		t.Error("nil receiver must return carousel unchanged")
	}
}

// ---------------------------------------------------------------------------
// SanitizeThemeName — path-traversal guard
// ---------------------------------------------------------------------------

func TestSanitizeThemeName(t *testing.T) {
	cases := []struct {
		in      string
		wantErr bool
	}{
		{"mybrand", false},
		{"  spaces-trimmed  ", false},
		{"", true},
		{"../escape", true},
		{"sub/dir", true},
		{`win\\path`, true},
	}
	for _, tc := range cases {
		_, err := SanitizeThemeName(tc.in)
		if (err != nil) != tc.wantErr {
			t.Errorf("SanitizeThemeName(%q): wantErr=%v got err=%v", tc.in, tc.wantErr, err)
		}
	}
}

// ---------------------------------------------------------------------------
// GlobalConfigDir / GlobalThemesDir — basic non-empty contract
// ---------------------------------------------------------------------------

func TestGlobalConfigDir_NonEmpty(t *testing.T) {
	dir, err := GlobalConfigDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dir == "" {
		t.Error("global config dir must not be empty")
	}
	if filepath.Base(dir) != appSlug {
		t.Errorf("expected leaf %q, got %q", appSlug, filepath.Base(dir))
	}
}

func TestGlobalThemesDir_EndsInThemes(t *testing.T) {
	dir, err := GlobalThemesDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Base(dir) != "themes" {
		t.Errorf("expected themes leaf, got %q", filepath.Base(dir))
	}
}
