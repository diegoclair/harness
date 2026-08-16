package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const configName = ".quality-gate.yml"

// DenyEdge carries its own rule ID: layer names are the project's, so the rule
// a violated edge reports has to be declared where the edge is.
type DenyEdge struct {
	From string   `yaml:"from"`
	To   []string `yaml:"to"`
	Rule string   `yaml:"rule"`
}

// CanonicalRow is one line of the project's canonical-components table, the
// ones a lint rule cannot express. Rows live in the config because the table
// belongs to the project, not to the gate.
type CanonicalRow struct {
	ID      string   `yaml:"id"`
	Scope   string   `yaml:"scope"`   // line (default) | element
	Element string   `yaml:"element"` // element-name pattern, scope: element
	Match   string   `yaml:"match"`   // context that must be present
	Forbid  string   `yaml:"forbid"`  // what must not appear
	Unless  string   `yaml:"unless"`  // presence of this exempts
	Only    []string `yaml:"only"`    // file globs the row polices, all when empty
	Except  []string `yaml:"except"`  // file globs the row does not police
	Message string   `yaml:"message"`
	Sev     string   `yaml:"severity"` // warn to make the row a question

	elementRe, matchRe, forbidRe, unlessRe *regexp.Regexp
}

type Config struct {
	Ruleset string `yaml:"ruleset"` // go | web | auto
	Module  string `yaml:"module"`  // Go module path, to resolve imports to files
	BaseRef string `yaml:"base_ref"`

	Layers   map[string][]string `yaml:"layers"`
	Deny     []DenyEdge          `yaml:"deny"`
	Allow    map[string][]string `yaml:"allow"`
	Forbid   map[string][]string `yaml:"forbid"`
	Contexts []string            `yaml:"contexts"`

	// Isolated units are the project's, so the rule a broken isolation reports
	// is declared with them — the same reason a deny edge carries its own ID.
	ContextRule string `yaml:"context_rule"`

	// Import prefixes a bundler resolves, so `@/lib/x` reaches a layer pattern.
	Aliases map[string]string `yaml:"aliases"`

	Canonical []CanonicalRow `yaml:"canonical"`

	// Must name declared layers; the leak rules stay quiet otherwise.
	Heuristics struct {
		SQLLayer        string   `yaml:"sql_layer"`
		HandlerLayer    string   `yaml:"handler_layer"`
		HTTPLayers      []string `yaml:"http_layers"`
		ComponentLayers []string `yaml:"component_layers"`
	} `yaml:"heuristics"`

	Exclude    []string           `yaml:"exclude"`
	Severity   map[string]string  `yaml:"severity"`
	Thresholds map[string]float64 `yaml:"thresholds"`

	// Must pass before any rule runs: gofmt, vet, tsc and the like.
	Prerequisites []string `yaml:"prerequisites"`

	Root string `yaml:"-"`
}

var defaultThresholds = map[string]float64{
	"comments.budget.package":   15,
	"comments.budget.type":      0, // no cap; CMT-02 openers still apply
	"comments.budget.interface": 0, // no cap; CMT-02 still applies
	"comments.budget.func":      6,
	"comments.budget.body":      3,
	"comments.budget.decl":      3,
	"comments.budget.trailing":  1,
	"comments.budget.orphan":    5,
	// The budget is the target; the tolerance is what keeps the rule off a wrap.
	// Measured across the four repos: 72 of 120 findings were one or two lines
	// over and carried real content, while everything at three or more had fat
	// to cut — including a 36-line design doc parked in a source file.
	"comments.budget_tolerance": 2,
	"comments.diff_ratio":       0.15,
	"comments.overlap_ratio":    0.6,
	"dup.min_tokens":            80,
	"dup.min_tokens_shape":      200,
	"cpx.cyclomatic":            15,
	"cpx.lines_cyclomatic":      8,
	"cpx.depth":                 4,
	"cpx.lines":                 120,
	"cpx.params":                6,
	"cpx.component_lines":       250,
	"cpx.component_hooks":       10,
	"dup.min_jsx_nodes":         12,
}

