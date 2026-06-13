#!/usr/bin/env bash
set -euo pipefail

ARTIFACT_PATH="${1:?usage: scripts/verify-macos-artifact.sh <app|zip|dmg>}"

if [ ! -e "${ARTIFACT_PATH}" ]; then
  echo "Artifact not found: ${ARTIFACT_PATH}" >&2
  exit 1
fi

ORIGINAL_KEYCHAINS="$(security list-keychains -d user | sed -E 's/^[[:space:]]*"?//; s/"?$//')"
VERIFY_ROOT="$(mktemp -d)"
VERIFY_ROOT="$(cd "${VERIFY_ROOT}" && pwd -P)"
VERIFY_KEYCHAIN="${VERIFY_ROOT}/verify.keychain"
VERIFY_KEYCHAIN_PASSWORD="$(openssl rand -base64 24)"
MOUNT_POINT=""

restore_keychains() {
  local keychain
  local keychains=()

  while IFS= read -r keychain; do
    [ -n "${keychain}" ] || continue
    [ -e "${keychain}" ] || continue
    keychains+=("${keychain}")
  done <<< "${ORIGINAL_KEYCHAINS}"

  if [ "${#keychains[@]}" -eq 0 ]; then
    keychains+=("${HOME}/Library/Keychains/login.keychain-db")
  fi

  security list-keychains -d user -s "${keychains[@]}" >/dev/null
}

cleanup() {
  if [ -n "${MOUNT_POINT}" ] && mount | grep -q "on ${MOUNT_POINT} "; then
    hdiutil detach "${MOUNT_POINT}" -quiet || true
  fi
  restore_keychains
  rm -rf "${VERIFY_ROOT}"
}
trap cleanup EXIT

security create-keychain -p "${VERIFY_KEYCHAIN_PASSWORD}" "${VERIFY_KEYCHAIN}"
security set-keychain-settings -lut 21600 "${VERIFY_KEYCHAIN}"
security unlock-keychain -p "${VERIFY_KEYCHAIN_PASSWORD}" "${VERIFY_KEYCHAIN}"
security list-keychains -d user -s "${VERIFY_KEYCHAIN}"

find_app() {
  local root="$1"
  find "${root}" -maxdepth 3 -type d -name "*.app" -print -quit
}

APP_PATH=""
case "${ARTIFACT_PATH}" in
  *.app)
    APP_PATH="$(cd "$(dirname "${ARTIFACT_PATH}")" && pwd)/$(basename "${ARTIFACT_PATH}")"
    ;;
  *.zip)
    EXTRACT_DIR="${VERIFY_ROOT}/zip"
    mkdir -p "${EXTRACT_DIR}"
    ditto -x -k "${ARTIFACT_PATH}" "${EXTRACT_DIR}"
    APP_PATH="$(find_app "${EXTRACT_DIR}")"
    ;;
  *.dmg)
    xcrun stapler validate "${ARTIFACT_PATH}"
    MOUNT_POINT="${VERIFY_ROOT}/mnt"
    mkdir -p "${MOUNT_POINT}"
    hdiutil attach "${ARTIFACT_PATH}" -readonly -nobrowse -mountpoint "${MOUNT_POINT}" -quiet
    APP_PATH="$(find_app "${MOUNT_POINT}")"
    ;;
  *)
    echo "Unsupported macOS artifact type: ${ARTIFACT_PATH}" >&2
    exit 1
    ;;
esac

if [ -z "${APP_PATH}" ] || [ ! -d "${APP_PATH}" ]; then
  echo "No .app bundle found in artifact: ${ARTIFACT_PATH}" >&2
  exit 1
fi

codesign --verify --deep --strict --verbose=4 "${APP_PATH}"
spctl -a -vvv -t execute "${APP_PATH}"

CERT_DIR="${VERIFY_ROOT}/certs"
mkdir -p "${CERT_DIR}"
(
  cd "${CERT_DIR}"
  codesign -d --extract-certificates "${APP_PATH}" >/dev/null
  if ! ls codesign* >/dev/null 2>&1; then
    echo "Signed app does not contain embedded signing certificates: ${APP_PATH}" >&2
    exit 1
  fi
)
