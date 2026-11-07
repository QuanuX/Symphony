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
