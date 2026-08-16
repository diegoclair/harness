# Rule catalog

Every rule has an ID, a severity, and a ruleset. `error` fails the run; `warn` is
reported and never fails. Both are subject to the baseline: a violation already
recorded in `.quality-gate-baseline.json` is silent until the line changes.

Suppress a single occurrence with a mandatory reason:

```go
// quality-gate:allow CMT-01 — the ADR this encodes has no other home
```

A suppression without a reason is itself an error (`GATE-01`).

---

## Philosophy the rules encode

1. **A comment explains purpose, never behavior.** Behavior changes and the
   comment rots into a lie; purpose survives the refactor. Every CMT rule is a
   consequence of this one sentence.
2. **A comment budget scales with the reader's distance from the code.** A
   package or interface is read by someone who has not seen the implementation —
   it can afford lines. A statement inside a function body has the code right
   there — it can afford almost none.
3. **On a declaration, describing is not commenting.** A field, a constant or an
   enum member is already named. The only comment it earns is a *constraint* the
   name cannot carry — a unit, an invariant, a format, an external contract.
   `// FirstName addresses the provider the way a client speaks about them` is a
   description of `FirstName`, and the field said it first.
4. **Complexity rules are advisory, never dogma.** A long function is not a
   defect. Two unrelated rules living in one function is. Cutting a function to
   satisfy a number produces worse code than leaving it alone, so every CPX rule
   is `warn` and every threshold is generous.
5. **Nothing here may push toward file fragmentation.** Hundreds of tiny files
   are not a quality signal — they are a navigation cost. See Non-goals.
6. **Duplication and misplaced domain rules are the real errors.** They are also
   the two things a human reviewer misses most reliably, because catching them
   requires holding the whole repo in your head.

---

## CMT — comments

| ID | Sev | Ruleset | Detects |
|---|---|---|---|
| CMT-01 | error | both | Comment block longer than the budget for its position |
| CMT-02 | error | both | Comment narrating the behavior of the code below it |
| CMT-03 | warn | both | A comment the delivery **adds** inside a function body |
| CMT-04 | error | both | Comment not written in English |
| CMT-05 | error | both | History or changelog inside a comment |
| CMT-06 | error | both | Commented-out code |
| CMT-07 | warn | both | Doc comment that only restates the symbol name |
| CMT-08 | warn | both | Delivery whose added comment ratio exceeds the budget |
| CMT-09 | warn | both | Comment on a declaration that describes instead of constraining |

### CMT-01 — block over budget

Budget by position, in lines (`comments.budget` in config):

| Position | Default | Why |
|---|---|---|
| Package / module doc | 15 | Read by someone who has not opened the package. |
| Interface | none | Prose is the product: the contract is read by someone who will never see an implementation. CMT-02 still applies — it may not describe behavior. |
| Type, struct, const block | none | Same reason. The docs that run long here are decisions with no other home (a spec reference, a CGNAT rationale), and length was catching those, never a wall of words. |
| Function / method doc | 6 | The signature already carries most of it. |
| Inside a function body | 3 | The code is one line away. |
| Declaration: struct field, const, enum member | 3 | Constraint only — see CMT-09. Two lines because a real constraint often wraps at 80 columns; the wrap is formatting, not a second thought. |
| Trailing (same line as code) | 1 | Structural. |

Position is resolved from the AST (Go) or the scanner's brace depth (web), not
from indentation. The span is the lines the block occupies **minus its own
delimiters**: a `/**` or `*/` sitting alone on a line is punctuation, and three
sentences cost the same written with `//` or with `/* */`. Without that,
counting the two JSDoc delimiter lines added 64 findings on `app/` that were
about a comment syntax rather than about a comment.

