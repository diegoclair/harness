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

Three more skills (`confluence-docs`, `jira-tickets`, `social-carousel`) are catalogued as **pending**: they still live in [`diegoclair/skills`](https://github.com/diegoclair/skills) and move here per [the migration plan](./docs/migration-from-skills.md). `harness list` marks them, and installing one tells you where it is.

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

`--from .` reads the working tree directly, so editing a skill and reinstalling is instant — no release, no symlink workaround.

### Commands

```
harness list [--skills|--agents]      Show every installable artifact
harness install <name>...             Install skills and/or agents
harness install --all                 Everything available
harness install --all-skills          Every skill
harness install --all-agents          Every agent
harness validate [PATH]               Check the artifacts in a repo tree
```

| Flag | Applies to | Meaning |
|---|---|---|
| `--repo OWNER/REPO` | both | Install from a fork |
| `--ref REF` | markdown artifacts | Branch, tag or SHA (default `main`) |
| `--from PATH` | markdown artifacts | Install from a local clone |
| `--version TAG` | release-backed skills | Pin a release tag, one skill at a time |

Unknown names are rejected before anything is installed, so a typo never leaves a half-applied selection. Re-running upgrades in place.

**Uninstall:** delete `~/.claude/skills/<name>/` or `~/.claude/agents/<name>.md`.

## How it works

Artifacts come in two flavours, and only the installer needs to care:

| | `source` | `release` |
|---|---|---|
| Payload | markdown, straight from the repo tree | per-platform zip with a compiled binary |
| Needs CI | no | yes |
| Pinning | `--ref <branch\|tag\|sha>` | `--version <tag>` |
| PATH wiring | none | symlink into `~/.local/bin` |

Markdown artifacts need no build step, so they ship the moment they're merged. Skills that drive a Go CLI resolve their release by tag prefix — GitHub's "latest" pointer is a single value per repository, which a multi-artifact repo can't use.

A source install replaces the skill directory so a renamed file can't linger, but it keeps `bin/` and refuses to overwrite a directory it didn't create.

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
3. `cd installer && make test`.

A markdown artifact needs no *skill* release — it ships from `main` the moment it is merged. The catalog entry does travel in the installer binary, so an installed `harness` picks up a brand-new artifact only after the next `harness-v*` release; running `go run ./installer` from a clone sees it immediately.

## License

[MIT](./LICENSE) © 2026 Diego Clair
