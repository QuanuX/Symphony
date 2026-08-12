package ssiagclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	maxConfigBytes     = 1 << 20
	maxResponseBytes   = 1 << 20
	serviceSubjectID   = "symphony.ssiag.service"
	serviceSubjectKind = "symphony.identity.service"
)

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

type ProviderDescriptor struct {
	Name         string   `json:"name"`
	Kind         string   `json:"kind"`
	Status       string   `json:"status"`
	Capabilities []string `json:"capabilities"`
	Exportable   bool     `json:"exportable"`
	Interactive  bool     `json:"interactive"`
}

type ProvidersResponse struct {
	Schema    string               `json:"schema"`
	Providers []ProviderDescriptor `json:"providers"`
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

type AuthorizationGrant struct {
	ID             string `json:"id"`
	SubjectID      string `json:"subject_id"`
	AuthorityBasis string `json:"authority_basis"`
	Operation      string `json:"operation"`
	Resource       string `json:"resource"`
	Audience       string `json:"audience"`
	Scope          string `json:"scope"`
}

type AuthorizationPolicy struct {
	DefaultEffect        string               `json:"default_effect"`
	MaxCapabilitySeconds uint64               `json:"max_capability_seconds"`
	Grants               []AuthorizationGrant `json:"grants"`
}

type PolicyProposalRequest struct {
	Protocol             string               `json:"protocol"`
	OperationID          string               `json:"operation_id"`
	RequestID            string               `json:"request_id"`
	CorrelationID        string               `json:"correlation_id"`
	AuthorityBasis       string               `json:"authority_basis"`
	ExpectedPolicyDigest string               `json:"expected_policy_digest"`
	Change               string               `json:"change"`
	DesiredPolicy        *AuthorizationPolicy `json:"desired_policy"`
	RequestedAt          time.Time            `json:"requested_at"`
	ExpiresAt            time.Time            `json:"expires_at"`
}

type PolicyProposal struct {
	Protocol             string               `json:"protocol"`
	OperationID          string               `json:"operation_id"`
	RequestID            string               `json:"request_id"`
	CorrelationID        string               `json:"correlation_id"`
	TOPSID               string               `json:"tops_id"`
	Subject              DecisionSubject      `json:"subject"`
	AuthorityBasis       string               `json:"authority_basis"`
	ExpectedPolicyDigest string               `json:"expected_policy_digest"`
	Change               string               `json:"change"`
	DesiredPolicy        *AuthorizationPolicy `json:"desired_policy"`
	DesiredPolicyDigest  string               `json:"desired_policy_digest"`
	ConfigDigest         string               `json:"config_digest"`
	CreatedAt            time.Time            `json:"created_at"`
	ExpiresAt            time.Time            `json:"expires_at"`
	CallerClassUsed      bool                 `json:"caller_class_used"`
	Canonical            bool                 `json:"canonical"`
	Applied              bool                 `json:"applied"`
	ProposalDigest       string               `json:"proposal_digest"`
}

type PolicyApplyRequest struct {
	Protocol string         `json:"protocol"`
	Proposal PolicyProposal `json:"proposal"`
}

type PolicyRecoveryRequest struct {
	Protocol              string `json:"protocol"`
	OperationID           string `json:"operation_id"`
	ExpectedAttemptDigest string `json:"expected_attempt_digest,omitempty"`
	Discover              bool   `json:"discover"`
}

type PolicyResult struct {
	Protocol         string    `json:"protocol"`
	Operation        string    `json:"operation"`
	TOPSID           string    `json:"tops_id"`
	Source           string    `json:"source"`
	Generation       uint64    `json:"generation"`
	PolicyDigest     string    `json:"policy_digest"`
	StateDigest      string    `json:"state_digest"`
	RecoveryRequired bool      `json:"recovery_required"`
	AttemptDigest    string    `json:"attempt_digest,omitempty"`
	Changed          bool      `json:"changed"`
	Recovered        bool      `json:"recovered"`
	ObservedAt       time.Time `json:"observed_at"`
	ReadOnly         bool      `json:"read_only"`
	CallerClassUsed  bool      `json:"caller_class_used"`
	Canonical        bool      `json:"canonical"`
}

type Client struct {
	httpClient *http.Client
	baseURL    string
}

type peerVerifier func(net.Conn, uint32, uint32) error

func SocketForTOPS(scope, topsID string) (string, error) {
	if err := validateTOPSID(topsID); err != nil {
		return "", err
	}
	if override := os.Getenv("SYMPHONY_SSIAG_SOCKET"); override != "" {
		if !filepath.IsAbs(override) {
			return "", fmt.Errorf("SYMPHONY_SSIAG_SOCKET must be absolute")
		}
		return filepath.Clean(override), nil
	}
	switch scope {
	case "system":
		return filepath.Join(systemRuntimeRoot(), topsID, "ssiag", "ssiag.sock"), nil
	case "user":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve user home: %w", err)
		}
		runtimeBase := os.Getenv("XDG_RUNTIME_DIR")
		if runtimeBase != "" {
			if !filepath.IsAbs(runtimeBase) {
				return "", fmt.Errorf("XDG_RUNTIME_DIR must be absolute")
			}
			return filepath.Join(runtimeBase, "symphony", topsID, "ssiag", "ssiag.sock"), nil
		}
		stateBase := os.Getenv("XDG_STATE_HOME")
		if stateBase == "" {
			stateBase = filepath.Join(home, ".local", "state")
		} else if !filepath.IsAbs(stateBase) {
			return "", fmt.Errorf("XDG_STATE_HOME must be absolute")
		}
		return filepath.Join(stateBase, "symphony", topsID, "ssiag", "run", "ssiag.sock"), nil
	default:
		return "", fmt.Errorf("unsupported SSIAG scope %q", scope)
	}
}

