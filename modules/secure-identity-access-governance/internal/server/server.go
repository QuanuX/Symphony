package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/config"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/identity"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/model"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/peerauth"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/policy"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/policyadmin"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/provider"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/stavproducer"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/version"
)

const (
	maxHeaderBytes               = 16 << 10
	shutdownTimeout              = 5 * time.Second
	activeSocketProbeTimeout     = 250 * time.Millisecond
	maxAuthorizationRequestBytes = 64 << 10
	maxPolicyRequestBytes        = 1 << 20
)

type auditSink interface {
	Submit(context.Context, stavproducer.Record) (stavprotocol.Receipt, error)
}

type Server struct {
	config   config.Config
	registry *provider.Registry
	resolver peerauth.Resolver
	policy   *policy.Engine
	admin    *policyadmin.Manager
	audit    auditSink
	now      func() time.Time
}

func New(cfg config.Config, registry *provider.Registry) (*Server, error) {
	return NewWithAudit(cfg, registry, nil)
}

func NewWithAudit(cfg config.Config, registry *provider.Registry, audit auditSink) (*Server, error) {
	return newServer(cfg, registry, audit, nil)
}

func NewWithPolicyAdministration(cfg config.Config, registry *provider.Registry, audit auditSink, admin *policyadmin.Manager) (*Server, error) {
	if admin == nil {
		return nil, fmt.Errorf("policy administration manager is required")
	}
	return newServer(cfg, registry, audit, admin)
}

