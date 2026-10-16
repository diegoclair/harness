package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, "dist": true,
	"build": true, ".next": true, ".quality-gate-cache": true,
}

func langOf(path string) (Lang, bool) {
	switch filepath.Ext(path) {
	case ".go":
		return LangGo, true
	case ".ts", ".tsx", ".js", ".jsx":
		return LangWeb, true
	}
	return "", false
}

// allSourceFiles is what the duplication index is built from: the whole repo,
// regardless of what this run is scanning.
func allSourceFiles(cfg *Config) ([]string, error) {
	var out []string
	err := filepath.WalkDir(cfg.Root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // an unreadable directory is not a quality finding
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(cfg.Root, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if lang, ok := langOf(rel); !ok || !cfg.wants(lang) || cfg.excluded(rel) {
			return nil
		}
		out = append(out, rel)
		return nil
	})
	sort.Strings(out)
	return out, err
}

func git(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// baseRef picks the branch the delivery is measured against.
func baseRef(cfg *Config) string {
	if cfg.BaseRef != "" {
		return cfg.BaseRef
	}
	for _, candidate := range []string{"origin/main", "main", "origin/master", "master"} {
		if _, err := git(cfg.Root, "rev-parse", "--verify", "--quiet", candidate); err == nil {
			return candidate
		}
	}
	return ""
}

// Empty with no error when the repo has no base branch. An error means a base
// exists and git could not relate HEAD to it: a shallow clone, an unrelated history.
func mergeBase(cfg *Config, since string) (string, error) {
	base := since
	if base == "" {
		base = baseRef(cfg)
	}
	if base == "" {
		return "", nil
	}
	mb, err := git(cfg.Root, "merge-base", "HEAD", base)
	if err != nil {
		return "", fmt.Errorf("git merge-base HEAD %s: %w", base, err)
	}
	return mb, nil
}

// changedFiles is the delivery: everything different from the base branch plus
// everything still uncommitted, because a gate that only sees commits is a gate
// you run too late.
func changedFiles(cfg *Config, since string) ([]string, error) {
	set := map[string]bool{}
	if mb, err := mergeBase(cfg, since); err == nil && mb != "" {
		if out, err := git(cfg.Root, "diff", "--name-only", "--diff-filter=ACMR", mb); err == nil {
			addLines(set, out)
		}
	}
	if out, err := git(cfg.Root, "status", "--porcelain"); err == nil {
		for _, line := range strings.Split(out, "\n") {
			if len(line) < 4 {
				continue
			}
			path := strings.TrimSpace(line[2:])
			if i := strings.Index(path, " -> "); i >= 0 { // rename
				path = path[i+4:]
			}
			set[path] = true
		}
	}

	var out []string
	for path := range set {
		path = filepath.ToSlash(path)
		if lang, ok := langOf(path); ok && cfg.wants(lang) && !cfg.excluded(path) {
			out = append(out, path)
		}
	}
	sort.Strings(out)
	return out, nil
}

func addLines(set map[string]bool, out string) {
	for _, line := range strings.Split(out, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			set[line] = true
		}
	}
}

// addedLineRanges maps each changed file to the lines this delivery added.
func addedLineRanges(cfg *Config, since string) map[string][]lineRange {
	out := map[string][]lineRange{}
	diff, ok := deliveryDiff(cfg, since)
	if !ok {
		return out
	}
	file := ""
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "+++ b/"):
			file = strings.TrimPrefix(line, "+++ b/")
		case strings.HasPrefix(line, "@@"):
			if r, ok := parseHunk(line); ok && file != "" {
				out[file] = append(out[file], r)
			}
		}
	}
	return out
}

var hunkRe = regexp.MustCompile(`^@@ -\S+ \+(\d+)(?:,(\d+))? @@`)

func parseHunk(line string) (lineRange, bool) {
	m := hunkRe.FindStringSubmatch(line)
	if m == nil {
		return lineRange{}, false
	}
	start, _ := strconv.Atoi(m[1])
	count := 1
	if m[2] != "" {
		count, _ = strconv.Atoi(m[2])
	}
	if count == 0 {
		return lineRange{}, false
	}
	return lineRange{from: start, to: start + count - 1}, true
}

func deliveryDiff(cfg *Config, since string) (string, bool) {
	mb, err := mergeBase(cfg, since)
	if err != nil || mb == "" {
		return "", false
	}
	// Uncommitted work is part of the delivery, so the diff runs against the
	// working tree, not against HEAD.
	out, err := git(cfg.Root, "diff", "-U0", mb)
	if err != nil {
		return "", false
	}
	return out, true
}

