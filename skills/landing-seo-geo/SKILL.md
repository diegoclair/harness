---
name: landing-seo-geo
description: Makes a landing page or small product site (static Next.js, Cloudflare) rank on Google/Bing AND get cited by ChatGPT, Gemini, Perplexity, Copilot and AI Overviews (GEO). Covers audit, metadata/JSON-LD/robots/sitemap/llms.txt, answer-first content pages (per search term, entity, comparison), a source rule with a link on every claim, publishing and indexing (Search Console, Bing Webmaster importing from Google, IndexNow via Cloudflare Crawler Hints) and a measurement panel. Use it EVERY time Diego mentions SEO, GEO, "aparecer no Google", "aparecer no ChatGPT/Gemini", "indexação", "sitemap", "Search Console", "Bing", "IndexNow", schema/"JSON-LD", "llms.txt", a "página X vs Y", a "página por marketplace/segmento", or wants "anyone who searches for X to find the brand", even without saying "SEO". Replies match the user's language.
---

# landing-seo-geo

This skill was born from a real round (Rednev, 26 Aug 2026) in which two specialist agents (SEO and
GEO) audited a landing page, applied the technical layer, generated three content pages, and the
page was published and indexed the same day. What is here is what **worked**, what **was a myth**
and the order that avoids rework. Each reference says where the lesson was learned.

## What the skill knows that changes the decision

- **Classic SEO is a prerequisite, not the whole game.** Google AI Overviews and Bing/Copilot read
  the search index; ChatGPT uses Bing. Without Search Console + Bing Webmaster nothing else gets measured.
- **What decides whether an assistant cites the brand is third-party presence**, not the brand's
  own site: ~68% of citations come from outside (listicles "best X", Reddit, G2, YouTube); a niche
  brand shows up in ~11% of answers versus 73% for a global brand. The site has to exist and be
  clear; the citation comes from the mentions plan. Details in `references/geo.md`.
- **A citable passage** is 1 to 3 sentences that answer the question on their own, with the entity
  named ("Rednev is..."). A page that only says "it"/"the AI" produces no extractable snippet. It is the most common mistake.
- **Myths that cost time:** `llms.txt` is not ranking (Google does not use it; logs show bots almost
  never fetching it); the FAQ rich result was discontinued by Google (May 2026); AI Overviews do not
  require structured data. Schema serves to disambiguate the entity and the price, not to rank.
  Publish `llms.txt` and JSON-LD because they cost zero, not expecting an effect.
- **One URL ranks for one topic.** The home covers the brand; "AI for X" needs a page whose
  title, h1 and first paragraph are about X. A page promising an integration that does not exist
  ("AI for Shopee" with Shopee under construction) does not rank and breaks the communication rule.
- **An indexed site can still get "I don't know" from an assistant, and the markup is not the
  cause.** The Gemini app answers from training data first and only grounds on Search above a
  relevance threshold; a fresh brand name that reads like a surname rarely crosses it. A
  re-registered domain keeps its *old* identity in the index for weeks (Google: "it takes time for
  the old state to be shaken off", nothing manual fixes it). Zero third-party mentions means
  nothing to corroborate. Diagnose in that order before touching schema. Details and the
  checks in `references/geo.md` § "Indexed but unknown".
- **Check the domain's previous life BEFORE launching on it** (RDAP registration date, Wayback,
  `site:` searches for old paths). Learned the hard way on a new brand: two weeks after launch Google was
  already clean, but the Bing-side index (ChatGPT, Copilot) still served the previous owner's
  affiliate site for the brand query. Check each index, not one search tool. Procedure in
  `references/publish-and-index.md` § "Domain pre-flight".
