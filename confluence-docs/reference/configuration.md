# Configuration — credentials, auth modes, spaces, cache, install check

> **When to read:** at setup / install time, when choosing between `login` (OAuth) and `setup` (API token), when switching between Confluence spaces, when troubleshooting "credentials invalid" or stale-cache issues, or when the user asks how the skill resolves the active space.

## Contents

- [Credentials and config files](#credentials-and-config-files)
- [Auth modes and resolution order](#auth-modes-and-resolution-order)
- [`login` — OAuth browser login](#login--oauth-browser-login)
- [Credentials-file keys](#credentials-file-keys)
- [Space management](#space-management)
- [Home cache lifecycle](#home-cache-lifecycle)
- [CLI installation check](#cli-installation-check)

---

## Credentials and config files

The CLI uses two separate config files:

- **Credentials** (`~/.config/atlassian/credentials`, perms `0600`): shared by all Atlassian skills (`confluence-docs`, `jira-tickets`). Holds an API token (`email` + `token`), an OAuth grant (`oauth_*` keys), or both. Never read raw — use `setup --check` to validate.
- **Config** (`~/.config/confluence-docs/config`, perms `0644`): `cloud`, `active_space_id`, `active_space_key`, `active_space_name`, `active_home_page_id`. Per-skill and non-secret.

Legacy per-skill credentials at `~/.config/confluence-docs/credentials` are still read as a last-resort fallback (with a one-line migration warning). Re-running `setup` once writes to the shared path.

All non-secret values are set automatically during `confluence-docs setup` (which auto-detects accessible spaces and asks the user to pick one) or `confluence-docs login`. **Agents must not read or write these files directly.** Use the CLI commands:

```bash
confluence-docs setup --check              # validate everything is configured
confluence-docs space current              # show active space
confluence-docs space list                 # list all accessible spaces
confluence-docs space use <key>            # switch active space
confluence-docs setup --set cloud acmecorp # change a single config key
```

The active space provides defaults for all commands that need `--space-id` or space key (CQL search, `index`, `home`, `page create`, `check`). Commands that previously required explicit `--space-id <ID>` flags now use the configured space automatically.

## Auth modes and resolution order

Two modes are supported:

| Mode | Command | Lifetime | When |
|---|---|---|---|
| **OAuth 2.0 (3LO)** — recommended | `confluence-docs login` | Auto-refreshes forever (rotating refresh tokens) | Default for interactive machines |
| **API token (Basic auth)** — fallback | `confluence-docs setup` | Token expires after at most 1 year (Atlassian mandate since Dec/2024) | Headless/CI environments, or users who prefer not to register an app |

Every command resolves auth in this order — first match wins:

1. CLI flags (`--email` / `--token`) → Basic auth
2. Env vars `ATLASSIAN_EMAIL` + `ATLASSIAN_API_TOKEN` → Basic auth
3. Stored OAuth grant (unless `auth_mode=apitoken` forces the token)
4. Stored `email`/`token`, including legacy per-skill paths → Basic auth

OAuth calls route through `https://api.atlassian.com/ex/confluence/<cloudId>`; Basic-auth calls go to `https://<cloud>.atlassian.net`.

## `login` — OAuth browser login

Run `confluence-docs login`. A browser opens, you authorize, and you're done — there is nothing to register and no token to paste. Access tokens refresh automatically from then on, so this is a one-time step.

The command opens the browser (WSL-aware), waits on a localhost callback (port 8517) and stores the grant in the shared credentials file. Add `--site` when your account has access to several Atlassian sites, or `--no-browser` on a headless machine to open the printed URL yourself.

**What the login authorizes.** The CLI ships with a shared OAuth app, so there is nothing to register. The grant is per user and revocable at any time at https://id.atlassian.com/manage-profile/apps; the app identifies the software, never the user. PKCE (S256) binds the authorization code to the process that started the login.

To authorize against your own Atlassian app instead — an org that requires its own OAuth integration — register it at https://developer.atlassian.com/console/myapps/ (**Create** → **OAuth 2.0 integration**, add the scopes below under **Permissions**, callback URL from `login --print-redirect-uri`) and pass `--client-id`/`--client-secret`.

If `login` reports that no app secret is bundled, the binary was built from source rather than installed from a release — reinstall via the install script, or use your own app as above.

**Flags:**

| Flag | Meaning |
|---|---|
| `--client-id ID` | Use your own app instead of the bundled one |
| `--client-secret SECRET` | Secret for that app |
| `--scopes "s1 s2"` | Override the default scope list |
| `--site NAME\|URL` | Pick the target site when the account has access to several |
| `--no-browser` | Don't open a browser; the user opens the printed URL manually |
| `--print-redirect-uri` | Print the callback URL to register in the app, then exit |

**Default scopes** (already granted in the bundled app; `offline_access` is what produces the refresh token — without it the session dies in 1 hour):

```
offline_access
read:page:confluence write:page:confluence
read:space:confluence
read:label:confluence write:label:confluence
read:content:confluence write:content:confluence
read:content-details:confluence read:content.metadata:confluence
read:user:confluence
read:jira-work write:jira-work read:jira-user
```

## Credentials-file keys

The credentials file is a flat `key=value` store managed exclusively by the CLI: `setup` writes `email`/`token`; `login` and the auto-refresh path write the `oauth_*` keys. Both modes coexist in one file. **Never hand-edit the `oauth_*` keys** — refresh tokens rotate on every refresh, and a stale value kills the grant.

| Key | Written by | Meaning |
|---|---|---|
| `auth_mode` | `login` / `setup` | Preferred mode: `oauth` or `apitoken` |
| `email`, `token` | `setup` | Basic-auth credentials |
| `oauth_client_id`, `oauth_client_secret` | `login` | Your app's credentials |
| `oauth_access_token` | `login`, auto-refresh | Short-lived bearer token |
| `oauth_refresh_token` | `login`, auto-refresh | Rotates on every refresh — always the newest wins |
| `oauth_expiry` | `login`, auto-refresh | Access-token expiry (unix seconds) |
| `oauth_cloud_id` | `login` | Site cloudId used by the API gateway |
| `oauth_site` | `login` | Cloud subdomain (for web links) |
| `oauth_scopes` | `login` | Scopes granted at login |

A sibling lock file `credentials.lock` serializes refresh-token rotation across concurrent CLI processes (stale locks older than 30s are reaped automatically).

## Space management

```bash
# List all spaces (cached 1h; shows active space with ✓)
confluence-docs space list

# Switch active space
confluence-docs space use eng

# Force cache refresh
confluence-docs space list --refresh

# JSON output
confluence-docs space list --json
confluence-docs space current --json
```

After `space use <key>`, all subsequent commands use the new space. The switch is persistent (written to `~/.config/confluence-docs/config`).

## Home cache lifecycle

The local cache at `~/.cache/confluence-docs/home.json` is **shared across all Claude sessions on the same machine** — so if one session refreshes (manually or via auto-refresh-on-write), every other session reading it next sees the updated state automatically. No per-session bookkeeping.

Three rules govern when the cache is updated:

| Trigger | Behavior |
|---|---|
| Read with stale cache (>1h old) or missing | **Auto-refresh** before serving. Caller doesn't have to think about it. |
| Write to the Home via CLI (`page apply` / `index *` on the Home pageId) | **Auto-refresh after PUT** succeeds. Your session sees the new state immediately. |
| Explicit `home --refresh` | **Always fetches**, ignores TTL. Use only when you know another machine just edited the Home and you don't want to wait for the TTL. |

What this means in practice: in a typical session, you never call `home --refresh` explicitly. You just query/show/digest, and writes refresh themselves.

**WRITE SAFETY (critical):** the cache is **read-only for navigation**. It is **NEVER** the source for an update. Any mutation of the Home (or any page) goes through `page apply`, which always GETs fresh ADF before PUT — ensuring you never overwrite changes someone made on another machine.

### Quick reference

- **Reads** (`home --query/--show/--digest`): auto-refresh when stale (>1h) or missing.
- **Writes** to the Home (`page apply` / `index *` on the Home pageId): auto-refresh the cache after the PUT.
- **Explicit `home --refresh`**: always fetches, regardless of cache age.
- **Override TTL**: `--max-age 30m` (more aggressive) or `--max-age 6h` (more relaxed) on any read command.

This invariant — **read-only cache, fresh fetch before every write** — is the core safety property. Don't bypass it (e.g. don't try to PUT the cached ADF directly).

## CLI installation check

Before running any `confluence-docs` command, verify the binary exists and credentials are valid. The bootstrap flow is:

```bash
confluence-docs --version          # binary present?
confluence-docs setup --check      # exit 0 = creds valid
```

`setup --check` validates whichever auth mode is active — API token or OAuth — and reports it on success (`credentials valid (<name>, oauth, space: <key>)`). Exit codes:

- `0` — credentials valid AND active space configured → proceed
- `1` — no credentials OR no active space configured → run `confluence-docs login` (or `setup`) interactively (or guide the user through Step 5 of `confluence-docs/cli/README.md`)
- `2` — credentials invalid → OAuth session dead: run `confluence-docs login` again. API token revoked or mistyped: ask the user to regenerate the token at `https://id.atlassian.com/manage-profile/security/api-tokens` and re-run setup
- `3` — network error → retry once; if it persists, surface the error to the user and fall back to MCP

If the binary is absent entirely, fall back to MCP for the current request and tell the user how to install: `confluence-docs/cli/README.md` has the one-shot install URL.
