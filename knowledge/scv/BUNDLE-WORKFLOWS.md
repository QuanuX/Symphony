# SCV Explicit Bundle Workflows

Canonical owner companion, ratified September 12, 2026. Exact release `0.10.0-dev`; thirty existing native operations and eight supplied domains. New qxctl routes coordinate the existing bundle evaluator. Earlier packages, declarations, routes, defaults, records and journals remain preserved.

## Interface

`qxctl scv composition bundle workflow run|status|recover` uses a separate `composition-runs-v2` journal namespace. Run accepts the seven v1 workflow fields plus `protocol: symphony.qxctl.scv-composition-workflow-request.v2` and `transport: bundle`. Status/recover accept exactly `operation_id`. The selected original installation and complete request are sealed before owner work. Run/journal, intent and result protocols advance to v2. Existing pack stages retain their original provider owner and ordinary bounds; exploration and reassessment use bundled evaluation under the pinned selected owner. No size-triggered switching or fallback occurs.

`qxctl scv composition bundle obligations retain|show` retains v2 immutable links. Retain accepts exactly `protocol: symphony.qxctl.scv-obligation-link-request.v2`, `transport: bundle`, `before_ref`, `after_ref`, `submissions`. Show accepts exactly `link_ref`. The references are immutable record digests. These v2 contracts explicitly admit an ordinary exploration record or a bundled evaluation of exploration. Other logical operations and transport-only inspection are rejected.

## Identity and replay

A derived logical descriptor contains `record_ref`, `logical_operation`, `logical_protocol`, `logical_digest`, `artifact_digest`, and `transport`. Transport is null for direct results or `{result_bundle_digest,result_root_digest}` for bundled results. These represent separate subjects: complete local record, native result self-seal, stored owner-output self-seal, transport self-seal, and complete logical content hash. Equal logical results may retain different valid provenance records. Descriptors are derived and checked against their referenced records, never trusted as caller assertions.

The v2 link contains protocol, root, tops_id, transport, before, after, followup and submissions_digest. Before/after/followup are derived descriptors. Its sealed payload remains immutable in the existing store. The follow-up record is published before the link; exact retries recover publication interruptions. Expanded follow-up inputs must match the selected before/after artifacts and submissions digest. No relationship hash authenticates its author or proves a deployment.

Run/recover and link retain/show replay every referenced record through its original installation. Status requires no executable: it verifies sealed records, complete bundle closure, derived descriptors and exact reconstruction of checkpoint inputs, while reporting `sealed_checkpoints_only`. It does not claim native replay. New raw input/journal/reference paths reject malformed Unicode before generic decoding.

## Bounds and caller authority

Existing one-MiB process requests, four-MiB results, 32,768 logical values and depth 64 remain unchanged. Complete bundles retain their object/reference/traversal/materialization bounds; fully wrapped transport and record/journal limits are checked. Pack stages are outside the bundle operation allowlist and fail explicitly if oversized. No evidence or fixtures are clipped. Requirements, recipes, selections, policy, guarantees and counterfactuals retain their native fixed-problem checks; explicit query time may change. Permissions and execution remain caller-owned.

## Ownership and operating scope

SCV retains native composition and obligation meanings. qxctl owns exact local coordination, reference admission, checkpoint publication and recovery. This companion owns `scv-bundle-workflow.schema.json` and `scv-bundle-obligation-link.schema.json`. The `.10` catalog includes 112 protocol definitions in 29 schemas and twelve owner companions. Existing `.9` bundle inputs additionally remain explicitly admitted by the independent consumer; unsupported future releases are rejected.

A reusable logical resolver does not merge semantic owners. SHV may adopt these mechanics through a separately scoped contract. Neither this local journal nor a graph link establishes an authoritative state head, provenance authentication, source completeness, account compatibility or runtime success. No ambient lookup, source acquisition or provider action occurs.
