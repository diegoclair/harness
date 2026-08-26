# GEO: getting cited by ChatGPT, Gemini, Perplexity, Copilot and AI Overviews

Sources from the round of 26 Aug 2026: Princeton study at scale (arXiv 2606.20065, 2026), crawler
docs from OpenAI/Anthropic/Perplexity/Google, tests by measurement platforms, and a reality
check on the niche's queries.

## What moves the needle

1. **Third-party presence.** ~68% of citations come from outside the brand's site; the most
   cited format is the ranked listicle "best X" (~21% of citations). A niche brand shows up in ~11%
   of relevant answers; a global brand in 73%. Conclusion: the site is a prerequisite, the citation comes
   from the mentions plan.
2. **Profile per engine.** ChatGPT weighs Wikipedia/encyclopedic sources and uses the Bing index
   (so Bing Webmaster is mandatory). Perplexity weighs Reddit (~47% of citations) and recency
   (content <30 days is cited ~3x more). AI Overviews/AI Mode use the Google index with E-E-A-T and
   classic links. Gemini follows Google.
3. **Citable passage.** 1 to 3 self-contained sentences, with the entity named, answering the question
   someone would ask the assistant. One per section is enough; "it"/"the AI" without the name does not become a snippet.
4. **HTML without JS and crawlers allowed.** Search bots do not execute JS.
5. **Citing an authoritative external source inside the text** (the platform's official rule, with a link)
   increased citation in tests: the site "borrows" authority and becomes verifiable.
6. **Clear entity**: an "X is ..." sentence above the fold or in the footer, a `/o-que-e-a-x` page,
   Organization with `sameAs` when profiles exist (empty is worse than absent).

## Myths (do not spend time waiting for an effect)

- `llms.txt`: Google does not use it (on record); logs show GPTBot/ClaudeBot/PerplexityBot almost
  never fetching it; adoption ~10%. Publish it because it costs zero and concentrates the facts in clean text.
- FAQPage "makes ChatGPT cite you": no evidence; the effect is indirect and about disambiguation.
- Blocking GPTBot/Google-Extended "protects": it removes the brand from base knowledge. A new brand does not
  block.
- Vendor numbers ("+30% sales") dominate listicles; competing on that without measurement breaks the
  data rule. The gap is a clear definition + honest comparison + third-party presence.

## User-agents that need `Allow: /`

`OAI-SearchBot`, `GPTBot`, `ChatGPT-User`, `ClaudeBot`, `Claude-SearchBot`, `Claude-User`,
`anthropic-ai`, `PerplexityBot`, `Perplexity-User`, `Google-Extended`, `Googlebot`, `Bingbot`,
`Applebot`, `Applebot-Extended`, `CCBot`, `DuckAssistBot`, `meta-externalagent`, `Amazonbot`.
`User-Agent: * / Allow: /` already covers them. Never block the `-SearchBot`/`-User` ones: they are the ones that cite.
Check in Cloudflare that "Content Signals"/"AI bots" are not blocking account-wide
(shows up as a block in the served `robots.txt`).

## Page GEO checklist (score before and after)

- [ ] An "X is ..." sentence with the entity, above the fold, in the footer or in the first FAQ
- [ ] For whom, what it does, how much it costs, where it works: all in text, without JS
- [ ] One citable passage per section with the entity as the subject
- [ ] FAQ with the questions people ask the assistant: "what is X", "is X a <category>?",
      "does X work on Y?", "how much does it cost", "does X increase Z?" (answered without a promise)
- [ ] Permissive robots; `llms.txt`; entity JSON-LD with real `sameAs`
- [ ] Third-party rules/facts with link and date
- [ ] No number without a published measurement

## Reality check (do it before writing copy)

Search 3 to 5 questions from the niche and list who is cited and **why** (per-solution pages,
educational blog, own "X vs Y vs Z" comparison, third-party listicles, docs). Pattern found in the
BR marketplace niche: educational PT-BR content answering a generic question + own comparison +
listicles, almost always with a vendor number. The opening is the honest definition.

## Mentions plan (priority × impact)

1. Reddit/communities/YouTube as the founder answering a real question (Perplexity, Gemini; recency).
2. Inclusion in the niche's listicles, with the definition ready in 2 lines and the honest comparison
   as the editorial differentiator.
3. Product Hunt / BetaList / startup press at launch (creates profiles and `sameAs`).
4. G2/Capterra/GetApp profiles pre-launch; reviews only with a real customer.
5. Reclame Aqui: reply, when it exists.
6. Niche guest posts/podcasts with the thesis and the public price.
7. Wikidata (feasible before Wikipedia): entity, site, founder, date.
8. The platforms' app marketplaces (gatekeepers of GMV; medium term).

## Measurement

- Manual biweekly panel, private window, 8 fixed queries (e.g. "which AI for <task>",
  "<brand> what is it", "alternative to <competitor>", "<task> from the phone"), on ChatGPT with
  search, Gemini, Perplexity, Google AI Mode, Copilot. Record cited/position/sentiment/source
  in `docs/measurements/geo-<month>.md`. New brand baseline: zero.
- Bing Webmaster → "AI Performance" (beta): citations on Copilot/ChatGPT for free.
- Cloudflare logs: monthly hits per AI user-agent and on `/llms.txt` (if zero for 3 months, the
  myth is confirmed).
- Paid tool (Peec, Profound, LLMrefs) only after launch.
