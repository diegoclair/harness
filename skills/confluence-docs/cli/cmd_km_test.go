package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/diegoclair/harness/pkg/atlassian/setup"
)

// ── TestKMClassifyFase ────────────────────────────────────────────────────

func TestKMClassifyFase(t *testing.T) {
	cases := []struct {
		name          string
		anomalia      string
		rawTags       []string
		wantAddFase   bool
		wantRealAnoma bool // whether realAnomaly should be non-empty
		wantFaseTag   bool // whether "fase-final-checkout-universal" ends up in cleanedTags
		wantTagLen    int  // expected number of cleaned tags
	}{
		{
			name:        "empty anomalia + clean tags",
			anomalia:    "",
			rawTags:     []string{"advisor"},
			wantAddFase: false, wantRealAnoma: false, wantFaseTag: false,
			wantTagLen: 1,
		},
		{
			name:        "anomalia with obsoleto keyword",
			anomalia:    "conteudo obsoleto do pre-pivot",
			rawTags:     []string{"fase-mvp"},
			wantAddFase: true, wantRealAnoma: false, wantFaseTag: true,
			wantTagLen: 2, // fase-mvp + fase-final-checkout-universal
		},
		{
			name:        "anomalia with b2b2c keyword",
			anomalia:    "pagina do produto b2b2c checkout universal",
			rawTags:     []string{},
			wantAddFase: true, wantRealAnoma: false, wantFaseTag: true,
			wantTagLen: 1,
		},
		{
			name:        "anomalia with borderline — real anomaly",
			anomalia:    "borderline entre decision e explanation",
			rawTags:     []string{},
			wantAddFase: false, wantRealAnoma: true, wantFaseTag: false,
			wantTagLen: 0,
		},
		{
			name:        "anomalia with duplicata — real anomaly",
			anomalia:    "possivel duplicata de pagina existente",
			rawTags:     []string{},
			wantAddFase: false, wantRealAnoma: true, wantFaseTag: false,
			wantTagLen: 0,
		},
		{
			name:        "anomalia with nome-desatualizado — real anomaly",
			anomalia:    "nome-desatualizado: pagina renomeada no produto novo",
			rawTags:     []string{},
			wantAddFase: false, wantRealAnoma: true, wantFaseTag: false,
			wantTagLen: 0,
		},
		{
			name:        "pejorative tag legacy",
			anomalia:    "",
			rawTags:     []string{"legacy-product", "bmc"},
			wantAddFase: true, wantRealAnoma: false, wantFaseTag: true,
			// "legacy-product" removed, "bmc" kept, "fase-final-checkout-universal" added
			wantTagLen: 2,
		},
		{
			name:        "pejorative tag pre-pivot",
			anomalia:    "",
			rawTags:     []string{"pre-pivot-checkout", "estrategia"},
			wantAddFase: true, wantRealAnoma: false, wantFaseTag: true,
			wantTagLen: 2, // estrategia + fase-final-checkout-universal
		},
		{
			name:        "canonical fase tag already present — no duplicate",
			anomalia:    "",
			rawTags:     []string{"fase-final-checkout-universal", "bmc"},
			wantAddFase: true, wantRealAnoma: false, wantFaseTag: true,
			wantTagLen: 2, // fase-final-checkout-universal + bmc (no dup)
		},
		{
			name:        "anomalia and real anomaly both — both detected",
			anomalia:    "conteudo pre-pivot e borderline com capture",
			rawTags:     []string{},
			wantAddFase: true, wantRealAnoma: true, wantFaseTag: true,
			wantTagLen: 1,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			addFase, cleanedTags, realAnomaly := classifyFase(c.anomalia, c.rawTags)
			if addFase != c.wantAddFase {
				t.Errorf("addFase = %v, want %v", addFase, c.wantAddFase)
			}
			if (realAnomaly != "") != c.wantRealAnoma {
				t.Errorf("realAnomaly = %q, wantNonEmpty = %v", realAnomaly, c.wantRealAnoma)
			}
			hasFase := false
			for _, tag := range cleanedTags {
				if tag == "fase-final-checkout-universal" {
					hasFase = true
					break
				}
			}
			if hasFase != c.wantFaseTag {
				t.Errorf("hasFaseTag = %v (tags %v), want %v", hasFase, cleanedTags, c.wantFaseTag)
			}
			if len(cleanedTags) != c.wantTagLen {
				t.Errorf("len(cleanedTags) = %d (tags %v), want %d", len(cleanedTags), cleanedTags, c.wantTagLen)
			}
		})
	}
}

