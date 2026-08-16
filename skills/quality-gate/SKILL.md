---
name: quality-gate
version: 0.1.0
description: >-
  Runs the review pass a linter can actually do, as the last step of a delivery — before you report the work as done, and again on the PR. A local Go binary that catches what a reviewer otherwise catches by hand: comments that narrate behavior instead of stating purpose, declarations described instead of constrained, blocks that already exist elsewhere in the repo (indexed whole-repo, not just the diff), functions holding two rules, and domain logic leaking into a handler or a SQL query. Baseline-frozen, so it blocks new violations without demanding the repo be clean first. Use this skill whenever you finish writing or refactoring code and are about to hand it over, when the user asks for a quality gate, a code-quality check, a duplication check, a comment/architecture review before commit or PR — and as the closing step of any multi-file delivery, even when the user does not name it. Replies match the user's language.
allowed-tools: |
  Bash(quality-gate *)
  Bash(make build)
  Bash(make test)
  Read
  Edit
  Grep
---

# quality-gate — the review pass a linter can actually do

(Instructions are in English so the model reasons robustly; reply to the user in their own language.)

## What this is for

A reviewer reading a delivery asks a handful of questions over and over: is this
comment explaining behavior that will change next month, does this block already
exist somewhere in the repo, is a business rule deciding things inside a query.
Those questions are mechanical. This binary answers them, so the human review is
spent on the ones that are not.

**Run it at the end of a delivery**, in the same breath as the project's build:

```bash
quality-gate check          # the delivery: diff vs base branch + uncommitted
quality-gate check --all    # the whole repo
quality-gate explain CMT-02 # why a rule exists and how to satisfy it
```

Exit 0 clean, 1 new errors, 2 configuration failure. **Warnings never fail the
run** — they are reported to the user, never silently suppressed.

## The three rules of using it

1. **Errors are fixed in the same delivery.** Not deferred, not baselined. The
   baseline is for debt that predates the gate, and `quality-gate baseline` is a
   deliberate act the user asks for — never something you run to make a red run
   go green.
2. **Warnings are reported with your reading of them.** A CPX warning is a
   question ("does this function hold two different rules?"), not an order. Say
   what you think and let the user decide. A CMT-03 on a comment carrying a real
   constraint is a comment that should stay.
3. **A suppression needs a reason, and the reason is the point.**
   `// quality-gate:allow CMT-01 — the ADR this encodes has no other home`. A
   reasonless directive is itself an error (GATE-01). Never suppress to get to
   green: if the rule is wrong for this project, the fix is the config, and
   that is the user's call.

## Reading the output

```
internal/provider/data/booking.go:88 ARC-05 warn  business-state literal compared in SQL — ...
app/src/features/agenda/SlotList.tsx:31 DUP-01 error lines 31-58 duplicate components/ui/date-picker.tsx:64-91 — ...

2 finding(s): 1 error(s), 1 warning(s) — 12 file(s) scanned, 418 indexed (baseline excused 1138)
```

A DUP finding always names the **original**, so the fix is "import that one", not
"go find it". If the original is worse than the copy, move the good version into
the shared place and point both at it — do not leave two.

## Per stack

**Backend (Go).** Fully implemented. Beyond the comment, duplication and
complexity rules, the ARC rules read the layers declared in
`.quality-gate.yml` — an import crossing a denied edge (domain reaching for an
adapter, transport reaching for a repository, one bounded context importing
another) is an error, and the two heuristics (ARC-05, ARC-06) flag a domain rule
that drifted into a query or a handler. When ARC-05 fires, the fix is almost
always to move the decision up into the service and let the query return rows.

**Frontend (ts/tsx).** Shipped. The comment, duplication and complexity rules
are language-neutral and now run on `.ts`/`.tsx`/`.js`/`.jsx` too, plus six
web-only rules: `ARC-10` (one feature importing another), `ARC-11` (shared code
importing a feature or a route), `ARC-12` (an HTTP call outside the layer that
owns the wire), `ARC-13` (the project's canonical-components table, declared in
`.quality-gate.yml`), `ARC-14` (a business rule computed inside a component,
`warn`) and `DUP-03` (a markup subtree rebuilt in another file), with `CPX-05`
for a component past its size or hook budget.

Two things to know before arguing with a web finding:

- **The front-end is a scanner, not a type-aware AST.** No cgo, no Node, one
  static binary — the cost is that a handful of findings are approximate, and
  exactly which ones is written down at the end of
  [reference/rules.md](reference/rules.md). Read that before calling one wrong.
- **`ARC-13` is the project's table, not the gate's.** A new canonical component
  ships with its row in the repo's `CLAUDE.md` *and* its row in
  `.quality-gate.yml`. What eslint already locks is never repeated there — one
  rule, one home.

**A backend-only project declares `ruleset: go`** and the web rules never load.
That is the point of one binary with two rulesets rather than two tools.

## The project's own written rules

A repo's `CLAUDE.md` usually carries rules no general linter knows: which layer
may import what, a route convention, a vendor name that must not leak. Those
belong in `.quality-gate.yml` — `forbid` for an import, a `canonical` row for a
pattern — not in a new detector. When the user points at a rule their docs
already state, check whether it fits one of those two before proposing code.

## Setting it up in a repo that has none

```bash
quality-gate init        # scaffolds .quality-gate.yml for the detected stack
# declare the layers and the denied edges — the ARC rules are mute without them
quality-gate baseline    # freeze today's debt; commit the file
```

**After upgrading the binary, run `quality-gate baseline --prune`, never a plain
`baseline`.** New rules retire old findings, and every retired one becomes a
stale-baseline error. `--prune` drops exactly those and freezes nothing new; a
full regenerate would silently re-freeze whatever debt arrived since.

`prerequisites` in the config is where the project's own gates go
(`gofmt -l .`, `go vet ./...`, `staticcheck ./...`, `tsc --noEmit`).
quality-gate runs them first and refuses to report on a tree the project's own
tooling already rejects — it never reimplements a formatter.

## What it deliberately does not do

Maximum lines per file, maximum functions per file, mandatory doc comments on
exported symbols: none of these will ever be rules here. They buy compliance by
fragmenting the codebase, and fragmentation costs more than it saves. The
reasoning, rule by rule, is in [reference/rules.md](reference/rules.md) — read it
before arguing with a finding, and before proposing a new rule.

Judgment calls — is this abstraction right, does this name lie, should these two
components really be one — are not linter work. They belong to the phase-2
judge, which does not exist yet.
