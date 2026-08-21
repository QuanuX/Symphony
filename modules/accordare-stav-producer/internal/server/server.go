package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/config"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/peer"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/producer"
	accordareprotocol "github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/protocol"
)

const (
	connectionDeadline = 5 * time.Second
	maxConnections     = 64
	maxFrameBytes      = 1 << 20
)

type Server struct {
	config   config.Config
	producer *producer.Producer
	subjects map[[2]uint32]stavprotocol.SafeReference
}

func New(cfg config.Config, runtimeProducer *producer.Producer) (*Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if runtimeProducer == nil {
		return nil, fmt.Errorf("Accordare server requires a runtime producer")
	}
	if uint64(os.Geteuid()) != cfg.Identity.UID || uint64(os.Getegid()) != cfg.Identity.GID {
		return nil, fmt.Errorf("Accordare producer process identity does not match configuration")
	}
	subjects := make(map[[2]uint32]stavprotocol.SafeReference, len(cfg.Submitters))
	for _, submitter := range cfg.Submitters {
		subjects[[2]uint32{uint32(submitter.UID), uint32(submitter.GID)}] = submitter.Subject
	}
	return &Server{config: cfg, producer: runtimeProducer, subjects: subjects}, nil
}

func (server *Server) Run(ctx context.Context) error {
	address := server.config.Listen.Address
	if err := ensureDirectory(filepath.Dir(address)); err != nil {
		return err
	}
	lease, err := acquireSocketLease(address)
	if err != nil {
		return err
	}
	defer lease.Close()
	if err := prepareSocket(address); err != nil {
		return err
	}
	listener, err := net.Listen("unix", address)
	if err != nil {
		return fmt.Errorf("Accordare producer listen: %w", err)
	}
	if err := os.Chmod(address, 0o660); err != nil {
		_ = listener.Close()
		return fmt.Errorf("Accordare producer socket mode: %w", err)
	}
	defer func() {
		_ = listener.Close()
		if err := os.Remove(address); err == nil || os.IsNotExist(err) {
			_ = syncDirectory(filepath.Dir(address))
		}
	}()

	// Reconciliation never prevents the submission endpoint from becoming
	// available. Pending evidence remains durable and visible if STAV is down.
	go func() { _, _, _ = server.producer.Reconcile(ctx) }()

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
			return fmt.Errorf("Accordare producer accept: %w", err)
		}
		select {
		case semaphore <- struct{}{}:
			workers.Add(1)
			go func() {
				defer workers.Done()
				defer func() { <-semaphore }()
				server.handle(ctx, conn)
			}()
		default:
			_ = conn.Close()
		}
	}
}

func (server *Server) handle(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(connectionDeadline))
	credentials, err := peer.FromConn(conn)
	if err != nil {
		return
	}
	subject, authorized := server.subjects[[2]uint32{credentials.UID, credentials.GID}]
	payload, err := stavprotocol.ReadFrame(conn, maxFrameBytes)
	if err != nil {
		return
	}
	request, err := accordareprotocol.DecodeRequest(payload)
	if err != nil {
		return
	}
	response := server.dispatch(ctx, request, subject, authorized)
	encoded, err := accordareprotocol.EncodeResponse(response)
	if err != nil {
		return
	}
	_ = stavprotocol.WriteFrame(conn, encoded, maxFrameBytes)
}

func (server *Server) dispatch(ctx context.Context, request accordareprotocol.LocalRequest, subject stavprotocol.SafeReference, authorized bool) accordareprotocol.LocalResponse {
	if request.TOPSID != server.config.TOPSID {
		return rejected(request, server.config.TOPSID, "symphony.accordare.audit.invalid-request")
	}
	if !authorized {
		return rejected(request, server.config.TOPSID, "symphony.accordare.audit.unauthorized-peer")
	}
	switch request.Operation {
	case accordareprotocol.OperationSubmit:
		verified, err := accordareprotocol.VerifySubmission(*request.Submission, subject, time.Now().UTC())
		if err != nil {
			return rejected(request, server.config.TOPSID, "symphony.accordare.audit.invalid-evidence")
		}
		receipt, pending, appendErr := server.producer.Submit(ctx, verified.Candidate, verified.CandidateDigest)
		if pending {
			return accordareprotocol.LocalResponse{CandidateDigest: verified.CandidateDigest, Disposition: "pending", Operation: request.Operation, ReasonCode: "symphony.accordare.audit.pending", RequestID: request.RequestID, Schema: accordareprotocol.SchemaResponse, TOPSID: server.config.TOPSID}
		}
		if appendErr != nil {
			return rejected(request, server.config.TOPSID, "symphony.accordare.audit.persistence-failed")
		}
		return accordareprotocol.LocalResponse{CandidateDigest: verified.CandidateDigest, Disposition: "committed", Operation: request.Operation, ReasonCode: "symphony.accordare.audit.committed", Receipt: &receipt, RequestID: request.RequestID, Schema: accordareprotocol.SchemaResponse, TOPSID: server.config.TOPSID}
	case accordareprotocol.OperationReconcile:
		_, pending, _ := server.producer.Reconcile(ctx)
		return statusResponse(request, server.config.TOPSID, pending)
	case accordareprotocol.OperationStatus:
		pending, err := server.producer.Pending()
		if err != nil {
			return rejected(request, server.config.TOPSID, "symphony.accordare.audit.outbox-unavailable")
		}
		return statusResponse(request, server.config.TOPSID, pending)
	default:
		return rejected(request, server.config.TOPSID, "symphony.accordare.audit.operation-denied")
	}
}

func statusResponse(request accordareprotocol.LocalRequest, topsID string, pending uint64) accordareprotocol.LocalResponse {
	return accordareprotocol.LocalResponse{Disposition: "succeeded", Operation: request.Operation, ReasonCode: "symphony.accordare.audit.succeeded", RequestID: request.RequestID, Schema: accordareprotocol.SchemaResponse, Status: &accordareprotocol.Status{Pending: pending, Ready: true, ReconciliationNeeded: pending > 0, Schema: accordareprotocol.SchemaStatus, TOPSID: topsID}, TOPSID: topsID}
}

func rejected(request accordareprotocol.LocalRequest, topsID, reason string) accordareprotocol.LocalResponse {
	return accordareprotocol.LocalResponse{Disposition: "rejected", Operation: request.Operation, ReasonCode: reason, RequestID: request.RequestID, Schema: accordareprotocol.SchemaResponse, TOPSID: topsID}
}

func ensureDirectory(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("create Accordare runtime directory: %w", err)
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("Accordare runtime directory is unsafe")
	}
	return nil
}

func prepareSocket(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("Accordare producer socket path is not absolute")
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil || info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("Accordare producer socket path is occupied")
	}
	conn, dialErr := net.DialTimeout("unix", path, 250*time.Millisecond)
	if dialErr == nil {
		_ = conn.Close()
		return fmt.Errorf("another Accordare producer is listening")
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove stale Accordare producer socket: %w", err)
	}
	return syncDirectory(filepath.Dir(path))
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
