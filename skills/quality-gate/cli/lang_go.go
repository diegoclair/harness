package main

import (
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func parseGo(root, relPath string) (*File, error) {
	src, err := os.ReadFile(filepath.Join(root, relPath))
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	af, err := parser.ParseFile(fset, relPath, src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	f := &File{
		Path:     relPath,
		Lang:     LangGo,
		Package:  af.Name.Name,
		SrcLines: strings.Split(string(src), "\n"),
		IsTest:   strings.HasSuffix(relPath, "_test.go"),
	}
	line := func(p token.Pos) int { return fset.Position(p).Line }

	for _, imp := range af.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		f.Imports = append(f.Imports, Import{Path: path, Line: line(imp.Pos())})
	}

	f.CodeLines = goCodeLines(f.SrcLines, af, fset)
	f.Comments = goComments(af, f.SrcLines, line)
	collectGoDecls(af, f, line)
	// Import blocks are near-identical across files by nature; leaving them in
	// the token stream would make every file a clone of its neighbours.
	f.Tokens = goTokens(fset, relPath, src, importRanges(af, line))
	attachNextIdents(f)
	return f, nil
}

// goCodeLines blanks every comment span so a pattern rule cannot match prose.
// A rule about a vendor name leaking, for instance, must see the code only —
// the mentions in comments are all legitimate.
func goCodeLines(srcLines []string, af *ast.File, fset *token.FileSet) []string {
	out := append([]string(nil), srcLines...)
	for _, g := range af.Comments {
		for _, c := range g.List {
			start, end := fset.Position(c.Pos()), fset.Position(c.End())
			for ln := start.Line; ln <= end.Line; ln++ {
				i := ln - 1
				if i < 0 || i >= len(out) {
					continue
				}
				from := 0
				if ln == start.Line {
					from = min(max(start.Column-1, 0), len(out[i]))
				}
				to := len(out[i])
				if ln == end.Line {
					to = min(max(end.Column-1, from), len(out[i]))
				}
				out[i] = out[i][:from] + strings.Repeat(" ", to-from) + out[i][to:]
			}
		}
	}
	return out
}

type docKind struct {
	pos    CommentPos
	target string
}

// goComments classifies every comment group by the reader's distance from the
// code. Docs and trailing comments are registered from the AST, which is exact;
// what remains is placed by asking whether it sits inside a function body.
func goComments(af *ast.File, srcLines []string, line func(token.Pos) int) []Comment {
	kinds := map[*ast.CommentGroup]docKind{}
	register := func(g *ast.CommentGroup, pos CommentPos, target string) {
		if g != nil {
			kinds[g] = docKind{pos, target}
		}
	}

	register(af.Doc, PosPackage, af.Name.Name)
	for _, d := range af.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			register(d.Doc, PosFunc, d.Name.Name)
		case *ast.GenDecl:
			register(d.Doc, genDeclPos(d), genDeclName(d))
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					register(s.Doc, typePosOf(s.Type), s.Name.Name)
					register(s.Comment, PosTrailing, s.Name.Name)
					registerFields(s.Type, register)
				case *ast.ValueSpec:
					register(s.Doc, PosDecl, specName(s))
					register(s.Comment, PosTrailing, specName(s))
				}
			}
		}
	}

	bodies := funcBodyRanges(af, line)
	var out []Comment
	for _, g := range af.Comments {
		start, end := line(g.Pos()), line(g.End())
		c := Comment{Line: start, EndLine: end, Lines: commentLines(g), Pos: PosOrphan}
		c.Text = strings.Join(c.Lines, " ")
		c.Delims = blockDelimLines(srcLines, start, end)

		if k, ok := kinds[g]; ok {
			c.Pos, c.Target = k.pos, k.target
		} else if hasCodeBefore(srcLines, g, line) {
			c.Pos = PosTrailing
		} else if inAnyRange(bodies, start) {
			c.Pos = PosBody
		}
		out = append(out, c)
	}
	return out
}

// A single `type X interface{}` hangs its doc on the GenDecl, not the spec,
// which is the common shape — missing it left every interface doc classified
// as a struct.
func genDeclPos(d *ast.GenDecl) CommentPos {
	if len(d.Specs) == 1 {
		if ts, ok := d.Specs[0].(*ast.TypeSpec); ok {
			return typePosOf(ts.Type)
		}
	}
	return PosType
}

