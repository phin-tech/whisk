#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

assert_file_contains() {
  file="$1"
  needle="$2"
  if ! grep -Fq "$needle" "$ROOT/$file"; then
    printf '%s missing expected text: %s\n' "$file" "$needle" >&2
    exit 1
  fi
}

assert_file_contains "build/linux/Taskfile.yml" "task: build:cli"
assert_file_contains "build/linux/nfpm/nfpm.yaml" 'src: "./bin/whisk"'
assert_file_contains "build/linux/nfpm/nfpm.yaml" 'dst: "/usr/local/bin/whisk"'

assert_file_contains ".github/workflows/release.yml" "whisk-aarch64-apple-darwin.tar.gz"
assert_file_contains ".github/workflows/release.yml" "whisk-x86_64-apple-darwin.tar.gz"
assert_file_contains ".github/workflows/release.yml" "whisk-x86_64-unknown-linux-gnu.tar.gz"
assert_file_contains ".github/workflows/release.yml" "whisk-aarch64-unknown-linux-gnu.tar.gz"
assert_file_contains ".github/workflows/release.yml" "sha256"
