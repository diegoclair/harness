package main

import (
	"fmt"
	"sort"
	"strings"
)

type checkOptions struct {
	all   bool
	since string
}

type checkResult struct {
	Findings     []Finding
	Errors       int
	Warnings     int
	Scanned      int
	Indexed      int
	Excused      int
	Stale        int
	StaleEntries []BaselineEntry
	ParseErrors  []string
}

func runCheck(cfg *Config, opts checkOptions) (*checkResult, error) {
	indexed, err := allSourceFiles(cfg)
	if err != nil {
		return nil, err
	}

	res := &checkResult{Indexed: len(indexed)}
	parsed, idx := indexRepo(cfg, indexed, res)

	targets := indexed
	if !opts.all {
		changed, err := changedFiles(cfg, opts.since)
		if err != nil {
			return nil, err
		}
		targets = changed
		for path, ranges := range addedLineRanges(cfg, opts.since) {
			if f, ok := parsed[path]; ok {
				f.AddedLines = ranges
			}
		}
	}

	var findings []Finding
	for _, path := range targets {
		f, ok := parsed[path]
		if !ok {
			continue
		}
		res.Scanned++

		var fileFindings []Finding
		add := func(fi Finding) { fileFindings = append(fileFindings, fi) }

		sups := collectSuppressions(cfg, f, add)
		checkComments(cfg, f, add)
		checkComplexity(cfg, f, add)
		checkArchitecture(cfg, f, add)
		checkDuplication(cfg, idx, f, add)

		// Identical comments in one file must not share a signature: one
		// baseline entry would excuse both, and fixing one hides the other.
		seen := map[string]int{}
		for _, fi := range fileFindings {
			if fi.Rule != "GATE-01" && suppressed(sups, fi) {
				continue
			}
			seen[fi.Signature]++
			if n := seen[fi.Signature]; n > 1 {
				fi.Signature = fmt.Sprintf("%s#%d", fi.Signature, n)
			}
			findings = append(findings, fi)
		}
	}

	if !opts.all {
		if fi, ok := deliveryCommentRatio(cfg, opts.since); ok {
			findings = append(findings, fi)
		}
	}

	findings = dedupePairs(findings)

	baseline, err := loadBaseline(cfg.Root)
	if err != nil {
		return nil, err
	}
	before := len(findings)
	findings = baseline.filter(findings)
	res.Excused = before - len(findings)

	if opts.all {
		res.StaleEntries = baseline.stale()
		for _, e := range res.StaleEntries {
			findings = append(findings, Finding{
				Rule: "GATE-02", Sev: severityOf(cfg, "GATE-02"), File: e.File, Line: 0,
				Message:   fmt.Sprintf("baseline excuses a %s that no longer exists — regenerate so the baseline can only shrink", e.Rule),
				Signature: signature("GATE-02", e.Signature),
			})
			res.Stale++
		}
	}

	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].File != findings[j].File {
			return findings[i].File < findings[j].File
		}
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		return findings[i].Rule < findings[j].Rule
	})

	for _, f := range findings {
		if f.Sev == SevError {
			res.Errors++
		} else {
			res.Warnings++
		}
	}
	res.Findings = findings
	return res, nil
}

// indexRepo parses every file in the repo and feeds the duplication index,
// which is what lets a one-file delivery still be compared against everything.
func indexRepo(cfg *Config, paths []string, res *checkResult) (map[string]*File, *dupIndex) {
	idx := newDupIndex(int(cfg.threshold("dup.min_tokens")), int(cfg.threshold("dup.min_tokens_shape")),
		int(cfg.threshold("dup.min_jsx_nodes")))
	parsed := make(map[string]*File, len(paths))
	for _, path := range paths {
		f, err := parseFile(cfg, path)
		if err != nil {
			res.ParseErrors = append(res.ParseErrors, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		if f == nil {
			continue // a language with no front-end yet
		}
		parsed[path] = f
		idx.add(f)
		idx.addJSX(f)
	}
	return parsed, idx
}

// parseFile returns nil when the file's language has no front-end yet, so a
// mixed repo reports on what it can instead of failing.
func parseFile(cfg *Config, path string) (*File, error) {
	lang, ok := langOf(path)
	if !ok {
		return nil, nil
	}
	switch lang {
	case LangGo:
		return parseGo(cfg.Root, path)
	case LangWeb:
		return parseWeb(cfg.Root, path)
	default:
		return nil, nil
	}
}

func deliveryCommentRatio(cfg *Config, since string) (Finding, bool) {
	comments, code, worst := addedLineRatio(cfg, since)
	if code == 0 {
		return Finding{}, false
	}
	ratio := float64(comments) / float64(code)
	limit := cfg.threshold("comments.diff_ratio")
	if ratio <= limit {
		return Finding{}, false
	}
	msg := fmt.Sprintf("this delivery added %d comment lines to %d code lines (%.0f%%, budget %.0f%%)",
		comments, code, ratio*100, limit*100)
	if len(worst) > 0 {
		msg += " — heaviest: " + strings.Join(worst, ", ")
	}
	return Finding{
		Rule: "CMT-08", Sev: severityOf(cfg, "CMT-08"), File: ".", Line: 0,
		Message: msg, Signature: signature("CMT-08", "delivery"),
	}, true
}
