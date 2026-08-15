# CLI Features — full-width pages, properties, smart links, templates, knowledge map

> **When to read:** when a user request touches one of the features below (full-width dashboards, page properties metadata, knowledge map regeneration, template generation, smart-link embeds, duplicate detection). The skill body keeps only the daily-use commands; details live here.

## Contents

- [Full-width pages](#full-width-pages)
- [Page Properties macro (`:::properties` block)](#page-properties-macro-properties-block)
- [Smart Link embeds](#smart-link-embeds)
- [`check` — duplicate detection](#check--duplicate-detection)
- [`new <type>` — template generator](#new-type--template-generator)
- [`map` — structural index of the space](#map--structural-index-of-the-space)

---

## Full-width pages

`page create` and `page upload` accept `--full-width` (and `--fixed-width` to revert):

```bash
# Create a full-width page
confluence-docs page create \
  --space-id <SPACE_ID> --parent-id <HOME_ID> \
  --title "Financial Dashboard" \
  --markdown content.md \
  --full-width

# Revert an existing page to fixed-width
confluence-docs page upload --page-id <id> --markdown content.md --fixed-width
```

Under the hood the CLI posts two properties (`content-appearance-draft` and `content-appearance-published`) to the page properties API after the create/update.

**Use full-width for:** dashboards, wide tables, anything where horizontal real estate matters. Fixed-width remains the Confluence default for prose pages.

## Page Properties macro (`:::properties` block)

In any markdown file, use the `:::properties` fenced block to generate the Confluence Page Properties macro:

```markdown
:::properties
type: reference
status: active
owner: user@example.com
tags: psp, billing, recurring
related: Stripe Brazil Analysis (212992001), [[Editorial Standards]]
created: 2026-01-01
updated: 2026-01-01
:::
```

Links in values use one of two forms:

- `Page Title (pageId)` — the preferred form for `related`. Rendered as a direct link to that page. The page id must have at least 6 digits, so `(2026)` and other short parentheticals stay plain text.
- `[[Page Title]]` / `[[id:N]]` — turned into `<ac:link><ri:page ri:content-title="..."/></ac:link>` storage XML.

**Do not quote values.** A single pair of wrapping quotes is stripped (`related: "A (123456)"` works), but quotes are never needed and per-item quoting (`"A", "B"`) renders literally.

The block is rendered as a `codeBlock` with language `confluence-storage` in ADF (so the storage XML passes through the pipeline). When creating pages via `page create --markdown`, Confluence accepts the storage XML inside the ADF body.

**Use properties for:** any page that should be discoverable in a Page Properties Report (KM index, reference catalogs, decision logs). Properties are how the Confluence KM macro aggregates pages.

## Smart Link embeds

In markdown, certain standalone URL patterns are automatically converted to Confluence Smart Link nodes:

| Markdown | ADF node | Renders as |
|---|---|---|
| `![embed](https://youtube.com/...)` | `embedCard` (wide layout) | Embedded player/preview |
| `https://linear.app/...` on its own line | `blockCard` | Preview card |
| `https://github.com/...` on its own line | `blockCard` | Preview card |
| `[text](url)` in the middle of a paragraph | normal link | Inline hyperlink |

Bare URL lines and `![embed](url)` trigger smart link conversion. Named links like `[Click here](url)` in prose always remain regular links.

## `check` — duplicate detection

**Always run `check` before creating a new page or running `new`.** Catches exact duplicates and near-duplicates before they clutter the space.

```bash
confluence-docs check --title "Stripe Brazil Analysis"
confluence-docs check --title "Stripe Brazil Analysis" --type reference --tags psp,competitor
confluence-docs check --title "..." --threshold 0.8   # stricter matching
confluence-docs check --title "..." --text            # plain-text output
```

JSON output:

```json
{
  "exists": false,
  "similar": [
    {"id": "456", "title": "Stripe Analysis (EN)", "url": "https://...", "similarity_score": 0.78}
  ],
  "suggestion": "create"
}
```

- `suggestion: "update_existing"` — a near-duplicate was found; consider updating it instead.
- `suggestion: "create"` — no close match; safe to create.

Uses trigram-based fuzzy matching (Jaccard similarity, threshold 0.7 by default). Accepts `--tags` to filter by Confluence labels and `--threshold` to tighten or loosen the match.

## `new <type>` — template generator

Generates a markdown template with a `:::properties` block and type-specific headings. Output goes to stdout or `--output FILE`.

```bash
# Reference doc
confluence-docs new reference \
  --title "PaymentCo — PSP reference" \
  --output /tmp/page.md

# Decision record (supersedes another page)
confluence-docs new decision \
  --title "PSP Wave 1: PaymentCo" \
  --supersedes 98765

# How-to guide
confluence-docs new how-to --title "How to deploy to production"

# Quick capture
confluence-docs new capture --title "Spike PaymentCo webhooks 2026-05-12"
```

Owner is read from `git config user.email`. Template includes `status: draft`, today's date for `created`/`updated`, and structured headings appropriate to the doc type.

For `decision` type, the template also includes: Alternatives Considered (table), Consequences, and Review date.

## `map` — structural index of the space

Answers "what is in here and where" without reading pages. The index is built
from the REST API — walking the page tree and reading the `type`/`status` the
pages already carry — so producing it costs **no model tokens**. It is cached
locally and read in slices, so a large space never enters the context at once.

```bash
confluence-docs map --refresh          # rebuild the cache from the API
confluence-docs map --depth 2          # top two levels of the tree
confluence-docs map --find "checkout"  # matching branches, ancestors kept
confluence-docs map --children 164232  # one level under a page
confluence-docs map --type decision    # only pages of that type
confluence-docs map --stale 90         # untouched for 90 days
confluence-docs map --status           # cache age and page count
confluence-docs map --json             # machine-readable
```

Output is one line per page, indented by depth:

```
164232  Home
  200441858  KNOWLEDGE_MAP — Mapa do Conhecimento  [reference]
  185303042  About the project  [reference]
```

The cache lives at `~/.cache/confluence-docs/map-<space>.tsv` and refreshes
itself when older than an hour; `--no-refresh` fails instead of fetching.
`--root ID` walks from somewhere other than the configured Home and is cached
separately, so a subtree never replaces the space-wide index.

**Metadata coverage.** The tree is always complete. `type`, `status` and the
last-modified date come from one bulk search, so they are known only for pages
that carry a `:::properties` block and only within the first 250 results.
`map --status` reports the coverage, and a filter run against mostly-unknown
metadata says so rather than looking like "nothing matches".

