# Accordare STAV Producer

The Accordare STAV Producer is Symphony's independently installable, cgo-free Go service for converting terminal, SSIAG-authorized SAV Named Version mutations into the four closed audit tuples registered by `knowledge/stav/registries/v1/accordare.json`.

For configured audit production, qxctl durably prepares an authenticated intent before coordinator mutation and completes that exact intent afterward. Status separates unresolved intents from append-pending candidates; exact command retry repairs interruption without guessing order or outcome. The service supports closed succeeded/failed/unavailable metadata, real STAV receipt acceptance, and independent per-TOPS launchd/systemd supervision administered through exact receipted qxctl routes.

It is not a general event writer. qxctl sends the exact command, exact coordinator result, and immutable coordinator installation identity over a bounded Unix socket. The producer authenticates qxctl with kernel peer credentials, verifies the embedded SSIAG decision and result digest, derives safe STAV metadata itself, persists that candidate to a private outbox, and then asks the per-TOPS STAV append authority to append it. A STAV outage returns `pending`; reconciliation replays the same request identity idempotently.

```sh
go build ./cmd/symphony-accordare-stav-producer
go test ./...
```

The executable provides `install`, `uninstall`, `enroll`, `unenroll`, `serve`, `status`, and `reconcile`. Enrollment never creates a STAV grant. qxctl administers the installation-specific grant while STAV is stopped:

```sh
qxctl stav accordare-grant install \
  --tops-id UUID --scope user --operation-id stable-id \
  --expected-config-digest sha256:...
```

See `INSTALL.md` for the full ordering and `THREAT-MODEL.md` for the trust boundary.
