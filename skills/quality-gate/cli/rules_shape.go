package main

import (
	"fmt"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Shape rules point at the design step a function is asking for; every one is
// a warning because only a reader can name the missing type.

type goPackage struct {
	files []string
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
		p.files = append(p.files, f.Path)
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
			Message: msg, subject: []string{subject},
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

// Candidates are filtered before the merge-base is read: reading it parses the
// whole module.
func (r *repoIndex) newSubjectPieces(cfg *Config, targets []string, since string, res *checkResult) (map[string]bool, error) {
	var candidates []*File
	for _, path := range targets {
		if f, ok := r.parsed[path]; ok && isSmallSiblingPiece(cfg, r.pkgs, f) {
			candidates = append(candidates, f)
		}
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	rev, err := mergeBase(cfg, since)
	if err != nil {
		res.Notices = append(res.Notices, fmt.Sprintf("CPX-09 did not run: %v", err))
		return nil, nil
	}
	if rev == "" {
		return nil, nil
	}
	blobs, err := goBlobsAt(cfg, rev)
	if err != nil {
		return nil, err
	}
	var absent []*File
	for _, f := range candidates {
		if _, existed := blobs[f.Path]; !existed {
			absent = append(absent, f)
		}
	}
	if len(absent) == 0 {
		return nil, nil
	}
	known, err := declarationsAt(cfg, blobs, res)
	if err != nil {
		return nil, err
	}
	pieces := map[string]bool{}
	for _, f := range absent {
		if !declaresAny(f.Decls, known) {
			pieces[f.Path] = true
		}
	}
	return pieces, nil
}

func isSmallSiblingPiece(cfg *Config, pkgs map[string]*goPackage, f *File) bool {
	if f.Lang != LangGo || f.IsTest || matchesAnyGlob(cfg.TinyFile.Except, f.Path) {
		return false
	}
	if len(f.Funcs) > int(cfg.threshold("cpx.tiny_file_funcs")) || lineCount(f) > int(cfg.threshold("cpx.tiny_file_lines")) {
		return false
	}
	pkg := pkgs[path.Dir(f.Path)]
	if pkg == nil {
		return false
	}
	prefix := subjectPrefix(f.Path)
	for _, sibling := range pkg.files {
		if sibling != f.Path && subjectPrefix(sibling) == prefix {
			return true
		}
	}
	return false
}

// The owner of a file's subject is the name before its first underscore, or the whole name.
func subjectPrefix(p string) string {
	name := strings.TrimSuffix(path.Base(p), ".go")
	owner, _, _ := strings.Cut(name, "_")
	return owner
}

func lineCount(f *File) int {
	n := len(f.SrcLines)
	if n > 0 && f.SrcLines[n-1] == "" {
		n--
	}
	return n
}

// A move often exports a name or turns a function into a method, so the key
// folds the first letter and ignores the receiver.
func declarationKey(name string) string {
	r, size := utf8.DecodeRuneInString(name)
	return string(unicode.ToLower(r)) + name[size:]
}

func declaresAny(decls []string, known map[string]bool) bool {
	for _, d := range decls {
		if known[declarationKey(d)] {
			return true
		}
	}
	return false
}

func checkNewPiece(cfg *Config, pieces map[string]bool, f *File, add func(Finding)) {
	if !pieces[f.Path] {
		return
	}
	prefix := subjectPrefix(f.Path)
	add(Finding{
		Rule: "CPX-09", Sev: severityOf(cfg, "CPX-09"), File: f.Path, Line: 1,
		Message: fmt.Sprintf("may be a piece of the %s subject a sibling already owns: join it there unless it is a subject of its own", prefix),
		subject: []string{prefix},
	})
}
