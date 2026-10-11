# Fact gathering

Learned in the original case: the user was on the phone with the company while describing the problem,
angry, and asked to go "one by one so it doesn't get complex". Every item below was collected that way
in a single session.

## Contents

- Folder layout
- The item list, in order
- Reading what arrives
- The case file (`dados.md`)
- Evidence that is usually missing, and what replaces it
- A co-plaintiff

## Folder layout

One folder per case, where the user can open it from their file manager. Subfolders appear as the
material arrives; do not create empty ones up front.

```
<case>/
├── dados.md              living case file
├── prints/               screenshots, numbered in arrival order
├── faturas/              invoices, named by due date (YYYY-MM-venc-DD-MM-YYYY.pdf)
├── comprovantes/         payment receipts
├── emails/
├── documentos-pessoais/  IDs, proof of address
├── <older-matter>/       one subfolder per agency case, plus a "nao-usar" suffix for what stays out
└── peticao/
    ├── peticao-inicial.md / .docx
    ├── anexos/           copies numbered exactly as the exhibit list
    └── jurisprudencia/   rulings cited; nao-citadas/ for the rest
```

Rules that paid off:

- The user drops files in the folder root under any name and says "done". You move and rename.
- Name files by what they prove and the date: `07-chamada-<number>-2026-09-17-13h28.png`.
- When a better version of a document arrives (a PDF of an email replacing three screenshots), delete
  the copies you made, keep the user's originals.
- Keep what is deliberately excluded, in a folder whose name says so. The decision to exclude is itself
  something a reviewer or a lawyer will want to see.

## The item list, in order

Ask one, wait, record, then ask the next. Skip what does not apply.

1. **The incident happening now** — date, time, which number or channel, what was said. Screenshot of
   the call log or chat.
2. **Who holds the contract** — the plaintiff must be the contract holder. Contract or customer number,
   service address, every line or service under it.
3. **Who uses each line or service** — a dependent line used by a relative makes that relative a
   possible co-plaintiff. Ask who owns every phone number that shows up, including the one used to call
   the company.
4. **The disputed charge** — reference month, due date, amount, breakdown. The defendant's own screen
   showing it as paid is the best exhibit there is.
5. **The payment** — bank receipt with date, time and authentication. Check the statement around the
   date for a second debit before claiming or ruling out double payment.
6. **Each incident** — date, how it was discovered, how long it lasted, how it ended.
7. **Each contact with the company** — date, time, duration, number called from, protocol if any, what
   the agent said. What the agent said is the plaintiff's narrative; mark it so.
8. **Written contact** — emails sent, to which address, what was attached, whether any reply came.
9. **Notices** — did any warning arrive before the service was cut (SMS, email, letter)?
10. **Concrete harm** — only what happened. Ask open questions; do not suggest harms. "No documented
    loss" is a legitimate entry.
11. **The surrounding invoices** — twelve months if possible, as full PDFs. They show discounts ending,
    plan changes, late fees, payment method per month, and often the protocol numbers of past contacts.
12. **Older matters with the same defendant** — agency complaints, their replies, the agency's opinion,
    and proof of what was paid.
13. **Personal documents** — ID for each plaintiff, proof of address. Marital status and occupation
    are needed for the heading.
14. **Witnesses** — full name, ID numbers and address of anyone who saw an incident. A party cannot be
    a witness.

## Reading what arrives

- `pdftotext -layout file.pdf -` for text PDFs. Collapse whitespace before grepping.
- Replies from companies to agencies often embed the key document (a signed term) as an image:
  `pdftoppm -r 110 -png` into the scratchpad and view the pages.
- Screenshots: read them and write down exactly what is visible, including what is cut off. A call log
  that says "Today" proves no date; say so and ask for a later capture if the user is willing.
- After reading a batch, tell the user what each document proves in one line each, then what it changes.

## The case file (`dados.md`)

Sections that earned their place:

- **Parties and contract** — names, IDs, address, contract number, each line and who uses it.
- **The disputed charge** — a small table if there are competing invoices.
- **How the petition tells it** — one paragraph fixing the narrative rule ("only proven facts; what
  failed inside the company is for the company to explain").
- **Month-by-month table** from the invoices, with payment method per month.
- **What the defendant's own documents admit.**
- **Timeline** — date | fact | proof. A row without proof says "—" in the proof column.
- **Older matter** — one subsection per agency case: what the plaintiff wrote, what the company
  answered, what the agency concluded, in the documents' own words.
- **Decisions** — what was excluded and why; the amounts chosen and on what basis.
- **Still missing** — a checklist. Remove an item when it arrives; do not let it rot.

Correct the file the moment a fact changes. Entries like "Correction (date): the number X belongs to
the co-plaintiff, not a third party" are worth keeping visible.

## Evidence that is usually missing, and what replaces it

| Missing | What covers it |
|---|---|
| Proof that the service was actually cut | Contemporaneous messages, a witness, the complaint emails, and a request that the defendant produce suspension and restoration logs and daily usage |
| The protocol number of a call | The defendant's next invoice often prints recent protocol numbers; request the contact history in the petition |
| An old payment receipt | The user's banks keep statements for years; narrow the window using the defendant's own dates (the suspension date, the agency timeline). Ask for the receipt once more with the narrowed window before giving up |
| The date of an old payment | Never assume the year. Ask the court to have the defendant state when and how that specific charge was settled |
| The screen the consumer saw when accepting an offer | Burden shifts: request the defendant's record of the digital flow and the approval message |

Insisting once, with a narrower window, found the decisive receipt in the original case after the user
had given up. Insisting twice is nagging.

## A co-plaintiff

A relative who uses a dependent line can join. It brings their own damages claim and, if they are
elderly, priority handling. It costs their signature and their presence at every hearing.

- State their harm exactly as it was. If they have another line for calls, the petition says so; the
  defendant has that information and silence about it reads as concealment.
- They can ask for moral damages and benefit from the injunction; restitution and declaratory requests
  belong to the contract holder. Say which plaintiff each request is for.
- One proof of address in the holder's name plus a short signed declaration that the relative lives
  there is enough.
