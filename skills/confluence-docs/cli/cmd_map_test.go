package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/diegoclair/harness/pkg/atlassian/adf"
	"github.com/diegoclair/harness/pkg/atlassian/auth"
	"github.com/diegoclair/harness/pkg/atlassian/setup"
)

// sampleMap mirrors what buildSpaceMap emits: depth-first, so a fixture can
// never disagree with the walker about ordering.
func sampleMap() spaceMap {
	return spaceMap{
		Space:     "ENG",
		Generated: time.Now().UTC(),
		Entries: []mapEntry{
			{ID: "1", Depth: 0, Title: "Home", Type: "reference"},
			{ID: "2", Depth: 1, ParentID: "1", Title: "Checkout decisions", Type: "decision", Updated: "2026-08-01"},
			{ID: "3", Depth: 2, ParentID: "2", Title: "Payment provider", Type: "decision", Updated: "2020-01-01"},
			{ID: "4", Depth: 1, ParentID: "1", Title: "Runbooks", Type: "howto", Updated: "2026-08-10"},
		},
	}
}

func TestMapCacheRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "map-ENG.tsv")
	want := sampleMap()

	if err := saveSpaceMap(path, want); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, age, err := loadSpaceMap(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Space != want.Space || len(got.Entries) != len(want.Entries) {
		t.Fatalf("got space=%q entries=%d", got.Space, len(got.Entries))
	}
	if age > time.Minute {
		t.Errorf("age = %v, want it derived from the header timestamp", age)
	}
	for i, e := range got.Entries {
		if e != want.Entries[i] {
			t.Errorf("entry %d = %+v, want %+v", i, e, want.Entries[i])
		}
	}
}

// A tab or newline in a title would shift every following column.
func TestMapCacheSurvivesAwkwardTitles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "map-ENG.tsv")
	sm := spaceMap{Space: "ENG", Generated: time.Now().UTC(), Entries: []mapEntry{
		{ID: "1", Depth: 0, Title: "Tabbed\ttitle\nsecond line"},
		{ID: "2", Depth: 1, ParentID: "1", Title: "Next"},
	}}

	if err := saveSpaceMap(path, sm); err != nil {
		t.Fatal(err)
	}
	got, _, err := loadSpaceMap(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 2 {
		t.Fatalf("got %d entries, want 2 — a title broke the row", len(got.Entries))
	}
	if got.Entries[1].ID != "2" || got.Entries[1].Title != "Next" {
		t.Errorf("second row corrupted: %+v", got.Entries[1])
	}
}

func TestMapLoadReportsAMissingCache(t *testing.T) {
	if _, _, err := loadSpaceMap(filepath.Join(t.TempDir(), "absent.tsv")); err == nil {
		t.Fatal("want an error for a missing cache")
	}
}

func TestMapFilterDepth(t *testing.T) {
	got := filterMap(sampleMap(), mapFilter{depth: 1, staleDays: -1})
	if len(got) != 3 {
		t.Fatalf("got %d entries, want the two top levels", len(got))
	}
	for _, e := range got {
		if e.Depth > 1 {
			t.Errorf("depth %d leaked past the limit", e.Depth)
		}
	}
}

// A match deep in the tree is useless without knowing where it sits.
func TestMapFilterFindKeepsAncestors(t *testing.T) {
	got := filterMap(sampleMap(), mapFilter{depth: -1, find: "payment", staleDays: -1})

	ids := mapIDs(got)
	for _, want := range []string{"1", "2", "3"} {
		if !strings.Contains(ids, want) {
			t.Errorf("ids = %s, want the match and its ancestors", ids)
		}
	}
	if strings.Contains(ids, "4") {
		t.Errorf("ids = %s, want unrelated branches excluded", ids)
	}
}

func TestMapFilterFindIsCaseInsensitive(t *testing.T) {
	if got := filterMap(sampleMap(), mapFilter{depth: -1, find: "RUNBOOKS", staleDays: -1}); len(got) == 0 {
		t.Error("find should not be case sensitive")
	}
}

