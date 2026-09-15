# Source maintenance acceptance

This focused specimen exercises exact installed SCV `0.10.0-dev` through qxctl, covering source adoption, retained refresh failures, source-bound packages and caller-directed reassessment. The C++ engine, native schemas, command routes and defaults are unchanged. qxctl separately repairs durable SSIAG/STAV correlation for readable operation IDs; see [source administration](../../../../knowledge/scv/SOURCE-KNOWLEDGE.md).

The September 11 DigitalOcean App Platform limits capture already followed the `index.md` redirect. The September 13 direct `index.html.md` capture has an identical body. This case removes known indirection; it does not establish a new migration or migration date. Publisher revision labels, HTTP validators, capture time, source generation and profile/package/engine versions remain distinct. No upstream API release is selected. The official Quickstart is a complete but unsuitable same-identity proposal, never adopted; the engine does not authenticate semantic publication continuity from a caller’s identity assertion.

## Offline runners

Build the four C++ acceptance executables from the repository root, then select an explicit qxctl executable and exact `.10` installation prefix:

```sh
cmake -S tools/qxctl/tests/scv-source-maintenance -B /absolute/maintenance-build -DCMAKE_BUILD_TYPE=Release
cmake --build /absolute/maintenance-build
```

Run the offline commands from this directory with new absolute output paths. All operations go through qxctl. These commands fetch nothing, start no services, select no protected head and run no unrelated tests.

```sh
/absolute/maintenance-build/scv-verify-retention --qxctl /absolute/qxctl --prefix /absolute/scv-prefix --old-capture fixtures/old-capture.json --fresh-capture fixtures/fresh-capture.json --out /absolute/new-retention-output
/absolute/maintenance-build/scv-verify-semantics --qxctl /absolute/qxctl --prefix /absolute/scv-prefix --old-capture fixtures/old-capture.json --fresh-capture fixtures/fresh-capture.json --insufficient-capture fixtures/insufficient-capture.json --profile fixtures/limits-profile.json --out /absolute/new-semantics-output
/absolute/maintenance-build/scv-verify-pack --qxctl /absolute/qxctl --prefix /absolute/scv-prefix --old-package fixtures/old-package --fresh-capture fixtures/fresh-capture.json --out /absolute/new-pack-output
```

The retention runner uses a private new corpus/TOPS namespace and authors an explicitly synthetic failed capture: no live failure request occurred. It checks last-complete versus latest-attempt selection, source-revision mismatch, exact age 86,400/86,401-second boundaries, unchanged graph evidence aging, tuple replacement rejection and completed-job replay. Query times are explicit simulations; actual import times are recorded separately. Old observations must not be future-dated. Completed replay is not interrupted-fetch recovery. Optional `--authority-result` compares a supplied source-state report with the fresh source digest; it does not authenticate or independently replay that report.

The semantic runner independently checks complete typed claim tuples for the three cited limits statements. At a fixed common query time the old evidence is expired, the refreshed identical body restores support, and the insufficient document leaves the questions unresolved. The caller’s disposable-host-local-filesystem requirement is satisfied; a caller-authored persistent-host-local-filesystem counterfactual is contradicted. This concerns App Platform host-local storage only, not all provider storage or architecture. The hypothetical recipe remains unimplemented. A separate time-only reassessment expires unchanged fresh evidence. Policy, provider, requirements, recipes and query time remain fixed in the evidence comparisons.

The package runner preserves both exact version-1 profiles, authors a new generation-2 package with matching detached fixtures, and rejects old declaration inputs under the new manifest. Three fixture passes do not renew historical Labs evidence or settle its disagreement with product documentation. See [package fixture provenance](PACK-FIXTURES.md) and [exact copied input inventory](fixtures/PROVENANCE.json).

## Real authority boundary

The separate increment authority harness builds and receipt-installs existing SSIAG/STAV services into a new isolated prefix, explicitly bootstraps a private TOPS trust configuration, then uses qxctl to grant exact source actions. Its providers list is empty. It exercises empty-policy denial, authorized recovery, audit unavailability, restart/recovery, stale plans and committed replay. Same-user diagnostic processes do not establish production supervision or distinct-user isolation. STAV records the SSIAG policy decision; the protected source journal records the actual source write. There is no dedicated source-write STAV receipt.

The maintained [authority runner](verify_authority.cpp) requires explicit paths. `setup` requires a nonexistent output directory and starts only its new diagnostic services; retain the same arguments for later actions. `adoption` deliberately uses human-readable operation IDs, verifies actual STAV events against the source journal decisions, and leaves the services available until `stop`. It grants only this specimen's onboard/relocate actions through the real SSIAG policy interface. A private Unix-socket permission is required. An optional explicit `GOCACHE` can reuse build artifacts; no service tests are run by the two service builds.

```sh
/absolute/maintenance-build/scv-verify-authority setup --repo /absolute/symphony --cli /absolute/qxctl --prefix /absolute/scv-prefix --out /absolute/new-authority-output
/absolute/maintenance-build/scv-verify-authority adoption --repo /absolute/symphony --cli /absolute/qxctl --prefix /absolute/scv-prefix --out /absolute/new-authority-output --source-plan fixtures/relocation-plan-input.json
/absolute/maintenance-build/scv-verify-authority stop --repo /absolute/symphony --cli /absolute/qxctl --prefix /absolute/scv-prefix --out /absolute/new-authority-output
```

`legacy-recovery` is a separate bounded diagnostic for a stopped, genuinely failed prior specimen containing the exact pending v1 operation `inc12-do-onboard`; it is not part of a fresh successful run. It preserves the original journal/runtime bytes, selects a corrected CLI explicitly, verifies unchanged intent and actual audited recovery, and stops its services. The earlier increment's live macOS authority evidence and the later checks of the transferred source remain historical records. The C++ runner preserves all six actions (`setup`, `start`, `adoption`, `audit`, `stop`, and `legacy-recovery`); record its actual native executions separately. A fresh successful specimen does not substitute for an original failed v1 journal.

A candidate capture may precede adoption. Compare its exact source digest to the adopted source digest explicitly before describing it as evidence of that selection. The maintenance manifest is a noncanonical evidence index, not a new permission token, authenticated provenance format or automatic latest-source alias. Caller requirements, provider choice, architecture, acceptable evidence age, permissions and execution stay caller-owned.

Only affected boundaries are tested in this increment. Broad regression, Linux acceptance and the full SCV gate remain separate milestone work.
