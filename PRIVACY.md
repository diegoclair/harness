# Privacy Policy — Lybel Skills

_Last updated: 2026-08-01_

This policy covers the open-source command-line skills in this repository
(`confluence-docs`, `jira-tickets`, `social-carousel`) and the Atlassian OAuth
2.0 integration used by their `login` command.

## No data reaches us

These skills are local command-line programs. They run entirely on your
machine and talk directly to Atlassian's APIs. There is no server, no backend
and no account on our side.

We — the maintainers — never receive, collect, store or process any of your
data. There is no telemetry, no analytics and no crash reporting.

## What is stored on your machine

| What | Where | Notes |
|---|---|---|
| OAuth tokens / API credentials | `~/.config/atlassian/credentials` | File permissions `0600` (owner-only) |
| Confluence page cache | `~/.config/confluence-docs/` | Navigation cache, refreshed hourly |

Both live only on your computer, under your control. Delete the files to
remove everything; revoke the grant at
https://id.atlassian.com/manage-profile/apps to invalidate the tokens.

## What the OAuth integration accesses

The `login` command requests the Confluence and Jira scopes needed for the
commands you run — reading and writing pages, spaces, labels and issues on the
Atlassian site you choose during login. Access is granted per user and can be
revoked by you at any time.

Data fetched from Atlassian is used to answer the command you ran and is
printed to your terminal. Nothing is transmitted anywhere else.

## Third parties

None. The only network destinations are Atlassian's own endpoints
(`auth.atlassian.com`, `api.atlassian.com`) and, for self-update checks,
GitHub's public release API.

## Contact

Questions or reports: https://github.com/diegoclair/skills/issues
