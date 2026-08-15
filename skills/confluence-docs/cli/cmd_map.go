// cmd_map.go — `confluence-docs map` subcommand.
//
// A structural index of the active space, built entirely from the REST API so
// producing it costs no model tokens, cached locally, and read in slices so an
// agent can orient itself in a large space without pulling the whole tree into
// context.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/diegoclair/harness/pkg/atlassian/adf"
)

// mapCacheTTL mirrors the home cache: fresh enough to reuse across a session,
// short enough that a stale tree is not silently trusted for long.
const mapCacheTTL = time.Hour

// searchEnrichLimit is what one CQL call returns. Beyond it, type/status/date
// are simply unknown for the remaining pages — never guessed.
const searchEnrichLimit = 250

// mapEntry is one page in the space tree.
type mapEntry struct {
	ID       string
	Depth    int
	ParentID string
	Type     string
	Status   string
	Updated  string
	Title    string
}

type spaceMap struct {
	Space     string
	Generated time.Time
	Entries   []mapEntry
}

// mapCachePath keys the cache by root as well as space: a subtree walked with
// --root must never overwrite, or be served from, the space-wide index.
func mapCachePath(space, root string) (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolve cache dir: %w", err)
	}
	name := "map-" + space
	if root != "" {
		name += "-root" + root
	}
	return filepath.Join(dir, "confluence-docs", name+".tsv"), nil
}

const mapUsage = `map — structural index of the active space, for finding your way around.

Built from the REST API, so generating it costs no model tokens. Cached
locally and read in slices, so a large space never has to be loaded at once.

USAGE:
  confluence-docs map [--depth N]        Outline of the tree (default view)
  confluence-docs map --refresh          Rebuild the cache from the API
  confluence-docs map --find TERM        Only branches whose title matches
  confluence-docs map --children ID      Direct children of one page
  confluence-docs map --type TYPE        Only pages of a given type (where pages carry one)
  confluence-docs map --stale DAYS       Pages untouched for that many days
  confluence-docs map --status           Cache age and page count
  confluence-docs map --json             Machine-readable output

FLAGS:
  --depth N        Limit the outline to N levels (default: all)
  --root ID        Walk from this page instead of the configured Home
  --no-refresh     Fail instead of refreshing a stale or missing cache
`

