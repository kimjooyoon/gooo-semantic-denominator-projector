#!/usr/bin/env bash
set -u

if test "$#" -lt 2; then
  echo "usage: measure-step.sh METRICS_FILE COMMAND [ARGS...]" >&2
  exit 64
fi

metrics=$1
shift
rss_file="${metrics}.rss"
start=$(date +%s%N)
set +e
/usr/bin/time -f '%M' -o "$rss_file" -- "$@"
status=$?
set -e
end=$(date +%s%N)
wall_ms=$(( (end - start) / 1000000 ))
if test "$wall_ms" -lt 1; then wall_ms=1; fi
peak_rss_kib=$(tr -d '[:space:]' < "$rss_file")
if test -z "$peak_rss_kib"; then peak_rss_kib=0; fi
printf 'wall_ms=%s\npeak_rss_kib=%s\n' "$wall_ms" "$peak_rss_kib" > "$metrics"
rm -f "$rss_file"
exit "$status"
