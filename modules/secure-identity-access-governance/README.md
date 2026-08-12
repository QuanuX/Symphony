# Symphony Secure Identity and Access Governance

SSIAG is Symphony's Go-only, independently installable identity, authentication, authorization, capability, credential-reference, lease, and provider-operation foundation.

The current foundation installs one host binary, enrolls multiple isolated TOPS instances, authenticates every accepted Darwin/Linux Unix-socket connection from kernel peer credentials, verifies the configured service identity on both sides, installs bounded per-TOPS launchd/systemd liveness profiles, serves safe metadata, evaluates exact deny-by-default authorization grants, administers a protected local policy overlay through proposal/apply/recovery, and provides a closed typed producer for durable STAV security outcomes. Policy changes are caller-neutral, compare-and-swap, audit-before-commit, and crash-recoverable. Its short-lived capabilities remain non-transferable and cannot authorize canonical knowledge apply. It does not release, store, or exercise credentials.

## Quick Start

Use a canonical lowercase UUID as the immutable TOPS ID. Keep the mutable display name separate.

```bash
go test ./...
CGO_ENABLED=0 go build -trimpath -o symphony-ssiag ./cmd/symphony-ssiag
./symphony-ssiag install --scope user
./symphony-ssiag enroll --scope user \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002 \
  --tops-name "Local TOPS"
./symphony-ssiag supervisor install --scope user \
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
```

Use `qxctl ssiag policy propose` with an exact current policy digest and either a bounded policy file or `--reset`; inspect the emitted proposal, then pass it to `qxctl ssiag policy apply`. If status reports `recovery_required`, use the exact attempt digest with `qxctl ssiag policy recover`. The enrolled config is never rewritten, and every result reports `canonical=false`.

Read `knowledge/ssiag/`, `ARCHITECTURE.md`, `REQUIREMENTS.md`, `THREAT-MODEL.md`, and `IMPLEMENTATION.md` before configuring grants or enabling any canonical mutation or operational provider behavior.
