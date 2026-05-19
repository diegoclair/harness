package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// AppendRule — basic add, defaults, idempotence
// ---------------------------------------------------------------------------

func TestAppendRule_AddsWithDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "taste.md")
	tf, err := LoadTasteFile(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := tf.AppendRule(TasteRule{Text: "Never use yellow"}); err != nil {
		t.Fatalf("append: %v", err)
	}
	if len(tf.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(tf.Rules))
	}
	r := tf.Rules[0]
	if r.Confidence != ConfidenceHigh {
		t.Errorf("expected default confidence=high, got %q", r.Confidence)
	}
	if r.Scope != string(ScopeGlobal) {
		t.Errorf("expected default scope=global, got %q", r.Scope)
	}
	if r.Captured == "" {
		t.Error("expected captured date to be auto-set")
	}
}

func TestAppendRule_RejectsEmpty(t *testing.T) {
	tf := &TasteFile{path: filepath.Join(t.TempDir(), "taste.md")}
	if err := tf.AppendRule(TasteRule{Text: "   "}); err == nil {
		t.Error("expected error on whitespace-only rule text")
	}
}

func TestAppendRule_IdempotentByText(t *testing.T) {
	path := filepath.Join(t.TempDir(), "taste.md")
	tf, _ := LoadTasteFile(path)
	_ = tf.AppendRule(TasteRule{Text: "Never use yellow", Captured: "2025-01-01"})
	_ = tf.AppendRule(TasteRule{Text: "NEVER use YELLOW.", Captured: "2026-05-19"})
	if len(tf.Rules) != 1 {
		t.Fatalf("expected idempotent merge (1 rule), got %d", len(tf.Rules))
	}
	if tf.Rules[0].Captured != "2026-05-19" {
		t.Errorf("expected Captured refreshed to latest, got %q", tf.Rules[0].Captured)
	}
}

// ---------------------------------------------------------------------------
// Persistence — save + reload preserves rules
// ---------------------------------------------------------------------------

func TestSaveAndReload_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "taste.md")
	tf, _ := LoadTasteFile(path)
	rules := []TasteRule{
		{Text: "Never use yellow"},
		{Text: "Prefer mint accent", Scope: "brand:lybel"},
		{Text: "Default to 8 slides", Confidence: ConfidenceLow},
	}
	for _, r := range rules {
		if err := tf.AppendRule(r); err != nil {
			t.Fatalf("append %q: %v", r.Text, err)
		}
	}

	reloaded, err := LoadTasteFile(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(reloaded.Rules) != 3 {
		t.Fatalf("expected 3 rules after reload, got %d", len(reloaded.Rules))
	}
	// Spot-check that scope + confidence survived.
	var brandRule, lowRule *TasteRule
	for i := range reloaded.Rules {
		switch reloaded.Rules[i].Text {
		case "Prefer mint accent":
			brandRule = &reloaded.Rules[i]
		case "Default to 8 slides":
			lowRule = &reloaded.Rules[i]
		}
	}
	if brandRule == nil || brandRule.Scope != "brand:lybel" {
		t.Errorf("brand-scoped rule did not round-trip: %+v", brandRule)
	}
	if lowRule == nil || lowRule.Confidence != ConfidenceLow {
		t.Errorf("low-confidence rule did not round-trip: %+v", lowRule)
	}
}

func TestSave_AtomicNoLeftoverTmp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "taste.md")
	tf, _ := LoadTasteFile(path)
	_ = tf.AppendRule(TasteRule{Text: "x"})
	// .tmp should not remain after successful save.
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("expected no leftover %s.tmp", path)
	}
}

// ---------------------------------------------------------------------------
// PruneExpired — 30-day cutoff on low-confidence only
// ---------------------------------------------------------------------------

func TestPruneExpired_DropsOldLowConfidence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "taste.md")
	tf, _ := LoadTasteFile(path)
	oldDate := time.Now().UTC().AddDate(0, 0, -60).Format("2006-01-02")
	freshDate := time.Now().UTC().AddDate(0, 0, -5).Format("2006-01-02")
	_ = tf.AppendRule(TasteRule{Text: "old low", Confidence: ConfidenceLow, Captured: oldDate})
	_ = tf.AppendRule(TasteRule{Text: "fresh low", Confidence: ConfidenceLow, Captured: freshDate})
	_ = tf.AppendRule(TasteRule{Text: "old high", Confidence: ConfidenceHigh, Captured: oldDate})

	removed, err := tf.PruneExpired(30)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if removed != 1 {
		t.Errorf("expected 1 removal (old low only), got %d", removed)
	}
	if len(tf.Rules) != 2 {
		t.Errorf("expected 2 remaining rules, got %d", len(tf.Rules))
	}
	for _, r := range tf.Rules {
		if r.Text == "old low" {
			t.Error("'old low' should have been pruned")
		}
	}
}

