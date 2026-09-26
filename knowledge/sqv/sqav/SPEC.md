# Symphony Quantitative Acquisition Vector Specification

## Source Binding

An admitted collector identifies the exact provider surface, operation, endpoint or SDK and adapter release, source dataset, requested scope, supported target, authorized credential reference, and finite resource and provider limits. Unversioned provider interfaces require dated source evidence; an adapter must not silently replace an older binding with a newer one. Provider-required external applications are declared separately from the first-party C++ implementation.

Acquisition records distinguish provider event, publication, revision, and receipt times where evidenced from Symphony acquisition time. Their precision, calendar, and uncertainty remain explicit. Source order is not automatically Symphony transfer order or a total order across providers.

## Collection and Coverage

The selected collector defines finite requests or a bounded long-running session, its cursor scope, retry and reconnect policy, and supported backfill. Pagination, rate limits, partial responses, skipped records, replay expiry, revisions, and missing observations produce attributable coverage or gap evidence. A successful request or active socket alone does not prove a complete dataset.

Provider payloads retain their actual rights and access scope; collection does not make them public knowledge. Rights to collect, retain, transform, share, model, and redistribute are distinct where the source distinguishes them. Source output and downloaded metadata are untrusted data, not instructions. Routine payloads do not pass through qxctl, a knowledge query, or an audit service.

## Native and Integration Boundary

First-party collection, parsing, and provider-facing movement are native C++. Suitable native dependencies are explicit and versioned. IBKR research collection must consume a separately admitted SCABV-owned read-only provider binding; SQAV does not fabricate that adapter or a duplicate broker authority. FIX remains SOOV-owned. No listed source is operational until its exact connector and conformance evidence exist.

## Deferred Technical Contract

The first source operation, credential and access path, payload schema, parser ABI, restart semantics, target platform, throughput bound, and package lifecycle require separate admission. This Quad creates no installed capability or namespace family.

## Non-Authorization Statement

SQAV does not authorize a provider session, credential use, subscription, data purchase, account access, order entry, or redistribution.
