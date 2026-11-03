# MVP launch ownership and support

This is an implementation tracking surface, not release authorization. The [30-task ownership register](launch-tasks.json) retains completion criteria and dependencies. It is separate from verified completion evidence; null maintainer/milestone fields are unresolved assignments. Proposed roles are not assigned people. The user has assigned @quantDIY launch approval and rollback ownership. Module maintainers, a backup, a release milestone and private security intake remain pending explicit confirmation; no GitHub issue, milestone, assignment or notification is created by this file.

| Gate | Accountable person | Required decision/evidence |
| --- | --- | --- |
| Capability/profile freeze (L01) | Pending confirmation | Exact shipped modules/versions and promised provider operations |
| Release approval and rollback (L25) | @quantDIY (primary); backup pending | Named backup; signed source/artifact inventory; documented rollback decision |
| Private security intake (L21/L25) | Pending confirmation | Authorized, tested private destination and response ownership; see SECURITY.md |
| Final candidate acceptance (L26) | Pending confirmation | Each required profile and scenario with exact artifact/host evidence |
| Provider acceptance (L14–L17) | Pending confirmation | Authorized profiles, SDK provenance, bounded actual provider results |
| Redistribution (L23) | Pending confirmation | Recursive runtime/TLS closure, source material and publisher review |

A release approver checks the capability manifest, source/receipt/version consistency, final candidate trust and all required scenario records before approving publication. A rollback owner stops further rollout, preserves uncertain operation evidence, selects a verified compatible prior version, and records the decision and observed recovery. Approval to install is not approval to deliver credentials or place orders. Do not retry uncertain provider/lifecycle mutations automatically.

Use the bug form for non-sensitive reports with module version, source, OS/architecture, receipt digest, bounded reproduction and redacted logs. Acknowledge only through a selected monitored channel; no service-level response promise is established. Known limitations include unavailable live provider qualification, unresolved final runtime/license closure, incomplete C++26 static-analysis coverage, unpublished release downloads, and unqualified physical DGX/GPU behavior. Proposal-only modules must remain labelled as proposals.

Track L24 source guides and L26 acceptance preparation in their PR, with remaining owners and milestone recorded here once authorized. Accepted tasks may be converted into GitHub issues in a later explicit tracking action; do not interpret role labels as assignments or a completed launch gate.
