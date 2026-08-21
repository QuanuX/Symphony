package protocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
)

func EncodeRequest(request LocalRequest) ([]byte, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	data, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	return canonicalJSON(data)
}

func DecodeRequest(data []byte) (LocalRequest, error) {
	canonical, err := canonicalJSON(data)
	if err != nil {
		return LocalRequest{}, err
	}
	var request LocalRequest
	if err := decodeStrict(canonical, &request); err != nil {
		return LocalRequest{}, err
	}
	return request, request.Validate()
}

func EncodeResponse(response LocalResponse) ([]byte, error) {
	if err := response.Validate(); err != nil {
		return nil, err
	}
	data, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}
	return canonicalJSON(data)
}

func DecodeResponse(data []byte) (LocalResponse, error) {
	canonical, err := canonicalJSON(data)
	if err != nil {
		return LocalResponse{}, err
	}
	var response LocalResponse
	if err := decodeStrict(canonical, &response); err != nil {
		return LocalResponse{}, err
	}
	return response, response.Validate()
}

func (request LocalRequest) Validate() error {
	if request.Schema != SchemaRequest || (request.Operation != OperationPrepare && request.Operation != OperationComplete && request.Operation != OperationSubmit && request.Operation != OperationStatus && request.Operation != OperationReconcile) {
		return fmt.Errorf("invalid Accordare producer request identity")
	}
	if err := stavprotocol.ValidateTOPSID(request.TOPSID); err != nil {
		return err
	}
	if err := stavprotocol.ValidateRequestUUID(request.RequestID); err != nil {
		return err
	}
	needsSubmission := request.Operation == OperationPrepare || request.Operation == OperationComplete || request.Operation == OperationSubmit
	if needsSubmission != (request.Submission != nil) {
		return fmt.Errorf("submission presence does not match operation")
	}
	if request.Submission != nil && (request.Submission.Schema != SchemaSubmission || request.Submission.TOPSID != request.TOPSID) {
		return fmt.Errorf("submission binding mismatch")
	}
	if request.Operation == OperationPrepare && (request.Submission.Outcome != nil || request.Submission.ReasonCode != nil || !nullRaw(request.Submission.Result)) {
		return fmt.Errorf("prepare request contains terminal evidence")
	}
	if (request.Operation == OperationComplete || request.Operation == OperationSubmit) && (request.Submission.Outcome == nil || request.Submission.ReasonCode == nil) {
		return fmt.Errorf("completion request lacks terminal evidence")
	}
	return nil
}

func nullRaw(raw json.RawMessage) bool {
	return len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}

func (response LocalResponse) Validate() error {
	if response.Schema != SchemaResponse || (response.Operation != OperationPrepare && response.Operation != OperationComplete && response.Operation != OperationSubmit && response.Operation != OperationStatus && response.Operation != OperationReconcile) {
		return fmt.Errorf("invalid Accordare producer response identity")
	}
	if err := stavprotocol.ValidateTOPSID(response.TOPSID); err != nil {
		return err
	}
	if err := stavprotocol.ValidateRequestUUID(response.RequestID); err != nil {
		return err
	}
	switch response.Disposition {
	case "prepared":
		if response.Operation != OperationPrepare || response.IntentID == "" || response.Receipt != nil || response.Status != nil || response.CandidateDigest != "" || response.ReasonCode != "symphony.accordare.audit.intent-prepared" {
			return fmt.Errorf("invalid prepared producer response")
		}
		return stavprotocol.ValidateRequestUUID(response.IntentID)
	case "committed":
		if (response.Operation != OperationComplete && response.Operation != OperationSubmit) || response.Receipt == nil || response.Status != nil || response.ReasonCode != "symphony.accordare.audit.committed" || response.CandidateDigest == "" || response.IntentID == "" {
			return fmt.Errorf("invalid committed producer response")
		}
		if err := stavprotocol.ValidateRequestUUID(response.IntentID); err != nil {
			return err
		}
		return response.Receipt.Validate()
	case "pending":
		if (response.Operation != OperationComplete && response.Operation != OperationSubmit) || response.Receipt != nil || response.Status != nil || response.ReasonCode != "symphony.accordare.audit.pending" || response.CandidateDigest == "" || response.IntentID == "" {
			return fmt.Errorf("invalid pending producer response")
		}
		return stavprotocol.ValidateRequestUUID(response.IntentID)
	case "succeeded":
		if response.Operation == OperationSubmit || response.Operation == OperationPrepare || response.Operation == OperationComplete || response.Receipt != nil || response.Status == nil || response.CandidateDigest != "" || response.IntentID != "" || response.ReasonCode != "symphony.accordare.audit.succeeded" {
			return fmt.Errorf("invalid status producer response")
		}
		if response.Status.Schema != SchemaStatus || response.Status.TOPSID != response.TOPSID || response.Status.Pending != response.Status.IntentPending+response.Status.AppendPending || response.Status.ReconciliationNeeded != (response.Status.Pending > 0) {
			return fmt.Errorf("invalid producer status")
		}
	case "rejected":
		if response.Receipt != nil || response.Status != nil || response.CandidateDigest != "" || response.IntentID != "" || response.ReasonCode == "" {
			return fmt.Errorf("invalid rejected producer response")
		}
	default:
		return fmt.Errorf("invalid producer response disposition")
	}
	return nil
}

func decodeStrict(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return fmt.Errorf("trailing JSON values")
	}
	return nil
}
