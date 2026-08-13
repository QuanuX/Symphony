# Symphony Semantic Feature Vector Registry

## Status

Canonical SSFV feature-routing registry. The partial catalog routes sixty-nine experimental records across the repository root and fourteen implemented owner scopes enumerated by `COVERAGE.md`.

## Purpose

Map each stable SSFV feature identity to one canonical distributed owner record without centralizing or duplicating feature semantics.

## Entry Model

Each entry MUST provide, in this exact order:

- `feature_id`: stable SSFV identity;
- `feature_file`: repository-relative canonical `FEATURES.md`;
- `owner_contract`: repository-relative owning Contract Quad file;
- `source_scope`: explicit repository-relative source scope;
- `status`: `experimental`, `implemented`, `deprecated`, or `retired`;
- `parent_feature_id`: stable SSFV identity or `none`;
- `record_digest`: lowercase `sha256:` digest of the normalized feature record;
- `notes`: safe routing context.

## Canonical Markdown Grammar

Each entry is one contiguous ordered block using Markdown list items in `- field: value` form. Outer backticks are presentation delimiters only. Duplicate, unknown, missing, empty, or reordered fields fail validation.

`feature_id` MUST be globally unique. One source scope maps to exactly one `feature_file` plus `owner_contract` routing tuple, and several feature IDs MAY share that tuple. A feature identity appears in exactly one owner file. Paths MUST be normalized, repository-relative, and no-follow. The exact literal `.` represents repository-root source scope and owns root `FEATURES.md`; every other source scope owns `<source_scope>/FEATURES.md`. Every registered file MUST also be indexed by SKVI.

The literal `None.` beneath `## Canonical Entries` is the only valid empty-registry representation. It is removed atomically with the first ratified entry and MUST NOT coexist with entry blocks.

## Canonical Entries

- feature_id: `ssfv:symphony:knowledge-session-coordinator`
- feature_file: `modules/knowledge-session-coordinator/FEATURES.md`
- owner_contract: `modules/knowledge-session-coordinator/SPEC.md`
- source_scope: `modules/knowledge-session-coordinator`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:platform`
- record_digest: `sha256:1daaf8ad5bbdd6f736422dceaa90b58b1eeeea40b1ff844a6ea94db549eb7d29`
- notes: Partial-catalog record for durable reconciliation, SSIAG-authorized authenticated sessions, persistent SSFV baseline maintenance, explicit idempotent qxctl host-event convergence, protected desired-profile and observation administration, dependency planning, report/apply journal recovery, and serialized externally executed lifecycle actions including authenticated Maestro presence.

- feature_id: `ssfv:symphony:knowledge-session-coordinator.authority-epochs`
- feature_file: `modules/knowledge-session-coordinator/FEATURES.md`
- owner_contract: `modules/knowledge-session-coordinator/SPEC.md`
- source_scope: `modules/knowledge-session-coordinator`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:knowledge-session-coordinator`
- record_digest: `sha256:ee5c2ccf2726c0ddd8244bc2649583b6cb1c33b5e4874bc9fcd294dc37a95c93`
- notes: Architect-ratified F1 nested record for Durable authenticated authority epochs; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:knowledge-session-coordinator.lifecycle-apply-coordination`
- feature_file: `modules/knowledge-session-coordinator/FEATURES.md`
- owner_contract: `modules/knowledge-session-coordinator/SPEC.md`
- source_scope: `modules/knowledge-session-coordinator`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:knowledge-session-coordinator`
- record_digest: `sha256:d3e7e843ce548ff616e6e3949d32b2f5c9e708013450cb4db27f38765466e10e`
- notes: Architect-ratified F1 nested record for Durable lifecycle apply coordination; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:knowledge-session-coordinator.lifecycle-planning`
- feature_file: `modules/knowledge-session-coordinator/FEATURES.md`
- owner_contract: `modules/knowledge-session-coordinator/SPEC.md`
- source_scope: `modules/knowledge-session-coordinator`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:knowledge-session-coordinator`
- record_digest: `sha256:8cdbd404474d2347a6707dd7646a99da7e036d5b498158e61883cb3092376192`
- notes: Architect-ratified F1 nested record for Deterministic two-way lifecycle planning; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:knowledge-session-coordinator.reconciliation`
- feature_file: `modules/knowledge-session-coordinator/FEATURES.md`
- owner_contract: `modules/knowledge-session-coordinator/SPEC.md`
- source_scope: `modules/knowledge-session-coordinator`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:knowledge-session-coordinator`
- record_digest: `sha256:e9fdbabf69f87094cf927d43de8a3e4842597ea4735280d254b69640ac29a2df`
- notes: Architect-ratified F1 nested record for Durable worktree reconciliation contexts; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:knowledge-session-coordinator.semantic-maintenance`
- feature_file: `modules/knowledge-session-coordinator/FEATURES.md`
- owner_contract: `modules/knowledge-session-coordinator/SPEC.md`
- source_scope: `modules/knowledge-session-coordinator`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:knowledge-session-coordinator`
- record_digest: `sha256:023e39ed455bb77d8584094004fff11b1eb9f1b4aeddc101224bcb4502d42a45`
- notes: Architect-ratified F1 nested record for Persistent SSFV semantic-maintenance lineage; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:knowledge-vector-engine-foundation`
- feature_file: `libraries/knowledge-vector-engine-cpp/FEATURES.md`
- owner_contract: `libraries/knowledge-vector-engine-cpp/SPEC.md`
- source_scope: `libraries/knowledge-vector-engine-cpp`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:platform`
- record_digest: `sha256:9d5669b210fc3be63d1eae2ccf7bad377df79a9579d2377b18f4e5daf3b4c409`
- notes: First partial bootstrap record for the implemented authority-free shared C++ mechanics, including canonical STSC Gregorian and UTC representation validation.

