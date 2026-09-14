package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/shvstate"
	"github.com/spf13/cobra"
)

const shvActivationProtocol = "symphony.qxctl.shv-source-activation-result.v1"

type shvActivationOptions struct{ prefix, version, input, topsID, stateRoot, sourceID, operationID string }

func newSHVSourceActivationCommand() *cobra.Command {
	root := structural("activation", fmt.Errorf("activation requires propose, apply, status, recover, schema or template"))
	for _, leaf := range []string{"propose", "apply", "status", "recover"} {
		o := shvActivationOptions{}
		c := &cobra.Command{Use: leaf, Args: usageOnlyArgs, Short: "Administer an exact SHV source transition under authenticated permission", RunE: func(*cobra.Command, []string) error { return runSHVSourceActivation(leaf, o) }}
		c.Flags().StringVar(&o.prefix, "prefix", "", "exact SHV source engine receipt prefix")
		c.Flags().StringVar(&o.version, "version", "", "exact source engine version; no default")
		c.Flags().StringVar(&o.topsID, "tops-id", "", "exact TOPS UUID")
		c.Flags().StringVar(&o.stateRoot, "state-root", "", "explicit owned local state root")
		c.Flags().StringVar(&o.sourceID, "source-id", "", "stable SHV source identity")
		c.Flags().Bool("json", false, "emit complete structured result")
		for _, f := range []string{"prefix", "version", "tops-id", "state-root", "source-id"} {
			_ = c.MarkFlagRequired(f)
		}
		if leaf == "propose" || leaf == "apply" {
			c.Flags().StringVar(&o.input, "input", "", "exact proposal or source plan JSON")
			_ = c.MarkFlagRequired("input")
		}
		if leaf == "status" || leaf == "recover" {
			c.Flags().StringVar(&o.operationID, "operation-id", "", "retained operation identity")
			if leaf == "recover" {
				_ = c.MarkFlagRequired("operation-id")
			}
		}
		c.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		interaction := map[string]string{"propose": "propose", "apply": "apply", "status": "inspect", "recover": "recover"}[leaf]
		s := commandSpec("shv.source.activation."+leaf, featureSHVAdministration, interaction)
		s.Mutability = "read_only"
		native := map[string]string{"propose": "source_plan", "apply": "source_reduce", "status": "source_status", "recover": "source_reduce"}[leaf]
		s.BackendOperationIDs = []string{"engop:symphony:shv-source." + strings.ReplaceAll(native, "_", ".")}
		s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:shv-source-engine", Interaction: interaction})
		s.InputProtocols = []string{}
		output := shvActivationProtocol
		if leaf == "propose" {
			s.Mutability = "proposal_only"
			s.InputProtocols = []string{"symphony.qxctl.shv-source-proposal-input.v1"}
			output = "symphony.shv.source-plan.v1"
		}
		if leaf == "apply" {
			s.InputProtocols = []string{"symphony.shv.source-plan.v1"}
		}
		if leaf == "apply" || leaf == "recover" {
			s.Mutability = "permission_backed_mutation"
			s.AuthorityMode = "target_host_permission"
			s.RecoveryCommandID = stringPointer("qxcmd:symphony:shv.source.activation.recover")
			s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: backendFeatureSSIAG, Interaction: "invoke"})
		}
		s.OutputProtocols = []string{output}
		s.ResultValidationProtocols = s.OutputProtocols
		commandregistry.Attach(c, s)
		root.AddCommand(c)
	}
	root.AddCommand(newSHVActivationDiscovery(false), newSHVActivationDiscovery(true))
	return root
}

func invokeSHVActivation(o shvActivationOptions, op string, input any) (json.RawMessage, error) {
	cwd, e := os.Getwd()
	if e != nil {
		return nil, e
	}
	raw, e := knowledgeengine.SCVCanonical(input)
	if e != nil {
		return nil, e
	}
	response, e := knowledgeengine.InvokeSHVSource(context.Background(), o.prefix, o.version, cwd, op, raw)
	return response.Result, e
}
func shvActivationResource(o shvActivationOptions) string {
	d, _ := knowledgeengine.SCVDigest(map[string]any{"tops_id": o.topsID, "owner_engine_id": "symphony-shv-source", "source_id": o.sourceID})
	return "symphony.shv.source:" + strings.TrimPrefix(d, "sha256:")
}
func shvActivationAction(kind string) (string, error) {
	if kind != "onboard" && kind != "relocation" && kind != "authority_change" {
		return "", fmt.Errorf("unsupported SHV source change kind")
	}
	return "symphony.shv.source." + strings.ReplaceAll(kind, "_", "-"), nil
}
func shvActivationResult(op string, tx *shvstate.Transaction, owner json.RawMessage, id string) (json.RawMessage, error) {
	var attempt any
	if id != "" {
		v, ok := tx.Attempt(id)
		if !ok {
			return nil, fmt.Errorf("unknown source operation")
		}
		attempt = v
	}
	s := tx.Snapshot()
	result := map[string]any{"protocol": shvActivationProtocol, "operation": op, "tops_id": s.TOPSID, "source_id": s.SourceID, "state_digest": s.StateDigest, "head_operation_id": s.HeadOperationID, "source": tx.Current(), "history": tx.History(), "attempt": attempt, "owner_result": owner, "canonical_apply_enabled": false, "authorization_audit": "ssiag_policy_decision_only", "source_write_stav_receipt": nil}
	return sealSHVActivation(result)
}

