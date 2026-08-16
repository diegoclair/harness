package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func detectRuleset(dir string) string {
	_, goErr := os.Stat(filepath.Join(dir, "go.mod"))
	_, webErr := os.Stat(filepath.Join(dir, "package.json"))
	switch {
	case goErr == nil && webErr == nil:
		return "auto"
	case goErr == nil:
		return "go"
	case webErr == nil:
		return "web"
	}
	return "auto"
}

func modulePath(dir string) string {
	f, err := os.Open(filepath.Join(dir, "go.mod"))
	if err != nil {
		return ""
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		if line := strings.TrimSpace(s.Text()); strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

func starterConfig(dir, ruleset string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "ruleset: %s\n", ruleset)
	if mod := modulePath(dir); mod != "" {
		fmt.Fprintf(&b, "module: %s\n", mod)
	}
	if ruleset != "go" {
		b.WriteString(`
# What the bundler resolves. Without this a web import reaches no layer and
# every ARC rule goes quiet.
aliases:
  "@/": "src/"
`)
	}
	b.WriteString(`
# Layers are yours to name. Every ARC rule reads them, and a layer you do not
# declare is a layer no rule can police.
layers: {}

# Each denied edge carries the rule it reports, because the layer names are
# yours and the rule ID has to be declared where the edge is.
#   deny:
#     - from: domain
#       to: [data, transport]
#       rule: ARC-01
deny: []

# Exceptions, matched against the imported path.
#   allow:
#     data: ["internal/*/service/dto"]
allow: {}

# Units that must not import each other, and the rule a broken isolation
# reports. A star in a pattern makes every directory under it its own unit.
#   contexts: ["src/features/*"]
#   context_rule: ARC-10
contexts: []

# Which declared layers own the wire (ARC-12) and which render (ARC-14).
#   heuristics:
#     http_layers: [services]
#     component_layers: [features, components, routes]

# Your canonical-components table — the rows a lint rule cannot express. Never
# repeat what eslint already locks: one rule, one home.
#   canonical:
#     - id: no-raw-hex
#       match: 'className|style='
#       forbid: '#[0-9a-fA-F]{6}\b'
#       message: "Use design tokens."
#     - id: drawer-is-app-chrome
#       scope: element          # match the opening tag, not the source line
#       element: '^Drawer$'
#       forbid: '^<Drawer'
#       except: ["src/components/ui/Drawer.tsx"]
#       message: "Drawer is app chrome — use Modal."
canonical: []

# The project's own gates. quality-gate never reimplements a formatter — it
# refuses to report on a tree the project's tooling already rejects.
#   prerequisites: ["gofmt -l .", "go vet ./...", "staticcheck ./..."]
prerequisites: []

exclude:
  - "**/mocks/**"
  - "**/*.gen.go"
  - "**/*.gen.ts"

# Thresholds not listed here fall back to the catalog defaults, so this stays
# a record of your deviations. See reference/rules.md.
thresholds: {}
`)
	return b.String()
}