- feature_id: `ssfv:symphony:knowledge-vector-engine-foundation.bounded-process-protocol`
- feature_file: `libraries/knowledge-vector-engine-cpp/FEATURES.md`
- owner_contract: `libraries/knowledge-vector-engine-cpp/SPEC.md`
- source_scope: `libraries/knowledge-vector-engine-cpp`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:knowledge-vector-engine-foundation`
- record_digest: `sha256:a9b520d54e4c82877c4529cba9d812dc2e15ff7da6375cc90615c616731ced88`
- notes: Architect-ratified F3 nested record for bounded deterministic engine-process protocol behavior; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:knowledge-vector-engine-foundation.content-addressed-evidence-snapshots`
- feature_file: `libraries/knowledge-vector-engine-cpp/FEATURES.md`
- owner_contract: `libraries/knowledge-vector-engine-cpp/SPEC.md`
- source_scope: `libraries/knowledge-vector-engine-cpp`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:knowledge-vector-engine-foundation`
- record_digest: `sha256:f8f78501ea292af6cc8f98819c072015920f19b71701e8d79865eecb23650e69`
- notes: Architect-ratified F3 nested record for no-follow content-addressed evidence snapshots; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:knowledge-vector-engine-foundation.temporal-representation-conformance`
- feature_file: `libraries/knowledge-vector-engine-cpp/FEATURES.md`
- owner_contract: `libraries/knowledge-vector-engine-cpp/SPEC.md`
- source_scope: `libraries/knowledge-vector-engine-cpp`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:knowledge-vector-engine-foundation`
- record_digest: `sha256:dd17771aaf383b03a26c60b5b9de7f3488bccbe6d6dca644ad9cb1fc988fc1eb`
- notes: Architect-ratified F3 nested record for canonical temporal representation conformance; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:maestro-presence-authority`
- feature_file: `modules/maestro/FEATURES.md`
- owner_contract: `modules/maestro/SPEC.md`
- source_scope: `modules/maestro`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:platform`
- record_digest: `sha256:89076bde7da3ddc19c185e1c05025e5d822c5e31ab6e299d6c4c6df5e86c5084`
- notes: Partial-catalog record for exact authenticated per-TOPS/per-receptor docking presence, complete read-only derived inventory, lifecycle integration, semantic retry, and evidence-preserving recovery without engine execution.

- feature_id: `ssfv:symphony:maestro-presence-authority.complete-inventory`
- feature_file: `modules/maestro/FEATURES.md`
- owner_contract: `modules/maestro/SPEC.md`
- source_scope: `modules/maestro`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:maestro-presence-authority`
- record_digest: `sha256:3b2d52f891adfd29b8908cae7304d3ed20114c03b52432ece8dc4a26445311f2`
- notes: Architect-ratified F1 nested record for Complete derived Maestro receptor inventory; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:platform`
- feature_file: `FEATURES.md`
- owner_contract: `INTENT.md`
- source_scope: `.`
- status: `experimental`
- parent_feature_id: `none`
- record_digest: `sha256:700f02746603ccebdb94711291414c5f840b5c28039dc525dc9b0837f041c0fd`
- notes: Repository-root capability record; bootstrap coverage is explicitly partial and does not imply production readiness or complete catalog coverage.

- feature_id: `ssfv:symphony:qxctl`
- feature_file: `tools/qxctl/FEATURES.md`
- owner_contract: `tools/qxctl/MANIFEST.md`
- source_scope: `tools/qxctl`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:platform`
- record_digest: `sha256:a92f2ae12c125be4ba43506586ace43906fbdc5b45075f3ef222976dba31f403`
- notes: Partial-catalog record for the Go Cobra/Viper administrative and query surface, including the explicit Linux report-only lifecycle receptor, across independently installed Symphony modules.

