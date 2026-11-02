package server

import (
	"io"
	"net/http"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/foundationaudit"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/peerauth"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/stavproducer"
)

// The bridge accepts a bounded native lifecycle fact, never a caller supplied
// event, actor, reason, target or producer. Kernel host ownership is mandatory.
func (s *Server) handleFoundationAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "request.method_not_allowed", "method not allowed")
		return
	}
	peer, err := peerauth.PeerFromContext(r.Context())
	if err != nil || !s.isHostOwner(peer) {
		writeError(w, 403, "foundation_audit.owner_required", "target-host owner required")
		return
	}
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 65536))
	var request foundationaudit.Request
	if err != nil || foundationaudit.Decode(data, &request) != nil || request.Validate() != nil || request.TOPSID != s.config.TOPS.ID || request.Scope != s.config.Mode {
		writeError(w, 400, "foundation_audit.invalid", "invalid foundational lifecycle request")
		return
	}
	candidate, err := foundationaudit.Candidate(request, peer.Credentials.UID)
	if err != nil {
		writeError(w, 400, "foundation_audit.invalid", "invalid foundational lifecycle candidate")
		return
	}
	kind := stavproducer.FoundationIntent
	if request.Phase == "outcome" {
		kind = stavproducer.FoundationOutcome
	}
	if s.audit == nil {
		writeError(w, 503, "foundation_audit.unavailable", "STAV append authority unavailable")
		return
	}
	receipt, err := s.audit.Submit(r.Context(), stavproducer.Record{
		Kind: kind, RequestID: request.RequestID, CorrelationID: request.CorrelationID,
		Actor: candidate.Actor.Principal, Authentication: candidate.Actor.Authentication,
		Target: candidate.Operation.Target, Outcome: candidate.Result.Outcome,
		Configuration: candidate.Configuration, TROG: candidate.Topology.TROG, Classification: candidate.Redaction.Classification,
	})
	response := foundationaudit.Response{Schema: foundationaudit.Schema, Phase: request.Phase, RequestDigest: foundationaudit.Hash(request), Receipt: receipt}
	// Independently check the sink, including injected implementations. A digest
	// supplied by an administrator is never a substitute for the actual receipt.
	if err != nil || response.Validate(request, peer.Credentials.UID) != nil {
		writeError(w, 503, "foundation_audit.unavailable", "exact committed STAV receipt unavailable")
		return
	}
	writeJSON(w, http.StatusOK, response)
}
