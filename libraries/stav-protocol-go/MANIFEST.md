# STAV Protocol Kernel Manifest

## Canonical Source

`knowledge/stav/`

## Go Namespace

- module: `github.com/QuanuX/Symphony/libraries/stav-protocol-go`
- package: `stavprotocol`
- binary: none
- runtime state: none

## Consumers

- `modules/stav-append-authority/`
- `tools/qxctl/`
- `modules/secure-identity-access-governance/`

## Constraints

The kernel uses only the Go standard library, is authored entirely in Go, uses no cgo, and does not open a transport or write state. Canonical schemas and fixtures outrank its Go types and implementation.
