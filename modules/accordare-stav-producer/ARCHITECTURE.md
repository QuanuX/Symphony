# Accordare STAV Producer Architecture

```text
qxctl Named Version command
  -> SSIAG exact allow/capability
  -> Knowledge Session Coordinator mutation/result
  -> authenticated Accordare producer Unix socket
  -> strict command/result verification
  -> closed safe-candidate derivation
  -> fsynced private outbox
  -> authenticated STAV append socket
  -> committed receipt OR durable pending state
```

The two sockets solve different authority problems. The first establishes which enrolled qxctl process supplied evidence and whether its SSIAG decision binds that exact operation. The second establishes that the isolated producer service holds the installation's four-tuple STAV grant. Neither peer credential alone is permission.

The outbox is a recovery queue, not a second ledger. It stores only the already-redacted candidate and candidate digest; STAV remains the sole sequence, time, event-ID, producer-ID, and chain authority.
