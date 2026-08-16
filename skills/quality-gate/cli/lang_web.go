package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// The web front-end is a tolerant scanner, not a type-aware parser. It tracks
// strings, template substitutions, comments, regex literals, brace depth and
// JSX nesting, and where it cannot know — a generic arrow that looks like an
// element, a return type carrying braces — it guesses and stays balanced. The
// approximations it makes are listed in reference/rules.md.

func parseWeb(root, relPath string) (*File, error) {
	src, err := os.ReadFile(filepath.Join(root, relPath))
	if err != nil {
		return nil, err
	}
	s := newWebScanner(relPath, src)
	s.run()
	return s.file(), nil
}

type webFrame struct {
	mode byte // c code, T template literal, X element children, G opening tag
	kind byte // code frames: r root, f function, i type body, k class body, o object, b block, x jsx expression, t template substitution
	fn   *webFunc
	ctrl bool
}

type webFunc struct {
	name     string
	declLine int
	endLine  int
	cyclo    int
	maxDepth int
	baseCtrl int
	params   int
	hooks    int
	hasJSX   bool
	named    bool
}

type webElem struct {
	name     string
	line     int
	startOff int
}

type webBracket struct {
	ch     byte
	tokIdx int
	commas int
	angle  int // open type-argument lists: their commas are not parameters
}

type parenGroup struct {
	openTok  int
	closeTok int
	commas   int
	empty    bool
	used     bool
}

type rawComment struct {
	startLine, endLine int
	lines              []string
	trailing           bool
	block              bool
	ctx                byte
	inFunc             bool
	firstInFile        bool
}

type webScanner struct {
	path string
	src  []byte
	mask []byte
	jsx  bool

	i         int
	line      int
	lineStart int

	frames    []webFrame
	brackets  []webBracket
	lastParen map[int]*parenGroup
	elemStack []webElem
	openFuncs []*webFunc
	ctrlDepth int

	tokens   []Token
	comments []rawComment
	imports  []Import
	strs     []StringLit
	elements []Element
	nodes    []JSXNode
	funcs    []*webFunc

	importLines  map[int]bool
	lastImportKw int
	typeDeclTok  int
	classDeclTok int
	nameTokIdx   int
}

func newWebScanner(path string, src []byte) *webScanner {
	mask := make([]byte, len(src))
	copy(mask, src)
	ext := filepath.Ext(path)
	return &webScanner{
		path:         path,
		src:          src,
		mask:         mask,
		jsx:          ext == ".tsx" || ext == ".jsx",
		line:         1,
		frames:       []webFrame{{mode: 'c', kind: 'r'}},
		lastParen:    map[int]*parenGroup{},
		importLines:  map[int]bool{},
		typeDeclTok:  -1,
		classDeclTok: -1,
	}
}

func (s *webScanner) run() {
	for s.i < len(s.src) {
		switch s.top().mode {
		case 'T':
			s.stepTemplate()
		case 'X':
			s.stepJSXText()
		case 'G':
			s.stepJSXTag()
		default:
			s.stepCode()
		}
	}
	for len(s.openFuncs) > 0 {
		s.closeFunc()
	}
}

func (s *webScanner) top() webFrame { return s.frames[len(s.frames)-1] }

func (s *webScanner) push(f webFrame) { s.frames = append(s.frames, f) }

func (s *webScanner) pop() webFrame {
	if len(s.frames) == 1 {
		return s.frames[0]
	}
	f := s.frames[len(s.frames)-1]
	s.frames = s.frames[:len(s.frames)-1]
	return f
}

func (s *webScanner) advance(n int) {
	for k := 0; k < n && s.i < len(s.src); k++ {
		if s.src[s.i] == '\n' {
			s.line++
			s.lineStart = s.i + 1
		}
		s.i++
	}
}

func (s *webScanner) peek(n int) byte {
	if s.i+n < len(s.src) {
		return s.src[s.i+n]
	}
	return 0
}

func (s *webScanner) nextNonSpace() byte {
	for j := s.i; j < len(s.src); j++ {
		if s.src[j] != ' ' && s.src[j] != '\t' {
			return s.src[j]
		}
	}
	return 0
}

func (s *webScanner) emit(t Token) { s.tokens = append(s.tokens, t) }

func (s *webScanner) lastTok() *Token {
	if len(s.tokens) == 0 {
		return nil
	}
	return &s.tokens[len(s.tokens)-1]
}

func (s *webScanner) prevText(back int) string {
	i := len(s.tokens) - 1 - back
	if i < 0 {
		return ""
	}
	return s.tokens[i].Text
}

func (s *webScanner) maskRange(from, to int) {
	for j := from; j < to && j < len(s.mask); j++ {
		if s.mask[j] != '\n' {
			s.mask[j] = ' '
		}
	}
}

