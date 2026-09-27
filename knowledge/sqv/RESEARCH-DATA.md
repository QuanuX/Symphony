# SQV Research-Data Responsibility and Requirement Alignment

## Status and Authority

This companion records the research-data purpose and owner boundaries under the thin SQV parent. It aligns Duncan's 26 September 2026 direction with the supplied *Symphony SQV Research Data Whitepaper v0.1* and its decision and acceptance register. The paper's `R`, `D`, `G`, and `T` identifiers are document-local trace labels, not Symphony protocol or command identities. Its mechanism proposals require admission under the appropriate owner before they become runtime contracts.

Duncan favored the six purpose labels. SQAV, SQMV, SQFV, SQTV, SQPV, and SQDV have Contract Quads and SKVI routes. SQFV owns the narrow `sqfv-batch-cpp` `0.2.0-dev` C++26 library for trusted same-process movement. SQMV and SQPV now own the `sqmv-metadata-cpp` and `sqpv-local-store-cpp` `0.1.0-dev` libraries for immutable bindings and local retention. SQAV, SQTV and SQDV remain architecture owners only. SOOV remains the separate SQV FIX child. This document establishes no bulk service, provider connection, provider-specific semantic schema, colon namespace, or qxctl operation.

The first-party research-data data plane is to be native C++. Duncan's subsequent explicit direction selects C++26 for proprietary SQV modules authored in this effort; each package must declare and verify its actual toolchain contract. This does not constrain user-authored modules or compatible integrations. Native connectors, movement, conversion, storage bridges, and delivery may be independently selected as their contracts become implemented. A composition may use direct compatible paths; it need not traverse six services, a central broker, or one normalized representation. qxctl administration follows Duncan's later instructions and is not a routine payload path. Research flows, even when live and high throughput, impose no compulsory data path on a live trading Nest.

## Purpose Owners

| Purpose label | Admission here | Owned meaning and handoff |
| --- | --- | --- |
| **SQAV — Symphony Quantitative Acquisition Vector** | Architecture owner | Selected source collection, source operation and version, authorized read scope, source position, reconnect and gap evidence. Broker research collection would consume a SCABV-owned exact non-FIX adapter if one is admitted; FRED/ALFRED, news, and dedicated market data need independent source contracts. |
| **SQMV — Symphony Quantitative Metadata Vector** | Architecture owner; first immutable manifest library | Dataset identity, schema and representation description, units, time roles, attributable provenance, revision, coverage, lineage, and source-supplied rights classification. `sqmv-metadata-cpp` resolves exact manifests and flow bindings; evidence references retain their producer attribution without proving domain meaning or access authority. |
| **SQFV — Symphony Quantitative Flow Vector** | Architecture owner; first same-process library | Bounded data ports, immutable-buffer handoff, partition transfer order, consumer-specific credits and cursors, lifetime and safe release, and selected locality adapters. `sqfv-batch-cpp` implements a narrow trusted-process subset. It neither selects sources nor asserts durability or destination processing. |
| **SQTV — Symphony Quantitative Transformation Vector** | Architecture owner | Exact selected conversion or research transformation, input/output interpretation, precision and information-loss declaration, state and workspace, and derived lineage. Re-encoding, narrowing, and deriving bars are different operations. |
| **SQPV — Symphony Quantitative Persistence Vector** | Architecture owner; first bounded local store | Selected retention/retrieval, segment and commit identity, integrity, durability guarantee, recovery and retained position. `sqpv-local-store-cpp` implements exact bound local retention and process-crash recovery; it does not own source collection or destination acceptance. |
| **SQDV — Symphony Quantitative Delivery Vector** | Architecture owner | Selected consumer view, access scope, format and completeness, destination-specific handoff, consumer resume and acknowledgement interpretation. SBV, SIV, web, and later external destinations retain their own acceptance and result authority. |

These are purpose boundaries, not a prescribed process topology. A provider adapter, queue, HTTP transport, file format, or database does not become a new vector merely because it is a separate implementation module. User-authored modules and alternate compositions remain possible under their actual compatibility and authority contracts.

## Cross-Owner Meaning

### Dataset and position

The source owner must preserve the source's actual dataset, provider version or observed API behavior, original event and revision information, acquisition binding, and supported source ordering. SQMV describes those attributable facts and distinguishes event time, publication time, revision or effective period, provider receipt time if supplied, Symphony acquisition time, and retention commit time. An absent time role stays absent; local receipt does not masquerade as source publication. Schema and physical layout versions, package versions, transport versions, and dataset revisions are independent dimensions.

