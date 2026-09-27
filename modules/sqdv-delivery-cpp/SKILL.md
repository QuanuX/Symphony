# SQDV Delivery C++ Use

## Purpose

Use the exact installed C++26 library for a caller-selected full-batch view and
trusted same-process recipient. Read the installed SPEC before treating any
processing, retention, or release event as evidence of another event.

## Operation

1. Resolve an exact SQMV manifest and configure one bounded SQFV context.
2. Select `disposable` for caller-supplied live batches, or create/open a
   `RetainedSource` using the actual SQPV root and exact options for
   `retained_before_delivery`. All source and session handles share the one
   locked underlying store.
3. Create a `Session` with exact view/recipient/interface, partition, producer
   generation, initial sequence, and finite byte and processing limits. Supply
   an exact matching checkpoint only when resuming a caller-selected position.
4. Offer disposable batches directly. For retained delivery, commit through
   `RetainedSource::commit` to obtain an opaque verified `RetainedBatch`, then
   optionally offer that live candidate; the session reads the exact next
   retained sequence when the candidate is elsewhere or absent.
5. Take a `Delivery`. Acknowledge processing in sequence when processing has
   actually completed. Independently release its payload when byte access is
   finished. Persist processed checkpoints through the caller's selected owner.
6. On an uncertain source outcome, preserve available processing progress,
   release payloads, close all source/session handles, and reopen the exact
   source. Resume with a matching checkpoint; old delivery tickets and old
   source proof handles cannot enter the new handle instance.

## Scope

The library does not authorize recipients or prove source rights. Metadata
classification, actual access authority, consumer processing assertions, local
retention, external destination acceptance, and payload release remain separate.
Discarding an unacknowledged ticket does not advance its session. Caller-owned
resume persistence and replay policy determine handling of a lost checkpoint.