// ── TestKMRender ─────────────────────────────────────────────────────────

func TestKMRender_SmallInput(t *testing.T) {
	pages := map[string]kmPage{
		"111": {ID: "111", Title: "Sobre a Lybel", Tipo: "reference", Tags: []string{}},
		"222": {ID: "222", Title: "Proposta fit HOJE", Tipo: "decision", Tags: []string{"fase-mvp"}},
		"333": {ID: "333", Title: "Growth Strategy", Tipo: "explanation", Tags: []string{}},
		"444": {ID: "444", Title: "Como fazer deploy", Tipo: "how-to", Tags: []string{}},
		"555": {ID: "555", Title: "Spike AbacatePay", Tipo: "capture", Tags: []string{"fase-mvp"}},
	}

	md := renderKMMD(pages, testPagesBase, testSpaceOverview)

	// Must contain the frontmatter block.
	if !strings.Contains(md, ":::properties") {
		t.Error("expected :::properties block")
	}
	// Must contain the rules section as own h2 (with info panel inside).
	if !strings.Contains(md, "## Rules for AI") {
		t.Error("expected '## Rules for AI' h2 section")
	}
	if !strings.Contains(md, ":::info Required sequence") {
		t.Error("expected ':::info Required sequence' panel inside Rules for AI")
	}
	// Should NOT contain :::expand for sections with ≤12 entries.
	if strings.Contains(md, ":::expand Show all") {
		t.Error("should not have :::expand for small tipo sections (≤12 entries)")
	}
	// Page entries must appear.
	if !strings.Contains(md, "Sobre a Lybel") {
		t.Error("expected 'Sobre a Lybel' in output")
	}
	if !strings.Contains(md, "fase-mvp") {
		t.Error("expected fase-mvp tag in output")
	}
}

func TestKMRender_ExpandForLargeSection(t *testing.T) {
	// Build >12 pages for the "reference" tipo to trigger :::expand.
	pages := make(map[string]kmPage)
	for i := 0; i < 14; i++ {
		id := fmt.Sprintf("%d", 100+i)
		pages[id] = kmPage{ID: id, Title: fmt.Sprintf("Ref Page %d", i), Tipo: "reference"}
	}
	// A couple of others to avoid empty sections.
	pages["999"] = kmPage{ID: "999", Title: "Decision A", Tipo: "decision"}

	md := renderKMMD(pages, testPagesBase, testSpaceOverview)

	if !strings.Contains(md, ":::expand Show all 14 pages") {
		t.Error("expected :::expand block for reference section with 14 items")
	}
}

func TestKMRender_RealAnomaliesSection(t *testing.T) {
	pages := map[string]kmPage{
		"111": {ID: "111", Title: "Borderline Doc", Tipo: "explanation",
			RealAnomaly: "borderline entre explanation e decision"},
		"222": {ID: "222", Title: "Normal Doc", Tipo: "reference"},
	}

	md := renderKMMD(pages, testPagesBase, testSpaceOverview)

	// :::expand anomalies block must appear.
	if !strings.Contains(md, ":::expand 1 pages with real anomaly for review") {
		t.Error("expected anomaly expand block")
	}
	if !strings.Contains(md, "Borderline Doc") {
		t.Error("expected anomaly page listed in expand block")
	}
	// Normal doc must NOT appear in anomaly section.
	lines := strings.Split(md, "\n")
	inAnomaly := false
	for _, l := range lines {
		if strings.Contains(l, ":::expand") && strings.Contains(l, "anomalia") {
			inAnomaly = true
		}
		if inAnomaly && strings.Contains(l, "Normal Doc") {
			t.Error("Normal Doc should not appear in anomaly section")
		}
		if inAnomaly && l == ":::" {
			break
		}
	}
}

