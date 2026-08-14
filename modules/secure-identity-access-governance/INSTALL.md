# Symphony Secure Identity and Access Governance Installation

## Status and Requirements

The receipt-v2 and legacy host installers, foundational lifecycle adapter, per-TOPS enrollment, native supervision, audited authorization, protected policy administration, metadata-only provider trust, and exact provider installation/binding lifecycle are implemented. Binding changes remain metadata-only and cannot enable Keychain access, provider operations, or secret delivery.

Requirements: a supported TOPS operating system and Go 1.26.5 for source builds. Python, cgo, containers, Kubernetes, NATS, and cloud infrastructure are not required.

## Build and Test

```bash
cd modules/secure-identity-access-governance
go test ./...
go vet ./...
CGO_ENABLED=0 go build -trimpath -o symphony-ssiag ./cmd/symphony-ssiag
# On a disposable privileged host, exercise the real distinct-account startup gate:
sudo go test -tags=integration ./internal/server
```

## Install the Shared Host Binary

Current package managers install one immutable version and its receipt last:

```bash
./symphony-ssiag package install --prefix /absolute/prefix --version '<compiled-version>'
```

The requested version must exactly match the binary's compiled version. The entry point is `<prefix>/libexec/symphony/secure-identity-access-governance/<version>/symphony-ssiag`; its strict `symphony.knowledge.install-receipt.v2` is under `<prefix>/share/symphony/receipts/secure-identity-access-governance/<version>/install-receipt.json`. The receipt declares adapter ID `ssiag.foundation-lifecycle`, protocol `symphony.foundation.lifecycle-command.v1`, and capability `symphony.foundation.lifecycle-adapter.v1`.

The historical fixed-path install remains readable for migration:

```bash
./symphony-ssiag install --scope user
# or, under an owner-approved privilege boundary:
./symphony-ssiag install --scope system
```

User binary and manifest:

```text
$HOME/.local/bin/symphony-ssiag
${XDG_STATE_HOME:-$HOME/.local/state}/symphony/ssiag/install.json
```

System binary and manifest:

```text
/usr/local/bin/symphony-ssiag
/var/lib/symphony/ssiag/install.json
```

## Enroll Each TOPS

Choose or obtain the immutable canonical TOPS UUID from topology governance; do not derive it from a name. Example:

```bash
TOPS_ID=018f0c3a-7b2d-7e11-8c12-0242ac120002
symphony-ssiag enroll --audit-deferred --scope user --tops-id "$TOPS_ID" --tops-name "Local TOPS"
```

Foundational enrollment and supervisor mutations use the same observe/plan/apply/recover engine as the machine adapter. Until a real lifecycle STAV receipt endpoint exists, ordinary mode fails before any attempt or external mutation. A target-host administrator may explicitly select the bootstrap contract with `--audit-deferred`; its result remains `reconciliation_required=true` and is never represented as committed audit:

User instance paths:

```text
config: ${XDG_CONFIG_HOME:-$HOME/.config}/symphony/<tops_id>/ssiag/config.json
state:  ${XDG_STATE_HOME:-$HOME/.local/state}/symphony/<tops_id>/ssiag/
socket: ${XDG_RUNTIME_DIR}/symphony/<tops_id>/ssiag/ssiag.sock
        or <state>/run/ssiag.sock when XDG_RUNTIME_DIR is absent
```

System instance paths:

```text
config: /etc/symphony/<tops_id>/ssiag/config.json
state:  /var/lib/symphony/<tops_id>/ssiag/
Linux socket: /run/symphony/<tops_id>/ssiag/ssiag.sock
macOS socket: /var/run/symphony/<tops_id>/ssiag/ssiag.sock
```

Repeat enrollment with the same ID and a different `--tops-name` to update display metadata without moving state.

New enrollment configuration contains an explicit peer-authentication block:

```json
"authentication": {
  "mechanism": "unix_peer_credentials",
  "subjects": []
}
```

Every accepted Darwin/Linux connection is kernel-authenticated even when this array is empty. To reserve a canonical subject for a later subject-gated operation, add an exact operating-system UID/GID mapping to that TOPS configuration:

```json
"authentication": {
  "mechanism": "unix_peer_credentials",
  "subjects": [
    {
      "id": "operator.primary",
      "kind": "operator",
      "uid": 501,
      "gid": 20
    }
  ]
}
```

Use numeric effective identities obtained from the operating system; do not derive them from a display name or accept a request-supplied subject. Subject IDs and UID/GID pairs must both be unique. Ambiguous mappings prevent SSIAG from starting. A mapping establishes identity only. Authorization additionally requires an explicit exact grant, and even an allow decision grants no credential, provider, policy-mutation, safeguard, or canonical-apply capability.

## Install Native Supervision and Verify One Enrollment

