// Package packageretirement fences only this native owner's participating uses.
// Its host-local state is not a portable package or proof about other consumers.
package packageretirement

type Identity struct {
	Owner           string `json:"owner"`
	Version         string `json:"version"`
	ReceiptDigest   string `json:"receipt_digest"`
	BinaryDigest    string `json:"binary_digest"`
	BinaryRelative  string `json:"binary_relative"`
	ReceiptRelative string `json:"receipt_relative"`
	BinarySize      uint64 `json:"binary_size"`
}

const Capability = "symphony.package-retirement.v1"
