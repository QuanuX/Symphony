# Multi-artifact acceptance evidence

`delivery-acceptance` composes the existing v1 acceptance checker with the complete
signed delivery set. Both checks must be ready before its result can say
`evidence_ready_for_review`. It never grants release acceptance, publication,
provider authority, signing authority, or permission to execute a campaign.
Anaconda is the selected distribution route; its publisher/trust and fresh-host
inputs remain end-of-W12 decisions. A local staging path is not a download URL.

The v1 packet, plan, schema and check semantics are preserved. The two original
CLI entry points now live in `acceptance-gates-main.go` and `delivery-set-main.go`
so this companion can call the same readers/checkers without duplicated logic.
Use the current build commands below and in [README.md](README.md); historical
single-file build commands need the corresponding `-main.go` file.

The companion [JSON schema](../../cmake/release/delivery-acceptance-v1.schema.json)
describes the four input protocols. The Go checker enforces the stronger
cross-file, independent-selection and signature requirements below.

## Inputs and preparation

Use exact Go 1.26.5. All paths are explicit absolute paths in private, quiescent
work directories. Inputs, evidence and reports must be outside delivery payloads.
No network, provider operation, payload execution or automatic provisioning occurs.
The verifier uses OpenSSH only for existing signed-index verification.

```sh
GOWORK=off GOENV=off GOTOOLCHAIN=local go build -trimpath \
  -o "$private_tools/acceptance-gates" \
  tests/launch-acceptance/acceptance-gates.go \
  tests/launch-acceptance/acceptance-gates-main.go
GOWORK=off GOENV=off GOTOOLCHAIN=local go build -trimpath \
  -o "$private_tools/delivery-acceptance" \
  tests/launch-acceptance/delivery-set.go \
  tests/launch-acceptance/acceptance-gates.go \
  tests/launch-acceptance/delivery-acceptance.go

"$private_tools/acceptance-gates" -mode init -plan "$core_plan" \
  -packet "$new_core_packet"
"$private_tools/delivery-acceptance" -mode init-requirements \
  -plan "$core_plan" -mapping "$source_acceptance_mapping" \
  -requirements "$new_artifact_requirements"
```

The independent requirements map is **a review input, not an observation**. It
binds the exact original plan and `tests/public-beta/acceptance-mapping.json` bytes.
It retains four core profiles, nine scenarios, 36 cells, 58 paper criteria and 31
supplemental entries (23 Cloudflare operations and eight further campaigns).
Templates use explicit nulls. Empty arrays cannot stand in for pending work.
No cell or criterion can be dropped by editing the observation packet.

For each profile, replace `artifact_ids: null` with the complete selected artifact
IDs. For each core scenario, replace its null with the exact required artifacts,
including transitive dependencies. Add supplemental profiles explicitly with an
empty `scenario_artifacts` object and a complete artifact list. Each supplemental
entry needs a profile-to-artifact-list mapping. Cloudflare operation records
require the actual macOS/amd64 closure and `sov-cloudflare-adapter` identity;
core Linux/Apple Silicon evidence cannot substitute for the Intel runtime.
Unresolved selection or retained missing implementation stays null/incomplete.
Do not remove a promise to make the checker pass.

Once this independent map is reviewed, prepare a fresh companion template:

```sh
"$private_tools/delivery-acceptance" -mode init \
  -plan "$core_plan" -mapping "$source_acceptance_mapping" \
  -requirements "$artifact_requirements" -core-packet "$core_packet" \
  -packet "$new_companion_packet"
```

`targets.json` uses protocol `symphony.release.delivery-targets.v1` and a
`profiles` object with exactly the requirements' profile keys. Each value is null
while pending, or an object with `root`, `selection`, `allowed_signers`, `signer`.
The last three values are independent operator inputs, not values trusted from
an archive. Each root must retain its exact signed `delivery-set.json`, signature
and complete payload. Use separate policies for native artifact signatures and
outer delivery signatures as described in [DELIVERY-SET.md](DELIVERY-SET.md).

The companion packet binds the v1 packet, original plan, mapping and requirements
hashes. Its core profile's `primary_native_artifact_id` resolves the actual
native archive, manifest and native policy hashes used in v1. Artifact IDs never
replace digest values. Every verified profile must use the same source commit
**and tree**. Supplemental profiles have no v1 primary and use null there.
An index reference is a copied exact index in the evidence directory, e.g.
`profile-index.json`; it does not replace verification of the actual retained set.

## Evidence records

