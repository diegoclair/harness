package main

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
)

// The index covers the whole repo, not the diff: the finding that matters is
// "this block already exists over there", not "the delivery repeats itself".

type occurrence struct {
	file string
	tok  int
	line int
}

type stream struct {
	exact []uint64
	shape []uint64
	lines []int

	// Computed once per file; per-position would make the scan quadratic.
	exactHashes []uint64
	shapeHashes []uint64

	branches []int // prefix sum of control-flow tokens
}

// A type-2 clone is copied logic, not copied shape. Without this floor, every
// `type x struct{...}` plus its constructor matches every other one, because
// folding identifiers is exactly what makes idiomatic Go look identical.
const (
	minBranchesForShapeClone = 3
	// Zero control flow means a duplicated declaration, not duplicated logic.
	minBranchesForExactClone = 2
)

var controlFlowTokens = map[string]bool{
	"if": true, "for": true, "switch": true, "case": true, "select": true,
	"range": true, "&&": true, "||": true,
	"while": true, "catch": true, "??": true, "?": true,
}

type dupIndex struct {
	window      int
	shapeWindow int
	jsxWindow   int
	files       map[string]stream
	exact       map[uint64][]occurrence
	shape       map[uint64][]occurrence
	tests       map[string]bool

	jsxFiles map[string]jsxStream
	jsx      map[uint64][]occurrence
}

// A markup subtree is compared by tag and relative nesting: a component rebuilt
// under a new name keeps the tree and changes the copy, the classes and the
// handlers, so anything finer than the shape would miss it.
type jsxStream struct {
	hashes []uint64
	lines  []int
	nodes  []JSXNode
}

const (
	hashBase       = 1000003
	maxOccurrences = 64 // a block repeated everywhere is boilerplate, not a clone
)

func newDupIndex(window, shapeWindow, jsxWindow int) *dupIndex {
	return &dupIndex{
		window:      window,
		shapeWindow: shapeWindow,
		jsxWindow:   jsxWindow,
		files:       map[string]stream{},
		exact:       map[uint64][]occurrence{},
		shape:       map[uint64][]occurrence{},
		tests:       map[string]bool{},
		jsxFiles:    map[string]jsxStream{},
		jsx:         map[uint64][]occurrence{},
	}
}

func (d *dupIndex) addJSX(f *File) {
	if len(f.JSXNodes) < d.jsxWindow {
		return
	}
	lines := make([]int, len(f.JSXNodes))
	for i, n := range f.JSXNodes {
		lines[i] = n.Line
	}
	s := jsxStream{hashes: rollingJSXHashes(f.JSXNodes, d.jsxWindow), lines: lines, nodes: f.JSXNodes}
	d.jsxFiles[f.Path] = s
	for i, h := range s.hashes {
		if len(d.jsx[h]) < maxOccurrences {
			d.jsx[h] = append(d.jsx[h], occurrence{f.Path, i, lines[i]})
		}
	}
}

// rollingJSXHashes hashes each window of nodes by tag plus depth relative to
// the window's first node, so the same subtree matches at any nesting level.
func rollingJSXHashes(nodes []JSXNode, window int) []uint64 {
	if len(nodes) < window {
		return nil
	}
	out := make([]uint64, len(nodes)-window+1)
	for i := range out {
		h := fnv.New64a()
		base := nodes[i].Depth
		for k := 0; k < window; k++ {
			n := nodes[i+k]
			h.Write([]byte(n.Tag))
			fmt.Fprintf(h, "@%d;", n.Depth-base)
		}
		out[i] = h.Sum64()
	}
	return out
}

// subtreeLen is how many nodes hang under the one at start, itself included. A
// run of siblings is not a subtree: twelve `<col>` and `<th>` in a row is table
// boilerplate every list page writes, and reporting it taught nothing.
func subtreeLen(nodes []JSXNode, start int) int {
	root := nodes[start].Depth
	n := 1
	for start+n < len(nodes) && nodes[start+n].Depth > root {
		n++
	}
	return n
}

