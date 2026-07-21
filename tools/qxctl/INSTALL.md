# qxctl Install

## Requirements
- Go 1.26.5 requirement
- No Python requirement
- No remote runtime requirement
- No provider/cloud/Docker/Kubernetes requirement

qxctl uses the Architect-ratified Cobra command framework and a constrained Viper mapper, plus their pinned cgo-free Go dependencies. Its STAV grammar uses Symphony's first-party pure-Go protocol kernel. The module remains independently buildable with `GOWORK=off`; it does not require Python, C bindings, a remote configuration backend, or a configuration daemon.

The direct CLI dependency pins are Cobra `v1.10.2` and Viper `v1.21.0`. `go.mod` and `go.sum` are the authoritative dependency lock surfaces.

Viper is not an SSIAG/STAV trust loader. qxctl creates private instances, binds only declared command keys and environment variables, and does not enable automatic environment discovery, configuration-file discovery, remote providers, watch/reload, or write-back.

## Build and Run
qxctl can be run locally using the Go toolchain:
```bash
go run ./cmd/qxctl --help
go run ./cmd/qxctl status
```
Or built locally:
```bash
go build -o qxctl ./cmd/qxctl
```

The SKVI and SCLV command groups additionally require separately installed `skvi-engine` and `sclv-engine` packages. Pass the applicable exact prefix and version on every current invocation; qxctl validates the complete versioned receipt and owned files before execution. Secure local engine installation access is implemented on Linux and the macOS development path and fails closed on other native operating systems. No lifecycle default or active-version selector is implemented yet.

## Migration Note
qxctl targets Go 1.27 only after general availability and the differential fixture/digest, default-vs-`nojsonv2`, vet, race, and supported-platform cross-build gates pass. The workspace and module pins change atomically, and the migration cannot alter qxctl grammar or STAV bytes.
