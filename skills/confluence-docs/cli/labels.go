package main

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/diegoclair/harness/pkg/atlassian/adf"
)

// propertiesBlockPattern captures the body of the first :::properties block.
var propertiesBlockPattern = regexp.MustCompile(`(?s):::properties[^\n]*\n(.*?)\n:::`)

// labelKeyPrefixes are the single-valued properties promoted to a prefixed
// label. The prefix keeps `type-decision` from colliding with a free-form tag
// and makes the CQL filter exact.
var labelKeyPrefixes = map[string]string{"type": "type", "status": "status"}

// labelsFromMarkdown turns a page's :::properties block into Confluence labels:
// `type`/`status` become `type-x`/`status-x`, and each entry in `tags` becomes
// a label of its own. Labels are indexed by Confluence, so metadata written
// this way is queryable in bulk instead of only readable page by page.
func labelsFromMarkdown(markdown string) []string {
	m := propertiesBlockPattern.FindStringSubmatch(markdown)
	if m == nil {
		return nil
	}

	var out []string
	seen := map[string]bool{}
	add := func(s string) {
		l := sanitizeLabel(s)
		if l == "" || seen[l] {
			return
		}
		seen[l] = true
		out = append(out, l)
	}

	for _, e := range adf.ParsePropertiesBlock(m[1]) {
		key := strings.ToLower(strings.TrimSpace(e.Key))
		switch {
		case labelKeyPrefixes[key] != "":
			if v := sanitizeLabel(e.Value); v != "" {
				add(labelKeyPrefixes[key] + "-" + v)
			}
		case key == "tags":
			for _, t := range strings.Split(e.Value, ",") {
				add(t)
			}
		}
	}
	return out
}

// sanitizeLabel makes a value usable as a Confluence label: lowercase, and
// anything that is not alphanumeric collapsed to a single dash.
func sanitizeLabel(s string) string {
	var sb strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			sb.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && sb.Len() > 0 {
				sb.WriteRune('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(sb.String(), "-")
}

// applyLabels attaches the labels a page's own properties declare. It never
// removes existing labels, and a failure is reported without failing the write
// that already succeeded.
func applyLabels(client *adf.ConfluenceClient, pageID, markdown string, enabled bool, out io.Writer) {
	if !enabled {
		return
	}
	labels := labelsFromMarkdown(markdown)
	if len(labels) == 0 {
		return
	}
	if err := client.AddLabels(pageID, labels); err != nil {
		fmt.Fprintf(out, "warning: page saved but labels not applied: %v\n", err)
		return
	}
	fmt.Fprintf(out, "labels: %s\n", strings.Join(labels, ", "))
}
