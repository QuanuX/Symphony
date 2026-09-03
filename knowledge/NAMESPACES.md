# Symphony Identity-Family and Namespace Delegation

## Canonical Target

`knowledge/NAMESPACES.md`

## Status and Authority

This is the canonical SKV companion for platform-wide identity-family registration and delegation doctrine. It is governed by the `knowledge/` Contract Quad but is not a fifth Contract Quad member, a vector, an identity service, or a universal grammar.

The SKV umbrella records which contract owns an identity family. The delegated owner retains its grammar, allocation evidence, lifecycle, compatibility, and individual identities. Root registration does not transfer those semantics to SKV.

## Purpose

Prevent collisions, accidental authority, and cross-domain identity confusion while allowing independently governed vectors, modules, third-party extensions, and external providers to retain their proper naming realities.

## Identity Layers

Symphony distinguishes:

1. **identity family** — the leading form that selects an owning contract, such as `ssfv:` or `savref:`;
2. **namespace allocation** — an owner-ratified subdivision inside a family, such as `symphony` in `ssfv:symphony:...`;
3. **individual identity** — one stable identifier assigned under the family owner's rules;
4. **name or alias** — a domain-owned identifier or reference only when the applicable contract says so; and
5. **display text** — presentation that is not identity unless an owner contract explicitly declares otherwise.

These layers are not interchangeable. The same lexical key in two families does not make the identities equal. A repository path, organization, package, hostname, account, caller classification, or user-visible label does not allocate any layer by implication.

## Universal Delegation Rules

- Every Symphony-defined identity family has exactly one owner contract.
- Every allocated namespace has an explicit owner, bounded scope, status, and ratification evidence under that family contract.
- A family owner may delegate allocations without surrendering its grammar or compatibility rules.
- An incompatible semantic meaning requires the treatment prescribed by the owner contract; a new spelling alone does not silently create, replace, or merge identity.
- Registered family prefixes are never reused. Stable identities and namespace allocations remain recorded and follow their owner's non-reuse rules; domain-scoped names may be reused only when the owner explicitly permits unambiguous scope and temporal lineage.
- Aliases do not become canonical identities unless the owning domain explicitly declares that relationship.
- Cross-family references carry their complete identity. Consumers do not strip a family or namespace and compare only the remaining key.
- New families and allocations require collision review and SKVI routing in the same reviewed change.
- The root register never gives an implementation, CLI, database, agent, or external provider authority to assign identities owned elsewhere.

## Registered Symphony Identity Families

| Family | Delegated owner contract | Allocation surface or evidence | Bounded role |
|---|---|---|---|
| `invariant:` | `knowledge/INVARIANTS.md` | `knowledge/INVARIANT-OWNERSHIP.json` | Cross-component invariant identity and lowest-authoritative ownership. |
| `adapter:` | `knowledge/INVARIANTS.md` | `knowledge/INVARIANT-OWNERSHIP.json` plus the allowed adapter's owner contract | Versioned adapter identity admitted for a registered invariant boundary. |
| `ssfv:` | `knowledge/ssfv/SPEC.md` | `knowledge/ssfv/NAMESPACES.md` and `knowledge/ssfv/REGISTRY.md` | Semantic feature identity. |
| `qxcmd:` | `knowledge/FEATURE-ADMINISTRATION.md` | Reviewed qxctl `CommandSpec` with exact expected-registry evidence | One executable qxctl leaf identity. |
| `engop:` | `knowledge/FEATURE-ADMINISTRATION.md` | The applicable backend owner contract and engine descriptor | One backend operation identity, distinct from command and feature identity. |
| `savref:` | `knowledge/sav/SPEC.md` | SAV-owned canonical declarations | Accord Reference identity. |
| `savrel:` | `knowledge/sav/SPEC.md` | SAV-owned canonical declarations | SAV relationship identity. |
| `savtrait:` | `knowledge/sav/SPEC.md` | `knowledge/sav/TRAITS.md` and SAV-owned declarations | SAV trait identity. |
| `savver:` | `knowledge/sav/SPEC.md` | `knowledge/sav/NAMED-VERSIONS.md` and governed registries | SAV Named Version identity. |
| `savcapsule:` | `knowledge/sav/MANIFEST.md` | SAV Extension Capsule contract and schema | Portable Extension Capsule identity. |
| `savblueprint:` | `knowledge/sav/MANIFEST.md` | SAV Installation Blueprint contract and schema | Noncanonical Installation Blueprint identity. |
| `sevcase:` | `knowledge/sev/SPEC.md` | SEV case contracts and schemas | Evolution case identity. |
| `sevdisp:` | `knowledge/sev/SPEC.md` | `knowledge/sev/DISPOSITIONS.md` and SEV schemas | Evolution disposition identity. |
| `sevnovelty:` | `knowledge/sev/SPEC.md` | `knowledge/sev/NOVELTY.md` and its SEV schemas | Optional Novelty Bundle identity. |
| `receptor:` | `modules/maestro/SPEC.md` | Maestro receptor contracts and exact owner declarations | Maestro receptor identity. |

