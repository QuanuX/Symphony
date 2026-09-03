# Symphony Emerging Vector Architecture

## Status

Architect-ratified architecture baseline for the Phase 2 through Phase 8 planning program. This surface records settled ownership and boundaries. It does not make a deferred engine, command, API, provider integration, or runtime operational.

## Delivery Phases and Architectural Vectors

Phases are chronological research and delivery campaigns. Vectors are enduring semantic ownership domains. A component first implemented during one phase does not make that phase its permanent owner.

The current delivery sequence is:

1. Phase 2: operations, provisioning, provider reality, Node identity, Habitat conditioning, and the research required for optional remote qxctl operation;
2. Phase 3: delivery, update, and manipulation of Nests on conditioned Nodes, including the optional Prima Parte state witness;
3. Phase 4: backtesting architecture;
4. Phase 5: broker C++ API integration;
5. Phase 6: indicator and quantitative-mathematics integration;
6. Phase 7: local and remote agentic collaboration, extended context, structured long-term logic, and communication;
7. Phase 8: governed conversational construction and deployment of indicators, strategies, and other user-selected tools.

Later phases remain direction, not complete specifications.

## Platform Purpose

Symphony is an agentic-first cockpit for quantitative developers, quantitative researchers, quantitative traders, and other user-defined participants. It is intended to help them research, construct, test, deploy, operate, and eventually observe trading systems without repeatedly rebuilding the constant framework surrounding their own work.

Symphony supplies framework capabilities, exact compatibility evidence, bounded administration, and composable independently installed modules. The user owns strategy logic, provider selection, hardware selection, bus topology, adapters, resource allocation, and the purposes for which Nests are built.

Symphony does not rank providers, impose a preferred trading architecture, or reinterpret a user strategy in order to make it conform to a platform preference.

## Node, Habitat, and Nest

A **Node** is the target physical computer resource with specifically defined resources.

A **Habitat** is the exact operating system and package conditioning deployed to a Node for the work the user intends to place there. A Habitat contains the required software surface without unrelated packages, hidden runtimes, containers, or orchestration layers. A broad general-purpose Habitat is a valid explicit exception when a user wants a wider compatibility surface.

A **Nest** serves a user-selected purpose. Live strategy execution is expected to be common, and one strategy may contain many modular code sections, but Symphony does not enumerate or restrict valid Nest purposes. A Nest is not synonymous with one source file, one process, one algorithm, or one mandatory broker or data interface.

Node, Habitat, and Nest identities, versions, digests, requirements, and observed capabilities form exact compatibility evidence. Symphony may report a match, mismatch, unresolved fact, or relevant consequence. It must not convert that evidence into provider favoritism or strategy doctrine.

## Physical Identity and Cluster Topology

A physical Node remains the same Node when it disconnects and reconnects or when only its software is reflashed. A destroyed or replaced resource is a new Node. A material change to local hardware creates a new physical-resource identity even when an external provider identifier does not change. Remotely attached or virtual resources are distinguishable from the Node's local physical resources and do not alone create a new local Node identity.

A Node incarnation begins when a Node is attached to a bus and established as a participant in a trading system, research cluster, or another Symphony-built system. Incarnation semantics do not replace the underlying physical identity.

Multiple Nodes connected to one another by one or more buses form a cluster. Nodes with no bus connection to one another are not a cluster. A cluster may use multiple bus fabrics, and Node-in-cluster relationships may distinguish the bus through which a relationship exists.

TOPS is the dominant Trade Operating System scope. A TROG is a deliberately subordinate, non-global trading-group scope. Despite the historical wording, a TROG is not necessarily virtual; its distinguishing property is that its scope and rules are not universal.

Names are user-assigned facts. Symphony records, scopes, relates, and resolves them but does not issue them. A name registered directly within a TOPS or TROG follows that scope. Within multiple named clusters, a Node short name may recur in different clusters but may not ambiguously identify two active Nodes in the same cluster. Retired names may be reused according to their owning identity contract.

## Performance and Bus Boundaries

Symphony's administrative and knowledge mechanisms are cold or freezing-path work. They must not become continuously resident dependencies of hot or warm execution.

The execution Node will often receive live data and execute a strategy at the same edge. Market, trade, hardware, network, or other information may leave that edge as exhaust, but no universal exhaust contents, schema, direction, synchronization rule, or downstream topology is imposed here.

Heavy workloads need not share an exhaust, control, research, or other bus where doing so could introduce jitter. A user may deploy multiple buses or bypass a bus for a particular path. NATS JetStream, ZeroMQ, and other fabrics remain adapters and user choices. The existence of qxctl bus administration does not make bus traversal mandatory.

## Vector Landscape

- SKV owns the canonical knowledge framework and the common Phase 1 contract, indexing, evolution, change, feature, API, publication, security, audit, Accordare, lifecycle, Maestro, and qxctl integration surfaces.
- SOV owns operations semantics exercised through qxctl, including provisioning, conditioning, delivery, provider/local actions, and bus-adapter administration.
- SCV owns private semantic knowledge about offsite provider resources, offerings, regions, constraints, and hybrid possibilities. SOV acts upon SCV evidence; it does not take SCV semantic ownership.
- SNV owns Node identity, resource, cluster-relationship, and SNV-bounded naming records through SNIV, SNRV, SCIV, and SCNV.
- SQV owns Symphony's quantitative framework domain without acquiring user strategy logic. SOOV is its high-performance C++-only FIX architecture.
- SHV owns hardware-capability knowledge and its future reproducible C++ graph engine.
- SAV remains the Symphony Accordare Vector.
- SIV is the Symphony Intelligence Vector. SMCV is its optional Markdown conversion component, and SAIV is reserved for a later integration sub-vector.

