package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var updateAgents = flag.Bool("update-agents", false, "rewrite the assembled agents from agents/_parts")

// The agents share rules; assembling each from parts keeps a single owner for what they share, and this
// test fails the day someone edits an assembled agent instead of its part.
func TestAgentsAreAssembledFromTheirParts(t *testing.T) {
	partsDir := filepath.Join("..", "agents", "_parts")
	readPart := func(name string) string {
		raw, err := os.ReadFile(filepath.Join(partsDir, name))
		if err != nil {
			t.Fatalf("read part %s: %v", name, err)
		}
		return strings.TrimRight(string(raw), "\n")
	}
	includes := strings.NewReplacer(
		"{{house-rules}}", readPart("house-rules.md"),
		"{{owners}}", readPart("owners.md"),
		"{{facts-and-errors}}", readPart("facts-and-errors.md"),
		"{{mutants}}", readPart("mutants.md"),
		"{{report}}", readPart("report.md"),
	)

	for _, name := range []string{"backend-implementer", "frontend-implementer", "architecture-reviewer", "blind-spot-reviewer", "unbiased-reviewer"} {
		want := includes.Replace(readPart(name+".md")) + "\n"
		path := filepath.Join("..", "agents", name+".md")

		if *updateAgents {
			if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
				t.Fatalf("write %s: %v", path, err)
			}
			continue
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if string(got) != want {
			t.Errorf("%s differs from agents/_parts; edit the part and run go test -run %s -update-agents", name, t.Name())
		}
	}
}
