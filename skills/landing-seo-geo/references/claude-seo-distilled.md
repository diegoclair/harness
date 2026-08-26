# claude-seo (v2.2.4) — what to reuse in a landing/small-site skill

Source: `~/.claude/plugins/marketplaces/agricidaniel-claude-seo/skills/`. Paths below are relative to `skills/` (or `scripts/` in section 3). Only what is there; nothing invented.

## 1. Objectively verifiable checklists

**Title / description / headings / URL** — `seo-page/SKILL.md`, `seo/references/quality-gates.md`
- Title: 30–60 chars, keyword at the start, brand at the end, unique per page.
- Meta description: 120–160 chars (Google truncates at ~155–160), CTA, natural keyword, unique.
- H1: exactly one; H2–H6 without skipping levels. Keyword in title, H1 and the first 100 words.
- URL: short, hyphens, lowercase, no query params; flag >100 chars; consistent trailing slash; redirects without chains (max 1 hop) — `seo-technical/SKILL.md`.
- Keyword density 1–3%, paragraphs of 2–4 sentences, sentences of 15–20 words — `seo-content/SKILL.md`.

**Canonical / robots / render** — `seo-technical/SKILL.md`
- Canonical self-referencing, absolute, identical between served HTML and post-JS (Google may use either).
- `noindex` in the initial HTML counts even if JS removes it; Google does not render JS on status ≠ 200.
- Title, description, canonical, meta robots and JSON-LD must be in the server HTML (SSR/static), not injected by JS.
- Googlebot reads only the first 2 MB of HTML: critical content + JSON-LD within that limit.
- Important pages ≤3 clicks from the home; no orphan pages.

**Hreflang** (only with 2+ languages) — `seo-hreflang/SKILL.md`
- Self-referencing required (without it Google ignores the whole set); return tags on all versions; a single `x-default`.
- Codes: ISO 639-1 (`pt`, not `por`) + optional ISO 3166-1 alpha-2 (`pt-BR`, `en-GB`; `en-uk`/`EU` invalid).
- Only on canonical URLs; hreflang URL = exact canonical (including trailing slash).

**JSON-LD per page type** — `seo/references/schema-types.md`, `seo-plan/assets/saas.md`, `seo-schema/SKILL.md`
- Home: Organization (name, url, logo, sameAs, contactPoint) + WebSite + SoftwareApplication/WebApplication.
- Product/features/pricing: SoftwareApplication (applicationCategory, operatingSystem, offers) or WebApplication (featureList, browserRequirements) + Offer (price, priceCurrency, availability).
- Blog: Article/BlogPosting (headline, author→Person, datePublished, dateModified, image, publisher). Authors: Person + ProfilePage. Navigation: BreadcrumbList.
- Comparison/roundup: ItemList; Product+AggregateRating only with real reviews.
- Validation: `@context` https, non-deprecated type, required props, no placeholders, absolute URLs, ISO 8601 dates.
- **Do not use**: HowTo (dead Sep 2023), FAQPage (rich result withdrawn 7 May 2026 — existing stays as Info, never add new ones for SERP; QAPage only for real user Q&A), SpecialAnnouncement, ClaimReview, VehicleListing, CourseInfo, EstimatedSalary, LearningVideo — `seo-schema/references/deprecated-types-2024-2026.md`.
- SearchAction on WebSite no longer produces a sitelinks search box.

**Core Web Vitals** — `seo/references/cwv-thresholds.md`
- LCP ≤2.5s (poor >4.0s) · INP ≤200ms (poor >500ms) · CLS ≤0.1 (poor >0.25). p75 of field data (CrUX); never cite FID.
- LCP = TTFB (<800ms) + resource load delay + load time + render delay. Ranking uses field; lab (Lighthouse) is for debugging.
- There is no "CWV 2.0", "VSI", LCP 2.0s — blog myth.
- Typical causes: hero without preload/WebP, render-blocking CSS/JS, font without `font-display: swap`, img without width/height, DOM >1,500 nodes, tasks >50ms.

