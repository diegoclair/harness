# Measurement: is the brand showing up, and where do customers come from

## Free signals (set up on launch day)

| Source | What it gives | Limit |
|---|---|---|
| **"How did you hear about us?" on signup** | The only attribution that sees an AI answer read without a click, a pasted link, a recommendation. One required field with fixed options (Google, ChatGPT, Gemini, Perplexity, Claude, Copilot, YouTube, Instagram, referral, other + free text), stored with the account | Self-reported; the option list must name each assistant or answers collapse into "Google" |
| Search Console → Performance | Clicks and impressions from AI Overviews/AI Mode, summed into the "Web" type ([AI features](https://developers.google.com/search/docs/appearance/ai-features)) | Not separable there |
| Search Console → Generative AI performance report | Impressions in AI Overviews/AI Mode and Discover; Google's AI guide points to it ([Google AI guide](https://developers.google.com/search/docs/fundamentals/ai-optimization-guide)) | Impressions only: no clicks, CTR or position; the same impressions are already in "Web". Announced Jun 2026 for a subset of sites |
| Bing Webmaster → AI Performance (preview since 10 Feb 2026) | Citations, average cited pages per day, grounding queries, citations per page ([Bing](https://blogs.bing.com/webmaster/February-2026/Introducing-AI-Performance-in-Bing-Webmaster-Tools-Public-Preview)) | Covers Copilot, Bing AI summaries and unnamed "select partners"; ChatGPT is not named. Grounding queries are a sample; no clicks |
| Analytics referrals | Sessions from `chatgpt.com`, `perplexity.ai`, `gemini.google.com`, `copilot.microsoft.com`, `claude.ai`; ChatGPT appends `utm_source=chatgpt.com` to part of its links | A pasted link loses the referrer and lands in Direct; `utm_source` without `utm_medium` can land in "Unassigned" |
| CDN logs by user-agent | Crawl of each AI bot, and whether `/llms.txt` is ever fetched | Crawl is not citation |

## Prompt panel: frequency over many runs

One run per prompt measures noise: the same brand list repeats in under 1% of runs, while the share
of runs a brand appears in is stable (SparkToro/Gumshoe, Jan 2026, via
[SEJ](https://www.searchenginejournal.com/ai-recommendations-change-with-nearly-every-query-sparktoro/566242/)).
Sentiment flips in 45.5% of re-runs against 6.8% for presence ([arXiv 2606.20065](https://arxiv.org/html/2606.20065)).
Position and sentiment are not panel metrics.

**Design.** N fixed prompts × M runs × each engine, monthly, fresh session each run:

- N ≈ 20: half category/recommendation ("which tool for <task>", "alternative to <competitor>"), half
  factual about the brand ("what is <brand>", "how much does <brand> cost", "does <brand> work on <Y>").
- M ≈ 5 runs per prompt, so 100 observations per engine per round.
- Engines: ChatGPT, Gemini, Perplexity, Google AI Mode, Copilot, Claude.
- Per run record: brand present (yes/no), whether it searched (sources shown), **every cited
  domain**. The cited domains are what turns into action: they name the third parties to be in.
- Keep the condition fixed and written down (logged in or out, model, country, language): free
  logged-out ChatGPT searches far less than paid ([Ahrefs, Oct 2026](https://ahrefs.com/blog/query-fan-out-chatgpt/)).
  A change of default model breaks the series; note it.

**The math (binomial proportion, 95%).**

- Margin on a rate near 50%: ±1.96·√(0.25/n). n=10 → ±31 pp; n=50 → ±14 pp; n=100 → ±10 pp.
  Report per engine with n ≥ 50; per-prompt rates from 5 runs are anecdotes.
- **New brand: measure presence, not rate.** With 0 appearances in n runs, the 95% upper bound is
  ~3/n (rule of three): 0 in 100 means the true rate is below 3%. If the true rate is 3%, 100 runs show
  at least one appearance with 95% probability (1 − 0.97¹⁰⁰).
- Two rounds differ only when the difference exceeds the combined margin
  (≈ 1.96·√(p₁(1−p₁)/n₁ + p₂(1−p₂)/n₂)); smaller moves are not change.

Record each round in the project's measurements folder, one file per month.

## Paid tools (only after launch, and only if they show their math)

Google: no third-party tool has access to its ranking or AI systems ([Google AI guide](https://developers.google.com/search/docs/fundamentals/ai-optimization-guide)).
Ask any vendor how many runs per prompt it takes.

| Tool | Price (read on the vendor page, Oct 2026) |
|---|---|
| Otterly | US$ 29/mo for 15 prompts, 189 for 100, 489 for 400; ChatGPT, AI Overviews, Perplexity, Copilot; Claude, AI Mode, Gemini as add-ons ([pricing](https://otterly.ai/pricing)) |
| Profound | Free 7-day trial (50 prompts, 3 engines); paid plans on request ([pricing](https://www.tryprofound.com/pricing)) |
| Peec AI, Scrunch, AthenaHQ, Semrush AI Visibility Toolkit | Not read at the vendor page (Peec's needs JS); check before quoting |
