package main

import (
	"encoding/json"
	"fmt"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvworkflow"
	"github.com/spf13/cobra"
)

func newSCVBundleObligationsCommand() *cobra.Command {
	group := structural("obligations", fmt.Errorf("bundled obligations subcommand is required: retain, show"))
	for _, action := range []string{"retain", "show"} {
		options := scvOptions{}
		var root string
		command := &cobra.Command{Use: action, Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error { return runSCVBundleObligationLink(action, options, root) }}
		if action == "retain" {
			scvFlagsVersion(command, &options, "0.10.0-dev")
		} else {
			scvRetainedReadFlags(command, &options)
		}
		command.Flags().StringVar(&root, "workflow-root", "", "explicit owned retained artifact and workflow root")
		command.Flags().StringVar(&options.topsID, "tops-id", "", "exact TOPS UUID for retained evidence namespace")
		interaction, mutability, input := "invoke", "evidence_only", "symphony.qxctl.scv-obligation-link-request.v2"
		if action == "show" {
			interaction, mutability, input = "inspect", "read_only", "symphony.qxctl.scv-obligation-link-show-input.v2"
		}
		attachSCVWorkflow(command, "scv.composition.bundle.obligations."+action, input, "symphony.qxctl.scv-obligation-link-result.v2", interaction, mutability, []string{"composition_explore", "composition_bundle_evaluate"}, false)
		group.AddCommand(command)
	}
	return group
}

func runSCVBundleObligationLink(action string, options scvOptions, root string) error {
	if root == "" {
		return fmt.Errorf("--workflow-root is required; no implicit obligation link")
	}
	store, err := scvworkflow.New(root, options.topsID)
	if err != nil {
		return err
	}
	raw, err := knowledgeengine.ReadPayload(options.input)
	if err != nil {
		return err
	}
	if err = knowledgeengine.ValidateSCVBundleText(raw); err != nil {
		return err
	}
	input, err := scvworkflow.Decode(raw)
	if err != nil {
		return err
	}
	r, err := newWorkflowRunner(options, action == "retain")
	if err != nil {
		return err
	}
	result, err := r.bundleObligationLink(store, action, input)
	if err != nil {
		return err
	}
	return outputSCV(options, result)
}

func logicalDescriptorRef(value any) (string, error) {
	descriptor, ok := value.(map[string]any)
	if !ok {
		return "", fmt.Errorf("missing logical reference descriptor")
	}
	if err := exactCorpusFields(descriptor, "record_ref", "logical_operation", "logical_protocol", "logical_digest", "artifact_digest", "transport"); err != nil {
		return "", err
	}
	return corpusDigest(descriptor["record_ref"], "record_ref")
}

func logicalDescriptorEqual(actual, declared map[string]any) bool {
	a, err := knowledgeengine.SCVCanonical(actual)
	if err != nil {
		return false
	}
	b, err := knowledgeengine.SCVCanonical(declared)
	return err == nil && scvworkflow.Same(a, b)
}

