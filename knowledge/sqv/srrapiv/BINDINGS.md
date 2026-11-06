# SRRAPIV exact artifact and strategy binding contract

Status: implemented pure C++ validation and inspection. Owner: SRRAPIV. Schema revisions: `symphony.srrapiv.catalog.v1`, `symphony.srrapiv.binding.v1`. These operations do not install artifacts, inspect their binary bytes, authorize provider activity, authenticate evidence, or establish live qualification.

SRRAPIV is a sovereign SQV child alongside SOOV and SCABV. Its binding semantics are not delegated to either connection owner. The native implementation is `modules/srrapiv-runtime-cpp/src/catalog.cpp`.

## Content identity

An artifact's `artifact_id` is `sha256:` followed by the lowercase SHA-256 of its compact JSON representation with `artifact_id` omitted. A binding's `binding_id` uses the same rule with `binding_id` omitted. JSON object keys are sorted lexicographically by the native nlohmann JSON representation. These schemas contain no JSON floating-point numbers. Array order remains part of content identity; duplicate set-like entries are rejected. Hashing is integrity binding, not an authenticity signature.

The catalog is `{schema, artifacts}`. `schema` is `symphony.srrapiv.catalog.v1`; each artifact has exactly:

| Field | Contract |
|---|---|
| `artifact_id` | Content digest as defined above. |
| `product` | Exactly `rithmic-rapi-plus-cpp`. |
| `sdk_version` | Four decimal version components separated by periods. |
| `archive_digest`, `header_digest`, `library_digest` | Exact content references, lowercase tagged SHA-256. Actual bytes must be checked at installation/runtime boundaries. |
| `adapter_version`, `adapter_digest` | Exact SRRAPIV implementation revision and binary content reference. |
| `platform` | `{os, architecture, compiler, abi, dependency_digest}`. `os` is `linux` or `darwin`; architecture, compiler and ABI are explicit identities. Accepting a declared target is not vendor support or host qualification. |
| `contracts` | `{api_mapping, control, event, state, journal, state_reader, readable_state_schemas, readable_journal_schemas}`. Exact identities; reader arrays must include the artifact's own writer schemas. |
| `connection` | `{profile_digest, trust_digest}`; independently identified connection and trust material. |
| `capabilities` | Nonempty unique exact public-element/operation identities implemented by the declared artifact. Declaration is not verified dispatch coverage. |

A binding has exactly:

| Field | Contract |
|---|---|
| `schema` | `symphony.srrapiv.binding.v1`. |
| `binding_id` | Content digest as defined above. |
| `artifact_id` | Exact catalog artifact digest. Missing identities fail; a matching SDK label or newer artifact is never substituted. |
| `strategy_id`, `environment` | Explicit consumer identity and exact provider environment name. The environment name permits printable ASCII spaces, such as `Rithmic Test`; it is not normalized or treated as a namespace identifier. |
| `account_ref` | Opaque namespaced account reference. No account credential value. |
| `credential_reference` | Exactly `{provider_id, resource_id}` with opaque namespaced references. No plaintext credential, token or password fields. |
| `required_capabilities` | Nonempty unique exact requirements. Missing artifact capabilities remain visible and incompatible. |
| `required_evidence_scope` | Exactly `native_fixture`, `provider_test`, or `production`; evidence in one scope does not satisfy another. |
| `feed_profile_digest`, `metadata_digest` | Exact feed and instrument/units interpretation dependencies. |

Credential rotation at an unchanged provider/resource reference does not rewrite the SDK or binding. A changed reference is a new binding. Secret generations, delivery and revocation are private runtime concerns, outside this pure schema. Trust or native dependency changes create a new artifact identity and require new scoped evidence.

All keys are strict. Unknown keys are rejected without reflecting their names or values in error diagnostics. Identifier strings are bounded ASCII without whitespace or query syntax; opaque references contain a namespace colon and cannot be URLs. The bare aliases `latest`, `current`, `default` and `auto` are rejected in identity fields. This is schema validation, not a detector that can prove a user-supplied string contains no secret.

## Operations

`catalog_inspect` input is `{catalog, artifact_id?}`. Without a selector it returns deterministically ordered inventory; with a selector it returns only that exact artifact or `exact_artifact_not_found`. It reports `substitution_performed:false` and `artifact_bytes_verified:false`.

`binding_validate` input is `{catalog, binding, compatibility_evidence}`. A structurally valid result has `ok:true, valid:true`. `compatibility` is independently `supported`, `partial`, `unknown`, `incompatible` or `conflicted`. A local native host may be constructed from a valid exact binding while qualification remains unknown; provider operations require separate native authorization, readiness and observation checks.

Each supplied compatibility assertion has exactly `{binding_id, capability, result, evidence_id, scope, observed_at}`. `result` is `supported` or `incompatible`. It matches only the exact binding, required capability and requested evidence scope. Assertions in another scope or for another binding do not count. Conflicting matched assertions produce `conflicted`; absent evidence yields `unknown`; unavailable artifact capabilities are incompatible even if an assertion claims support.

The result preserves supporting references but always reports `assertions_verified:false`, `live_qualified:false`, and `provider_activity_authorized:false`. Input assertions are caller-supplied records, not authenticated vendor acceptance. `observed_at` is retained as a bounded identity string; no freshness or current-server acceptance is inferred. Account entitlement and support policy are separate external observations, not inferred from successful catalog validation.

`binding_compare` input is `{catalog, from_binding, to_binding, compatibility_evidence}`. Both bindings must validate and name the same strategy. It reports all changed artifact and binding paths, added/removed capabilities, separate qualification assessments, and directional state/journal readability. It does not mutate either binding.

`forward_state_readable` and `forward_journal_readable` ask whether the target reader declares support for the source writer schemas. `rollback_state_readable` and `rollback_journal_readable` ask the reverse. Therefore a newer reader can accept old state while the old reader cannot accept new state. `migration_required` concerns missing forward reader declarations; `rollback_reader_gap` concerns the reverse. These are declared compatibility results, not executed reader qualification, so `reader_claims_verified` is always false. A positive declaration is insufficient to authorize restoring arbitrary mutable state or replaying provider operations.

Failures return `{ok:false, operation, errors:[{code,path}]}`. Paths are schema locations, not supplied secret values. Pure operations never initiate network traffic, install software, resolve a moving alias or replay an order.

## Bounds and executable examples

One operation accepts at most 256 artifacts, 16,384 elements per capability/reader set, and 32,768 compatibility assertions. These are bounded request limits, not global installation limits. Paginated catalog composition can select bounded subsets without changing identities. Identifier strings are at most 256 bytes, and tagged digests exactly 71 bytes. The surrounding native request boundary separately bounds serialized request size.

`modules/srrapiv-runtime-cpp/tests/catalog_test.cpp --sample-input` emits a complete synthetic `binding_validate` input with correctly sealed artifact/binding identities. Its SDK hashes are fixture values and must never be represented as vendor package hashes or used as live deployment proof. The focused tests cover immutable selection, tuple changes, scope-bound evidence, conflicts, secret-field rejection, side-by-side versions and directional reader gaps.
