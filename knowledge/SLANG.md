# Symphony Platform Nomenclature

## Canonical Target

`knowledge/SLANG.md`

## Status and Authority

This is the canonical SKV platform-nomenclature companion. It is governed by the `knowledge/` Contract Quad but is not a fifth Contract Quad member, a vector, a registry, a schema, or an executable specification.

SLANG maps a Symphony term to a concise platform meaning and the exact contract that owns its semantics. When this orientation and an owner contract differ, the owner contract controls. The discrepancy is drift to review; SLANG does not acquire the missing authority by restating it.

## Purpose

Give people and agents one reliable route from Symphony language to canonical owner truth without distributing competing definitions throughout prompts, generated documentation, validators, or implementation code.

## Entry Model

Every canonical entry contains:

- `term`: the exact preferred platform spelling;
- `meaning`: concise orientation sufficient to select the right owner;
- `owner_contract`: the canonical contract that defines the term's domain semantics;
- `example`: one representative correct use;
- `counterexample`: one nearby but incorrect use;
- `notes`: optional nomenclature or routing context, never copied runtime doctrine.

A term is admitted only after its semantic owner exists. A name used in discussion does not become canonical merely because it appears in a prompt, issue, source symbol, generated projection, or historical record. Synonyms, abbreviations, capitalization variants, and replacements require explicit reviewed treatment; tools must not infer them from spelling similarity.

## Canonical Entries

### Symphony Knowledge Vector

- term: `Symphony Knowledge Vector`
- meaning: The umbrella framework for Symphony's declarative platform knowledge and its common cross-vector contracts.
- owner_contract: `knowledge/INTENT.md`
- example: `SKVI is a vector surface within the Symphony Knowledge Vector.`
- counterexample: `SKV is one monolithic runtime that owns every vector's behavior.`
- notes: The preferred abbreviation is `SKV`. Each vector retains its own semantic ownership.

### vector

- term: `vector`
- meaning: An enduring bounded Symphony semantic-ownership domain whose canonical contract owns that domain's meaning and permitted operations.
- owner_contract: `knowledge/MANIFEST.md`
- example: `SEV owns Symphony evolution semantics as a vector within SKV.`
- counterexample: `Any directory, executable, database table, or topic is automatically a vector.`
- notes: A vector may have an independently installed engine, but the engine does not become the contract owner.

### Contract Quad

- term: `Contract Quad`
- meaning: The four canonical files `INTENT.md`, `MANIFEST.md`, `SKILL.md`, and `SPEC.md` through which an owning domain declares purpose, identity, safe use, and normative behavior.
- owner_contract: `knowledge/MANIFEST.md`
- example: `knowledge/sev/INTENT.md, MANIFEST.md, SKILL.md, and SPEC.md form the SEV Contract Quad.`
- counterexample: `Every canonical companion file becomes another member of the Quad.`
- notes: Schemas, registries, profiles, ledgers, and companion surfaces remain governed artifacts outside the four-member structure.

### companion surface

- term: `companion surface`
- meaning: A canonical knowledge file governed by an existing Contract Quad for a bounded cross-cutting purpose without becoming another Quad member or an independent vector.
- owner_contract: `knowledge/MANIFEST.md`
- example: `knowledge/NAMESPACES.md is an SKV-owned companion surface.`
- counterexample: `Adding a companion surface implicitly creates an engine, runtime service, or fifth Quad member.`
- notes: A companion routes or specializes owner truth; it cannot expand its own authority.

### canonical source truth

- term: `canonical source truth`
- meaning: Reviewed owner-controlled contract or data that authoritatively declares its bounded subject in the current repository state.
- owner_contract: `knowledge/SPEC.md`
- example: `A vector Contract Quad is canonical source truth for that vector's semantics.`
- counterexample: `A generated graph or engine response replaces the owner contract because it is easier to query.`
- notes: Canonical status is bounded by the owning surface; it is not a claim that every statement in the repository has equal authority.

### derived projection

