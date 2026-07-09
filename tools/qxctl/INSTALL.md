# qxctl Install

## Requirements
- Go 1.26.5 requirement
- No Python requirement
- No remote runtime requirement
- No provider/cloud/Docker/Kubernetes requirement

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
qxctl preserves a future migration posture for Go 1.27 after release/toolchain availability.
