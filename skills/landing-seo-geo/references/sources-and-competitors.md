# Sources: no link, no go-live. And how to name a competitor without risk

## The rule

Every claim about a platform, competitor or market has **URL + read date + literal excerpt**
recorded in the research doc and the link visible on the page. Without that, the sentence leaves or
becomes a sentence without the fact. In the round of 26 Aug 2026 the rule caught, before going live: a
false sentence ("no hub publishes its sync interval": one did), a wrong term ("envios
atrasados" (late shipments) when the official page says "envios incorretos" (incorrect shipments)), a channel without a source ("app
notification"), a nonexistent number ("60 characters" for the title: the limit is per category) and a rule
generalized improperly ("white background" is only mandatory in 4 categories).

Stamps: `oficial` (platform page) · `fornecedor` (the competitor's own page) ·
`imprensa` · `anedota` · `lei` · `[sem fonte]` · `A CONFERIR` (URL not re-read today).

## How to re-read at the source

- Mercado Livre pages return 403 to fetchers; `curl -sL -A "Mozilla/5.0 ..."` solves it.
  Shopee help center: JSON API `help/api/v3/article/detail`.
- "Read" = the excerpt was in the HTML/JSON returned today. A reading taken only from a search engine snippet
  is written down as such.
- Competitor price: full price of the comparable plan, not the month's promotion; note the checkout
  when the site does not publish it.
- The source page's title changes; the URL usually stays. Do not cite an old title.

## Mechanism in code

- `data/sources.ts`: all URLs, grouped by entity, plus `readAt` and the "lido em" (read on) string.
- Optional `sources: Source[]` field on checklist items, `compare` blocks, `callout`, `FaqItem`,
  `steps`; `link` on cited rules in scenes; `source` on price references.
- `components/ui/SourceLink.tsx` (`SourceLink`, `SourceList`): same link style as the site,
  `target="_blank" rel="noopener noreferrer"`.
- `llms-full.txt` gets the URLs as text next to each claim.
- Deliverable: table claim → where it appears → URL → status (`docs/research/links-landing.md`) and
  the updated source docs. A separate session for this worked better than the same agent that
  wrote the copy.

## Naming a competitor (survey of 13 players, 26 Aug 2026)

- 5 of 13 name a competitor on their own site (Plugg.to, Hub2b, Base.com, Predize, UpSeller);
  zero use a logo; only UpSeller cites someone else's price, and without link or date. Good reference: Nuvemshop
  `/compare/nuvemshop-ou-shopify` with the disclaimer "elaborado com base em informações públicas dos
  sites oficiais, <mês/ano>" (compiled from public information on the official sites, <month/year>).
- CONAR art. 32: comparison is accepted if objective, provable, without brand confusion, without denigrating,
  without using someone else's image/prestige, indicating when the price level is different. STJ REsp 1.668.550:
  naming a brand in a comparison does not infringe the trademark. Known punishments were for adjectives/superlatives
  without proof, not for naming.
- **Practical rule:** naming is allowed, with fact + link + date; price only for the comparable plan with URL and
  date; zero logo/slogan/color of others; zero adjective/superlative (also about us, without measurement);
  category when the fact is about the category, name when it is about one player. A competitor's name in the URL
  or title only with the owner's legal sign-off.
- The logo of a platform you integrate with (ML, Shopee) is third-party brand use: only with the
  integration live and the brand guideline read. Until then, name in text.