// ---------------------------------------------------------------- code mode

// trivia consumes whitespace and comments, which read the same in code and
// inside an opening tag.
func (s *webScanner) trivia() bool {
	switch c := s.src[s.i]; {
	case c == '\n' || c == ' ' || c == '\t' || c == '\r':
		s.advance(1)
	case c == '/' && s.peek(1) == '/':
		s.lineComment()
	case c == '/' && s.peek(1) == '*':
		s.blockComment()
	default:
		return false
	}
	return true
}

func (s *webScanner) stepCode() {
	if s.trivia() {
		return
	}
	c := s.src[s.i]
	switch {
	case c == '"' || c == '\'':
		s.stringLit(c)
	case c == '`':
		s.advance(1)
		s.push(webFrame{mode: 'T'})
	case c == '/' && s.regexPos():
		s.regexLit()
	case isIdentStart(c):
		s.identifier()
	case c >= '0' && c <= '9':
		s.number()
	case c == '<' && s.jsx && s.jsxExprPos() && s.jsxStart():
		s.openJSXTag()
	case c == '{':
		s.openBrace()
	case c == '}':
		s.closeBrace()
	default:
		s.operator()
	}
}

func (s *webScanner) identifier() {
	start, line := s.i, s.line
	for s.i < len(s.src) && isIdentPart(s.src[s.i]) {
		s.advance(1)
	}
	text := string(s.src[start:s.i])
	kind := byte('i')
	if webKeywords[text] {
		kind = 'o'
	}
	s.emit(Token{Line: line, Kind: kind, Text: text})

	if isHookName(text) && s.nextNonSpace() == '(' {
		s.eachNamedFunc(func(fn *webFunc) { fn.hooks++ })
	}
	if webBranchWords[text] {
		s.branch()
	}
	switch text {
	case "import", "export":
		s.lastImportKw = line
	case "type", "interface", "enum":
		s.typeDeclTok = len(s.tokens) - 1
	case "class":
		s.classDeclTok = len(s.tokens) - 1
	}
}

func (s *webScanner) number() {
	start, line := s.i, s.line
	for s.i < len(s.src) && (isIdentPart(s.src[s.i]) || s.src[s.i] == '.') {
		s.advance(1)
	}
	s.emit(Token{Line: line, Kind: 'l', Text: string(s.src[start:s.i])})
}

func (s *webScanner) stringLit(quote byte) {
	line := s.line
	s.advance(1)
	var b strings.Builder
	for s.i < len(s.src) {
		c := s.src[s.i]
		if c == '\\' {
			s.advance(2)
			continue
		}
		if c == quote {
			s.advance(1)
			break
		}
		if c == '\n' {
			break // unterminated: tolerate rather than swallow the file
		}
		b.WriteByte(c)
		s.advance(1)
	}
	value := b.String()
	s.emit(Token{Line: line, Kind: 'l', Text: value})
	s.strs = append(s.strs, StringLit{Line: line, Value: value})
	s.noteImport(value, line)
}

func (s *webScanner) regexLit() {
	line := s.line
	s.advance(1)
	inClass := false
	for s.i < len(s.src) {
		c := s.src[s.i]
		if c == '\\' {
			s.advance(2)
			continue
		}
		if c == '\n' {
			break
		}
		if c == '[' {
			inClass = true
		} else if c == ']' {
			inClass = false
		} else if c == '/' && !inClass {
			s.advance(1)
			break
		}
		s.advance(1)
	}
	for s.i < len(s.src) && isIdentPart(s.src[s.i]) {
		s.advance(1)
	}
	s.emit(Token{Line: line, Kind: 'l', Text: "regex"})
}

// afterOperator reports whether the last token leaves the scanner at the start
// of an expression, which is what tells a regex from a division and an element
// from a comparison.
func (s *webScanner) afterOperator(closers ...string) bool {
	t := s.lastTok()
	if t == nil {
		return true
	}
	if t.Kind == 'i' || t.Kind == 'l' {
		return false
	}
	return !contains(closers, t.Text)
}

func (s *webScanner) regexPos() bool {
	return s.afterOperator(")", "]", "}", "++", "--")
}

func (s *webScanner) openBrace() {
	line := s.line
	kind, fn := s.braceKind()
	s.advance(1)
	s.emit(Token{Line: line, Kind: 'o', Text: "{"})
	if kind == 'b' && s.isControlBrace() {
		s.ctrlDepth++
		s.eachNamedFunc(func(f *webFunc) {
			if d := s.ctrlDepth - f.baseCtrl; d > f.maxDepth {
				f.maxDepth = d
			}
		})
		s.push(webFrame{mode: 'c', kind: kind, ctrl: true})
	} else {
		s.push(webFrame{mode: 'c', kind: kind, fn: fn})
	}
	s.brackets = append(s.brackets, webBracket{ch: '{', tokIdx: len(s.tokens) - 1})
	if kind != 'i' {
		s.typeDeclTok = -1
	}
	if kind != 'k' {
		s.classDeclTok = -1
	}
	if fn != nil {
		s.openFuncs = append(s.openFuncs, fn)
	}
}