- feature_id: `ssfv:symphony:qxctl.authenticated-sessions`
- feature_file: `tools/qxctl/FEATURES.md`
- owner_contract: `tools/qxctl/MANIFEST.md`
- source_scope: `tools/qxctl`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:qxctl`
- record_digest: `sha256:1849a565bec4f523feef072dd22afdeec3a46934cec827adc852e250427200ae`
- notes: Architect-ratified F1 nested record for Authenticated knowledge-session administration; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:qxctl.command-registry`
- feature_file: `tools/qxctl/FEATURES.md`
- owner_contract: `tools/qxctl/MANIFEST.md`
- source_scope: `tools/qxctl`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:qxctl`
- record_digest: `sha256:3bc82d569b3020164cb42dee284138b84d2c53201d0c4c6bd67e88cfcb3537f9`
- notes: Architect-ratified command identity and coverage registry for deterministic headless administration evidence; coverage remains partial and no dynamic command or authority claim is implied.

- feature_id: `ssfv:symphony:qxctl.engine-bindings`
- feature_file: `tools/qxctl/FEATURES.md`
- owner_contract: `tools/qxctl/MANIFEST.md`
- source_scope: `tools/qxctl`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:qxctl`
- record_digest: `sha256:0a44e8d17d8914cc468a957d59e34bb1d77d1873964b60cf56d50445b5070b40`
- notes: Architect-ratified F1 nested record for Exact installed knowledge-engine bindings; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:qxctl.governed-validation`
- feature_file: `tools/qxctl/FEATURES.md`
- owner_contract: `tools/qxctl/MANIFEST.md`
- source_scope: `tools/qxctl`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:qxctl`
- record_digest: `sha256:d2bba969301b203f04186fa2f4996e25270bbf0f13c17b71af4d1853d8a0ba1b`
- notes: Architect-ratified F1 nested record for Governed repository validation evidence; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:qxctl.lifecycle-convergence`
- feature_file: `tools/qxctl/FEATURES.md`
- owner_contract: `tools/qxctl/MANIFEST.md`
- source_scope: `tools/qxctl`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:qxctl`
- record_digest: `sha256:d61cfc990fd6c67ac9536b611897faabe8bc7bdcaacbc33bfc3b299e09cdb32c`
- notes: Architect-ratified F1 nested record for Two-way module lifecycle convergence; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:qxctl.linux-host-receptor`
- feature_file: `tools/qxctl/FEATURES.md`
- owner_contract: `tools/qxctl/MANIFEST.md`
- source_scope: `tools/qxctl`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:qxctl`
- record_digest: `sha256:53b41299a29bf0260f87c93ab183f3e5b36dca3576ca5474a7ebb264ba57ec2f`
- notes: Architect-ratified F1 nested record for Explicit Linux report-only boot receptor; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:qxctl.maestro-administration`
- feature_file: `tools/qxctl/FEATURES.md`
- owner_contract: `tools/qxctl/MANIFEST.md`
- source_scope: `tools/qxctl`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:qxctl`
- record_digest: `sha256:68202bfea27e4bdf2dd578624ef4bfa1e7fec8549a055ef8e9b3facb1109794a`
- notes: Architect-ratified F1 nested record for Authenticated Maestro receptor administration; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:qxctl.ssiag-administration`
- feature_file: `tools/qxctl/FEATURES.md`
- owner_contract: `tools/qxctl/MANIFEST.md`
- source_scope: `tools/qxctl`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:qxctl`
- record_digest: `sha256:c8d29dec41692eb76c46a61d3a1fe7cafa4f4e1a6a925c8d0c4a3bda5892b4f5`
- notes: Architect-ratified F1 nested record for SSIAG policy, decision, enrollment, and native-supervision administration; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:qxctl.stav-administration`
- feature_file: `tools/qxctl/FEATURES.md`
- owner_contract: `tools/qxctl/MANIFEST.md`
- source_scope: `tools/qxctl`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:qxctl`
- record_digest: `sha256:8a5d5fa59eb2c3994722ec58699f1619e73e323534d24ab653defebfd1b882b2`
- notes: Architect-ratified F1 nested record for STAV audit, enrollment, and native-supervision administration; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:sacv-engine`
- feature_file: `modules/sacv-engine/FEATURES.md`
- owner_contract: `modules/sacv-engine/SPEC.md`
- source_scope: `modules/sacv-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:platform`
- record_digest: `sha256:c9e2817a941b4c6e36943ea78d510141d64fb2c13df6c8125b8c4337d2fdf94c`
- notes: Partial-catalog record for bounded read-only SACV API-contract governance behavior.

- feature_id: `ssfv:symphony:sacv-engine.api-contract-conformance`
- feature_file: `modules/sacv-engine/FEATURES.md`
- owner_contract: `modules/sacv-engine/SPEC.md`
- source_scope: `modules/sacv-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:sacv-engine`
- record_digest: `sha256:8d425abfcf727a291f71e2c4bee05db30f1cd89fa1409324bdeeb60c3f220a42`
- notes: Architect-ratified F3 nested record for registered OpenAPI contract conformance; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:sacv-engine.contract-inventory-projection`
- feature_file: `modules/sacv-engine/FEATURES.md`
- owner_contract: `modules/sacv-engine/SPEC.md`
- source_scope: `modules/sacv-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:sacv-engine`
- record_digest: `sha256:ea0c7675b59fb152538cc476f0a467187570ac4dbbdd4147eb99a184fdc72e57`
- notes: Architect-ratified F3 nested record for a rebuildable API contract inventory; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:sacv-engine.contract-registration-proposal`
- feature_file: `modules/sacv-engine/FEATURES.md`
- owner_contract: `modules/sacv-engine/SPEC.md`
- source_scope: `modules/sacv-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:sacv-engine`
- record_digest: `sha256:042c13f44a424ce8197db9208c7fd49bac7f2b875be44c92dbd33b6ccbd4ea46`
- notes: Architect-ratified F3 nested record for content-addressed API registration proposals; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:sacv-engine.openapi-compatibility-evidence`
- feature_file: `modules/sacv-engine/FEATURES.md`
- owner_contract: `modules/sacv-engine/SPEC.md`
- source_scope: `modules/sacv-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:sacv-engine`
- record_digest: `sha256:7e50a78ac6d732c6b2fbfae9402c97eea487e1a59c7c377aa2aa736aebe65b6c`
- notes: Architect-ratified F3 nested record for bounded OpenAPI compatibility evidence; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:sclv-engine`
- feature_file: `modules/sclv-engine/FEATURES.md`
- owner_contract: `modules/sclv-engine/SPEC.md`
- source_scope: `modules/sclv-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:platform`
- record_digest: `sha256:a176d14b53adab08034965e2a3cbe5f279cefc31bcf988a30ef0d8058080e5a1`
- notes: Partial-catalog record for bounded read-only SCLV change-truth governance behavior.

- feature_id: `ssfv:symphony:sclv-engine.append-only-ledger-assurance`
- feature_file: `modules/sclv-engine/FEATURES.md`
- owner_contract: `modules/sclv-engine/SPEC.md`
- source_scope: `modules/sclv-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:sclv-engine`
- record_digest: `sha256:52e3c1cf4c3145a72bb17b375842369262fc907a8a2a78116be0ff9e2192b35b`
- notes: Architect-ratified F3 nested record for append-only change-ledger assurance; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:sclv-engine.disposable-provider-neutral-history`
- feature_file: `modules/sclv-engine/FEATURES.md`
- owner_contract: `modules/sclv-engine/SPEC.md`
- source_scope: `modules/sclv-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:sclv-engine`
- record_digest: `sha256:9b25a1a1cdf8865fb9ee9e0ee2e750206fcea35d75a730a07685049d4ac74fff`
- notes: Architect-ratified F3 nested record for a disposable provider-neutral change-history projection; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:sclv-engine.evidence-bound-append-proposals`
- feature_file: `modules/sclv-engine/FEATURES.md`
- owner_contract: `modules/sclv-engine/SPEC.md`
- source_scope: `modules/sclv-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:sclv-engine`
- record_digest: `sha256:595f2ccf0bb49d624931a559f3440e022066a3b67c7ead9b52fa55bda5981496`
- notes: Architect-ratified F3 nested record for evidence-bound SCLV append proposals; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:sclv-engine.forward-only-closure-recovery`
- feature_file: `modules/sclv-engine/FEATURES.md`
- owner_contract: `modules/sclv-engine/SPEC.md`
- source_scope: `modules/sclv-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:sclv-engine`
- record_digest: `sha256:64a6aa753c4ae9bee4f103566f83938b6de197ed846c1e470e2adec0272bfb00`
- notes: Architect-ratified F3 nested record for forward-only interrupted-closure recovery; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:sclv-engine.provider-neutral-evidence-normalization`
- feature_file: `modules/sclv-engine/FEATURES.md`
- owner_contract: `modules/sclv-engine/SPEC.md`
- source_scope: `modules/sclv-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:sclv-engine`
- record_digest: `sha256:fa19e3b49c6c34cd552da01b1b163dce6e46d24cb3c4bf90cf0f78dc71dffe5d`
- notes: Architect-ratified F3 nested record for provider-neutral local-Git and air-gap evidence normalization; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:skvi-engine`
- feature_file: `modules/skvi-engine/FEATURES.md`
- owner_contract: `modules/skvi-engine/SPEC.md`
- source_scope: `modules/skvi-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:platform`
- record_digest: `sha256:cb89dfa3ad4ec4400d2e3ef1b108cbccb102f41c0ee613aac312553d2daf18b2`
- notes: Partial-catalog record for bounded read-only SKVI routing governance behavior.

