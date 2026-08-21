# Accordare Audit Boundary

## Purpose

This common knowledge contract connects SAV and SEV operational meaning to STAV without transferring authority between vectors. `knowledge/stav/` owns the event envelope and producer-vocabulary shape; the owning vector defines what one registered operation means; SSIAG owns authorization decisions; and an explicitly reviewed runtime producer must construct and submit safe candidates.

## Current State

The first closed vocabulary is `knowledge/stav/registries/v1/accordare.json`. It registers four SAV Named Version lifecycle tuples and now reports both architectural gates true because the reviewed runtime producer and installation-grant administration circuit are implemented. Those fields remain protocol-eligibility gates, not configuration switches. No installation may infer a grant, live process, append authority, or audit completion from the presence or validity of this file.

SEV currently has no registered producer tuple. Its report-only assessments, novelty checks, graphs, and plans do not justify routine STAV traffic. A later mutation or export operation must pass a separate safe-metadata and producer-boundary review before receiving vocabulary.

## Producer Requirements

The operational `modules/accordare-stav-producer/` boundary:

- accepts only typed operation results and immutable evidence identities;
- fsyncs an authenticated deterministic intent before qxctl invokes a configured coordinator mutation;
- verifies the exact installed component, result protocol, operation, TOPS, request, correlation, actor, expected state, and resulting state;
- maps only the closed vocabulary and never accepts caller-supplied event classes, operation identifiers, intents, reasons, producer identity, event identity, sequence, timestamp, or chain state;
- submits only safe references and previous/new digests through mutually authenticated local STAV IPC;
- persists the safe candidate before append, returns a verified committed receipt or an explicit durable pending state, and reconciles pending evidence idempotently;
- cannot reinterpret SSIAG permission, mutate canonical knowledge, or edit a STAV ledger.

qxctl orchestrates the operation, submits only the typed evidence envelope, validates the producer response, and administers the exact installation grant while STAV is stopped. It never receives arbitrary append authority. SAV and SEV remain read-only C++ engines. The Knowledge Session Coordinator remains the Named Version persistence owner and does not gain STAV transport merely because it produced a result.

Durable intent acceptance is the pre-mutation boundary. qxctl prepares the exact command, coordinator installation, SSIAG decision, peer, operation, and TOPS identity before coordinator invocation. Completion must match that intent exactly. A crash before or after coordinator persistence leaves `intent_pending`; retrying the exact stable command reuses the coordinator's idempotent operation and closes the same intent. Candidate append failures remain separately visible as `append_pending`. Generic reconciliation may replay candidates but cannot guess whether an interrupted coordinator mutated.

The producer implements the closed `succeeded`, `failed`, and `unavailable` vocabulary. Success requires the exact self-digested coordinator result. Failed or unavailable completion accepts no raw result or error payload and derives only unchanged digest metadata plus the registered reason. qxctl does not classify an ambiguous process interruption as failure or unavailability; it preserves the intent for exact retry until typed terminal evidence exists.

## Safe Metadata and Exclusions

The reserved Named Version events permit the authenticated actor reference, a digest-bound target reference, request and correlation UUIDs, TOPS/TROG topology, `administrative_metadata`, and exact previous/new registry or proposal digests. They exclude Named Version bodies, aliases as free text, component inventories, contract contents, local paths, receipts, capabilities, authorization proofs, policy bodies, credentials, tokens, provider payloads, raw errors, and source documents.

Read-only status, lookup, validation, diff, Capsule checks, Blueprint plans, SCSEV assessment, graph projection, and routine health traffic remain outside this vocabulary.
