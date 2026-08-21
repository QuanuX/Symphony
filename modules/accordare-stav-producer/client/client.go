package client

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/config"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/packageinstall"
	accordarepaths "github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/paths"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/peer"
	accordareprotocol "github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/protocol"
)

const (
	defaultDeadline = 5 * time.Second
	maxFrameBytes   = 1 << 20
)

type Client struct{ config config.Config }

type Installation struct{ Binary, Receipt, ReceiptDigest, Version string }

func VerifyInstallation(prefix, version string) (Installation, error) {
	result, err := packageinstall.Inspect(prefix, version)
	if err != nil {
		return Installation{}, err
	}
	return Installation{Binary: result.Binary, Receipt: result.Receipt, ReceiptDigest: result.ReceiptDigest, Version: result.Version}, nil
}

type InstallationEvidence struct {
	ComponentID      string `json:"component_id"`
	ExecutableDigest string `json:"executable_digest"`
	ReceiptDigest    string `json:"receipt_digest"`
	Version          string `json:"version"`
}

type Submission struct {
	Command     []byte
	Coordinator InstallationEvidence
	Operation   string
	Result      []byte
	TOPSID      string
}

func terminal(value string) *string { return &value }

type AuditResult struct {
	CandidateDigest string
	Disposition     string
	ReasonCode      string
	Receipt         *stavprotocol.Receipt
	Pending         uint64
	IntentID        string
	IntentPending   uint64
	AppendPending   uint64
}

func (client *Client) Prepare(ctx context.Context, requestID string, submission Submission) (AuditResult, error) {
	return client.submit(ctx, requestID, accordareprotocol.OperationPrepare, submission, "")
}

func (client *Client) Complete(ctx context.Context, requestID string, submission Submission) (AuditResult, error) {
	return client.submit(ctx, requestID, accordareprotocol.OperationComplete, submission, "succeeded")
}

func (client *Client) CompleteTerminal(ctx context.Context, requestID string, submission Submission, outcome string) (AuditResult, error) {
	if outcome != "failed" && outcome != "unavailable" {
		return AuditResult{}, fmt.Errorf("terminal outcome must be failed or unavailable")
	}
	return client.submit(ctx, requestID, accordareprotocol.OperationComplete, submission, outcome)
}

func LoadConfig(path string) (config.Config, error) { return config.Load(path) }

func ConfigPath(scope, topsID string) (string, error) {
	parsed, err := accordarepaths.ParseScope(scope)
	if err != nil {
		return "", err
	}
	layout, err := accordarepaths.Resolve(parsed, topsID)
	if err != nil {
		return "", err
	}
	return layout.ConfigFile, nil
}

func New(cfg config.Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &Client{config: cfg}, nil
}

func NewFromConfig(path string) (*Client, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return nil, err
	}
	return New(cfg)
}

func (client *Client) Submit(ctx context.Context, requestID string, submission Submission) (AuditResult, error) {
	return client.submit(ctx, requestID, accordareprotocol.OperationSubmit, submission, "succeeded")
}

func (client *Client) submit(ctx context.Context, requestID, operation string, submission Submission, outcome string) (AuditResult, error) {
	var result []byte
	var outcomeValue, reasonCode *string
	if outcome != "" {
		outcomeValue = terminal(outcome)
		reason := "symphony." + strings.ReplaceAll(submission.Operation, "named_version_", "sav.named-version.") + "." + outcome
		reasonCode = terminal(reason)
	}
	if outcome == "succeeded" {
		result = append([]byte(nil), submission.Result...)
	}
	response, err := client.Do(ctx, accordareprotocol.LocalRequest{
		Operation: operation, RequestID: requestID, Schema: accordareprotocol.SchemaRequest,
		Submission: &accordareprotocol.Submission{Command: append([]byte(nil), submission.Command...), Coordinator: accordareprotocol.InstallationEvidence(submission.Coordinator), Operation: submission.Operation, Outcome: outcomeValue, ReasonCode: reasonCode, Result: result, Schema: accordareprotocol.SchemaSubmission, TOPSID: submission.TOPSID},
		TOPSID:     submission.TOPSID,
	})
	if err != nil {
		return AuditResult{}, err
	}
	return auditResult(response), nil
}

