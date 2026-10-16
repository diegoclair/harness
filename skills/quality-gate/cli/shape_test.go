package main

import (
	"strings"
	"testing"
)

// Each silent case is the shape a reviewer would defend: a helper with two
// callers, one handed over as a value, one that composes its arguments, and a
// closure that never calls itself.
func TestShapeRulesStaySilentOnDefensibleShapes(t *testing.T) {
	res := runProbe(t, "testdata/probe")
	for _, f := range res.Findings {
		if f.File != "service/shape.go" {
			continue
		}
		switch f.Line {
		case 20, 28, 36, 53:
			t.Errorf("%s fired at service/shape.go:%d: %s", f.Rule, f.Line, f.Message)
		}
	}
}

func TestNewPieceFiresOnlyOnDeclarationsTheModuleNeverHad(t *testing.T) {
	piece := "service/shape_piece.go"
	cases := []struct {
		name   string
		before func(root string) string
		after  func(root string, saved string)
		file   string
		fires  bool
	}{
		{
			name:   "new file beside its subject's owner",
			before: func(root string) string { return takeOut(t, root, piece) },
			after:  func(root, saved string) { writeFile(t, root, piece, saved) },
			file:   piece, fires: true,
		},
		{
			name:  "new file of a subject with no home yet",
			after: func(root, _ string) { writeFile(t, root, "service/ledger.go", "package service\n\ntype Ledger int\n") },
			file:  "service/ledger.go",
		},
		{
			name: "function exported on its move",
			after: func(root, _ string) {
				takeOut(t, root, piece)
				writeFile(t, root, "domain/rule_piece.go", "package domain\n\ntype shapePiece struct{ Name string }\n")
			},
			file: "domain/rule_piece.go",
		},
		{
			name: "function turned into a method on its move",
			before: func(root string) string {
				writeFile(t, root, piece, "package service\n\nfunc ShapeName() string { return \"\" }\n")
				return ""
			},
			after: func(root, _ string) {
				takeOut(t, root, piece)
				writeFile(t, root, "domain/rule_piece.go",
					"package domain\n\ntype Ruler struct{}\n\nfunc (Ruler) ShapeName() string { return \"\" }\n")
			},
			file: "domain/rule_piece.go",
		},
		{
			name:  "rename",
			after: func(root, _ string) { moveFile(t, root, piece, "service/shape_part.go") },
			file:  "service/shape_part.go",
		},
		{
			name: "merge of two files into a new one",
			before: func(root string) string {
				writeFile(t, root, "service/shape_extra.go", "package service\n\ntype ShapeExtra int\n")
				return ""
			},
			after: func(root, _ string) {
				takeOut(t, root, piece)
				takeOut(t, root, "service/shape_extra.go")
				writeFile(t, root, "service/shape_joined.go",
					"package service\n\ntype ShapePiece struct{ Name string }\n\ntype ShapeExtra int\n")
			},
			file: "service/shape_joined.go",
		},
		{
			name: "move to another package",
			after: func(root, _ string) {
				takeOut(t, root, piece)
				writeFile(t, root, "domain/rule_piece.go", "package domain\n\ntype ShapePiece struct{ Name string }\n")
			},
			file: "domain/rule_piece.go",
		},
		{
			name: "existing file whose declaration is renamed in place",
			after: func(root, _ string) {
				writeFile(t, root, piece, "package service\n\ntype ShapeShard struct{ Name string }\n")
			},
			file: piece,
		},
		{
			name: "layout declared one file per subject",
			before: func(root string) string {
				cfg := readFile(t, root, ".quality-gate.yml") + "\ntiny_file:\n  except: [\"service/shape_*.go\"]\n"
				writeFile(t, root, ".quality-gate.yml", cfg)
				return takeOut(t, root, piece)
			},
			after: func(root, saved string) { writeFile(t, root, piece, saved) },
			file:  piece,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := copyTree(t, "testdata/probe")
			saved := ""
			if tc.before != nil {
				saved = tc.before(root)
			}
			commitAll(t, root)
			tc.after(root, saved)

			res, err := runCheck(loadProbe(t, root), checkOptions{})
			if err != nil {
				t.Fatalf("runCheck: %v", err)
			}
			if fired := hits(res)["CPX-09@"+tc.file+":1"]; fired != tc.fires {
				t.Errorf("CPX-09 on %s: fired=%v, want %v (findings: %v)", tc.file, fired, tc.fires, res.Findings)
			}
		})
	}
}

// A base branch git cannot relate HEAD to must say CPX-09 did not run, never
// pass for a delivery with no new piece.
func TestNewPieceSaysWhyItDidNotRun(t *testing.T) {
	root := copyTree(t, "testdata/probe")
	commitAll(t, root)
	gitIn(t, root, "checkout", "-q", "--orphan", "unrelated")
	gitIn(t, root, "-c", "user.name=probe", "-c", "user.email=probe@example.com", "-c", "commit.gpgsign=false",
		"-c", "core.hooksPath=/dev/null", "commit", "-q", "-m", "unrelated")
	writeFile(t, root, "service/shape_new.go", "package service\n\ntype ShapeNew int\n")

	res, err := runCheck(loadProbe(t, root), checkOptions{})
	if err != nil {
		t.Fatalf("runCheck: %v", err)
	}
	if len(res.Notices) != 1 || !strings.Contains(res.Notices[0], "CPX-09 did not run") {
		t.Errorf("want one notice naming CPX-09, got %v", res.Notices)
	}
}

// A merge-base file that does not parse is skipped and reported, as on disk;
// the rest of the module still answers.
func TestNewPieceSkipsAnUnparsableBaseFile(t *testing.T) {
	root := copyTree(t, "testdata/probe")
	writeFile(t, root, "service/broken.go", "package service\n\nfunc {\n")
	commitAll(t, root)
	writeFile(t, root, "service/broken.go", "package service\n")
	writeFile(t, root, "service/shape_new.go", "package service\n\ntype ShapeNew int\n")

	res, err := runCheck(loadProbe(t, root), checkOptions{})
	if err != nil {
		t.Fatalf("an unparsable base file must not abort the check: %v", err)
	}
	if !strings.Contains(strings.Join(res.ParseErrors, "\n"), "service/broken.go at the merge-base") {
		t.Errorf("the base file must be reported as skipped, got %v", res.ParseErrors)
	}
	if !hits(res)["CPX-09@service/shape_new.go:1"] {
		t.Error("the new piece must still be reported")
	}
}

func TestDeclarationsAtReportsAMissingBlob(t *testing.T) {
	root := copyTree(t, "testdata/probe")
	commitAll(t, root)
	res := &checkResult{}
	_, err := declarationsAt(loadProbe(t, root), map[string]string{"service/gone.go": strings.Repeat("ab", 20)}, res)
	if err != nil {
		t.Fatalf("a missing blob must be reported, not abort: %v", err)
	}
	if len(res.ParseErrors) != 0 || len(res.Notices) != 1 || !strings.Contains(res.Notices[0], "is missing") {
		t.Errorf("want the missing blob as a notice, never as unparseable, got %v / %v", res.ParseErrors, res.Notices)
	}
}
