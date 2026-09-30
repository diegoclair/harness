package main

import (
	"fmt"
	"path"
	"strings"
)

// Shape rules point at the design step a function is asking for; every one is
// a warning because only a reader can name the missing type.

type goPackage struct {
	refs  map[string]int
	calls map[string]int
}

// Unexported names are only reachable from their own directory, so counting
// references per directory is enough to know how many callers a helper has.
func indexGoPackages(parsed map[string]*File) map[string]*goPackage {
	pkgs := map[string]*goPackage{}
	for _, f := range parsed {
		if f.Lang != LangGo || f.IsTest {
			continue
		}
		dir := path.Dir(f.Path)
		p, ok := pkgs[dir]
		if !ok {
			p = &goPackage{refs: map[string]int{}, calls: map[string]int{}}
			pkgs[dir] = p
		}
		for name, n := range f.Refs {
			p.refs[name] += n
		}
		for name, n := range f.Calls {
			p.calls[name] += n
		}
	}
	return pkgs
}

func checkShape(cfg *Config, pkgs map[string]*goPackage, f *File, add func(Finding)) {
	if f.Lang != LangGo || f.IsTest {
		return
	}
	emit := func(rule string, line int, subject, msg string) {
		add(Finding{
			Rule: rule, Sev: severityOf(cfg, rule), File: f.Path, Line: line,
			Message: msg, Signature: signature(rule, f.Path, subject),
		})
	}
	pkg := pkgs[path.Dir(f.Path)]
	for _, fn := range f.Funcs {
		if len(fn.BoolParams) > 0 {
			emit("CPX-06", fn.Line, fn.Name, fmt.Sprintf(
				"%s takes bool %s — does each value select a different rule? split by rule, or name the choice with a type",
				fn.Name, strings.Join(fn.BoolParams, ", ")))
		}
		if fn.Forwards && !isExported(fn.Name) && pkg != nil && pkg.hasOneCaller(fn.Name) {
			emit("CPX-07", fn.Line, fn.Name, fmt.Sprintf(
				"%s has one caller and only forwards a call — inline it, or give it the rule it is named after", fn.Name))
		}
	}
	for _, c := range f.RecursiveClosures {
		emit("CPX-08", c.Line, c.Name, fmt.Sprintf(
			"%s is a closure calling itself through a var — extract a named function, or a type if it shares state", c.Name))
	}
}

func (p *goPackage) hasOneCaller(name string) bool {
	return p.refs[name] == 1 && p.calls[name] == 1
}
