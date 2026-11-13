# Manual decisions and ownership

This dossier makes the next work reviewable. It does not assign people, waive promises, choose credentials or authorize release. `owners.json` and `dossier.json` carry the machine-readable roster and decisions. The existing source assigns **@quantDIY** release approval and primary rollback; other named assignments remain open.

## Decisions for this handoff

| ID | Concrete proposal | Input or decision still required | Needed before |
| --- | --- | --- | --- |
| D01 | Select the proposed independently versioned core/SDK closure and explicitly selected optional payloads from the capability/package files. Keep every root accounted for, including build support and unavailable proposals. | Approve exact components, operations, package recipe choices and exclusions with preserved-promise dispositions. An inventory is not an instruction to ship every root. | Selected W07 candidate closure. |
| D02 | Retain all four core profiles; propose a separate Cloudflare Intel macOS runtime closure. | Select its exact OS/profile and reconcile any promise of vendor execution on core targets. Do not replace a core profile with Intel or infer Linux/ARM vendor admission from source builds. | Target-specific build and claims. |
| D03 | Keep the freeze; qualify and repair implemented behavior. Retain missing general credentials/OS/native consumers, TWS, Cloudflare rotation/project execution and any missing risk/state composition as blocked promises. | For each required missing capability, explicitly approve deferral/scope change or a bounded freeze exception with owner, contract and new acceptance cells. If neither is chosen, that promise stays blocked while independent packaging work proceeds. | Dependent capability work or acceptance; no blanket new-feature authorization. |
| D04 | Use independently distributed publisher verification policy and private signing custody; review actual staged runtime/TLS/notices/source/vendor closure. | Select publisher/signing identities, authorized signing process, recovery/revocation policy, any OS signing/notarization requirement and payload reviewer. Actual staged hashes and candidate URLs remain pending. | Signing, distribution and final trust acceptance. |
| D05 | Use the role roster below; retain @quantDIY as approval/rollback primary. | Name responsible people and independent reviewers, per-component escalation and per-profile operators, plus a backup and security-response coverage. No agent/model name is a personnel assignment. | Respective execution and W12 sign-off. |
| D06 | Propose verified direct candidate downloads as the initial packaging route. | Select any additional archive/mirror, MCPB/registry, OpenAI directory or other channel explicitly. The separately maintained MCP repository needs exact native/catalogue/package/client alignment; external directory approval remains separate. | Promised channel construction/qualification/publication. |
| D07 | Preserve all 58 paper criteria and all 36 existing core acceptance cells. Apply later freeze and user-led live-after-packaging direction. | Review criterion applicability and any proposed 8.5 threshold or E3/E4 substitution policy explicitly. Missing mandatory evidence cannot become a pass through an average score or calendar date. | W12 complete-acceptance decision. |

Approval to continue W01–W12 preparation can leave external inputs unresolved. It cannot make an unimplemented route operational, approve a provider/account effect, waive a retained mandatory promise, or turn a testing candidate into an accepted release. Record each approved decision with the exact selection, reviewer, date and affected source/artifact identities in a subsequent decision artifact; preserve this preparation snapshot.

## Ownership roster

| Role | Current person | Concrete responsibility |
| --- | --- | --- |
| Release approver; rollback primary | @quantDIY | Exact scope and publisher decision; compatible rollback selection and uncertainty preservation. |
| Release coordinator | Unassigned | Dependency order, capability/criterion applicability, blocker and source/candidate tracking. |
| Component maintainers | Unassigned per capability group | Native/Go/Swift semantics, independent versions, exact SDK/interfaces, defects and escalation. |
| Packaging engineer | Unassigned | Per-profile build/staging, receipts, dependency closure, recipes and hashes. |
| Payload reviewer | Unassigned | Actual recursive loaders/TLS, optional/vendor payload and notices/source/build-material review. |
| Signing custodian | Unassigned | Independent trust policy, authorized signing, key custody and recovery. Private key material is never an input to Git. |
| Platform operators | Unassigned per selected profile | Fresh-host inventory, exact downloads, installed execution and lifecycle evidence. |
| SDK/research owner | Unassigned | External-engine source, two existing public routes, complete scientific oracle and documentation-only agent campaign. |
| Authority/security owner | Unassigned | Native identity, permission/audit, credential and revocation/refusal boundaries. |
| Provider testers; provider review owner | User-led after packaging; named participants/reviewer pending | Exact authorized environment/SDK/account/operation observations and review; no secret or paid payload publication. |
| State/recovery owner | Unassigned | Supported composition/state owners, fault/reconciliation and expected-state oracle. |
| Performance owner | Unassigned | Predeclared budgets/workload/clock policy and complete measurements. |
| Independent acceptance reviewer | Unassigned | Actual final-byte evidence, failures/skips, source drift and retained-promise closure. |
| Backup rollback operator | Unassigned | Backup-operated restore, incident, escalation and compatible rollback drill. |
| Security-response owner | Unassigned | Private ticket access/routing/acknowledgement and response/escalation coverage. |
| Channel owner | Unassigned | Selected client/catalogue/package/distribution prerequisites and final-channel tests. |

The preparer/implementer may coordinate evidence but does not certify their own independent review. Where multiple roles are held by one person, document the actual review arrangement rather than inventing independence. No GitHub issues, milestones, assignments or messages are created by this roster.

Unassigned operational roles do not prevent local preparation, authorized code fixes or synthetic qualification. Obtain the actual responsible person when a step needs publisher custody, independent human acceptance, provider authority or an operational handoff; continue unrelated work in parallel.

## Inputs to acquire without blocking unrelated work

- **Before candidate construction:** approved scope/profile/receipt closure; actual toolchains and SDK/runtime artifacts with provenance; private build and staging locations; owner-specific installer contracts; target hosts. Package metadata preparation can continue while these are collected.
- **Before signing or offering a candidate:** actual payload and notices review, independent trust policy, publisher authorization, candidate identity and real retrieval location. Local unsigned/disposable-key fixtures are not substitutes.
- **After packaging, before provider acceptance:** users' explicitly authorized provider environments and operation bounds, exact SDK/session/account identities, live refusal/recovery evidence. Never request secrets in public issues or embed them in evidence.
- **Before W12 complete acceptance:** named reviewer/backup/security coverage, all retained mandatory applicable observations, supported restore/rollback and intake drill, resolved critical defects and explicit publisher decision. An incomplete decision is valid evidence of remaining work; it is not release completion.
