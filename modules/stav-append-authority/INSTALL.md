# STAV Append Authority Installation

## Install the Current Public Source Release

Install the current supervised executable through the public Go module and checksum database:

```text
GOWORK=off go install github.com/QuanuX/Symphony/modules/stav-append-authority/cmd/symphony-stav-append-authority@v0.2.0
symphony-stav-append-authority --version
symphony-stav-append-authority install --scope user
```

The exact `go install` path was verified from an empty module cache through `proxy.golang.org` and `sum.golang.org`. It builds from source into `GOBIN`; Symphony does not currently publish a GitHub binary release. Use the checkout workflow below for development or reviewed local modifications.

## Build and Install

```text
CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 go build -o ./symphony-stav-append-authority ./cmd/symphony-stav-append-authority
./symphony-stav-append-authority install --scope user
```

Use `sudo ... install --scope system` for `/usr/local/bin`. The module uses the first-party STAV protocol kernel and `golang.org/x/sys` for cgo-free kernel peer credentials.

## Enroll One TOPS

```text
symphony-stav-append-authority enroll --scope user --tops-id <UUID>
```

User enrollment records the current effective UID/GID as the expected authority identity and rejects an override. System enrollment requires both explicit `--authority-uid` and `--authority-gid`; repeated enrollment must match the preserved identity. Every enrollment creates explicit empty `producers` and `readers` arrays. Review `append-authority.json`, then add only the exact producer tuples and reader classifications required. A production authority and producer should use distinct OS identities.

User configuration is written as `0600`. System configuration is non-secret administrator-authored trust metadata written as `0644`, allowing distinct service identities to read it without modifying it. The loader rejects a symbolic-link final component and any configuration writable by group or other.

The fixed policies are `fsync-before-receipt`, `preserve-incomplete-tail`, `preserve_all`, and disabled rotation. The default maximum ledger size is 1 GiB and may be changed to another schema-valid finite value before launch.

## Install Native Supervision

```text
symphony-stav-append-authority supervisor install --scope user --tops-id <UUID>
```

This installs and starts `io.github.quanux.symphony.stav.<tops-id>` as a user launchd agent on macOS or `symphony-stav@<tops-id>.service` as a systemd user unit on Linux. The STAV unit has no SSIAG dependency. `--no-start` writes the deterministic descriptor for an owner-provided supervisor without invoking the native manager. Direct user-scope `serve` is a development diagnostic and emits a warning.

For system scope, the owner or package manager provisions the authority account first. Enrollment receives its exact numeric UID/GID and makes only the selected TOPS state, recovery, and runtime children authority-owned and `0700`; it never creates an account or infers root. Then install the system service:

```text
sudo symphony-stav-append-authority enroll --scope system --tops-id <UUID> --authority-uid <uid> --authority-gid <gid>
sudo symphony-stav-append-authority supervisor install --scope system --tops-id <UUID>
```

The process owns `append.sock`, acquires the persistent adjacent `append.sock.lock` before stale inspection, drains accepted bounded requests on SIGTERM, removes its socket, and releases the lock last. Native restart cadence is bounded. System direct-run requires the explicit `--supervised` assertion from the installed profile or an owner-controlled equivalent; it is not authorization evidence.

After configuring a reader grant for the qxctl OS identity:

```text
qxctl stav doctor --scope user --tops-id <UUID>
qxctl stav status --scope user --tops-id <UUID>
qxctl stav verify --scope user --tops-id <UUID>
qxctl stav query --scope user --tops-id <UUID> --limit 100
```

## Remove

Host uninstall preserves all TOPS data:

```text
symphony-stav-append-authority uninstall --scope user
```

Remove a selected supervisor first; this stops only that TOPS service and preserves every configuration, ledger, and recovery artifact:

```text
symphony-stav-append-authority supervisor uninstall --scope user --tops-id <UUID>
```

Use `--no-stop` only when an owner-provided manager already owns process shutdown.

Default per-TOPS unenrollment also preserves data. Destructive one-TOPS removal requires explicit purge and refuses an active listener:

```text
symphony-stav-append-authority unenroll --scope user --tops-id <UUID>
symphony-stav-append-authority unenroll --scope user --tops-id <UUID> --purge
```
