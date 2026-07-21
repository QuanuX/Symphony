# qxctl

Symphony's Go-based administrative spine.

This tool is seeded with `Go 1.26.5` as the deterministic scripted baseline. Its administrative command grammar uses Cobra and its explicitly bound command configuration uses private Viper instances. SSIAG/STAV trust loading remains outside Viper in dedicated cgo-free clients.

**Future Posture:** `qxctl` will migrate to Go 1.27 after general availability and the differential conformance/cross-build gate passes. It does not currently require or use unreleased features, and the migration cannot alter STAV wire bytes or CLI grammar.

## Usage

```bash
# Print help
go run ./cmd/qxctl --help

# Perform local repository checks
go run ./cmd/qxctl doctor

# Verify the first runtime-set module contract surfaces
go run ./cmd/qxctl contracts

# List canonical runtime modules
go run ./cmd/qxctl modules

# Verify contract shape for all modules
go run ./cmd/qxctl modules check

# Inspect a specific runtime module
go run ./cmd/qxctl module inspect <module-name>

# Verify contract shape for a specific module
go run ./cmd/qxctl module check <module-name>

# Extract contract metadata for all modules
go run ./cmd/qxctl modules metadata [--json]

# Extract contract metadata for a specific module
go run ./cmd/qxctl module metadata <module-name> [--json]

# Emit deterministic runtime inventory snapshot
go run ./cmd/qxctl inventory [--json]

# Emit deterministic runtime inventory SHA-256 digest
go run ./cmd/qxctl inventory digest [--json]

# Report consolidated administrative status
go run ./cmd/qxctl status [--json]

# Read safe local SSIAG status
go run ./cmd/qxctl ssiag status --tops-id UUID [--json] [--scope user|system]

# List safe provider metadata
go run ./cmd/qxctl ssiag providers --tops-id UUID [--json] [--scope user|system]

# Verify local SSIAG availability
go run ./cmd/qxctl ssiag doctor --tops-id UUID [--scope user|system]

# Query the authenticated local STAV append authority
go run ./cmd/qxctl stav status --tops-id UUID [--scope user|system] [--json]
go run ./cmd/qxctl stav verify --tops-id UUID [--scope user|system] [--json]
go run ./cmd/qxctl stav query --tops-id UUID [--scope user|system] [--after-sequence N] [--through-sequence N] [--from-time UTC] [--through-time UTC] [--event-class ID]... [--outcome VALUE]... [--correlation-id UUID] [--request-id UUID] [--limit 1..1000] [--json]
go run ./cmd/qxctl stav doctor --tops-id UUID [--scope user|system]

# Invoke an exact independently installed SKVI engine
go run ./cmd/qxctl skvi inspect --prefix /chosen/prefix [--version 0.1.0-dev] [--json]
go run ./cmd/qxctl skvi check --prefix /chosen/prefix [--expected-index-digest sha256:...] [--json]
go run ./cmd/qxctl skvi propose --prefix /chosen/prefix --input proposal-input.json [--json]
go run ./cmd/qxctl skvi project --prefix /chosen/prefix [--json]

# Invoke an exact independently installed SCLV engine
go run ./cmd/qxctl sclv inspect --prefix /chosen/prefix [--version 0.1.0-dev] [--json]
go run ./cmd/qxctl sclv check --prefix /chosen/prefix [--expected-ledger-digest sha256:...] [--json]
go run ./cmd/qxctl sclv propose --prefix /chosen/prefix --input proposal-input.json [--json]
go run ./cmd/qxctl sclv recover --prefix /chosen/prefix --input recovery-input.json [--json]
go run ./cmd/qxctl sclv project --prefix /chosen/prefix [--json]

# Invoke an exact independently installed SACV engine
go run ./cmd/qxctl sacv inspect --prefix /chosen/prefix [--version 0.1.0-dev] [--json]
go run ./cmd/qxctl sacv check --prefix /chosen/prefix [--expected-registry-digest sha256:...] [--json]
go run ./cmd/qxctl sacv diff --prefix /chosen/prefix --input diff-input.json [--json]
go run ./cmd/qxctl sacv propose --prefix /chosen/prefix --input proposal-input.json [--json]
go run ./cmd/qxctl sacv project --prefix /chosen/prefix [--json]
```

SSIAG commands require an immutable TOPS UUID through `--tops-id` or `SYMPHONY_SSIAG_TOPS_ID`. They use `SYMPHONY_SSIAG_SOCKET` only as an explicit override; otherwise the selected scope and TOPS ID determine the isolated socket. They never accept or print credential values.

The ratified future administrative model separates non-mutating proposal from permission-backed local apply. Authorization will use target-host ownership or granted permission and caller-neutral, owner-configured safeguards; qxctl will not request or evaluate caller type. The current qxctl implementation remains read-only for every caller, and qxctl will never write STAV ledger files directly.

Future safeguard administration will let a target-host administrator inspect and change optional governance interlocks through supported qxctl commands, including selecting a direct profile. That future surface does not make parser bounds, path safety, atomic writes, expected-state validation, ledger framing, or secret exclusion optional. No safeguard-management, apply, or audit-deferred recovery command is implemented today.

The four STAV commands use mutually authenticated, TOPS-scoped Unix-socket IPC to the local append authority. `status`, `verify`, and bounded `query` return only classification-authorized read projections; `doctor` composes client-side availability and verification checks. qxctl verifies the configured authority identity before sending application bytes and never opens the ledger file. `qxctl stav append` is intentionally absent.

The SKVI, SCLV, and SACV commands are cold/freezing-path local process operations. qxctl validates the exact inactive-undocked receipt and all package-owned files, invokes only the versioned engine path with an empty environment, enforces a hard deadline, and verifies response identity, digest, and safety assertions. Secure local receipt traversal is implemented on Linux and the macOS development path; other native operating systems fail closed rather than substituting a weaker file-open routine. Proposal, diff, and recovery input comes from one bounded no-follow JSON file. SKVI cannot decide membership. SCLV cannot grant permission, ratify, append, commit, mutate or delete journals, or treat a projection as canonical. SACV cannot decide semantic ownership, create endpoints, publish, generate bindings, or treat compatibility evidence or a projection as canonical. No command selects an active version, docks with Maestro, or writes canonical knowledge.
