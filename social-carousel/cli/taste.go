// taste.go — design-taste memory layer (v0.2.0).
//
// "Taste memory" is the accumulated set of design preferences the user has
// expressed over time. Unlike a Theme (structured YAML the renderer reads),
// taste rules are unstructured-but-machine-filterable markdown bullets that
// inform how the AGENT makes choices before scaffolding. Examples:
//
//	- Never use yellow or gold in palettes.
//	- Prefer thin font weights (300-400).
//	- For B2B posts use slate as primary.
//
// Storage uses the Cursor MDC pattern: YAML frontmatter + markdown body.
// Each rule is one bullet so diffs are clean and the user can edit in any
// markdown editor.
//
// Two layers, project wins on overlap:
//
//	~/.claude/skills/social-carousel/memory/taste.md   ← global, cross-project
//	<project>/carousel-design.md                       ← project-local, versionable
//
// Capture trigger is intentionally narrow: an AGENT-side decision based on
// corrective imperatives in user input ("never", "always", "prefer",
// "stop using", "from now on") OR an explicit `remember: <rule>` field in
// the carousel YAML brief. NEVER silently captured — the agent echoes
// before appending, per the gate-2 anti-hallucination guard.
//
// This file owns persistence; rule classification (generalizable vs
// brand-scoped vs one-off) is the agent's job, not the CLI's.

package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// GlobalTasteDirSuffix is appended to ~/.claude/skills/social-carousel/
	// when resolving the global taste file location. Inside the skill's
	// install directory so it persists across CLI updates (which overwrite
	// SKILL.md and reference/ but leave memory/ alone).
	GlobalTasteDirSuffix = "memory"

	// GlobalTasteFilename is the global taste file's basename.
	GlobalTasteFilename = "taste.md"

	// ProjectTasteFilename is the project-local taste file the agent looks
	// for alongside (or above) the carousel YAML. Distinct name from the
	// global file so `taste show` can report which layer a rule came from.
	ProjectTasteFilename = "carousel-design.md"
)

// Confidence levels for a taste rule. High = explicit corrective from the
// user ("never X"). Low = inferred by the agent from softer signals.
// Low-confidence rules expire after 30 days unless reinforced — protects
// against early-session noise becoming permanent.
type Confidence string

const (
	ConfidenceHigh Confidence = "high"
	ConfidenceLow  Confidence = "low"
)

// TasteScope distinguishes where the rule applies.
type TasteScope string

const (
	// ScopeGlobal applies to every carousel the user authors.
	ScopeGlobal TasteScope = "global"
	// ScopeBrand applies only when a specific brand context is active.
	// Encoded as "brand:<slug>" in the YAML; the slug is anything after
	// the colon. Used as a string match by the agent.
	ScopeBrand TasteScope = "brand"
)

// TasteRule is one captured preference. Stored as one bullet in the
// markdown body, with the frontmatter holding metadata. The frontmatter
// is a list — one entry per bullet — so a single file holds N rules.
type TasteRule struct {
	// Text is the user-facing rule sentence, e.g. "Never use yellow".
	// Renders as a markdown bullet. Required.
	Text string `yaml:"text"`

	// Scope is "global" OR "brand:<slug>". Default "global" when blank.
	Scope string `yaml:"scope,omitempty"`

	// Captured is the date (YYYY-MM-DD) the rule was first added. Used
	// for the 90-day hygiene review and the 30-day low-confidence expiry.
	Captured string `yaml:"captured,omitempty"`

	// Confidence is "high" (explicit) OR "low" (inferred). Default "high"
	// when blank — most rules ARE explicit; inferred rules must opt in to
	// low confidence so they get the auto-expiry treatment.
	Confidence Confidence `yaml:"confidence,omitempty"`
}

// TasteFile is the on-disk shape: a frontmatter list of rules + a markdown
// body that mirrors them as bullets (for human readability). The body is
// regenerated from frontmatter on every save — the frontmatter is the
// source of truth.
type TasteFile struct {
	Rules []TasteRule `yaml:"rules,omitempty"`

	// path is the absolute path on disk. Empty for newly-constructed
	// in-memory files.
	path string `yaml:"-"`
}

// Path returns where this taste file lives on disk. Empty for in-memory.
func (t *TasteFile) Path() string {
	if t == nil {
		return ""
	}
	return t.path
}

// GlobalTastePath returns the absolute path to the global taste file.
// Located inside the skill's install dir (under ~/.claude/skills/) rather
// than in the OS config dir so it travels with the skill rather than the
// user's general app config — themes belong to the user's environment,
// but taste belongs to the skill's brain.
//
// The dir is NOT created on disk; callers that write must MkdirAll.
func GlobalTastePath() (string, error) {
	skillDir, err := skillInstallDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(skillDir, GlobalTasteDirSuffix, GlobalTasteFilename), nil
}

