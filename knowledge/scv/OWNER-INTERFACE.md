# SCV Owner Interface Declaration v1

## Scope

The `0.6.0-dev` development interface introduces `OWNER-INTERFACE.json` as the maintained declaration of mechanical SCV interface metadata. It declares exact releases, supplied engine domains, operations, protocol identities, interactions, mutation classifications, retained artifact admissions, adapter surfaces, and ordered companion contracts. It does not define provider facts, semantic interpretation, connection policy, source authority, or executable admission for arbitrary third-party programs.

The supplied engine domains are a release inventory. Caller-owned provider identities remain admissible through existing SCV/family contracts. Adding a supplied native domain to a later release is an explicit declaration with `introduced_in`; it does not retroactively change an earlier release's installed surface.

## Generated Projections

`modules/scv-engine/tools/generate_interface.py` generates three checked-in projections:

- `modules/scv-engine/src/interface.generated.inc`: native dispatch metadata and companion contract identities. The implementations and handler signatures remain ordinary C++ source.
- `tools/qxctl/internal/knowledgeengine/scv_interface_generated.go`: exact operation/protocol, domain/release, adapter surface, artifact kind/admission, interaction, and companion inventories. These tables do not implement semantic consumer checks.
- `cmake/ScvInterface.generated.cmake`: the current exact release, supplied domains, schema inventory, companion file inventory, and declaration source path. Schema files are derived from the catalog and its local reference closure. Unreferenced files, escaping references, absent native protocol entries, and a mismatched catalog release fail generation.

Normal engine and qxctl builds consume these checked-in files. There is no generator process, Python runtime, network lookup, or mutable interface discovery in a running engine. Python and gofmt are authoring/check tools. CMake integration consumes the generated inventory rather than retyping companion and schema lists.

Run `python3 modules/scv-engine/tools/generate_interface.py` after an authorized interface change. Run the same command with `--check` to reject drift without writing. `--metadata-only` is an explicitly incomplete authoring bootstrap while new schemas are still being assembled; it does not produce the installation inventory and cannot stand in for the complete regeneration check.

The frozen `modules/scv-engine/tests/fixtures/interface-history.v1.json` records the `.1` through `.5` mechanical projections at source base `4078f3a`. The generator never rewrites this fixture. A change to an older operation, protocol, admission, domain inventory, or companion contract fails comparison. Release admission uses declared ordinal membership, never a `latest` alias or a lexical/semantic version comparison. Existing CLI defaults remain separately explicit and unchanged.

## Installed Inspection

Each supplied `.6` package owns its declaration at `share/symphony/contracts/<module>/<exact-version>/OWNER-INTERFACE.json`. `qxctl scv interface show` reads the explicitly selected package's receipt-owned declaration without needing a source checkout or invoking provider operations.

The reader verifies the exact receipt, regular-file ownership entry, protected path, byte length and file digest. It then checks the declaration identity and canonical digest against the interface admission compiled into qxctl, and rechecks the installation identity before returning. A caller cannot broaden operation support by replacing the installed declaration, even if a replacement carries a self-consistent new digest. Source JSON schemas describe structure; the compiled admission check supplies the independent exact-definition binding.

The `symphony.qxctl.scv-owner-interface.v1` result returns the complete declaration and installed owner. `interface_digest` identifies exact retained file bytes; `definition_digest` identifies canonical declaration content. `receipt_validation=exact_owned_file_verified` and `scope=interface_metadata_only` state what was established. Neither digest grants mutation authority or certifies provider knowledge. Earlier releases without the interface surface fail explicitly rather than borrowing a newer declaration.

Canonical definition digests for supported resource releases derive from release-truncated declarations. Future operations, artifact admissions, or supplied domains therefore do not change the definition expected from an already supported earlier release.

## Independent Validation

Generated structure is not a shared semantic oracle. Native source/pack/interpretation/composition implementations remain authoritative for their published contracts. Go independently checks result identity, exact retained inputs, evidence anchors, comparison results, inventory correspondence, and reported accounting. Hand-authored adversarial tests continue to reject correctly resealed but false results, copied provenance, altered requirements, and invalid recovery checkpoints.

Generator regressions separately cover deterministic regeneration, frozen historical projections, missing/duplicate fields and identities, undeclared releases, invalid handler tokens, broader mutation classes, artifact admission before owner availability, schema escape/missing protocol failures, and modified generated files. Installed tests check receipt-bound interface inspection and preserve the existing exact package/version tests.

## Maintained Composition Coordination

The exact `0.7.0-dev` release retains 26 native operations and adds the installed `knowledge/scv/COMPOSITION-WORKFLOWS.md` companion and its workflow schema. qxctl coordinates original-owner package evaluations, finite exploration and optional reassessment with pinned intent, immutable artifact records and interruption recovery. Nine owner companions and 24 schemas expose 89 protocol entries. The existing native meanings remain unchanged; a completed run preserves source gaps, failed fixtures and implementation obligations. Earlier exact `.1`–`.6` installations and command defaults remain available.

The additional frozen `modules/scv-engine/tests/fixtures/owner-interface-0.6.v1.json` preserves the exact `.6` declaration from closure `186ec79`; regressions compare both its complete release-truncated definition and mechanical projection. The earlier `.1`–`.5` history fixture remains byte-preserved.