Backtesting, broker/data API lineage, non-FIX broker integration, indicator mathematics, freezing-path data handling, persistence/replay adapters, and account truth are recognized future domains whose exact vector allocation remains deferred.

## Contract Integration

Each new semantic owner receives a Contract Quad or an explicit subordinate contract assignment from its parent. Companion surfaces remain governed by that owner and do not enlarge the Quad merely because a focused document helps agents understand the domain.

Every new vector and subvector is evaluated against the Phase 1 surfaces below. Evaluation does not mean that every relationship is active: `not applicable`, `deferred`, and `not implemented` remain exact results when supported by the applicable owner evidence.

| Phase 1 surface | Required link from emerging vectors | Activation boundary |
|---|---|---|
| SKV Contract Quad | Inherit common source-truth, proposal, projection, path-safety, compatibility, and thermal doctrine while retaining domain ownership. | Every vector and subvector. |
| `SLANG.md` | Route each platform term to its exact semantic owner with examples and counterexamples, without copying the owner's doctrine. | When canonical nomenclature is added, changed, or retired. |
| `NAMESPACES.md` | Register and delegate stable identity families without turning aliases, display names, or external identifiers into Symphony authority. | When a machine identity family or namespace is ratified. |
| SKVI | Index each current canonical surface once, including owner, truth role, relationships, consumers, and projection eligibility. | When current source truth is added, moved, changed, or removed. |
| SCLV | Preserve an append-only account of a completed material source change and its exact evidence; do not keep deleted current prose alive merely to make history resolvable. | After the material source change is complete. |
| SEV | Assess additions, changes, renames, supersessions, deprecations, retirements, and removals through the affected owners and compatibility evidence. | When meaning, identity, compatibility, ownership, or governed relationships evolve. |
| SSFV and feature administration | Register real installable or user-facing capability and map real administrative interactions, or record an evidence-backed disposition. Architecture prose alone is not a feature. | When an implemented capability or administration surface enters scope. |
| SACV | Govern a versioned API contract and its compatibility only after an API surface is actually ratified. Mention of a future private API does not create one. | When a network/API contract exists. |
| SAV | Resolve declared component composition and compatibility without taking ownership of the components' semantics or choosing a design for the user. | When installable components and their exact compatibility declarations exist. |
| qxctl | Present agentic-first administration through stable commands that invoke owner-defined operations without absorbing their semantics. | When Symphony supports an administrative operation. |
| lifecycle, receipts, bindings, and Maestro | Preserve exact package identity, side-by-side compatibility, selected installation, state, and optional docking/presence evidence. | When an independently installed executable exists. |
| SSIAG and STAV | Bind mutations to target-host ownership or explicit permission and record only the exact audited outcomes their contracts admit. | When an operation bears authority or an audited outcome. |
| Accordare | Preserve pre-mutation intent, terminal outcome, and bounded recovery without becoming a generic mutation authority. | When an applicable ratified mutation circuit uses it. |
| validator and invariant ownership | Close deterministic source checks and assign each new cross-component invariant to its lowest authoritative owner with producer and consumer evidence. | When a checkable source rule or cross-boundary invariant is introduced. |
| SODV | Govern only an authorized official QuanuX publication or documentation projection; private installation truth remains private to that installation. | When official publication is proposed. |

An emerging vector therefore links back by evidence and declared consequence, not by ritual duplication. A deferred engine has no receipt or binding; a proposed API has no SACV registry entry; a contract-only domain has no fabricated SSFV feature; and an unselected publication has no SODV completion claim.

The admission and change sequence is:

1. Architect ratification;
2. owner Contract Quad and declared companions;
3. SKVI source routing;
4. implementation and compatibility evidence;
5. SSFV, qxctl, lifecycle, SSIAG, and STAV consequences where applicable;
6. validator and invariant closure;
7. an append-only SCLV record after the material source change is complete;
8. optional SODV-governed official publication.

SKVI maps current truth. SCLV and repository history preserve change truth. SODV governs official QuanuX publication. A private Symphony installation may build and use its own knowledge and agentic projections, but those facts do not rewrite the official source.

## Historical Engine Intake

Historical QuanuX engines will be considered only when their author supplies them for the applicable phase. Their major functions may be separated into reusable example components and later reassembled into coherent public engines. Function-to-vector assignment follows architectural review, and final engine naming remains with their author.

## Deferred Design

This baseline does not yet define:

- provider-specific schemas or Terraform recipes;
- SCV or SHV graph ontology, database product, evidence acquisition, or query API;
- exact SNV identifier encodings or the SNRV remote-resource `::` grammar;
- Habitat image construction, signing, or package-resolution protocols;
- a cluster-bus adapter protocol;
- FIX session or message behavior inside SOOV;
- a general broker or market-data API contract;
- data-island, persistence, replay, or account-vector architecture;
- Phase 4 through Phase 8 engine designs;
- autonomous canonical application of knowledge changes.

Missing details remain explicit research obligations. They must not be filled by inference.
