//go:build darwin || linux

package store

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	c "github.com/QuanuX/Symphony/tools/sov-transport-profile-admin/contract"
	a "github.com/QuanuX/Symphony/tools/sov-transport-profile-admin/internal/transportaudit"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const pendingProtocol = "symphony.sov.transport-profile-intent.v1"

var errPending = errors.New("transport_profile.recovery_required")
var errCAS = errors.New("transport_profile.stale_state")

type Submit func(context.Context, a.Request) (a.Response, error)
type Store struct {
	dir, profiles, history, attempts, packages *directory
	scope                                      c.Scope
	administrator                              string
	audit                                      Submit
	now                                        func() time.Time
	fault                                      func(string) error // Private test seam; never populated by process input.
}
type scopeRecord struct {
	Protocol string  `json:"protocol"`
	Scope    c.Scope `json:"scope"`
}
type pending struct {
	IntentRequestID  string      `json:"intent_request_id"`
	OutcomeRequestID string      `json:"outcome_request_id"`
	Protocol         string      `json:"protocol"`
	Plan             c.Plan      `json:"plan"`
	Prior            *c.Snapshot `json:"prior"`
	Stage            string      `json:"stage"`
	Resolution       string      `json:"resolution"`
	Intent           *a.Response `json:"intent"`
	Outcome          *a.Response `json:"outcome"`
	Digest           string      `json:"digest"`
}

