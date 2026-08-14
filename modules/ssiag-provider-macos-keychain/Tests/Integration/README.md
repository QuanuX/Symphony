# Real Adapter Integration Fixture

`prepare-real-adapter.sh` creates an actual immutable receipt-v2 Swift adapter package at an explicit disposable prefix. It emits the adapter's package-result JSON to standard output and sends Swift build progress to standard error.

This fixture is intentionally the reusable adapter half of mutual trust. The complete receipt-backed test lives at `modules/secure-identity-access-governance/tests/provider_trust_integration_darwin.sh`, where it can use the real Go package installer, protected per-TOPS configuration and trust declaration, Unix server, internal launcher, and qxctl client. That test:

1. build and receipt the actual Go `symphony-ssiag` executable;
2. invoke this fixture to build and receipt the actual Swift adapter;
3. configure the exact enabled metadata provider with `interactive: true` and `exportable: false`;
4. create the strict digest-bound executable-trust declaration;
5. start the receipted Go foundation as the adapter's direct parent;
6. execute `qxctl ssiag provider verify`; and
7. require `foundation_verified_adapter: true`, `adapter_verified_foundation: true`, all three operational flags false, and a valid result digest.

A fake launcher, shell provider, fabricated response, direct adapter invocation, or unreceipted parent cannot satisfy this gate.