- term: `derived projection`
- meaning: A noncanonical, disposable, rebuildable representation produced from identified canonical inputs.
- owner_contract: `knowledge/SPEC.md`
- example: `A digest-bound SKVI JSON view is a derived projection.`
- counterexample: `A projection becomes source truth when cached, indexed, or stored in a database.`
- notes: The owning contract determines which projections are permitted and what evidence they must retain.

### vector engine

- term: `vector engine`
- meaning: A separately bounded implementation process that performs only the operations authorized by one vector contract.
- owner_contract: `knowledge/SPEC.md`
- example: `symphony-sev evaluates SEV evidence through the common process envelope.`
- counterexample: `The engine owns the vector's semantics or may rewrite canonical truth because it can validate it.`
- notes: Installation, selection, docking, invocation, authority, and canonical ratification remain separate concerns.

### qxctl

- term: `qxctl`
- meaning: Symphony's Go-based headless administrative command surface, which owns its command grammar and presentation while invoking owner-defined capabilities.
- owner_contract: `tools/qxctl/INTENT.md`
- example: `qxctl may select and invoke an exact receipt-validated vector engine.`
- counterexample: `qxctl defines a vector's domain semantics because it exposes the command.`
- notes: A qxctl route is required only where the applicable administration contract requires one.

### SKVI

- term: `SKVI`
- meaning: The Symphony Knowledge Vector Index, which maps current canonical surfaces, their owners, roles, relationships, consumers, and projection eligibility.
- owner_contract: `knowledge/skvi/INTENT.md`
- example: `SKVI records one current route for an SOV Contract Quad surface.`
- counterexample: `SKVI decides a vector's meaning or makes an unratified file canonical.`
- notes: Manifest declarations define required discovery closure; SKVI remains the canonical current index.

### SCLV

- term: `SCLV`
- meaning: The Symphony Change Log Vector, which preserves selective append-only truth about completed, ratified source changes and their evidence.
- owner_contract: `knowledge/sclv/INTENT.md`
- example: `A completed contract transition may receive one evidence-bound SCLV record after merge.`
- counterexample: `SCLV mirrors every Git event or stores live operational audit events.`
- notes: Git is supporting evidence; STAV owns runtime audit truth.

### SACV

- term: `SACV`
- meaning: The Symphony API Contract Vector, which governs versioned declarative HTTP API contracts and their registry without taking endpoint semantics from their owners.
- owner_contract: `knowledge/sacv/INTENT.md`
- example: `A ratified private HTTP API may register its owner-controlled OpenAPI contract through SACV.`
- counterexample: `Mentioning a possible API creates an endpoint, SDK, public playground, or SACV entry.`
- notes: qxctl grammar, binary IPC, bus payloads, and publication approval remain outside SACV.

### SODV

- term: `SODV`
- meaning: The Symphony Official Documentation Vector, which governs how authorized canonical knowledge and module-release evidence may become official public projections.
- owner_contract: `knowledge/sodv/INTENT.md`
- example: `SODV may verify an evidence-backed release record before an official projection is claimed complete.`
- counterexample: `The presence of a new vector contract automatically publishes documentation or grants SODV authority to publish.`
- notes: SODV is governance, not the publisher, documentation site, or source-truth owner.

### SSFV

- term: `SSFV`
- meaning: The Symphony Semantic Feature Vector, which owns feature identity, semantics, hierarchy, lifecycle, and distributed owner-record routing.
- owner_contract: `knowledge/ssfv/INTENT.md`
- example: `An implemented administrator-visible capability may receive a reviewed SSFV record and administration mapping.`
- counterexample: `A directory, source file, or architecture idea is automatically a feature.`
- notes: Feature-worthiness is an owner-reviewed semantic decision, not an engine inference.

### SAV

