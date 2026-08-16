package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const baselineName = ".quality-gate-baseline.json"

// A baseline entry is matched by signature, which is content-derived: editing
// the code loses the pass, everything around it moving does not. Moving the
// file itself does lose it — the alternative, a path-free signature, would let
// a copied violation inherit the original's excuse.
type BaselineEntry struct {
	Rule      string `json:"rule"`
	File      string `json:"file"`
	Signature string `json:"signature"`
	Note      string `json:"note,omitempty"`
}

type Baseline struct {
	Version int             `json:"version"`
	Entries []BaselineEntry `json:"entries"`

	matched map[string]bool
}

func loadBaseline(root string) (*Baseline, error) {
	b := &Baseline{Version: 1, matched: map[string]bool{}}
	raw, err := os.ReadFile(filepath.Join(root, baselineName))
	if os.IsNotExist(err) {
		return b, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, b); err != nil {
		return nil, fmt.Errorf("%s: %w", baselineName, err)
	}
	b.matched = map[string]bool{}
	return b, nil
}

func (b *Baseline) index() map[string]BaselineEntry {
	m := make(map[string]BaselineEntry, len(b.Entries))
	for _, e := range b.Entries {
		m[e.Signature] = e
	}
	return m
}

// filter splits findings into the ones the baseline already excuses and the
// ones it does not.
func (b *Baseline) filter(findings []Finding) (fresh []Finding) {
	known := b.index()
	for _, f := range findings {
		if _, ok := known[f.Signature]; ok {
			b.matched[f.Signature] = true
			continue
		}
		fresh = append(fresh, f)
	}
	return fresh
}

// stale returns the entries whose code is gone. Only meaningful after a full
// scan: on a diff run the unmatched entries are simply the files not read.
func (b *Baseline) stale() []BaselineEntry {
	var out []BaselineEntry
	for _, e := range b.Entries {
		if !b.matched[e.Signature] {
			out = append(out, e)
		}
	}
	return out
}

func writeBaseline(root string, findings []Finding) (int, error) {
	b := Baseline{Version: 1}
	seen := map[string]bool{}
	for _, f := range findings {
		if seen[f.Signature] {
			continue
		}
		seen[f.Signature] = true
		b.Entries = append(b.Entries, BaselineEntry{
			Rule: f.Rule, File: f.File, Signature: f.Signature, Note: f.Message,
		})
	}
	sort.Slice(b.Entries, func(i, j int) bool {
		if b.Entries[i].File != b.Entries[j].File {
			return b.Entries[i].File < b.Entries[j].File
		}
		if b.Entries[i].Rule != b.Entries[j].Rule {
			return b.Entries[i].Rule < b.Entries[j].Rule
		}
		return b.Entries[i].Signature < b.Entries[j].Signature
	})

	raw, err := encodeBaseline(b)
	if err != nil {
		return 0, err
	}
	return len(b.Entries), os.WriteFile(filepath.Join(root, baselineName), raw, 0o644)
}

// One entry per line: the file is committed, so a git diff has to read as
// "this debt was paid", not as a reflowed blob. Indenting each entry over six
// lines turned a 1051-entry ledger into 6311.
func encodeBaseline(b Baseline) ([]byte, error) {
	var out bytes.Buffer
	fmt.Fprintf(&out, "{\n  \"version\": %d,\n  \"entries\": [\n", b.Version)
	for i, e := range b.Entries {
		line, err := json.Marshal(e)
		if err != nil {
			return nil, err
		}
		out.WriteString("    ")
		out.Write(line)
		if i < len(b.Entries)-1 {
			out.WriteByte(',')
		}
		out.WriteByte('\n')
	}
	out.WriteString("  ]\n}\n")
	return out.Bytes(), nil
}
