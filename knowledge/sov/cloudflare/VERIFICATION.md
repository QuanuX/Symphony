# C0/I1 local verification — November 7, 2026

Implemented on isolated branch `codex/cloudflare-native-c0-i1`, starting from refreshed main `3d32f149ef554dd4695edf3dc3fd326d05f29324`. C0's first source record is commit `a00d4eb9`; its complete catalogue and system-library observations are included with I1. This is source and local qualification evidence, not a merged release or live-provider claim.

## Implemented scope

Independent C++26 adapter 0.1.0-dev; twelve exact Go qxctl commands; receipt-owned owner declaration, closed input/request/result schemas and safe templates; offline verified bundle installation, selection, update, retained rollback and removal; durable plans, local attempt settlement/reconciliation; bounded offline cf search/schema; nonsecret context inspect/proposals. Native C++ generation and bundle-preparation tools obey repository language policy. The prepared bundle has exactly the same digest as the independently assembled C0 bundle.

New explicit invariant-registry v4 admits the separate owner protocol. Existing formats retain their original rules and reject the new adapter format. SSFV registers two experimental source features and twelve command identities, with exact administration mappings and SKVI routes. The root summary is regenerated from those sources.

The direct SSH v1 adapter, its contract, qxctl sovremote implementation and sov_ssh.go have no source diff from the baseline. There is no transport-profile flag or exception in I1. Sudo-gated profile administration begins at I2; new connections will remain user processes.

## Focused completed checks

| Boundary | Result |
| --- | --- |
| Native owner projection | Exact generated schema/template/Go operation projection passed. |
| Native process fixtures | Seven case groups passed: strict framing/bindings, discovery, lifecycle/CAS/replay, changed dependencies and unsafe files, exact context/unqualified-execution refusal, committed/partial-copy/retirement-gap reconciliation, store/plan substitution. |
| Installed native consumer | Complete receipt-owned file verification, native process discovery and unknown-input refusal passed. |
| Go consumer | Independent digest vectors, duplicate/unsafe numeric/Unicode rejection, closed result schema, conservative lost-mutation result, installed receipt dispatch and tampered-schema rejection passed. |
| Registry boundary | Go invariant-registry package and native invariant-ownership tests passed, including v1–v3 compatibility and v4 owner-protocol admission. |
| Command surface | Focused command registry, complete SOV presentation and manifest parity checks passed; 487 stable identities include the twelve new leaves. |
| Feature administration | Native feature-administration tests passed; source mappings remain explicit. |
| Repository validation | Completed with exit 0 after regenerating the root summary; four retained historical-reference warnings concern earlier SCLV paths, with no violations. |
| Real pinned cf/Node | Nine offline checks passed against the native-prepared exact bundle: native/qxctl parity, safe text, import, full inventory, search under hostile ambient settings, schema/C0 parity, runtime/dependency drift refusal, context proposal and cleanup. No provider call or SSH connection was made. |
| Direct SSH v1 | Existing binding/canonical/lost-response Go cases plus independent carrier option/path tests passed. |
| Adapter removal | Common receipt-bound uninstall refused altered owned content, removed the exact package after restoration, retained an unrelated file and accepted already-absent completion. |

Representative reproducible entry points (all build/install paths are explicit):

```sh
cmake -S modules/sov-cloudflare-adapter -B /absolute/build -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/absolute/prefix
cmake --build /absolute/build
ctest --test-dir /absolute/build --output-on-failure
cmake --install /absolute/build
/absolute/build/cloudflare-installed-tests /absolute/prefix
/absolute/build/cloudflare-prepare_bundle --node /absolute/verified-node --cf /absolute/script-disabled-cf --output /absolute/new-bundle
/absolute/build/cloudflare-qualification-tests /absolute/qxctl /absolute/prefix /absolute/new-bundle
# From tools/qxctl, select the installation explicitly for the installed test:
SYMPHONY_CLOUDFLARE_TEST_PREFIX=/absolute/prefix go test ./internal/sovcloudflare ./internal/invariantregistry -count=1
go test ./internal/sovremote ./cmd/qxctl -run 'TestSSH|TestCommandRegistry' -count=1
```

## Exact local identities

Platform: macOS amd64. cf 1.0.0-beta.28; Node 26.12.0. The host's older Node 26.10.0 was not used. Full archive, dependency/license and build-tool provenance is in `knowledge/sov/cloudflare/c0/manifest.json`.

