---
name: dev-loop
version: 0.7.0
description: >-
  IMPLEMENTS (builds/refactors) a NON-trivial code feature with a built-in quality gate — the loop implementer → `architecture-reviewer` on every closed code path → `unbiased-reviewer` (mutation testing) where an error costs a lot → the parent decides → [correction → re-review], with the gate fired once per code path rather than per micro-deliverable. Use WHENEVER the task is to WRITE non-trivial code: a change spanning multiple files, new logic with state or concurrency, a feature with regression risk, or when the user wants the implementation done with rigor / doesn't trust "it compiled" alone — EVEN if they don't say "loop", "review" or "gate". Also when you (Claude) are about to implement a large feature yourself and a second unbiased pair of eyes is worth it. Do NOT use for: one-liners, renames, copy tweaks, 1:1 mechanical edits; and NOT for reviewing existing code, a PR, or a diff (that is the `unbiased-reviewer` agent on its own — this skill BUILDS, the review is only its inner gate).
allowed-tools:
  - Read
  - Grep
  - Glob
  - Bash
  - Edit
  - Write
  - Agent
  - Task
---

# dev-loop — ship a large feature with adversarial review

(Instructions are in English so the model reasons robustly; respond to the user in their own language.)

## When to use (and when NOT)

Use it for a **non-trivial** feature: a refactor with regression risk, new logic with state/concurrency, a change touching several files, anything where "it compiled" is no guarantee of "it's correct".

**Do NOT use** it for a one-liner, rename, copy tweak, or 1:1 mechanical edit — that goes straight in; the loop's ceremony doesn't pay off. And **do NOT use it to review existing code / a PR / a diff** — this skill BUILDS a feature (the review is only its inner gate); to review something already written, dispatch the `unbiased-reviewer` agent on its own. If you're unsure whether it's trivial: it's trivial if a `build`+diff-check settles it; it's not if you'd want a second pair of eyes.

## The parent's role: orchestrate, don't code

Whoever invokes the skill is the **orchestrator**. They do NOT edit production — they dispatch subagents and decide. This keeps the parent's context clean and enforces "unbiased" (whoever reviews is never whoever wrote it).

## The review unit is the code path, not the deliverable

Split the feature into deliverables for IMPLEMENTATION, in dependency order — but the review fires **once per code path** (the deliverables that touch the same package and the same files), when that path is closed. **A review does not scale down:** it costs about the same on 2 lines as on 200, and five deliverables over the same files reviewed one by one are five full reviews of the same code. The floor is what pays for the gate, not the ceiling.

- **Same package, same files → one review scope.** Open a second scope only when the path is genuinely another one: another package, another wire contract, another consumer.
- **A shared package is serial anyway.** An implementer editing while a reviewer runs `build` or a mutant gives a false verdict, so deliverables on one package can never overlap — grouping them costs nothing in risk and saves an entire review.
- **The round cap counts per path**, not per deliverable.

## Which review a closed path gets

- **`architecture-reviewer` on every closed code path.** It reads and greps, runs nothing, and is cheap by construction — no path skips it.
- **`blind-spot-reviewer` beside it, on a path that added or changed error handling.** Also read-only: it follows each error exit of the diff up to the layer that logs and says whether a failure there can be diagnosed — logged once, with the identifiers that find the case, the cause kept. Pass it the path's entry points, the files, and the doc and section where the project writes its logging rules, so it walks the exits instead of reading the path.
- **`unbiased-reviewer` (mutation, adversarial fixtures, real infrastructure) only where an error costs a lot:** money, writes to a marketplace or any external platform, data transactions, concurrency. The mini-spec names which paths carry one of these; the rest close on the architecture pass plus the static gates.

## The loop, per code path

1. **Mini-spec, if there isn't one.** If the feature arrived without a spec, write a short mini-spec BEFORE coding: gap → what to deliver → **verifiable exit criterion** (a command/assertion that proves done) → what's out of scope → whether the path carries money, external writes, data transactions or concurrency. Save it to disk (the subagent rereads it from there — survives context compaction).