func typePosOf(t ast.Expr) CommentPos {
	if _, ok := t.(*ast.InterfaceType); ok {
		return PosInterface
	}
	return PosType
}

// A field and an interface method both speak through their names, so a comment
// beside either is a member comment wherever it sits, above or trailing.
func registerFields(t ast.Expr, register func(*ast.CommentGroup, CommentPos, string)) {
	switch t := t.(type) {
	case *ast.StructType:
		for _, fld := range fieldList(t.Fields) {
			register(fld.Doc, PosField, fieldName(fld))
			register(fld.Comment, PosField, fieldName(fld))
		}
	case *ast.InterfaceType:
		for _, m := range fieldList(t.Methods) {
			register(m.Doc, PosMethod, fieldName(m))
			register(m.Comment, PosMethod, fieldName(m))
		}
	}
}

func fieldList(l *ast.FieldList) []*ast.Field {
	if l == nil {
		return nil
	}
	return l.List
}

func fieldName(f *ast.Field) string {
	if len(f.Names) > 0 {
		return f.Names[0].Name
	}
	if id, ok := f.Type.(*ast.Ident); ok {
		return id.Name // embedded
	}
	return ""
}

func genDeclName(d *ast.GenDecl) string {
	for _, spec := range d.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			return s.Name.Name
		case *ast.ValueSpec:
			return specName(s)
		}
	}
	return ""
}

func specName(s *ast.ValueSpec) string {
	if len(s.Names) > 0 {
		return s.Names[0].Name
	}
	return ""
}

func commentLines(g *ast.CommentGroup) []string {
	var out []string
	for _, c := range g.List {
		text := c.Text
		switch {
		case strings.HasPrefix(text, "//"):
			out = append(out, strings.TrimSpace(text[2:]))
		case strings.HasPrefix(text, "/*"):
			inner := strings.TrimSuffix(strings.TrimPrefix(text, "/*"), "*/")
			for _, l := range strings.Split(inner, "\n") {
				out = append(out, strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l), "*")))
			}
		}
	}
	return out
}

func hasCodeBefore(srcLines []string, g *ast.CommentGroup, line func(token.Pos) int) bool {
	n := line(g.Pos())
	if n-1 < 0 || n-1 >= len(srcLines) {
		return false
	}
	l := srcLines[n-1]
	idx := strings.Index(l, "//")
	if idx < 0 {
		idx = strings.Index(l, "/*")
	}
	return idx > 0 && strings.TrimSpace(l[:idx]) != ""
}

type lineRange struct{ from, to int }

func funcBodyRanges(af *ast.File, line func(token.Pos) int) []lineRange {
	var out []lineRange
	ast.Inspect(af, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncDecl:
			if n.Body != nil {
				out = append(out, lineRange{line(n.Body.Lbrace), line(n.Body.Rbrace)})
			}
		case *ast.FuncLit:
			out = append(out, lineRange{line(n.Body.Lbrace), line(n.Body.Rbrace)})
		}
		return true
	})
	return out
}

func inAnyRange(rs []lineRange, n int) bool {
	for _, r := range rs {
		if n > r.from && n <= r.to {
			return true
		}
	}
	return false
}

func collectGoDecls(af *ast.File, f *File, line func(token.Pos) int) {
	for _, d := range af.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		f.Funcs = append(f.Funcs, Func{
			Name:         fd.Name.Name,
			Line:         line(fd.Pos()),
			EndLine:      line(fd.End()),
			Cyclomatic:   cyclomatic(fd.Body),
			MaxDepth:     maxDepth(fd.Body),
			Params:       countParams(fd),
			TakesContext: takesContext(fd),
			ReturnsError: returnsError(fd),
			BoolParams:   boolParams(fd),
			Forwards:     forwards(fd.Body),
		})
		f.RecursiveClosures = append(f.RecursiveClosures, recursiveClosures(fd.Body, line)...)
	}
	f.Refs, f.Calls = identUses(af)

	ast.Inspect(af, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.BasicLit:
			if n.Kind == token.STRING {
				if v, err := strconv.Unquote(n.Value); err == nil {
					f.Strings = append(f.Strings, StringLit{Line: line(n.Pos()), Value: v})
				}
			}
		case *ast.IfStmt:
			f.Conds = append(f.Conds, condOf(n.Cond, line(n.Pos()), f.SrcLines))
		}
		return true
	})
}

