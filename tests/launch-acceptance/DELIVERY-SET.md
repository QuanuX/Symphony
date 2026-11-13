# Multi-artifact delivery index

This standard-library Go release tool binds an explicitly selected set of native,
Go, Swift, vendor and corresponding-source artifacts. It prepares an immutable
index, validates the complete file set and verifies an SSH signature under the
independently supplied `symphony-delivery-set` namespace. It does not install or
execute downloaded programs, provision services or credentials, run providers,
prove owner semantics, or approve release/publication.

The byte-verification modes report `unsigned_bytes_only` or `signed_bytes_only`.
It lists every `pending_owner_adapters` entry and leaves owner, fresh-host, release
and publication flags false. Explicit owner modes supply separate per-artifact
records; the versioned acceptance companion remains a separate increment.
An index signature does not replace an
artifact's own signer policy or actual owner verification.

The separate `inspect-owner` mode now invokes actual read-only receipt readers
for SSIAG, STAV and Accordare. Its narrower result covers only the selected
artifact. Other selected artifacts and the acceptance
companion still need their own evidence; one owner pass is not a set-wide pass.

## Inputs and bounds

- [Delivery selection](../../cmake/release/delivery-selection-v1.schema.json):
  independently reviewed exact source/tree, profile, capability/map digests and
  closed artifact IDs/kinds/adapters/versions/dependencies. Keep it outside the
  downloaded root; the candidate cannot select its own expected contents.
- [Preparation definition](../../cmake/release/delivery-definition-v1.schema.json):
  selected files and artifact associations, primary/inventory/receipt paths,
  install/removal recipes, notices, corresponding-source and build evidence.
  This is data; none of its paths selects an executable command.
- [Delivery index](../../cmake/release/delivery-set-v1.schema.json): resulting
  observations of every file's exact size, SHA-256 and POSIX mode, plus selection
  and independent index-policy hashes. Independent component versions are kept.
- An independently obtained allowed-signers policy and expected signer/profile.
  Production custody and authorization remain the publisher's responsibility.

JSON is limited to 16 MiB, depth 32 and 200,000 nodes. Missing, unknown or duplicate
fields, primitive nulls and invalid UTF-8 are refused. Lists use `[]`; inapplicable
pointer fields use explicit `null`. The set admits at most 4,096 artifacts and
65,536 files, 16 GiB aggregate content and 131,072 filesystem entries. Metadata
references are bounded to 16 MiB; source archives may use the aggregate payload
budget. Files must be regular, singly linked and not group/other writable.
Symlinks, hardlinks, special files, traversal, controls, reserved index names,
extra/missing files, changed modes and dependency cycles are refused.

Use a private, quiescent root under operator control. This is not a hostile
same-user filesystem boundary or a guarantee against concurrent writes. Signed
bytes must be reverified immediately before downstream consumption. The verifier
uses independent snapshots of policy/index/signature bytes for the SSH operation
and performs a stable index recheck before issuing its result.

Native records must include exactly the existing eight candidate files:
`candidate.tar`, `candidate.json`, `candidate.json.sig`, `platform-evidence.json`,
`build-evidence.json`, `dependency-map.json`, `identity-catalog.json`, and `SHA256SUMS` in the
same selected subdirectory. Native archive and artifact-policy hashes remain
distinct from delivery-index and index-policy hashes. Byte verification alone
does not parse or attest the native payload.

Go/Swift/source units keep their declared owner contracts. They are never given
fictional native receipts. Vendor provenance is separately typed and cannot
masquerade as first-party same-source provenance. Selecting an adapter ID here
records the required verification, not evidence that it has run.

## Build and run

Use Go 1.26.5, CMake 3.30+ and OpenSSH with SSHSIG. All paths below are explicit
operator-selected absolute paths. Output/work locations must be new and private.
The tool needs no Go modules or provider credentials.

