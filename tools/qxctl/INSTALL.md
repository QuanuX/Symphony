# qxctl Install

## Requirements
- Go 1.26.5 requirement
- No Python requirement
- No remote runtime requirement
- No provider/cloud/Docker/Kubernetes requirement

qxctl has no third-party dependency. Its STAV grammar uses Symphony's first-party pure-Go protocol kernel. The monorepo `go.work` resolves the unreleased kernel during development; before qxctl is published for independent source installation, the kernel must be tagged and qxctl must require that real compatible version.

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

## Migration Note
qxctl targets Go 1.27 only after general availability and the differential fixture/digest, default-vs-`nojsonv2`, vet, race, and supported-platform cross-build gates pass. The workspace and module pins change atomically, and the migration cannot alter qxctl grammar or STAV bytes.