func (s *webScanner) closeBrace() {
	line := s.line
	s.advance(1)
	s.emit(Token{Line: line, Kind: 'o', Text: "}"})
	if n := len(s.brackets); n > 0 && s.brackets[n-1].ch == '{' {
		s.brackets = s.brackets[:n-1]
	}
	f := s.pop()
	if f.ctrl && s.ctrlDepth > 0 {
		s.ctrlDepth--
	}
	if f.fn != nil {
		f.fn.endLine = line
		s.closeFunc()
	}
}

func (s *webScanner) closeFunc() {
	if n := len(s.openFuncs); n > 0 {
		fn := s.openFuncs[n-1]
		s.openFuncs = s.openFuncs[:n-1]
		if fn.named {
			s.funcs = append(s.funcs, fn)
		}
	}
}

// braceKind reads the tokens already emitted to decide what this block is. A
// function body and a control block look the same from the brace; what tells
// them apart is the head of the parenthesized group in front of it.
func (s *webScanner) braceKind() (byte, *webFunc) {
	prev := s.prevText(0)
	if prev == "=>" {
		return 'f', s.newFunc(true)
	}
	if s.inTypeDecl() {
		return 'i', nil
	}
	if s.inClassDecl() {
		return 'k', nil
	}
	if lp := s.headParen(); lp != nil {
		head := ""
		if lp.openTok > 0 {
			head = s.tokens[lp.openTok-1].Text
		}
		if webControlHeads[head] {
			lp.used = true
			return 'b', nil
		}
		return 'f', s.newFunc(false)
	}
	switch prev {
	case "", ";", "{", "}", "else", "try", "catch", "finally", "do", ")":
		return 'b', nil
	}
	return 'o', nil
}

// isControlBrace reports whether the block just opened is a control-flow body,
// which is the only kind of nesting CPX-02 counts.
func (s *webScanner) isControlBrace() bool {
	switch s.prevText(1) {
	case "else", "try", "catch", "finally", "do":
		return true
	}
	return s.prevText(1) == ")"
}

// headParen returns the parameter list of the function whose body is about to
// open, allowing a return-type annotation between the `)` and the `{`.
func (s *webScanner) headParen() *parenGroup {
	lp := s.lastParen[len(s.brackets)]
	if lp == nil || lp.used {
		return nil
	}
	rest := s.tokens[lp.closeTok+1:]
	if len(rest) > 48 {
		return nil
	}
	if len(rest) > 0 {
		if rest[0].Text != ":" {
			return nil
		}
		for _, t := range rest {
			if t.Text == ";" || t.Text == "=" {
				return nil
			}
		}
	}
	return lp
}

func (s *webScanner) newFunc(arrow bool) *webFunc {
	fn := &webFunc{baseCtrl: s.ctrlDepth}
	var lp *parenGroup
	if arrow {
		if s.prevText(1) == ")" {
			lp = s.lastParen[len(s.brackets)]
		}
	} else {
		lp = s.headParen()
	}
	if lp != nil {
		lp.used = true
		if !lp.empty {
			fn.params = lp.commas + 1
		}
		fn.name = s.nameBeforeParen(lp.openTok, arrow)
	} else if arrow {
		fn.params = 1 // a single unparenthesised parameter
		fn.name = s.nameBeforeParen(len(s.tokens)-2, true)
	}
	fn.named = fn.name != ""
	fn.declLine = s.line
	if fn.named {
		if i := s.nameTokIdx; i >= 0 && i < len(s.tokens) {
			fn.declLine = s.tokens[i].Line
		}
	}
	return fn
}

// nameBeforeParen walks back from a parameter list to the name the function was
// declared under: `function NAME(`, `NAME(` in a class, `const NAME = (`.
func (s *webScanner) nameBeforeParen(openTok int, arrow bool) string {
	s.nameTokIdx = -1
	j := openTok - 1
	if j >= 0 && s.tokens[j].Text == ">" {
		j = s.skipTypeParams(j)
	}
	if j < 0 {
		return ""
	}
	if arrow {
		if s.tokens[j].Text == "async" {
			j--
		}
		if j < 0 || (s.tokens[j].Text != "=" && s.tokens[j].Text != ":") {
			return ""
		}
		j--
		if j < 0 || s.tokens[j].Kind != 'i' {
			return ""
		}
		s.nameTokIdx = j
		return s.tokens[j].Text
	}
	if s.tokens[j].Kind != 'i' {
		return ""
	}
	s.nameTokIdx = j
	return s.tokens[j].Text
}

