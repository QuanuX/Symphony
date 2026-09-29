# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/native-source-support-cpp/SPEC.md",
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
        "modules/native-source-support-cpp/tests/test.cpp verifies the selected positive and refusal boundaries.",
        "modules/native-source-support-cpp/tests/sdk-consumer/main.cpp verifies public installed admission and refusal independently of source include paths."
      ],
      "feature_id": "ssfv:symphony:native-source-support-cpp",
      "how": "JSON: 1–64 MiB input, 1–1,048,576 syntax events, caller-bounded strings, depth 1–64, duplicate-key and invalid UTF-8 refusal. Numbers may be inspected in the DOM; exact financial evidence remains the original bytes and is never reserialized from binary floating point. Date validation covers Gregorian years 1–9999. HTTP: 1–300,000 ms total deadline, positive connection deadline no greater than total, 1–65,536 header bytes, 1–64 MiB body. GET/POST only, explicit HTTPS origin/path, verified TLS, no redirects, proxies, netrc, cookies or diagnostics. HTTP error bodies are counted against the budget and discarded. Interrupted bodies cannot become complete when a credential callback swallows an error. The actual credential owner must enforce its authenticated lease/recipient and callback deadline; no operational SSIAG bridge is provided. FRED query-key and Databento Basic username are separate exact credential profiles. All secret-bearing URL/key copies are transient and locally overwritten; libcurl/system copies do not carry an impossible erasure guarantee. The production library has no test endpoint override. After fork, network calls refuse before using inherited library state.",
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
        "modules/native-source-support-cpp/CMakeLists.txt",
        "modules/native-source-support-cpp/include/symphony/source/json.hpp",
        "modules/native-source-support-cpp/include/symphony/source/support.hpp",
        "modules/native-source-support-cpp/src/http.cpp",
        "modules/native-source-support-cpp/src/support.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "No operational identity authority, automatic retry, durable spend authority, provider semantics, generic guarantee of bounded allocator overhead or knowledge-process payload envelope."
      ],
      "owner_contract": "modules/native-source-support-cpp/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [],
      "source_scope": "modules/native-source-support-cpp",
      "status": "experimental",
      "title": "Bounded native source support",
      "what": "Strict bounded provider JSON syntax and explicit one-shot HTTPS acquisition.",
      "when": "Only when a compatible trusted caller explicitly invokes the installed library; installation starts no service.",
      "where": "The independently installed native-source-support-cpp library.",
      "who": "Explicit native C++26 callers within the selected source contract.",
      "why": "Strict bounded provider JSON syntax and explicit one-shot HTTPS acquisition."
    }
  ],
  "source_scope": "modules/native-source-support-cpp"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
