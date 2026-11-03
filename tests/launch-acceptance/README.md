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
