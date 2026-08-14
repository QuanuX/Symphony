# Symphony Common Invariant Ownership

## Authority

This contract assigns every cross-component invariant to the lowest layer that can authoritatively enforce it. A caller, client, validator, or AI assistant may detect a violation, but it MUST NOT duplicate, weaken, rename, or manufacture the owning rule. The canonical machine inventory is `knowledge/INVARIANT-OWNERSHIP.json`, governed by `knowledge/schemas/v1/invariant-ownership-registry.schema.json`.

## Stable Identity

An invariant ID is an actual stable alphanumeric protocol identifier of the form `invariant:symphony:<name>`. It is not a display label, generated recommendation, C++ symbol, qxctl command, feature ID, or engine-operation ID. A human owner assigns it in a reviewed contract change. Implementations compare the exact ID; they do not infer one from prose or ask an engine to name it.

The same separation applies to feature administration. A feature ID identifies ratified product semantics, a `qxcmd:` ID identifies one qxctl executable leaf, and an `engop:` ID identifies one backend operation. The C++ engine evaluates supplied identities and evidence. It does not recommend names or decide that an undeclared capability is a feature.

## Lowest-Authoritative-Layer Rule

Each invariant has exactly one `owner_contract` and one `owner_component`. The owner defines the rule and must reject invalid producer state before exporting it. Consumers independently validate the evidence at their trust boundary and fail closed, but consumer acceptance never becomes the source of truth. A higher layer may compose owner results; it may not recreate the mutation, recovery, digest, authority, or domain decision.

Every active registry record MUST link:

- the stable invariant ID and normative statement;
- the owning contract and component;
- every authoritative producer implementation;
- at least one owner-side producer regression;
- at least one consumer-side boundary rejection regression;
- the finite set of allowed versioned adapters, or an empty set when no adapter is permitted; and
- for an IPC invariant, at least one real-process regression that invokes a receipt-backed executable across its actual stdin/stdout boundary.

A source-only helper test does not satisfy the real-process requirement. An IPC test must be able to catch wrong executable selection, argument forwarding, environment assumptions, extra standard-output text, malformed framing, exit-status loss, and installation-evidence drift.

The v1 registry is an explicitly incremental inventory, not a claim that every legacy invariant has already been enumerated. Its forward gate is `enforce_new_or_modified`: any new cross-component invariant, or material change to a registered invariant, MUST add or update its record and evidence in the same reviewed change. Advancing to a complete-catalog claim requires a separate bounded audit of legacy contracts and implementations.

The deterministic C++ validator enforces the registry's closed shape, stable identifiers, recursive omit-self digest, ordering, single ownership, referenced paths and named regression definitions, evidence-role separation, finite adapter closure, and mandatory per-adapter real-process evidence for IPC records. It exits `26` when this rule family fails. Discovering a named test proves traceability, not execution; the applicable build and test campaign remains the execution evidence.

`qxctl knowledge invariant status|list|show` provides digest-bearing read-only projections under `symphony.knowledge.invariant-query-result.v1`. These consumer checks explicitly report `semantic_validity: not_asserted`. `qxctl knowledge invariant check` invokes one exact receipt-validated Symphony Validator and preserves its complete `symphony.validation.result.v1` evidence and exit status. Neither surface allocates identities or writes the registry.

## Module Admission and Uncovered Administration

A new independently developed module cannot opt out by forgetting qxctl. Its admission change MUST declare a stable module/owner-scope feature record and review each administrator-facing interaction under `knowledge/FEATURE-ADMINISTRATION.md`. Each interaction must bind stable `qxcmd:` and, where applicable, `engop:` identities, or carry an explicit evidence-backed `runtime_only`, `system_orchestrated`, `prohibited`, or `not_applicable` disposition. Omission is an uncovered surface, not an exemption.

The module admission check is engine-first and headless. The SSFV engine consumes the canonical feature registry, administration profile, qxctl command registry, and backend descriptors and reports unresolved or uncovered surfaces. qxctl may later present, scaffold, or recommend Cobra/Viper commands, but it is not required for detection and cannot silently close coverage. Until the module supplies a reviewed mapping or disposition, integration remains incomplete.

Repository validation closes the source-level independent-developer omission case by enumerating bounded direct children of `modules/` with actual build or source markers. Each implemented module requires its exact `FEATURES.md`, SSFV source-scope route, and feature-administration profile mapping in the same change. Contract-Quad-only proposal seeds remain explicit documentation-only exclusions. The SSFV engine stays repository-independent and evaluates only supplied bounded evidence; it does not infer feature-worthiness from ambient files.