func TestKMRender_NoAnomalies(t *testing.T) {
	pages := map[string]kmPage{
		"111": {ID: "111", Title: "Clean Page", Tipo: "reference"},
	}
	md := renderKMMD(pages, testPagesBase, testSpaceOverview)

	if !strings.Contains(md, "_No real anomalies flagged._") {
		t.Error("expected '_No real anomalies flagged._' when no anomalies")
	}
}

// ── TestKMLoadTriage ──────────────────────────────────────────────────────

func TestKMLoadTriage_ValidBatch(t *testing.T) {
	dir := t.TempDir()

	entries := []triageEntry{
		{
			PageID:        "185303042",
			Title:         "Sobre a Lybel",
			TipoProposto:  "reference",
			Confidence:    "high",
			TagsSugeridas: []string{"lybel"},
			Rationale:     "pagina descritiva",
			Anomalia:      nil,
		},
		{
			PageID:        "187695141",
			Title:         "Proposta fit HOJE",
			TipoProposto:  "decision",
			Confidence:    "high",
			TagsSugeridas: []string{"fase-mvp"},
			Rationale:     "decisao de produto",
			Anomalia:      strPtr("borderline com explanation"),
		},
	}

	data, _ := json.Marshal(entries)
	batchPath := filepath.Join(dir, "batch-1.json")
	if err := os.WriteFile(batchPath, data, 0644); err != nil {
		t.Fatalf("writing batch: %v", err)
	}

	loaded, err := loadTriageDir(dir)
	if err != nil {
		t.Fatalf("loadTriageDir: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("want 2 entries, got %d", len(loaded))
	}
	if loaded[0].PageID != "185303042" {
		t.Errorf("entry 0 pageId = %q, want 185303042", loaded[0].PageID)
	}
	if loaded[1].Anomalia == nil || *loaded[1].Anomalia != "borderline com explanation" {
		t.Errorf("entry 1 anomalia = %v, want 'borderline com explanation'", loaded[1].Anomalia)
	}
}

func TestKMLoadTriage_MultipleBatchesMerged(t *testing.T) {
	dir := t.TempDir()

	writeJSON := func(name string, entries []triageEntry) {
		data, _ := json.Marshal(entries)
		os.WriteFile(filepath.Join(dir, name), data, 0644)
	}

	writeJSON("batch-1.json", []triageEntry{{PageID: "1", Title: "A", TipoProposto: "reference"}})
	writeJSON("batch-2.json", []triageEntry{{PageID: "2", Title: "B", TipoProposto: "decision"}})
	writeJSON("batch-3.json", []triageEntry{{PageID: "3", Title: "C", TipoProposto: "explanation"}})

	loaded, err := loadTriageDir(dir)
	if err != nil {
		t.Fatalf("loadTriageDir: %v", err)
	}
	if len(loaded) != 3 {
		t.Fatalf("want 3 entries total, got %d", len(loaded))
	}
}

func TestKMLoadTriage_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	loaded, err := loadTriageDir(dir)
	if err != nil {
		t.Fatalf("expected no error for empty dir, got: %v", err)
	}
	if len(loaded) != 0 {
		t.Errorf("expected empty slice, got %d entries", len(loaded))
	}
}

func TestKMLoadTriage_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "batch-1.json"), []byte("{invalid json"), 0644)
	_, err := loadTriageDir(dir)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

// ── TestKMMergePages ──────────────────────────────────────────────────────

func TestKMMergePages_BaselineOverridesTriage(t *testing.T) {
	bl := baseline{
		Pages: []baselineEntry{
			{PageID: "111", Title: "Baseline Title", Tipo: "reference", Tags: []string{"bmc"}},
		},
	}
	triage := []triageEntry{
		{PageID: "111", Title: "Triage Title", TipoProposto: "capture", TagsSugeridas: []string{}},
	}

	pages := mergePages(bl, triage)
	p := pages["111"]

	if p.Title != "Baseline Title" {
		t.Errorf("baseline title should not be overridden by triage, got %q", p.Title)
	}
	if p.Tipo != "reference" {
		t.Errorf("baseline tipo should not be overridden by triage, got %q", p.Tipo)
	}
}