Web positions: `package` is a file-leading doc followed by a blank line or an
import; `type` is above an `interface`/`type`/`enum`/`class`; `func` is above a
function, a const arrow-function, a class method **or a function-typed member of
a type** — that last one is the web's interface method, and it earns the
function budget for the same reason Go's does; `decl` is a data member of a
type, an object or an enum; `body` is anything inside a function, JSX included.
A module-scope `const`/`let`/`var` is none of those and takes the orphan budget:
it is not a member, so CMT-09 never asks it for a constraint.

### CMT-02 — behavior narration

**Scope: body, trailing, declaration and orphan comments only.** A package, type
or function doc legitimately states what the thing promises — that is what a
contract is. Applying narration detection there floods the report; the narrower
CMT-07 covers the case where the promise is only the name spelled out.

A comment carrying a *why* marker (`because`, `otherwise`, `so that`, `must`,
`never`, an em dash) is purpose whatever verb it opened with, and never fires.
**On a declaration, CMT-09 is the whole family**: the comment carries a
constraint or it does not, and a second pass from the narration detector only
punished `// Current value, in cents.` for naming the field it constrains.

Two independent detectors, either one fires:

- **Overlap.** Strip stopwords from the comment; if ≥ 60% of what remains are
  identifiers or keywords appearing in the next 3 statements, the comment is
  restating the code. `// increment retryCount and return the error` above
  `retryCount++; return err` is the canonical hit.
- **Narration openers.** The comment opens with a step-narration pattern:
  `Now `, `Then `, `First `, `Next `, `Here we `, `This function `, `This method `,
  `Loop over `, `Iterate `, `Call the `, `Set the `, `Get the `, `Check if `,
  `Returns the ` / `Retorna `, when followed by a paraphrase of the code.

Rewrite as purpose: *why* this exists, *what constraint* it satisfies, *what
gotcha* in an external API forced it.

### CMT-03 — comment inside a function body

**Scoped to the delivery's own hunks.** Asked about every body comment a repo
already carries it was 43% of the whole baseline at 14% precision — the sampled
comments were overwhelmingly the kind this catalog says survives review. Asked
about the lines a change adds, it is the question the rule was written for.
`check --all` therefore does not report it at all.

Always a warning, never automatic. The question it asks is fixed: **is this
explaining what the code already says?** A comment inside a body survives review
when it carries a non-obvious constraint (a third-party quirk, an ordering
requirement, a reason for an unusual branch). It does not survive when it
labels a section — extract and name instead.

### CMT-04 — non-English comment

Fires on an accented letter, or on two Portuguese function words (`que`, `não`,
`para`, `porque`, `então`, `isso`, `aqui`, `deve`, `pois`) — one alone also
occurs in English. Quoted spans are stripped first: a comment *about* a
Portuguese word is still English. Applies to comments only; user-facing strings
stay in the product's language.

