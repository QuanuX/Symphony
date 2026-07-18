# Symphony Secure Identity and Access Governance Manifest

## Module Identity

- **name**: `secure-identity-access-governance`
- **path**: `modules/secure-identity-access-governance/`
- **binary**: `symphony-ssiag`
- **schema prefix**: `symphony.ssiag.*`
- **environment prefix**: `SYMPHONY_SSIAG_*`
- **language/runtime**: Go 1.26.5 with pinned `golang.org/x/sys` for cgo-free kernel peer credentials
- **cgo**: prohibited
- **status**: DRAFT foundation; supervised metadata-only runtime

## Canonical Authority

`knowledge/ssiag/` owns vocabulary, relationships, configuration extensions, provider protocol, compatibility, and authority boundaries. This module implements that truth. qxctl administers and queries it but owns neither schema nor runtime state.

## Contract Files

- `INTENT.md`, `MANIFEST.md`, `INSTALL.md`, `SKILL.md`, `SPEC.md`
- `ARCHITECTURE.md`, `REQUIREMENTS.md`, `IMPLEMENTATION.md`, `THREAT-MODEL.md`

## Implemented Surfaces

- `install` / `uninstall`: one host binary and digest-bearing install manifest;
- `enroll` / `unenroll`: isolated per-TOPS configuration and state;
- `serve`: one metadata-only Unix-socket API for one TOPS;
- `supervisor install` / `supervisor uninstall`: per-TOPS launchd or systemd liveness profile with conservative state preservation;
- Darwin/Linux kernel peer authentication on every accepted API connection;
- exact per-TOPS UID/GID-to-canonical-subject resolution for future subject-gated operations;
- stable per-TOPS service identity, pre-listen process verification, and client-side exact endpoint verification;
- owner-provisioned system identity validation, distinct service-owned state/runtime children, bounded restart/shutdown, and serialized stale-socket recovery;
- `status` / `providers`: safe local inspection;
- `qxctl ssiag status|providers|doctor`: provider-neutral query interface.

## Install and Enrollment Separation

Host uninstall never deletes per-TOPS state. `unenroll` preserves state by default; `unenroll --purge` targets exactly one validated TOPS UUID. Display names are configuration metadata and never path components.

## Provider Boundary

No operational provider is enabled. Native dependencies remain in independently installed adapters. The first adapter scaffold is `modules/ssiag-provider-macos-keychain/`, a separate Swift executable whose current capability is metadata only.

## Contamination Boundary

Secret values, proofs, assertions, raw tokens, provider payloads, policy contents, and native errors must not cross into qxctl, SKV, SKVI, SCLV, SODV, STAV, manifests, inventories, logs, or status responses.
