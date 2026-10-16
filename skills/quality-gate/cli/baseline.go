package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
	sortEntries(b.Entries)

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

type remapResult struct {
	Remapped  int
	Unmatched []BaselineEntry
	Fresh     int
}

// An entry is only ever carried to a new signature, never created: a remap
// freezes no debt.
func remapBaseline(cfg *Config, from []string, to string) (*remapResult, error) {
	b, err := loadBaseline(cfg.Root)
	if err != nil {
		return nil, err
	}
	indexed, err := allSourceFiles(cfg)
	if err != nil {
		return nil, err
	}
	for _, old := range from {
		if _, err := os.Stat(filepath.Join(cfg.Root, old)); err == nil {
			return nil, fmt.Errorf("%s still exists: remap carries code that moved away from it", old)
		}
	}
	repo := indexRepo(cfg, indexed, &checkResult{})
	f, ok := repo.parsed[to]
	if !ok {
		return nil, fmt.Errorf("%s is not a source file the gate reads", to)
	}
	current := repo.findingsIn(cfg, f, nil)
	renamed := signaturesAtOldPaths(to, current, from)

	res := &remapResult{}
	res.Remapped, res.Unmatched = b.carry(renamed, from, to)
	kept := map[string]bool{}
	for _, e := range b.Entries {
		kept[e.Signature] = true
	}
	for _, fi := range current {
		if !kept[fi.Signature] {
			res.Fresh++
		}
	}

	sortEntries(b.Entries)
	raw, err := encodeBaseline(*b)
	if err != nil {
		return nil, err
	}
	return res, os.WriteFile(filepath.Join(cfg.Root, baselineName), raw, 0o644)
}

// An entry whose new signature another entry already holds or also claims is
// left as it was: one signature never excuses two findings.
func (b *Baseline) carry(renamed map[string]string, from []string, to string) (int, []BaselineEntry) {
	existing := map[string]bool{}
	claims := map[string]int{}
	for _, e := range b.Entries {
		existing[e.Signature] = true
		if next, ok := renamed[e.Signature]; ok {
			claims[next]++
		}
	}
	remapped := 0
	var unmatched []BaselineEntry
	for i, e := range b.Entries {
		next, ok := renamed[e.Signature]
		if !ok || claims[next] > 1 || existing[next] {
			if mentionsAny(e, from) {
				unmatched = append(unmatched, e)
			}
			continue
		}
		b.Entries[i].Signature = next
		if contains(from, e.File) {
			b.Entries[i].File = to
		}
		remapped++
	}
	return remapped, unmatched
}

// A numbered signature depends on its neighbours' order, so it is never carried.
// The old paths are gone, so no clone peer is one and no old signature has two sources.
func signaturesAtOldPaths(to string, current []Finding, from []string) map[string]string {
	out := map[string]string{}
	for _, fi := range current {
		if strings.Contains(fi.Signature, "#") {
			continue
		}
		if !strings.HasPrefix(fi.Rule, "DUP-") {
			for _, old := range from {
				out[fileSignature(fi.Rule, old, fi.subject)] = fi.Signature
			}
			continue
		}
		peers := []string{fi.clonePeer}
		if fi.clonePeer == to {
			peers = from
		}
		for _, old := range from {
			for _, peer := range peers {
				out[pairSignature(fi.Rule, old, peer)] = fi.Signature
			}
		}
	}
	return out
}

func mentionsAny(e BaselineEntry, paths []string) bool {
	for _, p := range paths {
		if e.File == p || (strings.HasPrefix(e.Rule, "DUP-") && e.Signature == pairSignature(e.Rule, e.File, p)) {
			return true
		}
	}
	return false
}

func sortEntries(entries []BaselineEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].File != entries[j].File {
			return entries[i].File < entries[j].File
		}
		if entries[i].Rule != entries[j].Rule {
			return entries[i].Rule < entries[j].Rule
		}
		return entries[i].Signature < entries[j].Signature
	})
}