func condOf(expr ast.Expr, ln int, srcLines []string) Cond {
	c := Cond{Line: ln}
	if ln-1 >= 0 && ln-1 < len(srcLines) {
		c.Src = strings.TrimSpace(srcLines[ln-1])
	}
	ast.Inspect(expr, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Ident:
			switch n.Name {
			case "err", "nil", "ok", "error", "len":
				c.Guard = true
			}
		case *ast.CallExpr:
			if sel, ok := n.Fun.(*ast.SelectorExpr); ok && isGuardPredicate(sel.Sel.Name) {
				c.Guard = true
			}
		case *ast.BinaryExpr:
			if (n.Op == token.EQL || n.Op == token.NEQ) && isZeroValue(n.Y) {
				c.Guard = true
			}
		case *ast.SelectorExpr:
			if name := n.Sel.Name; name != "" && isExported(name) {
				c.Fields = append(c.Fields, name)
			}
		}
		return true
	})
	return c
}

// A presence or validity test is not the business deciding anything: emptiness,
// zero values and input validation are exactly what a handler is for.
func isGuardPredicate(name string) bool {
	return guardPredicates[name] || predicatePrefixRe.MatchString(name)
}

var predicatePrefixRe = regexp.MustCompile(`^(Is|Has|Can|Should)[A-Z]`)

// A comparison against the zero value asks whether something is there, which is
// presence, not a business decision.
func isZeroValue(e ast.Expr) bool {
	lit, ok := e.(*ast.BasicLit)
	if !ok {
		return false
	}
	return lit.Value == `""` || lit.Value == "0"
}

var guardPredicates = map[string]bool{
	"IsZero": true, "IsEmpty": true, "IsNil": true, "IsValid": true, "Valid": true,
	"Exists": true, "Has": true, "OK": true, "Ok": true, "Empty": true, "Any": true,
}

func isExported(s string) bool { return s != "" && s[0] >= 'A' && s[0] <= 'Z' }

func cyclomatic(body *ast.BlockStmt) int {
	n := 1
	ast.Inspect(body, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.CommClause:
			n++
		case *ast.CaseClause:
			if len(node.List) > 0 {
				n++
			}
		case *ast.BinaryExpr:
			if node.Op == token.LAND || node.Op == token.LOR {
				n++
			}
		}
		return true
	})
	return n
}

func maxDepth(body *ast.BlockStmt) int {
	return depthBelow(body, 0)
}

func depthBelow(n ast.Node, depth int) int {
	best := depth
	ast.Inspect(n, func(child ast.Node) bool {
		if child == n {
			return true
		}
		switch child.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt,
			*ast.TypeSwitchStmt, *ast.SelectStmt:
			best = max(best, depthBelow(child, depth+1))
			return false
		}
		return true
	})
	return best
}

func countParams(fd *ast.FuncDecl) int {
	n := 0
	for _, p := range fieldList(fd.Type.Params) {
		if isContext(p.Type) {
			continue
		}
		if len(p.Names) == 0 {
			n++
			continue
		}
		n += len(p.Names)
	}
	return n
}

func takesContext(fd *ast.FuncDecl) bool {
	for _, p := range fieldList(fd.Type.Params) {
		if isContext(p.Type) {
			return true
		}
	}
	return false
}

func returnsError(fd *ast.FuncDecl) bool {
	for _, r := range fieldList(fd.Type.Results) {
		if id, ok := r.Type.(*ast.Ident); ok && id.Name == "error" {
			return true
		}
	}
	return false
}

func isContext(e ast.Expr) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "context" && sel.Sel.Name == "Context"
}

func importRanges(af *ast.File, line func(token.Pos) int) []lineRange {
	var out []lineRange
	for _, d := range af.Decls {
		if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.IMPORT {
			out = append(out, lineRange{line(g.Pos()) - 1, line(g.End())})
		}
	}
	return out
}

