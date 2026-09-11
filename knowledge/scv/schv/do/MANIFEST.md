# Symphony Cloud Hyperscalers Vector — DigitalOcean Manifest

## Canonical Target

`knowledge/scv/schv/do/`

## Declared Contract Truth Role

DigitalOcean-native source, offering, service, version and constraint knowledge. Parent semantics are delegated by `knowledge/scv/schv/SPEC.md`; the installed engine remains subordinate to this contract.

## Canonical Surfaces

- `knowledge/scv/schv/do/INTENT.md`
- `knowledge/scv/schv/do/MANIFEST.md`
- `knowledge/scv/schv/do/SKILL.md`
- `knowledge/scv/schv/do/SPEC.md`

## Implementation and Projection

`modules/schv-do-engine/` provides independently packaged C++26 `symphony-schv-do` at `0.2.0-dev`, using the shared SCV source-knowledge implementation. The source corpus, interpretation records and selected graph remain private installation data; they do not rewrite the canonical repository. Complete provider coverage, a graph database product, a network API and provider operations are not claimed.

The additive corpus operations retain explicit immutable snapshot identities and introduce no selected corpus head. Existing `0.1.0-dev` installations retain their original exact operation and data contracts.
