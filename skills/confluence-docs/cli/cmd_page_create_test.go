// cmd_page_create_test.go — unit tests for `confluence-docs page create`
// space-id resolution: default from active config, space-key lookup, and
// the nil-deref / panic regression this guards against.
package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/diegoclair/harness/pkg/atlassian/adf"
	"github.com/diegoclair/harness/pkg/atlassian/setup"
)

// runPageCreateCmd runs `confluence-docs page create <args...>` against dir-scoped
// config/cache and the given mock transport.
func runPageCreateCmd(t *testing.T, dir string, rt http.RoundTripper, args ...string) (stdout, stderr string, code int) {
	t.Helper()

	orig := http.DefaultTransport
	http.DefaultTransport = rt
	t.Cleanup(func() { http.DefaultTransport = orig })

	t.Setenv("ATLASSIAN_API_TOKEN", "testtoken")
	t.Setenv("ATLASSIAN_EMAIL", "test@example.com")
	t.Setenv("ATLASSIAN_CLOUD", "testcloud")

	var outBuf, errBuf bytes.Buffer
	code, _ = run(append([]string{"page", "create"}, args...), strings.NewReader(""), &outBuf, &errBuf)
	return outBuf.String(), errBuf.String(), code
}

func createPageResponseJSON(id, title string) string {
	b, _ := json.Marshal(map[string]any{
		"id":    id,
		"title": title,
		"_links": map[string]any{
			"webui": "/spaces/lybel/pages/" + id,
		},
	})
	return string(b)
}

func writeMarkdownFile(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "page.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPageCreate_NoSpaceIDUsesActiveConfig(t *testing.T) {
	dir := t.TempDir()
	overrideConfigDirMain(t, dir)
	overrideCacheDirMain(t, dir)
	writeTestConfig(t, dir, setup.Config{
		Cloud:     "testcloud",
		SpaceID:   "131352",
		SpaceKey:  "lybel",
		SpaceName: "Lybel",
	})

	md := writeMarkdownFile(t, dir, "# Hello\n\nBody text.\n")
	rt := &mockRoundTripper{statusCode: 200, body: createPageResponseJSON("999", "X")}

	out, errOut, code := runPageCreateCmd(t, dir, rt,
		"--parent-id", "212992001", "--title", "X", "--markdown", md)
	if code != exitOK {
		t.Fatalf("want exit 0, got %d\nstdout: %s\nstderr: %s", code, out, errOut)
	}
	if !strings.Contains(out, `"pageId": "999"`) {
		t.Errorf("expected pageId 999 in output, got: %s", out)
	}
}

func TestPageCreate_MissingSpaceIDAndConfigErrorsCleanly(t *testing.T) {
	dir := t.TempDir()
	overrideConfigDirMain(t, dir)
	overrideCacheDirMain(t, dir)
	// No config written — no active space.

	md := writeMarkdownFile(t, dir, "# Hello\n")
	rt := &mockRoundTripper{statusCode: 200}

	_, errOut, code := runPageCreateCmd(t, dir, rt,
		"--parent-id", "212992001", "--title", "X", "--markdown", md)
	if code == exitOK {
		t.Fatal("expected non-zero exit when no --space-id and no active space configured")
	}
	if !strings.Contains(errOut, "space") {
		t.Errorf("expected a space-related error message, got: %s", errOut)
	}
}

func TestPageCreate_SpaceKeyMatchingActiveConfigResolvesWithoutSpaceList(t *testing.T) {
	dir := t.TempDir()
	overrideConfigDirMain(t, dir)
	overrideCacheDirMain(t, dir)
	writeTestConfig(t, dir, setup.Config{
		Cloud:     "testcloud",
		SpaceID:   "131352",
		SpaceKey:  "lybel",
		SpaceName: "Lybel",
	})

	md := writeMarkdownFile(t, dir, "# Hello\n")
	rt := &mockRoundTripper{statusCode: 200, body: createPageResponseJSON("999", "X")}

	out, errOut, code := runPageCreateCmd(t, dir, rt,
		"--space-id", "lybel", "--parent-id", "212992001", "--title", "X", "--markdown", md)
	if code != exitOK {
		t.Fatalf("want exit 0, got %d\nstdout: %s\nstderr: %s", code, out, errOut)
	}
	// Resolved via config, not the spaces API — only the create call should fire.
	if rt.calls != 1 {
		t.Errorf("expected exactly 1 HTTP call (create only), got %d", rt.calls)
	}
}

