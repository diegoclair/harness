# Migrating `diegoclair/skills` into `harness/skills/`

Executable plan. **Nothing here has been run.** It moves the three Atlassian/social skills out of `diegoclair/skills` and into this repo without breaking a single existing user.

**Governing rule:** the old repo keeps working throughout. No force-push, no history rewrite, no deleted tags or releases. Deprecation is a pointer, never a deletion.

---

## Measured migration surface

Taken from `diegoclair/skills` at `f2c2afa`:

| Surface | Count | Where |
|---|---|---|
| Go modules | 6 | `installer`, `pkg/release`, `pkg/atlassian`, `{confluence-docs,jira-tickets,social-carousel}/cli` |
| `diegoclair/skills` in `*.go` | 77 | imports + constants |
| `go.mod` module paths | 6 | one per module |
| **`Makefile` `AUTH_PKG`** | **2** | `confluence-docs/cli/Makefile:24`, `jira-tickets/cli/Makefile:24` |
| Self-update constants | 3 | `*/cli/cmd_update.go` — `repoOwnerRepo`, `installShURL`, `installPS1URL` |
| Doc/script references | 16 | 4 READMEs, `PRIVACY.md`, 2 CHANGELOGs, 8 install stubs, `install-for-ai.md` |
| Release workflows | 4 | `confluence-v*`, `jira-v*`, `carousel-v*`, `installer-v*` |

Two hazards below are silent — they leave CI green while shipping broken binaries. They are called out where they bite.

---

## Decisions

| | Decision | Status |
|---|---|---|
| **OD-1** | Repository visibility | **Settled: public.** Nothing here is private, and the point is that anyone can use it. The installer needs no auth. |
| **OD-2** | Licence and copyright | **Settled: MIT © Diego Clair.** The old repo's Lybel attribution was stale — that code is no longer Lybel's. |
| **OD-3** | Preserve git history (`git subtree` / `git filter-repo`) or flat copy plus a pointer? | Open — cost vs. provenance. |
| **OD-5** | Branding of the migrated skills in the README | Resolved by OD-2: personal, no company branding. |

## Phase 0 — done

Repo built locally: structure, the three new artifacts, the installer covering both payload kinds, README, migration plan. Unpushed. Old repo untouched.

## Phase 1 — publish the harness

```bash
gh repo create diegoclair/harness --public --source=. --remote=origin
git push -u origin main
git tag harness-v0.1.0 && git push origin harness-v0.1.0   # triggers release-installer.yml
```

Pre-publish manual checks (tooling not available locally):

```bash
docker run --rm -v "$PWD:/repo" -w /repo rhysd/actionlint:latest -color
pwsh -NoProfile -Command "[void][System.Management.Automation.Language.Parser]::ParseFile('install.ps1',[ref]$null,[ref]$null)"
```

Smoke test, sandboxed:

```bash
SB=$(mktemp -d)
env -i HOME="$SB" PATH=/usr/bin:/bin sh -c \
  'curl -fsSL https://raw.githubusercontent.com/diegoclair/harness/main/install.sh | sh -s -- install dev-loop'
find "$SB/.claude" -type f       # expect the skill AND unbiased-reviewer.md
```

**At the end of Phase 1 the three new artifacts are publicly installable and the Atlassian skills have not moved.** Nothing is broken; the old repo still serves them.

## Phase 2 — move the legacy skills *(needs OD-3)*

Each step is independently verifiable. Do them in order.

**2.1 History** — settle OD-3. To preserve:
```bash
git remote add old git@github.com:diegoclair/skills.git && git fetch old
git merge -s ours --no-commit --allow-unrelated-histories old/main
git read-tree --prefix=skills/ -u old/main
```
Rollback: this is a local commit until pushed — `git reset --hard` is forbidden by the governing rule, so branch first and abandon the branch instead.

