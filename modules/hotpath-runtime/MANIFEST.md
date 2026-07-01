# hotpath-runtime Manifest

## Module Identity
- **Name**: hotpath-runtime
- **Type**: Foundation module

## Runtime Doctrine
hotpath-runtime is not a troll; it is the native hot-path runtime substrate.

## Role
hotpath-runtime owns the native hot path.

## Scope
native runtime capability
low-latency local execution surfaces
performance-sensitive runtime loops
native C++ runtime foundation, where applicable
bounded telemetry or bridge outputs when explicitly declared
optional module/tool-declared high-performance runtime capability surfaces

## Non-Scope
node identity
node heartbeat
bus lifecycle
global coordination
Symphony-wide market-data doctrine
Symphony-wide order-flow doctrine
universal strategy plugin ABI
mandatory feed-provider SDK behavior
Python runtime dependency for hot-path execution

## Precision
hotpath-runtime may expose optional high-performance capability surfaces when declared by a module or tool, but those surfaces are not Symphony-wide doctrine, not universal ABI requirements, and not required for the native runtime substrate.

Databento, Rithmic, and similar SDKs remain optional external provider edges.

## Installability
- **module install status**: PROPOSED
- **install scope**: Native runtime substrate.
- **supported installation modes**: development, production, source-evidence.
- **required runtime assumptions**: Bare-metal-first capability.
- **optional capabilities**: None inherently required platform-wide.
- **non-required dependencies**: Kubernetes, containers, specific cloud providers.
- **prohibited platform-wide dependencies**: Python must not be required for remote native hot-path execution.
- **install verification expectations**: Native binaries run on bare-metal targets independently.
- **current ratification status**: DRAFT
