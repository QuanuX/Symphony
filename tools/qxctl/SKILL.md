# qxctl Skill

## How Callers Should Use qxctl
Any caller operating within its effective target-host permission should use `qxctl` as the primary local administrative spine to verify repository status, module integrity, and runtime inventory. Caller type does not expand or reduce authority.

## Command Examples
- `go run ./cmd/qxctl status`
- `go run ./cmd/qxctl status --json`
- `go run ./cmd/qxctl commands expected --json`
- `go run ./cmd/qxctl commands verify --input COMMANDS.json --json`
- `go run ./cmd/qxctl commands manifest --json`
- `go run ./cmd/qxctl inventory digest`
- `go run ./cmd/qxctl modules check`
- `go run ./cmd/qxctl ssiag doctor --tops-id UUID`
- `go run ./cmd/qxctl ssiag status --tops-id UUID --json`
- `go run ./cmd/qxctl ssiag providers --tops-id UUID --json`
- `go run ./cmd/qxctl stav status --tops-id UUID`
- `go run ./cmd/qxctl stav verify --tops-id UUID`
- `go run ./cmd/qxctl stav query --tops-id UUID --limit 100`
- `go run ./cmd/qxctl knowledge engines list --json`
- `go run ./cmd/qxctl knowledge engines bind skvi --prefix /chosen/prefix --expected-registry-digest absent`
- `go run ./cmd/qxctl knowledge engines doctor`
- `go run ./cmd/qxctl knowledge reconcile compatibility --json`
- `go run ./cmd/qxctl knowledge reconcile begin --operation-id ID --expected-journal-digest absent --path INTENT.md`
- `go run ./cmd/qxctl knowledge reconcile status --json`
- `go run ./cmd/qxctl knowledge reconcile checkpoint --operation-id ID --expected-journal-digest sha256:...`
- `go run ./cmd/qxctl knowledge reconcile close --operation-id ID --expected-journal-digest sha256:...`
- `go run ./cmd/qxctl knowledge reconcile recover --operation-id ID --discover`
- `go run ./cmd/qxctl knowledge session transition --tops-id UUID --event login --event-id HOST-EVENT-ID --json`
- `go run ./cmd/qxctl knowledge session features begin --tops-id UUID --operation-id FEATURE-SESSION-ID --expected-journal-digest absent --json`
- `go run ./cmd/qxctl knowledge session features checkpoint --tops-id UUID --operation-id FEATURE-CHECKPOINT-ID --expected-journal-digest sha256:... --json`
- `go run ./cmd/qxctl maestro inventory --prefix /chosen/maestro-prefix --tops-id UUID --json`
- `go run ./cmd/qxctl knowledge lifecycle profile set --tops-id UUID --input profile.json --expected-profile-digest absent --json`
- `go run ./cmd/qxctl knowledge lifecycle ownership status --tops-id UUID --root /configured/root --profile-id default --json`
- `go run ./cmd/qxctl knowledge lifecycle ownership reconcile --tops-id UUID --root /configured/root --profile-id default --json`
- `go run ./cmd/qxctl knowledge lifecycle observe --tops-id UUID --profile-id default --json`
- `go run ./cmd/qxctl knowledge lifecycle report --tops-id UUID --profile-id default --json`
- `go run ./cmd/qxctl knowledge lifecycle boot --tops-id UUID --profile-id default --operation-id BOOT-ID --expected-journal-digest absent --json`
- `go run ./cmd/qxctl knowledge lifecycle apply --tops-id UUID --profile-id default --operation-id APPLY-ID --source-journal-digest sha256:... --expected-apply-journal-digest absent --expected-applied-state-digest absent --source-root /trusted/stage --json`
- `go run ./cmd/qxctl knowledge lifecycle apply-status --tops-id UUID --profile-id default --json`
- `go run ./cmd/qxctl skvi check --prefix /chosen/prefix`
- `go run ./cmd/qxctl skvi project --prefix /chosen/prefix --json`
- `go run ./cmd/qxctl skvi propose --prefix /chosen/prefix --input proposal-input.json`
- `go run ./cmd/qxctl sclv check --prefix /chosen/prefix`
- `go run ./cmd/qxctl sclv propose --prefix /chosen/prefix --input proposal-input.json`
- `go run ./cmd/qxctl sclv recover --prefix /chosen/prefix --input recovery-input.json`
- `go run ./cmd/qxctl sclv project --prefix /chosen/prefix --json`
- `go run ./cmd/qxctl sclv evidence local-git --prefix /chosen/prefix --input local-git-evidence.json --json`
- `go run ./cmd/qxctl sclv evidence airgap --prefix /chosen/prefix --input airgap-evidence.json --json`
- `go run ./cmd/qxctl sacv check --prefix /chosen/prefix`
- `go run ./cmd/qxctl sacv diff --prefix /chosen/prefix --input diff-input.json`
- `go run ./cmd/qxctl sacv propose --prefix /chosen/prefix --input proposal-input.json`
- `go run ./cmd/qxctl sacv project --prefix /chosen/prefix --json`
- `go run ./cmd/qxctl sodv check --prefix /chosen/prefix`
- `go run ./cmd/qxctl sodv verify --prefix /chosen/prefix --input observed-state.json`
- `go run ./cmd/qxctl sodv propose --prefix /chosen/prefix --input proposal-input.json`
- `go run ./cmd/qxctl sodv recover --prefix /chosen/prefix --input recovery-input.json`
- `go run ./cmd/qxctl sodv project --prefix /chosen/prefix --json`
- `go run ./cmd/qxctl ssfv check --prefix /chosen/prefix`
- `go run ./cmd/qxctl ssfv check --prefix /chosen/prefix --baseline semantic-snapshot.json --freshness report`
- `go run ./cmd/qxctl ssfv diff --prefix /chosen/prefix --input diff-input.json`
- `go run ./cmd/qxctl ssfv propose --prefix /chosen/prefix --input proposal-input.json`
- `go run ./cmd/qxctl ssfv graph --prefix /chosen/prefix --json`
- `go run ./cmd/qxctl ssfv administration-check --prefix /chosen/prefix --input administration-input.json --json`

