#!/bin/sh
set -eu

BINARY=${1:?coordinator binary is required}
REPO=${2:?repository root is required}

"$BINARY" --help >/dev/null
"$BINARY" --version | grep '^symphony-knowledge-session 0.1.0-dev$' >/dev/null
"$BINARY" --descriptor | grep '"canonical_apply_enabled":false' >/dev/null

DEADLINE=$(( $(date +%s) * 1000 + 60000 ))
REQUEST=$(printf '{"protocol":"symphony.knowledge.engine-process.v1","request_id":"smoke-1","correlation_id":"smoke-1","operation":"inspect","target_engine":"symphony-knowledge-session","deadline_unix_ms":%s,"payload":{}}' "$DEADLINE")
RESPONSE=$(printf '%s' "$REQUEST" | "$BINARY")
RESPONSE_AGAIN=$(printf '%s' "$REQUEST" | "$BINARY")
test "$RESPONSE" = "$RESPONSE_AGAIN"
printf '%s\n' "$RESPONSE" | grep '"outcome":"ok"' >/dev/null
printf '%s\n' "$RESPONSE" | grep '"session_mutation_enabled":false' >/dev/null
printf '%s\n' "$RESPONSE" | grep '"response_digest":"sha256:' >/dev/null

DEADLINE=$(( $(date +%s) * 1000 + 60000 ))
CHECK=$(printf '{"protocol":"symphony.knowledge.engine-process.v1","request_id":"smoke-2","correlation_id":"smoke-2","operation":"check","target_engine":"symphony-knowledge-session","deadline_unix_ms":%s,"payload":{"expected_snapshot_digest":null,"paths":["INTENT.md","knowledge/INTENT.md"]}}' "$DEADLINE")
CHECK_RESPONSE=$(cd "$REPO" && printf '%s' "$CHECK" | "$BINARY")
printf '%s\n' "$CHECK_RESPONSE" | grep '"read_only":true' >/dev/null
printf '%s\n' "$CHECK_RESPONSE" | grep '"path":"INTENT.md"' >/dev/null

set +e
INVALID_RESPONSE=$(printf '%s' '{"protocol":"symphony.knowledge.engine-process.v1","request_id":"bad","request_id":"duplicate"}' | "$BINARY")
INVALID_STATUS=$?
set -e
test "$INVALID_STATUS" -eq 2
printf '%s\n' "$INVALID_RESPONSE" | grep '"code":"json.duplicate_key"' >/dev/null

DEADLINE=$(( $(date +%s) * 1000 + 60000 ))
RESERVED=$(printf '{"protocol":"symphony.knowledge.engine-process.v1","request_id":"smoke-3","correlation_id":"smoke-3","operation":"begin","target_engine":"symphony-knowledge-session","deadline_unix_ms":%s,"payload":{}}' "$DEADLINE")
set +e
RESERVED_RESPONSE=$(printf '%s' "$RESERVED" | "$BINARY")
RESERVED_STATUS=$?
set -e
test "$RESERVED_STATUS" -eq 4
printf '%s\n' "$RESERVED_RESPONSE" | grep '"code":"operation.unsupported"' >/dev/null

echo "knowledge session process smoke tests passed"
