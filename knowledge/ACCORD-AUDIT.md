# Accordare Audit Boundary

## Purpose

This common knowledge contract connects SAV and SEV operational meaning to STAV without transferring authority between vectors. `knowledge/stav/` owns the event envelope and producer-vocabulary shape; the owning vector defines what one registered operation means; SSIAG owns authorization decisions; and an explicitly reviewed runtime producer must construct and submit safe candidates.

## Current State

The first closed vocabulary is `knowledge/stav/registries/v1/accordare.json`. It reserves four SAV Named Version lifecycle tuples. It explicitly reports `runtime_producer_implemented: false` and `producer_grant_authorized: false`. Those fields are architectural gates, not configuration switches. No installation may infer a grant, append authority, or audit completion from the presence or validity of this file.

SEV currently has no registered producer tuple. Its report-only assessments, novelty checks, graphs, and plans do not justify routine STAV traffic. A later mutation or export operation must pass a separate safe-metadata and producer-boundary review before receiving vocabulary.

## Producer Requirements

An operational Accordare producer must be a separately authenticated runtime boundary that:

- accepts only typed operation results and immutable evidence identities;
- verifies the exact installed component, result protocol, operation, TOPS, request, correlation, actor, expected state, and resulting state;
- maps only the closed vocabulary and never accepts caller-supplied event classes, operation identifiers, intents, reasons, producer identity, event identity, sequence, timestamp, or chain state;
- submits only safe references and previous/new digests through mutually authenticated local STAV IPC;
- requires a committed receipt when an operation is governed by STAV-before-commit;
- cannot reinterpret SSIAG permission, mutate canonical knowledge, or edit a STAV ledger.

qxctl may orchestrate the operation and validate the receipt, but it must never receive arbitrary append authority. SAV and SEV remain read-only C++ engines. The Knowledge Session Coordinator remains the Named Version persistence owner but does not gain STAV transport merely because it produced a result.

## Safe Metadata and Exclusions

The reserved Named Version events permit the authenticated actor reference, a digest-bound target reference, request and correlation UUIDs, TOPS/TROG topology, `administrative_metadata`, and exact previous/new registry or proposal digests. They exclude Named Version bodies, aliases as free text, component inventories, contract contents, local paths, receipts, capabilities, authorization proofs, policy bodies, credentials, tokens, provider payloads, raw errors, and source documents.

Read-only status, lookup, validation, diff, Capsule checks, Blueprint plans, SCSEV assessment, graph projection, and routine health traffic remain outside this vocabulary.