func Root() string {
	if runtime.GOOS == "darwin" {
		return "/Library/Application Support/Symphony/sov/transport-profiles"
	}
	return "/var/lib/symphony/sov/transport-profiles"
}
func PackageRoot() string {
	if runtime.GOOS == "darwin" {
		return "/Library/Application Support/Symphony/transport-packages"
	}
	return "/opt/symphony/transport-packages"
}
func HelperPath() string {
	if runtime.GOOS == "darwin" {
		return "/Library/Application Support/Symphony/libexec/sov-transport-profile-admin/0.1.0-dev/sov-transport-profile-admin"
	}
	return "/opt/symphony/libexec/sov-transport-profile-admin/0.1.0-dev/sov-transport-profile-admin"
}
func Open(scope c.Scope, audit Submit) (*Store, error) {
	if !c.ValidScope(scope) {
		return nil, c.ErrBinding
	}
	d, e := openProtected(filepath.Join(Root(), scope.HostID, scope.TOPSID))
	if e != nil {
		return nil, e
	}
	s := &Store{dir: d, scope: scope, audit: audit, now: time.Now}
	success := false
	defer func() {
		if !success {
			s.Close()
		}
	}()
	if s.administrator, e = AdministratorIdentity(); e != nil {
		return nil, e
	}
	raw, e := d.read("scope.json", c.MaxDocumentBytes)
	var r scopeRecord
	if e != nil || c.Decode(raw, &r) != nil || r.Protocol != "symphony.sov.transport-profile-scope.v1" || r.Scope != scope {
		return nil, errTrust
	}
	if s.profiles, e = d.child("profiles"); e != nil {
		return nil, e
	}
	if s.history, e = d.child("history"); e != nil {
		return nil, e
	}
	if s.attempts, e = d.child("attempts"); e != nil {
		return nil, e
	}
	// A disabled draft can exist before a separately installed transport package.
	s.packages, _ = openProtected(PackageRoot())
	success = true
	return s, nil
}
func (s *Store) Close() {
	for _, d := range []*directory{s.dir, s.profiles, s.history, s.attempts, s.packages} {
		d.close()
	}
}
func (s *Store) checkpoint(stage string) error {
	if s.fault != nil {
		return s.fault(stage)
	}
	return nil
}
func (s *Store) current(id string) (*c.Snapshot, error) {
	if !c.ValidUUID(id) {
		return nil, c.ErrBinding
	}
	raw, e := s.profiles.read(id+".json", c.MaxDocumentBytes)
	if errors.Is(e, os.ErrNotExist) {
		return nil, nil
	}
	var v c.Snapshot
	if e != nil || c.Decode(raw, &v) != nil || c.ValidateSnapshot(v) != nil || v.Profile.Scope != s.scope || v.Profile.ProfileID != id {
		return nil, errTrust
	}
	return &v, nil
}
func (s *Store) readPending(name string, dir *directory) (*pending, error) {
	raw, e := dir.read(name, c.MaxDocumentBytes)
	if errors.Is(e, os.ErrNotExist) {
		return nil, nil
	}
	var p pending
	if e != nil || c.Decode(raw, &p) != nil || p.Protocol != pendingProtocol || !c.ValidRequestID(p.IntentRequestID) || !c.ValidRequestID(p.OutcomeRequestID) || p.IntentRequestID == p.OutcomeRequestID || p.Digest != c.Digest(pendingProtocol, p) || p.Plan.Scope != s.scope || c.ValidatePlan(p.Plan, p.Prior, s.now(), true) != nil {
		return nil, errTrust
	}
	if p.Stage != "prepared" && p.Stage != "admitted" && p.Stage != "published" && p.Stage != "settled" {
		return nil, errTrust
	}
	if p.Resolution != "pending" && p.Resolution != "published" && p.Resolution != "aborted" {
		return nil, errTrust
	}
	if p.Intent != nil && p.Intent.Validate(auditRequest(&p, "intent")) != nil {
		return nil, errTrust
	}
	if p.Outcome != nil && p.Outcome.Validate(auditRequest(&p, "outcome")) != nil {
		return nil, errTrust
	}
	switch p.Stage {
	case "prepared":
		if p.Intent != nil || p.Outcome != nil || p.Resolution != "pending" {
			return nil, errTrust
		}
	case "admitted":
		if p.Intent == nil || p.Outcome != nil || (p.Resolution != "pending" && p.Resolution != "aborted") {
			return nil, errTrust
		}
	case "published":
		if p.Intent == nil || p.Outcome != nil || p.Resolution != "published" {
			return nil, errTrust
		}
	case "settled":
		if p.Intent == nil || p.Outcome == nil || p.Resolution == "pending" {
			return nil, errTrust
		}
	}
	return &p, nil
}
func (s *Store) persist(p *pending) error {
	p.Digest = c.Digest(pendingProtocol, *p)
	raw, e := c.Canonical(p)
	if e != nil || len(raw) > c.MaxDocumentBytes {
		return c.ErrDocument
	}
	return s.dir.write("pending.json", raw)
}
func auditRequest(p *pending, phase string) a.Request {
	prior := c.Digest("symphony.sov.transport-profile-absence.v1", map[string]any{"scope": p.Plan.Scope, "profile_id": p.Plan.ProfileID})
	if p.Prior != nil {
		prior = p.Prior.Digest
	}
	resolution, observed := "pending", ""
	if phase == "outcome" {
		resolution = p.Resolution
		observed = prior
		if resolution == "published" {
			observed = p.Plan.Desired.Digest
		}
	}
	id := p.IntentRequestID
	if phase == "outcome" {
		id = p.OutcomeRequestID
	}
	return a.Request{Protocol: a.Protocol, RequestID: id, OperationID: p.Plan.OperationID, HostID: p.Plan.Scope.HostID, TOPSID: p.Plan.Scope.TOPSID, ProfileID: p.Plan.ProfileID, Action: p.Plan.Change.Action, PlanDigest: p.Plan.Digest, PriorDigest: prior, DesiredDigest: p.Plan.Desired.Digest, Phase: phase, Resolution: resolution, ObservedDigest: observed}
}
func (s *Store) commitAudit(ctx context.Context, p *pending, phase string) (*a.Response, error) {
	if s.audit == nil || ctx.Err() != nil {
		return nil, errors.New("transport_profile.audit_unavailable")
	}
	request := auditRequest(p, phase)
	response, e := s.audit(ctx, request)
	if e != nil || response.Validate(request) != nil {
		return nil, errors.New("transport_profile.audit_unavailable")
	}
	return &response, nil
}
func (s *Store) uniqueName(desired c.Snapshot) error {
	names, e := s.profiles.names()
	if e != nil {
		return e
	}
	for _, n := range names {
		if strings.HasPrefix(n, ".stage-") {
			continue
		}
		if !strings.HasSuffix(n, ".json") {
			return errTrust
		}
		id := strings.TrimSuffix(n, ".json")
		snapshot, e := s.current(id)
		if e != nil || snapshot == nil {
			return errTrust
		}
		if id != desired.Profile.ProfileID && snapshot.Profile.Configuration.Name == desired.Profile.Configuration.Name {
			return errors.New("transport_profile.name_reserved")
		}
	}
	return nil
}
func (s *Store) publish(p *pending) error {
	if e := s.verifyTrust(p.Plan.Desired); e != nil {
		return e
	}
	if e := s.uniqueName(p.Plan.Desired); e != nil {
		return e
	}
	raw, e := c.Canonical(p.Plan.Desired)
	if e != nil {
		return e
	}
	name := p.Plan.ProfileID + "-" + p.Plan.Desired.Profile.Generation + ".json"
	retained, e := s.history.read(name, c.MaxDocumentBytes)
	if e == nil {
		var existing c.Snapshot
		if c.Decode(retained, &existing) != nil || !c.Same(existing, p.Plan.Desired) {
			return errTrust
		}
	} else if errors.Is(e, os.ErrNotExist) {
		if e = s.history.write(name, raw); e != nil {
			return e
		}
	} else {
		return errTrust
	}
	if e = s.profiles.write(p.Plan.ProfileID+".json", raw); e != nil {
		return e
	}
	return s.checkpoint("state_published")
}
func (s *Store) settle(ctx context.Context, p *pending, resolution string) error {
	if p.Stage == "settled" {
		if (resolution == "abort") != (p.Resolution == "aborted") {
			return errCAS
		}
		current, e := s.current(p.Plan.ProfileID)
		if e != nil {
			return e
		}
		expected := p.Prior
		if p.Resolution == "published" {
			if e = s.verifyTrust(p.Plan.Desired); e != nil {
				return e
			}
			expected = &p.Plan.Desired
		}
		if !c.Same(current, expected) {
			return errCAS
		}
		return s.archive(p)
	}
	if p.Intent == nil {
		r, e := s.commitAudit(ctx, p, "intent")
		if e != nil {
			return e
		}
		if e = s.checkpoint("audit_intent_committed"); e != nil {
			return e
		}
		p.Intent = r
		p.Stage = "admitted"
		if e = s.persist(p); e != nil {
			return e
		}
	}
	current, e := s.current(p.Plan.ProfileID)
	if e != nil {
		return e
	}
	if resolution == "abort" {
		if !c.Same(current, p.Prior) || p.Stage == "published" || p.Resolution == "published" {
			return errCAS
		}
		p.Resolution = "aborted"
	} else {
		if e = s.verifyTrust(p.Plan.Desired); e != nil {
			return e
		}
		if p.Resolution == "aborted" {
			return errCAS
		}
		if c.Same(current, p.Prior) {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if e = s.publish(p); e != nil {
				return e
			}
		} else if !c.Same(current, &p.Plan.Desired) {
			return errCAS
		}
		p.Stage = "published"
		p.Resolution = "published"
	}
	// Persist resolution before submitting outcome so an explicit recovery
	// reuses the same exact candidate after a lost audit response.
	if e = s.persist(p); e != nil {
		return e
	}
	r, e := s.commitAudit(ctx, p, "outcome")
	if e != nil {
		return e
	}
	if e = s.checkpoint("audit_outcome_committed"); e != nil {
		return e
	}
	p.Outcome = r
	p.Stage = "settled"
	if e = s.persist(p); e != nil {
		return e
	}
	return s.archive(p)
}
func (s *Store) archive(p *pending) error {
	raw, e := c.Canonical(p)
	if e != nil {
		return e
	}
	existing, readErr := s.attempts.read(p.Plan.OperationID+".json", c.MaxDocumentBytes)
	if readErr == nil {
		if string(existing) != string(raw) {
			return errTrust
		}
	} else if errors.Is(readErr, os.ErrNotExist) {
		if e = s.attempts.write(p.Plan.OperationID+".json", raw); e != nil {
			return e
		}
	} else {
		return errTrust
	}
	if e = s.checkpoint("outcome_archived"); e != nil {
		return e
	}
	return s.dir.remove("pending.json")
}
func (s *Store) apply(ctx context.Context, plan c.Plan) error {
	p, e := s.readPending("pending.json", s.dir)
	if e != nil {
		return e
	}
	if p != nil {
		return errPending
	}
	priorAttempt, e := s.readPending(plan.OperationID+".json", s.attempts)
	if e != nil {
		return e
	}
	if priorAttempt != nil {
		return errors.New("transport_profile.attempt_exists")
	}
	current, e := s.current(plan.ProfileID)
	if e != nil {
		return e
	}
	if plan.AdministratorDigest != s.administrator || c.ValidatePlan(plan, current, s.now(), false) != nil {
		return errCAS
	}
	if e = s.uniqueName(plan.Desired); e != nil {
		return e
	}
	if e = s.verifyTrust(plan.Desired); e != nil {
		return e
	}
	intentID, e := randomID()
	if e != nil {
		return e
	}
	outcomeID, e := randomID()
	if e != nil {
		return e
	}
	p = &pending{IntentRequestID: intentID, OutcomeRequestID: outcomeID, Protocol: pendingProtocol, Plan: plan, Prior: current, Stage: "prepared", Resolution: "pending"}
	if e = s.persist(p); e != nil {
		return e
	}
	if e = s.checkpoint("intent_prepared"); e != nil {
		return e
	}
	return s.settle(ctx, p, "complete")
}
func (s *Store) recover(ctx context.Context, r c.Recovery) error {
	p, e := s.readPending("pending.json", s.dir)
	if e != nil {
		return e
	}
	if p == nil || p.Plan.AdministratorDigest != s.administrator || p.Plan.OperationID != r.OperationID || p.Plan.Digest != r.PlanDigest || p.Digest != r.ExpectedPendingDigest {
		return errCAS
	}
	return s.settle(ctx, p, r.Action)
}
func summary(p *pending, isPending bool) *c.Attempt {
	if p == nil {
		return nil
	}
	r := &c.Attempt{OperationID: p.Plan.OperationID, PlanDigest: p.Plan.Digest, Stage: p.Stage, Outcome: p.Resolution}
	if isPending {
		r.PendingDigest = &p.Digest
	}
	if p.Intent != nil {
		d := p.Intent.Receipt.CandidateDigest
		r.AuditIntentDigest = &d
	}
	if p.Outcome != nil {
		d := p.Outcome.Receipt.CandidateDigest
		r.AuditOutcomeDigest = &d
	}
	return r
}
func (s *Store) Handle(ctx context.Context, r c.Request) c.Result {
	result := c.Result{Protocol: c.ResultProtocol, RequestID: r.RequestID, Operation: r.Operation, Scope: r.Scope, Profiles: []c.Snapshot{}, Code: "transport_profile.rejected", NextAction: "inspect"}
	if c.ValidateRequest(r) != nil || r.Scope != s.scope {
		return result
	}
	mutating := r.Operation == "apply" || r.Operation == "recover"
	if mutating && os.Geteuid() != 0 {
		result.Code = "transport_profile.root_required"
		return result
	}
	unlock, e := s.dir.lock(mutating)
	if e != nil {
		result.Code = safeCode(e)
		return result
	}
	defer unlock()
	p, e := s.readPending("pending.json", s.dir)
	if e != nil {
		result.Code = safeCode(e)
		return result
	}
	result.Attempt = summary(p, true)
	switch r.Operation {
	case "apply":
		e = s.apply(ctx, *r.Plan)
	case "recover":
		e = s.recover(ctx, *r.Recovery)
	case "inspect":
		result.Snapshot, e = s.current(*r.ProfileID)
	case "list":
		var names []string
		names, e = s.profiles.names()
		if e == nil {
			for _, n := range names {
				if strings.HasPrefix(n, ".stage-") {
					continue
				}
				if !strings.HasSuffix(n, ".json") {
					e = errTrust
					break
				}
				var v *c.Snapshot
				v, e = s.current(strings.TrimSuffix(n, ".json"))
				if e != nil || v == nil {
					e = errTrust
					break
				}
				result.Profiles = append(result.Profiles, *v)
			}
		}
	case "plan":
		if p != nil {
			e = errPending
			break
		}
		var current *c.Snapshot
		current, e = s.current(*r.ProfileID)
		if e == nil {
			var plan c.Plan
			plan, e = c.MakePlan(r.Scope, *r.ProfileID, *r.OperationID, s.administrator, current, *r.Change, s.now())
			if e == nil {
				e = s.uniqueName(plan.Desired)
				result.Plan = &plan
			}
		}
	case "status":
		if p == nil || p.Plan.OperationID != *r.OperationID {
			var prior *pending
			prior, e = s.readPending(*r.OperationID+".json", s.attempts)
			result.Attempt = summary(prior, false)
		}
	}
	if mutating {
		p, readErr := s.readPending("pending.json", s.dir)
		if readErr != nil {
			e = readErr
		}
		result.Attempt = summary(p, true)
		if e == nil {
			id := r.Plan
			op := ""
			if id != nil {
				op = id.OperationID
			} else {
				op = r.Recovery.OperationID
			}
			prior, readErr := s.readPending(op+".json", s.attempts)
			if readErr != nil {
				e = readErr
			} else {
				result.Attempt = summary(prior, false)
			}
		}
	}
	if e != nil {
		result.Code = safeCode(e)
		if result.Attempt != nil && result.Attempt.PendingDigest != nil {
			result.NextAction = "recover"
		}
		return result
	}
	result.Completed = true
	result.Code = "transport_profile.completed"
	result.NextAction = "none"
	if result.Attempt != nil && result.Attempt.PendingDigest != nil {
		result.Code = "transport_profile.recovery_required"
		result.NextAction = "recover"
	}
	return result
}
func safeCode(e error) string {
	if e == nil {
		return "transport_profile.completed"
	}
	switch e.Error() {
	case "transport_profile.invalid_binding", "transport_profile.invalid_document", "transport_profile.generation_exhausted", "transport_profile.untrusted_store", "transport_profile.recovery_required", "transport_profile.stale_state", "transport_profile.name_reserved", "transport_profile.busy", "transport_profile.attempt_exists", "transport_profile.audit_unavailable":
		return e.Error()
	}
	return "transport_profile.storage_unavailable"
}

func AdministratorIdentity() (string, error) {
	d, e := openProtected(filepath.Dir(HelperPath()))
	if e != nil {
		return "", e
	}
	defer d.close()
	digest, _, executable, e := d.digestFile(filepath.Base(HelperPath()))
	if e != nil || !executable {
		return "", errTrust
	}
	return digest, nil
}

func randomID() (string, error) {
	var b [16]byte
	if _, e := rand.Read(b[:]); e != nil {
		return "", e
	}
	b[6] = b[6]&15 | 0x40
	b[8] = b[8]&63 | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}
