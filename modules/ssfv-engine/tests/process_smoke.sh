#!/bin/sh
set -eu

BINARY=${1:?SSFV binary is required}
REPO=${2:?repository root is required}

"$BINARY" --help | grep '^Usage: symphony-ssfv ' >/dev/null
"$BINARY" --version | grep '^symphony-ssfv 0.1.0-dev$' >/dev/null
"$BINARY" --descriptor | grep '"canonical_apply_enabled":false' >/dev/null
"$BINARY" --descriptor | grep '"thermal_path":"freezing"' >/dev/null
"$BINARY" --descriptor | grep '"install_state":"installed_undocked"' >/dev/null
"$BINARY" --descriptor | grep '"network_listener":false' >/dev/null

DEADLINE=$(( $(date +%s) * 1000 + 60000 ))
INSPECT=$(printf '{"protocol":"symphony.knowledge.engine-process.v1","request_id":"smoke-inspect","correlation_id":"smoke-inspect","operation":"inspect","target_engine":"symphony-ssfv","deadline_unix_ms":%s,"payload":{}}' "$DEADLINE")
INSPECT_RESPONSE=$(printf '%s' "$INSPECT" | "$BINARY")
printf '%s\n' "$INSPECT_RESPONSE" | grep '"outcome":"ok"' >/dev/null
printf '%s\n' "$INSPECT_RESPONSE" | grep '"engine_decides_feature_worthiness":false' >/dev/null

DEADLINE=$(( $(date +%s) * 1000 + 60000 ))
CHECK=$(printf '{"protocol":"symphony.knowledge.engine-process.v1","request_id":"smoke-check","correlation_id":"smoke-check","operation":"check","target_engine":"symphony-ssfv","deadline_unix_ms":%s,"payload":{"expected_namespace_digest":null,"expected_registry_digest":null,"freshness":"disabled","baseline":null}}' "$DEADLINE")
CHECK_RESPONSE=$(cd "$REPO" && printf '%s' "$CHECK" | "$BINARY")
printf '%s\n' "$CHECK_RESPONSE" | grep '"outcome":"ok"' >/dev/null
printf '%s\n' "$CHECK_RESPONSE" | grep '"coverage_state":"partial"' >/dev/null
printf '%s\n' "$CHECK_RESPONSE" | grep '"structural_state":"valid"' >/dev/null
printf '%s\n' "$CHECK_RESPONSE" | grep '"read_only":true' >/dev/null

DEADLINE=$(( $(date +%s) * 1000 + 60000 ))
GRAPH=$(printf '{"protocol":"symphony.knowledge.engine-process.v1","request_id":"smoke-graph","correlation_id":"smoke-graph","operation":"graph","target_engine":"symphony-ssfv","deadline_unix_ms":%s,"payload":{"format":"json"}}' "$DEADLINE")
GRAPH_RESPONSE=$(cd "$REPO" && printf '%s' "$GRAPH" | "$BINARY")
GRAPH_AGAIN=$(cd "$REPO" && printf '%s' "$GRAPH" | "$BINARY")
test "$GRAPH_RESPONSE" = "$GRAPH_AGAIN"
printf '%s\n' "$GRAPH_RESPONSE" | grep '"noncanonical":true' >/dev/null
printf '%s\n' "$GRAPH_RESPONSE" | grep '"rebuildable":true' >/dev/null

set +e
INVALID_RESPONSE=$(printf '%s' '{"protocol":"symphony.knowledge.engine-process.v1","request_id":"bad","request_id":"duplicate"}' | "$BINARY")
INVALID_STATUS=$?
set -e
test "$INVALID_STATUS" -eq 2
printf '%s\n' "$INVALID_RESPONSE" | grep '"code":"json.duplicate_key"' >/dev/null

DEADLINE=$(( $(date +%s) * 1000 + 60000 ))
APPLY=$(printf '{"protocol":"symphony.knowledge.engine-process.v1","request_id":"smoke-apply","correlation_id":"smoke-apply","operation":"apply","target_engine":"symphony-ssfv","deadline_unix_ms":%s,"payload":{}}' "$DEADLINE")
set +e
APPLY_RESPONSE=$(printf '%s' "$APPLY" | "$BINARY")
APPLY_STATUS=$?
set -e
test "$APPLY_STATUS" -eq 4
printf '%s\n' "$APPLY_RESPONSE" | grep '"code":"operation.unsupported"' >/dev/null

echo "SSFV engine process smoke tests passed"
