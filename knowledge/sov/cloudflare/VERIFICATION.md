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


## I3 remote gate and I4 planning increment — November 8, 2026

Before I4, main and the inspected push were both `38d3ee8593d8943a36fc7b429213ced5c2054a69`. All 12 workflows, 58 jobs and 58 commit checks completed successfully. The complete job-ID and check-ID sets match exactly; all six CodeQL jobs passed. The installed real-OpenSSH profile fixture passed on Linux amd64 and ARM64. The latest chronological workflow was dependency graph submission; it also passed on that same commit. Workflow details: [CodeQL](https://github.com/QuanuX/Symphony/actions/runs/42461083975), [transport profiles](https://github.com/QuanuX/Symphony/actions/runs/42461084020). I3's observed counts agree: 72 component roots, 59 native components, 1,202 source hashes, seven Go commands, 24 owner declarations, 153 features and 495 qxctl commands. An outdated SSFV registry introduction (149 features/67 scopes) was corrected to the machine-checked 153/68; the earlier canonical registry already contained 153 features.

The first I4 increment adds `qxctl sov cloudflare cf resource plan`. This is typed, credential-free planning for 23 named Tunnel, DNS and account-scoped Access profiles. Plans bind the actual non-root principal, full context, selected exact bundle, adapter identity, store state and expiry. No resource plan is executable; the local installer rejects its separate protocol and digest domain. Closed templates exclude API-token and tunnel-secret bytes, and sensitive result classes explicitly identify their required future channel. The selected cf version remains 1.0.0-beta.28 with Node 26.12.0; no provider version or pin changed.

Focused results on macOS amd64:

- Both native CTests pass, including eight contract groups with 23 profiles, unselected-toolchain rejection, context/state/UID mismatch, secret-bearing or unknown fields, page bounds, delete confirmation, DNS/address and tunnel ingress validation, and installer/resource-plan separation.
- Independently installed receipt and native consumer pass. Receipt-backed Go Cloudflare consumer, command registry and affected SSH consumer tests pass with the race detector. Direct SSH carrier option/protected-path regressions pass; SSH v1 source is unchanged.
- Exact real bundle `sha256:02dd198aa1fc6a3e8e7a8ec1b6a26ef75b4f47619428085aed6306060d4dc813` passes 11 offline qualification groups: all 23 native request previews exactly match the pinned CLI's dry-run JSON, and all 23 complete qxctl text results retain unavailable execution/authority. Node permissions deny network, writes and subprocess creation. Ambient project/token/proxy/Node settings are excluded. Provider calls: zero; SSH connections in this local qualification: zero.
- SSFV/source validation has zero violations and four retained historical warnings. Registries, root summary and release selections agree: 496 qxctl commands, 153 features, 72 component roots, 59 native components and 1,207 source hashes.

This completes only the typed planning increment, not I4 or M1. Remaining I4 work includes native credential export/import, authenticated private secret-use and sensitive-result channels, user-only provider execution, actual request-count/retry qualification, bounded provider observations and pagination, admitted mutation journals/reconciliation, Tunnel token rotation resolution and connector lifecycle. expected_observation is a caller reference, not a verified provider observation. No live-provider, full SCV/SHV or full installed-release campaign was run. I3's remote results do not qualify this later increment. Existing C0/I1 evidence above retains its original identities and scope.


## I4 native credential channels and provider execution — November 8, 2026

This increment completes the implemented I4 administration surface: 33 native operations, including the 23 typed provider profiles, local context CAS, bounded provider reads/mutations and separate connector installation/instance lifecycle. An independently installed C++ SSIAG private-store provider supplies purpose-bound export and sensitive-result import over private inherited descriptors. Go qxctl and SSIAG handle only nonsecret control metadata. SSIAG holds the exact mapped-principal policy decision through bounded use and requires committed STAV intent before export; outcome uses its own audit request identity. This explicit private-file backend is opt-in and is not an encrypted vault or protection against hostile same-UID/root processes.

Provider execution requires the selected exact bundle and native provider configuration. It excludes inherited credentials/proxies/Node settings, checks deadline immediately before dispatch, admits only the planned URL/method/body and permits one API attempt per child. Lists report page bounds and coverage. Mutations require a retained complete observation, perform a fresh preflight, journal before dispatch and independently observe the result. Preflight is not atomic provider CAS. Status/reconcile never replay a mutation. An uncertain sensitive-result import acknowledgment remains unresolved for native metadata inspection; it cannot silently complete or replay token retrieval. Connector start runs as the actual user through a retained exact supervisor; authenticated local control stops only its own process group. A running process does not establish provider connectivity.

Focused local results on macOS amd64:

| Changed boundary | Result |
| --- | --- |
| Native credential store and exact SDK | Contract fixtures passed configuration CAS, FD-only import/export, purpose/scope/generation/expiry admission, duplicate refusal, private result import and local revocation. Independent source and installed-foundation builds passed. Exact package uninstall preserved unrelated prefix content. |
| SSIAG use admission | Three focused tests passed exact grant/committed audit, forged and expired controls, policy pinning and lost-completion outcome. Fixtures exercise a real Unix admission server with synthetic audit receipts; they do not claim a live STAV deployment. |
| Resource execution and recovery | Native fixtures passed typed execution, bounded pagination, mutation preflight/readback, replay refusal, stale/scope refusal, revoked credentials, lost-after-effect observation, sensitive import and uncertain-import recovery. Unpublished atomic temporary files do not block future work. |
| Connector lifecycle | Native provider/synthetic-authority/inert-child fixture passed install/configure/start/status/stop/recovery boundaries, with zero provider calls. |
| Pinned CLI request guard | All 23 profiles passed across 28 real CLI processes and 26 synthetic API attempts; zero real provider calls. Exact request matching, delete force flags, wrong-target/expired-use refusal, 429/500/network-error one-attempt behavior and error-body exclusion passed. |
| Production HTTP runner | Real pinned CLI passed private-FD framing, pre-network target refusal, raw-output exclusion and scratch cleanup; zero API attempts. |
| Installed native/Go boundary | Exact receipt, embedded/installed owner equality, native consumer and receipt-backed Go consumer passed. Final owner projection and native contract checks passed. |
| qxctl integration | Changed Cloudflare consumer tests and command-registry tests passed. Provider reads retain explicit authority metadata. The catalog reader admits the expanded inventory within its existing one-MiB byte bound; engine request bounds are unchanged. |
| Repository integration | Source validation reports zero violations and four historical-reference warnings. Dependency/release selections agree: 73 roots, 60 native components, 1,311 source hashes, seven Go commands and 24 owner declarations. SSFV and command inventories agree at 154 features, 69 owner scopes and 516 qxctl commands. |

Final review also corrected CMake dependency tracking for the embedded owner and request guard, so an incremental build cannot retain stale declarations. The installed-consumer regression explicitly compares the embedded and receipt-owned owner. Feature-administration parsing uses its existing four-MiB file budget instead of the smaller generic value-count default.

Exact runtime selection remains cf 1.0.0-beta.28 with Node 26.12.0, bundle `sha256:02dd198aa1fc6a3e8e7a8ec1b6a26ef75b4f47619428085aed6306060d4dc813`, and cloudflared 2026.10.7, executable `sha256:5c55e77da7012cebca19170357e196176c48c9647cc2ead404702fa125ddbd86`. Only the retained C0 macOS amd64 runtime artifacts are admitted; source portability and remote build coverage do not qualify other runtime digests. No real credentials, live account/zone mutations or Cloudflare tunnel/Access sessions were used. Token rotation is absent from the pinned CLI and remains explicitly unsupported with M1-02/I6 open. I5 SSH session qualification, I6 live acceptance and I7 project deployment remain later work.

SSH v1 adapter, consumer and command files are unchanged from the successful first pass; their tests were not rerun. Full SCV/SHV and full release campaigns were not run. Earlier successful checks were retained unless code or an affected contract changed. Evidence is retained separately under the workspace's `implementation/cloudflare-i4/execution/`. Remote results for this commit are not awaited or asserted.

## I4 CI prerequisite repairs before I5 — November 11, 2026

The completed PR runs for `4a1c9a83b72614e5e5cdac9aaee13633f75a3533` included successful CodeQL and protected-transport workflows, but the Cloudflare and dependency workflows failed. Three causes were identified in retained job logs/artifacts: Linux could not deduce a common `std::min` type from `int64_t` and `stoll`; the SSFV canonical-administration test reader applied the generic JSON value limit to the expanded command catalog; and native SDK distribution had no classification for the new private-store SDK.

The deadline comparison now explicitly selects `int64_t`. The test-only canonical file reader retains its four-MiB byte bound while admitting the complete catalog; engine protocol limits remain unchanged. The credential SDK now ships an independent installed consumer and is registered for copied-consumer distribution coverage. A focused inventory regression checks all current SDK registrations without running a distribution campaign.

Focused local checks: the previously failing SSFV target passes; the affected resource-execution target passes; the exact installed private-store SDK consumer configures, builds and runs successfully, including transitive foundation hashing and strict duplicate-key rejection; canonical SDK coverage passes. Dependency projection reports 73 roots, 60 native components and 1,313 source hashes. No unchanged SSH tests or complete installed-release campaigns were rerun. Linux qualification remains the remote runner's responsibility; no new remote result is asserted or awaited. Evidence: workspace `implementation/cloudflare-i5/evidence/`.

## I5 local implementation — November 11, 2026

Base: I4 repair commit `96db8a7240c019a30f17dd1d6d9e0939123e9244` on PR #192.
I5 is isolated on `codex/cloudflare-i5-sessions`. Seven qxctl session leaves bring
the registry to 523 commands; 154 registered features and 73 component roots remain.
The new native helper is independently receipt-owned, with an exact optional runtime
configuration and no default provider activation. SSH v1 code and remote-operation
contracts are unchanged; only the carrier's separate `--session` dispatch is added.

Focused local evidence (macOS amd64, Go 1.26.5, LLVM 23.1.2):

- `cloudflare-access-sessions` passes, covering explicit login, token/identity/expiry
  checks, one-use plans and stale state, renewal failure/success, bounded child
  cancellation, private clear, nonblocking special-file rejection, and a connection
  cache that cannot create a lock/remove its token and contains no organization
  token. The inert child attempts ambient credential/proxy use and emits secret
  sentinels; the test captures both output streams and requires no output.
- Both independent Go session contracts pass malformed controls, unknown/duplicate
  fields, result identity/semantic binding, and uncertain dispatch tests. Protected
  carrier tests reject login after disable/removal while permitting private cleanup.
  Separate dispatch refusal and the changed carrier's SSH v1 options test pass.
- Installed native receipt/hash checks, helper schema parity and refusal without a
  protected runtime pass. qxctl discovery/text parity, command authority, registry
  count, checked-in registry parity and invariant registry consumption pass.
- The existing generated-profile check now also verifies the independent session
  consumer and installed schema projection. SSFV canonical administration passes
  with the changed features/command catalog. Dependency and release-manifest checks
  pass at 73 roots, 60 native components and 1,329 source hashes.
- Source validation reports zero violations and four pre-existing historical
  warnings. Broad release/SDK campaigns and unchanged SSH transport campaigns were
  deferred under the focused increment testing direction. Remote checks are not
  awaited or represented as local evidence.

Live I5/TR02–TR05 acceptance is **not complete**. `symphony-i5.quanux.dev` in the
existing `quanux.dev` zone is the selected test hostname; an isolated temporary
Linux SSH target is the selected target approach. No tunnel, DNS record, Access
application/policy, live session or protected profile was created. Read-only
inventory found no existing test tunnel or SSH Access app, and no suitable running
Symphony test container. The existing connected Symphony installation is an older
read-only demo and was left untouched. The local Mac has no provisioned protected
profile store; `sudo -n -l` reported that a password is required. Live execution
therefore needs administrator-provisioned carrier/helper packages and SSIAG/STAV
scope, an exact target/host key/agent selection, and interactive Access browser
completion. Neither ordinary prefix installation nor fixture receipts substitute
for those trust gates. Other runtime platforms, unattended service-token Access
mode, provider policy-denial tests and I6 token rotation remain unqualified.

Retained local logs: workspace `implementation/cloudflare-i5/evidence/`, including
`access-final-test.log`, `access-final-installed-test.log`, `qxctl-sessions.log`,
`carrier-sessions-final.log`, `registry-final.log`, `generated-contracts.log`,
`ssfv-i5.log`, `dependency-verify.log`, `release-manifest.log`, and
`validator-final.json`. No credential material is included in this evidence.

## I5 CI portability repair — November 11, 2026

Completed runs for `5970b38471a32bfb53ee660714a39f5975efe360` had four
failed jobs across [Cloudflare native local foundation](https://github.com/QuanuX/Symphony/actions/runs/43120351184)
and [MVP dependency map](https://github.com/QuanuX/Symphony/actions/runs/43120351103).
All four failed `cloudflare-resource-execution` on Ubuntu 24.04 (amd64/arm64)
with `profile`; the other nine workflows, including CodeQL, passed.

Clang 18 does not extend the lifetime of the temporary owner JSON through a
range-for loop over its nested array. Three loops now retain a named owner
declaration: resource profile selection, connector plan validation and the
resource fixture. A new regression requires rejection of a malformed connector
plan even when its digest is recomputed, and verifies that rejection preserves
the installation state. This covers the related validation bypass that the
earlier connector fixture did not exercise. SSH v1 code is unchanged.

Focused verification:

- Ubuntu 24.04 amd64, Clang 18.1.3: the original resource test reproduces
  `profile`; the new connector regression fails against the original production
  code. After the repair, both resource execution and connector lifecycle pass
  as an ordinary user against local inert fixtures.
- macOS amd64, LLVM 23.1.2, installed-foundation configuration: both affected
  tests pass. The connector regression's initial fixture-path error was corrected
  before its final passing run.
- Dependency verification passes at 73 roots, 60 native components and 1,329
  source hashes. Only the four changed source hashes were refreshed.

Unchanged SSH/session tests and broad campaigns were not rerun. Linux arm64
was not tested locally; remote results for the repair are not awaited or asserted.
Retained repair evidence: workspace `implementation/cloudflare-i5/ci-repair/`.
These local fixture checks make no new live-provider or runtime-admission claim.

## I4 Clang 18 CI repair — November 11, 2026

PR #192 head `96db8a7240c019a30f17dd1d6d9e0939123e9244` had four failed
Ubuntu 24.04 jobs (amd64/arm64) across
[Cloudflare native local foundation](https://github.com/QuanuX/Symphony/actions/runs/43115145864)
and [MVP dependency map](https://github.com/QuanuX/Symphony/actions/runs/43115145663).
Each failed `cloudflare-resource-execution` with `profile`. The other nine
workflows, including CodeQL and protected transport profiles, passed.

The owner JSON temporary was destroyed before iteration on Clang 18. Three
loops now retain a named declaration for resource profile selection, connector
plan validation and the fixture. This applies the code repair from PR #193
commit `08a8f16d` independently to I4. Its regression also rejects a malformed
connector plan with a recomputed digest and requires unchanged installation
state. No I5 session implementation or SSH v1 change is included.

Retained Clang 18.1.3 / Ubuntu 24.04 amd64 evidence from the PR #193 repair
shows both failures before the fix and both focused tests passing afterward.
Input comparison confirms 62 shared native source, fixture, provider, foundation
and owner files are byte-identical; this is reused evidence, not a new I4 Linux
run. Evidence: workspace `implementation/cloudflare-i4/ci-repair/` and
`implementation/cloudflare-i5/ci-repair/`. Live-provider and runtime-admission
qualification remain unchanged. Remote results for this repair are not awaited.

Fresh I4 verification on macOS amd64 / LLVM 23.1.2 with installed foundation
and credential SDKs: `cloudflare-resource-execution` and
`cloudflare-connector-lifecycle` both pass (2/2). Dependency verification passes
at 73 roots, 60 native components and 1,313 source hashes; only the four changed
source hashes were refreshed. Unchanged SSH tests and broad campaigns were
not rerun. No new Linux arm64 result is claimed.

## I6 local integrated gate — November 12, 2026

I6 starts from integration `73829980`, retaining both I4/I5 repair records.
Runtime code and SSH v1 are unchanged. The new local evidence reducer validates
hashed CTest/Go artifacts, complete events and required test names; missing,
skipped, malformed or failed coverage cannot pass. Its five regression tests
pass, and its support matrix binds all 23 current resource profiles and M1-01
through M1-07. It explicitly cannot certify live acceptance or release authority.

Fresh macOS amd64 evidence with LLVM 23.1.2 and explicitly selected Go 1.26.5:

- Native administration/Access CTests: 5/5; native credential contract: 1/1.
- Profile suite: 40 passing test events and one explicit installed-root skip.
- Carrier suite: 43 passing test events; the Linux-only installed OpenSSH case
  is not compiled on macOS. The generated wire package has no test files.
- qxctl consumer/boundary suites: 181 passing test events and one installed SBV
  target skip; the generated wire package has no test files. Discovery: 22
  passing events. SSIAG/audit: 16 passing events. Parent/subtest events are not
  summed into independent acceptance-case totals.
- Installed receipt/owner/helper-schema and default-runtime refusal pass.
  Disposable removal removes all 148 owned files and its receipt, preserving
  an unrelated sentinel. Generated profile/carrier contracts, native SCV
  installed-process integration and native SSFV projection pass.
- The pinned CLI guard passes 23 profiles in 28 processes and 26 synthetic API
  attempts, with zero real provider calls. Production runner framing, target
  refusal before network and scratch cleanup pass. Installed real-CLI
  qualification passes all 11 checks, including all 23 typed plans against
  pinned CLI dry runs, hostile ambient settings, runtime-drift refusal and
  cleanup, with zero provider calls or SSH connections.
- Dependency/release validation passes: 73 roots, 60 native components and
  1,335 source hashes. Source validation has zero violations and four existing
  historical warnings. Commands/features remain 523/154.

The broader offline qualification exposed two pre-I4 assertions that still
expected resource administration/local context persistence to be unavailable.
Those assertions now verify the exact implemented descriptor's required SSIAG
authority and credentialless local context planning without provider dispatch.
The retained runtime copy stalled on filesystem reads. A fresh copy reconstructed
from the pinned lock and retained Node artifact matches all 2,337 file hashes;
versions are unchanged and no package scripts ran. Earlier PATH-Go exploratory
runs and failed environment/probe attempts are separate from the exact-toolchain
results above.

The local reducer correctly reports **incomplete** for the required skipped or
platform-missing cases. I6 remains **open**, including native private Tunnel token
rotation and LIVE-01/TR02–TR05. The pinned CLI has the underlying `tunnels edit`
PATCH primitive: the missing piece is native secret-body/result/recovery handling,
not an upstream absence. See `I6-QUALIFICATION.md` for this correction, current
read-only live preflight and exact next actions. No live resource was created,
no profile was privilegedly published, and no canonical SCLV closure is claimed.
Retained evidence: workspace `implementation/cloudflare-i6/evidence/`. Remote
results for this increment are not awaited.

### PR #194 focused SSH fixture repair — November 12, 2026

Protected transport profile run `43168157246` failed only on macOS Intel in
`TestSSHProcessReplyBindingAndLoss/wait`. Its 250 ms deadline started before
the selected executable was copied and hashed; the 380 ms invocation reached
process start with an expired context instead of exercising reply loss.

The fixture now acknowledges successful invocation decoding through a private
ready marker before the test cancels. A separate 30-second watchdog bounds
startup failure; the assertions require acknowledged startup, explicit
cancellation, an unverified lost reply, no target result and no automatic retry.
Production SSH v1 code and workflow coverage are unchanged.

Only `TestSSHProcessReplyBindingAndLoss/wait` was rerun, using Go 1.26.5 on
macOS amd64 with the repository source workspace and `-mod=readonly -count=1`:
pass, 0.43 seconds. No other test or full suite was rerun. Only the changed test's
dependency source hash was refreshed. Remote results are not awaited.