This table registers family ownership; it does not enumerate every identity. A family not shown here is not allocated by analogy. Its owner and grammar must be ratified before Symphony treats it as a registered platform family.

## Protocol and Domain-Local Identities

Not every exact identifier is a namespace family. Versioned `symphony.*` protocol identifiers are owned by their declaring contract and schema. UUIDs, tagged content digests, operation fingerprints, causal sequence values, provider-native identifiers, registry-local keys, and domain-specific record IDs retain the meaning assigned by their exact protocol or owner contract.

The following established domains therefore remain delegated without being normalized into the colon-family table:

- SACV owns its registry-local API identities and registered API-contract paths;
- historical SCLV record identities retain the meaning established by their immutable records; this register creates no active SCLV authority;
- SODV owns release-record identities and publication evidence;
- SSIAG owns subject, policy, capability, provider, installation, and binding identities;
- STAV owns its event, producer, ledger, and receipt identities;
- Maestro owns topology UUIDs, receptor streams, and recorded component relationships;
- SNIV, SNRV, SCIV, and SCNV own their respective Node, resource, cluster, and SNV-bounded name identities once their currently deferred encodings and lifecycles are ratified; and
- each process/schema owner controls its request, correlation, generation, predecessor, and digest fields.

A shared field label such as `module_id` does not create a shared identity space. Consumers qualify it by its protocol and owner contract.

The existence of the SCV, SOV, SNV, SHV, SQV, SIV, SOOV, SMCV, SAIV, SNIV, SNRV, SCIV, or SCNV domain name does not allocate a matching colon-prefixed family. Any future machine identity for those domains requires its own reviewed owner grammar and an entry in this register.

## Reserved Non-Reusable Module Tombstones

The following exact first-party module identities are retired and remain reserved as non-reusable tombstones:

| Module identity | Status | Reservation effect |
|---|---|---|
| `node-troll` | retired | The exact module identity and its historical association remain recognizable but cannot be assigned to a new module or meaning. |
| `bus-troll` | retired | The exact module identity and its historical association remain recognizable but cannot be assigned to a new module or meaning. |

These tombstones do not rewrite existing historical or proposal records, allocate or retire the general word `troll`, prohibit a differently identified optional resident, or claim that either module is implemented. A path, package, feature, command, or future component cannot revive the retired identity by reusing its spelling.

## SSFV Subordinate Namespace Registry

`knowledge/ssfv/NAMESPACES.md` is the delegated namespace registry for the `ssfv:` family. It remains canonical SSFV allocation truth and retains its existing `ssfv:symphony:` allocation. This universal surface references that allocation; it does not copy the entry, widen its scope, or authorize SSFV identities.

Other family owners may use a separate namespace registry only when their Contract Quad authorizes one. Directory symmetry is not authority to create it.

## External and User-Assigned Names

External providers, hardware, operating systems, networks, brokers, users, and other sovereign sources may assign names and identifiers under their own rules. Symphony records or relates those values only through a domain contract that preserves the source, scope, and evidence needed to interpret them.

An external identifier is not made more canonical by translating it into a Symphony-looking prefix. Conversely, this contract makes no judgment about which provider or user naming scheme is preferable. Namespace registration exists to preserve exact meaning, not to steer user choice.

## Registration and Evolution Gate

A proposed family or allocation is incomplete until the reviewed change supplies:

1. exact owner contract and bounded purpose;
2. grammar or explicit declaration that the identity is protocol-local and opaque;
3. collision analysis against registered families and allocations;
4. lifecycle and non-reuse treatment;
5. compatibility behavior for existing readers and stored identities;
6. SKVI routing and applicable validation evidence; and
7. updates to `knowledge/SLANG.md` only when new platform terminology needs an owner-routed entry.

Rename, supersession, deprecation, retirement, and removal follow the identity owner's contract and the SEV knowledge-surface evolution profile. Neither this register nor a lexical similarity test decides those outcomes.

## Non-Authorization Statement

This companion allocates no individual identity, external name, account, permission, package coordinate, network address, or runtime principal. It creates no global name service, parser, database, lookup API, qxctl command, engine operation, compatibility judgment, or hot/warm dependency. It does not make independent identity families interchangeable.
