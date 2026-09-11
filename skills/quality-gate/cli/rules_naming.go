package main

import "fmt"

// The signature is the only proof of I/O a linter can trust; whether a name is
// merely opaque is judgment, and that stays with the reviewer.
func checkNaming(cfg *Config, f *File, add func(Finding)) {
	for _, fn := range f.Funcs {
		tell := ioTell(fn)
		if tell == "" || !endsInOf(fn.Name) {
			continue
		}
		add(Finding{
			Rule: "NAM-01", Sev: severityOf(cfg, "NAM-01"), File: f.Path, Line: fn.Line,
			Message: fmt.Sprintf("%s reads like a pure value but %s — name the cost with a verb (read, load, fetch, Get)",
				fn.Name, tell),
			Signature: signature("NAM-01", f.Path, fn.Name),
		})
	}
}

// `Of` promises only as its own camelCase word, never as the tail of a longer one.
func endsInOf(name string) bool {
	words := splitIdent(name)
	return len(words) > 1 && words[len(words)-1] == "of"
}

func ioTell(fn Func) string {
	switch {
	case fn.TakesContext && fn.ReturnsError:
		return "takes a context and returns an error"
	case fn.TakesContext:
		return "takes a context"
	case fn.ReturnsError:
		return "returns an error"
	}
	return ""
}
