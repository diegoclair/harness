---
name: jec-petition
description: "Builds a Brazilian small-claims (Juizado Especial Cível) initial petition for a consumer acting without a lawyer, from a raw complaint to a file-ready package: guided fact gathering one item at a time, an evidence folder with a living case file, claims filtered against the user's own documents, legal grounds and case law verified at the primary source (never from memory), a forensic-format petition, numbered exhibits, and an independent skeptical review before filing. Use it WHENEVER the user wants to sue or 'processar' a company, mentions Juizado Especial, JEC, pequenas causas, petição inicial, Procon, Anatel, consumidor.gov, cobrança indevida, bloqueio or suspensão de serviço, negativação, multa de fidelidade, dano moral, 'entrar com ação', or is venting about a company that wronged them and asks where to complain — even if they never say 'petição'. Replies match the user's language."
---

# jec-petition

This skill was born from a real case (October 2026): a consumer whose mobile lines were blocked three
times over an invoice already paid, who started by venting and ended the same day with a petition, 25
numbered exhibits, six appellate rulings read in full and an independent review. What is here is the
order that worked and the mistakes that were caught before filing. Each reference says where its lesson
came from.

The person is not a lawyer and neither are you. The petition will be read by a judge and attacked by the
defendant's lawyers. That sets the two rules everything else follows from:

- **Nothing enters the petition that the user's own documents do not support.** A claim the defendant
  can knock down with one document makes the judge doubt the others, including the moral-damages claim,
  which depends on the judge believing the narrative.
- **No statute, article or ruling is cited from memory.** Each one is opened at its primary source in
  this session, or it is left out. An invented or misdated citation is the single worst error this
  skill can produce.

## When the user arrives angry

They usually arrive mid-incident. Answer the question they asked (where to complain, what their rights
are) in a few lines, then offer the path: regulator and consumer portals are fast and cheap but only
forward the complaint; the Juizado is what binds the company. Do not open with a legal lecture, and do
not start drafting. Ask which state they live in — it decides the court and the filing channel.

If they have a relative or friend who is a lawyer, say early that the final text should be read by that
person. With a lawyer signing, the cap doubles and electronic filing becomes direct.

## The flow

Work through the phases in order. Each phase has an exit condition; do not draft before phase 3 closes.

### Phase 1 — Gather facts, one item at a time

The user asked for it explicitly in the original case and it is the right default: **ask for one thing,
wait, record it, ask for the next.** A list of eleven questions gets a partial answer and loses the
rest. Read `references/fact-gathering.md` for the item list, the folder layout and the case-file
template.

- Create the case folder **outside any code repository**, somewhere the user can open (on WSL that is
  the Windows Documents folder under `/mnt/c/Users/<user>/Documents/`, not the Linux home and never a
  temp dir). Ask once if unsure.
- Keep a living `dados.md` in that folder: parties, contract, timeline with a proof column, what each
  document shows, and a "still missing" checklist. Update it after every item. It is your memory across
  a long session and the map a reviewer will use.
- The user drops files in the folder root with any name; you read each one, rename it descriptively and
  file it in a subfolder. Read every document yourself — PDFs with `pdftotext -layout`, images and
  scanned pages by viewing them. Documents contradict memories; that is their job.
- Record what a document **says**, not what the user said it says. Dates drift (the user remembered
  "two or three years ago"; the receipt said four and a half), amounts drift, and which body received
  the complaint drifts.

Exit: every claim you intend to make has a document or an explicit "narrated by the plaintiff" tag.

### Phase 2 — Read the defendant's own paper

Invoices, replies to regulators, contract terms and portal screenshots issued by the defendant are the
strongest evidence there is, and they cut both ways. Extract and compare them month by month.

- Look for the defendant **admitting the rule it broke** (an invoice that prints the suspension rule
  it did not follow; a reply that confirms the suspension date).
- Look for facts that **destroy a claim the user is sure of**. In the original case "they raised my
  plan several times a year" was three discount expirations, each with its end date printed on the
  invoice. That claim was dropped and a narrower, provable one took its place.
- Recompute every number the defendant charged. A fine that turns out to be exactly the contractual
  pro-rata means the argument cannot be "the fine was made up"; it has to be about information and
  proportionality.

### Phase 3 — Decide what goes in

For each candidate claim write one line: the fact, the document, the defendant's most likely answer.
Then sort:

- **In as fact** — documented and uncontradicted.
- **In as a narrower claim** — the documents support less than the user believes; claim that much.
- **In as a request for the defendant's records** — things only the defendant can show (suspension
  logs, call recordings, notices it says it sent). Do not hypothesize about what failed inside the
  company; state what happened and ask the judge to make them explain.
- **Out** — contradicted by the user's own documents, or already resolved at the time.

Tell the user plainly what came out and why. They usually agree once they see the document.

An older grievance against the same defendant can enter in two ways: as its own claim (check limitation
periods at the source and whether small-claims rules allow joining it — see
`references/legal-sources.md`), or only as history showing the conduct repeats. The second is nearly
free; the first needs its own evidence of payment.

Exit: the user has confirmed the list of claims and the amounts.