func (s *webScanner) skipTypeParams(closeIdx int) int {
	depth := 0
	for j := closeIdx; j >= 0; j-- {
		switch s.tokens[j].Text {
		case ">":
			depth++
		case "<":
			depth--
			if depth == 0 {
				return j - 1
			}
		}
	}
	return -1
}

func (s *webScanner) inTypeDecl() bool {
	if s.typeDeclTok < 0 || s.typeDeclTok+1 >= len(s.tokens) {
		return false
	}
	if s.tokens[s.typeDeclTok+1].Kind != 'i' {
		return false
	}
	if s.tokens[s.typeDeclTok].Text != "type" {
		return true
	}
	for _, t := range s.tokens[s.typeDeclTok+2:] {
		if t.Text == "=" {
			return true
		}
	}
	return false
}

func (s *webScanner) inClassDecl() bool {
	return s.classDeclTok >= 0 && s.classDeclTok+1 < len(s.tokens) &&
		s.tokens[s.classDeclTok+1].Kind == 'i'
}

func (s *webScanner) operator() {
	text := s.matchOperator()
	line := s.line
	s.advance(len(text))
	s.emit(Token{Line: line, Kind: 'o', Text: text})

	switch text {
	case "(", "[":
		s.brackets = append(s.brackets, webBracket{ch: text[0], tokIdx: len(s.tokens) - 1})
	case ")", "]":
		s.closeBracket(text[0])
	case ",":
		if n := len(s.brackets); n > 0 && s.brackets[n-1].angle == 0 {
			s.brackets[n-1].commas++
		}
	case "<":
		// A type-argument list, not a comparison: `Pick<Slot, "start">` is one
		// parameter, and its comma must not read as a second one.
		if n := len(s.brackets); n > 0 && s.prevText(1) != "" && s.tokens[len(s.tokens)-2].Kind == 'i' {
			s.brackets[n-1].angle++
		}
	case ">", ">>", ">>>":
		if n := len(s.brackets); n > 0 && s.brackets[n-1].angle > 0 {
			s.brackets[n-1].angle -= min(len(text), s.brackets[n-1].angle)
		}
	case ";":
		s.typeDeclTok, s.classDeclTok = -1, -1
	case "&&", "||", "??", "?":
		s.branch()
	}
}

func (s *webScanner) closeBracket(ch byte) {
	n := len(s.brackets)
	if n == 0 {
		return
	}
	b := s.brackets[n-1]
	s.brackets = s.brackets[:n-1]
	if ch != ')' || b.ch != '(' {
		return
	}
	closeTok := len(s.tokens) - 1
	s.lastParen[len(s.brackets)] = &parenGroup{
		openTok:  b.tokIdx,
		closeTok: closeTok,
		commas:   b.commas,
		empty:    closeTok == b.tokIdx+1,
	}
}

func (s *webScanner) matchOperator() string {
	for _, op := range webOperators {
		if strings.HasPrefix(string(s.src[s.i:min(s.i+4, len(s.src))]), op) {
			return op
		}
	}
	return string(s.src[s.i : s.i+1])
}

func (s *webScanner) branch() {
	s.eachNamedFunc(func(f *webFunc) { f.cyclo++ })
}

func (s *webScanner) eachNamedFunc(fn func(*webFunc)) {
	for _, f := range s.openFuncs {
		if f.named {
			fn(f)
		}
	}
}

// ------------------------------------------------------------ template mode

func (s *webScanner) stepTemplate() {
	line := s.line
	start := s.i
	for s.i < len(s.src) {
		c := s.src[s.i]
		if c == '\\' {
			s.advance(2)
			continue
		}
		if c == '`' || (c == '$' && s.peek(1) == '{') {
			break
		}
		s.advance(1)
	}
	if chunk := string(s.src[start:s.i]); strings.TrimSpace(chunk) != "" {
		s.emit(Token{Line: line, Kind: 'l', Text: chunk})
		s.strs = append(s.strs, StringLit{Line: line, Value: chunk})
	}
	if s.i >= len(s.src) {
		return
	}
	if s.src[s.i] == '`' {
		s.advance(1)
		s.pop()
		return
	}
	s.advance(2)
	s.brackets = append(s.brackets, webBracket{ch: '{', tokIdx: len(s.tokens)})
	s.push(webFrame{mode: 'c', kind: 't'})
}

// ----------------------------------------------------------------- JSX mode

