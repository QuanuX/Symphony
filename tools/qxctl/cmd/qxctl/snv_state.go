package main

import (
	"context"
	"encoding/json"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/ssiagclient"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/snvstate"
)

func snvProposal(raw json.RawMessage, inst knowledgeengine.Installation) (json.RawMessage, json.RawMessage, error) {
	m, e := snvObject(raw)
	if e != nil {
		return nil, nil, e
	}
	if len(m) != 5 || m["protocol"] != snvProposalProtocol {
		return nil, nil, snvstate.Refusal("snv.invalid_input", nil)
	}
	claimed, ok := m["digest"].(string)
	if !ok {
		return nil, nil, snvstate.Refusal("snv.invalid_input", nil)
	}
	delete(m, "digest")
	actual, e := knowledgeengine.SCVDigest(m)
	if e != nil || claimed != actual {
		return nil, nil, snvstate.Refusal("snv.intent_conflict", nil)
	}
	expected, e := knowledgeengine.SCVCanonical(inst)
	if e != nil {
		return nil, nil, e
	}
	supplied, e := knowledgeengine.SCVCanonical(m["installation"])
	if e != nil || !snvSame(json.RawMessage(expected), json.RawMessage(supplied)) {
		return nil, nil, snvstate.Refusal("snv.installation_drift", nil)
	}
	input, e := knowledgeengine.SCVCanonical(m["input"])
	if e != nil {
		return nil, nil, e
	}
	plan, e := knowledgeengine.SCVCanonical(m["plan"])
	return input, plan, e
}
func snvCheckView(o snvOptions, tx *snvstate.Transaction, input json.RawMessage) error {
	m, e := snvObject(input)
	if e != nil {
		return e
	}
	view := map[string]any{"tops_id": o.topsID, "view_id": o.viewID}
	if !snvSame(m["view"], view) || !snvSame(m["prior_head"], tx.Current()) || !snvSame(m["expected_state_digest"], tx.Snapshot().StateDigest) {
		return snvstate.Refusal("snv.state_conflict", nil)
	}
	return nil
}
func runSNVState(spec snvLeafSpec, o snvOptions, inst knowledgeengine.Installation) error {
	if spec.path == "state.apply" && ((o.input == "") == (o.operationID == "")) {
		return errUsageOnly
	}
	store, e := snvstate.NewView(o.stateRoot, o.topsID, o.viewID)
	if e != nil {
		return e
	}
	var raw json.RawMessage
	if o.input != "" {
		raw, e = knowledgeengine.ReadPayload(o.input)
		if e != nil {
			return e
		}
	}
	var output json.RawMessage
	withStore := store.WithLock
	if spec.path == "state.status" || spec.path == "state.plan" {
		withStore = store.WithRead
	}
	e = withStore(func(tx *snvstate.Transaction) error {
		if spec.path == "state.status" {
			var err error
			if o.operationID == "" && (o.authorizationOffset != 0 || o.authorizationLimit != 16 || o.expectedJournal != "") {
				return errUsageOnly
			}
			output, err = snvStoreResult(spec.path, tx, o.operationID, nil, snvAuthorizationPage{offset: o.authorizationOffset, limit: o.authorizationLimit, expectedJournal: o.expectedJournal})
			return err
		}
		if spec.path == "state.plan" {
			if err := snvCheckView(o, tx, raw); err != nil {
				return err
			}
			if err := snvProveBundle(o, raw); err != nil {
				return err
			}
			plan, err := snvInvoke(o, "snv", "snv_state_plan", raw)
			if err != nil {
				return err
			}
			output, err = snvSeal(map[string]any{"protocol": snvProposalProtocol, "input": raw, "plan": plan, "installation": inst})
			return err
		}
		var input, plan json.RawMessage
		if o.operationID != "" {
			a, ok := tx.Attempt(o.operationID)
			if !ok {
				return snvstate.Refusal("snv.evidence_missing", nil)
			}
			if a.Installation != inst {
				return snvstate.Refusal("snv.installation_drift", nil)
			}
			input, plan = a.Input, a.Plan
		} else {
			var err error
			input, plan, err = snvProposal(raw, inst)
			if err != nil {
				return err
			}
		}
		body, err := snvObject(input)
		if err != nil {
			return err
		}
		id, ok := body["operation_id"].(string)
		if !ok {
			return snvstate.Refusal("snv.invalid_input", nil)
		}
		if prior, ok := tx.Attempt(id); ok && prior.Status == "committed" {
			if prior.Installation != inst || !snvSame(prior.Input, input) || !snvSame(prior.Plan, plan) {
				return snvstate.Refusal("snv.intent_conflict", nil)
			}
			output, err = snvStoreResult(spec.path, tx, id, prior.Transition)
			return err
		}
		if err = snvCheckView(o, tx, input); err != nil {
			return err
		}
		if err = snvProveBundle(o, input); err != nil {
			return err
		}
		transition, err := snvInvoke(o, "snv", "snv_state_reduce", map[string]any{"protocol": "symphony.snv.state-reduce-input.v1", "input": input, "plan": plan})
		if err != nil {
			return err
		}
		intent, err := snvstate.NewAttempt("selection", input, plan, transition, inst)
		if err != nil {
			return err
		}
		if _, err = tx.Prepare(intent); err != nil {
			return err
		}
		prepared, _ := tx.Attempt(id)
		decision, err := authorizeSNVSelection(o.topsID, prepared.CorrelationID, snvstate.Resource(o.topsID, o.viewID))
		if err != nil {
			return err
		}
		authorization, err := knowledgeengine.SCVCanonical(decision)
		if err != nil {
			return err
		}
		guard := func() error {
			current, e := knowledgeengine.InspectSNV(o.prefix, o.version, "snv")
			if e != nil || current != inst {
				return snvstate.Refusal("snv.installation_drift", e)
			}
			if e = snvProveBundle(o, input); e != nil {
				return e
			}
			replayed, e := snvInvoke(o, "snv", "snv_state_reduce", map[string]any{"protocol": "symphony.snv.state-reduce-input.v1", "input": input, "plan": plan})
			if e != nil {
				return e
			}
			if !snvSame(replayed, transition) {
				return snvstate.Refusal("snv.candidate_drift", nil)
			}
			if e = knowledgeAuthorizationFreshness(decision)(); e != nil {
				return snvstate.Refusal("snv.authority_expired", e)
			}
			live, e := authorizeSNVSelection(o.topsID, prepared.CorrelationID, snvstate.Resource(o.topsID, o.viewID))
			if e != nil {
				return e
			}
			if live.PolicyDigest != decision.PolicyDigest || live.ConfigDigest != decision.ConfigDigest || live.Subject != decision.Subject || live.Capability == nil || decision.Capability == nil || live.Capability.GrantID != decision.Capability.GrantID {
				return snvstate.Refusal("snv.authority_conflict", nil)
			}
			current, e = knowledgeengine.InspectSNV(o.prefix, o.version, "snv")
			if e != nil || current != inst {
				return snvstate.Refusal("snv.installation_drift", e)
			}
			if e := knowledgeAuthorizationFreshness(decision)(); e != nil {
				return snvstate.Refusal("snv.authority_expired", e)
			}
			return nil
		}
		if err = tx.CommitSelection(id, authorization, guard); err != nil {
			return err
		}
		output, err = snvStoreResult(spec.path, tx, id, transition)
		return err
	})
	if e != nil {
		return e
	}
	return snvPrintJSON(output)
}

