# SPEC — `diegoclair/harness`: umbrella repo for Claude Code skills + agents

**Status:** v2 — adversarial review applied (1 REJECT round: 3 BLOCKER, 6 HIGH, 8 MEDIUM, 5 NIT, all closed)
**Date:** 2026-08-14
**Objective:** create `diegoclair/harness` — a clonable/installable umbrella of Claude Code **skills AND agents** — and produce an executable migration plan for `diegoclair/skills` into `harness/skills/`.

---

## 0. Verified state (recon — every claim checked against SOURCE, not docs)

Old repo cloned to `<scratch>/recon/skills-old` (`git@github.com:diegoclair/skills.git`, HEAD `f2c2afa`, branch `main`, clean, **read-only all session**).

> The adversarial review independently re-verified this entire section and found it sound. Only §0.5's composition prose was corrected (M5) and §0.3 gained an anchor (N4).

### 0.1 The installer — what it actually does

| Claim | Verdict | Evidence |
|---|---|---|
| Selective install (`install a b c`) | **TRUE** | `installer/main.go:81-145` (`runInstall`), `:149-177` (`selectSkills`) |
| `--all` installs everything | **TRUE** | `main.go:98-99`, `:151-156` |
| `--version TAG` pins, single skill only | **TRUE** | `main.go:106-111`, guard at `:125-128` |
| Unknown name rejected **before** any install | **TRUE** | `main.go:165-169` — whole selection fails, no half-applied state |
| `--repo OWNER/REPO` for forks | **TRUE** | `main.go:100-105`; default `install.go:18` |
| Installs to `~/.claude/skills/<name>/` | **TRUE** | `platform.go:56-62`; `CLAUDE_HOME` override `:44-53` |
| Idempotent, upgrades in place | **TRUE** | `installBinary` temp+rename (`install.go:229-242`) survives ETXTBSY on self-update |
| Version by **tag prefix**, not `/releases/latest` | **TRUE** | `install.go:107` → `pkg/release/release.go:FindLatestByPrefix`; GitHub's "latest" is one pointer per repo |

**Empirical proof (executed, sandboxed `HOME`, real network, real releases):**

| Test | Command | Result |
|---|---|---|
| A — one skill | `install jira-tickets` | ✅ 0 — resolved `jira-v0.4.1`, installed to `<HOME>/.claude/skills/jira-tickets/{SKILL.md,reference/,bin/}`, symlink + `.profile`, verified `v0.4.1` |
| B — unknown name | `install jira-tickets bogus-skill` | ✅ 2, nothing installed |
| C — `--version` + 2 | `install --version jira-v0.4.1 a b` | ✅ 2, explicit message |
| D — no name | `install` | ✅ 2, lists available |
| E — pin + reinstall | `install --version jira-v0.4.0 jira-tickets` | ✅ 0, downgraded in place (idempotent) |
| F — multi-select | `install jira-tickets social-carousel` | ✅ 0, both |
| Unit tests | `go test ./...` in `installer/` | ✅ `ok` |

> **Sandbox rule:** override `HOME`, never just `CLAUDE_HOME` — `userBinDir` (`platform.go:65-71`) honours only `os.UserHomeDir()`, so a `CLAUDE_HOME`-only sandbox still overwrites the real `~/.local/bin`.

### 0.2 The three blocking gaps

1. **No agents slot.** `skillDir` (`platform.go:56-62`) is the only destination; `~/.claude/agents/` appears nowhere. Agents are undeliverable today.
2. **A binary is MANDATORY.** `installPayload` (`install.go:186-195`) hard-fails `binary %s not found in archive`; `verify` (`:265-271`) then executes `binPath --version`, fatal on error. **A markdown-only skill cannot be installed** — which is exactly what the three new artifacts are.
3. **A GitHub release is MANDATORY.** Every path resolves a tag → per-platform zip built by CI. Markdown has no build step; the ceremony blocks day-one publishing.

### 0.3 The three artifacts — and a real defect in two

| Artifact | Path | Frontmatter |
|---|---|---|
| `unbiased-reviewer` (agent) | `~/.claude/agents/unbiased-reviewer.md` (5 410 B) | ✅ valid — `name, description, tools, model`; description 438 chars |
| `dev-loop` (skill) | `~/.claude/skills/dev-loop/SKILL.md` (6 131 B) | ❌ **INVALID YAML** |
| `implementation-plan` (skill) | `~/.claude/skills/implementation-plan/SKILL.md` (6 860 B) | ❌ **INVALID YAML** |

