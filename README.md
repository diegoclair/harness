# harness

> The rigging around the model — Claude Code **skills** and **agents**, installable into any project with one command.

[![Claude Skills](https://img.shields.io/badge/Claude-Skills%20%2B%20Agents-6E56CF)](https://docs.claude.com/en/docs/claude-code/skills)

Two kinds of artifact live here, and the installer knows the difference:

- **Skills** → `~/.claude/skills/<name>/` — a process Claude follows.
- **Agents** → `~/.claude/agents/<name>.md` — a role Claude dispatches.

Skills compose agents. Installing a skill pulls in the agents it needs.

## What's inside

### Agents

| Agent | What it does |
|---|---|
| **`unbiased-reviewer`** | Adversarial reviewer that never saw the implementer's reasoning. Proves the tests aren't hollow (mutation testing), writes its own adversarial fixtures, runs integration against real infrastructure when mocks can't prove it, and returns APPROVE/REJECT with anchored evidence. Read-only on production code. |

### Skills

| Skill | What it does |
|---|---|
| **`implementation-plan`** | Turns a large or fuzzy objective into a bulletproof, executable spec. Recons the code verifying every claim against the source, pre-consolidates settled decisions, marks open ones as explicit stop-points, then runs an adversarial review of the draft spec to catch blockers before a line is written. |
| **`dev-loop`** | Builds a non-trivial feature through a quality gate: fresh implementer → `unbiased-reviewer` → the parent decides → [corrector → re-review]. Not for one-liners. |

Both skills dispatch `unbiased-reviewer`, so it comes along automatically.

| Skill | What it does |
|---|---|
| **`confluence-docs`** | Search, create, classify and update Confluence Cloud pages in natural language. Ships a Go CLI returning page digests and single sections instead of full ADF bodies — far cheaper in tokens than the raw MCP path, which stays as a fallback. |
| **`jira-tickets`** | Read, create, transition and link Jira issues without burning context. Shares `pkg/atlassian` with `confluence-docs`, so one login covers both. |
| **`social-carousel`** | Generates Instagram and LinkedIn carousels from a small YAML brief, rendered locally through headless Chrome. Ships design presets, layout templates, and a linter of research-backed rules that blocks a bad render. |
| **`quality-gate`** | Gates a delivery on what a reviewer otherwise checks by hand: comments that narrate behavior instead of stating purpose, declarations described instead of constrained, blocks that already exist elsewhere in the repo, functions holding two rules, and domain logic leaking into a handler or a query. Baseline-frozen, so it blocks new violations without demanding a clean repo. |

These four drive a Go CLI, so installing them also puts a binary on your PATH.

## Install

```bash
# see what's available
curl -fsSL https://raw.githubusercontent.com/diegoclair/harness/main/install.sh | sh

# pick what you want — skills and agents mix freely
curl -fsSL .../install.sh | sh -s -- install dev-loop implementation-plan

# or take a whole category
curl -fsSL .../install.sh | sh -s -- install --all-skills
curl -fsSL .../install.sh | sh -s -- install --all
```

A bare pipe with no arguments lists the catalog and exits — it never writes into `~/.claude` for a command you didn't type.

**Windows (PowerShell):** `iwr -useb https://raw.githubusercontent.com/diegoclair/harness/main/install.ps1 | iex`. To choose artifacts, download the script first and pass arguments to it.

> **Before the first release is published**, the one-liner has nothing to fetch. Use the clone path below — it works today and needs only Go.

### From a clone

```bash
git clone https://github.com/diegoclair/harness.git && cd harness
go run ./installer list
go run ./installer install --from . dev-loop
```

`--from .` reads the working tree directly, so editing a skill and reinstalling is instant — no release, no symlink workaround. A skill that ships a `cli/` is the exception: its binary only exists in a release, so `--from` still fetches that release for it.

### Commands

```
harness list [--skills|--agents]      Show every installable artifact
harness install <name>...             Install skills and/or agents
harness install --all                 Everything available
harness install --all-skills          Every skill
harness install --all-agents          Every agent
harness validate [PATH]               Check the artifacts in a repo tree
```

| Flag | Meaning |
|---|---|
| `--repo OWNER/REPO` | Install from a fork |
| `--ref REF` | Branch, tag or SHA of the repo tree (default `main`) |
| `--from PATH` | Install from a local clone instead of downloading |
| `--version TAG` | Pin a release tag — only for a skill that ships a binary, one at a time |

Unknown names are rejected before anything is installed, so a typo never leaves a half-applied selection. Re-running upgrades in place.

**Uninstall:** delete `~/.claude/skills/<name>/` or `~/.claude/agents/<name>.md`.

## How it works

**One installer, one pipeline.** Nothing about an artifact's installation is declared in a config: it is discovered from the artifact itself.

```
fetch the repo tree (a git ref, or your local clone)
   │
   ├─ agent ..................... copy agents/<name>.md
   │
   └─ skill
        ├─ ships a cli/ ? ── no ─→ install its files from the tree.  Done.
        │
        └─ yes ─→ a compiled binary only exists in a release archive,
                  so the payload is that release: install its files,
                  then the binary, symlink it onto PATH, run the
                  skill's own verification and post-install hooks.
```

A skill declares that it drives a binary by **shipping a `cli/` directory**. The expectation is then enforced in both directions: a skill with `cli/` whose release carries no binary fails **before anything is written**, rather than leaving a skill Claude Code would load with every command missing; and a skill that stops shipping `cli/` has its stale binary and PATH link removed instead of leaving an old executable runnable forever.

Release versions resolve by tag prefix, because GitHub's "latest" pointer is a single value per repository and this repo ships several independently versioned skills.

Installing replaces the skill directory so a renamed file can't linger, refuses to overwrite a directory it didn't create, and backs up a hand-written agent before replacing it. `cli/` is a build input and is never installed.

## Layout

```
harness/
├── skills/
│   └── <name>/
│       ├── SKILL.md          # frontmatter + instructions
│       ├── reference/        # (optional) workflows, specs
│       ├── cli/              # (optional) Go CLI the skill drives
│       └── install/          # (optional) one-liner stubs
├── agents/
│   └── <name>.md             # frontmatter + role
├── installer/                # the `harness` CLI
├── pkg/                      # shared Go packages
└── .github/workflows/        # one release-<component>.yml per releasable component
```

## Contributing

Artifacts here must work for anyone who installs them. PR rules:

- **Company-agnostic.** No data specific to any company hardcoded in a skill body, in `reference/`, or in CLI source — no people, page IDs, instance URLs, product lists.
- **Configurable defaults.** If an artifact needs an instance-specific value, expose it via a setup wizard, frontmatter, or an environment variable, and document the override.
- **Query at runtime, don't cache in the repo.** For data that changes (taxonomies, indexes, entity lists), read the external system at runtime. This is what keeps the repo timeless and safe to publish.
- **English body.** Instructions in English for reasoning quality; the agent replies in whatever language the user wrote in.

### Conventions

- `name`: lowercase with hyphens, max 64 characters, and it must match the directory (skills) or filename (agents).
- `description`: max 1024 characters, including the triggers that activate the artifact.
- **Quote a description containing `: `** — unquoted, YAML reads it as a nested mapping. `harness validate` catches this.
- References use relative paths (`reference/foo.md`), never absolute URLs.

Run `go run ./installer validate .` before opening a PR.

### Adding an artifact

1. Create `skills/<name>/SKILL.md` or `agents/<name>.md`.
2. Add it to `catalog` in `installer/manifest.go`, with `Requires` if it dispatches an agent.
3. If it drives a CLI, add `skills/<name>/cli/`, set `TagPrefix` on its catalog entry, and add a `release-<name>.yml` workflow. The installer needs no code change — `cli/` is what it looks for.
4. `cd installer && make test`.

A skill without a `cli/` needs no release: it ships from `main` the moment it is merged. The catalog entry travels in the installer binary, so an already-installed `harness` sees a brand-new artifact after the next `harness-v*` release; `go run ./installer` from a clone sees it immediately.

## Credentials

The Atlassian skills share one login via `~/.config/atlassian/credentials`:

```bash
confluence-docs login          # browser OAuth; tokens refresh themselves
confluence-docs setup --check  # validate what is configured
jira-tickets setup --check     # reuses the same credentials
```

Uninstalling a skill never touches that file. Remove `~/.config/atlassian/` only when you are done with **both** Atlassian skills.

## Credentials

The Atlassian skills share one login via `~/.config/atlassian/credentials`:

```bash
confluence-docs login          # browser OAuth; tokens refresh themselves
confluence-docs setup --check  # validate what is configured
jira-tickets setup --check     # reuses the same credentials
```

Uninstalling a skill never touches that file. Remove `~/.config/atlassian/` only when you are done with **both** Atlassian skills.

## Credentials

The Atlassian skills share one login via `~/.config/atlassian/credentials`:

```bash
confluence-docs login          # browser OAuth; tokens refresh themselves
confluence-docs setup --check  # validate what is configured
jira-tickets setup --check     # reuses the same credentials
```

Uninstalling a skill never touches that file. Remove `~/.config/atlassian/` only when you are done with **both** Atlassian skills.

## License

[MIT](./LICENSE) © 2026 Diego Clair
