package main

import (
	"strings"
	"testing"
)

const probeWeb = "testdata/probe-web"

// F1 shipped with two ARC rules silently mute, and only a fixture carrying one
// deliberate violation per rule caught it. This is that fixture for the web
// ruleset: a rule that stops firing fails here instead of passing a repo.
func TestProbeWebFiresEveryRule(t *testing.T) {
	got := hits(runProbe(t, probeWeb))

	for _, want := range []string{
		"CMT-02@src/features/agenda/CommentProbe.tsx:21",
		"CMT-04@src/features/agenda/CommentProbe.tsx:23",
		"CMT-05@src/features/agenda/CommentProbe.tsx:25",
		"CMT-06@src/features/agenda/CommentProbe.tsx:25",
		"CMT-07@src/features/agenda/CommentProbe.tsx:14",
		"CMT-09@src/features/agenda/CommentProbe.tsx:6",
		"CPX-01@src/lib/rank.ts:19",
		"CPX-02@src/lib/rank.ts:1",
		"CPX-04@src/lib/rank.ts:1",
		"CPX-05@src/features/agenda/HookHeavy.tsx:3",
		"DUP-01@src/lib/normalizeFirst.ts:1",
		"DUP-02@src/lib/scoreSlots.ts:1",
		"DUP-03@src/components/ui/SummaryPanel.tsx:3",
		"DUP-04@src/features/agenda/first.test.ts:5",
		"ARC-10@src/features/agenda/CrossFeature.tsx:1",
		"ARC-11@src/components/ui/Shared.tsx:1",
		"ARC-12@src/features/agenda/loader.ts:2",
		"ARC-13@src/features/agenda/DayCard.tsx:11",     // line scope: raw hex
		"ARC-13@src/features/agenda/ConfirmSheet.tsx:5", // element scope: Drawer
		"ARC-14@src/features/agenda/DayCard.tsx:7",      // date arithmetic
		"ARC-14@src/features/agenda/DayCard.tsx:8",      // money arithmetic
		"ARC-14@src/features/agenda/DayCard.tsx:9",      // a state decided inline
	} {
		if !got[want] {
			t.Errorf("expected finding %s, missing", want)
		}
	}
}

// Each entry is a false-positive family the calibration run turned up. A rule
// that fires on everything is as useless as one that fires on nothing.
func TestProbeWebLeavesGoodCodeAlone(t *testing.T) {
	const probe = "src/features/agenda/CommentProbe.tsx"
	unwanted := map[string]string{
		"CMT-09@" + probe + ":4":                     "a member comment carrying a unit and an invariant",
		"CMT-09@" + probe + ":8":                     "a function-typed member is a contract and earns the function budget",
		"CMT-07@" + probe + ":12":                    "a box-drawing section divider claims nothing about the code below it",
		"CMT-02@" + probe + ":12":                    "a box-drawing section divider claims nothing about the code below it",
		"CMT-01@" + probe + ":29":                    "a `{/* … */}` comment sits on its own line; the `{` is not code it trails",
		"CMT-06@" + probe + ":36":                    "an @example body documents the call, it is not code left behind",
		"CMT-01@src/features/agenda/BigDoc.tsx:13":   "a module-scope binding gets the orphan budget, not the member budget",
		"ARC-13@src/components/ui/Drawer.tsx:3":      "the canonical component itself is exempt from its own row",
		"ARC-10@src/features/agenda/first.test.ts:3": "a test reaches across the layers it exercises",
	}
	got := hits(runProbe(t, probeWeb))
	for finding, why := range unwanted {
		if got[finding] {
			t.Errorf("%s fired: %s", finding, why)
		}
	}
	for _, f := range runProbe(t, probeWeb).Findings {
		if f.Rule == "ARC-12" && strings.HasPrefix(f.File, "src/services/") {
			t.Errorf("ARC-12 fired inside the layer that owns HTTP: %s:%d", f.File, f.Line)
		}
	}
}

