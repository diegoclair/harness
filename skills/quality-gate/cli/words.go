package main

import (
	"regexp"
	"strings"
	"unicode"
)

var identRe = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)

// splitIdent turns FirstName / first_name / firstName into {first, name}, so a
// comment's prose can be compared against the identifiers around it.
func splitIdent(s string) []string {
	var out []string
	for _, part := range strings.Split(s, "_") {
		var cur strings.Builder
		for i, r := range part {
			if unicode.IsUpper(r) && i > 0 && cur.Len() > 0 {
				out = append(out, strings.ToLower(cur.String()))
				cur.Reset()
			}
			cur.WriteRune(r)
		}
		if cur.Len() > 0 {
			out = append(out, strings.ToLower(cur.String()))
		}
	}
	return out
}

func identWords(text string) []string {
	var out []string
	for _, id := range identRe.FindAllString(text, -1) {
		out = append(out, splitIdent(id)...)
	}
	return out
}

var stopwords = map[string]bool{
	"a": true, "an": true, "and": true, "are": true, "as": true, "at": true,
	"be": true, "but": true, "by": true, "for": true, "from": true, "in": true,
	"is": true, "it": true, "its": true, "of": true, "on": true, "or": true,
	"that": true, "the": true, "this": true, "to": true, "we": true, "when": true,
	"which": true, "with": true, "so": true, "if": true, "into": true,
}

// contentWords are the words a comment actually asserts: no stopwords, no
// one-letter noise, stemmed enough that "gets" and "get" compare equal.
func contentWords(text string) []string {
	var out []string
	for _, w := range identWords(text) {
		if len(w) < 3 || stopwords[w] {
			continue
		}
		out = append(out, stem(w))
	}
	return out
}

func stem(w string) string {
	switch {
	case strings.HasSuffix(w, "ies") && len(w) > 4:
		return w[:len(w)-3] + "y"
	case strings.HasSuffix(w, "es") && len(w) > 4:
		return w[:len(w)-2]
	case strings.HasSuffix(w, "s") && !strings.HasSuffix(w, "ss") && len(w) > 3:
		return w[:len(w)-1]
	case strings.HasSuffix(w, "ing") && len(w) > 5:
		return w[:len(w)-3]
	case strings.HasSuffix(w, "ed") && len(w) > 4:
		return w[:len(w)-2]
	}
	return w
}

func toSet(words []string) map[string]bool {
	set := make(map[string]bool, len(words))
	for _, w := range words {
		set[stem(w)] = true
	}
	return set
}

func overlapRatio(words []string, against map[string]bool) float64 {
	if len(words) == 0 {
		return 0
	}
	hits := 0
	for _, w := range words {
		if against[w] {
			hits++
		}
	}
	return float64(hits) / float64(len(words))
}

// containsWord matches on word boundaries. Plain substring search reads
// "removed from" as the marker "moved from".
func containsWord(haystack string, needles []string) (string, bool) {
	for _, n := range needles {
		if hasPhrase(haystack, n) {
			return n, true
		}
	}
	return "", false
}

func hasPhrase(haystack, phrase string) bool {
	for i := 0; i <= len(haystack)-len(phrase); {
		j := strings.Index(haystack[i:], phrase)
		if j < 0 {
			return false
		}
		start := i + j
		if isWordBoundary(haystack, start-1) && isWordBoundary(haystack, start+len(phrase)) {
			return true
		}
		i = start + 1
	}
	return false
}

func isWordBoundary(s string, i int) bool {
	if i < 0 || i >= len(s) {
		return true
	}
	c := s[i]
	return !(c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9'))
}

func containsAny(haystack string, needles []string) (string, bool) {
	for _, n := range needles {
		if strings.Contains(haystack, n) {
			return n, true
		}
	}
	return "", false
}

func hasWord(haystack string, word string) bool {
	for _, w := range identWords(haystack) {
		if w == word {
			return true
		}
	}
	return false
}
