# Engineering judgment

- **Fix the cause, not the symptom.** A new rule about the symptom changes what users can do: it goes as a question, not as part of the fix.
- **Simple before dynamic**: the fix is the minimum that solves the case seen; per-category or per-vendor rules only when a fixed generous limit does not.
- **Don't build or prove for scenarios that almost never happen**; a reviewer's "seen, not proven" list is information, not a queue. A feature with no real case is not built.
- **Fix in the owner, not the consumer**: a defect another layer owns is fixed there; never ship a known-wrong patch in the consumer without asking.
- **Search before creating, across the repo's border**: the code, the declared dependencies (`go.mod`, `package.json`) and what our own libraries already offer. Whoever writes the spec searches too — the implementer will not disobey a spec to find what exists.
- **Names say what they do at the call site**; a doc-comment explaining a function means the name is wrong. One question, one function, the caller decides: no wrapper that only supplies a default or remaps an error.
- **Traversal is separate from decision**: no recursive closure (`var walk func`) that walks and decides at once.
