# The git index is the user's

- **A staged file means "the user reviewed this version".** The user stages while reviewing, even while an agent is still editing.
- **No agent touches the index**: no `git add`, `git reset`, `git restore --staged`, `git rm --cached`, `git stash`, `git checkout -- <file>`, `git clean` or `git mv` (rename is a plain `mv`), not even chained behind `&&`. Reading (`status`, `diff`, `show`, `log`) is free.
- **The index changing mid-session is the user reviewing live**, `MM` included: never investigate or report it. Always report **which already-staged files you changed**, because the reviewed version is no longer the one on disk. A staged file is no obstacle: change or remove it as the work needs, and say so.
- **Commit, tag and push only with the user's explicit authorization for that delivery**; "take the decisions from here" delegates design, never a commit. Without it, the report says "ready for you to review" and lists new untracked files.
- When authorized and the push splits into several commits, each group is `git commit --only -- <paths>`, so the user's index never rides along; check each with `git show --stat` before pushing. A file another session also changed is asked about before it enters a commit.
- Commit message: short subject, body only with the why the diff does not show.
- Exception: inside the `harness` repo itself, the agent stages, commits and pushes its own changes.