**Defect:** both `description:` are unquoted YAML scalars containing `: ` (colon-space), which ends the scalar and breaks the mapping.

- `dev-loop` — line 3, col 300 (offset 287 **within the description value**): `…WRITE non-trivial code: a change spanning…`
- `implementation-plan` — line 3, col 875 (offset 862 within the value): `…not during. Proven: the adversarial spec…`

`yaml.safe_load` fails on both: `mapping values are not allowed here`.

**Anchored evidence that Claude Code's own loader is lenient and does not truncate:** this session's available-skills listing renders `dev-loop`'s description in full, including the text *after* the offending colon (`…Proven on a 23-deliverable run (0 final rejects…)`). So triggering is unaffected today — the fix matters for **strict consumers** (the official packager rejects both, §0.4), not for current behaviour.

Both skills also carry `version: 0.1.0`, rejected by the official validator (`quick_validate.py` `ALLOWED_PROPERTIES = {name, description, license, allowed-tools, metadata, compatibility}`). → **OD-4**.

**Measured description lengths (limit 1024):** `dev-loop` **1014** (10 chars of headroom — no editorial additions allowed), `implementation-plan` **995**. Double-quoting does not change the *parsed* length (YAML unescapes).

### 0.4 `package_skill.py` — what it actually is

`~/.claude/skills/skill-creator/scripts/package_skill.py`, 136 lines. It validates via `quick_validate.validate_skill`, then zips the folder to `<name>.skill`. Nothing else — no install, no destination, no binaries, no PATH, no version resolution, no update.