func validateTOPSID(value string) error {
	if len(value) != 36 || strings.ToLower(value) != value {
		return fmt.Errorf("TOPS ID must be a canonical lowercase UUID")
	}
	for i, r := range value {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return fmt.Errorf("TOPS ID must be a canonical lowercase UUID")
			}
		default:
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
				return fmt.Errorf("TOPS ID must be a canonical lowercase UUID")
			}
		}
	}
	if value == "00000000-0000-0000-0000-000000000000" || value[14] < '1' || value[14] > '8' || !strings.Contains("89ab", value[19:20]) {
		return fmt.Errorf("TOPS ID must be a non-nil RFC UUID with version 1 through 8")
	}
	return nil
}

type serviceIdentityConfig struct {
	ID   string  `json:"id"`
	Kind string  `json:"kind"`
	UID  *uint32 `json:"uid"`
	GID  *uint32 `json:"gid"`
}

type authenticationConfig struct {
	Mechanism string                 `json:"mechanism"`
	Service   *serviceIdentityConfig `json:"service"`
	Subjects  []json.RawMessage      `json:"subjects"`
}

type ssiagConfig struct {
	Schema string `json:"schema"`
	Mode   string `json:"mode"`
	TOPS   struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"tops"`
	Listen struct {
		Network string `json:"network"`
		Address string `json:"address"`
	} `json:"listen"`
	Authentication *authenticationConfig `json:"authentication"`
	Authorization  json.RawMessage       `json:"authorization"`
	Providers      []json.RawMessage     `json:"providers"`
}

func ConfigForTOPS(scope, topsID string) (string, error) {
	if err := validateTOPSID(topsID); err != nil {
		return "", err
	}
	switch scope {
	case "system":
		return filepath.Join("/etc/symphony", topsID, "ssiag", "config.json"), nil
	case "user":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve user home: %w", err)
		}
		configBase := os.Getenv("XDG_CONFIG_HOME")
		if configBase == "" {
			configBase = filepath.Join(home, ".config")
		} else if !filepath.IsAbs(configBase) {
			return "", fmt.Errorf("XDG_CONFIG_HOME must be absolute")
		}
		return filepath.Join(configBase, "symphony", topsID, "ssiag", "config.json"), nil
	default:
		return "", fmt.Errorf("unsupported SSIAG scope %q", scope)
	}
}

func NewForTOPS(scope, topsID string, timeout time.Duration) (*Client, error) {
	return newForTOPS(scope, topsID, timeout, verifyPeer)
}

func newForTOPS(scope, topsID string, timeout time.Duration, verifier peerVerifier) (*Client, error) {
	if verifier == nil {
		return nil, fmt.Errorf("SSIAG endpoint verifier is required")
	}
	configPath, err := ConfigForTOPS(scope, topsID)
	if err != nil {
		return nil, err
	}

	file, err := openNoFollow(configPath)
	if err != nil {
		return nil, fmt.Errorf("open trusted SSIAG configuration: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat trusted SSIAG configuration: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxConfigBytes {
		return nil, fmt.Errorf("trusted SSIAG configuration is not a bounded regular file")
	}
	owner, err := fileOwnerUID(info)
	if err != nil {
		return nil, err
	}
	switch scope {
	case "user":
		if owner != uint32(os.Geteuid()) || info.Mode().Perm()&0o077 != 0 {
			return nil, fmt.Errorf("user SSIAG configuration is not effective-user-owned and owner-only")
		}
	case "system":
		if owner != 0 || info.Mode().Perm()&0o022 != 0 {
			return nil, fmt.Errorf("system SSIAG configuration is not administrator-owned and protected")
		}
	default:
		return nil, fmt.Errorf("unsupported SSIAG scope %q", scope)
	}

	var cfg ssiagConfig
	decoder := json.NewDecoder(io.LimitReader(file, maxConfigBytes+1))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode SSIAG configuration: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("SSIAG configuration must contain exactly one JSON value")
	}

	if cfg.Schema != "symphony.ssiag.config.v1" {
		return nil, fmt.Errorf("unsupported config schema %q", cfg.Schema)
	}
	if cfg.TOPS.ID != topsID || cfg.Mode != scope {
		return nil, fmt.Errorf("SSIAG configuration does not match requested TOPS and scope")
	}
	if cfg.Listen.Network != "unix" || !filepath.IsAbs(cfg.Listen.Address) {
		return nil, fmt.Errorf("SSIAG configuration does not declare an absolute Unix socket")
	}
	configuredSocket, err := canonicalSocketForTOPS(scope, topsID)
	if err != nil {
		return nil, err
	}
	if filepath.Clean(cfg.Listen.Address) != configuredSocket {
		return nil, fmt.Errorf("SSIAG configuration socket does not match requested TOPS layout")
	}
	if cfg.Authentication == nil || cfg.Authentication.Mechanism != "unix_peer_credentials" || cfg.Authentication.Subjects == nil || cfg.Authentication.Service == nil {
		return nil, fmt.Errorf("SSIAG configuration lacks canonical service identity")
	}
	if cfg.Providers == nil {
		return nil, fmt.Errorf("SSIAG configuration providers must be an explicit array")
	}
	service := cfg.Authentication.Service
	if service.ID != serviceSubjectID || service.Kind != serviceSubjectKind || service.UID == nil || service.GID == nil {
		return nil, fmt.Errorf("SSIAG configuration contains an invalid canonical service identity")
	}

	expectedUID := *service.UID
	expectedGID := *service.GID

	socket := cfg.Listen.Address
	if override := os.Getenv("SYMPHONY_SSIAG_SOCKET"); override != "" {
		if !filepath.IsAbs(override) {
			return nil, fmt.Errorf("SYMPHONY_SSIAG_SOCKET must be absolute")
		}
		socket = filepath.Clean(override)
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			info, err := os.Lstat(socket)
			if err != nil {
				return nil, fmt.Errorf("inspect SSIAG socket: %w", err)
			}
			if info.Mode()&os.ModeSocket == 0 {
				return nil, fmt.Errorf("SSIAG endpoint is not a Unix socket")
			}
			owner, err := socketOwnerUID(info)
			if err != nil {
				return nil, err
			}
			if owner != expectedUID {
				return nil, fmt.Errorf("SSIAG socket owner uid=%d does not match configured service uid=%d", owner, expectedUID)
			}
			conn, err := (&net.Dialer{}).DialContext(ctx, "unix", socket)
			if err != nil {
				return nil, err
			}
			if err := verifier(conn, expectedUID, expectedGID); err != nil {
				_ = conn.Close()
				return nil, err
			}
			return conn, nil
		},
	}

	return &Client{
		httpClient: &http.Client{Transport: transport, Timeout: timeout},
		baseURL:    "http://unix",
	}, nil
}

func verifyPeer(conn net.Conn, expectedUID, expectedGID uint32) error {
	credentials, err := getPeerCredentials(conn)
	if err != nil {
		return fmt.Errorf("extract SSIAG peer credentials: %w", err)
	}
	if credentials.UID != expectedUID || credentials.GID != expectedGID {
		return fmt.Errorf("SSIAG peer identity uid=%d gid=%d does not match configured service uid=%d gid=%d", credentials.UID, credentials.GID, expectedUID, expectedGID)
	}
	return nil
}

func canonicalSocketForTOPS(scope, topsID string) (string, error) {
	switch scope {
	case "system":
		return filepath.Join(systemRuntimeRoot(), topsID, "ssiag", "ssiag.sock"), nil
	case "user":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve user home: %w", err)
		}
		runtimeBase := os.Getenv("XDG_RUNTIME_DIR")
		if runtimeBase != "" {
			if !filepath.IsAbs(runtimeBase) {
				return "", fmt.Errorf("XDG_RUNTIME_DIR must be absolute")
			}
			return filepath.Join(runtimeBase, "symphony", topsID, "ssiag", "ssiag.sock"), nil
		}
		stateBase := os.Getenv("XDG_STATE_HOME")
		if stateBase == "" {
			stateBase = filepath.Join(home, ".local", "state")
		} else if !filepath.IsAbs(stateBase) {
			return "", fmt.Errorf("XDG_STATE_HOME must be absolute")
		}
		return filepath.Join(stateBase, "symphony", topsID, "ssiag", "run", "ssiag.sock"), nil
	default:
		return "", fmt.Errorf("unsupported SSIAG scope %q", scope)
	}
}

func systemRuntimeRoot() string {
	if runtime.GOOS == "darwin" {
		return "/var/run/symphony"
	}
	return "/run/symphony"
}

func (c *Client) Status(ctx context.Context) (Status, error) {
	var result Status
	if err := c.get(ctx, "/v1/status", &result); err != nil {
		return result, err
	}
	if result.Schema != "symphony.ssiag.status.v1" {
		return result, fmt.Errorf("unsupported SSIAG status schema %q", result.Schema)
	}
	return result, nil
}

func (c *Client) Providers(ctx context.Context) (ProvidersResponse, error) {
	var result ProvidersResponse
	if err := c.get(ctx, "/v1/providers", &result); err != nil {
		return result, err
	}
	if result.Schema != "symphony.ssiag.providers.v1" {
		return result, fmt.Errorf("unsupported SSIAG providers schema %q", result.Schema)
	}
	return result, nil
}

func (c *Client) Authorize(ctx context.Context, request AuthorizationRequest) (AuthorizationDecision, error) {
	var result AuthorizationDecision
	if err := c.post(ctx, "/v1/authorization/decisions", request, &result); err != nil {
		return result, err
	}
	if result.Schema != "symphony.ssiag.authorization-decision.v1" {
		return result, fmt.Errorf("unsupported SSIAG authorization decision schema %q", result.Schema)
	}
	return result, nil
}

func (c *Client) PolicyStatus(ctx context.Context) (PolicyResult, error) {
	var result PolicyResult
	if err := c.get(ctx, "/v1/policy/status", &result); err != nil {
		return result, err
	}
	if result.Protocol != "symphony.ssiag.policy-result.v1" {
		return result, fmt.Errorf("unsupported SSIAG policy result protocol %q", result.Protocol)
	}
	return result, nil
}

func (c *Client) ProposePolicy(ctx context.Context, request PolicyProposalRequest) (PolicyProposal, error) {
	var result PolicyProposal
	if err := c.post(ctx, "/v1/policy/proposals", request, &result); err != nil {
		return result, err
	}
	if result.Protocol != "symphony.ssiag.policy-proposal.v1" {
		return result, fmt.Errorf("unsupported SSIAG policy proposal protocol %q", result.Protocol)
	}
	return result, nil
}

func (c *Client) ApplyPolicy(ctx context.Context, request PolicyApplyRequest) (PolicyResult, error) {
	var result PolicyResult
	if err := c.post(ctx, "/v1/policy/apply", request, &result); err != nil {
		return result, err
	}
	if result.Protocol != "symphony.ssiag.policy-result.v1" {
		return result, fmt.Errorf("unsupported SSIAG policy result protocol %q", result.Protocol)
	}
	return result, nil
}

func (c *Client) RecoverPolicy(ctx context.Context, request PolicyRecoveryRequest) (PolicyResult, error) {
	var result PolicyResult
	if err := c.post(ctx, "/v1/policy/recover", request, &result); err != nil {
		return result, err
	}
	if result.Protocol != "symphony.ssiag.policy-result.v1" {
		return result, fmt.Errorf("unsupported SSIAG policy result protocol %q", result.Protocol)
	}
	return result, nil
}

func ReadBoundedJSON(path string, target any) error {
	file, err := openNoFollow(path)
	if err != nil {
		return fmt.Errorf("open SSIAG policy input: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxConfigBytes {
		return fmt.Errorf("SSIAG policy input is not a bounded regular file")
	}
	decoder := json.NewDecoder(io.LimitReader(file, maxConfigBytes+1))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode SSIAG policy input: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("SSIAG policy input must contain exactly one JSON value")
	}
	return nil
}

func (c *Client) get(ctx context.Context, path string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("SSIAG request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("SSIAG returned HTTP %d", response.StatusCode)
	}
	payload, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read SSIAG response: %w", err)
	}
	if len(payload) > maxResponseBytes {
		return fmt.Errorf("SSIAG response exceeds %d bytes", maxResponseBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode SSIAG response: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("decode SSIAG response: multiple JSON values")
		}
		return fmt.Errorf("decode trailing SSIAG response: %w", err)
	}
	return nil
}

func (c *Client) post(ctx context.Context, path string, value, target any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode SSIAG request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("SSIAG request failed: %w", err)
	}
	defer response.Body.Close()
	responsePayload, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read SSIAG response: %w", err)
	}
	if len(responsePayload) > maxResponseBytes {
		return fmt.Errorf("SSIAG response exceeds %d bytes", maxResponseBytes)
	}
	if response.StatusCode != http.StatusOK {
		var failure struct {
			Code string `json:"code"`
		}
		if json.Unmarshal(responsePayload, &failure) == nil && failure.Code != "" {
			return fmt.Errorf("SSIAG request rejected: %s", failure.Code)
		}
		return fmt.Errorf("SSIAG returned HTTP %d", response.StatusCode)
	}
	decoder := json.NewDecoder(bytes.NewReader(responsePayload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode SSIAG response: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("decode SSIAG response: multiple JSON values")
	}
	return nil
}
