---
name: dev-loop
version: 0.3.0
description: >-
  IMPLEMENTS (builds/refactors) a NON-trivial code feature with a built-in quality gate — the loop fresh-implementer → unbiased-reviewer (the `unbiased-reviewer` agent, with mutation testing) → the parent decides → [corrector→re-review], with the gate fired once per code path rather than per micro-deliverable. Use WHENEVER the task is to WRITE non-trivial code: a change spanning multiple files, new logic with state or concurrency, a feature with regression risk, or when the user wants the implementation done with rigor / doesn't trust "it compiled" alone — EVEN if they don't say "loop", "review" or "gate". Also when you (Claude) are about to implement a large feature yourself and a second unbiased pair of eyes is worth it. Do NOT use for: one-liners, renames, copy tweaks, 1:1 mechanical edits; and NOT for reviewing existing code, a PR, or a diff (that is the `unbiased-reviewer` agent on its own — this skill BUILDS, the review is only its inner gate). Proven on a 23-deliverable run (0 final rejects; the reviewer caught real bugs and hollow tests on nearly every deliverable).
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

Whoever invokes the skill is the **orchestrator**. They do NOT edit production — they dispatch fresh subagents and decide. This keeps the parent's context clean and enforces "unbiased" (whoever reviews is never whoever wrote it).

## The review unit is the code path, not the deliverable

Split the feature into deliverables for IMPLEMENTATION, in dependency order — but the adversarial gate fires **once per code path** (the deliverables that touch the same package and the same files), when that path is closed. **The review does not scale down:** measured on a Go backend run (6/set/2026), a 2-line deliverable cost a 12m30 review, the same order as a 200-line one, and 5 deliverables over the same 3–4 files became 4 full reviews of the same code. The floor is what pays for the gate, not the ceiling.

- **Same package, same files → one review scope.** Open a second scope only when the path is genuinely another one: another package, another wire contract, another consumer.
- **A shared package is serial anyway.** An implementer editing while a reviewer runs `build` or a mutant gives a false verdict, so deliverables on one package can never overlap — grouping them costs nothing in risk and saves an entire review.
- **The round cap counts per path**, not per deliverable.

## The loop, per code path

1. **Mini-spec, if there isn't one.** If the feature arrived without a spec, write a short mini-spec BEFORE coding: gap → what to deliver → **verifiable exit criterion** (a command/assertion that proves done) → what's out of scope. Save it to disk (the subagent rereads it from there — survives context compaction).

2. **Dispatch the IMPLEMENTER**, one per deliverable, in dependency order (fresh subagent, `model: opus` by default). It reads the mini-spec from disk, codes ONLY this deliverable, and leaves the static gates green before returning. It returns a short summary of what changed + the files touched.

3. **Between deliverables of the same path, the static gates are the proof.** When you want a second pair of eyes before the path grows on a wrong foundation, dispatch a **READ REVIEW** — the `unbiased-reviewer` with `Mode: READ REVIEW`, which reads the spec and the diff and **runs nothing**: no mutant, no fixture, no suite. It returns findings as advice, with no verdict; you or the next implementer decide what to take. Cheap by construction — it catches a wrong direction early, it never replaces the gate.