func TestMapFilterChildrenIsOneLevelOnly(t *testing.T) {
	got := filterMap(sampleMap(), mapFilter{depth: -1, children: "1", staleDays: -1})

	if ids := mapIDs(got); ids != "2,4" {
		t.Errorf("ids = %s, want only the direct children of 1", ids)
	}
}

func TestMapFilterByType(t *testing.T) {
	got := filterMap(sampleMap(), mapFilter{depth: -1, typeFilter: "howto", staleDays: -1})

	ids := mapIDs(got)
	if !strings.Contains(ids, "4") {
		t.Errorf("ids = %s, want the howto page", ids)
	}
	if strings.Contains(ids, "3") {
		t.Errorf("ids = %s, want other types excluded", ids)
	}
}

func TestMapFilterStale(t *testing.T) {
	got := filterMap(sampleMap(), mapFilter{depth: -1, staleDays: 90})

	ids := mapIDs(got)
	if !strings.Contains(ids, "3") {
		t.Errorf("ids = %s, want the page last touched in 2020", ids)
	}
	if strings.Contains(ids, "4") {
		t.Errorf("ids = %s, want recently updated pages excluded", ids)
	}
}

// An entry with no date must not be reported as stale on a parse failure.
func TestMapStaleIgnoresPagesWithNoDate(t *testing.T) {
	sm := spaceMap{Entries: []mapEntry{{ID: "1", Depth: 0, Title: "No date"}}}

	if got := filterMap(sm, mapFilter{depth: -1, staleDays: 30}); len(got) != 0 {
		t.Errorf("got %d entries, want none: an unknown date is not evidence of staleness", len(got))
	}
}

func TestMapOutlineIndentsByDepth(t *testing.T) {
	var out bytes.Buffer
	printMapOutline(sampleMap().Entries, &out)

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if strings.HasPrefix(lines[0], " ") {
		t.Errorf("root should not be indented: %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "  2") {
		t.Errorf("depth 1 should be indented once: %q", lines[1])
	}
	if !strings.HasPrefix(lines[2], "    3") {
		t.Errorf("depth 2 should be indented twice: %q", lines[2])
	}
	if !strings.Contains(lines[1], "[decision]") {
		t.Errorf("the type should be shown when known: %q", lines[1])
	}
}

func TestMapJSONIsParseable(t *testing.T) {
	var out bytes.Buffer
	if _, err := printMapJSON(sampleMap(), sampleMap().Entries, &out); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{`"space":"ENG"`, `"pages":4`, `"id":"3"`, `"parentId":"2"`, `"type":"decision"`} {
		if !strings.Contains(s, want) {
			t.Errorf("json is missing %s:\n%s", want, s)
		}
	}
}

// The properties block the pages already carry is where type and status come
// from — the whole reason this index needs no classification pass.
func TestPropertyPatternReadsTypeAndStatus(t *testing.T) {
	excerpt := "PayPal type reference status active owner Diego Clair related 213024814 created 2026-08-05"

	got := map[string]string{}
	for _, m := range propertyPattern.FindAllStringSubmatch(excerpt, -1) {
		got[m[1]] = m[2]
	}
	if got["type"] != "reference" || got["status"] != "active" {
		t.Errorf("parsed %v, want type=reference status=active", got)
	}
}

// A configured space is required, otherwise every case would "pass" on the
// missing-config error instead of on flag handling — and the command would read
// the developer's real config and hit the network.
func TestMapRejectsBadFlagValues(t *testing.T) {
	cases := []struct {
		args     []string
		wantHint string
	}{
		{[]string{"--depth", "-1"}, "--depth must be"},
		{[]string{"--depth", "abc"}, "--depth must be"},
		{[]string{"--stale", "soon"}, "--stale must be"},
		{[]string{"--find"}, "--find requires a value"},
		{[]string{"--children"}, "--children requires a value"},
		{[]string{"--type"}, "--type requires a value"},
		{[]string{"--nope"}, "unknown flag"},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			dir := t.TempDir()
			overrideConfigDirMain(t, dir)
			overrideCacheDirMain(t, dir)
			writeTestConfig(t, dir, setup.Config{Cloud: "acme", SpaceKey: "ENG", HomePageID: "1"})

			var stdout, stderr bytes.Buffer
			code, _ := runMap(tc.args, &stdout, &stderr)
			if code != exitInputErr {
				t.Errorf("code = %d, want %d (stderr=%q)", code, exitInputErr, stderr.String())
			}
			if !strings.Contains(stderr.String(), tc.wantHint) {
				t.Errorf("stderr = %q, want it to mention %q", stderr.String(), tc.wantHint)
			}
		})
	}
}

