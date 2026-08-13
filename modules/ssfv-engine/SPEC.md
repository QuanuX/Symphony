# SSFV Engine Specification

## Status

Implemented read/comparison/proposal development slice, version `0.1.0-dev`. It is source-installable and not a published module release.

## Process Contract

The engine uses `symphony.knowledge.engine-process.v1`. Direct diagnostics are `--help`, `--version`, `--descriptor`, and `--descriptor-v2`. Descriptor v1 remains exactly backward compatible; descriptor v2 adds stable `engop:` identities and administration metadata without claiming installation state. Exit statuses are 0 completed, 2 malformed input, 3 process protocol or deadline mismatch, 4 invalid operation semantics, and 5 bounded repository/internal failure.

## Operations

- `inspect`: exact empty payload; reports compatibility, bounds, lifecycle, and disabled authority.
- `check`: exact SSFV v2 check input; validates namespace, registry, owner files, records, paths, hierarchy, relationships, SKVI coverage, and optional semantic freshness.
- `diff`: exact SSFV v2 diff input; compares one caller-supplied semantic snapshot with live canonical state.
- `propose`: exact SSFV v2 proposal input; creates one immutable namespace or feature proposal with a bounded multi-file write set.
- `graph`: exact `{"format":"json"}` payload; emits the complete bounded portable feature graph.
- `administration-check`: exact caller-supplied semantic snapshot, administration profile, expected and optional observed qxctl registries, and engine descriptors; emits read-only design/live/authorization surfaces, module-integration admission, and non-inventive remediation constraints.

Feature proposals validate namespace allocation, strict prospective hierarchy, route ownership, owner and implementation evidence, applicable cross-vector references, and typed absence for an unregistered target file before rendering a write set. Existing but unregistered target files are never treated as absent.

## Bounds

The common 1 MiB request, 4 MiB response, depth, string, path, and deadline bounds apply. Descriptor v1 preserves its exact historical 16,384-value declaration. Descriptor v2 advertises and the process enforces a finite 65,536 JSON value/event limit so one complete caller-supplied feature-administration envelope can carry the bounded semantic snapshot, profile, command registry, and descriptors without weakening another engine's shared default. One file is at most 4 MiB, all SSFV evidence reads total at most 64 MiB, and one operation accepts at most 4,096 namespaces, 1,024 feature files, 8,192 feature records, and 32,768 graph edges within the response ceiling.

## Freshness

Structural validation is mandatory. Freshness is per invocation:

- `disabled` performs no baseline comparison;
- `report` emits semantic candidates as warnings;
- `require` emits the same unresolved candidates as violations.

Freshness evidence means cited implementation content changed. It does not decide that a feature changed semantically.

## Non-Authorization

No operation writes a file, creates a feature record, authenticates or classifies a caller, grants permission, ratifies a proposal, persists a snapshot/graph, invokes Git or a provider, starts a listener, mutates a session, activates a version, or docks with Maestro.

Administration checking is repository-independent after its bounded request is supplied. The current registered profile is `enforce_new_records`; a new empty or unreviewed feature disposition therefore fails closed. Missing command bindings and backend-operation mappings remain uncovered gaps, while qxctl absence is a live state rather than a design failure. Installation may proceed independently, but semantic-registration or required-administration gaps make integration and docking readiness false. Remediation output describes required evidence and never generates canonical feature IDs, command IDs, grammar, or AI-specific authority.

Composed administration evaluates a union: each expected qxctl command must bind the exact feature/interaction, and each expected backend operation must be targeted by at least one valid expected command. It does not require every command to target every operation. This preserves multi-step begin/checkpoint/close and prepare/finalize/recover workflows while still exposing every missing command edge and backend edge independently.
