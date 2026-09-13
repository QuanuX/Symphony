package main

import (
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvcorpus"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvworkflow"
	"github.com/spf13/cobra"
	"strings"
)

func newSCVBundleWorkflowCommand() *cobra.Command {
	group := structural("workflow", fmt.Errorf("composition workflow subcommand is required: run, status, recover"))
	for _, leaf := range []string{"run", "status", "recover"} {
		options := scvOptions{}
		var root string
		command := &cobra.Command{Use: leaf, Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error { return runSCVBundleWorkflow(leaf, options, root) }}
		if leaf == "status" {
			scvRetainedReadFlags(command, &options)
		} else {
			scvFlagsVersion(command, &options, "0.10.0-dev")
		}
		command.Flags().StringVar(&root, "workflow-root", "", "explicit owned retained artifact and workflow root")
		command.Flags().StringVar(&options.topsID, "tops-id", "", "exact TOPS UUID for retained evidence namespace")
		key := "scv.composition.bundle.workflow." + leaf
		interaction := map[string]string{"run": "invoke", "status": "inspect", "recover": "recover"}[leaf]
		spec := commandSpec(key, featureSCVAdministration, interaction)
		spec.Mutability = "evidence_only"
		spec.InputProtocols = []string{"symphony.qxctl.scv-composition-workflow-" + leaf + "-input.v2"}
		if leaf == "run" {
			spec.InputProtocols = []string{"symphony.qxctl.scv-composition-workflow-request.v2"}
		}
		spec.OutputProtocols = []string{"symphony.qxctl.scv-composition-workflow-result.v2"}
		spec.ResultValidationProtocols = spec.OutputProtocols
		if leaf == "status" {
			spec.Mutability = "read_only"
		} else {
			spec.RecoveryCommandID = stringPointer("qxcmd:symphony:scv.composition.bundle.workflow.recover")
			for _, domain := range knowledgeengine.SCVDomains() {
				seen := map[string]bool{}
				for _, op := range []string{"provider_pack_prepare", "provider_pack_evaluate", "provider_interpret", "knowledge_interpret", "composition_explore", "composition_bundle_evaluate"} {
					spec.BackendOperationIDs = append(spec.BackendOperationIDs, "engop:symphony:"+domain+"."+strings.ReplaceAll(op, "_", "."))
					kind := knowledgeengine.SCVOperationInteraction(op)
					if !seen[kind] {
						seen[kind] = true
						spec.FeatureBindings = append(spec.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:" + domain + "-engine", Interaction: kind})
					}
				}
			}
		}
		commandregistry.Attach(command, spec)
		group.AddCommand(command)
	}
	return group
}
func runSCVBundleWorkflow(operation string, options scvOptions, root string) error {
	if root == "" {
		return fmt.Errorf("--workflow-root is required; no implicit composition run")
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
	r, err := newWorkflowRunner(options, false)
	if err != nil {
		return err
	}
	if operation != "status" {
		if !knowledgeengine.SCVSupports(options.version, "composition_bundle_workflow") {
			return fmt.Errorf("composition workflow requires an exact supporting engine release")
		}
		r.installation, err = r.inspect(options.domain, options.prefix, options.version)
		if err != nil {
			return err
		}
	}
	result, err := r.executeBundleComposition(store, operation, input)
	if err != nil {
		return err
	}
	return outputSCV(options, result)
}
func (r *workflowRunner) executeBundleComposition(store scvworkflow.Store, operation string, input map[string]any) (json.RawMessage, error) {
	var proposed scvworkflow.CompositionRunV2
	if operation == "run" {
		if _, err := scvworkflow.CompositionPlanV2(input); err != nil {
			return nil, err
		}
		raw, err := workflowRaw(input)
		if err != nil {
			return nil, err
		}
		proposed, err = scvworkflow.NewCompositionRunV2(store, r.installation, raw)
		if err != nil {
			return nil, err
		}
	} else if operation == "status" || operation == "recover" {
		if err := exactCorpusFields(input, "operation_id"); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unknown composition workflow operation")
	}
	id, err := corpusString(input["operation_id"], "operation_id")
	if err != nil {
		return nil, err
	}
	var result json.RawMessage
	err = store.WithCompositionV2(id, operation != "status", func(s *scvworkflow.Session, existing *scvworkflow.CompositionRunV2) error {
		run := proposed
		if existing != nil {
			run = *existing
			if operation == "run" && run.IntentDigest != proposed.IntentDigest {
				return fmt.Errorf("operation ID binds a different exact composition intent")
			}
			if operation != "status" && r.installation != run.Installation {
				return fmt.Errorf("composition recovery requires the original exact installation")
			}
		} else if operation != "run" {
			return fmt.Errorf("retained composition workflow not found")
		}
		pinned, err := scvworkflow.Decode(run.Request)
		if err != nil {
			return err
		}
		plan, err := scvworkflow.CompositionPlanV2(pinned)
		if err != nil {
			return err
		}
		// References remain explicit even before a checkpoint exists. Status verifies
		// their envelope/type linkage but neither invokes nor requires an owner.
		if err = r.bundleCompositionReferences(s, pinned, false); err != nil {
			return err
		}
		if operation == "status" {
			if err = r.verifyBundleCompositionCheckpoints(s, run, pinned, false); err != nil {
				return err
			}
			result, err = r.bundleCompositionWorkflowResult(s, run, "sealed_checkpoints_only")
			return err
		}
		if existing == nil {
			run, err = s.SaveCompositionV2(store, run)
			if err != nil {
				return err
			}
			if err = r.checkpoint("prepared"); err != nil {
				return err
			}
		}
		if err = r.bundleCompositionReferences(s, pinned, true); err != nil {
			return err
		}
		if err = r.verifyBundleCompositionCheckpoints(s, run, pinned, true); err != nil {
			return err
		}
		for _, name := range plan {
			payload, owner, op, err := r.bundleCompositionStageInput(s, run, pinned, name)
			if err != nil {
				return err
			}
			if _, err = r.bundleCompositionStage(store, s, &run, name, op, owner, payload); err != nil {
				return err
			}
		}
		if !run.Complete {
			run.Complete = true
			if _, err = r.bundleCompositionWorkflowResult(s, run, "owner_replayed"); err != nil {
				return err
			}
			run, err = s.SaveCompositionV2(store, run)
			if err != nil {
				return err
			}
			if err = r.checkpoint("complete"); err != nil {
				return err
			}
		}
		result, err = r.bundleCompositionWorkflowResult(s, run, "owner_replayed")
		return err
	})
	return result, err
}

func (r *workflowRunner) bundleCompositionReferences(s *scvworkflow.Session, pinned map[string]any, replay bool) error {
	for _, raw := range pinned["pack_evaluations"].([]any) {
		p := raw.(map[string]any)
		if _, err := r.compositionBundleReference(s, p["pack_ref"].(string), "provider_pack", replay); err != nil {
			return err
		}
	}
	for _, pair := range [][2]string{{"pack_evaluation_refs", "provider_pack_evaluation"}, {"interpretation_refs", "interpretation"}, {"knowledge_refs", "knowledge"}} {
		for _, raw := range pinned[pair[0]].([]any) {
			if _, err := r.compositionBundleReference(s, raw.(string), pair[1], replay); err != nil {
				return err
			}
		}
	}
	if pinned["prior_composition_ref"] != nil {
		if _, err := r.resolveLogicalReference(s, pinned["prior_composition_ref"].(string), "composition_explore", replay); err != nil {
			return err
		}
	}
	return nil
}

// Inputs are mechanically rebuilt from the pinned request and exact records on
// every status/recovery path. A matched but unrelated input/result pair cannot
// substitute for a stage merely by having internally consistent seals.
func (r *workflowRunner) bundleCompositionStageInput(s *scvworkflow.Session, run scvworkflow.CompositionRunV2, pinned map[string]any, name string) (json.RawMessage, knowledgeengine.Installation, string, error) {
	owner := run.Installation
	if strings.HasPrefix(name, "pack:") {
		for _, raw := range pinned["pack_evaluations"].([]any) {
			p := raw.(map[string]any)
			if name != "pack:"+p["evaluation_id"].(string) {
				continue
			}
			record, err := r.compositionBundleReference(s, p["pack_ref"].(string), "provider_pack", false)
			if err != nil {
				return nil, owner, "", err
			}
			input := map[string]any{"pack": record.Artifact}
			for k, v := range p["evaluation"].(map[string]any) {
				input[k] = v
			}
			payload, err := workflowRaw(input)
			return payload, record.Installation, "provider_pack_evaluate", err
		}
		return nil, owner, "", fmt.Errorf("unrequested pack stage")
	}
	if name == "explore" {
		input := map[string]any{}
		for k, v := range pinned["composition"].(map[string]any) {
			input[k] = v
		}
		packs, interpretations, knowledge := []any{}, []any{}, []any{}
		for _, raw := range pinned["pack_evaluations"].([]any) {
			p := raw.(map[string]any)
			ref := run.Stages["pack:"+p["evaluation_id"].(string)].ResultRef
			if ref == "" {
				return nil, owner, "", fmt.Errorf("composition precedes requested pack evaluation")
			}
			record, err := r.compositionBundleReference(s, ref, "provider_pack_evaluation", false)
			if err != nil {
				return nil, owner, "", err
			}
			packs = append(packs, record.Artifact)
		}
		for _, pair := range [][2]string{{"pack_evaluation_refs", "provider_pack_evaluation"}, {"interpretation_refs", "interpretation"}, {"knowledge_refs", "knowledge"}} {
			for _, raw := range pinned[pair[0]].([]any) {
				record, err := r.compositionBundleReference(s, raw.(string), pair[1], false)
				if err != nil {
					return nil, owner, "", err
				}
				switch pair[0] {
				case "pack_evaluation_refs":
					packs = append(packs, record.Artifact)
				case "interpretation_refs":
					interpretations = append(interpretations, record.Artifact)
				case "knowledge_refs":
					knowledge = append(knowledge, record.Artifact)
				}
			}
		}
		input["provider_packs"], input["interpretations"], input["additional_knowledge"] = packs, interpretations, knowledge
		payload, err := bundleLogicalInput(owner, "composition_explore", input)
		return payload, owner, "composition_bundle_evaluate", err
	}
	if name == "reassess" && pinned["prior_composition_ref"] != nil {
		before, err := r.resolveLogicalReference(s, pinned["prior_composition_ref"].(string), "composition_explore", false)
		if err != nil {
			return nil, owner, "", err
		}
		after, err := r.resolveLogicalReference(s, run.Stages["explore"].ResultRef, "composition_explore", false)
		if err != nil {
			return nil, owner, "", err
		}
		payload, err := bundleLogicalInput(owner, "composition_reassess", map[string]any{"before": before.Artifact, "after": after.Artifact})
		return payload, owner, "composition_bundle_evaluate", err
	}
	return nil, owner, "", fmt.Errorf("unknown composition stage")
}
func (r *workflowRunner) verifyBundleCompositionCheckpoints(s *scvworkflow.Session, run scvworkflow.CompositionRunV2, pinned map[string]any, replay bool) error {
	plan, err := scvworkflow.CompositionPlanV2(pinned)
	if err != nil {
		return err
	}
	for _, name := range plan {
		checkpoint, ok := run.Stages[name]
		if !ok {
			continue
		}
		expected, owner, op, err := r.bundleCompositionStageInput(s, run, pinned, name)
		if err != nil {
			return err
		}
		raw, err := s.Get("payloads", checkpoint.InputRef)
		if err != nil {
			return err
		}
		if err = knowledgeengine.ValidateSCVBundleUnicode(raw); err != nil {
			return err
		}
		payload, err := scvworkflow.Decode(raw)
		if err != nil {
			return err
		}
		if err = exactCorpusFields(payload, "protocol", "input", "digest"); err != nil || payload["protocol"] != "symphony.qxctl.scv-workflow-payload.v1" {
			return fmt.Errorf("invalid bundle workflow checkpoint envelope")
		}
		if !scvcorpus.Same(payload["input"], expected) {
			return fmt.Errorf("composition checkpoint differs from pinned request/artifacts")
		}
		if checkpoint.ResultRef == "" {
			continue
		}
		record, err := r.compositionBundleReference(s, checkpoint.ResultRef, scvworkflow.Kind(op), replay)
		if err != nil {
			return err
		}
		if record.Operation != op || record.Installation != owner || !scvworkflow.Same(record.Input, expected) {
			return fmt.Errorf("composition checkpoint changed original owner/input")
		}
		if op == "composition_bundle_evaluate" {
			logical := "composition_explore"
			if name == "reassess" {
				logical = "composition_reassess"
			}
			if _, err = r.resolveLogicalReference(s, checkpoint.ResultRef, logical, false); err != nil {
				return err
			}
		}
	}
	return nil
}
func (r *workflowRunner) bundleCompositionStage(store scvworkflow.Store, s *scvworkflow.Session, run *scvworkflow.CompositionRunV2, name, operation string, owner knowledgeengine.Installation, payload json.RawMessage) (scvworkflow.Record, error) {
	wrapper, err := scvworkflow.Seal(map[string]any{"protocol": "symphony.qxctl.scv-workflow-payload.v1", "input": payload})
	if err != nil {
		return scvworkflow.Record{}, err
	}
	inputRef, err := scvworkflow.Digest(wrapper)
	if err != nil {
		return scvworkflow.Record{}, err
	}
	checkpoint, exists := run.Stages[name]
	if exists && checkpoint.InputRef != inputRef {
		return scvworkflow.Record{}, fmt.Errorf("composition stage input changed")
	}
	if !exists {
		if _, err = putBundleWorkflowObject(s, "payloads", wrapper); err != nil {
			return scvworkflow.Record{}, err
		}
		run.Stages[name] = scvworkflow.Checkpoint{InputRef: inputRef}
		*run, err = s.SaveCompositionV2(store, *run)
		if err != nil {
			return scvworkflow.Record{}, err
		}
		if err = r.checkpoint(name + "_input"); err != nil {
			return scvworkflow.Record{}, err
		}
		checkpoint = run.Stages[name]
	}
	if checkpoint.ResultRef != "" {
		return r.compositionBundleReference(s, checkpoint.ResultRef, scvworkflow.Kind(operation), false)
	}
	result, err := r.owner(owner, operation, payload)
	if err != nil {
		return scvworkflow.Record{}, err
	}
	if err = knowledgeengine.ValidateJSONObject(result, scvworkflow.MaxBytes); err != nil {
		return scvworkflow.Record{}, err
	}
	if err = knowledgeengine.ValidateSCVBundleUnicode(result); err != nil {
		return scvworkflow.Record{}, err
	}
	record, err := scvworkflow.NewRecord(operation, owner, payload, result)
	if err != nil {
		return record, err
	}
	raw, err := scvworkflow.Canonical(record)
	if err != nil {
		return record, err
	}
	ref, err := putBundleWorkflowObject(s, "records", raw)
	if err != nil {
		return record, err
	}
	if err = r.checkpoint(name + "_artifact"); err != nil {
		return record, err
	}
	run.Stages[name] = scvworkflow.Checkpoint{InputRef: inputRef, ResultRef: ref}
	*run, err = s.SaveCompositionV2(store, *run)
	if err != nil {
		return record, err
	}
	return record, r.checkpoint(name + "_result")
}
func (r *workflowRunner) bundleCompositionWorkflowResult(s *scvworkflow.Session, run scvworkflow.CompositionRunV2, validation string) (json.RawMessage, error) {
	status := "prepared"
	packs := map[string]string{}
	for name, c := range run.Stages {
		if strings.HasPrefix(name, "pack:") {
			status = "evaluating_packs"
			if c.ResultRef != "" {
				packs[strings.TrimPrefix(name, "pack:")] = c.ResultRef
			}
		}
	}
	var composition, reassessment any
	if ref := run.Stages["explore"].ResultRef; ref != "" {
		composition = ref
		status = "explored"
	}
	if ref := run.Stages["reassess"].ResultRef; ref != "" {
		reassessment = ref
	}
	if run.Complete {
		status = "complete"
	}
	var compositionDescriptor, reassessmentDescriptor any
	for _, pair := range [][2]string{{"explore", "composition_explore"}, {"reassess", "composition_reassess"}} {
		if ref := run.Stages[pair[0]].ResultRef; ref != "" {
			view, err := r.resolveLogicalReference(s, ref, pair[1], false)
			if err != nil {
				return nil, err
			}
			if pair[0] == "explore" {
				compositionDescriptor = view.Descriptor
			} else {
				reassessmentDescriptor = view.Descriptor
			}
		}
	}
	raw, err := scvworkflow.Canonical(map[string]any{"protocol": "symphony.qxctl.scv-composition-workflow-result.v2", "composition": compositionDescriptor, "reassessment": reassessmentDescriptor, "run": run, "status": status, "validation": validation, "pack_evaluation_refs": packs, "composition_ref": composition, "reassessment_ref": reassessment})
	if err != nil {
		return nil, err
	}
	if err = knowledgeengine.ValidateJSONObject(raw, scvworkflow.MaxBytes); err != nil {
		return nil, err
	}
	return raw, nil
}

// The v2 adapter guards every selected raw record, including old non-bundle
// artifacts, before legacy canonicalization can normalize Unicode spellings.
func (r *workflowRunner) compositionBundleReference(s *scvworkflow.Session, ref, kind string, replay bool) (scvworkflow.Record, error) {
	raw, err := s.Get("records", ref)
	if err != nil {
		return scvworkflow.Record{}, err
	}
	if err = knowledgeengine.ValidateSCVBundleUnicode(raw); err != nil {
		return scvworkflow.Record{}, err
	}
	record, err := scvworkflow.ReadRecord(raw)
	if err != nil {
		return record, err
	}
	if record.Digest != ref || record.Kind != kind {
		return record, fmt.Errorf("bundle workflow reference has a different exact record identity/kind")
	}
	if replay {
		if r.owner == nil {
			return record, fmt.Errorf("original owner unavailable")
		}
		_, err = r.replay(record)
	}
	return record, err
}

// Immutable reuse must inspect the actual retained bytes before acknowledging
// a checkpoint: the shared legacy store intentionally retains its old Unicode
// normalization rules. This v2 adapter requires strict raw spellings.
func putBundleWorkflowObject(s *scvworkflow.Session, kind string, raw []byte) (string, error) {
	if err := knowledgeengine.ValidateSCVBundleUnicode(raw); err != nil {
		return "", err
	}
	ref, err := s.Put(kind, raw)
	if err != nil {
		return "", err
	}
	retained, err := s.Get(kind, ref)
	if err != nil {
		return "", err
	}
	if err = knowledgeengine.ValidateSCVBundleUnicode(retained); err != nil {
		return "", err
	}
	return ref, nil
}
