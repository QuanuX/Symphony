# Symphony Semantic Feature Vector Registry

## Status

Canonical SSFV feature-routing registry. The partial catalog routes fifteen experimental records across the repository root and fourteen implemented owner scopes enumerated by `COVERAGE.md`.

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

- feature_id: `ssfv:symphony:knowledge-vector-engine-foundation`
- feature_file: `libraries/knowledge-vector-engine-cpp/FEATURES.md`
- owner_contract: `libraries/knowledge-vector-engine-cpp/SPEC.md`
- source_scope: `libraries/knowledge-vector-engine-cpp`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:platform`
- record_digest: `sha256:c31e519f59335ed7e145874795409bbe36a555006f7bf1e18c93afea9952b4ce`
- notes: First partial bootstrap record for the implemented authority-free shared C++ mechanics, including canonical STSC Gregorian and UTC representation validation.

- feature_id: `ssfv:symphony:maestro-presence-authority`
- feature_file: `modules/maestro/FEATURES.md`
- owner_contract: `modules/maestro/SPEC.md`
- source_scope: `modules/maestro`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:platform`
- record_digest: `sha256:89076bde7da3ddc19c185e1c05025e5d822c5e31ab6e299d6c4c6df5e86c5084`
- notes: Partial-catalog record for exact authenticated per-TOPS/per-receptor docking presence, complete read-only derived inventory, lifecycle integration, semantic retry, and evidence-preserving recovery without engine execution.

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

- feature_id: `ssfv:symphony:sacv-engine`
- feature_file: `modules/sacv-engine/FEATURES.md`
- owner_contract: `modules/sacv-engine/SPEC.md`
- source_scope: `modules/sacv-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:platform`
- record_digest: `sha256:30ea222966c890b7da45cbebc3341d0c5ecd1e3e54f8b299f6a77e152ab03895`
- notes: Partial-catalog record for bounded read-only SACV API-contract governance behavior.

- feature_id: `ssfv:symphony:sclv-engine`
- feature_file: `modules/sclv-engine/FEATURES.md`
- owner_contract: `modules/sclv-engine/SPEC.md`
- source_scope: `modules/sclv-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:platform`
- record_digest: `sha256:d75de6d50edef708c273e859de397893b2adb6b87ff456fad0d62464880cf8ca`
- notes: Partial-catalog record for bounded read-only SCLV change-truth governance behavior.

- feature_id: `ssfv:symphony:skvi-engine`
- feature_file: `modules/skvi-engine/FEATURES.md`
- owner_contract: `modules/skvi-engine/SPEC.md`
- source_scope: `modules/skvi-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:platform`
- record_digest: `sha256:a5530ebd69cf7f364ff993df14992a1219e3d0992381bfeb134d0a3e2b728d99`
- notes: Partial-catalog record for bounded read-only SKVI routing governance behavior.

- feature_id: `ssfv:symphony:sodv-engine`
- feature_file: `modules/sodv-engine/FEATURES.md`
- owner_contract: `modules/sodv-engine/SPEC.md`
- source_scope: `modules/sodv-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:platform`
- record_digest: `sha256:624a05a6b051f515b7c901f6f98eef29528867f850caf045ce19c5d1df42be4b`
- notes: Partial-catalog record for bounded read-only SODV release-publication governance behavior.

- feature_id: `ssfv:symphony:ssfv-engine`
- feature_file: `modules/ssfv-engine/FEATURES.md`
- owner_contract: `modules/ssfv-engine/SPEC.md`
- source_scope: `modules/ssfv-engine`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:platform`
- record_digest: `sha256:d3d68409048baff8784665c45d4363c084d14fbfb50ea40ebe74e61d233d789b`
- notes: Partial-catalog record for bounded read-only SSFV semantic-feature governance behavior.

- feature_id: `ssfv:symphony:ssiag-foundation`
- feature_file: `modules/secure-identity-access-governance/FEATURES.md`
- owner_contract: `modules/secure-identity-access-governance/SPEC.md`
- source_scope: `modules/secure-identity-access-governance`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:platform`
- record_digest: `sha256:32a1d25fef5364661bd42662d1deb1f19f2de23b38088550053bbe94c58840e6`
- notes: Partial-catalog record for caller-neutral local identity, endpoint trust, authorization decisions, supervision, and safe STAV production.

- feature_id: `ssfv:symphony:ssiag.macos-keychain-metadata`
- feature_file: `modules/ssiag-provider-macos-keychain/FEATURES.md`
- owner_contract: `modules/ssiag-provider-macos-keychain/SPEC.md`
- source_scope: `modules/ssiag-provider-macos-keychain`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:ssiag-foundation`
- record_digest: `sha256:6063491fec631ed5bf6297da786c230239ea421ae98edd53425c8536f6fd73c4`
- notes: Partial-catalog subfeature record for the isolated Swift metadata-only macOS Keychain adapter; operational secret access remains disabled.

- feature_id: `ssfv:symphony:stav-append-authority`
- feature_file: `modules/stav-append-authority/FEATURES.md`
- owner_contract: `modules/stav-append-authority/SPEC.md`
- source_scope: `modules/stav-append-authority`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:platform`
- record_digest: `sha256:f0004091a329f063c799cd10cc89a5ce200e9c1e26698c0ea3f32c9e376a1f69`
- notes: Partial-catalog record for serialized durable tamper-evident STAV append, recovery, and bounded query behavior.

- feature_id: `ssfv:symphony:stav-protocol-kernel`
- feature_file: `libraries/stav-protocol-go/FEATURES.md`
- owner_contract: `libraries/stav-protocol-go/MANIFEST.md`
- source_scope: `libraries/stav-protocol-go`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:platform`
- record_digest: `sha256:a4039a8fc93043847546eae3a6ae71436e51ed4b4c4e861eb052aa3512ef4d0d`
- notes: Partial-catalog record for the authority-free Go STAV protocol, validation, digest, framing, and conformance kernel.

- feature_id: `ssfv:symphony:symphony-validator`
- feature_file: `tools/symphony-validator/FEATURES.md`
- owner_contract: `tools/symphony-validator/SPEC.md`
- source_scope: `tools/symphony-validator`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:platform`
- record_digest: `sha256:2016b2ff8e63382ad2f24068e1e327815a2d4a8a8f9b9dcf9486fb2d37170590`
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

This fifteen-record registry is an explicitly partial catalog governed by `COVERAGE.md`. It does not authorize an unratified feature record, an unregistered distributed file, repository-wide completeness, engine-decided feature-worthiness, or canonical mutation.
