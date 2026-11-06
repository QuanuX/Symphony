# SRRAPIV architecture and delivery goals

The conceptual architecture has three peer SQV connection classes: SOOV for FIX, SCABV for component-scoped broker/API integrations, and SRRAPIV for dedicated R API+ C++. The native implementation is sovereign over provider meaning; common indexing, installation, composition, authority and audit remain reusable explicit contracts.

The durable design goals are full typed qxctl administration, provider fidelity, exact immutable versions, visible failures, reproducible maintenance and open composition. A strategy can retain its selected SDK while another selects a different version. Compatibility includes API, ABI, data/state schemas, environment, account capability and provider support; pinning local artifacts cannot freeze a remote service.

A release is assessed against independent source inventories, observed native handlers, actual command routes and qualification evidence. Candidate installation, binding selection and provider activation are different actions. Historical readers and original state remain retained when migrations create derived records. Rollback requires a compatible reader plus reconciliation of external state.

The native module's SPEC is the operating contract. This architecture states the whole design target and does not turn unavailable SDK lanes, live environments, remote credential delivery, provider conformance or automatic account reconciliation into implemented capabilities.
