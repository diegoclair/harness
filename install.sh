#!/bin/sh
# install.sh — bootstrap for the `skills` installer.
#
# Downloads the installer binary for this platform and runs it. Everything
# else — resolving each skill's release, extracting, PATH wiring — lives in
# the Go binary, which is tested and identical on every OS.
#
#   curl -fsSL https://raw.githubusercontent.com/diegoclair/skills/main/install.sh | sh
#   curl -fsSL .../install.sh | sh -s -- list
#   curl -fsSL .../install.sh | sh -s -- install confluence-docs jira-tickets
#
# Optional environment:
#   SKILLS_INSTALLER_VERSION   Pin a tag (default: latest installer-v* release)
#   SKILLS_REPO                Install from a fork

set -e

REPO="${SKILLS_REPO:-diegoclair/skills}"
TAG_PREFIX="installer-v"

die() { echo "error: $*" >&2; exit 1; }

case "$(uname -s)" in
  Linux)  os="linux" ;;
  Darwin) os="darwin" ;;
  *) die "unsupported OS: $(uname -s). On Windows use install.ps1" ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) die "unsupported architecture: $(uname -m)" ;;
esac

command -v curl >/dev/null 2>&1 || die "curl is required"

VERSION="$SKILLS_INSTALLER_VERSION"
if [ -z "$VERSION" ]; then
  # This repo tags every component separately, so /releases/latest (a single
  # pointer per repo) cannot be followed — filter the list by prefix instead.
  VERSION="$(curl -fsSL "https://api.github.com/repos/$REPO/releases?per_page=30" \
    | grep -oE "\"tag_name\":[[:space:]]*\"${TAG_PREFIX}[^\"]+\"" \
    | head -1 | sed "s/.*\"\(${TAG_PREFIX}[^\"]*\)\"/\1/")"
  [ -n "$VERSION" ] || die "could not resolve the latest $TAG_PREFIX release; set SKILLS_INSTALLER_VERSION"
fi

BIN="$(mktemp)"
trap 'rm -f "$BIN"' EXIT
URL="https://github.com/$REPO/releases/download/$VERSION/skills-${os}-${arch}"

curl -fsSL --retry 3 --retry-delay 2 -o "$BIN" "$URL" || die "download failed: $URL"
chmod +x "$BIN"

# No arguments: install everything, since a bare pipe has no tty to ask on.
if [ "$#" -eq 0 ]; then
  exec "$BIN" install --all
fi
exec "$BIN" "$@"
