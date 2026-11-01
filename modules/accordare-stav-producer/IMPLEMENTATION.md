# Accordare STAV Producer Implementation

The implementation is split into public client orchestration and private configuration, enrollment, peer authentication, evidence protocol, intent, outbox, producer, server, native supervision, package-install, path, and version packages. This prevents qxctl from importing the append implementation or constructing a candidate.

The highest-value regressions cover evidence/result/authority drift, deterministic retry identity with fresh authorization only, intent and outbox restart/collision behavior, unavailable-STAV reconciliation, real installed producer-to-STAV receipt acceptance, authenticated Unix-socket exchange, safe native-supervisor rendering and path handling, receipt-v2 replay and enrollment-safe uninstall, qxctl command parity, exact grant idempotency/removal, and two-sided grant-attempt recovery.

## Development Dependency Observation

During this prototype slice, an ordinary module-cache reuse surfaced the obsolete pre-correction `stav-protocol-go/v0.2.0` archive checksum. SODV already held the correct public checksum, so the tracked consumer sums were repaired from that canonical record and the producer was then resolved and tested successfully with `GOWORK=off` in a new empty module cache. This was stale derived state, not an immutable-tag defect, and no replacement protocol version was required.

The public source dependency set now uses `stav-protocol-go/v0.2.0`, `stav-append-authority/v0.2.1`, `accordare-stav-producer/v0.1.0-dev`, and `sov-ssh-adapter/v0.1.0-dev`. Accordare, SSIAG, and qxctl select append-authority `v0.2.1`; qxctl selects the published Accordare and SSH adapter prereleases. The current consumers do not require the newer producer-vocabulary APIs present in the workspace protocol source.

Standalone qxctl dependency resolution and compilation were verified with Go 1.26.5, `GOWORK=off`, fresh module and build caches, `proxy.golang.org`, and `sum.golang.org`, without module replacements or changes to committed dependency metadata. `go.work` remains the monorepo development composition; it is no longer needed to resolve this qxctl source dependency set. This establishes source-build closure, not binary-package publication or additional runtime capability.
