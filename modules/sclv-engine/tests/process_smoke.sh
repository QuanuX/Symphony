#!/bin/sh
set -eu

ENGINE=${1:?SCLV engine binary is required}
LOCAL_GIT=${2:?local Git adapter binary is required}
AIRGAP=${3:?air-gap adapter binary is required}
REPO=${4:?repository root is required}

"$ENGINE" --help | grep '^Usage: symphony-sclv ' >/dev/null
"$ENGINE" --version | grep '^symphony-sclv 0.1.0-dev$' >/dev/null
"$ENGINE" --descriptor | grep '"canonical_apply_enabled":false' >/dev/null
"$ENGINE" --descriptor | grep '"network_listener":false' >/dev/null
"$LOCAL_GIT" --version | grep '^symphony-sclv-evidence-local-git 0.1.0-dev$' >/dev/null
"$AIRGAP" --version | grep '^symphony-sclv-evidence-airgap 0.1.0-dev$' >/dev/null

DEADLINE=$(( $(date +%s) * 1000 + 60000 ))
INSPECT=$(printf '{"protocol":"symphony.knowledge.engine-process.v1","request_id":"smoke-inspect","correlation_id":"smoke-inspect","operation":"inspect","target_engine":"symphony-sclv","deadline_unix_ms":%s,"payload":{}}' "$DEADLINE")
INSPECT_RESPONSE=$(printf '%s' "$INSPECT" | "$ENGINE")
printf '%s\n' "$INSPECT_RESPONSE" | grep '"outcome":"ok"' >/dev/null
printf '%s\n' "$INSPECT_RESPONSE" | grep '"read_only":true' >/dev/null
printf '%s\n' "$INSPECT_RESPONSE" | grep '"response_digest":"sha256:' >/dev/null

DEADLINE=$(( $(date +%s) * 1000 + 60000 ))
CHECK=$(printf '{"protocol":"symphony.knowledge.engine-process.v1","request_id":"smoke-check","correlation_id":"smoke-check","operation":"check","target_engine":"symphony-sclv","deadline_unix_ms":%s,"payload":{"expected_ledger_digest":null}}' "$DEADLINE")
CHECK_RESPONSE=$(cd "$REPO" && printf '%s' "$CHECK" | "$ENGINE")
printf '%s\n' "$CHECK_RESPONSE" | grep '"state":"valid"' >/dev/null
printf '%s\n' "$CHECK_RESPONSE" | grep '"violation":0' >/dev/null
printf '%s\n' "$CHECK_RESPONSE" | grep '"read_only":true' >/dev/null

DEADLINE=$(( $(date +%s) * 1000 + 60000 ))
PROJECT=$(printf '{"protocol":"symphony.knowledge.engine-process.v1","request_id":"smoke-project","correlation_id":"smoke-project","operation":"project","target_engine":"symphony-sclv","deadline_unix_ms":%s,"payload":{"format":"json"}}' "$DEADLINE")
PROJECT_RESPONSE=$(cd "$REPO" && printf '%s' "$PROJECT" | "$ENGINE")
PROJECT_AGAIN=$(cd "$REPO" && printf '%s' "$PROJECT" | "$ENGINE")
test "$PROJECT_RESPONSE" = "$PROJECT_AGAIN"
printf '%s\n' "$PROJECT_RESPONSE" | grep '"noncanonical":true' >/dev/null
printf '%s\n' "$PROJECT_RESPONSE" | grep '"rebuildable":true' >/dev/null

REVISION=$(cd "$REPO" && git rev-parse HEAD)
case ${#REVISION} in
  40) SCHEME=git-sha1 ;;
  64) SCHEME=git-sha256 ;;
  *) echo "unexpected Git object format" >&2; exit 1 ;;
esac
DEADLINE=$(( $(date +%s) * 1000 + 60000 ))
LOCAL_REQUEST=$(printf '{"protocol":"symphony.knowledge.engine-process.v1","request_id":"smoke-local","correlation_id":"smoke-local","operation":"normalize","target_engine":"symphony-sclv-evidence-local-git","deadline_unix_ms":%s,"payload":{"observed_at":"2026-07-21T16:00:00Z","source_reference":"local-smoke","revision_scheme":"%s","revision_value":"%s"}}' "$DEADLINE" "$SCHEME" "$REVISION")
LOCAL_RESPONSE=$(cd "$REPO" && printf '%s' "$LOCAL_REQUEST" | "$LOCAL_GIT")
printf '%s\n' "$LOCAL_RESPONSE" | grep '"provider_namespace":"local-git"' >/dev/null
printf '%s\n' "$LOCAL_RESPONSE" | grep '"state":"not_asserted"' >/dev/null
printf '%s\n' "$LOCAL_RESPONSE" | grep '"evidence_digest":"sha256:' >/dev/null

DEADLINE=$(( $(date +%s) * 1000 + 60000 ))
AIRGAP_REQUEST=$(printf '{"protocol":"symphony.knowledge.engine-process.v1","request_id":"smoke-airgap","correlation_id":"smoke-airgap","operation":"normalize","target_engine":"symphony-sclv-evidence-airgap","deadline_unix_ms":%s,"payload":{"observed_at":"2026-07-21T16:00:00Z","source_reference":"airgap-smoke","repository":{"revision_scheme":"git-sha1","revision_value":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","tree_digest":"sha256:1111111111111111111111111111111111111111111111111111111111111111"},"change_request":{"state":"not_applicable","provider":"not_applicable","id":"not_applicable","reference":"not_applicable","absence_reason":"air-gapped change has no change request"},"ratification":{"state":"asserted","subject":"fixture-owner","effective_permission":"repository-transition-owner","method":"airgap-declaration","evidence_reference":"fixture-ratification","evidence_digest":"sha256:2222222222222222222222222222222222222222222222222222222222222222","absence_reason":"not_applicable"}}}' "$DEADLINE")
AIRGAP_RESPONSE=$(printf '%s' "$AIRGAP_REQUEST" | "$AIRGAP")
printf '%s\n' "$AIRGAP_RESPONSE" | grep '"provider_namespace":"airgap"' >/dev/null
printf '%s\n' "$AIRGAP_RESPONSE" | grep '"state":"asserted"' >/dev/null

set +e
INVALID_RESPONSE=$(printf '%s' '{"protocol":"symphony.knowledge.engine-process.v1","request_id":"bad","request_id":"duplicate"}' | "$ENGINE")
INVALID_STATUS=$?
set -e
test "$INVALID_STATUS" -eq 2
printf '%s\n' "$INVALID_RESPONSE" | grep '"code":"json.duplicate_key"' >/dev/null

DEADLINE=$(( $(date +%s) * 1000 + 60000 ))
APPLY=$(printf '{"protocol":"symphony.knowledge.engine-process.v1","request_id":"smoke-apply","correlation_id":"smoke-apply","operation":"apply","target_engine":"symphony-sclv","deadline_unix_ms":%s,"payload":{}}' "$DEADLINE")
set +e
APPLY_RESPONSE=$(printf '%s' "$APPLY" | "$ENGINE")
APPLY_STATUS=$?
set -e
test "$APPLY_STATUS" -eq 4
printf '%s\n' "$APPLY_RESPONSE" | grep '"code":"operation.unsupported"' >/dev/null

echo "SCLV engine process smoke tests passed"
