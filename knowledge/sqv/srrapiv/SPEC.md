# Symphony Rithmic RAPI Vector Specification

## Ownership

SRRAPIV owns R API+ SDK identity, typed operation/callback interpretation, exact native bindings, Rithmic session lifecycle, provider-specific recovery and complete administration. SCABV does not own this dedicated integration; SOOV owns FIX semantics only. Other Rithmic products, Windows connectors and a generic cross-provider order model are outside this owner.

## Initial implementation

`modules/srrapiv-runtime-cpp/SPEC.md` declares the native 0.1.0-dev module and its exact implemented scope. The source supports a metadata/control build without proprietary SDK files and an explicit locally supplied 13.7.0.0 SDK build. A metadata build cannot execute vendor methods. Typed mapping, executable method dispatch, compiled SDK evidence and live acceptance are separately reported.

## Version selection

`BINDINGS.md` owns immutable artifact and strategy-binding records. `MAINTENANCE.md` owns exact compatibility evidence, directed reader compatibility and complete element-level coverage accounting. A newer package never rewrites an older binding. Separate persistent processes isolate selected SDKs; session limits and account effects remain shared provider conditions.

## Control and data

The finite knowledge engine process protocol carries qxctl administration. It does not carry a perpetual provider feed. An explicitly started native process owns its SDK and exposes same-uid local Unix-domain control. The runtime provides bounded callback observation with explicit generation, cursor and loss. Native strategy integration and optional research retention are independent compositions; no graph, central broker or qxctl process is implicitly inserted into a strategy's execution path.

## Authority and recovery

The first runtime profile is scoped to one local OS user and private directories. Explicit runtime and per-request permission selections gate provider and trading calls. This is not an implementation of remote grants or SSIAG credential delivery. Private configuration references and protected native credentials remain outside ordinary command arguments and returned evidence.

SDK acceptance does not prove provider acceptance. Attempt identity prevents duplicate local admission while durable uncertainty is retained across lost responses and process failure. Recovery reads attempts and provider observations; it never resubmits uncertain orders automatically. Software restart or rollback does not reverse orders or fills. Callers retain operation-specific responsibility for reconciliation and resumption policy.

## Coverage and evidence

Complete R API+ coverage remains the acceptance requirement, including overloads, callbacks, fields, lifecycle and process-local utilities. Exact generated descriptors report represented and unresolved elements instead of concealing omissions. A count of mapped methods alone is not full conformance. Live-provider and platform claims require their own exact evidence.