func runMap(args []string, stdout, stderr io.Writer) (int, error) {
	var (
		depth      = -1
		find       string
		children   string
		typeFilter string
		staleDays  = -1
		root       string
		refresh    bool
		showStatus bool
		asJSON     bool
		noRefresh  bool
	)

	remaining, cloud, email, token, err := parseCommonPageFlags(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitInputErr, errInvalidUsage
	}

	for i := 0; i < len(remaining); i++ {
		a := remaining[i]
		next := func() (string, bool) {
			if i+1 >= len(remaining) {
				fmt.Fprintf(stderr, "map: %s requires a value\n", a)
				return "", false
			}
			i++
			return remaining[i], true
		}
		var ok bool
		switch a {
		case "-h", "--help":
			fmt.Fprint(stdout, mapUsage)
			return exitOK, nil
		case "--refresh":
			refresh = true
		case "--no-refresh":
			noRefresh = true
		case "--status":
			showStatus = true
		case "--json":
			asJSON = true
		case "--depth":
			v, k := next()
			if !k {
				return exitInputErr, errInvalidUsage
			}
			if depth, err = strconv.Atoi(v); err != nil || depth < 0 {
				fmt.Fprintln(stderr, "map: --depth must be a non-negative number")
				return exitInputErr, errInvalidUsage
			}
		case "--stale":
			v, k := next()
			if !k {
				return exitInputErr, errInvalidUsage
			}
			if staleDays, err = strconv.Atoi(strings.TrimSuffix(v, "d")); err != nil || staleDays < 0 {
				fmt.Fprintln(stderr, "map: --stale must be a number of days, e.g. 90")
				return exitInputErr, errInvalidUsage
			}
		case "--find":
			if find, ok = next(); !ok {
				return exitInputErr, errInvalidUsage
			}
		case "--children":
			if children, ok = next(); !ok {
				return exitInputErr, errInvalidUsage
			}
		case "--type":
			if typeFilter, ok = next(); !ok {
				return exitInputErr, errInvalidUsage
			}
		case "--root":
			if root, ok = next(); !ok {
				return exitInputErr, errInvalidUsage
			}
		default:
			fmt.Fprintf(stderr, "map: unknown flag %q\n", a)
			return exitInputErr, errInvalidUsage
		}
	}

	space, err := currentSpaceKey()
	if err != nil {
		fmt.Fprintln(stderr, "map:", err)
		return exitInputErr, err
	}
	path, err := mapCachePath(space, root)
	if err != nil {
		fmt.Fprintln(stderr, "map:", err)
		return exitUnknownErr, err
	}

	sm, age, cacheErr := loadSpaceMap(path)
	stale := cacheErr != nil || age > mapCacheTTL

	if showStatus {
		return reportMapStatus(sm, age, cacheErr, path, stdout)
	}

	if refresh || stale {
		if noRefresh && !refresh {
			fmt.Fprintln(stderr, "map: cache is missing or stale and --no-refresh was given")
			return exitInputErr, errInvalidUsage
		}
		client, ok := buildClient(cloud, email, token, stderr)
		if !ok {
			return exitUnknownErr, nil
		}
		built, berr := buildSpaceMap(client, space, root, stderr)
		if berr != nil {
			// A stale cache still answers better than nothing.
			if cacheErr != nil {
				fmt.Fprintln(stderr, "map: building the index:", berr)
				return exitUnknownErr, berr
			}
			fmt.Fprintln(stderr, "map: refresh failed, using the cached index:", berr)
		} else {
			sm = built
			if werr := saveSpaceMap(path, sm); werr != nil {
				fmt.Fprintln(stderr, "map: writing cache:", werr)
			}
		}
	}

	if typeFilter != "" || staleDays >= 0 {
		warnThinMetadata(sm, typeFilter, staleDays, stderr)
	}

	entries := filterMap(sm, mapFilter{
		depth: depth, find: find, children: children,
		typeFilter: typeFilter, staleDays: staleDays,
	})

	if asJSON {
		return printMapJSON(sm, entries, stdout)
	}
	printMapOutline(entries, stdout)
	return exitOK, nil
}

// buildSpaceMap walks the page tree from the root. Children come one level at
// a time; there is no bulk endpoint that returns a whole space tree.
func buildSpaceMap(client *adf.ConfluenceClient, space, root string, stderr io.Writer) (spaceMap, error) {
	if root == "" {
		var err error
		if root, err = currentHomePageID(); err != nil {
			return spaceMap{}, fmt.Errorf("no root page: %w", err)
		}
	}

	sm := spaceMap{Space: space, Generated: time.Now().UTC()}
	rootTitle := "(root)"
	if meta, err := client.GetPage(root, ""); err == nil && meta != nil && meta.Title != "" {
		rootTitle = meta.Title
	}
	sm.Entries = append(sm.Entries, mapEntry{ID: root, Depth: 0, Title: rootTitle})

	// Depth-first, so the emitted order is the tree order the outline draws.
	// Breadth-first would interleave siblings of different parents and put each
	// child under whatever shallower row happened to come before it.
	seen := map[string]bool{root: true}
	var walk func(id string, depth int)
	walk = func(id string, depth int) {
		kids, err := client.GetPageChildren(id)
		if err != nil {
			// One unreadable branch must not lose the rest of the tree.
			fmt.Fprintf(stderr, "map: skipping children of %s: %v\n", id, err)
			return
		}
		for _, k := range kids {
			if seen[k.ID] {
				continue // a cycle would otherwise walk forever
			}
			seen[k.ID] = true
			sm.Entries = append(sm.Entries, mapEntry{
				ID: k.ID, Depth: depth + 1, ParentID: id, Title: k.Title,
			})
			walk(k.ID, depth+1)
		}
	}
	walk(root, 0)

	enrichFromSearch(client, space, sm.Entries, stderr)
	return sm, nil
}

// propertyPattern reads the `type`/`status` values the :::properties macro
// renders into the page, which CQL returns in the excerpt. This is metadata the
// pages already carry, so no classification pass is needed to obtain it.
var propertyPattern = regexp.MustCompile(`\b(type|status)\s+([a-z0-9-]+)`)

