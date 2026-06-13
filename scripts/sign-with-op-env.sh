#!/usr/bin/env bash
set -euo pipefail

: "${APPLE_CERTIFICATE:?base64-encoded .p12 required}"
: "${APPLE_CERTIFICATE_PASSWORD:?.p12 password required}"

RUNNER_TEMP="${RUNNER_TEMP:-$(mktemp -d)}"
export RUNNER_TEMP

scripts/ci-setup-macos-keychain.sh

if [ -z "${APPLE_SIGNING_IDENTITY:-}" ]; then
  cert_path="${RUNNER_TEMP}/identity.p12"
  printf "%s" "${APPLE_CERTIFICATE}" | base64 --decode > "${cert_path}"
  APPLE_SIGNING_IDENTITY="$(
    openssl pkcs12 -in "${cert_path}" -nokeys -clcerts -passin env:APPLE_CERTIFICATE_PASSWORD 2>/dev/null |
      openssl x509 -noout -subject -nameopt RFC2253 |
      sed -n "s/^subject=.*CN=\\([^,]*\\).*/\\1/p"
  )"
  export APPLE_SIGNING_IDENTITY
fi

task sign