**Mobile / page experience** — `seo-technical/SKILL.md`
- Viewport meta; touch target ≥48×48 px with 8 px spacing; base font ≥16 px; no horizontal scroll.
- Mobile/desktop parity (content, meta, schema); key content visible on load (not behind tab/accordion); no full-page interstitial.
- Only CWV is a direct ranking signal; HTTPS is light; security headers (CSP, HSTS, X-Frame-Options, X-Content-Type-Options, Referrer-Policy) are hygiene, not ranking. Back-button hijacking (`pushState`) = Critical since Jun 2026.

**robots.txt for AI crawlers** — `seo-geo/SKILL.md`, `seo-technical/SKILL.md`
- Allow for AI search visibility: `GPTBot`, `OAI-SearchBot`, `ClaudeBot`, `PerplexityBot`. Optionally block training: `CCBot`, `anthropic-ai`, `Bytespider`, `cohere-ai`, `Google-Extended` (blocking does NOT affect Search/AI Overviews, which use Googlebot).
- Ignore robots.txt by design (user-triggered): `ChatGPT-User`, `Google-Agent`, `Google-NotebookLM`, `Google Messages`.
- There is no specific AI Overviews opt-out: it is controlled with `nosnippet`, `data-nosnippet`, `max-snippet`, `noindex`.
- AI crawlers **do not execute JS** → SSR/static is mandatory.

**llms.txt** — `seo-geo/references/llmstxt-evidence.md`
- Google Search ignores it (official doc 29 Jun 2026); Mueller: "dead end"; only 1 of the 50 most-cited domains has one; 0.1% of bot traffic. Report presence, weight zero. Worth it for dev docs (coding agents consume them). Format: `# Title`, `> description`, `## section` with `- [Page](url): description`.

**Citable passage (GEO)** — `seo-geo/SKILL.md`
- Block of 134–167 words, self-contained (extractable without context), direct answer in the first 40–60 words of the section, "X is…" definition, fact/statistic with a source, unique data.
- ~44% of citations come from the first 30% of the page → front-load. Headings in question form; table for comparison; list for steps.
- Recency: <3 months ≈ 3× more cited; >6 months without update loses eligibility. Publication and update dates visible.
- Prerequisite: indexed and snippet-eligible (there is no separate "AI index") — `seo-geo/references/google-ai-optimization-guide.md`.

**E-E-A-T** — `seo/references/eeat-framework.md`, `seo-content/SKILL.md`
- Who/How/Why test (visible byline; how it was made, especially with AI; exists to help, not to rank).
- Trust (internal weight 30): contact, address, privacy/terms, HTTPS, testimonials, dates, corrections. Expertise/Authority (25 each): credentials, sources, external citations, mentions. Experience (20): own data, real screenshots, before/after, real case.
- Markers of weak AI content: generic sentences, no insight, structure repeated across pages, no author, factual error.

**Thin content** — `seo/references/quality-gates.md`, `seo-programmatic/SKILL.md`
- Floors (coverage, not a target; word count is not ranking): home 500, landing 600, feature/service 800, about 400, blog 1,500, comparison 1,500 (`seo-competitor-pages/SKILL.md`).
- Pages in series: <40% unique content = thin (hard stop <30%); test "would it be worth publishing if no other like it existed?"; publish in batches of 50–100 and observe 2–4 weeks.

**Internal linking** — `seo/references/quality-gates.md`, `seo-content/SKILL.md`
- 3–5 internal links per 1,000 words; long blog 5–10; service page 3–5. Descriptive and varied anchors; zero orphan pages; breadcrumb.

**Sitemap** — `seo-sitemap/SKILL.md`
- ≤50,000 URLs and ≤50 MB per file; only 200, canonical, indexable, HTTPS URLs; no redirected ones.
- `<lastmod>` W3C Datetime reflecting a real content change (uniform = Google ignores it); `<priority>`/`<changefreq>` ignored; reference it in robots.txt.

