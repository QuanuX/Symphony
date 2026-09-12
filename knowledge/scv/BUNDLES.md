# SCV Exact Evidence Bundles

Canonical owner companion for exact `0.9.0-dev`, ratified September 12, 2026. SCV owns bounded, complete evidence transport and preserves the composition semantics in `COMPOSITION.md` and `OBLIGATIONS.md`. qxctl administers all routes under the caller-selected exact installation.

## Typed, complete bundle

Bundle fields exactly `{protocol,root_digest,objects,digest}`. Protocol `symphony.scv.evidence-bundle.v1`. The ordinary bundle self-seal excludes its own `digest`. `root_digest` and each object digest are SHA-256 of the **complete canonical expanded JSON value**, including any existing native self-digest field. They are transport content identities, distinct from native self-seals, body digests, local record IDs or receipts.

Every object and array is one content-addressed node. Node inventory `objects` is nonempty and strictly sorted by digest; identical content is stored once. Object node exactly `{digest,kind,members}` with kind `object` and `members` a JSON object whose arbitrary keys map to children. Array node exactly `{digest,kind,items}` with kind `array` and ordered child array. Child is exactly `{scalar:value}` for null/bool/integer/string (never a container), or `{ref:digest}` for a container node. There are no magic keys inside caller JSON: even an original object named `ref` is represented as ordinary object members. Empty objects and arrays are retained.

Transport reference cycles are rejected because reconstruction must yield a finite JSON value. User-authored graph-edge IDs and cyclic provider or application topologies remain ordinary literal data with their existing semantics.

Root resolves to an object. Reject missing nodes, duplicate or unsorted IDs, unreachable nodes, cycles, digest mismatches, unknown/extra/missing fields and wrong scalar/container types. Common duplicate-key, UTF-8, integer-range, string and depth checks remain active. Canonical integers use decimal form, normalize negative zero to zero, and retain the shared interoperable range from -9,007,199,254,740,991 through 9,007,199,254,740,991; preserve arbitrary source-string bytes. Tests must cover integer extrema, negative zero handling, Unicode line separators and literal reference-looking data.

Limits: unchanged native wire request 1MiB and response 4MiB, 32,768 JSON values and depth64. New logical expansion cap4MiB,32,768 values (keys count),depth64; at most2048 unique objects,8192 stored reference edges,16384 graph traversal steps (objects+stored reference edges),64MiB cumulative canonical expanded bytes across unique nodes. Enforce checked bottom-up size/value/depth/fanout arithmetic BEFORE materializing expanded values, then verify exact hashes. Check native deadline during graph work, hashing and packing. Every selected node is reachable; graph sharing does not exempt repeated occurrences from the logical-root bounds. These new logical limits apply only behind the explicit new bundle operation.

Metrics exactly `{object_count,reference_count,traversal_steps,expanded_bytes,expanded_values,expanded_depth,materialized_bytes}`. Byte counts are canonical UTF-8 JSON bytes with no trailing newline; root depth0. Materialized bytes sum canonical expanded sizes of unique nodes. Count stored reference edges once per stored node occurrence. The materialization budget prevents nested nodes from accumulating unbounded hashing/storage work.

## Native operations

Allowlisted logical operations: `composition_explore`, `composition_reassess`, `composition_obligations`, `composition_followup`. No nested bundle dispatch or other operations. Both new operations accept the exact same input: `{operation,owner,bundle}`; owner exactly `{domain,version}`. Owner must match the invoked C++ domain and compiled version `.9`. This pins the evaluator identity, not the producer of every legitimately scoped subordinate provider fact. Historical producer installations remain separately bound by local retained records; bundle digests do not authenticate them.

`bundle_inspect`: validate owner/allowlist/complete closure and reconstruct it. Result exactly `{protocol,domain,operation,owner,input_digest,root_digest,metrics,validation,limitations,digest}`, protocol `symphony.scv.bundle-inspection.v1`; input_digest is full canonical SHA of the complete invocation input; validation `transport_only`. No semantic compatibility claim.

