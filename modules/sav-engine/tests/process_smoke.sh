#!/bin/sh
set -eu
BINARY=${1:?SAV binary is required}
"$BINARY" --help | grep '^Usage: symphony-sav ' >/dev/null
"$BINARY" --version | grep '^symphony-sav 0.1.0-dev$' >/dev/null
"$BINARY" --descriptor | grep '"protocol":"symphony.knowledge.engine-descriptor.v2"' >/dev/null
"$BINARY" --descriptor | grep '"descriptor_digest":"sha256:' >/dev/null
"$BINARY" --descriptor | grep '"json_values":16384' >/dev/null
DEADLINE=$(( $(date +%s) * 1000 + 60000 ))
REQUEST=$(printf '{"protocol":"symphony.knowledge.engine-process.v1","request_id":"smoke","correlation_id":"smoke","operation":"inspect","target_engine":"symphony-sav","deadline_unix_ms":%s,"payload":{}}' "$DEADLINE")
RESPONSE=$(printf '%s' "$REQUEST" | "$BINARY")
printf '%s\n' "$RESPONSE" | grep '"outcome":"ok"' >/dev/null
printf '%s\n' "$RESPONSE" | grep '"caller_neutral":true' >/dev/null
set +e
BAD=$(printf '%s' "$REQUEST" | sed 's/"inspect"/"apply"/' | "$BINARY")
STATUS=$?
set -e
test "$STATUS" -eq 4
printf '%s\n' "$BAD" | grep '"code":"operation.unsupported"' >/dev/null
echo "SAV engine process smoke tests passed"
