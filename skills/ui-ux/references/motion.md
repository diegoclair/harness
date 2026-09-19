# Motion patterns that were approved (Rednev landing, Sep 2026)

Numbers are the ones that passed review; treat them as starting points, then measure. The scenes tell
Rednev's story; reuse the mechanism and tell your own product's.
All patterns: vanilla JS + CSS custom properties (a single scroll handler writes progress vars; CSS does
the rest), no motion library; no JS / reduced motion = final state.

## Hero: device pair in 3D that rests flat
- Window + phone share **one plane**: tilt back together (~36° around the bottom edge, one perspective),
  phone lifted in depth (translateZ ≥ 40px) above the window. Below 1024: no tilt, depth by scale.
- Progress `--t` from scroll: flattens as the pair reaches the viewport centre; `t = 1` exactly when the
  pair's centre = viewport centre. Scene content only starts once ~60% of the pair is visible.
- At the end add a `.rest` state: `transform:none`, no perspective/preserve-3d/filter/will-change, 0.6 s
  settle. 3D returns only if the user scrolls back up.
- Light: a dark panel with an orange **rim** on its top edge + backlight + grain, `--ig = .1 + .9·(1-t)`
  style mapping — scroll-driven and reversible. Panel z 0, devices z 2. Devices catch the same rim light.
- The accent word in the title ("ok.") lights at the scene's ok step OR on the first scroll pixel,
  whichever comes first.
- One story at a time: the phone leads the first scene while the window shows the same item; at the
  second scene the window comes forward (scale 1.02) and the phone recedes by **brightness** (0.62), never
  opacity.

## Sticky card stack ("see how the work arrives")
- Cards content-sized (not viewport-sized), sticky top 24px, 16px between cards; a card recedes
  (scale/brightness) **only while the next card overlaps it**; the last never recedes.
- Each card's scene: starts at ≥50% visible, loops (play → hold 4 s → fade 0.38 s → replay), pauses
  off-screen, when > 90% covered, and on hover.
- Traps: margins on sticky children push the card out early; a 100vh dwell spacer creates a void; sizing
  cards to the viewport makes giant cards on tall screens.

## Hub with pulses (stock follows the sale)
- Vertical hub: marketplace node A (top) — centre product node with the stock — marketplace node B.
- Loop: "sold 1" pops on A (semantic green) → light pulse runs A→centre along its line → centre digit
  rolls 12→11 (odometer, only the changing digit) → pulses centre→A and centre→B → each rolls to 11 with
  a brief ring → hold → reset. The pulse is clipped to its line so light never crosses node text.
- Engine trap: the rewind must clear every step class (a regex that only matched letters left `a0`/`a1`
  classes set, so from cycle 2 the scene looked frozen).

## "Settings fold into one question" (market complexity vs our question)
- Desktop: the block pins at the **top** (sticky 24px) once its top reaches it; the fold runs over ~300 px
  of pinned scroll (list of whole rows → one question card), then ~150 px hold, then release; scroll-driven,
  reversible. The resolved card then loops "Sim" pressed → second item slides into the first; the card's
  height eases to its content (no gap).
- Mobile: panel rows fade/collapse staggered while the card fades/scales in (0.96→1), runway ≥ 300 px,
  progress jump ≤ 0.1 per 20 px sample.
- Show the list **complete** (whole rows only) before anything folds.

## Colour turn into pricing
- After the previous block releases (+40 px), a pill around the section title grows over ~250 px of scroll
  to full width, fading in over its first 24 px; the pricing ground **unrolls from the join** only when the
  pill is full width. Reversible. On tall screens without this rule the ground appeared under a small pill.
- The coloured ground covers the pricing section only; FAQ and close return to graphite with a straight edge.

## Pricing interaction
- A log slider of orders/month (50 → 50,000) lights the matching tier card and shows monthly, annual and
  overage from the published grid (never hand-typed numbers). Without JS: the whole grid.
- Below the tiers, one solid ink panel: grouped "everything included" list, the three rules, the CTA.