4. **When the path is closed, dispatch the full gate** — the **`unbiased-reviewer`** agent (fresh, did NOT see any implementer's conversation), over ALL the deliverables of the path at once. Pass it: `Mode: FIRST REVIEW`, the mini-spec path, every file touched in the path, the exit criteria. It returns VERDICT (APPROVE/REJECT) + mutation (killed/survivors) + anchored findings. **A green build is not enough — the reviewer proves the tests aren't hollow and hunts bugs/regressions.** This is the one full pass the path gets; everything after it is scoped.

5. **DECIDE, following the spec** (you are the arbiter — accept neither the implementer nor the reviewer blindly):
   - APPROVE with no relevant finding → confirm by running the static gates yourself once → **next path**.
   - Trivial finding you agree with → fix it pointwise and re-run the gate. Only for a minimal, obvious correction. No re-review for this.
   - Relevant finding (BLOCKER/HIGH) or you disagree with the reviewer → dispatch a **CORRECTOR** (fresh subagent) with the findings: "apply these fixes per the spec, don't touch anything else". Then dispatch a **new `unbiased-reviewer` in RE-REVIEW mode** (below).
   - Finding that contradicts the spec → **the spec wins**; reject the finding and record why. This one rule avoided 3 correction rounds on the 6/set run — a reviewer raises legitimate things that are out of scope, and the parent refuses them without spending a re-review.

6. Only with the path **approved + green**, move to the next.

## Re-review is scoped, capped, and its goalposts are frozen

A correction round changes a few files, yet the re-review is the round that runs away: **measured 6/set/2026, it was the longest of the whole run — 14m38 against 8m33–12m30 for the first reviews**. Scoping what it RE-READS was not enough, because the turns are spent on what it RE-PROVES (8 fresh mutants, a re-run concurrency fixture, 8 new adversarial bodies, the full suite). Both budgets are capped:

- **RE-REVIEW mode is a different dispatch.** Pass the reviewer: `Mode: RE-REVIEW`, the previous verdict (its findings and its surviving mutants, verbatim), the corrector's diff (`git diff` on the touched files — read-only), and the spec path. Its scope is: (a) prove each finding it was told is closed is actually closed, (b) regression on the files the corrector touched. It does NOT re-read untouched files and does NOT re-mutate tests that already killed their mutants.
- **A finding first raised in a re-review blocks only if it is a regression introduced by the correction or a genuine BLOCKER.** Anything else the re-reviewer notices is registered (LOW/MEDIUM), never a REJECT — it had its chance in the first review.
- **Severity is frozen at first sight.** A finding reported as LOW/MEDIUM in one round cannot be promoted to HIGH in a later round without new evidence (a failing test, a reproduced scenario). Reword it, don't re-rank it.
- **The re-review proves the correction; it does not hunt.** Re-run only the mutant or the fixture attached to each finding declared closed. No new mutants unless the corrector added or changed a test — and then only on that test. No new adversarial fixtures; one written in the first review is re-run only if it failed there.
- **The full suite does not run inside a re-review.** The parent runs it once before closing the path; a second full run in the same round proves nothing new.
- **Cap: 2 correction rounds per path.** If the second re-review still REJECTs, the parent decides with what it has: fix the residue pointwise, or accept with the gap registered in the spec. A third round is a sign the spec is wrong, not that the code needs another pass — reread the spec before spending more.
- **Owner items don't ride the corrector.** A decision the user makes mid-loop (naming, a config value, a rule) is either applied pointwise by the parent without re-review, or written into the spec and handed to the NEXT implementer. Bundling owner items into a correction round reopens the full review for work that was never in question.

## Static gates (the minimum proof, per deliverable)

The implementer delivers and the reviewer confirms, but **you run the gates once before closing** each deliverable — don't trust the summary blindly. The exact set is the project's (read its `CLAUDE.md`); typically: build · vet/typecheck · tests · linter · plus whatever proves non-regression. Run the test on an ISOLATED line (`... ; echo $?`), never in a pipe with grep (the grep's exit code lies).

**Test-run budget.** The full suite (`./...`, integration containers, `-race -count=N`) runs **once per agent, at the end** — implementer, corrector and reviewer each close with one full green run, and the parent runs it once more before closing the path. Everything in between (a mutant, a fixture, a fix) runs **scoped**: the package under change, `-run` on the test that matters, `-count=1`, and always `-timeout` (a mutant that deletes a rollback or a cancel can hang the suite; on the run above one did, for 10 minutes). Tell every subagent this in its prompt; a reviewer left to itself runs the full suite 5–7 times.

## Rules the run proved

- **Sequential, fresh subagents.** Implementer ≠ reviewer ≠ corrector. "Unbiased" is the point: whoever reviews can't be whoever wrote it.
- **Mutation testing is the anti-hollow gate** (it lives inside `unbiased-reviewer`): a test that stays green with production broken is hollow. This is what caught the hollow tests across the whole run. The durable artifact of a mutation is the **test that kills the survivor** — the corrector adds it and it stays in the repo; the mutant itself is thrown away. A killed mutant is never re-run in a later round.
- **A test must not force shape onto production** (project rule): if the reviewer says a mutant would only die by widening production (a clock injected just for the test), that's a registered gap, not a REJECT.
- **The gate is expensive AND worth it — on the right-sized unit.** A full review costs 8–15 min and 115–175k tokens, on par with the implementation it reviews, and it costs the same on 2 lines as on 200 (6/set/2026). That is the argument for the floor above, never for skipping the gate: on that same run 1 review in 4 found a BLOCKER (a write path ingesting from a notification without checking who owns the resource; two concurrent deliveries creating duplicate listings), and one that APPROVED still caught a migration default that would have pushed the entire mirror into the refresh queue, silently. **Trade the number of gates, not the gate.**
- **The git index belongs to the user.** Staged files are the user's review markers. No subagent runs `git add`, `git reset`, `git restore`, `git rm --cached`, `git commit`, `git mv`, `git stash`, `git checkout -- <path>` or `git clean` — every one of them moves or hides the user's work (a `stash push -u` empties the whole worktree silently). Reading is fine (`git status`, `git diff`, `git show`). Reverting a mutation means restoring the copy the reviewer saved before editing, never a git command. Put this list, verbatim, in every subagent prompt.

## Parameters

- **Subagent model:** `opus` by default (implementer, reviewer, corrector) — it's where quality matters. Pass `model: opus` explicitly when dispatching if the orchestrator runs on a different model (a subagent inherits the parent's model).
- **Scale to the request:** a small-but-non-trivial feature = 1 path, 1 gate. A large feature = several deliverables grouped into a few paths, each gated once when it closes — not one gate per deliverable. For long multi-wave orchestration (all-night, per-wave gate, RUN-LOG, doc syncing), that's the scope of a separate orchestration skill — dev-loop is the unit it reuses.
