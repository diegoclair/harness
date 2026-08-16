package main

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

// Tests are exempt from every ARC rule: a test legitimately reaches across the
// layers it is exercising, and blocking that would shape production code to
// suit the gate.
func checkArchitecture(cfg *Config, f *File, add func(Finding)) {
	if f.IsTest {
		return
	}
	checkImportEdges(cfg, f, add)
	checkForbiddenImports(cfg, f, add)
	checkCanonical(cfg, f, add)
	checkContextIsolation(cfg, f, add)
	checkWebArchitecture(cfg, f, add)

	switch cfg.layerOf(f.Path) {
	case cfg.Heuristics.SQLLayer:
		checkSQLDomainRules(cfg, f, add)
	case cfg.Heuristics.HandlerLayer:
		checkHandlerDomainRules(cfg, f, add)
	}
}

// importPath resolves an import to a repo-relative path, so layer patterns can
// be written once and match both the file and what it imports. A Go import is
// absolute under the module; a web one is relative to the importing file or
// carries a bundler alias.
func (c *Config) importPath(f *File, imp string) string {
	if f.Lang == LangWeb {
		return c.webImportPath(f.Path, imp)
	}
	if c.Module == "" || !strings.HasPrefix(imp, c.Module) {
		return ""
	}
	return strings.TrimPrefix(strings.TrimPrefix(imp, c.Module), "/")
}

func (c *Config) webImportPath(from, imp string) string {
	if strings.HasPrefix(imp, ".") {
		return path.Clean(path.Join(path.Dir(from), imp))
	}
	best, bestLen := "", -1
	for prefix, target := range c.Aliases {
		if strings.HasPrefix(imp, prefix) && len(prefix) > bestLen {
			best, bestLen = target+strings.TrimPrefix(imp, prefix), len(prefix)
		}
	}
	if bestLen < 0 {
		return "" // a bare specifier is a package, not a layer
	}
	return path.Clean(best)
}

func (c *Config) allowed(layer, importedPath string) bool {
	for _, pattern := range c.Allow[layer] {
		if globMatch(pattern, importedPath) {
			return true
		}
	}
	return false
}

func checkImportEdges(cfg *Config, f *File, add func(Finding)) {
	from := cfg.layerOf(f.Path)
	if from == "" {
		return
	}
	for _, edge := range cfg.Deny {
		if edge.From != from {
			continue
		}
		for _, imp := range f.Imports {
			rel := cfg.importPath(f, imp.Path)
			if rel == "" || cfg.allowed(from, rel) {
				continue
			}
			to := cfg.layerOf(rel)
			if to == "" || to == from || !contains(edge.To, to) {
				continue
			}
			add(Finding{
				Rule: edge.Rule, Sev: severityOf(cfg, edge.Rule), File: f.Path, Line: imp.Line,
				Message:   fmt.Sprintf("%s imports %s (%s → %s is denied)", from, imp.Path, from, to),
				Signature: signature(edge.Rule, f.Path, imp.Path),
			})
		}
	}
}

func checkForbiddenImports(cfg *Config, f *File, add func(Finding)) {
	layer := cfg.layerOf(f.Path)
	if layer == "" {
		return
	}
	for _, pattern := range cfg.Forbid[layer] {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		for _, imp := range f.Imports {
			if !re.MatchString(imp.Path) {
				continue
			}
			add(Finding{
				Rule: "ARC-07", Sev: severityOf(cfg, "ARC-07"), File: f.Path, Line: imp.Line,
				Message:   fmt.Sprintf("%s must not import %s", layer, imp.Path),
				Signature: signature("ARC-07", f.Path, imp.Path),
			})
		}
	}
}

func checkContextIsolation(cfg *Config, f *File, add func(Finding)) {
	own := cfg.contextOf(f.Path)
	if own == "" {
		return
	}
	for _, imp := range f.Imports {
		rel := cfg.importPath(f, imp.Path)
		if rel == "" {
			continue
		}
		other := cfg.contextOf(rel)
		if other == "" || other == own || cfg.allowed(own, rel) {
			continue
		}
		rule := cfg.ContextRule
		add(Finding{
			Rule: rule, Sev: severityOf(cfg, rule), File: f.Path, Line: imp.Line,
			Message:   fmt.Sprintf("%s imports %s — %s", own, other, isolationAdvice[rule]),
			Signature: signature(rule, f.Path, imp.Path),
		})
	}
}