func (c *Config) threshold(key string) float64 {
	if v, ok := c.Thresholds[key]; ok {
		return v
	}
	v, ok := defaultThresholds[key]
	if !ok {
		panic("unknown threshold " + key) // a typo in a rule, not in user config
	}
	return v
}

// A budget of zero is no budget at all, which is how a position opts out of
// CMT-01 without opting out of the rest.
func (c *Config) budget(pos CommentPos) int {
	return int(c.threshold("comments.budget." + string(pos)))
}

// loadConfig walks up from dir looking for the config file, and treats the
// directory holding it as the repo root — every path in a report is relative
// to that, so findings are stable wherever the command was invoked from.
func loadConfig(dir string) (*Config, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	for d := abs; ; {
		path := filepath.Join(d, configName)
		if _, err := os.Stat(path); err == nil {
			return parseConfig(path, d)
		}
		parent := filepath.Dir(d)
		if parent == d {
			break
		}
		d = parent
	}
	return nil, fmt.Errorf("no %s found in %s or any parent — run `quality-gate init`", configName, abs)
}

func parseConfig(path, root string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &Config{Root: root}
	if err := yaml.Unmarshal(raw, cfg); err != nil {
		return nil, fmt.Errorf("%s: %w", configName, err)
	}
	if cfg.Ruleset == "" {
		cfg.Ruleset = "auto"
	}
	switch cfg.Ruleset {
	case "go", "web", "auto":
	default:
		return nil, fmt.Errorf("%s: ruleset must be go, web or auto (got %q)", configName, cfg.Ruleset)
	}
	for id, sev := range cfg.Severity {
		if _, ok := catalog[id]; !ok {
			return nil, fmt.Errorf("%s: severity override for unknown rule %q", configName, id)
		}
		if sev != string(SevError) && sev != string(SevWarn) {
			return nil, fmt.Errorf("%s: severity for %s must be error or warn (got %q)", configName, id, sev)
		}
	}
	for key := range cfg.Thresholds {
		if _, ok := defaultThresholds[key]; !ok {
			return nil, fmt.Errorf("%s: unknown threshold %q", configName, key)
		}
	}
	for _, row := range cfg.Canonical {
		if row.Sev != "" && row.Sev != string(SevWarn) && row.Sev != string(SevError) {
			return nil, fmt.Errorf("%s: canonical row %q severity must be warn or error", configName, row.ID)
		}
	}
	for _, edge := range cfg.Deny {
		if _, ok := catalog[edge.Rule]; !ok {
			return nil, fmt.Errorf("%s: deny edge from %q names unknown rule %q", configName, edge.From, edge.Rule)
		}
		if _, ok := cfg.Layers[edge.From]; !ok {
			return nil, fmt.Errorf("%s: deny edge names undeclared layer %q", configName, edge.From)
		}
	}
	if cfg.Heuristics.SQLLayer == "" {
		cfg.Heuristics.SQLLayer = "data"
	}
	if cfg.Heuristics.HandlerLayer == "" {
		cfg.Heuristics.HandlerLayer = "transport"
	}
	if cfg.ContextRule == "" {
		cfg.ContextRule = "ARC-02"
	}
	if _, ok := catalog[cfg.ContextRule]; !ok {
		return nil, fmt.Errorf("%s: context_rule names unknown rule %q", configName, cfg.ContextRule)
	}
	if err := cfg.compileCanonical(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) compileCanonical() error {
	seen := map[string]bool{}
	for i := range c.Canonical {
		row := &c.Canonical[i]
		if row.ID == "" {
			return fmt.Errorf("%s: every canonical row needs an id", configName)
		}
		if seen[row.ID] {
			return fmt.Errorf("%s: duplicate canonical row %q", configName, row.ID)
		}
		seen[row.ID] = true
		if row.Forbid == "" {
			return fmt.Errorf("%s: canonical row %q has no forbid pattern", configName, row.ID)
		}
		switch row.Scope {
		case "", "line", "element":
		default:
			return fmt.Errorf("%s: canonical row %q: scope must be line or element (got %q)", configName, row.ID, row.Scope)
		}
		for _, p := range []struct {
			src string
			dst **regexp.Regexp
		}{{row.Element, &row.elementRe}, {row.Match, &row.matchRe}, {row.Forbid, &row.forbidRe}, {row.Unless, &row.unlessRe}} {
			if p.src == "" {
				continue
			}
			re, err := regexp.Compile(p.src)
			if err != nil {
				return fmt.Errorf("%s: canonical row %q: %w", configName, row.ID, err)
			}
			*p.dst = re
		}
	}
	return nil
}

func (c *Config) wants(lang Lang) bool {
	return c.Ruleset == "auto" || c.Ruleset == string(lang)
}

// runPrerequisites reports the project's own gates that are failing. A failing
// prerequisite is not a finding: it aborts, because rules read positions the
// project's formatter is about to move anyway.
func (c *Config) runPrerequisites() error {
	for _, cmdline := range c.Prerequisites {
		cmd := exec.Command("sh", "-c", cmdline)
		cmd.Dir = c.Root
		out, err := cmd.CombinedOutput()
		trimmed := strings.TrimSpace(string(out))
		// gofmt -l reports by printing paths, not by failing.
		if err != nil || trimmed != "" {
			msg := fmt.Sprintf("prerequisite failed: %s", cmdline)
			if trimmed != "" {
				msg += "\n" + trimmed
			}
			return fmt.Errorf("%s", msg)
		}
	}
	return nil
}

// globMatch supports ** (any depth), * (one segment) and plain prefixes, which
// is everything the layer patterns need and nothing more.
func globMatch(pattern, path string) bool {
	re, ok := globCache[pattern]
	if !ok {
		re = regexp.MustCompile(globToRegexp(pattern))
		globCache[pattern] = re
	}
	return re.MatchString(path)
}

var globCache = map[string]*regexp.Regexp{}

func globToRegexp(pattern string) string {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		switch {
		case strings.HasPrefix(pattern[i:], "/**"):
			// `data/**` must match the package path `data` too: an import
			// resolves to a directory, and every ARC rule would go mute.
			b.WriteString("(?:/.*)?")
			i += 2
		case strings.HasPrefix(pattern[i:], "**/"):
			b.WriteString("(?:.*/)?")
			i += 2
		case strings.HasPrefix(pattern[i:], "**"):
			b.WriteString(".*")
			i++
		case pattern[i] == '*':
			b.WriteString("[^/]*")
		case pattern[i] == '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(string(pattern[i])))
		}
	}
	// A layer pattern names a directory; everything under it belongs to it.
	b.WriteString("(?:/.*)?$")
	return b.String()
}