- Toolchain bundle: `sha256:02dd198aa1fc6a3e8e7a8ec1b6a26ef75b4f47619428085aed6306060d4dc813`.
- Initial I1 adapter executable (real CLI qualification): `sha256:0d8734f8116c7dac7d0e2b912ea31adae6c1bba5339c3ece76428c9c18640496`.
- Initial I1 installed adapter receipt: `sha256:28e10baaa61ca5254b7eb76124aa6096cfe01a1211612f32682c5f88d68ffeb7`.
- qxctl test executable: `sha256:86097d5317e1d5d8d35636653670780b0d21a39f6e3f7bf3861b0b642a83eff9`.

## Open qualification

C0/I1 satisfies the local foundation slice, not all of M1-01 or I6. Context apply, provider reads/mutations, protected credential/result channels, Tunnel token rotation, connector lifecycle, sudo-gated transport profiles, Access sessions and live SSH transport remain unavailable. All seven mandatory M1 requirements stay open until their actual I2–I6 evidence exists. Workers/project execution remains I7.

Linux/ARM and macOS ARM builds are not locally qualified. A credential-free hosted workflow covers the local fixtures; remote checks are not awaited after push, per the user's instruction. No full SCV/SHV, provider, production or multi-host campaign was run. Source checks do not establish account entitlements or live-provider acceptance.

Canonical SCLV post-merge closure is not asserted before merge. This dated source record provides the completion inputs for that later closure; no merge, release, deployment or provider authority is inferred.

## Final journal strictness follow-up

Status, apply admission and recovery now parse only the four exact attempt fields and bind the stored plan identity. Unknown fields, mismatched identities and invalid stage/effect values reject without returning the stored text. A sentinel-bearing corrupted-journal test covers status and recovery. The seven native contract groups and the receipt-backed native/Go consumer checks passed for this follow-up. The real cf search/schema paths were unchanged; their nine-check evidence above remains tied to the initial I1 executable.

- Reviewed adapter executable: `sha256:16f7e15e6ad38655d2760e935b921821ce430ca173131580fe2b512a8b87615c`.
- Reviewed adapter receipt: `sha256:e212b803dfbf0b03a6194c378c9d76faa2b9f6ccdce313884ad1de28840af065`.

## C0/I1 shared qualification repair — November 7, 2026

PR #189 at `229fd6ba415cd2f6706b1efbe7b3fe6feae62a8c` completed with 34 successful and 20 failed checks. CodeQL and the three Cloudflare jobs passed. Sixteen SDK/DuckDB jobs stopped at the shared dependency-map gate because the adapter root was not registered. Four Go-release jobs rejected the unclassified `TestCloudflareInstalledReceiptProcess` skip; their retained events contain no failed Go test results. These were integration omissions, not provider or SSH failures.

The repair registers the component, exact foundation dependency, build order, cloud qualification group, owner/schema inputs and optional C0 runtime dependencies. Four changed source hashes and their applicable catalogue records are refreshed after review. Native tests and the installed consumer are included in the release manifest. The source-only test classification is paired with mandatory installed cloud execution; installed skips and missing tests remain failures. The recorded native totals now match the actual selection: 59 roots, 172 CTests and 17 direct suites.

`SYMPHONY_CLOUDFLARE_INSTALLED=ON` requires `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` without source fallback. Shared qualification and the dedicated workflow select this mode; the default source-build mode remains available. No adapter runtime, qxctl operation, dependency pin or SSH v1 source was changed.

Focused local results on macOS amd64 with Go 1.26.5, CMake 4.4.3 and Apple Clang 21:

| Boundary | Result |
| --- | --- |
| Dependency projection and release manifest | Passed: 71 component roots, 59 native components, 1,110 source hashes; exact release selections validated. |
| Independent installed SDK | Built foundation separately; adapter configure passed the existing imported-target, exact-version, selected-prefix and no-source-fallback proof. |
| SDK refusal | Missing/disabled required package and supplied incompatible 0.2.1 package both rejected during configure. |
| Source and installed builds | Both passed owner projection and seven-group native contract CTests; installed CTest discovery exactly matched the manifest. |
| Installed receipt and native consumer | Receipt self-digest, every owned file, executable identity and native process consumer passed. |
| Installed Go consumer | Actual shared release runner, cloud selection, race-enabled: one required test passed, zero skips. |
| Qualification result gates | Classified source-only skip remains separate from coverage; missing, skipped or incorrectly bound installed execution is rejected. |
| Retained release events | Re-evaluated all eight module event streams from each of four failed-run artifacts (32 summaries). The corrected manifest passes; removing the new classification reproduces the original qxctl error on each runner. This is evidence replay, not a fresh multi-platform run. |
| SSH v1 | Focused race-enabled carrier, consumer and command-surface regressions passed; SSH source diff against baseline `3d32f149` remains empty. |