**OG / Twitter** — `seo-page/SKILL.md`
- `og:title`, `og:description`, `og:image`, `og:url`; `twitter:card`, `twitter:title`, `twitter:description`. OG removal = WARNING in drift.

**Images** — `seo-page/SKILL.md`, `seo-images/SKILL.md`, `seo/references/quality-gates.md`
- Alt on every non-decorative image, 10–125 chars, describes the content (decorative: `alt=""`).
- Size: content >200 KB warning, >500 KB critical; hero >300 KB warning, >700 KB critical; thumbnail >100 KB warning.
- WebP/AVIF with `<picture>` fallback; `width`/`height` always (CLS); `srcset`/`sizes`; `loading="lazy"` below the fold (never on the LCP).

## 2. Reusable methodologies

**Audit: order and scoring** — `seo/SKILL.md`, `seo-audit/SKILL.md`, `seo/references/thinking-framework.md`
- Order: (1) render the home with headless (raw vs rendered, detect SPA) → (2) detect business type (SaaS: /pricing, /features, "free trial") → (3) crawl (≤500 pages, respects robots, 5 competitors, 1s delay) → (4) specialists in parallel: technical, content, schema, sitemap, performance, visual, geo, sxo (always); drift only if a baseline exists → (5) score → (6) plan.
- Health Score: Technical 22% · Content 23% · On-page 20% · Schema 10% · CWV 10% · AI readiness 10% · Images 5%. The score is a heuristic, not a Google signal — say so in the report.
- Priority: Critical (blocks indexing/penalty, now) · High (ranking, 1 week) · Medium (1 month) · Low (backlog). Page score: On-page, Content, Technical, Schema, Images, each 0–100.
- PERCEIVE→ANALYZE→VALIDATE→ACT synthesis: collect without scoring; find the **highest-leverage constraint** (not indexed, LCP, canonical) and put it first; the plan is a dependency graph ("what unblocks the most?"); every recommendation carries "how would we know it failed?" and a leading indicator; close with a drift baseline and what the next audit should look at. If the SERP contradicts the recommendation, the SERP wins.

**SERP backwards / page-type mismatch (SXO)** — `seo-sxo/SKILL.md`, `seo-sxo/references/page-type-taxonomy.md`, `user-story-framework.md`, `persona-scoring.md`
- A technically 95/100 page does not rank if it is the **wrong type** for the keyword. Pipeline: keyword (title ∩ H1) → Google top 10: classify each result into 8 types (Landing, Blog, Product, Hybrid, Service, Comparison, Local, Tool) → consensus (>60% strong, 40–60% mixed, <40% fragmented = chance to differentiate) → compare with the page.
- Severity: Blog where the SERP wants Product/Tool = CRITICAL; Blog where it wants Comparison = HIGH; Landing where it wants Tool = HIGH; Product where it wants informational = HIGH.
- Landing page (taxonomy): hero + 1 proposition, CTA above the fold, social proof, pricing, WebSite/SoftwareApplication schema; SERP signals: brand query, many ads, sitelinks to /pricing.
- Gap score 0–100 across 7 dimensions (type, depth, UX, schema, media, authority 0–15 each; freshness 0–10), separate from the Health Score.
- User stories only from observable signals: PAA (knowledge gap: definitional=awareness, evaluative=consideration, comparative=decision), ad copy (commercial trigger and objection: "free trial"=fear of commitment, "trusted by"=trust, price=cost), related searches (journey; "alternatives"=dissatisfaction), snippet format (paragraph/list/table), AI Overview (sources Google trusts). Format: "As [persona], I want [goal], because [driver], but I get stuck on [barrier]".
- Personas 4–7, each traceable to a signal; score 4×25 (Relevance, Clarity "finds it in 10 s?", Trust, Action); attack the weakest persona with the highest volume first.

