# STAV Protocol Kernel Requirements

## Functional

1. Reject invalid UTF-8, unpaired surrogate escapes, Unicode noncharacters, duplicate keys, unknown members, case-mismatched members, trailing values, and `null`.
2. Accept only non-negative JSON integers no greater than `9007199254740991`; reject floats, exponents, negative values, and negative zero.
3. Produce RFC 8785 member ordering and string escaping for the ratified STAV scalar profile.
4. Require canonical wire bytes for typed decode operations.
5. Validate every required event group and every tagged-union presence rule.
6. Bound candidates, events, requests, and responses before allocation or emission.
7. Keep candidate, event, and genesis digest domains distinct.
8. Implement exact configuration, status, local request, and local response unions without opening a transport.
9. Reconstruct the candidate portion of a canonical event for idempotency recovery without changing bytes.

## Security

- Standard library only; pure Go; no cgo.
- No network or filesystem side effects.
- No free-form native error or secret-bearing field.
- Unknown closed-enum values fail closed; producer-specific registered identifiers remain subject to runtime grant/producer contracts.
- The protocol kernel never makes an authorization decision.

## Compatibility

Go 1.26.5 is the ratified production baseline. Go 1.27 adoption requires byte-identical corpus results and cannot change the public protocol API or wire representation.
