package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// The report is written for an agent to act on: one line per finding, path and
// line first so an editor can jump, then the rule, then what to do about it.
func reportText(w io.Writer, res *checkResult, quiet bool) {
	width := 0
	for _, f := range res.Findings {
		if n := len(location(f)); n > width {
			width = n
		}
	}
	for _, f := range res.Findings {
		if quiet && f.Sev != SevError {
			continue
		}
		fmt.Fprintf(w, "%-*s %s %-5s %s\n", width, location(f), f.Rule, f.Sev, f.Message)
	}

	for _, e := range res.ParseErrors {
		fmt.Fprintf(w, "skipped (unparseable): %s\n", e)
	}

	if len(res.Findings) > 0 {
		fmt.Fprintln(w)
	}
	fmt.Fprintf(w, "%d finding(s): %d error(s), %d warning(s) — %d file(s) scanned, %d indexed",
		len(res.Findings), res.Errors, res.Warnings, res.Scanned, res.Indexed)
	if res.Excused > 0 {
		fmt.Fprintf(w, " (baseline excused %d)", res.Excused)
	}
	if res.Stale > 0 {
		fmt.Fprintf(w, " (%d stale baseline entries)", res.Stale)
	}
	fmt.Fprintln(w)
}

func location(f Finding) string {
	if f.Line == 0 {
		if f.File == "." {
			return "(delivery)"
		}
		return f.File
	}
	return fmt.Sprintf("%s:%d", f.File, f.Line)
}

type jsonReport struct {
	Findings []Finding `json:"findings"`
	Summary  struct {
		Errors   int `json:"errors"`
		Warnings int `json:"warnings"`
		Scanned  int `json:"scanned"`
		Indexed  int `json:"indexed"`
		Excused  int `json:"baseline_excused"`
		Stale    int `json:"baseline_stale"`
	} `json:"summary"`
	Skipped []string `json:"skipped,omitempty"`
}

func reportJSON(w io.Writer, res *checkResult) error {
	r := jsonReport{Findings: res.Findings, Skipped: res.ParseErrors}
	if r.Findings == nil {
		r.Findings = []Finding{}
	}
	r.Summary.Errors = res.Errors
	r.Summary.Warnings = res.Warnings
	r.Summary.Scanned = res.Scanned
	r.Summary.Indexed = res.Indexed
	r.Summary.Excused = res.Excused
	r.Summary.Stale = res.Stale

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

func reportRules(w io.Writer) {
	for _, id := range ruleIDs() {
		r := catalog[id]
		fmt.Fprintf(w, "%-8s %-5s %-4s %s\n", r.ID, r.Sev, r.Lang, r.Summary)
	}
	fmt.Fprintf(w, "\n%d rules. `quality-gate explain <ID>` for the reasoning behind one.\n", len(catalog))
}

func indent(s string, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if l != "" {
			lines[i] = prefix + l
		}
	}
	return strings.Join(lines, "\n")
}