// jsxExprPos reports whether a `<` here can open an element at all: after an
// identifier or a literal it is a comparison or a type argument, never JSX.
func (s *webScanner) jsxExprPos() bool {
	if !s.afterOperator(")", "]", ">", "++", "--") {
		return false
	}
	if t := s.lastTok(); t != nil && webKeywords[t.Text] {
		return jsxOpeningKeywords[t.Text]
	}
	return true
}

var jsxOpeningKeywords = map[string]bool{
	"return": true, "case": true, "default": true, "yield": true, "await": true,
	"typeof": true, "void": true, "in": true, "else": true, "do": true,
}

// jsxStart checks the shape after the `<`, which is what separates an element
// from the generic arrow `<T,>(x: T) => x` that .tsx also allows.
func (s *webScanner) jsxStart() bool {
	j := s.i + 1
	if j >= len(s.src) {
		return false
	}
	if s.src[j] == '>' {
		return true
	}
	if !isIdentStart(s.src[j]) {
		return false
	}
	for j < len(s.src) && (isIdentPart(s.src[j]) || s.src[j] == '.' || s.src[j] == '-' || s.src[j] == ':') {
		j++
	}
	if j >= len(s.src) {
		return false
	}
	switch s.src[j] {
	case ' ', '\t', '\r', '\n', '>', '/':
		return true
	}
	return false
}

func (s *webScanner) openJSXTag() {
	startOff, line := s.i, s.line
	s.advance(1)
	name := s.readJSXName()
	s.elemStack = append(s.elemStack, webElem{name: name, line: line, startOff: startOff})
	s.nodes = append(s.nodes, JSXNode{Tag: name, Depth: len(s.elemStack) - 1, Line: line})
	s.emit(Token{Line: line, Kind: 'o', Text: "<"})
	s.emit(Token{Line: line, Kind: 'i', Text: name})
	s.eachNamedFunc(func(f *webFunc) { f.hasJSX = true })
	s.push(webFrame{mode: 'G'})
}

func (s *webScanner) readJSXName() string {
	start := s.i
	for s.i < len(s.src) && (isIdentPart(s.src[s.i]) || s.src[s.i] == '.' || s.src[s.i] == '-' || s.src[s.i] == ':') {
		s.advance(1)
	}
	return string(s.src[start:s.i])
}

func (s *webScanner) stepJSXTag() {
	if c := s.src[s.i]; c == '/' && s.peek(1) == '>' {
		s.advance(2)
		s.finishOpenTag(true)
		return
	}
	if s.trivia() {
		return
	}
	c := s.src[s.i]
	switch {
	case c == '>':
		s.advance(1)
		s.finishOpenTag(false)
	case c == '{':
		s.advance(1)
		s.brackets = append(s.brackets, webBracket{ch: '{', tokIdx: len(s.tokens)})
		s.push(webFrame{mode: 'c', kind: 'x'})
	case c == '"' || c == '\'':
		s.stringLit(c)
	case isIdentStart(c):
		line := s.line
		s.emit(Token{Line: line, Kind: 'i', Text: s.readJSXName()})
	case c == '=':
		s.emit(Token{Line: s.line, Kind: 'o', Text: "="})
		s.advance(1)
	default:
		s.advance(1)
	}
}

func (s *webScanner) finishOpenTag(selfClosing bool) {
	if s.top().mode == 'G' {
		s.pop()
	}
	if n := len(s.elemStack); n > 0 {
		e := &s.elemStack[n-1]
		if e.startOff < s.i {
			s.elements = append(s.elements, Element{Name: e.name, Line: e.line, EndLine: s.line, Open: string(s.src[e.startOff:s.i])})
		}
	}
	if selfClosing {
		s.popElem()
		return
	}
	s.push(webFrame{mode: 'X'})
}

func (s *webScanner) popElem() {
	if n := len(s.elemStack); n > 0 {
		s.elemStack = s.elemStack[:n-1]
	}
}

func (s *webScanner) stepJSXText() {
	line, start := s.line, s.i
	for s.i < len(s.src) {
		c := s.src[s.i]
		if c == '{' {
			break
		}
		if c == '<' {
			n := s.peek(1)
			if n == '/' || n == '>' || isIdentStart(n) {
				break
			}
		}
		s.advance(1)
	}
	if txt := strings.TrimSpace(string(s.src[start:s.i])); txt != "" {
		s.emit(Token{Line: line, Kind: 'l', Text: txt})
	}
	if s.i >= len(s.src) {
		return
	}
	switch {
	case s.src[s.i] == '{':
		s.advance(1)
		s.brackets = append(s.brackets, webBracket{ch: '{', tokIdx: len(s.tokens)})
		s.push(webFrame{mode: 'c', kind: 'x'})
	case s.peek(1) == '/':
		s.closeJSXElement()
	default:
		s.openJSXTag()
	}
}

