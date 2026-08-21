package protocol

import (
	"encoding/json"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
)

const (
	SchemaSubmission = "symphony.accordare.stav-producer.submission.v1"
	SchemaRequest    = "symphony.accordare.stav-producer.local-request.v1"
	SchemaResponse   = "symphony.accordare.stav-producer.local-response.v1"
	SchemaStatus     = "symphony.accordare.stav-producer.status.v1"

	OperationSubmit    = "submit"
	OperationPrepare   = "prepare"
	OperationComplete  = "complete"
	OperationStatus    = "status"
	OperationReconcile = "reconcile"
)

type InstallationEvidence struct {
	ComponentID      string `json:"component_id"`
	ExecutableDigest string `json:"executable_digest"`
	ReceiptDigest    string `json:"receipt_digest"`
	Version          string `json:"version"`
}

type Submission struct {
	Command     json.RawMessage      `json:"command"`
	Coordinator InstallationEvidence `json:"coordinator"`
	Operation   string               `json:"operation"`
	Outcome     *string              `json:"outcome"`
	ReasonCode  *string              `json:"reason_code"`
	Result      json.RawMessage      `json:"result"`
	Schema      string               `json:"schema"`
	TOPSID      string               `json:"tops_id"`
}

type LocalRequest struct {
	Operation  string      `json:"operation"`
	RequestID  string      `json:"request_id"`
	Schema     string      `json:"schema"`
	Submission *Submission `json:"submission"`
	TOPSID     string      `json:"tops_id"`
}

type Status struct {
	Pending              uint64 `json:"pending"`
	IntentPending        uint64 `json:"intent_pending"`
	AppendPending        uint64 `json:"append_pending"`
	Ready                bool   `json:"ready"`
	ReconciliationNeeded bool   `json:"reconciliation_needed"`
	Schema               string `json:"schema"`
	TOPSID               string `json:"tops_id"`
}

type LocalResponse struct {
	CandidateDigest string                `json:"candidate_digest,omitempty"`
	IntentID        string                `json:"intent_id,omitempty"`
	Disposition     string                `json:"disposition"`
	Operation       string                `json:"operation"`
	ReasonCode      string                `json:"reason_code"`
	Receipt         *stavprotocol.Receipt `json:"receipt"`
	RequestID       string                `json:"request_id"`
	Schema          string                `json:"schema"`
	Status          *Status               `json:"status"`
	TOPSID          string                `json:"tops_id"`
}
