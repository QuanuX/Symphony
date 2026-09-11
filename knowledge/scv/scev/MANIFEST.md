# Symphony Cloud Edge Vector Manifest

## Canonical Target

`knowledge/scv/scev/`

## Declared Contract Truth Role

Edge-provider family knowledge, preserving distinct product and service boundaries. Parent semantics are delegated by `knowledge/scv/SPEC.md`; the installed engine remains subordinate to this contract.

## Canonical Surfaces

- `knowledge/scv/scev/INTENT.md`
- `knowledge/scv/scev/MANIFEST.md`
- `knowledge/scv/scev/SKILL.md`
- `knowledge/scv/scev/SPEC.md`

## Implementation and Projection

`modules/scev-engine/` provides independently packaged C++26 `symphony-scev` at `0.3.0-dev`, using the shared SCV source-knowledge implementation. The source corpus, interpretation records and selected graph remain private installation data; they do not rewrite the canonical repository. Complete provider coverage, a graph database product, a network API and provider operations are not claimed.

The additive corpus operations retain explicit immutable snapshot identities and introduce no selected corpus head. Existing `0.1.0-dev` and `0.2.0-dev` installations retain their original exact operation and data contracts. The additive interpretation/connection profile preserves exact authored mappings, qualification and caller-selected checks under `knowledge/scv/INTERPRETATION.md`; it does not establish full native comprehension or deployed compatibility.