Verified by execution:
- `python3 scripts/package_skill.py <skill>` → `ModuleNotFoundError: No module named 'scripts'` (line 17's `from scripts.quick_validate import …` requires `python3 -m scripts.package_skill` from the skill-creator root).
- `-m scripts.package_skill ~/.claude/skills/dev-loop` → **exit 1**, validation failed (the §0.3 defect).
- Control: `-m scripts.package_skill ~/.claude/skills/skill-creator` → **exit 0**, `skill-creator.skill` (72 984 B).

It is a **packager for the `.skill` upload bundle** (claude.ai / Desktop), not an installer.

### 0.5 Migration surface — measured

- **6 Go modules** in `go.work`, all `github.com/diegoclair/skills/…`: `installer`, `pkg/release`, `pkg/atlassian`, `{confluence-docs,jira-tickets,social-carousel}/cli`.
- **77 occurrences** of `diegoclair/skills` in `*.go`.
- **24 non-`.go` files** reference the repo, in three buckets — *all three must be migrated*:
  - **6 `go.mod`** (module paths)
  - **2 `Makefile`** — `confluence-docs/cli/Makefile:24` and `jira-tickets/cli/Makefile:24`: `AUTH_PKG := github.com/diegoclair/skills/pkg/atlassian/auth`, consumed at `:26` via `-X $(AUTH_PKG).DefaultClientSecret=…`. **See the H2 hazard in M2.3.**
  - **16 doc/script files**: 4 READMEs (root, `confluence-docs/`, `confluence-docs/cli/`, `social-carousel/`), `PRIVACY.md`, 2 CHANGELOGs (`confluence-docs`, `jira-tickets` — *not* `social-carousel`, which contains no reference), 8 install stubs (root `install.{sh,ps1}` + 3 skills × `install/install.{sh,ps1}`), `confluence-docs/reference/install-for-ai.md`.
- **3 hardcoded self-update targets** — `repoOwnerRepo = "diegoclair/skills"` at `confluence-docs/cli/cmd_update.go:15`, `jira-tickets/cli/cmd_update.go:15`, `social-carousel/cli/cmd_update.go:18`, each with `installShURL`/`installPS1URL` → `raw.githubusercontent.com/diegoclair/skills/main/…`. `runUpdate` shells out to `curl -fsSL $installShURL | bash`.
- **4 release workflows**, tag-prefix triggered (`confluence-v*`, `jira-v*`, `carousel-v*`, `installer-v*`); exactly one `make_latest: true` (`release-confluence.yml:83`).
- **Build-time secret** `ATLASSIAN_OAUTH_CLIENT_SECRET` injected via `-ldflags` (`release-confluence.yml:40-42`); lives only in repo settings + local `.env.local`.
- **Tags/releases exist only in the old repo.** Releases are per-repo; tags do not follow a file move.
- Licence MIT © 2026 **Lybel**; README branded "Lybel Skills".

### 0.6 Contribution rules that must survive (README:150-178)

Company-agnostic skill content · configurable defaults via setup/env/frontmatter · "Home page is the source of truth" (volatile data queried at runtime, never cached in-repo) · README/CHANGELOG/commits may name the maintaining company · `name` lowercase-hyphen ≤64 · `description` ≤1024 with triggers · body in **English** · relative reference paths.
Leakage check the repo already ships (README:159): `git diff main | grep -iE 'lybel|11C47E|164232'`.

---

## 1. The three questions — answered as DECISIONS

### D1 — Selective install today? **YES. Confirmed in code and proven by execution.**

`install <a> <b>`, `--all`, `--version` (single), `--repo`; unknown names rejected atomically. §0.1 A–F.
**But only for release-backed, binary-bearing skills** (§0.2). The selection *mechanism* is sound → **kept and extended**. The *payload pipeline* grows a second mode.

### D2 — `package_skill.py`: better or complementary? **Complementary. NOT adopted as the install path.**

| | `package_skill.py` | harness installer (Go) |
|---|---|---|
| Output | a `.skill` zip bundle | files in `~/.claude/{skills,agents}/` |
| Purpose | upload bundle for claude.ai / Desktop | install on this machine |
| Binaries / PATH | none | atomic install + symlink + profile |
| Versioning / update | none | tag-prefix resolution, pinning, self-update |
| Agents | unsupported | required (D3) |
| Runtime dep | Python + PyYAML + skill-creator layout | none (static binary) |

Adopting it as the installer is a strict downgrade. **Keep and extend the Go installer.** Two qualified, non-blocking uses of the packager:
- a **future `harness package <name>`** export target for the `.skill` bundle — deferred, out of scope now;
- as a **rule reference**: the harness ships its **own Go validator** with equivalent rules → zero Python dependency. Rules borrowed, code not.

### D3 — Agents coverage + is a CLI right? **YES to both.** CLI is the proven shape (§0.1), a single dependency-free static binary. Extended, not replaced.

**Central architectural decision: ONE pipeline that discovers.**

An earlier draft split installation into two kinds declared by a catalog field (`Source: source|release`). **Rejected by Diego, 2026-08-14:** there must be one installer that handles everything consistently — if an artifact has a binary, install it with the binary; if not, don't. Declaring the pipeline per artifact makes one tool feel like two and creates a flag matrix that only half-applies.

The pipeline, in order:

1. Fetch the repo tree once per run — a git ref (`--ref`, default `main`) or a local clone (`--from`).
2. **Agent** → copy `agents/<name>.md`. Agents are always plain files.
3. **Skill** → does it ship a `cli/` directory?
   - **No** → its payload is the tree subdirectory. Install the files. Done: no `bin/`, no PATH link, no binary verification, because there is no binary.
   - **Yes** → a cross-compiled binary exists only in a release archive, so the payload is that release (one consistent snapshot of files + binary). Install the files, then the binary by atomic rename, symlink it onto PATH, and run the skill's own `--version`, `setup --check` and `postinstall`.

`cli/` **is** the declaration — no catalog flag to forget, and the artifact carries its own truth. Enforced in both directions:
- a skill with `cli/` whose release carries no binary fails **before any file is written** (a broken release must not leave a loadable skill whose commands are all missing);
- a skill that stops shipping `cli/` has its stale `bin/` and PATH symlink removed;
- `cli/` is a build input and is never copied into `~/.claude`.

Consequences, deliberately chosen: the three new artifacts ship day one with no CI and no release; the binary pipeline is preserved bit-for-bit for the Atlassian/carousel skills; `--from <path>` installs from a local clone, which is what makes the repo genuinely "clone it into any project" and replaces the old symlink workaround (old README:166-170).

**Design holes closed by review:**

- **Dependencies are real and modelled (H5).** `dev-loop` (`SKILL.md:38,43,55`) and `implementation-plan` (`:47`) dispatch the `unbiased-reviewer` agent as the core of their loop. `Artifact` carries `Requires []string`; `selectSkills` auto-resolves and prints what it pulled in. Installing `dev-loop` alone must never yield a silently broken skill.
- **Catalog vs. tree (M2).** Hybrid: the baked catalog stays authoritative for `list` (offline UX, summaries) and for all `release` metadata; for `source` **install**, the fetched tree is authoritative — a name present in `skills/` or `agents/` installs even if absent from the baked catalog. So adding a 4th markdown skill needs **no installer release**.
- **`--from`/`--ref` × `--all` (M1).** With a wildcard (`--all`, `--all-skills`), the flag **filters to source artifacts and reports what it skipped** — never silently. With an **explicitly named** release artifact, it is an input error. `--from . --all` therefore works and is the documented local flow.
- **Kind change source→release (M3).** Requires an installer release. Older binaries would install the new version as source (no `bin/`, command silently missing); the release path overwrites the same directory, so leftovers are cleaned on upgrade. `--ref` against a now-release artifact errors telling the user to upgrade the installer.
- **Clean-slate is bounded (M4).** Today's clean slate is scoped to `reference/` (`install.go:201-210`) and never removes the skill root; the source pipeline replaces the whole directory. It **must preserve `bin/`** and must refuse to wipe a directory the harness did not install (marker file). Credentials are unaffected — they live in `os.UserConfigDir()` (`pkg/atlassian/auth/store.go:70`).
- **Bare `curl | sh` no longer means `--all` (M7).** Today `install.sh:54-57` runs `install --all` with no args. In the harness that would silently write into `~/.claude/agents/` for a user who typed nothing → it **prints the catalog and exits 2**.

---

## 2. Inviolable rules

1. **NEVER destructive git** — no `restore`/`reset`/`clean`/`checkout --`, no force-push, no history rewrite, in **either** repo. Check `git status` before assuming state.
2. **NOTHING is pushed without Diego's explicit go-ahead.** `gh repo create` and any push are stop-points. The old repo is **read-only** this session.
3. **The old repo keeps working** — releases, tags, one-liners, installed binaries all remain functional. Deprecation is a pointer, never a deletion.
4. **Verify against source, not docs** — every behavioural claim needs a `file:line` or an executed command.
5. **Sandbox tests with `HOME`**, never bare `CLAUDE_HOME` (§0.1). No test touches the real `~/.claude` or `~/.local/bin`.
6. **Terse comments, English** — only the non-obvious constraint or gotcha. No doc-comment per function, no line narration.
7. **Tests never bend production shape** — no `nil`-accepting required dependency, no `isTest` flag, no export-for-test, no `time.Sleep` for sync. Use fakes/helpers/temp dirs. (The existing package-level `var httpClient` at `install.go:21` is already swappable — use it; do **not** widen production further for tests.)
8. **Commit messages, PR titles, branches, tags in English.**
9. **Path-traversal guard on every archive extraction** — zip already guards (`install.go:145-149`); the tar path needs the equivalent, proven by a fixture containing `../evil`.
10. **Catalog names globally unique across kinds** — a skill and an agent may not share a name, or `install <name>` is ambiguous. Enforced by test.
11. **`bin/` survives only while the payload still carries a binary**, and installing refuses to wipe a directory the harness did not install. A skill that stops shipping `cli/` loses its stale binary and PATH link.
12. **Company-agnostic content rule survives** (§0.6) — it is what makes the repo publishable.

---

## 3. Deliverables

Every exit criterion below is producible **in this environment at M0** (verified: `pwsh`, `actionlint`, `shellcheck` are **not installed**; `go`, `python3`+PyYAML, `sh`, `git`, `gh` are).

### DL-1 — Repo skeleton + licence + README
`harness/{skills/,agents/,installer/,docs/,README.md,LICENSE,.gitignore,go.work}`. README: what it is, both install kinds, selection, local `--from`, the §0.6 contribution rules, layout, and the honest day-one path (DL-6).

**Exit:** `test -d harness/{skills,agents,installer} && test -f harness/README.md` → 0; **and** the repo's own leakage check `grep -rniE 'lybel|11C47E|164232' harness/ --exclude-dir=.git` returns nothing outside an explicit maintainer credit.

### DL-2 — The three artifacts, frontmatter FIXED
`agents/unbiased-reviewer.md` (as-is), `skills/dev-loop/SKILL.md` + `skills/implementation-plan/SKILL.md` (description quoted). Body unchanged.

**Exit:** a script `yaml.safe_load`s all three → 3/3 OK; asserts `name` lowercase-hyphen ≤64 **and matching the directory/file name**, `description` ≤1024; and `diff` of the post-frontmatter body against the installed originals is **empty**.

### DL-3 — Installer: catalog
`Artifact{Name, Kind(skill|agent), TagPrefix, Summary, VersionEnv, Requires []string}` replacing `Skill`. **No pipeline field** — the pipeline is discovered (D3). `TagPrefix` is consulted only for a skill that ships a `cli/`.

**Exit:** `go test ./installer/...` passes, including tests asserting (a) name uniqueness across kinds (rule 10), (b) every catalogued artifact exists in the tree, (c) a skill shipping `cli/` declares a `TagPrefix`, and (d) an agent never carries one.

### DL-4 — Installer: the single install pipeline
Fetch `https://codeload.github.com/<repo>/tar.gz/<ref>` **once per run** (cached in the run temp dir → 5 artifacts = 1 download), gunzip, untar with a traversal guard, **strip the `<repo>-<ref>` root prefix**. `--from <path>` bypasses the download. Then per artifact: agent → `~/.claude/agents/<name>.md`; skill → tree payload, or release payload if it ships `cli/`, followed by the binary stage.

> **Measured gotcha (do not guess):** codeload's root entry is `<repo>-<ref>` with GitHub's leading-`v` stripping quirk — `…/skills/tar.gz/main` → `skills-main/`; `…/skills/tar.gz/jira-v0.4.1` → `skills-jira-v0.4.1/`.

**Exit — `--from` alone proves nothing about the download path, so:**
1. **Tarball path:** a real `.tar.gz` fixture with a `harness-<ref>/` root served over `httptest`, swapping the package-level `httpClient` (`install.go:21`) — no production widening. Asserts prefix strip and placement.
2. **Traversal:** fixtures containing `../evil` for **both** `untar` and `unzip` → non-zero exit, nothing written outside the destination.
3. **Local path:** sandboxed `HOME`, `install --from <repo> dev-loop implementation-plan unbiased-reviewer` → 0, all three placed, `diff -r` byte-identical.
4. **Discovery, both directions:** a skill without `cli/` gets no `bin/` and no PATH link; a skill with `cli/` gets both, its binary verified by running it. A release missing its binary fails **and leaves the filesystem untouched**. `cli/` is never installed. A skill that loses `cli/` loses its stale binary and PATH link.
5. **Atomic replace:** reinstalling over an installed binary that cannot be opened for writing succeeds (the shape of the running-executable case).

### DL-5 — Installer: CLI surface
```
harness list [--skills|--agents]      grouped, kind-labelled
harness install <name>...             any mix of kinds; dependencies auto-resolved
harness install --all | --all-skills | --all-agents   (combinable, as a union)
harness install --ref REF <name>...   git ref of the tree (default main)
harness install --from PATH <name>... local clone
harness install --version TAG <name>  pin a release tag; skills with a binary, one at a time
harness validate | --version | --help
```
Unknown names rejected atomically. An empty flag value (`--from=`) is a usage error, not a silent fallback. A bare `curl | sh` prints the catalog and exits 2 (M7).

**Exit (all producible at M0 — no criterion may depend on the unpublished repo):**
- `harness list` prints every artifact with its kind.
- `install --all --from <clone>` in a sandboxed `HOME` → exit 0.
- **`install dev-loop` alone also lands `~/.claude/agents/unbiased-reviewer.md`**, and the chain is transitive.
- Error matrix, each exit 2 with a **specific** message: unknown name · no name · `--version` on an artifact with no binary · `--version` with two pinnable artifacts · every flag with a missing or empty value. On rejection the filesystem is asserted untouched.
- One failed artifact yields exit 1, a `N of M` summary, and does not abort the others.

### DL-6 — Bootstrap `install.sh` / `install.ps1` + installer release workflow
Same thin-bootstrap shape retargeted at `diegoclair/harness`, tag prefix `harness-v`, plus `release-installer.yml` adapted. Bare invocation prints the catalog and exits 2 (M7).

**Rename surface (M6) — all of it, consistently:** `installer/Makefile:13` `BIN := skills` → asset `dist/harness-<os>-<arch>`; `install.sh:19,49` + `install.ps1:19,36` hardcoded asset name, `TAG_PREFIX`, and env vars `SKILLS_REPO`/`SKILLS_INSTALLER_VERSION`; `release-installer.yml:6` trigger; `install.go:99`'s `SKILL_VERSION`.

**Chicken-and-egg, stated honestly:** the one-liner only works after `harness-v0.1.0` exists. Until then the README's documented path is `git clone && go run ./installer install <names>`. The README must not imply a working curl.

**Exit:** `sh -n install.sh` → 0; `python3 -c "import yaml,sys; yaml.safe_load(open(...))"` parses both workflows; **a Go test greps the asset name and tag prefix out of `install.sh`, `install.ps1`, `installer/Makefile` and `release-installer.yml` and asserts they agree** (a mismatch is otherwise a 404 discovered only after publishing, invisible to `sh -n`). `pwsh`/`actionlint` are unavailable → declared a manual pre-publish check in the migration doc, not a fake green.

### DL-7 — Own validator (no Python dependency)
`harness validate [<name>...]` — strict YAML frontmatter parse; `name` (lowercase-hyphen, ≤64, matches directory/file name); `description` (present, ≤1024, **and free of `<`/`>`** — `quick_validate.py:80-81`, a real rejection cause for anyone later running the official packager); for skills, `SKILL.md` exists.

**It validates the artifacts present in the tree** (walking `skills/` and `agents/`), **not the catalog** — the 3 legacy skills do not arrive until M2.

**Exit:** run against the repo → exit 0, reporting **3 artifacts at M0** (and 6 after M2). Deliberately reintroduce the unquoted-colon defect in a temp copy → non-zero exit naming the file. This is the regression guard for the §0.3 defect class.

### DL-8 — Migration plan document
`harness/docs/migration-from-skills.md` — §4 below, executable. Not executed this session.

**Exit:** file exists; every phase has concrete commands and a rollback note; the `AUTH_PKG`, OAuth-secret, bridge-release, and tag/release steps are all present.

---

## 4. Migration plan (WRITTEN now, EXECUTED later — after Diego's go-ahead)

### M0 — this session
Harness built locally, complete, tested, **unpushed**. Old repo untouched.

### M1 — publish the harness (needs OD-1, OD-2)
1. `gh repo create diegoclair/harness` (visibility per OD-1). 2. Push `main`. 3. Tag `harness-v0.1.0` → installer release CI → one-liner live. 4. Smoke: sandboxed `HOME`, `curl … | sh -s -- install dev-loop unbiased-reviewer` → 0.
Manual pre-publish checks (no local tooling): `actionlint` on the workflows, `pwsh` parse of `install.ps1`.
At the end of M1 the three new pieces are publicly installable; **the Atlassian skills have not moved and are still served by the old repo** — nothing is broken at any point.

### M2 — move the legacy skills (the delicate phase)
1. **History** — settle OD-3 first: `git subtree`/`filter-repo` to preserve, or flat copy + pointer.
2. **Files** — `<skill>/` → `skills/<skill>/`; `pkg/` stays at root; `installer/` is already the harness's.
3. **Module paths — 6 `go.mod` + 77 Go refs + the 2 `Makefile` `AUTH_PKG` lines** (§0.5), plus `go.work` `use` paths.
   > **⚠ H2 — silent production failure if the Makefiles are missed.** The Go linker **ignores an unknown `-X` symbol without error**. Proven: `-X github.com/diegoclair/skills/pkg/atlassian/auth.DefaultClientSecret=X` → build exit 0, secret **empty**; the correct path → secret set. So `go build ./...` and `go test ./...` stay **green**, CI stays green, and every released binary ships an empty `DefaultClientSecret` (`auth/login.go:38` → `authorizer.go:190` omits it) → **`login` broken for all users.**
   *Verify:* `go build ./... && go test ./...` green **AND** a built binary asserts a non-empty `DefaultClientSecret` when the secret is passed. The build alone is not proof.
4. **Self-update targets** — the 3 `repoOwnerRepo` + `installShURL`/`installPS1URL` constants (§0.5).
5. **CI** — `release-<skill>.yml` path prefixes → `skills/<skill>/cli`; tag prefixes unchanged; exactly one `make_latest: true`.
6. **Secret** — recreate `ATLASSIAN_OAUTH_CLIENT_SECRET` in the harness repo **before** the first skill release, or binaries ship without the bundled OAuth app.
7. **Re-release** — cut a fresh tag per skill in the harness. **Tags/releases do not migrate**; without this `FindLatestByPrefix` finds nothing and `install confluence-docs` fails.
   *Verify:* sandboxed `HOME`, `harness install --all` → all 6 land, exit 0.
8. **Non-Go doc/script refs** — the 16 files of §0.5.

### M2.5 — the BRIDGE (⚠ H3 — must happen while the old repo is still writable)
`cmd_update.go` hardcodes the old repo **and** re-fetches `install.sh` from its `main` at runtime (`curl -fsSL $installShURL | bash`). If the old repo is archived first, every already-installed CLI is **permanently frozen**, reports "up to date" (`cmd_update.go:57-59`), and cannot be fixed — archived repos are read-only.
1. On the old repo's `main`, repoint `install.{sh,ps1}` (root + the 3 `*/install/`) at `diegoclair/harness`.
2. Cut **one final bridge release per skill** in the old repo, whose binary carries the new constants.
3. Only then proceed to M3.

### M3 — retire the old repo gracefully
README banner pointing at `diegoclair/harness` with the new one-liner · GitHub **Archive** (read-only; releases stay downloadable, old installs keep working) · **never** delete tags/releases, never force-push, never make it private.

---

## 5. Open decisions — the executor STOPS here

| ID | Decision | Why it's Diego's | Blocks |
|---|---|---|---|
| **OD-1** | Harness repo **public or private**? | Positioning. **The installer has zero auth** — `grep -rn "Authorization\|GITHUB_TOKEN" installer/ pkg/release/` → no matches. Private ⇒ **all three** network paths 404 (codeload tarball, releases API `release.go:73`, asset download `install.go:57`) ⇒ an extra, currently unscoped deliverable: token header on all three + `GH_TOKEN` plumbing through the bootstrap. D3's "ships day one with no CI" holds **only if public**. | **M1** |
| **OD-2** | **Licence + copyright holder.** Old repo is MIT © **Lybel**; harness is Diego's personal umbrella. Moving Lybel-copyrighted code (`pkg/atlassian`, 3 CLIs) into a personal repo is an ownership call. | Legal/ownership. | **M2** |
| **OD-3** | Preserve git history (subtree/filter-repo) or flat copy + pointer? | Cost vs. provenance. | M2.1 |
| **OD-4** | Keep `version:` in skill frontmatter? Claude Code accepts it; the official validator rejects it (§0.3). | Convention; cheap either way; sets DL-7's rule set. | — |
| **OD-5** | Harness README keeps **"Lybel Skills"** branding for migrated skills, or rebrands to personal? | Positioning; interacts with OD-2. | M2.8 |
| **OD-6** | Is `goal-run` (the pending third skill) in scope now? | Scope. Default: later — it isn't written yet. | — |

**Only OD-1 and OD-2 block anything** (M1 and M2 respectively). All of M0 — this session's build — proceeds without any of them.

---

## 6. Reference paths

- Old repo clone: `<scratch>/recon/skills-old/` — README (rules), `installer/*.go`, `.github/workflows/release-*.yml`, `pkg/release/release.go`, the 2 `cli/Makefile`
- Artifacts: `~/.claude/agents/unbiased-reviewer.md`, `~/.claude/skills/{dev-loop,implementation-plan}/SKILL.md`
- Official packager + validator: `~/.claude/skills/skill-creator/scripts/{package_skill,quick_validate}.py`
- History/rationale: `~/.claude/projects/-home-diegoclair-www-isquads/memory/feedback_orchestrator_reviewer_flow.md` (blocks "CRISTALIZADO", "PUBLISH no GitHub — HELD")
- Global conventions: `~/.claude/CLAUDE.md`
- Session sandbox: `<scratch>/recon/sandbox-home/`, built installer `<scratch>/recon/skills-bin`