- term: `SAV`
- meaning: The Symphony Accordare Vector, which resolves one exact composition and evaluates declared relationships without taking truth from the referenced owners.
- owner_contract: `knowledge/sav/INTENT.md`
- example: `SAV may report an unresolved relationship in a coverage-qualified CURRENT snapshot.`
- counterexample: `SAV means the intelligence vector, selects newest versions, or repairs an incompatible composition.`
- notes: Accord Reference, CURRENT, Named Version, and three-axis evaluation semantics remain SAV-owned.

### SEV

- term: `SEV`
- meaning: The Symphony Evolution Vector, which evaluates planned change or encountered novelty, its consequences, disposition, recalculation, verification, and closure.
- owner_contract: `knowledge/sev/INTENT.md`
- example: `SEV may identify which owner evidence is still required before a proposed transition can converge.`
- counterexample: `A successful SEV plan applies itself or grants canonical mutation authority.`
- notes: SCSEV and knowledge-surface evolution are governed profiles, not separate mutation engines.

### SSIAG

- term: `SSIAG`
- meaning: Symphony Secure Identity and Access Governance, which owns identity, authentication, authorization, bounded capability, credential-reference, lease, and provider-operation relationships.
- owner_contract: `knowledge/ssiag/INTENT.md`
- example: `SSIAG may issue a short-lived caller-neutral decision after the required safe audit event commits.`
- counterexample: `An AI caller receives authority merely because it is an agent, or qxctl receives raw credentials.`
- notes: Target-host ownership and explicit grants remain the root of supported local authority.

### STAV

- term: `STAV`
- meaning: The Symphony TOPS Audit Vector, which owns the safe per-TOPS administrative and security audit protocol while runtime ledgers remain installation-local.
- owner_contract: `knowledge/stav/INTENT.md`
- example: `An authorized producer submits one allowlisted candidate to the per-TOPS append authority.`
- counterexample: `STAV is SCLV, Git history, a general telemetry bus, or a place for credentials and raw provider payloads.`
- notes: Protocol truth, append authority, producer grants, and ledger state remain distinct.

### Maestro

- term: `Maestro`
- meaning: The freezing-path presence authority that records exact authenticated vector-engine docking relationships and derives read-only receptor inventory.
- owner_contract: `modules/maestro/INTENT.md`
- example: `Maestro may record that one exact installed engine is docked at an administrator-selected receptor.`
- counterexample: `Maestro starts, schedules, supervises, invokes, or owns the semantics of docked engines.`
- notes: Presence is distinct from installation, selection, liveness, execution, and semantic compatibility.

### Symphony Ops Vector

- term: `Symphony Ops Vector`
- meaning: The owner of supported provisioning, conditioning, delivery, remote-administration, and bus-adapter operation semantics exercised through qxctl.
- owner_contract: `knowledge/sov/INTENT.md`
- example: `SOV may define an exact qxctl-administered Habitat-conditioning operation.`
- counterexample: `SOV owns provider facts, physical Node identity, or the purpose of a user's Nest.`
- notes: The preferred abbreviation is `SOV`. qxctl implements and presents ratified operations without taking their semantic ownership.

### Symphony Cloud Vector

- term: `Symphony Cloud Vector`
- meaning: The private semantic-knowledge domain for offsite provider resources, offerings, constraints, observations, and hybrid possibilities.
- owner_contract: `knowledge/scv/INTENT.md`
- example: `SCV may preserve an observed provider-region constraint as private evidence.`
- counterexample: `SCV ranks providers or authorizes SOV to provision a resource.`
- notes: The preferred abbreviation is `SCV`. Owned hardware enters only where its comparison with offsite resources is applicable.

### SCHV

- term: `SCHV`
- meaning: Symphony Cloud Hyperscalers Vector. Hyperscaler and cloud-hosting family knowledge, preserving each provider vocabulary.
- owner_contract: `knowledge/scv/schv/SPEC.md`
- example: `SCHV retains a provider-native fact with its source revision and scope.`
- counterexample: `SCHV membership proves equivalence, excludes a custom bridge or grants provider authority.`
- notes: Subordinate to `SCV`; its engine is independently installable.

### SCEV