func (client *Client) Status(ctx context.Context, requestID, topsID string) (AuditResult, error) {
	return client.control(ctx, requestID, topsID, accordareprotocol.OperationStatus)
}

func (client *Client) Reconcile(ctx context.Context, requestID, topsID string) (AuditResult, error) {
	return client.control(ctx, requestID, topsID, accordareprotocol.OperationReconcile)
}

func (client *Client) control(ctx context.Context, requestID, topsID, operation string) (AuditResult, error) {
	response, err := client.Do(ctx, accordareprotocol.LocalRequest{Operation: operation, RequestID: requestID, Schema: accordareprotocol.SchemaRequest, TOPSID: topsID})
	if err != nil {
		return AuditResult{}, err
	}
	return auditResult(response), nil
}

func auditResult(response accordareprotocol.LocalResponse) AuditResult {
	result := AuditResult{CandidateDigest: response.CandidateDigest, Disposition: response.Disposition, IntentID: response.IntentID, ReasonCode: response.ReasonCode, Receipt: response.Receipt}
	if response.Status != nil {
		result.Pending = response.Status.Pending
		result.IntentPending = response.Status.IntentPending
		result.AppendPending = response.Status.AppendPending
	}
	return result
}

func (client *Client) Do(ctx context.Context, request accordareprotocol.LocalRequest) (accordareprotocol.LocalResponse, error) {
	if err := request.Validate(); err != nil {
		return accordareprotocol.LocalResponse{}, err
	}
	if request.TOPSID != client.config.TOPSID {
		return accordareprotocol.LocalResponse{}, fmt.Errorf("Accordare client request TOPS does not match configuration")
	}
	info, err := os.Lstat(client.config.Listen.Address)
	if err != nil || info.Mode()&os.ModeSocket == 0 {
		return accordareprotocol.LocalResponse{}, fmt.Errorf("Accordare client endpoint is unavailable or unsafe")
	}
	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", client.config.Listen.Address)
	if err != nil {
		return accordareprotocol.LocalResponse{}, fmt.Errorf("Accordare client connect: %w", err)
	}
	defer conn.Close()
	deadline := time.Now().Add(defaultDeadline)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return accordareprotocol.LocalResponse{}, err
	}
	credentials, err := peer.FromConn(conn)
	if err != nil || uint64(credentials.UID) != client.config.Identity.UID || uint64(credentials.GID) != client.config.Identity.GID {
		return accordareprotocol.LocalResponse{}, fmt.Errorf("Accordare client connected endpoint is not the configured producer")
	}
	payload, err := accordareprotocol.EncodeRequest(request)
	if err != nil {
		return accordareprotocol.LocalResponse{}, err
	}
	if err := stavprotocol.WriteFrame(conn, payload, maxFrameBytes); err != nil {
		return accordareprotocol.LocalResponse{}, fmt.Errorf("Accordare client write: %w", err)
	}
	responsePayload, err := stavprotocol.ReadFrame(conn, maxFrameBytes)
	if err != nil {
		return accordareprotocol.LocalResponse{}, fmt.Errorf("Accordare client read: %w", err)
	}
	response, err := accordareprotocol.DecodeResponse(responsePayload)
	if err != nil {
		return accordareprotocol.LocalResponse{}, fmt.Errorf("Accordare client decode: %w", err)
	}
	if response.RequestID != request.RequestID || response.Operation != request.Operation || response.TOPSID != request.TOPSID {
		return accordareprotocol.LocalResponse{}, fmt.Errorf("Accordare client response binding mismatch")
	}
	return response, nil
}
