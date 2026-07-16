# STAV Append Authority Requirements

## Namespace and Build

- **STAV-AA-N-001**: The module must live at `modules/stav-append-authority/`.
- **STAV-AA-N-002**: The executable must be named `symphony-stav-append-authority`.
- **STAV-AA-N-003**: Symphony-authored foundation source must be Go and must not use cgo.
- **STAV-AA-N-004**: The module must remain its own Go module and independently installable. Monorepo development may use the root workspace; a published source build must require a real tagged kernel version and must not rely on a relative `replace` directive.
- **STAV-AA-N-005**: Shared STAV protocol mechanics must come from the first-party pure-Go kernel and must not be reimplemented divergently.

## Lifecycle

- **STAV-AA-L-001**: User and system installation must target the ratified paths.
- **STAV-AA-L-002**: Installation must be atomic and idempotent for an identical executable.
- **STAV-AA-L-003**: Replacement or removal of a differing executable must require explicit `--force`.
- **STAV-AA-L-004**: Lifecycle operations must reject non-regular executable targets.
- **STAV-AA-L-005**: Uninstall must remove only the executable and must preserve all configuration, state, runtime artifacts, ledgers, and projections.
- **STAV-AA-L-006**: The lifecycle scaffold must not invent an installation-manifest schema.

## Identity and Paths

- **STAV-AA-P-001**: Instance resolution must accept only canonical lowercase TOPS UUIDs.
- **STAV-AA-P-002**: User and system configuration, state, runtime, and socket paths must match `knowledge/stav/SPEC.md`.
- **STAV-AA-P-003**: User runtime fallback must resolve to the per-TOPS state `stav/run/append.sock` path only when `XDG_RUNTIME_DIR` is absent.
- **STAV-AA-P-004**: Path resolution must not create per-TOPS files or directories.

## Closed Gates

- **STAV-AA-G-001**: No configuration, status, or local-envelope schema content may be inferred from a reserved identifier.
- **STAV-AA-G-002**: Ratified codec availability does not authorize a listener, producer enrollment, candidate ingestion, committed receipt, trusted-field assignment, or ledger mutation.
- **STAV-AA-G-003**: No `append` or implicit repair command may exist in the executable or qxctl.
- **STAV-AA-G-004**: HTTP and OpenAPI must not govern producer ingestion.
