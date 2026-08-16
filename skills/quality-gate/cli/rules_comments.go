package main

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

func checkComments(cfg *Config, f *File, add func(Finding)) {
	for _, c := range f.Comments {
		// A directive above the comment it excuses lands in the same group;
		// stripping instead of skipping keeps the rest of it under the rules.
		if stripped, ok := withoutDirectives(c); ok {
			checkComment(cfg, f, stripped, add)
		}
	}
}

func checkComment(cfg *Config, f *File, c Comment, add func(Finding)) {
	emit := func(rule, msg string) {
		add(Finding{
			Rule: rule, Sev: severityOf(cfg, rule), File: f.Path, Line: c.Line,
			Message: msg, Signature: signature(rule, f.Path, c.Text, c.Target),
		})
	}
	if commentedOutCode(c.Lines) {
		emit("CMT-06", "commented-out code — delete it, git remembers")
	}
	if budget := cfg.budget(c.Pos); budget > 0 && c.Span() > budget {
		emit("CMT-01", fmt.Sprintf("comment block is %d lines; the budget for a %s comment is %d",
			c.Span(), c.Pos, budget))
	}

	// Everything below reads the prose only: an example block is code the
	// author is showing on purpose, product copy and all.
	c.Lines = proseLines(c.Lines)
	if len(c.Lines) == 0 {
		return
	}
	c.Text = strings.Join(c.Lines, " ")
	text := strings.ToLower(c.Text)

	if reason, ok := notEnglish(c.Text); ok {
		emit("CMT-04", "comment is not in English ("+reason+")")
	}
	// A quoted marker is being mentioned, not used — this rule's own doc says
	// "moved from" while describing the bug that phrase caused.
	if marker, ok := containsWord(quotedRe.ReplaceAllString(text, " "), historyMarkers); ok && !f.IsTest {
		emit("CMT-05", fmt.Sprintf("comment carries history (%q) — git owns that", marker))
	}

	// One question — does this add anything the code did not say — so only the
	// most specific of the three is reported.
	described := true
	switch {
	case isSectionLabel(c):
		described = false
	case c.Pos == PosDecl:
		// On a declaration CMT-09 is the whole question, and a second go from
		// the narration detector punished a constraint for naming its field.
		if hasConstraint(text) {
			described = false
			break
		}
		emit("CMT-09", fmt.Sprintf("%s is described, not constrained — the name already said it; "+
			"keep only a unit, invariant, format, external contract or contrast", quoteTarget(c.Target)))
	case narratesBehavior(cfg, c) != "":
		emit("CMT-02", "comment narrates the code below ("+narratesBehavior(cfg, c)+
			") — state the purpose or delete it")
	case restatesName(c):
		emit("CMT-07", fmt.Sprintf("doc only restates %s — delete it, or say why the symbol exists",
			quoteTarget(c.Target)))
	default:
		described = false
	}

	if c.Pos == PosBody && !described && f.isAdded(c.Line) {
		emit("CMT-03", "comment inside a function body — is it explaining what the code already says?")
	}
}

func quoteTarget(name string) string {
	if name == "" {
		return "the declaration"
	}
	return "`" + name + "`"
}

var narrationOpeners = []string{
	"now ", "then ", "first ", "next ", "here we ", "this function ", "this method ",
	"loop over ", "iterate ", "call the ", "set the ", "get the ", "check if ",
	"returns the ", "we then ", "start by ", "finally ", "create the ", "build the ",
	"add the ", "remove the ", "update the ", "convert the ", "parse the ", "increment ",
}

// whyMarkers are what turns a sentence about the code into a sentence about the
// reason for the code. A comment carrying one is purpose, whatever verb it
// opened with.
var whyMarkers = []string{
	"because", "otherwise", "so", "since", "to avoid", "must", "never",
	"only", "would", "cannot", "can't", "does not", "doesn't", "—", "instead of",
	"at the cost", "on purpose", "deliberately", "guard", "gotcha",
}

func narratesBehavior(cfg *Config, c Comment) string {
	switch c.Pos {
	case PosPackage, PosFunc:
		// A contract legitimately states what it promises; CMT-07 covers the
		// case where the promise is only the name spelled out.
		return ""
	}
	contract := c.Pos == PosInterface || c.Pos == PosType
	words := contentWords(c.Text)
	if len(words) < 3 {
		return ""
	}
	lower := strings.ToLower(strings.TrimSpace(c.Text))
	if _, ok := containsAny(lower, whyMarkers); ok {
		return ""
	}
	if opener, ok := hasPrefixAny(lower, narrationOpeners); ok {
		return "opens with " + strings.TrimSpace(opener)
	}
	if contract {
		return ""
	}
	if r := overlapRatio(words, toSet(c.NextIdents)); r >= cfg.threshold("comments.overlap_ratio") {
		return fmt.Sprintf("%.0f%% of its words are identifiers from the code below", r*100)
	}
	return ""
}

