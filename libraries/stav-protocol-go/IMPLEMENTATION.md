# STAV Protocol Kernel Implementation

## Layers

1. A strict recursive parser validates UTF-8/string scalars, rejects duplicate names and unsafe numeric forms, and builds a closed JSON value tree.
2. The canonical encoder sorts object names by UTF-16 code units and applies the RFC 8785 string form.
3. Typed decode first requires canonical bytes, then verifies exact case-sensitive member shape before standard-library binding.
4. Domain validation applies schema identifiers, tagged unions, closed registries, limits, and cross-field constraints.
5. Digest and framing helpers operate only on already validated data or caller-provided streams.

No layer opens a socket or touches runtime state. Transport authentication and runtime mutation remain append-authority gates.