- feature_id: `ssfv:symphony:skvi-engine.content-addressed-index-change-proposals`
- feature_file: `modules/skvi-engine/FEATURES.md`
- owner_contract: `modules/skvi-engine/SPEC.md`
- source_scope: `modules/skvi-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:skvi-engine`
- record_digest: `sha256:729e7ed87d7945b93be320e45907ca78576318e23f727d73f1ff876c2fb26cbf`
- notes: Architect-ratified F3 nested record for content-addressed SKVI change proposals; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:skvi-engine.disposable-structural-projection`
- feature_file: `modules/skvi-engine/FEATURES.md`
- owner_contract: `modules/skvi-engine/SPEC.md`
- source_scope: `modules/skvi-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:skvi-engine`
- record_digest: `sha256:9cb53a42eae790bb36c1f06fe7588814edf982d5b9480659c6cec66080803b11`
- notes: Architect-ratified F3 nested record for a disposable SKVI structural projection; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:skvi-engine.structural-index-assurance`
- feature_file: `modules/skvi-engine/FEATURES.md`
- owner_contract: `modules/skvi-engine/SPEC.md`
- source_scope: `modules/skvi-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:skvi-engine`
- record_digest: `sha256:09c351ecc7d218aeee76b3f9369c98c65ceee20878a1cb57a752dd3b94bc30c3`
- notes: Architect-ratified F3 nested record for canonical index structural assurance; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:sodv-engine`
- feature_file: `modules/sodv-engine/FEATURES.md`
- owner_contract: `modules/sodv-engine/SPEC.md`
- source_scope: `modules/sodv-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:platform`
- record_digest: `sha256:441de39e7b84f7315ebc37c0cc67c3736679b867c05bc0510f55a4aedbdefde7`
- notes: Partial-catalog record for bounded read-only SODV release-publication governance behavior.

- feature_id: `ssfv:symphony:sodv-engine.forward-release-record-proposal`
- feature_file: `modules/sodv-engine/FEATURES.md`
- owner_contract: `modules/sodv-engine/SPEC.md`
- source_scope: `modules/sodv-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:sodv-engine`
- record_digest: `sha256:b797ba4812fba1de31d3469336cc401cd8c9ea807a4353c68c11413817122e66`
- notes: Architect-ratified F3 nested record for forward-only release-record proposals; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:sodv-engine.interrupted-publication-reconciliation`
- feature_file: `modules/sodv-engine/FEATURES.md`
- owner_contract: `modules/sodv-engine/SPEC.md`
- source_scope: `modules/sodv-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:sodv-engine`
- record_digest: `sha256:076668f893a27bee4e2ec9c1022340c4a5413a73189de5ed035b82a9ffb4e90c`
- notes: Architect-ratified F3 nested record for interrupted publication reconciliation; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:sodv-engine.observed-publication-verification`
- feature_file: `modules/sodv-engine/FEATURES.md`
- owner_contract: `modules/sodv-engine/SPEC.md`
- source_scope: `modules/sodv-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:sodv-engine`
- record_digest: `sha256:7f265627e98145308be5f3b7ebef8025f0bffab01a07749b4e1a86fee564e427`
- notes: Architect-ratified F3 nested record for caller-observed publication verification; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:sodv-engine.release-ledger-validation`
- feature_file: `modules/sodv-engine/FEATURES.md`
- owner_contract: `modules/sodv-engine/SPEC.md`
- source_scope: `modules/sodv-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:sodv-engine`
- record_digest: `sha256:3afc15b5952a1e5f3026495180761ac1b7d5846f13adfc3848b146956e330260`
- notes: Architect-ratified F3 nested record for append-only release-ledger validation; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:sodv-engine.release-transaction-projection`
- feature_file: `modules/sodv-engine/FEATURES.md`
- owner_contract: `modules/sodv-engine/SPEC.md`
- source_scope: `modules/sodv-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:sodv-engine`
- record_digest: `sha256:4b4f58ad687266e33feb00ca7c0a97a9488dd4ac8c0ee5e88dc597e1c2552ea7`
- notes: Architect-ratified F3 nested record for a rebuildable release-transaction projection; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:ssfv-engine`
- feature_file: `modules/ssfv-engine/FEATURES.md`
- owner_contract: `modules/ssfv-engine/SPEC.md`
- source_scope: `modules/ssfv-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:platform`
- record_digest: `sha256:e7faae1c0008cbd373a0a9786a0ebc8cdf3b47c3460d40b273caff1c58c5c0f5`
- notes: Partial-catalog record for bounded read-only SSFV semantic-feature governance behavior.