// enrichFromSearch fills type, status and the last-modified date from a single
// bulk CQL call — the metadata pages already carry, so no classification pass
// is needed. Pages beyond the call's limit, or without a :::properties block,
// keep empty values, and the filters treat empty as "unknown" rather than
// guessing.
func enrichFromSearch(client *adf.ConfluenceClient, space string, entries []mapEntry, stderr io.Writer) {
	rows, err := client.SearchCQL(fmt.Sprintf("space=%q AND type=page", space), searchEnrichLimit)
	if err != nil {
		fmt.Fprintf(stderr, "map: type/status unavailable: %v\n", err)
		return
	}
	byID := map[string]*mapEntry{}
	for i := range entries {
		byID[entries[i].ID] = &entries[i]
	}
	for _, r := range rows {
		e, ok := byID[r.PageID]
		if !ok {
			continue
		}
		if len(r.LastModified) >= 10 {
			e.Updated = r.LastModified[:10]
		}
		for _, m := range propertyPattern.FindAllStringSubmatch(r.Excerpt, -1) {
			switch m[1] {
			case "type":
				if e.Type == "" {
					e.Type = m[2]
				}
			case "status":
				if e.Status == "" {
					e.Status = m[2]
				}
			}
		}
	}
}

type mapFilter struct {
	depth      int
	find       string
	children   string
	typeFilter string
	staleDays  int
}

// filterMap narrows the tree. When a filter matches, ancestors are kept so the
// result still reads as a tree rather than a flat list of orphans.
func filterMap(sm spaceMap, f mapFilter) []mapEntry {
	if f.children != "" {
		var out []mapEntry
		for _, e := range sm.Entries {
			if e.ParentID == f.children {
				out = append(out, e)
			}
		}
		return out
	}

	matched := map[string]bool{}
	byID := map[string]mapEntry{}
	for _, e := range sm.Entries {
		byID[e.ID] = e
	}

	anyFilter := f.find != "" || f.typeFilter != "" || f.staleDays >= 0
	cutoff := time.Now().AddDate(0, 0, -f.staleDays)

	for _, e := range sm.Entries {
		if f.depth >= 0 && e.Depth > f.depth {
			continue
		}
		if !anyFilter {
			matched[e.ID] = true
			continue
		}
		if f.find != "" && !strings.Contains(strings.ToLower(e.Title), strings.ToLower(f.find)) {
			continue
		}
		if f.typeFilter != "" && !strings.EqualFold(e.Type, f.typeFilter) {
			continue
		}
		if f.staleDays >= 0 {
			t, err := time.Parse("2006-01-02", e.Updated)
			if err != nil || t.After(cutoff) {
				continue
			}
		}
		matched[e.ID] = true
		for p := e.ParentID; p != ""; {
			parent, ok := byID[p]
			if !ok || matched[p] {
				break
			}
			matched[p] = true
			p = parent.ParentID
		}
	}

	var out []mapEntry
	for _, e := range sm.Entries {
		if matched[e.ID] {
			out = append(out, e)
		}
	}
	return out
}

// warnThinMetadata keeps a filter from reading as "nothing matches" when the
// truth is that most pages never carried the metadata being filtered on.
func warnThinMetadata(sm spaceMap, typeFilter string, staleDays int, stderr io.Writer) {
	known := 0
	for _, e := range sm.Entries {
		if typeFilter != "" && e.Type != "" {
			known++
		}
		if staleDays >= 0 && e.Updated != "" {
			known++
		}
	}
	total := len(sm.Entries)
	if total == 0 || known*2 >= total {
		return
	}
	fmt.Fprintf(stderr, "map: only %d of %d pages carry this metadata; the rest are unknown, not excluded\n",
		known, total)
}

func printMapOutline(entries []mapEntry, out io.Writer) {
	for _, e := range entries {
		meta := ""
		if e.Type != "" {
			meta = "  [" + e.Type + "]"
		}
		fmt.Fprintf(out, "%s%s  %s%s\n", strings.Repeat("  ", e.Depth), e.ID, e.Title, meta)
	}
}

