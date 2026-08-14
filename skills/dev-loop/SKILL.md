---
name: dev-loop
version: 0.1.0
description: >-
  IMPLEMENTS (builds/refactors) a NON-trivial code feature with a built-in quality gate — the loop fresh-implementer → unbiased-reviewer (the `unbiased-reviewer` agent, with mutation testing) → the parent decides → [corrector→re-review]. Use WHENEVER the task is to WRITE non-trivial code: a change spanning multiple files, new logic with state or concurrency, a feature with regression risk, or when the user wants the implementation done with rigor / doesn't trust "it compiled" alone — EVEN if they don't say "loop", "review" or "gate". Also when you (Claude) are about to implement a large feature yourself and a second unbiased pair of eyes is worth it. Do NOT use for: one-liners, renames, copy tweaks, 1:1 mechanical edits; and NOT for reviewing existing code, a PR, or a diff (that is the `unbiased-reviewer` agent on its own — this skill BUILDS, the review is only its inner gate). Proven on a 23-deliverable run (0 final rejects; the reviewer caught real bugs and hollow tests on nearly every deliverable).
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

## The loop, per deliverable

If the feature has independent parts, break it into deliverables and run the loop on each, **in dependency order**. For EACH deliverable:

1. **Mini-spec, if there isn't one.** If the feature arrived without a spec, write a short mini-spec BEFORE coding: gap → what to deliver → **verifiable exit criterion** (a command/assertion that proves done) → what's out of scope. Save it to disk (the subagent rereads it from there — survives context compaction).

2. **Dispatch the IMPLEMENTER** (fresh subagent, `model: opus` by default). It reads the mini-spec from disk, codes ONLY this deliverable, and leaves the static gates green before returning. It returns a short summary of what changed + the files touched.

3. **Dispatch the REVIEWER** — the **`unbiased-reviewer`** agent (fresh, did NOT see the implementer's conversation). Pass it: the mini-spec path, the files the implementer touched, the exit criterion. It returns VERDICT (APPROVE/REJECT) + mutation (killed/survivors) + anchored findings. **A green build is not enough — the reviewer proves the tests aren't hollow and hunts bugs/regressions.**

4. **DECIDE, following the spec** (you are the arbiter — accept neither the implementer nor the reviewer blindly):
   - APPROVE with no relevant finding → confirm by running the static gates yourself once → **next deliverable**.
   - Trivial finding you agree with → fix it pointwise and re-run the gate. Only for a minimal, obvious correction.
   - Relevant finding (BLOCKER/HIGH) or you disagree with the reviewer → dispatch a **CORRECTOR** (fresh subagent) with the findings: "apply these fixes per the spec, don't touch anything else". Then dispatch a **new `unbiased-reviewer`** to validate. Repeat until APPROVE.
   - Finding that contradicts the spec → **the spec wins**; reject the finding and record why.

5. Only with the deliverable **approved + green**, move to the next.

## Static gates (the minimum proof, per deliverable)

The implementer delivers and the reviewer confirms, but **you run the gates once before closing** each deliverable — don't trust the summary blindly. The exact set is the project's (read its `CLAUDE.md`); typically: build · vet/typecheck · tests · linter · plus whatever proves non-regression. Run the test on an ISOLATED line (`... ; echo $?`), never in a pipe with grep (the grep's exit code lies).

## Rules the run proved

- **Sequential, fresh subagents.** Implementer ≠ reviewer ≠ corrector. "Unbiased" is the point: whoever reviews can't be whoever wrote it.
- **Mutation testing is the anti-hollow gate** (it lives inside `unbiased-reviewer`): a test that stays green with production broken is hollow. This is what caught the hollow tests across the whole run.
- **A test must not force shape onto production** (project rule): if the reviewer says a mutant would only die by widening production (a clock injected just for the test), that's a registered gap, not a REJECT.
- **The reviewer's cost is negligible, the payoff is asymmetric:** keep the reviewer even on a "mechanical" deliverable — on the run, an apparently-mechanical deliverable hid a silent failure that only mutation testing caught.

## Parameters

- **Subagent model:** `opus` by default (implementer, reviewer, corrector) — it's where quality matters. Pass `model: opus` explicitly when dispatching if the orchestrator runs on a different model (a subagent inherits the parent's model).
- **Scale to the request:** a small-but-non-trivial feature = 1 deliverable, 1 loop. A large feature = several sequential deliverables. For long multi-wave orchestration (all-night, per-wave gate, RUN-LOG, doc syncing), that's the scope of a separate orchestration skill — dev-loop is the unit it reuses.
