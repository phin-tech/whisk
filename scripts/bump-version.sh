#!/usr/bin/env bash
set -euo pipefail

BUMP="${BUMP:-}"
PRE="${PRE:-}"

case "$BUMP" in
  patch | minor | major) ;;
  *) echo "BUMP must be one of: patch, minor, major" >&2; exit 2 ;;
esac

last_stable="$(git tag -l 'v[0-9]*.[0-9]*.[0-9]*' | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | sort -V | tail -1 || true)"
if [[ -z "$last_stable" ]]; then
  current="$(scripts/current-version.sh)"
else
  current="${last_stable#v}"
fi

IFS='.' read -r major minor patch <<< "$current"

case "$BUMP" in
  major) major=$((major + 1)); minor=0; patch=0 ;;
  minor) minor=$((minor + 1)); patch=0 ;;
  patch) patch=$((patch + 1)) ;;
esac

base="${major}.${minor}.${patch}"
if [[ -n "$PRE" ]]; then
  last_pre="$(git tag -l "v${base}-${PRE}.*" | sed "s/^v${base}-${PRE}\.//" | sort -n | tail -1 || true)"
  if [[ -n "$last_pre" ]]; then
    pre_num=$((last_pre + 1))
  else
    pre_num=1
  fi
  next="${base}-${PRE}.${pre_num}"
else
  next="$base"
fi

python3 - "$next" <<'PY'
import re
import sys
from pathlib import Path

version = sys.argv[1]
path = Path("build/config.yml")
text = path.read_text()
text, count = re.subn(r'(^\s+version:\s*")[^"]+(".*$)', rf'\g<1>{version}\2', text, count=1, flags=re.MULTILINE)
if count != 1:
    raise SystemExit("failed to update build/config.yml info.version")
path.write_text(text)
PY

echo "$next"
