# STAV Append Authority

This independently buildable Go module holds the owner-ratified implementation namespace for Symphony's per-TOPS STAV append authority.

The current increment is intentionally limited to executable lifecycle, pure namespace resolution, and shared protocol-kernel identity validation. Canonical STAV semantic/read schemas now live under `knowledge/stav/`, but this executable does not run a service, open `append.sock`, accept producer traffic, emit events/receipts, or create or mutate a ledger.

Start with `INTENT.md`, `MANIFEST.md`, `INSTALL.md`, and `SKILL.md`. `IMPLEMENTATION.md` records the gated sequence toward an operational authority.
