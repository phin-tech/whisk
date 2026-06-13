#!/usr/bin/env bash
set -euo pipefail

: "${APPLE_ID:?Apple ID required}"
: "${APPLE_PASSWORD:?app-specific password required}"
: "${APPLE_TEAM_ID:?Apple team ID required}"
: "${KEYCHAIN_PROFILE:?notary keychain profile name required}"

xcrun notarytool store-credentials "${KEYCHAIN_PROFILE}" \
  --apple-id "${APPLE_ID}" \
  --password "${APPLE_PASSWORD}" \
  --team-id "${APPLE_TEAM_ID}"
