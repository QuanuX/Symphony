# Accordare STAV Producer Skill

## Safe Procedure

1. Read `knowledge/ACCORD-AUDIT.md`, `knowledge/sav/STAV.md`, the Accordare registry, and this module's Contract Quad.
2. Preserve the separation between typed evidence input, safe candidate derivation, durable outbox, and STAV append.
3. Treat package installation, producer enrollment, SSIAG permission, STAV producer grant, and live supervision as distinct states.
4. Use qxctl for grant administration; require the STAV authority to be stopped and bind exact expected configuration state.
5. Run module tests, qxctl command-registry parity, race tests, cgo-disabled builds, and repository validation.

## Prohibited Changes

Do not add arbitrary append fields, accept caller-selected STAV vocabulary, persist upstream bodies/proofs, weaken kernel peer checks, silently drop pending candidates, infer a grant from installation, rewrite ledgers, add Windows-native support, or move this freezing-path service toward a trading hot/warm path.
