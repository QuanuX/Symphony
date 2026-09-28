# SQAV request engine specification

## Identity and scope

Contract v1; module/package `sqav-request-engine`, engine `symphony-sqav-request`, vector `sqav`, release `0.1.0-dev`, C++26, freezing path. The one operation is `request_validate` (`engop:symphony:sqav-request.validate`). It is read-only, idempotent and requires no authority.

## Exact dependency admission

Link `sqav-fred-cpp` 0.1.0-dev, `sqav-databento-dbn-cpp` 0.5.0-dev and `sqav-databento-reference-cpp` 0.1.0-dev with their pinned dependencies and knowledge-vector-engine-cpp 0.2.0-dev. These are libraries, not embedded process engines. Source and installed dependency builds expose the same operation.

## Input and native semantics

Use `symphony.knowledge.engine-process.v1`. Payload protocol `symphony.sqav.request-validation-input.v1` has exactly protocol, adapter, selection, limits. `schemas/v1/request.schema.json` describes the three tagged request structures. The native `Plan::create` implementations own all provider rules and plan references. The process performs shape, size and integer-width checks before conversion; Go never recomputes provider rules.

Adapters are `fred`, `databento_historical`, `databento_reference`. FRED supports observations and vintage_dates with explicit ALFRED real-time dates; reference supports corporate_actions, adjustment_factors, security_master_range and security_master_last. Historical timestamps are canonical unsigned decimal strings to preserve nanosecond values above the interoperable JSON integer range. Other numeric fields are nonnegative interoperable integers. No omitted selection defaults are applied. Templates have unanswered nulls and are not validated requests.

## Bounds, output and failures

Process input at most 65,536 bytes; payload at most 32,768 canonical bytes; standard foundation duplicate-key, integer, depth and value limits apply. Five-second maximum request deadline. Descriptor snapshot bounds are unused positive mechanical ceilings of one file/one byte; no snapshot input or file-reading operation is admitted. Result `symphony.sqav.request-validation.v1` binds adapter id/release, canonical request digest and native plan reference; result_digest covers the canonical result excluding itself. Fixed validation_scope `native_request_only` and provider_observation `not_performed` prevent conflating local validity with provider availability. Structured errors use fixed messages and never include caller selection or credentials. Invalid shape or native rejection exits 2; unsupported operation/deadline exits 3; internal unavailability exits 5.

## Installation and admission

Immutable receipt v2 owns executable, contracts, owner interface, schema/templates and licenses. qxctl selects explicit prefix and exact release, verifies all receipt files plus compiled interface and schema byte digests before use, and rechecks installation after invocation/resource reading. The existing registration-driven owner-interface format retains its `symphony.shv.*` wire name for compatibility; SQAV owns this module and operation. No operation aliases or provider-specific CLI trees are introduced.

## Extent

Validation does not fetch credentials, call a provider, acquire data, spend money, start a job, inspect a store, or mutate source/runtime state. A valid request does not establish authentication, entitlement, symbol resolution, cost, completeness or a successful acquisition. Live feeds, operational acquisition lifetime, SSIAG credential bridges and safe store observation remain later work. SCABV broker receipt semantics stay with their native owner.
