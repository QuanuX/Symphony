package stavprotocol

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const (
	candidateDigestDomain = "SYMPHONY-STAV-CANDIDATE-V1"
	genesisDigestDomain   = "SYMPHONY-STAV-GENESIS-V1"
	eventDigestDomain     = "SYMPHONY-STAV-EVENT-V1"
)

// CandidateDigest returns the domain-separated digest of a valid candidate.
func CandidateDigest(v Candidate) (string, error) {
	b, err := EncodeCandidate(v)
	if err != nil {
		return "", err
	}
	return digestDomain(candidateDigestDomain, b), nil
}

// EventDigest returns the domain-separated digest of a valid canonical event.
func EventDigest(v Event) (string, error) {
	b, err := EncodeEvent(v)
	if err != nil {
		return "", err
	}
	return digestDomain(eventDigestDomain, b), nil
}

// CandidateFromEvent reconstructs the exact producer-proposed candidate
// portion of an authority-assigned event for durable idempotency recovery.
func CandidateFromEvent(v Event) (Candidate, error) {
	if err := v.Validate(); err != nil {
		return Candidate{}, err
	}
	return Candidate{
		Actor: CandidateActor{
			Authentication: v.Actor.Authentication,
			Principal:      v.Actor.Principal,
		},
		Configuration: v.Configuration,
		Correlation:   v.Correlation,
		Operation:     v.Operation,
		Redaction:     v.Redaction,
		Result:        v.Result,
		Schema:        SchemaCandidate,
		Topology:      v.Topology,
	}, nil
}

// GenesisDigest derives the per-TOPS predecessor value for the first event.
func GenesisDigest(topsID string) (string, error) {
	if err := ValidateTOPSID(topsID); err != nil {
		return "", err
	}
	return digestDomain(genesisDigestDomain, []byte(topsID)), nil
}

func digestDomain(domain string, payload []byte) string {
	h := sha256.New()
	_, _ = h.Write([]byte(domain))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write(payload)
	return fmt.Sprintf("sha256:%s", hex.EncodeToString(h.Sum(nil)))
}
