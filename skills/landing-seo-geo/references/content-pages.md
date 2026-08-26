# Content pages: one per topic, scalable

## Architecture (the one that survived review)

```
data/pages/types.ts        closed schema: ContentPage { slug, kind: article|about, seo{title,description},
                           eyebrow, h1, lead, status?, blocks: Block[], faq, related, cta }
                           Block = prose | checklist | steps | compare | callout | pricing | facts | capture
data/pages/<slug>.ts       one page = one file; every visible string lives here
data/pages/index.ts        registry (array) + findPage
app/[slug]/page.tsx        generateStaticParams from the registry, generateMetadata per page
components/content/        ContentPage (breadcrumb, eyebrow, h1, lead, blocks, FAQ, "read also", capture)
                           + blocks.tsx, using ONLY the site's tokens/fonts/scale
components/seo/PageJsonLd  Article|AboutPage + BreadcrumbList + Organization
app/sitemap.ts             reads the registry; footer and "read also" do too
```

New page = 1 data file + 1 line in the registry. Zero code, zero new token. Note this in the
repo's `CLAUDE.md` in the "where to change what" table.

## Answer-first lead

The first paragraph answers the title's question in 2 to 3 self-contained sentences that name the
entity. That is what the assistant cites; the rest of the page is proof. Title ≤60 with the search
term; description ≤155.

## The first three pages

| Order | Page | Content | Caution |
|---|---|---|---|
| 1 | `/ia-para-<segmento>` | what the AI does in that context, the platform's official rules with link, what the platform already gives for free and what changes, price, own FAQ (5 to 7 assistant questions), capture | Do not promise an integration that does not exist; "under construction" stated explicitly |
| 2 | `/o-que-e-a-<marca>` | one-sentence definition, for whom, what it does per unit every day, what it is not, pricing model, stage, who makes it (only what the docs state), short FAQ; AboutPage JSON-LD | No "company being born"; nothing invented about the team |
| 3 | `/<marca>-vs-<categoria>` | what the category does, what the brand does, side-by-side table **including where the category wins today**, when to pick each, FAQ | Third-party facts only with URL + date; no adjectives; competitor name per `sources-and-competitors.md` |

Comparison table: at most ~6 rows to fit in 2 screens of 844 px on the phone; below
`md` it stacks via CSS (a single markup, no duplicated content). Source line with date underneath.

## Brand 404 page

`app/not-found.tsx` + `data/notFound.ts`; same header/footer/tokens; honest title ("Esta
página não está no ar." — this page is not live), primary button to the home and secondary to capture; `robots: noindex`.
One brand detail (at Rednev, the mirrored "404", because the brand is "selling backwards") is worth
more than an illustration.

## Before going live

- The pages' copy goes through the owner (it is brand text); the agent delivers a list of claims to
  check against the source doc.
- Screenshots at 390/834/1100/1440/1920 without overflow; no section >2 screens.
- Align the home with what the pages say (stage, supported marketplaces): the inconsistency
  shows up when new pages are born with facts more recent than the home.
