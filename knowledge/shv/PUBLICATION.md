# SHV protected catalogue publication v1

Status: implemented bounded SHV-18 contract. Verification and build identities are recorded in the accompanying increment packet.

The independently installed C++ shv-publication-engine 0.1.0-dev owns immutable
catalogue revisions, caller completeness policy, expected-head transitions and
history validation. Its pure operations never write or grant permission.
qxctl owns protected publication transactions and independently checks results.

## Caller authority and scope

One catalogue belongs to one TOPS and caller-selected catalogue_id. Callers select
the manifest, exact dependency installations, original refresh evidence and durable
graph snapshots. Explicit missing_partitions and missing_references policies are
allow or reject. Missing evidence is not hardware incompatibility. Empty and partial
catalogues are valid when caller policy permits them. No vendor universe is imposed.

Publication binds every loaded partition to its original source refresh endpoint
and exact durable graph snapshot. qxctl replays source capture, kernel catalogue,
partition and graph export before preparing an intent and again before the final
head rename. A digest declaration alone is insufficient for publication.

## Control plane

The single command family is `qxctl shv catalogue publication`:

- inspect: native descriptor from an exact receipt-owned installation.
- plan: replay desired evidence and derive a plan against the current head.
- apply: execute a plan via --input, or recover it via --operation-id (exactly one).
- status: current head and retained history; optional operation selects a historical attempt.
- schema and template: receipt-owned discovery resources.

All commands require --prefix and --version. Plan, apply and status also require
--tops-id, --catalogue-id and --state-root. Plan input has operation_id, reason and
desired. Desired has manifest, policy, partition_prefix, partition_version and
members. Each member has partition_digest, endpoint and store. Endpoint uses the
existing SHV comparison endpoint fields; store selects root, tops_id, namespace,
snapshot_digest, prefix and version. qxctl enriches these with exact installation
receipts and replayed bundle/graph digests. Native schema describes that enriched
contract. Templates intentionally contain null placeholders requiring caller input.

## Protected transaction

The distinct action is `symphony.shv.catalogue.publish`. Its resource is
`symphony.shv.catalogue:<sha256 hex>` over canonical JSON containing tops_id,
owner_engine_id=symphony-shv-publication and catalogue_id. Audience is qxctl and
scope is tops:<TOPS UUID>. Source activation grants do not grant publication.

The journal resides at <state-root>/symphony/qxctl/shv/catalogues-v1/<tops>/<hash>/state.json.
Private permissions, no-follow ancestry checks, a filesystem lock, temporary-file
fsync, atomic rename and directory fsync protect updates. An immutable intent is
saved before requesting SSIAG authorization. The actual policy decision is saved
before publication. Immediately before rename qxctl checks original expiration,
replays all dependencies, and requests a fresh authenticated decision. A changed
policy/configuration digest, subject or grant stops publication. This is a final
check, not an atomic transaction spanning SSIAG and filesystem state: authority or evidence changes after
the check can race the rename. Recorded authorization is the original decision;
the recheck has its own SSIAG audit. No STAV head-write receipt is fabricated.

Interrupted prepared or authorized operations recover through apply --operation-id.
Committed retries return retained evidence with the current head; they never rewind
it. Competing plans use exact expected-state compare-and-swap. Changed intent under
an existing operation ID is rejected. Status replays retained semantics; apply also
requires present original sources, snapshots and exact installations.

This local named-head selection does not enable Symphony-wide CanonicalApply.
Results explicitly report canonical_apply_enabled=false and
head_write_stav_receipt=null. Digests detect corruption; they are not signatures
against an owner who can replace their own journal.

## Bounds and compatibility

This release admits eight partition entries and members, 32 head revisions,
128 retained attempts, 16 MiB journal bytes and a bounded native history envelope.
No silent history pruning is provided. Source 0.1.0-dev, partition 0.2.0-dev,
kernel 0.1/0.2/0.3.0-dev and graph DuckDB connector 0.1.0-dev are selected exactly.
Publication compiles partition 0.2 semantics and independently verifies its selected
installation; changing that dependency requires a reviewed release. These are tool
version contracts, not restrictions on user architectures. Graph backend choice
remains open. Hardware extension fields retain their original partition semantics.

## Verification boundary

Native conformance, independent Go validation, protected journal tests, installed
receipt tests and live SSIAG interruption/revocation evidence are separate claims.
Only completed evidence recorded in the SHV-18 packet establishes a tested boundary.
Test process barriers compile only with symphony_publication_faults; production
builds contain no environment-controlled stop behavior.

## Explicit writer admission, publisher 0.2.0-dev

This publisher accepts exact graph-store writer installations 0.1.0-dev, 0.2.0-dev
and 0.3.0-dev. The unchanged 0.1 publisher and its independent consumer retain
0.1-only admission. Each historical journal attempt is validated using its recorded
publisher version; a newer history reader does not retroactively broaden an old
attempt. Newer publication can bind a transferred graph only through the existing
source/kernel replay, partition correspondence and actual SSIAG authorization checks.
Copying a graph alone never changes a catalogue head.

## Exact interface metadata — 0.3.0-dev

OWNER-INTERFACE.json owns mechanical descriptor metadata, supported releases and package inventory. `tools/shv-interface-codegen/generate.py --owner shv-publication-engine` generates src/interface.generated.hpp, the owner Go admission file and the CMake inventory. tests/fixtures/interface-history.v1.json retains original installed descriptors; the shared history lock and CTest protect exact parity. Semantic handlers and independent qxctl replay are handwritten. The package installs its receipt-owned declaration; qxctl checks it against compiled admission. All existing command routes and defaults remain unchanged.

Publication 0.1/0.2 remain supported. Publication 0.3 preserves 0.2 semantics and store-writer admission 0.1/0.2/0.3. The embedded partition reader remains exactly 0.2, and source-member admission remains exactly 0.1. New source 0.2 is not silently substituted into publication. Retained publication intents replay under their original owner.