func (s *webScanner) closeJSXElement() {
	line := s.line
	s.advance(2)
	name := s.readJSXName()
	for s.i < len(s.src) && s.src[s.i] != '>' {
		s.advance(1)
	}
	s.advance(1)
	s.emit(Token{Line: line, Kind: 'o', Text: "</"})
	if name != "" {
		s.emit(Token{Line: line, Kind: 'i', Text: name})
	}
	if s.top().mode == 'X' {
		s.pop()
	}
	s.popElem()
}

// ----------------------------------------------------------------- comments

func (s *webScanner) lineComment() {
	line, start := s.line, s.i
	trailing := hasCodeBeforeComment(string(s.src[s.lineStart:s.i]))
	for s.i < len(s.src) && s.src[s.i] != '\n' {
		s.advance(1)
	}
	s.maskRange(start, s.i)
	body := strings.TrimSpace(string(s.src[start+2 : s.i]))
	s.addComment(rawComment{startLine: line, endLine: line, lines: []string{body}, trailing: trailing})
}

func (s *webScanner) blockComment() {
	line, start := s.line, s.i
	trailing := hasCodeBeforeComment(string(s.src[s.lineStart:s.i]))
	s.advance(2)
	for s.i < len(s.src) {
		if s.src[s.i] == '*' && s.peek(1) == '/' {
			s.advance(2)
			break
		}
		s.advance(1)
	}
	s.maskRange(start, s.i)
	end := s.i - 2
	if end < start+2 {
		end = start + 2
	}
	var lines []string
	for _, l := range strings.Split(string(s.src[start+2:end]), "\n") {
		lines = append(lines, strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l), "*")))
	}
	for len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	s.addComment(rawComment{startLine: line, endLine: s.line, lines: lines, trailing: trailing, block: true})
}

// addComment drops tooling directives — they are machine instructions, not
// prose — and merges a run of consecutive `//` lines into one block, which is
// what the CMT-01 budget is measured against.
func (s *webScanner) addComment(rc rawComment) {
	kept := rc.lines[:0:0]
	for _, l := range rc.lines {
		if isToolingDirective(l) {
			continue
		}
		kept = append(kept, l)
	}
	if len(kept) == 0 {
		return
	}
	rc.lines = kept
	rc.ctx = s.top().kind
	rc.inFunc = s.insideFunc()
	rc.firstInFile = len(s.tokens) == 0

	if n := len(s.comments); n > 0 && !rc.block && !rc.trailing {
		prev := &s.comments[n-1]
		if !prev.block && !prev.trailing && prev.endLine == rc.startLine-1 {
			prev.lines = append(prev.lines, rc.lines...)
			prev.endLine = rc.endLine
			return
		}
	}
	s.comments = append(s.comments, rc)
}

func (s *webScanner) insideFunc() bool {
	for _, f := range s.frames {
		if f.kind == 'f' {
			return true
		}
	}
	return false
}

// ------------------------------------------------------------------ imports

func (s *webScanner) noteImport(value string, line int) {
	prev1, prev2 := s.prevText(1), s.prevText(2)
	switch {
	case prev1 == "from" || prev1 == "import":
		s.imports = append(s.imports, Import{Path: value, Line: line})
		from := s.lastImportKw
		if from == 0 || from > line {
			from = line
		}
		for n := from; n <= line; n++ {
			s.importLines[n] = true
		}
	case prev1 == "(" && (prev2 == "import" || prev2 == "require"):
		s.imports = append(s.imports, Import{Path: value, Line: line})
	}
}

// ---------------------------------------------------------------- assembling

func (s *webScanner) file() *File {
	f := &File{
		Path:      s.path,
		Lang:      LangWeb,
		SrcLines:  strings.Split(string(s.src), "\n"),
		CodeLines: strings.Split(string(s.mask), "\n"),
		IsTest:    isWebTest(s.path),
		Imports:   s.imports,
		Strings:   s.strs,
		Elements:  s.elements,
		JSXNodes:  s.nodes,
	}
	for _, t := range s.tokens {
		if !s.importLines[t.Line] {
			f.Tokens = append(f.Tokens, t)
		}
	}
	sort.SliceStable(s.funcs, func(i, j int) bool { return s.funcs[i].declLine < s.funcs[j].declLine })
	for _, fn := range s.funcs {
		end := fn.endLine
		if end < fn.declLine {
			end = fn.declLine
		}
		f.Funcs = append(f.Funcs, Func{
			Name:       fn.name,
			Line:       fn.declLine,
			EndLine:    end,
			Cyclomatic: fn.cyclo + 1,
			MaxDepth:   fn.maxDepth,
			Params:     fn.params,
			Hooks:      fn.hooks,
			Component:  fn.hasJSX && isComponentName(fn.name),
		})
	}
	s.classifyComments(f)
	f.Conds = webConds(f)
	attachNextIdents(f)
	return f
}