// addedLineRatio measures the delivery's comment budget (CMT-08) from the diff
// itself: what the delivery added, not what the file already carried.
func addedLineRatio(cfg *Config, since string) (comments, code int, worst []string) {
	mb, err := mergeBase(cfg, since)
	if err != nil || mb == "" {
		return 0, 0, nil
	}
	out, err := git(cfg.Root, "diff", "-U0", mb)
	if err != nil {
		return 0, 0, nil
	}

	perFile := map[string]int{}
	file := ""
	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(line, "+++ b/"):
			file = strings.TrimPrefix(line, "+++ b/")
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			if lang, ok := langOf(file); !ok || !cfg.wants(lang) || cfg.excluded(file) {
				continue
			}
			body := strings.TrimSpace(line[1:])
			if body == "" {
				continue
			}
			if isCommentLine(body) {
				comments++
				perFile[file]++
			} else {
				code++
			}
		}
	}
	return comments, code, topFiles(perFile, 5)
}

func isCommentLine(s string) bool {
	return strings.HasPrefix(s, "//") || strings.HasPrefix(s, "/*") || strings.HasPrefix(s, "*")
}

func topFiles(counts map[string]int, n int) []string {
	type kv struct {
		file string
		n    int
	}
	var all []kv
	for f, c := range counts {
		all = append(all, kv{f, c})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].n != all[j].n {
			return all[i].n > all[j].n
		}
		return all[i].file < all[j].file
	})
	var out []string
	for i, e := range all {
		if i == n {
			break
		}
		out = append(out, e.file)
	}
	return out
}

// Keyed by the root-relative path a parsed File carries, so the two compare.
func goBlobsAt(cfg *Config, rev string) (map[string]string, error) {
	out, err := git(cfg.Root, "ls-tree", "-r", "-z", rev, "--", ".")
	if err != nil {
		return nil, fmt.Errorf("git ls-tree %s: %w", rev, err)
	}
	blobs := map[string]string{}
	for _, entry := range strings.Split(out, "\x00") {
		meta, path, ok := strings.Cut(entry, "\t")
		fields := strings.Fields(meta)
		if !ok || len(fields) != 3 || fields[1] != "blob" {
			continue
		}
		if lang, ok := langOf(path); !ok || lang != LangGo || strings.HasSuffix(path, "_test.go") || cfg.excluded(path) {
			continue
		}
		blobs[path] = fields[2]
	}
	return blobs, nil
}

// declarationsAt reads every blob through one `git cat-file --batch`: one
// process per file costs seconds on a module of a thousand files.
func declarationsAt(cfg *Config, blobs map[string]string, res *checkResult) (map[string]bool, error) {
	paths := make([]string, 0, len(blobs))
	for path := range blobs {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	var in strings.Builder
	for _, path := range paths {
		in.WriteString(blobs[path] + "\n")
	}
	cmd := exec.Command("git", "cat-file", "--batch")
	cmd.Dir = cfg.Root
	cmd.Stdin = strings.NewReader(in.String())
	raw, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git cat-file --batch: %w", err)
	}

	known := map[string]bool{}
	r := bufio.NewReader(bytes.NewReader(raw))
	for _, path := range paths {
		header, err := r.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("git cat-file --batch: %s: %w", path, err)
		}
		fields := strings.Fields(header)
		if len(fields) == 2 && fields[1] == "missing" {
			res.Notices = append(res.Notices, fmt.Sprintf("CPX-09 read no declarations from %s at the merge-base: blob %s is missing", path, fields[0]))
			continue
		}
		if len(fields) != 3 {
			return nil, fmt.Errorf("git cat-file --batch: %s: unexpected header %q", path, strings.TrimSpace(header))
		}
		size, err := strconv.Atoi(fields[2])
		if err != nil {
			return nil, fmt.Errorf("git cat-file --batch: %s: %w", path, err)
		}
		src := make([]byte, size+1) // the content is followed by a newline
		if _, err := io.ReadFull(r, src); err != nil {
			return nil, fmt.Errorf("git cat-file --batch: %s: %w", path, err)
		}
		decls, err := goTopLevelDecls(path, src[:size])
		if err != nil {
			res.ParseErrors = append(res.ParseErrors, fmt.Sprintf("%s at the merge-base: %v", path, err))
			continue
		}
		for _, d := range decls {
			known[declarationKey(d)] = true
		}
	}
	return known, nil
}