- **Every claim about a platform, competitor or market has a link and a read date.** Without a
  link, it leaves the page. The rule was imposed by Diego and caught a false sentence ("no hub
  publishes its sync interval") before going live. Details in `references/sources-and-competitors.md`.

## Process (in order; skipping a step is what creates rework)

### 0. Domain pre-flight (5 min, before anything else)

`references/publish-and-index.md` § "Domain pre-flight". A domain with a previous life carries it
into the index; knowing that on day 1 changes what "indexing time has passed" means.

### 1. Audit before touching anything

Run the static build and read the generated HTML, not the source code: it is what the crawler sees.

```bash
pnpm build && grep -o '<title>[^<]*' out/index.html
grep -o '<h[12][^>]*>[^<]*' out/index.html          # single h1? do h2s carry the search term?
grep -o 'application/ld+json' out/index.html | wc -l
grep -o 'name="description" content="[^"]*' out/index.html
ls out/robots.txt out/sitemap.xml out/llms.txt 2>&1
```

Score 0 to 100 with the checklist in `references/technical-checklist.md` and the GEO one in
`references/geo.md`, and write the diagnosis before applying. Two agents in parallel (SEO and
GEO) work well **if the file territories are disjoint**; the split that produced no conflicts is
in `references/technical-checklist.md` § "Two agents".

### 2. Apply the technical layer (it is the cheap part; do it all at once)

Title ≤60 with the term at the start, description ≤155, canonical, OG/Twitter, robots meta,
`robots.ts`/`sitemap.ts`/`manifest.ts` with `force-static`, JSON-LD `@graph` (Organization,
WebSite, SoftwareApplication with `offers` read from the pricing file; `aggregateRating` only when measured),
FAQPage generated from the same file as the visible FAQ, `llms.txt` with facts from the docs. Search
strings live in a data file (`data/seo.ts`), never in a component. Next.js export gotchas
(dynamic route in dev, `trailingSlash`, escaping `<` in JSON-LD) in `references/technical-checklist.md`.

### 3. Visible copy: propose, do not apply

Title, h1, h2 and FAQ are brand text and go through the product's communication rule. The agent
delivers **current → suggested → search term/question it answers**; the owner decides. What
usually passes: the entity named once per section, FAQ "What is X?", "Is X a hub/ERP/...?",
"Does X work on Y?". What usually gets blocked: promising an unmeasured result, focusing the brand on a
single segment/marketplace when the strategy is more than one.

### 4. Content pages (one per topic)

The pattern that scales: **page = data file + shared template + registry**; dynamic route,
metadata, JSON-LD per type, sitemap, footer and "read also" all derive from the registry. First
three pages, in this order: `/ia-para-<segmento>` (where the search volume is),
`/o-que-e-a-<marca>` (entity, for "what is X"), `/<marca>-vs-<categoria>` (comparison, the
most-cited format). Structure, answer-first lead and honest comparison table in
`references/content-pages.md`.

### 5. Sources: link everything, before going live

Set aside a session/agent just to re-read every claim **at the source** (curl with a browser UA on
pages that block bots), update the research docs with URL + date + literal excerpt and return a
table claim → where it appears → URL → status. Only then does the landing page get the links.
Mechanism in code: URLs in a single `data/sources.ts`, optional `sources` field on any
block/FAQ, `SourceLink` component with `rel="noopener noreferrer"`. Details and the rule for naming
a competitor in `references/sources-and-competitors.md`.

### 6. Publish and index (30 min, and it is what makes everything exist)

Deploy → confirm live (`curl` on title, JSON-LD, robots, sitemap, llms.txt, 404) → Search
Console (domain property, TXT in DNS, submit sitemap) → Bing Webmaster **importing from
Search Console** (one click, covers Copilot/ChatGPT/DuckDuckGo) → Cloudflare **Caching →
Configuration → Crawler Hints: On** (that is IndexNow, notifies Bing/Yandex on every deploy). Step by
step, what is confusing in the Bing UI and what to check under "Content Signals" in
`references/publish-and-index.md`.

### 7. Measure and follow the mentions plan

Manual biweekly panel (8 fixed queries × ChatGPT, Gemini, Perplexity, AI Mode, Copilot; private
window) recorded in `docs/measurements/`; Bing Webmaster "AI Performance" and Cloudflare logs
by user-agent as the crawler signal. A new brand's baseline is zero citations; the curve comes from the
third-party plan in `references/geo.md` § "Mentions".

## Rules that apply to every agent dispatched by this skill

- The git index belongs to the repo owner: no `git add`/`commit`/`reset`. Comments state the why, not the
  behavior. Tests do not shape production. Copy these rules into the subagent prompt.
- Content cannot depend on JS (AI crawlers do not execute it); verify in `out/`.
- No visible string in a component; no new design token (font, color, size) for a new
  page: reusing the site's tokens is what keeps "change in one place, changes everywhere".
- No number without a source goes live; `aggregateRating`, "X% increase", "leader", "best" stay out
  until a published measurement exists.
- Final report always with: diagnosis before (score), what was applied file by file, copy proposals
  not applied, claims to check against the source doc, external pending items by
  priority, typecheck/build/screenshot results.

## References

| File | When to read |
|---|---|
| `references/technical-checklist.md` | Audit and application on static Next.js: limits, JSON-LD, export gotchas, split between two agents |
| `references/geo.md` | What moves citation in each engine, myths, list of AI user-agents, citable passage, mentions plan, measurement panel |
| `references/content-pages.md` | Architecture page = data + template, answer-first lead, honest comparison, brand 404 |
| `references/sources-and-competitors.md` | The "no link, no go-live" rule, how to re-read sources that block bots, what CONAR allows when naming a competitor |
| `references/publish-and-index.md` | Cloudflare deploy, Search Console, Bing Webmaster (import, sitemaps, IndexNow, AI Performance, Site Scan), Crawler Hints |
| `references/claude-seo-distilled.md` | Checklists and methodologies reused from the claude-seo plugin (audit, SERP backwards, drift, competitor pages) |