- feature_id: `ssfv:symphony:ssfv-engine.administration-assurance`
- feature_file: `modules/ssfv-engine/FEATURES.md`
- owner_contract: `modules/ssfv-engine/SPEC.md`
- source_scope: `modules/ssfv-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:ssfv-engine`
- record_digest: `sha256:aaa2f34e07d9cb6eecd6a146468bae4b40585c6193f6b3747f657e7e9f492e2d`
- notes: Architect-ratified engine-first feature-administration coverage and independent-module integration assessment; coverage remains partial and docking readiness grants no docking authority.

- feature_id: `ssfv:symphony:ssfv-engine.catalog-change-proposal`
- feature_file: `modules/ssfv-engine/FEATURES.md`
- owner_contract: `modules/ssfv-engine/SPEC.md`
- source_scope: `modules/ssfv-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:ssfv-engine`
- record_digest: `sha256:e774c4b836ba10e76e9de72bb09145196be16f91b09b47b0a23e5117c6a87e1e`
- notes: Architect-ratified F3 nested record for content-addressed semantic-catalog proposals; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:ssfv-engine.catalog-integrity-snapshot`
- feature_file: `modules/ssfv-engine/FEATURES.md`
- owner_contract: `modules/ssfv-engine/SPEC.md`
- source_scope: `modules/ssfv-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:ssfv-engine`
- record_digest: `sha256:1305843de665b58d039d0428d935d41871f75d53b3c4caf31b74e74e05bdcb38`
- notes: Architect-ratified F3 nested record for semantic-catalog integrity snapshots; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:ssfv-engine.semantic-freshness-comparison`
- feature_file: `modules/ssfv-engine/FEATURES.md`
- owner_contract: `modules/ssfv-engine/SPEC.md`
- source_scope: `modules/ssfv-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:ssfv-engine`
- record_digest: `sha256:5c72d2434f507fc64de466314b556c636bc9c176ba64a12b3dfb374ddce4ecc5`
- notes: Architect-ratified F3 nested record for bounded semantic-freshness comparison; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:ssfv-engine.semantic-graph-projection`
- feature_file: `modules/ssfv-engine/FEATURES.md`
- owner_contract: `modules/ssfv-engine/SPEC.md`
- source_scope: `modules/ssfv-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:ssfv-engine`
- record_digest: `sha256:390115d0aac23d35d012f268ce5a229238e0d32e964c5facb93436f9716d3a45`
- notes: Architect-ratified F3 nested record for a portable semantic-feature graph projection; coverage remains partial and no broader runtime or canonical authority is implied.

- feature_id: `ssfv:symphony:ssiag-foundation`
- feature_file: `modules/secure-identity-access-governance/FEATURES.md`
- owner_contract: `modules/secure-identity-access-governance/SPEC.md`
- source_scope: `modules/secure-identity-access-governance`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:platform`
- record_digest: `sha256:32a1d25fef5364661bd42662d1deb1f19f2de23b38088550053bbe94c58840e6`
- notes: Partial-catalog record for caller-neutral local identity, endpoint trust, authorization decisions, supervision, and safe STAV production.

