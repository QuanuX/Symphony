# SOV Cloudflare adapter manifest

This independently installed C++26 component implements the subordinate SOV owner defined in `knowledge/sov/cloudflare/MANIFEST.md`.

Canonical subordinate owner: `knowledge/sov/cloudflare/`, assigned by `knowledge/sov/MANIFEST.md`. Mechanical declaration: `owner.json`. Implementation: `modules/sov-cloudflare-adapter/`; Go presentation: `tools/qxctl/internal/sovcloudflare/`.

Version 0.1.0-dev implements C0/I1 local toolchain lifecycle, bounded offline vendor discovery, context proposals and the first I4 increment of typed resource proposals. No provider resources, credentials, sessions, listeners, system installation or SSH behavior is implemented here. `cf` and Node remain external, separately identified dependencies. `cloudflared` is a future carrier dependency, not an alternative spelling of cf.

Contract companions are INTENT.md, this manifest, SPEC.md and SKILL.md. `owner.json` drives the native descriptor, schemas, templates and Go operation projection. The independent module's receipt owns its executable, descriptor, schemas, templates, contract companions and licenses. The toolchain bundle is a separate immutable content inventory and is not an adapter receipt.

Ownership: provider knowledge stays at `knowledge/scv/scev/cf/`; SOV owns local operations; SSIAG/STAV authority, credentials and audit relationships for provider work are unavailable until I4. OS identity and private store ownership authorize I1 local file operations; no SSIAG/STAV attestation is fabricated.

Optional profile changes use the separate sudo-gated I2 helper; the I3 carrier admits connections as the user. Cloudflare Access session qualification remains I5. The direct SSH v1 contract, digests, proxy prohibitions and defaults remain unchanged. This I1 feature does not grant an exception.
