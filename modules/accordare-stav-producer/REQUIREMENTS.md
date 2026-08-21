# Accordare STAV Producer Requirements

- Go 1.26.5-compatible source with the explicit migration path to Go 1.27 after release confirmation.
- `CGO_ENABLED=0`; Darwin and Linux kernel APIs only.
- Independent receipt-v2 install/uninstall and per-TOPS enrollment; uninstall fails closed while any visible enrollment retains the package.
- Exact UID/GID endpoint authentication and exact SSIAG subject binding.
- Closed four-operation vocabulary with no caller-selected audit fields.
- Safe metadata only; no secret, proof, capability, payload, body, alias, or path persistence.
- Intent durability precedes coordinator mutation; candidate durability precedes append; committed receipts are verified; both pending states are explicit and recoverable.
- Exact retry preserves mutation semantics while allowing only a freshly validated SSIAG authorization proof.
- Native launchd/systemd supervision is per TOPS, liveness-only, independently installable, and receipt-bound through qxctl.
- qxctl administers installation grants using SSIAG authorization, stopped-authority mutation, CAS, and durable recovery.
- All behavior remains outside hot and warm paths.