## Constraints
- Use the Go standard library, ratified first-party Go libraries, and only their approved cgo-free platform dependencies.
- Treat Cobra as the command grammar and Viper only as a private, explicitly bound command-configuration mapper. Do not enable `AutomaticEnv`, configuration-file discovery, remote providers, watch/reload, or write-back.
- Treat stable command IDs and attached `CommandSpec` data as ratified protocol evidence. Preserve the command's qxctl wrapper binding and add reviewed backend feature/interaction bindings only through `reviewedBackendFeatureBindings` in `cmd/qxctl/command_specs.go`; never fabricate a binding for a route that does not exist. Generate `COMMANDS.json` only with `commands expected`, verify it before review, and use `commands manifest` only for exact executable-bound observation. Never hand-edit the projection, infer identities or engine-operation IDs, or let any proposal bypass feature and command ratification.
- Keep SSIAG/STAV trust configuration and endpoint authentication outside Viper in their dedicated clients.
- Run commands synchronously in the active execution session.
- SSIAG commands may read safe metadata and request exact audited authorization decisions. Treat every capability as non-transferable evidence for its bound operation, never a bearer token or canonical apply authority. Never pass secret values through qxctl arguments, input, output, logs, or fixtures.
- Canonical knowledge, STAV, and security-governed surfaces remain read-only. Current mutations are limited to protected noncanonical engine bindings, reconciliation/session/lifecycle journals, lifecycle runtime selection/activation state, exact local receipt-v2 packages, and verified applied-state evidence. Use only operations permitted by the target host and satisfy its configured safeguards; never emulate, manufacture, or bypass host authority.
- STAV commands require an enrolled, running authority and an explicit reader grant. Never bypass endpoint authentication, reader classification, or add raw append behavior.
- For `ssiag|stav enrollment|supervisor`, always provide the exact installation prefix. Provide `--version` when multiple compatible versions coexist; never infer recency. Run `status`, create an exact unexpired `plan`, apply it with the expected attempt state, and use only digest-linked `apply-status` or `recover`. Ordinary audit fails closed until the closed receipt route exists; explicit `audit_deferred` remains reconciliation-required. Never expose purge, force, no-stop, arbitrary manager arguments, or `serve` through this qxctl circuit.
- Use `knowledge engines bind` only with an exact inactive-undocked installation. Supply `absent` for the first mutation and the exact digest returned by `list` for every later mutation. A binding is noncanonical selection for later reconciliation; it is not installation, invocation, authentication, permission, repository activation, or Maestro docking.
- Use `maestro inspect` for public receptor metadata and `maestro status|recover` for freshly SSIAG-authorized presence administration. Supply `--maestro-prefix` and `--maestro-receptor-id` to lifecycle observe/report/apply when docking is desired. Dock and undock are intentionally available only through the prepared, journaled, re-observed lifecycle apply circuit.
- Use `knowledge reconcile begin|checkpoint|close` with a stable unique operation ID and the exact state reported by `status`; safe retries reuse the same operation ID. Use `recover --discover` only when the head is unavailable or inconsistent and retain all reported repair evidence. Never delete or edit journal slots to force a result. Unknown critical/newer state and ambiguous slots are stop conditions, not reasons to select an older version.
- For implemented SKVI commands, invoke only an explicit installation prefix and exact version. Treat proposal and projection output as noncanonical; `qxctl skvi propose` does not apply its result.
- Keep SKVI proposal input to the exact nonsecret operation schema. Never place credentials, proofs, raw tokens, provider payloads, environment data, or executable instructions in its semantic fields.
- For implemented SCLV commands, invoke only an explicit installation prefix and exact version. Treat checks and normalized provider results as evidence and proposals, recovery results, and projections as noncanonical; recovery never updates or deletes the journal.
- Supply SCLV v3 proposal/recovery and provider-normalization input only through bounded no-follow JSON files. Use `sclv evidence local-git|airgap` only with the exact SCLV receipt-v2 package that owns the typed adapter entry point. Provider evidence must never contain credentials, raw proofs, unnormalized provider payloads, shell fragments, or environment dumps. A successful normalization, including an asserted air-gap ratification field, does not let qxctl decide truth, permission, ratification, or canonical apply.
- For implemented SACV commands, use an exact inactive-undocked installation and treat check/diff output as evidence, proposals as unratified, and projections as disposable. The development engine validates JSON OpenAPI 3.2.0 entry documents and reports YAML parser availability fail-closed; never use qxctl to invent ownership, endpoints, security profiles, publication approval, SDK eligibility, or runtime bindings.
- For implemented SODV commands, use an exact inactive-undocked installation and provide external tag/package observations only through a bounded no-follow JSON file. Treat verification and recovery as noncanonical evidence, proposals as unratified, and projections as disposable. qxctl and the engine never create or move tags, contact providers, declare completion, append release records, mutate recovery journals, or publish artifacts.
- For implemented SSFV commands, use an exact inactive-undocked installation. Treat snapshots, diffs, and graphs as noncanonical evidence and proposals as unratified. Supply baseline/proposal input only through bounded no-follow JSON files. Never use qxctl to decide feature-worthiness, ratify semantics, apply a proposal, create a `FEATURES.md`, or persist a graph.
- Use `ssfv administration-check` with its complete repository-independent input when evaluating design, live, authorization, and module-integration coverage. qxctl absence is an evaluated live state, not a design failure; missing registered command or engine-operation bindings remain uncovered evidence. Treat remediation as constraints requiring review, never generated command identity or authority.
- For repository validation, use `validate scan` with one exact installed validator. Use `profile` to adjust optional warning disposition/presentation and `baseline` to classify warning deltas. `debug` filters display only after a complete scan. Never treat a baseline as ratification or use policy to downgrade a violation or alter detector scope.
- Treat the default knowledge session as a login/authentication-to-logout/expiry/revocation authority epoch containing separate worktree reconciliation contexts. Never extend authority across a required re-authentication boundary.
- Use `knowledge session transition` only from an explicit reviewed host integration. Reuse one stable event ID for retry, and use `--recover` only when damaged local head/journal evidence should be reconciled. Do not convert denial, incompatibility, or ambiguity into recovery and do not imply that qxctl installs a login or boot integration.
- Use `knowledge session features` only inside an open authenticated session. Preserve the first SSFV snapshot as the immutable baseline, reuse stable operation IDs for retry, and supply the exact current maintenance digest. Optional Maestro observation must be complete and authenticated; omit `--maestro-prefix` to record explicit `not_configured` evidence. Treat `review_required` as a review signal, never ratification or write authority.
- Use `maestro inventory` for a complete derived TOPS-wide view. Do not treat an inventory error as an empty deployment and do not use its timestamped observation digest where stable inventory identity is required.
- Keep `symphony.knowledge.engine-binding-registry.v1` fixed to its six roles. Treat profile input, desired, observed, runtime, planned, report-journal, apply-journal, and applied state as separate evidence governed by `knowledge/LIFECYCLE.md`; discovering a new or missing package is never implicit permission to execute, remove, switch, bind, or dock it. Use `boot` only to establish the exact report-only source. Use `apply` only for an `apply-compatible` profile with exact source/report/apply/applied digests and explicit staged receipt-v2 roots. Safe retry reuses the base operation ID and current compare-and-swap state. Use `apply-status` without mutation and `apply-recover --discover` only for one unique digest-linked v2 chain. Never hand-edit a prepared action, runtime state, journal slot, head, or applied file.
- Use `ssiag grants lifecycle` only to generate proposal input for one configured subject, authority basis, TOPS, and profile. Its stable profile resource and separate read-only profile-catalog resource prevent policy churn, but every operation remains exact and every lifecycle digest remains independently mandatory. The result has `apply_enabled=false`; do not present generation as policy mutation or authorization.
- Established-role selection is a binding-registry compare-and-swap, not generic runtime selection. For a coordinator change, install versions side by side and let lifecycle apply perform the candidate journal-read preflight. Do not remove the old version during the handoff or directly rewrite the registry.
- Treat receipt-v2 installation and removal as local content-addressed file operations, not package download or script execution. Installation publishes the receipt last. Uninstall requires a separate trusted rollback source, validates every remaining owned path, and requires an enforced root-local ownership registry with no retained or legacy claim before deletion; it removes the receipt last. Use `ownership adopt` only after reviewing conservative legacy claims and draining operations prepared from pre-fence observations; use `ownership release` for one intentional legacy receipt at a time. Do not remove the receipt-layout ownership fence: it makes older lifecycle clients preserve the root and stop on their next observation. A conflicting administrator file, competing claim, damaged fence, or adoption gap is an integrity/dependency stop.
- Treat generic `active` runtime state as administrative eligibility, not proof that a process, service, engine binding, or receipt entry point ran. Dock/undock and coordinator self-replacement remain stop conditions.
- Keep vector administration, recovery, and audit reconciliation away from hot and warm paths.

