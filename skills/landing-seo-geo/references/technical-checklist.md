# Technical checklist (Next.js App Router, `output: "export"`)

Learned in the Rednev round of 26 Aug 2026. Score before: 38/100 with a 6-character `<title>` and
no canonical/robots/sitemap/JSON-LD; everything below took one afternoon.

## Limits and what each item does

| Item | Threshold / rule | Why |
|---|---|---|
| `<title>` | ≤60 chars, search term at the start, brand at the end or start | Biggest isolated CTR gain; "Brand" alone is zero search term |
| meta description | ≤155 chars, term + honest promise | Snippet; Bing uses it more than Google does |
| h1 | one per page, with the page's topic | One URL = one topic |
| h2 | one per section; at least 2 with a search term | h2s made only of brand phrases are good for conversion and weak for search |
| canonical | self-referencing, with or without trailing slash consistent with the export | Avoids `www`/root and `/x` vs `/x/` duplicates |
| hreflang | optional on a single-language site; `pt-BR` + `x-default` is harmless | Closes the door on duplicates in Search Console |
| robots meta | `index, follow`, `max-image-preview: large`, `max-snippet: -1` | Unlocks a large snippet for the engines |
| OG/Twitter | `og:url`, `og:site_name`, generated 1200×630 image, `alt` with the search title | Sharing and Bing |
| `robots.txt` | `User-Agent: *` / `Allow: /` + `Sitemap:`; permissive for AI crawlers | A new brand blocking GPTBot stays invisible |
| `sitemap.xml` | generated from the page registry; real `lastModified` | Only XML gets into Bing/GSC; a single page goes through URL Submission/IndexNow |
| manifest | name, design system colors | Hygiene; Bing reads it |
| JSON-LD | `@graph` with `@id`: Organization (logo, slogan, `sameAs` only with real profiles), WebSite, SoftwareApplication (`applicationCategory`, `operatingSystem`, `featureList`, `offers`/`AggregateOffer` read from the pricing file) | Disambiguates entity and price; **not** ranking |
| FAQPage | generated from the same file as the visible FAQ (zero string duplication) | Rich result is gone (May 2026); still counts as an entity/question-answer signal |
| BreadcrumbList + Article/AboutPage | per content page | Explicit page type |
| Core Web Vitals | LCP ≤2.5 s, INP ≤200 ms, CLS ≤0.1 | Gate; 115 kB of First Load JS is healthy |
| Content without JS | everything in the served HTML; reveal-on-scroll only hides after JS confirms it is alive | AI crawlers do not execute JS |
| Images | `alt` on all; inline base64 photo bloats the HTML (700 kB) and drops out of Google Images | A conscious decision, not an accident |
| `lang` | `pt-BR` on `<html>` | SERP language |
| Internal links | every content page linked from the footer and from at least one other page, with the search term as anchor text | Anchor text is a relevance signal the page itself cannot send |

## `output: "export"` gotchas (they cost hours)

- **Dynamic route `app/[slug]/page.tsx`**: needs `generateStaticParams`. Do **not** set
  `dynamicParams = false`: with it the Next 14.2 dev server throws "missing exported function
  generateStaticParams()" (a misleading message) on every route, including valid ones, and caches the error
  until `next dev` is restarted. In production the export ignores the flag. Even without the flag, a
  nonexistent slug in dev shows a "missing param" overlay instead of the 404 page, because export
  mode rejects the route before `notFound()` runs. Fix: `output: process.env.NODE_ENV ===
  "development" ? undefined : "export"` in `next.config`; the build still exports, dev gets the real 404.
- **`trailingSlash: true`** makes the export generate `out/<slug>/index.html`; canonical, sitemap and
  internal links then use `/slug/`. Cloudflare Workers Assets serves `/slug/` directly and redirects
  `/slug`. Decide before writing the canonical.
- **`sitemap.ts`, `robots.ts`, `manifest.ts`, `opengraph-image.tsx`**: `export const dynamic =
  "force-static"`.
- **Inline JSON-LD**: escape `<` (`<`) in `JSON.stringify`, otherwise a `</script>` inside a
  FAQ answer closes the tag.
- **`not-found.tsx`** generates `out/404.html`; in `wrangler.jsonc`, `assets.not_found_handling:
  "404-page"`. Text in a data file, `robots: { index: false }`.
- **Header anchors** (`#precos`) break outside the home: prefix with `/` (`/#precos`).
- Minified HTML is a single line: `grep -c` counts 1; use `grep -o ... | wc -l`.
- Two builds in parallel (two agents) corrupt `.next/export`; use a distinct `NEXT_DIST_DIR`
  or serialize.

## Two agents in parallel without conflict (the split that worked)

| Agent | Owns | Does not touch |
|---|---|---|
| SEO | `app/layout.tsx`, `app/robots.ts`, `app/sitemap.ts`, `app/manifest.ts`, `app/opengraph-image.tsx`, `data/seo.ts`, `components/seo/JsonLd.tsx` (Organization/WebSite/SoftwareApplication) | `app/page.tsx`, `public/llms*.txt`, FAQPage |
| GEO | `public/llms.txt`, `public/llms-full.txt`, `components/seo/FaqJsonLd.tsx` + insertion in `app/page.tsx`, proposals in `data/faq.ts` | `app/layout.tsx`, robots, sitemap, entity JSON-LD |

Both: research this year's practices with WebSearch before applying (what was a rule last year may
have been dropped), audit `out/`, apply, run typecheck + build, report in the SKILL's format.

## Minimum verification before saying "done"

```bash
pnpm typecheck && pnpm build
ls out/robots.txt out/sitemap.xml out/manifest.webmanifest out/llms.txt out/404.html
grep -o '<loc>[^<]*' out/sitemap.xml
grep -o 'application/ld+json' out/index.html | wc -l
grep -o '"@type":"[A-Za-z]*"' out/index.html | sort | uniq -c
grep -n "—" data/ public/llms*.txt      # if the copy rule forbids em dashes
pnpm shots http://localhost:3000/<slug>  # scrollWidth == viewport at 390..1920
```
