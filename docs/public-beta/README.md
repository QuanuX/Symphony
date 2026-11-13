# Source-bound Public Beta dossier

**Execution update:** the user approved W01–W12 continuation and selected Anaconda,
with final signing/host/reviewer inputs deferred to the end of W12. See the
[execution record](execution/README.md). The preparation snapshot below remains
historical; approval does not fabricate acceptance results.

Prepared November 13, 2026 for manual review before the requested switch to **Astra Extra High**. This increment prepares W01 and W06 and a full execution plan through W12. It does not execute that later plan, approve scope, build a candidate, sign a package or authorize publication. The feature freeze remains in force.

The reviewed product source is [`887e074780e1d0633bd6e6f57c3d4e07ca3925fc`](https://github.com/QuanuX/Symphony/commit/887e074780e1d0633bd6e6f57c3d4e07ca3925fc), tree `3c989eb589bc2e84eff012aa5428a7e93170d81d`. This is the main revision containing PR #196's three P1 repairs and credential-documentation correction. The future commit containing this dossier has a different identity; no self-referential source hash is implied.

## Review order

1. Read [capabilities and platform limits](CAPABILITY-MATRIX.md), then the [package and trust proposal](PACKAGE-AND-TRUST.md).
2. Review the seven [decisions and ownership gaps](OWNERS-AND-DECISIONS.md). Proposed selections and roles are concrete recommendations, not recorded approvals or assignments.
3. Review the [full W01–W12 implementation plan](IMPLEMENTATION-PLAN.md), including its inputs, dependency order, focused checks and completion criteria.
4. Use the [Astra handoff](ASTRA-HANDOFF.md) after the user's manual approval and model change. A model change alone does not resolve scope conflicts or provide signing/provider authority.

## Prepared artifacts

| File | Purpose and authority |
| --- | --- |
| [dossier.json](dossier.json) | Reviewed source/tree, fingerprinted contracts, freeze, authorization boundary, D01–D07 and preparation status. |
| [source-inventory.json](source-inventory.json) | Existing clean-source helper's inventory, with trailing whitespace normalized and parsed JSON unchanged: 73 component roots, 60 native components, nine Go modules, Swift provider and external/platform identities. It intentionally leaves selection/approval null. |
| [capability-groups.json](capability-groups.json) | Full component-root accounting with proposed selection, operations, platform and qualification limits. Independent versions remain in the source inventory. |
| [package-plan.json](package-plan.json) | Proposed receipt units/delivery closures, content recipes, trust requirements and unresolved actual-byte inputs. |
| [owners.json](owners.json) | Existing named release/rollback primary and explicit unassigned engineering, review, platform and operational roles. No notifications or GitHub assignments. |
| [paper-criteria.json](paper-criteria.json) | Preserved 52 category criteria plus UE1–UE6 and their paper provenance. Historical proposed requirements are data, not automatic implementation authority. |
| [acceptance-mapping.json](acceptance-mapping.json) | All 36 core profile/scenario cells, all 58 paper criteria, 23 Cloudflare operation cells and eight supplemental campaigns, with owners, methods, oracles and pending evidence. |
| [validate.py](validate.py) | Standard-library consistency/source-binding check for this preparation. Success is not release acceptance. |
| [VALIDATION.md](VALIDATION.md) | Focused checks performed for this increment and their limits. |

## Proposed delivery shape

Preserve the four core profiles: macOS 15 ARM64, Ubuntu 24.04 amd64, Ubuntu 24.04 ARM64 and Ubuntu 26.04 amd64. Propose independently versioned native receipt units and public SDK exports, with separate Go/Swift installer contracts and explicit optional/vendor closures. Do not flatten component versions into an invented suite version or split files out of a receipt without addressing its owner contract.

Cloudflare's currently admitted vendor runtime is `darwin-amd64`, separate from the core profile set. Source builds on another target do not admit its vendor artifacts. D02 must select an exact Intel host profile and resolve any retained cross-platform promise. Package preparation can proceed without live-provider profiles; user-led live qualification follows packaging.

General L14 native delivery, OS keychain/Secret Service paths, native credential consumers, the TWS bridge, Cloudflare rotation/project execution and any missing promised risk/state composition remain visible implementation gaps. D03 must resolve a retained promise explicitly before new capability work or its removal. Neither this dossier nor approval to continue the frozen plan is a blanket exception to the freeze.

## Acceptance mapping and evidence

The JSON mapping is a planning companion, **not** input to the existing `acceptance-gates.go` checker. W07/W08 initialize the real checker packet from [`tests/launch-acceptance/plan.json`](../../tests/launch-acceptance/plan.json), then attach exact final-candidate observations. Every selected operation/profile/SDK/channel and paper criterion needs its applicable evidence expansion. Reusing one campaign does not merge independent criteria or waive any of the 36 core cells.

Real fresh-host, live-provider and publisher outcomes remain pending. The existing checker distinguishes invalid evidence (exit 1), incomplete evidence (exit 2), and structurally review-ready evidence (exit 0); even exit 0 does not establish truth or publication authority. Scores remain unassessed. No November 16 completion promise is made.

No credentials, private keys, licensed SDK bytes or private provider payloads belong in this branch. Private observations should be retained under operator control with redacted, exact-identity pointers suitable for review. Public source inventories and package hashes do not replace publisher authority or redistribution review.

## Reproduction and source drift

Run from the repository root with Python 3 and Git:

```sh
python3 docs/public-beta/validate.py
```

The inventory was generated from the clean reviewed source using the existing `prepare-review.cmake` helper. Its command and output digest are recorded in `dossier.json`. Reproducing that inventory requires a clean checkout of the recorded source and a new output file outside that checkout; do not overwrite the committed snapshot or run the clean-source helper while edits are present.

Before any later candidate build, bind the actual candidate commit/tree, regenerate the inventory in a clean checkout, reconcile all capability/package/evidence mappings and run affected source checks. Any production source, dependency, SDK, recipe or artifact change invalidates the corresponding earlier claims and observations. Keep this prepared snapshot and subsequent approvals/evidence distinguishable; never edit null results into passes without the actual observations.
