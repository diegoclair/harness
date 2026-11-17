---
paths:
  - "**/*.{tsx,jsx,ts,js,css,scss,html,vue,svelte,mdx}"
---
# A UI change is proven in the browser

- **Type-check and lint prove nothing about a screen.** Changed layout, color, height, column, focus or keyboard? Open it in Playwright, screenshot at **390×844 and 1440×900**, and look before saying it is done.
- **Don't wait for the real backend**: run the front against a stand-in of the API and inject the session into `localStorage` with `addInitScript`. How each front does it is in that front's `AGENTS.md`.
- **Measure, don't only look**: `boundingBox()` for row and modal height, `isVisible()` per column per breakpoint, `scrollWidth > clientWidth` for horizontal scroll.
- **Open what is collapsed** (details, accordion, sheet) and measure inside; a "closed/invisible" warning is never dismissed as an artifact.
- **Switching a tab, option or state never changes the screen's height**: prove it with `boundingBox()` across states.
- **A visual fix is the smallest change that answers the complaint**: list what does not change; touching layout nobody criticized needs a question first.
- Stubs, scripts and screenshots live in the scratchpad, never in the repo; stop the processes when done. A screenshot handed to the user is first copied out of the scratchpad to a durable folder in the user's home and given as a clickable markdown link with the absolute path.
