# Final-download MVP acceptance

L26 remains blocked on completion of the native credential/provider integration, a final selected capability manifest, published candidate bytes, independent publisher trust policy, runtime/license closure and authorized provider profiles/SDK. Current CI builds and packaging fixtures are development evidence. This procedure must run on a fresh selected host after those prerequisites exist; never relabel source checkout/cached fixture results as final-download acceptance.

Required profiles: macOS 15 ARM64; Ubuntu 24.04 amd64 and ARM64; Ubuntu 26.04 amd64. Record actual OS image/kernel, CPU architecture, libc/C++ runtime, filesystem and tool versions. Generic Ubuntu ARM64 is sufficient for this sprint, without physical DGX Spark, GPU or native Windows claims.

1. Start from a new host/private prefix with no inherited Symphony installation or module/build cache. Record a clean pre-install inventory and source of host provisioning. Select the approved manifest and independent trust policy before retrieving bytes.
2. Retrieve the exact candidate URL and retain final resolved URL, retrieval time, size and digest. Verify its signed manifest and independently supplied signer policy with the existing candidate tooling described in `cmake/` and `tests/package-release/README.md`. Reject modified bytes, wrong signer and incompatible profile. Fixture keys cannot authorize production candidates.
3. Extract through the reviewed package recipe into a new prefix. Retain archive-to-extracted-byte correspondence, loader resolution, exact receipt verification and discovery/version output. Run without the source checkout or build-time dependency prefixes. Do not repair missing libraries from a developer cache.
4. Execute each scenario in `plan.json` on the exact final artifacts. Use the SHV/SNV/SBV, package and protected-administration harnesses as source specifications, not evidence substitutes. Adapt only with a reviewed exact-version recipe; record command, inputs, output digest, refusal codes, before/after state and no-skip results.
5. Execute every promised provider path with authorized bounded read-only profiles. Include entitlement refusal, expiry, disconnect/cancellation, uncertainty and reconciliation. TWS requires authorized SDK provenance and actual host integration. No credentials or private data belong in uploaded evidence. Synthetic data remains explicitly synthetic; it cannot satisfy provider acceptance.
6. Exercise side-by-side upgrade, compatible rollback, interrupted operations and removal. Preserve user stores and unrelated sentinel files. Keep failed staging and uncertain operation receipts for reconciliation. Prove the original user data survives, rather than checking only command exit status.
7. A named reviewer verifies artifact/source/profile bindings and all required results. Missing/skipped/fixture-only results leave acceptance incomplete. Publisher approval is a separate decision; even complete evidence does not authorize publication automatically.

Retain per-profile reports outside installations. Minimum fields are in `plan.json`: no fabricated URLs, evidence paths, timestamps, approvals or success flags. At present no final candidate or operational provider result is available, so this increment supplies executable source-smoke checks and the acceptance procedure, not a completed L26 run.

## Exact candidate verification commands

After an authorized operator has retrieved all manifest-bound candidate files into `candidate`, independently selected `allowed_signers` outside that directory, and set `signer`, `profile`, and a fresh private `evidence` directory, run from the reviewed verification-tool source:

```sh
cmake -DRELEASE_CANDIDATE="$candidate" \
  -DRELEASE_ALLOWED_SIGNERS="$allowed_signers" -DRELEASE_SIGNER="$signer" \
  -DRELEASE_EXPECTED_PROFILE="$profile" \
  -DRELEASE_VERIFICATION_OUTPUT="$evidence/downloaded-bytes.json" \
  -P cmake/release/VerifyCandidate.cmake
```

After the separately reviewed extraction procedure has produced `unpacked`, repeat with `-DRELEASE_UNPACKED_ROOT="$unpacked"` and a new output path `"$evidence/unpacked.json"`. The first record proves signed byte bindings; the second compares caller-supplied payload files. Neither proves extraction origin, native execution, clean-host provisioning or provider success. Retain those observations separately. The package fixture regression demonstrates these mechanics with disposable keys only; never install its trust policy as publisher authority.

## Scope review before candidate selection

Use the [finish-line checklist](../../.github/RELEASE-GATES.md) and `prepare-review.cmake` to produce an exact-source component/version inventory with unfilled acceptance inputs. It is preparation for selecting the shipped capability manifest, not a candidate or approval.

## Verified extraction on the selected host

Build the standalone standard-library helper with the pinned Go 1.26.5 toolchain:

```sh
GOWORK=off GOTOOLCHAIN=local go build -trimpath \
  -o /absolute/private-tools/candidate-host tests/launch-acceptance/candidate-host.go
```

After the operator has obtained exact candidate files and the independently
approved signer policy, invoke it with a reviewed verification-tool checkout:

```sh
/absolute/private-tools/candidate-host \
  -repository /absolute/reviewed-verification-source \
  -candidate /absolute/retrieved-candidate \
  -allowed-signers /absolute/independent-policy/allowed-signers \
  -signer SELECTED_PUBLISHER -profile SELECTED_PROFILE \
  -work /absolute/new-private-work
```

All filesystem arguments are absolute. The work parent must exist and be under
operator control; work must be new and outside the source/candidate directories.
Use quiescent private inputs, retain the source/tool identity separately, and
never substitute the package regression's disposable signing policy for publisher
trust. The helper performs no network request or key enrollment. Candidate
retrieval URL, time and digest must still be recorded by the operator.

