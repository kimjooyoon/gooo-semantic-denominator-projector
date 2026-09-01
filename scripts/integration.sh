#!/usr/bin/env bash
set -euo pipefail

if test "$#" -ne 2; then
  echo "usage: integration.sh BINARY ROOT" >&2
  exit 64
fi

binary=$1
root=$2
snapshot() {
  find "$root" -path "$root/.git" -prune -o -type f -print0 | sort -z | xargs -0 sha256sum | sha256sum | awk '{print $1}'
}
before=$(snapshot)
output=$(mktemp -d "${TMPDIR:-/tmp}/gooo-sdp-integration.XXXXXX")
trap 'rm -rf "$output"' EXIT
"$binary" generate \
  --source "$root/.gooo/semantic-denominator-projector.gooo" \
  --cases "$root/fixtures/cases" \
  --output "$output" \
  --root "$root" >/dev/null
after=$(snapshot)
test "$before" = "$after"
test "$(find "$output" -maxdepth 1 -type f | wc -l | tr -d ' ')" = "6"