func (r *workflowRunner) bundleObligationLink(store scvworkflow.Store, action string, input map[string]any) (json.RawMessage, error) {
	var output json.RawMessage
	err := store.With("", action == "retain", func(s *scvworkflow.Session, _ *scvworkflow.Run) error {
		var link map[string]any
		var linkRef string
		if action == "retain" {
			if err := exactCorpusFields(input, "protocol", "transport", "before_ref", "after_ref", "submissions"); err != nil {
				return err
			}
			if input["protocol"] != "symphony.qxctl.scv-obligation-link-request.v2" || input["transport"] != "bundle" {
				return fmt.Errorf("bundled obligation retention requires the explicit v2 request")
			}
			if !knowledgeengine.SCVSupports(r.installation.Version, "composition_bundle_obligation_link") {
				return fmt.Errorf("selected release does not support bundled obligation relationships")
			}
			beforeRef, err := corpusDigest(input["before_ref"], "before_ref")
			if err != nil {
				return err
			}
			afterRef, err := corpusDigest(input["after_ref"], "after_ref")
			if err != nil {
				return err
			}
			before, err := r.resolveLogicalReference(s, beforeRef, "composition_explore", true)
			if err != nil {
				return err
			}
			after, err := r.resolveLogicalReference(s, afterRef, "composition_explore", true)
			if err != nil {
				return err
			}
			record, err := r.bundledLogicalRecord(r.installation, "composition_followup", map[string]any{"before": before.Artifact, "after": after.Artifact, "submissions": input["submissions"]})
			if err != nil {
				return err
			}
			raw, err := scvworkflow.Canonical(record)
			if err != nil {
				return err
			}
			followupRef, err := putBundleWorkflowObject(s, "records", raw)
			if err != nil {
				return err
			}
			if err = r.checkpoint("obligation_result_retained"); err != nil {
				return err
			}
			followup, err := projectLogicalRecord(record, "composition_followup")
			if err != nil {
				return err
			}
			submissionsDigest, err := knowledgeengine.SCVDigest(map[string]any{"submissions": input["submissions"]})
			if err != nil {
				return err
			}
			if followup.Descriptor["record_ref"] != followupRef {
				return fmt.Errorf("retained follow-up reference mismatch")
			}
			link = map[string]any{"protocol": "symphony.qxctl.scv-obligation-link.v2", "root": store.Root, "tops_id": store.TOPSID, "transport": "bundle", "before": before.Descriptor, "after": after.Descriptor, "followup": followup.Descriptor, "submissions_digest": submissionsDigest}
			wrapper, err := scvworkflow.Seal(map[string]any{"protocol": "symphony.qxctl.scv-workflow-payload.v1", "input": link})
			if err != nil {
				return err
			}
			linkRef, err = putBundleWorkflowObject(s, "payloads", wrapper)
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
			if err = knowledgeengine.ValidateSCVBundleUnicode(raw); err != nil {
				return err
			}
			wrapper, err := scvworkflow.Decode(raw)
			if err != nil {
				return err
			}
			var ok bool
			link, ok = wrapper["input"].(map[string]any)
			if !ok {
				return fmt.Errorf("missing bundled obligation link")
			}
			if err = exactCorpusFields(link, "protocol", "root", "tops_id", "transport", "before", "after", "followup", "submissions_digest"); err != nil {
				return err
			}
			if link["protocol"] != "symphony.qxctl.scv-obligation-link.v2" || link["transport"] != "bundle" || link["root"] != store.Root || link["tops_id"] != store.TOPSID {
				return fmt.Errorf("bundled obligation link namespace or transport mismatch")
			}
			resolved := map[string]logicalArtifactReference{}
			for _, name := range []string{"before", "after", "followup"} {
				ref, err := logicalDescriptorRef(link[name])
				if err != nil {
					return err
				}
				op := "composition_explore"
				if name == "followup" {
					op = "composition_followup"
				}
				value, err := r.resolveLogicalReference(s, ref, op, false)
				if err != nil {
					return err
				}
				if !logicalDescriptorEqual(value.Descriptor, link[name].(map[string]any)) {
					return fmt.Errorf("bundled obligation descriptor does not match its exact record")
				}
				if name == "followup" && (value.Record.Operation != "composition_bundle_evaluate" || !knowledgeengine.SCVSupports(value.Record.Installation.Version, "composition_bundle_obligation_link")) {
					return fmt.Errorf("link follow-up lacks its recorded supporting bundle owner")
				}
				if r.owner == nil {
					return fmt.Errorf("original owner invocation is unavailable")
				}
				if _, err = r.replay(value.Record); err != nil {
					return err
				}
				resolved[name] = value
			}
			payload, err := scvworkflow.Decode(resolved["followup"].Input)
			if err != nil {
				return err
			}
			if err = exactCorpusFields(payload, "before", "after", "submissions"); err != nil {
				return err
			}
			before, err := knowledgeengine.SCVCanonical(payload["before"])
			if err != nil {
				return err
			}
			after, err := knowledgeengine.SCVCanonical(payload["after"])
			if err != nil {
				return err
			}
			submissionsDigest, err := knowledgeengine.SCVDigest(map[string]any{"submissions": payload["submissions"]})
			if err != nil {
				return err
			}
			if !scvworkflow.Same(before, resolved["before"].Artifact) || !scvworkflow.Same(after, resolved["after"].Artifact) || submissionsDigest != link["submissions_digest"] {
				return fmt.Errorf("bundled obligation link differs from its retained logical input")
			}
		} else {
			return fmt.Errorf("unknown bundled obligation link action")
		}
		followup := link["followup"].(map[string]any)
		var err error
		output, err = scvworkflow.Seal(map[string]any{"protocol": "symphony.qxctl.scv-obligation-link-result.v2", "link_ref": linkRef, "link": link, "followup_ref": followup["record_ref"], "validation": "owner_replayed"})
		return err
	})
	return output, err
}