func TestPruneExpired_NoOp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "taste.md")
	tf, _ := LoadTasteFile(path)
	_ = tf.AppendRule(TasteRule{Text: "x"})
	removed, err := tf.PruneExpired(0)
	if err != nil || removed != 0 {
		t.Errorf("expected no-op for cutoff=0, got removed=%d err=%v", removed, err)
	}
}

// ---------------------------------------------------------------------------
// LoadEffectiveTaste — global + project merge, project last
// ---------------------------------------------------------------------------

func TestLoadEffectiveTaste_GlobalAndProject(t *testing.T) {
	// Set HOME to a temp dir so GlobalTastePath resolves under it.
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome) //nolint:errcheck
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome) //nolint:errcheck

	// Seed global taste.
	globalPath, err := GlobalTastePath()
	if err != nil {
		t.Fatalf("GlobalTastePath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(globalPath), 0o755); err != nil {
		t.Fatalf("mkdir global: %v", err)
	}
	gtf, _ := LoadTasteFile(globalPath)
	_ = gtf.AppendRule(TasteRule{Text: "Global rule A"})

	// Seed project taste.
	projectDir := t.TempDir()
	projectPath := filepath.Join(projectDir, ProjectTasteFilename)
	ptf, _ := LoadTasteFile(projectPath)
	_ = ptf.AppendRule(TasteRule{Text: "Project rule B"})

	rules, sources, err := LoadEffectiveTaste(projectDir)
	if err != nil {
		t.Fatalf("LoadEffectiveTaste: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules merged, got %d", len(rules))
	}
	if rules[0].Text != "Global rule A" || rules[1].Text != "Project rule B" {
		t.Errorf("expected global-then-project order, got %v", rules)
	}
	if len(sources) != 2 {
		t.Errorf("expected 2 sources, got %d", len(sources))
	}
}

// ---------------------------------------------------------------------------
// Frontmatter parsing — robustness
// ---------------------------------------------------------------------------

func TestParseTasteFile_EmptyBody(t *testing.T) {
	tf := &TasteFile{}
	err := parseTasteFile([]byte("---\nrules:\n  - text: foo\n---\n"), tf)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(tf.Rules) != 1 || tf.Rules[0].Text != "foo" {
		t.Errorf("expected 1 rule 'foo', got %+v", tf.Rules)
	}
}

func TestParseTasteFile_NoFrontmatter(t *testing.T) {
	tf := &TasteFile{}
	err := parseTasteFile([]byte("just markdown, no fm\n"), tf)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(tf.Rules) != 0 {
		t.Errorf("expected 0 rules when no fm present, got %d", len(tf.Rules))
	}
}

func TestRenderTasteFile_BodyMirrorsFrontmatter(t *testing.T) {
	tf := &TasteFile{
		Rules: []TasteRule{
			{Text: "Never yellow", Captured: "2026-05-19", Confidence: ConfidenceHigh, Scope: "global"},
			{Text: "Brand mint", Captured: "2026-05-19", Confidence: ConfidenceLow, Scope: "brand:lybel"},
		},
	}
	data, err := renderTasteFile(tf)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "---\n") {
		t.Error("expected frontmatter delimiters")
	}
	if !strings.Contains(s, "- Never yellow") {
		t.Error("expected first rule as body bullet")
	}
	if !strings.Contains(s, "scope: brand:lybel") {
		t.Error("expected scope annotation in body for branded rule")
	}
	if !strings.Contains(s, "confidence: low") {
		t.Error("expected confidence annotation in body for low-confidence rule")
	}
}

// ---------------------------------------------------------------------------
// FindProjectTaste — walk-up symmetry with FindProjectConfig
// ---------------------------------------------------------------------------

func TestFindProjectTaste_WalksUp(t *testing.T) {
	root := makeTree(t, map[string]string{
		ProjectTasteFilename: "---\nrules: []\n---\n",
		"sub/main.go":        "package main\n",
	})
	got, err := FindProjectTaste(filepath.Join(root, "sub"))
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	want := filepath.Join(root, ProjectTasteFilename)
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestFindProjectTaste_StopsAtGit(t *testing.T) {
	root := makeTree(t, map[string]string{
		ProjectTasteFilename: "---\nrules: []\n---\n",
		"inner/.git":         "",
		"inner/sub/main.go":  "package main\n",
	})
	got, err := FindProjectTaste(filepath.Join(root, "inner", "sub"))
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty (.git boundary), got %q", got)
	}
}
