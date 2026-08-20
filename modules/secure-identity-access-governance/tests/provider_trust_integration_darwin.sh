#!/bin/sh
set -eu

provider_trust_receipt_backed_go_swift_integration() {
if [ "$(uname -s)" != "Darwin" ]; then
  echo "provider trust integration: skipped (Darwin required)"
  exit 0
fi

for tool in go swift jq shasum stat; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "provider trust integration: missing required tool: $tool" >&2
    exit 1
  fi
done

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
MODULE_DIR=$(dirname -- "$SCRIPT_DIR")
REPOSITORY=$(CDPATH= cd -- "$MODULE_DIR/../.." && pwd)
PROVIDER_DIR="$REPOSITORY/modules/ssiag-provider-macos-keychain"
QXCTL_DIR="$REPOSITORY/tools/qxctl"
WORK=$(mktemp -d "${TMPDIR:-/private/tmp}/symphony-ssiag-mutual-trust.XXXXXX")
WORK=$(CDPATH= cd -- "$WORK" && pwd -P)
RUNTIME=$(mktemp -d "/tmp/sxr.XXXXXX")
SERVER_PID=""

cleanup() {
  if [ -n "$SERVER_PID" ]; then
    kill "$SERVER_PID" 2>/dev/null || true
    wait "$SERVER_PID" 2>/dev/null || true
  fi
  rm -rf "$WORK"
  rm -rf "$RUNTIME"
}
trap cleanup EXIT HUP INT TERM

export XDG_CONFIG_HOME="$WORK/config"
export XDG_STATE_HOME="$WORK/state"
export XDG_RUNTIME_DIR="$RUNTIME"
PREFIX="$WORK/prefix"
TOPS_ID="018f0c3a-7b2d-7e11-8c12-0242ac120002"
CONFIG_DIR="$XDG_CONFIG_HOME/symphony/$TOPS_ID/ssiag"
TRUST_DIR="$CONFIG_DIR/provider-trust"
SOCKET="$XDG_RUNTIME_DIR/symphony/$TOPS_ID/ssiag/ssiag.sock"
UID_VALUE=$(id -u)
GID_VALUE=$(id -g)

mkdir -p "$XDG_CONFIG_HOME" "$XDG_STATE_HOME" "$XDG_RUNTIME_DIR" "$TRUST_DIR"
chmod 0700 "$XDG_CONFIG_HOME" "$XDG_STATE_HOME" "$XDG_RUNTIME_DIR" "$CONFIG_DIR" "$TRUST_DIR"

(cd "$MODULE_DIR" && go build -o "$WORK/symphony-ssiag-source" ./cmd/symphony-ssiag)
"$WORK/symphony-ssiag-source" package install --prefix "$PREFIX" --version dev >/dev/null
FOUNDATION="$PREFIX/libexec/symphony/secure-identity-access-governance/dev/symphony-ssiag"
FOUNDATION_RECEIPT="$PREFIX/share/symphony/receipts/secure-identity-access-governance/dev/install-receipt.json"

(cd "$PROVIDER_DIR" && swift build --scratch-path "$WORK/swift-build" >/dev/null)
SWIFT_BIN_DIR=$(cd "$PROVIDER_DIR" && swift build --scratch-path "$WORK/swift-build" --show-bin-path)
"$SWIFT_BIN_DIR/symphony-ssiag-provider-macos-keychain" install \
  --scope user --prefix "$PREFIX" --version 0.1.0-draft >/dev/null
ADAPTER="$PREFIX/libexec/symphony/ssiag-macos-keychain-provider/0.1.0-draft/symphony-ssiag-provider-macos-keychain"
ADAPTER_RECEIPT="$PREFIX/share/symphony/receipts/ssiag-macos-keychain-provider/0.1.0-draft/install-receipt.json"

FOUNDATION_INSTALLATION_DIGEST=$(jq -er '.receipt_digest' "$FOUNDATION_RECEIPT")
FOUNDATION_EXECUTABLE_DIGEST=$(jq -er '.files[0].digest' "$FOUNDATION_RECEIPT")
ADAPTER_INSTALLATION_DIGEST=$(jq -er '.receipt_digest' "$ADAPTER_RECEIPT")
ADAPTER_EXECUTABLE_DIGEST=$(jq -er '.files[0].digest' "$ADAPTER_RECEIPT")
ADAPTER_MODE=$(stat -f '%Lp' "$ADAPTER")

jq -n \
  --arg tops_id "$TOPS_ID" \
  --arg executable_path "$ADAPTER" \
  --arg installation_digest "$ADAPTER_INSTALLATION_DIGEST" \
  --arg executable_digest "$ADAPTER_EXECUTABLE_DIGEST" \
  --arg file_mode "0$ADAPTER_MODE" \
  --arg foundation_path "$FOUNDATION" \
  --arg foundation_installation_digest "$FOUNDATION_INSTALLATION_DIGEST" \
  --arg foundation_executable_digest "$FOUNDATION_EXECUTABLE_DIGEST" \
  --argjson uid "$UID_VALUE" \
  --argjson gid "$GID_VALUE" \
  '{
    protocol: "symphony.ssiag.provider-executable-trust.v1",
    tops_id: $tops_id,
    scope: "user",
    provider_name: "native",
    provider_kind: "macos-keychain",
    adapter_identifier: "adapter:symphony:ssiag.macos-keychain-provider.v1",
    adapter_version: "0.1.0-draft",
    provider_protocol: "symphony.ssiag.provider.v1",
    executable_path: $executable_path,
    installation_digest: $installation_digest,
    executable_digest: $executable_digest,
    owner_uid: $uid,
    owner_gid: $gid,
    file_mode: $file_mode,
    adapter_signing_identity: "not_applicable",
    foundation_executable_path: $foundation_path,
    foundation_installation_digest: $foundation_installation_digest,
    foundation_executable_digest: $foundation_executable_digest,
    foundation_owner_uid: $uid,
    foundation_owner_gid: $gid,
    foundation_signing_identity: "not_applicable",
    operational_access_enabled: false,
    provider_operations_enabled: false,
    secret_channel_enabled: false,
    declaration_digest: "pending"
  }' > "$WORK/trust.pending.json"
