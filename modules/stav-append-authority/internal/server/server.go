package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/peerauth"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/storage"
)

const (
	connectionDeadline = 5 * time.Second
	maxConnections     = 64
)

type Server struct {
	config   stavprotocol.AppendAuthorityConfig
	resolver peerauth.Resolver
	ledger   *storage.Ledger
}

func New(cfg stavprotocol.AppendAuthorityConfig) (*Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	resolver, err := peerauth.NewResolver(cfg.Authentication)
	if err != nil {
		return nil, err
	}
	if err := resolver.VerifyProcessAuthority(); err != nil {
		return nil, err
	}
	recoveryDir := filepath.Join(filepath.Dir(cfg.Ledger.Path), "recovery")
	ledger, err := storage.Open(cfg.Ledger.Path, recoveryDir, cfg.TOPSID, cfg.Ledger.MaxBytes)
	if err != nil {
		return nil, err
	}
	return &Server{config: cfg, resolver: resolver, ledger: ledger}, nil
}

func (s *Server) Run(ctx context.Context) error {
	if err := prepareSocket(s.config.Listen.Address); err != nil {
		return err
	}
	listener, err := net.Listen("unix", s.config.Listen.Address)
	if err != nil {
		return fmt.Errorf("stav server: listen: %w", err)
	}
	if err := os.Chmod(s.config.Listen.Address, 0o660); err != nil {
		_ = listener.Close()
		return fmt.Errorf("stav server: set socket mode: %w", err)
	}
	defer listener.Close()

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = listener.Close()
		case <-done:
		}
	}()
	defer close(done)

	semaphore := make(chan struct{}, maxConnections)
	var workers sync.WaitGroup
	defer workers.Wait()
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("stav server: accept: %w", err)
		}
		select {
		case semaphore <- struct{}{}:
			workers.Add(1)
			go func() {
				defer workers.Done()
				defer func() { <-semaphore }()
				s.handle(conn)
			}()
		default:
			_ = conn.Close()
		}
	}
}

func (s *Server) Close() error {
	return s.ledger.Close()
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(connectionDeadline))
	credentials, err := peerauth.CredentialsFromConn(conn)
	if err != nil {
		return
	}
	payload, err := stavprotocol.ReadFrame(conn, stavprotocol.MaxRequestBytes)
	if err != nil {
		return
	}
	request, err := stavprotocol.DecodeLocalRequest(payload)
	if err != nil {
		return
	}
	response := s.dispatch(request, credentials)
	encoded, err := stavprotocol.EncodeLocalResponse(response)
	if err != nil {
		return
	}
	_ = stavprotocol.WriteFrame(conn, encoded, stavprotocol.MaxResponseBytes)
}

func (s *Server) dispatch(request stavprotocol.LocalRequest, credentials peerauth.Credentials) stavprotocol.LocalResponse {
	if request.TOPSID != s.config.TOPSID {
		return rejected(request, s.config.TOPSID, stavprotocol.ReasonResponseInvalidRequest)
	}
	if request.Operation == stavprotocol.LocalOperationAppend {
		grant, ok := s.resolver.Producer(credentials)
		if !ok {
			return rejected(request, s.config.TOPSID, stavprotocol.ReasonResponseUnauthorizedPeer)
		}
		return s.append(request, grant)
	}
	grant, ok := s.resolver.Reader(credentials)
	if !ok {
		return rejected(request, s.config.TOPSID, stavprotocol.ReasonResponseUnauthorizedPeer)
	}
	switch request.Operation {
	case stavprotocol.LocalOperationStatus:
		status := s.ledger.Status(s.config.Mode, true)
		return succeeded(request, s.config.TOPSID, func(response *stavprotocol.LocalResponse) { response.Status = &status })
	case stavprotocol.LocalOperationVerify:
		verification := s.ledger.Verify(request.Verify.AfterSequence, request.Verify.ThroughSequence)
		return succeeded(request, s.config.TOPSID, func(response *stavprotocol.LocalResponse) { response.Verification = &verification })
	case stavprotocol.LocalOperationQuery:
		classifications := make(map[string]bool, len(grant.Classifications))
		for _, classification := range grant.Classifications {
			classifications[classification] = true
		}
		page, err := s.ledger.Query(*request.Query, classifications)
		if err != nil {
			return rejected(request, s.config.TOPSID, stavprotocol.ReasonResponseInvalidRequest)
		}
		return succeeded(request, s.config.TOPSID, func(response *stavprotocol.LocalResponse) { response.Page = &page })
	default:
		return rejected(request, s.config.TOPSID, stavprotocol.ReasonResponseOperationDenied)
	}
}

