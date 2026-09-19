---
name: ui-ux
version: 0.1.0
description: >-
  Designs and reviews landing pages, product pages and app screens that hold attention on first load — live product UI instead of illustration, scroll-driven reversible motion, neutral surfaces with one accent — through one divergent taste round, a review loop where every complaint becomes the smallest fix plus a measurement, and bundled Playwright checks (void scroll, blurred rest state, loops, no-JS final state, clipped content, fold runway). Use it WHENEVER Diego builds, designs, redesigns or "reimagines" a landing, page, section or screen, wants it bonito, "encantador", with WOW, motion, effects or "coisas vivas", reviews a design from screenshots, or complains it looks ugly, crooked, blurred, cramped, too far, too short or startling — even without saying "UI" or "UX". Not for SEO or indexing (that is landing-seo-geo). Replies match the user's language.
---

# ui-ux

Born from the Rednev landing rewrite (18–19 Sep 2026): the old page was rejected as "not enchanting,
a lot to read"; after one divergent round and ~15 small review loops the new one was called "a leap,
in line with what I expected". What is here is what **worked**, what was **rejected and why**, and the
order that avoids redoing work. Rules quote the complaint that produced them, so a familiar complaint
leads straight to its fix.

**Scope.** A landing or marketing page gets the whole method: stage, motion, choreography. An app screen
takes the visual rules, the copy rules and the proof, not scroll choreography — people come to a screen to
work, and motion there has to earn its place on every visit, not once.

## The order that works

1. **Words and promises before pixels.** Write the page's arc (sections, title, support line, what moves)
   and a **promise table swept against the code**: what may be said (built) vs what may not (not built,
   divergent). Every scene, list and caption later draws only from the left column. A beautiful page that
   promises an unbuilt feature is the most expensive defect there is.
2. **One divergent taste round.** 2–3 directions that differ in ground, composition *and* motion (not three
   palettes of one layout), each a self-contained HTML file in a lab folder outside git, served locally so
   the human **scrolls** it — motion can't be judged from a screenshot. Same two sections in every
   direction (hero + one proof section) so the comparison is fair.
3. **The human picks; then there is ONE file**, edited in place. No `-v2`, `-v3` copies: they multiply
   context and confusion. Keep rejected directions only as reference.
4. **Every complaint becomes (a) the smallest change that resolves it and (b) a measurement** that would
   have caught it. The suite re-runs on every change, so the human stops finding regressions one by one.
5. **When the human describes the choreography in steps, the spec IS those steps**, numbered and literal
   ("reaches the top → does the swap → a small scroll → moves up → orange appears a bit later"). Your own
   acceptance criteria only measure those steps; never substitute them. Inventing criteria ("≥200px",
   "centred pin") and swapping them each round cost four wasted rounds here.
6. **Side effect → revert to the last approved state**, don't stack a fix on a fix.
   **Same piece rejected twice, or the human hedges ("not sure", "maybe")** → stop iterating in series:
   build 2–3 variants side by side on one comparison page and let them pick by looking.
7. Lab approved → handoff doc + technical note → build in the real repo (a new session).

## Motion: the rules

Before building a 3D hero, card stack, pinned fold, colour turn or hub scene, read `references/motion.md`
for the parameters that passed review.

