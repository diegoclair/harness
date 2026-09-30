## A fact lands after its effect; an error never becomes a value

- **A fact is written after the effect it records, or in the same transaction under a condition.** "Paid",
  "answered", "published", "sent" written before the act turns every failure between the two into a lie
  nothing walks back: the invoice reads paid while the plan never advanced, the question reads answered
  while nothing reached the buyer.
- **When the effect is external and cannot share the transaction,** take a claim that is named as a claim
  ("publishing", "in flight") on the row that owns the fact, act, then write the result conditioned on
  that claim (`… WHERE status = 'publishing'`); every failure path releases the claim by an update. The
  claim and the result are two facts, never one flag doing both jobs.
- **Two writes that decide one outcome share one transaction,** and the second checks the state the first
  read (compare-and-set on the period, the status, the version), so a redelivery cannot apply it twice or
  to the wrong row.
- **Treating an error as a non-error is not forbidden, and strongly not recommended.** When one case
  really is ordinary, give it a specific named error at its source, and the caller checks for exactly
  that one (`errors.Is`) and ignores it explicitly, where the reader sees the decision. **Never turn a
  generic error into a zero value a caller reads as an answer**: a read that failed is not "none found",
  `0` or empty, and logging a warning before carrying on with the zero value turns a database blip into
  a wrong invoice.
- **An error is logged once, where it is handled** — and handling means returning it, retrying it, or
  recording why the flow continues without it.
