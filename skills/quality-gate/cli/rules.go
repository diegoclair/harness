package main

import (
	"fmt"
	"sort"
	"strings"
)

type Severity string

const (
	SevError Severity = "error"
	SevWarn  Severity = "warn"
)

type Rule struct {
	ID      string
	Sev     Severity
	Lang    string // "go", "web" or "both"
	Summary string
	Explain string
}

type Finding struct {
	Rule    string   `json:"rule"`
	Sev     Severity `json:"severity"`
	File    string   `json:"file"`
	Line    int      `json:"line"`
	Message string   `json:"message"`

	// Content-derived, never line-derived: the baseline is matched on it.
	Signature string `json:"-"`
}

var catalog = map[string]Rule{
	"CMT-01": {"CMT-01", SevError, "both", "comment block longer than the budget for its position",
		"A comment's budget scales with the reader's distance from the code: package 15 lines, type 10, func doc 5, inside a body 2, on a declaration 2. Cut it, or move the prose to the package doc where a reader who has not opened the file will find it."},
	"CMT-02": {"CMT-02", SevError, "both", "comment narrates the code below instead of stating its purpose",
		"Behavior changes and the comment rots into a lie; purpose survives the refactor. Say why this exists, what constraint it satisfies, or what external gotcha forced it — never what the next lines do."},
	"CMT-03": {"CMT-03", SevWarn, "both", "comment inside a function body",
		"Always a question, never automatic: is this explaining what the code already says? It survives when it carries a non-obvious constraint. It does not survive when it labels a section — extract and name instead."},
	"CMT-04": {"CMT-04", SevError, "both", "comment is not in English",
		"Technical comments are English; user-facing strings stay in the product's language."},
	"CMT-05": {"CMT-05", SevError, "both", "comment carries history",
		"Git owns history. The code describes today."},
	"CMT-06": {"CMT-06", SevError, "both", "commented-out code",
		"Delete it. Git remembers."},
	"CMT-07": {"CMT-07", SevWarn, "both", "doc only restates the symbol name",
		"`// GetUser gets the user` adds nothing the signature did not already say. Delete it, or replace it with why the symbol exists."},
	"CMT-08": {"CMT-08", SevWarn, "both", "delivery adds more comment than its budget",
		"A wall of comments no single rule flags. Reported at delivery level with the worst files named."},
	"CMT-09": {"CMT-09", SevWarn, "both", "declaration described instead of constrained",
		"A field, constant or enum member is already named. The only comment it earns carries a constraint the name cannot: a unit, an invariant, a format, an external contract, a contrast with a sibling, or a spec reference."},

	"DUP-01": {"DUP-01", SevError, "both", "block already exists elsewhere in the repo",
		"Reported against the whole repo, not just the diff: the finding that matters is that this block already exists. Import the original instead."},
	"DUP-02": {"DUP-02", SevError, "both", "same block shape with renamed identifiers",
		"A type-2 clone: identical structure, different names. Usually a copy-paste with a search-and-replace on top."},
	"DUP-03": {"DUP-03", SevError, "web", "markup subtree duplicated in another file",
		"A component rebuilt under a new name: same tree, new copy and new classes. The fix is a canonical component plus its row in the project's canonical table, not a third copy."},
	"DUP-04": {"DUP-04", SevWarn, "both", "clone confined to test files",
		"Table-driven tests legitimately repeat. Reported, never blocking."},

	"CPX-01": {"CPX-01", SevWarn, "both", "cyclomatic complexity above threshold",
		"An invitation to one question: does this function hold two different rules? If yes, split by responsibility. If it is one long rule expressed linearly, leave it alone and suppress with that reason."},
	"CPX-02": {"CPX-02", SevWarn, "both", "nesting deeper than threshold",
		"Deep nesting usually means an early return was missed, not that the function is too big."},
	"CPX-03": {"CPX-03", SevWarn, "both", "function longer than threshold",
		"Length alone is not a defect. Cutting a function to satisfy a number produces worse code than leaving it alone."},
	"CPX-04": {"CPX-04", SevWarn, "both", "too many parameters",
		"Go: ctx is not counted. Consider a parameter struct when the list is genuinely open-ended."},
	"CPX-05": {"CPX-05", SevWarn, "web", "React component above the size or hook budget",
		"A component past 250 lines or 10 hooks is usually two things: the markup and a piece of state logic with a name it has not been given yet. Extract the hook, or extract the sub-component — do not split the file to satisfy the number."},

	"ARC-01": {"ARC-01", SevError, "go", "domain imports an outer layer",
		"The domain names its ports; it never reaches for an adapter. Verified at zero violations when the rule was written."},
	"ARC-02": {"ARC-02", SevError, "go", "bounded contexts import each other",
		"Two contexts in one binary stay two contexts. Cross-context work goes through an in-process adapter wired at the composition root."},
	"ARC-03": {"ARC-03", SevError, "go", "transport imports the data layer",
		"A handler parses, delegates and renders. It never talks to a repository."},
	"ARC-04": {"ARC-04", SevError, "go", "data imports the service layer",
		"A repository answers where the data is. Declared DTO packages are the allowed exception."},
	"ARC-05": {"ARC-05", SevWarn, "go", "domain rule inside the data layer",
		"A CASE WHEN, a business-state literal or a date window computed in SQL is the business deciding inside the query. Heuristic by design — it is warn, and it is the phase-2 judge's main input."},
	"ARC-06": {"ARC-06", SevWarn, "go", "domain rule inside the transport layer",
		"A conditional over an entity field beyond validation and error mapping, arithmetic on a domain value, or a time comparison deciding an outcome."},

	"ARC-07": {"ARC-07", SevError, "both", "layer imports a package it is not allowed to reach",
		"A denied edge is about the project's own layers; this one names a package outright, which is how a rule like \"the data layer never logs\" is expressed. It matches the raw import path, so third-party packages are in reach."},

	"ARC-10": {"ARC-10", SevError, "web", "one feature imports another",
		"A feature is a unit you can read, change and delete on its own. The moment two of them import each other they are one feature with two folders. What both need moves to components/ or lib/."},
	"ARC-11": {"ARC-11", SevError, "web", "shared code imports a feature or a route",
		"components/ and lib/ are what features are built from; the arrow only points one way. A shared component reaching into a feature is a feature component filed in the wrong place."},
	"ARC-12": {"ARC-12", SevError, "web", "HTTP call outside the layer that owns the wire",
		"Base URL, auth header, refresh and error envelope live in one client. A fetch outside it is a request none of that applies to — this project already shipped an invalid host that way."},
	"ARC-13": {"ARC-13", SevError, "web", "canonical component bypassed",
		"One solved problem, one component. The rows are the project's canonical table, declared in .quality-gate.yml so the table has one home; what eslint can already express is not repeated here."},
	"ARC-14": {"ARC-14", SevWarn, "web", "business rule computed inside a component",
		"Date arithmetic, money arithmetic or a state derived inline. A component renders what it is given; the rule belongs in a hook, a service, or the backend that already owns it. Heuristic by design, hence warn."},

	"GATE-01": {"GATE-01", SevError, "both", "suppression without a reason",
		"`quality-gate:allow RULE — reason`. The reason is the point: it is the review comment the next reader needs."},
	"GATE-02": {"GATE-02", SevError, "both", "stale baseline entry",
		"The code this entry excused is gone. Regenerate the baseline so it can only shrink."},
}

func ruleIDs() []string {
	ids := make([]string, 0, len(catalog))
	for id := range catalog {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// severityOf applies the project's overrides on top of the catalog default.
func severityOf(cfg *Config, id string) Severity {
	if s, ok := cfg.Severity[id]; ok {
		return Severity(s)
	}
	return catalog[id].Sev
}

func explain(id string) (string, error) {
	r, ok := catalog[strings.ToUpper(id)]
	if !ok {
		return "", fmt.Errorf("unknown rule %q — try one of: %s", id, strings.Join(ruleIDs(), ", "))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s  [%s, ruleset: %s]\n\n", r.ID, r.Sev, r.Lang)
	fmt.Fprintf(&b, "%s\n\n%s\n", r.Summary, wrap(r.Explain, 76))
	return b.String(), nil
}

func wrap(s string, width int) string {
	var out strings.Builder
	line := 0
	for _, w := range strings.Fields(s) {
		if line > 0 && line+1+len(w) > width {
			out.WriteByte('\n')
			line = 0
		} else if line > 0 {
			out.WriteByte(' ')
			line++
		}
		out.WriteString(w)
		line += len(w)
	}
	return out.String()
}
