# qxctl Intent

qxctl is the Go-based local administrative spine for Symphony.

## Purpose
qxctl is a repository and module inspection/control surface. It is a deterministic local status/inventory/digest tool designed to read and report Symphony repository state.

## Scope
qxctl operates as a local utility to verify modules, aggregate contracts, and digest runtime inventory. It uses Go 1.26.5 as the current scripted baseline and preserves a future migration posture for Go 1.27 after release/toolchain availability.

## Non-goals
- qxctl does not execute hotpath-runtime workloads.
- qxctl does not make bus traversal mandatory.
- qxctl does not require Python.
- qxctl does not perform remote execution.
- qxctl does not manage NATS directly.
- qxctl does not own hotpath-runtime execution.
- qxctl does not replace node-troll.
- qxctl does not replace bus-troll.
- qxctl does not replace hotpath-runtime.
- qxctl does not choose infrastructure.
- qxctl does not assume Docker/Kubernetes/cloud.
- qxctl does not assume trading, market-data, strategy, provider, or plugin ABI behavior.
- qxctl does not write generated SKVI/SCLV records.
- qxctl does not enforce runtime behavior.

## Relationship
qxctl reads and reports Symphony repository state. It relates to node-troll, bus-troll, and hotpath-runtime as an administrative inspector, not as an active participant in their workloads.