- term: `SCEV`
- meaning: Symphony Cloud Edge Vector. Edge-provider family knowledge, preserving distinct product and service boundaries.
- owner_contract: `knowledge/scv/scev/SPEC.md`
- example: `SCEV retains a provider-native fact with its source revision and scope.`
- counterexample: `SCEV membership proves equivalence, excludes a custom bridge or grants provider authority.`
- notes: Subordinate to `SCV`; its engine is independently installable.

### SCHV-AWS

- term: `SCHV-AWS`
- meaning: Symphony Cloud Hyperscalers Vector — AWS. AWS-native source, offering, service, version and constraint knowledge.
- owner_contract: `knowledge/scv/schv/aws/SPEC.md`
- example: `SCHV-AWS retains a provider-native fact with its source revision and scope.`
- counterexample: `SCHV-AWS membership proves equivalence, excludes a custom bridge or grants provider authority.`
- notes: Subordinate to `SCHV`; its engine is independently installable.

### SCHV-AZURE

- term: `SCHV-AZURE`
- meaning: Symphony Cloud Hyperscalers Vector — Azure. Azure-native source, offering, service, version and constraint knowledge.
- owner_contract: `knowledge/scv/schv/azure/SPEC.md`
- example: `SCHV-AZURE retains a provider-native fact with its source revision and scope.`
- counterexample: `SCHV-AZURE membership proves equivalence, excludes a custom bridge or grants provider authority.`
- notes: Subordinate to `SCHV`; its engine is independently installable.

### SCHV-DO

- term: `SCHV-DO`
- meaning: Symphony Cloud Hyperscalers Vector — DigitalOcean. DigitalOcean-native source, offering, service, version and constraint knowledge.
- owner_contract: `knowledge/scv/schv/do/SPEC.md`
- example: `SCHV-DO retains a provider-native fact with its source revision and scope.`
- counterexample: `SCHV-DO membership proves equivalence, excludes a custom bridge or grants provider authority.`
- notes: Subordinate to `SCHV`; its engine is independently installable.

### SCHV-GCP

- term: `SCHV-GCP`
- meaning: Symphony Cloud Hyperscalers Vector — Google Cloud. Google Cloud-native source, offering, service, version and constraint knowledge.
- owner_contract: `knowledge/scv/schv/gcp/SPEC.md`
- example: `SCHV-GCP retains a provider-native fact with its source revision and scope.`
- counterexample: `SCHV-GCP membership proves equivalence, excludes a custom bridge or grants provider authority.`
- notes: Subordinate to `SCHV`; its engine is independently installable.

### SCEV-CF

- term: `SCEV-CF`
- meaning: Symphony Cloud Edge Vector — Cloudflare. Cloudflare-native source, service, platform version and constraint knowledge.
- owner_contract: `knowledge/scv/scev/cf/SPEC.md`
- example: `SCEV-CF retains a provider-native fact with its source revision and scope.`
- counterexample: `SCEV-CF membership proves equivalence, excludes a custom bridge or grants provider authority.`
- notes: Subordinate to `SCEV`; its engine is independently installable.

### Symphony Node Vector

- term: `Symphony Node Vector`
- meaning: The composition domain that relates Node identity, resources, cluster relationships, and user- or provider-assigned names without collapsing them.
- owner_contract: `knowledge/snv/INTENT.md`
- example: `SNV relates one physical Node identity to its resource and cluster evidence.`
- counterexample: `SNV provisions hardware or issues a user's Node name.`
- notes: The preferred abbreviation is `SNV`; SNIV, SNRV, SCIV, and SCNV retain their subordinate ownership.

### Symphony Hardware Vector

- term: `Symphony Hardware Vector`
- meaning: The owner of evidenced hardware-capability knowledge used to understand physical possibilities, constraints, porting, and extension consequences.
- owner_contract: `knowledge/shv/INTENT.md`
- example: `SHV may describe a processor cache hierarchy or NIC timestamping capability with provenance.`
- counterexample: `SHV identifies a user's Node, allocates hardware, or chooses an algorithm for the user.`
- notes: The preferred abbreviation is `SHV`. SNV owns a Node and its resources; SHV owns knowledge about what such hardware can do.

