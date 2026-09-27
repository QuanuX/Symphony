# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/sqtv-integer-conversion-cpp/SPEC.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "Source evolution and scope claims remain reviewable through SCLV.",
          "reference": "knowledge/sclv/SPEC.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "The exact module contract, implementation and focused evidence have independent indexed routes.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        }
      ],
      "distinctions": [],
      "evidence": [
        "modules/sqtv-integer-conversion-cpp/tests/integer_conversion_test.cpp verifies all 256 format pairs, lineage, rejection, retained delivery and allocation rollback.",
        "modules/sqtv-integer-conversion-cpp/tests/sdk-consumer/main.cpp verifies installed conversion, an independent operation digest and atomic rejection.",
        "modules/sqtv-integer-conversion-cpp/tests/package_lifecycle_test.cmake verifies exact installation and guarded removal."
      ],
      "feature_id": "ssfv:symphony:sqtv-integer-conversion-cpp",
      "how": "Two bounded passes validate exact representability and encode a new SQFV batch with derived SQMV metadata.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements the module owner contract and focused native verification."
        },
        {
          "language": "CMake",
          "role": "Builds, exports, receipts, and guarded-uninstalls the exact library package."
        }
      ],
      "implementation_paths": [
        "modules/sqtv-integer-conversion-cpp/CMakeLists.txt",
        "modules/sqtv-integer-conversion-cpp/include/symphony/sqtv/integer_conversion.hpp",
        "modules/sqtv-integer-conversion-cpp/src/integer_conversion.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "No provider parsing, nullable/sentinel, decimal, floating-point, timestamp reinterpretation or lossy conversion.",
        "No stateful operation, transformation cache, background service, foreign ABI or SQV qxctl surface."
      ],
      "owner_contract": "modules/sqtv-integer-conversion-cpp/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Uses actual immutable SQFV batches and leases with caller-selected output positions.",
          "target_feature_id": "ssfv:symphony:sqfv-batch-cpp",
          "type": "depends_on"
        },
        {
          "rationale": "Verifies exact input binding and preserves semantic evidence with derived conversion lineage.",
          "target_feature_id": "ssfv:symphony:sqmv-metadata-cpp",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sqtv-integer-conversion-cpp",
      "status": "experimental",
      "title": "SQTV exact integer representation conversion library",
      "what": "Converts dense signed or unsigned integer width and byte order without value loss, retaining input and operation lineage.",
      "when": "Only when a compatible trusted caller explicitly invokes the installed library; installation starts no service.",
      "where": "Inside an explicit native caller using the exact installed static library.",
      "who": "Trusted C++26 callers selecting the declared integer domain.",
      "why": "Allow explicit compatible representations without silent integer loss or discarded evidence."
    }
  ],
  "source_scope": "modules/sqtv-integer-conversion-cpp"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
