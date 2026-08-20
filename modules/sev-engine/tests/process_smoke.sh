#!/bin/sh
set -eu
BINARY=${1:?SEV binary is required}
"$BINARY" --help | grep '^Usage: symphony-sev ' >/dev/null
"$BINARY" --version | grep '^symphony-sev 0.1.0-dev$' >/dev/null
"$BINARY" --descriptor | grep '"protocol":"symphony.knowledge.engine-descriptor.v2"' >/dev/null
"$BINARY" --descriptor | grep '"descriptor_digest":"sha256:' >/dev/null
"$BINARY" --descriptor | grep '"json_values":16384' >/dev/null
DEADLINE=$(( $(date +%s) * 1000 + 60000 ))
REQUEST=$(printf '{"protocol":"symphony.knowledge.engine-process.v1","request_id":"smoke","correlation_id":"smoke","operation":"inspect","target_engine":"symphony-sev","deadline_unix_ms":%s,"payload":{}}' "$DEADLINE")
RESPONSE=$(printf '%s' "$REQUEST" | "$BINARY")
printf '%s\n' "$RESPONSE" | grep '"outcome":"ok"' >/dev/null
printf '%s\n' "$RESPONSE" | grep '"caller_neutral":true' >/dev/null
echo "SEV engine process smoke tests passed"
