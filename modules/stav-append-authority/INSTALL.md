# STAV Append Authority Installation

## Build

From this module directory:

```text
go test ./...
go build -o ./symphony-stav-append-authority ./cmd/symphony-stav-append-authority
```

The module uses Go source, the standard library, and Symphony's first-party pure-Go STAV protocol kernel. It has no third-party dependency and does not use cgo. The monorepo `go.work` resolves the kernel during development; an independent release must require a tagged compatible kernel version.

## User Installation

```text
./symphony-stav-append-authority install --scope user
```

This atomically installs the executable at `~/.local/bin/symphony-stav-append-authority`. Repeating the command with the same binary is idempotent. A differing installed binary requires explicit `--force`.

## System Installation

```text
sudo ./symphony-stav-append-authority install --scope system
```

This targets `/usr/local/bin/symphony-stav-append-authority`. Filesystem privileges remain the operator's responsibility.

## Uninstall

```text
symphony-stav-append-authority uninstall --scope user
sudo symphony-stav-append-authority uninstall --scope system
```

Uninstall removes only the executable. It is idempotent when no executable exists and refuses to remove a differing binary unless `--force` is explicit. It never removes configuration, state, runtime directories, sockets, ledgers, or projections.

## Not Yet Available

Installation does not enroll a TOPS, write a configuration, create a socket, register a service, or start an append authority. Those operations remain unavailable until their canonical contracts are ratified.