| Position | Meaning | Architecture owner |
| --- | --- | --- |
| **Source position** | Provider-defined sequence, cursor, page, vintage, or other exact resumption evidence within its actual scope. It may restart or expire. | SQAV; source fact described by SQMV. |
| **Internal transfer cursor** | Exact dataset/view revision, partition, producer generation, and movement position. It is not automatically event time or a provider sequence. | SQFV architecture owner. |
| **Retained position** | Range or highest contiguous position proven under a named storage guarantee; gaps remain visible. | SQPV. |
| **Consumer resume position** | Recipient/view-bound position and destination-specific acknowledgement or replay obligation. | SQDV, with SQFV transfer state. |

No total order across unrelated providers is inferred. Live-memory and retained catch-up may share an exact sequence only after the selected storage and delivery owners establish cutover, gap, duplicate, revision, and generation behavior. A timestamp or raw file offset alone does not establish that guarantee. Historical corrections must remain attributable rather than silently replacing earlier research snapshots.

### Progress, obligation, and release

| Term | Meaning to declare at the exact interface |
| --- | --- |
| **Received** | The named actor accepted bytes or a frame into its stated boundary. A transport receipt does not prove parsing or consumer processing. |
| **Processed** | The named consumer completed its admitted interpretation or application step. It does not imply retention or remote commit. |
| **Durable** | Under an implemented SQPV contract, evidence would prove a specific artifact/range under a specified local or replicated durability guarantee. A buffered write or staged file is not this state. |
| **Destination committed** | Under the selected destination contract, evidence would prove the stated remote commit or apply boundary, where supported. Local upload acceptance is a separate event. |
| **Released** | No remaining admitted reader, transport, device, or retention obligation uses the physical allocation, so the owner may reclaim it. Release is a memory-lifetime fact, not a delivery or durability claim. |
| **Queryable** | The destination confirms availability under its own read/query contract. This may follow acceptance or commit and cannot be inferred from either alone. |

Local queue admission, transport receipt, consumer processing, durable retention, destination acceptance, destination commit, queryability, and physical release need distinct evidence where applicable. No universal exactly-once guarantee follows from any one acknowledgement. An obsolete producer or writer generation cannot reclaim current buffers or publish a newer retained head.

### Finite resources and locality

Preparation buffers hold mutable input, decoding workspace, and output under construction. In-flight allocations back immutable readers, transport work, or required retention obligations. Reusable cache entries are reconstructible or retrievable and can be evicted only when not borrowed. These three roles require separate accounting; an undelivered sole copy is not cache. Budgets also cover indexes, queue entries, decompression, transport buffers, and retained readers. Shared physical pages should not be charged as duplicate allocations, while each consumer's outstanding allowance remains finite and independent.

SQFV's architecture requires bounded frames, exact encoded and decoded limits, per-edge credits/cursors, cancellation and release progress when data capacity is full, and observable exhaustion. A slow optional branch cannot silently consume another branch's allowance. A required durable branch may intentionally gate committed delivery under the selected policy. Locality choices may include compatible in-process immutable views, reviewed same-Node shared memory, native cross-Node transfer, or retained retrieval. None is implicitly installed. Safe same-Node reuse needs exact region, generation, bounds, access scope, and release/fencing evidence; a timeout alone does not permit overwrite.

The whitepaper describes **ephemeral**, **immediate with asynchronous retention**, and **durable before committed delivery** profiles. These remain composition descriptions rather than registered universal enum or command values. The first SQPV library explicitly selects a bounded macOS local APFS path and its stated synchronization/recovery guarantee; callers select finite limits and admission stops when full. It supplies no asynchronous writer service, universal retention default or pruning policy. A preview may run ahead of committed retention only when its view says so; a consumer may still read from the original immutable RAM allocation after the selected commit boundary.

## Initial Source and Neighbor Boundaries

The intended source families are FRED/ALFRED economic series and vintages, a news provider still to be selected, Databento historical/live data within selected operations, and IBKR C++ market/account read surfaces through a future SCABV-owned broker adapter. This is target scope, not current provider support or permission to collect data. The relevant SCABV and SBV papers remain draft evidence; neither current module is inferred from them. Research collection has no order-entry authority. Rights to collect, retain, transform, model, share, and redistribute are separate source-specific questions.