func TestKMMergePages_TriageOnlyPage(t *testing.T) {
	anomalia := "conteudo obsoleto pos-pivot"
	bl := baseline{}
	triage := []triageEntry{
		{
			PageID:        "999",
			Title:         "B2B2C Legacy Page",
			TipoProposto:  "reference",
			TagsSugeridas: []string{"legacy-b2b2c"},
			Anomalia:      &anomalia,
		},
	}

	pages := mergePages(bl, triage)
	p := pages["999"]

	hasFase := false
	for _, tag := range p.Tags {
		if tag == "fase-final-checkout-universal" {
			hasFase = true
		}
		// Pejorative tags must not appear.
		if strings.Contains(strings.ToLower(tag), "legacy") || strings.Contains(strings.ToLower(tag), "obsoleto") {
			t.Errorf("pejorative tag should be cleaned: %q", tag)
		}
	}
	if !hasFase {
		t.Errorf("expected fase-final-checkout-universal tag, got tags %v", p.Tags)
	}
	// anomalia was "obsoleto" keyword only — NOT a real anomaly.
	if p.RealAnomaly != "" {
		t.Errorf("obsoleto in anomalia should NOT become real anomaly, got %q", p.RealAnomaly)
	}
}

func TestKMMergePages_FaseTagPropagatedToBaseline(t *testing.T) {
	bl := baseline{
		Pages: []baselineEntry{
			{PageID: "123", Title: "Old Checkout", Tipo: "explanation", Tags: []string{}},
		},
	}
	anomalia := "conteudo pre-pivot b2b2c"
	triage := []triageEntry{
		{PageID: "123", TipoProposto: "explanation", Anomalia: &anomalia},
	}

	pages := mergePages(bl, triage)
	p := pages["123"]

	hasFase := false
	for _, tag := range p.Tags {
		if tag == "fase-final-checkout-universal" {
			hasFase = true
		}
	}
	if !hasFase {
		t.Errorf("expected fase-final-checkout-universal propagated to baseline entry, got tags %v", p.Tags)
	}
}

// ── CLI integration tests ─────────────────────────────────────────────────

func TestKMGenerate_Help(t *testing.T) {
	out, _, code := runCLI(t, "km", "--help")
	if code != 0 {
		t.Fatalf("want exit 0, got %d", code)
	}
	if !strings.Contains(out, "generate") {
		t.Errorf("expected 'generate' in help, got %q", out)
	}
}

func TestKMGenerate_NoInput_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	out, _, code := runCLI(t, "km", "generate", "--input", dir)
	if code != 0 {
		t.Fatalf("want exit 0, got %d", code)
	}
	// With no pages, should still render a valid document.
	if !strings.Contains(out, ":::properties") {
		t.Errorf("expected :::properties in output")
	}
	if !strings.Contains(out, "## Rules for AI") {
		t.Errorf("expected '## Rules for AI' h2 section")
	}
}

func TestKMGenerate_WithBatch(t *testing.T) {
	dir := t.TempDir()
	entries := []triageEntry{
		{PageID: "185303042", Title: "Sobre a Lybel", TipoProposto: "reference",
			TagsSugeridas: []string{}, Confidence: "high"},
		{PageID: "187695141", Title: "Proposta fit", TipoProposto: "decision",
			TagsSugeridas: []string{"fase-mvp"}, Confidence: "high"},
	}
	data, _ := json.Marshal(entries)
	os.WriteFile(filepath.Join(dir, "batch-1.json"), data, 0644)

	out, _, code := runCLI(t, "km", "generate", "--input", dir)
	if code != 0 {
		t.Fatalf("want exit 0, got %d", code)
	}
	if !strings.Contains(out, "Sobre a Lybel") {
		t.Errorf("expected 'Sobre a Lybel' in output")
	}
	if !strings.Contains(out, "fase-mvp") {
		t.Errorf("expected fase-mvp tag in output")
	}
}