func (s *webScanner) classifyComments(f *File) {
	for _, rc := range s.comments {
		c := Comment{Line: rc.startLine, EndLine: rc.endLine, Lines: rc.lines}
		c.Text = strings.Join(rc.lines, " ")
		if rc.block {
			c.Delims = blockDelimLines(f.SrcLines, rc.startLine, rc.endLine)
		}
		switch {
		case rc.trailing:
			c.Pos = PosTrailing
		case rc.ctx == 'i' || rc.ctx == 'o':
			c.Pos, c.Target = memberBelow(f, rc.endLine)
		case rc.inFunc:
			c.Pos = PosBody
		case rc.firstInFile && leadsModule(f, rc.endLine):
			c.Pos = PosPackage
		default:
			c.Pos, c.Target = declBelow(f, rc.endLine)
		}
		f.Comments = append(f.Comments, c)
	}
}

// leadsModule reports whether a file-leading comment is a module doc rather
// than the doc of the first declaration: a blank line or an import below it.
func leadsModule(f *File, after int) bool {
	if after < len(f.SrcLines) && strings.TrimSpace(f.SrcLines[after]) == "" {
		return true
	}
	line := strings.TrimSpace(codeLineAt(f, nextCodeLine(f, after)))
	return strings.HasPrefix(line, "import ") || strings.HasPrefix(line, "import{")
}

func declBelow(f *File, after int) (CommentPos, string) {
	n := nextCodeLine(f, after)
	if n == 0 {
		return PosOrphan, ""
	}
	for _, fn := range f.Funcs {
		if fn.Line == n {
			return PosFunc, fn.Name
		}
	}
	line := strings.TrimSpace(codeLineAt(f, n))
	if m := webTypeDeclRe.FindStringSubmatch(line); m != nil {
		if m[1] == "interface" {
			return PosInterface, m[2]
		}
		return PosType, m[2]
	}
	if m := webFuncDeclRe.FindStringSubmatch(line); m != nil {
		return PosFunc, m[1]
	}
	// A module-scope binding is neither a member nor a contract: it gets the
	// orphan budget, and CMT-09 — which is about members — leaves it alone.
	if m := webVarDeclRe.FindStringSubmatch(line); m != nil {
		return PosOrphan, m[1]
	}
	if pos, name := memberBelow(f, after); name != "" {
		return pos, name
	}
	return PosOrphan, ""
}

// memberBelow places a member of a type, an object or an enum. A function-typed
// member is a contract and earns the function budget, the same way a Go
// interface method does; a data member is named already and earns a constraint.
func memberBelow(f *File, after int) (CommentPos, string) {
	line := strings.TrimSpace(codeLineAt(f, nextCodeLine(f, after)))
	m := webMemberRe.FindStringSubmatch(line)
	if m == nil {
		// Nothing below reads as a declaration, so this is not a declaration
		// comment and CMT-09 has nothing to ask about.
		return PosOrphan, ""
	}
	if webCallableMemberRe.MatchString(line) {
		return PosFunc, m[1]
	}
	return PosDecl, m[1]
}

// nextCodeLine is the first line below n carrying code. Comments are blanked
// out of CodeLines, so skipping blanks skips them too.
func nextCodeLine(f *File, n int) int {
	for i := n; i < len(f.CodeLines); i++ {
		if strings.TrimSpace(f.CodeLines[i]) != "" {
			return i + 1
		}
	}
	return 0
}

func codeLineAt(f *File, n int) string {
	if n-1 < 0 || n-1 >= len(f.CodeLines) {
		return ""
	}
	return f.CodeLines[n-1]
}

// webConds records the lines that decide something, which is what ARC-14 asks
// about. Equality against a literal is the shape that matters; a `<` in a .tsx
// file is far more often an element than a comparison.
func webConds(f *File) []Cond {
	byLine := map[int]bool{}
	for i, t := range f.Tokens {
		switch t.Text {
		case "if":
			if t.Kind == 'o' {
				byLine[t.Line] = true
			}
		case "===", "!==", "==", "!=":
			byLine[t.Line] = true
			_ = i
		}
	}
	lines := make([]int, 0, len(byLine))
	for l := range byLine {
		lines = append(lines, l)
	}
	sort.Ints(lines)
	out := make([]Cond, 0, len(lines))
	for _, l := range lines {
		src := strings.TrimSpace(codeLineAt(f, l))
		out = append(out, Cond{Line: l, Src: src, Guard: webGuard(src)})
	}
	return out
}

