# New brand: why "what is X" still returns "I don't know", and the mentions plan

## Indexed but unknown: ranked causes, each verifiable

A site can be indexed, crawlable by every AI bot, with JSON-LD live, and still get "I don't know".
The markup is not the cause.

1. **The answer came from training, not search.** Most responses do not search (14.4% did on free
   ChatGPT, [Ahrefs, Oct 2026](https://ahrefs.com/blog/query-fan-out-chatgpt/)), and the Gemini app searches
   only when the model judges it helps ([grounding doc](https://ai.google.dev/gemini-api/docs/google-search)).
   A fresh brand token, worse if it reads like a surname, rarely triggers a search. Expect the
   search-first surfaces first (Perplexity, ChatGPT with search, Copilot), AI Mode next, the Gemini
   app last.
2. **Re-registered domain, old identity still in an index.** "It takes a lot of time for the old
   state of a domain to be shaken off ... nothing manual that you can / need to do" (John Mueller via
   [SEJ, 30 Sep 2025](https://www.searchenginejournal.com/google-explains-expired-domains-and-ranking-issues/557283/)).
   Each index clears on its own schedule: Google can be clean while Bing (ChatGPT, Copilot) still
   serves the previous owner. Check each index separately.
3. **Zero third-party mentions.** One self-owned source has nothing to corroborate; category answers
   come from third parties (`references/geo.md` § "What moves citation").
4. **Name collision.** A string that is also a surname, a company abroad or a GitHub handle is not
   read as a brand query. The definition sentence and off-site profiles with the identical name + URL
   disambiguate.
5. **No Knowledge Graph node.** Wikidata only after two independent references exist; an item with
   self-references only gets deleted.

Order of work: confirm each index → off-site anchors with identical name, description and URL
(LinkedIn company page, Crunchbase, Google Business Profile, GitHub org, Product Hunt), all in
`Organization.sameAs` → independent mentions in the site's language → Wikidata.

| Check | Query / URL | Failure looks like |
|---|---|---|
| In the Google index | `site:<domain>` | 0 results, or the old titles |
| Old state still served | `site:<domain>/<old-path>` | any hit |
| Per-URL truth | Search Console → URL Inspection → Test live URL | Discovered/Crawled, not indexed |
| Coverage | Search Console → Pages | "sitemap processed" is not "indexed" |
| Bing side (ChatGPT, Copilot) | `site:<domain>` on bing.com; Bing Webmaster → AI Performance | absent, or the old site |
| Brave side (Claude) | `site:<domain>` on search.brave.com | absent, or the old site |
| Brand SERP | `<brand>`, `<brand> <category>`, "what is <brand>", incognito, site language | site absent or below the collisions |
| Grounding proof | ask Gemini `site:<domain> what is <brand>` | answers with the URL but not without it: entity gap, not content |

## Mentions plan

Aim at the domains the niche's answers already cite (measurement panel), not at a fixed list. Which
source types weigh most shifts by engine and month (`references/geo.md`). Channels, with the evidence
for each:

| Channel | Evidence |
|---|---|
| YouTube video with the brand in title, description and speech (own channel and creators') | Highest measured correlation, ~0.74 ([Ahrefs, Dec 2025](https://ahrefs.com/blog/ai-brand-visibility-correlations)) |
| Inclusion in the niche's "best X" listicles and comparisons, with a 2-line definition ready | ~21% of citations are listicles ([arXiv 2606.20065](https://arxiv.org/html/2606.20065), vendor author, Jun 2026) |
| Earned press: trade and tech outlets in the site's language | ~84% of cited sources are earned media (Muck Rack, May 2026, [summary](https://muckrack.com/blog/ai-citations-for-pr)); ChatGPT's share of licensing-partner outlets rose to 12.5–15.8% in 2026 (Ahrefs, Oct 2026) |
| Founder answering real questions in communities and forums (Reddit where the niche is there) | Weight swings by engine and month: from 1 in 5 to 1 in 43 ChatGPT answers in the US in 2026 (Ahrefs); 9.0% of ChatGPT citations in Brazil ([Profound, Aug 2026](https://www.tryprofound.com/reports-guides/comportamento-de-prompts-e-citacoes-no-brasil)) |
| English-language sources of the category (directories, comparisons) | 43% of non-English fan-outs run in English ([Peec, Feb 2026](https://docs.peec.ai/research/chatgpt-searches-in-english)) |
| Launch directories (Product Hunt, BetaList), review platforms (G2, Capterra), complaint sites | Profiles for `sameAs`; reviews only from real customers; no measured citation effect |
| Wikidata | Entity disambiguation; after two independent references |

Never buy or fabricate mentions: Google's AI guide calls inauthentic mentions unhelpful and its spam
policy covers manipulating AI responses (`references/content-pages.md` § "Scale").

Not measured, do not budget on: "Wikidata makes you 3× more eligible", "72% vs 14%", "100% of cited
tools are on G2/Capterra", "content <30 days cited 3× more", "44% of citations come from the first 30%
of the page". No primary study found for any of them (searched Oct 2026).
