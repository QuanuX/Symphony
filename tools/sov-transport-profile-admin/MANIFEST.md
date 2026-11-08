# Protected transport profile manifest

- Component/module: `github.com/QuanuX/Symphony/tools/sov-transport-profile-admin`
- Version: `0.1.0-dev`; exact Go `1.26.5`.
- Owner: SPEC.md; subordinate SOV route: `knowledge/sov/transport-profiles/MANIFEST.md`.
- Production command: `sov-transport-profile-admin`; fixed protected versioned installation, no PATH lookup.
- Wire owner: `contract/`; private local store: `internal/store/`; authenticated audit consumer: `internal/auditclient/`.
- Registration: SSFV feature `ssfv:symphony:sov-transport-profile-admin`; invariant registry v5 explicitly represents this protected tool without pretending it is a native receipt-v2 adapter.
- Package and profile installation are separate. No host policy, provider or service is activated by a build.
