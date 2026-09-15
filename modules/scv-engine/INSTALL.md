# Symphony Cloud Vector Engine Installation

Requires CMake 3.25+, a C++26 compiler and the Linux-first or supported macOS POSIX development path. Each module can build from the monorepo and links the authority-free `Symphony::KnowledgeVectorEngine` mechanics.

```bash
cmake -S modules/scv-engine -B build/scv-engine -DBUILD_TESTING=ON
cmake --build build/scv-engine
ctest --test-dir build/scv-engine --output-on-failure
cmake --install build/scv-engine --prefix /chosen/prefix
```

The exact executable is `libexec/symphony/scv-engine/0.10.0-dev/symphony-scv` beneath the selected prefix. Install receipts and contract documents use the common exact `share` layout. Installation is inactive `installed_undocked`; it neither selects a binding nor installs another domain. Use an explicit prefix, exact version and selected domain through the admitted qxctl command surface. Direct `--descriptor` returns the supported operations.

Only receipt-owned files may be removed by the module's receipt-verified uninstaller. Source stores, captured evidence, selected graphs, other versions and containing directories are not package files. An immutable version cannot be overwritten in place. No remote acquisition, provider credential, network activation or publication is part of installation.

The additive `0.3.0-dev` package can coexist with retained exact `0.1.0-dev` and `0.2.0-dev` installations. Do not rebuild changed source under the old version identity, overwrite an existing receipt, or install a latest alias. Corpus operations retain exact `.2` admission; provider interpretation and connection operations retain exact `.3` admission. Later explicitly selected supported releases add their declared interfaces. Existing command defaults remain unchanged until the caller selects another admitted exact version.

The current `0.10.0-dev` installation includes29 receipt-owned SCV schemas, templates, twelve owner companions and `OWNER-INTERFACE.json`. Profile preparation and schema discovery retain their original `.4` defaults; choose `.10` explicitly for current definitions. Existing operations may explicitly select `.4`; their earlier defaults and installed versions are preserved. Do not overwrite an existing exact-version receipt with rebuilt bytes.

## Maintained Provider Coverage

The additive `0.5.0-dev` package has 22 operations. `knowledge/scv/COVERAGE.md` owns native accounting of declared sources, exact corpus selection and independently replayed interpretations. Missing, unlisted, unselected, partial and stale evidence remains explicit. The operation does not rank providers or establish runtime compatibility. Earlier exact `.4` and prior installations remain preserved.

## Portable Provider Authoring and Composition Exploration

The additive `0.6.0-dev` package exposes 26 operations. `knowledge/scv/PROVIDER-PACKS.md` owns portable caller-authored provider packs, native sealing, detached conformance fixtures and bounded structured extraction. A pack links an explicit provider declaration, authored profiles, source references and selected fixture expectations; a successful fixture comparison describes those cases alone. Qualified knowledge remains reusable through the existing evidence interfaces. This is an authoring surface for independently chosen providers, without requiring a new compiled provider enum in SCV; a supplied leaf still enforces its advertised family/provider identity.

`knowledge/scv/COMPOSITION.md` owns finite exploration of caller-selected recipes against caller-selected requirements. Native operations preserve evidence, prerequisites, interface declarations, guarantee changes and authored resolution pointers; missing evidence remains unresolved. The engine does not enumerate an open-ended design space, rank providers, choose a user's requirements, provision infrastructure or turn a declaration into observed compatibility. Reassessment preserves exact before/after inputs and changed axes.

`knowledge/scv/OWNER-INTERFACE.json` is the versioned owner declaration for operation metadata, release admission, artifact kinds and installation inventories. Its checked-in generated projections drive native dispatch metadata and Go interface admission; domain validation and adversarial consumer tests remain independent. The package includes eight owner companions, the schema catalog and a receipt-owned copy of the declaration, inspectable through `qxctl scv interface show`. Normal builds and engine invocation do not require the C++ authoring generator. Earlier exact `.1`–`.5` receipts and CLI defaults remain unchanged; new pack, composition and interface commands default to exact `.6`.

## Maintained Composition Coordination

The exact `0.7.0-dev` release retains 26 native operations and adds the installed `knowledge/scv/COMPOSITION-WORKFLOWS.md` companion and its workflow schema. qxctl coordinates original-owner package evaluations, finite exploration and optional reassessment with pinned intent, immutable artifact records and interruption recovery. Nine owner companions and 24 schemas expose 89 protocol entries. The existing native meanings remain unchanged; a completed run preserves source gaps, failed fixtures and implementation obligations. Earlier exact `.1`–`.6` installations and command defaults remain available.

## Precise Obligation Follow-up

Exact `0.8.0-dev` exposes 28 native operations, including replayed obligation inventory and subsequent-evidence comparison under `knowledge/scv/OBLIGATIONS.md`. qxctl exposes direct operations and immutable original-owner relationship retention/show. Native check state remains separate from supplied reference provenance, causal attribution and runtime verification. Ten owner companions and 26 schemas expose 97 catalog protocols. Earlier exact packages, protocols and command defaults remain preserved.

## Exact Evidence Bundles

Exact `0.9.0-dev` exposes 30 native operations. `knowledge/scv/BUNDLES.md` owns complete bounded evidence-reference transport, transport-only inspection and evaluation through the unchanged composition owner. qxctl exposes pack, inspect, evaluate and expanded-result routes, plus existing exact-owner artifact retention/show. Repeated evidence is stored once inside a complete bundle; reconstruction counts all logical occurrences before allocation. Caller requirements, provider selections and semantics remain unchanged. Eleven owner companions and 27 schemas expose 103 catalog protocols. Earlier exact packages and command defaults remain preserved.

## Explicit Bundle Workflows

Exact `0.10.0-dev` retains thirty native operations and adds qxctl bundle workflow run/status/recover and immutable bundled obligation retain/show. `knowledge/scv/BUNDLE-WORKFLOWS.md` owns the versioned coordination and logical-reference contracts. Separate v2 journals pin explicit transport, caller input and exact installation before work; original records preserve both native and transport identities. Status reports sealed checkpoint validation, while run/recovery and relationship inspection replay original owners. Twenty-nine schemas expose112 catalog protocols with twelve owner companions. Prior routes, defaults, records and packages remain preserved.
