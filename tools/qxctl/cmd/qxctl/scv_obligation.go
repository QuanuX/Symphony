package main

import (
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvworkflow"
	"github.com/spf13/cobra"
)

func newSCVObligationsCommand() *cobra.Command {
	group := structural("obligations", fmt.Errorf("obligations subcommand is required: inspect, followup, retain, show"))
	for _, item := range []struct{ leaf, operation, interaction, mutability string }{
		{"inspect", "composition_obligations", "query", "read_only"},
		{"followup", "composition_followup", "validate", "evidence_only"},
	} {
		options := scvOptions{}
		command := &cobra.Command{Use: item.leaf, Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error { return runSCV(item.operation, options) }}
		scvFlagsVersion(command, &options, "0.8.0-dev")
		attachSCV(command, "scv.composition.obligations."+item.leaf, item.operation, item.interaction, item.mutability, false)
		group.AddCommand(command)
	}
	for _, leaf := range []string{"retain", "show"} {
		options := scvOptions{}
		var root string
		command := &cobra.Command{Use: leaf, Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error { return runSCVObligationLink(leaf, options, root) }}
		if leaf == "show" {
			scvRetainedReadFlags(command, &options)
		} else {
			scvFlagsVersion(command, &options, "0.8.0-dev")
		}
		command.Flags().StringVar(&root, "workflow-root", "", "explicit owned retained artifact and workflow root")
		command.Flags().StringVar(&options.topsID, "tops-id", "", "exact TOPS UUID for retained evidence namespace")
		interaction, mutability := "invoke", "evidence_only"
		if leaf == "show" {
			interaction, mutability = "inspect", "read_only"
		}
		attachSCVWorkflow(command, "scv.composition.obligations."+leaf, "symphony.qxctl.scv-obligation-"+leaf+"-input.v1", "symphony.qxctl.scv-obligation-link-result.v1", interaction, mutability, []string{"composition_explore", "composition_followup"}, false)
		group.AddCommand(command)
	}
	return group
}
func runSCVObligationLink(action string, options scvOptions, root string) error {
	if root == "" {
		return fmt.Errorf("--workflow-root is required; no implicit obligation link")
	}
	store, err := scvworkflow.New(root, options.topsID)
	if err != nil {
		return err
	}
	input, err := scvInput(options)
	if err != nil {
		return err
	}
	r, err := newWorkflowRunner(options, action == "retain")
	if err != nil {
		return err
	}
	if action == "retain" && !knowledgeengine.SCVArtifactSupported(options.version, "composition_followup") {
		return fmt.Errorf("obligation follow-up requires an exact supporting release")
	}
	result, err := r.obligationLink(store, action, input)
	if err != nil {
		return err
	}
	return outputSCV(options, result)
}
func (r *workflowRunner) obligationLink(store scvworkflow.Store, action string, input map[string]any) (json.RawMessage, error) {
	var result json.RawMessage
	err := store.With("", action == "retain", func(s *scvworkflow.Session, _ *scvworkflow.Run) error {
		var link map[string]any
		var linkRef string
		if action == "retain" {
			if err := exactCorpusFields(input, "before_ref", "after_ref", "submissions"); err != nil {
				return err
			}
			if !knowledgeengine.SCVArtifactSupported(r.installation.Version, "composition_followup") {
				return fmt.Errorf("unsupported exact follow-up owner")
			}
			beforeRef, err := corpusDigest(input["before_ref"], "before_ref")
			if err != nil {
				return err
			}
			afterRef, err := corpusDigest(input["after_ref"], "after_ref")
			if err != nil {
				return err
			}
			before, err := r.compositionReference(s, beforeRef, "composition", true)
			if err != nil {
				return err
			}
			after, err := r.compositionReference(s, afterRef, "composition", true)
			if err != nil {
				return err
			}
			payload, err := workflowRaw(map[string]any{"before": before.Artifact, "after": after.Artifact, "submissions": input["submissions"]})
			if err != nil {
				return err
			}
			native, err := r.owner(r.installation, "composition_followup", payload)
			if err != nil {
				return err
			}
			record, err := scvworkflow.NewRecord("composition_followup", r.installation, payload, native)
			if err != nil {
				return err
			}
			raw, err := scvworkflow.Canonical(record)
			if err != nil {
				return err
			}
			followupRef, err := s.Put("records", raw)
			if err != nil {
				return err
			}
			if err = r.checkpoint("obligation_result_retained"); err != nil {
				return err
			}
			submissionsDigest, err := knowledgeengine.SCVDigest(map[string]any{"submissions": input["submissions"]})
			if err != nil {
				return err
			}
			link = map[string]any{"protocol": "symphony.qxctl.scv-obligation-link.v1", "root": store.Root, "tops_id": store.TOPSID, "before_ref": beforeRef, "after_ref": afterRef, "followup_ref": followupRef, "submissions_digest": submissionsDigest}
			wrapper, err := scvworkflow.Seal(map[string]any{"protocol": "symphony.qxctl.scv-workflow-payload.v1", "input": link})
			if err != nil {
				return err
			}
			linkRef, err = s.Put("payloads", wrapper)
			if err != nil {
				return err
			}
			if err = r.checkpoint("obligation_link_retained"); err != nil {
				return err
			}
		} else if action == "show" {
			if err := exactCorpusFields(input, "link_ref"); err != nil {
				return err
			}
			var err error
			linkRef, err = corpusDigest(input["link_ref"], "link_ref")
			if err != nil {
				return err
			}
			raw, err := s.Get("payloads", linkRef)
			if err != nil {
				return err
			}
			wrapper, err := scvworkflow.Decode(raw)
			if err != nil {
				return err
			}
			var ok bool
			link, ok = wrapper["input"].(map[string]any)
			if !ok {
				return fmt.Errorf("missing obligation link")
			}
			if err = exactCorpusFields(link, "protocol", "root", "tops_id", "before_ref", "after_ref", "followup_ref", "submissions_digest"); err != nil {
				return err
			}
			if link["protocol"] != "symphony.qxctl.scv-obligation-link.v1" || link["root"] != store.Root || link["tops_id"] != store.TOPSID {
				return fmt.Errorf("obligation link namespace mismatch")
			}
			refs := map[string]string{}
			for _, key := range []string{"before_ref", "after_ref", "followup_ref", "submissions_digest"} {
				refs[key], err = corpusDigest(link[key], key)
				if err != nil {
					return err
				}
			}
			before, err := r.compositionReference(s, refs["before_ref"], "composition", true)
			if err != nil {
				return err
			}
			after, err := r.compositionReference(s, refs["after_ref"], "composition", true)
			if err != nil {
				return err
			}
			followup, err := r.compositionReference(s, refs["followup_ref"], "composition_followup", true)
			if err != nil {
				return err
			}
			payload, err := scvworkflow.Decode(followup.Input)
			if err != nil {
				return err
			}
			if err = exactCorpusFields(payload, "before", "after", "submissions"); err != nil {
				return err
			}
			b, err := scvworkflow.Canonical(payload["before"])
			if err != nil {
				return err
			}
			a, err := scvworkflow.Canonical(payload["after"])
			if err != nil {
				return err
			}
			sd, err := knowledgeengine.SCVDigest(map[string]any{"submissions": payload["submissions"]})
			if err != nil {
				return err
			}
			if !scvworkflow.Same(b, before.Artifact) || !scvworkflow.Same(a, after.Artifact) || sd != refs["submissions_digest"] {
				return fmt.Errorf("obligation link does not match retained owner input")
			}
		} else {
			return fmt.Errorf("unknown obligation link operation")
		}
		var err error
		result, err = scvworkflow.Seal(map[string]any{"protocol": "symphony.qxctl.scv-obligation-link-result.v1", "link_ref": linkRef, "link": link, "followup_ref": link["followup_ref"], "validation": "owner_replayed"})
		return err
	})
	return result, err
}
