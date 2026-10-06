# qxctl Command Registry

`COMMANDS.json` is the canonical machine projection of qxctl's expected command registry. It is repository-owned evidence for engine-first administration evaluation and remains usable when no qxctl executable is installed.

The source is not a second hand-maintained command list. Each public or hidden executable Cobra leaf carries one attached `CommandSpec`; one full-tree parity walk derives grammar and JSON support from Cobra, combines them with the stable semantic fields, sorts the records, and computes the registry self-digest. Structural namespaces and internal non-executable plumbing are explicit roles. Hidden prohibited leaves remain registered but cannot satisfy required coverage. A retired ID moves through the dedicated tombstone factory: its old path returns only a deterministic fail-closed diagnostic, its manifest grammar becomes `null`, and its stable ID remains unavailable for reuse.

Each executable command retains its qxctl-owned wrapper binding and may add reviewed backend feature/interaction bindings from the single table in `cmd/qxctl/command_specs.go`. The current registry contains 414 executable commands. Six exact provider-installation/binding routes expose SSIAG-owned inventory, state, non-mutating planning, audited compare-and-swap apply, attempt observation, and recovery without accepting adapter paths or raw versions. The singular SSIAG provider trust routes expose one safe snapshot and one fresh evidence-only verification without duplicating the existing plural provider list or extending doctor semantics. `provider show` binds the Go trust snapshot; `provider verify` binds both the Go trust-verification operation and the nested Swift metadata-handshake operation, while the generic provider list remains configuration metadata rather than fabricated adapter execution. The four SSIAG/STAV supervision and TOPS-enrollment features each bind five explicit qxctl leaves to five distinct module operations, closing all previously uncovered administration surfaces without collapsing feature, command, engine-operation, or caller-operation identity. Nine protected warning-lifecycle leaves and the exact receipt-backed root-summary route expose validation administration without altering raw detector evidence. The four `knowledge invariant status|list|show|check` leaves bind `ssfv:symphony:qxctl.invariant-assurance`; `check` additionally binds `ssfv:symphony:symphony-validator.invariant-ownership-assurance`. Direct query leaves emit `symphony.knowledge.invariant-query-result.v1` with `semantic_validity=not_asserted`, while `check` returns the exact complete validator result and preserves exit 26. The registry records actual grammar and dispatch; it never fabricates coverage for an unavailable route.

Generate and verify from `tools/qxctl`:

The Accordare additions are stable status, reconciliation, supervisor install/uninstall, and two grant-administration identities. Status separates durable intent and append backlog; reconciliation replays only exact candidates; supervisor routes invoke only the verified receipt-v2 executable; grant leaves mutate only the stopped STAV configuration under SSIAG and expected-state recovery.

```text
go run ./cmd/qxctl commands expected --json > COMMANDS.json
go run ./cmd/qxctl commands verify --input COMMANDS.json --json
```

An installed client can emit executable-bound evidence with:

```text
qxctl commands manifest --json
```

Expected evidence has null client version, executable digest, and receipt digest. Observed evidence binds the exact client version and executable digest, plus a receipt digest when one is available. Both forms use `symphony.qxctl.command-registry.v1` and cover every public or hidden executable leaf, including executable parents with subcommands.

The registry records identities and reviewed evidence; it does not inject commands, allocate names, infer feature-worthiness, or grant authority. Any proposal generator must know the registry protocol, the target feature and interaction, stable engine operation and I/O protocols, mutability and authority boundary, recovery requirements, noninteractive/JSON behavior, and required tests. Only reviewed source changes can ratify a new `qxcmd` identity, regardless of caller class.

## Semantic ownership and routing

Canonical paths follow semantic owner, capability, operation. SHV catalogue/evaluation remains under `shv catalogue` and `shv evaluate`; PDF source interpretation remains under `shv pdf`; transport remains under the explicit graph adapter surface. Hardware classes are caller mapping/profile data, not duplicate command trees. `--version` selects the exact primary owner, while `--source-version` and `--adapter-version` name their distinct selected dependencies. Schema and mapping revisions are data identities, not aliases for installed engine versions.

Registry validation rejects sibling name/alias collisions, including structural and hidden routes. Distinct flags cannot disambiguate the same command name. Aliases may be reused under different parents. No command renames or aliases are introduced by the consolidated SHV gate.

A conformance check groups matching backend operations, input/output protocols, mutability and target scope to expose potential competing surfaces. Two exact lifecycle pairs are reviewed: `scv graph-index transfer` versus `transfer-recover`, and `shv materialization run` versus `resume`. They preserve separate lifecycle behavior while reusing an owner operation. Other matching groups require review; a shared native operation alone is not grounds for creating a new semantic owner. This check is a review aid, not a proof that all differently described commands have different meaning.

SHV `--json` errors use the common CLI error boundary once, without printing an additional native envelope or plain-text suffix. Successful native results retain their own protocol.

SHV durable storage is administered only through `shv graph store inspect|prepare|commit|status|query|export|schema|template`. `commit` with the retained expected intent digest is also recovery; no alias or competing recovery command exists. Store scope is declared in the input, connector identity in flags. Structural storage does not duplicate semantic `shv graph validate` or select a catalogue head.

SHV catalogue publication is administered only through `shv catalogue publication inspect|plan|apply|status|schema|template`. Recovery is `apply --operation-id`, mutually exclusive with `--input`; no recovery alias is introduced. This permission-backed local catalogue selection is distinct from source activation and generic graph storage.

SHV storage release 0.2 adds only `shv graph store inventory`: a revision-bound logical read with explicit retained writer identities. Legacy 0.1 selection remains supported; inventory does not confer publication or retention authority.

