# Changelog

All notable changes to `social-carousel` will be documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.3.0] - 2026-07-24

### Added

- **`reference/copy-review.md`** — 7-check adversarial copy audit distilled from two production carousels that scored 6.5/10 with reviewers, where every defect was editorial (never structural, the linter caught nothing). Checks: slide-3 novelty vs neighbors, arming the strongest trigger early, self-sufficient CTA referents, a read-aloud/cringe test (influencer-speak and press-release voice), PT-BR register + phrase list + hook-formula adaptation notes, the big-number "1" glyph pitfall, and a binary-question caption hook. Includes guidance for dispatching an adversarial-reviewer subagent before shipping.
- **Linter rule ST-06 (slide-3 redundancy, WARN)** — flags when slide 3 shares ≥60% of its content words (stopwords stripped) with slide 2 or slide 4, the mechanical proxy for "value bomb reworded a neighbor instead of introducing new information."
- **Linter rule CR-01 (stock influencer phrase, WARN)** — substring match against a short PT/EN cliché list ("te leio", "salva esse post" bare, "smash that follow button", etc.).
- **`reference/layouts.md`** — pitfall note under `big-number`: a lone numeral "1" in a geometric sans (Outfit) can misread as a lowercase "l" at 200–300px; real production failure on a "1 link" slide.

### Changed

- **SKILL.md** — `reference/copy-review.md` is now a mandatory read alongside `examples.md` before writing slide copy; the 4 taste questions gained 3 more (slide-3 novelty, read-aloud/cringe, self-sufficient CTA); the language rule now points PT-BR carousels at copy-review.md's register + phrase-list section.

## [0.2.1] - 2026-07-09

### Fixed

- **LinkedIn PDF page size** — the combined PDF now renders at the platform's logical size (1080×1350 pt for 4:5) with the 2× retina image embedded, instead of a 2160×2700 pt page. pdfcpu v0.12.1's `ImportImagesFile` sizes each page to the source image's pixel dimensions and ignores the import `PageDim`, producing a ~76×95 cm page that LinkedIn re-framed by stacking two slides per card. `combinePDF` now imports then `Resize`s each page to the logical dimensions.
- **Invisible text on `clarity` / `spotlight` tone slides** — layouts paint `color: var(--fg-primary)`, but the tone scopes redefined `--fg-primary` with a self-referential fallback that resolved to black, rendering text invisibly on the tone's own background. Added immutable `--theme-fg`/`--theme-bg` anchors for the tone scopes to fall back to; contrast is now 14–17:1.

### Changed

- **Layout hierarchy + brand cohesion** — `cover`/`text`/`list`/`quote`/`cta` gained an optional mint eyebrow (`label`) and supporting line (`sub`), so intermediate slides carry the same hierarchy as the cover instead of flat centered text. `text` is now left-aligned with a mint anchor bar; the `list` title is set dominant over its items with accent-tinted dividers; the `cover` eyebrow is mint (was muted grey) for consistency. `text` supports three tiers: eyebrow → body → sub.
- **`clarity` tone is on-brand by default** — renders on the theme's secondary-dark surface rather than a flat white card, so a clarity slide between dark slides no longer reads as a second design system. Dot texture derives from `--fg-primary` so it stays visible on any tone.

### Added

- **Theme `bg_clarity` / `fg_clarity`** — optional light-variant palette for the `clarity` tone. When set, clarity slides render on a deliberate light background (kept coherent via eyebrow + accent + left-align), restoring the dark→light→dark rhythm without the generic white-card look. Leave empty to keep clarity on the dark secondary surface.

## [0.2.0] - 2026-05-19

### Added

- **Project-local config** — `carousel.config.yml` in the project root provides defaults for `theme`, `handle`, and `platform`. The CLI walks up from the carousel YAML's directory looking for the file (stops at a `.git/` boundary or after 10 levels — matches prettier / editorconfig semantics). Carousel YAML fields ALWAYS win over project config; it only fills blanks. Picked up automatically by `check` and `render`.
- **Taste memory** — accumulated design preferences the agent reads before generating, persisted as markdown with YAML frontmatter. Two layers, project wins on overlap:
  - Global: `~/.claude/skills/social-carousel/memory/taste.md` (cross-project taste)
  - Project: `<project>/carousel-design.md` (versionable, brand-specific)
  - Rule fields: `text`, `scope` (global / brand:slug), `confidence` (high / low), `captured` (YYYY-MM-DD).
  - Low-confidence rules auto-expire after 30 days on next render (silent housekeeping).