func TestMapHelpNeedsNoConfiguration(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var stdout, stderr bytes.Buffer

	code, err := runMap([]string{"--help"}, &stdout, &stderr)
	if code != exitOK || err != nil {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if !strings.Contains(stdout.String(), "no model tokens") {
		t.Errorf("help should state the point of the command:\n%s", stdout.String())
	}
}

func TestMapStatusReportsAMissingCache(t *testing.T) {
	var out bytes.Buffer
	code, err := reportMapStatus(spaceMap{}, 0, os.ErrNotExist, "/tmp/x.tsv", &out)
	if code != exitOK || err != nil {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if !strings.Contains(out.String(), "--refresh") {
		t.Errorf("status should say how to build the index:\n%s", out.String())
	}
}

func TestMapStatusFlagsAStaleCache(t *testing.T) {
	var out bytes.Buffer
	sm := spaceMap{Space: "ENG", Generated: time.Now().Add(-3 * time.Hour), Entries: sampleMap().Entries}

	if _, err := reportMapStatus(sm, 3*time.Hour, nil, "/tmp/x.tsv", &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "stale") {
		t.Errorf("a 3h-old cache should be reported stale:\n%s", out.String())
	}
}

func mapIDs(entries []mapEntry) string {
	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		ids = append(ids, e.ID)
	}
	return strings.Join(ids, ",")
}

// treeTransport answers the children endpoint from a fixture tree, so the walk
// itself can be exercised without a network.
type treeTransport struct {
	children map[string][]string
	titles   map[string]string
	fail     map[string]bool
	calls    int
}

func (tt *treeTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	tt.calls++
	respond := func(body string) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
			Request:    r,
		}, nil
	}
	path := r.URL.Path

	if strings.Contains(path, "/children") {
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/wiki/api/v2/pages/"), "/children")
		if tt.fail[id] {
			return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader(`{"message":"boom"}`)),
				Header: make(http.Header), Request: r}, nil
		}
		var rows []string
		for _, c := range tt.children[id] {
			rows = append(rows, fmt.Sprintf(`{"id":%q,"title":%q}`, c, tt.titles[c]))
		}
		return respond(`{"results":[` + strings.Join(rows, ",") + `]}`)
	}
	if strings.Contains(path, "/search") {
		return respond(`{"results":[]}`)
	}
	// single page fetch, for the root title
	id := strings.TrimPrefix(path, "/wiki/api/v2/pages/")
	return respond(fmt.Sprintf(`{"id":%q,"title":%q}`, id, tt.titles[id]))
}

func mapTestClient(t *testing.T, tt *treeTransport) *adf.ConfluenceClient {
	t.Helper()
	prev := http.DefaultTransport
	http.DefaultTransport = tt
	t.Cleanup(func() { http.DefaultTransport = prev })
	return adf.NewClientWithAuthorizer(auth.Basic{Cloud: "acme", Email: "e@x.com", Token: "t"})
}

// Two branches at the same depth must not have their children interleaved:
// the outline indents by depth, so emission order IS the tree.
func TestBuildSpaceMapEmitsDepthFirst(t *testing.T) {
	tt := &treeTransport{
		children: map[string][]string{"1": {"2", "3"}, "2": {"4"}, "3": {"5"}},
		titles:   map[string]string{"1": "Home", "2": "A", "3": "B", "4": "A1", "5": "B1"},
	}
	sm, err := buildSpaceMap(mapTestClient(t, tt), "ENG", "1", io.Discard)
	if err != nil {
		t.Fatal(err)
	}

	if got := mapIDs(sm.Entries); got != "1,2,4,3,5" {
		t.Errorf("order = %s, want depth-first 1,2,4,3,5 — otherwise the outline\n"+
			"draws a child under the wrong parent", got)
	}
	var out bytes.Buffer
	printMapOutline(sm.Entries, &out)
	for _, want := range []string{"1  Home", "  2  A", "    4  A1", "  3  B", "    5  B1"} {
		if !strings.Contains(out.String(), want+"\n") {
			t.Errorf("outline is missing %q:\n%s", want, out.String())
		}
	}
}

