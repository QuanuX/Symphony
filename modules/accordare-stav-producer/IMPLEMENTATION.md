# Accordare STAV Producer Implementation

The implementation is split into public client orchestration and private configuration, enrollment, peer authentication, evidence protocol, intent, outbox, producer, server, native supervision, package-install, path, and version packages. This prevents qxctl from importing the append implementation or constructing a candidate.

The highest-value regressions cover evidence/result/authority drift, deterministic retry identity with fresh authorization only, intent and outbox restart/collision behavior, unavailable-STAV reconciliation, real installed producer-to-STAV receipt acceptance, authenticated Unix-socket exchange, safe native-supervisor rendering and path handling, receipt-v2 replay and enrollment-safe uninstall, qxctl command parity, exact grant idempotency/removal, and two-sided grant-attempt recovery.

## Development Dependency Observation

During this prototype slice, an ordinary module-cache reuse surfaced the obsolete pre-correction `stav-protocol-go/v0.2.0` archive checksum. SODV already held the correct public checksum, so the tracked consumer sums were repaired from that canonical record and the producer was then resolved and tested successfully with `GOWORK=off` in a new empty module cache. This was stale derived state, not an immutable-tag defect, and no replacement protocol version was required.

The active monorepo intentionally composes newer protocol source through `go.work` while module versions are still rolling. Consequently, standalone qxctl resolution awaits publication of the Accordare producer module, and current SSIAG workspace behavior may depend on protocol extensions newer than the last published protocol package. The eventual packaging campaign must publish and verify one coherent module set from empty caches; prototype development does not pretend that rolling source revisions already form that final release set.
