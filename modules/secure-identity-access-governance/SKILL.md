# Symphony Secure Identity and Access Governance Skill

## Preferred Verification

1. read `knowledge/ssiag/` and `knowledge/stav/`;
2. run `go test ./...` and `go vet ./...`;
3. build with `CGO_ENABLED=0`;
4. install the host binary in a temporary home;
5. enroll two distinct TOPS UUIDs and verify no path collision;
6. serve one enrollment and query it with `qxctl ssiag ... --tops-id`;
7. install one descriptor with `supervisor install --no-start`, verify its TOPS-bound label and liveness-only contents, then remove it with `--no-stop`;
8. verify a display-name change leaves paths unchanged;
9. verify uninstall preserves both TOPS configurations;
10. run the repository validator.
11. exercise `qxctl ssiag provider installations` and all five `provider binding` commands against an installed metadata adapter, including interrupted recovery and completed apply-status replay.

## Safe-Use Rules

- Treat `knowledge/ssiag/` and `knowledge/stav/` as protocol truth and code as implementation truth.
- Keep ID and name fields separate; use IDs for security scope.
- Never place secrets, proofs, tokens, or provider payloads in flags, environment variables, logs, fixtures, JSON output, manifests, or Knowledge Vectors.
- Keep qxctl provider-neutral and free of secrets; safe authorization decisions are metadata, not transferable credentials.
- Keep all foundation source Go-only and cgo-free.
- Keep native platform code in independent adapters.
- Fail closed when a provider or capability is absent.
- Select provider bindings only by SSIAG-issued opaque installation ID and exact state digest; never by path, raw version, filesystem order, or newest semantics.
- Do not create an SCLV merge record before real review and merge evidence exists.

## Do Not Use For

Kernel peer authentication is enabled automatically for the local API; never substitute caller-supplied identity or socket permissions. Use qxctl for ordinary policy and provider-binding administration. Binding mutation requires target-host ownership or an exact current grant, independent candidate verification, a durably bound safe audit identity, and a committed STAV receipt. The native offline recovery command is not a general bypass: it requires the immutable receipt-v2 foundation, target-host ownership, entry into the enrolled service UID/GID, exclusive ownership of the persistent socket-lifecycle lease, an absent socket, existing exact attempt evidence, and normal idempotent STAV commitment. Do not use binding success as credential or Keychain authority.