- **`social-carousel paths [--cwd DIR]`** — single new command, the only one added for the v0.2.0 surface. Prints all filesystem locations (taste files, project config, themes dir) cross-OS. Designed for the AI agent: it doesn't need to know where macOS / Linux / Windows put config — the CLI resolves and prints, the agent reads/writes the files with its native Read/Write tools. No CRUD wrappers (`taste show/add`, `config init/resolve`) — those would be surface area for nothing since the agent already has file IO.
- **Cross-platform global config dir** — replaced the v0.1.x XDG-hardcoded `~/.config/social-carousel/` with `os.UserConfigDir()` so macOS gets `~/Library/Application Support/social-carousel/` and Windows gets `%AppData%\social-carousel\` per platform convention. Legacy XDG path stays readable as a Linux fallback so any v0.1.x user themes are preserved.
- **SKILL.md Step 0a — Resolve paths + read taste memory** documents the new layer and explicitly tells the agent: "use `social-carousel paths` to resolve locations, then Read/Write the files directly. Echo-back before persisting a rule. Default scope=global / confidence=high. Slide-specific corrections are NOT preferences — don't save them."

### Changed

- `customThemeDir()` in `theme.go` now resolves via the new cross-SO `GlobalThemesDir()` helper, with the legacy XDG path as a Linux-only fallback.
- `render` now performs a silent best-effort prune of expired low-confidence taste rules (30-day cutoff). Failure to prune does not block render.

## [0.1.1] - 2026-05-19

### Changed (breaking — first 24h after v0.1.0)
- **Example theme rename**: `dark-tech` → `example-dark-tech`, `light-editorial` → `example-light-editorial`, `minimal-mono` → `example-minimal-mono`, `duotone-deep` → `example-duotone-deep`, `neo-brutalist` → `example-neo-brutalist`. The `example-` prefix signals "starting point, not destination": users are expected to author brand-specific themes via `theme create`. Carousels with `theme: dark-tech` etc. must be updated to the new names.
- **Comparison layout redesign**: removed full-color background fills (red/green) that created a striped slate→wine→green→slate effect fighting the theme palette. New rendering uses uniform slide canvas + thin 4px accent stripes on top of each side (coral for BEFORE, theme accent for AFTER) + icons/typography as the semantic load. Reference: Broekema brand-locked frames + Alić oversized hierarchy.
- **Spotlight tone on `text` layout**: now applies a pull-quote treatment with oversized decorative quotation marks (white at 14% opacity, 480px) on opposite corners + larger font tier (poster 168px / medium 120px / long 84px). Replaces the previous flat-fill treatment that left text floating with no visual anchor.
- **Linter rule R1 (comparison labels present) removed**: superseded by **CM-05** which checks the same condition with a more actionable message that explains the visual failure mode (✗/✓ icons need context).

### Added
- **SKILL.md Step 0 — Brand check** before scaffolding the first carousel of a session. The agent MUST ask the user about visual identity (colors, fonts, logo) and either author a custom theme block, use partial info with explicit defaults, or pick an example theme while warning the user it's an on-ramp. Prevents every carousel made with this skill from defaulting to the same example theme.
- **SKILL.md Cross-cutting visual patterns section** — 6 patterns the linter cannot catch but every top reference in `examples.md` obeys (one idea per slide, single accent color, slide-3 surprise, numeric promise on cover, 8–12 slides, cover hook readable at thumbnail).
- **SKILL.md Gate 2 — visual QA after render** — 5-item checklist for things only the rendered pixels reveal (overflow/clipping, font fallback, contrast in practice, cross-slide consistency, whitespace balance). Default inline by parent agent; subagent dispatch on `--strict` or ≥10 slides. Binary FIX/NOTE severity with grounded-evidence requirement.
- **Linter rule SP-01** — spotlight tone overuse: WARN at 2 spotlight slides, ERR at 3+. Spotlight is an interstitial role and dilutes with repetition.
- **Linter rule SP-02** — body word count on `text` + `tone: spotlight`: WARN past 12 words. Beyond that the pull-quote treatment looks cramped.
- **Linter rule CM-04** — comparison item-count parity: WARN when `|before_items - after_items| > 1`. Lopsided columns make the eye latch onto the extra row instead of the contrast.
- **Linter rule CM-05** — comparison labels required (ERR). Replaces R1 with an actionable message tied to the visual failure mode.
- **`reference/examples.md` reference Broekema** (Nick Broekema, ugcPost-7237051775151673345, 906 likes / 379 comments verified) replaced the prior Fishkin reference, which was a text post rather than a verified carousel. Cross-checked engagement and document format via the LinkedIn `ugcPost-` token.

### Fixed
- Stale `cream-lifestyle` reference in scaffold templates (no such theme ever shipped) replaced with `example-light-editorial`.
- Stale doc comment in `types.go` listing presets incorrectly.

## [0.1.0] - 2026-05-18

Initial release.

### Added
- CLI binary `social-carousel` with subcommands: `render`, `check`, `new`, `preview`, `theme`, `setup`, `update`.
- YAML schema for carousels with 8 slide layouts: `cover`, `list`, `big-number`, `quote`, `comparison`, `screenshot`, `cta`, `text`.
- 3 platform targets: `instagram-4x5` (1080×1350 PNG), `instagram-1x1` (1080×1080 PNG), `linkedin-4x5` (1080×1350 PDF).
- Renderer via `chromedp` (headless Chrome, Go-native) at 2× device scale (final output 2160×2700).
- PDF combiner via `pdfcpu`.
- 5 design presets: `dark-tech`, `light-editorial`, `minimal-mono`, `neo-brutalist`, `duotone-deep` (renamed with `example-` prefix in 0.1.1 — see above).
- `theme create` to author custom themes from an iterated carousel YAML; persisted to `~/.config/social-carousel/themes/`.
- 6 scaffold kinds in `new`: `listicle`, `case-study`, `framework`, `comparison`, `story`, `data-drop` — each lint-clean by construction.
- Linter (`check`) with 27 rules across universals, anti-patterns, and per-layout constraints. Blocks render unless `--force`.
- `Slide.tone` (optional) — per-slide visual tone override (`authority` / `clarity` / `spotlight`) for visual-rhythm rotation across a 10-slide deck. Slides without `tone` inherit the theme.
- `Slide.hook_style` (optional, cover only) — `gradient` opts in to the linear accent → accent_alt hook treatment. Default is solid.
- Linter rule **RH-01** (warning) — flags any run of 5+ consecutive slides sharing the same `layout` or `tone`.
- `data-tone` attribute on the slide root + base.css tone overrides (bg/fg + accent-derived primitives).
- Embedded font assets pipeline: Outfit, DM Sans, Playfair Display, Space Grotesk, Inter, Noto Color Emoji (via `make fonts`).
- `preview` HTTP server for quick visual iteration without `chromedp`.
- Self-update via GitHub API filtered by `carousel-v*` tag prefix.
- Installer stub reusing `pkg/install/install.sh` shared across the monorepo.
