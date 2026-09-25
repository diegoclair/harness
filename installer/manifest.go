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
	// Setup marks a binary with a `setup` command whose --check reports
	// whether credentials or dependencies are in place. A binary without one
	// has nothing to configure, so probing it would only print an unknown
	// command as if it were a failure.
	Setup bool
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
		Name:    "backend-implementer",
		Kind:    KindAgent,
		Summary: "Implements backend code with responsibilities in their layer, and stops on product decisions",
	},
	{
		Name:    "frontend-implementer",
		Kind:    KindAgent,
		Summary: "Implements frontend code with the right layers and components, and stops on product decisions",
	},
	{
		Name:     "dev-loop",
		Kind:     KindSkill,
		Summary:  "Build a non-trivial feature through implement -> unbiased review -> decide",
		Requires: []string{"backend-implementer", "frontend-implementer", "unbiased-reviewer"},
	},
	{
		Name:     "implementation-plan",
		Kind:     KindSkill,
		Summary:  "Turn a fuzzy objective into a bulletproof spec, adversarially reviewed",
		Requires: []string{"unbiased-reviewer"},
	},
	{
		Name:     "orchestrator",
		Kind:     KindSkill,
		Summary:  "Lead a multi-agent delivery: co-built specs, decision triage, nothing shipped unreviewed",
		Requires: []string{"implementation-plan", "dev-loop", "backend-implementer", "frontend-implementer", "unbiased-reviewer"},
	},
	{
		Name:       "confluence-docs",
		Kind:       KindSkill,
		TagPrefix:  "confluence-v",
		Setup:      true,
		Summary:    "Search, create and update Confluence pages from natural language",
		VersionEnv: "CONFLUENCE_DOCS_VERSION",
	},
	{
		Name:       "jira-tickets",
		Kind:       KindSkill,
		TagPrefix:  "jira-v",
		Setup:      true,
		Summary:    "Read, create and transition Jira issues without burning context",
		VersionEnv: "JIRA_TICKETS_VERSION",
	},
	{
		Name:      "social-carousel",
		Kind:      KindSkill,
		TagPrefix: "carousel-v",
		Setup:     true,
		Summary:   "Generate Instagram and LinkedIn carousels from a YAML brief",
	},
	{
		Name:    "business-ai-first",
		Kind:    KindSkill,
		Summary: "AI-first company playbook: deliver results, weekly ritual, playbook audit, thinking partner",
	},
	{
		Name:    "landing-seo-geo",
		Kind:    KindSkill,
		Summary: "SEO + GEO for a static landing: audit, metadata/JSON-LD, content pages, sourced claims, indexing",
	},
	{
		Name:    "ui-ux",
		Kind:    KindSkill,
		Summary: "Design rounds, studio-grade scroll motion, review loop and cheap proof for pages that must hold attention",
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