`composition_bundle_evaluate`: validate/reconstruct input, copy existing Request retaining target/deadline, replace only operation/payload, call unchanged handle_composition. Rebundle the full native result and validate its expansion budgets. Result exactly `{protocol,domain,operation,owner,input_digest,input_root_digest,input_metrics,result_bundle,native_result_digest,result_metrics,validation,limitations,digest}`. Protocol `symphony.scv.composition-bundle-evaluation.v1`; input_digest full canonical invocation hash, input_root_digest full expanded-input hash; native_result_digest is existing logical result's own native digest; result_bundle root_digest is its distinct full-value content hash. Validation `owner_evaluated`. Do not embed the expanded input/result or duplicate the whole request bundle in this wrapper. Ordinary result self-seal excludes `digest`.

Go independently checks bundle reconstruction/digests/metrics and exact result shape, operation/owner/input correspondence. For evaluate, reconstruct result_bundle and invoke the existing independent logical operation validator with the expanded input and result. Do not change existing scvObject or legacy InvokeSCVDomain/process limits. Codec shape/self-seal alone cannot count as native semantic validation.

Fixed limitations (same text/order in native and independent consumer):

Inspection: `["bundle validation establishes exact bounded reconstruction only; it does not validate composition semantics or authenticate evidence producers", "all dependencies are explicit and complete; no ambient source, graph, filesystem or network lookup occurs"]`.

Evaluation: `["bundled execution preserves the existing composition owner's meanings and caller criteria; it does not certify deployment or authenticate submitted provenance", "content references reduce repeated transport; expanded values and graph work remain explicitly bounded", "historical installation identity belongs to retained owner records; a transport content digest is not a receipt"]`.

## qxctl routes and retention

Group `qxctl scv composition bundle pack|inspect|evaluate|expand`, default exact `.9`.

Pack reads an explicit no-follow file with exact `{operation,input}` using a separate bounded local input reader (total file<=4MiB and envelope<=33,024 values, logical input<=4MiB/32768/depth64). It mechanically encodes input, creates owner from exact selected installation options, invokes native bundle_inspect and validates the result before outputting the ready-to-evaluate `{operation,owner,bundle}` payload. This output is transport input, not a claim of semantic evaluation.

Inspect invokes bundle_inspect with the packed payload. Evaluate invokes composition_bundle_evaluate and returns its bundled native result. Expand accepts the same packed payload, performs the same native evaluation and independent validation, then emits the full original logical native result reconstructed from result_bundle. This is an explicit expanded view of a new bundle invocation, not a historical legacy process invocation. Its semantic result protocol remains the selected operation's existing protocol.

The existing artifact import/show/list retains the two new native operations under new exact kinds `bundle_inspection` and `composition_bundle_evaluation`; store new-operation input/result without expanding them into a legacy record whose input would violate1MiB. Replay uses the exact recorded `.9` installation. Retained records must roundtrip to their exact typed canonical representation; case aliases or missing installation fields cannot change the meaning of sealed bytes. All bundle input routes reject invalid Unicode spellings before a generic decoder can normalize them. No new storage namespace, journal, selected head or database adapter is required.

New schema file `bundle.schema.json`; definitions Bundle, InvocationInput, InspectionResult, EvaluationResult, PackInput and referenced helpers. Six catalog entries: evidence-bundle.v1, bundle-inspect-input.v1, composition-bundle-evaluate-input.v1, bundle-inspection.v1, composition-bundle-evaluation.v1 (all symphony.scv), and symphony.qxctl.scv.bundle-pack-input.v1. Two native ops bring current release to30; one schema brings27; six catalogentries bring103; BUNDLES.md is11th companion; four leaves bring255total/57SCV.

## Operating scope

This release operates on complete explicit local evidence. Content identities establish exact reconstruction; semantic replay establishes only the selected operation's existing findings. Neither establishes a producer's authority, a successful deployment or universal provider compatibility. A bundle is portable without an ambient filesystem, network, graph database or selected state head. Local artifact records separately preserve the original installation and its receipt-owned resources.

References here are an internal transport representation. User-authored provider, recipe and topology choices retain their existing meanings. Other vectors may adopt these tested mechanics through an explicit contract; this companion does not assign them SCV semantics or impose a universal graph ontology.
