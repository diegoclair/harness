# Copy review — the adversarial pass the linter can't run

> When to read: together with `reference/examples.md`, BEFORE writing slide copy — not after. Re-read the checklist once copy is drafted, before `check`/`render`.

The linter validates structure (word counts, layout placement, contrast). It cannot tell you whether the copy is redundant, whether the strongest hook is buried, whether a CTA makes sense in isolation, or whether a line would make you cringe read aloud. Two adversarially-reviewed carousels shipped from this skill scored 6.5/10 — every defect was copy/editorial, none structural. This file codifies what those reviews found so the first draft is better, not just lint-clean.

## The 7 checks

Run all 7 against the drafted YAML before `check`. Each failure mode below is a *pattern*, not a one-off — if you catch yourself doing it, assume the reader will too.

### 1. Slide 3 must say something NEW, not reword slides 2/4

The value bomb (ST-05) is a placement rule — layout must be `big-number`/`list`/`quote`. It says nothing about whether the *content* is actually new information. The common failure: slides 2, 3, and 4 all restate the same problem in different words — "redundancy disguised as rhetorical repetition."

**Test:** cover slides 2, 3, and 4. Read only their core claims (ignore phrasing). If two of them could be merged without losing an idea, one is filler.

- Bad: slide 2 = "Providers lose clients when follow-up is manual." Slide 3 = "87% of providers lose clients to no-follow-up." Slide 4 = "Manual follow-up is why clients disappear." → same fact three times.
- Good: slide 2 states the problem qualitatively (no number). Slide 3 introduces the number/mechanism that REFRAMES it (e.g., a cause the reader didn't expect — "it's not lack of leads, it's the 6th day of silence"). Slide 4 moves to the solution/mechanism, a different beat entirely.

**Fix:** if slide 3 and a neighbor overlap, keep the more surprising one and replace the other with a different function — setup (pain, no number) before, mechanism/proof after.

### 2. Arm the strongest trigger EARLY (cover or slide 2), never save it for the end

A carousel loses roughly half its audience by slide 5–6. A real production failure: the strongest trigger (scarcity) only appeared on slides 8–9 — by then the reader who would have reacted to it was already gone.

**Test:** name the single strongest psychological trigger in the whole carousel (scarcity, contrarian claim, specific number, named pain). What slide is it on? If the answer is higher than slide 2, move it forward or plant an explicit forward-reference to it in the cover/slide 2 ("and by the end you'll see why this expires in days, not months").

**Fix:** the cover or slide 2 should either contain the trigger directly, or explicitly promise it's coming ("the reason has an expiration date — slide 3"). Don't hide your best material as a reward for the reader's patience; use it to buy that patience.

### 3. The CTA must be self-sufficient — name the object, not just the action

Many readers jump straight to the last slide without reading the rest. A vague referent kills the trigger even when the verb and urgency are right. Real failure: "Your name is still available?" — name *of what*? The object was defined on slide 2, invisible from the last slide.

**Test:** read ONLY the final slide, pretending you saw nothing else. Does every pronoun/reference resolve? If "it", "your name", "this", or "that" points to something defined earlier and not restated, it fails.

- Bad: `headline: "Seu nome ainda está livre?"` (nome de quê?)
- Good: `headline: "Seu nome de usuário lybel.com.br/seunome ainda está livre?"` or restate the noun: `"Seu link ainda está disponível?"`

**Fix:** name the object explicitly in the CTA slide's own text. Never rely on the reader having read slide 2.

### 4. Read-aloud / cringe audit

Some phrases pass every rational check (word count, structure, contrast) and still fail because a human wouldn't say them out loud without feeling embarrassed. Two failure directions, both real:

- **Influencer-speak** — phrases that exist only in social-media-guru voice, never in real speech: "te leio" ("I read you"), "salva esse post" as a bare command with no reason attached, "deixa nos comentários que eu respondo todos", "bora", "não é sobre X, é sobre Y" overused as a rhetorical crutch.
- **Press-release / institutional voice** — phrases that sound like corporate copy, not a person talking: "de quem constrói pra quem vive de atender", "somos apaixonados por resolver problemas reais", "conectando pessoas e soluções".

**Test:** read every slide's copy out loud (or sub-vocalize). For each line ask: "Eu diria isso numa conversa de verdade?" If the honest answer is "no, I'd never say that, but I'd write it" — it fails the audit even if the linter is silent.

**Fix:** replace the phrase with what you'd actually text a friend. If you can't imagine saying it to one specific person, don't put it on a slide.

### 5. Mixing PT and EN registers

See the PT-BR section below — this check applies whether the carousel is Portuguese, English, or mixed.

### 6. Big-number "1" glyph legibility

See `reference/layouts.md` big-number section — a lone numeral "1" in a geometric sans (Outfit and similar) can read as a lowercase "l". Applies whenever the `number` field is or starts with "1" (e.g. `"1"`, `"1 link"`, `"1x"`).

### 7. Engagement question in the caption should be a short binary choice anchored in the reader's own experience

Applies to `caption_seed`, not the slides. A rhetorical question ("Você já perdeu um cliente?") invites no reply because the answer is assumed. A binary question anchored in a real, concrete choice the reader has actually made gets replies because it's easy and low-stakes to answer.

- Bad (rhetorical, no real answer expected): "Você também acha que atendimento manual não escala?"
- Good (binary, anchored in the reader's actual setup): "Lista de espera no papel ou no WhatsApp?" / "Agenda no caderno ou no Google Calendar?"

**Fix:** phrase the caption's engagement hook as "A ou B?" where A and B are two concrete things the reader plausibly already uses — not a yes/no about whether they agree with your thesis.

## PT-BR guidance

`reference/examples.md` documents that there is no canonical PT-language reference carousel to mirror (see its "Brazilian/PT-language gap" section). This section fills part of that gap: not a design reference, but a copy register and a cringe list, so a PT-BR carousel doesn't default to a stiff translation of EN patterns or to influencer cliché.

**Register: informal Brazilian, not formal/institutional, not slang-heavy.**
- Use "você", never "tu" (unless the brief specifies a regional voice) and never the ultra-formal "o senhor/a senhora".
- Contractions and everyday connectors are fine: "pra" over "para" in casual copy, "tá" only if the brand voice is very informal — otherwise keep "está".
- Avoid formal business PT: "outrora", "destarte", "no que tange a", "supracitado" — nobody talks like this and it reads as translated-from-a-memo.

**PT influencer-speak to avoid** (parallel to the EN list in check #4):
- "te leio" / "leio vocês"
- "salva esse post" without a reason attached (fine WITH a reason: "salva esse post pra quando for montar sua agenda")
- "deixa aqui nos comentários que eu respondo todo mundo"
- "bora" as a generic hype filler ("bora lá", "bora nessa")
- "printa e me manda" used generically instead of a specific ask
- "gente" as a filler opener ("Gente, vocês não vão acreditar...")
- excessive "❗️🔥🚀" emoji stacking to fake urgency

**Adapting the EN hook formulas (H-01..H-13, see `reference/hooks.md`):** the formulas translate structurally (number + result, contrarian, confessional, etc.) but the literal phrasing doesn't. Don't calque "Save this post" into "Salve este post" — rewrite in the register above: "Guarda esse aqui." Don't calque "Here's what I learned" into "Aqui está o que eu aprendi" (stiff) — use "foi o que eu aprendi" or "foi isso que descobri". Translate the mechanism of the formula (what makes it stop the scroll), not the sentence.

## Dispatching an adversarial reviewer (editorial gate)

For carousels the user intends to publish (ship-intent signals — see SKILL.md Gate 2), run this checklist yourself first, then optionally dispatch a fresh subagent as an adversarial reviewer before declaring the carousel ready. This is a copy/editorial gate, distinct from Gate 2's pixel-level QA.

**When to dispatch a subagent instead of self-reviewing:**
- The user explicitly asks for a review, a second opinion, or a score.
- The carousel is ≥10 slides (same threshold as Gate 2 — volume makes self-review less reliable).
- You authored the copy yourself in this session and want a reviewer with no attachment to the phrasing (self-review under-catches your own blind spots).

**What to hand the subagent:** the full YAML content (not the rendered PNGs — this is a copy review, not a visual one) plus this file's 7 checks. Ask it to return a score, then per check: pass/fail with the specific slide and quote.

**Prompt shape:**
> "Review this carousel copy adversarially against the 7 checks in `reference/copy-review.md`. For each check, say pass/fail and quote the offending slide verbatim if it fails. Then give an overall score out of 10 and the single highest-impact fix. Be harsh — assume this is going to be read by someone looking for reasons to scroll past it."

**Acting on the result:** any check-4 (cringe) or check-3 (CTA referent) failure should be treated as a blocker before shipping — both are cheap to fix and expensive to ship. Check-1 (slide 3 redundancy) and check-2 (trigger placement) usually require restructuring, not just a rewrite — budget for it rather than patching around it.

## What we deliberately did NOT turn into a hard linter rule

Checks 2, 3, 4, 5, and 7 above require judgment about meaning, tone, and cross-slide semantics that a word-count/regex linter cannot reliably assess without false positives that would erode trust in the linter's warnings. Two checks (1 — slide-3 redundancy, and part of 4 — an influencer-speak phrase list) are mechanical enough that a lightweight heuristic was evaluated for the Go linter; see `reference/linter-rules.md` for what shipped as a rule versus what stayed here as editorial guidance only, and why.