// skillInstallDir returns ~/.claude/skills/social-carousel/, the directory
// the install script copies the skill into. Stable across self-updates.
func skillInstallDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".claude", "skills", appSlug), nil
}

// FindProjectTaste walks up from startDir looking for ProjectTasteFilename,
// applying the same .git/-boundary + 10-level cap rules as the project
// config walk-up. Returns the absolute path or "" if not found.
func FindProjectTaste(startDir string) (string, error) {
	if startDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get cwd: %w", err)
		}
		startDir = cwd
	}
	abs, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	dir := abs
	for level := 0; level < projectWalkUpMaxLevels; level++ {
		candidate := filepath.Join(dir, ProjectTasteFilename)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
		if info, err := os.Stat(filepath.Join(dir, ".git")); err == nil && info.IsDir() {
			return "", nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
	return "", nil
}

// LoadTasteFile reads a taste file from path. Returns an empty (non-nil)
// TasteFile if the path does not exist on disk — callers can immediately
// append + save without a separate "is this the first rule?" check.
func LoadTasteFile(path string) (*TasteFile, error) {
	t := &TasteFile{path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return t, nil
		}
		return nil, fmt.Errorf("read taste file %q: %w", path, err)
	}
	if err := parseTasteFile(data, t); err != nil {
		return nil, fmt.Errorf("parse taste file %q: %w", path, err)
	}
	return t, nil
}

// LoadEffectiveTaste returns the merged set of rules from (global + project),
// project rules following global so they appear last in the slice. The
// agent SHOULD let the later (project) rule win when text content overlaps,
// but the merging is intentionally not deduplicated by text — preserving
// both means the agent can see "global says X, project also says X" as a
// signal of consistency.
//
// Either layer may be empty. Returns ("", "") for the paths the rules came
// from (in resolution order) so callers can report sources to the user.
func LoadEffectiveTaste(projectStartDir string) (rules []TasteRule, sources []string, err error) {
	// Global first.
	if gPath, gErr := GlobalTastePath(); gErr == nil {
		if g, lErr := LoadTasteFile(gPath); lErr == nil && len(g.Rules) > 0 {
			rules = append(rules, g.Rules...)
			sources = append(sources, gPath)
		}
	}
	// Project on top (wins on overlap, lexically).
	if pPath, fErr := FindProjectTaste(projectStartDir); fErr == nil && pPath != "" {
		if p, lErr := LoadTasteFile(pPath); lErr == nil && len(p.Rules) > 0 {
			rules = append(rules, p.Rules...)
			sources = append(sources, pPath)
		}
	}
	return rules, sources, nil
}

// AppendRule adds a rule to the file and writes it back to disk. The rule's
// Captured field is auto-set if blank; Confidence defaults to high; Scope
// defaults to global.
//
// Idempotent on Text (case-insensitive trimmed match): re-adding the same
// rule text does NOT create a duplicate — it refreshes the existing rule's
// Captured timestamp instead. This is how a user "reinforces" a low-
// confidence rule into permanence.
func (t *TasteFile) AppendRule(r TasteRule) error {
	if strings.TrimSpace(r.Text) == "" {
		return fmt.Errorf("rule text is empty")
	}
	if r.Captured == "" {
		r.Captured = time.Now().UTC().Format("2006-01-02")
	}
	if r.Confidence == "" {
		r.Confidence = ConfidenceHigh
	}
	if r.Scope == "" {
		r.Scope = string(ScopeGlobal)
	}

	// Idempotent merge: refresh existing rule with the same text.
	key := normaliseRuleKey(r.Text)
	for i := range t.Rules {
		if normaliseRuleKey(t.Rules[i].Text) == key {
			t.Rules[i].Captured = r.Captured
			t.Rules[i].Confidence = r.Confidence
			t.Rules[i].Scope = r.Scope
			return t.save()
		}
	}

	t.Rules = append(t.Rules, r)
	return t.save()
}

// PruneExpired removes low-confidence rules older than the cutoff in days.
// Returns the number of rules removed. Default protection for early-
// session noise: 30 days for low-confidence is the recommendation; callers
// can override with whatever cutoff fits.
//
// High-confidence rules are never auto-pruned. They need the
// quarterly hygiene review (`taste review` command) for retirement.
func (t *TasteFile) PruneExpired(lowConfidenceDays int) (int, error) {
	if lowConfidenceDays <= 0 {
		return 0, nil
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -lowConfidenceDays)
	kept := t.Rules[:0]
	removed := 0
	for _, r := range t.Rules {
		if r.Confidence != ConfidenceLow {
			kept = append(kept, r)
			continue
		}
		captured, err := time.Parse("2006-01-02", r.Captured)
		if err != nil {
			// Unparseable date — keep the rule rather than delete blindly.
			kept = append(kept, r)
			continue
		}
		if captured.After(cutoff) {
			kept = append(kept, r)
			continue
		}
		removed++
	}
	t.Rules = kept
	if removed > 0 {
		return removed, t.save()
	}
	return 0, nil
}