SCLV historical-warning list/show/acknowledge/reopen are intentionally scoped entry points to the existing governed-validation lifecycle. Their exact subject filter and local persistence are shared with validation administration; they bind no native SCLV backend operation and claim no new semantic authority.

## SQV non-live administration

Ten canonical leaves: acquisition validate/attempts, metadata validate/inspect, flow validate, conversion validate, store inspect, checkpoint inspect, and the shared schema/template pair. Seven leaves extend the original three; no aliases or provider-specific command trees are added.

Every native action selects an explicit `--prefix`, exact `--version`, and bounded `--input FILE`. Shared `sqv schema` / `sqv template` selects the native `--operation`: request_validate, metadata_validate, metadata_inspect, flow_validate, conversion_validate, store_inspect, checkpoint_inspect or attempts_inspect. Only request_validate takes `--adapter fred|databento_historical|databento_reference`; other operations reject an adapter selection. Templates contain unanswered fields and are not validated plans.

Metadata inspection selects evidence projections in its request. Flow validation covers binding, allocation, frame, credits and queue together through an isolated native trial. Conversion validation proves format/count/limit selection only. Store inspection verifies the published-head prefix and reports unverified tails; checkpoint and attempt projections refuse stores that require recovery. All uint64 quantities use decimal strings and opaque bytes use hex/chunks in the new contracts.

The eight existing module/inventory routes now include twenty SQV/SCABV modules (twenty-four total modules, 136 contract files). These are source-contract observations. Exact installed native OWNER-INTERFACE, schemas and templates remain receipt-bound resources. Live status, detached jobs, provider activation and mutation/recovery retain their separate prerequisites.

## SNV administration

Fifteen canonical leaves under `snv`: `identity validate`, `resources validate`, `clusters validate`, `names validate|resolve`, `inspect`, `evidence prepare|commit|status`, `state plan|apply|status|recover`, `schema` and `template`. Typed proposal modes share the matching validation leaves. Shared discovery accepts exact owner and operation; it introduces no provider-specific tree.

Each invocation selects an exact prefix/version. State/evidence commands additionally select an explicit private root; views require exact TOPS and view IDs. Native plans remain pure; evidence preparation/commit retains private immutable lineage; apply/recover reauthorize and use compare-and-swap. Observation creates or repairs no journal state. Timeout during mutation is an uncertain outcome resolved by the exact operation ID. See `modules/snv-engine/SKILL.md` for the public capsule workflow and `SPEC.md` for capacity, installation and audit boundaries.

The native state reducer is shared by apply/recover, while the evidence-plan reducer is reused for prepare/commit. These paths represent separately observable durable boundaries and preserve one operation lineage; no duplicate alias is admitted. Schema/template carry no domain interpretation or permission. The existing generic module/inventory routes now disclose 29 direct modules and 166 Contract Quad files.

## SBV native experiments and portable results

Eight new leaves administer capabilities, run, compose, result inspect/query/export and shared schema/template resources. Run and compose declare target-host permission-backed local artifact creation and recover by result inspection; no canonical apply or live execution occurs. Inspection can discover a destination digest after an uncertain write, while query/export pin an expected snapshot. Export uses native inspection but emits the complete portable result or pointer-stream protocol; this is a separate data projection from metadata inspection. All implemented native operation/interaction pairs have command bindings.

SBV 0.2 adds `sbv catalogue`, `sbv evaluate --input request.json`, and `sbv compose-joint --input scenario.json`. Catalogue entries state their supported operation names. Evaluation imports a source-bound external census and preserves producer/causality declarations; model measure domains distinguish probabilities, scenario weights and signed coefficients. Explicit joint paths are never expanded into unlisted combinations. Use `sbv schema --operation evaluate` for installed census/model/outcome/price-excursion definitions, and the existing result query/export commands for every field. The original matching 0.1 qxctl/package may remain separately installed; the new exact wrapper does not silently upgrade it.

`qxctl sbv economics --prefix <exact-prefix> --version 0.3.0-dev --input <request.json>` writes a new portable result from an exact model artifact. Use `sbv template --operation economics` and `sbv schema --operation economics` to discover the contract. SBV 0.3 had twelve command leaves; catalogue, result inspect/query/export, output formats and target-host permission/recovery behavior apply to economics as to other result-producing operations.

SBV 0.4 adds `qxctl sbv book --prefix <exact-prefix> --version 0.4.0-dev --input <request.json>`. Installed schema/template and catalogue disclose source-bound census selection, reconstruction profile, replay/cadence/depth controls and checkpoint behavior. All frame and checkpoint fields use `sbv result inspect|query|export`. At 0.4 there were thirteen SBV leaves and 412 total commands.

SBV 0.5 adds `qxctl sbv liquidity --prefix <exact-prefix> --version 0.5.0-dev --input <request.json>`. Schema/template expose order intents, frame selection, participation, optional activation probability, IOC/FOK and shared-snapshot policies. Catalogue and full result query/export include every conditional fill, probability and study field. Current totals: fourteen SBV leaves and 413 commands.

SBV 0.6 adds `qxctl sbv allocation-economics --prefix <exact-prefix> --version 0.6.0-dev --input <request.json>`. Installed schema/template/catalogue expose filled-quantity markout, caller marks, units, costs and optional activation mixtures. All conditional economics, exact moments, source allocations and replay are available through result inspect/query/export. Current totals: fifteen SBV leaves and 414 commands.

SBV 0.7 adds `sbv result select`, `sbv backend-plan` and `sbv live-plan`, each with an exact JSON `--input` request and installed schema/template. Selection supports typed exact predicates/order and stable pagination. Planning reports no resource reservation or live activation. Current totals: eighteen SBV leaves and 417 commands.
