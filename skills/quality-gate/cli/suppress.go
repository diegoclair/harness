package main

import (
	"hash/fnv"
	"regexp"
	"strconv"
	"strings"
)

const directivePrefix = "quality-gate:"

// directiveRe accepts an em dash or a double hyphen before the reason, because
// the reason is the point and a keyboard should not stand in its way.
var directiveRe = regexp.MustCompile(`^quality-gate:allow\s+([A-Z]+-\d+)\s*(?:—|--)?\s*(.*)$`)

type suppression struct {
	Rule string
	// Reach starts at the end of the comment group, not at the directive: a
	// reason spanning two lines would otherwise run out before the code, and in
	// JSX the directive often cannot sit adjacent to the line it excuses.
	Line   int
	Reason string
}

func isDirectiveLine(l string) bool {
	return strings.HasPrefix(strings.TrimSpace(l), directivePrefix)
}

// withoutDirectives drops the directive lines from a comment group and reports
// whether anything is left to judge.
func withoutDirectives(c Comment) (Comment, bool) {
	kept := make([]string, 0, len(c.Lines))
	first := -1
	for i, l := range c.Lines {
		if isDirectiveLine(l) {
			continue
		}
		if first < 0 {
			first = i
		}
		kept = append(kept, l)
	}
	if len(kept) == 0 {
		return c, false
	}
	// Only a dropped line can move the span; recomputing it unconditionally
	// understated every JSDoc by its two delimiter lines.
	if len(kept) != len(c.Lines) {
		c.Line += first
		c.EndLine = c.Line + len(kept) - 1
		c.Delims = 0
	}
	c.Lines = kept
	c.Text = strings.Join(kept, " ")
	return c, true
}

// collectSuppressions reads the allow directives of a file and reports the ones
// missing a reason as GATE-01.
func collectSuppressions(cfg *Config, f *File, add func(Finding)) []suppression {
	var out []suppression
	for _, c := range f.Comments {
		for i, raw := range c.Lines {
			line := strings.TrimSpace(raw)
			if !strings.HasPrefix(line, directivePrefix) {
				continue
			}
			m := directiveRe.FindStringSubmatch(line)
			if m == nil {
				add(Finding{
					Rule: "GATE-01", Sev: severityOf(cfg, "GATE-01"), File: f.Path, Line: c.Line + i,
					Message:   "malformed directive — expected `quality-gate:allow RULE — reason`",
					Signature: signature("GATE-01", f.Path, line),
				})
				continue
			}
			rule, reason := m[1], strings.TrimSpace(m[2])
			if _, known := catalog[rule]; !known {
				add(Finding{
					Rule: "GATE-01", Sev: severityOf(cfg, "GATE-01"), File: f.Path, Line: c.Line + i,
					Message:   "directive names unknown rule " + rule,
					Signature: signature("GATE-01", f.Path, line),
				})
				continue
			}
			if reason == "" {
				add(Finding{
					Rule: "GATE-01", Sev: severityOf(cfg, "GATE-01"), File: f.Path, Line: c.Line + i,
					Message:   "suppression of " + rule + " has no reason — the reason is the review comment the next reader needs",
					Signature: signature("GATE-01", f.Path, line),
				})
				continue
			}
			out = append(out, suppression{Rule: rule, Line: c.EndLine, Reason: reason})
		}
	}
	return out
}

// suppressed reports whether a directive covers a finding. A directive speaks
// for its own line and the two below it, which is the construct it sits on.
func suppressed(sups []suppression, f Finding) bool {
	const reach = 2
	for _, s := range sups {
		if s.Rule == f.Rule && f.Line >= s.Line && f.Line <= s.Line+reach {
			return true
		}
	}
	return false
}

// signature anchors a finding to its content, so the baseline survives a line
// moving and forgets a line changing.
func signature(parts ...string) string {
	h := fnv.New64a()
	for _, p := range parts {
		h.Write([]byte(normalizeSig(p)))
		h.Write([]byte{0})
	}
	return strconv.FormatUint(h.Sum64(), 36)
}

var spaceRe = regexp.MustCompile(`\s+`)

func normalizeSig(s string) string {
	return strings.TrimSpace(spaceRe.ReplaceAllString(s, " "))
}
