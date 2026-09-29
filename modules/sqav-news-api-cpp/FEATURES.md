# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/sqav-news-api-cpp/SPEC.md",
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
        "modules/sqav-news-api-cpp/tests/test.cpp verifies the selected positive and refusal boundaries.",
        "modules/sqav-news-api-cpp/tests/sdk-consumer/main.cpp verifies public installed admission and refusal independently of source include paths."
      ],
      "feature_id": "ssfv:symphony:sqav-news-api-cpp",
      "how": "The user selected an API surface without a news vendor. The exact UTF-8 JSON envelope has thirteen fields: schema=sqav-news-ingress-v1, provider, publisher, article, revision, supersedes, published_at, updated_at, language, rights_ref, source_uri, headline and text. Unknown envelope fields are refused in version 1. Metadata tokens are bounded; headline is nonempty up to 1,024 bytes, source URI begins https:// and is at most 2,048 bytes, body is nonempty and bounded by the selected JSON string/input limits. Publication/update times require UTC RFC3339 seconds with optional 1–9 fractional digits; leap-second values are not admitted in this profile. A first revision has empty supersedes and no predecessor. A correction supplies the actual immutable predecessor, exact reference, same provider/publisher/article/publication value, a changed revision ID and nondecreasing update time. All article bytes remain original. Capture reports completion only for one declared article revision, never an entire feed. Rights metadata is retained in the original envelope and grants no permissions. Source URI and article text are data and are never fetched or executed by this API.",
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
        "modules/sqav-news-api-cpp/CMakeLists.txt",
        "modules/sqav-news-api-cpp/include/symphony/sqav/news.hpp",
        "modules/sqav-news-api-cpp/src/news.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "No selected publisher/feed, vendor wire connector, HTTP server, article scraping, full-feed completeness, licence grant, persistent correction catalogue or mutation of prior revisions."
      ],
      "owner_contract": "modules/sqav-news-api-cpp/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Consumes the exact selected native dependency.",
          "target_feature_id": "ssfv:symphony:native-source-support-cpp",
          "type": "depends_on"
        },
        {
          "rationale": "Consumes the exact selected native dependency.",
          "target_feature_id": "ssfv:symphony:sqav-capture-cpp",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sqav-news-api-cpp",
      "status": "experimental",
      "title": "Vendor-neutral news ingress API",
      "what": "Immutable original/corrected article revisions with exact predecessor evidence and separate time roles.",
      "when": "Only when a compatible trusted caller explicitly invokes the installed library; installation starts no service.",
      "where": "The independently installed sqav-news-api-cpp library.",
      "who": "Explicit native C++26 callers within the selected source contract.",
      "why": "Immutable original/corrected article revisions with exact predecessor evidence and separate time roles."
    }
  ],
  "source_scope": "modules/sqav-news-api-cpp"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
