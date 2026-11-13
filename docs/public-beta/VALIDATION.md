# Preparation validation — November 13, 2026

These checks concern the W01/W06 dossier and its reviewed source binding. They do not qualify a product build, final download, signing identity, provider or release. The full W01–W12 implementation and qualification plan remains for the user's manual approval and model switch.

## Checks performed

1. Verified clean local main and remote `refs/heads/main` at `887e074780e1d0633bd6e6f57c3d4e07ca3925fc`, tree `3c989eb589bc2e84eff012aa5428a7e93170d81d`, before branching. No earlier source change was present.
2. Ran the existing clean-source `prepare-review.cmake` helper to a new file outside the checkout. Its dependency source verification passed **73 roots, 60 native components and 1,364 source hashes**, including Go/Swift/interface/schema identities. Its release manifest check passed complete component/native/Go root accounting. The helper emits trailing whitespace; the committed `source-inventory.json` removes that whitespace with verified parsed-JSON equality. Original helper output remains unchanged outside the checkout. Both raw and committed digests are in `dossier.json`.
3. Ran `python3 docs/public-beta/validate.py`. Result: `valid_preparation_only`, with **26 capability groups, 36 core cells, 58 paper criteria and 23 Cloudflare operations**. Scope approval, release acceptance and publication remain false.
4. Ran the validator against eleven deliberately modified temporary copies. All eleven returned exit 1 with the expected refusal. See [validation-results.json](validation-results.json) for the observed diagnostics. The unchanged positive copy returned exit 0.
5. Independently cross-reviewed capability and package selection: all 73 roots are accounted for; each proposed core delivery set contains exactly **51 complete native receipt units and six Go commands**. Four dependency-only native library inclusions do not grant their optional live-provider claims. Optional units remain explicit, with decisions pending.
6. Reviewed plan sequencing and helper boundaries: W06 pre-build decisions are separate from W06 actual-byte closure during W07; native receipt packaging, Go/Swift owner verification and multi-artifact acceptance require distinct recorded handling. Existing acceptance-v1 is not portrayed as a universal delivery-set checker.
7. Checked 135 relative Markdown file links, Git whitespace and the prepared files for user-machine paths/private-key blocks. These passed. This is a bounded preparation check, not a substitute for later payload or secret review.

## Deliberate corruption cases

- Missing mandatory core profile/scenario cell.
- Duplicate paper criterion.
- Fabricated publication approval.
- Stale source fingerprint.
- Missing capability root.
- Unadmitted Cloudflare runtime platform.
- Fabricated candidate archive digest.
- Changed native package version.
- Omitted native dependency in a core delivery set.
- Duplicate JSON key.
- Altered source inventory with a recomputed wrapper hash; the projection still must match the bound Git source.

These were copies under a new temporary directory. No production source, live state, candidate payload or credential was changed. The validator uses local Git and Python's standard library; it does not execute JSON-supplied commands, run builds or access the network.

## Limits and later checks

The preparation snapshot intentionally contains no final candidate identities or acceptance observations. Its validator checks source binding and consistency; it cannot establish the truth of future observations, legal rights, authority, fresh-host state or live outcomes. Manual decisions must be recorded separately, with affected source/artifact identities; do not change preparation flags to fabricate completion.

No broad product, native, race, sanitizer, live-provider or fresh-host campaign was rerun for this documentation/preparation increment. Run focused affected checks during later implementation and the required candidate gates at W07–W12. The branch is to be pushed as instructed **without waiting for hosted CI results**; a push is not a recorded CI pass.
