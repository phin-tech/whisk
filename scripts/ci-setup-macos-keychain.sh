#!/usr/bin/env bash
set -euo pipefail

: "${APPLE_CERTIFICATE:?base64-encoded .p12 required}"
: "${APPLE_CERTIFICATE_PASSWORD:?.p12 password required}"
: "${RUNNER_TEMP:?RUNNER_TEMP must be set}"

RUNNER_TEMP="$(cd "${RUNNER_TEMP}" && pwd -P)"
KEYCHAIN_NAME="${RUNNER_TEMP}/whisk-build.keychain"
KEYCHAIN_PASSWORD="$(openssl rand -base64 24)"
CERT_PATH="${RUNNER_TEMP}/cert.p12"
APPLE_ROOT_CA_PATH="${RUNNER_TEMP}/AppleIncRootCertificate.cer"
DEVELOPER_ID_CA_PATH="${RUNNER_TEMP}/DeveloperIDG2CA.cer"

cleanup() { rm -f "${CERT_PATH}" "${APPLE_ROOT_CA_PATH}" "${DEVELOPER_ID_CA_PATH}"; }
trap cleanup EXIT

cert_len=${#APPLE_CERTIFICATE}
cert_sha=$(printf '%s' "${APPLE_CERTIFICATE}" | shasum -a 256 | awk '{print $1}')
echo "APPLE_CERTIFICATE env: length=${cert_len} sha256=${cert_sha}"

printf '%s' "${APPLE_CERTIFICATE}" | base64 --decode > "${CERT_PATH}"
echo "Decoded p12: size=$(wc -c < "${CERT_PATH}" | tr -d ' ') bytes, type=$(file -b "${CERT_PATH}")"
openssl pkcs12 -info -noout -in "${CERT_PATH}" -passin env:APPLE_CERTIFICATE_PASSWORD >/dev/null

curl -fsSL https://www.apple.com/appleca/AppleIncRootCertificate.cer -o "${APPLE_ROOT_CA_PATH}"
curl -fsSL https://www.apple.com/certificateauthority/DeveloperIDG2CA.cer -o "${DEVELOPER_ID_CA_PATH}"

security create-keychain -p "${KEYCHAIN_PASSWORD}" "${KEYCHAIN_NAME}"
security set-keychain-settings -lut 21600 "${KEYCHAIN_NAME}"
security unlock-keychain -p "${KEYCHAIN_PASSWORD}" "${KEYCHAIN_NAME}"

security list-keychains -d user -s "${KEYCHAIN_NAME}"

security import "${APPLE_ROOT_CA_PATH}" \
  -k "${KEYCHAIN_NAME}" \
  -T /usr/bin/codesign \
  -T /usr/bin/security

security import "${DEVELOPER_ID_CA_PATH}" \
  -k "${KEYCHAIN_NAME}" \
  -T /usr/bin/codesign \
  -T /usr/bin/security

security import "${CERT_PATH}" \
  -k "${KEYCHAIN_NAME}" \
  -P "${APPLE_CERTIFICATE_PASSWORD}" \
  -T /usr/bin/codesign \
  -T /usr/bin/security \
  -T /usr/bin/productbuild

security set-key-partition-list \
  -S apple-tool:,apple: \
  -s -k "${KEYCHAIN_PASSWORD}" "${KEYCHAIN_NAME}" >/dev/null

echo "Imported signing identities:"
security find-identity -v -p codesigning "${KEYCHAIN_NAME}"
