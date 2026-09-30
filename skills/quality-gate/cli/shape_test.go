package main

import "testing"

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
