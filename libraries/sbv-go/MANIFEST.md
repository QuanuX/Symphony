# SBV Go adapter manifest

- Module: `github.com/QuanuX/Symphony/libraries/sbv-go`
- Source release: `v0.1.2-dev` (local; unpublished)
- Package: `sbv`
- Native compatibility: `sbv-engine` `0.28.0-dev` through selected qxctl
- Dependencies: Go 1.26.5 standard library; no cgo, Python or vendor runtime
- Canonical owner: `knowledge/sbv/SPEC.md`
- Implementation contract: `libraries/sbv-go/IMPLEMENTATION.md`
- Independent source install/remove: `libraries/sbv-go/README.md`
- Runtime state and native install ownership: none

Sources: `client.go`, `stream.go`, `rows.go`. Focused tests: `client_test.go`,
`stream_test.go`. Independent consumer example: `examples/roundtrip/main.go`.
Owner companions: `INTENT.md`, `README.md`, `IMPLEMENTATION.md`, `FEATURES.md`,
`SKILL.md`, `MANIFEST.md`, `go.mod`.

qxctl remains the administration spine and native receipt verifier. The adapter
has no separately installed executable or mutable service to register. Consumer
binary compilation and module selection remain explicit Go dependency choices;
the example is a caller composition, not a second SBV control plane.
