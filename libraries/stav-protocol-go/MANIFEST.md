# STAV Protocol Kernel Manifest

## Canonical Source

`knowledge/stav/`

## Go Namespace

- module: `github.com/QuanuX/Symphony/libraries/stav-protocol-go`
- current source release: `v0.2.0`
- package: `stavprotocol`
- binary: none
- runtime state: none

The version is published as an immutable public Go source module. It is not a binary release and grants no runtime authority.

## Consumers

- `modules/stav-append-authority/`
- `tools/qxctl/`
- `modules/secure-identity-access-governance/`

## Constraints

The kernel uses only the Go standard library, is authored entirely in Go, uses no cgo, and does not open a transport or write state. Canonical schemas and fixtures outrank its Go types and implementation.
