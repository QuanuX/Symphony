# hotpath-runtime Install

## Install Status
PROPOSED

## Install Scope
Native runtime substrate.

## Supported Installation Modes
- **development**: Source-based build with test stubs.
- **production**: Bare-metal optimized binary deployment.
- **source-evidence / proposal-only state**: Current state. Contract seeds only.

## Precision
hotpath-runtime may expose optional high-performance capability surfaces when declared by a module or tool, but those surfaces are not Symphony-wide doctrine, not universal ABI requirements, and not required for runtime core.

Databento, Rithmic, and similar SDKs remain optional external provider edges.

## Explicit Non-Requirements
- Kubernetes, containers, specific cloud providers.
- Python must not be required for remote native hot-path execution. Optional isolated Python habitats may exist only when explicitly declared by a module or tool.

## Current Limitations
Proposal-only state. No implementation code or installer scripts exist.

## Ratification Status
DRAFT