```sh
GOWORK=off GOENV=off GOTOOLCHAIN=local go build -trimpath \
  -o "$private_tools/delivery-set" tests/launch-acceptance/delivery-set.go

"$private_tools/delivery-set" -mode prepare -root "$delivery_root" \
  -definition "$definition" -selection "$selection" \
  -allowed-signers "$allowed_signers" -profile "$profile"

"$private_tools/delivery-set" -mode validate -root "$delivery_root" \
  -selection "$selection" -allowed-signers "$allowed_signers" \
  -profile "$profile" -report "$new_unsigned_report"
```

Prepare hashes actual admitted files and creates `delivery-set.json` exclusively.
It refuses an existing index or signature. No signing key is created and no
signature is inferred. Reports must be new files outside the delivery root.

When signing is actually authorized, the custodian may run:

```sh
cmake -DRELEASE_DELIVERY_TOOL="$private_tools/delivery-set" \
  -DRELEASE_DELIVERY_ROOT="$delivery_root" \
  -DRELEASE_DELIVERY_SELECTION="$selection" \
  -DRELEASE_ALLOWED_SIGNERS="$allowed_signers" \
  -DRELEASE_EXPECTED_PROFILE="$profile" -DRELEASE_SIGNER="$signer" \
  -DRELEASE_SIGN_KEY="$private_key_path" \
  -P cmake/release/SignDeliverySet.cmake

"$private_tools/delivery-set" -mode verify -root "$delivery_root" \
  -selection "$selection" -allowed-signers "$allowed_signers" \
  -profile "$profile" -signer "$signer" -report "$new_signed_report"
```

The CMake helper validates before signing, refuses overwrite, checks stable index
bytes and verifies against the independent policy. A signing failure may leave a
signature for diagnosis; do not blindly retry or delete evidence. The key path is
an explicit custodian input, never a field loaded from the index. Fixtures use
disposable keys only.

## Focused qualification

```sh
GOWORK=off GOENV=off GOTOOLCHAIN=local go test -race -count=1 \
  tests/launch-acceptance/delivery-set.go \
  tests/launch-acceptance/delivery-set_test.go
```

The existing package-release harness runs these tests on its selected profiles.
Cases include independent versions, closed schema/selection/source and file
bindings, missing secondary artifacts, dependency cycles, unsafe paths/modes,
hardlinks/symlinks, wrong policy/key/principal/namespace, changed signed bytes,
immutable output and the real CMake helper. These are local fixture results,
not Anaconda distribution, actual publisher trust or final acceptance.

## Read-only Go owner qualification

The owning modules contain qualification-only `tests/release-owner` harnesses.
They call SSIAG `InspectExecutable`, STAV `VerifyExecutable` with user scope,
and Accordare `Inspect`; they do not add an installed product command. They
refuse changed, linked, foreign-owned or group/other-writable files and noncanonical
paths, bound reads, and observe the binary and receipt before and after the owner
reader. This mode admits private **user-owned prefixes only**; system installation
qualification remains separate. A quiescent directory is required.

The existing `.github/scripts/build-mvp-go-identities.sh` now also builds these
harnesses from each clean source archive with the same independent package version,
exact Go toolchain and public module graph as its product binary. It emits
`<component>/owner-tool-build.json` beside the build, with source commit/tree,
source archive digest, tool digest, adapter, version and host target. Retain the
recipe logs and module verification. Supply that reviewed build record and its
executable independently of the downloaded payload; the record is an operator
trust input, not a self-authenticating attestation.

```sh
"$private_tools/delivery-set" -mode inspect-owner -root "$delivery_root" \
  -selection "$selection" -allowed-signers "$allowed_signers" \
  -profile "$profile" -signer "$signer" -artifact "$artifact_id" \
  -owner-tool "$trusted_owner_tool" -owner-build "$trusted_owner_build_record" \
  -report "$new_owner_report"
```

