# Symphony Knowledge Vector Common Schemas v1

## Authority

These exact JSON Schema files are canonical common process and lifecycle contract truth owned by the `knowledge/` umbrella. Implementations remain subordinate to them.

## Schemas

- `engine-process-request.schema.json`: one bounded local process request envelope.
- `engine-process-response.schema.json`: one bounded local process response envelope.
- `engine-descriptor.schema.json`: installed engine/coordinator identity and capability truth.
- `install-receipt.schema.json`: versioned, prefix-relative package ownership and docking state.

All schemas use JSON Schema Draft 2020-12, close every governed object with `additionalProperties: false`, and carry no secrets. Operation-specific payload/result schemas remain owned by the applicable engine Contract Quad.

## Boundary

These artifacts do not authorize canonical apply, network access, active-version selection, live Maestro docking, or any vector-specific semantic decision.