func TestKMGenerate_WithBaseline(t *testing.T) {
	dir := t.TempDir()
	triageDir := t.TempDir()

	bl := baseline{
		Pages: []baselineEntry{
			{PageID: "111", Title: "Sobre a Lybel", Tipo: "reference", Tags: []string{}},
			{PageID: "222", Title: "Princípios", Tipo: "decision", Tags: []string{}},
		},
	}
	blData, _ := json.Marshal(bl)
	blPath := filepath.Join(dir, "baseline.json")
	os.WriteFile(blPath, blData, 0644)

	out, _, code := runCLI(t, "km", "generate", "--input", triageDir, "--baseline", blPath)
	if code != 0 {
		t.Fatalf("want exit 0, got %d", code)
	}
	if !strings.Contains(out, "Sobre a Lybel") {
		t.Errorf("expected 'Sobre a Lybel' in output")
	}
	if !strings.Contains(out, "Princípios") {
		t.Errorf("expected 'Princípios' in output")
	}
}

func TestKMGenerate_OutputToFile(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "km.md")

	_, _, code := runCLI(t, "km", "generate", "--input", dir, "--output", outFile)
	if code != 0 {
		t.Fatalf("want exit 0, got %d", code)
	}
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	if !strings.Contains(string(data), ":::properties") {
		t.Errorf("expected :::properties in written file")
	}
}

func TestKMGenerate_UnknownFlag(t *testing.T) {
	_, errOut, code := runCLI(t, "km", "generate", "--bogus")
	if code == 0 {
		t.Fatal("expected non-zero exit for unknown flag")
	}
	if !strings.Contains(errOut, "unknown flag") {
		t.Errorf("expected 'unknown flag' in error: %s", errOut)
	}
}

func TestKMClassify_NotImplemented(t *testing.T) {
	_, errOut, code := runCLI(t, "km", "classify", "--page-id", "123")
	if code == 0 {
		t.Fatal("expected non-zero exit for unimplemented classify")
	}
	if !strings.Contains(errOut, "not implemented") {
		t.Errorf("expected 'not implemented' in error: %s", errOut)
	}
}

func TestKMUnknownSubcommand(t *testing.T) {
	_, errOut, code := runCLI(t, "km", "bogus")
	if code == 0 {
		t.Fatal("expected non-zero exit for unknown subcommand")
	}
	if !strings.Contains(errOut, "unknown subcommand") {
		t.Errorf("expected 'unknown subcommand' in error: %s", errOut)
	}
}

// ── helpers ────────────────────────────────────────────────────────────────

func strPtr(s string) *string { return &s }

// A configured space yields links; without one the map must degrade to plain
// ids rather than pointing at somebody else's instance.
const (
	testPagesBase     = "https://example.atlassian.net/wiki/spaces/ENG/pages/"
	testSpaceOverview = "https://example.atlassian.net/wiki/spaces/ENG/overview"
)

// kmSample exercises every branch that emits a link: the plain list, the
// >12-item expand block, and the anomalies section.
func kmSample() map[string]kmPage {
	pages := map[string]kmPage{}
	for i := 1; i <= 14; i++ {
		id := strconv.Itoa(i)
		pages[id] = kmPage{ID: id, Title: "Page " + id, Tipo: "reference"}
	}
	pages["99"] = kmPage{ID: "99", Title: "Odd one", Tipo: "reference", RealAnomaly: "suspected duplicate"}
	return pages
}

// Asserting only the absence of a hostname is not enough: a broken guard would
// emit a relative link like `](1)`, which is still a wrong link.
func TestUnconfiguredRenderEmitsNoMarkdownLinkAtAll(t *testing.T) {
	md := renderKMMD(kmSample(), "", "")

	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "- ") && strings.Contains(line, "](") {
			t.Errorf("unconfigured render must not link anywhere, got: %s", line)
		}
	}
	if !strings.Contains(md, "Page 1 `(1)`") {
		t.Errorf("pages must still be listed, just unlinked:\n%s", md)
	}
}

// Every link-emitting branch must use the resolved base, not just the first one.
func TestConfiguredRenderLinksEverySection(t *testing.T) {
	md := renderKMMD(kmSample(), testPagesBase, testSpaceOverview)

	for _, id := range []string{"1", "14", "99"} {
		if !strings.Contains(md, "]("+testPagesBase+id+")") {
			t.Errorf("page %s is not linked to the configured instance:\n%s", id, md)
		}
	}
	if !strings.Contains(md, testSpaceOverview) {
		t.Errorf("the space overview link is missing:\n%s", md)
	}
}