**2.2 Files** — `<skill>/` → `skills/<skill>/`; `pkg/atlassian` joins `pkg/` at the root; drop the old `installer/` (this repo's supersedes it).

**2.2b Catalog entries** — add the three skills to `catalog` in `installer/manifest.go` with their `TagPrefix` (`confluence-v`, `jira-v`, `carousel-v`) and `VersionEnv`. Nothing else in the installer changes: each arrives carrying `cli/`, which is how it declares it drives a binary. `TestEveryCatalogEntryExistsInTheTree` fails until their files are in place, so the entry cannot be added early by mistake.

**2.3 Module paths — ⚠ silent-failure hazard**

Rewrite `github.com/diegoclair/skills/…` → `github.com/diegoclair/harness/…` across **6 `go.mod` + 77 Go references + the 2 `Makefile` `AUTH_PKG` lines**, and update `go.work`.

> **The Makefiles are the trap.** `confluence-docs/cli/Makefile:26` injects the bundled OAuth secret with
> `-X $(AUTH_PKG).DefaultClientSecret=…`. **The Go linker ignores an unknown `-X` symbol without any error.**
> Measured:
> ```
> -X github.com/diegoclair/skills/pkg/atlassian/auth.DefaultClientSecret=X  → build ok, secret ""
> -X <correct-path>.DefaultClientSecret=X                                   → secret set
> ```
> Miss them and `go build ./...`, `go test ./...` and CI all stay **green** while every released binary ships an
> empty `DefaultClientSecret` (`auth/login.go:38` → `authorizer.go:190` omits it) → **`login` broken for every user.**

Verify — the build alone is not proof:
```bash
go build ./... && go test ./...
cd skills/confluence-docs/cli && make build ATLASSIAN_OAUTH_CLIENT_SECRET=probe
./bin/confluence-docs login --check-bundled-secret   # must report a secret is present
```

**2.4 Self-update constants** — the 3 `repoOwnerRepo` + `installShURL` + `installPS1URL` in `*/cli/cmd_update.go`.

**2.5 CI** — retarget `release-<skill>.yml` paths to `skills/<skill>/cli`; keep the tag prefixes identical; exactly one workflow may carry `make_latest: true`.

**2.6 Secret — before any skill release.** Recreate `ATLASSIAN_OAUTH_CLIENT_SECRET` in this repo's Actions secrets. Released binaries get it via `-ldflags`; without it they ship without the bundled OAuth app.

**2.7 Re-release each skill.** Tags and releases are per-repo and **do not migrate**. Cut a fresh tag per skill here (`confluence-v*`, `jira-v*`, `carousel-v*`), or `FindLatestByPrefix` finds nothing and `harness install confluence-docs` fails.

```bash
SB=$(mktemp -d); env -i HOME="$SB" PATH=/usr/bin:/bin harness install --all   # all 6 land
```

**2.8 Doc/script references** — the 16 files, plus this repo's README catalogue *(OD-5)*.

## Phase 2.5 — the bridge ⚠ **must happen while the old repo is still writable**

`cmd_update.go` hardcodes the old repo *and* re-fetches `install.sh` from its `main` at runtime (`curl -fsSL $installShURL | bash`). Archive first and every already-installed CLI is **permanently frozen**, reporting "up to date" (`cmd_update.go:57-59`) with no way to fix it — archived repos are read-only.

1. On the old repo's `main`, repoint `install.sh`, `install.ps1` and the three `*/install/install.{sh,ps1}` at `diegoclair/harness`.
2. Cut **one final bridge release per skill** in the old repo, whose binary carries the new constants.
3. Verify: an old installed binary running `<skill> update` lands on a harness release.

Only then continue.

## Phase 3 — retire the old repo

- README banner pointing here, with the new one-liner.
- GitHub **Archive** (read-only). Releases stay downloadable, so old installs and old one-liners keep working.
- **Never** delete tags or releases, never force-push, never flip it to private.

---

## Post-migration checklist

- [ ] `go build ./... && go test ./...` green across the workspace
- [ ] A built CLI reports a non-empty bundled OAuth secret (2.3)
- [ ] `ATLASSIAN_OAUTH_CLIENT_SECRET` exists in this repo's Actions secrets
- [ ] Exactly one release workflow carries `make_latest: true`
- [ ] `harness install --all` in a sandboxed `HOME` installs all six artifacts
- [ ] `harness validate .` reports six artifacts, no problems
- [ ] Old repo's `install.sh` points here, bridge releases cut, only then archived
- [ ] No reference to `diegoclair/skills` remains outside `docs/` and CHANGELOGs
- [ ] `installer/manifest.go` lists all six artifacts and `go test ./installer/...` is green
