# Symphony Cloud Hyperscalers Vector — AWS Manifest

## Canonical Target

`knowledge/scv/schv/aws/`

## Declared Contract Truth Role

AWS-native source, offering, service, version and constraint knowledge. Parent semantics are delegated by `knowledge/scv/schv/SPEC.md`; the installed engine remains subordinate to this contract.

## Canonical Surfaces

- `knowledge/scv/schv/aws/INTENT.md`
- `knowledge/scv/schv/aws/MANIFEST.md`
- `knowledge/scv/schv/aws/SKILL.md`
- `knowledge/scv/schv/aws/SPEC.md`

## Implementation and Projection

`modules/schv-aws-engine/` provides independently packaged C++26 `symphony-schv-aws` at `0.3.0-dev`, using the shared SCV source-knowledge implementation. The source corpus, interpretation records and selected graph remain private installation data; they do not rewrite the canonical repository. Complete provider coverage, a graph database product, a network API and provider operations are not claimed.

The additive corpus operations retain explicit immutable snapshot identities and introduce no selected corpus head. Existing `0.1.0-dev` and `0.2.0-dev` installations retain their original exact operation and data contracts. The additive interpretation/connection profile preserves exact authored mappings, qualification and caller-selected checks under `knowledge/scv/INTERPRETATION.md`; it does not establish full native comprehension or deployed compatibility.
