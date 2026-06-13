#!/usr/bin/env bash
set -euo pipefail

VARS=(
  APPLE_SIGNING_IDENTITY
  APPLE_CERTIFICATE
  APPLE_CERTIFICATE_PASSWORD
  APPLE_ID
  APPLE_PASSWORD
  APPLE_TEAM_ID
)

command -v op >/dev/null || { echo "op CLI required" >&2; exit 1; }
command -v gh >/dev/null || { echo "gh CLI required" >&2; exit 1; }
[ -f .env.signing ] || { echo ".env.signing not found in $(pwd)" >&2; exit 1; }

missing=()
for v in "${VARS[@]}"; do
  if ! grep -qE "^${v}=" .env.signing; then
    missing+=("$v")
  fi
done
if (( ${#missing[@]} > 0 )); then
  echo "Missing from .env.signing:" >&2
  printf '  - %s\n' "${missing[@]}" >&2
  exit 1
fi

op run --env-file=.env.signing --no-masking -- bash -c '
  set -euo pipefail
  for v in '"${VARS[*]}"'; do
    val="${!v-}"
    if [ -z "$val" ]; then
      echo "Skipping ${v} (empty after op resolution)" >&2
      continue
    fi
    printf "%s" "$val" | gh secret set "$v" --repo phin-tech/whisk --body -
    echo "Set ${v}"
  done
'