2. **Dispatch the IMPLEMENTER**, one per context, in dependency order, on the model the work calls for (Parameters). It reads the mini-spec from disk, codes ONLY its deliverable, and leaves the static gates green before returning. It returns a short summary of what changed + the files touched.

3. **Between deliverables of the same path, the static gates are the proof.** When you want a second pair of eyes before the path grows on a wrong foundation, dispatch a **READ REVIEW** — the `unbiased-reviewer` with `Mode: READ REVIEW`, which reads the spec and the diff and **runs nothing**: no mutant, no fixture, no suite. It returns findings as advice, with no verdict; you or the next implementer decide what to take. Cheap by construction — it catches a wrong direction early, it never replaces the gate.

4. **When the path is closed, dispatch the structural pass, then — where an error costs a lot — the correctness gate.** First the **`architecture-reviewer`** (fresh, read-only, opus): `Mode: PATH`, the mini-spec path with its owner table, every file touched in the path. It returns APPROVE/REJECT per invariant — one owner per question, forwarders and flags and threaded parameters, files holding several scopes, facts after effects, swallowed errors. A REJECT goes to correction (step 5) before anything else runs, because a correctness gate spent on a shape about to change is spent twice. Then, on a path that carries money, external writes, data transactions or concurrency, the **`unbiased-reviewer`** agent (fresh, did NOT see any implementer's conversation), over ALL the deliverables of the path at once, handed the architecture report. Pass it: `Mode: FIRST REVIEW`, the mini-spec path, every file touched in the path, the exit criteria. It returns VERDICT (APPROVE/REJECT) + mutation (killed/survivors) + anchored findings. **A green build is not enough — the reviewer proves the tests aren't hollow and hunts bugs/regressions.** This is the one full pass the path gets; everything after it is scoped.

5. **DECIDE, following the spec** (you are the arbiter — accept neither the implementer nor the reviewer blindly):
   - APPROVE with no relevant finding → confirm by running the static gates yourself once → **next path**.
   - Trivial finding you agree with → fix it pointwise and re-run the gate. Only for a minimal, obvious correction. No re-review for this.
   - Relevant finding (BLOCKER/HIGH) or you disagree with the reviewer → send the findings to the **path's implementer** by `SendMessage` — it already holds the recon: "apply these fixes per the spec, don't touch anything else". Then dispatch a **new reviewer of the same kind in RE-REVIEW mode** (below).
   - Finding that contradicts the spec → **the spec wins**; reject the finding and record why. A reviewer raises legitimate things that are out of scope, and the parent refuses them without spending a re-review.

6. Only with the path **approved + green**, move to the next.

## Re-review is scoped, capped, and its goalposts are frozen

A correction round changes a few files, yet the re-review is the round that runs away: capping what it RE-READS is not enough, because the turns are spent on what it RE-PROVES (fresh mutants, re-run concurrency fixtures, new adversarial bodies, the full suite). Both budgets are capped:

- **RE-REVIEW mode is a different dispatch.** Pass the reviewer: `Mode: RE-REVIEW`, the previous verdict (its findings and its surviving mutants, verbatim), the correction's diff (`git diff` on the touched files — read-only), and the spec path. Its scope is: (a) prove each finding it was told is closed is actually closed, (b) regression on the files the correction touched. It does NOT re-read untouched files and does NOT re-mutate tests that already killed their mutants.
- **A finding first raised in a re-review blocks only if it is a regression introduced by the correction or a genuine BLOCKER.** Anything else the re-reviewer notices is registered (LOW/MEDIUM), never a REJECT — it had its chance in the first review.
- **Severity is frozen at first sight.** A finding reported as LOW/MEDIUM in one round cannot be promoted to HIGH in a later round without new evidence (a failing test, a reproduced scenario). Reword it, don't re-rank it.
- **The re-review proves the correction; it does not hunt.** Re-run only the mutant or the fixture attached to each finding declared closed. No new mutants unless the correction added or changed a test — and then only on that test. No new adversarial fixtures; one written in the first review is re-run only if it failed there.
- **The full suite does not run inside a re-review.** The parent runs it once before closing the path; a second full run in the same round proves nothing new.
- **Cap: 2 correction rounds per path.** If the second re-review still REJECTs, the parent decides with what it has: fix the residue pointwise, or accept with the gap registered in the spec. A third round is a sign the spec is wrong, not that the code needs another pass — reread the spec before spending more.
- **Owner items don't ride the correction.** A decision the user makes mid-loop (naming, a config value, a rule) is either applied pointwise by the parent without re-review, or written into the spec and handed to the NEXT deliverable. Bundling owner items into a correction round reopens the full review for work that was never in question.

## Static gates (the minimum proof, per deliverable)

The implementer delivers and the reviewer confirms, but **you run the gates once before closing** each deliverable — don't trust the summary blindly. The exact set is the project's (read its `AGENTS.md`); typically: build · vet/typecheck · tests · linter · plus whatever proves non-regression. Run the test on an ISOLATED line (`... ; echo $?`), never in a pipe with grep (the grep's exit code lies).

**Test-run budget.** The full suite (`./...`, integration containers, `-race -count=N`) runs **once per agent, at the end** — implementer and reviewer each close with one full green run, and the parent runs it once more before closing the path. Everything in between (a mutant, a fixture, a fix) runs **scoped**: the package under change, `-run` on the test that matters, `-count=1`, and always `-timeout` (a mutant that deletes a rollback or a cancel can hang the suite). Tell every subagent this in its prompt; a reviewer left to itself runs the full suite several times.

## Rules

- **The reviewer never wrote the code.** "Unbiased" is the point: every reviewer is a fresh subagent that never saw the implementer's reasoning.
- **Mutation testing is the anti-hollow gate** (it lives inside `unbiased-reviewer`): a test that stays green with production broken is hollow. The durable artifact of a mutation is the **test that kills the survivor** — the correction adds it and it stays in the repo; the mutant itself is thrown away. A killed mutant is never re-run in a later round.
- **A test must not force shape onto production** (project rule): if the reviewer says a mutant would only die by widening production (a clock injected just for the test), that's a registered gap, not a REJECT.
- **Trade the number of gates, not the gate.** A full correctness review costs about as much as the implementation it reviews, whatever the diff's size — that is the argument for grouping deliverables into one path, never for skipping the gate where money, external writes, data transactions or concurrency are at stake.
- **The git index belongs to the user.** The implementer and reviewer agents carry the rule; a general-purpose subagent gets it in its prompt: no `git add`, `reset`, `restore`, `rm --cached`, `commit`, `mv`, `stash`, `checkout -- <path>` or `clean`, and it reports which staged files it changed.
- **A mutant never touches the working tree.** It runs from a scratch copy — `go test -overlay` in Go — so there is nothing to revert. After every agent that mutated, `git status --short` lists only the files the delivery changed; anything else is a mutant left behind.

## Parameters

- **Subagent types:** the implementer is **`backend-implementer`** or **`frontend-implementer`**, by the stack of the repo — each carries the house rules and stops on product decisions; the reviewers are `architecture-reviewer` and, where an error costs a lot, `unbiased-reviewer`. The prompt carries only the mini-spec, the files in scope and the test-run budget above.
- **Subagent model:** `opus` for any work that holds a decision — concurrency, state, side effects, writes to a marketplace or external platform, design — and always for both reviewers. `sonnet` only for mechanical work: a rename, a 1:1 port, a sweep the spec fully dictates. Pass `model` explicitly when dispatching.
- **Scale to the request:** a small-but-non-trivial feature = 1 path, 1 gate. A large feature = several deliverables grouped into a few paths, each gated once when it closes — not one gate per deliverable. For long multi-wave orchestration (per-wave gate, doc syncing), that's the `orchestrator` skill — dev-loop is the unit it reuses.