All references are `{ "path": "relative/evidence.json", "sha256": "..." }`
inside the companion packet's directory. Evidence must be nonempty regular,
single-link files without symlink traversal, special bits or group/other write.
Each metadata file is bounded to 16 MiB, with at most 10,000 distinct referenced
files and 512 MiB of unique companion evidence. Preserve larger raw transcripts
in an independent retained evidence store and reference bounded manifests that
identify them for the reviewer; this checker does not verify remote manifests'
contents. Inputs, signed sets and companion evidence are checked again after the
walk. This is private-directory integrity checking, not hostile same-UID isolation.

Each selected artifact has `owner`, `installation`, `loader`, and `trust` fields.
The last three are observation references and remain mandatory even when a
particular artifact's loader/trust outcome is a reviewed explicit applicability
assessment. The checker does not invent a platform exemption.

`owner` is `{ "report": ref, "support": [ref, ...] }`:

- Native: use the actual `inspect-native` result plus exact host-preparation,
  payload-verification and native tool-record bytes. These hashes and their
  source/target/archive/manifest/signature/policy fields must agree.
- SSIAG/STAV/Accordare: use the actual `inspect-owner` result, independent owner
  build record and original raw inspection output. The raw inspection must agree
  with the embedded one and actual selected receipt/binary metadata. Retain the
  harness stdout when collecting it; reserialization is not a byte identity.
- qxctl/plain SOV: use the actual `inspect-executable` result. Build identity and
  immutable inventory are read from the signed set; full version output and its
  digest are checked. No synthetic receipt or protected-profile result is added.
- A selected adapter without an implemented acceptance reader stays incomplete.
  Naming Swift/vendor/source adapters in an index does not qualify those owners.

An observation uses protocol `symphony.release.delivery-observation.v1` with all
fields defined in `daObservation` in the source. Its `id` is exactly one of:
`profile::scenario`, `profile:artifact:installation`, `profile:artifact:loader`,
`profile:artifact:trust`, paper criterion ID, or supplemental ID. It contains:

- `result` (`pending`, `failed`, `passed`), `scope` (`unqualified`, `fixture`,
  `operational`), nonnegative `skips`, nonempty `observer`, RFC3339 `observed_at`,
  and nonempty `expected_outcome` / `actual_outcome`.
- `bindings`: profile keys, each with `source`, `delivery_index_sha256` and exact
  `artifacts`. Each artifact identity includes full `primary` and `inventory`
  file records (`path`, `sha256`, `size`, `mode`), `receipts` file records and
  nullable `artifact_signer_policy_sha256`. The checker recomputes this entire
  structure from the signed index, including dependency closure.
- `core_cells`: the single exact scenario cell, all mapped campaign cells on all
  four profiles for a paper criterion, or an empty list for artifact/supplemental
  observations. Paper evidence cannot omit its mapped scenarios or profiles.
- `evidence`: one to 128 hashed nonempty attachments containing actual commands,
  input/output/oracle results, platform, trust, retrieval and reviewer evidence
  appropriate to that observation. No opaque "passed" string is independent proof.
- `release_acceptance_complete` and `publication_authorized`, both false.

Only passed operational observations with zero skips are structurally ready.
Failed, fixture, unqualified, missing and skipped observations remain incomplete.
The source mapping's scoring policy and missing promises remain human decisions;
this tool neither computes an invented 8.5 score nor overrides a freeze blocker.

## Checking and interpretation

```sh
"$private_tools/delivery-acceptance" -mode check \
  -plan "$core_plan" -mapping "$source_acceptance_mapping" \
  -requirements "$artifact_requirements" -core-packet "$core_packet" \
  -packet "$companion_packet" -targets "$independent_targets" \
  -report "$new_result"
```

Exit 1 means invalid, unsafe, changed or mismatched input. Exit 2 means valid but
incomplete evidence; the report retains the complete original v1 result and its
own missing list. Exit 0 means structural evidence is ready for independent
review. All exits preserve false release/publication authority. Existing output
files are refused; revise a packet as a new retained revision and rebind its
hashes. Do not erase failed attempts.

Structural checking cannot determine whether an observer told the truth, whether
an attached transcript proves its claim, whether a rights review is adequate, or
whether the human reviewer is independent. The positive regression fixture
intentionally supplies synthetic assertions to test structure; it is clearly
labeled and is never operational acceptance. Final W12 reviewers must inspect the
actual evidence and decisions, including original missing claims and platform
limits, before recording their separate decision.

Focused regression command:

```sh
GOWORK=off GOENV=off GOTOOLCHAIN=local go test -race -count=1 \
  tests/launch-acceptance/delivery-set.go \
  tests/launch-acceptance/delivery-set_test.go \
  tests/launch-acceptance/acceptance-gates.go \
  tests/launch-acceptance/acceptance-gates_test.go \
  tests/launch-acceptance/delivery-acceptance.go \
  tests/launch-acceptance/delivery-acceptance_test.go
```
