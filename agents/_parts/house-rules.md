## Before the first line

- **Read the `AGENTS.md` of every folder you will touch, and of the folders above it.** The most specific
  one wins. A real conflict between two is a finding: report it, never choose silently.
- **If the lead asked for recon, recon and stop.** Return the real state of the code, what the objective
  demands, and every item that needs a decision — each with your recommendation. Build nothing yet.

## Decisions — the rule that prevents a rewrite

**Implementation you decide and record; product you bring back before implementing.**
- *Implementation* is the how: names, structure, where code lives, which existing pattern to reuse.
- *Product* is the what and the who: who may do what, what the user sees, deadlines, money, what a state
  means — or anything the spec does not cover that changes the outcome for someone.

**Meet a product decision mid-work: stop, bring the item with your recommendation, and wait.** Do not
implement your choice so it surfaces as a finding at the end. Asking costs a message; redoing costs the
delivery. The same applies to a spec instruction that looks like it frees, blocks or charges more than its
stated case — ask "who else does this reach?" before writing it.

## Git — the index belongs to the human

A staged file means "the human reviewed this version". **Never** run `git add`, `git reset`,
`git restore --staged`, `git rm --cached`, `git stash`, `git commit`, `git mv` or `git push`; rename with
plain `mv`. **The ban holds inside a compound command too**: a `stash` chained behind an `&&` to get a
clean tree for one step empties the worktree the human was reviewing, and you are the only one who knows. If the index changes while you work, the human is reviewing live — do not investigate it. **Do
report which already-staged files you changed**, because the version they reviewed is no longer on disk.

## Search before you create

- **No function, helper, hook, component, constant or config key is born before you search for one that
  already does it** — and the search crosses the repo border: the project's own libraries are code too.
- **Search for the behaviour, not the name you had in mind.** A new window, deadline or config value is
  often an existing one under another name; two knobs governing one behaviour is a defect.
- **Finding something is not the end of the question.** Open it and judge whether it is worth its cost —
  a wrapper that adds hops and parameters for one log line is not a reason to reuse. If half the codebase
  already bypasses it, there is no convention to preserve.
- If it does not exist and a second place will need it, it is born shared, with its line in that repo's
  `AGENTS.md`, in the same delivery. Report what you searched and what you reused.

## One owner per business question

Every business answer — "may this user do it?", "how much is owed?", "is this listing live?" — has one
owner, and everyone else asks it. **Never recompute an answer that has an owner, never re-read config the
owner already reads, and never reinterpret what a port returns into a decision of your own.** Two
implementations of one answer agree only until the rule changes, and no linter sees it, because different
code does not look like duplication. If your task seems to need a second answer, the owner is missing a
question — say so.

## Code

- **Names say what they do**, read as a sentence at the call site, and never name the mechanism instead of
  the question. A name carries the intent of the call, never the condition of the query behind it, and an
  accessor drops the suffix its type already says. A short name that forces the reader to the constructor
  is a defect. A name that needs a
  doc-comment to be understood asks to be renamed. **No `Of` suffix**: a mapping between types is `toX`, a
  calculation is a verb, and the gate errors on an `…Of` that takes a context or returns an error. **One
  verb per gesture** — the project's `AGENTS.md` fixes the verb for each gesture and bans its synonyms, and
  its list wins over your habit. **Before returning, sweep every function and type
  you added: read only its name as a caller would, and if you cannot say what happens, rename it; if it
  carries a comment explaining what it does, the comment is the symptom — rename, then delete the comment.**
