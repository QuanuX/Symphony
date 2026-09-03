# Symphony Node Resource Vector Specification

## Local Resource Boundary

SNRV records the material local hardware composition that belongs to the physical Node identity. A material local hardware change requires a new SNIV physical-resource identity rather than silent mutation of the old Node.

The selected provider-offering reference may be retained as evidence for the composition, but the offering declaration is not substituted for observed local resources. SCV owns the offering's catalog meaning; SNIV owns the particular Node's association with it.

Software, operating-system packages, and Habitat contents are not local hardware resources. They remain separately identified even when their compatibility depends on SNRV evidence.

## Remote Resource Boundary

A remotely attached or virtual resource, including a CLI-attached GPU or comparable external allocation, is represented as remote to the local Node. The architecture reserves an explicitly qualified notation analogous to a port and using `::` before the remote portion. The exact grammar, escaping, nesting, authority, and round-trip rules are not yet ratified and must not be inferred from this statement.

## Evidence

Provider declarations, firmware, local probes, user declarations, and SHV knowledge may be distinct evidence sources. SNRV must preserve their source and time rather than collapsing an observation into permanent hardware truth.

## Non-Authorization Statement

SNRV records resources. It does not attach them, reserve them, score them, recommend them, or assert that a Nest can use them.