The installed native executable is byte-identical to the reviewed journal-hardening artifact: `sha256:16f7e15e6ad38655d2760e935b921821ce430ca173131580fe2b512a8b87615c`. Its receipt, including updated installation documentation, is `sha256:6c24898f4b861e900d3f132ec3aaeb2a6b7080d1782e9bf6825f06680df53b15`. The existing real-CLI evidence retains its original stated identity and scope; no new provider or real-CLI execution is claimed here.

A local shared Go-cache checksum mismatch was resolved by using a fresh task module cache and normal public-proxy/checksum verification. No checksums or dependency pins were relaxed or changed. Evidence is retained separately in the workspace's `implementation/cloudflare-c0-i1/ci-repair/` directory. Full SCV/SHV, complete installed-group and fresh remote campaigns are deferred. Remote checks are not awaited after pushing this corrective increment. I2 remains unstarted.

## SKVI capacity and registration repair — November 7, 2026

The next inspected head, `d806924161c1e29103196d3627fe646f2763f843`, completed with 37 successful and 17 failed checks. All six CodeQL jobs, the dedicated Cloudflare foundation jobs and all four installed cloud SDK jobs passed. Nine jobs stopped at the separate release identity catalog, four knowledge jobs exceeded the SKVI entry quota, and four quantitative jobs rejected the Cloudflare feature-file envelope during SBV administration validation.

The adapter is now registered in the candidate identity catalog with its existing `0.1.0-dev` package identity and no invented SDK export. Its feature-file envelope now includes the required top-level `source_scope`; the normalized feature record and registry digest are unchanged. Reviewed source and schema hashes in the dependency projection are refreshed.

SKVI v2 no longer imposes an index-entry count ceiling in native parsing, result schemas or qxctl admission. The original quota budgeted a complete projection and initially matched the snapshot-file limit. SKVI now snapshots owner manifests separately. Check/proposal admission can therefore follow the index-file byte bound independently of whether a complete projection fits one process response. The 4 MiB index/response byte bounds, 65,536 process JSON-event bound, field/evidence/manifest/path limits and deadlines remain. Oversized projections fail explicitly without truncation. Legacy SKVI v1 schemas and its qxctl 1,024-entry limit remain unchanged.

Focused local verification on macOS amd64:

| Boundary | Result |
| --- | --- |
| Candidate identity catalog | Passed: 71 roots, six Go commands and 24 owner declarations. |
| Dependency and release-test projections | Passed: 59 native components, 1,110 reviewed source hashes, and exact release selections. |
| Native SKVI, built against the installed foundation | Both CTests passed. A 2,500-entry projection round-trips; 3,000-entry check/proposal succeeds while an over-budget projection is refused. Independent index-byte and response-byte refusals pass. |
| Actual SKVI process and installed qxctl | Check and project pass for all 2,049 repository entries. The installed receipt-backed check reports zero violations and four retained warnings; projection is complete, noncanonical and rebuildable. |
| Native SSFV and feature administration | Fresh SSFV process smoke and validator feature-administration test passed. |
| SBV administration | Fresh current-source native target built; the single affected `sbv-administration-contract` CTest passed. |
| Go client and SSH regression | Race-enabled SKVI, JSON-bound, SSH/SOVSSH and command-registry selections passed in the qxctl command, knowledgeengine and sovremote packages. |
| Cloudflare installation | Fresh receipt-owned install and native installed consumer passed. Corrected feature metadata is covered by the new receipt. |
| Source validation | Passed with zero violations and four retained historical-reference warnings. |

The Cloudflare executable remains byte-identical to the reviewed I1 executable (`sha256:16f7e15e6ad38655d2760e935b921821ce430ca173131580fe2b512a8b87615c`). Its new receipt digest is `sha256:66ec5b6703cbd2ab0f918a5d75acf585d24ae68e69afab792528a8e7047da0f8`. The exact installed SKVI executable is `sha256:d00c4bf413972669a6db3d874611dd7ed240a7f9bd63ce28588539a8122d0bb2`; its receipt file hash is `sha256:d6d655cf1bbf51510adb04c906f9acf8214360124b3715ecf735da769db8f7d2`. The current qxctl integration binary is `sha256:2352320fba66d83b6ad78c0a2cd43b9caf2b53a69ea46f7d93ee26f29dd59343`.

SSH v1 adapter/consumer/command source remains unchanged from `3d32f149`. Evidence is retained under the workspace's `implementation/cloudflare-c0-i1/index-repair/`, `ci-repair/skvi-build/` and `ci-repair-2/` directories. These are focused local results, not a fresh full SCV/SHV or installed-group campaign. No new provider calls, transport-profile implementation or live SSH qualification occurred. I2 remains on hold; remote checks are not awaited after push.
