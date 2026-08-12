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

## Safe-Use Rules

- Treat `knowledge/ssiag/` and `knowledge/stav/` as protocol truth and code as implementation truth.
- Keep ID and name fields separate; use IDs for security scope.
- Never place secrets, proofs, tokens, or provider payloads in flags, environment variables, logs, fixtures, JSON output, manifests, or Knowledge Vectors.
- Keep qxctl provider-neutral and free of secrets; safe authorization decisions are metadata, not transferable credentials.
- Keep all foundation source Go-only and cgo-free.
- Keep native platform code in independent adapters.
- Fail closed when a provider or capability is absent.
- Do not create an SCLV merge record before real review and merge evidence exists.

## Do Not Use For

Kernel peer authentication is enabled automatically for the local API; never substitute a caller-supplied identity or socket permissions for it. Exact-grant authorization decisions require a mapped kernel subject and committed STAV receipt. Local policy administration is the narrow exception: use only `qxctl ssiag policy ...`, keep enrolled config immutable, require target-host ownership or exact current grants, preserve CAS/audit/recovery evidence, and keep `canonical=false`. Capabilities are short-lived and non-transferable. No caller may submit arbitrary STAV events. Supervision owns liveness only. Do not use this foundation for credential access, general safeguard administration, plaintext providers, canonical knowledge apply, or hot-path authorization.