Only three fixed adapter IDs are implemented by this mode:
`ssiag-receipt-v2`, `stav-receipt-v2`, `accordare-receipt-v2`.
The payload never selects a program. The caller-selected executable must match
the independently supplied build record, source, component, package version and
target. The harness is invoked with only the fixed `-binary` argument after
signature and complete byte verification, with a minimal environment, one-minute
deadline and bounded output. Its raw receipt hash, canonical owner receipt digest,
binary hash/size/mode, paths and identities are checked separately. The entire
signed set, tool and build record are checked again afterward.

The `symphony.release.delivery-owner-verification.v1` result records the index,
artifact, source/profile, build record, executable and raw inspection hashes.
`selected_artifact_owner_verified=true` means that the actual owner reader ran
against the selected bytes. Fresh-host, provider, release and publication flags
remain false. Unsupported adapters fail; they never inherit another artifact's
success. A later consumer must reverify bytes before use and must execute the
reviewed harness again where its acceptance contract requires it.

Focused tests include real owner install/inspect/removal with unrelated-file
sentinels and no operational state, plus an explicitly inert subprocess fixture
for signature, dispatch, identity, omission and post-inspection drift boundaries.
Those are source/fixture results, not final-host or provider acceptance.

`qualify-go-owners.py` consumes the clean candidate recipe's actual binaries and
owner tools, stages all three via their existing install CLIs in a new private
prefix, signs the set with a disposable fixture key, runs all owner adapters,
rejects a tampered receipt and removes a disposable copy while preserving a
sentinel. Its home/config/state/runtime paths are private; no service is started.
The key is destroyed, and safe logs/results remain. The release-qualification
workflow runs this after candidate identity builds on its four source-test lanes.
The observed target uses an explicit local-fixture profile and never satisfies a
core final-host acceptance cell merely because the machine has the same CPU.

## Native candidate adapter

`inspect-native` calls the existing `candidate-host` tool, which invokes the
reviewed `VerifyCandidate.cmake` before and after bounded extraction. It never
executes extracted programs. Supply the exact native artifact signer and policy
separately from the delivery-index signer and policy, even when the approved
values happen to match.

The independent `symphony.release.native-verifier-tool.v1` record contains
`protocol`, `tool_sha256`, `go_version` (exactly `go1.26.5`) and
`source_files_sha256`. Its closed source map includes `candidate-host.go`,
`cmake/release/VerifyCandidate.cmake`, `Support.cmake`, `StrictJSON.cmake` and
`cmake/SymphonyReceiptV2Support.cmake`, using repository-relative keys. Build the
host tool from these reviewed source bytes; keep this record and executable
outside the delivery. The record binds observed source bytes, not an invented
clean commit or a publisher attestation. CMake/OpenSSH on the operator's selected
PATH remain trusted host tooling.

```sh
"$private_tools/delivery-set" -mode inspect-native -root "$delivery_root" \
  -selection "$selection" -allowed-signers "$index_policy" \
  -profile "$profile" -signer "$index_signer" -artifact "$native_artifact_id" \
  -owner-tool "$trusted_candidate_host" -owner-build "$native_tool_record" \
  -owner-repository "$reviewed_verifier_source" -owner-work "$new_extraction_work" \
  -native-allowed-signers "$native_policy" -native-signer "$native_signer" \
  -report "$new_native_report"
```

The result binds archive, manifest, independent native policy, exact source/tree,
profile/OS/architecture, dependency and identity maps, selected component versions,
actual extracted receipt closure, verifier executable/source hashes and both host
and unpacked-verification reports. The tool checks all delivery bytes again after
extraction. Each native artifact selects one version per component; side-by-side
versions use distinct artifacts. The eight-file candidate includes the existing
`SHA256SUMS`; its contents must agree with the actual archive and manifest hashes.
The real package integration test exposed its omission from the initial seven-file
draft contract, which this increment corrects.

`selected_artifact_owner_verified=true` is extraction/receipt evidence only.
Fresh-host/runtime/provider execution, installer/lifecycle qualification, human
acceptance and publication remain separate. `qualify-native-delivery.py` and the
existing package-release harness exercise actual signed native extraction,
independent-policy refusal and immutable work directories using disposable fixtures.
