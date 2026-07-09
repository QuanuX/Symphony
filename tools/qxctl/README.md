# qxctl

`qxctl` is the Go-based administrative spine for Symphony.

It provides a headless command-center for interacting with Symphony nodes, the bus, and the native hot path.

## Design
* Go-based
* Zero provider/cloud/Kubernetes assumptions
* Modular boundaries

## Go version posture

qxctl is seeded against Go 1.26.5 as the current scripted baseline.
qxctl intentionally uses the Go standard library only.
Symphony intends to migrate qxctl to Go 1.27 after Go 1.27 is released and available in the local toolchain.
Until that migration PR, qxctl must avoid Go 1.27-only language features and standard-library APIs.
