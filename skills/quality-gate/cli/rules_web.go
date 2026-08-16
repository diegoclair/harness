package main

import (
	"fmt"
	"regexp"
	"strings"
)

func checkWebArchitecture(cfg *Config, f *File, add func(Finding)) {
	if f.Lang != LangWeb {
		return
	}
	checkHTTPBoundary(cfg, f, add)
	checkComponentDomainRules(cfg, f, add)
}

// ARC-12 — an HTTP call outside the layers that own the wire. The layer list is
// the project's, so a repo that keeps its client somewhere else names it there.
var httpPackages = map[string]bool{
	"axios": true, "ky": true, "got": true, "superagent": true,
	"node-fetch": true, "redaxios": true,
}

func checkHTTPBoundary(cfg *Config, f *File, add func(Finding)) {
	layers := cfg.Heuristics.HTTPLayers
	if len(layers) == 0 || cfg.hasLayer(layers, cfg.layerOf(f.Path)) {
		return
	}
	emit := func(line int, what string) {
		add(Finding{
			Rule: "ARC-12", Sev: severityOf(cfg, "ARC-12"), File: f.Path, Line: line,
			Message: what + " — HTTP lives in " + strings.Join(layers, "/") +
				"; a component asks a hook, the hook asks a service",
			Signature: signature("ARC-12", f.Path, what),
		})
	}
	// The rule is about a call, not an import: `import { HTTPError } from "ky"`
	// to narrow an error is a type reaching for a name, not a request.
	for i, t := range f.Tokens {
		if t.Kind != 'i' || i+1 >= len(f.Tokens) {
			continue
		}
		next := f.Tokens[i+1].Text
		switch {
		case t.Text == "fetch" && next == "(" && f.Tokens[max(0, i-1)].Text != ".":
			emit(t.Line, "calls fetch() directly")
		case httpPackages[t.Text] && (next == "(" || next == "."):
			emit(t.Line, "calls "+t.Text+" directly")
		}
	}
}

// ARC-13 — the canonical-components table. Every row is the project's; the rule
// only knows how to ask a row whether a line or an element breaks it.
func checkCanonical(cfg *Config, f *File, add func(Finding)) {
	for i := range cfg.Canonical {
		row := &cfg.Canonical[i]
		if row.excepts(f.Path) {
			continue
		}
		if row.Scope == "element" {
			checkCanonicalElements(cfg, f, row, add)
			continue
		}
		for n, line := range f.codeSource() {
			if row.hits(line) {
				emitCanonical(cfg, f, row, n+1, line, add)
			}
		}
	}
}

func checkCanonicalElements(cfg *Config, f *File, row *CanonicalRow, add func(Finding)) {
	for _, el := range f.Elements {
		if row.elementRe != nil && !row.elementRe.MatchString(el.Name) {
			continue
		}
		if row.hits(el.Open) {
			emitCanonical(cfg, f, row, el.Line, "<"+el.Name, add)
		}
	}
}

func (r *CanonicalRow) hits(text string) bool {
	if r.matchRe != nil && !r.matchRe.MatchString(text) {
		return false
	}
	if !r.forbidRe.MatchString(text) {
		return false
	}
	return r.unlessRe == nil || !r.unlessRe.MatchString(text)
}

// A row polices everything unless it says otherwise, and `except` always wins:
// a rule scoped to a tree still has holes inside it.
func (r *CanonicalRow) excepts(path string) bool {
	for _, p := range r.Except {
		if globMatch(p, path) {
			return true
		}
	}
	if len(r.Only) == 0 {
		return false
	}
	for _, p := range r.Only {
		if globMatch(p, path) {
			return false
		}
	}
	return true
}

func emitCanonical(cfg *Config, f *File, row *CanonicalRow, line int, where string, add func(Finding)) {
	msg := row.Message
	if msg == "" {
		msg = "breaks the canonical rule " + row.ID
	}
	// A row that codifies a judgment call declares itself a question.
	sev := severityOf(cfg, "ARC-13")
	if row.Sev == string(SevWarn) {
		sev = SevWarn
	}
	add(Finding{
		Rule: "ARC-13", Sev: sev, File: f.Path, Line: line,
		Message:   row.ID + ": " + msg,
		Signature: signature("ARC-13", f.Path, row.ID, strings.TrimSpace(where)),
	})
}