// JSON structs/raw messages must become ordinary JSON maps before canonical
// hashing; otherwise Go declaration order can leak into a portable self-seal.
func sealSHVActivation(value map[string]any) (json.RawMessage, error) {
	raw, e := json.Marshal(value)
	if e != nil {
		return nil, e
	}
	var result map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if e = decoder.Decode(&result); e != nil {
		return nil, e
	}
	delete(result, "digest")
	digest, e := knowledgeengine.SCVDigest(result)
	if e != nil {
		return nil, e
	}
	result["digest"] = digest
	return knowledgeengine.SCVCanonical(result)
}

func runSHVSourceActivation(op string, o shvActivationOptions) error {
	if op != "propose" && op != "apply" && op != "status" && op != "recover" {
		return fmt.Errorf("unsupported SHV activation operation")
	}
	installed, e := knowledgeengine.InspectSHVSource(o.prefix, o.version)
	if e != nil {
		return e
	}
	store, e := shvstate.New(o.stateRoot, o.topsID, o.sourceID)
	if e != nil {
		return e
	}
	var raw json.RawMessage
	if op == "propose" || op == "apply" {
		raw, e = knowledgeengine.ReadPayload(o.input)
		if e != nil {
			return e
		}
		if e = knowledgeengine.ValidateSCVBundleText(raw); e != nil {
			return e
		}
	} else if o.input != "" {
		return fmt.Errorf("status/recover use retained state, not replacement input")
	}
	if op == "recover" && o.operationID == "" {
		return fmt.Errorf("recover requires operation-id")
	}
	var output json.RawMessage
	e = store.WithLock(func(tx *shvstate.Transaction) error {
		if op == "status" {
			var result json.RawMessage
			var err error
			if history := tx.History(); len(history) > 0 {
				result, err = invokeSHVActivation(o, "source_status", map[string]any{"history": history})
				if err != nil {
					return err
				}
			}
			output, err = shvActivationResult(op, tx, result, o.operationID)
			return err
		}
		if op == "propose" {
			if e := knowledgeengine.ValidateSCVBundleText(raw); e != nil {
				return e
			}
			var p map[string]any
			decoder := json.NewDecoder(bytes.NewReader(raw))
			decoder.UseNumber()
			if e := decoder.Decode(&p); e != nil {
				return e
			}
			if len(p) != 3 || p["operation_id"] == nil || p["desired"] == nil || p["reason"] == nil {
				return fmt.Errorf("proposal requires exactly operation_id, desired, reason; current comes from protected state")
			}
			desired, ok := p["desired"].(map[string]any)
			if !ok || desired["source_id"] != o.sourceID {
				return fmt.Errorf("desired source differs from source-id")
			}
			p["current"] = tx.Current()
			var err error
			output, err = invokeSHVActivation(o, "source_plan", p)
			return err
		}
		plan := raw
		if op == "recover" {
			prior, ok := tx.Attempt(o.operationID)
			if !ok {
				return fmt.Errorf("no stored operation matches operation-id")
			}
			if prior.Intent.Installation != installed {
				return fmt.Errorf("recovery requires exact original receipt and executable")
			}
			plan = prior.Intent.Plan
		}
		var binding struct {
			OperationID string `json:"operation_id"`
			ChangeKind  string `json:"change_kind"`
			Source      struct {
				Definition struct {
					SourceID string `json:"source_id"`
				} `json:"definition"`
			} `json:"source"`
		}
		if e := json.Unmarshal(plan, &binding); e != nil {
			return e
		}
		if binding.Source.Definition.SourceID != o.sourceID || binding.OperationID == "" {
			return fmt.Errorf("plan source or operation identity mismatch")
		}
		if o.operationID != "" && o.operationID != binding.OperationID {
			return fmt.Errorf("operation-id differs from plan")
		}
		if prior, ok := tx.Attempt(binding.OperationID); ok && prior.Status == "committed" {
			if !scvSameJSON(prior.Intent.Plan, plan) || prior.Intent.Installation != installed {
				return fmt.Errorf("operation_id binds a different committed intent")
			}
			var err error
			output, err = shvActivationResult(op, tx, prior.Intent.Transition, binding.OperationID)
			return err
		}
		transition, err := invokeSHVActivation(o, "source_reduce", map[string]any{"current": tx.Current(), "plan": plan})
		if err != nil {
			return err
		}
		after, err := knowledgeengine.InspectSHVSource(o.prefix, o.version)
		if err != nil || after != installed {
			return fmt.Errorf("SHV source owner installation changed during reduction")
		}
		intent, err := shvstate.NewIntent(binding.OperationID, plan, transition, installed)
		if err != nil {
			return err
		}
		if _, err = tx.Prepare(intent); err != nil {
			return err
		}
		prepared, ok := tx.Attempt(binding.OperationID)
		if !ok {
			return fmt.Errorf("durable authorization correlation is absent")
		}
		action, err := shvActivationAction(binding.ChangeKind)
		if err != nil {
			return err
		}
		decision, err := authorizeKnowledgeRequest(o.topsID, prepared.CorrelationID, action, shvActivationResource(o))
		if err != nil {
			return fmt.Errorf("source intent retained; authorization failed before source mutation: %w", err)
		}
		evidence, err := knowledgeengine.SCVCanonical(decision)
		if err != nil {
			return err
		}
		freshness := knowledgeAuthorizationFreshness(decision)
		guard := func() error {
			current, e := knowledgeengine.InspectSHVSource(o.prefix, o.version)
			if e != nil || current != installed {
				return fmt.Errorf("SHV source installation changed before publication")
			}
			return freshness()
		}
		if err = tx.Commit(binding.OperationID, evidence, guard); err != nil {
			return err
		}
		output, err = shvActivationResult(op, tx, transition, binding.OperationID)
		return err
	})
	if e != nil {
		return e
	}
	return printIndentedJSON(output)
}
