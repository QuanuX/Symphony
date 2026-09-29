# Symphony Quantitative Persistence Vector Specification

## Retention and Retrieval Contract

An admitted SQPV operation identifies the selected dataset and revision, representation, partition, writer generation, retained artifact or segment, exact covered positions or ranges, integrity method, storage backend and release, and applicable access and retention scope. Source-native captures and transformed outputs remain separately attributable. A physical re-encoding is not a new source observation; any change of meaning belongs to SQTV with lineage described by SQMV.

The storage contract states its actual commit and read-consistency boundary. Buffering, staging, a completed write, verified bytes, local durable commit, selected replication, and external API acceptance are distinct states. An integrity digest alone proves neither authority nor crash durability. A retained position covers only positions proven under the named guarantee; a highest position does not conceal gaps below it.

## Recovery and Retention

The writer records enough exact identity to reconcile an interruption before commit and a lost acknowledgement after commit. It must not infer a committed head from file recency or blindly append a duplicate when the earlier outcome is uncertain. An obsolete writer generation cannot publish over a newer committed generation. Readers and retention work preserve admitted read obligations before reclaiming an artifact; a retention limit has an explicit full-capacity outcome rather than silent overwrite.

An implementation must validate synchronization, atomicity, fencing, and crash behavior on its selected platform and storage backend. Immutable segments, manifests, bounded journals, and read-back verification are candidate mechanisms, not mandatory topology or a selected format. Each backend must admit its exact durability level, retention limits, failure policy and public storage interface. The first implementation below selects one bounded profile; other backend decisions remain open.

## Live and Retained Relationship

SQPV supplies verified retained ranges for a selected consumer to catch up through SQDV and SQFV. Sharing an immutable RAM batch after its storage commit is permitted under SQFV's lifetime contract; delivery need not read the same bytes back from storage. Cutover from RAM to retained data requires exact sequence, revision, generation, duplicate, and gap rules at the selected interfaces. A storage result does not by itself establish consumer acknowledgement or universal exactly-once delivery.

## First Native Local Store

`modules/sqpv-local-store-cpp/SPEC.md` owns the exact `0.1.0-dev` C++26 macOS local APFS library. One caller-selected root binds an immutable SQMV manifest, partition, producer generation, store generation, initial sequence and finite capacity. An exclusive writer lock serializes the selected store. Verified immutable SQFV frames and commit records, plus a checked head checkpoint, preserve a contiguous retained range. Exact retries return the original verified result; changed data at the same position conflicts. Capacity exhaustion stops admission without evicting committed data.

This module's selected guarantee uses explicit file and directory synchronization and is verified with process interruption/reopen tests on its stated development target. It does not claim power-loss certification, replication, runtime access enforcement, generation replacement, concurrent external mutation tolerance or a universal storage default. A retained read returns an independent SQFV batch under the caller's limits. SQDV still owns consumer cutover and destination interpretation. Alternate backends, retention pruning, broader durability profiles and performance thresholds remain later contracts; no colon namespace or SQV qxctl surface follows from this library.

## Non-Authorization Statement

SQPV does not authorize private-data access, deletion, remote publication, provider collection, destination delivery, or a claim that an untested target filesystem supplies a particular durability guarantee.
