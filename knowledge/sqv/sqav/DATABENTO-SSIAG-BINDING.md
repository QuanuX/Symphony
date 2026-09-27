# Databento source → SSIAG credential binding

Status: implementation dependency map, 2026-09-27. Duncan explicitly selected
SSIAG and the local key ring for Databento. This map does not claim that the
current SSIAG metadata scaffold can retrieve a key.

## Exact owners

| Concern | Owner / source contract |
|---|---|
| Data source and request scope | SQAV; `knowledge/sqv/sqav/SPEC.md` |
| Original response and request attribution | `sqav-capture-cpp` and `sqav-databento-dbn-cpp` |
| Credential reference, identity, grants and leases | SSIAG; `modules/secure-identity-access-governance/internal/credential/model.go` |
| Actual macOS key-ring storage and access | Separate `ssiag-provider-macos-keychain` Swift process |
| Native Databento transport | Future SQAV C++26 collector; receives only an explicitly authorized credential use/delivery |
| Safe authorization audit | SSIAG/STAV; neither receives market payloads as audit data |

The proposed logical source is `databento`, with three separate surface bindings:

- Historical: `https://hist.databento.com/v0/`, HTTP Basic authentication,
  `metadata.*`, `symbology.resolve`, and separately authorized `timeseries.get_range`.
- Live: exact selected dataset gateway (for this test,
  `glbx-mdp3.lsg.databento.com:13000`), provider CRAM authentication and bounded
  subscriptions. Authentication is not evidence of subscription entitlement.
- Reference: distinct Databento reference API operations, to be selected and
  admitted separately. The shared account/key does not establish endpoint or
  entitlement compatibility.

All three refer to one selected SSIAG credential reference by provider, name,
version and type, plus exact immutable TOPS, SSIAG subject/consumer, target
provider binding digest and authorized operation/audience. These deployment
values remain **unbound**, not guessed from a developer fixture. A provider name
such as `native` is configured per TOPS; it is not a universal provider identity.
The key label `test-1` supplied by the user is account metadata, not proof of a
Keychain item identity, SSIAG reference, export grant or entitlement.

## Resolution path required for an operational connector

1. Resolve the source's nonsecret credential reference under the chosen TOPS.
2. SSIAG verifies the exact consumer, current grants, provider binding and lease.
3. The signed Keychain provider authenticates SSIAG and selects the exact item
   within the TOPS namespace and selected private access group.
4. For live CRAM, assess provider-owned challenge response before exporting a
   key. Historical Basic auth still requires an authorized native HTTP sink;
   generating the Basic header is itself secret-bearing delivery.
5. Only a ratified protected one-shot channel or provider-owned operation may
   deliver/use the credential. Control JSON, environment, command arguments,
   qxctl and STAV remain nonsecret surfaces. No `security` CLI fallback.
6. Revoke/expire/lock/unavailable states fail before provider network activity.
   Rotation changes the reference generation; an old lease must not resolve the
   replacement credential silently.
7. Source provenance retains safe binding/authorization references, never key
   bytes, Basic headers, CRAM response/bucket identifiers or secret-derived hashes.

The map is not an executable SSIAG protocol and allocates no colon namespace.

## Actual blocking contracts at a1b6958

`modules/secure-identity-access-governance/README.md` states that the foundation
does not release, store or exercise credentials. The Keychain provider's SPEC
admits only metadata and signed-bundle readiness. Item lifecycle and protected
secret delivery are unimplemented. Provider control v1 has only capabilities,
handshake and status. Its synthetic descriptor is -1 with zero capacity and
cannot become operational by toggling flags.

The missing work is a named operational SSIAG increment: item lifecycle and
access-control policy; authenticated importer; exact signed provider/foundation
and TOPS binding; bounded request-bound consumer delivery or challenge operation;
lease/revocation/cancellation and audit conformance. An actual deployment also
needs the selected TOPS, subject and production signing/access-group identities.
No configured deployment was inferred from test installation directories.

## Evidence and open scope

The SQV-13 research probe separately exercised the supplied key through a
private process pipe. It did not perform a Keychain import or SSIAG retrieval.
Production integration remains unavailable until the above SSIAG owner gates
and concrete deployment binding are implemented. The offline DBN library
neither resolves credentials nor claims provider authentication.

Provider references reviewed 2026-09-27:
- https://databento.com/docs/api-reference-historical?historical=http
- https://databento.com/docs/api-reference-live?live=raw
