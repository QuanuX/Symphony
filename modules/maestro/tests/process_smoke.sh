#!/usr/bin/env sh
set -eu

binary="$1"

"$binary" --help | grep -q 'symphony-maestro'
"$binary" --version | grep -q 'symphony-maestro 0.1.0-dev'

deadline_unix_ms=$((($(date +%s) + 60) * 1000))
response=$(printf '%s\n' "{\"protocol\":\"symphony.knowledge.engine-process.v1\",\"request_id\":\"request-inspect\",\"correlation_id\":\"correlation-inspect\",\"operation\":\"inspect\",\"target_engine\":\"symphony-maestro\",\"deadline_unix_ms\":${deadline_unix_ms},\"payload\":{\"protocol\":\"symphony.maestro.knowledge-engine-docking.v1\",\"format_version\":1,\"operation\":\"inspect\",\"state_root\":null,\"tops_id\":\"018f0c3a-7b2d-7e11-8c12-0242ac120002\",\"receptor_id\":\"maestro-primary\",\"operation_id\":null,\"expected_registry_digest\":null,\"component\":null,\"authorization_decision\":null,\"client\":{\"client_id\":\"qxctl\",\"client_version\":\"qxctl-dev\",\"process_protocols\":[\"symphony.knowledge.engine-process.v1\"],\"presence_read_versions\":[1],\"presence_write_versions\":[1],\"capabilities\":[\"atomic-head-v1\",\"dual-slot-presence-v1\",\"exact-receipt-binding-v1\",\"expected-state-cas-v1\",\"idempotent-operation-v1\",\"recovery-forward-v1\",\"ssiag-capability-binding-v1\"]}}}" | "$binary")
printf '%s\n' "$response" | grep -q '"outcome":"ok"'
printf '%s\n' "$response" | grep -q '"receptor_kind":"symphony.maestro.knowledge-engine.v1"'
printf '%s\n' "$response" | grep -q '"execution_enabled":false'