func newServer(cfg config.Config, registry *provider.Registry, audit auditSink, admin *policyadmin.Manager) (*Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if registry == nil {
		return nil, fmt.Errorf("provider registry is required")
	}
	mappings := make([]peerauth.Mapping, 0)
	if cfg.Authentication != nil {
		mappings = make([]peerauth.Mapping, 0, len(cfg.Authentication.Subjects))
		for _, subject := range cfg.Authentication.Subjects {
			mappings = append(mappings, peerauth.Mapping{
				SubjectID:   subject.ID,
				SubjectKind: subject.Kind,
				UID:         *subject.UID,
				GID:         *subject.GID,
			})
		}
	}
	resolver, err := peerauth.NewResolver(mappings)
	if err != nil {
		return nil, fmt.Errorf("configure peer subject mapping: %w", err)
	}
	policyEngine, err := policy.New(cfg, time.Now)
	if err != nil {
		return nil, fmt.Errorf("configure authorization policy: %w", err)
	}
	if admin != nil {
		effective, digest, err := admin.Effective()
		if err != nil {
			return nil, fmt.Errorf("load effective authorization policy: %w", err)
		}
		if err := policyEngine.Replace(effective, digest); err != nil {
			return nil, fmt.Errorf("activate effective authorization policy: %w", err)
		}
	}
	return &Server{
		config: cfg, registry: registry, resolver: resolver,
		policy: policyEngine, admin: admin, audit: audit, now: time.Now,
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/status", s.handleStatus)
	mux.HandleFunc("/v1/providers", s.handleProviders)
	mux.HandleFunc("/v1/authorization/decisions", s.handleAuthorization)
	if s.admin != nil {
		mux.HandleFunc("/v1/policy/status", s.handlePolicyStatus)
		mux.HandleFunc("/v1/policy/proposals", s.handlePolicyProposal)
		mux.HandleFunc("/v1/policy/apply", s.handlePolicyApply)
		mux.HandleFunc("/v1/policy/recover", s.handlePolicyRecover)
	}
	return s.requireAuthenticatedPeer(mux)
}

func (s *Server) handlePolicyStatus(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeError(writer, http.StatusMethodNotAllowed, "request.method_not_allowed", "method not allowed")
		return
	}
	result, err := s.admin.Status()
	if err != nil {
		writeError(writer, http.StatusServiceUnavailable, "policy.state_unavailable", "SSIAG policy state is unavailable")
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handlePolicyProposal(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeError(writer, http.StatusMethodNotAllowed, "request.method_not_allowed", "method not allowed")
		return
	}
	var candidate policyadmin.ProposalRequest
	if !decodeBoundedJSON(writer, request, &candidate) {
		return
	}
	subject, err := s.policyAdministrator(request.Context(), candidate.AuthorityBasis, "symphony.ssiag.policy.propose")
	if err != nil {
		writeError(writer, http.StatusForbidden, "policy.permission_denied", "target-host authority or an exact SSIAG permission is required")
		return
	}
	proposal, err := s.admin.Propose(subject, candidate)
	if err != nil {
		writePolicyAdminError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, proposal)
}

func (s *Server) handlePolicyApply(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeError(writer, http.StatusMethodNotAllowed, "request.method_not_allowed", "method not allowed")
		return
	}
	var candidate policyadmin.ApplyRequest
	if !decodeBoundedJSON(writer, request, &candidate) {
		return
	}
	if candidate.Protocol != policyadmin.ApplyRequestProtocol {
		writeError(writer, http.StatusBadRequest, "policy.request_invalid", "unsupported SSIAG policy apply protocol")
		return
	}
	subject, err := s.policyAdministrator(request.Context(), candidate.Proposal.AuthorityBasis, "symphony.ssiag.policy.apply")
	if err != nil || subject.ID != candidate.Proposal.Subject.ID || subject.Kind != candidate.Proposal.Subject.Kind || subject.Authority != candidate.Proposal.Subject.Authority {
		writeError(writer, http.StatusForbidden, "policy.permission_denied", "policy proposal is not bound to the authenticated target-host authority")
		return
	}
	attempt, alreadyApplied, err := s.admin.Prepare(candidate.Proposal)
	if err != nil {
		writePolicyAdminError(writer, err)
		return
	}
	if alreadyApplied {
		result, statusErr := s.admin.Status()
		if statusErr != nil {
			writePolicyAdminError(writer, statusErr)
			return
		}
		result.Operation = "apply"
		result.ReadOnly = false
		writeJSON(writer, http.StatusOK, result)
		return
	}
	if attempt.Stage == "prepared" {
		receipt, err := s.auditPolicyChange(request.Context(), attempt.Proposal)
		if err != nil {
			writeError(writer, http.StatusServiceUnavailable, "audit.append_failed", "policy mutation is prepared but cannot proceed until STAV audit succeeds")
			return
		}
		attempt, err = s.admin.MarkAudited(attempt.Proposal.ProposalDigest, receipt)
		if err != nil {
			writePolicyAdminError(writer, err)
			return
		}
	}
	result, effective, err := s.admin.Commit(attempt.Proposal.ProposalDigest, false)
	if err != nil {
		writePolicyAdminError(writer, err)
		return
	}
	if err := s.policy.Replace(effective, result.PolicyDigest); err != nil {
		writeError(writer, http.StatusInternalServerError, "policy.activation_failed", "policy state committed but runtime activation requires service restart")
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handlePolicyRecover(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeError(writer, http.StatusMethodNotAllowed, "request.method_not_allowed", "method not allowed")
		return
	}
	var candidate policyadmin.RecoveryRequest
	if !decodeBoundedJSON(writer, request, &candidate) {
		return
	}
	pending, err := s.admin.Pending(candidate)
	if err != nil {
		writePolicyAdminError(writer, err)
		return
	}
	subject, err := s.policyAdministrator(request.Context(), pending.Proposal.AuthorityBasis, "symphony.ssiag.policy.recover")
	if err != nil || subject.ID != pending.Proposal.Subject.ID || subject.Kind != pending.Proposal.Subject.Kind || subject.Authority != pending.Proposal.Subject.Authority {
		writeError(writer, http.StatusForbidden, "policy.permission_denied", "recovery is not bound to the authenticated target-host authority")
		return
	}
	if pending.Stage == "prepared" {
		receipt, err := s.auditPolicyChange(request.Context(), pending.Proposal)
		if err != nil {
			writeError(writer, http.StatusServiceUnavailable, "audit.append_failed", "policy recovery cannot proceed until the idempotent STAV audit succeeds")
			return
		}
		pending, err = s.admin.MarkAudited(pending.Proposal.ProposalDigest, receipt)
		if err != nil {
			writePolicyAdminError(writer, err)
			return
		}
	}
	result, effective, err := s.admin.Commit(pending.Proposal.ProposalDigest, true)
	if err != nil {
		writePolicyAdminError(writer, err)
		return
	}
	result.Operation = "recover"
	if err := s.policy.Replace(effective, result.PolicyDigest); err != nil {
		writeError(writer, http.StatusInternalServerError, "policy.activation_failed", "recovered policy committed but runtime activation requires service restart")
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func (s *Server) policyAdministrator(ctx context.Context, basis, operation string) (identity.Subject, error) {
	peer, err := peerauth.PeerFromContext(ctx)
	if err != nil {
		return identity.Subject{}, err
	}
	if basis == "host_owner" {
		if !s.isHostOwner(peer) {
			return identity.Subject{}, fmt.Errorf("peer is not the target-host owner")
		}
		return identity.Subject{
			ID:   fmt.Sprintf("symphony.host.owner.uid.%d", peer.Credentials.UID),
			Kind: "symphony.identity.host-owner", Authority: peerauth.Mechanism,
		}, nil
	}
	if basis != "granted_permission" || !peer.Mapped {
		return identity.Subject{}, fmt.Errorf("peer lacks an exact SSIAG subject mapping")
	}
	now := s.now().UTC().Truncate(time.Second)
	decision := s.policy.Evaluate(ctx, peer.Subject, model.AuthorizationRequest{
		Schema:    "symphony.ssiag.authorization-request.v1",
		RequestID: "ssiag-policy-admin-check", CorrelationID: "ssiag-policy-admin-check",
		Operation: operation, Resource: "symphony.ssiag.policy:" + s.config.TOPS.ID,
		Audience: "ssiag", Scope: "tops:" + s.config.TOPS.ID,
		RequestedAt: now, RequestedExpiresAt: now.Add(time.Minute),
	})
	if decision.Effect != "allow" {
		return identity.Subject{}, fmt.Errorf("peer lacks exact policy administration permission")
	}
	return peer.Subject, nil
}

func (s *Server) isHostOwner(peer peerauth.Peer) bool {
	if s.config.Mode == "system" {
		return peer.Credentials.UID == 0
	}
	return s.config.Authentication != nil && s.config.Authentication.Service != nil &&
		s.config.Authentication.Service.UID != nil && peer.Credentials.UID == *s.config.Authentication.Service.UID
}

func (s *Server) auditPolicyChange(ctx context.Context, proposal policyadmin.Proposal) (stavprotocol.Receipt, error) {
	if s.audit == nil {
		return stavprotocol.Receipt{}, fmt.Errorf("STAV append authority is unavailable")
	}
	return s.audit.Submit(ctx, stavproducer.Record{
		Kind: stavproducer.PolicyDecision, RequestID: proposal.RequestID, CorrelationID: proposal.CorrelationID,
		Actor:          stavprotocol.SafeReference{ID: proposal.Subject.ID, Kind: proposal.Subject.Kind},
		Authentication: stavprotocol.Authentication{MethodID: "symphony.ssiag.local-peer", State: "identified"},
		Target:         stavprotocol.SafeReference{ID: "symphony.ssiag.policy:" + proposal.TOPSID, Kind: "symphony.ssiag.policy"},
		Outcome:        "allowed",
		Configuration:  stavprotocol.Configuration{PreviousDigest: proposal.ExpectedPolicyDigest, NewDigest: proposal.DesiredPolicyDigest, State: "digests"},
		TROG:           stavprotocol.TROG{ReasonCode: "symphony.stav.trog.not-applicable", State: "not_applicable"},
		Classification: "administrative_metadata",
	})
}

func decodeBoundedJSON(writer http.ResponseWriter, request *http.Request, target any) bool {
	request.Body = http.MaxBytesReader(writer, request.Body, maxPolicyRequestBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(writer, http.StatusBadRequest, "request.invalid_json", "request is not valid bounded JSON")
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		writeError(writer, http.StatusBadRequest, "request.multiple_values", "request contains multiple JSON values")
		return false
	} else if !errors.Is(err, io.EOF) {
		writeError(writer, http.StatusBadRequest, "request.invalid_trailing_data", "request contains invalid trailing data")
		return false
	}
	return true
}

func writePolicyAdminError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, policyadmin.ErrConflict):
		writeError(writer, http.StatusConflict, "policy.compare_and_swap_conflict", err.Error())
	case errors.Is(err, policyadmin.ErrRecoveryRequired):
		writeError(writer, http.StatusConflict, "policy.recovery_required", err.Error())
	case errors.Is(err, policyadmin.ErrNoRecovery):
		writeError(writer, http.StatusNotFound, "policy.recovery_absent", err.Error())
	default:
		writeError(writer, http.StatusBadRequest, "policy.request_invalid", err.Error())
	}
}

