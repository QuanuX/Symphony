#!/bin/sh
set -eu

BINARY=${1:?SACV binary is required}
REPO=${2:?repository root is required}

"$BINARY" --help | grep '^Usage: symphony-sacv ' >/dev/null
"$BINARY" --version | grep '^symphony-sacv 0.1.0-dev$' >/dev/null
"$BINARY" --descriptor | grep '"canonical_apply_enabled":false' >/dev/null
"$BINARY" --descriptor | grep '"openapi_target":"3.2.0"' >/dev/null
"$BINARY" --descriptor | grep '"network_listener":false' >/dev/null

DEADLINE=$(( $(date +%s) * 1000 + 60000 ))
INSPECT=$(printf '{"protocol":"symphony.knowledge.engine-process.v1","request_id":"smoke-inspect","correlation_id":"smoke-inspect","operation":"inspect","target_engine":"symphony-sacv","deadline_unix_ms":%s,"payload":{}}' "$DEADLINE")
INSPECT_RESPONSE=$(printf '%s' "$INSPECT" | "$BINARY")
printf '%s\n' "$INSPECT_RESPONSE" | grep '"outcome":"ok"' >/dev/null
printf '%s\n' "$INSPECT_RESPONSE" | grep '"engine_decides_ownership":false' >/dev/null

DEADLINE=$(( $(date +%s) * 1000 + 60000 ))
CHECK=$(printf '{"protocol":"symphony.knowledge.engine-process.v1","request_id":"smoke-check","correlation_id":"smoke-check","operation":"check","target_engine":"symphony-sacv","deadline_unix_ms":%s,"payload":{"expected_registry_digest":null}}' "$DEADLINE")
CHECK_RESPONSE=$(cd "$REPO" && printf '%s' "$CHECK" | "$BINARY")
printf '%s\n' "$CHECK_RESPONSE" | grep '"state":"valid"' >/dev/null
printf '%s\n' "$CHECK_RESPONSE" | grep '"entries_checked":0' >/dev/null
printf '%s\n' "$CHECK_RESPONSE" | grep '"read_only":true' >/dev/null

DEADLINE=$(( $(date +%s) * 1000 + 60000 ))
PROJECT=$(printf '{"protocol":"symphony.knowledge.engine-process.v1","request_id":"smoke-project","correlation_id":"smoke-project","operation":"project","target_engine":"symphony-sacv","deadline_unix_ms":%s,"payload":{"format":"json"}}' "$DEADLINE")
PROJECT_RESPONSE=$(cd "$REPO" && printf '%s' "$PROJECT" | "$BINARY")
PROJECT_AGAIN=$(cd "$REPO" && printf '%s' "$PROJECT" | "$BINARY")
test "$PROJECT_RESPONSE" = "$PROJECT_AGAIN"
printf '%s\n' "$PROJECT_RESPONSE" | grep '"noncanonical":true' >/dev/null
printf '%s\n' "$PROJECT_RESPONSE" | grep '"rebuildable":true' >/dev/null

set +e
INVALID_RESPONSE=$(printf '%s' '{"protocol":"symphony.knowledge.engine-process.v1","request_id":"bad","request_id":"duplicate"}' | "$BINARY")
INVALID_STATUS=$?
set -e
test "$INVALID_STATUS" -eq 2
printf '%s\n' "$INVALID_RESPONSE" | grep '"code":"json.duplicate_key"' >/dev/null

DEADLINE=$(( $(date +%s) * 1000 + 60000 ))
APPLY=$(printf '{"protocol":"symphony.knowledge.engine-process.v1","request_id":"smoke-apply","correlation_id":"smoke-apply","operation":"apply","target_engine":"symphony-sacv","deadline_unix_ms":%s,"payload":{}}' "$DEADLINE")
set +e
APPLY_RESPONSE=$(printf '%s' "$APPLY" | "$BINARY")
APPLY_STATUS=$?
set -e
test "$APPLY_STATUS" -eq 4
printf '%s\n' "$APPLY_RESPONSE" | grep '"code":"operation.unsupported"' >/dev/null

echo "SACV engine process smoke tests passed"