func TestBuildSpaceMapRecordsParentAndDepth(t *testing.T) {
	tt := &treeTransport{
		children: map[string][]string{"1": {"2"}, "2": {"3"}},
		titles:   map[string]string{"1": "Home", "2": "Mid", "3": "Leaf"},
	}
	sm, err := buildSpaceMap(mapTestClient(t, tt), "ENG", "1", io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	leaf := sm.Entries[2]
	if leaf.ID != "3" || leaf.ParentID != "2" || leaf.Depth != 2 {
		t.Errorf("leaf = %+v, want id=3 parent=2 depth=2", leaf)
	}
}

// A page reachable from two parents must not send the walk into a loop.
func TestBuildSpaceMapStopsOnACycle(t *testing.T) {
	tt := &treeTransport{
		children: map[string][]string{"1": {"2"}, "2": {"3"}, "3": {"2"}},
		titles:   map[string]string{"1": "Home", "2": "A", "3": "B"},
	}
	done := make(chan spaceMap, 1)
	go func() {
		sm, _ := buildSpaceMap(mapTestClient(t, tt), "ENG", "1", io.Discard)
		done <- sm
	}()
	select {
	case sm := <-done:
		if len(sm.Entries) != 3 {
			t.Errorf("got %d entries, want each page once", len(sm.Entries))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the walk did not terminate on a cycle")
	}
}

// One unreadable branch must cost that branch, not the whole index.
func TestBuildSpaceMapSurvivesAFailingBranch(t *testing.T) {
	tt := &treeTransport{
		children: map[string][]string{"1": {"2", "3"}, "2": {"4"}, "3": {"5"}},
		titles:   map[string]string{"1": "Home", "2": "A", "3": "B", "4": "A1", "5": "B1"},
		fail:     map[string]bool{"2": true},
	}
	var errOut bytes.Buffer
	sm, err := buildSpaceMap(mapTestClient(t, tt), "ENG", "1", &errOut)
	if err != nil {
		t.Fatal(err)
	}
	ids := mapIDs(sm.Entries)
	if !strings.Contains(ids, "3") || !strings.Contains(ids, "5") {
		t.Errorf("ids = %s, want the healthy branch kept", ids)
	}
	if !strings.Contains(errOut.String(), "skipping children of 2") {
		t.Errorf("the skipped branch must be reported: %q", errOut.String())
	}
}

// A subtree walked with --root must not be served as, or overwrite, the
// space-wide index.
func TestMapCachePathSeparatesRootedWalks(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "/tmp/cache")
	whole, err := mapCachePath("ENG", "")
	if err != nil {
		t.Fatal(err)
	}
	rooted, err := mapCachePath("ENG", "42")
	if err != nil {
		t.Fatal(err)
	}
	if whole == rooted {
		t.Errorf("both walks share the cache file %s", whole)
	}
}

// A half-written cache must never be readable as a complete one.
func TestSaveSpaceMapIsAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "map-ENG.tsv")
	if err := saveSpaceMap(path, sampleMap()); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("a temp file was left behind: %s", e.Name())
		}
	}
}

func TestMapJSONParsesAsJSON(t *testing.T) {
	sm := sampleMap()
	sm.Entries[0].Title = "Weird \"quoted\" \\ title"

	var out bytes.Buffer
	if _, err := printMapJSON(sm, sm.Entries, &out); err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Space   string `json:"space"`
		Pages   int    `json:"pages"`
		Entries []struct {
			ID       string `json:"id"`
			ParentID string `json:"parentId"`
			Title    string `json:"title"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out.String())
	}
	if parsed.Space != "ENG" || parsed.Pages != 4 || parsed.Entries[0].Title != sm.Entries[0].Title {
		t.Errorf("parsed = %+v", parsed)
	}
}
