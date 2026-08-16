package main

import "fmt"

// Kind is where an artifact is installed: skills and agents live in different
// directories and Claude Code loads them differently.
type Kind int

const (
	KindSkill Kind = iota
	KindAgent
)

func (k Kind) String() string {
	if k == KindAgent {
		return "agent"
	}
	return "skill"
}

// Artifact is one installable unit — a skill or an agent.
type Artifact struct {
	Name string
	Kind Kind
	// TagPrefix selects the releases of a skill that ships a binary; the repo
	// tags every such skill separately (confluence-v*, jira-v*, …) because
	// GitHub's "latest" pointer is a single value per repository. Ignored for
	// artifacts that are plain files.
	TagPrefix string
	Summary   string
	// VersionEnv is the pre-monorepo env var some skills still honour.
	VersionEnv string
	// Requires names artifacts this one cannot work without. A skill that
	// dispatches an agent is broken without it, so selection pulls it in.
	Requires []string
}

// catalog is the source of truth for what can be installed: a name absent
// here is rejected, so a typo never reaches the filesystem. Adding an artifact
// means adding an entry.
var catalog = []Artifact{
	{
		Name:    "unbiased-reviewer",
		Kind:    KindAgent,
		Summary: "Adversarial reviewer: mutation testing, own fixtures, APPROVE/REJECT with evidence",
	},
	{
		Name:     "dev-loop",
		Kind:     KindSkill,
		Summary:  "Build a non-trivial feature through implement -> unbiased review -> decide",
		Requires: []string{"unbiased-reviewer"},
	},
	{
		Name:     "implementation-plan",
		Kind:     KindSkill,
		Summary:  "Turn a fuzzy objective into a bulletproof spec, adversarially reviewed",
		Requires: []string{"unbiased-reviewer"},
	},
	{
		Name:       "confluence-docs",
		Kind:       KindSkill,
		TagPrefix:  "confluence-v",
		Summary:    "Search, create and update Confluence pages from natural language",
		VersionEnv: "CONFLUENCE_DOCS_VERSION",
	},
	{
		Name:       "jira-tickets",
		Kind:       KindSkill,
		TagPrefix:  "jira-v",
		Summary:    "Read, create and transition Jira issues without burning context",
		VersionEnv: "JIRA_TICKETS_VERSION",
	},
	{
		Name:      "social-carousel",
		Kind:      KindSkill,
		TagPrefix: "carousel-v",
		Summary:   "Generate Instagram and LinkedIn carousels from a YAML brief",
	},
	{
		Name:       "quality-gate",
		Kind:       KindSkill,
		TagPrefix:  "gate-v",
		Summary:    "Gate a delivery on comments, duplication, complexity and layer boundaries",
		VersionEnv: "QUALITY_GATE_VERSION",
	},
}

func findArtifact(name string) (Artifact, bool) {
	for _, a := range catalog {
		if a.Name == name {
			return a, true
		}
	}
	return Artifact{}, false
}

func artifactNames() []string {
	names := make([]string, 0, len(catalog))
	for _, a := range catalog {
		names = append(names, a.Name)
	}
	return names
}

// resolveRequires expands the selection with everything the chosen artifacts
// depend on, preserving order and reporting what was pulled in so the user
// never wonders where an extra file came from.
func resolveRequires(selected []Artifact) ([]Artifact, []string, error) {
	seen := map[string]bool{}
	for _, a := range selected {
		seen[a.Name] = true
	}

	var added []string
	// Index-based: appended dependencies are themselves scanned for theirs.
	for i := 0; i < len(selected); i++ {
		for _, req := range selected[i].Requires {
			if seen[req] {
				continue
			}
			dep, ok := findArtifact(req)
			if !ok {
				return nil, nil, fmt.Errorf("%s requires unknown artifact %q", selected[i].Name, req)
			}
			seen[req] = true
			selected = append(selected, dep)
			added = append(added, req)
		}
	}
	return selected, added, nil
}
