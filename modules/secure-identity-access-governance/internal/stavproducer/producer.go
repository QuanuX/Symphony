package stavproducer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	ssiagpaths "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
	appendclient "github.com/QuanuX/Symphony/modules/stav-append-authority/client"
)

type Kind string

const (
	AuthenticationDecision Kind = "authentication_decision"
	PolicyDecision         Kind = "policy_decision"
	ProviderOperation      Kind = "provider_operation"
	CredentialRotation     Kind = "credential_rotation"
	EnrollmentLifecycle    Kind = "enrollment_lifecycle"
	LeaseIssuance          Kind = "lease_issuance"
	LeaseRevocation        Kind = "lease_revocation"
)

type eventContract struct {
	eventClass  string
	operationID string
	intentID    string
	reasons     map[string]string
}

var contracts = map[Kind]eventContract{
	AuthenticationDecision: {
		eventClass:  "symphony.ssiag.authentication.decision",
		operationID: "symphony.ssiag.authenticate",
		intentID:    "symphony.ssiag.authentication.evaluate",
		reasons: map[string]string{
			"allowed":     "symphony.ssiag.authentication.allowed",
			"denied":      "symphony.ssiag.authentication.denied",
			"failed":      "symphony.ssiag.authentication.failed",
			"unavailable": "symphony.ssiag.authentication.unavailable",
		},
	},
	PolicyDecision: {
		eventClass:  "symphony.ssiag.policy.decision",
		operationID: "symphony.ssiag.authorize",
		intentID:    "symphony.ssiag.policy.evaluate",
		reasons: map[string]string{
			"allowed":     "symphony.ssiag.policy.allowed",
			"denied":      "symphony.ssiag.policy.denied",
			"failed":      "symphony.ssiag.policy.failed",
			"unavailable": "symphony.ssiag.policy.unavailable",
		},
	},
	ProviderOperation: {
		eventClass:  "symphony.ssiag.provider.operation",
		operationID: "symphony.ssiag.provider.execute",
		intentID:    "symphony.ssiag.provider.execute",
		reasons: map[string]string{
			"succeeded":   "symphony.ssiag.provider.succeeded",
			"failed":      "symphony.ssiag.provider.failed",
			"unavailable": "symphony.ssiag.provider.unavailable",
		},
	},
	CredentialRotation: {
		eventClass:  "symphony.ssiag.credential.rotation",
		operationID: "symphony.ssiag.credential.rotate",
		intentID:    "symphony.ssiag.credential.rotate",
		reasons: map[string]string{
			"succeeded":   "symphony.ssiag.credential.rotation.succeeded",
			"failed":      "symphony.ssiag.credential.rotation.failed",
			"unavailable": "symphony.ssiag.credential.rotation.unavailable",
		},
	},
	EnrollmentLifecycle: {
		eventClass:  "symphony.ssiag.enrollment.lifecycle",
		operationID: "symphony.ssiag.enrollment.change",
		intentID:    "symphony.ssiag.enrollment.change",
		reasons: map[string]string{
			"succeeded": "symphony.ssiag.enrollment.succeeded",
			"failed":    "symphony.ssiag.enrollment.failed",
		},
	},
	LeaseIssuance: {
		eventClass:  "symphony.ssiag.lease.lifecycle",
		operationID: "symphony.ssiag.lease.issue",
		intentID:    "symphony.ssiag.lease.issue",
		reasons: map[string]string{
			"succeeded": "symphony.ssiag.lease.issued",
			"failed":    "symphony.ssiag.lease.failed",
		},
	},
	LeaseRevocation: {
		eventClass:  "symphony.ssiag.lease.lifecycle",
		operationID: "symphony.ssiag.lease.revoke",
		intentID:    "symphony.ssiag.lease.revoke",
		reasons: map[string]string{
			"succeeded": "symphony.ssiag.lease.revoked",
			"failed":    "symphony.ssiag.lease.failed",
		},
	},
}

type Record struct {
	Kind           Kind
	RequestID      string
	CorrelationID  string
	Actor          stavprotocol.SafeReference
	Authentication stavprotocol.Authentication
	Target         stavprotocol.SafeReference
	Outcome        string
	Configuration  stavprotocol.Configuration
	TROG           stavprotocol.TROG
	Classification string
}

type transport interface {
	Do(context.Context, stavprotocol.LocalRequest) (stavprotocol.LocalResponse, error)
}

type Producer struct {
	topsID    string
	transport transport
}

