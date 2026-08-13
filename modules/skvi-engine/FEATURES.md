# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/skvi-engine/SPEC.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed engine and vector changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes engine contracts, implementation, and feature ownership.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future engine release.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "The engine interprets one vector read-only; the coordinator persists cross-session noncanonical administrative evidence and never decides vector truth.",
          "target_feature_id": "ssfv:symphony:knowledge-session-coordinator"
        }
      ],
      "evidence": [
        "modules/skvi-engine tests exercise the implemented operation set, deterministic results, bounds, invalid input, and disabled apply.",
        "The module CMake and receipt-v2 surfaces prove independent versioned installation and conservative uninstall."
      ],
      "feature_id": "ssfv:symphony:skvi-engine",
      "how": "The C++ process uses the shared bounded engine envelope, no-follow reads, canonical digests, deterministic ordering, strict schemas, explicit resource ceilings, and disabled canonical apply.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements bounded SKVI inspection, validation, evidence, proposal, and contract-defined noncanonical projection operations without canonical writes."
        },
        {
          "language": "CMake",
          "role": "Builds, tests, installs, receipts, and uninstalls the exact inactive-undocked engine package."
        }
      ],
      "implementation_paths": [
        "modules/skvi-engine/CMakeLists.txt",
        "modules/skvi-engine/src/main.cpp",
        "modules/skvi-engine/src/skvi.cpp",
        "modules/skvi-engine/tests/process_smoke.sh",
        "modules/skvi-engine/tests/skvi_test.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "Does not mutate canonical knowledge, decide semantic membership or ratification, start a listener, or infer permission from caller type.",
        "Does not activate or dock itself, publish artifacts, or claim an overall production release."
      ],
      "owner_contract": "modules/skvi-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "The engine statically links the common bounded process, digest, path, temporal, and snapshot mechanics.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/skvi-engine",
      "status": "experimental",
      "title": "SKVI independently installed knowledge engine",
      "what": "Provides the independently installable freezing-path engine that reads canonical SKVI truth and emits deterministic bounded evidence, proposals, and disposable projections under that vector's contract.",
      "when": "Runs only on explicit qxctl or exact local process invocation under a bounded deadline and against an exact repository snapshot.",
      "where": "Executes from an inactive-undocked versioned installation and reads repository truth owned by knowledge/skvi/ without a network listener.",
      "who": "Any host-authorized caller using qxctl, vector maintainers, reviewers, integration tooling, and tests.",
      "why": "Gives SKVI application-owned programmatic behavior without transferring semantic, ratification, mutation, or publication authority to tooling."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed changes to SKVI proposal behavior and its evidence boundary.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI owns the canonical index semantics and schemas interpreted by this proposal operation.",
          "reference": "knowledge/skvi/SPEC.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future publication of the engine implementing this proposal capability.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "Prepares one bounded caller-declared index change without canonical content or apply; structural assurance only reports the current index's integrity.",
          "target_feature_id": "ssfv:symphony:skvi-engine.structural-index-assurance"
        }
      ],
      "evidence": [
        "modules/skvi-engine/tests/skvi_test.cpp verifies deterministic add proposals, stale replace rejection, required entry evidence, proposal digests, false ratification, and disabled canonical apply.",
        "modules/skvi-engine/SPEC.md defines the exact add, replace, remove, expected-state, read-set, and prospective-write-set contract."
      ],
      "feature_id": "ssfv:symphony:skvi-engine.content-addressed-index-change-proposals",
      "how": "Requires a structurally clean current index, validates one caller-selected add, replace, or remove operation against exact entry and index state, binds repository and contract reads plus the prospective index write, and emits a deterministic immutable proposal that explicitly denies engine-decided domain truth and canonical apply.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements strict proposal payload validation, stale-state refusal, content-addressed read and prospective write sets, and deterministic proposal evidence."
        }
      ],
      "implementation_paths": [
        "modules/skvi-engine/src/skvi.cpp",
        "modules/skvi-engine/src/skvi.hpp",
        "modules/skvi-engine/tests/skvi_test.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not decide whether an entry is architecturally true, render or apply canonical index content, ratify a change, authenticate a caller, or grant permission.",
        "qxctl mediates exact installed-process selection and response validation; it does not own SKVI truth, proposal semantics, or canonical mutation."
      ],
      "owner_contract": "modules/skvi-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:skvi-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl selects and revalidates one exact installed SKVI engine before invoking this operation; that mediation grants neither tool semantic authority nor an apply path.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings",
          "type": "composes_with"
        },
        {
          "rationale": "Proposal parsing, deadlines, stable errors, and deterministic responses use the shared bounded process contract.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation.bounded-process-protocol",
          "type": "depends_on"
        },
        {
          "rationale": "Proposal read sets and current index evidence use the shared no-follow content-addressed snapshot mechanics.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation.content-addressed-evidence-snapshots",
          "type": "depends_on"
        },
        {
          "rationale": "A proposal is produced only after the current canonical index passes the engine's bounded structural check.",
          "target_feature_id": "ssfv:symphony:skvi-engine.structural-index-assurance",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/skvi-engine",
      "status": "experimental",
      "title": "Content-addressed SKVI change proposals",
      "what": "Produces one immutable evidence proposal for a caller-declared SKVI entry addition, replacement, or removal against exact current index state.",
      "when": "On an explicit `qxctl skvi propose` or exact local process invocation after the caller supplies one bounded operation, repository identity, revision, and strict UTC proposal times.",
      "where": "Inside one exact inactive-undocked SKVI engine installation, reading canonical knowledge/skvi/ truth and returning evidence through the bounded local process protocol.",
      "who": "Any permission-backed caller using qxctl or the exact process, SKVI maintainers, reviewers, integration tooling, and tests.",
      "why": "Makes prospective index maintenance deterministic and stale-state-safe while preserving ordinary reviewed source change as the only canonical mutation path."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed changes to the SKVI projection operation.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI owns the canonical index semantics from which this noncanonical projection is derived.",
          "reference": "knowledge/skvi/SPEC.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future publication of the engine or derived projection format.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "Normalizes a clean index for disposable consumption; content-addressed change proposals describe one prospective caller-declared edit.",
          "target_feature_id": "ssfv:symphony:skvi-engine.content-addressed-index-change-proposals"
        }
      ],
      "evidence": [
        "modules/skvi-engine/tests/skvi_test.cpp verifies deterministic JSON projections, noncanonical and rebuildable markers, entry and projection digests, clean-index gating, format refusal, and maximum-size bounds.",
        "knowledge/skvi/schemas/v1/projection.schema.json defines the portable projection result consumed through the engine boundary."
      ],
      "feature_id": "ssfv:symphony:skvi-engine.disposable-structural-projection",
      "how": "Requires a clean structural check, normalizes every bounded index entry into deterministic order, binds each entry and the whole result to content and contract digests, and returns the single supported portable JSON format marked noncanonical and rebuildable.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements clean-index gating, normalized entry projection, deterministic ordering, per-entry and aggregate digests, and response-bound enforcement."
        }
      ],
      "implementation_paths": [
        "modules/skvi-engine/src/skvi.cpp",
        "modules/skvi-engine/src/skvi.hpp",
        "modules/skvi-engine/tests/process_smoke.sh",
        "modules/skvi-engine/tests/skvi_test.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not provide JSONL, a canonical index, persistent graph, external database, filesystem artifact, watcher, background refresh, or publication.",
        "qxctl administers exact process invocation and validates the returned envelope; it does not own the projection's SKVI semantics or convert it into canonical truth."
      ],
      "owner_contract": "modules/skvi-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:skvi-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl revalidates one exact installed SKVI engine and invokes the project operation without absorbing projection or vector authority.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings",
          "type": "composes_with"
        },
        {
          "rationale": "Projection parsing, limits, deadlines, errors, and response binding use the shared bounded process contract.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation.bounded-process-protocol",
          "type": "depends_on"
        },
        {
          "rationale": "Projection source evidence uses the shared no-follow content-addressed snapshot mechanics.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation.content-addressed-evidence-snapshots",
          "type": "depends_on"
        },
        {
          "rationale": "Projection is available only when the canonical index passes bounded structural assurance.",
          "target_feature_id": "ssfv:symphony:skvi-engine.structural-index-assurance",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/skvi-engine",
      "status": "experimental",
      "title": "Disposable SKVI structural projection",
      "what": "Emits a portable content-addressed JSON view of normalized canonical SKVI entries for bounded downstream inspection and integration.",
      "when": "On an explicit `qxctl skvi project` or exact local process invocation after the live index has passed the same invocation's structural analysis.",
      "where": "Returned in-process from one exact inactive-undocked SKVI engine installation and never written by the engine.",
      "who": "Any permission-backed caller using qxctl or the exact process, reviewers, integration tooling, tests, and consumers needing a rebuildable index view.",
      "why": "Lets downstream tools consume normalized routing structure without parsing Markdown themselves or promoting a derived representation into source truth."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed changes to SKVI structural validation behavior.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI owns the canonical index structure, coverage, relationship, and routing semantics checked here.",
          "reference": "knowledge/skvi/SPEC.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future publication of the engine implementing this assurance operation.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "Reports deterministic structural integrity of current canonical routing; a change proposal describes a prospective caller-selected index operation without repair.",
          "target_feature_id": "ssfv:symphony:skvi-engine.content-addressed-index-change-proposals"
        },
        {
          "distinction": "This is bounded vector-owned SKVI assurance; Symphony Validator independently checks broader repository conformance across vectors and module contracts.",
          "target_feature_id": "ssfv:symphony:symphony-validator"
        }
      ],
      "evidence": [
        "modules/skvi-engine/tests/skvi_test.cpp verifies the live index, expected-digest match and mismatch, duplicate paths, symlink refusal, relationship and count bounds, and invalid-state evidence.",
        "modules/skvi-engine/tests/process_smoke.sh exercises the installed-style bounded check process against repository truth."
      ],
      "feature_id": "ssfv:symphony:skvi-engine.structural-index-assurance",
      "how": "Parses the canonical Markdown index under fixed fields and count bounds, verifies required metadata and canonical status, rejects duplicate or unsafe paths, opens indexed targets as no-follow regular files, checks required umbrella and vector coverage plus relationship targets, and compares an optional expected index digest.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements bounded index parsing, path and target verification, coverage and relationship checks, deterministic findings, and expected-state comparison."
        }
      ],
      "implementation_paths": [
        "modules/skvi-engine/src/skvi.cpp",
        "modules/skvi-engine/src/skvi.hpp",
        "modules/skvi-engine/tests/process_smoke.sh",
        "modules/skvi-engine/tests/skvi_test.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not interpret architectural meaning, decide whether a routed surface should exist, repair the index, edit an indexed target, or make findings authorization evidence.",
        "qxctl selects and supervises the exact local invocation only; it does not own SKVI structure, finding semantics, or canonical repair authority."
      ],
      "owner_contract": "modules/skvi-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:skvi-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl resolves and revalidates one exact installed SKVI engine before invoking its structural check, without becoming the source of index truth.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings",
          "type": "composes_with"
        },
        {
          "rationale": "Index request parsing, deadlines, stable findings responses, and digest framing use the shared bounded process contract.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation.bounded-process-protocol",
          "type": "depends_on"
        },
        {
          "rationale": "Safe indexed-target acquisition and index evidence use the shared no-follow content-addressed snapshot mechanics.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation.content-addressed-evidence-snapshots",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/skvi-engine",
      "status": "experimental",
      "title": "Canonical index structural assurance",
      "what": "Produces deterministic bounded evidence that the current canonical SKVI index is structurally valid, safely routed, relationship-closed, and tied to an optional expected digest.",
      "when": "On explicit `qxctl skvi check`, exact local check invocation, and internally before proposal or projection operations that require a clean index.",
      "where": "Inside one exact inactive-undocked SKVI engine installation reading knowledge/skvi/INDEX.md and its repository-relative targets without a listener.",
      "who": "Any permission-backed caller using qxctl or the exact process, SKVI maintainers, reviewers, release gates, integration tooling, and tests.",
      "why": "Canonical routing must fail visibly on ambiguity, missing coverage, broken targets, or stale expected state before downstream evidence is trusted."
    }
  ],
  "source_scope": "modules/skvi-engine"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
