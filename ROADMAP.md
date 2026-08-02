# Roadmap — skills monorepo

Cross-cutting work. Per-skill backlogs live in `<skill>/ROADMAP.md`.

## Backlog

### Rework install & setup around independent skills

The install and setup flow was shaped when `confluence-docs` was the only
skill, and still leaks that history. Installing one skill should be a complete,
self-explanatory operation regardless of which others are present.

Known symptoms:

- `pkg/atlassian/setup` defaults to `ProductConfluence` and to the
  `confluence-docs` skill name. Callers must remember to set both in `main()`;
  forgetting either fails silently with Confluence behaviour. Making the
  identity a required argument would turn that into a compile error.
- The shared installer reports credential state from the `setup --check` exit
  code alone and discards its stderr, so precise diagnoses are replaced by a
  generic "Not yet configured. Run setup" — currently misleading for a valid
  grant on a site that lacks the product (`pkg/install/install.sh`).
- "Configured" is a single verdict, but the pieces are independent:
  credentials are shared across skills, while active space is Confluence-only.
  Installing a second skill should reuse what already exists and only ask for
  what is genuinely missing.
- Exit codes conflate cases: a valid grant on a site without the product
  returns `ExitNoCreds`, same as having no credentials at all.

Not urgent — no external users yet.
