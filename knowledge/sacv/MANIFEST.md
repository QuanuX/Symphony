# Symphony API Contract Vector Manifest

## Canonical Target

`knowledge/sacv/`

## Identity

SACV is the Symphony API Contract Vector.

## Classification

- autonomous Symphony Knowledge Vector contract surface;
- canonical API-contract governance and registry authority;
- OpenAPI 3.2.0 profile authority;
- not a global endpoint owner, runtime gateway, documentation site, or SDK generator.

## Declared Contract Truth Role

SACV owns:

- API-first doctrine for HTTP surfaces;
- the accepted OpenAPI feature/version profile;
- ownership and placement rules;
- compatibility and versioning policy;
- registry entry requirements;
- validation and derivation boundaries;
- publication-eligibility metadata;
- security and safe-example constraints.

The domain-owning knowledge vector or module owns endpoint paths, operations, semantics, data shapes, and ratified security-profile references. SACV MUST NOT centralize or duplicate those contracts merely for tooling convenience.

## Contract Placement

Canonical API descriptions use one of these ownership patterns:

```text
knowledge/<domain-vector>/api/<api-id>-v<major>.openapi.yaml
modules/<module>/api/<api-id>-v<major>.openapi.yaml
knowledge/sacv/apis/<api-id>-v<major>.openapi.yaml
```

The third pattern is reserved for an API whose cross-module composition is itself owned by SACV. A Mintlify bundle or SDK-generator input assembled from multiple owner documents is derived and MUST NOT be placed in `knowledge/sacv/apis/` as canonical truth.

## Relationship to the Contract Quad

An OpenAPI document is a conditional typed contract artifact referenced by the owning Contract Quad. It is not a mandatory fifth Contract Quad file, and modules without HTTP interfaces do not create one.

## Relationship to SKVI, SODV, and SCLV

- SKVI indexes SACV and every canonical API entry document.
- SODV decides whether and how an API may be published.
- SCLV records reviewed changes after actual PR and merge evidence exists.
- Mintlify, SDK generators, validators, and qxctl implement or consume contracts but own no canonical API truth.

## Installability Considerations

SACV has an independently installable C++ proposal engine at `modules/sacv-engine/` with executable `symphony-sacv`. Its initial operations are inspect, check, diff, propose, and project under `knowledge/SPEC.md`. It may install as `installed_undocked`; no endpoint owner depends on the engine being resident.

## Non-Authorization Statement

No canonical apply, endpoint document, listener, generated runtime code, public documentation, Mintlify configuration, SDK release, MCP tool, remote gateway, or live request playground is authorized by this manifest alone. Engine output is evidence, proposal, or disposable projection.

## Status

Owner-ratified governance vector with proposal-engine architecture authorized. No API entry document or engine implementation exists yet.
