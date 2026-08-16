# quality-gate — design

Draft. The rule catalog lives in [reference/rules.md](reference/rules.md); this
file covers the engine, the CLI contract, and the delivery phases.

## What it is

One Go binary, installed by the harness installer like any other skill that
ships a `cli/`. It runs at the end of a delivery — next to `go build` and
`pnpm build` — and again on the PR, from the same binary.

It answers the questions a human reviewer currently answers by hand: is this
comment explaining behavior, does this block already exist somewhere in the
repo, is a domain rule leaking into a handler or a query.

## One binary, two rulesets

`ruleset` is declared per repo in `.quality-gate.yml`. A backend-only project
declares `go` and the web rules are never loaded, never reported, never a source
of noise. `auto` picks per file extension, which is what a mixed repo wants.

```yaml
# backend/.quality-gate.yml
ruleset: go
module: github.com/lybel-app/backend

layers:
  domain:    ["internal/*/domain/**"]
  service:   ["internal/*/service/**"]
  data:      ["internal/*/data/**", "infra/data/**"]
  transport: ["internal/*/transport/**"]
  bridges:   ["bridges/**"]

deny:
  - from: domain
    to: [data, bridges, transport]
    rule: ARC-01
  - from: transport
    to: [data]
    rule: ARC-03
  - from: data
    to: [service]
    rule: ARC-04

allow:
  data: ["internal/*/service/dto"]      # repositories return service DTOs

contexts: ["internal/provider", "internal/nexus"]   # mutually isolated

exclude: ["**/mocks/**", "docs/**", "*.gen.go"]
```

```yaml
# app/.quality-gate.yml — as shipped
ruleset: web

aliases:                       # what the bundler resolves, so a layer can be hit
  "@/": "src/"

layers:
  features:   ["src/features/*"]
  components: ["src/components/**"]
  lib:        ["src/lib/**"]
  apiclient:  ["src/lib/api/**"]   # longest pattern wins, so this beats lib
  routes:     ["src/routes/**"]
  services:   ["src/services/**"]

contexts: ["src/features/*"]   # a `*` means "each directory here is its own unit"
context_rule: ARC-10           # which rule a broken isolation reports

deny:
  - from: components
    to: [features, routes]
    rule: ARC-11

heuristics:
  http_layers: [services, apiclient]        # ARC-12 is quiet in these
  component_layers: [features, components, routes]   # where ARC-14 looks

canonical:
  - id: no-raw-hex
    match: 'className|style='
    forbid: '#[0-9a-fA-F]{6}\b'
    message: "Use design tokens (bg-primary, text-blocked-strong) — see CLAUDE.md."
  - id: drawer-is-app-chrome
    scope: element             # match the opening tag, not the source line
    element: '^Drawer'
    forbid: '^<Drawer'
    except: ["src/components/ui/Drawer.tsx", "src/components/layout/TopBar.tsx"]
    message: "Drawer is app chrome, not a dialog primitive. Use Modal."
  - id: mobile-touch-target
    scope: element
    element: '^(button|Button|Input|Select)$'
    match: 'className'
    forbid: '\bh-(9|10)\b'
    unless: '\bh-12\b|sm:h-'   # presence of this exempts the element
    message: "Mobile-reachable controls are h-12 sm:h-10 — 48dp minimum."

thresholds:
  comments.diff_ratio: 0.15

exclude: ["src/routeTree.gen.ts", "**/*.test.ts", "dist/**"]
```

Thresholds not declared fall back to the catalog defaults, so a config file is
short and only records the deviations.

## Engine

| Concern | Go | Web (ts/tsx) |
|---|---|---|
| Parsing | `go/ast` + `go/token` — real positions, real comment groups | tolerant scanner: strings, comments, braces, JSX tags, import statements |
| Comments | comment groups attached to declarations | scanner + brace depth for position |
| Complexity | branch nodes from the AST | branch tokens (`if`, `for`, `while`, `case`, `&&`, `\|\|`, `?`) with strings and comments removed |
| Imports | import specs | `import` / `require` / dynamic `import()` |
| Duplication | shared token normalizer over both languages | same, plus a markup-subtree index for DUP-03 |

No cgo, no tree-sitter, no Node dependency — the binary stays a single static
file the installer can ship. The cost is honest: the web side is a scanner, not
a type-aware AST, so a handful of web findings will be approximate. That is what
`quality-gate:allow` and the `warn` severity are for.

**Duplication index.** Rolling hash over normalized token windows, built once per
run over the whole repo. No cache: the full pass over 418 files is 0.7s, so a
cache would be complexity bought with nothing. Revisit if a repo makes it hurt.

## CLI

```
quality-gate check                 # changed files vs merge-base, dup against the whole repo
quality-gate check --all           # whole repo
quality-gate check --since <ref>   # explicit base
quality-gate baseline              # (re)generate .quality-gate-baseline.json
quality-gate explain CMT-02        # rule, rationale, how to fix
quality-gate init                  # scaffold .quality-gate.yml for a detected stack
```

Exit codes: `0` clean, `1` new errors, `2` config or invocation failure. Warnings
never change the exit code.

**Output is written for an agent to act on, not for a dashboard.** One line per
finding, then a one-line summary:

