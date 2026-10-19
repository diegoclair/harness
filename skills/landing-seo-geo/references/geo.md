# GEO: how assistants pick what to cite, and what moves it

Every number carries its source and date. Most studies come from vendors that sell GEO tooling and
are correlational: direction, never a promise for the page.

## How each surface retrieves

| Surface | Index it reads when it searches | What follows |
|---|---|---|
| ChatGPT search | Own crawler `OAI-SearchBot` + third-party providers (OpenAI names none; Bing is the one reported). Blocking `OAI-SearchBot` removes the site from search answers; robots changes take ~24 h ([OpenAI bots doc](https://developers.openai.com/api/docs/bots), Oct 2026) | Bing Webmaster + IndexNow; `OAI-SearchBot` allowed |
| AI Overviews / AI Mode | Google index with *query fan-out* (several related sub-queries); different models, so different links. Only requirement: indexed and snippet-eligible ([AI features](https://developers.google.com/search/docs/appearance/ai-features); [Google AI guide](https://developers.google.com/search/docs/fundamentals/ai-optimization-guide), Jul 2026) | Classic SEO; nothing "for AI" |
| Gemini app | Google Search, when the model judges a search improves the answer ([grounding doc](https://ai.google.dev/gemini-api/docs/google-search), Oct 2026); the consumer app's trigger is undocumented | Expect it last for a fresh brand |
| Perplexity | Own index (`PerplexityBot`, honors robots); `Perplexity-User` fetches on demand and "generally ignores robots.txt" ([Perplexity bots](https://docs.perplexity.ai/guides/bots)) | Most Google-aligned assistant: 28.6% of its citations sit in Google's top 10 ([Ahrefs, Aug 2025](https://ahrefs.com/blog/ai-search-overlap/)) |
| Claude | Brave Search (since Mar 2025) and turbopuffer (May 2026) on Anthropic's subprocessor list, as reported by [xponent21, Jul 2026](https://xponent21.com/insights/claude-web-search-brave-turbopuffer/); fetchers `Claude-SearchBot`, `Claude-User` ([Anthropic](https://support.claude.com/en/articles/8896518-does-anthropic-crawl-data-from-the-web-and-how-can-site-owners-block-the-crawler)) | Check the brand on search.brave.com too; Brave's crawler has no distinct user-agent |
| Copilot | Bing index; "grounding queries" are rewrites of the prompt ([Bing, Feb 2026](https://blogs.bing.com/webmaster/February-2026/Introducing-AI-Performance-in-Bing-Webmaster-Tools-Public-Preview)) | Bing Webmaster |

**Most answers do not search.** Only 14.4% of 66.6M free, logged-out ChatGPT responses ran a web
search; paid and reasoning models search more ([Ahrefs, Oct 2026](https://ahrefs.com/blog/query-fan-out-chatgpt/)).
Training data decides the rest, so a brand that wants to be known never blocks training bots
(`GPTBot`, `ClaudeBot`, `Google-Extended`, `CCBot`).

**Sub-queries drift to English.** 43% of fan-outs from non-English prompts ran in English, and local
leaders lost to global brands ([Peec, Feb 2026](https://docs.peec.ai/research/chatgpt-searches-in-english), 10M prompts).
In a non-English market, find the English sources the category cites and be present there.

**Answers vary run to run.** The same brand list repeats in under 1% of runs on ChatGPT and Google
AI; the share of runs a brand appears in is stable, its position is not (SparkToro/Gumshoe, 2,961
runs, Jan 2026, via [SEJ](https://www.searchenginejournal.com/ai-recommendations-change-with-nearly-every-query-sparktoro/566242/)).
Measure frequency, never rank: `references/measurement.md`.

## What moves citation (ranked by evidence)

1. **Be in every index that feeds a surface** (table above). Without it nothing else counts.
2. **Official pages answer the factual questions about the brand** (price, what it does, where it
   works, "what is X", integrations): one URL per question, answer in the first paragraph, in served
   HTML. For factual questions 63.4% of ChatGPT citations went to official sites (46.4% for
   comparisons), and 75.9% of `site:` sub-queries targeted brand sites ([Ahrefs, Oct 2026](https://ahrefs.com/blog/query-fan-out-chatgpt/)).
3. **Third-party mentions decide category and recommendation answers.** Branded web mentions
   correlate 0.66–0.71 with AI visibility, Domain Rating 0.27–0.33, backlinks "very weak" (Spearman,
   75k brands, [Ahrefs, Dec 2025](https://ahrefs.com/blog/ai-brand-visibility-correlations)). Earned media
   is ~84% of cited sources, paid 0.3% (Muck Rack, May 2026, [summary](https://muckrack.com/blog/ai-citations-for-pr)).
   "Best X" listicles are ~21% of citations ([arXiv 2606.20065](https://arxiv.org/html/2606.20065), single
   author at a GEO vendor, Jun 2026). Plan in `references/geo-new-brand.md`.
4. **YouTube.** Mentions in video titles, descriptions and transcripts have the highest measured
   correlation (~0.74, Ahrefs Dec 2025); 18.2% of AI Overview citations from outside Google's top 100
   are YouTube URLs (Ahrefs, Mar 2026, via [SEJ](https://www.searchenginejournal.com/google-ai-overview-citations-from-top-ranking-pages-drop-sharply/568637/)).
5. **Classic ranking still helps, less than it did.** 38% of AI Overview citations come from the top
   10 for the same query (76% in Jul 2025; parser and model changed in between). Assistants overlap
   the original query's top 10 by ~11% on average (Ahrefs, Aug 2025).
6. **Own data, quotations and linked sources in the text.** In the GEO paper, quotation addition gave
   +41% and statistics +33% on a simulated engine, +22% on Perplexity, concentrated in low-ranked
   pages ([Aggarwal et al., KDD 2024](https://arxiv.org/abs/2311.09735); 2023–24 engines, no replication
   found). Google says no special writing for AI is needed: it counts as original content, not a trick.
7. **Update when the fact changes, with a visible date.** Assistant citations are 25.7% newer than
   organic results; ChatGPT is the most date-sensitive, AI Overviews the least; the average cited page
   is still ~2.9 years old ([Ahrefs, Jul 2025](https://ahrefs.com/blog/do-ai-assistants-prefer-to-cite-fresh-content/)).
   A new date without a content change is discouraged by Google.

The citation mix shifts by month and engine: ChatGPT's Reddit share fell from ~1 in 5 answers to 1
in 43 and Wikipedia from 1 in 7 to 1 in 35 (US, Mar→Sep 2026, Ahrefs Oct 2026), while Brazil showed
Reddit at 9.0% and Wikipedia at 1.8% of ChatGPT citations ([Profound, Aug 2026](https://www.tryprofound.com/reports-guides/comportamento-de-prompts-e-citacoes-no-brasil)).
No fixed source ranking holds; the panel says which domains the niche cites today.

## Citable passage

1 to 3 self-contained sentences that answer the question with the entity named ("X is ..."). One
per section is enough; "it"/"the AI" without the name gives nothing to extract.

## Myths (no effect on citation)

- **`llms.txt`, Markdown for agents, "AI text files", schema "for AI".** "You don't need to create
  new machine readable files, AI text files, markup, or Markdown" ([Google AI guide](https://developers.google.com/search/docs/fundamentals/ai-optimization-guide)).
  97% of `llms.txt` files got zero requests in May 2026; of those read, AI search bots made 1.1% of
  fetches and coding agents 10.5% ([Ahrefs, Jun 2026](https://ahrefs.com/blog/llmstxt-study/)). It serves
  agents reading developer docs, not citation. Publish only if it costs zero.
- **Chunking or rewriting "for AI".** Not required; no ideal length (same guide).
- **Inauthentic mentions.** "Seeking inauthentic 'mentions' across the web isn't as helpful as it
  might seem" (same guide), and manipulating AI responses is spam (`references/content-pages.md` § "Scale").
- **FAQPage for citation.** No evidence; the FAQ rich result stopped showing on 7 May 2026
  ([Google](https://developers.google.com/search/docs/appearance/structured-data/faqpage)).
- **Blocking training bots "protects".** It keeps the brand out of what most answers are built from.

## User-agents that need `Allow: /`

`OAI-SearchBot`, `GPTBot`, `ChatGPT-User`, `ClaudeBot`, `Claude-SearchBot`, `Claude-User`,
`PerplexityBot`, `Perplexity-User`, `Google-Extended`, `Googlebot`, `Bingbot`, `Applebot`,
`Applebot-Extended`, `CCBot`, `DuckAssistBot`, `meta-externalagent`, `Amazonbot`.
`User-Agent: * / Allow: /` covers them. Check that Cloudflare does not block them account-wide
(`references/publish-and-index.md` § "Content Signals").

## Page GEO checklist (score before and after)

- [ ] An "X is ..." sentence above the fold, in the footer or in the first FAQ
- [ ] Price, what it does, for whom, where it works: each answerable from one URL, in text, without JS
- [ ] One citable passage per section with the entity as the subject
- [ ] FAQ with the questions people ask the assistant, answered without a promise
- [ ] Permissive robots; entity JSON-LD with real `sameAs`
- [ ] Third-party rules/facts with link and date; own data where it exists
- [ ] Visible update date that changes only with the content
- [ ] No number without a published measurement

## Reality check (before writing copy)

Run 3 to 5 niche questions a few times each and list which domains get cited and **why**
(per-solution pages, educational content, own comparison, third-party listicles, video, docs). Pages
and the mentions plan aim at those domains, not at a generic list.

## Agent-facing distribution

Being the option an agent installs, scaffolds or recommends by default, a separate channel from
citation because the agent's pick becomes the stack. **Applies only when an agent building software
chooses the product** (APIs, hosting, databases, email, payments, dev tools). It does not reach a
non-technical SMB buyer asking a chat assistant; for that buyer the citation levers above decide.

Measured picks: Claude Code chose Stripe for 91.4% of payment tasks and built its own solution in 12%
of all picks ([Amplifying.ai, Feb 2026](https://amplifying.ai/research/claude-code-picks/report)).
**Defaults are volatile across model versions:** Railway held 82% of Python deploy picks (one FastAPI
repo, three models, Feb 2026) and fell to 3 primary picks in 810 responses on the next model, where
Docker took Python deploys ([Amplifying.ai, Jul 2026](https://amplifying.ai/research/claude-code-picks-fable)).
Re-measure on every major model release; never sell a pick rate as durable.

Mechanisms, ranked by evidence:

1. **Default inside the app builder's template or integration.** The builder wires the vendor
   without the user choosing, and it survives model changes. Resend's founder reported a record
   signup month with no launch, "a ton of new users coming from Lovable, Bolt and v0"
   ([Feb 2025](https://x.com/zenorocha/status/1887880098105495794)); it ships as a native v0
   integration ([Vercel Marketplace](https://vercel.com/marketplace/resend), read Oct 2026).
2. **Official plugin, MCP server and skill, plus a CLI that signs the user up.** Listing in the agent's
   official directory (Claude Code's `claude-plugins-official`, read Oct 2026) and a command that
   creates the account and deploys in one step (`railway up -y`) turn a recommendation into an
   execution path. Usage is self-reported only ("tens of thousands of users every week" via agent
   actions, [Railway, Jun 2026](https://blog.railway.com/p/state-of-railway-agents)).
3. **Third-party pages paid by revenue share.** One page per "how do I host X", written by community
   template authors who keep 25–50% of the revenue they generate: ~1,000 pages (Mar 2025) to ~6,700
   (Oct 2026) at Railway, cost paid only after revenue
   ([Railway, Mar 2025](https://blog.railway.com/p/1M-open-source-kickbacks)). Volume is verifiable;
   the effect on agent picks is unmeasured.
4. **Per-stack guides, markdown docs, `llms.txt`, markdown on `Accept: text/markdown`.** These make
   the agent accurate after it chose you; no source ties them to being chosen. Ship them as hygiene.

No vendor publishes the share of signups coming from agents or assistants (searched Oct 2026), so
every mechanism above is ranked by mechanism plausibility plus self-report, not by controlled test.