### Symphony Quantitative Vector

- term: `Symphony Quantitative Vector`
- meaning: The owner of reusable Symphony framework contracts for quantitative research, trading-system construction, execution support, and later quantitative engines.
- owner_contract: `knowledge/sqv/INTENT.md`
- example: `SQV may provide a reusable ratified framework component that a user's Nest elects to use.`
- counterexample: `SQV owns user strategy logic or imposes one feed, broker, data, order, or execution design.`
- notes: The preferred abbreviation is `SQV`. Its first named subvector is SOOV.

### SOOV

- term: `SOOV`
- meaning: SQV's high-performance, C++-only Symphony architecture for FIX framework surfaces.
- owner_contract: `knowledge/sqv/soov/INTENT.md`
- example: `Historic QuanuX FIX work may be examined as design input for SOOV.`
- counterexample: `SOOV makes FIX mandatory or makes the historic rough draft automatically canonical.`
- notes: Preferred expansion: `Symphony Orchestra Omega Vector`. Detailed FIX behavior remains deferred to later ratification.

### Symphony Intelligence Vector

- term: `Symphony Intelligence Vector`
- meaning: The future owner of agentic collaboration, context, communication, and governed interaction with Symphony.
- owner_contract: `knowledge/siv/INTENT.md`
- example: `SIV may later coordinate simultaneous local and remote agents using owner-provided context.`
- counterexample: `SIV duplicates another vector's truth or grants authority because a caller is an agent.`
- notes: The preferred abbreviation is `SIV`. SAV remains the Symphony Accordare Vector.

### SMCV

- term: `SMCV`
- meaning: An optional SIV translator between bounded precise Markdown and the exact machine representation required by an owner contract.
- owner_contract: `knowledge/siv/smcv/INTENT.md`
- example: `An enabled SMCV mapping may present a schema-bound result to a model as precise Markdown.`
- counterexample: `SMCV owns IPC, repairs invalid data, invents meaning, or replaces the receiving contract's format.`
- notes: Preferred expansion: `Symphony Markdown Conversion Vector`. Its future qxctl enablement remains optional and unimplemented.

### SAIV

- term: `SAIV`
- meaning: The reserved name for a later Symphony intelligence-integration subvector within SIV.
- owner_contract: `knowledge/siv/INTENT.md`
- example: `SAIV may receive semantics only through a later Architect-ratified owner contract.`
- counterexample: `The reserved acronym proves that SAIV behavior, an engine, or an integration already exists.`
- notes: Preferred expansion: `Symphony Agentic Integration Vector`. It currently has no behavior.

### Prima Parte

- term: `Prima Parte`
- meaning: An optional Phase 3 C++ executable that, when explicitly invoked through qxctl, witnesses, durably records, and reports bounded state for only its own Node before exiting.
- owner_contract: `knowledge/sov/PRIMA-PARTE.md`
- example: `After a material qxctl operation, Prima Parte may record a new local generation and send one bounded state observation.`
- counterexample: `Prima Parte is a daemon, general telemetry collector, mutation authority, bus listener, or smaller running Maestro.`
- notes: The qxctl-addressable surface is not a Phase 1 Maestro receptor. No executable is implemented yet.

### Troll

- term: `Troll`
- meaning: Optional nomenclature for a user-programmed resident placed at a selected connection point; the word alone assigns no required behavior or authority.
- owner_contract: `INTENT.md`
- example: `A user may create a Troll that communicates only the information they program it to communicate.`
- counterexample: `Every Node or cluster requires a Troll, or Symphony defines how every Troll behaves.`
- notes: The exact first-party module identities `node-troll` and `bus-troll` are retired non-reusable tombstones; the general concept may become unnecessary.

### Node

