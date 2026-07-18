# Symphony Secure Identity and Access Governance Installation

## Status and Requirements

The host installer, per-TOPS enrollment lifecycle, and metadata-only API are implemented. Service supervision and operational providers are intentionally not installed or enabled.

Requirements: a supported TOPS operating system and Go 1.26.5 for source builds. Python, cgo, containers, Kubernetes, NATS, and cloud infrastructure are not required.

## Build and Test

```bash
cd modules/secure-identity-access-governance
go test ./...
go vet ./...
CGO_ENABLED=0 go build -trimpath -o symphony-ssiag ./cmd/symphony-ssiag
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
socket: ${XDG_RUNTIME_DIR}/symphony/<tops_id>/ssiag.sock
        or <state>/run/symphony/<tops_id>/ssiag.sock when XDG_RUNTIME_DIR is absent
```

System instance paths:

```text
config: /etc/symphony/<tops_id>/ssiag/config.json
state:  /var/lib/symphony/<tops_id>/ssiag/
socket: /run/symphony/<tops_id>/ssiag.sock
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

## Run and Verify One Enrollment

```bash
symphony-ssiag serve --scope user --tops-id "$TOPS_ID"
```

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
symphony-ssiag unenroll --scope user --tops-id "$TOPS_ID"
```

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

## Supervision

User enrollment records the effective UID/GID of the enrolling service process. A new system enrollment requires explicit `--service-uid` and `--service-gid`; it never silently selects root. User trust configuration is `0600`. System trust configuration is administrator-owned `0644`, contains no secrets, and is readable without becoming service-writable. The server verifies its effective identity before changing runtime state, and qxctl/self-client verify the exact connected endpoint before sending HTTP bytes.

No launchd, systemd, node-troll, or other supervisor configuration is written by this increment. Exact labels, shared per-TOPS runtime-directory provisioning, restart policy, direct-run production behavior, and distinct-account integration tests remain installation gates; endpoint authentication itself is implemented.
