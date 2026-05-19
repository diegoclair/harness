// config.go — paths, project-local config walk-up, theme resolution chain.
//
// v0.2.0 added two layers around theme resolution:
//
//	1. **Project-local config** — a `carousel.config.yml` file at the user's
//	   project root that wins precedence over the global theme dir. Walked
//	   up from CWD until first hit, or a `.git/` boundary, or 10 levels
//	   ascended. Pattern borrowed from prettier / editorconfig.
//	2. **Cross-platform global config dir** — replaced the previous
//	   XDG-hardcoded `~/.config/social-carousel/` with `os.UserConfigDir()`
//	   so macOS gets `~/Library/Application Support/social-carousel/` and
//	   Windows gets `%AppData%\social-carousel\` per platform conventions.
//
// Both paths feed into the theme resolution chain documented on
// ResolveTheme below.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// ProjectConfigFilename is the file the CLI looks for when walking up
	// from CWD to find a project-local override. Visible (not a dotfile)
	// because users actively edit it — matches the `*.config.{js,ts,yml}`
	// pattern of next.config.js / vite.config.ts / tailwind.config.js.
	ProjectConfigFilename = "carousel.config.yml"

	// projectWalkUpMaxLevels caps the directory walk so an invocation deep
	// inside a monorepo doesn't traverse the whole filesystem.
	projectWalkUpMaxLevels = 10

	// appSlug names the subdirectory under the OS-specific config root.
	appSlug = "social-carousel"
)

// ProjectConfig is the on-disk shape of `carousel.config.yml`.
//
// Currently it only carries a theme block (inline or by name reference),
// but it's a struct rather than a raw map so future fields (default
// `handle`, default `platform`, default `caption_seed`, etc.) can land
// without breaking older configs.
type ProjectConfig struct {
	// Theme matches Carousel.Theme — a string that is either a preset
	// name (e.g. "example-dark-tech") OR a path to a yaml file on disk.
	// Inline theme blocks are not supported in v0.2.0 (parked for v0.3+).
	Theme string `yaml:"theme,omitempty" json:"theme,omitempty"`

	// Handle, if set, becomes the default footer handle for any carousel
	// rendered inside this project that doesn't override it.
	Handle string `yaml:"handle,omitempty" json:"handle,omitempty"`

	// Platform, if set, becomes the default platform for new carousels.
	Platform string `yaml:"platform,omitempty" json:"platform,omitempty"`

	// path is the absolute path the config was loaded from (for error
	// messages and `config resolve` diagnostics). Not serialised.
	path string `yaml:"-"`
}

// Path returns the absolute path the config was loaded from. Empty if
// the config came from defaults (no file found).
func (p *ProjectConfig) Path() string {
	if p == nil {
		return ""
	}
	return p.path
}

// GlobalConfigDir returns the OS-specific config directory for this app.
//
//	Linux:   $XDG_CONFIG_HOME/social-carousel  (fallback $HOME/.config/social-carousel)
//	macOS:   $HOME/Library/Application Support/social-carousel
//	Windows: %AppData%\social-carousel
//
// The directory is NOT created on disk — callers that write must `MkdirAll`.
//
// Uses `os.UserConfigDir()` from the stdlib, which returns the correct
// Apple-HIG path on macOS (the old folklore that Go returned
// `Library/Preferences` is outdated as of recent Go versions).
func GlobalConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(base, appSlug), nil
}

// GlobalThemesDir returns the path where user-authored themes live.
// Equivalent to `GlobalConfigDir() / themes/`.
func GlobalThemesDir() (string, error) {
	cfg, err := GlobalConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "themes"), nil
}

// FindProjectConfig walks up from startDir looking for ProjectConfigFilename.
// It stops at the first match, or when a `.git/` directory is encountered
// (project boundary), or after projectWalkUpMaxLevels ascended.
//
// Returns the absolute path of the config OR an empty string if none was
// found. An error is returned only when filesystem operations fail (eg.
// permission denied) — "not found" is a clean ("", nil).
//
// startDir == "" defaults to the current working directory.
func FindProjectConfig(startDir string) (string, error) {
	if startDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get cwd: %w", err)
		}
		startDir = cwd
	}

	abs, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("resolve abs path %q: %w", startDir, err)
	}

	dir := abs
	for level := 0; level < projectWalkUpMaxLevels; level++ {
		// Look for the config file at this level.
		candidate := filepath.Join(dir, ProjectConfigFilename)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}

		// `.git/` boundary — once we hit a repo root, stop ascending so a
		// parent repo's carousel config doesn't accidentally apply.
		if info, err := os.Stat(filepath.Join(dir, ".git")); err == nil && info.IsDir() {
			return "", nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root.
			return "", nil
		}
		dir = parent
	}
	return "", nil
}

// LoadProjectConfig finds and parses the nearest carousel.config.yml.
// Returns a populated *ProjectConfig OR nil if none was found (never
// returns an empty pointer + nil error to avoid the "did it load?"
// ambiguity).
func LoadProjectConfig(startDir string) (*ProjectConfig, error) {
	path, err := FindProjectConfig(startDir)
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read project config %q: %w", path, err)
	}

	var pc ProjectConfig
	if err := yaml.Unmarshal(data, &pc); err != nil {
		return nil, fmt.Errorf("parse project config %q: %w", path, err)
	}
	pc.path = path
	return &pc, nil
}

// ApplyToCarousel merges this project config into a Carousel value,
// filling in fields the carousel left blank. Project config NEVER
// overrides an explicit value set in the carousel YAML — only fills
// gaps. This is the "project provides defaults; carousel YAML wins
// when it speaks" rule.
//
// Returns the (potentially mutated) carousel for chaining.
func (p *ProjectConfig) ApplyToCarousel(c *Carousel) *Carousel {
	if p == nil || c == nil {
		return c
	}
	if c.Theme == "" && p.Theme != "" {
		c.Theme = p.Theme
	}
	if c.Handle == "" && p.Handle != "" {
		c.Handle = p.Handle
	}
	if c.Platform == "" && p.Platform != "" {
		c.Platform = p.Platform
	}
	return c
}

// migrateLegacyThemeDir checks whether the v0.1.x XDG-hardcoded path
// (`~/.config/social-carousel/themes/`) contains any user themes and the
// new `os.UserConfigDir()` path doesn't yet. Returns the legacy path so
// callers can either migrate or read from both.
//
// macOS is the only OS where the legacy path is wrong (it pointed at
// `~/.config/` instead of `~/Library/Application Support/`). Linux paths
// are identical because `os.UserConfigDir()` returns `~/.config/` there.
// Windows users on v0.1.x had no themes saved (the old code only worked
// on Unix), so there's nothing to migrate.
func legacyXDGThemesDir() (string, bool) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, appSlug, "themes"), true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	return filepath.Join(home, ".config", appSlug, "themes"), true
}

// SanitizeThemeName checks that a name is safe to use as a filename and
// doesn't try to escape the themes directory. Returns the cleaned name
// or an error for problematic inputs.
func SanitizeThemeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("theme name is empty")
	}
	if strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
		return "", fmt.Errorf("theme name %q contains path separators or '..'", name)
	}
	return name, nil
}
