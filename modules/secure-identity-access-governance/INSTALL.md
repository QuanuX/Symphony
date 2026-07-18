# Symphony Secure Identity and Access Governance Installation

## Status and Requirements

The host installer, per-TOPS enrollment lifecycle, native supervision, and metadata-only API are implemented. Operational providers remain intentionally disabled.

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
symphony-ssiag enroll --scope user --tops-id "$TOPS_ID" --tops-name "Local TOPS"
```

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

Use numeric effective identities obtained from the operating system; do not derive them from a display name or accept a request-supplied subject. Subject IDs and UID/GID pairs must both be unique. Ambiguous mappings prevent SSIAG from starting. The current endpoints are read-only, so a mapping reserves identity but grants no credential, provider, policy, or apply capability.

## Install Native Supervision and Verify One Enrollment

```bash
symphony-ssiag supervisor install --scope user --tops-id "$TOPS_ID"
```

This writes and starts `io.github.quanux.symphony.ssiag.<tops-id>` through a per-user launchd agent on macOS or `symphony-ssiag@<tops-id>.service` through a systemd user unit on Linux. The unit owns liveness only and has no STAV dependency. Use `--no-start` when an owner-provided supervisor will consume the generated descriptor. Direct user-scope `serve` remains available only as a foreground development/diagnostic mode and emits a warning.

For system scope, provision the service account through the owner or package manager first, then enroll with its numeric UID/GID. Enrollment makes only the selected TOPS state/runtime children service-owned and `0700`; shared parents remain root-owned and traversable. It never creates an account or infers root:

```bash
sudo symphony-ssiag enroll --scope system --tops-id "$TOPS_ID" \
  --tops-name "System TOPS" --service-uid <uid> --service-gid <gid>
sudo symphony-ssiag supervisor install --scope system --tops-id "$TOPS_ID"
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
symphony-ssiag supervisor uninstall --scope user --tops-id "$TOPS_ID"
symphony-ssiag unenroll --scope user --tops-id "$TOPS_ID"
```

Supervisor uninstall stops the selected job and removes only its descriptor. It preserves configuration and state. `--no-stop` supports a separately controlled owner-provided manager.

Explicitly remove that TOPS configuration and SSIAG state:

```bash
symphony-ssiag unenroll --scope user --tops-id "$TOPS_ID" --purge
```

Purge refuses to replace or remove a non-socket object at the socket path. It never targets another TOPS.

## Uninstall the Host Binary

```bash
symphony-ssiag uninstall --scope user
```

Uninstall validates the binary digest, requires `--force` if it changed, and always preserves every TOPS enrollment. Unenroll/purge instances separately before or after binary uninstall when that is the owner's intent.

## Supervision Security Contract

User enrollment records the effective UID/GID of the enrolling service process. A new system enrollment requires explicit `--service-uid` and `--service-gid`; it never silently selects root. User trust configuration is `0600`. System trust configuration is administrator-owned `0644`, contains no secrets, and is readable without becoming service-writable. The server verifies its effective identity before changing runtime state, and qxctl/self-client verify the exact connected endpoint before sending HTTP bytes.

The process owns socket creation. It acquires `ssiag.sock.lock` before stale inspection, refuses live/foreign endpoints, drains on SIGTERM, removes `ssiag.sock`, and releases the persistent lock last. launchd retries failed exits no faster than ten seconds. systemd retries after five seconds and stops after five starts in one minute. Both allow ten seconds for graceful shutdown. Neither supervisor grants SSIAG or STAV authority.
