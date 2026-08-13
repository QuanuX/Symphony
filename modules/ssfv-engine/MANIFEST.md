# SSFV Engine Manifest

## Identity

- Module: `ssfv-engine`
- Executable: `symphony-ssfv`
- Version: `0.1.0-dev`
- Language: C++26
- Thermal path: freezing
- State after installation: inactive and undocked

## Contract

The engine implements `knowledge/ssfv/` and the common knowledge-engine process/proposal contracts. Canonical truth remains in the SSFV contracts, namespace and feature registries, and registered distributed owner files.

## Operations

`inspect`, `check`, `diff`, `propose`, `graph`, and read-only `administration-check` are implemented. `apply`, feature bootstrap, listener, session mutation, activation, and docking operations are disabled. Descriptor v2 publishes stable engine-operation IDs from the same registry used for dispatch and an exact 65,536-value process bound for complete administration envelopes, while descriptor v1 remains byte-for-byte compatible.

## Installation

The exact `0.1.0-dev` receipt owns one versioned executable, one receipt, five contract documents, and two license files. No unversioned alias, service, socket, hook, active receptor, or default docking state is installed.

## Boundaries

The engine has no network access, writes no repository or state file, decides no semantic claim, classifies no caller, and cannot grant permission or ratification. Its graph, snapshots, coverage findings, module-integration readiness, and remediation constraints are noncanonical evidence. It never invents feature/command identities or grammar.
