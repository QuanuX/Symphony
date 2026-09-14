package shvtransfer

import _ "embed"

//go:embed transfer.schema.json
var schema []byte

// Schema returns this CLI's exact orchestration contract; it is not a native
// connector receipt-owned resource.
func Schema() []byte { return append([]byte{}, schema...) }
