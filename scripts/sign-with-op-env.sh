#!/usr/bin/env bash
set -euo pipefail

: "${APPLE_CERTIFICATE:?base64-encoded .p12 required}"
: "${APPLE_CERTIFICATE_PASSWORD:?.p12 password required}"

ORIGINAL_KEYCHAINS="$(security list-keychains -d user | sed -E 's/^[[:space:]]*"?//; s/"?$//')"
CREATED_RUNNER_TEMP=""
if [ -z "${RUNNER_TEMP:-}" ]; then
  RUNNER_TEMP="$(mktemp -d)"
  CREATED_RUNNER_TEMP="true"
fi
RUNNER_TEMP="$(cd "${RUNNER_TEMP}" && pwd -P)"
export RUNNER_TEMP
SIGN_KEYCHAIN="${RUNNER_TEMP}/whisk-build.keychain"
export SIGN_KEYCHAIN

restore_keychains() {
  local keychain
  local keychains=()

  while IFS= read -r keychain; do
    [ -n "${keychain}" ] || continue
    [ "${keychain}" != "${SIGN_KEYCHAIN}" ] || continue
    case "${keychain}" in
      /private/var/folders/*/T/tmp.*/whisk-build.keychain | /var/folders/*/T/tmp.*/whisk-build.keychain | \
      /private/var/folders/*/T/tmp.*/sign.keychain | /var/folders/*/T/tmp.*/sign.keychain)
        continue
        ;;
    esac
    [ -e "${keychain}" ] || continue
    keychains+=("${keychain}")
  done <<< "${ORIGINAL_KEYCHAINS}"

  if [ "${#keychains[@]}" -eq 0 ]; then
    keychains+=("${HOME}/Library/Keychains/login.keychain-db")
  fi

  security list-keychains -d user -s "${keychains[@]}"
}

cleanup() {
  restore_keychains
  security delete-keychain "${SIGN_KEYCHAIN}" >/dev/null 2>&1 || true
  if [ "${CREATED_RUNNER_TEMP}" = "true" ]; then
    rm -rf "${RUNNER_TEMP}"
  fi
}
trap cleanup EXIT

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