**"X vs Y" and "alternatives to X" pages** — `seo-competitor-pages/SKILL.md`, `seo-plan/assets/saas.md`
- Conversion 4–7% vs 0.5–1.8% for blog. URLs: `/{product}-vs-{competitor}`, `/{competitor}-alternative`, `/compare/{category}`, `/best-{category}-tools`.
- Title: vs = `[A] vs [B]: [Differentiator] ([Year])`; alternatives = `[N] Best [A] Alternatives in [Year] (Free & Paid)`; roundup = `[N] Best [Cat] Tools in [Year], Compared & Ranked`. H1 <70 chars.
- Sections (comparison): intro → criteria → feature matrix (✅/⚠️/❌ + "from" price + free tier, with "as of [date]") → individual review → verdict/"best for" → visible FAQ (no FAQPage). Alternatives: each item with summary, pros/cons, best-for. ≥1,500 words.
- CTA: above the fold (summary + CTA), after the table, at the end; **never** inside the competitor's description. Social proof: testimonial from someone who migrated, G2/Capterra ratings with link.
- Trust: "last updated", author, methodology, declare which product is yours; only verifiable claims; review quarterly. Schema: SoftwareApplication (+Offer) per product; ItemList on roundup; AggregateRating only with real reviews.
- Internal linking: cross-link A-vs-B ↔ A-vs-C, cited features → feature pages, breadcrumb Home > Comparisons > X.

**Prioritization in seo-plan** — `seo-plan/SKILL.md`, `seo-plan/assets/saas.md`
- There is no explicit impact×effort matrix; prioritization is (a) the audit's Critical/High/Medium/Low buckets and (b) a 4-phase roadmap: Foundation (weeks 1–4: technical, home/about/contact/product, schema, analytics) → Expansion (5–12: primary content, blog, linking) → Scale (13–24: GEO, performance, links) → Authority (months 7–12: PR, mentions). "Effort estimates" only appear in the `google_report.py` PDF.
- SaaS: high priority = home, features, pricing, integrations, 3–5 use cases; medium = individual feature, cases, comparison. BOFU (comparison, ROI) before TOFU. KPIs: organic → pricing, organic signups, comparison ranking.

**Drift: baseline/diff** — `seo-drift/SKILL.md`, `seo-drift/references/comparison-rules.md`
- Baseline per URL (normalized: lowercase, no default port/UTM/trailing slash) with title, description, canonical, meta robots, h1/h2/h3, JSON-LD, OG, CWV, status, SHA-256 hash of the HTML and of the schema; local SQLite.
- 17 rules. CRITICAL: schema gone; canonical changed/gone; noindex added; H1 gone; H1 with similarity <0.5 (SequenceMatcher); title gone; status 2xx→4xx/5xx. WARNING: title/description changed; CWV p75 worsened >20%; Lighthouse −10 pts; OG gone; schema_hash changed. INFO: new schema; H2s changed; html_hash changed.
- Usage: baseline before deploy → compare after; when investigating a drop, compare + history.

**GEO per platform and myths** — `seo-geo/SKILL.md`, `seo-geo/references/google-ai-optimization-guide.md`
- **AI Overviews**: strongly correlated with ranking (92% of citations come from the top 10) → classic SEO + citable passages. **AI Mode** (custom Gemini 2.5): wider pool (~9 domains/query), cite the same URLs only 13.7% of the time → recency and entity weigh more than position; treat as a separate surface. Preferred Sources: ask the audience to add the brand.
- **ChatGPT**: Wikipedia 47.9%, Reddit 11.3% → entity presence. **Perplexity**: Reddit 46.7%, Wikipedia → community validation. **Copilot**: Bing index + IndexNow. **Gemini**: only via `Google-Extended` (training/grounding); Search/AIO are independent.
- Brand mentions correlate 3× more than backlinks (YouTube 0.737, Reddit high, DR 0.266); only 11% of domains are cited by ChatGPT and AIO on the same query.
- **Myths rejected by Google**: llms.txt/"for AI" files; "chunking"; rewriting with "for AI" phrasing; chasing inauthentic mentions; extra schema "for AI". "AEO/GEO is SEO". What counts: first-hand, non-commodity content.