```
internal/provider/service/booking.go:142 CMT-02 error  comment narrates the code below — state the purpose or delete
internal/provider/data/booking.go:88     ARC-05 warn   CASE WHEN in SQL decides a business state — move to the service
app/src/features/agenda/SlotList.tsx:31  DUP-03 error  JSX subtree duplicated from components/ui/date-picker.tsx:64

3 findings: 2 errors, 1 warning (baseline: 41 known, 0 stale)
```

`--format json` for CI, `--quiet` to print only errors.

## Baseline

`quality-gate baseline` records today's debt as `{rule, file, signature}`, where
the signature is derived from the offending content, not from a line number.
Editing the code loses the pass; everything around it moving does not.

**Moving the file does lose the pass**, and that is the deliberate trade: a
path-free signature would let a violation copied into a new file inherit the
original's excuse, which is exactly the failure the gate exists to prevent.
Identical findings within one file are numbered, for the same reason — one entry
must never excuse two.

`GATE-02` reports entries whose code is gone, so the file can only shrink.

The baseline is committed. It is the honest record of what the repo owes.

## How it reaches the workflow

1. Each repo's `CLAUDE.md` gains one line in its commands section: run
   `quality-gate check` before delivering, next to the existing build gate.
   `dev-loop` already reads the project's `CLAUDE.md` for its static gates, so
   this is all the wiring the loop needs.
2. `SKILL.md` tells Claude what to do with the output: errors are fixed in the
   same delivery; warnings are reported to the user with the reasoning, never
   silently suppressed.
3. PR: a GitHub Action running the same binary, `--format json`, annotating the
   diff.

## Phases

| Phase | Delivers | Done when |
|---|---|---|
| F1 ✅ | Go engine: CMT, DUP, CPX, ARC-01..06 + baseline + CLI | **Done.** `check --all` on `backend/` is baseline-clean, the four ARC locks are verified at zero, and the gate passes its own rules with no baseline (0 errors) |
| F2 ✅ | Web ruleset: scanner, CMT/DUP/CPX for ts/tsx, ARC-10..14, DUP-03, CPX-05 | **Done.** `check --all` is baseline-clean on `app/`, `landingpage/` and `nexus/`; the Go path is byte-identical to F1; `cli/testdata/probe-web` asserts every web rule at an exact file:line |
| F3 | Harness packaging: `cli/`, release tag, install hooks, `CLAUDE.md` wiring in the four repos, GitHub Action | `harness install quality-gate` puts a working binary on PATH |
| F4 | Phase-2 judge: `quality-judge` agent consuming the linter output + the diff, for what a linter cannot decide | out of this scope, planned |

## Open calls

- **Where the spec lives.** The harness is a public repo; Lybel specs live in
  `docs/specs/`. This design doc is inside the skill for now.
- ~~Landing page and nexus in F2~~ — done in F2; each carries its own
  `canonical` block, which is what made the shared ruleset workable.

## F1 as built

418 files, 63k lines, in 0.7s. The calibration pass and what it cost is recorded
at the end of [reference/rules.md](reference/rules.md).

Two things the build changed from this design:

- **`prerequisites`** was added to the config. The project's own gates (`gofmt`,
  `go vet`, `staticcheck`) run before any rule and abort the run when red. The
  gate never reimplements a formatter, but a delivery deserves one command, and
  a repo with four unformatted files nobody catches has a gate missing.
- **Deny edges carry their own rule ID** in the config, as a list rather than a
  map. Layer names belong to the project, so the rule a violated edge reports
  has to be declared where the edge is, not inferred from a built-in table.

Open from the run, for the user to decide:

- **Dead exported symbols.** `staticcheck` catches an unused unexported const;
  nothing catches an exported one that no other package references. The
  whole-repo index the duplication engine already builds makes this nearly free
  — a `DEAD-01` warn. Not built: out of F1 scope.
- **Tests in scope for the comment rules.** 33% of the backend's findings are in
  `_test.go`. They are subject to the same rules today; `exclude` turns that off
  in one line if the noise is not worth it.

## F2 as built

414 ts/tsx files across three repos in 0.4s. The calibration pass, the eight
false-positive families it killed, and the list of things a scanner cannot know
are at the end of [reference/rules.md](reference/rules.md).

Four things the build added to this design:

- **`aliases`.** A web import resolves through the bundler, not through a module
  path. Without `@/` → `src/` every ARC rule on `app/` was mute — the same class
  of silence F1 hit, caught the same way, by a fixture.
- **`context_rule` and a `*` in `contexts`.** ARC-10 is ARC-02's rule pointed at
  feature folders; the ID lives with the units because the units are the
  project's, exactly as a deny edge carries its own ID.
- **`scope: element` on a canonical row.** A `className` three lines below its
  `<button` is invisible to a line rule. The IR gained `Element` (an opening tag
  with its attributes) and `JSXNode` (tag + depth, for DUP-03) so the rules
  could ask about markup without knowing what markup is.
- **`heuristics.http_layers` / `component_layers`.** ARC-12 and ARC-14 need to
  know which layers own the wire and which render, and neither is guessable.

The IR also gained `File.CodeLines` — the source with comments blanked — because
a rule matching on source must never fire on prose, and `Comment.Delims`, so a
JSDoc's `/**` and `*/` do not spend a comment's budget.
