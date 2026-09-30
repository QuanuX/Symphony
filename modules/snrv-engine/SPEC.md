# SNRV engine v1

The exact `resources_validate` operation admits `symphony.snrv.resources-validate-input.v1` and returns `symphony.snrv.resources-validate.v1`. Installed strict schema and template belong to this owner. Domain meaning remains with `knowledge/snv/snrv/SPEC.md`.

## Evidence records

Input requires `protocol`, snapshot/transition `mode`, a selected 512/1024/2048 `record_limit`, `inventories` and nullable `transition`. Unknown fields and versions reject. The record capacity limits inventories and the aggregate local-resource/remote-attachment entries across supplied history and proposals. Per-source/reference arrays are independently bounded at 128. Text is exact supplied UTF-8, preserved without normalization; ASCII controls and invalid encodings reject. Source provenance follows the same source/method/time/artifact contract as SNIV, with source IDs local to the inventory and explicit references.

Every inventory has a supplied physical Node subject, record ID, positive generation, nullable causal predecessor, active/retired status, complete/partial/unknown coverage, nullable whole-second UTC observation time, sources, local resources, remote attachments and catalogue/model references. Explicit predecessor/generation establishes correction history; wall-clock time alone cannot supersede another source. A missing predecessor is insufficient. Independent active leaf assertions that contradict known resource facts remain conflicting; unknown fields do not become contradictory truth. `source_digest` is tagged SHA-256 over canonical `Json::dump()` of the exact input: sorted object keys, input array order, exact spelling and safe integers. It binds bytes/representation, not physical identity.

## Quantities and visibility

A local resource keeps its component kind, exact instance reference, `instance_scope` (physical/exposed/unknown), model reference, present/absent/unknown presence, `capacity_kind` (physical/exposed/unknown), exact capacity, separately available capacity, operational state and source references. Models and provider offerings remain opaque references; they never synthesize local hardware facts. Firmware/probe/provider/user evidence retain their declared kinds.

Quantities are canonical unsigned decimal strings from 0 through 18446744073709551615, with exact units bytes, count, hertz or bits_per_second. Floating point, leading zeroes, numeric JSON coercion and overflow reject. Available capacity compares exactly only when the same unit applies and cannot exceed declared capacity. An absent resource cannot claim available capacity. Unknown and unavailable differ from absence. Guest-exposed capacity or instance changes do not prove physical substrate replacement.

Remote attachments use structured allocation, optional subject/endpoint references, resource kind, exact capacity, attached/detached/unknown state and provenance. No `::` shorthand is parsed or allocated. A remote resource never becomes local composition by inference.

## Pure proposals and changes

Typed transition kinds are correction, retirement, restoration and resource_change. The caller supplies the exact predecessor and candidate inventory, new record ID and next generation. A physical subject change requires `resource_change` and a retained `identity_rebind` containing the exact supplied SNIV successor record reference and predecessor/successor Node IDs. The candidate inventory retains that binding so later snapshot replay preserves it. Same-subject correction preserves this identity lineage. The parent independently verifies the reference against the paired exact SNIV classification; SNRV does not invent a subject or decide materiality. Retirement and restoration explicitly change lifecycle state; other transitions retain it. Proposals remain pure supplied records until parent retention/selection.

The deterministic diff distinguishes added, removed, replaced, capacity_changed, availability_changed, unchanged and unknown local evidence. Missing entries in partial/unknown captures do not prove removal. A complete capture plus prior observed presence can support removal. Known physical instance/composition or physical-capacity changes produce an explicit `identity_handoff.required` for SNIV classification. Availability and guest-exposed changes do not create physical identity consequences. Remote differences are `attachment_changed`, always with `physical_change_evidence=false`. Change evidence retains both record references; SNRV never applies materiality rules or changes physical identity itself.

The result preserves owner/version, source binding, supplied subject IDs, attributed records, proposed records, findings, resource changes and the SNIV handoff. Conflicting or insufficient causal evidence produces no proposal. Pure import validation accepts useful partial inventories with visible coverage findings.

## Execution and installation

C++26 extensions are disabled. The owner checks the deadline before and during bounded processing, performs no filesystem/network/probe operation, and exposes one finite administrative process operation with strict framing inherited from the exact foundation. Separate owner installation and SDK export do not require a running SNIV, parent SNV, graph server, provider account, SHV/SCV installation or bus. Parent semantic replay embeds the exact admitted release. Resource allocation, compatibility selection and operational authority remain outside this reducer.