- feature_id: `ssfv:symphony:ssiag-foundation.authorization-capabilities`
- feature_file: `modules/secure-identity-access-governance/FEATURES.md`
- owner_contract: `modules/secure-identity-access-governance/SPEC.md`
- source_scope: `modules/secure-identity-access-governance`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:ssiag-foundation`
- record_digest: `sha256:74ebe85a8c00c0f70742b2b1c39d339016bcd20e34140b3f24d84320a83e10b5`
- notes: Architect-ratified F2 subfeature for exact caller-neutral authorization and bounded capability evidence; coverage remains partial.

- feature_id: `ssfv:symphony:ssiag-foundation.kernel-peer-trust`
- feature_file: `modules/secure-identity-access-governance/FEATURES.md`
- owner_contract: `modules/secure-identity-access-governance/SPEC.md`
- source_scope: `modules/secure-identity-access-governance`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:ssiag-foundation`
- record_digest: `sha256:19b98511edda84a906a439155c43730cfddcc8cb1fe8f79a0d2396cfe98d2560`
- notes: Architect-ratified F2 subfeature for kernel-attested local peer and endpoint trust; coverage remains partial.

- feature_id: `ssfv:symphony:ssiag-foundation.native-supervision`
- feature_file: `modules/secure-identity-access-governance/FEATURES.md`
- owner_contract: `modules/secure-identity-access-governance/SPEC.md`
- source_scope: `modules/secure-identity-access-governance`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:ssiag-foundation`
- record_digest: `sha256:46eb1a6c93b13cbc43321dc73d8e9af0bb9500855c58ce6e12ea21ea668873ad`
- notes: Architect-ratified F2 subfeature for per-TOPS native SSIAG supervision; coverage remains partial.

- feature_id: `ssfv:symphony:ssiag-foundation.policy-administration`
- feature_file: `modules/secure-identity-access-governance/FEATURES.md`
- owner_contract: `modules/secure-identity-access-governance/SPEC.md`
- source_scope: `modules/secure-identity-access-governance`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:ssiag-foundation`
- record_digest: `sha256:5aaa607e1b8959183eaf78b825eaa810c6ce5e06b1e17357a82cd6217f6042a4`
- notes: Architect-ratified F2 subfeature for durable local SSIAG policy administration; coverage remains partial.

