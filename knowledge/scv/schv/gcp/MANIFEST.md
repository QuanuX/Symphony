# Symphony Cloud Hyperscalers Vector — Google Cloud Manifest

## Canonical Target

`knowledge/scv/schv/gcp/`

## Declared Contract Truth Role

Google Cloud-native source, offering, service, version and constraint knowledge. Parent semantics are delegated by `knowledge/scv/schv/SPEC.md`; the installed engine remains subordinate to this contract.

## Canonical Surfaces

- `knowledge/scv/schv/gcp/INTENT.md`
- `knowledge/scv/schv/gcp/MANIFEST.md`
- `knowledge/scv/schv/gcp/SKILL.md`
- `knowledge/scv/schv/gcp/SPEC.md`

## Implementation and Projection

`modules/schv-gcp-engine/` provides independently packaged C++26 `symphony-schv-gcp` at `0.1.0-dev`, using the shared SCV source-knowledge implementation. The source corpus, interpretation records and selected graph remain private installation data; they do not rewrite the canonical repository. Complete provider coverage, a graph database product, a network API and provider operations are not claimed.