// genericVerbs carry no information beyond the symbol name they sit on.
var genericVerbs = map[string]bool{
	"return": true, "get": true, "set": true, "creat": true, "build": true,
	"make": true, "new": true, "hold": true, "represent": true, "contain": true,
	"use": true, "do": true, "provid": true, "defin": true, "stor": true,
	"valu": true, "field": true, "struct": true, "function": true, "method": true,
}

func restatesName(c Comment) bool {
	if c.Target == "" || c.Span() > 1 || isSectionLabel(c) {
		return false
	}
	words := contentWords(c.Text)
	if len(words) == 0 {
		return false
	}
	name := toSet(splitIdent(c.Target))
	for _, w := range words {
		if !name[w] && !genericVerbs[w] {
			return false
		}
	}
	return true
}

var constraintMarkers = []string{
	// unit or scale
	"cents", "reais", "second", "millisecond", "minute", "hour", "day", "byte",
	"utc", "percent", "timezone", "offset",
	// invariant and zero semantics
	"must", "never", "only", "always", "at most", "at least", "cannot", "can't",
	"required", "optional", "absent", "empty", "zero", "unset", "nil", "null",
	"default", "fallback", "fall back", "falls back", "not", "immutable",
	"once", "max", "min", "limit", "unrecoverable", "read-only", "write-only",
	"ignored", "overrides", "precedence", "nullable", "sentinel", "per",
	"neither", "nor", "non", "excludes", "excluding", "derived", "computed",
	"no", "without", "filled", "even", "unless", "except", "beyond", "all-time",
	// format
	"rfc", "e.164", "iso", "format", "lowercase", "uppercase", "slug", "digits",
	"regex", "pattern", "base64", "uuid", "encoded",
	// external contract
	"api", "webhook", "upstream", "sdk", "header", "provider returns", "returns this",
	// reference
	"spec", "adr", "see", "rfc3339", "e.g.", "i.e.",
}

// A path, placeholder or numeric range is a format even with no marker word.
var formatLiteralRe = regexp.MustCompile(`/[a-z_:<{][\w:<>{}$/-]*|<[a-z][\w-]*>|\$\{[^}]+\}|\b\d+\s*[-–]\s*\d+\b`)

// isSectionLabel spots the short noun phrase that groups a run of declarations
// ("// Instagram Showcase.") rather than describing the one below it. It says
// nothing about the declaration, so it cannot be restating it.
// A rule of repeated punctuation is a divider whatever words ride on it.
var dividerRe = regexp.MustCompile(`[-=_*~#─━═]{4,}`)

func isDivider(c Comment) bool {
	for _, l := range c.Lines {
		if dividerRe.MatchString(l) {
			return true
		}
	}
	return false
}

func isSectionLabel(c Comment) bool {
	words := contentWords(c.Text)
	if c.Span() > 1 || len(words) == 0 || len(words) > 6 {
		return false
	}
	if isDivider(c) {
		return true
	}
	// Go's doc convention opens with the symbol's own name, so a comment whose
	// first word is the declaration's first word is documenting it, not
	// labelling the group it sits in.
	if name := splitIdent(c.Target); len(name) > 0 && words[0] == stem(name[0]) {
		return false
	}
	for _, w := range words {
		if descriptiveVerbs[w] || genericVerbs[w] {
			return false
		}
	}
	return true
}

var descriptiveVerbs = map[string]bool{
	"was": true, "hold": true, "carry": true,
	"mean": true, "represent": true, "count": true, "address": true,
	"return": true, "contain": true, "store": true, "keep": true, "point": true,
}

// A numeric range is a constraint the name cannot carry: `0–5 short options`
// says how many, and the field is called `quick_replies`.
var numericRangeRe = regexp.MustCompile(`\b\d+\s*[-–—]\s*\d+\b`)

func hasConstraint(lowerText string) bool {
	if _, ok := containsWord(lowerText, constraintMarkers); ok {
		return true
	}
	if formatLiteralRe.MatchString(lowerText) {
		return true
	}
	if numericRangeRe.MatchString(lowerText) {
		return true
	}
	_, ok := containsAny(lowerText, whyMarkers)
	return ok
}

// Markers are deliberately narrow. "used to" and "no longer" also form ordinary
// English about the present ("the domain used to serve reads", "can no longer
// change the status because the response started") — a marker that fires on
// those trains the reader to ignore the rule.
var historyMarkers = []string{
	"used to be", "we used to", "it used to", "this used to",
	"previously this", "previously it", "previously we", "previously the",
	"was renamed", "refactored", "deprecated in",
}

