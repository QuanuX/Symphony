# SOV stdio adapter manifest

- Module: `github.com/QuanuX/Symphony/tools/sov-stdio-adapter`
- Release: `0.1.0-dev`, unpublished local source
- Go baseline: 1.26.5; standard library only
- Supported stock platforms: darwin, linux; locally qualified on darwin only
- Owner: `knowledge/sov/PROCESS-EXECUTION.md`
- Files: `main.go`, `main_test.go`, `go.mod`, `README.md`, `INTENT.md`, `SKILL.md`, `FEATURES.md`, `MANIFEST.md`
- Independent build/removal: `README.md`
- Runtime control: `qxctl sov remote invoke`; target configuration: `qxctl sov target configure|inspect|remove|invoke`

No engine receipt or shared installation is owned by this transport. Exact executable digests select source-built deployment artifacts. No language adapter is required by native SBV.
