#!/bin/sh
# install.sh — bootstrap for the `harness` installer.
#
# Downloads the installer binary for this platform and runs it. Everything
# else — fetching artifacts, placing skills and agents, PATH wiring — lives in
# the Go binary, which is tested and identical on every OS.
#
#   curl -fsSL https://raw.githubusercontent.com/diegoclair/harness/main/install.sh | sh -s -- list
#   curl -fsSL .../install.sh | sh -s -- install dev-loop unbiased-reviewer
#
# Optional environment:
#   HARNESS_INSTALLER_VERSION   Pin a tag (default: latest harness-v* release)
#   HARNESS_REPO                Install from a fork

set -e

REPO="${HARNESS_REPO:-diegoclair/harness}"
TAG_PREFIX="harness-v"
ASSET="harness"

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

VERSION="$HARNESS_INSTALLER_VERSION"
if [ -z "$VERSION" ]; then
  # This repo tags every component separately, so /releases/latest (a single
  # pointer per repo) cannot be followed — filter the list by prefix instead.
  # Line-oriented grep over the JSON cannot tell which release a "draft" or
  # "prerelease" flag belongs to, so this takes the newest matching tag as-is.
  # Pin HARNESS_INSTALLER_VERSION to bypass the guess.
  VERSION="$(curl -fsSL "https://api.github.com/repos/$REPO/releases?per_page=30" \
    | grep -oE "\"tag_name\":[[:space:]]*\"${TAG_PREFIX}[^\"]+\"" \
    | head -1 | sed "s/.*\"\(${TAG_PREFIX}[^\"]*\)\"/\1/")"
  [ -n "$VERSION" ] || die "could not resolve the latest $TAG_PREFIX release; set HARNESS_INSTALLER_VERSION"
fi

BIN="$(mktemp)"
trap 'rm -f "$BIN"' EXIT
URL="https://github.com/$REPO/releases/download/$VERSION/${ASSET}-${os}-${arch}"

curl -fsSL --retry 3 --retry-delay 2 -o "$BIN" "$URL" || die "download failed: $URL"
chmod +x "$BIN"

# No arguments: show what is available rather than writing files into
# ~/.claude for someone who piped a script without asking for anything.
if [ "$#" -eq 0 ]; then
  "$BIN" list
  echo
  echo "Pick what you want, e.g.:  curl -fsSL .../install.sh | sh -s -- install dev-loop"
  exit 2
fi
exec "$BIN" "$@"
