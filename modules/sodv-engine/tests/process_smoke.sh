#!/bin/sh
set -eu

engine=$1
repository=$2

"$engine" --help | grep -F "Usage: symphony-sodv" >/dev/null
"$engine" --version | grep -F "symphony-sodv 0.1.0-dev" >/dev/null
"$engine" --descriptor | jq -e '
  .engine_id == "symphony-sodv" and
  .language == "C++26" and
  .thermal_path == "freezing" and
  .network_access == false and
  .canonical_apply_enabled == false
' >/dev/null

deadline=$(( $(date +%s) * 1000 + 60000 ))
request=$(jq -cn --argjson deadline "$deadline" '{
  protocol: "symphony.knowledge.engine-process.v1",
  request_id: "smoke-request",
  correlation_id: "smoke-correlation",
  operation: "check",
  target_engine: "symphony-sodv",
  deadline_unix_ms: $deadline,
  payload: {expected_ledger_digest: null}
}')

checked=$(cd "$repository" && printf '%s\n' "$request" | "$engine")
printf '%s\n' "$checked" | jq -e '
  .outcome == "ok" and
  .result.summary.state == "valid" and
  .result.canonical_apply_enabled == false
' >/dev/null

projection_request=$(printf '%s\n' "$request" | jq -c '
  .operation = "project" | .payload = {format: "json"}
')
(cd "$repository" && printf '%s\n' "$projection_request" | "$engine") | jq -e --argjson checked "$checked" '
  .outcome == "ok" and
  (.result |
    .noncanonical == true and .rebuildable == true and
    .record_count == (.records | length) and
    .transaction_count == (.transactions | length) and
    .transaction_count == ([.records[] | select(.record_type == "authorization")] | length) and
    .record_count == $checked.result.records_checked and
    .transaction_count == $checked.result.transactions_checked and
    [.records[0:3][].release_record_id] == ["SODV-REL-001", "SODV-REL-002", "SODV-REL-003"] and
    .transactions[0].authorization_record_id == "SODV-REL-001" and
    .transactions[0].latest_correction_record == "SODV-REL-002" and
    .transactions[0].completion_record == "SODV-REL-003" and
    .transactions[0].state == "completed")
' >/dev/null

reserved=$(jq -cn --argjson deadline "$deadline" '{
  protocol: "symphony.knowledge.engine-process.v1",
  request_id: "smoke-reserved",
  correlation_id: "smoke-correlation",
  operation: "publish",
  target_engine: "symphony-sodv",
  deadline_unix_ms: $deadline,
  payload: {}
}')

set +e
response=$(cd "$repository" && printf '%s\n' "$reserved" | "$engine")
status=$?
set -e
[ "$status" -eq 4 ]
printf '%s\n' "$response" | jq -e '.outcome == "error" and .error.code == "operation.unsupported"' >/dev/null

printf '%s\n' "sodv process smoke passed"