func goTokens(fset *token.FileSet, path string, src []byte, skip []lineRange) []Token {
	var s scanner.Scanner
	file := fset.AddFile(path+"#tokens", fset.Base(), len(src))
	s.Init(file, src, nil, 0)

	var out []Token
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		ln := fset.Position(pos).Line
		if inAnyRange(skip, ln) {
			continue
		}
		t := Token{Line: ln}
		switch {
		case tok == token.IDENT:
			t.Kind, t.Text = 'i', lit
		case tok.IsLiteral():
			t.Kind, t.Text = 'l', lit
		default:
			t.Kind, t.Text = 'o', tok.String()
		}
		out = append(out, t)
	}
	return out
}

func boolParams(fd *ast.FuncDecl) []string {
	var out []string
	for _, p := range fieldList(fd.Type.Params) {
		if id, ok := p.Type.(*ast.Ident); !ok || id.Name != "bool" {
			continue
		}
		if len(p.Names) == 0 {
			out = append(out, "_")
		}
		for _, n := range p.Names {
			out = append(out, n.Name)
		}
	}
	return out
}

// Only plain arguments count: a call that composes expressions is naming a
// computation, which is what a function is for.
func forwards(body *ast.BlockStmt) bool {
	if len(body.List) != 1 {
		return false
	}
	var call *ast.CallExpr
	switch st := body.List[0].(type) {
	case *ast.ReturnStmt:
		if len(st.Results) == 1 {
			call, _ = st.Results[0].(*ast.CallExpr)
		}
	case *ast.ExprStmt:
		call, _ = st.X.(*ast.CallExpr)
	}
	if call == nil {
		return false
	}
	for _, arg := range call.Args {
		if !isPlainArg(arg) {
			return false
		}
	}
	return true
}

func isPlainArg(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.Ident, *ast.BasicLit:
		return true
	case *ast.SelectorExpr:
		return isPlainArg(e.X)
	case *ast.UnaryExpr:
		return e.Op == token.AND && isPlainArg(e.X)
	case *ast.CallExpr:
		return len(e.Args) == 0 && isPlainArg(e.Fun)
	}
	return false
}

func recursiveClosures(body *ast.BlockStmt, line func(token.Pos) int) []RecursiveClosure {
	declared := map[string]int{}
	ast.Inspect(body, func(n ast.Node) bool {
		ds, ok := n.(*ast.DeclStmt)
		if !ok {
			return true
		}
		gd, ok := ds.Decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR {
			return true
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || len(vs.Values) > 0 {
				continue
			}
			if _, isFunc := vs.Type.(*ast.FuncType); !isFunc {
				continue
			}
			for _, name := range vs.Names {
				declared[name.Name] = line(name.Pos())
			}
		}
		return true
	})
	if len(declared) == 0 {
		return nil
	}

	var out []RecursiveClosure
	ast.Inspect(body, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
			return true
		}
		id, ok := as.Lhs[0].(*ast.Ident)
		lit, isLit := as.Rhs[0].(*ast.FuncLit)
		if !ok || !isLit {
			return true
		}
		if at, known := declared[id.Name]; known && mentions(lit.Body, id.Name) {
			out = append(out, RecursiveClosure{Name: id.Name, Line: at})
		}
		return true
	})
	return out
}

func mentions(n ast.Node, name string) bool {
	found := false
	ast.Inspect(n, func(child ast.Node) bool {
		if id, ok := child.(*ast.Ident); ok && id.Name == name {
			found = true
		}
		return !found
	})
	return found
}

// A function handed over as a value cannot be inlined, so a use that is not a
// call is counted apart from the calls.
func identUses(af *ast.File) (refs, calls map[string]int) {
	declNames := map[*ast.Ident]bool{}
	for _, d := range af.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok {
			declNames[fd.Name] = true
		}
	}
	refs, calls = map[string]int{}, map[string]int{}
	ast.Inspect(af, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Ident:
			if !declNames[n] {
				refs[n.Name]++
			}
		case *ast.CallExpr:
			if name := calleeName(n.Fun); name != "" {
				calls[name]++
			}
		}
		return true
	})
	return refs, calls
}

func calleeName(fun ast.Expr) string {
	switch fun := fun.(type) {
	case *ast.Ident:
		return fun.Name
	case *ast.SelectorExpr:
		return fun.Sel.Name
	case *ast.IndexExpr:
		return calleeName(fun.X)
	}
	return ""
}
