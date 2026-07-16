package peerauth

import (
	"context"
	"errors"
	"net"
	"testing"
)

func TestResolverMapsExactUIDAndGID(t *testing.T) {
	resolver, err := NewResolver([]Mapping{{
		SubjectID: "operator.primary", SubjectKind: "operator", UID: 501, GID: 20,
	}})
	if err != nil {
		t.Fatal(err)
	}
	subject, ok := resolver.Resolve(Credentials{PID: 123, UID: 501, GID: 20})
	if !ok || subject.ID != "operator.primary" || subject.Authority != Mechanism {
		t.Fatalf("unexpected subject: %+v mapped=%t", subject, ok)
	}
	if _, ok := resolver.Resolve(Credentials{PID: 123, UID: 501, GID: 21}); ok {
		t.Fatal("UID-only match must not resolve a subject")
	}
}

func TestResolverRejectsAmbiguousIdentity(t *testing.T) {
	_, err := NewResolver([]Mapping{
		{SubjectID: "operator.primary", SubjectKind: "operator", UID: 501, GID: 20},
		{SubjectID: "service.other", SubjectKind: "service", UID: 501, GID: 20},
	})
	if err == nil {
		t.Fatal("expected ambiguous identity error")
	}
}

func TestSubjectFromContextFailsClosedWhenPeerIsUnmapped(t *testing.T) {
	ctx := context.WithValue(context.Background(), contextKey{}, contextResult{
		peer: Peer{Credentials: Credentials{PID: 123, UID: 501, GID: 20}},
	})
	if _, err := SubjectFromContext(ctx); !errors.Is(err, ErrSubjectUnmapped) {
		t.Fatalf("error = %v, want ErrSubjectUnmapped", err)
	}
}

func TestCredentialsRejectNonSocketConnection(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()
	if _, err := CredentialsFromConn(server); err == nil {
		t.Fatal("expected non-socket connection to be rejected")
	}
}
