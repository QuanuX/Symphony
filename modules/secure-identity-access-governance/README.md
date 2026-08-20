# Symphony Secure Identity and Access Governance

SSIAG is Symphony's Go-only, independently installable identity, authentication, authorization, capability, credential-reference, lease, and provider-operation foundation.

The current foundation installs immutable receipt-v2 packages, enrolls isolated TOPS instances, authenticates Darwin/Linux Unix-socket peers from kernel credentials, supervises per-TOPS services, evaluates exact deny-by-default grants, administers a protected policy overlay, verifies the metadata-only macOS adapter, reconstructs exact receipt-owned app bundles in private staging, reports three separate signed-bundle/policy/session readiness layers, and manages exact provider bindings. Binding inventory exposes opaque exact-pair IDs; plan/apply uses state CAS, independent candidate verification, a durably bound safe audit identity, a distinct committed STAV event, a retained predecessor, completed-operation status, and crash recovery across every durable stage. All of this is headlessly administered through qxctl. It does not release, store, or exercise credentials, and operational Keychain access and secret delivery remain disabled.

## Quick Start

Use a canonical lowercase UUID as the immutable TOPS ID. Keep the mutable display name separate.

```bash
go test ./...
CGO_ENABLED=0 go build -trimpath -o symphony-ssiag ./cmd/symphony-ssiag
./symphony-ssiag install --scope user
./symphony-ssiag enroll --audit-deferred --scope user \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002 \
  --tops-name "Local TOPS"
./symphony-ssiag supervisor install --audit-deferred --scope user \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002
```

In another terminal:

```bash
qxctl ssiag doctor --scope user \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002
qxctl ssiag status --json --scope user \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002
qxctl ssiag policy status --json --scope user \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002
qxctl ssiag provider show native --json --scope user \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002
qxctl ssiag provider verify native --json --scope user \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002
qxctl ssiag provider installations native --json --scope user \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002
qxctl ssiag provider binding status native --json --scope user \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002
```

Use `qxctl ssiag policy propose` with an exact current policy digest and either a bounded policy file or `--reset`; inspect the emitted proposal, then pass it to `qxctl ssiag policy apply`. If status reports `recovery_required`, use the exact attempt digest with `qxctl ssiag policy recover`. The enrolled config is never rewritten, and every result reports `canonical=false`.

Read `knowledge/ssiag/`, `ARCHITECTURE.md`, `REQUIREMENTS.md`, `THREAT-MODEL.md`, and `IMPLEMENTATION.md` before configuring grants or enabling any canonical mutation or operational provider behavior.