func TestWebScannerFillsTheIR(t *testing.T) {
	f, err := parseWeb(probeWeb, "src/features/agenda/CommentProbe.tsx")
	if err != nil {
		t.Fatalf("parseWeb: %v", err)
	}

	byName := map[string]Func{}
	for _, fn := range f.Funcs {
		byName[fn.Name] = fn
	}
	for _, want := range []Func{
		{Name: "helper", Line: 15, EndLine: 17, Params: 0},
		// A destructured props object is one parameter, not one per prop.
		{Name: "CommentProbe", Line: 19, EndLine: 34, Params: 1, Hooks: 1, Component: true},
		{Name: "formatSlot", Line: 41, EndLine: 43, Params: 1},
	} {
		got, ok := byName[want.Name]
		if !ok {
			t.Errorf("function %s not found; got %v", want.Name, f.Funcs)
			continue
		}
		if got.Line != want.Line || got.EndLine != want.EndLine {
			t.Errorf("%s spans %d-%d, want %d-%d", want.Name, got.Line, got.EndLine, want.Line, want.EndLine)
		}
		if got.Params != want.Params {
			t.Errorf("%s takes %d params, want %d", want.Name, got.Params, want.Params)
		}
		if got.Hooks != want.Hooks || got.Component != want.Component {
			t.Errorf("%s: hooks %d component %v, want %d/%v",
				want.Name, got.Hooks, got.Component, want.Hooks, want.Component)
		}
	}

	if want := "@/features/clientes/types"; len(f.Imports) == 0 {
		t.Errorf("no imports recorded")
	} else if f.Imports[0].Path != "react" {
		t.Errorf("first import is %q, want react (%s is another file's)", f.Imports[0].Path, want)
	}

	// Comments are blanked out of CodeLines so a source-matching rule never
	// fires on prose, and the line count has to stay aligned with the source.
	if len(f.CodeLines) != len(f.SrcLines) {
		t.Fatalf("CodeLines has %d lines, SrcLines %d", len(f.CodeLines), len(f.SrcLines))
	}
	if strings.Contains(strings.Join(f.CodeLines, "\n"), "português") {
		t.Error("comment text survived into CodeLines")
	}
	if !strings.Contains(f.CodeLines[23], "const label") {
		t.Errorf("CodeLines lost the code on line 24: %q", f.CodeLines[23])
	}
}

func TestWebScannerSurvivesAmbiguousAngleBrackets(t *testing.T) {
	f, err := parseWeb(probeWeb, "src/features/agenda/HookHeavy.tsx")
	if err != nil {
		t.Fatalf("parseWeb: %v", err)
	}
	// A type argument read as an element would swallow the rest of the file
	// into markup and lose every function below it.
	if len(f.Funcs) != 1 || f.Funcs[0].Name != "HookHeavy" {
		t.Fatalf("expected HookHeavy alone, got %v", f.Funcs)
	}
	if f.Funcs[0].Hooks != 11 {
		t.Errorf("HookHeavy calls 11 hooks, scanner counted %d", f.Funcs[0].Hooks)
	}
	for _, n := range f.JSXNodes {
		if n.Tag == "HTMLDivElement" {
			t.Error("a generic type argument was scanned as a JSX element")
		}
	}
}

func TestWebImportsResolveThroughAliasesAndRelatives(t *testing.T) {
	cfg := loadProbe(t, probeWeb)
	f := &File{Path: "src/features/agenda/DayCard.tsx", Lang: LangWeb}
	cases := map[string]string{
		"@/lib/utils":       "src/lib/utils",
		"./types":           "src/features/agenda/types",
		"../clientes/types": "src/features/clientes/types",
		"react":             "",
		"date-fns/format":   "",
	}
	for imp, want := range cases {
		if got := cfg.importPath(f, imp); got != want {
			t.Errorf("importPath(%q) = %q, want %q", imp, got, want)
		}
	}
	if got := cfg.contextOf("src/features/clientes/types"); got != "src/features/clientes" {
		t.Errorf("contextOf glob = %q, want src/features/clientes", got)
	}
}