## Do-Not-Use-For List
- Do not use qxctl for managing NATS directly.
- Do not use qxctl for deploying to cloud/Docker/Kubernetes.
- Do not use qxctl to replace `node-troll`, `bus-troll`, or `hotpath-runtime`.
- Do not use qxctl to write generated SKVI/SCLV/SACV/SODV/SSFV records directly; use ratified proposal operations and the separately gated apply path when available.
- Do not use qxctl to enforce runtime behavior or execute hotpath workloads.
- Do not use qxctl to implement provider SDK behavior or bypass SSIAG policy.
- Do not use qxctl to append STAV events or edit ledger files.

## Preferred Verification Sequence
1. `go run ./cmd/qxctl doctor`
2. `go run ./cmd/qxctl contracts`
3. `go run ./cmd/qxctl modules check`
4. `go run ./cmd/qxctl inventory digest`
5. `go run ./cmd/qxctl status`
6. `go run ./cmd/qxctl ssiag doctor --tops-id UUID` when the selected SSIAG enrollment is running
7. `go run ./cmd/qxctl stav doctor --tops-id UUID` when the selected STAV enrollment is running
8. `go run ./cmd/qxctl knowledge engines doctor`
9. `go run ./cmd/qxctl knowledge reconcile compatibility --json`
10. `go run ./cmd/qxctl knowledge reconcile status --json`
11. `go run ./cmd/qxctl knowledge session status --tops-id UUID --json` when the exact SSIAG session grants and coordinator binding are available
12. `go run ./cmd/qxctl knowledge session features status --tops-id UUID --json` when the SSFV maintenance grant and coordinator binding are available
13. `go run ./cmd/qxctl knowledge lifecycle apply-status --tops-id UUID --json` when exact lifecycle grants and coordinator binding are available
