# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/shv-pdf-adapter/SPEC.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "Completed material source changes require reviewed append-only closure.",
          "reference": "knowledge/sclv/SPEC.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes current owner contracts and distributed feature files.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "Future official publication follows a separate authorized release boundary.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [],
      "evidence": [
        "Focused SHV-15 extraction and SHV-16 assertion graph replay evidence."
      ],
      "feature_id": "ssfv:symphony:shv-pdf-adapter",
      "how": "Independent C++ adapter plus strict qxctl correspondence and replay.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Owns bounded PDF decoder invocation, table interpretation and derivation identity."
        },
        {
          "language": "CMake",
          "role": "Builds and receipts the independently installed adapter; the caller supplies the external decoder."
        }
      ],
      "implementation_paths": [
        "modules/shv-pdf-adapter/CMakeLists.txt",
        "modules/shv-pdf-adapter/src/descriptor.cpp",
        "modules/shv-pdf-adapter/src/graph.cpp",
        "modules/shv-pdf-adapter/src/main.cpp",
        "modules/shv-pdf-adapter/src/pdf.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "No universal PDF parser, OCR, publisher authentication, OPN/tray namespace equivalence or catalogue publication."
      ],
      "owner_contract": "modules/shv-pdf-adapter/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Authority-free process, path and digest mechanics.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/shv-pdf-adapter",
      "status": "experimental",
      "title": "Bounded SHV PDF table interpretation",
      "what": "Extracts one explicit AMD document table through a caller-selected PDFium binary with original-byte and decoder identity binding.",
      "when": "Explicit bounded read-only operations only.",
      "where": "Explicit installed adapter, retained PDF and selected trusted PDFium library.",
      "who": "Agents and humans invoking the exact installed SHV through qxctl.",
      "why": "Interpret document evidence without losing original lineage or inventing identifier namespace equivalence."
    }
  ],
  "source_scope": "modules/shv-pdf-adapter"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
