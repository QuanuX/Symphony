package server

import (
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/connectionaudit"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/peerauth"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/stavproducer"
	"io"
	"net/http"
)

// Records caller-attributed dispatch intent only. This receipt never replaces
// protected profile/policy admission or target-host authorization.
func (s *Server) handleTransportConnectionAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "request.method_not_allowed", "method not allowed")
		return
	}
	peer, e := peerauth.PeerFromContext(r.Context())
	if e != nil || s.config.Mode != "system" || peer.Credentials.UID == 0 {
		writeError(w, 403, "connection_audit.user_required", "authenticated user required")
		return
	}
	raw, e := io.ReadAll(http.MaxBytesReader(w, r.Body, 65536))
	var input connectionaudit.Request
	if e != nil || connectionaudit.Decode(raw, &input) != nil || input.Validate() != nil || input.TOPSID != s.config.TOPS.ID {
		writeError(w, 400, "connection_audit.invalid", "invalid binding")
		return
	}
	c, e := connectionaudit.Candidate(input, peer.Credentials.UID)
	if e != nil || s.audit == nil {
		writeError(w, 503, "connection_audit.unavailable", "audit unavailable")
		return
	}
	receipt, e := s.audit.Submit(r.Context(), stavproducer.Record{Kind: stavproducer.TransportConnectionIntent, RequestID: input.RequestID, CorrelationID: input.RequestID, Actor: c.Actor.Principal, Authentication: c.Actor.Authentication, Target: c.Operation.Target, Outcome: c.Result.Outcome, Configuration: c.Configuration, TROG: c.Topology.TROG, Classification: c.Redaction.Classification})
	response := connectionaudit.Response{Protocol: connectionaudit.Protocol, RequestDigest: connectionaudit.Hash(input), Receipt: receipt}
	if e != nil || response.ValidateFor(input, peer.Credentials.UID) != nil {
		writeError(w, 503, "connection_audit.unavailable", "committed audit unavailable")
		return
	}
	writeJSON(w, 200, response)
}
