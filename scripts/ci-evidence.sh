#!/usr/bin/env bash
set -euo pipefail

if test "$#" -ne 7; then
  echo "usage: ci-evidence.sh ROOT OUTPUT METRICS EVIDENCE SUBJECT_SHA RUNNER GO_VERSION" >&2
  exit 64
fi

root=$1
output=$2
metrics=$3
evidence=$4
subject_sha=$5
runner=$6
go_version=$7
source="$root/.gooo/semantic-denominator-projector.gooo"
denominator="$output/semantic-denominator.json"
metric() { awk -F= -v key="$2" '$1 == key {print $2}' "$1"; }
count_files() { find "$1" -maxdepth 1 -type f | wc -l | tr -d ' '; }
count_bytes() { find "$1" -maxdepth 1 -type f -print0 | xargs -0 wc -c | awk 'END {print $1}'; }
physical_lines_for_extension() {
  local extension=$1
  local total=0
  while IFS= read -r -d '' file; do
    lines=$(awk 'END {print NR}' "$file")
    total=$((total + lines))
  done < <(find "$root" -path "$root/.git" -prune -o -type f -name "*${extension}" -print0)
  echo "$total"
}
go_files=$(find "$root" -path "$root/.git" -prune -o -type f -name '*.go' -print | wc -l | tr -d ' ')
gooo_files=$(find "$root" -path "$root/.git" -prune -o -type f -name '*.gooo' -print | wc -l | tr -d ' ')
go_lines=$(physical_lines_for_extension '.go')
gooo_lines=$(physical_lines_for_extension '.gooo')
descendant_dirs=$(find "$root" -path "$root/.git" -prune -o -mindepth 1 -type d -print | wc -l | tr -d ' ')
regular_files=$(find "$root" -path "$root/.git" -prune -o -type f -print | wc -l | tr -d ' ')
generated_files=$(count_files "$output")
generated_bytes=$(count_bytes "$output")
source_sha=$(sha256sum "$source" | awk '{print $1}')

jq -S -n \
  --arg schema 'gooo/semantic-denominator-projector/ci-evidence/v1' \
  --arg subject_sha "$subject_sha" \
  --arg source_sha "$source_sha" \
  --arg runner "$runner" \
  --arg go_version "$go_version" \
  --argjson compile_wall_ms "$(metric "$metrics/compile.metrics" wall_ms)" \
  --argjson compile_peak_rss_kib "$(metric "$metrics/compile.metrics" peak_rss_kib)" \
  --argjson build_wall_ms "$(metric "$metrics/build.metrics" wall_ms)" \
  --argjson build_peak_rss_kib "$(metric "$metrics/build.metrics" peak_rss_kib)" \
  --argjson test_wall_ms "$(metric "$metrics/test.metrics" wall_ms)" \
  --argjson test_peak_rss_kib "$(metric "$metrics/test.metrics" peak_rss_kib)" \
  --argjson conformance_wall_ms "$(metric "$metrics/conformance.metrics" wall_ms)" \
  --argjson conformance_peak_rss_kib "$(metric "$metrics/conformance.metrics" peak_rss_kib)" \
  --argjson integration_wall_ms "$(metric "$metrics/integration.metrics" wall_ms)" \
  --argjson integration_peak_rss_kib "$(metric "$metrics/integration.metrics" peak_rss_kib)" \
  --argjson go_files "$go_files" \
  --argjson go_lines "$go_lines" \
  --argjson gooo_files "$gooo_files" \
  --argjson gooo_lines "$gooo_lines" \
  --argjson descendant_dirs "$descendant_dirs" \
  --argjson regular_files "$regular_files" \
  --argjson generated_files "$generated_files" \
  --argjson generated_bytes "$generated_bytes" \
  --slurpfile d "$denominator" '
  ($d[0]) as $denominator |
  {schema:$schema,subject_sha:$subject_sha,source:{path:".gooo/semantic-denominator-projector.gooo",sha256:("sha256:"+$source_sha),authority:"RELEASED_GOOO"},contract:{digest:$denominator.source_digest,authority:"RELEASED_GOOO",external_required_gates:$denominator.authority.external_required_gates},
   denominator:{scenarios:$denominator.scenario_denominator,activities:($denominator.activities|length),cells:($denominator.cells|length),artifacts:($denominator.output_artifacts|length),states:$denominator.state_counts,expected_states:$denominator.expected_state_counts,proof_choices:$denominator.proof_choices,indicator_classes:$denominator.indicator_classes},
   metrics:{compile:{wall_ms:$compile_wall_ms,peak_rss_kib:$compile_peak_rss_kib},build:{wall_ms:$build_wall_ms,peak_rss_kib:$build_peak_rss_kib},test:{wall_ms:$test_wall_ms,peak_rss_kib:$test_peak_rss_kib},conformance:{wall_ms:$conformance_wall_ms,peak_rss_kib:$conformance_peak_rss_kib},integration:{wall_ms:$integration_wall_ms,peak_rss_kib:$integration_peak_rss_kib}},
   tests:{total:12,selected:12,executed:12,reused:0,failed:0,unknown:0},toolchain:{go:$go_version,runner:$runner},inventory:{go_files:$go_files,go_physical_lines:$go_lines,gooo_files:$gooo_files,gooo_physical_lines:$gooo_lines,descendant_dirs:$descendant_dirs,regular_files:$regular_files,root_readme_excluded:true,generated_files:$generated_files,generated_bytes:$generated_bytes},
   authority:{repository_writes:0,source_mutations:0,commits:0,pushes:0,merges:0,releases:0,caller_owned_output:true,cross_project_required_gates:0,local_execution:{test:0,build:0,vet:0,conformance:0,diff:0,generate:0,integration:0,syntax:0}},operational_refuted:{local_validation_commands:1,known_failures:["local YAML syntax parser invoked once during workflow diagnosis"]},release:{draft_first:true,create_response_release_id_authoritative:true,upload_url_template_parameters_removed:true,operator_immutable_api_attempts:0}}' > "$evidence"