- **No comment is the default; write one only when deleting it would lose something the code cannot say.**
  Every comment the gate later makes you cut or reword is a round of rework, so decide at writing time. A
  comment is allowed to be three things: the purpose the code cannot show, a non-obvious constraint (unit,
  invariant, format, an external contract, the spec or incident it encodes), or a gotcha in someone else's
  API. Test before typing it: *if the implementation changed and kept its intent, would this line still be
  true?* If not, it is behaviour — don't write it.
  - **Struct field and interface method: none.** The name and signature are the contract. The rare exception
    is one line carrying a constraint the name cannot (a unit, an invariant, an external format) — never what
    the member is or does. Two lines on a member is an error with no tolerance.
  - **Function and type doc: none by default**, at most two lines, and only the purpose or the constraint a
    caller must know. A doc that says what the function does means the name failed: rename it.
  - **Inside a body: none**, unless an ordering requirement, a third-party quirk or the reason for an unusual
    branch — one line. A comment that labels a section is a function asking to be extracted and named.
  - **Never:** narration of the lines below, a file path, an example value, "e.g.", a date or a person's
    name, history ("now", "used to", "no longer"), or another component's behaviour.
  - **Rewording a described comment into "so that…" words while it still paraphrases the code is still
    behaviour.** When the gate flags a comment, the first answer is deletion; rewrite only when a real
    constraint remains, and then state that constraint alone.
  - English; user-facing strings in the product's language. **Before returning, run `quality-gate check` on
    the files you touched:** zero comment errors, and each comment warning either deleted or kept for a
    constraint you can name. The gate's ceilings are the upper bound, never a budget to fill.
- **Tests never reshape production.** No field exported for a test, no test flag, no branch that only runs
  in tests, no sleep to synchronise. **A required dependency validates in its constructor and returns an
  error** — accepting nil so a test compiles trades a boot failure for silent loss in production. A
  registration that tolerates never being called is the same defect wearing a friendlier face: what the
  product needs is required where the thing is built, not hoped for at the first read.
- **Money in integer minor units**, never float. On a path delivered at least once, write totals rather
  than deltas, so a redelivery cannot count twice. When an external call and our own record must both
  happen, decide their order deliberately and say what a failure between them leaves behind.
- **Regenerate file by file, never the target that wipes the folder.** While other agents are working, a
  generator that clears and rebuilds a shared directory (mocks, clients, types) hands them a broken build
  they did not cause, and they will debug it as if it were theirs.
- **Anything that sends comes back with its cadence.** For an e-mail, a message or a push, the report
  carries a table `trigger → worst case per recipient per month`, and the code sends once per recipient
  per event, grouped. A send whose worst case nobody computed is how a feature becomes spam.
- **Do not infer an external system's behaviour.** What you did not observe, mark as unverified.
- **A doc that lies about the code is a finding**: fix it in the same delivery. A `AGENTS.md` states the
  rule in force, never history, and never points to local memory.

## Proof — scoped, and across the seams

- **Proof covers everything you changed and the existing behaviour that depends on it.** A change is not
  proven by its own tests passing; it is proven when what already worked still works. Run the project's
  quality check before returning, if it has one.
- **A seam is where correct pieces break together.** A field missing from a struct literal compiles, vets
  and passes per-package tests; a gate can block the very action that starts a flow. **When you wire a
  dependency, add a registration, or change who may do what, prove it across the seam** — the composition
  root, or a journey through the real pieces — not only each piece alone.
- **Make the test able to fail.** When the change guards money, access or a destructive act, check that
  removing the guard makes a test fail.

### How to prove fast without proving less

The project's `AGENTS.md` names the exact commands; these are the principles they serve. Slow validation is
almost never thoroughness — it is the same wide run repeated.

- **Iterate narrow, close wide, once.** While working, run only the tests of what you are changing. When the
  work is done, run **one** complete pass over the blast radius: the units you changed **and every unit that
  depends on them**. Never re-run the wide pass after each small fix — fix, re-run the narrow test, and
  close with a single wide pass.
- **The blast radius decides how wide.** A leaf change reaches its own unit and its dependents. A shared
  contract, a shared utility, the composition root or a schema change reaches far enough that the whole
  suite is the honest pass — once, at the end.
- **Serialise only what shares state.** Tests that share a database, a port or a directory run one at a
  time; everything else runs in parallel. Forcing the whole run serial is the most common cause of a
  twenty-minute validation for a three-file change.
- **Heavy tools only where they prove something the fast tests cannot:** containers for the code whose
  queries changed, mutation for the guards that protect money, access or a destructive act — never across
  the whole diff.
- **Build and static analysis clean** before returning.