- term: `Node`
- meaning: A physical target computer resource whose continuity is recorded independently from its software conditioning and cluster participation.
- owner_contract: `knowledge/snv/sniv/SPEC.md`
- example: `Reflashing the operating system alone does not create a different Node.`
- counterexample: `Every restart, Habitat change, remote attachment, or provider display-label change automatically creates a new Node.`
- notes: Symphony capitalizes `Node` when referring to this platform concept. Exact identity encodings remain deferred to SNIV.

### Node incarnation

- term: `Node incarnation`
- meaning: The establishment of a Node on a bus as a participant in one trading, research, or other Symphony-built system, recorded separately from physical identity.
- owner_contract: `knowledge/snv/SPEC.md`
- example: `A physical Node may acquire incarnation evidence when it joins a cluster.`
- counterexample: `An incarnation replaces or proves the underlying physical Node identity.`
- notes: Exact transition and cardinality rules remain with future SNV record contracts.

### Habitat

- term: `Habitat`
- meaning: The exact operating-system and package conditioning delivered to a Node for the user-selected work intended there.
- owner_contract: `knowledge/sov/SPEC.md`
- example: `A user may select a narrowly conditioned Habitat or explicitly choose a broader general-purpose one.`
- counterexample: `A Habitat silently adds unrelated packages, containers, orchestration layers, or hidden runtimes.`
- notes: Compatibility with a Node or Nest is evidence; it does not let Symphony redefine the user's work.

### Nest

- term: `Nest`
- meaning: User-purpose code delivered into a Habitat; live strategy execution is common, but Symphony does not define or restrict the Nest's purpose.
- owner_contract: `knowledge/ARCHITECTURE.md`
- example: `A Nest may contain the many modular code sections needed for one user-selected purpose.`
- counterexample: `A Nest is necessarily one source file, one process, one algorithm, or one mandatory broker or data interface.`
- notes: The user's strategy logic remains the user's domain. SOV owns supported Nest delivery operations, not Nest purpose.

### cluster

- term: `cluster`
- meaning: A user-established identity for multiple Nodes related through one or more bus connections.
- owner_contract: `knowledge/snv/sciv/SPEC.md`
- example: `Several bus fabrics may connect the participating Nodes of one cluster.`
- counterexample: `Unconnected Nodes form a cluster merely because a desired configuration lists them together.`
- notes: SCIV owns cluster identity and connectivity relationships; it does not configure the buses.

### TOPS

- term: `TOPS`
- meaning: The dominant Trade Operating System scope in Symphony's trading-system topology.
- owner_contract: `knowledge/ARCHITECTURE.md`
- example: `A Node or user-assigned name may be scoped within one TOPS.`
- counterexample: `TOPS is interchangeable with a Node, cluster, provider account, or globally mandatory deployment topology.`
- notes: Domain contracts retain their exact TOPS identity, authority, and lifecycle rules.

### TROG

- term: `TROG`
- meaning: A deliberately subordinate, non-global trading-group scope beneath the applicable system context.
- owner_contract: `knowledge/ARCHITECTURE.md`
- example: `A user-assigned name declared in a TROG follows that declared scope.`
- counterexample: `A TROG is necessarily virtual or may impose universal rules outside its bounded scope.`
- notes: Its defining distinction is bounded subordination, not virtualization.

### SNIV

- term: `SNIV`
- meaning: The SNV subvector that records physical Node identity, continuity, incarnation, and applicable lineage evidence.
- owner_contract: `knowledge/snv/sniv/INTENT.md`
- example: `SNIV distinguishes a software reflash from material physical replacement.`
- counterexample: `SNIV owns resource inventory, provisioning, or user-facing name resolution.`
- notes: Preferred expansion: `Symphony Node Identity Vector`.

### SNRV

- term: `SNRV`
- meaning: The SNV subvector that records the evidenced installation-local resource composition of a Node and distinguishes explicitly qualified remote resources.
- owner_contract: `knowledge/snv/snrv/INTENT.md`
- example: `SNRV records a remote attachment separately from local physical hardware.`
- counterexample: `SNRV allocates hardware or decides that a Nest can use it.`
- notes: Preferred expansion: `Symphony Node Resource Vector`. The reserved `::` concept is not a ratified grammar.

