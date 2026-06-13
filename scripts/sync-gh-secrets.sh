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
  if [ -z "${APPLE_SIGNING_IDENTITY-}" ] && [ -n "${APPLE_CERTIFICATE-}" ] && [ -n "${APPLE_CERTIFICATE_PASSWORD-}" ]; then
    cert_path="$(mktemp)"
    cleanup() { rm -f "${cert_path}"; }
    trap cleanup EXIT
    printf "%s" "${APPLE_CERTIFICATE}" | base64 --decode > "${cert_path}"
    APPLE_SIGNING_IDENTITY="$(
      openssl pkcs12 -in "${cert_path}" -nokeys -clcerts -passin env:APPLE_CERTIFICATE_PASSWORD 2>/dev/null |
        openssl x509 -noout -subject -nameopt RFC2253 |
        sed -n "s/^subject=.*CN=\\([^,]*\\).*/\\1/p"
    )"
  fi
  for v in '"${VARS[*]}"'; do
    val="${!v-}"
    if [ -z "$val" ]; then
      echo "Skipping ${v} (empty after op resolution)" >&2
      continue
    fi
    printf "%s" "$val" | gh secret set "$v" --repo phin-tech/whisk
    echo "Set ${v}"
  done
'
