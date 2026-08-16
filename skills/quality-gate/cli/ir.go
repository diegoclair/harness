package main

import "strings"

// The IR is what every rule reads. A language front-end (Go today, the ts/tsx
// scanner in F2) is the only thing that knows about syntax; rules never do.

type Lang string

const (
	LangGo  Lang = "go"
	LangWeb Lang = "web"
)

// CommentPos is the reader's distance from the code, which is what sets a
// comment's budget. See reference/rules.md, CMT-01.
type CommentPos string

const (
	PosPackage   CommentPos = "package"
	PosType      CommentPos = "type"
	PosInterface CommentPos = "interface"
	PosFunc      CommentPos = "func"
	PosDecl      CommentPos = "decl"
	PosBody      CommentPos = "body"
	PosTrailing  CommentPos = "trailing"
	PosOrphan    CommentPos = "orphan"
)

type Comment struct {
	Line    int
	EndLine int
	Lines   []string // content with the // or /* markers stripped
	Text    string
	Pos     CommentPos
	Target  string // symbol this documents, empty when it documents nothing

	// Lines spent on `/*` and `*/` alone: punctuation, never budget.
	Delims int

	// Identifiers of the code below, split on camelCase. Empty for PosPackage.
	NextIdents []string
}

func (c Comment) Span() int {
	if n := c.EndLine - c.Line + 1 - c.Delims; n > 1 {
		return n
	}
	return 1
}

// blockDelimLines counts the delimiter lines of a block comment that carry no
// prose of their own, so a front-end can report them without a rule ever
// learning what a comment marker looks like.
func blockDelimLines(srcLines []string, startLine, endLine int) int {
	if endLine <= startLine {
		return 0
	}
	n := 0
	if strings.TrimLeft(strings.TrimSpace(lineAt(srcLines, startLine)), "{ \t/*") == "" {
		n++
	}
	if strings.TrimRight(strings.TrimSpace(lineAt(srcLines, endLine)), " \t*/}") == "" {
		n++
	}
	return n
}

func lineAt(srcLines []string, n int) string {
	if n-1 < 0 || n-1 >= len(srcLines) {
		return ""
	}
	return srcLines[n-1]
}

type Func struct {
	Name       string
	Line       int
	EndLine    int
	Cyclomatic int
	MaxDepth   int
	Params     int

	// Web only: a function that renders markup is measured against the
	// component budget rather than the plain function one.
	Hooks     int
	Component bool
}

// Element is one markup element's opening tag, attributes included, so a rule
// can ask about the element instead of the line it happened to wrap onto.
type Element struct {
	Name    string
	Line    int
	EndLine int
	Open    string
}

// JSXNode is one node of a markup tree, carried so DUP-03 can compare subtrees
// across files without any rule knowing what a tag is.
type JSXNode struct {
	Tag   string
	Depth int
	Line  int
}

type Import struct {
	Path string
	Line int
}

type StringLit struct {
	Line  int
	Value string
}

// Cond is a conditional that decides an outcome, carried so the ARC rules can
// ask whether a layer is deciding something it has no business deciding.
type Cond struct {
	Line   int
	Src    string
	Guard  bool // nil/err/ok check — never a domain rule
	Fields []string
}

type Token struct {
	Line int
	Text string
	Kind byte // 'i' identifier, 'l' literal, 'o' operator or keyword
}

// attachNextIdents records the identifiers of the code each comment sits
// above. It reads source lines rather than the AST so every front-end gets it
// for free, and it is what lets CMT-02 ask whether the prose is just the code
// spelled out in words.
func attachNextIdents(f *File) {
	const lookahead = 3
	for i := range f.Comments {
		c := &f.Comments[i]
		if c.Pos == PosTrailing {
			c.NextIdents = identWords(lineBefore(f.SrcLines, c.Line))
			continue
		}
		seen := 0
		for n := c.EndLine; n < len(f.SrcLines) && seen < lookahead; n++ {
			line := strings.TrimSpace(f.SrcLines[n])
			if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "*") {
				continue
			}
			c.NextIdents = append(c.NextIdents, identWords(line)...)
			seen++
		}
	}
}

func (f *File) isAdded(line int) bool {
	for _, r := range f.AddedLines {
		if line >= r.from && line <= r.to {
			return true
		}
	}
	return false
}

func lineBefore(srcLines []string, n int) string {
	if n-1 < 0 || n-1 >= len(srcLines) {
		return ""
	}
	line := srcLines[n-1]
	for _, marker := range []string{"//", "/*", "{/*"} {
		if i := strings.Index(line, marker); i >= 0 {
			line = line[:i]
		}
	}
	return line
}

// AddedLines carries the delivery's own hunks, so a rule can ask about what
// this change introduced rather than about the whole file.
type File struct {
	AddedLines []lineRange

	Path     string // repo-relative, slash-separated
	Lang     Lang
	Package  string
	SrcLines []string
	IsTest   bool

	// SrcLines with every comment blanked, so a rule matching on source never
	// fires on prose. Nil when the front-end cannot produce it.
	CodeLines []string

	Comments []Comment
	Funcs    []Func
	Imports  []Import
	Strings  []StringLit
	Conds    []Cond
	Tokens   []Token
	Elements []Element
	JSXNodes []JSXNode
}

func (f *File) codeSource() []string {
	if f.CodeLines != nil {
		return f.CodeLines
	}
	return f.SrcLines
}
