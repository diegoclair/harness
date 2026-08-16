package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadProbe(t *testing.T, root string) *Config {
	t.Helper()
	cfg, err := loadConfig(root)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	return cfg
}

func runProbe(t *testing.T, root string) *checkResult {
	t.Helper()
	res, err := runCheck(loadProbe(t, root), checkOptions{all: true})
	if err != nil {
		t.Fatalf("runCheck: %v", err)
	}
	return res
}

func hits(res *checkResult) map[string]bool {
	out := map[string]bool{}
	for _, f := range res.Findings {
		out[fmt.Sprintf("%s@%s:%d", f.Rule, f.File, f.Line)] = true
		out[f.Rule] = true
	}
	return out
}

// The fixture carries one deliberate violation per rule, so a rule that stops
// firing fails here instead of going quietly mute in a repo.
func TestProbeFiresEveryRule(t *testing.T) {
	got := hits(runProbe(t, "testdata/probe"))

	for _, want := range []string{
		"CMT-01@service/verbose.go:3",
		"CMT-02@service/probe.go:19",
		"CMT-04@service/probe.go:21",
		"CMT-05@service/probe.go:23",
		"CMT-06@service/probe.go:23",
		"CMT-07@service/probe.go:28",
		"CMT-09@service/probe.go:8",
		"CPX-02@service/verbose.go:10",
		"CPX-04@service/verbose.go:10",
		"DUP-01@data/first.go:5",
		"ARC-01@domain/rule.go:3",
		"ARC-03@transport/handler.go:4",
		"ARC-06@transport/handler.go:10",
		"ARC-07@data/logs.go:4",
		"ARC-13@transport/router.go:5",
		"ARC-13@transport/router.go:9",
	} {
		if !got[want] {
			t.Errorf("expected finding %s, missing", want)
		}
	}
}

// A rule that fires on everything is as useless as one that fires on nothing.
func TestProbeLeavesGoodCommentsAlone(t *testing.T) {
	res := runProbe(t, "testdata/probe")
	for _, f := range res.Findings {
		switch {
		case f.File == "service/probe.go" && f.Line == 10:
			t.Errorf("CMT-09 fired on a field comment carrying a unit and an invariant: %s", f.Message)
		case f.File == "service/probe.go" && f.Line == 5 && strings.HasPrefix(f.Rule, "CMT"):
			t.Errorf("a type doc within budget was reported: %s %s", f.Rule, f.Message)
		case f.File == "transport/router.go" && f.Line == 8:
			// A pattern rule reads code, never prose: every mention of a vendor
			// in a comment is legitimate, and matching them would be all noise.
			t.Errorf("a canonical rule matched inside a comment: %s", f.Message)
		}
	}
}

func TestBaselineExcusesKnownAndCatchesNew(t *testing.T) {
	root := copyTree(t, "testdata/probe")

	before := runProbe(t, root)
	if before.Errors == 0 {
		t.Fatal("fixture must start dirty for this test to mean anything")
	}
	if _, err := writeBaseline(root, before.Findings); err != nil {
		t.Fatalf("writeBaseline: %v", err)
	}

	after := runProbe(t, root)
	if len(after.Findings) != 0 {
		t.Fatalf("baseline should excuse every known finding, still got %d: %v",
			len(after.Findings), after.Findings[0])
	}

	// The point of the gate: today's debt is forgiven, tomorrow's is not.
	fresh := "package service\n\n// Nova regra que não está em inglês.\nfunc Fresh() {}\n"
	if err := os.WriteFile(filepath.Join(root, "service", "fresh.go"), []byte(fresh), 0o644); err != nil {
		t.Fatal(err)
	}
	withNew := runProbe(t, root)
	if !hits(withNew)["CMT-04@service/fresh.go:3"] {
		t.Errorf("a new violation must survive the baseline, got %d findings", len(withNew.Findings))
	}
}

func TestSuppressionNeedsAReason(t *testing.T) {
	root := copyTree(t, "testdata/probe")
	body := "package service\n\n" +
		"// quality-gate:allow CMT-04\n" +
		"// Comentário em português sem justificativa.\n" +
		"func Bare() {}\n\n" +
		"// quality-gate:allow CMT-04 — the message is quoted from the vendor's docs\n" +
		"// Comentário em português com justificativa.\n" +
		"func Justified() {}\n"
	if err := os.WriteFile(filepath.Join(root, "service", "sup.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got := hits(runProbe(t, root))

	if !got["GATE-01@service/sup.go:3"] {
		t.Error("a suppression without a reason must be reported as GATE-01")
	}
	if !got["CMT-04@service/sup.go:4"] {
		t.Error("a suppression without a reason must not suppress anything")
	}
	if got["CMT-04@service/sup.go:8"] {
		t.Error("a suppression with a reason must silence its rule")
	}
}

func copyTree(t *testing.T, src string) string {
	t.Helper()
	dst := t.TempDir()
	err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, raw, 0o644)
	})
	if err != nil {
		t.Fatalf("copyTree: %v", err)
	}
	return dst
}

// Prune is what makes an engine upgrade survivable: it drops the entries whose
// code is gone and freezes nothing new. Deriving that set from the findings is
// the bug it shipped with — a matched entry produces no finding by definition.
func TestPruneDropsOrphansAndFreezesNothing(t *testing.T) {
	root := copyTree(t, "testdata/probe")
	before := runProbe(t, root)
	if _, err := writeBaseline(root, before.Findings); err != nil {
		t.Fatal(err)
	}

	// Delete the code one entry excused, and add a fresh violation.
	if err := os.Remove(filepath.Join(root, "service", "verbose.go")); err != nil {
		t.Fatal(err)
	}
	fresh := "package service\n\n// Comentário novo que não está em inglês.\nfunc Fresh() {}\n"
	if err := os.WriteFile(filepath.Join(root, "service", "fresh.go"), []byte(fresh), 0o644); err != nil {
		t.Fatal(err)
	}

	var out, errOut strings.Builder
	if code := cmdPrune(loadProbe(t, root), &out, &errOut); code != exitOK {
		t.Fatalf("prune exit %d: %s", code, errOut.String())
	}

	after := runProbe(t, root)
	if after.Stale != 0 {
		t.Errorf("prune left %d stale entries behind", after.Stale)
	}
	if !hits(after)["CMT-04@service/fresh.go:3"] {
		t.Error("prune must not freeze a violation that arrived since the baseline")
	}
}
