# Symphony Cloud Edge Vector — Cloudflare Manifest

## Canonical Target

`knowledge/scv/scev/cf/`

## Declared Contract Truth Role

Cloudflare-native source, service, platform version and constraint knowledge. Parent semantics are delegated by `knowledge/scv/scev/SPEC.md`; the installed engine remains subordinate to this contract.

## Canonical Surfaces

- `knowledge/scv/scev/cf/INTENT.md`
- `knowledge/scv/scev/cf/MANIFEST.md`
- `knowledge/scv/scev/cf/SKILL.md`
- `knowledge/scv/scev/cf/SPEC.md`

## Implementation and Projection

`modules/scev-cf-engine/` provides independently packaged C++26 `symphony-scev-cf` at `0.1.0-dev`, using the shared SCV source-knowledge implementation. The source corpus, interpretation records and selected graph remain private installation data; they do not rewrite the canonical repository. Complete provider coverage, a graph database product, a network API and provider operations are not claimed.
