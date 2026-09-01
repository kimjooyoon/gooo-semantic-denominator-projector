#!/usr/bin/env bash
set -euo pipefail

if test "$#" -ne 3; then
  echo "usage: conformance.sh BINARY ROOT OUTPUT" >&2
  exit 64
fi

binary=$1
root=$2
output=$3
mkdir -p "$output"
"$binary" conformance \
  --source "$root/.gooo/semantic-denominator-projector.gooo" \
  --cases "$root/fixtures/cases" \
  --output "$output" \
  --root "$root"
