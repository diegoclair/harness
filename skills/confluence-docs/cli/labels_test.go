package main

import (
	"strings"
	"testing"
)

const propsPage = `# Page title

:::properties
type: decision
status: active
owner: @someone
tags: checkout, PSP, recurring billing
:::

Body text.
`

// type/status are prefixed so a value can never collide with a free-form tag,
// and every tag becomes a label of its own.
func TestLabelsFromMarkdown(t *testing.T) {
	got := labelsFromMarkdown(propsPage)

	want := []string{"type-decision", "status-active", "checkout", "psp", "recurring-billing"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestLabelsFromMarkdownWithoutAPropertiesBlock(t *testing.T) {
	if got := labelsFromMarkdown("# Just a title\n\nSome text.\n"); got != nil {
		t.Errorf("got %v, want none: nothing declared them", got)
	}
}

// A collapsed block is still a properties block.
func TestLabelsFromMarkdownReadsACollapsedBlock(t *testing.T) {
	md := ":::properties collapsed\ntype: reference\n:::\n"
	if got := labelsFromMarkdown(md); len(got) != 1 || got[0] != "type-reference" {
		t.Errorf("got %v, want [type-reference]", got)
	}
}

func TestLabelsIgnoreUnknownKeys(t *testing.T) {
	md := ":::properties\nowner: @x\nrelated: 123, 456\ncreated: 2026-01-01\n:::\n"
	if got := labelsFromMarkdown(md); got != nil {
		t.Errorf("got %v, want none: only type, status and tags become labels", got)
	}
}

func TestLabelsAreDeduplicated(t *testing.T) {
	md := ":::properties\ntags: Checkout, checkout, CHECKOUT\n:::\n"
	if got := labelsFromMarkdown(md); len(got) != 1 {
		t.Errorf("got %v, want one label", got)
	}
}

func TestSanitizeLabel(t *testing.T) {
	cases := map[string]string{
		"Recurring Billing": "recurring-billing",
		"  spaced  ":        "spaced",
		"UPPER":             "upper",
		"a/b\\c":            "a-b-c",
		"emoji 🤖 tag":       "emoji-tag",
		"---":               "",
		"":                  "",
		"café":              "caf",
	}
	for in, want := range cases {
		if got := sanitizeLabel(in); got != want {
			t.Errorf("sanitizeLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

// A value that sanitises to nothing must not produce a dangling prefix.
func TestLabelsSkipAnUnusableValue(t *testing.T) {
	md := ":::properties\ntype: ---\nstatus: active\n:::\n"
	got := labelsFromMarkdown(md)
	for _, l := range got {
		if l == "type-" || l == "type" {
			t.Errorf("got %v, want no dangling prefix", got)
		}
	}
	if len(got) != 1 || got[0] != "status-active" {
		t.Errorf("got %v, want only [status-active]", got)
	}
}

// Only the first block is the page's own metadata; a later one is content.
func TestLabelsReadOnlyTheFirstBlock(t *testing.T) {
	md := ":::properties\ntype: reference\n:::\n\ntext\n\n:::properties\ntype: decision\n:::\n"
	got := labelsFromMarkdown(md)
	if len(got) != 1 || got[0] != "type-reference" {
		t.Errorf("got %v, want only the first block's type", got)
	}
}