func (c *Config) excluded(path string) bool {
	for _, p := range c.Exclude {
		if globMatch(p, path) {
			return true
		}
	}
	return false
}

// layerOf returns the declared layer a repo-relative path belongs to.
func (c *Config) layerOf(path string) string {
	best := ""
	bestLen := -1
	for name, patterns := range c.Layers {
		for _, p := range patterns {
			if globMatch(p, path) && len(p) > bestLen {
				best, bestLen = name, len(p)
			}
		}
	}
	return best
}

// contextOf returns the isolated unit a path belongs to. A pattern may carry a
// `*`, which is how "every directory under features/ is its own unit" is
// declared without naming them one by one.
func (c *Config) contextOf(path string) string {
	for _, ctx := range c.Contexts {
		if !strings.Contains(ctx, "*") {
			if path == ctx || strings.HasPrefix(path, ctx+"/") {
				return ctx
			}
			continue
		}
		if m := contextRe(ctx).FindStringSubmatch(path); m != nil {
			return m[1]
		}
	}
	return ""
}

func contextRe(pattern string) *regexp.Regexp {
	if re, ok := contextCache[pattern]; ok {
		return re
	}
	body := globToRegexp(pattern)
	body = strings.TrimSuffix(strings.TrimPrefix(body, "^"), "(?:/.*)?$")
	re := regexp.MustCompile("^(" + body + ")(?:/.*)?$")
	contextCache[pattern] = re
	return re
}

var contextCache = map[string]*regexp.Regexp{}

func (c *Config) hasLayer(list []string, layer string) bool {
	return layer != "" && contains(list, layer)
}
