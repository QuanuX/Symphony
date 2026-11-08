package server

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	stav "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	use "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/cloudflareuse"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/model"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/peerauth"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/stavproducer"
	"io"
	"net/http"
	"time"
)

// The HTTP request upgrades to bounded frames on this same authenticated Unix
// connection. The policy snapshot remains pinned through native cleanup. The
// reply is connection-local admission, never a reusable local-v1 capability.
func (s *Server) handleCloudflareUse(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeError(w, 405, "cloudflare_use.method", "POST required")
		return
	}
	peer, e := peerauth.PeerFromContext(r.Context())
	if e != nil || !peer.Mapped || peer.Credentials.UID == 0 {
		writeError(w, 403, "cloudflare_use.identity", "mapped non-root peer required")
		return
	}
	b, e := io.ReadAll(http.MaxBytesReader(w, r.Body, 32768))
	var in use.Request
	if e != nil || use.Decode(b, &in) != nil || in.Validate(s.now()) != nil || in.TOPSID != s.config.TOPS.ID {
		writeError(w, 400, "cloudflare_use.invalid", "invalid use binding")
		return
	}
	if s.audit == nil || s.policy == nil {
		writeError(w, 503, "cloudflare_use.unavailable", "authority unavailable")
		return
	}
	now := s.now().UTC()
	ctx, cancel := context.WithDeadline(r.Context(), in.Deadline())
	defer cancel()
	auth := model.AuthorizationRequest{Schema: "symphony.ssiag.authorization-request.v1", RequestID: in.RequestID, CorrelationID: in.RequestID, Operation: "symphony.sov.cloudflare." + in.Operation, Resource: in.Resource(), Audience: "sov-cloudflare", Scope: "tops:" + in.TOPSID, RequestedAt: now, RequestedExpiresAt: in.Deadline().UTC()}
	upgraded := false
	e = s.policy.WithDecision(ctx, peer.Subject, auth, func(d model.AuthorizationDecision) error {
		if d.Effect != "allow" {
			writeError(w, 403, "cloudflare_use.denied", "exact permission required")
			return nil
		}
		target := stav.SafeReference{ID: in.Resource(), Kind: "symphony.sov.cloudflare-resource"}
		record := stavproducer.Record{Kind: stavproducer.CloudflareUseIntent, RequestID: in.RequestID, CorrelationID: in.RequestID, Actor: stav.SafeReference{ID: peer.Subject.ID, Kind: peer.Subject.Kind}, Authentication: stav.Authentication{MethodID: "symphony.ssiag.local-peer", State: "identified"}, Target: target, Outcome: "allowed", Configuration: stav.Configuration{PreviousDigest: in.PlanDigest, NewDigest: use.Hash(in), State: "digests"}, TROG: stav.TROG{State: "not_applicable", ReasonCode: "symphony.stav.trog.not-applicable"}, Classification: "administrative_metadata"}
		receipt, err := s.audit.Submit(ctx, record)
		if err != nil || !cloudflareReceipt(record, receipt, in.TOPSID) {
			writeError(w, 503, "cloudflare_use.audit", "committed audit required")
			return nil
		}
		hj, ok := w.(http.Hijacker)
		if !ok {
			return fmt.Errorf("upgrade unavailable")
		}
		conn, rw, err := hj.Hijack()
		if err != nil {
			return err
		}
		upgraded = true
		defer conn.Close()
		deadline := in.Deadline()
		if d.ExpiresAt == nil || !d.ExpiresAt.After(s.now()) {
			return fmt.Errorf("expired")
		}
		if d.ExpiresAt.Before(deadline) {
			deadline = *d.ExpiresAt
		}
		if err = conn.SetDeadline(deadline); err != nil {
			return err
		}
		stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
		defer stop()
		if _, err = rw.WriteString("HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: symphony-cloudflare-use-v1\r\n\r\n"); err != nil {
			return err
		}
		grant := map[string]any{"protocol": use.Protocol, "request_digest": use.Hash(in), "uid": fmt.Sprint(peer.Credentials.UID), "deadline_unix_ms": fmt.Sprint(deadline.UnixMilli()), "policy_digest": d.PolicyDigest, "receipt": receipt}
		if err = writeUseFrame(rw, grant); err != nil {
			return err
		}
		outcome := "unknown"
		raw, err := readUseFrame(rw.Reader)
		var completion use.Completion
		if err == nil && use.Decode(raw, &completion) == nil && completion.Protocol == use.Protocol && completion.RequestDigest == use.Hash(in) && (completion.Outcome == "completed" || completion.Outcome == "failed" || completion.Outcome == "unknown") {
			outcome = completion.Outcome
		}
		// A lost native reply is still an auditable uncertain use. It grants no replay.
		record.Kind = stavproducer.CloudflareUseOutcome
		record.RequestID = use.OutcomeID(in)
		record.Outcome = map[string]string{"completed": "succeeded", "failed": "failed", "unknown": "unavailable"}[outcome]
		auditCtx, done := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer done()
		receipt, err = s.audit.Submit(auditCtx, record)
		if err != nil || !cloudflareReceipt(record, receipt, in.TOPSID) {
			return fmt.Errorf("outcome audit unavailable")
		}
		return writeUseFrame(rw, map[string]any{"protocol": use.Protocol, "request_digest": use.Hash(in), "outcome": outcome, "receipt": receipt})
	})
	if e != nil && !upgraded {
		writeError(w, 503, "cloudflare_use.unavailable", "use admission unavailable")
	}
}
func cloudflareReceipt(record stavproducer.Record, r stav.Receipt, tops string) bool {
	d, e := stavproducer.CandidateDigest(tops, record)
	return e == nil && r.Validate() == nil && r.Disposition == "committed" && r.RequestID == record.RequestID && r.TOPSID == tops && r.CandidateDigest == d
}
func writeUseFrame(w *bufio.ReadWriter, v any) error {
	b, e := json.Marshal(v)
	if e != nil || len(b) > 32768 {
		return fmt.Errorf("frame")
	}
	var h [4]byte
	binary.BigEndian.PutUint32(h[:], uint32(len(b)))
	if _, e = w.Write(h[:]); e != nil {
		return e
	}
	if _, e = w.Write(b); e != nil {
		return e
	}
	return w.Flush()
}
func readUseFrame(r io.Reader) ([]byte, error) {
	var h [4]byte
	if _, e := io.ReadFull(r, h[:]); e != nil {
		return nil, e
	}
	n := binary.BigEndian.Uint32(h[:])
	if n == 0 || n > 32768 {
		return nil, fmt.Errorf("frame")
	}
	b := make([]byte, n)
	_, e := io.ReadFull(r, b)
	return b, e
}
