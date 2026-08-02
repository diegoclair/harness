#!/bin/sh
# install.sh — installs the confluence-docs skill.
#
# Kept at this URL for the published one-liner and for `confluence-docs update`,
# which shells out to it. The work is done by the `skills` installer binary;
# this only forwards the skill name to it.
#
#   curl -fsSL https://raw.githubusercontent.com/diegoclair/skills/main/confluence-docs/install/install.sh | sh
#
# To install several skills at once, use the root install.sh instead.

set -e

REPO="${SKILL_REPO:-diegoclair/skills}"
ROOT_URL="https://raw.githubusercontent.com/$REPO/main/install.sh"

# Version pins used the SKILL_VERSION / <SKILL>_VERSION env vars; the
# installer still honours both, so nothing extra is needed here.
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$ROOT_URL" | sh -s -- install confluence-docs
elif command -v wget >/dev/null 2>&1; then
  wget -qO- "$ROOT_URL" | sh -s -- install confluence-docs
else
  echo "error: neither curl nor wget found; install one and retry" >&2
  exit 1
fi
