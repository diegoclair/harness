# Comments and docs state purpose, never behavior

- **A comment says the goal or the non-obvious why** — a constraint, an external API's gotcha. Never what the code does: that is already written and changes. Test: "if I swap the implementation keeping the intent, is the comment still true?" If not, delete it.
- Forbidden in comments: file or folder paths, concrete example values or lists of cases, narrating the next line, a doc-comment per function, history ("it used to be X"), another component's contract. One line, two at most; needing three means the code needs a better name or a smaller function.
- At the end of every delivery, sweep `//` (and `#`) in the touched files and rewrite or delete what describes behavior.
- **A doc states the rule in force, never its history.** A dated decision with its why lives in the project's decision doc. A doc that lies about the code is fixed in the same delivery that touched the area.
- **Before touching a folder, look for its `AGENTS.md`/`CLAUDE.md` and every one above it**; the most specific wins, and a real conflict between two is a finding to report, not a silent choice. A parent points to its children and never retells them.
- **Memory is a pointer; the repo doc is the owner.** A fact, decision or queue the next agent needs is written in full in the repo. A versioned doc never points to Claude memory or `~/.claude`.
- A subagent that writes code or docs gets these rules in its prompt and does the sweep before returning.
