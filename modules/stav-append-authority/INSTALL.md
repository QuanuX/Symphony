# STAV Append Authority Installation

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

Enrollment records the current effective UID/GID as the expected authority identity and creates explicit empty `producers` and `readers` arrays. For a dedicated service account, pass `--authority-uid` and `--authority-gid`. Review `append-authority.json`, then add only the exact producer tuples and reader classifications required. A production authority and producer should use distinct OS identities.

User configuration is written as `0600`. System configuration is non-secret administrator-authored trust metadata written as `0644`, allowing distinct service identities to read it without modifying it. The loader rejects a symbolic-link final component and any configuration writable by group or other.

The fixed policies are `fsync-before-receipt`, `preserve-incomplete-tail`, `preserve_all`, and disabled rotation. The default maximum ledger size is 1 GiB and may be changed to another schema-valid finite value before launch.

## Run

```text
symphony-stav-append-authority serve --scope user --tops-id <UUID>
```

Direct run is suitable for development. Production liveness may be owned by a native supervisor, but supervision grants no producer, reader, policy, or ledger authority. This module does not install service-manager definitions.

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

Default per-TOPS unenrollment also preserves data. Destructive one-TOPS removal requires explicit purge and refuses an active listener:

```text
symphony-stav-append-authority unenroll --scope user --tops-id <UUID>
symphony-stav-append-authority unenroll --scope user --tops-id <UUID> --purge
```