Two exemptions, both from the web calibration: a **single capitalised accented
word inside a sentence of five content words or more** is a proper noun ("slots
render in São Paulo time" is English), and a **fenced block or an `@example`
body is not prose at all** — it is a usage sample whose strings are product
copy, and judging it as a sentence reported half the `components/ui` docs.

### CMT-05 — history in a comment

Markers: `previously`, `used to be` / `we used to` / `it used to`,
`was renamed`, `refactored`, `changed from`, `moved from`, `deprecated in`,
`antes`, `agora`, plus a bare date. Git owns history; the code describes today.

Deliberately **not** markers: a bare `used to` and `no longer`. Both form
ordinary English about the present — "the CDN domain used to serve reads",
"can no longer change the status because the response started streaming" — and
a rule that fires on those teaches the reader to ignore it. Three of the three
sampled hits on the Lybel backend were exactly that.

### CMT-06 — commented-out code

A comment line that parses as code: contains `:=` or `=>`, or ends in `{` / `}`.
A trailing `;` counts only alongside the shape of a statement — ten words or
fewer, with a call or an assignment in it — because English prose uses a
semicolon too, and that alone was a false positive on this project's own source.

A line that **reads as a sentence** is prose whatever punctuation it ends on: an
em dash, or a period followed by a space. `* POST /provider/profile/instagram/ —
add instagram post { url }` ends in a brace and is documentation. Fenced blocks,
`@example` bodies and `@param`-style tags are skipped for the same reason.

### CMT-07 — restated doc

The doc's content words are a subset of the symbol name's words.
`// GetUser gets the user` on `func GetUser`. Idiomatic in Go, and still noise:
delete it or say why the function exists.

A **section label is not a doc**, and the whole description family (CMT-02,
CMT-07, CMT-09) skips it. `// ── Schema ──────` above `const serviceSchema` is
one content word that happens to be in the symbol's name; reading it as a
restatement produced 42 findings on `app/` and taught nothing.

### CMT-08 — delivery comment ratio

Diff-level, not per-line. Fires when the delivery adds comment lines beyond
`comments.diff_ratio` (default 15%) of added code lines, and names the top 5
files. This is the single number that catches a wall of comments no individual
rule flags. Repo reference at time of writing: backend sits at 10.3%.

### CMT-09 — declaration described instead of constrained

Applies to a struct field, a constant, an enum member or a config key. **Not to
an interface method** — a method is a contract and gets the function budget.

A short noun phrase that never makes a claim (`// Instagram Showcase.` above a
run of constants) is a section label, not a description, and is left alone. The comment survives only if it carries at least one **constraint
marker**:

| Marker | Example |
|---|---|
| Unit or scale | `cents, not reais`, `seconds`, `always UTC` |
| Invariant | `must`, `never`, `only`, `at most`, `nil means` |
| Format | `RFC3339`, `E.164`, `digits only`, `slug, lowercase` |
| External contract | `Asaas returns this unset for sandbox keys` |
| Contrast with a sibling | `the legal name, not the display one` |
| Reference | a spec, ADR or ticket the field encodes |

No marker means the comment is describing what the name already says, and it is
deleted. The example that motivated the rule:

```go
type ProviderCard struct {
	Name string
	// FirstName addresses the provider the way a client speaks about them.
	FirstName string        // CMT-09: description, no constraint
	URL       template.URL
}
```

`// the first name only, never the full one` would pass — it constrains. The
original does not, and the field name already carried everything it said.

Repo reference at time of writing: 176 field-level comments in the Go backend,
which is why this one is `error` rather than `warn` — at `warn` it would be
noise nobody clears.

---

## DUP — duplication

The index covers the **whole repo**, not the diff. The finding that matters is
"this new block already exists over there", not "the PR repeats itself".

| ID | Sev | Ruleset | Detects |
|---|---|---|---|
| DUP-01 | error | both | New block ≥ 80 normalized tokens identical to existing code |
| DUP-02 | error | both | Same block shape with renamed identifiers, ≥ 200 tokens (type-2 clone) |
| DUP-03 | error | web | JSX subtree ≥ 12 element nodes repeated across files |
| DUP-04 | warn | both | Clone whose every occurrence is in test files |

Normalization drops comments and whitespace, folds literals to a placeholder,
and (for DUP-02) folds identifiers to positional slots. Import blocks are cut
from the token stream — they are near-identical across files by nature, and
leaving them in makes every file a clone of its neighbours. Generated files
(`*.gen.ts`, `mocks/`, `docs/`) are excluded from both index and scan.

**A clone must contain logic, not just shape.** A window with no control-flow
token is a duplicated *declaration* — a route table, a struct literal — and
those repeat by nature. DUP-01 needs two branch tokens in the match and DUP-02 needs three. `return`,
`go` and `defer` are **not** counted: in Go every snippet has a `return`, which
left the floor inert until an audit measured it.
Without those two floors, folding identifiers makes every idiomatic
`type x struct{…}` plus its constructor a clone of every other: on the Lybel
backend that alone was 1063 findings, of which none was actionable.

Every finding reports the **original** location, so the fix is "import that" and
not "look for it". DUP-03 is the one that catches a React component rebuilt
under a new name; when it fires, the fix is a canonical component plus its row
in the project's canonical table.

**DUP-03 compares a subtree, never a run of siblings.** A window matches on tag
plus depth relative to the window's first node — copy and classes change when a
component is rebuilt, the tree does not — and the first node has to be the root
of everything that follows it. Without that second condition, twelve `<col>` and
`<th>` in a row made every list page a clone of every other: on `nexus` that was
6 findings of which 2 were real, and the subtree condition left exactly the 2.
Same-file repetition never counts: a row rendered twice is a `map` away.

---

## CPX — complexity and size

All `warn`. All thresholds deliberately above the industry default, because the
failure mode being avoided is a codebase cut into pieces to satisfy a linter.

| ID | Sev | Ruleset | Threshold |
|---|---|---|---|
| CPX-01 | warn | both | Cyclomatic complexity > 15 |
| CPX-02 | warn | both | Nesting depth > 4 |
| CPX-03 | warn | both | Function > 120 lines **and** cyclomatic ≥ 8, outside tests |
| CPX-04 | warn | both | Parameters > 6 (Go: `ctx` excluded; web: a props object counts as 1) |
| CPX-05 | warn | web | React component > 250 lines or > 10 hook calls |

A component, for CPX-05, is a function whose name starts with a capital and
whose body opened at least one element. Hooks are `useX(` calls counted into
every named function open at the time — `ProfilePage` in `app/` calls 50.

A CPX warning is an invitation to ask one question: **does this function hold two
different rules?** If yes, split by responsibility. If it is one long rule
expressed linearly, leave it alone and suppress with that reason.

---

## ARC — boundaries and misplaced domain rules

Layers are declared in config, so a project with another shape redeclares them.
The defaults below are Lybel's.

### Go

| ID | Sev | Rule | Status today |
|---|---|---|---|
| ARC-01 | error | `internal/*/domain/**` must not import `infra/data`, `bridges/**`, `internal/*/transport/**` | clean |
| ARC-02 | error | `internal/nexus/**` and `internal/provider/**` must not import each other | clean |
| ARC-03 | error | `internal/*/transport/**` must not import `internal/*/data/**` | clean |
| ARC-04 | error | `internal/*/data/**` must not import `internal/*/service` (`service/dto` allowed) | clean |
| ARC-07 | error | A layer imports a package it may not reach at all | clean |
| ARC-05 | warn | Domain rule inside the data layer | to baseline |
| ARC-06 | warn | Domain rule inside the transport layer | to baseline |

ARC-01..04 are verified at zero violations, so they lock the future with no
baseline entry.

**ARC-05 — domain rule in the data layer.** Not reported: `SUM(CASE WHEN …)` and
`COUNT(*) FILTER (…)`, which is how SQL pivots rows into columns; a `CASE WHEN $1`
over a bound parameter, where the service already decided; and an `INTERVAL`
inside `generate_series(…)`, which is a chart's calendar spine.

Markers, inside `data/**`: `CASE WHEN`
in SQL, arithmetic on a domain value, a business-state string literal compared or
assigned in SQL, a date-window computed in the query, or a Go conditional over
an entity field that is not query assembly. A repository answers *where the data
is*, never *what the business decides*.

**ARC-06 — domain rule in the transport layer.** Markers, inside `transport/**`:
a conditional over an entity field beyond input validation and error mapping,
arithmetic on domain values, a time comparison deciding an outcome. A handler
parses, delegates, and renders.

Presence and validity tests are not the business deciding anything: any
`Is`/`Has`/`Can`/`Should` predicate, a comparison against the zero value, `len()`
and the `err`/`nil`/`ok` idiom are all read as guards. A `**/viewmodel/**`
package is skipped outright, and so is any function named `parse*`, `to*`,
`from*`, `validate*`, `map*`, `build*`, `render*` or `new*` — parsing,
validating and rendering are what a handler is for.

Both are heuristics, hence `warn`, and both are prime input for the phase-2
judge, which can read intent instead of markers.

**ARC-07 — a forbidden import.** A denied edge is about the project's own
layers; this one names a package outright, so it reaches third-party packages
the module prefix never covers. It is how "the data layer never logs" is
expressed:

```yaml
forbid:
  data: ["diegoclair/logger"]
```

### The canonical table is not web-only

`canonical` rows are patterns over source lines, which makes them the place for
a project's written rules that no general detector can carry. They run on every
language. A row reads **code only** — comment spans are blanked first, because
every mention of a vendor name or a banned construct inside a comment is
legitimate, and matching those is pure noise.

| Field | Meaning |
|---|---|
| `forbid` | the pattern that must not appear |
| `match` | context that must be present on the line for the row to apply |
| `unless` | presence of this exempts the line |
| `only` / `except` | file globs the row does and does not police |
| `severity` | `warn` when the row codifies a judgment call |
| `scope: element` | match a JSX opening tag instead of a source line (web) |

Lybel's backend rows, all lifted straight out of `backend/CLAUDE.md`: a route
parameter at the end must carry a trailing slash (`error` — the shape that broke
`customerroute` in May 2026), the same convention on a fixed segment (`warn` —
consistent, but it cannot bite), a vendor name outside its bridge, and an error
invented on the API path without an `errcodes` code (`warn`).

### Web

| ID | Sev | Rule | Status today |
|---|---|---|---|
| ARC-10 | error | `features/A` must not import `features/B` — shared code goes to `components/` or `lib/` | 5 on `app`, baselined |
| ARC-11 | error | `components/**` and `lib/**` must not import `features/**` or `routes/**` | 6 on `app`, baselined |
| ARC-12 | error | No HTTP **call** (`fetch`, `ky`, `axios`) outside the layers that own the wire | 1 on `app` |
| ARC-13 | error | Canonical-table violations eslint cannot express (see below) | 32 / 19 / 0 |
| ARC-14 | warn | Business rule in a component: date math, price math, or a status decision computed inline | 1 / 3 / 0 |

**ARC-10 and ARC-11 are declared, not hard-coded.** ARC-11 is a set of deny
edges like the Go ones. ARC-10 is the isolation rule ARC-02 already implements,
pointed at `contexts: ["src/features/*"]` — a context pattern may carry a `*`,
which is how "every directory under `features/` is its own unit" is said without
naming them — and `context_rule: ARC-10` declares which ID a broken isolation
reports. The rule ID lives with the units for the same reason a deny edge
carries its own: the units belong to the project.

**ARC-12 is about a call, not an import.** `import { HTTPError } from "ky"` to
narrow an error is a type reaching for a name; the layers where HTTP is allowed
are `heuristics.http_layers`, so a repo that keeps its client in `lib/api` says
so instead of exempting all of `lib`.

**ARC-13** reads the canonical table from config, so the rows live in one place.
A row is `{id, forbid}` plus any of `match` (context that must be present),
`unless` (presence exempts), `except` (file globs — **always list the canonical
component's own file**, or `formatDurationMin` reports itself), and
`scope: element`, which matches against a whole opening tag instead of a source
line. Scope matters: a `className` three lines below its `<button` is invisible
to a line rule and obvious to an element one.

Lybel's non-lint-locked rows: raw hex in `className`/`style`, `Drawer` outside
the declared allowlist, a hand-built `` `${date}T${time}` `` string, inline
`` `${h}h${m}` `` duration math, and an interactive control at `h-9`/`h-10` with
no `h-12 sm:h-10` mobile size. The pickers already locked by
`eslint.config.js` are **not** duplicated here — one rule, one home.

**ARC-14 fires only inside a component's own body.** "Inline in a component" is
literal: a rule already extracted into a named helper is a rule with a home,
whatever folder that home sits in, and firing on `features/agenda/types.ts`'s
`isCancelable()` would punish the extraction the rule is asking for. Money and
date arithmetic are read off the code lines with the quoted spans blanked —
`</span>` on a line mentioning `priceDisplay` is a closing tag, not a division.
A status decision needs the literal comparison **and** a combinator **and**
either a second comparison or a date: comparing a status to pick a badge is
rendering, and treating it as a decision reported every `.filter()` in the repo.

---

## GATE — the gate about the gate

| ID | Sev | Detects |
|---|---|---|
| GATE-01 | error | `quality-gate:allow` with no reason after the em dash |
| GATE-02 | error | Baseline entry for a line that no longer exists (stale baseline) |

GATE-02 keeps the baseline shrinking: delete the code, lose the free pass.

---

## Non-goals

Rules that will **not** be added, recorded so nobody adds them later:

- Maximum lines per file, maximum functions per file, maximum file size. These
  produce fragmentation, and fragmentation is the cost this project refuses.
- Mandatory doc comment on every exported symbol. That manufactures exactly the
  noise CMT-07 deletes.
- Any style rule a formatter already owns (`gofmt`, Prettier).
- Rules requiring semantic judgment — "is this abstraction right", "does this
  name lie", "should these two components be one". Those belong to the phase-2
  judge, not to a linter pretending it can decide them.

## Backlog — not now

- **Tests shaping production** (project rule): required dependency accepting
  `nil`, `isTest` flags, symbol exported only for tests, `time.Sleep` used for
  synchronization. Detectable, valuable, out of v1 scope.
- **Logging convention**: `WithAttrs` on the context instead of a repeated attr.
- **Error taxonomy**: `apperr` code missing from both dictionaries.

---

## Calibration — Lybel backend, first run

The thresholds above are not guesses. They were set by running the gate over
418 Go files (63k lines) and reading a sample of every rule's output.

| Stage | Errors | Warnings |
|---|---|---|
| First run, defaults straight from the catalog | 1554 | 1158 |
| After the four false-positive families were fixed | 494 | 644 |

The four were: type-2 clones matching idiomatic declarations (1063 findings, all
noise); `used to` and `no longer` reading as history inside ordinary
present-tense English; a generated file (`goswag/`) counted as source; and
`IsZero`-style guards read as domain rules.

What the same run proved about the repo: `ARC-01` to `ARC-04` are at **zero**
violations, so the four import boundaries lock the future with no baseline
entry. That was verified twice — the first time the rules were mute, because a
layer pattern written for files did not match the package path an import
resolves to. A gate that passes for the wrong reason is worse than no gate, and
the fixture in `cli/testdata/probe` exists so that class of silence fails a test
instead of a repo.

The gate's own source passes its own rules with **no baseline**: 0 errors, 13
warnings, all of them CMT-03 questions about comments that carry constraints.

---

## Calibration — the web front-ends, first run

Same method: run `check --all`, sample every rule's output, open the source,
judge. 414 files across three repos with three different shapes — a Vite SPA, a
Next app and an admin panel.

| Stage | Errors | Warnings |
|---|---|---|
| First run, `app/` only, defaults straight from the catalog | 429 | 468 |
| After the eight false-positive families were fixed (`app/`) | 275 | 580 |
| `landingpage/` after the same fixes | 145 | 207 |
| `nexus/` after the same fixes | 38 | 82 |

Warnings went **up** on `app/` while errors went down, and that is the fix
working: 147 JSX comments had been read as trailing code comments, which both
overcharged them on CMT-01 and let CMT-02 fire on their own words; correcting
the position moved them to CMT-03, which is a question, not a failure.

The eight families, each found by reading the source behind a finding:

1. **A `{/* … */}` comment read as trailing code.** The `{` before it is JSX
   syntax, not code the comment sits after. It also poisoned `NextIdents`,
   which is read from the line *before* a trailing comment — the comment's own
   words — so CMT-02 saw 100% overlap. One fix, two rules. (63 → 5 CMT-02.)
2. **A function-typed member charged as a data member.** `onApplied: (msg) =>
   void` is the web's interface method, and Go already gives an interface method
   the function budget. (84 → 52 CMT-09.)
3. **A module-scope `const` charged as a member.** Two lines is the budget for a
   field; a module-scope binding is not a field.
4. **A box-drawing section divider read as a doc.** `// ── Schema ──` above
   `const serviceSchema` restates a name it never claimed to describe. (42 → 0
   CMT-07.)
5. **Documentation read as leftover code.** A fenced block, an `@example` body,
   a `@param {Type}` line, and any line that reads as a sentence. (6 → 0 CMT-06.)
6. **JSDoc delimiters charged to the budget.** `/**` and `*/` alone on a line
   are punctuation. (192 → 128 CMT-01 on `app`, 0 change on the Go backend.)
7. **A flat run of siblings read as a duplicated subtree.** Twelve `<col>` and
   `<th>` is what a table is. (6 → 2 DUP-03 on `nexus`, both real.)
8. **ARC noise from three directions at once**: `ky` imported for a type read as
   an HTTP call; `</span>` read as a division on a line naming a price; a status
   comparison read as a decision; and `catch { … }` classified as an object
   literal, which turned a comment inside it into a member description.

Precision on an 18-finding random sample of the errors, across all three repos:
**17 true positives by the rule as written, 1 soft miss** (`0–5 short options`
read as a description rather than a count bound — fixed by reading a numeric
range as a constraint). The residue the sampling did find and did not fix is two
CMT-02 overlap hits on comments that name the identifiers they explain — the
react-day-picker gotcha in `calendar.tsx` and an algebra note in
`DailyBarChart.tsx`. Both are the 60% overlap threshold doing what it was
calibrated to do; the baseline absorbs them.

`ARC-14` finds nothing on `app/` and three genuine hits on `landingpage/`. That
is the honest answer, not a mute rule: `app/` keeps its date arithmetic in
module-level helpers, and the probe fixture proves all three markers fire.

**The Go path is unchanged**, verified by diffing the F1 binary against this one
over the 418-file backend with the baseline off: 388/645 before, 387/645 after,
zero findings added. The one removed is a false positive of family 8 — a
`// Now is injected so the time rules are testable.` doc on a field named `Now`,
which fired CMT-02 for opening with its own field name after CMT-09 had already
accepted its constraint. The backend baseline therefore has one more stale entry
than it did, and `quality-gate baseline` is the user's call to make.

### What the web front-end cannot know

It is a tolerant scanner, not a type-aware AST, and these are the places it
guesses:

- **`<` in `.tsx`.** An element is separated from a comparison and a type
  argument by the previous token and the shape after the `<`. The generic arrow
  `<T,>(x: T) => x` is rejected by the comma; `<Select<Option> …>` — a component
  with an explicit type argument — is not recognised as an element.
- **Function detection** reads the tokens before a `{`. A body is found for
  `function f()`, `const f = () =>`, a class or object method, and a callback;
  an arrow with an expression body has no block and so no entry, which costs
  nothing to measure but means it is not counted as a component.
- **Parameters** count top-level commas, ignoring those inside `<…>` type
  arguments. A props object is one parameter, which is the point.
- **Nesting depth** counts brace-bodied control flow. Unlike the Go side, an
  `else if` chain stays at one level, which is the more honest reading.
- **`Cond`** is filled from lines carrying `if` or an equality comparison, not
  from an expression tree. `<` and `>` are excluded: in a `.tsx` file they are
  far more often a tag than a comparison.
- **ARC-13 and ARC-14 match on source lines** (with comments blanked and, for
  ARC-14, quoted spans blanked) or on an element's opening tag. A construct
  wrapped across lines in a way neither shape catches is missed. `scope:
  element` exists because that miss was common enough to matter.

---

## Second calibration — the precision audit

The first calibration counted findings. An audit then sampled them against the
source and asked how many a reviewer would act on, which is the only number that
matters. It was bad: several `error` rules sat below 30% precision, and roughly
300 of the 404 frozen errors were noise or non-actionable.

| Repo | Before the audit | After |
|---|---|---|
| `backend` | 494 err / 644 warn | **71 err / 66 warn** |
| `app` | 274 err / 580 warn | **140 err / 113 warn** |
| `landingpage` | 145 err / 207 warn | **79 err / 26 warn** |
| `nexus` | 38 err / 82 warn | **20 err / 32 warn** |

Baseline entries across the four repos: **2377 → 547**.

Each change was a defect, not a threshold nudge:

- **Three duplication bugs.** The same intra-file clone was reported from both
  ends (101 findings of pure repetition); the token count sat in the signature,
  so editing one line near any clone failed the gate with stale-baseline errors
  in untouched files; and `return` counted as control flow, which made the
  "a clone must contain logic" floor inert. An exact clone and a shape clone over
  the same lines are now one finding.
- **Markers matched substrings.** "re**moved from** the cache" was reported as
  history. Markers now match on word boundaries.
- **A quoted word is mentioned, not used.** CMT-04 already stripped quoted spans;
  CMT-05 did not, and flagged this project's own comment describing that bug.
- **CMT-05's bare-date sub-rule was 0% precision** — every hit was a test comment
  where the date *is* the fixture. Deleted.
- **Section labels.** Six content words instead of three, plus a divider rule and
  a "starts with the declaration's own first word" test, which is what separates
  a label from a Go-convention doc. CMT-07 consults it too.
- **CMT-09 dropped to `warn`.** After every marker fix it still sat near 40%.
  Describing versus constraining is semantic, and a keyword whitelist cannot
  reach error-grade precision on it; at `error` it was deleting comments that
  carried real information, the exact failure this catalog warns about. As a
  question it still catches the case that motivated the rule.
- **Budgets moved one line** (func 6, body 3, decl 3). Two thirds of CMT-01's
  findings were *exactly* one line over and every sampled one carried a real
  constraint. gofmt owns line breaks; one line over measures wrapping, not prose.
- **CPX-01 20 → 15.** At 20 it sat one notch above the worst function in the repo
  and reported nothing. CPX-03 now needs branchiness too: a 267-line route table
  holds one rule, not two.

---

## Budgets are per position, and the positions are measured

`interface`, `type`, `func`, `decl`, `body`, `trailing` and `package` each carry
their own budget, because they are read at different distances. The numbers are
not chosen by taste — this is the distribution across the Lybel repos when they
were set:

| Position | n | median | p90 | max | budget |
|---|---|---|---|---|---|
| interface | 56 | 3 | 5 | 6 | none |
| type / struct | 368 | 2 | 4 | 12 | none |
| func | 844 | 2 | 5 | 15 | 6 |

**A budget above the worst case in the repo is inert** — `type` sat at 10 with a
p90 of 4 and reported nothing, the same mistake CPX-01 made at 20. Dropping it
to 8 then caught exactly two comments, and both were decisions with nowhere else
to live: an unsubscribe token that must not expire because "your link expired"
produces a spam complaint, and a rate ceiling loosened for CGNAT. Length was
never the failure mode on a contract — so the cap came off, and CMT-02 carries
the rule that matters: **a contract may be as long as it needs, but it may not
describe behavior.**

On those positions CMT-02 runs its opener detector only. The overlap detector is
meaningless there: a doc for `AgendaSettingsRepo` naturally repeats the type's
own name and its members, and that is naming the contract, not narrating it.

A budget of `0` in config means no cap — that is how a position opts out of
CMT-01 without opting out of anything else.
