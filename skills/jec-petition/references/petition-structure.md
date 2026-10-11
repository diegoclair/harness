# Petition structure

The first draft in the original case was correct and readable, and a lawyer's verdict was "almost a
lawyer — only statute and case law missing", plus "the format is not the usual one". This is the format
that replaced it. Write the petition in the language of the court.

## Contents

- Skeleton
- Writing the facts
- Writing the law
- Urgent relief
- Requests
- Exhibits
- Producing the document

## Skeleton

1. **Addressing** — the competent small-claims court (confirm by the plaintiff's postal code on the
   court's own tool).
2. **Flags** under the addressing, when they apply: priority handling; request for urgent relief.
3. **Heading** — each plaintiff: full name, nationality, marital status, occupation, ID numbers; common
   address, e-mail and phone; the statutory basis for acting without a lawyer; the name of the action;
   the defendant with its tax number and registered address (take both from the defendant's own
   documents and from the receipts — payments show which entity was paid).
4. **I – Summary** — three short paragraphs: what happened, that it is not the first time (if true),
   what is asked.
5. **II – Jurisdiction, priority, connection of requests.**
6. **III – Facts**, in numbered subsections (see below).
7. **IV – Law**, one subsection per legal question.
8. **V – Urgent relief.**
9. **VI – Requests**, lettered, then evidence to be produced, witnesses, value of the claim, closing
   formula, place and date, one signature line per plaintiff.
10. **Exhibit list.**

## Writing the facts

- One subsection per block: the contract; the paid charge; each incident, dated; absence of notice and
  of a working complaint channel; why the risk continues; overcharges; the older matter.
- Every factual sentence that can be proven ends with its exhibit: `(doc. 7)`, `(docs. 8 a 10)`.
- What an agent said on the phone is narrated as said to the plaintiff, without quotation marks unless
  the wording is certain.
- State how each incident was discovered and ended. "Restored minutes after the call, blocked again
  days later" is the pattern that supports the injunction.
- Explain the limits of your own evidence in one sentence (a call log that shows no year), and point to
  the request for the defendant's records.
- Name who owns every phone number used.
- No hypothesis about the defendant's internals. No adjectives doing the work of facts.

## Writing the law

- Open with the consumer relationship and the burden of proof, then the central wrong, then each money
  claim, then moral damages.
- For each rule: one sentence of what it requires (quoted when read literally), one sentence of how the
  facts breach it.
- Put case law where it supports a specific point, as bullets, most similar first.
- For the weak claim, structure the argument in the order the favourable rulings use, and face the
  adverse fact (the signed term) in the text.
- Moral damages: list the concrete circumstances, then the precedents with amounts, then why this case
  sits at or above them (repetition, number of incidents, an elderly user, no working channel), then the
  amount per plaintiff with its anchor.
- Add one sentence on limitation when a claim is old.

## Urgent relief

When the service is working on the filing date the danger is repetition, not ongoing deprivation. Ask
for an order to **refrain** from cutting or negativating over the specific charge, **and** to restore
within a set time if the service is cut when the order is served. Cite the provision that dispenses
proof of damage for relief that inhibits repetition. Ask a daily fine **with a cap**; an uncapped fine
tends to be reduced.

Keep the request narrow (this charge, these lines). A narrow, reversible order is the one that gets
granted without hearing the other side.

## Requests

- Priority handling; urgent relief and its confirmation; service of process and hearing (by video if
  possible).
- Burden shifting **with a list of what the defendant must produce**: suspension and restoration logs
  with timestamps; contact history and protocols for the period; recordings of each call, identified by
  date and originating number; the notices it claims to have sent; daily usage for the period; for an
  offer accepted online, the screens or log of that flow.
- Declaration that the specific debt does not exist.
- Restitution, each with a subsidiary simple-restitution request, naming the plaintiff it belongs to,
  with correction from payment and interest from service.
- Moral damages per plaintiff, with correction from the award and interest from service.
- Value of the claim: the sum of everything asked in money. Recompute after every change.

## Exhibits

- Number in the order the petition first needs them: IDs and address, contract, the disputed charge,
  payment, the defendant's "paid" screens, each contact, then the older matter, then old receipts.
- One exhibit number can hold two files (`22a`, `22b`); never the same file under two numbers.
- After any change to the list, re-scan every `(doc. N)` in the body. A renumbering pass by script is
  safer than by hand; print the list of references afterwards and read it.
- Copy (do not move) each file into `peticao/anexos/` with the number as prefix and a descriptive name.
- Check that each reference points at the exhibit that actually shows the fact. A reviewer found a
  due date cited to a document that printed a different one.

## Producing the document

Write Markdown and convert with `pandoc peticao-inicial.md -o peticao-inicial.docx`. Check that
`pandoc` exists before promising a Word file.

Regenerate freely while only you are editing. From the first manual edit by the user, the `.docx` is
the master: apply later changes by giving the user the exact "from / to" text, keep the Markdown in
sync, and do not run the conversion again without asking.

Short signed declarations that the case needs (for example, that a co-plaintiff lives at the holder's
address) are produced the same way and placed among the exhibits for signature.