SOOV retains FIX architecture. SBV owns backtest computation and result acceptance; SIV and presentation consumers cannot rewrite source or quantitative truth. SSIAG operational credential access is not currently available for these integrations. STAV safe audit records are not bulk payload storage. SODV official publication does not publish private research data by implication. SOV, SNV, SCV, and SHV continue to own their own lifecycle, Node, cloud, and hardware facts. These neighbor contracts are consumed only where implemented and selected.

## Requirement Trace

This table aligns the supplied register without wholesale ratification of its proposed mechanisms. `Admitted` means an architectural owner boundary is recorded here or in an admitted Quad; it does not mean an operational capability exists. `Pending` means an exact runtime contract, package, or operation still needs its own gate.

| Paper ID | Requirement alignment | Status |
| --- | --- | --- |
| R01 | Thin SQV parent; six distinct research-data purposes; SOOV separate. | Six architecture owners admitted; bounded SQMV, SQFV and SQPV libraries admitted. |
| R02–R04 | Native first-party data plane; independent native adapters; selected FRED/ALFRED, news, IBKR, Databento extents. | C++26 library packages admitted; provider-specific acquisition and native adapters remain pending. |
| R05 | Open-ended logical compositions with finite declared budgets. | Explicit local module limits admitted; production workload thresholds remain pending. |
| R06 | Exact source meaning, versions, time, revision, coverage, and uncertainty. | SQMV immutable manifest/binding library admitted; source-specific meaning and evidence resolution remain pending. |
| R07–R09 | Selectable branches; administrative qxctl; live trading path independence. | Native library composition admitted; provider branches, consumer integration and SQV qxctl remain pending. |
| R10 | Preserve neighboring owner truth and authority. | Boundary recorded; particular integrations pending. |
| R11–R13 | Separate memory roles, immutable batches, explicit lifetime, independent consumer credits/cursors. | SQFV's first library specifies copied immutable batches, lease lifetime, and bounded independent port credits/cursors; broader locality remains pending. |
| R14–R16 | Distinct progress and release meanings; optional durability profiles; traceable live/retained resume. | Local SQPV commit/recovery boundary admitted; broader profiles and SQDV cutover remain pending. |
| R17 | Separate lossless conversion, information loss, and derived research transformation. | SQTV owner admitted; converter operations pending. |
| R18 | Bulk/private payload isolation, access scope, rights classification. | SQMV attributable evidence references admitted; privacy and access enforcement remain pending. |
| R19 | Exact package, ABI, schema, operation, compatibility, lifecycle, and namespace before support claims. | SQFV, SQMV and SQPV have exact C++26 development packages and owner contracts; later providers and transports need independent admission. |
| R20 | Memory, failure, semantics, and measured capacity evidence. | Focused native, installed-consumer and interruption evidence belongs to each module; workload-based release thresholds remain pending. |
| R21 | Advanced transports, GPU, Arrow, and storage choices independently selectable/removable. | Local file retention is one explicitly selected module; advanced transports, GPU, Arrow and alternative backends remain independent. |
| R22 | Publish only evidenced capability and extent. | Status rule; each library claims only its tested native contract and exact development target. |

## Technical Identity and Next Gates

The isolated `prototypes/sqv-research-data/` C++ fixture remains historical development evidence. Its 1 MiB frame ceiling is prototype-specific. The first source module is `modules/sqfv-batch-cpp/`, now with a native C++26 API, separate exact descriptor, local frame, package ceilings, and numerical test fixture. Its tested host and installed-consumer evidence belong to the implementation increment, not a supported-platform matrix. The former local `0.1.0-dev` C-facing development interface is superseded by `0.2.0-dev`.

`sqfv-batch-cpp` admits those contracts for a trusted same-process development slice only. Its binding remains opaque to the flow library. `sqmv-metadata-cpp` now owns a bounded immutable reference grammar and its exact binding bridge; `sqpv-local-store-cpp` consumes that resolved binding and the exact SQFV codec for local retention. Provider-specific schema meaning, evidence verification, runtime access enforcement and SQDV cutover remain separate. The broad candidate name `sqv-research-data-core` remains unadmitted as an omnibus runtime. Do not make the finite knowledge-engine JSON envelope a bulk stream protocol.

The child and module routes use the 2,048-entry SKVI v2 contract; an older 1,024-entry SKVI v1 engine cannot check the expanded index. SKVI capacity is a knowledge-control bound. Domain labels do not allocate colon namespace families. Exact production workloads, numerical release thresholds, additional durability/backend profiles, access enforcement and source-specific operations remain open gate decisions. qxctl surface instructions are expected from Duncan after SQV implementation.
