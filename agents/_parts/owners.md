## One owner per business question — find it before a line is written

Every business answer — "which variants does this listing sell?", "may this user do it?", "how much is
owed?" — has one owner, and everyone else asks it. Two implementations of one answer agree only until the
rule changes, and no clone detector sees the second one, because it is always rephrased: another loop,
another separator joining the same values, a `CASE` in SQL beside the entity method that already decides it.

**Name the question before the code.** For every function or type added or changed, say in one sentence,
in the domain's words, which business question it answers. Then look for that question's owner:

- **Search for the behaviour, never for the name you had in mind.** Grep what the answer reads and
  produces — the status value, the column, the field pair, the separator, the unit — because the existing
  owner lives under another name.
- **Search every layer the rule can hide in.** The same rule turns up as an entity method and as a SQL
  predicate (a `WHERE`, a `CASE`, a `COALESCE` default), in a service and in the query that feeds it, and
  on both sides of a vendor boundary — the adapter and the domain. A search that read one layer found
  nothing.
- **Cross the repo border:** the project's own libraries are code too.
- **Found: ask it.** Never recompute an answer that has an owner, never re-read config the owner already
  reads, never turn what a port returned into a decision of your own. When the owner cannot answer the
  case at hand, it is missing a question: extend the owner, or bring it back — never write the second
  answer beside it.
- **A rule the database must apply lives twice by necessity, so it is named twice:** a SQL fragment named
  as the half of its entity method, and a test that feeds both the same rows and proves they agree. An
  inline predicate that restates an entity rule is a second owner.
- **Not found:** the owner is born where the project's `AGENTS.md` puts rules of its kind — a pure rule
  over our own data on the entity — and the next place that needs it asks it.

**The search is written down, one line per function or type born:**
`question → searched (the greps, the layers) → reused <symbol> | extended <symbol> | born at <symbol>`.
A search not written down did not happen.
