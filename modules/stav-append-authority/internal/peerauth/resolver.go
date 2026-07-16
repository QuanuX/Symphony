package peerauth

import (
	"fmt"
	"net"
	"os"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
)

type credentialKey struct {
	uid uint32
	gid uint32
}

type Resolver struct {
	authority stavprotocol.AuthorityGrant
	producers map[credentialKey]stavprotocol.ProducerGrant
	readers   map[credentialKey]stavprotocol.ReaderGrant
}

func NewResolver(authentication stavprotocol.AppendAuthorityAuthentication) (Resolver, error) {
	resolver := Resolver{
		authority: authentication.Authority,
		producers: make(map[credentialKey]stavprotocol.ProducerGrant, len(authentication.Producers)),
		readers:   make(map[credentialKey]stavprotocol.ReaderGrant, len(authentication.Readers)),
	}
	for _, grant := range authentication.Producers {
		key := credentialKey{uid: uint32(grant.UID), gid: uint32(grant.GID)}
		if _, exists := resolver.producers[key]; exists {
			return Resolver{}, fmt.Errorf("stav peer authentication: ambiguous producer grant")
		}
		resolver.producers[key] = grant
	}
	for _, grant := range authentication.Readers {
		key := credentialKey{uid: uint32(grant.UID), gid: uint32(grant.GID)}
		if _, exists := resolver.readers[key]; exists {
			return Resolver{}, fmt.Errorf("stav peer authentication: ambiguous reader grant")
		}
		resolver.readers[key] = grant
	}
	return resolver, nil
}

func (r Resolver) VerifyProcessAuthority() error {
	if uint64(os.Geteuid()) != r.authority.UID || uint64(os.Getegid()) != r.authority.GID {
		return fmt.Errorf("stav peer authentication: process identity does not match configured authority")
	}
	return nil
}

func (r Resolver) VerifyAuthority(conn net.Conn) error {
	credentials, err := CredentialsFromConn(conn)
	if err != nil {
		return err
	}
	if uint64(credentials.UID) != r.authority.UID || uint64(credentials.GID) != r.authority.GID {
		return fmt.Errorf("stav peer authentication: connected endpoint is not the configured authority")
	}
	return nil
}

func (r Resolver) Producer(credentials Credentials) (stavprotocol.ProducerGrant, bool) {
	grant, ok := r.producers[credentialKey{uid: credentials.UID, gid: credentials.GID}]
	return grant, ok
}

func (r Resolver) Reader(credentials Credentials) (stavprotocol.ReaderGrant, bool) {
	grant, ok := r.readers[credentialKey{uid: credentials.UID, gid: credentials.GID}]
	return grant, ok
}

func (r Resolver) Authority() stavprotocol.AuthorityGrant {
	return r.authority
}