func printMapJSON(sm spaceMap, entries []mapEntry, out io.Writer) (int, error) {
	type jsonEntry struct {
		ID       string `json:"id"`
		Depth    int    `json:"depth"`
		ParentID string `json:"parentId"`
		Type     string `json:"type"`
		Status   string `json:"status"`
		Updated  string `json:"updated"`
		Title    string `json:"title"`
	}
	payload := struct {
		Space     string      `json:"space"`
		Generated string      `json:"generated"`
		Pages     int         `json:"pages"`
		Entries   []jsonEntry `json:"entries"`
	}{Space: sm.Space, Generated: sm.Generated.Format(time.RFC3339), Pages: len(entries),
		Entries: make([]jsonEntry, 0, len(entries))}
	for _, e := range entries {
		payload.Entries = append(payload.Entries, jsonEntry{
			e.ID, e.Depth, e.ParentID, e.Type, e.Status, e.Updated, e.Title,
		})
	}
	enc := json.NewEncoder(out)
	if err := enc.Encode(payload); err != nil {
		return exitUnknownErr, err
	}
	return exitOK, nil
}

func reportMapStatus(sm spaceMap, age time.Duration, cacheErr error, path string, out io.Writer) (int, error) {
	if cacheErr != nil {
		fmt.Fprintf(out, "no index cached yet (%s)\nrun: confluence-docs map --refresh\n", path)
		return exitOK, nil
	}
	withType, withDate := 0, 0
	for _, e := range sm.Entries {
		if e.Type != "" {
			withType++
		}
		if e.Updated != "" {
			withDate++
		}
	}
	fmt.Fprintf(out, "space:     %s\npages:     %d\ntype known: %d/%d\ndate known: %d/%d\ngenerated: %s (%s ago)\ncache:     %s\n",
		sm.Space, len(sm.Entries), withType, len(sm.Entries), withDate, len(sm.Entries),
		sm.Generated.Format(time.RFC3339), age.Truncate(time.Minute), path)
	if age > mapCacheTTL {
		fmt.Fprintln(out, "status:    stale — the next read refreshes it")
	} else {
		fmt.Fprintln(out, "status:    fresh")
	}
	return exitOK, nil
}

// ── cache format ──────────────────────────────────────────────────────────
//
// TSV with a comment header: compact, greppable, and cheap to read a slice of.

func saveSpaceMap(path string, sm spaceMap) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "# space=%s generated=%s pages=%d\n",
		sm.Space, sm.Generated.Format(time.RFC3339), len(sm.Entries))
	sb.WriteString("# id\tdepth\tparentId\ttype\tstatus\tupdated\ttitle\n")
	for _, e := range sm.Entries {
		fmt.Fprintf(&sb, "%s\t%d\t%s\t%s\t%s\t%s\t%s\n",
			e.ID, e.Depth, e.ParentID, e.Type, e.Status, e.Updated, sanitizeTSV(e.Title))
	}
	// Temp + rename: two sessions refreshing at once must not interleave into a
	// file that looks complete.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(sb.String()), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// sanitizeTSV keeps a title from breaking the row it lives in.
func sanitizeTSV(s string) string {
	return strings.NewReplacer("\t", " ", "\n", " ", "\r", " ").Replace(s)
}

func loadSpaceMap(path string) (spaceMap, time.Duration, error) {
	f, err := os.Open(path)
	if err != nil {
		return spaceMap{}, 0, err
	}
	defer f.Close()

	var sm spaceMap
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "#") {
			parseMapHeader(line, &sm)
			continue
		}
		cols := strings.Split(line, "\t")
		if len(cols) != 7 {
			continue
		}
		depth, _ := strconv.Atoi(cols[1])
		sm.Entries = append(sm.Entries, mapEntry{
			ID: cols[0], Depth: depth, ParentID: cols[2],
			Type: cols[3], Status: cols[4], Updated: cols[5], Title: cols[6],
		})
	}
	if err := sc.Err(); err != nil {
		return spaceMap{}, 0, err
	}
	if len(sm.Entries) == 0 {
		return sm, 0, fmt.Errorf("cache at %s has no entries", path)
	}
	return sm, time.Since(sm.Generated), nil
}

func parseMapHeader(line string, sm *spaceMap) {
	for _, field := range strings.Fields(strings.TrimPrefix(line, "#")) {
		k, v, ok := strings.Cut(field, "=")
		if !ok {
			continue
		}
		switch k {
		case "space":
			sm.Space = v
		case "generated":
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				sm.Generated = t
			}
		}
	}
}
