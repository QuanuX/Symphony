# SBV follow-on work after the selected M3 release

Recorded 13 October 2026 at Duncan's request. This is a resumable backlog, not a
claim that the capabilities below have shipped or a restriction on user systems.
The selected M3 gate is complete; its exact evidence and limitations are in
[M3 qualification](M3-QUALIFICATION.md) and
[milestone readiness](M2-M3-READINESS.json).

## Where we stopped

Native SBV 0.33.0-dev, optional Go transport 0.1.7-dev and SOV SSH carrier
0.1.0-dev are qualified within the selected release. This includes caller-authored
account/feedback/settlement models, durable jobs, complete terminal results,
resident datasets and multithreaded CPU execution. macOS-to-Linux guest SSH passes
with real same-UID, user-scope SSIAG/STAV, all nine selected remote operations,
complete typed account/replay access and loss/reconciliation without duplicate
execution. The guest has a separate kernel on the same physical Mac. Native worker,
controller, connection and completed-response loss are distinct tested cases.

The macOS/Linux native gates, focused carrier checks, independent final review
and cross-OS output comparisons remain attributed to their recorded revisions.
Private evidence is retained in the project workspace under
`implementation/sbv-m3-remote-final/`; its `CLOSEOUT.json` binds the 3,850-file
evidence seal. Licensed samples, binaries, credentials and machine-specific
qualification records are not part of this source publication.

## Follow-on lanes and completion evidence

| Lane | Work left and prerequisite | Evidence required before claiming support |
| --- | --- | --- |
| Physical hosts and WAN | Select reachable independent hosts, transport, network route, OS/toolchains, data placement and authenticated identities. The existing Codex cloud environment had no usable inbound/private route; creating a runner alone does not supply one. | Repeat actual controller-to-target admission, all selected operations, complete typed results, cancellation, connection/response loss and replacement observation on those hosts. Bind exact binaries, receipts and endpoint identities. |
| Authority isolation and access | Select distinct service/operator principals and, where desired, system-scope SSIAG/STAV. Current positive evidence is same-UID user scope. Certificates, jump hosts and other SSH mechanisms need a compatible selected carrier/profile; they are not silently enabled. | Prove permitted and denied operations, scope/audience/resource isolation, expiry/revocation, audit outage and key/runtime/configuration changes without unauthorized dispatch or result access. |
| Cluster resources and storage | Add or qualify SOV-owned placement/reservation, NUMA/affinity, large-RAM and shared-storage arrangements on actual target hardware. Terabyte capacity has not been measured. | Correctness and complete result access under selected concurrency, placement, memory/spill and filesystem policies; fault/reboot/power-loss evidence appropriate to each durability claim. Existing process-crash tests do not certify physical durability. |
| Checkpoint and recovery | Define native checkpoint eligibility and exact continuation state for each selected engine/provider. Durable observation is implemented; automatic mid-calculation checkpoint/resume is not. | Interrupted/resumed output and provenance agree with uninterrupted work, including provider memory, RNG/tapes, account state, source position and publication identity. Never silently retry an uncertain admitted side effect. |
| GPU/tensor and data adapters | Select actual CUDA/tensor runtime integrations, vendor/language adapters and any borrowed Arrow/DLPack buffers. Capability seams exist; device execution and zero-copy integration remain unqualified. | Exact version/dtype/shape/ownership contracts, lifetime/release/fence tests, cancellation and errors, CPU-reference output comparison and qxctl discovery/configuration/results. Measure transfers and loading as well as kernel time. |
| Data acquisition and calibration | Finish the operational SQAV/SSIAG historical-use/secret bridge and select its deployment. Obtain suitable initialized-book, multi-source/session/correction and execution-label evidence for each intended model. Retained capped samples do not establish full market state or calibrated fills. | Explicit costs and unknown-outcome accounting, source identity/entitlement/lineage, reset/gap handling and retained exports. Any calibrated likelihood needs appropriate held-out labels/domain checks. Preserve the existing aggregate historical-data budget; the recorded spend remains $0 of the authorized $10. |
| Models, instruments and studies | Select further models individually: endogenous impact/full exchange matching, margin/liquidation/nonlinear derivatives, transactional multi-event multileg behavior, stitched combinatorial paths, additional studies and optimizer/provider algorithms. Existing external extension paths remain available. | Explicit assumptions and unsupported cases, independent calculations/reference fixtures, conservation/ordering checks, fixed-census versus feedback lineage, uncertainty/calibration evidence where applicable, and complete qxctl/SDK result parity. |
| Visualization and live capture | Build a GUI against existing portable typed results and replay event/time/source identities. Select any additional deployment access routes. Live shadow/capture remains a disconnected conceptual scaffold under the original instruction. | GUI and terminal reconstruct the same result with completeness and unavailable states preserved. Connected live capture/reconnect requires a separate explicit instruction and its own coverage/loss/backpressure tests; it is not implicitly enabled by a valid plan. |
| Peer comparisons and release claims | Select installed peer versions, editions, licenses, hardware, workload and equivalent scientific fidelity before benchmarking. The dated primary capability review and current CPU/RAM timings are not an installed competitor benchmark. | Correct outputs first; then reproducible end-to-end loading, calculation, memory, terminal/export and warm/cold-cache measurements. State unsupported/unselected domains and uncertainty. No fastest-engine or exhaustive-parity claim from feature lists. |

## Suggested next starting point

For broader deployment confidence, start with a genuinely separate physical host
and a named authority/network profile. Reuse the already qualified carrier and
fixtures where unchanged; do not rebuild an entire qualification campaign merely
because the agent or chat changed. If the next goal is user-facing research or
visualization instead, select that lane explicitly; the deployment backlog does
not prevent further scientific or frontend work.

Before implementation, name the chosen capability, actual owner, prerequisite,
original work package/cases and observable completion gate. Keep the original
96-case register and its inherited partial/deferred dispositions visible rather
than treating this note as an exhaustive specification. The private workspace
`implementation/sbv-m3-remote-final/CASE_EVIDENCE.json` and
`PROGRAMME_STATUS.json` provide the case/package crosswalk.

All new choices, mutable settings, lifecycle operations and results must remain
qxctl-first and portable to independent consumers. Users choose criteria, models,
weights, studies, data retention and resource constraints, including no dataset/RAM
ceiling and no job deadline. A test fixture's cap or timeout is not product policy.
Preserve observed data versus modeled outcomes, unavailable versus zero, and
caller authority over experimentation. Run focused affected tests during each
increment and the applicable broader gate at its defined release milestone.
