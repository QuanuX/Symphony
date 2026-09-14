package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/shvpublicationstate"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

const publicationProtocol = "symphony.qxctl.shv-publication-result.v1"

type publicationOptions struct{ prefix, version, topsID, catalogueID, stateRoot, input, operationID, selection string }

func newSHVPublicationCommand() *cobra.Command {
	root := structural("publication", fmt.Errorf("publication operation required"))
	for _, op := range []string{"inspect", "plan", "apply", "status", "schema", "template"} {
		o := publicationOptions{}
		c := &cobra.Command{Use: op, Short: "Select an exact caller catalogue revision under its own permission", Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error { return runSHVPublication(op, o) }}
		c.Flags().StringVar(&o.prefix, "prefix", "", "exact publication engine installation")
		c.Flags().StringVar(&o.version, "version", "", "exact publication engine version")
		c.MarkFlagRequired("prefix")
		c.MarkFlagRequired("version")
		c.Flags().Bool("json", false, "complete structured result")
		if op == "plan" || op == "apply" || op == "status" {
			for _, f := range []struct {
				name string
				v    *string
			}{{"tops-id", &o.topsID}, {"catalogue-id", &o.catalogueID}, {"state-root", &o.stateRoot}} {
				c.Flags().StringVar(f.v, f.name, "", "explicit publication scope or state location")
				c.MarkFlagRequired(f.name)
			}
		}
		if op == "plan" || op == "apply" {
			c.Flags().StringVar(&o.input, "input", "", "proposal or exact native publication plan")
			if op == "plan" {
				c.MarkFlagRequired("input")
			}
		}
		if op == "apply" || op == "status" {
			c.Flags().StringVar(&o.operationID, "operation-id", "", "retained operation; apply recovery accepts this instead of input")
		}
		if op == "template" {
			c.Flags().StringVar(&o.selection, "operation", "", "plan or apply")
			c.MarkFlagRequired("operation")
		}
		c.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		interaction := map[string]string{"inspect": "inspect", "plan": "propose", "apply": "apply", "status": "inspect", "schema": "discover", "template": "discover"}[op]
		s := commandSpec("shv.catalogue.publication."+op, featureSHVAdministration, interaction)
		s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:shv-publication-engine", Interaction: interaction})
		out := publicationProtocol
		native := map[string]string{"inspect": "inspect", "plan": "publication_plan", "apply": "publication_reduce", "status": "publication_status"}[op]
		if native != "" {
			s.BackendOperationIDs = []string{"engop:symphony:shv-publication." + strings.ReplaceAll(native, "_", ".")}
		}
		if op == "inspect" {
			out = "symphony.knowledge.engine-descriptor.v2"
		}
		if op == "plan" {
			s.Mutability = "proposal_only"
			s.InputProtocols = []string{"symphony.qxctl.shv-publication-proposal.v1"}
			out = "symphony.shv.publication-plan.v1"
		}
		if op == "apply" {
			s.Mutability = "permission_backed_mutation"
			s.AuthorityMode = "target_host_permission"
			s.RecoveryCommandID = stringPointer("qxcmd:symphony:shv.catalogue.publication.apply")
			s.InputProtocols = []string{"symphony.shv.publication-plan.v1"}
			s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:shv-publication-engine", Interaction: "recover"}, commandregistry.FeatureBinding{FeatureID: backendFeatureSSIAG, Interaction: "invoke"})
		}
		if op == "plan" || op == "apply" {
			for _, f := range []string{"shv-source-engine", "shv-engine", "shv-partition-engine", "shv-graph-duckdb-connector"} {
				ix := "invoke"
				if f == "shv-graph-duckdb-connector" {
					ix = "query"
				}
				s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:" + f, Interaction: ix})
			}
			s.BackendOperationIDs = append(s.BackendOperationIDs, "engop:symphony:shv-partition.partition.build", "engop:symphony:shv-partition.manifest.build", "engop:symphony:shv.graph-store.export", "engop:symphony:shv-source.capture.import", "engop:symphony:shv-source.graph.project", "engop:symphony:shv.catalogue.build", "engop:symphony:shv.coverage.plan", "engop:symphony:shv.evaluate", "engop:symphony:shv.graph.project")
		}
		if op == "schema" || op == "template" {
			out = "symphony.qxctl.shv-publication-" + op + ".v1"
		}
		s.OutputProtocols = []string{out}
		s.ResultValidationProtocols = s.OutputProtocols
		commandregistry.Attach(c, s)
		root.AddCommand(c)
	}
	return root
}
func invokePublication(o publicationOptions, op string, input any) (json.RawMessage, error) {
	cwd, e := os.Getwd()
	if e != nil {
		return nil, e
	}
	p, e := knowledgeengine.SCVCanonical(input)
	if e != nil {
		return nil, e
	}
	r, e := knowledgeengine.InvokeSHVPublication(context.Background(), o.prefix, o.version, cwd, op, p)
	return r.Result, e
}
func publicationResource(o publicationOptions) string {
	d, _ := knowledgeengine.SCVDigest(map[string]any{"tops_id": o.topsID, "owner_engine_id": "symphony-shv-publication", "catalogue_id": o.catalogueID})
	return "symphony.shv.catalogue:" + strings.TrimPrefix(d, "sha256:")
}
func publicationResult(op string, tx *shvpublicationstate.Transaction, owner json.RawMessage, id string) (json.RawMessage, error) {
	var attempt any
	if id != "" {
		a, ok := tx.Attempt(id)
		if !ok {
			return nil, fmt.Errorf("unknown publication operation")
		}
		attempt = a
	}
	s := tx.Snapshot()
	return sealSHVActivation(map[string]any{"protocol": publicationProtocol, "operation": op, "tops_id": s.TOPSID, "catalogue_id": s.CatalogueID, "state_digest": s.StateDigest, "head_operation_id": s.HeadOperationID, "head": tx.Current(), "history": tx.History(), "attempt": attempt, "owner_result": owner, "canonical_apply_enabled": false, "authorization_audit": "ssiag_policy_decision_only", "head_write_stav_receipt": nil})
}
func runSHVPublication(op string, o publicationOptions) error {
	installed, e := knowledgeengine.InspectSHVPublication(o.prefix, o.version)
	if e != nil {
		return e
	}
	if op == "inspect" {
		r, e := invokePublication(o, "inspect", map[string]any{})
		if e != nil {
			return e
		}
		return printIndentedJSON(r)
	}
	if op == "schema" || op == "template" {
		inst, r, e := knowledgeengine.SHVPublicationResource(o.prefix, o.version, op == "template")
		if e != nil {
			return e
		}
		m := map[string]any{"protocol": "symphony.qxctl.shv-publication-" + op + ".v1", "installation": inst}
		if op == "schema" {
			m["schema"] = r
		} else {
			var all map[string]json.RawMessage
			json.Unmarshal(r, &all)
			v, ok := all[o.selection]
			if !ok {
				return fmt.Errorf("unknown publication template")
			}
			m["template"] = v
			m["status"] = "unanswered_template_not_validated_input"
		}
		out, e := sealSHVActivation(m)
		if e != nil {
			return e
		}
		return printIndentedJSON(out)
	}
	if op == "apply" && ((o.input == "") == (o.operationID == "")) {
		return fmt.Errorf("apply requires exactly input or retained operation-id")
	}
	store, e := shvpublicationstate.New(o.stateRoot, o.topsID, o.catalogueID)
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
	e = store.WithLock(func(tx *shvpublicationstate.Transaction) error {
		if op == "status" {
			var r json.RawMessage
			var err error
			if h := tx.History(); len(h) > 0 {
				r, err = invokePublication(o, "publication_status", map[string]any{"history": h})
				if err != nil {
					return err
				}
			}
			output, err = publicationResult(op, tx, r, o.operationID)
			return err
		}
		if op == "plan" {
			p, err := refreshObject(raw, "operation_id", "desired", "reason")
			if err != nil {
				return err
			}
			d, err := publicationEvidence(p["desired"], o.topsID, o.catalogueID)
			if err != nil {
				return err
			}
			output, err = invokePublication(o, "publication_plan", map[string]any{"operation_id": p["operation_id"], "desired": d, "reason": p["reason"], "current": tx.Current()})
			return err
		}
		plan := raw
		if o.operationID != "" {
			a, ok := tx.Attempt(o.operationID)
			if !ok || a.Intent.Installation != installed {
				return fmt.Errorf("recovery requires exact original intent and installation")
			}
			plan = a.Intent.Plan
		}
		var binding struct {
			OperationID string `json:"operation_id"`
			Head        struct {
				Definition json.RawMessage `json:"definition"`
			} `json:"head"`
		}
		if err := json.Unmarshal(plan, &binding); err != nil {
			return err
		}
		var d map[string]json.RawMessage
		if err := json.Unmarshal(binding.Head.Definition, &d); err != nil {
			return err
		}
		if refreshString(d["tops_id"]) != o.topsID || refreshString(d["catalogue_id"]) != o.catalogueID {
			return fmt.Errorf("publication scope differs")
		}
		if a, ok := tx.Attempt(binding.OperationID); ok && a.Status == "committed" {
			if a.Intent.Installation != installed || !scvSameJSON(a.Intent.Plan, plan) {
				return fmt.Errorf("operation-id binds another committed intent")
			}
			var err error
			output, err = publicationResult(op, tx, a.Intent.Transition, binding.OperationID)
			return err
		}
		transition, err := invokePublication(o, "publication_reduce", map[string]any{"current": tx.Current(), "plan": plan})
		if err != nil {
			return err
		}
		if err = replayPublicationDefinition(binding.Head.Definition, o.topsID, o.catalogueID); err != nil {
			return err
		}
		intent, err := shvpublicationstate.NewIntent(binding.OperationID, plan, transition, installed)
		if err != nil {
			return err
		}
		if _, err = tx.Prepare(intent); err != nil {
			return err
		}
		prepared, _ := tx.Attempt(binding.OperationID)
		decision, err := authorizeKnowledgeRequest(o.topsID, prepared.CorrelationID, "symphony.shv.catalogue.publish", publicationResource(o))
		if err != nil {
			return fmt.Errorf("publication intent retained; authorization failed: %w", err)
		}
		evidence, err := knowledgeengine.SCVCanonical(decision)
		if err != nil {
			return err
		}
		guard := func() error {
			current, err := knowledgeengine.InspectSHVPublication(o.prefix, o.version)
			if err != nil || current != installed {
				return fmt.Errorf("publication owner changed")
			}
			if err = replayPublicationDefinition(binding.Head.Definition, o.topsID, o.catalogueID); err != nil {
				return err
			}
			if err = knowledgeAuthorizationFreshness(decision)(); err != nil {
				return err
			}
			live, err := authorizeKnowledgeRequest(o.topsID, prepared.CorrelationID, "symphony.shv.catalogue.publish", publicationResource(o))
			if err != nil {
				return fmt.Errorf("publication permission recheck failed: %w", err)
			}
			if live.PolicyDigest != decision.PolicyDigest || live.ConfigDigest != decision.ConfigDigest || live.Subject != decision.Subject || live.Capability == nil || decision.Capability == nil || live.Capability.GrantID != decision.Capability.GrantID {
				return fmt.Errorf("publication authority changed; retry with a new decision")
			}
			return knowledgeAuthorizationFreshness(decision)()
		}
		if err = tx.Commit(binding.OperationID, evidence, guard); err != nil {
			return err
		}
		output, err = publicationResult(op, tx, transition, binding.OperationID)
		return err
	})
	if e != nil {
		return e
	}
	return printIndentedJSON(output)
}
