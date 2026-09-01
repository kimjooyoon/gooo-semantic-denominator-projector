#!/usr/bin/env bash
set -euo pipefail

if test "$#" -ne 4; then
  echo "usage: release-assets.sh ROOT GENERATED EVIDENCE ASSETS" >&2
  exit 64
fi

root=$1
generated=$2
evidence=$3
assets=$4
mkdir -p "$assets"
cp "$root/.gooo/semantic-denominator-projector.gooo" "$assets/semantic-denominator-projector.gooo"
cp "$root/contracts/release-lock-v1.json" "$assets/release-lock-v1.json"
cp "$evidence" "$assets/ci-evidence.json"
for name in semantic-denominator.json semantic-distribution.json generated-assertions.json projection-events.ndjson replay-receipt.json report.md; do
  cp "$generated/$name" "$assets/$name"
done
(cd "$assets" && sha256sum semantic-denominator-projector.gooo release-lock-v1.json ci-evidence.json semantic-denominator.json semantic-distribution.json generated-assertions.json projection-events.ndjson replay-receipt.json report.md > SHA256SUMS)
