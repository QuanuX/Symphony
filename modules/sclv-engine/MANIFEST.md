# SCLV Engine Manifest

## Identity

- module: `sclv-engine`
- engine: `symphony-sclv`
- adapters: `symphony-sclv-evidence-local-git`, `symphony-sclv-evidence-airgap`
- vector: `sclv`
- version: `0.1.0-dev`
- language: C++26
- thermal path: freezing

## Package Files

The package owns three exact versioned executables, one inactive-undocked receipt, five contract documents, the repository AGPL license, and the pinned nlohmann/json MIT license. It creates no active alias, hook, journal, canonical record, or Maestro state.

## Read and Proposal Boundaries

The engine reads `knowledge/sclv/CHANGELOG.md`, the SCLV Contract Quad, `RECOVERY.md`, the v3 schema/template surfaces, and record-referenced regular files. Its prospective write set may name only `knowledge/sclv/CHANGELOG.md`, and only inside a noncanonical proposal.

The local-Git adapter invokes `/usr/bin/git` directly with fixed arguments and a sanitized environment. The air-gapped adapter validates and digests caller-declared safe metadata. Neither adapter writes provider or repository state.

## Installability

The exact version installs under a caller-selected prefix as inactive `installed_undocked`. qxctl currently requires that prefix and version. Uninstall removes only the eleven receipt-owned files and never removes containing directories or canonical knowledge.
