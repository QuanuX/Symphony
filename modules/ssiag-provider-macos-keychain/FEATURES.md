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
          "distinction": "The Go foundation owns receipt-backed adapter selection and launcher-side trust; this Swift adapter owns its independently verified metadata response and disabled-operation behavior.",
          "target_feature_id": "ssfv:symphony:ssiag-foundation.provider-trust-assurance"
        }
      ],
      "evidence": [
        "Swift tests verify receipt-backed installation, invoker trust, handshake, status, capabilities, unknown-field rejection, bounded input, secret-shaped control rejection, credential-operation refusal, and receipt-last uninstall recovery.",
        "The manifest and install contract declare operational_access_enabled false and an independent lifecycle."
      ],
      "feature_id": "ssfv:symphony:ssiag.macos-keychain-metadata",
      "how": "The separately installed Swift executable validates immutable receipt-v2 package evidence, independently verifies the invoking SSIAG identity, then a one-request/one-response bounded JSON standard-input/output protocol accepts only metadata operations and rejects unknown fields, credential operations, and secret-shaped control data.",
      "implementation_languages": [
        {
          "language": "Swift",
          "role": "Implements the independently installed macOS-native bounded metadata handshake and explicit rejection of credential operations."
        }
      ],
      "implementation_paths": [
        "modules/ssiag-provider-macos-keychain/Package.swift",
        "modules/ssiag-provider-macos-keychain/Sources/SSIAGMacOSKeychainSupport/FoundationTrust.swift",
        "modules/ssiag-provider-macos-keychain/Sources/SSIAGMacOSKeychainSupport/Lifecycle.swift",
        "modules/ssiag-provider-macos-keychain/Sources/SSIAGMacOSKeychainSupport/Protocol.swift",
        "modules/ssiag-provider-macos-keychain/Sources/SSIAGMacOSKeychainSupport/ReceiptV2.swift",
        "modules/ssiag-provider-macos-keychain/Sources/SSIAGMacOSKeychainSupport/StrictJSON.swift",
        "modules/ssiag-provider-macos-keychain/Sources/SymphonySSIAGMacOSKeychain/main.swift",
        "modules/ssiag-provider-macos-keychain/Tests/Integration/prepare-real-adapter.sh",
        "modules/ssiag-provider-macos-keychain/Tests/SSIAGMacOSKeychainSupportTests/LifecycleTests.swift",
        "modules/ssiag-provider-macos-keychain/Tests/SSIAGMacOSKeychainSupportTests/ProtocolTests.swift"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not read, write, sign with, decrypt with, rotate, export, or deliver any Keychain credential.",
        "Does not create an implicit fallback, accept secrets through JSON, or claim operational readiness.",
        "The Go SSIAG foundation invokes only the verified metadata handshake; qxctl does not invoke the adapter, and operational Keychain access, credential operations, and secret delivery remain disabled."
      ],
      "owner_contract": "modules/ssiag-provider-macos-keychain/SPEC.md",
      "parent_feature_id": "ssfv:symphony:ssiag-foundation",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "The adapter-side verified metadata response composes with the Go foundation's receipt-bound launcher and response validation.",
          "target_feature_id": "ssfv:symphony:ssiag-foundation.provider-trust-assurance",
          "type": "composes_with"
        }
      ],
      "source_scope": "modules/ssiag-provider-macos-keychain",
      "status": "experimental",
      "title": "macOS Keychain provider metadata adapter",
      "what": "Provides an independently buildable macOS adapter that reports provider identity, status, and capabilities while proving that operational Keychain access is disabled.",
      "when": "Invoked through the verified SSIAG metadata path during macOS provider discovery or diagnostics; no supported request accesses Keychain values.",
      "where": "Runs as a separate Swift executable on macOS 13 or later and never links native Apple frameworks into SSIAG's Go process.",
      "who": "The exact verified SSIAG foundation executable, maintainers, macOS integration tests, and administrators inspecting bounded metadata through SSIAG.",
      "why": "Establishes the correct native-language and out-of-process provider boundary before sensitive Keychain operations are separately authorized."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records the reviewed signed-bundle readiness change after merge.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the native readiness source and evidence.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future Developer ID and notarized provider publication.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        },
        {
          "applicability": "applicable",
          "reason": "SSIAG owns the complete bundle and three-layer readiness protocol.",
          "reference": "knowledge/ssiag/PROVIDER-READINESS.md",
          "vector": "ssiag"
        }
      ],
      "distinctions": [
        {
          "distinction": "The Swift adapter owns native observation and bundle lifecycle; the Go foundation owns exact selection, private reconstruction, route authority, and result binding.",
          "target_feature_id": "ssfv:symphony:ssiag-foundation.provider-readiness-assurance"
        },
        {
          "distinction": "Metadata protocol behavior proves a bounded mutual handshake; this feature observes complete bundle structure, protected native policy, and session capability through a separate protocol.",
          "target_feature_id": "ssfv:symphony:ssiag.macos-keychain-metadata"
        }
      ],
      "evidence": [
        "Swift lifecycle tests cover sorted multi-file receipt ownership, idempotent bundle installation, unknown/changed-byte refusal, and receipt-last interrupted-uninstall recovery.",
        "Readiness tests cover structural/policy separation, native SecRequirement mismatch, unsafe profile shape, explicit disabled eligibility, safe JSON exclusions, and absence of Keychain item/key operations.",
        "The production bundle builder creates the complete bundle before hardened Developer ID signing, secure timestamp, notarization, and stapling."
      ],
      "feature_id": "ssfv:symphony:ssiag.macos-signed-bundle-readiness",
      "how": "Installs a complete app-like bundle under exact receipt-v2 ownership, observes only safe provisioning-file shape, validates the complete signature envelope, compiles and evaluates the receipt-owned native code requirement, observes bounded Security Session flags, and returns separate structural, policy, and disabled-eligibility layers.",
      "implementation_languages": [
        {
          "language": "Swift",
          "role": "Implements app-bundle lifecycle and Apple Security structural, native requirement, and session observation outside the Go process."
        }
      ],
      "implementation_paths": [
        "modules/ssiag-provider-macos-keychain/Package.swift",
        "modules/ssiag-provider-macos-keychain/Sources/SSIAGMacOSKeychainSupport/Lifecycle.swift",
        "modules/ssiag-provider-macos-keychain/Sources/SSIAGMacOSKeychainSupport/ReceiptV2.swift",
        "modules/ssiag-provider-macos-keychain/Sources/SSIAGMacOSKeychainSupport/SignedBundleReadiness.swift",
        "modules/ssiag-provider-macos-keychain/Sources/SymphonySSIAGMacOSKeychain/main.swift",
        "modules/ssiag-provider-macos-keychain/Tests/SSIAGMacOSKeychainSupportTests/LifecycleTests.swift",
        "modules/ssiag-provider-macos-keychain/Tests/SSIAGMacOSKeychainSupportTests/SignedBundleReadinessTests.swift",
        "modules/ssiag-provider-macos-keychain/scripts/build-production-bundle.sh"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not parse or emit certificate, profile, entitlement, native-error, requirement-text, or security-session-identity payloads.",
        "Does not claim notarization from local signature validity and does not call any Keychain item or key operation.",
        "Does not make unsigned, ad-hoc, bare, policy-absent, headless, or remote execution operational and does not fall back to another provider."
      ],
      "owner_contract": "modules/ssiag-provider-macos-keychain/SPEC.md",
      "parent_feature_id": "ssfv:symphony:ssiag-foundation",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Native observation is bound and validated by the cgo-free foundation before reaching qxctl.",
          "target_feature_id": "ssfv:symphony:ssiag-foundation.provider-readiness-assurance",
          "type": "composes_with"
        },
        {
          "rationale": "The existing metadata protocol remains the independently authenticated provider-control foundation.",
          "target_feature_id": "ssfv:symphony:ssiag.macos-keychain-metadata",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/ssiag-provider-macos-keychain",
      "status": "experimental",
      "title": "macOS signed-bundle and security-session readiness",
      "what": "Provides production-bundle lifecycle and decision-neutral native evidence needed before any data-protection Keychain operation can be designed or enabled.",
      "when": "On explicit readiness observation, installation, upgrade, rollback, uninstall, or recovery of one exact adapter version in a macOS user-login environment.",
      "where": "Inside the independently installed Swift app-like bundle and its Apple Security boundary, entirely outside the cgo-free Go foundation and all hot or warm paths.",
      "who": "The exact verified SSIAG foundation, target-host administrators and agents through qxctl, release engineers, maintainers, and security reviewers.",
      "why": "Prevents packaging, signing, policy, and session prerequisites from being hidden or conflated with operational authority while preserving first-class headless administration."
    }
  ],
  "source_scope": "modules/ssiag-provider-macos-keychain"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