Installed-host completeness is a separate future gate. A caller-supplied engine descriptor is not proof that every installed package was supplied or that it matches an immutable receipt. A future versioned inventory contract must bind a declared-complete package/receipt inventory to every executable descriptor and non-engine administration declaration before installed admission can claim completeness. Until then, installed-host assessment remains explicit and partial rather than silently complete.

## Foundational Lifecycle Invariants

The foundational SSIAG/STAV lane applies the registry to the following rule families:

- exact receipt-backed adapter identity and capability negotiation;
- bounded one-value JSON process framing;
- stable observation and transaction digests;
- plan and attempt compare-and-swap before mutation;
- evidence-driven forward recovery;
- audit commitment or explicit durable audit deferral before successful closure; and
- one module-owned transaction implementation shared by human and machine entry points.

`knowledge/FOUNDATIONAL-LIFECYCLE.md` remains the normative owner of those semantics. The registry makes their ownership and regression evidence mechanically discoverable; it does not create another lifecycle authority.

## SSIAG Provider Boundary Invariants

`invariant:symphony:ssiag.provider-executable-provenance` assigns mutual executable trust to the SSIAG provider boundary. The Go foundation must select and verify one exact receipt-v2 adapter entry point before launch, and the native adapter must independently verify the invoking SSIAG executable before returning metadata. Path presence, a custom manifest, a child claim, or a code signature alone is insufficient. Any ambiguity, replacement, identity drift, protocol mismatch, or trust failure reports unavailable without fallback.

`invariant:symphony:ssiag.provider-control-secret-noninterference` assigns channel separation to the same boundary. General metadata/control framing remains bounded and secret-free. The current trust-only slice creates no secret-bearing channel and keeps `operational_access_enabled` false. Any later export channel requires its own request binding, one-shot lifetime, replay refusal, byte and time bounds, closure proof, memory/crash policy, and real-process regressions before operational enablement; qxctl, JSON control, standard output/error, arguments, environment variables, STAV, OpenAPI, examples, and diagnostics remain prohibited secret paths.

Both invariants are IPC invariants and require an actual receipt-backed Go-to-native process test. The only installed adapter identity at this boundary is `adapter:symphony:ssiag.macos-keychain-provider.v1`, whose receipt entry point speaks `symphony.ssiag.provider.control.v1`. The Go launcher is an in-process verifier and receptor, not an independently installed adapter or a synthetic receipt entry point. Operational Keychain access, credential operations, and secret delivery remain disabled.

## Versioned Adapters

An allowed adapter record names an exact entry-point ID, protocol major, owner contract, implementation path, and version-selection policy. Version permission is never “latest.” The current foundational adapters are allowed only when an immutable receipt-v2 package proves the exact executable and entry point and capability compatibility accepts the command protocol. Adding a component, entry point, protocol major, or adapter authority requires a reviewed registry and owner-contract change.

Registry v1 distinguishes the two SSIAG/STAV foundational-lifecycle adapters from the separately reviewed macOS provider-control adapter. Other process and socket boundaries—including vector engines, Maestro, the coordinator, validator results, SSIAG authorization, and STAV local requests—must not be mislabeled as non-IPC or forced through those adapters. They require a separately reviewed generic-adapter protocol version plus receipt-backed real-process evidence.

## AI and Agent Use

AI is optional. To assist without inventing protocol, it needs the exact owner contract, invariant registry and digest, SSFV feature registry, feature-administration profile, expected qxctl registry, relevant backend descriptor, and the affected implementation/test paths. It may propose human-readable names or candidate stable IDs, explain uncovered evidence, and scaffold code for review. It MUST label suggestions as proposals, preserve existing IDs, emit no coverage claim without machine evidence, and never infer authority from being AI or from possessing a command.

## Digest and Ordering

`registry_digest` is tagged SHA-256 over compact UTF-8 recursively key-sorted JSON with `registry_digest` omitted. Invariants sort by `invariant_id`; adapters sort by `adapter_id`; path, case, operation, and adapter-reference arrays are unique and lexicographically sorted. Digest validity does not prove that a referenced test passed; validation must also resolve every path and execute the applicable producer, boundary, and real-process suites.

## Non-Authorization

This contract does not authorize canonical apply, invent feature semantics, add arbitrary adapter execution, require qxctl at runtime, or let a validator repair missing coverage. The engine itself remains the only guaranteed executable in an engine-first assessment; registries and contracts are supplied evidence, and optional clients or AI remain replaceable consumers.