func New(topsID string, transport transport) (*Producer, error) {
	if err := stavprotocol.ValidateTOPSID(topsID); err != nil {
		return nil, err
	}
	if transport == nil {
		return nil, fmt.Errorf("ssiag STAV producer: transport is required")
	}
	return &Producer{topsID: topsID, transport: transport}, nil
}

func Open(scope ssiagpaths.Scope, topsID string) (*Producer, error) {
	path, err := configPath(scope, topsID)
	if err != nil {
		return nil, err
	}
	cfg, err := appendclient.LoadConfig(path)
	if err != nil {
		return nil, err
	}
	if cfg.TOPSID != topsID || cfg.Mode != string(scope) {
		return nil, fmt.Errorf("ssiag STAV producer: configuration binding mismatch")
	}
	client, err := appendclient.New(cfg)
	if err != nil {
		return nil, err
	}
	return New(topsID, client)
}

func (p *Producer) Submit(ctx context.Context, record Record) (stavprotocol.Receipt, error) {
	contract, ok := contracts[record.Kind]
	if !ok {
		return stavprotocol.Receipt{}, fmt.Errorf("ssiag STAV producer: unsupported event kind")
	}
	reason, ok := contract.reasons[record.Outcome]
	if !ok {
		return stavprotocol.Receipt{}, fmt.Errorf("ssiag STAV producer: unsupported outcome for event kind")
	}
	candidate := stavprotocol.Candidate{
		Actor:         stavprotocol.CandidateActor{Authentication: record.Authentication, Principal: record.Actor},
		Configuration: record.Configuration,
		Correlation:   stavprotocol.Correlation{CorrelationID: record.CorrelationID, RequestID: record.RequestID},
		Operation: stavprotocol.Operation{
			EventClass:  contract.eventClass,
			OperationID: contract.operationID,
			Target:      record.Target,
		},
		Redaction: stavprotocol.Redaction{Classification: record.Classification},
		Result:    stavprotocol.Result{IntentID: contract.intentID, Outcome: record.Outcome, ReasonCode: reason},
		Schema:    stavprotocol.SchemaCandidate,
		Topology:  stavprotocol.Topology{TOPSID: p.topsID, TROG: record.TROG},
	}
	if err := candidate.Validate(); err != nil {
		return stavprotocol.Receipt{}, fmt.Errorf("ssiag STAV producer: invalid safe record: %w", err)
	}
	response, err := p.transport.Do(ctx, stavprotocol.LocalRequest{
		Candidate: &candidate,
		Operation: stavprotocol.LocalOperationAppend,
		RequestID: record.RequestID,
		Schema:    stavprotocol.SchemaLocalRequest,
		TOPSID:    p.topsID,
	})
	if err != nil {
		return stavprotocol.Receipt{}, fmt.Errorf("ssiag STAV producer: submit: %w", err)
	}
	if err := response.Validate(); err != nil {
		return stavprotocol.Receipt{}, fmt.Errorf("ssiag STAV producer: invalid response: %w", err)
	}
	if response.Disposition != stavprotocol.LocalDispositionSucceeded || response.Receipt == nil || response.Receipt.Disposition != "committed" {
		if response.Receipt != nil {
			return stavprotocol.Receipt{}, fmt.Errorf("ssiag STAV producer: append rejected: %s", response.Receipt.ReasonCode)
		}
		return stavprotocol.Receipt{}, fmt.Errorf("ssiag STAV producer: append rejected: %s", response.ReasonCode)
	}
	return *response.Receipt, nil
}

func configPath(scope ssiagpaths.Scope, topsID string) (string, error) {
	if err := stavprotocol.ValidateTOPSID(topsID); err != nil {
		return "", err
	}
	if override := os.Getenv("SYMPHONY_STAV_CONFIG"); override != "" {
		if !filepath.IsAbs(override) {
			return "", fmt.Errorf("SYMPHONY_STAV_CONFIG must be absolute")
		}
		return filepath.Clean(override), nil
	}
	switch scope {
	case ssiagpaths.ScopeUser:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base := os.Getenv("XDG_CONFIG_HOME")
		if base == "" {
			base = filepath.Join(home, ".config")
		}
		return filepath.Join(base, "symphony", topsID, "stav", "append-authority.json"), nil
	case ssiagpaths.ScopeSystem:
		return filepath.Join("/etc/symphony", topsID, "stav", "append-authority.json"), nil
	default:
		return "", fmt.Errorf("ssiag STAV producer: unsupported scope")
	}
}
