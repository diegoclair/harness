package main

import (
	"fmt"
	"os"
	"os/exec"
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
		"CMT-02@service/probe.go:19",
		"CMT-04@service/probe.go:21",
		"CMT-05@service/probe.go:23",
		"CMT-06@service/probe.go:23",
		"CMT-07@service/probe.go:28",
		"CMT-09@service/probe.go:8",
		"CMT-10@service/probe.go:10",
		"CPX-02@service/verbose.go:10",
		"CPX-04@service/verbose.go:10",
		"NAM-01@service/naming.go:5",
		"CPX-06@service/shape.go:5",
		"CPX-07@service/shape.go:12",
		"CPX-08@service/shape.go:41",
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
		case f.File == "service/probe.go" && f.Line == 10 && f.Rule != "CMT-10":
			t.Errorf("%s fired on a field comment carrying a unit and an invariant: %s", f.Rule, f.Message)
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

func parseGoSource(t *testing.T, src string) *File {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "x.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := parseGo(root, "x.go")
	if err != nil {
		t.Fatalf("parseGo: %v", err)
	}
	return f
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

// A directive whose reason wraps must still reach the code below it: in JSX the
// comment cannot sit adjacent to the attribute it excuses.
func TestSuppressionReachesPastAWrappedReason(t *testing.T) {
	root := copyTree(t, "testdata/probe")
	body := "package service\n\n" +
		"// quality-gate:allow CMT-04 — the vendor's own docs are quoted here, and\n" +
		"// translating them would send the reader looking for labels that do not exist\n" +
		"// Comentário em português que a diretiva precisa alcançar.\n" +
		"func Reached() {}\n"
	if err := os.WriteFile(filepath.Join(root, "service", "reach.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if hits(runProbe(t, root))["CMT-04@service/reach.go:5"] {
		t.Error("the directive must reach the comment below its own wrapped reason")
	}
}

// The flagged comment usually sits inside the directive's own group, below the
// reason — and in JSX the directive cannot sit adjacent to the code at all. One
// anchor cannot serve both, so the reach spans the group and what follows it.
func TestSuppressionCoversItsGroupAndWhatFollows(t *testing.T) {
	root := copyTree(t, "testdata/probe")
	body := "package service\n\n" +
		"// quality-gate:allow CMT-04 — the vendor's own labels are quoted below and\n" +
		"// translating them would send the reader looking for menus that do not exist\n" +
		"// Comentário em português dentro do mesmo bloco.\n" +
		"func Inside() {}\n\n" +
		"// quality-gate:allow CMT-04 — same reason, wrapped across two lines so the\n" +
		"// anchor cannot simply be the directive's own line\n" +
		"//\n" +
		"// Comentário em português no bloco seguinte.\n" +
		"func Below() {}\n"
	if err := os.WriteFile(filepath.Join(root, "service", "reach.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, line := range []int{5, 11} {
		if hits(runProbe(t, root))[fmt.Sprintf("CMT-04@service/reach.go:%d", line)] {
			t.Errorf("the directive must reach line %d", line)
		}
	}
}

func commitAll(t *testing.T, root string) {
	t.Helper()
	gitIn(t, root, "init", "-q", "-b", "main")
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "-c", "user.name=probe", "-c", "user.email=probe@example.com", "-c", "commit.gpgsign=false",
		"-c", "core.hooksPath=/dev/null", "commit", "-q", "-m", "base")
}

func gitIn(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func readFile(t *testing.T, root, rel string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func writeFile(t *testing.T, root, rel, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, rel), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func takeOut(t *testing.T, root, rel string) string {
	t.Helper()
	body := readFile(t, root, rel)
	if err := os.Remove(filepath.Join(root, rel)); err != nil {
		t.Fatal(err)
	}
	return body
}

func moveFile(t *testing.T, root, from, to string) {
	t.Helper()
	if err := os.Rename(filepath.Join(root, from), filepath.Join(root, to)); err != nil {
		t.Fatal(err)
	}
}

func baselineProbe(t *testing.T) string {
	t.Helper()
	root := copyTree(t, "testdata/probe")
	if _, err := writeBaseline(root, runProbe(t, root).Findings); err != nil {
		t.Fatal(err)
	}
	return root
}

func remap(t *testing.T, root, to string, from ...string) string {
	t.Helper()
	args := append([]string{"remap", "--dir", root, "--from"}, from...)
	args = append(args, "--to", to)
	var out, errOut strings.Builder
	if code := cmdBaseline(args, &out, &errOut); code != exitOK {
		t.Fatalf("remap exit %d: %s", code, errOut.String())
	}
	return out.String()
}

// A moved file keeps its excuses: without the remap every entry dies with the
// old path and the same debt reappears as new.
func TestRemapCarriesARenamedFile(t *testing.T) {
	root := baselineProbe(t)
	moveFile(t, root, "data/first.go", "data/origin.go")
	moveFile(t, root, "service/naming.go", "service/profile.go")
	if len(runProbe(t, root).Findings) == 0 {
		t.Fatal("a rename must lose the excuses before the remap, or this test proves nothing")
	}

	remap(t, root, "data/origin.go", "data/first.go")
	remap(t, root, "service/profile.go", "service/naming.go")

	after := runProbe(t, root)
	if len(after.Findings) != 0 {
		t.Errorf("after the remap the rename must report nothing, got %v", after.Findings)
	}
}

func TestRemapCarriesTwoFilesMergedIntoOne(t *testing.T) {
	root := baselineProbe(t)
	naming := takeOut(t, root, "service/naming.go")
	verbose := takeOut(t, root, "service/verbose.go")
	writeFile(t, root, "service/merged.go", naming+strings.TrimPrefix(verbose, "package service\n"))
	first := takeOut(t, root, "data/first.go")
	second := takeOut(t, root, "data/second.go")
	writeFile(t, root, "data/both.go", first+strings.TrimPrefix(second, "package data\n\nimport \"strings\"\n"))
	writeFile(t, root, "domain/rule.go", strings.ReplaceAll(readFile(t, root, "domain/rule.go"), "NormalizeFirst", "NormalizeSecond"))

	remap(t, root, "service/merged.go", "service/naming.go", "service/verbose.go")
	remap(t, root, "data/both.go", "data/first.go", "data/second.go")

	after := runProbe(t, root)
	for _, f := range after.Findings {
		if f.File == "service/merged.go" || f.File == "data/both.go" || f.Rule == "GATE-02" {
			t.Errorf("a merge must keep the excuses of both files, got %s %s: %s", f.Rule, f.File, f.Message)
		}
	}
}

// A clone whose two ends moved in separate calls cannot be rebuilt from either
// call alone: it is listed for a person, and the baseline is left as it was.
func TestRemapListsAClonePairItCannotRebuild(t *testing.T) {
	root := baselineProbe(t)
	moveFile(t, root, "data/first.go", "data/a.go")
	moveFile(t, root, "data/second.go", "data/b.go")
	before := readFile(t, root, baselineName)

	out := remap(t, root, "data/a.go", "data/first.go")

	if !strings.Contains(out, "unmatched") || !strings.Contains(out, "DUP-01") {
		t.Errorf("the clone pair must be listed as unmatched, got:\n%s", out)
	}
	if !strings.Contains(out, "baseline --prune") {
		t.Errorf("an unmatched entry must point at prune for code that is gone, got:\n%s", out)
	}
	if readFile(t, root, baselineName) != before {
		t.Error("an unmatched entry must be kept as it was, never regenerated")
	}
	if !hits(runProbe(t, root))["DUP-01"] {
		t.Error("the clone the remap could not carry must still surface in check")
	}
}

func baselineProbeWith(t *testing.T, files map[string]string) string {
	t.Helper()
	root := copyTree(t, "testdata/probe")
	for rel, body := range files {
		writeFile(t, root, rel, body)
	}
	if _, err := writeBaseline(root, runProbe(t, root).Findings); err != nil {
		t.Fatal(err)
	}
	return root
}

func withBodyComment(name string) string {
	return "func " + name + "() {\n\t// Comentário que não está em inglês.\n}\n"
}

// A numbered finding's signature depends on its neighbours' order, so its
// entry is never carried, even by a plain rename.
func TestRemapLeavesANumberedFindingUnmatched(t *testing.T) {
	root := baselineProbeWith(t, map[string]string{
		"service/twice.go": "package service\n\n" + withBodyComment("One") + "\n" + withBodyComment("Two"),
	})
	moveFile(t, root, "service/twice.go", "service/again.go")

	out := remap(t, root, "service/again.go", "service/twice.go")

	if !strings.Contains(out, "remapped 1 entr") || !strings.Contains(out, "CMT-04 service/twice.go") {
		t.Errorf("the first comment must be carried and the numbered one listed, got:\n%s", out)
	}
	for _, f := range runProbe(t, root).Findings {
		if f.File == "service/again.go" && f.Rule == "CMT-04" && !strings.Contains(f.Signature, "#") {
			t.Error("the unnumbered finding must carry the unnumbered entry, never the numbered one")
		}
	}
}

// A file merged into an owner that already holds the same finding would give
// two entries one signature: the moved one is listed instead.
func TestRemapIntoAnOwnerListsWhatItAlreadyExcuses(t *testing.T) {
	root := baselineProbeWith(t, map[string]string{
		"service/owner.go": "package service\n\n" + withBodyComment("One"),
		"service/piece.go": "package service\n\n" + withBodyComment("Two"),
	})
	piece := takeOut(t, root, "service/piece.go")
	writeFile(t, root, "service/owner.go", readFile(t, root, "service/owner.go")+strings.TrimPrefix(piece, "package service\n"))

	out := remap(t, root, "service/owner.go", "service/piece.go")

	if !strings.Contains(out, "CMT-04 service/piece.go") {
		t.Errorf("the moved entry must be listed, not given the owner's signature, got:\n%s", out)
	}
}

// Two entries claiming one signature cannot both be carried, and picking one
// would drop the other in silence: both are listed.
func TestRemapListsEveryEntryClaimingOneSignature(t *testing.T) {
	root := baselineProbeWith(t, map[string]string{
		"service/one.go": "package service\n\n" + withBodyComment("One"),
		"service/two.go": "package service\n\n" + withBodyComment("Two"),
	})
	one := takeOut(t, root, "service/one.go")
	two := takeOut(t, root, "service/two.go")
	writeFile(t, root, "service/both.go", one+strings.TrimPrefix(two, "package service\n"))
	before := readFile(t, root, baselineName)

	out := remap(t, root, "service/both.go", "service/one.go", "service/two.go")

	if !strings.Contains(out, "CMT-04 service/one.go") || !strings.Contains(out, "CMT-04 service/two.go") {
		t.Errorf("both entries claiming one signature must be listed, got:\n%s", out)
	}
	if readFile(t, root, baselineName) != before {
		t.Error("an entry claimed twice must be kept as it was")
	}
}

// A move across packages carries what still holds at the new path and lists
// what the new package no longer produces.
func TestRemapAcrossPackagesCarriesOnlyWhatStillHolds(t *testing.T) {
	root := baselineProbe(t)
	moveFile(t, root, "service/naming.go", "data/naming.go")
	writeFile(t, root, "data/naming.go", strings.Replace(readFile(t, root, "data/naming.go"), "package service", "package data", 1))
	moveFile(t, root, "domain/rule.go", "service/rule.go")
	writeFile(t, root, "service/rule.go", strings.Replace(readFile(t, root, "service/rule.go"), "package domain", "package service", 1))

	remap(t, root, "data/naming.go", "service/naming.go")
	out := remap(t, root, "service/rule.go", "domain/rule.go")

	if !strings.Contains(out, "ARC-01 domain/rule.go") {
		t.Errorf("an import the new layer allows must leave its entry unmatched, got:\n%s", out)
	}
	for _, f := range runProbe(t, root).Findings {
		if f.File == "data/naming.go" {
			t.Errorf("a finding that still holds after the move must keep its excuse, got %s: %s", f.Rule, f.Message)
		}
	}
}

// An old path still on disk could be a clone peer of the new file, which makes
// an old signature ambiguous: remap refuses instead of guessing.
func TestRemapRefusesAnOldPathStillOnDisk(t *testing.T) {
	root := baselineProbe(t)
	writeFile(t, root, "data/copy.go", readFile(t, root, "data/first.go"))
	var out, errOut strings.Builder
	code := cmdBaseline([]string{"remap", "--dir", root, "--from", "data/first.go", "--to", "data/copy.go"}, &out, &errOut)
	if code != exitUsage || !strings.Contains(errOut.String(), "still exists") {
		t.Errorf("want a usage error naming the old path, got %d: %s", code, errOut.String())
	}
}
