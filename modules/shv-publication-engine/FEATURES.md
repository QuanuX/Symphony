# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/shv-publication-engine/SPEC.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SKVI routes current owner contracts and distributed feature files.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "Completed material source changes require reviewed append-only closure.",
          "reference": "knowledge/sclv/SPEC.md",
          "vector": "sclv"
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
        "SHV-18 focused native, consumer, journal, installed authority and interrupted-process evidence is recorded in the increment packet."
      ],
      "feature_id": "ssfv:symphony:shv-publication-engine",
      "how": "Pure C++ publication owner, independent Go verifier and protected qxctl journal. Explicit publisher 0.2 admission for store writers 0.1, 0.2 and 0.3; original publisher 0.1 remains strict.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Owns catalogue definition, revision and history semantics."
        },
        {
          "language": "CMake",
          "role": "Builds and receipts the exact independently selected connector and dependency."
        }
      ],
      "implementation_paths": [
        "modules/shv-publication-engine/CMakeLists.txt",
        "modules/shv-publication-engine/src/publication.cpp",
        "modules/shv-publication-engine/src/descriptor.cpp",
        "modules/shv-publication-engine/src/main.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "No universal hardware universe, compatibility decision, default graph vendor or Symphony-wide CanonicalApply.",
        "Native semantic validation alone does not grant publication authority."
      ],
      "owner_contract": "modules/shv-publication-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Authority-free process, path and digest mechanics.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/shv-publication-engine",
      "status": "experimental",
      "title": "SHV protected catalogue publication",
      "what": "Derives caller-selected catalogue heads and expected-state transitions.",
      "when": "Explicit qxctl plan, apply, recovery and status operations.",
      "where": "Independent receipt-owned installation and caller-scoped private catalogue journal.",
      "who": "Agents and humans invoking the exact installed SHV through qxctl.",
      "why": "Preserve caller completeness policy while requiring original evidence and distinct publication permission."
    }
  ],
  "source_scope": "modules/shv-publication-engine"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->

## Explicit composition release 0.4.0-dev

Publication 0.4.0-dev explicitly embeds partition 0.4, admits selected partition 0.2/0.3/0.4, source 0.1/0.2, kernel 0.1/0.2/0.3/0.4 and storage writers 0.1/0.2/0.3/0.4. A partition selected as 0.2 or 0.3 cannot carry source 0.2 or kernel 0.4 dependencies. Historical publishers retain their prior admissions. Retained state transitions replay against each original publisher; aggregate history validation admits the supported union without changing those owners.

Use a fresh install prefix and explicit qxctl version selection. Version 0.3 declarations remain checked against their original compiled digest. No command or default changes; metadata is not semantic authorization and future versions are not automatically admitted.
