# Measurement suite

Each script exists because a real defect passed type-check, lint and a screenshot glance. They print one
`PASS`/`FAIL` line per check and a verdict, and exit non-zero on failure, so they chain in a gate.

**Run** from a folder where Playwright is installed (`npm i -D playwright`), or point `PLAYWRIGHT=` at a
project's copy. When that package's own browser build was never downloaded, reuse any Chromium already on
the machine with `CHROME=`. The scripts live in this skill's folder, so call them by that absolute path:
`CHROME=$(ls -d ~/.cache/ms-playwright/chromium-*/chrome-linux/chrome | head -1) PLAYWRIGHT=<project>/node_modules/playwright/index.js node <skill-dir>/scripts/clipped.mjs <url> --config my-page.json`
The URL can be a local server or a `file://` path to a lab HTML file.
Viewports come from `--viewports 390x844,…`, else from the config's `viewports`. Always include
**1440x1211**: the human reviews on that screen, and bugs hid there that 900 never showed.
The config holds the page's selectors under one key per script; `scripts/examples/noite.json` is the
Rednev landing's, with the reason written next to every allowance.

| Script | The bug that produced it | Pass means |
|---|---|---|
| `empty-bands` | A pinned section left a black void of scroll; "too far" between sections. | No visible stretch of a section shows only ground for more than `limit` (64px), except the section's own padding at its edge or an `allow` with a written reason. Configure `surfaces` (class names of panels/cards/devices) so a filled panel is not read as empty. |
| `rest-flat` | The hero's 3D pair stayed in `translateZ`/perspective at rest and every word looked blurred. | Every element on each listed chain, up to `stopAt`, computes `transform: none`, `perspective: none`, no filter, no scale, `will-change: auto` after the scene settles, with JS and without. |
| `loops` | Scenes played once and were missed by whoever was reading the other column. | Each listed scene's state classes (`stateClass`) come back to their first state after other states within ~2 cycles; with `hover`, the state does not change while hovered. |
| `final-state` | Content born empty until JS ran; a scene's "before" text left on screen with JS off. | With reduced motion and with JS off, no text in `scope` is hidden except the listed `allowHidden` before-states; the first-screen elements sit in the first viewport; no horizontal scroll at any scroll position; no page errors. |
| `neutral-and-light` | A warm-tinted dark read brown; a glow layer passed in front of the phone ("behind glass"). | Every listed surface has saturation ≤ `maxSaturation`%; each light's z-index is below its objects; each listed object has opacity 1. |
| `clipped` | A window mock wider than its card on a phone cut the text on the right; page `scrollWidth` saw nothing because the card clipped it. | No text or image is partly inside and partly outside an `overflow: hidden/clip` ancestor (1px tolerance), in JS and reduced-motion runs, measured in drawn space so a scaled or tilted device is not misread. An ellipsis counts as an intended cut. |
| `fold-sampler` | A fold smooth on desktop was an instant swap on a phone ("it even startles"). | Progress read from `var` on `selector` runs 0→1 over ≥ `minRunway` px, never moves more than `maxJump` per `step` px, and reads the same scrolling down and up. |
| `contrast` | The light body of the reading pages inherited the dark floor's white ink: whole chapters invisible, and type-check, clipped and final-state passed. | Every visible text node's color against its effective (alpha-composited) ancestor background meets WCAG — 4.5, or 3 for ≥24px or ≥18.66px bold. Known gap: a background painted by a sibling element instead of an ancestor, such as a color layer growing behind a title, is not seen. |

## Reading a FAIL

- Fix the page, not the threshold. Raise an `allow` only with a reason the human agreed to, written in the
  config beside it (`allowWhy`).
- A failure at one viewport only is still a failure: the human reviews on tall screens (1211) and phones.
- Motion checks read the page's own progress variables and state classes; if the page has none, the effect
  is timed, not scroll-driven, and that is the first thing to fix.
- Run the whole suite after every change; a regression found by the human is a missing check — add it.
