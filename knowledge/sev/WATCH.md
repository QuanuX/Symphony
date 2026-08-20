# SEV Session Trigger and Watch Policy

## Default

Watch behavior is opt-in and disabled by default. The engine remains independently usable through bounded direct process IPC. No watcher enters a hot or warm path, performs ambient mutation, or treats filesystem order as authority.

## Session Boundary

The default session begins after successful authentication and ends at logout or when re-authentication is required. qxctl administrators may select another explicit contract-defined boundary. Policy is host-permission based and caller-neutral.

## Trigger Rules

One policy declares bounded source scopes, event classes, a debounce interval, and a coalescing ceiling. Events within the window produce one deterministic novelty candidate bound to the first and last STSC instants and the ordered event digests. The trigger proposes a case; it never opens, applies, or closes one without the ordinary SEV and SSIAG procedures.

Interrupted collection is recoverable. An ambiguous or incompatible trigger journal fails closed and preserves evidence. Hot/warm paths, native Windows engines, uncontrolled daemons, and automatic export are prohibited.
