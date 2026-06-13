#!/usr/bin/env bash
set -euo pipefail

: "${SIGN_IDENTITY:?Developer ID signing identity required}"
: "${KEYCHAIN_PROFILE:?notary keychain profile name required}"

APP_PATH="${1:-bin/Whisk.app}"

if [ ! -d "${APP_PATH}" ]; then
  echo "App bundle not found: ${APP_PATH}" >&2
  exit 1
fi
APP_PATH="$(cd "$(dirname "${APP_PATH}")" && pwd)/$(basename "${APP_PATH}")"

assert_embedded_certificates() {
  local cert_dir
  cert_dir="$(mktemp -d)"
  (
    cd "${cert_dir}"
    codesign -d --extract-certificates "${APP_PATH}" >/dev/null
    if ! ls codesign* >/dev/null 2>&1; then
      echo "Signed app does not contain embedded signing certificates" >&2
      exit 1
    fi
  )
  rm -rf "${cert_dir}"
}

# Remove any stale stapled ticket before replacing the app signature.
rm -f "${APP_PATH}/Contents/CodeResources"

codesign --force --deep \
  --options runtime \
  --timestamp \
  --sign "${SIGN_IDENTITY}" \
  "${APP_PATH}"

codesign --verify --deep --strict --verbose=4 "${APP_PATH}"
assert_embedded_certificates

ZIP_PATH="$(mktemp "${RUNNER_TEMP:-/tmp}/whisk-notary.XXXXXX.zip")"
cleanup() { rm -f "${ZIP_PATH}"; }
trap cleanup EXIT

ditto -c -k --sequesterRsrc --keepParent "${APP_PATH}" "${ZIP_PATH}"
xcrun notarytool submit "${ZIP_PATH}" \
  --keychain-profile "${KEYCHAIN_PROFILE}" \
  --wait

codesign --verify --deep --strict --verbose=4 "${APP_PATH}"
assert_embedded_certificates
spctl -a -vvv -t execute "${APP_PATH}"