- feature_id: `ssfv:symphony:ssiag-foundation.provider-metadata-registry`
- feature_file: `modules/secure-identity-access-governance/FEATURES.md`
- owner_contract: `modules/secure-identity-access-governance/SPEC.md`
- source_scope: `modules/secure-identity-access-governance`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:ssiag-foundation`
- record_digest: `sha256:0c323365fa8425f562fb3124df4c47936a22d030208fdf5a09bb085391487dc5`
- notes: Architect-ratified F2 subfeature for safe configured provider metadata; no operational adapter bridge is claimed.

- feature_id: `ssfv:symphony:ssiag-foundation.safe-audit-production`
- feature_file: `modules/secure-identity-access-governance/FEATURES.md`
- owner_contract: `modules/secure-identity-access-governance/SPEC.md`
- source_scope: `modules/secure-identity-access-governance`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:ssiag-foundation`
- record_digest: `sha256:1b399689c832b297451a8304392a6c61b46889f7f3a59ea0ec7925a44624b73b`
- notes: Architect-ratified F2 subfeature for closed safe SSIAG audit production; coverage remains partial.

- feature_id: `ssfv:symphony:ssiag-foundation.tops-enrollment`
- feature_file: `modules/secure-identity-access-governance/FEATURES.md`
- owner_contract: `modules/secure-identity-access-governance/SPEC.md`
- source_scope: `modules/secure-identity-access-governance`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:ssiag-foundation`
- record_digest: `sha256:dbbede83433c4356f388b94ef10e86a9997a8cab3ff96947328cfa4f7659bc9b`
- notes: Architect-ratified F2 subfeature for per-TOPS SSIAG enrollment and isolation; coverage remains partial.

- feature_id: `ssfv:symphony:ssiag.macos-keychain-metadata`
- feature_file: `modules/ssiag-provider-macos-keychain/FEATURES.md`
- owner_contract: `modules/ssiag-provider-macos-keychain/SPEC.md`
- source_scope: `modules/ssiag-provider-macos-keychain`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:ssiag-foundation`
- record_digest: `sha256:0c648fe3bca0ceb876989cf2f3ff95b7d338a7ef3f35a0dd1a931b7e597b9149`
- notes: Partial-catalog subfeature for the isolated Swift metadata-only macOS Keychain adapter; no Go-foundation or qxctl invocation bridge exists and operational secret access remains disabled.

- feature_id: `ssfv:symphony:stav-append-authority`
- feature_file: `modules/stav-append-authority/FEATURES.md`
- owner_contract: `modules/stav-append-authority/SPEC.md`
- source_scope: `modules/stav-append-authority`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:platform`
- record_digest: `sha256:f0004091a329f063c799cd10cc89a5ce200e9c1e26698c0ea3f32c9e376a1f69`
- notes: Partial-catalog record for serialized durable tamper-evident STAV append, recovery, and bounded query behavior.

- feature_id: `ssfv:symphony:stav-append-authority.authorized-query`
- feature_file: `modules/stav-append-authority/FEATURES.md`
- owner_contract: `modules/stav-append-authority/SPEC.md`
- source_scope: `modules/stav-append-authority`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:stav-append-authority`
- record_digest: `sha256:a29998983331a6468a0ffe7b716ea99b7afb441aeac28ac7ff32c47f8254ee18`
- notes: Architect-ratified F2 subfeature for event-class-scoped STAV query and verification; coverage remains partial.

