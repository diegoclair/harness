package service

import "strings"

func Label(name string, loud bool) string {
	if loud {
		return strings.ToUpper(name)
	}
	return trimmed(name)
}

func trimmed(name string) string {
	return strings.TrimSpace(name)
}

func Shout(name string) string {
	return upper(name) + upper(name)
}

func upper(name string) string {
	return strings.ToUpper(name)
}

func Handlers() []func(string) string {
	return []func(string) string{asValue}
}

func asValue(name string) string {
	return strings.ToLower(name)
}

func Composed(names []string) string {
	return joined(names)
}

func joined(names []string) string {
	return strings.Join(names[1:], strings.Repeat(" ", len(names)))
}

func Depth(tree map[string][]string, root string) int {
	var walk func(node string) int
	walk = func(node string) int {
		best := 0
		for _, child := range tree[node] {
			best = max(best, walk(child)+1)
		}
		return best
	}
	return walk(root)
}

func Later(tree map[string][]string) int {
	var pick func() int
	pick = func() int { return len(tree) }
	return pick()
}
