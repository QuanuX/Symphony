package model

import "time"

// Status is the safe, metadata-only SSIAG status projection.
type Status struct {
	Schema        string `json:"schema"`
	Name          string `json:"name"`
	Version       string `json:"version"`
	Ready         bool   `json:"ready"`
	Mode          string `json:"mode"`
	TOPSID        string `json:"tops_id"`
	TOPSName      string `json:"tops_name"`
	Transport     string `json:"transport"`
	ProviderCount int    `json:"provider_count"`
}

// ProviderDescriptor describes capabilities without exposing provider credentials.
type ProviderDescriptor struct {
	Name         string   `json:"name"`
	Kind         string   `json:"kind"`
	Status       string   `json:"status"`
	Capabilities []string `json:"capabilities"`
	Exportable   bool     `json:"exportable"`
	Interactive  bool     `json:"interactive"`
}

// ProvidersResponse is the versioned provider discovery response.
type ProvidersResponse struct {
	Schema    string               `json:"schema"`
	Providers []ProviderDescriptor `json:"providers"`
}

// ErrorResponse contains only safe error metadata.
type ErrorResponse struct {
	Schema  string `json:"schema"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type AuthorizationRequest struct {
	Schema             string    `json:"schema"`
	RequestID          string    `json:"request_id"`
	CorrelationID      string    `json:"correlation_id"`
	Operation          string    `json:"operation"`
	Resource           string    `json:"resource"`
	Audience           string    `json:"audience"`
	Scope              string    `json:"scope"`
	RequestedAt        time.Time `json:"requested_at"`
	RequestedExpiresAt time.Time `json:"requested_expires_at"`
}

type DecisionSubject struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Authority string `json:"authority"`
}

type DecisionTarget struct {
	Operation string `json:"operation"`
	Resource  string `json:"resource"`
	Audience  string `json:"audience"`
	Scope     string `json:"scope"`
}

// Capability is safe decision evidence, not a secret or transferable bearer
// credential. Later canonical mutation must revalidate through a ratified
// SSIAG trust channel rather than trusting this object by possession alone.
type Capability struct {
	Protocol       string          `json:"protocol"`
	CapabilityID   string          `json:"capability_id"`
	Subject        DecisionSubject `json:"subject"`
	TOPSID         string          `json:"tops_id"`
	Target         DecisionTarget  `json:"target"`
	AuthorityBasis string          `json:"authority_basis"`
	GrantID        string          `json:"grant_id"`
	RequestID      string          `json:"request_id"`
	CorrelationID  string          `json:"correlation_id"`
	IssuedAt       time.Time       `json:"issued_at"`
	ExpiresAt      time.Time       `json:"expires_at"`
	PolicyDigest   string          `json:"policy_digest"`
	ConfigDigest   string          `json:"config_digest"`
	BindingDigest  string          `json:"binding_digest"`
	Transferable   bool            `json:"transferable"`
	CanonicalApply bool            `json:"canonical_apply"`
}

type AuthorizationDecision struct {
	Schema          string          `json:"schema"`
	DecisionID      string          `json:"decision_id"`
	RequestID       string          `json:"request_id"`
	CorrelationID   string          `json:"correlation_id"`
	TOPSID          string          `json:"tops_id"`
	Subject         DecisionSubject `json:"subject"`
	Target          DecisionTarget  `json:"target"`
	Effect          string          `json:"effect"`
	ReasonCode      string          `json:"reason_code"`
	AuthorityBasis  *string         `json:"authority_basis"`
	Capability      *Capability     `json:"capability"`
	PolicyDigest    string          `json:"policy_digest"`
	ConfigDigest    string          `json:"config_digest"`
	DecidedAt       time.Time       `json:"decided_at"`
	ExpiresAt       *time.Time      `json:"expires_at"`
	CallerClassUsed bool            `json:"caller_class_used"`
	CanonicalApply  bool            `json:"canonical_apply"`
}
