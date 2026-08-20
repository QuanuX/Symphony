# SEV Engine Specification

The C++26 process implements every v1 operation in `knowledge/sev/SPEC.md` over exact bounded JSON. Descriptor v2 binds stable operation IDs and SSFV feature IDs. Case, action, impact, plan, verification, SCSEV, graph, recovery, closure, and compatibility outputs are content addressed.

Disposition planning validates all action self-digests, rejects absent or duplicate identities, rejects dependency cycles, and computes a deterministic ready set. A blocker removes its action and transitive dependents while leaving independent zero-indegree actions ready. Transition verification evaluates only the SAV closed rule algebra against a complete caller-supplied successor CURRENT snapshot.

SCSEV requires the fourteen canonical consequence families in contract order. It reports missing or prohibited consequences and never invents a final command identity, grammar, exemption, or patch.

Successor recalculation derives readiness from the complete supplied plan after every completed or failed action and propagates blockers only through dependents. Watch and novelty checks enforce freezing-path, offline/private, disclosure, redaction, and prohibited-secret boundaries. Session binding ties an exact case and source CURRENT to the established lifecycle profile/report-journal stream without persistence or apply authority.

All mutation, persistence, host discovery, authorization, audit submission, coordinator state changes, network export, and hot/warm execution are external and unimplemented by this process.