// findJSXClones reports a markup subtree that also exists in another file. Same
// file does not count: a list rendering the same row twice is a map away, and
// the finding that matters is the component rebuilt somewhere else.
func (d *dupIndex) findJSXClones(path string) []clone {
	s, ok := d.jsxFiles[path]
	if !ok {
		return nil
	}
	var out []clone
	i := 0
	for i < len(s.hashes) {
		span := subtreeLen(s.nodes, i)
		if span < d.jsxWindow {
			i++
			continue
		}
		best, found := clone{}, false
		for _, occ := range d.jsx[s.hashes[i]] {
			if occ.file == path {
				continue
			}
			other := d.jsxFiles[occ.file]
			limit := min(span, subtreeLen(other.nodes, occ.tok))
			if limit < d.jsxWindow {
				continue
			}
			length := d.jsxWindow
			for length < limit &&
				s.nodes[i+length].Tag == other.nodes[occ.tok+length].Tag &&
				s.nodes[i+length].Depth-s.nodes[i].Depth == other.nodes[occ.tok+length].Depth-other.nodes[occ.tok].Depth {
				length++
			}
			if length > best.tokens {
				best = clone{
					rule:     "DUP-03",
					line:     s.lines[i],
					endLine:  s.nodes[min(i+length-1, len(s.nodes)-1)].Line,
					other:    occ,
					otherEnd: other.nodes[min(occ.tok+length-1, len(other.nodes)-1)].Line,
					tokens:   length,
					window:   d.jsxWindow,
				}
				found = true
			}
		}
		if !found {
			i++
			continue
		}
		out = append(out, best)
		i += max(1, best.tokens-d.jsxWindow+1)
	}
	return out
}

func tokenValue(t Token, foldIdents bool) uint64 {
	h := fnv.New64a()
	switch t.Kind {
	case 'l':
		h.Write([]byte("L"))
	case 'i':
		if foldIdents {
			h.Write([]byte("I"))
		} else {
			h.Write([]byte("i" + t.Text))
		}
	default:
		h.Write([]byte("o" + t.Text))
	}
	return h.Sum64()
}

func (d *dupIndex) add(f *File) {
	if len(f.Tokens) < d.window {
		return
	}

	s := stream{
		exact: make([]uint64, len(f.Tokens)),
		shape: make([]uint64, len(f.Tokens)),
		lines: make([]int, len(f.Tokens)),
	}
	s.branches = make([]int, len(f.Tokens)+1)
	for i, t := range f.Tokens {
		s.exact[i] = tokenValue(t, false)
		s.shape[i] = tokenValue(t, true)
		s.lines[i] = t.Line
		s.branches[i+1] = s.branches[i]
		if t.Kind == 'o' && controlFlowTokens[t.Text] {
			s.branches[i+1]++
		}
	}
	s.exactHashes = rollingHashes(s.exact, d.window)
	s.shapeHashes = rollingHashes(s.shape, d.shapeWindow)
	d.files[f.Path] = s
	d.tests[f.Path] = f.IsTest

	for i, h := range s.exactHashes {
		if len(d.exact[h]) < maxOccurrences {
			d.exact[h] = append(d.exact[h], occurrence{f.Path, i, s.lines[i]})
		}
	}
	for i, h := range s.shapeHashes {
		if len(d.shape[h]) < maxOccurrences {
			d.shape[h] = append(d.shape[h], occurrence{f.Path, i, s.lines[i]})
		}
	}
}

func rollingHashes(values []uint64, window int) []uint64 {
	if len(values) < window {
		return nil
	}
	out := make([]uint64, len(values)-window+1)
	var h, pow uint64 = 0, 1
	for i := 0; i < window; i++ {
		h = h*hashBase + values[i]
		if i > 0 {
			pow *= hashBase
		}
	}
	out[0] = h
	for i := window; i < len(values); i++ {
		h = (h-values[i-window]*pow)*hashBase + values[i]
		out[i-window+1] = h
	}
	return out
}

type clone struct {
	rule          string
	line, endLine int
	other         occurrence
	otherEnd      int
	tokens        int
	window        int
}

// findClones walks the scanned file's windows, keeps the ones that match code
// living elsewhere, and merges consecutive matches so one copied block is one
// finding instead of fifty.
func (d *dupIndex) findClones(path string) []clone {
	s, ok := d.files[path]
	if !ok {
		return nil
	}
	var out []clone
	i := 0
	for i < len(s.exact) {
		c, matched := d.matchAt(path, s, i, "DUP-01")
		if !matched {
			c, matched = d.matchAt(path, s, i, "DUP-02")
		}
		if !matched {
			i++
			continue
		}
		out = append(out, c)
		i += max(1, c.tokens-c.window+1)
	}
	return out
}

