# Native package handoff and candidate trust

This is the MVP packaging path for immutable receipt-v2 native payloads. It prepares and verifies candidates; it does not publish packages, approve a release, enroll a signing identity, configure Anaconda, or activate credentials/providers. Existing Go package installers and their independently owned receipt formats remain separate.

## Ownership

| Surface | Writer and responsibility |
| --- | --- |
| Private build/staging root | Build or package manager installs files, performs configured loader-path edits, strips/signs the final payload, and seals its exact-byte receipt. |
| Package-manager cache/prefix | Package manager owns its archive, metadata and removal. It must not claim a qxctl-managed root or rewrite a committed payload. |
| qxctl-managed installation root | The authenticated lifecycle circuit stages/copies exact receipt-owned files, serializes ownership, retains side-by-side versions and performs selected retirement. Direct CMake install/uninstall remains fenced out. |
| Runtime/user state | Its native owner retains authority. Package removal touches only validated receipt-owned files and preserves foreign/user state. |
| Release approval and publication | External release authority approves an exact source/profile/unit and performs publication. SODV records the governed evidence; candidates are not completion records. |

A package manager may keep a finalized payload as a read-only source for qxctl. It must never install directly over a managed root, bypass its fence, or remove files qxctl owns. Updating a package-manager cache does not switch an active qxctl selection. A receipt's digest identifies immutable installed bytes; a version label alone does not.

## Finalize before sealing

Ordinary CMake installation is unchanged. To opt into a source-selected transformation, configure with an absolute `SYMPHONY_PACKAGE_FINALIZER` script. The script's SHA-256 is captured during configuration. The install must explicitly name the same private root through `SYMPHONY_PACKAGE_STAGE_ROOT`; the preflight checks that declaration, script hash, immutable receipt and shared-root fence before copying files.

For example, from a clean checkout:

```sh
cmake -S libraries/knowledge-vector-engine-cpp -B /absolute/private-build \
  -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/absolute/private-stage \
  -DSYMPHONY_PACKAGE_FINALIZER="$PWD/cmake/release/FinalizeNativePayload.cmake"
cmake --build /absolute/private-build
cmake -DSYMPHONY_PACKAGE_STAGE_ROOT=/absolute/private-stage \
  -P /absolute/private-build/cmake_install.cmake
```

The finalizer receives only the configured component/version, stage path and owned-file declarations. It is trusted build code, not a downloaded package hook. It runs after CMake's configured loader edits and before any receipt hash. The supplied native finalizer strips executable/shared-library files; on macOS an explicitly supplied `SYMPHONY_CODESIGN_IDENTITY` also signs and verifies them. `-` is ad-hoc testing only. Static archives, headers and metadata are not silently transformed. No runtime or provider activation occurs.

Build-local scripts require a quiescent administrator-controlled private stage. They reject existing path redirections; they do not replace the authenticated lifecycle circuit's concurrent filesystem protections. A failed transform leaves no committed receipt. Discard that stage or repeat a complete, verified installation from the exact source; never reseal arbitrary edited installed bytes. Once a receipt exists, reinstall is refused. Use a separate version or exact unmanaged-package uninstall. Managed-root reclamation always goes through qxctl.

The hook does not invent correct loader paths. Each release build must select relocatable dependency paths, retain dependency/license provenance and execute the moved package with ambient build paths absent. The packaging regression uses an actual shared library/executable and tests stripping, macOS ad-hoc signing, movement, extraction, version coexistence, interruption, retries and ownership fences. It is infrastructure evidence, not execution qualification of every final MVP download.

## Candidate evidence

`cmake/release/PrepareCandidate.cmake` accepts absolute `RELEASE_STAGE`, a new `RELEASE_OUTPUT` outside the source/payload roots, clean `RELEASE_REPOSITORY`, `RELEASE_PLATFORM_EVIDENCE` and `RELEASE_BUILD_EVIDENCE` paths.

