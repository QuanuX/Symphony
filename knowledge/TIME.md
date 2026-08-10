# Symphony Temporal Semantics Contract

## Status and Authority

This Architect-ratified Symphony Temporal Semantics Contract (STSC) is canonical cross-vector truth owned by the `knowledge/` umbrella.

STSC is a compact common contract, not a knowledge vector. It creates no engine, executable, resident service, install receipt, qxctl command, state directory, network surface, clock-synchronization authority, or Maestro receptor. Vector and module contracts retain ownership of their domain-specific temporal fields and policies.

## Purpose

STSC gives Symphony one portable meaning for timestamps, dates, durations, elapsed time, ordering, and clock failure. It prevents local timezone, daylight-saving, precision, and clock-skew differences from becoming competing sources of truth across globally distributed TOPS deployments.

## Temporal Types

Symphony distinguishes these concepts explicitly:

| Type | Meaning | Canonical representation |
|---|---|---|
| instant | one point on the UTC timeline | a normalized UTC timestamp ending in `Z` |
| civil date | a Gregorian calendar date with no time or timezone | `YYYY-MM-DD` |
| zoned civil time | human scheduling or presentation in an IANA timezone | domain-owned local value plus explicit IANA zone and resolved UTC instant |
| duration | an amount of elapsed time | an integer with the unit declared by the owning field or schema |
| monotonic interval | elapsed process time immune to wall-clock correction | live process state only; never serialized as a portable instant |
| causal position | authoritative order or identity within a protocol | domain sequence, generation, operation identifier, predecessor/digest link, or other domain-owned causal evidence |

A bare number is not a portable duration. A civil date is not an instant. A formatted timestamp is evidence of wall-clock time, not proof of causality.

## Canonical UTC Profiles

Common administrative evidence uses whole-second UTC:

`YYYY-MM-DDTHH:MM:SSZ`

STAV event evidence uses its existing exact nanosecond UTC profile:

`YYYY-MM-DDTHH:MM:SS.NNNNNNNNNZ`

Both profiles:

- use the proleptic Gregorian calendar with years `0001` through `9999`;
- require a real calendar date, including Gregorian leap-year rules;
- require hours `00` through `23`, minutes `00` through `59`, and seconds `00` through `59`;
- reject timezone offsets, lowercase `t`/`z`, surrounding whitespace, missing fields, and variable fractional precision;
- reject leap-second text (`SS=60`) because Symphony does not define a portable leap-second encoding;
- preserve only precision actually supplied by the authoritative producer.

Implementations MUST NOT fabricate nanosecond certainty by padding a lower-precision observation and then representing it as higher-quality evidence. A domain requiring another precision profile must ratify it explicitly rather than silently broadening these formats.

## Durable Timestamp Authority

The target TOPS host that commits durable state owns the authoritative commit timestamp for that state. A remote qxctl client may supply a request timestamp for freshness validation, correlation, or diagnostics, but it cannot dictate the durable timestamp recorded by the target.

Every durable timestamp is stored in canonical UTC. Local time, locale, calendar formatting, and timezone abbreviations are presentation only and MUST NOT enter content digests, identifiers, permission decisions, compare-and-swap values, ledger order, or canonical machine records.

When a user requests local presentation, the renderer must identify the IANA timezone and numeric UTC offset used. Ambiguous or nonexistent local input during daylight-saving transitions must be rejected or resolved through a domain-owned explicit disambiguation policy; it must never be guessed silently.

## Ordering and Identity

Wall-clock time MUST NOT be the sole identity, uniqueness key, or causal-order mechanism for a Symphony operation. Protocols use their existing sequence, generation, stable operation identifier, expected-state digest, predecessor link, or hash-chain evidence.

Lexical timestamp order is valid only as a comparison between normalized values of the same ratified precision profile. Even then it expresses reported temporal order, not a cross-host causal proof. Symphony makes no claim of a globally total event order across independent TOPS nodes merely because their clocks are synchronized.

