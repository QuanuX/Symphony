# Native SSIAG private credential provider manifest

Component `ssiag-provider-private-store`, package `0.1.0-dev`, SDK `SymphonySSIAGPrivateStore 0.1.0 EXACT`, target `Symphony::SSIAGPrivateChannel`. The header-only native IPC SDK depends on `SymphonyKnowledgeVectorEngine 0.2.0 EXACT`; it does not include Go or qxctl. The independent executable is `symphony-ssiag-provider-private-store`.

The contract quad is INTENT.md, MANIFEST.md, SPEC.md and SKILL.md. INSTALL.md documents separate source and installed-SDK builds. FEATURES.md records experimental availability. Receipt v2 owns the executable, exact SDK exports/header, documentation and licenses. Source tests exercise configuration CAS, private FD import, metadata inspection and revocation. The SOV adapter's execution/connector fixtures exercise authenticated native sessions and sensitive import; SSIAG server tests independently exercise actual policy pinning and committed STAV receipts.

Go SSIAG owns authorization and audit admission at `/v1/cloudflare/use`. The native owner verifies its kernel peer and selected service executable digest. Explicit local bootstrap imports an operator-provided token through inherited FD 3; it is never an implicit provider login or an automatic secret-store choice. Other credential backends remain independently composable.
