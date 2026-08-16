package main

import "testing"

// An interface is the one place prose is the product: no length cap. What it
// may not do is describe behaviour, which changes under it.
func TestInterfaceDocHasNoCapButStillCannotNarrate(t *testing.T) {
	long := make([]string, 20)
	for i := range long {
		long[i] = "a contract line that carries real prose about the port"
	}
	run := func(c Comment) []Finding {
		var out []Finding
		checkComments(&Config{}, &File{Path: "x.go", Comments: []Comment{c}}, func(f Finding) { out = append(out, f) })
		return out
	}
	verbose := Comment{Line: 1, EndLine: 20, Pos: PosInterface, Lines: long, Text: "prose"}
	for _, f := range run(verbose) {
		if f.Rule == "CMT-01" {
			t.Errorf("an interface doc must have no length cap, got %s", f.Message)
		}
	}
	narrating := Comment{
		Line: 1, EndLine: 1, Pos: PosInterface,
		Lines: []string{"Set the status and return the booking"},
		Text:  "Set the status and return the booking",
	}
	found := false
	for _, f := range run(narrating) {
		if f.Rule == "CMT-02" {
			found = true
		}
	}
	if !found {
		t.Error("an interface doc that narrates behaviour must still be reported")
	}
}