func (s *Server) handleAuthorization(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeError(writer, http.StatusMethodNotAllowed, "request.method_not_allowed", "method not allowed")
		return
	}
	subject, err := peerauth.SubjectFromContext(request.Context())
	if err != nil {
		writeError(writer, http.StatusForbidden, "request.subject_unmapped", "authenticated peer has no canonical subject mapping")
		return
	}
	request.Body = http.MaxBytesReader(writer, request.Body, maxAuthorizationRequestBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var candidate model.AuthorizationRequest
	if err := decoder.Decode(&candidate); err != nil {
		writeError(writer, http.StatusBadRequest, "request.invalid_json", "authorization request is not valid bounded JSON")
		return
	}
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		writeError(writer, http.StatusBadRequest, "request.multiple_values", "authorization request contains multiple JSON values")
		return
	} else if !errors.Is(err, io.EOF) {
		writeError(writer, http.StatusBadRequest, "request.invalid_trailing_data", "authorization request contains invalid trailing data")
		return
	}
	now := s.now().UTC()
	if err := policy.ValidateRequest(candidate, now); err != nil {
		writeError(writer, http.StatusBadRequest, "request.authorization_invalid", err.Error())
		return
	}
	decision := s.policy.Evaluate(request.Context(), subject, candidate)
	if s.audit == nil {
		writeError(writer, http.StatusServiceUnavailable, "audit.unavailable", "STAV append authority is unavailable")
		return
	}
	outcome := "denied"
	if decision.Effect == "allow" {
		outcome = "allowed"
	}
	_, err = s.audit.Submit(request.Context(), stavproducer.Record{
		Kind: stavproducer.PolicyDecision, RequestID: decision.RequestID,
		CorrelationID:  decision.CorrelationID,
		Actor:          stavprotocol.SafeReference{ID: decision.Subject.ID, Kind: decision.Subject.Kind},
		Authentication: stavprotocol.Authentication{MethodID: "symphony.ssiag.local-peer", State: "identified"},
		Target:         stavprotocol.SafeReference{ID: decision.Target.Resource, Kind: "symphony.ssiag.resource"},
		Outcome:        outcome,
		Configuration: stavprotocol.Configuration{
			PreviousDigest: decision.ConfigDigest, NewDigest: decision.ConfigDigest, State: "digests",
		},
		TROG:           stavprotocol.TROG{ReasonCode: "symphony.stav.trog.not-applicable", State: "not_applicable"},
		Classification: "administrative_metadata",
	})
	if err != nil {
		writeError(writer, http.StatusServiceUnavailable, "audit.append_failed", "authorization decision could not be durably audited")
		return
	}
	writeJSON(writer, http.StatusOK, decision)
}

