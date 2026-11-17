package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// Limits are in characters, not bytes: descriptions routinely carry
// multi-byte punctuation, and the documented limit counts runes.
const maxNameLen = 64
const maxDescriptionLen = 1024

// frontmatter is the subset every artifact must get right. Extra keys are
// allowed — Claude Code accepts more than the reference validator does.
type frontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type validationIssue struct {
	Path    string
	Problem string
}

type ruleFrontmatter struct {
	Paths []string `yaml:"paths"`
}

// validateTree walks the tree, not the catalog: it judges what is on disk,
// including files the catalog does not know yet.
func validateTree(root string, out io.Writer) (int, []validationIssue) {
	var issues []validationIssue
	checked := 0

	for _, f := range discoverArtifactFiles(root) {
		checked++
		if f.kind == KindRule {
			issues = append(issues, validateRuleFile(f.path)...)
			continue
		}
		issues = append(issues, validateFile(f.path, f.name, f.kind)...)
	}
	fmt.Fprintf(out, "Validated %d artifact(s) in %s\n", checked, root)
	return checked, issues
}

type discovered struct {
	path string
	name string
	kind Kind
}

// discoverArtifactFiles lists the artifacts physically present in a tree.
func discoverArtifactFiles(root string) []discovered {
	var found []discovered

	if entries, err := os.ReadDir(filepath.Join(root, "skills")); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			p := filepath.Join(root, "skills", e.Name(), "SKILL.md")
			if _, err := os.Stat(p); err == nil {
				found = append(found, discovered{p, e.Name(), KindSkill})
			}
		}
	}
	found = append(found, discoverMarkdownFiles(root, KindAgent)...)
	found = append(found, discoverMarkdownFiles(root, KindRule)...)
	sort.Slice(found, func(i, j int) bool { return found[i].path < found[j].path })
	return found
}

func discoverMarkdownFiles(root string, kind Kind) []discovered {
	entries, err := os.ReadDir(filepath.Join(root, kind.dir()))
	if err != nil {
		return nil
	}
	var found []discovered
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		p := filepath.Join(root, kind.dir(), e.Name())
		found = append(found, discovered{p, strings.TrimSuffix(e.Name(), ".md"), kind})
	}
	return found
}

func validateRuleFile(path string) []validationIssue {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []validationIssue{{path, fmt.Sprintf("cannot read: %v", err)}}
	}
	if !strings.HasPrefix(strings.TrimPrefix(string(raw), "\ufeff"), "---\n") {
		return nil
	}
	block, ok := extractFrontmatter(string(raw))
	if !ok {
		return []validationIssue{{path, "frontmatter opened with --- but never closed"}}
	}
	dec := yaml.NewDecoder(strings.NewReader(block))
	// A misspelt key would leave the rule loading for every file instead of failing here.
	dec.KnownFields(true)
	var fm ruleFrontmatter
	if err := dec.Decode(&fm); err != nil && err != io.EOF {
		return []validationIssue{{path, fmt.Sprintf("invalid rule frontmatter (only a 'paths' list is read): %v", err)}}
	}
	return nil
}

func validateFile(path, expectedName string, kind Kind) []validationIssue {
	var issues []validationIssue
	add := func(format string, args ...any) {
		issues = append(issues, validationIssue{path, fmt.Sprintf(format, args...)})
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		add("cannot read: %v", err)
		return issues
	}

	block, ok := extractFrontmatter(string(raw))
	if !ok {
		add("no YAML frontmatter delimited by --- at the top of the file")
		return issues
	}

	var fm frontmatter
	// Strict YAML on purpose: an unquoted description containing ": " parses
	// as a nested mapping and breaks every consumer that is not Claude Code.
	if err := yaml.Unmarshal([]byte(block), &fm); err != nil {
		add("invalid YAML frontmatter: %v", err)
		return issues
	}

	switch {
	case fm.Name == "":
		add("missing 'name'")
	case utf8.RuneCountInString(fm.Name) > maxNameLen:
		add("name is %d chars, max %d", utf8.RuneCountInString(fm.Name), maxNameLen)
	case fm.Name != strings.ToLower(fm.Name) || strings.ContainsAny(fm.Name, " _"):
		add("name %q must be lowercase with hyphens", fm.Name)
	case fm.Name != expectedName:
		add("name %q does not match its location (expected %q)", fm.Name, expectedName)
	}

	switch {
	case fm.Description == "":
		add("missing 'description'")
	case utf8.RuneCountInString(fm.Description) > maxDescriptionLen:
		add("description is %d chars, max %d", utf8.RuneCountInString(fm.Description), maxDescriptionLen)
	case strings.ContainsAny(fm.Description, "<>"):
		add("description contains < or >, which the reference skill validator rejects")
	}

	return issues
}

// extractFrontmatter returns the text between the leading --- fences.
func extractFrontmatter(s string) (string, bool) {
	s = strings.TrimPrefix(s, "\ufeff")
	if !strings.HasPrefix(s, "---\n") {
		return "", false
	}
	rest := s[len("---\n"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", false
	}
	return rest[:end], true
}

func runValidate(args []string, stdout, stderr io.Writer) int {
	root := "."
	if len(args) > 0 {
		root = args[0]
	}
	checked, issues := validateTree(root, stdout)
	if checked == 0 {
		fmt.Fprintf(stderr, "no artifacts found under %s (expected skills/, agents/ or rules/)\n", root)
		return exitInputErr
	}
	if len(issues) == 0 {
		fmt.Fprintln(stdout, "All artifacts valid.")
		return exitOK
	}
	for _, i := range issues {
		fmt.Fprintf(stderr, "%s: %s\n", i.Path, i.Problem)
	}
	fmt.Fprintf(stderr, "\n%d problem(s) found\n", len(issues))
	return exitErr
}
