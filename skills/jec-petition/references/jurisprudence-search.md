# Case-law search

Learned in the original case: the state court's search sits behind a captcha, so neither web search nor
a fetch tool can run it. Web search returned news articles (one with a wrong judgment date) and
aggregator pages that refuse automated access. What worked was the user running the queries, sending a
screenshot of the result list, downloading the PDFs you point at, and you reading the full text.

## What you can reach yourself

- Official news pages of courts — good leads, with the case number. Treat every detail as unverified
  until the ruling is read.
- Some courts expose full rulings through a public API or direct links; a reviewer agent managed to read
  full texts from one such court. Try before assuming a court is closed.
- A known case number can often be checked in the court's public case lookup even when the search
  engine is closed.

## What the user runs

Give the search page URL and one query at a time, in a code block they can copy.

Syntax rules learned on a state-court engine (check the engine's own help page — ask the user for a
screenshot of it if the first query misbehaves):

- No operator means AND. `OU` is OR. Parentheses group. Quotes are an exact phrase, at most five words.
- The free-text field has a **character limit** (120 on that engine). Count before sending; a rejected
  query costs the user a round trip.
- Mixing two topics with one OR and no parentheses returned four results. Search one topic per query.
- Loose words pull other areas of law (criminal, health plans). Anchor with the sector word and with
  phrases.
- Offer the subject filter (for example "Telefonia") and the small-claims appellate panels as origin,
  since that is where the user's own case would be reviewed.

Query shapes that worked:

```
<remedy> E <clause type> E ("phrase A" OU "phrase B" OU "phrase C") E <duty>
("wrong act A" OU "wrong act B") E <object> E ("paid phrase A" OU "paid phrase B") E "dano moral"
```

## Triage from a screenshot

From each result line you have: case number, chamber, rapporteur, date, and a truncated headnote.
Point the user at the ones to download, by position and number, with one line on why. Ask also for the
ones that look **adverse**. Skip, and say so: corporate plans and companies as plaintiffs when the user
is an individual; motions for clarification of a case already listed; other sectors.

Ten to fifteen PDFs read is enough for one case. Stop when each claim has a ruling from the user's own
state court, ideally against the same defendant.

## Reading

- Extract the cover page and headnote first: parties, chamber, rapporteur, date, outcome.
- Then find the amount (`grep` for the currency symbol near "arbitra", "mantida", "fixada") and who the
  appellant was. "Appeal dismissed" means something different depending on who appealed.
- Read enough of the body to know the facts. A headline match with different facts (blocks caused by
  third-party fraud, not by billing) can still be cited, described accurately.
- Note the date printed on the ruling and the one on the search listing if they differ; cite what the
  document shows.

## Filing the PDFs

Rename each: `<court>-<chamber>-<case number>-<defendant>-<one-line gist>-<amount or "contrario">.pdf`.
Keep the cited ones in `peticao/jurisprudencia/` and move the rest to `nao-citadas/`. The adverse ones
stay there on purpose: whoever prepares for the hearing needs them.

## Citing

One bullet per ruling: court, chamber, class and number, rapporteur, judgment date, the defendant if it
is the same company, the facts in a clause, the outcome and amount, and a literal excerpt from the
headnote when it states the principle better than you can.

Order by closeness to the user's facts. A ruling against the same defendant, on a single bill paid on
time, with the service cut anyway, leads — even ahead of a higher amount.

When the best ruling expresses the award in minimum wages, give today's equivalent next to the request.
