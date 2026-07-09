# qxctl

Symphony's Go-based administrative spine.

This tool is seeded with `Go 1.26.5` as the deterministic scripted baseline. It is written to exclusively use the Go standard library, ensuring zero third-party dependencies are required for core repository validation.

**Future Posture:** `qxctl` will migrate to Go 1.27 once released and officially adopted as the system baseline. It does not currently require or use unreleased features.

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
```