var isolationAdvice = map[string]string{
	"ARC-02": "bounded contexts stay separate; cross-context work goes through an adapter wired at the composition root",
	"ARC-10": "one feature never reaches into another; what both need moves to components/ or lib/",
}

var (
	sqlShapeRe     = regexp.MustCompile(`(?is)\b(select|insert\s+into|update|delete\s+from)\b.*\b(from|set|values|where)\b`)
	caseWhenRe     = regexp.MustCompile(`(?is)\bcase\s+when\b`)
	stateLiteralRe = regexp.MustCompile(`(?is)\b(status|state|kind|type|situation)\s*(=|<>|!=|\bin\b)\s*\(?\s*'`)
	dateWindowRe   = regexp.MustCompile(`(?is)\binterval\s+'|\bnow\(\)\s*[-+]|\bcurrent_date\s*[-+]`)
)

func checkSQLDomainRules(cfg *Config, f *File, add func(Finding)) {
	for _, lit := range f.Strings {
		if !sqlShapeRe.MatchString(lit.Value) {
			continue
		}
		emit := func(what string) {
			add(Finding{
				Rule: "ARC-05", Sev: severityOf(cfg, "ARC-05"), File: f.Path, Line: lit.Line,
				Message:   what + " — the repository answers where the data is, not what the business decides",
				Signature: signature("ARC-05", f.Path, what, firstLine(lit.Value)),
			})
		}
		if caseWhenRe.MatchString(lit.Value) && !aggregatedCase(lit.Value) {
			emit("CASE WHEN in SQL decides a business state")
		}
		if stateLiteralRe.MatchString(lit.Value) {
			emit("business-state literal compared in SQL")
		}
		if dateWindowRe.MatchString(lit.Value) && !seriesSpineRe.MatchString(lit.Value) {
			emit("date window computed in SQL")
		}
	}
}

var (
	// SUM(CASE WHEN …) and COUNT(*) FILTER (WHERE …) are how SQL pivots rows
	// into columns; the decision they encode was made by whoever wrote the
	// column list, not by the query.
	aggregatedCaseRe = regexp.MustCompile(`(?is)\b(sum|count|avg|max|min)\s*\(\s*(case|\*)`)
	paramCaseRe      = regexp.MustCompile(`(?is)\bcase\s+when\s+\$\d`)
	seriesSpineRe    = regexp.MustCompile(`(?is)\bgenerate_series\s*\(`)
)

func aggregatedCase(sql string) bool {
	return aggregatedCaseRe.MatchString(sql) || paramCaseRe.MatchString(sql)
}

var timeDecisionRe = regexp.MustCompile(`time\.Now\(\)|\.After\(|\.Before\(|\.Sub\(`)

var renderOrParseFn = regexp.MustCompile(`^(parse|to|from|validate|map|build|render|new)[A-Z_]|^(Parse|To|From|Validate|Map|Build|Render|New)`)

// enclosingFunc names the function a line sits in, so the rule can tell a
// mapper from a decision.
func enclosingFunc(f *File, line int) string {
	for _, fn := range f.Funcs {
		if line >= fn.Line && line <= fn.EndLine {
			return fn.Name
		}
	}
	return ""
}

func checkHandlerDomainRules(cfg *Config, f *File, add func(Finding)) {
	// A viewmodel package is the rendering half of transport by definition.
	if strings.Contains(f.Path, "/viewmodel") {
		return
	}
	for _, c := range f.Conds {
		if c.Guard || renderOrParseFn.MatchString(enclosingFunc(f, c.Line)) {
			continue
		}
		emit := func(what string) {
			add(Finding{
				Rule: "ARC-06", Sev: severityOf(cfg, "ARC-06"), File: f.Path, Line: c.Line,
				Message:   what + " — a handler parses, delegates and renders",
				Signature: signature("ARC-06", f.Path, c.Src),
			})
		}
		switch {
		case timeDecisionRe.MatchString(c.Src):
			emit("time comparison decides an outcome in the transport layer")
		case len(c.Fields) > 0:
			emit(fmt.Sprintf("conditional over entity field %s in the transport layer", c.Fields[0]))
		}
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
