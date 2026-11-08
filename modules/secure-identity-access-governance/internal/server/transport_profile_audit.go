package server

import (
	"context"
	"fmt"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/peerauth"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/stavproducer"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/transportaudit"
	"io"
	"net/http"
)

func (s *Server) handleTransportProfileAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "request.method_not_allowed", "method not allowed")
		return
	}
	peer, err := peerauth.PeerFromContext(r.Context())
	if err != nil || s.config.Mode != "system" || peer.Credentials.UID != 0 {
		writeError(w, 403, "transport_audit.root_required", "authenticated system host authority required")
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 65536))
	var input transportaudit.Request
	if err != nil || transportaudit.Decode(raw, &input) != nil || input.Validate() != nil || input.TOPSID != s.config.TOPS.ID {
		writeError(w, 400, "transport_audit.invalid", "invalid profile audit binding")
		return
	}
	response, err := s.commitTransportProfileAudit(r.Context(), peer, input)
	if err != nil {
		writeError(w, 503, "transport_audit.unavailable", "exact committed profile audit unavailable")
		return
	}
	writeJSON(w, http.StatusOK, response)
}

// The handler obtains peer exclusively from the accepted kernel connection.
// Submitted actor/receipt fields are not part of this closed request.
func (s *Server) commitTransportProfileAudit(ctx context.Context, peer peerauth.Peer, input transportaudit.Request) (transportaudit.Response, error) {
	if s.config.Mode != "system" || peer.Credentials.UID != 0 || input.Validate() != nil || input.TOPSID != s.config.TOPS.ID || s.audit == nil {
		return transportaudit.Response{}, fmt.Errorf("transport_audit.unavailable")
	}
	c, err := transportaudit.Candidate(input, peer.Credentials.UID)
	if err != nil {
		return transportaudit.Response{}, err
	}
	kind := stavproducer.TransportProfileIntent
	if input.Phase == "outcome" {
		kind = stavproducer.TransportProfilePublished
		if input.Resolution == "aborted" {
			kind = stavproducer.TransportProfileAborted
		}
	}
	receipt, err := s.audit.Submit(ctx, stavproducer.Record{Kind: kind, RequestID: input.RequestID, CorrelationID: input.OperationID, Actor: c.Actor.Principal, Authentication: c.Actor.Authentication, Target: c.Operation.Target, Outcome: c.Result.Outcome, Configuration: c.Configuration, TROG: c.Topology.TROG, Classification: c.Redaction.Classification})
	response := transportaudit.Response{Protocol: transportaudit.Protocol, RequestDigest: transportaudit.Hash(input), Receipt: receipt}
	if err != nil || response.Validate(input) != nil {
		return transportaudit.Response{}, fmt.Errorf("transport_audit.unavailable")
	}
	return response, nil
}
