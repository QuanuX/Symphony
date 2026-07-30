# Symphony Knowledge Vector Common Schemas v1

## Authority

These exact JSON Schema files are canonical common process and lifecycle contract truth owned by the `knowledge/` umbrella. Implementations remain subordinate to them.

## Schemas

- `engine-process-request.schema.json`: one bounded local process request envelope.
- `engine-process-response.schema.json`: one bounded local process response envelope.
- `engine-descriptor.schema.json`: installed engine/coordinator identity and capability truth.
- `install-receipt.schema.json`: versioned, prefix-relative package ownership and docking state.
- `engine-binding-registry.schema.json`: protected, noncanonical user-default selection of exact inactive-undocked engine and coordinator installations. A binding is not installation, Maestro docking, authentication, permission, or canonical apply authority.
- `proposal.schema.json`: provider-neutral immutable proposal envelope and vector-neutral authority boundary. Its explicit `engine_decided_domain_truth: false` assertion prevents any engine from converting validation into ownership, membership, ratification, publication, or other semantic authority.
- `provider-evidence.schema.json`: bounded provider-neutral revision, change-request, and ratification evidence normalized by separately discoverable adapters.

All schemas use JSON Schema Draft 2020-12, close every common-governed object with `additionalProperties: false`, and carry no secrets. The proposal operation's bounded `data` object is deliberately governed by the applicable vector schema; operation-specific payload/result schemas remain owned by that engine Contract Quad.

## Boundary

The binding schema authorizes only explicit user-scope selection among exact validated local installations. These artifacts do not authorize canonical apply, network access, system/TOPS binding changes, repository-specific overrides, live Maestro docking, or any vector-specific semantic decision.