// ARC-14 — a business rule computed inside a component. Money and date
// arithmetic are read off the code lines; a derived state is read off the
// conditionals, because that is what a decision is.
var (
	dateMathRe = regexp.MustCompile(`(?:\.getTime\(\)|Date\.now\(\))\s*[-+]|` +
		`[-+]\s*(?:new Date\(|\w+\.getTime\(\)|Date\.now\(\))|` +
		`\b(?:86400000|3600000|604800000)\b|` +
		`\b60\s*\*\s*60\s*\*\s*1000\b|\b24\s*\*\s*60\s*\*\s*60\b`)
	// A money identifier standing next to an arithmetic operator. Adjacency is
	// the whole point: a `/` closing a JSX tag on the same line is not maths.
	moneyName   = `\w*(?:price|amount|valor|preco|cents|centavos|subtotal|discount|desconto)\w*`
	moneyMathRe = regexp.MustCompile(`(?i)(?:` + moneyName + `\s*[-+*/]\s*[\w(])|(?:[\w)]\s*[-+*/]\s*` + moneyName + `\b)`)

	formattingRe = regexp.MustCompile(`(?i)tofixed|tolocale|intl\.|format|currency|mask`)

	statusCompareRe = regexp.MustCompile(`(?i)\b\w*(?:status|situacao|situation|state)\w*\s*(?:===|!==|==|!=)\s*["'\x60]`)
	comparisonRe    = regexp.MustCompile(`===|!==|==|!=|<=|>=`)
	andOrRe         = regexp.MustCompile(`&&|\|\|`)
	dateRefRe       = regexp.MustCompile(`\bDate\b|\.getTime\(\)|\bisBefore\b|\bisAfter\b`)
)

// "Inline in a component" is literal: a rule already extracted into a named
// helper is a rule with a home, whatever folder that home sits in.
func componentRanges(f *File) []lineRange {
	var out []lineRange
	for _, fn := range f.Funcs {
		if fn.Component {
			out = append(out, lineRange{fn.Line, fn.EndLine})
		}
	}
	return out
}

func checkComponentDomainRules(cfg *Config, f *File, add func(Finding)) {
	if !cfg.hasLayer(cfg.Heuristics.ComponentLayers, cfg.layerOf(f.Path)) {
		return
	}
	inside := componentRanges(f)
	if len(inside) == 0 {
		return
	}
	emit := func(line int, src, what string) {
		add(Finding{
			Rule: "ARC-14", Sev: severityOf(cfg, "ARC-14"), File: f.Path, Line: line,
			Message:   what + " — a component renders; the rule belongs in a hook, a service or the backend",
			Signature: signature("ARC-14", f.Path, what, src),
		})
	}
	for n, line := range f.codeSource() {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || !inAnyRange(inside, n+1) || formattingRe.MatchString(trimmed) {
			continue
		}
		bare := quotedSpanRe.ReplaceAllString(trimmed, `""`)
		switch {
		case dateMathRe.MatchString(bare):
			emit(n+1, trimmed, "date arithmetic inline in a component")
		case moneyMathRe.MatchString(bare):
			emit(n+1, trimmed, "money arithmetic inline in a component")
		}
	}
	// The comparison alone is how a component picks a badge, which is
	// rendering; a decision needs the status combined with something else.
	for _, c := range f.Conds {
		if c.Guard || !inAnyRange(inside, c.Line) {
			continue
		}
		if !statusCompareRe.MatchString(c.Src) || !andOrRe.MatchString(c.Src) {
			continue
		}
		if len(comparisonRe.FindAllString(c.Src, -1)) < 2 && !dateRefRe.MatchString(c.Src) {
			continue
		}
		emit(c.Line, c.Src, "a state is decided inline in a component")
	}
}

// Quoted spans hold class names and copy, never arithmetic.
var quotedSpanRe = regexp.MustCompile(`["'\x60][^"'\x60]*["'\x60]`)

// CPX-05 — a component above the size or hook budget.
func checkComponentSize(cfg *Config, f *File, add func(Finding)) {
	lineLimit := int(cfg.threshold("cpx.component_lines"))
	hookLimit := int(cfg.threshold("cpx.component_hooks"))
	for _, fn := range f.Funcs {
		if !fn.Component {
			continue
		}
		emit := func(msg string) {
			add(Finding{
				Rule: "CPX-05", Sev: severityOf(cfg, "CPX-05"), File: f.Path, Line: fn.Line,
				Message: msg, Signature: signature("CPX-05", f.Path, fn.Name),
			})
		}
		if n := fn.EndLine - fn.Line + 1; n > lineLimit {
			emit(fmt.Sprintf("%s is %d lines (limit %d) — is a piece of it a component of its own?",
				fn.Name, n, lineLimit))
		}
		if fn.Hooks > hookLimit {
			emit(fmt.Sprintf("%s calls %d hooks (limit %d) — usually a custom hook waiting to be named",
				fn.Name, fn.Hooks, hookLimit))
		}
	}
}
