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

codesign_args=(--force --options runtime --timestamp)
if [ -n "${ENTITLEMENTS:-}" ]; then
  codesign_args+=(--entitlements "${ENTITLEMENTS}")
fi
if [ -n "${SIGN_KEYCHAIN:-}" ]; then
  codesign_args+=(--keychain "${SIGN_KEYCHAIN}")
fi

# Sign inside-out: every nested Mach-O must carry a Developer ID signature with the
# hardened runtime and a secure timestamp, or notarization rejects the bundle. Sign the
# bundled daemon helper(s) first, then the bundle itself (which signs the main executable
# and seals all resources). --deep is intentionally avoided; it is unreliable for nested
# executables and cannot apply entitlements per binary.
HELPER="${APP_PATH}/Contents/MacOS/whisk"
if [ -f "${HELPER}" ]; then
  codesign "${codesign_args[@]}" --sign "${SIGN_IDENTITY}" "${HELPER}"
fi

codesign "${codesign_args[@]}" --sign "${SIGN_IDENTITY}" "${APP_PATH}"

codesign --verify --deep --strict --verbose=4 "${APP_PATH}"
assert_embedded_certificates

ZIP_PATH="$(mktemp "${RUNNER_TEMP:-/tmp}/whisk-notary.XXXXXX.zip")"
cleanup() { rm -f "${ZIP_PATH}"; }
trap cleanup EXIT

ditto -c -k --sequesterRsrc --keepParent "${APP_PATH}" "${ZIP_PATH}"
xcrun notarytool submit "${ZIP_PATH}" \
  --keychain-profile "${KEYCHAIN_PROFILE}" \
  --wait

# Staple the notarization ticket into the .app so it passes Gatekeeper offline, including
# when dragged out of the DMG. This must happen before the app is packaged into the DMG.
xcrun stapler staple "${APP_PATH}"
xcrun stapler validate "${APP_PATH}"

codesign --verify --deep --strict --verbose=4 "${APP_PATH}"
assert_embedded_certificates
spctl -a -vvv -t execute "${APP_PATH}"