// save persists the file as frontmatter + body. Body bullets are
// regenerated from frontmatter — frontmatter is the source of truth.
// Writes are atomic (write to tmp, rename) so a half-written file never
// remains on disk if the process is interrupted mid-save.
func (t *TasteFile) save() error {
	if t.path == "" {
		return fmt.Errorf("taste file has no path; cannot save")
	}
	if err := os.MkdirAll(filepath.Dir(t.path), 0o755); err != nil {
		return fmt.Errorf("create taste dir: %w", err)
	}

	content, err := renderTasteFile(t)
	if err != nil {
		return err
	}

	tmp := t.path + ".tmp"
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return fmt.Errorf("write tmp taste file: %w", err)
	}
	if err := os.Rename(tmp, t.path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename tmp taste file: %w", err)
	}
	return nil
}

// parseTasteFile reads frontmatter + (optional) markdown body. The body is
// IGNORED on read — frontmatter wins as source of truth, and the body is
// always regenerated on write. This keeps human edits to the body from
// being silently lost: users who want to add a rule should edit the
// frontmatter, not the body.
func parseTasteFile(data []byte, into *TasteFile) error {
	frontmatter, _ := splitFrontmatter(data)
	if len(frontmatter) == 0 {
		// File exists but no frontmatter — treat as empty rules.
		return nil
	}
	if err := yaml.Unmarshal(frontmatter, into); err != nil {
		return fmt.Errorf("unmarshal frontmatter: %w", err)
	}
	return nil
}

// splitFrontmatter extracts the YAML frontmatter block from a markdown file.
// Convention: file STARTS with `---\n`, frontmatter follows, terminated by
// `\n---\n`. Returns (frontmatter, body) bytes. If no frontmatter delimiter
// is found, returns (nil, full-content).
func splitFrontmatter(data []byte) (frontmatter, body []byte) {
	const delim = "---\n"
	if !bytes.HasPrefix(data, []byte(delim)) {
		return nil, data
	}
	rest := data[len(delim):]
	end := bytes.Index(rest, []byte("\n"+delim))
	if end < 0 {
		// Unterminated frontmatter — return all of rest as frontmatter so
		// the user sees the parse error rather than a silent body
		// reinterpretation.
		return rest, nil
	}
	return rest[:end+1], rest[end+1+len(delim):]
}

// renderTasteFile produces the on-disk bytes: frontmatter (YAML) + body
// (markdown bullets, one per rule, with metadata as italics for human
// readability).
func renderTasteFile(t *TasteFile) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("---\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(t); err != nil {
		return nil, fmt.Errorf("encode taste frontmatter: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("close encoder: %w", err)
	}
	buf.WriteString("---\n\n")

	// Body — markdown view. Regenerated from rules; manual edits to the
	// body are dropped on every save (frontmatter is source of truth).
	buf.WriteString("# Carousel design taste\n\n")
	buf.WriteString("Accumulated design preferences. Frontmatter above is the source of truth — edit there.\n\n")
	if len(t.Rules) == 0 {
		buf.WriteString("_No rules yet._\n")
	} else {
		for _, r := range t.Rules {
			fmt.Fprintf(&buf, "- %s", r.Text)
			meta := []string{}
			if r.Scope != "" && r.Scope != string(ScopeGlobal) {
				meta = append(meta, "scope: "+r.Scope)
			}
			if r.Confidence != "" && r.Confidence != ConfidenceHigh {
				meta = append(meta, "confidence: "+string(r.Confidence))
			}
			if r.Captured != "" {
				meta = append(meta, "captured: "+r.Captured)
			}
			if len(meta) > 0 {
				fmt.Fprintf(&buf, "  _(%s)_", strings.Join(meta, "; "))
			}
			buf.WriteString("\n")
		}
	}
	return buf.Bytes(), nil
}

// normaliseRuleKey lowercases + trims + collapses whitespace so case and
// trailing-period variants of the same rule deduplicate.
func normaliseRuleKey(text string) string {
	s := strings.ToLower(strings.TrimSpace(text))
	s = strings.TrimRight(s, ".!?")
	return strings.Join(strings.Fields(s), " ")
}