### SCIV

- term: `SCIV`
- meaning: The SNV subvector for cluster identity and the bus-connected relationships through which Nodes participate in a cluster.
- owner_contract: `knowledge/snv/sciv/INTENT.md`
- example: `SCIV may distinguish an alternate-bus relationship without re-identifying a Node.`
- counterexample: `SCIV creates the cluster, configures transport, or declares desired connectivity observed.`
- notes: Preferred expansion: `Symphony Cluster Identity Vector`.

### SCNV

- term: `SCNV`
- meaning: The SNV-bounded subvector that relates names already assigned to SNV subjects and resolves them within their valid scope and time.
- owner_contract: `knowledge/snv/scnv/INTENT.md`
- example: `SCNV may resolve the same short name differently in two distinct clusters when each result is unambiguous.`
- counterexample: `SCNV is Symphony's universal namespace authority or issues provider and user names.`
- notes: Preferred expansion: `Symphony Consolidated Naming Vector`. Universal identity-family doctrine remains in `knowledge/NAMESPACES.md`.

### stable identity

- term: `stable identity`
- meaning: An owner-assigned identifier whose continuity and compatibility meaning are governed by its registered identity family and domain contract.
- owner_contract: `knowledge/NAMESPACES.md`
- example: `qxcmd:symphony:sev.case-open remains distinct from its displayed command spelling.`
- counterexample: `A title, path, timestamp, newest version, or database row automatically supplies stable identity.`
- notes: Identity families are not interchangeable even when their keys contain the same text.

### identity namespace

- term: `identity namespace`
- meaning: An owner-ratified allocation boundary inside a registered identity family.
- owner_contract: `knowledge/NAMESPACES.md`
- example: `symphony is the first-party namespace token in ssfv:symphony:... identities.`
- counterexample: `A repository owner, package name, hostname, cloud account, or caller type automatically receives the matching namespace.`
- notes: Family registration, namespace allocation, and individual identity assignment are separate acts.

### current contract

- term: `current contract`
- meaning: The presently applicable prospective owner truth in the checked-out canonical contract surface.
- owner_contract: `knowledge/INTENT.md`
- example: `An agent uses the current vector SPEC to describe supported behavior now.`
- counterexample: `An older append-only record overrides a later ratified current contract merely because it remains canonical history.`
- notes: A disagreement among current contract, implementation evidence, and applicable history is surfaced as drift.

### historical record

- term: `historical record`
- meaning: Preserved evidence of what was known, decided, attempted, or completed at an earlier point, interpreted with later corrections and current owner truth.
- owner_contract: `knowledge/INTENT.md`
- example: `A superseded record remains evidence of the earlier state after its successor becomes current.`
- counterexample: `Renaming or removing a current surface means its historical names and evidence never existed.`
- notes: History is not silently rewritten to resemble the present.

## Admission and Review Rules

A reviewed change that introduces or materially changes platform terminology MUST:

1. identify the existing semantic owner or establish that owner first;
2. update the owner contract before or with this routing entry;
3. give examples and counterexamples that clarify the ownership boundary;
4. update affected current references without rewriting immutable history;
5. route the surface through SKVI; and
6. assess compatibility, publication, feature-administration, and validation consequences only where their owning contracts make them applicable.

A validator may check entry shape, uniqueness, owner-path existence, SKVI routing, and prohibited ambiguity. It must not become a second glossary, infer definitions from prose, or decide that two meanings are equivalent. Semantic admission remains reviewed owner truth.

## Non-Authorization Statement

SLANG does not allocate identity namespaces, define domain protocols, prescribe user behavior, create aliases, rewrite owner contracts or history, implement parsers, authorize canonical mutation, or create a runtime dependency. It is a cold knowledge-routing surface for people and agents.
