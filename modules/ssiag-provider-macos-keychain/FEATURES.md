# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/ssiag-provider-macos-keychain/SPEC.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed adapter changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the adapter contracts and implementation.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future adapter release.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        },
        {
          "applicability": "applicable",
          "reason": "SSIAG owns provider protocol, trust, and secret-delivery boundaries.",
          "reference": "knowledge/ssiag/SPEC.md",
          "vector": "ssiag"
        }
      ],
      "distinctions": [
        {
          "distinction": "The Go foundation owns identity and policy; this Swift adapter owns only macOS provider metadata and disabled-operation behavior.",
          "target_feature_id": "ssfv:symphony:ssiag-foundation"
        }
      ],
      "evidence": [
        "Swift tests verify hello, status, capabilities, unknown-field rejection, bounded input, and credential-operation refusal.",
        "The manifest and install contract declare operational_access_enabled false and an independent lifecycle."
      ],
      "feature_id": "ssfv:symphony:ssiag.macos-keychain-metadata",
      "how": "A bounded JSON-lines standard-input/output protocol accepts only metadata operations, rejects unknown fields and credential operations, and links only Apple system frameworks outside the Go foundation.",
      "implementation_languages": [
        {
          "language": "Swift",
          "role": "Implements the independently installed macOS-native bounded metadata handshake and explicit rejection of credential operations."
        }
      ],
      "implementation_paths": [
        "modules/ssiag-provider-macos-keychain/Package.swift",
        "modules/ssiag-provider-macos-keychain/Sources/SymphonySSIAGMacOSKeychain/main.swift",
        "modules/ssiag-provider-macos-keychain/Tests/SSIAGMacOSKeychainSupportTests/ProtocolTests.swift"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not read, write, sign with, decrypt with, rotate, export, or deliver any Keychain credential.",
        "Does not create an implicit fallback, accept secrets through JSON, or claim operational readiness.",
        "The current Go SSIAG foundation and qxctl do not invoke this adapter; operational trust, provider execution, Keychain access, and secret delivery remain unimplemented."
      ],
      "owner_contract": "modules/ssiag-provider-macos-keychain/SPEC.md",
      "parent_feature_id": "ssfv:symphony:ssiag-foundation",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "The adapter independently demonstrates the platform-native metadata protocol intended for SSIAG's future provider boundary, but no Go-foundation invocation bridge is implemented.",
          "target_feature_id": "ssfv:symphony:ssiag-foundation",
          "type": "extends"
        }
      ],
      "source_scope": "modules/ssiag-provider-macos-keychain",
      "status": "experimental",
      "title": "macOS Keychain provider metadata adapter",
      "what": "Provides an independently buildable macOS adapter that reports provider identity, status, and capabilities while proving that operational Keychain access is disabled.",
      "when": "Invoked explicitly for provider metadata during macOS development or administration; no supported request accesses Keychain values.",
      "where": "Runs as a separate Swift executable on macOS 13 or later and never links native Apple frameworks into SSIAG's Go process.",
      "who": "Maintainers, macOS integration tests, and administrators who invoke the independently installed adapter directly for its bounded metadata handshake.",
      "why": "Establishes the correct native-language and out-of-process provider boundary before sensitive Keychain operations are separately authorized."
    }
  ],
  "source_scope": "modules/ssiag-provider-macos-keychain"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