// This is neutral authenticated SSIAG request/response admission. A returned
// error or malformed decision is unavailable authority, never a policy denial.
// Successful allows use the established exact capability validation unchanged.
func authorizeSNVSelection(topsID, correlationID, resource string) (ssiagclient.AuthorizationDecision, error) {
	if err := stavprotocol.ValidateRequestUUID(correlationID); err != nil {
		return ssiagclient.AuthorizationDecision{}, snvstate.Refusal("snv.invalid_input", err)
	}
	client, err := ssiagclient.NewForTOPS("user", topsID, 4*time.Second)
	if err != nil {
		return ssiagclient.AuthorizationDecision{}, snvstate.Refusal("snv.authority_unavailable", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if _, err := requireSSIAGStatus(ctx, client, topsID, "user"); err != nil {
		return ssiagclient.AuthorizationDecision{}, snvstate.Refusal("snv.authority_unavailable", err)
	}
	requestID, err := randomUUID()
	if err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	now := time.Now().UTC().Truncate(time.Second)
	request := ssiagclient.AuthorizationRequest{Schema: "symphony.ssiag.authorization-request.v1", RequestID: requestID, CorrelationID: correlationID, Operation: "symphony.snv.view.select", Resource: resource, Audience: "qxctl", Scope: "tops:" + topsID, RequestedAt: now, RequestedExpiresAt: now.Add(time.Minute)}
	decision, err := client.Authorize(ctx, request)
	if err != nil {
		return ssiagclient.AuthorizationDecision{}, snvstate.Refusal("snv.authority_unavailable", err)
	}
	if err := snvSelectionDecision(decision, request, topsID); err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	return decision, nil
}
func snvSelectionDecision(decision ssiagclient.AuthorizationDecision, request ssiagclient.AuthorizationRequest, topsID string) error {
	if decision.Effect == "deny" {
		now := time.Now().UTC()
		if decision.Schema == "symphony.ssiag.authorization-decision.v1" && validSessionToken(decision.DecisionID) &&
			decision.RequestID == request.RequestID && decision.CorrelationID == request.CorrelationID && decision.TOPSID == topsID &&
			decision.Target.Operation == request.Operation && decision.Target.Resource == request.Resource && decision.Target.Audience == request.Audience && decision.Target.Scope == request.Scope &&
			validSessionToken(decision.Subject.ID) && validSessionToken(decision.Subject.Kind) && decision.Subject.Authority == "unix_peer_credentials" &&
			validSessionToken(decision.ReasonCode) && validTaggedDigest(decision.PolicyDigest) && validTaggedDigest(decision.ConfigDigest) &&
			!decision.CallerClassUsed && !decision.CanonicalApply && decision.AuthorityBasis == nil && decision.Capability == nil && decision.ExpiresAt == nil &&
			decision.DecidedAt.Location() == time.UTC && !decision.DecidedAt.Before(request.RequestedAt.Add(-30*time.Second)) && !decision.DecidedAt.After(now.Add(30*time.Second)) {
			return snvstate.Refusal("snv.authority_denied", nil)
		}
		return snvstate.Refusal("snv.authority_unavailable", nil)
	}
	if err := validateSessionAuthorization(decision, request, topsID); err != nil {
		return snvstate.Refusal("snv.authority_unavailable", err)
	}
	return nil
}
