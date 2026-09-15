# Engine integration findings

## Status and ownership

SKV-maintained, impersonal engineering observations from the bounded SCV/SHV review at source `7918e00443794cc57755236f9ebb0ceaf074ec9d`. This companion routes reusable findings to their existing owners. It is not a fifth Contract Quad member, a new engine, a universal composition policy, or an assertion that proposed integrations already work. The review and reproducible inventory are retained in `../shv-scv-integration-review/` relative to the repository root.

A Contract Quad is `INTENT.md`, `MANIFEST.md`, `SPEC.md`, and `SKILL.md`. SKVI, SSFV, SCLV and SODV have complementary responsibilities; they are not the four files of a Quad. Existing `ARCHITECTURE.md` and `INVARIANTS.md` retain common integration and invariant authority.

## Reusable findings

| Finding | Observed mechanism and source | Reuse boundary |
|---|---|---|
| Exact mechanical interface declarations reduce drift | SCV `scv/OWNER-INTERFACE.md` generates native, Go and CMake projections with frozen historical admission. SHV profile outputs and version admission are explicit in `tools/qxctl/internal/knowledgeengine/shv_profile.go`. | A future SHV declaration may remove repeated metadata. Keep semantic C++ implementations and independent Go validation separate. Do not change earlier releases or defaults. |
| Evidence survives movement when identities are explicit | `scv/SOURCE-KNOWLEDGE.md`, `shv/SOURCES.md`, `shv/RELOCATION.md` separate stable identity, source revision, locator and retained bytes. | Reuse expected-state mechanics; relocating a source does not prove publisher continuity, activate an adapter or refresh a capture. |
| Persistence and semantics are separate layers | `scv/GRAPH-INDEX.md` retains an exact SCV validating owner and query-time replay. `shv/GRAPH-STORE.md` stores generic structural graphs; original SHV owners replay meaning. | Transaction and receipt mechanics are candidates for a shared private library. Stored schemas, digest domains, validation time and semantic owners are not interchangeable. No cross-vector database migration is implied. |
| Recovery has distinct checkpoints | `scv/INDEX-TRANSFER.md`, `shv/GRAPH-STORE.md` and `shv/PUBLICATION.md` retain intent, expected state, exact writer and actual outcome. | A generic test harness can exercise interruption boundaries. Each owner must retain its own authority, replay and commit checks. Copying data alone does not publish a head. |
| Mapping conformance is not source truth | `shv/PROFILES.md` separates declarations, source extraction diagnostics and local binding. `scv/PROVIDER-PACKS.md` and `scv/INTERPRETATION.md` retain source/mapping evidence and fixture obligations. | Share report conventions only through a versioned owner contract. A valid shape or successful fixture does not prove coverage, publisher authority or physical compatibility. |
| Composition needs explicit bridges | `scv/COMPOSITION.md` evaluates caller-selected requirements and finite recipes. `shv/DOSSIERS.md` preserves caller associations and source-qualified assertions. | A cloud offering's CPU name cannot establish actual silicon, tenancy, firmware or observed performance. Cross-vector relationships need exact subjects, versions, qualifiers, evidence and unresolved findings. |
| Retained reader identity is part of reproducibility | `shv/PROFILES.md` and the profile engine's compile-time assertion pin the embedded kernel reader to 0.3.0-dev. SCV consumers retain exact original owners. | “Independently installed” does not mean “no dependencies.” Embedded and invoked readers must both be disclosed; a newer installation cannot silently replace either. |
| Graph reachability is scoped evidence | `shv/PROFILES.md` reference analysis checks caller-supplied digest mentions and rooted paths. | Additional owner adapters may discover references, but must distinguish inspected and opaque objects. Neither absence in a supplied graph nor structural validity grants deletion authority. |
| Agent access requires complete routing | `FEATURE-ADMINISTRATION.md`, `skvi/SPEC.md`, and `tools/qxctl/COMMANDS.json` separate feature, command and engine operation identities. | Add a new command only for a new administered surface. Documentation, examples and generated metadata alone do not warrant commands or feature IDs. |

## Integration dispositions

- **Existing:** common C++ process/receipt foundation, exact qxctl selection, source-owned replay, structural graph adapters, distributed feature records, registered command leaves, and local SCLV evidence. Scope is each owner's implemented contract, not universal interoperability.
- **Corrected in this review:** missing individual SKVI routes for twelve existing Quad files in the four earliest SHV modules, and stale SHV overview statements about activation, PDF ingestion, persistence and publication.
- **Next implementation candidate:** one SHV owner-interface declaration with frozen current/historical metadata and generated native/Go/CMake projections. Begin with one bounded owner to prove parity before widening to all eight. Preserve independently authored semantic and adversarial checks.
- **Next design candidate:** a read-only, explicit relationship envelope for SCV offering evidence and SHV component evidence. It should preserve two owner results and a caller-authored relation without creating an equivalence oracle. Decide its semantic owner before allocating an operation, schema or command.
- **Later extraction candidate:** shared private storage/recovery mechanics, justified by concrete duplicate code and paired owner tests. Do not replace the two public storage protocols with a universal database abstraction merely because both use DuckDB.
- **Conditional:** SACV registration if an actual HTTP API is introduced; SODV publication governance when official release/publication is authorized; SAV composition where exact typed owner projections fit. qxctl, local IPC and a graph edge alone activate none of these.
- **Separate domain work:** SNV Node identity/resource observations, SOV deployment, SIV intelligence integration and future Composer synthesis. Source capability evidence does not perform those actions.

## Caller authority and coverage

Caller choices include providers, hardware classes, date windows, required/optional fields, tools, graph backends and architectural compositions. A supported operation may reject a malformed envelope or mismatched installation; it must not reinterpret missing evidence as provider impossibility. DuckDB is the selected default SQL technology, not a selected default graph database. Representative fixtures verify ingestion methods; they are not a distributed vendor catalogue. Units and qualifiers require explicit mapping rather than inferred conversion.

No new runtime behavior, automatic source acquisition, provider action, default graph engine, platform-wide hardware ontology, destructive retention, public release or personal research data is introduced by these findings.