- **Background and transition effects follow the scroll and go back** ("beautiful effects have a reason:
  they go and come back, they are not timed"). Light intensity, tilt, folds, colour turns: all a function
  of scroll progress, reversible, monotonic.
- **Content inside devices loops like a video**: starts when the block is ≥50% visible, plays, holds the
  final state ~3–4 s, soft-resets (fade, never a jump), replays; pauses off-screen and on hover. A scene
  that plays once is missed by whoever was reading the other column.
- **Choreography**: one thing at a time, staggered, physical easing. Everything animating at once reads as AI.
- **3D only while moving; flat at rest.** A rest state with `translateZ`/perspective/`will-change`/filter
  rasterises the layer and the text looks **blurred**. At rest: `transform: none`, no filter, depth by
  z-index + shadow.
- **Tilt is decisive or absent.** A few degrees, or two objects at different angles, reads as a layout
  mistake ("crooked"). Both objects share one perspective and one plane.
- **Light sits behind objects, always.** Never let a glow/rim layer pass over a device; never recede an
  object with opacity (the light shows through, "behind glass") — recede with brightness.
- **No void scroll.** Sticky/pinned sections must not create empty bands; measure the max empty band while
  pinned. A pinned block pins at the **top** so the whole block is visible, not centred.
- **Colour turns join seamlessly**: the growing shape reaches full width exactly when the next ground
  appears; the ground unrolls from the join. Never a small pill above a separate full-width strip.
- **Timing starts where the eye is**: a highlight tied to a timer also lights on the first scroll pixel;
  recompute scroll progress on `load` (layout settles after the first read).
- **Mobile gets its own runway**: a fold that is smooth on desktop can be an instant swap on a phone.
  Sample every 20px: progress may not jump > 0.1 between samples.
- Content never depends on JS; with no JS and with `prefers-reduced-motion`, everything is in its final state.

## Visual rules

- **A dark ground is neutral graphite, saturation 0** on every surface (ground, card, node, overlay). A warm-tinted
  black reads brown. The accent never washes a surface; it lights the key moment and the CTA.
  **Semantic colour stays** (green = a good event like "sold"/"sent"; amber/red = attention).
- **Real product UI, rebuilt as live HTML** inside the devices (not screenshots, not illustrations), so it
  can move and stays sharp. Real third-party marks (the actual logo of each service shown), never a
  generic letter.
- **Text on a coloured ground goes in a solid panel**, never loose; internal notes (sources, file names)
  never render — keep them as HTML comments.
- **Like-for-like comparisons**: a "WhatsApp vs Telegram" contrast shows both chat *lists*, not a list vs
  an open chat.
- **The home is a stage; reading pages are light.** Long informational pages use a light neutral variant of
  the same identity (dark header/footer/opening/CTA bands, light body, ~68ch). No white bands inside the
  stage page.
- **Brand name in running text is plain text**; the coloured wordmark is a signature (header, footer, close).
  A wordplay on the name stays legible: the simplest encoding wins (an accent on part of the word) and
  needs no explanation, never a distortion ("more like a shadow" means softness — opacity, blur, fade —
  and a skewed name was "horrible").
- **A notice is born from the object it speaks for**, never a chip floating between objects ("lost in the
  middle"). Shadows are tight contact shadows; a wide halo reads as "too much".
- Anti-AI-look, forbidden: purple-blue gradients, glass blobs, sparkles icons, emoji decoration, 3D mascots,
  a punchline in every block, everything moving at once.

## Copy that touches design

- Promise only what the promise table allows; an "in testing" badge must be consistent across every
  section that shows the same thing (price list included).
- **A count of what we do reads as the limit of what we do** ("Three jobs that arrive done", cards numbered
  01/02/03). Examples are presented as examples ("see how the work arrives"), unnumbered, and show the
  customer's common case — an unusual setup makes the reader think "that's not me".
- A price section sells with a **grouped, rich list of built work** (grouped by the job each item does),
  not 4 basic bullets that make the price look high.
- Trust without customers: a **signed founder note** (`references/trust-and-founder.md`), first person, the
  real trigger story, laid out as a founder card — never a CV line, never "passionate about".

## Proof that is cheap and real

- **Measure in text** (boundingBox, computed style, scrollWidth, fold progress); save screenshots, never
  read them back. Give an explicit image budget per task (0–4) only when visual judgement is the work.
- **Viewports: 390×844, 1024×768, 1440×900, 1440×1211, 1920×1080.** The human reviews on tall screens;
  bugs hid at 1211 that 900 never showed.
- **Use the bundled suite instead of writing checks from scratch**: `scripts/*.mjs` (Playwright, one JSON
  config per page) cover empty bands, rest-state transforms (blur), loops running, final state without
  JS / reduced motion, saturation-0 surfaces + light z-order, **clipped content inside overflow-hidden
  boxes** (page-level scrollWidth misses it), fold runway and **contrast of every text node against its
  real ground**. Read `references/measurement.md` for how to run and configure them, and start the config
  from `scripts/examples/noite.json`.
- **Text in the DOM is not text on screen.** A written title was present, selectable and invisible (split
  words forced to `display:block`); a light reading body inherited white ink ("letters that turned white").
  Both passed a text diff, `clipped` and `final-state`. Measure what renders: each word's width and opacity
  after the scene's last step and again in cycle 2, and `contrast` on every page.
- **A port from the lab to the repo regresses the same ways every time**: a mobile value leaking as the
  base, a specificity rule left behind, a sticky offset that ignored the real header, a token the light
  variant never redefined (color inherits resolved). Compare the build to the lab **per element**
  (boundingBox of each piece of a row), not per section height.
- **The lead's final look is a few screenshots, read once.** Eight passing scripts still let through ten
  defects the human saw in one review; a handful of images at the end costs less than that round.
- Long-lived preview servers run from the orchestrating session, not from a subagent (they die with it).

## Working with the human and agents

A question the human asks twice gets the answer first, alone, then the reasons — the first time it was
buried in a long reply.

Dispatch itself follows the `orchestrator` skill; what is specific to a design round:
one design agent carries the whole round by continuation; switch only near ~70% context and at a natural
seam, after it writes a technical note + copies its tools out of the scratchpad. Every follow-up is a
numbered minimal delta with "change nothing else" and a 1–8 line report ceiling.

## Adding a lesson

A new lesson becomes a rule in the section it belongs to, quoting the complaint that produced it; when it
contradicts a rule, replace that rule. Parameters of a pattern go to `references/motion.md`. No log of
rejections: a rule read only after the complaint repeats is a rule applied too late.
