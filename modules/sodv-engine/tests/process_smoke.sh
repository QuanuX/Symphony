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

(cd "$repository" && printf '%s\n' "$request" | "$engine") | jq -e '
  .outcome == "ok" and
  .result.summary.state == "valid" and
  .result.records_checked == 3 and
  .result.transactions_checked == 1 and
  .result.canonical_apply_enabled == false
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
