# Symphony Secure Identity and Access Governance

SSIAG is Symphony's Go-only, independently installable identity, authentication, authorization, capability, credential-reference, lease, and provider-operation foundation.

The current foundation is deliberately safe and limited: it installs one host binary, enrolls multiple isolated TOPS instances, authenticates every accepted Darwin/Linux Unix-socket connection from kernel peer credentials, verifies the configured service identity on both sides of each client connection, serves metadata, integrates read-only status/provider inspection with qxctl, and provides a closed typed producer for durable STAV security outcomes. It does not release, store, or exercise credentials.

## Quick Start

Use a canonical lowercase UUID as the immutable TOPS ID. Keep the mutable display name separate.

```bash
go test ./...
CGO_ENABLED=0 go build -trimpath -o symphony-ssiag ./cmd/symphony-ssiag
./symphony-ssiag install --scope user
./symphony-ssiag enroll --scope user \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002 \
  --tops-name "Local TOPS"
./symphony-ssiag serve --scope user \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002
```

In another terminal:

```bash
qxctl ssiag doctor --scope user \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002
qxctl ssiag status --json --scope user \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002
```

Read `knowledge/ssiag/`, `ARCHITECTURE.md`, `REQUIREMENTS.md`, `THREAT-MODEL.md`, and `IMPLEMENTATION.md` before enabling any mutation or operational provider behavior.
