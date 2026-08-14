---
name: implementation-plan
version: 0.1.0
description: >-
  Turns a large or fuzzy objective into a BULLETPROOF, executable SPEC — the spec that `dev-loop` or an autonomous goal then executes. It recons the code verifying every claim against the SOURCE (not docs/comments that lie), pre-consolidates the already-settled decisions the human points at (so the executor won't re-ask or reinvent), marks the open decisions as explicit stop-points, writes inviolable rules + verifiable exit criteria, and runs an ADVERSARIAL REVIEW of the draft spec (via the `unbiased-reviewer` agent) to catch BLOCKERs before a line is written. Use WHENEVER the user wants to plan, spec, scope, or think through a large/risky change; de-risk a big refactor; or set up an autonomous goal — EVEN if they just say 'let's plan X', 'how should we approach Y?', or 'I want to build a big feature'. Fire it BEFORE implementation, not during. Proven: the adversarial spec review caught real BLOCKERs on every run (wrong premise, internal contradiction, a contract comment that lied).
allowed-tools:
  - Read
  - Grep
  - Glob
  - Bash
  - Write
  - Edit
  - Agent
  - Task
  - WebSearch
  - WebFetch
---

# implementation-plan — objective → bulletproof spec

(Instructions are in English so the model reasons robustly; respond to the user in their own language.)

Planning is where disasters are cheap to avoid. Every BLOCKER an adversarial review catches **in the spec** costs one text edit; the same BLOCKER discovered mid-execution costs hours built on a wrong premise. This skill produces a spec that `dev-loop` (with you in the loop) or an autonomous goal executes without stalling or inventing.

## The principle behind everything: CODE is the truth, docs are hints

Docs, contract comments, roadmaps, ADRs — **can be stale or lie** (it happens: a roadmap marks a milestone "in progress" that is actually done; a function's doc-comment promises an error the code never returns). Every claim that enters the spec is **verified against the source code** first. A doc says where to look; the code says what it is.

## The phases

### 1. Recon — verify the real state
Survey the current state relevant to the objective, **confirming each claim in the code** (cite file:line). What already exists? What's missing? Verify not just "does this exist?" but **"does this work under the new invariant?"** (a mechanism can exist and not serve — e.g., a lock that exists but doesn't exclude under concurrency).

### 2. Pre-consolidate the already-SETTLED decisions
The human points at the reference docs (ADRs, STATUS, decisions, architecture). **You consult what they pointed at** — you don't go hunting the repo. Pull into the spec **every decision relevant to the scope that's already been settled** (even if it lives in an ADR the executor wouldn't read). Reason, proven: nearly every question an autonomous goal asks is about something **already decided**, just not in the source doc it reads. Consolidating here = the executor won't re-ask or reinvent. Also put the **paths** of the reference docs in the spec, for the executor/subagents to consult on demand (the spec stays lean: it points, it doesn't copy).

### 3. Separate what's genuinely OPEN
List the decisions the human **hasn't settled yet**. Ask them to settle the ones they can now (pre-deciding everything they already know shrinks the real list). Whatever stays open becomes an explicit **stop-point** in the spec: "here the executor STOPS and asks / skips and records — it does not decide alone". Distinguish: an *implementation* decision (the executor decides) vs a *product/architecture* decision (the human decides).

### 4. Write the spec
Minimal structure that proved to work:
- **Inviolable rules** — always include the ones that prevent catastrophe/rework: **NEVER destructive git** (`restore`/`reset`/`clean`/`checkout --`), especially if there's uncommitted work (check `git status` — don't assume "state = last commit"); **D0 snapshot** before editing; project conventions (read the `CLAUDE.md`: layers/hexagonal, error handling, i18n, **mocks via the project's tool, not hand-rolled fakes**, terse comments); **static proof per deliverable** with a verifiable command; **locate by symbol, not by line** (line numbers drift); **verify a flag/API by real probe, not by doc**.
- **Deliverables** — each with a **VERIFIABLE exit criterion** (a command/assertion that proves done), target files, and the gap→what-to-deliver.
- **What the spec SKIPS** — the open decisions from phase 3, explicit.
- **Downstream export** — when a type/contract is consumed by another layer, mandate exporting it AND its nested types (a real bug that already broke a consumer).

### 5. Adversarial review of the spec (the gate that catches BLOCKERs)
Dispatch the **`unbiased-reviewer`** agent (or a fresh adversarial reviewer) against the **draft spec + the code**, with the explicit mission of finding BLOCKERs **before execution**. Classes it must hunt (all already caught in real runs):
- **Wrong recon premise** — the spec claims something exists/works and the code says otherwise.
- **Internal contradiction** — two sections of the spec contradict each other (e.g., an allowlist with 3 items in one, 4 in another).
- **The spec trusts a COMMENT that lies** — require verifying the behavior in the code, not the comment; and that the test exercises the real path, not mocks the ready result.
- **A spec rule conflicts with what the repo already has** (e.g., "mocks only" but there's a pre-existing hand-rolled fake that won't disappear).
- **Test/infra contradiction** — the spec asks for proof the test infra can't give (e.g., a "real concurrency test" in a mock-only repo) → resolve it in the text (authorize scoped ephemeral infra) or downgrade.

### 6. Apply the fixes, re-review if needed
Fix the spec with the findings (architecture decisions that surface, the human settles). The result is the **bulletproof spec**, ready for `dev-loop` (execution with a human) or for an autonomous goal.

## Output
A spec file on disk (the executor rereads it from there — survives compaction), with: verified state, inviolable rules, deliverables with verifiable criteria, pre-settled decisions, stop-points for the open ones, and the reference-doc paths. Plus a short summary for the human: what the adversarial review caught and what they still need to settle.

## Boundary with the other pieces
- This skill **plans**; `dev-loop` **executes** (composing `unbiased-reviewer` on each deliverable's review).
- An autonomous overnight goal = composes this skill (produces the spec) + `dev-loop` per deliverable + waves/gate/never-block/RUN-LOG. Its difference is **autonomy**: it decides on its own within the spec's bounds, never blocks (hits an open decision → records and skips), runs unattended.