## 3. Scripts worth copying/rewriting (in `scripts/`)

- `parse_html.py` — extracts title, description, canonical, robots, h1–h3, JSON-LD, OG, images (alt, dims, lazy). Deps: bs4, lxml. The base of everything.
- `fetch_page.py` / `url_safety.py` — fetch with UA and SSRF/DNS-pinning. Dep: requests. Copy only the URL validation.
- `render_page.py` — Playwright + trafilatura (text without boilerplate) + htmldate; compares raw vs rendered to detect SPA; `--a11y-tree`. Deps: playwright, trafilatura, htmldate.
- `pagespeed_check.py` — PSI v5 + CrUX in one output; `lcp_subparts.py` — decomposes LCP via CrUX; `crux_history.py` — 25 weeks. Dep: Google API key.
- `preload_check.py` — detects preload/Speculation Rules/bfcache in static HTML (no browser).
- `schema_generate.py` — generates JSON-LD for the main types; `schema/templates.json` at the plugin root.
- `drift_baseline.py` / `drift_compare.py` / `drift_history.py` / `drift_report.py` — SQLite snapshot + 17 rules + HTML. Rewrite lightly (JSON in repo, runs in CI post-deploy).
- `sitemap_discovery.py` — robots.txt `Sitemap:` + common paths, validates XML. Deps: requests, lxml.
- `content_quality.py` — QRG heuristics (typical LLM phrases, Wikipedia AI Cleanup list); `content_verify.py` — claims without citation; `content_humanize.py` — deterministic swap of AI clichés. Stdlib.
- `agent_ux_check.py` — 0–100 "agent-friendly" score (real buttons, labels, a11y tree). Dep: Playwright.
- `indexnow_submit.py` — IndexNow POST (Bing/Yandex; Google does not use it). Stdlib.
- `unlighthouse_run.py` — Unlighthouse wrapper (Lighthouse across the whole site). Dep: Node 18+.
- `capture_screenshot.py` / `analyze_visual.py` — desktop/mobile and above-the-fold screenshot. Dep: Playwright.

## 4. What NOT to bring

- **Local/Maps** (`seo-local`, `seo-maps`, `seo/references/local-*`, `maps-*`, `gbp_deprecation_lint.py`): GBP, NAP, geo-grid — there is no physical store.
- **E-commerce** (`seo-ecommerce`, `schema_ecommerce_validate.py`, `dataforseo_merchant.py`, UCP, Product/MerchantReturnPolicy, IPTC on product images): Rednev sells SaaS, not its own catalog.
- **DataForSEO / Ahrefs / SE Ranking / Profound / Firecrawl** (`seo-dataforseo`, `extensions/`, `dataforseo_*.py`): paid; replace with WebSearch + GSC.
- **Backlinks** (`seo-backlinks`, `moz_api.py`, `commoncrawl_graph.py`, `verify_backlinks.py`): for a small site the plugin itself says brand mentions weigh 3× more; skip.
- **seo-cluster / seo-content-brief / seo-flow**: volume content pipeline and 41 generic prompts; only when there is a real blog.
- **seo-programmatic** entirely (keep only the 40%/30% uniqueness rule and the "would it stand alone?" test).
- **seo-hreflang** in full: retain only the 5 points from section 1 until a 2nd language exists.
- **Runtime** (`runtime.py`, `bin/claude-seo`, isolated venv, `consistency_check.py`, `release_sign.py`, `google_report.py` PDF/weasyprint/matplotlib): plugin infrastructure, not knowledge.
- **Verbosity**: community footer, "Error Handling" tables repeated in every skill, 2026 I/O statistics without a Google source, agentic browsing/WebMCP (origin trial, "opportunity, not defect"), RSL 1.0, Soft Navigations API — none of it changes a decision on a landing page today.
- **Numeric E-E-A-T and Health Score weights**: they are the plugin's internal model (it says so itself); keep only the order "Trust > Expertise/Authority > Experience".
