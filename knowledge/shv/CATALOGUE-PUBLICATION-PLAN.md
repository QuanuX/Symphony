# Canonical catalogue publication — next transaction

SHV-18 implementation: see `PUBLICATION.md` for the current protected local catalogue-head contract. The design below remains provenance; completed validation is recorded separately in the SHV-18 packet.

Implementation plan, not an enabled publication command. The durable graph store is a prerequisite, not the head authority.

SHV owns a named, TOPS-scoped selected catalogue head. A publication candidate must bind the exact immutable partition manifest, complete declared dependency inventory, caller selection policy, retained refresh endpoints and original owner replay. Missing partitions remain visible; whether incompleteness is permitted is an explicit caller publication requirement, never inferred from a store commit.

Use a dedicated `shv catalogue publication` family after inspecting existing catalogue controls. Proposed operations plan/apply/status; the existing apply retry handles recovery. Planning performs no authority mutation. An apply intent binds the expected prior head digest, candidate digest, operation ID, exact semantic owner installations and replay evidence. Persist that intent before requesting SSIAG authorization under a separately registered catalogue-publication resource/action. Bind the actual committed STAV policy decision and correlation to the retained attempt. At mutation time recheck expiry, revocation and exact prior head with strong compare-and-swap. Never borrow source activation permission.

Persist immutable data first. Publish a head only after verifying referenced content is durable and replayable. No distributed transaction across DuckDB and the head store is promised: a crash before head commit may leave unselected immutable evidence. Retrying a committed operation returns the same selected revision; conflicting concurrent publication stays a conflict, with no implicit rebase. History preserves retired fields and old manifests. Explicit future retention/deletion requires reference accounting; this increment deletes nothing.

Acceptance before enabling apply: denied/expired/wrong-resource authority, concurrent expected-head conflict, killed process at each durability boundary, exact retry, receipt/source change during replay, missing partition policy, historical reads, and independent consumer validation. qxctl naming, registry, schemas, SSIAG/STAV bindings and the contract quad must land together. No new public command is reserved until its implementation exists.
