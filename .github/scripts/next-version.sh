#!/bin/sh
# Print the next release tag (e.g. v2.1.0) from the commits since the last
# v* tag, using Conventional Commit messages:
#   "feat!: ..." / "BREAKING CHANGE"  -> major
#   "feat: ..."                         -> minor
#   anything else                       -> patch
# Prints nothing when there are no new commits.
# Usage: next-version.sh [auto|patch|minor|major]   (default auto)
set -eu

bump="${1:-auto}"
initial="${INITIAL_VERSION:-v2.0.0}"

last=$(git describe --tags --abbrev=0 --match 'v[0-9]*.[0-9]*.[0-9]*' 2>/dev/null || true)
if [ -z "$last" ]; then
  echo "$initial"
  exit 0
fi

if [ -z "$(git rev-list "$last..HEAD")" ]; then
  exit 0 # nothing new since the last release
fi

if [ "$bump" = auto ]; then
  log=$(git log --format='%s%n%b' "$last..HEAD")
  if printf '%s\n' "$log" | grep -Eq '^[a-z]+(\([^)]*\))?!:|^BREAKING[ -]CHANGE'; then
    bump=major
  elif printf '%s\n' "$log" | grep -Eq '^feat(\([^)]*\))?:'; then
    bump=minor
  else
    bump=patch
  fi
fi

version=${last#v}
version=${version%%-*} # drop any pre-release suffix
major=${version%%.*}
rest=${version#*.}
minor=${rest%%.*}
patch=${rest#*.}

case "$bump" in
  major) major=$((major + 1)); minor=0; patch=0 ;;
  minor) minor=$((minor + 1)); patch=0 ;;
  patch) patch=$((patch + 1)) ;;
  *) echo "unknown bump: $bump" >&2; exit 1 ;;
esac
echo "v$major.$minor.$patch"
