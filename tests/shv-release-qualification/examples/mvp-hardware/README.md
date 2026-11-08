# One documentary hardware starter

This optional source example reuses the exact L18 qualification recipe. It is a small, provenance-bound illustration for caller-supplied tools, not a bundled hardware atlas, coverage policy, operational provider or machine inventory.

The single authority for source and decoder bytes is [the qualification lock](../../dependencies.json). It selects AMD publication **69290**, revision **1.00**, issued **April 2026**, zero-based page **12**, Table 8; source SHA-256 `5178cba64d77199eb7b8e7baa2cf5f6c15308784939396a0d3338c0fb81e04f7`. The adapter checks 28 extracted rows; the example selects only **EPYC 7313 / AMD.OPN 100-000000329**, one documentary lineage. The source's APP quantity is documentary data, not a measured performance benchmark. Product ID Tray equivalence remains unresolved rather than inferred from a similar number.

Freshness is the source's issue/revision plus the exact retained capture and acquisition evidence from the run. A current download time does not make an April 2026 assertion current hardware truth. Do not silently refresh to changed publisher bytes, choose a newer PDFium, or call a stale retained capture live. A replacement document or mapping requires an explicitly reviewed lock/profile revision and replay.

After installing the exact knowledge SDK/owner group, with Go 1.26.5, CMake and the selected C++26 compiler available:

```sh
sh tests/shv-release-qualification/examples/mvp-hardware/run.sh /absolute/knowledge-prefix /absolute/new-private-work
```

The wrapper runs the existing required SHV campaign. It verifies exact assets and receipts, extracts the documentary rows, exercises authenticated source activation/recovery, compiles the OPN mapping, produces partition/graph evidence, and builds a publication preview. It refuses wrong decoder hashes and changed source bytes, preserves the unresolved identifier namespace and confirms the publication head remains absent. It does not publish a catalogue or activate a live provider.

An operator may provide retained inputs with `SHV_RELEASE_SOURCE` and `SHV_RELEASE_ARCHIVE`; digest and size admission remain mandatory. An explicit `SHV_RELEASE_STORE_PREFIX` selects a pre-existing connector under exact receipt admission. The evidence distinguishes selected installations from fresh builds. See [the full recipe](../../README.md) for dependency closure and output fields.

Retain `summary.json`, `assets/assets.json`, the actual request, installed PDF acceptance, `catalogue/summary.json`, mapping/refresh/partition/graph/preview requests and responses, and policy-decision evidence. Use the actual files listed in the campaign summary rather than fabricating missing output. Those records bind captures, source revision, selected model, coverage, unresolved facts and replay. Do not upload the publisher PDF, decoder binaries, private state or caches. No redistribution permission or publisher approval is inferred.

Synthetic grants and private test service configurations remain explicitly synthetic; the AMD document itself is real documentary evidence. A successful run on one host is not proof of all required platforms. Existing knowledge-group CI executes the same campaign on macOS ARM64 and Ubuntu amd64/ARM64. Physical DGX Spark/GPU qualification remains separate.
