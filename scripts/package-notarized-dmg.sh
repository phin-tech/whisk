#!/usr/bin/env bash
set -euo pipefail

: "${KEYCHAIN_PROFILE:?notary keychain profile name required}"

APP_PATH="${1:-bin/Whisk.app}"
DMG_PATH="${2:-bin/Whisk-macos.dmg}"
VOLNAME="${VOLNAME:-Whisk}"

if [ ! -d "${APP_PATH}" ]; then
  echo "App bundle not found: ${APP_PATH}" >&2
  exit 1
fi

# The app must already be signed, notarized, and stapled before it is placed into the DMG,
# so the copy inside the DMG carries its own offline ticket. Fail fast otherwise.
xcrun stapler validate "${APP_PATH}"

mkdir -p "$(dirname "${DMG_PATH}")"
rm -f "${DMG_PATH}"

hdiutil create \
  -volname "${VOLNAME}" \
  -srcfolder "${APP_PATH}" \
  -ov \
  -format UDZO \
  "${DMG_PATH}"

xcrun notarytool submit "${DMG_PATH}" \
  --keychain-profile "${KEYCHAIN_PROFILE}" \
  --wait

xcrun stapler staple "${DMG_PATH}"
xcrun stapler validate "${DMG_PATH}"

