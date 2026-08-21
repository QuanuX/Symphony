# Accordare STAV Producer Architecture

```text
qxctl Named Version command
  -> SSIAG exact allow/capability
  -> authenticated Accordare prepare
  -> fsynced private intent
  -> Knowledge Session Coordinator mutation/result
  -> authenticated Accordare completion
  -> strict intent/command/result verification
  -> closed safe-candidate derivation
  -> fsynced private outbox
  -> authenticated STAV append socket
  -> committed receipt OR durable pending state
```

The two sockets solve different authority problems. The first establishes which enrolled qxctl process supplied evidence and whether its SSIAG decision binds that exact operation. The second establishes that the isolated producer service holds the installation's four-tuple STAV grant. Neither peer credential alone is permission. An interrupted coordinator call leaves the intent unresolved; exact command retry reuses coordinator idempotency and never guesses whether mutation occurred.

The intent store and outbox are recovery queues, not secondary ledgers. The intent store retains bounded authenticated command evidence; the outbox retains only the already-redacted candidate and candidate digest. STAV remains the sole sequence, time, event-ID, producer-ID, and chain authority.
