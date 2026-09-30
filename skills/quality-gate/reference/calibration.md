# Calibration record

The measurement runs behind the thresholds in [rules.md](rules.md): what each run
sampled, what it found, and which defect each change fixed. A run is a record —
it is not edited; a new calibration is a new section.

## Run 1 — Lybel backend

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

## Run 2 — the web front-ends

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

---

## Run 3 — the precision audit

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
  history. Markers match on word boundaries.
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
