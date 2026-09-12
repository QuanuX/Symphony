# Symphony Cloud Hyperscalers Vector Engine Manifest

## Canonical Surfaces

- `modules/schv-engine/FEATURES.md`
- `modules/schv-engine/INSTALL.md`
- `modules/schv-engine/INTENT.md`
- `modules/schv-engine/MANIFEST.md`
- `modules/schv-engine/SKILL.md`
- `modules/schv-engine/SPEC.md`

## Identity

- module: `schv-engine`
- engine: `symphony-schv`
- vector: `schv`
- version: `0.9.0-dev`
- language: C++26
- thermal path: freezing
- semantic owner: `knowledge/scv/schv/SPEC.md`

## Package Boundary

The module owns its exact versioned executable, receipt, installed contract documents and licenses. It shares implementation source with `modules/scv-engine/`; source reuse does not merge package identity, select an active version, or require its parent/provider siblings at runtime. The immutable receipt declares the actual owned files and entry point. No active alias, service, listener, login hook or Maestro presence is implied.

## Domain State

Source configurations, captured artifacts, accepted interpretations and selected graph revisions are distinct private installation data. The process receives bounded explicit inputs and emits bounded evidence or transitions. It cannot rewrite canonical repository knowledge, grant permissions or commit remote provider changes.

The `0.3.0-dev` descriptor has twenty operations, including the four corpus operations owned by `knowledge/scv/CORPUS.md` and three profile/connection operations owned by `knowledge/scv/INTERPRETATION.md`. Immutable corpus bytes and job bookkeeping are qxctl adapter state, not this process's package files or a selected corpus head. Existing exact `0.1.0-dev` and `0.2.0-dev` installations remain independently invocable; interpretation wrappers retain authored mappings and their exact evidence without altering prior data shapes.

The additive `0.4.0-dev` package exposes 21 operations, including native profile preparation, and owns its exact schema catalog, schemas and templates. `knowledge/scv/AGENT-WORKFLOWS.md` defines the new engine/CLI boundary. Retained workflow evidence is outside immutable package files; `.3` and earlier installations remain separately invocable.

## Maintained Provider Coverage

The additive `0.5.0-dev` package has 22 operations. `knowledge/scv/COVERAGE.md` owns native accounting of declared sources, exact corpus selection and independently replayed interpretations. Missing, unlisted, unselected, partial and stale evidence remains explicit. The operation does not rank providers or establish runtime compatibility. Earlier exact `.4` and prior installations remain preserved.

## Portable Provider Authoring and Composition Exploration

The additive `0.6.0-dev` package exposes 26 operations. `knowledge/scv/PROVIDER-PACKS.md` owns portable caller-authored provider packs, native sealing, detached conformance fixtures and bounded structured extraction. A pack links an explicit provider declaration, authored profiles, source references and selected fixture expectations; a successful fixture comparison describes those cases alone. Qualified knowledge remains reusable through the existing evidence interfaces. This is an authoring surface for independently chosen providers, without requiring a new compiled provider enum in SCV; a supplied leaf still enforces its advertised family/provider identity.

`knowledge/scv/COMPOSITION.md` owns finite exploration of caller-selected recipes against caller-selected requirements. Native operations preserve evidence, prerequisites, interface declarations, guarantee changes and authored resolution pointers; missing evidence remains unresolved. The engine does not enumerate an open-ended design space, rank providers, choose a user's requirements, provision infrastructure or turn a declaration into observed compatibility. Reassessment preserves exact before/after inputs and changed axes.

`knowledge/scv/OWNER-INTERFACE.json` is the versioned owner declaration for operation metadata, release admission, artifact kinds and installation inventories. Its checked-in generated projections drive native dispatch metadata and Go interface admission; domain validation and adversarial consumer tests remain independent. The package includes eight owner companions, the schema catalog and a receipt-owned copy of the declaration, inspectable through `qxctl scv interface show`. Normal builds and engine invocation do not require the Python authoring generator. Earlier exact `.1`–`.5` receipts and CLI defaults remain unchanged; new pack, composition and interface commands default to exact `.6`.

## Maintained Composition Coordination

The exact `0.7.0-dev` release retains 26 native operations and adds the installed `knowledge/scv/COMPOSITION-WORKFLOWS.md` companion and its workflow schema. qxctl coordinates original-owner package evaluations, finite exploration and optional reassessment with pinned intent, immutable artifact records and interruption recovery. Nine owner companions and 24 schemas expose 89 protocol entries. The existing native meanings remain unchanged; a completed run preserves source gaps, failed fixtures and implementation obligations. Earlier exact `.1`–`.6` installations and command defaults remain available.

## Precise Obligation Follow-up

Exact `0.8.0-dev` exposes 28 native operations, including replayed obligation inventory and subsequent-evidence comparison under `knowledge/scv/OBLIGATIONS.md`. qxctl exposes direct operations and immutable original-owner relationship retention/show. Native check state remains separate from supplied reference provenance, causal attribution and runtime verification. Ten owner companions and 26 schemas expose 97 catalog protocols. Earlier exact packages, protocols and command defaults remain preserved.

## Exact Evidence Bundles

Exact `0.9.0-dev` exposes 30 native operations. `knowledge/scv/BUNDLES.md` owns complete bounded evidence-reference transport, transport-only inspection and evaluation through the unchanged composition owner. qxctl exposes pack, inspect, evaluate and expanded-result routes, plus existing exact-owner artifact retention/show. Repeated evidence is stored once inside a complete bundle; reconstruction counts all logical occurrences before allocation. Caller requirements, provider selections and semantics remain unchanged. Eleven owner companions and 27 schemas expose 103 catalog protocols. Earlier exact packages and command defaults remain preserved.
