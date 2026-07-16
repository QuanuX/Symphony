package peerauth

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/identity"
)

const Mechanism = "unix_peer_credentials"

var (
	ErrPeerUnauthenticated = errors.New("kernel peer credentials are unavailable")
	ErrSubjectUnmapped     = errors.New("kernel peer credentials are not mapped to a canonical subject")
)

// Credentials are captured from the accepted Unix connection by the kernel.
// PID is diagnostic identity evidence for the connection lifetime; subject
// mapping intentionally uses the stable UID/GID pair only.
type Credentials struct {
	PID int32
	UID uint32
	GID uint32
}

type Mapping struct {
	SubjectID   string
	SubjectKind string
	UID         uint32
	GID         uint32
}

type Peer struct {
	Credentials Credentials
	Subject     identity.Subject
	Mapped      bool
}

type credentialKey struct {
	uid uint32
	gid uint32
}

type Resolver struct {
	byCredentials map[credentialKey]identity.Subject
}

func NewResolver(mappings []Mapping) (Resolver, error) {
	resolver := Resolver{byCredentials: make(map[credentialKey]identity.Subject, len(mappings))}
	seenSubjects := make(map[string]struct{}, len(mappings))
	for i, mapping := range mappings {
		if mapping.SubjectID == "" {
			return Resolver{}, fmt.Errorf("subject mapping %d has an empty subject ID", i)
		}
		if mapping.SubjectKind == "" {
			return Resolver{}, fmt.Errorf("subject mapping %q has an empty subject kind", mapping.SubjectID)
		}
		if _, exists := seenSubjects[mapping.SubjectID]; exists {
			return Resolver{}, fmt.Errorf("duplicate subject mapping %q", mapping.SubjectID)
		}
		key := credentialKey{uid: mapping.UID, gid: mapping.GID}
		if existing, exists := resolver.byCredentials[key]; exists {
			return Resolver{}, fmt.Errorf("operating-system identity uid=%d gid=%d maps ambiguously to %q and %q", mapping.UID, mapping.GID, existing.ID, mapping.SubjectID)
		}
		seenSubjects[mapping.SubjectID] = struct{}{}
		resolver.byCredentials[key] = identity.Subject{
			ID:        mapping.SubjectID,
			Kind:      mapping.SubjectKind,
			Authority: Mechanism,
		}
	}
	return resolver, nil
}

func (r Resolver) Resolve(credentials Credentials) (identity.Subject, bool) {
	subject, ok := r.byCredentials[credentialKey{uid: credentials.UID, gid: credentials.GID}]
	return subject, ok
}

func Authenticate(conn net.Conn, resolver Resolver) (Peer, error) {
	credentials, err := CredentialsFromConn(conn)
	if err != nil {
		return Peer{}, fmt.Errorf("%w: %v", ErrPeerUnauthenticated, err)
	}
	peer := Peer{Credentials: credentials}
	if subject, ok := resolver.Resolve(credentials); ok {
		peer.Subject = subject
		peer.Mapped = true
	}
	return peer, nil
}

type contextKey struct{}

type contextResult struct {
	peer Peer
	err  error
}

func ContextWithConnection(ctx context.Context, conn net.Conn, resolver Resolver) context.Context {
	peer, err := Authenticate(conn, resolver)
	return context.WithValue(ctx, contextKey{}, contextResult{peer: peer, err: err})
}

func PeerFromContext(ctx context.Context) (Peer, error) {
	result, ok := ctx.Value(contextKey{}).(contextResult)
	if !ok || result.err != nil {
		if ok && result.err != nil {
			return Peer{}, result.err
		}
		return Peer{}, ErrPeerUnauthenticated
	}
	return result.peer, nil
}

func SubjectFromContext(ctx context.Context) (identity.Subject, error) {
	peer, err := PeerFromContext(ctx)
	if err != nil {
		return identity.Subject{}, err
	}
	if !peer.Mapped {
		return identity.Subject{}, ErrSubjectUnmapped
	}
	return peer.Subject, nil
}