It runs existing signature/profile/evidence verification before reading the
archive, checks the opened archive's complete size/digest, and admits only the
signed inventory's exact regular files. Symlinks, hardlinks, devices, path escapes,
duplicate/unlisted/missing members, altered bytes, unsafe file modes and nonzero
trailing content are refused. Its current explicit ceiling is 16 GiB total
payload/archive, 16 MiB inventory and 65,536 files. Output stays inside a new
private prefix. A failure leaves diagnostic work for inspection and produces no
successful host report; retry uses a fresh work directory.

After extraction, the existing verifier checks the receipt/file closure again.
Both observations must bind the same manifest and independent signer policy.
`host-preparation.json` records actual host OS/architecture/kernel and the exact
source/artifact identities. It deliberately reports `verified_extraction_only`,
`fresh_host_verified=false`, `provider_qualification_complete=false`,
`release_acceptance_complete=false` and `publication_authorized=false`.

The helper does not execute downloaded code or prove native compatibility,
production signing/notarization, a clean host, loader closure, redistribution
rights or provider success. Run the nine approved scenarios separately against
`payload/`, retaining exact command/input/output and host-provisioning evidence.
The four-target package fixture workflow exercises this path and executes its
two independently built fixture probes; those results remain development evidence.

## Runnable evidence gates while live inputs are unavailable

The standard-library `acceptance-gates` tool creates a packet from the current
plan and checks the evidence supplied later. It never connects to a provider,
executes a packet's commands, imports credentials, or approves publication.

```sh
GOWORK=off GOTOOLCHAIN=local go build -trimpath \
  -o /absolute/private-tools/acceptance-gates tests/launch-acceptance/acceptance-gates.go
/absolute/private-tools/acceptance-gates -mode init \
  -plan /absolute/reviewed-source/tests/launch-acceptance/plan.json \
  -packet /absolute/private-evidence/packet.json
/absolute/private-tools/acceptance-gates -mode check \
  -plan /absolute/reviewed-source/tests/launch-acceptance/plan.json \
  -packet /absolute/private-evidence/packet.json \
  -report /absolute/private-evidence/new-report.json
```

Create the private evidence directory first. The template includes every required
input and all nine scenarios for all four profiles. Missing inputs remain null;
all scenarios start pending and unqualified. Initial checking intentionally exits
**2** with an `incomplete` report enumerating the missing items. Exit **1** means
invalid/tampered evidence or an output conflict. Exit **0** means only that the
packet is structurally ready for independent review. All reports keep
`release_acceptance_complete=false` and `publication_authorized=false`.

Every attachment is `{ "path": "relative/path", "sha256": "64-lowercase-hex" }`.
Paths resolve beneath the packet directory; links and traversal are refused.
Evidence is limited to 16 MiB per file. Use a private quiescent directory, retain
actual redacted observation files, and record their SHA-256 hashes. Do not attach
credentials, private SDK source, signing keys or paid/private provider payloads.
An attachment is evidence to review, not proof that its author had authority.

For each profile, supply source commit and artifact/manifest/signer-policy hashes;
attach the actual `caller_supplied_payload` output from candidate verification,
clean-host inventory and runtime/notice review. The checker requires verification
fields to match that profile and those exact identities. All profiles must use
the same selected source commit. Run `candidate-host` first; do not fabricate its
upstream verification record or treat an inventory attachment as host execution.

A scenario's attached record has this closed structure:

```json
{
  "profile_id": "ubuntu24.04-arm64",
  "scenario": "every_promised_provider_operation_and_refusal",
  "source_commit": "EXACT_40_HEX_COMMIT",
  "artifact_sha256": "EXACT_64_HEX_ARTIFACT_DIGEST",
  "result": "pending",
  "scope": "unqualified",
  "skips": 0,
  "evidence": [{"path": "observations/provider.json", "sha256": "EXACT_64_HEX_DIGEST"}]
}
```

The uppercase values above are documentation placeholders and intentionally fail
validation. Bind actual command/input/output and expected/observed result records
under `evidence`. Paths in this list are relative to the packet directory too.
Each scenario envelope must match its packet profile, scenario, source, artifact,
result, scope and skip count. At least one digest-verified observation is required.
`result` is `pending`, `failed` or `passed`; `scope` is `unqualified`, `fixture` or
`operational`. Only a passed, operational, zero-skip result can satisfy the
structural gate. Here operational means the approved scenario actually ran on the
selected final candidate; an intentionally synthetic offline-research scenario
remains labelled synthetic in its observations. Live-provider scenarios require
actual authorized provider results, not synthetic callbacks.

The tool detects missing/drifted bindings and corrupted attachments. It does not
independently establish the truth of supplied observations, re-verify a publisher
signature, evaluate licence rights, inspect provider entitlements, or attest a
fresh host. A reviewer must examine the referenced evidence and record acceptance
through the existing release authority. Complete structure never changes that
boundary. The package CI matrix runs fixture regressions for missing coverage,
changed hashes, wrong identities, duplicate/unknown fields, unsafe paths, skips,
failed/fixture results and overwrite refusal.
