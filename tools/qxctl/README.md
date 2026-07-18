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

# Exercise the ratified STAV read-only grammar gate
go run ./cmd/qxctl stav status --tops-id UUID [--scope user|system] [--json]
go run ./cmd/qxctl stav verify --tops-id UUID [--scope user|system] [--json]
go run ./cmd/qxctl stav query --tops-id UUID [--scope user|system] [--after-sequence N] [--through-sequence N] [--from-time UTC] [--through-time UTC] [--event-class ID]... [--outcome VALUE]... [--correlation-id UUID] [--request-id UUID] [--limit 1..1000] [--json]
go run ./cmd/qxctl stav doctor --tops-id UUID [--scope user|system]
```

SSIAG commands require an immutable TOPS UUID through `--tops-id` or `SYMPHONY_SSIAG_TOPS_ID`. They use `SYMPHONY_SSIAG_SOCKET` only as an explicit override; otherwise the selected scope and TOPS ID determine the isolated socket. They never accept or print credential values.

The ratified future administrative model separates non-mutating proposal from authorized local apply. The current qxctl implementation remains read-only. AI agents will be limited to query and proposal, and qxctl will never write STAV ledger files directly.

The four STAV command names currently return a deliberate runtime-contract gate error after validating the immutable TOPS identity and selected path scope. Query also validates the ratified bounded filter model through the shared protocol kernel. No command opens a socket. Local request/response envelopes, reader authentication/authorization, status semantics, and the relevant runtime contracts must be ratified before these commands become operational; `qxctl stav append` will not be introduced.
