package main

// Skill is one installable skill in this monorepo.
type Skill struct {
	Name string
	// TagPrefix selects this skill's releases; the repo tags every skill
	// separately (confluence-v*, jira-v*, …) because GitHub's "latest"
	// pointer is a single value per repository.
	TagPrefix string
	Summary   string
	// VersionEnv is the pre-monorepo env var some skills still honour for
	// pinning a version.
	VersionEnv string
}

// catalog is the single source of truth for what can be installed. The README
// table and the per-skill install stubs should agree with it.
var catalog = []Skill{
	{
		Name:       "confluence-docs",
		TagPrefix:  "confluence-v",
		Summary:    "Search, create and update Confluence pages from natural language",
		VersionEnv: "CONFLUENCE_DOCS_VERSION",
	},
	{
		Name:       "jira-tickets",
		TagPrefix:  "jira-v",
		Summary:    "Read, create and transition Jira issues without burning context",
		VersionEnv: "JIRA_TICKETS_VERSION",
	},
	{
		Name:      "social-carousel",
		TagPrefix: "carousel-v",
		Summary:   "Generate Instagram and LinkedIn carousels from a YAML brief",
	},
}

// findSkill resolves a name to its catalog entry.
func findSkill(name string) (Skill, bool) {
	for _, s := range catalog {
		if s.Name == name {
			return s, true
		}
	}
	return Skill{}, false
}

// skillNames lists every installable name, for error messages.
func skillNames() []string {
	names := make([]string, 0, len(catalog))
	for _, s := range catalog {
		names = append(names, s.Name)
	}
	return names
}