func (s *Server) Run(ctx context.Context) error {
	if s.config.Authentication == nil || s.config.Authentication.Service == nil {
		return fmt.Errorf("configuration lacks canonical service identity")
	}
	service := s.config.Authentication.Service
	if uint32(os.Geteuid()) != *service.UID || uint32(os.Getegid()) != *service.GID {
		return fmt.Errorf("process identity mismatch: effective uid=%d gid=%d, expected config uid=%d gid=%d",
			os.Geteuid(), os.Getegid(), *service.UID, *service.GID)
	}

	address := s.config.Listen.Address
	parent := filepath.Dir(address)
	if err := ensureRuntimeDir(parent); err != nil {
		return err
	}
	lease, err := acquireSocketLease(address)
	if err != nil {
		return err
	}
	defer lease.Close()
	if err := removeStaleSocket(address); err != nil {
		return err
	}

	listener, err := net.Listen("unix", address)
	if err != nil {
		return fmt.Errorf("listen on SSIAG socket: %w", err)
	}
	defer func() {
		_ = listener.Close()
		if err := os.Remove(address); err == nil || os.IsNotExist(err) {
			_ = syncDirectory(filepath.Dir(address))
		}
	}()
	if err := os.Chmod(address, 0600); err != nil {
		return fmt.Errorf("restrict SSIAG socket permissions: %w", err)
	}

	httpServer := &http.Server{
		Handler: s.Handler(),
		ConnContext: func(ctx context.Context, conn net.Conn) context.Context {
			return peerauth.ContextWithConnection(ctx, conn, s.resolver)
		},
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    maxHeaderBytes,
	}

	done := make(chan error, 1)
	go func() {
		done <- httpServer.Serve(listener)
	}()

	select {
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve SSIAG API: %w", err)
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := httpServer.Shutdown(shutdownContext); err != nil {
			return fmt.Errorf("shutdown SSIAG API: %w", err)
		}
		err := <-done
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve SSIAG API: %w", err)
		}
		return nil
	}
}