```bash
symphony-ssiag supervisor install --audit-deferred --scope user --tops-id "$TOPS_ID"
```

This writes and starts `io.github.quanux.symphony.ssiag.<tops-id>` through a per-user launchd agent on macOS or `symphony-ssiag@<tops-id>.service` through a systemd user unit on Linux. The unit owns liveness only and has no STAV dependency. Use `--no-start` when an owner-provided supervisor will consume the generated descriptor. Direct user-scope `serve` remains available only as a foreground development/diagnostic mode and emits a warning.

For system scope, provision the service account through the owner or package manager first, then enroll with its numeric UID/GID. Enrollment makes only the selected TOPS state/runtime children service-owned and `0700`; shared parents remain root-owned and traversable. It never creates an account or infers root:

```bash
sudo symphony-ssiag enroll --audit-deferred --scope system --tops-id "$TOPS_ID" \
  --tops-name "System TOPS" --service-uid <uid> --service-gid <gid>
sudo symphony-ssiag supervisor install --audit-deferred --scope system --tops-id "$TOPS_ID"
```

System-scope `serve` accepts `--supervised` only as an explicit assertion from the installed native profile or an owner-controlled equivalent. It is not authorization evidence.

In another terminal:

```bash
qxctl ssiag doctor --scope user --tops-id "$TOPS_ID"
qxctl ssiag status --json --scope user --tops-id "$TOPS_ID"
qxctl ssiag providers --json --scope user --tops-id "$TOPS_ID"
```

`SYMPHONY_SSIAG_TOPS_ID` may supply the ID. `SYMPHONY_SSIAG_CONFIG` and `SYMPHONY_SSIAG_SOCKET` are explicit test/deployment overrides and must not carry secret values.

## Unenroll and Purge One TOPS

Default unenrollment removes only its enrollment marker and preserves recovery data:

```bash
symphony-ssiag supervisor uninstall --audit-deferred --scope user --tops-id "$TOPS_ID"
symphony-ssiag unenroll --audit-deferred --scope user --tops-id "$TOPS_ID"
```

Supervisor uninstall stops the selected job and removes only its descriptor. It preserves configuration and state. Transactional administration recognizes but refuses `--force` and `--no-stop`; they cannot bypass verified shutdown. `--no-start` produces the explicit `native_installed_stopped` state.

Explicitly remove that TOPS configuration and SSIAG state:

```bash
symphony-ssiag unenroll --scope user --tops-id "$TOPS_ID" --purge
```

Purge is intentionally native-only in lifecycle v1. It refuses a present supervisor descriptor, a live socket, a held adjacent socket lifecycle lock, or a foreign non-socket object. Protected lifecycle attempts live separately under `${XDG_STATE_HOME:-$HOME/.local/state}/symphony/ssiag/lifecycle/` (or `/var/lib/symphony/ssiag/lifecycle/`) and are not deleted by per-TOPS purge.

## Uninstall the Host Binary

```bash
symphony-ssiag uninstall --scope user
```

Legacy uninstall validates the binary digest and refuses while any TOPS enrollment or descriptor still references it. Receipt-v2 uninstall is explicit and versioned:

```bash
symphony-ssiag package uninstall --prefix /absolute/prefix --version '<compiled-version>'
```

It validates the exact receipt-owned bytes and refuses a matching supervisor descriptor, live SSIAG endpoint, or unresolved digest-bound lifecycle attempt. Neither form purges TOPS data.

## Foundation Lifecycle Machine Adapter

`symphony-ssiag foundation-lifecycle describe --json` emits the adapter descriptor. Without arguments it reads exactly one bounded strict JSON command from stdin and emits exactly one bounded JSON result to stdout. Supported surfaces are `enrollment` and `supervisor`; operations are `observe`, `plan`, `apply`, `apply_status`, and `recover`. Observation is offline and never calls launchd/systemd. Each mutation uses exact state and attempt compare-and-swap, digest-linked protected attempts, STSC timestamps, deadline enforcement, replay, and exact recovery. The adapter does not listen on a network socket and does not expose purge.

## Supervision Security Contract

User enrollment records the effective UID/GID of the enrolling service process. A new system enrollment requires explicit `--service-uid` and `--service-gid`; it never silently selects root. User trust configuration is `0600`. System trust configuration is administrator-owned `0644`, contains no secrets, and is readable without becoming service-writable. The server verifies its effective identity before changing runtime state, and qxctl/self-client verify the exact connected endpoint before sending HTTP bytes.

The process owns socket creation. It acquires `ssiag.sock.lock` before stale inspection, refuses live/foreign endpoints, drains on SIGTERM, removes `ssiag.sock`, and releases the persistent lock last. launchd retries failed exits no faster than ten seconds. systemd retries after five seconds and stops after five starts in one minute. Both allow ten seconds for graceful shutdown. Neither supervisor grants SSIAG or STAV authority.
