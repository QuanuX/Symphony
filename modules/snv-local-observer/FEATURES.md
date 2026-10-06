# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/snv-local-observer/SPEC.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "Completed source changes receive attributable closure evidence at their actual gates.",
          "reference": "knowledge/sclv/SPEC.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "Canonical owner truth, implementation and checks have explicit indexed routes.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        }
      ],
      "distinctions": [],
      "evidence": [
        "modules/snv-local-observer/tests/observer_test.cpp",
        "modules/snv-local-observer/tests/fixtures/observe.json",
        "modules/snv-local-observer/tests/process.cpp"
      ],
      "feature_id": "ssfv:symphony:snv-local-observer",
      "how": "Fixed no-follow proc/sysfs reads, private boot consistency comparison, bounded capture/deadlines and independently checked receipt-bound process resources.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements the exact owner contract and its focused checks."
        },
        {
          "language": "CMake",
          "role": "Builds, installs and receipts the independently controlled package."
        }
      ],
      "implementation_paths": [
        "modules/snv-local-observer/src/main.cpp",
        "modules/snv-local-observer/src/observer.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "No physical identity or attestation, complete DIMM/device inventory, materiality decision, name/incarnation/membership allocation, retained evidence, selected head or operational authority.",
        "No scan, directory enumeration, configurable filesystem root, network provider, hostnames, machine IDs, raw boot IDs or permission escalation."
      ],
      "owner_contract": "modules/snv-local-observer/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Exact static foundation parsing, framing, digest, descriptor and receipt mechanics; no dependency on a running SNV owner.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/snv-local-observer",
      "status": "experimental",
      "title": "Selected local CPU and memory observation",
      "what": "Observe only caller-selected fixed Linux proc/sysfs CPU masks and memory fields with attributed source coverage, exact quantities and explicit partial/changed/unavailable states.",
      "when": "One explicit finite read-only capture; no background observation.",
      "where": "Independently installed optional collector module, separate from the five SNV semantic engines.",
      "who": "Direct native consumers and qxctl agents selecting the optional collector.",
      "why": "Provide bounded attributed local observations without allocating identity or converting availability into physical inventory."
    }
  ],
  "source_scope": "modules/snv-local-observer"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
