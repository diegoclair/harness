package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A release-backed artifact without a tag prefix silently resolves to the
// wrong release series; a source-backed one has no releases at all, so the
// requirement is per-kind rather than global.
func TestCatalogIsWellFormed(t *testing.T) {
	seenPrefix := map[string]bool{}
	for _, a := range catalog {
		if a.Name == "" || a.Summary == "" {
			t.Errorf("incomplete catalog entry: %+v", a)
		}
		if a.TagPrefix == "" {
			continue
		}
		// A tag prefix only means anything for a skill that ships a binary,
		// and a wrong one silently resolves to another skill's release series.
		if !strings.HasSuffix(a.TagPrefix, "-v") {
			t.Errorf("%s: tag prefix %q should end in -v", a.Name, a.TagPrefix)
		}
		if seenPrefix[a.TagPrefix] {
			t.Errorf("%s: tag prefix %q is used by another artifact", a.Name, a.TagPrefix)
		}
		seenPrefix[a.TagPrefix] = true
		if a.Kind == KindAgent {
			t.Errorf("%s: an agent is plain files and cannot carry a tag prefix", a.Name)
		}
	}
}

// A skill that ships cli/ resolves its binary by tag prefix; without one the
// install fails at run time with a confusing message from deep in the resolver.
func TestSkillsWithACLIDeclareATagPrefix(t *testing.T) {
	for _, a := range catalog {
		if a.Kind != KindSkill {
			continue
		}
		if !shipsCLI(filepath.Join("..", "skills", a.Name)) {
			continue
		}
		if a.TagPrefix == "" {
			t.Errorf("%s ships a cli/ and needs a TagPrefix to resolve its releases", a.Name)
		}
	}
}

// Every catalogued artifact must exist in the tree, or `install` offers a
// name it cannot deliver.
func TestEveryCatalogEntryExistsInTheTree(t *testing.T) {
	for _, a := range catalog {
		path := filepath.Join("..", "agents", a.Name+".md")
		if a.Kind == KindSkill {
			path = filepath.Join("..", "skills", a.Name, "SKILL.md")
		}
		if _, err := os.Stat(path); err != nil {
			t.Errorf("%s is catalogued but missing from the repo: %v", a.Name, err)
		}
	}
}

// `install <name>` is ambiguous if two kinds share a name.
func TestCatalogNamesAreUniqueAcrossKinds(t *testing.T) {
	seen := map[string]Kind{}
	for _, a := range catalog {
		if kind, dup := seen[a.Name]; dup {
			t.Errorf("name %q used by both %s and %s", a.Name, kind, a.Kind)
		}
		seen[a.Name] = a.Kind
	}
}

func TestEveryRequirementExists(t *testing.T) {
	for _, a := range catalog {
		for _, req := range a.Requires {
			if _, ok := findArtifact(req); !ok {
				t.Errorf("%s requires %q, which is not in the catalog", a.Name, req)
			}
		}
	}
}

func TestResolveRequiresPullsInDependencies(t *testing.T) {
	devLoop, ok := findArtifact("dev-loop")
	if !ok {
		t.Fatal("dev-loop missing from catalog")
	}

	got, added, err := resolveRequires([]Artifact{devLoop})
	if err != nil {
		t.Fatalf("resolveRequires: %v", err)
	}
	if len(added) != 1 || added[0] != "unbiased-reviewer" {
		t.Errorf("added = %v, want [unbiased-reviewer]", added)
	}
	if !containsName(got, "unbiased-reviewer") {
		t.Errorf("selection %v is missing the required agent", names(got))
	}
	if got[0].Name != "dev-loop" {
		t.Errorf("original selection should stay first, got %v", names(got))
	}
}

func TestResolveRequiresIsIdempotent(t *testing.T) {
	devLoop, _ := findArtifact("dev-loop")
	reviewer, _ := findArtifact("unbiased-reviewer")

	got, added, err := resolveRequires([]Artifact{devLoop, reviewer})
	if err != nil {
		t.Fatalf("resolveRequires: %v", err)
	}
	if len(added) != 0 {
		t.Errorf("added = %v, want none when the dependency is already selected", added)
	}
	if len(got) != 2 {
		t.Errorf("selection = %v, want 2 entries with no duplicate", names(got))
	}
}

func TestResolveRequiresRejectsUnknownDependency(t *testing.T) {
	broken := Artifact{Name: "broken", Kind: KindSkill, Requires: []string{"does-not-exist"}}
	if _, _, err := resolveRequires([]Artifact{broken}); err == nil {
		t.Fatal("want an error for an unresolvable requirement")
	}
}

func containsName(in []Artifact, name string) bool {
	for _, a := range in {
		if a.Name == name {
			return true
		}
	}
	return false
}

func names(in []Artifact) []string {
	out := make([]string, 0, len(in))
	for _, a := range in {
		out = append(out, a.Name)
	}
	return out
}

// A skill that names an agent in its body but forgets Requires installs
// silently broken — the failure mode the dependency field exists to prevent.
func TestSkillsDeclareTheAgentsTheyDispatch(t *testing.T) {
	agents := map[string]bool{}
	for _, a := range catalog {
		if a.Kind == KindAgent {
			agents[a.Name] = true
		}
	}

	for _, a := range catalog {
		if a.Kind != KindSkill {
			continue
		}
		body, err := os.ReadFile(filepath.Join("..", "skills", a.Name, "SKILL.md"))
		if err != nil {
			t.Fatalf("%s: %v", a.Name, err)
		}
		declared := map[string]bool{}
		for _, r := range a.Requires {
			declared[r] = true
		}
		for agent := range agents {
			if strings.Contains(string(body), agent) && !declared[agent] {
				t.Errorf("%s dispatches the %q agent but does not list it in Requires", a.Name, agent)
			}
		}
	}
}

// A dependency can itself have dependencies; expanding only the original
// selection would leave the chain half-installed.
func TestResolveRequiresIsTransitive(t *testing.T) {
	// dev-loop already requires unbiased-reviewer, so requiring dev-loop makes
	// a real two-hop chain.
	meta := Artifact{Name: "meta-skill", Kind: KindSkill, Requires: []string{"dev-loop"}}

	got, added, err := resolveRequires([]Artifact{meta})
	if err != nil {
		t.Fatalf("resolveRequires: %v", err)
	}
	for _, want := range []string{"dev-loop", "unbiased-reviewer"} {
		if !containsName(got, want) {
			t.Errorf("selection %v is missing %q from the dependency chain", names(got), want)
		}
	}
	if len(added) != 2 {
		t.Errorf("added = %v, want both hops reported", added)
	}
}
