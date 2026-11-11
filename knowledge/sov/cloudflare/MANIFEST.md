# Native Cloudflare administration manifest

Canonical subordinate SOV owner: `knowledge/sov/cloudflare/`, assigned by `knowledge/sov/MANIFEST.md`. Mechanical declaration: `owner.json`. Independent C++26 implementation: `modules/sov-cloudflare-adapter`; Go discovery/presentation: `tools/qxctl/internal/sovcloudflare/` and `cmd/qxctl/sov_cloudflare.go`.

Package 0.1.0-dev supplies exact cf/Node lifecycle, bounded discovery, persisted nonsecret contexts, 23 typed resource profiles and their execution/status/reconciliation, and separate user-mode cloudflared package/instance lifecycle. The opt-in native credential provider is independently installed from `modules/ssiag-provider-private-store`. Runtime version, complete package receipts, exact provider configuration and current authorization are explicit selections; installation alone provides no credential or authority.

INTENT.md, this manifest, SPEC.md and SKILL.md form the Contract Quad. owner.json drives native descriptors, closed schemas/templates and Go operation projection. Receipt v2 owns the executable, declaration, schema companions, documentation and licenses. External toolchain bundles and connector binaries retain separate exact content identities. The private credential provider exports an exact installed SDK; source and installed-only builds are supported.

SSIAG admits native use and committed STAV records, while OS identity/private ownership govern local state lifecycle. Cloudflare is responsible for actual provider effects; safe readback and reconciliation determine reported outcomes. Local process readiness, provider connectivity and SSH readiness remain separate. Source and fixture qualification do not establish live account/platform acceptance. Token rotation is absent from the pinned cf release, so its mandatory I6 gate remains open. I5 Access sessions and I7 project builds/deployments are separate work.

I5 assigns the native Access entry point and its receipt-owned `access-sessions.json`
projection to `modules/sov-cloudflare-adapter`. The subordinate contract is
`knowledge/sov/CLOUDFLARE-ACCESS-SESSIONS.md`. Local explicit user-session
implementation supersedes the earlier pending-implementation statement; live
Access SSH, additional runtime platforms, and I6 qualification remain open.
