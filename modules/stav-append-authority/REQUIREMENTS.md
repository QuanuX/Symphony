# STAV Append Authority Requirements

## Build and Lifecycle

- **STAV-AA-N-001**: The module and executable names must remain canonical.
- **STAV-AA-N-002**: Symphony-authored foundation code must be Go and cgo-free.
- **STAV-AA-N-003**: The module must remain independently buildable and use tagged shared-module dependencies outside `go.work`.
- **STAV-AA-L-001**: Install must be atomic and identical-install idempotent; differing replacement/removal requires `--force`.
- **STAV-AA-L-002**: Host uninstall must preserve all per-TOPS configuration and state.
- **STAV-AA-L-003**: Enrollment must create zero implicit producer or reader grants.
- **STAV-AA-L-004**: Unenrollment preserves state unless one-TOPS purge is explicit and the listener is inactive.

## Identity and IPC

- **STAV-AA-I-001**: Every instance path and protocol scope uses one canonical immutable TOPS UUID.
- **STAV-AA-I-002**: The authority must verify its configured effective UID/GID before listening.
- **STAV-AA-I-003**: Server and client must mutually authenticate Unix peers from kernel credentials on Darwin/Linux.
- **STAV-AA-I-004**: Producer grants must authorize exact event-class/operation tuples; reader grants must authorize exact classifications.
- **STAV-AA-I-005**: Unknown, ambiguous, mismatched, or ungranted identities fail closed.
- **STAV-AA-I-006**: Frames and connection time are bounded before allocation or dispatch.

## Ledger

- **STAV-AA-D-001**: One non-blocking exclusive lock protects one ledger for the process lifetime.
- **STAV-AA-D-002**: Every startup verifies frame length, checksum, canonical event, TOPS, sequence, and predecessor chain.
- **STAV-AA-D-003**: A committed receipt is returned only after the complete frame is synchronized.
- **STAV-AA-D-004**: Request-ID idempotency is reconstructed from canonical events; conflicting reuse fails closed.
- **STAV-AA-D-005**: Only an incomplete final frame may be recovered automatically, with exact evidence preserved and synchronized first.
- **STAV-AA-D-006**: Complete corruption, middle-frame damage, or chain mismatch prevents readiness.
- **STAV-AA-D-007**: v1 retention is preserve-all, rotation is disabled, and a finite maximum ledger size is mandatory.

## Administration and Producers

- **STAV-AA-A-001**: qxctl exposes only status, verify, bounded query, and doctor composition.
- **STAV-AA-A-002**: Projection filtering and redaction authorization occur before output.
- **STAV-AA-A-003**: SSIAG uses a closed typed safe-metadata vocabulary and treats any non-committed receipt as failure.
- **STAV-AA-A-004**: No producer spool, secondary writer, raw append, general repair, HTTP, OpenAPI, or remote transport may exist in v1.