func (d *dupIndex) matchAt(path string, s stream, start int, rule string) (clone, bool) {
	values, index, hashes, window := s.exact, d.exact, s.exactHashes, d.window
	if rule == "DUP-02" {
		values, index, hashes, window = s.shape, d.shape, s.shapeHashes, d.shapeWindow
	}
	if start >= len(hashes) {
		return clone{}, false
	}
	best := clone{}
	found := false
	for _, occ := range index[hashes[start]] {
		if occ.file == path && (occ.tok < start || occ.tok-start < window) {
			continue // the pair is reported once, from its first occurrence
		}
		other, ok := d.files[occ.file]
		if !ok {
			continue
		}
		otherValues := other.exact
		if rule == "DUP-02" {
			otherValues = other.shape
		}
		if !equalWindow(values, start, otherValues, occ.tok, window) {
			continue // hash collision
		}
		length := window
		for start+length < len(values) && occ.tok+length < len(otherValues) &&
			values[start+length] == otherValues[occ.tok+length] {
			length++
		}
		floor := minBranchesForExactClone
		if rule == "DUP-02" {
			floor = minBranchesForShapeClone
		}
		if s.branches[start+length]-s.branches[start] < floor {
			continue
		}
		if length > best.tokens {
			best = clone{
				rule:     rule,
				line:     s.lines[start],
				endLine:  s.lines[min(start+length-1, len(s.lines)-1)],
				other:    occ,
				otherEnd: other.lines[min(occ.tok+length-1, len(other.lines)-1)],
				tokens:   length,
				window:   window,
			}
			found = true
		}
	}
	return best, found
}

func equalWindow(a []uint64, ai int, b []uint64, bi, n int) bool {
	if ai+n > len(a) || bi+n > len(b) {
		return false
	}
	for k := 0; k < n; k++ {
		if a[ai+k] != b[bi+k] {
			return false
		}
	}
	return true
}

func checkDuplication(cfg *Config, idx *dupIndex, f *File, add func(Finding)) {
	clones := idx.findClones(f.Path)
	clones = append(clones, idx.findJSXClones(f.Path)...)
	for _, c := range clones {
		if c.rule == "DUP-02" && overlapsExactClone(clones, c) {
			continue
		}
		rule := c.rule
		if idx.tests[f.Path] && idx.tests[c.other.file] {
			rule = "DUP-04"
		}
		shape := ""
		switch c.rule {
		case "DUP-02":
			shape = " (same shape, renamed identifiers)"
		case "DUP-03":
			shape = fmt.Sprintf(" (%d-element markup subtree)", c.tokens)
		}
		// Canonical across the pair and free of line numbers: both sides report
		// the clone once, and moving it down a file keeps its baseline entry.
		ends := []string{f.Path, c.other.file}
		sort.Strings(ends)
		add(Finding{
			Rule: rule, Sev: severityOf(cfg, rule), File: f.Path, Line: c.line,
			Message: fmt.Sprintf("lines %d-%d duplicate %s:%d-%d%s — import the original instead",
				c.line, c.endLine, c.other.file, c.other.line, c.otherEnd, shape),
			Signature: signature(rule, ends[0], ends[1]),
		})
	}
}

// An exact clone and a shape clone over the same lines are one finding, and
// the exact one says more.
func overlapsExactClone(clones []clone, shape clone) bool {
	for _, c := range clones {
		if c.rule == "DUP-01" && c.line <= shape.endLine && shape.line <= c.endLine {
			return true
		}
	}
	return false
}

// dedupePairs drops the mirror finding: when the whole repo is scanned, A
// duplicating B is also B duplicating A, and one of them is enough.
func dedupePairs(findings []Finding) []Finding {
	seen := map[string]bool{}
	var out []Finding
	for _, f := range findings {
		if !strings.HasPrefix(f.Rule, "DUP-") {
			out = append(out, f)
			continue
		}
		key := f.Signature
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, f)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