- feature_id: `ssfv:symphony:stav-append-authority.ledger-durability`
- feature_file: `modules/stav-append-authority/FEATURES.md`
- owner_contract: `modules/stav-append-authority/SPEC.md`
- source_scope: `modules/stav-append-authority`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:stav-append-authority`
- record_digest: `sha256:d0fc1b99ff45577951f991379661c692a2e55f3ee2672672da071e046eb62dc5`
- notes: Architect-ratified F2 subfeature for checksummed tamper-evident ledger durability and bounded tail recovery; coverage remains partial.

- feature_id: `ssfv:symphony:stav-append-authority.native-supervision`
- feature_file: `modules/stav-append-authority/FEATURES.md`
- owner_contract: `modules/stav-append-authority/SPEC.md`
- source_scope: `modules/stav-append-authority`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:stav-append-authority`
- record_digest: `sha256:d6bed2d6d81651c26ae4e359d2e1f4b1ad95ae1e001533e9a6ef74bd946a592f`
- notes: Architect-ratified F2 subfeature for per-TOPS native append-authority supervision; coverage remains partial.

- feature_id: `ssfv:symphony:stav-append-authority.serialized-append`
- feature_file: `modules/stav-append-authority/FEATURES.md`
- owner_contract: `modules/stav-append-authority/SPEC.md`
- source_scope: `modules/stav-append-authority`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:stav-append-authority`
- record_digest: `sha256:7679344baa9d42d7e5b2c0db3126e8264a054493677eeeeb39738947f3fb4c10`
- notes: Architect-ratified F2 subfeature for authenticated serialized STAV commitment; coverage remains partial.

- feature_id: `ssfv:symphony:stav-append-authority.tops-enrollment`
- feature_file: `modules/stav-append-authority/FEATURES.md`
- owner_contract: `modules/stav-append-authority/SPEC.md`
- source_scope: `modules/stav-append-authority`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:stav-append-authority`
- record_digest: `sha256:39dd01d227c6ce26f370f8966baff21a737e5ff509f49c20c0768e8e03a62a80`
- notes: Architect-ratified F2 subfeature for per-TOPS STAV serialization-domain enrollment; coverage remains partial.

- feature_id: `ssfv:symphony:stav-protocol-kernel`
- feature_file: `libraries/stav-protocol-go/FEATURES.md`
- owner_contract: `libraries/stav-protocol-go/MANIFEST.md`
- source_scope: `libraries/stav-protocol-go`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:platform`
- record_digest: `sha256:aa8715b51e5dd82a84d3de0de27101798f298a6cd1597d41a08f50ffeca11515`
- notes: Partial-catalog record for authority-free canonical STAV bytes, digests, typed codecs, semantic validation, and bounded local IPC framing; durable checksummed ledger framing belongs to the append authority.

- feature_id: `ssfv:symphony:stav-protocol-kernel.canonical-wire-representation`
- feature_file: `libraries/stav-protocol-go/FEATURES.md`
- owner_contract: `libraries/stav-protocol-go/MANIFEST.md`
- source_scope: `libraries/stav-protocol-go`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:stav-protocol-kernel`
- record_digest: `sha256:1d077453494e50d6ec211ba4ed52213be3d120916778764471d315833dbba8be`
- notes: Architect-ratified F2 subfeature for canonical STAV bytes, digests, and bounded local frames; coverage remains partial.

- feature_id: `ssfv:symphony:stav-protocol-kernel.semantic-validation`
- feature_file: `libraries/stav-protocol-go/FEATURES.md`
- owner_contract: `libraries/stav-protocol-go/MANIFEST.md`
- source_scope: `libraries/stav-protocol-go`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:stav-protocol-kernel`
- record_digest: `sha256:c2931ce80338d4be722c2f9cf7d7d198dc0be8d3c1e834dfdd12afeda6628609`
- notes: Architect-ratified F2 subfeature for exact STAV content and identifier validation; coverage remains partial.

- feature_id: `ssfv:symphony:symphony-validator`
- feature_file: `tools/symphony-validator/FEATURES.md`
- owner_contract: `tools/symphony-validator/SPEC.md`
- source_scope: `tools/symphony-validator`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:platform`
- record_digest: `sha256:2ce39998bc064984d03971777493e9261659a4dccf08fb0fe8c75688c6608a45`
- notes: Partial-catalog record for deterministic read-only repository validation and structured evidence projection.

## Prohibited Entries

Do not register:

- proposals or planned behavior as present application truth;
- generated inventories, graph projections, summaries, documentation, or marketing pages;
- a directory merely because it exists;
- a language, source file, or symbol without the full feature-worthiness gate;
- an owner record using implicit globs or traversal;
- proposal-only modules such as node-troll, bus-troll, or hotpath-runtime before implementation exists;
- a record not covered by SKVI and an owner contract.

## Non-Authorization Statement

This sixty-nine-record registry is an explicitly partial catalog governed by `COVERAGE.md`. It does not authorize an unratified feature record, an unregistered distributed file, repository-wide completeness, engine-decided feature-worthiness, or canonical mutation.