CANONICAL_DECLARATION=$(jq -cS 'del(.declaration_digest)' "$WORK/trust.pending.json")
DECLARATION_DIGEST="sha256:$(printf '%s' "$CANONICAL_DECLARATION" | shasum -a 256 | awk '{print $1}')"
jq --arg digest "$DECLARATION_DIGEST" '.declaration_digest = $digest' \
  "$WORK/trust.pending.json" > "$TRUST_DIR/native.json"
chmod 0600 "$TRUST_DIR/native.json"

jq -n \
  --arg tops_id "$TOPS_ID" \
  --arg socket "$SOCKET" \
  --argjson uid "$UID_VALUE" \
  --argjson gid "$GID_VALUE" \
  '{
    schema: "symphony.ssiag.config.v1",
    mode: "user",
    tops: {id: $tops_id, name: "Mutual Trust Integration"},
    listen: {network: "unix", address: $socket},
    authentication: {
      mechanism: "unix_peer_credentials",
      service: {id: "symphony.ssiag.service", kind: "symphony.identity.service", uid: $uid, gid: $gid},
      subjects: []
    },
    authorization: {default_effect: "deny", max_capability_seconds: 900, grants: []},
    providers: [{
      name: "native",
      kind: "macos-keychain",
      enabled: true,
      capabilities: ["capability-discovery", "metadata"],
      exportable: false,
      interactive: true
    }]
  }' > "$CONFIG_DIR/config.json"
chmod 0600 "$CONFIG_DIR/config.json"

"$FOUNDATION" serve --scope user --tops-id "$TOPS_ID" --config "$CONFIG_DIR/config.json" \
  >"$WORK/server.stdout" 2>"$WORK/server.stderr" &
SERVER_PID=$!
for _ in $(jot 100 1); do
  if [ -S "$SOCKET" ]; then
    break
  fi
  if ! kill -0 "$SERVER_PID" 2>/dev/null; then
    cat "$WORK/server.stderr" >&2
    exit 1
  fi
  sleep 0.05
done
if [ ! -S "$SOCKET" ]; then
  cat "$WORK/server.stderr" >&2
  echo "provider trust integration: SSIAG socket did not become ready" >&2
  exit 1
fi

(cd "$QXCTL_DIR" && go build -o "$WORK/qxctl" ./cmd/qxctl)
"$WORK/qxctl" ssiag provider verify native --scope user --tops-id "$TOPS_ID" \
  --authority-basis host_owner --json > "$WORK/result.json"

if ! jq -e '
  .protocol == "symphony.ssiag.provider-trust-result.v1" and
  .operation == "verify" and
  .trust_state == "verified" and
  .mutual_trust.foundation_verified_adapter == true and
  .mutual_trust.adapter_verified_foundation == true and
  .operational_access_enabled == false and
  .provider_operations_enabled == false and
  .secret_channel_enabled == false and
  .read_only == true and
  .canonical == false
' "$WORK/result.json" >/dev/null; then
  cat "$WORK/result.json" >&2
  cat "$WORK/server.stderr" >&2
  echo "provider trust integration: mutual verification failed" >&2
  exit 1
fi

"$WORK/qxctl" ssiag provider readiness native --scope user --tops-id "$TOPS_ID" \
  --authority-basis host_owner --json > "$WORK/readiness.json"

if ! jq -e '
  .protocol == "symphony.ssiag.provider-readiness-result.v1" and
  .operation == "engop:symphony:ssiag.provider.readiness.observe" and
  (.readiness_state == "not_ready" or .readiness_state == "readiness_proven_operations_disabled") and
  .observation.protocol == "symphony.ssiag.provider-readiness-observation.v1" and
  .observation.metadata_only == true and
  .observation.operational_eligibility.state == "disabled" and
  .observation.authorization_decision_made == false and
  .operational_access_enabled == false and
  .provider_operations_enabled == false and
  .secret_channel_enabled == false and
  .read_only == true and
  .canonical == false
' "$WORK/readiness.json" >/dev/null; then
  cat "$WORK/readiness.json" >&2
  cat "$WORK/server.stderr" >&2
  echo "provider readiness integration: observation boundary failed" >&2
  exit 1
fi

echo "provider trust integration: passed (actual installed Go foundation <-> actual installed Swift adapter)"
echo "provider readiness integration: passed (metadata-only signed-bundle/session observation; operations disabled)"
}

provider_trust_receipt_backed_go_swift_integration
