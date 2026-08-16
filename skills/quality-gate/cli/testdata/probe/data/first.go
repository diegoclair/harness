package data

import "strings"

func NormalizeFirst(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		lower := strings.ToLower(trimmed)
		if len(lower) > 64 {
			lower = lower[:64]
		}
		out = append(out, lower)
	}
	return out
}
