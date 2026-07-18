# STAV Append Authority Architecture

```text
knowledge/stav/                    canonical protocol truth
        |
libraries/stav-protocol-go/       strict types, JCS, digests, frames
        |
        +--> SSIAG producer ------+ authenticated append
        |                         v
        +--> qxctl reader --> local append authority --> locked STAV ledger
                                      |
                                      +--> read projection / verification
```

One process serves one TOPS serialization domain. Startup loads the strict configuration, verifies the process service identity, opens and exclusively locks the ledger, scans every frame, verifies the canonical event and linear digest chain, reconstructs receipts, and becomes ready only after those checks pass.

Production liveness is a per-TOPS launchd job, systemd unit, or explicit owner-provided equivalent. It has no SSIAG dependency and gains no producer, reader, or ledger authority. After identity verification, the process separately locks the persistent adjacent socket lifecycle file before stale inspection and binding. SIGTERM closes admission, drains bounded workers, removes the socket, and releases the socket lock; ledger durability remains governed by the independent ledger lock and fsync contract.

The accept loop captures kernel peer credentials before dispatch. Producer credentials resolve to one configured producer identity and exact permission tuples. Reader credentials resolve to one safe subject and classification allowlist. The public client authenticates the server peer before transmitting.

Append assignment and durable storage occur under one mutex. A partial write or sync failure poisons the running ledger instance; retry is resolved after restart from the persisted event/idempotency index. Queries operate over verified in-memory entries and omit ungranted classifications.

Canonical direction is always knowledge → protocol kernel → implementation. Runtime types and code never become schema truth.
