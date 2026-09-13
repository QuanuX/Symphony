# qxctl Install

## Requirements
- Go 1.26.5 requirement
- No Python requirement
- No remote runtime requirement
- No provider/cloud/Docker/Kubernetes requirement

qxctl uses the Architect-ratified Cobra command framework and a constrained Viper mapper, plus their pinned cgo-free Go dependencies. Its STAV grammar uses Symphony's first-party pure-Go protocol kernel. The module remains independently buildable with `GOWORK=off`; it does not require Python, C bindings, a remote configuration backend, or a configuration daemon.

The direct CLI dependency pins are Cobra `v1.10.2` and Viper `v1.21.0`. `go.mod` and `go.sum` are the authoritative dependency lock surfaces.

Viper is not an SSIAG/STAV trust loader. qxctl creates private instances, binds only declared command keys and environment variables, and does not enable automatic environment discovery, configuration-file discovery, remote providers, watch/reload, or write-back.

## Build and Run
qxctl can be run locally using the Go toolchain:
```bash
go run ./cmd/qxctl --help
go run ./cmd/qxctl status
```
Or built locally:
```bash
go build -o qxctl ./cmd/qxctl
```

The SKVI, SCLV, SACV, SODV, and SSFV command groups require their separately installed engine packages. `qxctl validate` and `qxctl knowledge invariant check` require the separately installed Symphony Validator. Pass the applicable exact prefix and version; qxctl validates the complete versioned receipt and owned files before execution. `knowledge invariant status|list|show` need no installed engine or validator because they only read and structurally validate the canonical repository registry; their `semantic_validity=not_asserted` result is not a substitute for `check`. Secure local installation access is implemented on Linux and macOS and fails closed on other native operating systems. Validation profiles and baselines remain outside an installation prefix and survive validator upgrade or uninstall; version-incompatible baselines must be recreated explicitly.

## Migration Note
qxctl targets Go 1.27 only after general availability and the differential fixture/digest, default-vs-`nojsonv2`, vet, race, and supported-platform cross-build gates pass. The workspace and module pins change atomically, and the migration cannot alter qxctl grammar or STAV bytes.

SCV `.6` adds `scv provider pack prepare|evaluate`, `scv composition explore|reassess` and `scv interface show`. The new leaves select exact `.6` by default; existing leaf defaults and `.1`–`.5` installations retain their admitted contracts. Provider packs, detached fixtures, requirements, recipes and counterfactual changes are caller-authored inputs. qxctl validates installed native results and can retain all four new owner artifact kinds through `scv artifact import|show|list`; it does not choose provider policy or execute a recipe. Read `knowledge/scv/PROVIDER-PACKS.md`, `COMPOSITION.md` and `OWNER-INTERFACE.md` for the exact semantics and finite bounds.

## Maintained SCV composition workflow (`0.7.0-dev`)

`qxctl scv composition workflow run|status|recover` coordinates retained provider-package evaluation, finite composition and optional reassessment. The exact request and result contract is `knowledge/scv/COMPOSITION-WORKFLOWS.md`, discoverable in the selected `.7` installation. Run/recovery selects an exact `.7` composition owner; each package stage retains and invokes the original pack owner, including preserved `.6` installations. Status requires only the explicit workflow root, TOPS and operation ID and reports sealed-checkpoint inspection without native replay.

The new journal pins exact caller requirements, policies, query time, record references, root/TOPS and installation. Completed stages retain immutable owner-validated records; retries and recovery reconstruct their exact inputs before replay. This adds no native C++ operation, implicit source refresh, provider choice, graph selection or obligation execution. Completed workflow progress is separate from failed fixtures, unresolved findings and unimplemented recipes. Earlier command defaults and workflow protocols remain unchanged.

## Precise Obligation Follow-up

Exact `0.8.0-dev` exposes 28 native operations, including replayed obligation inventory and subsequent-evidence comparison under `knowledge/scv/OBLIGATIONS.md`. qxctl exposes direct operations and immutable original-owner relationship retention/show. Native check state remains separate from supplied reference provenance, causal attribution and runtime verification. Ten owner companions and 26 schemas expose 97 catalog protocols. Earlier exact packages, protocols and command defaults remain preserved.

## Exact Evidence Bundles

Exact `0.9.0-dev` exposes 30 native operations. `knowledge/scv/BUNDLES.md` owns complete bounded evidence-reference transport, transport-only inspection and evaluation through the unchanged composition owner. qxctl exposes pack, inspect, evaluate and expanded-result routes, plus existing exact-owner artifact retention/show. Repeated evidence is stored once inside a complete bundle; reconstruction counts all logical occurrences before allocation. Caller requirements, provider selections and semantics remain unchanged. Eleven owner companions and 27 schemas expose 103 catalog protocols. Earlier exact packages and command defaults remain preserved.

## Explicit Bundle Workflows

Exact `0.10.0-dev` retains thirty native operations and adds qxctl bundle workflow run/status/recover and immutable bundled obligation retain/show. `knowledge/scv/BUNDLE-WORKFLOWS.md` owns the versioned coordination and logical-reference contracts. Separate v2 journals pin explicit transport, caller input and exact installation before work; original records preserve both native and transport identities. Status reports sealed checkpoint validation, while run/recovery and relationship inspection replay original owners. Twenty-nine schemas expose112 catalog protocols with twelve owner companions. Prior routes, defaults, records and packages remain preserved.
