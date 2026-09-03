# Symphony Orchestra Omega Vector Specification

## Status

Architect-ratified identity, ownership, placement, and language boundary. Detailed FIX architecture remains deferred until the historic QuanuX work is examined with further Architect guidance.

## Protocol Boundary

FIX systems must be designed according to the applicable FIX standards, counterparty profiles, certification requirements, versions, and extensions. SOOV must not reinterpret FIX as a generic broker REST or data API.

FIX session behavior, data representation, symbol matching, message flow, order behavior, persistence, replay, recovery, timing, network design, and user extension points remain unresolved here.

## Performance Boundary

Performance-critical SOOV runtime surfaces will be native C++ and must disclose their exact hardware, operating-system, compiler, network, and dependency compatibility. No container, hidden runtime, Go data plane, or agentic inline dependency is inferred.

## Non-Authorization Statement

SOOV's existence does not authorize a broker connection, exchange access, strategy execution, order, message interpretation, or claim of FIX conformance.