func webGuard(src string) bool {
	lower := strings.ToLower(src)
	for _, m := range []string{"null", "undefined", "typeof ", ".length", "instanceof",
		"isarray", "?.", "=== \"\"", "!== \"\"", "loading", "error", "!!"} {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return webPredicateRe.MatchString(src)
}

// ------------------------------------------------------------------ helpers

// hasCodeBeforeComment decides whether a comment trails code on its line. The
// `{` of a JSX expression container is the syntax that carries the comment, not
// code it comments on: `{/* … */}` is a comment on its own line.
func hasCodeBeforeComment(prefix string) bool {
	trimmed := strings.TrimSpace(prefix)
	return trimmed != "" && trimmed != "{"
}

func isIdentStart(c byte) bool {
	return c == '_' || c == '$' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c >= 0x80
}

func isIdentPart(c byte) bool { return isIdentStart(c) || (c >= '0' && c <= '9') }

func isHookName(s string) bool {
	return len(s) > 3 && strings.HasPrefix(s, "use") && s[3] >= 'A' && s[3] <= 'Z'
}

func isComponentName(s string) bool { return s != "" && s[0] >= 'A' && s[0] <= 'Z' }

func isWebTest(path string) bool {
	return strings.Contains(path, ".test.") || strings.Contains(path, ".spec.") ||
		strings.Contains(path, "__tests__/") || strings.Contains(path, "__mocks__/")
}

func isToolingDirective(line string) bool {
	l := strings.ToLower(strings.TrimLeft(strings.TrimSpace(line), "/"))
	for _, p := range toolingDirectives {
		if strings.HasPrefix(l, p) {
			return true
		}
	}
	return false
}

var toolingDirectives = []string{
	"eslint", "@ts-", "@typescript-eslint", "prettier-ignore", "biome-ignore",
	"oxlint-", "deno-lint-", "stylelint-", "noinspection", "istanbul ignore",
	"c8 ignore", "v8 ignore", "webpackchunkname", "@vite-ignore", "<reference",
	"#region", "#endregion", "@jsx", "@vitest-environment", "global ", "jshint",
}

var webKeywords = map[string]bool{
	"break": true, "case": true, "catch": true, "class": true, "const": true,
	"continue": true, "debugger": true, "default": true, "delete": true, "do": true,
	"else": true, "enum": true, "export": true, "extends": true, "false": true,
	"finally": true, "for": true, "function": true, "if": true, "implements": true,
	"import": true, "in": true, "instanceof": true, "let": true, "new": true,
	"null": true, "private": true, "protected": true, "public": true, "return": true,
	"static": true, "super": true, "switch": true, "this": true, "throw": true,
	"true": true, "try": true, "typeof": true, "var": true, "void": true,
	"while": true, "with": true, "yield": true, "async": true, "await": true,
	"interface": true,
}

var webBranchWords = map[string]bool{
	"if": true, "for": true, "while": true, "case": true, "catch": true,
}

var webControlHeads = map[string]bool{
	"if": true, "for": true, "while": true, "switch": true, "catch": true, "do": true,
}

// Longest first: matchOperator takes the first prefix that fits.
var webOperators = []string{
	">>>=", "...", "===", "!==", "**=", "<<=", ">>=", "&&=", "||=", "??=", ">>>",
	"=>", "==", "!=", "<=", ">=", "&&", "||", "??", "?.", "?:", "++", "--",
	"+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "**", "<<", ">>",
	"(", ")", "[", "]", ";", ",", "<", ">", "=", "+", "-", "*", "/",
	"%", "!", "?", ":", ".", "&", "|", "^", "~", "@", "#",
}

var (
	webTypeDeclRe = regexp.MustCompile(`^(?:export\s+)?(?:default\s+)?(?:declare\s+)?(?:abstract\s+)?(interface|type|enum|class)\s+([A-Za-z_$][\w$]*)`)
	webFuncDeclRe = regexp.MustCompile(`^(?:export\s+)?(?:default\s+)?(?:async\s+)?function\s*\*?\s*([A-Za-z_$][\w$]*)`)
	webVarDeclRe  = regexp.MustCompile(`^(?:export\s+)?(?:const|let|var)\s+([A-Za-z_$][\w$]*)`)
	webMemberRe   = regexp.MustCompile(`^(?:readonly\s+|public\s+|private\s+|protected\s+|static\s+)*([A-Za-z_$][\w$]*)\s*[?!]?\s*[:(=,]`)

	// A member whose type is a function signature — the web's interface method.
	webCallableMemberRe = regexp.MustCompile(`^(?:readonly\s+|public\s+|private\s+|protected\s+|static\s+)*[A-Za-z_$][\w$]*\s*[?!]?\s*(?:\(|:\s*(?:<[^>]*>\s*)?\()`)

	// A predicate *call* is a guard; a name that starts with `is` is just a name.
	webPredicateRe = regexp.MustCompile(`\b(is|has|should|can)[A-Z]\w*\(`)
)