// accentedWord returns the diacritic that proves the comment is not English. A
// single capitalised accented word inside a long English sentence is a proper
// noun — "slots render in São Paulo time" is English, and firing on it teaches
// the reader to ignore the rule.
func accentedWord(text string) (string, bool) {
	var accented []string
	capitalisedOnly := true
	for _, w := range strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && r != '\''
	}) {
		for _, r := range w {
			// Latin only: the π in `0..π → fan outward` is maths, not Portuguese.
			if r > unicode.MaxASCII && unicode.IsLetter(r) && unicode.Is(unicode.Latin, r) {
				accented = append(accented, string(r))
				if !unicode.IsUpper([]rune(w)[0]) {
					capitalisedOnly = false
				}
				break
			}
		}
	}
	if len(accented) == 0 {
		return "", false
	}
	if capitalisedOnly && len(accented) == 1 && len(contentWords(text)) >= 5 {
		return "", false
	}
	return accented[0], true
}

var quotedRe = regexp.MustCompile("\"[^\"]*\"|`[^`]*`|'[^']*'")

var ptStopwords = []string{"que", "não", "nao", "para", "porque", "então", "entao",
	"isso", "aqui", "deve", "pois", "quando", "cada", "mesmo", "sempre", "está", "esta"}

// notEnglish trusts a diacritic on its own; without one it needs two Portuguese
// function words, because a single "para" or "que" also occurs in English text.
func notEnglish(text string) (string, bool) {
	// A quoted word is being mentioned, not used: English prose about
	// Portuguese words must not read as Portuguese.
	text = quotedRe.ReplaceAllString(text, " ")
	if r, ok := accentedWord(text); ok {
		return "accented letter " + r, true
	}
	hits := 0
	for _, w := range ptStopwords {
		if hasWord(strings.ToLower(text), w) {
			hits++
		}
	}
	if hits >= 2 {
		return "Portuguese function words", true
	}
	return "", false
}

// proseLines keeps the sentences of a comment. A fenced block and an `@example`
// body are code showing how the symbol is called, and a doc tag is a machine
// annotation whose `{Type}` happens to end in a brace: none of the three is
// prose about the code, and reading them as prose reported product copy as a
// non-English comment and a usage sample as leftover code.
func proseLines(lines []string) []string {
	var out []string
	fenced, example := false, false
	for _, l := range lines {
		t := strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(t, "```"):
			fenced = !fenced
		case strings.HasPrefix(t, "@"):
			example = strings.HasPrefix(t, "@example")
		case !fenced && !example:
			out = append(out, l)
		}
	}
	return out
}

func commentedOutCode(lines []string) bool {
	strong, weak := 0, 0
	introduced := false
	for _, l := range proseLines(lines) {
		l = strings.TrimSpace(l)
		if l == "" || readsAsProse(l) {
			continue
		}
		// Prose ending in a colon is introducing a sample: what follows is code
		// to write, not code left behind.
		if strings.HasSuffix(l, ":") {
			introduced = true
			continue
		}
		if introduced {
			continue
		}
		switch {
		case strings.Contains(l, ":="), strings.Contains(l, "=>"),
			strings.HasSuffix(l, "{"), strings.HasSuffix(l, "}"):
			strong++
		case strings.HasSuffix(l, ";") && looksLikeStatement(l):
			strong++
		case codeStmtRe.MatchString(l):
			weak++
		}
	}
	return strong >= 1 || weak >= 2
}

// A trailing semicolon alone is not code: English prose uses one too. It only
// counts alongside the shape of a statement — short, with a call or an
// assignment in it.
func looksLikeStatement(l string) bool {
	return strings.HasSuffix(l, ");") || assignmentRe.MatchString(l)
}

var assignmentRe = regexp.MustCompile(`\w\s*=[^=]`)

// readsAsProse spots the sentence a documentation line is, whatever punctuation
// it happens to end on. An API line closing on `{ url }` and a sentence closing
// on a semicolon were both read as leftover code before this.
func readsAsProse(l string) bool {
	return strings.Contains(l, "—") || strings.Contains(l, " – ") ||
		strings.Contains(l, ". ") || strings.HasSuffix(l, ".")
}

var codeStmtRe = regexp.MustCompile(`^(if|for|return|func|var|const|switch|case|else|import|export)\b.*[)\]}]$`)

func hasPrefixAny(s string, prefixes []string) (string, bool) {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return p, true
		}
	}
	return "", false
}