Collection time is document evidence. Where a contract defines a stable semantic inventory or plan identity, collection-only timestamps MUST be excluded from that stable identity exactly as the owning contract specifies.

## Deadlines, Freshness, and Elapsed Time

Live deadlines, timeouts, retries, and elapsed measurements SHOULD use a monotonic clock provided by the operating-system runtime. Persisted evidence records the applicable UTC wall-clock instant and any domain-owned duration; a serialized wall-clock timestamp must not be reinterpreted as a monotonic reading.

Freshness is evaluated by the target host against its current UTC clock and the exact domain policy. SSIAG's current request-skew and capability-lifetime limits remain SSIAG-owned policy. Clock uncertainty, remote-client claims, or retry pressure MUST NOT silently widen a freshness window or extend authority.

## Clock Failure and Regression

Failure to read or normalize the required clock is a typed failure. An implementation must not substitute local time, the remote caller's time, file modification time, a zero value, or a previously cached instant.

A detected backward wall-clock movement must be surfaced as typed diagnostic evidence when it would violate a domain invariant. Implementations preserve causal sequence and digest evidence, do not rewrite prior timestamps, and do not reorder permanent records to make clock output appear monotonic. A domain may pause a time-sensitive operation until its clock is trustworthy; it may not manufacture time.

Operating-system time synchronization, NTP/PTP selection, clock-quality attestation, and trading-node clock doctrine are outside STSC. Those capabilities require separate contracts if introduced.

## Implementation and Schema Contract

`knowledge/schemas/v1/temporal.schema.json` owns reusable structural definitions for the ratified seconds, nanoseconds, and civil-date encodings. Regex and JSON Schema `format` checks are not sufficient proof of a real Gregorian date; implementations must pass the common temporal conformance cases as well.

Symphony-authored C++ vector engines and the coordinator use the authority-free temporal validation in `libraries/knowledge-vector-engine-cpp/`. Go components use Go's standard time facilities, normalize durable values with `UTC()`, and must satisfy equivalent conformance cases. Independently built tools may keep an isolated implementation only when they preserve their architectural independence and prove the same accepted/rejected values.

Existing schemas remain immutable at their current version. New schemas should reference the common temporal definitions where practical. An existing schema adopts them only through its normal compatible revision or versioning process; STSC does not authorize a bulk historical rewrite.

### Legacy Read Compatibility

A versioned reader MAY accept a pre-STSC temporal encoding that its existing schema and implementation explicitly supported when that is necessary to preserve upgrade, rollback, or out-of-order installation compatibility. This is a bounded migration adapter, not a second canonical profile. The reader preserves the original evidence without silently rewriting it; every new write uses the current STSC profile. A later compare-and-swap mutation may normalize the field as part of the ordinary new generation while retaining predecessor and digest continuity.

The qxctl engine-binding registry v1 follows this rule: new generations emit whole-second UTC, while its reader continues to accept previously valid UTC fractional-second values so an upgrade cannot strand protected local state.

## Cross-Vector Relationships

- STAV retains ownership of append sequence, hash-chain semantics, and its exact nanosecond event profile.
- SSIAG retains ownership of authentication freshness, capability lifetime, expiry, and denial semantics.
- SCLV and SODV retain ownership of their append-only record order and completion evidence.
- SSFV retains ownership of semantic feature lifecycle fields and freshness meaning.
- `knowledge/LIFECYCLE.md` retains ownership of lifecycle generations, operation identities, observations, plans, journals, and recovery.
- SKVI indexes this contract and its reusable schema without converting STSC into a vector.

## Promotion Gate

STSC remains a common contract unless Symphony later needs an independently governed temporal dataset or operational capability such as clock-source attestation, clock-quality history, timezone-database release governance, cross-node temporal projections, or regulatory timestamp provenance. Such a requirement would need its own namespace review, Contract Quad, implementation gate, and Architect ratification. It must not be inferred from this document.
