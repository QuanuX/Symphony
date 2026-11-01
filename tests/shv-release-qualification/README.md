# Bounded SHV launch qualification

This required knowledge-group gate exercises the installed SHV PDF adapter and a single deliberately selected hardware catalogue workflow. It does not discover vendor sites or attest that a machine has the described hardware.

Run after the installed knowledge SDK/owner campaign, using Go 1.26.5, CMake and the selected C++26 compiler:

```sh
sh tests/shv-release-qualification/run.sh /absolute/knowledge-prefix /absolute/fresh-work
```

The runner downloads only the exact reviewed assets, builds the current SHV DuckDB connector against the installed foundation, builds the installed PDF replay test, and composes current qxctl, SSIAG and STAV from the declared source workspace. The L05 standalone public Go dependency gate remains separate. Private user-scope SSIAG/STAV processes use synthetic exact grants, private state and short Unix sockets; no provider, credential, production installation or OS supervisor is activated. Processes are stopped on success and failure.

Required hosts are Apple Silicon macOS 15, Ubuntu 24.04 amd64, Ubuntu 26.04 amd64 and Ubuntu 24.04 arm64. Ubuntu ARM64 is the accepted sprint surrogate for DGX Spark. No physical Spark, GPU or native Windows claim is made. A supplemental macOS Intel asset supports local diagnosis only. Every required host must actually run both tests without skips; inspected or downloaded assets are not execution evidence.

## Exact decoder and documentary source

`dependencies.json` selects PDFium **158.0.8125.0**, release `chromium/8125`, from the independent [bblanchon/pdfium-binaries publisher](https://github.com/bblanchon/pdfium-binaries/releases/tag/chromium/8125). This distributor is not Google or Foxit. The lock binds release ID, publisher build-recipe commit, published archive digests, sizes and exact runtime/notice file identities for each architecture. It replaces 153.0.7999.0 only in this current qualification profile; dated 153 evidence and caller-selected retained decoder identities remain unchanged. There is no `latest` URL or version fallback at execution.

The standalone archives disable V8 and XFA. The runtime closure retains the library, VERSION, args.gn, top-level LICENSE and every bundled third-party license; headers are also protected by the archive digest. There is no separate ICU or font data file in these selected archives. Preparation checks the machine architecture and records Mach-O/ELF dependencies, refusing dependencies outside the declared platform runtimes. The Apple ARM library has minimum macOS 13.0, below the required macOS 15 host. System runtime libraries remain host dependencies, not bytes covered by the PDFium library hash. Preparation is not a sandbox or a publisher-signature verification.

The source is [AMD publication 69290, revision 1.00, April 2026](https://www.amd.com/content/dam/amd/en/documents/legal/epyc-app-export-control.pdf), specifically Table 8 on zero-based page 12. The lock requires 738352 bytes and SHA256 `5178cba64d77199eb7b8e7baa2cf5f6c15308784939396a0d3338c0fb81e04f7`. It is one documentary lineage. The parser expects 28 OPN rows; this workflow selects only model 7313 and OPN 100-000000329 for the catalogue. It does not interpret the source's APP metric as measured performance, infer a Product ID Tray namespace equivalence, or claim complete hardware coverage.

If the publisher is unavailable or has replaced those bytes, preparation fails. A caller may explicitly select a retained copy with `SHV_RELEASE_SOURCE=/absolute/exact.pdf` and a retained exact platform archive with `SHV_RELEASE_ARCHIVE=/absolute/exact.tgz`; the same digest and size checks apply and acquisition mode is recorded. No PDF or decoder binary is vendored or published by this gate. These inputs are distinct from the native adapter receipt and uninstall never owns them.

## Required exercised boundaries

The installed native PDF replay checks receipt-v2/compiled interface admission, extraction, graph projection and verification, namespace forgery refusal, incorrect decoder hash refusal and changed source-byte refusal. Its six subprocess calls must succeed or refuse exactly as expected.

`TestInstalledSHVBoundedCatalogueWorkflow` then:

1. Independently replays the pinned extraction and refuses a wrong decoder digest.
2. Retains a source onboarding intent when SSIAG is absent, then recovers it through real private SSIAG/STAV, verifies the exact kernel-authenticated policy event by correlation/request/resource and confirms replay creates no duplicate policy event. The event audits permission; it is not a source-write receipt.
3. Compiles a typed OPN profile, diagnoses mapping declarations, refreshes the activated source from the exact retained bytes/revision, and verifies coverage for the selected model. AMD.OPN is supported; the unasserted Product ID Tray namespace remains unresolved.
4. Builds a partition and manifest, commits the verified graph to the installed DuckDB connector, and constructs a catalogue publication preview bound to the exact retained bundle, source, partition and graph snapshot. It checks that preview leaves the publication head absent.
5. Alters only its private source copy and requires refresh verification and publication preview to refuse it.

The general Go suite may skip this opt-in installed test when any of `SHV_RELEASE_PREFIX`, `SHV_RELEASE_STORE_PREFIX`, `SHV_RELEASE_PDF_REQUEST`, `SHV_RELEASE_SSIAG`, `SHV_RELEASE_STAV` is absent. The dedicated runner supplies every input and its event validator rejects any skipped, failed or omitted test. The exact required profile outcomes remain pending until the hosted jobs complete; a local Intel result is supplemental.

`SHV_RELEASE_STORE_PREFIX` permits an explicitly selected installed connector for local diagnosis; it is still receipt/compiled-admission checked. The required runner rejects a supplied `SHV_RELEASE_PDF_REQUEST` and always prepares a new request from the lock, so arbitrary caller-declared decoder/source hashes cannot masquerade as locked asset proof. Retained source/archive inputs are still permitted with identical hash admission. Reused connector selection is recorded as an input, not a fresh connector build claim.

Retain `summary.json`, source/head/tree/status, lock/asset and loader metadata, installed PDF `ACCEPTANCE.json`, catalogue requests/responses, the bound policy query and authority status, and process/test logs. Summaries bind executable and receipt hashes. Retain extracted notices privately with the decoder; do not upload source documents, decoder binaries, caches or private state as CI artifacts. Publication remains preview-only.