func (s *Server) requireAuthenticatedPeer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if _, err := peerauth.PeerFromContext(request.Context()); err != nil {
			writeError(writer, http.StatusUnauthorized, "request.peer_authentication_failed", "kernel peer authentication failed")
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func (s *Server) handleStatus(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeError(writer, http.StatusMethodNotAllowed, "request.method_not_allowed", "method not allowed")
		return
	}
	status := model.Status{
		Schema:        "symphony.ssiag.status.v1",
		Name:          "secure-identity-access-governance",
		Version:       version.Version,
		Ready:         true,
		Mode:          s.config.Mode,
		TOPSID:        s.config.TOPS.ID,
		TOPSName:      s.config.TOPS.Name,
		Transport:     "unix",
		ProviderCount: len(s.registry.Descriptors()),
	}
	writeJSON(writer, http.StatusOK, status)
}

func (s *Server) handleProviders(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeError(writer, http.StatusMethodNotAllowed, "request.method_not_allowed", "method not allowed")
		return
	}
	response := model.ProvidersResponse{
		Schema:    "symphony.ssiag.providers.v1",
		Providers: s.registry.Descriptors(),
	}
	writeJSON(writer, http.StatusOK, response)
}

func ensureRuntimeDir(path string) error {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		return fmt.Errorf("runtime directory must be absolute")
	}
	parent := filepath.Dir(path)
	if parent != path {
		if err := ensureRuntimeDir(parent); err != nil {
			return err
		}
	}
	info, err := os.Lstat(path)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 && permittedSystemAlias(path) {
			return nil
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("runtime parent is not a directory or is a symlink: %s", path)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("inspect runtime directory: %w", err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		return fmt.Errorf("create runtime directory: %w", err)
	}
	return nil
}

func permittedSystemAlias(path string) bool {
	expected := map[string]string{
		"/var": "/private/var",
		"/tmp": "/private/tmp",
		"/etc": "/private/etc",
	}
	want, ok := expected[path]
	if !ok {
		return false
	}
	resolved, err := filepath.EvalSymlinks(path)
	return err == nil && resolved == want
}

func removeStaleSocket(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect SSIAG socket: %w", err)
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("refusing to replace non-socket path: %s", path)
	}
	connection, dialErr := net.DialTimeout("unix", path, activeSocketProbeTimeout)
	if dialErr == nil {
		_ = connection.Close()
		return fmt.Errorf("refusing to replace active SSIAG socket: %s", path)
	}
	if !errors.Is(dialErr, syscall.ECONNREFUSED) && !errors.Is(dialErr, syscall.ENOENT) {
		return fmt.Errorf("cannot prove SSIAG socket is stale: %w", dialErr)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove stale SSIAG socket: %w", err)
	}
	return syncDirectory(filepath.Dir(path))
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open SSIAG runtime directory for sync: %w", err)
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("sync SSIAG runtime directory: %w", err)
	}
	return nil
}

func writeError(writer http.ResponseWriter, status int, code, message string) {
	writeJSON(writer, status, model.ErrorResponse{
		Schema:  "symphony.ssiag.error.v1",
		Code:    code,
		Message: message,
	})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Cache-Control", "no-store")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