### Phase 4 — Ground it in law that you opened today

Read `references/legal-sources.md` before citing anything. It lists the primary sources, what to check
at each, and the traps that were hit: provisions annulled after publication, a regulation whose entry
into force was postponed, a superior-court thesis with a cut-off date, a joinder rule specific to small
claims.

- Quote only what you read. If a fetch tool summarises instead of quoting, ask it again for the literal
  text, and treat a paraphrase as a paraphrase (no quotation marks).
- Regulations change. Check whether each cited provision was in force **on the date of the fact**, and
  cite the regulation in force then for older facts.
- If you cannot reach the source, the citation stays out and you say so.

### Phase 5 — Case law, with the user's hands

Court search engines sit behind captchas, so the user runs the searches and downloads the PDFs; you
write the queries and read the full text. Read `references/jurisprudence-search.md` for query syntax,
length limits and how to triage a results screenshot.

- Cite a ruling only after reading its full text. Take court, chamber, case number, rapporteur,
  judgment date and amount **from the document**, not from a news article about it.
- Read the adverse rulings too and keep them in a separate subfolder. They are what the defendant will
  cite; the petition should already be built on the distinction.
- Rulings against the same defendant on the same conduct are worth more than higher amounts elsewhere.
- Use the amounts actually awarded to calibrate the moral-damages request, and tell the user the
  realistic range next to the amount asked.

### Phase 6 — Draft in forensic format

Read `references/petition-structure.md` for the skeleton, the exhibit-numbering rule and how to produce
the Word file.

- Write for the judge: plain, short paragraphs, each fact followed by its exhibit number.
- Address the weak point head-on. If there is a signed term, say so and argue around it; a petition
  that hides it loses credibility the moment the defendant attaches it.
- Add a subsidiary request wherever double restitution is asked.
- Keep the total under the cap for acting without a lawyer, with room for monetary correction.
- Copy every exhibit into an `anexos/` folder named with the same numbers as the exhibit list.

### Phase 7 — Independent review, then a second round

Dispatch the `petition-reviewer` agent with the petition path, the exhibits folder and today's date.
Give it the files, not your conclusions. It verifies every citation at the source, checks every fact
against the exhibits, and attacks the piece as the defendant's lawyer would.

- Apply what it finds. In the original case it caught a wrong judgment date, a quotation cut in a way
  that turned the consumer's allegation into the agency's finding, and several facts stated beyond what
  the exhibits showed.
- When the reviewer recommends cutting a claim and the user disagrees, **send the user's argument back
  to the same reviewer** (it keeps its context) and ask for a firm position with the conditions that
  would change it. It may reverse itself on new research; that reversal is worth more than deference
  either way.
- A fact the reviewer flags as unsupported is a question for the user, not something to soften with
  wording.

### Phase 8 — The user's final read

The user is the only one who knows what happened. Ask them to read the whole petition marking three
things: a fact that is wrong or not how they remember it; something missing; a sentence they would not
sign. Expect real corrections here — in the original case the user caught two statements that were
simply untrue (a protocol number that had in fact been given; a "third party's phone" that belonged to
the co-plaintiff).

Once the user starts editing the Word file directly, **that file is the master**. Stop regenerating it
from your source; hand over exact replacement text instead, or you overwrite their edits.

### Phase 9 — Filing

Read `references/filing.md`. Filing channels change; confirm the current one on the court's own site
in this session. Do not file on the user's behalf. Tell both plaintiffs that missing a hearing ends the
case for whoever is absent.

## What to tell the user as you go

- Lead with what changed in the case, not with what you did.
- Say probabilities as estimates and say whose they are.
- When a favourable-looking fact turns out weaker (an agency opinion that records an allegation rather
  than a finding), say it once, clearly, and adjust the text.
- Never promise an outcome. State what is documented, what is disputed and what the realistic range is.

## Mistakes this skill exists to prevent

| What happened | Rule |
|---|---|
| A claim was drafted from the user's phrasing ("no protocol was given") and was false | Record facts in the user's own terms and re-ask before turning an absence into an accusation |
| "Borrowed phone" written for a line that belonged to the co-plaintiff | Ask who owns every number, device and account that appears |
| Agency "ruled twice" for the consumer; the second document was a re-send of the first | Read both documents and compare dates and signatures before counting |
| A ruling's date and facts taken from a news article were wrong | Dates and facts come from the ruling itself |
| A small recurring discount claimed as overcharge; the user's own payment history defeated it | Check the claim against every payment method and date in the folder |
| The Word file regenerated over the user's manual edits (avoided in time) | The edited file is the master from the first manual edit |
| The legal theory for an old fine relied on "no prior knowledge" despite a signed term | Find the argument the favourable rulings actually use; here it was the missing warning at the later digital offer |

## References

- `references/fact-gathering.md` — item list, folder layout, case-file template, evidence that is usually missing.
- `references/legal-sources.md` — where each kind of rule lives, how to verify, known traps.
- `references/jurisprudence-search.md` — query syntax, triage, reading and filing rulings.
- `references/petition-structure.md` — skeleton, exhibits, requests, producing the document.
- `references/filing.md` — how to find the competent court and the current filing channel.
