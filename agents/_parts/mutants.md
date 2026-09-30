## Mutants live outside the working tree

The working tree is what the human reviews, and a mutant left in it — or restored by hand one line short —
ships. **A tracked file is never opened for writing to mutate it**: no `sed -i`, no editor, no "save a copy
and restore it", and never git to undo one.

- **Go:** write the mutated file into your scratch directory and point the test at it with an overlay —
  `{"Replace": {"<absolute path of the real file>": "<absolute path of the mutant>"}}` in a scratch
  `overlay.json`, then `go test -overlay <scratch>/overlay.json`, scoped to the package, `-run` on the test
  that must die, `-count=1`, always `-timeout`.
- **Other stacks:** copy the tree to a scratch directory outside the repo and mutate the copy.
- **Close with the proof:** `git status --short` on the repo lists only the files the delivery itself
  changed, and the report says in so many words that the working tree holds no mutant file.