// The whole point of the change: the instance comes from configuration. A
// swapped or truncated base would still "look like" a URL.
func TestKMLinksDerivesFromConfiguration(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("ATLASSIAN_CLOUD", "")
	writeTestConfig(t, dir, setup.Config{Cloud: "acme", SpaceKey: "ENG"})

	pagesBase, overview := kmLinks("")
	if pagesBase != "https://acme.atlassian.net/wiki/spaces/ENG/pages/" {
		t.Errorf("pagesBase = %q", pagesBase)
	}
	if overview != "https://acme.atlassian.net/wiki/spaces/ENG/overview" {
		t.Errorf("overview = %q", overview)
	}
}

// `login` stores the site in the credential store and never writes cloud= to
// the config, so resolving from the config alone drops every link for the
// recommended auth mode.
func TestKMLinksFallsBackToTheStoredSite(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("ATLASSIAN_CLOUD", "")
	writeTestConfig(t, dir, setup.Config{SpaceKey: "ENG"}) // no cloud=, as login leaves it
	writeOAuthSite(t, dir, "acme")

	pagesBase, _ := kmLinks("")
	if pagesBase != "https://acme.atlassian.net/wiki/spaces/ENG/pages/" {
		t.Errorf("pagesBase = %q, want the site from the credential store", pagesBase)
	}
}

func TestKMLinksPrefersAnExplicitOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("ATLASSIAN_CLOUD", "")
	writeTestConfig(t, dir, setup.Config{Cloud: "fromconfig", SpaceKey: "ENG"})

	pagesBase, _ := kmLinks("fromflag")
	if !strings.HasPrefix(pagesBase, "https://fromflag.atlassian.net/") {
		t.Errorf("pagesBase = %q, want the override to win", pagesBase)
	}
}

func TestKMLinksReturnsNothingWhenUnconfigured(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("ATLASSIAN_CLOUD", "")

	pagesBase, overview := kmLinks("")
	if pagesBase != "" || overview != "" {
		t.Errorf("unconfigured should yield empty links, got %q / %q", pagesBase, overview)
	}
}

// A space with no known instance must not produce a half-built URL.
func TestKMLinksNeedsBothInstanceAndSpace(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("ATLASSIAN_CLOUD", "")
	writeTestConfig(t, dir, setup.Config{Cloud: "acme"}) // instance known, space not

	if pagesBase, _ := kmLinks(""); pagesBase != "" {
		t.Errorf("pagesBase = %q, want empty without a space", pagesBase)
	}
}

// login stores the site in the shared credential store, not in the CLI config.
func writeOAuthSite(t *testing.T, home, site string) {
	t.Helper()
	dir := filepath.Join(home, "atlassian")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	body := "auth_mode=oauth\noauth_site=" + site + "\n"
	if err := os.WriteFile(filepath.Join(dir, "credentials"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// Deleting the wiring between the resolved configuration and the renderer
// would leave every unit test green while the command emitted an unlinked map.
func TestKMGenerateWiresTheResolvedInstanceIntoTheOutput(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("ATLASSIAN_CLOUD", "")
	writeTestConfig(t, home, setup.Config{Cloud: "acme", SpaceKey: "ENG"})

	input := t.TempDir()
	batch := `[{"pageId":"777","title":"A page","tipo_proposto":"reference","anomalia":null}]`
	if err := os.WriteFile(filepath.Join(input, "batch-1.json"), []byte(batch), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "km.md")

	var stdout, stderr bytes.Buffer
	code, err := runKMGenerate([]string{"--input", input, "--output", out}, &stdout, &stderr)
	if err != nil || code != exitOK {
		t.Fatalf("km generate: code=%d err=%v stderr=%s", code, err, stderr.String())
	}

	md, rerr := os.ReadFile(out)
	if rerr != nil {
		t.Fatal(rerr)
	}
	want := "https://acme.atlassian.net/wiki/spaces/ENG/pages/777"
	if !strings.Contains(string(md), want) {
		t.Errorf("the generated map does not link to the configured instance (%s):\n%s", want, md)
	}
}