func (s *Server) append(request stavprotocol.LocalRequest, grant stavprotocol.ProducerGrant) stavprotocol.LocalResponse {
	eventClassGranted := false
	permitted := false
	for _, permission := range grant.Permissions {
		if permission.EventClass == request.Candidate.Operation.EventClass {
			eventClassGranted = true
			if permission.OperationID == request.Candidate.Operation.OperationID {
				permitted = true
				break
			}
		}
	}
	if !permitted {
		reason := stavprotocol.ReasonReceiptOperationDenied
		if !eventClassGranted {
			reason = stavprotocol.ReasonReceiptEventClassDenied
		}
		receipt, err := rejectedReceipt(*request.Candidate, reason)
		if err != nil {
			return rejected(request, s.config.TOPSID, stavprotocol.ReasonResponseInternalFailure)
		}
		return succeeded(request, s.config.TOPSID, func(response *stavprotocol.LocalResponse) { response.Receipt = &receipt })
	}
	receipt, err := s.ledger.Append(*request.Candidate, grant.Producer, time.Now())
	if err != nil {
		reason := stavprotocol.ReasonReceiptLedgerUnavailable
		switch {
		case errors.Is(err, storage.ErrIdempotencyConflict):
			reason = stavprotocol.ReasonReceiptIdempotencyConflict
		case errors.Is(err, storage.ErrLedgerFull):
			reason = stavprotocol.ReasonReceiptLedgerFull
		}
		receipt, receiptErr := rejectedReceipt(*request.Candidate, reason)
		if receiptErr != nil {
			return rejected(request, s.config.TOPSID, stavprotocol.ReasonResponseInternalFailure)
		}
		return succeeded(request, s.config.TOPSID, func(response *stavprotocol.LocalResponse) { response.Receipt = &receipt })
	}
	return succeeded(request, s.config.TOPSID, func(response *stavprotocol.LocalResponse) { response.Receipt = &receipt })
}

func rejectedReceipt(candidate stavprotocol.Candidate, reason string) (stavprotocol.Receipt, error) {
	digest, err := stavprotocol.CandidateDigest(candidate)
	if err != nil {
		return stavprotocol.Receipt{}, err
	}
	return stavprotocol.Receipt{
		CandidateDigest: digest,
		Commit:          stavprotocol.CommitResult{ReasonCode: reason, State: "not_committed"},
		Disposition:     "rejected",
		ReasonCode:      reason,
		RequestID:       candidate.Correlation.RequestID,
		Schema:          stavprotocol.SchemaReceipt,
		TOPSID:          candidate.Topology.TOPSID,
	}, nil
}

func succeeded(request stavprotocol.LocalRequest, topsID string, payload func(*stavprotocol.LocalResponse)) stavprotocol.LocalResponse {
	response := stavprotocol.LocalResponse{
		Disposition: stavprotocol.LocalDispositionSucceeded,
		Operation:   request.Operation,
		ReasonCode:  stavprotocol.ReasonResponseSucceeded,
		RequestID:   request.RequestID,
		Schema:      stavprotocol.SchemaLocalResponse,
		TOPSID:      topsID,
	}
	payload(&response)
	return response
}

func rejected(request stavprotocol.LocalRequest, topsID, reason string) stavprotocol.LocalResponse {
	return stavprotocol.LocalResponse{
		Disposition: stavprotocol.LocalDispositionRejected,
		Operation:   request.Operation,
		ReasonCode:  reason,
		RequestID:   request.RequestID,
		Schema:      stavprotocol.SchemaLocalResponse,
		TOPSID:      topsID,
	}
}

func prepareSocket(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("stav server: socket path is not absolute")
	}
	if err := ensureDirectory(filepath.Dir(path)); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("stav server: inspect socket: %w", err)
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("stav server: refusing non-socket object at listen path")
	}
	connection, dialErr := net.DialTimeout("unix", path, 200*time.Millisecond)
	if dialErr == nil {
		_ = connection.Close()
		return fmt.Errorf("stav server: an active listener already owns the socket")
	}
	if !errors.Is(dialErr, syscall.ECONNREFUSED) && !errors.Is(dialErr, syscall.ENOENT) {
		return fmt.Errorf("stav server: cannot prove existing socket is stale: %w", dialErr)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("stav server: remove stale socket: %w", err)
	}
	return syncDirectory(filepath.Dir(path))
}

func ensureDirectory(path string) error {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		return fmt.Errorf("stav server: unsafe directory")
	}
	parent := filepath.Dir(path)
	if parent != path {
		if err := ensureDirectory(parent); err != nil {
			return err
		}
	}
	info, err := os.Lstat(path)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 && permittedSystemAlias(path) {
			return nil
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("stav server: unsafe directory component")
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("stav server: inspect directory: %w", err)
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		return fmt.Errorf("stav server: create directory: %w", err)
	}
	return nil
}

func permittedSystemAlias(path string) bool {
	expected := map[string]string{
		"/etc": "/private/etc",
		"/tmp": "/private/tmp",
		"/var": "/private/var",
	}
	want, ok := expected[path]
	if !ok {
		return false
	}
	resolved, err := filepath.EvalSymlinks(path)
	return err == nil && resolved == want
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
