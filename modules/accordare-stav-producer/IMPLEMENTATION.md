# Accordare STAV Producer Implementation

The implementation is split into public client orchestration and private configuration, enrollment, peer authentication, evidence protocol, outbox, producer, server, package-install, path, and version packages. This prevents qxctl from importing the append implementation or constructing a candidate.

The highest-value regressions cover evidence/result/authority drift, deterministic retry identity, outbox restart and collision behavior, unavailable-STAV reconciliation, real authenticated Unix-socket status/reconcile exchange, receipt-v2 replay and enrollment-safe uninstall, qxctl command parity, exact grant idempotency/removal, and two-sided grant-attempt recovery.