func TestPageCreate_SpaceKeyResolvedViaSpacesAPI(t *testing.T) {
	dir := t.TempDir()
	overrideConfigDirMain(t, dir)
	overrideCacheDirMain(t, dir)
	writeTestConfig(t, dir, setup.Config{
		Cloud:     "testcloud",
		SpaceID:   "131352",
		SpaceKey:  "lybel",
		SpaceName: "Lybel",
	})

	md := writeMarkdownFile(t, dir, "# Hello\n")

	spaces := []adf.SpaceResult{
		{ID: "131352", Key: "lybel", Name: "Lybel"},
		{ID: "555000", Key: "eng", Name: "Engineering"},
	}
	idx := 0
	responses := []struct {
		code int
		body string
	}{
		{200, spaceListJSON(spaces)},
		{200, createPageResponseJSON("999", "X")},
	}
	mrt := &multiRoundTripper{responses: responses, idx: &idx}

	out, errOut, code := runPageCreateCmd(t, dir, mrt,
		"--space-id", "eng", "--parent-id", "212992001", "--title", "X", "--markdown", md)
	if code != exitOK {
		t.Fatalf("want exit 0, got %d\nstdout: %s\nstderr: %s", code, out, errOut)
	}
	if !strings.Contains(out, `"pageId": "999"`) {
		t.Errorf("expected pageId 999 in output, got: %s", out)
	}
}

func TestPageCreate_UnresolvableSpaceKeyErrorsCleanlyNoPanic(t *testing.T) {
	dir := t.TempDir()
	overrideConfigDirMain(t, dir)
	overrideCacheDirMain(t, dir)
	writeTestConfig(t, dir, setup.Config{
		Cloud:     "testcloud",
		SpaceID:   "131352",
		SpaceKey:  "lybel",
		SpaceName: "Lybel",
	})

	md := writeMarkdownFile(t, dir, "# Hello\n")

	spaces := []adf.SpaceResult{
		{ID: "131352", Key: "lybel", Name: "Lybel"},
	}
	rt := &mockRoundTripper{statusCode: 200, body: spaceListJSON(spaces)}

	out, errOut, code := runPageCreateCmd(t, dir, rt,
		"--space-id", "does-not-exist", "--parent-id", "212992001", "--title", "X", "--markdown", md)
	if code == exitOK {
		t.Fatalf("expected non-zero exit for unresolvable space key, got stdout: %s", out)
	}
	if !strings.Contains(errOut, "does-not-exist") {
		t.Errorf("expected the bad key echoed back in the error, got: %s", errOut)
	}
}

func TestPageCreate_APIErrorOnInvalidNumericSpaceIDNoPanic(t *testing.T) {
	dir := t.TempDir()
	overrideConfigDirMain(t, dir)
	overrideCacheDirMain(t, dir)
	writeTestConfig(t, dir, setup.Config{
		Cloud:     "testcloud",
		SpaceID:   "131352",
		SpaceKey:  "lybel",
		SpaceName: "Lybel",
	})

	md := writeMarkdownFile(t, dir, "# Hello\n")
	// A numeric space-id is passed straight to the create call; simulate
	// Confluence rejecting it (e.g. a non-existent space ID) — must surface
	// as a clean error, never dereference a nil result.
	rt := &mockRoundTripper{statusCode: 400, body: `{"message":"Space not found"}`}

	out, errOut, code := runPageCreateCmd(t, dir, rt,
		"--space-id", "999999999", "--parent-id", "212992001", "--title", "X", "--markdown", md)
	if code == exitOK {
		t.Fatalf("expected non-zero exit on API error, got stdout: %s", out)
	}
	if !strings.Contains(errOut, "Space not found") {
		t.Errorf("expected API error message surfaced, got: %s", errOut)
	}
}
