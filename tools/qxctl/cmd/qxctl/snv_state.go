package main

import (
	"encoding/json"
	"fmt"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/snvstate"
)

func snvProposal(raw json.RawMessage, inst knowledgeengine.Installation) (json.RawMessage, json.RawMessage, error) {
	m, e := snvObject(raw)
	if e != nil {
		return nil, nil, e
	}
	if len(m) != 5 || m["protocol"] != snvProposalProtocol {
		return nil, nil, fmt.Errorf("exact public SNV proposal capsule required")
	}
	claimed, ok := m["digest"].(string)
	if !ok {
		return nil, nil, fmt.Errorf("SNV proposal digest required")
	}
	delete(m, "digest")
	actual, e := knowledgeengine.SCVDigest(m)
	if e != nil || claimed != actual {
		return nil, nil, fmt.Errorf("SNV proposal changed")
	}
	expected, e := knowledgeengine.SCVCanonical(inst)
	if e != nil {
		return nil, nil, e
	}
	supplied, e := knowledgeengine.SCVCanonical(m["installation"])
	if e != nil || !snvSame(json.RawMessage(expected), json.RawMessage(supplied)) {
		return nil, nil, fmt.Errorf("SNV proposal installation differs")
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
		return fmt.Errorf("SNV request scope or captured prior head differs")
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
			output, err = snvStoreResult(spec.path, tx, o.operationID, nil)
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
			if !ok || a.Installation != inst {
				return fmt.Errorf("SNV recovery needs exact original installation and input")
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
			return fmt.Errorf("SNV operation identity required")
		}
		if prior, ok := tx.Attempt(id); ok && prior.Status == "committed" {
			if prior.Installation != inst || !snvSame(prior.Input, input) || !snvSame(prior.Plan, plan) {
				return fmt.Errorf("SNV operation binds another committed intent")
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
		decision, err := authorizeKnowledgeRequest(o.topsID, prepared.CorrelationID, "symphony.snv.view.select", snvstate.Resource(o.topsID, o.viewID))
		if err != nil {
			return fmt.Errorf("SNV intent retained; authorization refused: %w", err)
		}
		authorization, err := knowledgeengine.SCVCanonical(decision)
		if err != nil {
			return err
		}
		guard := func() error {
			current, e := knowledgeengine.InspectSNV(o.prefix, o.version, "snv")
			if e != nil || current != inst {
				return fmt.Errorf("SNV owner installation changed")
			}
			if e = snvProveBundle(o, input); e != nil {
				return e
			}
			replayed, e := snvInvoke(o, "snv", "snv_state_reduce", map[string]any{"protocol": "symphony.snv.state-reduce-input.v1", "input": input, "plan": plan})
			if e != nil || !snvSame(replayed, transition) {
				return fmt.Errorf("SNV original input replay changed")
			}
			if e = knowledgeAuthorizationFreshness(decision)(); e != nil {
				return e
			}
			live, e := authorizeKnowledgeRequest(o.topsID, prepared.CorrelationID, "symphony.snv.view.select", snvstate.Resource(o.topsID, o.viewID))
			if e != nil {
				return e
			}
			if live.PolicyDigest != decision.PolicyDigest || live.ConfigDigest != decision.ConfigDigest || live.Subject != decision.Subject || live.Capability == nil || decision.Capability == nil || live.Capability.GrantID != decision.Capability.GrantID {
				return fmt.Errorf("SNV selection permission or policy changed")
			}
			current, e = knowledgeengine.InspectSNV(o.prefix, o.version, "snv")
			if e != nil || current != inst {
				return fmt.Errorf("SNV installation changed during permission recheck")
			}
			return knowledgeAuthorizationFreshness(decision)()
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
	return printIndentedJSON(output)
}
