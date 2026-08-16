package main

import "fmt"

// Every CPX rule is a warning by design: a long function is not a defect, two
// unrelated rules living in one function is. See reference/rules.md.
func checkComplexity(cfg *Config, f *File, add func(Finding)) {
	for _, fn := range f.Funcs {
		emit := func(rule, msg string) {
			add(Finding{
				Rule: rule, Sev: severityOf(cfg, rule), File: f.Path, Line: fn.Line,
				Message: msg, Signature: signature(rule, f.Path, fn.Name),
			})
		}
		if limit := int(cfg.threshold("cpx.cyclomatic")); fn.Cyclomatic > limit {
			emit("CPX-01", fmt.Sprintf("%s has cyclomatic complexity %d (limit %d) — does it hold two different rules?",
				fn.Name, fn.Cyclomatic, limit))
		}
		if limit := int(cfg.threshold("cpx.depth")); fn.MaxDepth > limit {
			emit("CPX-02", fmt.Sprintf("%s nests %d deep (limit %d) — usually a missing early return",
				fn.Name, fn.MaxDepth, limit))
		}
		branchy := fn.Cyclomatic >= int(cfg.threshold("cpx.lines_cyclomatic"))
		if limit := int(cfg.threshold("cpx.lines")); fn.EndLine-fn.Line+1 > limit && branchy && !f.IsTest {
			emit("CPX-03", fmt.Sprintf("%s is %d lines (limit %d)", fn.Name, fn.EndLine-fn.Line+1, limit))
		}
		if limit := int(cfg.threshold("cpx.params")); fn.Params > limit {
			emit("CPX-04", fmt.Sprintf("%s takes %d parameters (limit %d, ctx not counted)",
				fn.Name, fn.Params, limit))
		}
	}
	checkComponentSize(cfg, f, add)
}
