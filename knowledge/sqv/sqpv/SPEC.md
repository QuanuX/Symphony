# Symphony Quantitative Persistence Vector Specification

## Retention and Retrieval Contract

An admitted SQPV operation identifies the selected dataset and revision, representation, partition, writer generation, retained artifact or segment, exact covered positions or ranges, integrity method, storage backend and release, and applicable access and retention scope. Source-native captures and transformed outputs remain separately attributable. A physical re-encoding is not a new source observation; any change of meaning belongs to SQTV with lineage described by SQMV.

The storage contract states its actual commit and read-consistency boundary. Buffering, staging, a completed write, verified bytes, local durable commit, selected replication, and external API acceptance are distinct states. An integrity digest alone proves neither authority nor crash durability. A retained position covers only positions proven under the named guarantee; a highest position does not conceal gaps below it.

## Recovery and Retention

The writer records enough exact identity to reconcile an interruption before commit and a lost acknowledgement after commit. It must not infer a committed head from file recency or blindly append a duplicate when the earlier outcome is uncertain. An obsolete writer generation cannot publish over a newer committed generation. Readers and retention work preserve admitted read obligations before reclaiming an artifact; a retention limit has an explicit full-capacity outcome rather than silent overwrite.

An implementation must validate synchronization, atomicity, fencing, and crash behavior on its selected platform and storage backend. Immutable segments, manifests, bounded journals, and read-back verification are candidate mechanisms, not mandatory topology or a selected format. The first backend, durability level, retention limit, failure policy, and public storage interface remain open admission decisions.

## Live and Retained Relationship

SQPV supplies verified retained ranges for a selected consumer to catch up through SQDV and SQFV. Sharing an immutable RAM batch after its storage commit is permitted under SQFV's lifetime contract; delivery need not read the same bytes back from storage. Cutover from RAM to retained data requires exact sequence, revision, generation, duplicate, and gap rules at the selected interfaces. A storage result does not by itself establish consumer acknowledgement or universal exactly-once delivery.

## Deferred Technical Contract

No backend, segment layout, durability profile name, storage credential path, retention policy, public ABI, or performance threshold is selected by this Quad. Each future adapter needs its own version, resource bounds, recovery evidence, read semantics, and lifecycle admission. No installed capability or namespace family is created here.

## Non-Authorization Statement

SQPV does not authorize private-data access, deletion, remote publication, provider collection, destination delivery, or a claim that an untested target filesystem supplies a particular durability guarantee.