The platform observation comes from L08's collector and must identify the same source commit/tree and native OS/architecture as every receipt. The build evidence must bind that commit/tree and profile ID. It must retain the actual compiler, dependency and test/loader observations for the selected payload. The preparer binds supplied evidence; it cannot prove that the asserted build happened or that all required tests ran.

The output contains:

- `candidate.tar`: the exact receipt/file closure, with no symlinks, duplicate ownership, managed-root markers or foreign files.
- `candidate.json`: source commit/tree, platform and build evidence digests, dependency/identity catalog digests, component versions/receipt digests, every installed file hash/size and final archive hash/size.
- The bound platform/build observations and source catalogs, plus `SHA256SUMS`.

This component inventory describes shipped receipt-owned files. The dependency map separately declares source/build dependencies. It is not a complete discovered runtime dependency or license audit; redistributed external libraries must have their own reviewed provenance, notices, ownership and loader proof. Archive creation makes no cross-host byte-reproducibility claim. Do not modify payload files after sealing.

## Sign and independently verify

Sign the exact candidate inventory with an explicitly selected existing key:

```sh
cmake -DRELEASE_CANDIDATE=/absolute/candidate \
  -DRELEASE_SIGN_KEY=/absolute/publisher-key \
  -P cmake/release/SignCandidate.cmake
```

This uses SSHSIG with namespace `symphony-release`. The signature authenticates inventory bytes, which bind the archive and evidence digests. It is not a direct signature over the archive. No production key is generated by this recipe.

A recipient must obtain the approved signer identity, fingerprint and allowed-signers policy independently of the download:

```sh
cmake -DRELEASE_CANDIDATE=/absolute/download \
  -DRELEASE_ALLOWED_SIGNERS=/absolute/trusted/allowed-signers \
  -DRELEASE_SIGNER=approved-publisher \
  -DRELEASE_EXPECTED_PROFILE=ubuntu24.04-arm64 \
  -DRELEASE_UNPACKED_ROOT=/absolute/private-unpacked-payload \
  -DRELEASE_VERIFICATION_OUTPUT=/absolute/new-verification.json \
  -P cmake/release/VerifyCandidate.cmake
```

The verifier checks the signature, expected profile, bound archive/evidence bytes and, when supplied, the exact unpacked receipts/files. It never extracts or executes an unverified download. The unpacked-root comparison establishes that the caller-supplied directory matches the signed inventory; it does not prove that directory came from the archive. Independent release verification must also establish the archive-content relationship after safe private extraction. Without an unpacked root it reports only download-byte verification. Its optional record retains the observed signing-key fingerprint, principal, trusted-policy digest, signature/inventory/archive digests and verification time/scope. Successful verification is not publication authorization, fresh public retrieval evidence, ABI/loader qualification, or provider readiness.

Developer ID selection, notarization/stapling and trusted public download acceptance remain release qualification work. Any operation changing payload bytes must precede their receipt; any operation changing the distributed archive must precede its final digest and inventory signature. A final release record requires separate external authorization and independently observed public completion under the governed SODV companion.

## Recovery and revocation

On a failed install, preserve diagnostics and the immutable source selection; reconcile through qxctl when the target is managed. Never delete foreign files to make a retry pass. Preserve previous exact versions until the applicable runtime references/ownership claims permit retirement.

If a signing key or artifact is revoked, stop distribution of that exact digest, update the independently distributed trust policy/revocation evidence, and publish a replacement under a new immutable artifact identity and fresh authorization. Preserve the original record and evidence for review; do not overwrite completed ledger entries or reuse a changed artifact's digest/version record. Reverify cached candidates against the current approved trust policy before installing. Removing a compromised signer from that policy makes this verifier reject its signature.

The sprint uses Ubuntu ARM64 as the accepted ARM qualification target. Physical DGX Spark testing is a later optional observation, not an inferred result of hosted tests. Native Windows is unsupported.

