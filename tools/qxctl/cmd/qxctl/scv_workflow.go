package main

import (
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvcorpus"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvworkflow"
	"github.com/spf13/cobra"
	"path/filepath"
	"sort"
	"strings"
)

func newSCVArtifactCommand() *cobra.Command {
	group := structural("artifact", fmt.Errorf("artifact subcommand is required: import, show, list"))
	for _, leaf := range []string{"import", "show", "list"} {
		options := scvOptions{}
		var root string
		child := &cobra.Command{Use: leaf, Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error { return runSCVArtifact(leaf, options, root) }}
		if leaf == "import" {
			scvFlagsVersion(child, &options, "0.4.0-dev")
		} else {
			scvRetainedReadFlags(child, &options)
		}
		child.Flags().StringVar(&root, "workflow-root", "", "explicit owned retained artifact and workflow root")
		child.Flags().StringVar(&options.topsID, "tops-id", "", "exact TOPS UUID for retained evidence namespace")
		interaction := map[string]string{"import": "invoke", "show": "inspect", "list": "query"}[leaf]
		mutability := "read_only"
		if leaf == "import" {
			mutability = "evidence_only"
		}
		output := "symphony.qxctl.scv-artifact.v1"
		if leaf == "list" {
			output = "symphony.qxctl.scv-artifact-list.v1"
		}
		operations := []string{}
		if leaf != "list" {
			operations = knowledgeengine.SCVArtifactOperations()
		}
		attachSCVWorkflow(child, "scv.artifact."+leaf, "symphony.qxctl.scv-artifact-"+leaf+"-input.v1", output, interaction, mutability, operations, false)
		group.AddCommand(child)
	}
	return group
}
func newSCVWorkflowCommand() *cobra.Command {
	group := structural("workflow", fmt.Errorf("workflow subcommand is required: run, status, recover"))
	for _, leaf := range []string{"run", "status", "recover"} {
		options := scvOptions{}
		var root, corpusRoot string
		child := &cobra.Command{Use: leaf, Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error { return runSCVWorkflow(leaf, options, root, corpusRoot) }}
		if leaf == "status" {
			scvRetainedReadFlags(child, &options)
		} else {
			scvFlagsVersion(child, &options, "0.4.0-dev")
		}
		child.Flags().StringVar(&root, "workflow-root", "", "explicit owned retained artifact and workflow root")
		if leaf != "status" {
			child.Flags().StringVar(&corpusRoot, "corpus-root", "", "explicit existing immutable corpus root; pinned by the initial run")
		}
		child.Flags().StringVar(&options.topsID, "tops-id", "", "exact TOPS UUID for retained evidence namespace")
		interaction := map[string]string{"run": "invoke", "status": "inspect", "recover": "recover"}[leaf]
		mutability := "evidence_only"
		operations := []string{"capture_index", "corpus_build", "corpus_query", "profile_prepare", "provider_interpret", "connection_evaluate", "connection_reassess", "knowledge_interpret"}
		if leaf == "status" {
			mutability = "read_only"
			operations = nil
		}
		attachSCVWorkflow(child, "scv.workflow."+leaf, "symphony.qxctl.scv-workflow-"+leaf+"-input.v1", "symphony.qxctl.scv-workflow-result.v1", interaction, mutability, operations, leaf != "status")
		group.AddCommand(child)
	}
	return group
}

func scvRetainedReadFlags(command *cobra.Command, options *scvOptions) {
	command.Flags().StringVar(&options.input, "input", "", "bounded no-follow JSON payload file")
	command.Flags().BoolVar(&options.jsonOutput, "json", false, "emit validated JSON")
	command.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
}
func attachSCVWorkflow(command *cobra.Command, key, input, output, interaction, mutability string, operations []string, recovery bool) {
	spec := commandSpec(key, featureSCVAdministration, interaction)
	spec.Mutability = mutability
	spec.InputProtocols = []string{input}
	spec.OutputProtocols = []string{output}
	spec.ResultValidationProtocols = []string{output}
	for _, domain := range knowledgeengine.SCVDomains() {
		seen := map[string]bool{}
		for _, op := range operations {
			ownerInteraction := knowledgeengine.SCVOperationInteraction(op)
			spec.BackendOperationIDs = append(spec.BackendOperationIDs, "engop:symphony:"+domain+"."+strings.ReplaceAll(op, "_", "."))
			if !seen[ownerInteraction] {
				spec.FeatureBindings = append(spec.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:" + domain + "-engine", Interaction: ownerInteraction})
				seen[ownerInteraction] = true
			}
		}
	}
	if recovery {
		spec.RecoveryCommandID = stringPointer("qxcmd:symphony:scv.workflow.recover")
	}
	commandregistry.Attach(command, spec)
}

type workflowRunner struct {
	options      scvOptions
	installation knowledgeengine.Installation
	inspect      func(string, string, string) (knowledgeengine.Installation, error)
	owner        func(knowledgeengine.Installation, string, any) (json.RawMessage, error)
	corpusExport func(scvcorpus.Store, knowledgeengine.Installation, map[string]any) (json.RawMessage, error)
	// Tests inject failure immediately after an immutable checkpoint was flushed.
	afterCheckpoint func(string) error
}

func newWorkflowRunner(options scvOptions, requireCurrent bool) (*workflowRunner, error) {
	r := &workflowRunner{options: options, inspect: knowledgeengine.InspectSCVDomain}
	if requireCurrent {
		if !knowledgeengine.SCVSupports(options.version, "workflow") {
			return nil, fmt.Errorf("artifact/workflow execution requires an exact engine release supporting this interface")
		}
		var err error
		r.installation, err = r.inspect(options.domain, options.prefix, options.version)
		if err != nil {
			return nil, err
		}
	}
	r.owner = func(inst knowledgeengine.Installation, op string, input any) (json.RawMessage, error) {
		current, err := r.inspect(inst.Role, inst.Prefix, inst.Version)
		if err != nil || current != inst {
			return nil, fmt.Errorf("retained owner requires original exact installation: %s %s", inst.Role, inst.Version)
		}
		selected := options
		selected.domain = inst.Role
		selected.prefix = inst.Prefix
		selected.version = inst.Version
		result, err := invokeSCV(selected, op, input)
		if err != nil {
			return nil, err
		}
		current, err = r.inspect(inst.Role, inst.Prefix, inst.Version)
		if err != nil || current != inst {
			return nil, fmt.Errorf("retained owner installation changed during invocation")
		}
		return result, nil
	}
	r.corpusExport = func(store scvcorpus.Store, inst knowledgeengine.Installation, input map[string]any) (json.RawMessage, error) {
		current, err := r.inspect(inst.Role, inst.Prefix, inst.Version)
		if err != nil || current != inst {
			return nil, fmt.Errorf("workflow corpus owner installation changed")
		}
		cr := &corpusRunner{installation: inst, owner: func(op string, input any) (json.RawMessage, error) { return r.owner(inst, op, input) }}
		return cr.execute(store, "export", input)
	}
	return r, nil
}
func workflowRaw(v any) (json.RawMessage, error) {
	raw, err := scvworkflow.Canonical(v)
	if err != nil {
		return nil, err
	}
	if err = knowledgeengine.ValidateJSONObject(raw, scvworkflow.MaxInputBytes); err != nil {
		return nil, err
	}
	return raw, nil
}
func (r *workflowRunner) replay(record scvworkflow.Record) (json.RawMessage, error) {
	result, err := r.owner(record.Installation, record.Operation, record.Input)
	if err != nil {
		return nil, err
	}
	if !scvworkflow.Same(result, record.Artifact) {
		return nil, fmt.Errorf("retained artifact %s differs from exact owner replay", record.Digest)
	}
	return record.Artifact, nil
}
func (r *workflowRunner) reference(s *scvworkflow.Session, ref, kind string, replay bool) (scvworkflow.Record, error) {
	raw, err := s.Get("records", ref)
	if err != nil {
		return scvworkflow.Record{}, err
	}
	record, err := scvworkflow.ReadRecord(raw)
	if err != nil {
		return record, err
	}
	if record.Kind != kind {
		return record, fmt.Errorf("record %s is %s, expected %s", ref, record.Kind, kind)
	}
	if replay {
		_, err = r.replay(record)
	}
	return record, err
}
func runSCVArtifact(operation string, options scvOptions, root string) error {
	if root == "" {
		return fmt.Errorf("--workflow-root is required; no implicit artifact selection")
	}
	store, err := scvworkflow.New(root, options.topsID)
	if err != nil {
		return err
	}
	input, err := scvInput(options)
	if err != nil {
		return err
	}
	r, err := newWorkflowRunner(options, operation == "import")
	if err != nil {
		return err
	}
	result, err := r.artifact(store, operation, input)
	if err != nil {
		return err
	}
	return outputSCV(options, result)
}
func (r *workflowRunner) artifact(store scvworkflow.Store, operation string, input map[string]any) (json.RawMessage, error) {
	var result json.RawMessage
	err := store.With("", operation == "import", func(s *scvworkflow.Session, _ *scvworkflow.Run) error {
		switch operation {
		case "import":
			if err := exactCorpusFields(input, "operation", "input", "result"); err != nil {
				return err
			}
			op, ok := input["operation"].(string)
			if !ok || scvworkflow.Kind(op) == "" {
				return fmt.Errorf("unsupported retained artifact operation")
			}
			if !knowledgeengine.SCVArtifactSupported(r.installation.Version, op) {
				return fmt.Errorf("retained artifact is unsupported by the exact selected installation")
			}
			payload, err := workflowRaw(input["input"])
			if err != nil {
				return err
			}
			native, err := r.owner(r.installation, op, payload)
			if err != nil {
				return err
			}
			if input["result"] != nil {
				expected, err := workflowRaw(input["result"])
				if err != nil {
					return err
				}
				if !scvworkflow.Same(native, expected) {
					return fmt.Errorf("supplied artifact differs from exact owner replay")
				}
			}
			record, err := scvworkflow.NewRecord(op, r.installation, payload, native)
			if err != nil {
				return err
			}
			result, err = scvworkflow.Canonical(record)
			if err != nil {
				return err
			}
			_, err = s.Put("records", result)
			return err
		case "show":
			if err := exactCorpusFields(input, "record_digest"); err != nil {
				return err
			}
			ref, err := corpusDigest(input["record_digest"], "record_digest")
			if err != nil {
				return err
			}
			raw, err := s.Get("records", ref)
			if err != nil {
				return err
			}
			record, err := scvworkflow.ReadRecord(raw)
			if err != nil {
				return err
			}
			if _, err = r.replay(record); err != nil {
				return err
			}
			result = raw
			return nil
		case "list":
			if err := exactCorpusFields(input, "after_digest", "limit"); err != nil {
				return err
			}
			after := ""
			var err error
			if input["after_digest"] != nil {
				after, err = corpusDigest(input["after_digest"], "after_digest")
				if err != nil {
					return err
				}
			}
			n, ok := input["limit"].(json.Number)
			if !ok {
				return fmt.Errorf("list limit must be an integer")
			}
			limit, err := n.Int64()
			if err != nil || limit < 1 || limit > 128 {
				return fmt.Errorf("list limit must be 1..128")
			}
			records, next, err := s.List(after, int(limit))
			if err != nil {
				return err
			}
			metadata := []any{}
			for _, record := range records {
				metadata = append(metadata, map[string]any{"kind": record.Kind, "operation": record.Operation, "installation": record.Installation, "artifact_digest": record.ArtifactDigest, "digest": record.Digest})
			}
			result, err = scvworkflow.Canonical(map[string]any{"protocol": "symphony.qxctl.scv-artifact-list.v1", "records": metadata, "next_after_digest": next, "validation": "sealed_envelopes_only"})
			return err
		}
		return fmt.Errorf("unknown artifact operation")
	})
	return result, err
}
func runSCVWorkflow(operation string, options scvOptions, root, corpusRoot string) error {
	if root == "" {
		return fmt.Errorf("--workflow-root is required; no implicit run selection")
	}
	store, err := scvworkflow.New(root, options.topsID)
	if err != nil {
		return err
	}
	input, err := scvInput(options)
	if err != nil {
		return err
	}
	r, err := newWorkflowRunner(options, operation != "status")
	if err != nil {
		return err
	}
	result, err := r.execute(store, operation, input, corpusRoot)
	if err != nil {
		return err
	}
	return outputSCV(options, result)
}
func workflowRefs(value any, label string) ([]string, error) {
	list, ok := value.([]any)
	if !ok || len(list) > 16 {
		return nil, fmt.Errorf("%s requires at most 16 exact references", label)
	}
	out := []string{}
	seen := map[string]bool{}
	for _, v := range list {
		ref, err := corpusDigest(v, label)
		if err != nil {
			return nil, err
		}
		if seen[ref] {
			return nil, fmt.Errorf("duplicate %s", label)
		}
		seen[ref] = true
		out = append(out, ref)
	}
	return out, nil
}
func validateWorkflowInput(input map[string]any) error {
	if err := exactCorpusFields(input, "operation_id", "corpus", "profile_bindings", "interpretation_refs", "knowledge_refs", "selection_policy", "query_time", "connections", "prior_evaluation_ref"); err != nil {
		return err
	}
	if _, err := corpusString(input["operation_id"], "operation_id"); err != nil {
		return err
	}
	for _, name := range []string{"interpretation_refs", "knowledge_refs"} {
		if _, err := workflowRefs(input[name], name); err != nil {
			return err
		}
	}
	if input["prior_evaluation_ref"] != nil {
		if _, err := corpusDigest(input["prior_evaluation_ref"], "prior_evaluation_ref"); err != nil {
			return err
		}
	}
	bindings, ok := input["profile_bindings"].([]any)
	if !ok || len(bindings) > 16 {
		return fmt.Errorf("profile_bindings must contain at most 16")
	}
	if _, ok := input["query_time"].(string); !ok {
		return fmt.Errorf("query_time must be explicit")
	}
	if _, ok := input["connections"].([]any); !ok {
		return fmt.Errorf("connections must be an explicit owner requirements array")
	}
	if input["corpus"] == nil {
		if input["selection_policy"] != nil {
			return fmt.Errorf("selection_policy must be null when reusing retained artifacts without a new corpus interpretation")
		}
		if len(bindings) != 0 {
			return fmt.Errorf("profile bindings require an explicit corpus selection")
		}
		return nil
	}
	if _, ok := input["selection_policy"].(map[string]any); !ok {
		return fmt.Errorf("new corpus interpretation requires an explicit selection_policy object")
	}
	corpus, ok := input["corpus"].(map[string]any)
	if !ok {
		return fmt.Errorf("invalid workflow corpus selection")
	}
	if err := exactCorpusFields(corpus, "domain", "snapshot_digest", "member_ids", "selection", "max_age_seconds"); err != nil {
		return err
	}
	if _, err := corpusString(corpus["domain"], "corpus domain"); err != nil {
		return err
	}
	if _, err := corpusDigest(corpus["snapshot_digest"], "snapshot_digest"); err != nil {
		return err
	}
	members, ok := corpus["member_ids"].([]any)
	if !ok || len(members) < 1 || len(members) > 16 {
		return fmt.Errorf("workflow requires 1..16 explicit corpus member IDs")
	}
	selected := map[string]bool{}
	for _, raw := range members {
		id, err := corpusString(raw, "member_id")
		if err != nil {
			return err
		}
		if selected[id] {
			return fmt.Errorf("duplicate selected member ID")
		}
		selected[id] = true
	}
	if len(bindings) == 0 {
		return fmt.Errorf("corpus selection requires explicit profile bindings")
	}
	seen := map[string]bool{}
	covered := map[string]bool{}
	for _, raw := range bindings {
		binding, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid profile binding")
		}
		if err := exactCorpusFields(binding, "profile_ref", "member_id"); err != nil {
			return err
		}
		ref, err := corpusDigest(binding["profile_ref"], "profile_ref")
		if err != nil {
			return err
		}
		id, err := corpusString(binding["member_id"], "member_id")
		if err != nil {
			return err
		}
		if seen[ref] || !selected[id] {
			return fmt.Errorf("duplicate profile reference or unselected member binding")
		}
		seen[ref] = true
		covered[id] = true
	}
	if len(covered) != len(selected) {
		return fmt.Errorf("each selected corpus member requires a profile binding")
	}
	return nil
}

func (r *workflowRunner) execute(store scvworkflow.Store, operation string, input map[string]any, corpusRoot string) (json.RawMessage, error) {
	var proposed scvworkflow.Run
	if corpusRoot != "" && (!filepath.IsAbs(corpusRoot) || filepath.Clean(corpusRoot) != corpusRoot || corpusRoot == "/") {
		return nil, fmt.Errorf("workflow corpus root must be a clean absolute descendant path")
	}
	if operation == "run" {
		if err := validateWorkflowInput(input); err != nil {
			return nil, err
		}
		var corpusOwner *knowledgeengine.Installation
		if input["corpus"] != nil {
			if corpusRoot == "" {
				return nil, fmt.Errorf("initial workflow corpus selection requires --corpus-root")
			}
			c := input["corpus"].(map[string]any)
			installed, err := r.inspect(c["domain"].(string), r.installation.Prefix, r.installation.Version)
			if err != nil {
				return nil, err
			}
			corpusOwner = &installed
		}
		raw, err := workflowRaw(input)
		if err != nil {
			return nil, err
		}
		proposed, err = scvworkflow.NewRun(input["operation_id"].(string), r.installation, corpusOwner, corpusRoot, raw)
		if err != nil {
			return nil, err
		}
	} else if operation == "recover" || operation == "status" {
		if err := exactCorpusFields(input, "operation_id"); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unknown workflow operation")
	}
	id, err := corpusString(input["operation_id"], "operation_id")
	if err != nil {
		return nil, err
	}
	var result json.RawMessage
	err = store.With(id, operation != "status", func(s *scvworkflow.Session, existing *scvworkflow.Run) error {
		run := proposed
		if existing != nil {
			run = *existing
			if operation == "run" && run.IntentDigest != proposed.IntentDigest {
				return fmt.Errorf("operation ID already binds a different exact workflow intent")
			}
			if operation != "status" && run.Installation != r.installation {
				return fmt.Errorf("workflow recovery requires original exact owner installation")
			}
			if corpusRoot != "" && corpusRoot != run.CorpusRoot {
				return fmt.Errorf("workflow corpus root differs from pinned intent")
			}
		} else if operation != "run" {
			return fmt.Errorf("retained workflow operation not found")
		}
		pinned, err := scvworkflow.Decode(run.Request)
		if err != nil {
			return err
		}
		if err = validateWorkflowInput(pinned); err != nil {
			return err
		}
		if operation == "status" {
			if err = r.verifyCheckpoints(store, s, run, false); err != nil {
				return err
			}
			result, err = workflowResult(run, "sealed_checkpoints_only")
			return err
		}
		if existing == nil {
			run, err = s.Save(run)
			if err != nil {
				return err
			}
			if err = r.checkpoint("prepared"); err != nil {
				return err
			}
		}
		if err = r.verifyCheckpoints(store, s, run, true); err != nil {
			return err
		}
		interpretations := []any{}
		if pinned["corpus"] != nil {
			payload, err := r.interpretInput(store, s, run, pinned, true)
			if err != nil {
				return err
			}
			record, err := r.stage(s, &run, "interpret", "provider_interpret", payload)
			if err != nil {
				return err
			}
			interpretations = append(interpretations, record.Artifact)
		}
		for _, raw := range pinned["interpretation_refs"].([]any) {
			record, err := r.reference(s, raw.(string), "interpretation", true)
			if err != nil {
				return err
			}
			interpretations = append(interpretations, record.Artifact)
		}
		knowledge := []any{}
		for _, raw := range pinned["knowledge_refs"].([]any) {
			record, err := r.reference(s, raw.(string), "knowledge", true)
			if err != nil {
				return err
			}
			knowledge = append(knowledge, record.Artifact)
		}
		payload, err := workflowRaw(map[string]any{"interpretations": interpretations, "additional_knowledge": knowledge, "query_time": pinned["query_time"], "connections": pinned["connections"]})
		if err != nil {
			return err
		}
		evaluation, err := r.stage(s, &run, "evaluate", "connection_evaluate", payload)
		if err != nil {
			return err
		}
		if pinned["prior_evaluation_ref"] != nil {
			prior, err := r.reference(s, pinned["prior_evaluation_ref"].(string), "evaluation", true)
			if err != nil {
				return err
			}
			payload, err := workflowRaw(map[string]any{"before": prior.Artifact, "after": evaluation.Artifact})
			if err != nil {
				return err
			}
			if _, err = r.stage(s, &run, "reassess", "connection_reassess", payload); err != nil {
				return err
			}
		}
		if !run.Complete {
			run.Complete = true
			run, err = s.Save(run)
			if err != nil {
				return err
			}
			if err = r.checkpoint("complete"); err != nil {
				return err
			}
		}
		result, err = workflowResult(run, "owner_replayed")
		return err
	})
	return result, err
}
func (r *workflowRunner) checkpoint(stage string) error {
	if r.afterCheckpoint != nil {
		return r.afterCheckpoint(stage)
	}
	return nil
}
func (r *workflowRunner) stage(s *scvworkflow.Session, run *scvworkflow.Run, name, operation string, payload json.RawMessage) (scvworkflow.Record, error) {
	wrapper, err := scvworkflow.Seal(map[string]any{"protocol": "symphony.qxctl.scv-workflow-payload.v1", "input": payload})
	if err != nil {
		return scvworkflow.Record{}, err
	}
	inputRef, err := scvworkflow.Digest(wrapper)
	if err != nil {
		return scvworkflow.Record{}, err
	}
	c, exists := run.Stages[name]
	if exists && c.InputRef != inputRef {
		return scvworkflow.Record{}, fmt.Errorf("retained stage %s input differs from exact pinned request/artifacts", name)
	}
	if !exists {
		if _, err = s.Put("payloads", wrapper); err != nil {
			return scvworkflow.Record{}, err
		}
		run.Stages[name] = scvworkflow.Checkpoint{InputRef: inputRef}
		*run, err = s.Save(*run)
		if err != nil {
			return scvworkflow.Record{}, err
		}
		if err = r.checkpoint(name + "_input"); err != nil {
			return scvworkflow.Record{}, err
		}
		c = run.Stages[name]
	}
	if c.ResultRef != "" {
		return r.reference(s, c.ResultRef, scvworkflow.Kind(operation), true)
	}
	result, err := r.owner(run.Installation, operation, payload)
	if err != nil {
		return scvworkflow.Record{}, err
	}
	record, err := scvworkflow.NewRecord(operation, run.Installation, payload, result)
	if err != nil {
		return record, err
	}
	raw, err := scvworkflow.Canonical(record)
	if err != nil {
		return record, err
	}
	ref, err := s.Put("records", raw)
	if err != nil {
		return record, err
	}
	if err = r.checkpoint(name + "_artifact"); err != nil {
		return record, err
	}
	run.Stages[name] = scvworkflow.Checkpoint{InputRef: inputRef, ResultRef: ref}
	*run, err = s.Save(*run)
	if err != nil {
		return record, err
	}
	return record, r.checkpoint(name + "_result")
}
func (r *workflowRunner) verifyCheckpoints(store scvworkflow.Store, s *scvworkflow.Session, run scvworkflow.Run, replay bool) error {
	request, err := scvworkflow.Decode(run.Request)
	if err != nil {
		return err
	}
	for _, name := range []string{"interpret", "evaluate", "reassess"} {
		c, ok := run.Stages[name]
		if !ok {
			continue
		}
		raw, err := s.Get("payloads", c.InputRef)
		if err != nil {
			return err
		}
		payload, err := scvworkflow.Decode(raw)
		if err != nil {
			return err
		}
		var expected json.RawMessage
		switch name {
		case "interpret":
			expected, err = r.interpretInput(store, s, run, request, false)
		case "evaluate":
			interpretations, knowledge := []any{}, []any{}
			if run.Stages["interpret"].ResultRef != "" {
				record, e := r.reference(s, run.Stages["interpret"].ResultRef, "interpretation", false)
				if e != nil {
					return e
				}
				interpretations = append(interpretations, record.Artifact)
			}
			for _, ref := range request["interpretation_refs"].([]any) {
				record, e := r.reference(s, ref.(string), "interpretation", false)
				if e != nil {
					return e
				}
				interpretations = append(interpretations, record.Artifact)
			}
			for _, ref := range request["knowledge_refs"].([]any) {
				record, e := r.reference(s, ref.(string), "knowledge", false)
				if e != nil {
					return e
				}
				knowledge = append(knowledge, record.Artifact)
			}
			expected, err = workflowRaw(map[string]any{"interpretations": interpretations, "additional_knowledge": knowledge, "query_time": request["query_time"], "connections": request["connections"]})
		case "reassess":
			prior, e := r.reference(s, request["prior_evaluation_ref"].(string), "evaluation", false)
			if e != nil {
				return e
			}
			current, e := r.reference(s, run.Stages["evaluate"].ResultRef, "evaluation", false)
			if e != nil {
				return e
			}
			expected, err = workflowRaw(map[string]any{"before": prior.Artifact, "after": current.Artifact})
		}
		if err != nil {
			return err
		}
		if !scvcorpus.Same(payload["input"], expected) {
			return fmt.Errorf("retained stage %s input differs from exact pinned request/artifacts", name)
		}
		if c.ResultRef == "" {
			continue
		}
		operation := map[string]string{"interpret": "provider_interpret", "evaluate": "connection_evaluate", "reassess": "connection_reassess"}[name]
		record, err := r.reference(s, c.ResultRef, scvworkflow.Kind(operation), replay)
		if err != nil {
			return err
		}
		if record.Operation != operation || record.Installation != run.Installation || !scvcorpus.Same(payload["input"], record.Input) {
			return fmt.Errorf("workflow checkpoint is not bound to its pinned owner input")
		}
	}
	return nil
}

// Status verifies exact storage/reference linkage without repeating native
// capture indexing, corpus reduction or profile semantics. The owner-replayed
// run/recover path still reconstructs the full corpus through corpusRunner.
func retainedWorkflowSelection(store scvcorpus.Store, input map[string]any) (json.RawMessage, error) {
	var result json.RawMessage
	err := store.WithRead(func(s *scvcorpus.Session, _ *scvcorpus.Job) error {
		raw, err := s.Get("snapshots", input["snapshot_digest"].(string))
		if err != nil {
			return err
		}
		snapshot, err := scvcorpus.Decode(raw)
		if err != nil {
			return err
		}
		selection, ok := input["selection"].(string)
		if !ok || (selection != "latest_attempt" && selection != "last_complete") {
			return fmt.Errorf("invalid retained corpus selection")
		}
		members, ok := snapshot["members"].([]any)
		if !ok {
			return fmt.Errorf("invalid retained corpus snapshot members")
		}
		byID := map[string]map[string]any{}
		for _, raw := range members {
			member, ok := raw.(map[string]any)
			if !ok {
				return fmt.Errorf("invalid retained corpus member")
			}
			id, ok := member["member_id"].(string)
			if !ok || byID[id] != nil {
				return fmt.Errorf("invalid retained member identity")
			}
			byID[id] = member
		}
		queryMembers, captures := []any{}, []any{}
		// The native query orders selected members by ID, irrespective of caller order.
		ids := []string{}
		for _, raw := range input["member_ids"].([]any) {
			ids = append(ids, raw.(string))
		}
		sort.Strings(ids)
		for _, id := range ids {
			member := byID[id]
			if member == nil {
				return fmt.Errorf("missing pinned corpus member %s", id)
			}
			index, ok := member[selection].(map[string]any)
			if !ok {
				return fmt.Errorf("pinned member has no selected capture")
			}
			indexRef, err := corpusDigest(index["digest"], "index digest")
			if err != nil {
				return err
			}
			stored, err := s.Get("indexes", indexRef)
			if err != nil {
				return err
			}
			if !scvcorpus.Same(index, json.RawMessage(stored)) {
				return fmt.Errorf("retained corpus member/index mismatch")
			}
			captureRef, err := corpusDigest(index["capture_digest"], "capture digest")
			if err != nil {
				return err
			}
			capture, err := s.Get("captures", captureRef)
			if err != nil {
				return err
			}
			captures = append(captures, json.RawMessage(capture))
			queryMembers = append(queryMembers, map[string]any{"member_id": id, "selected": index})
		}
		result, err = scvworkflow.Canonical(map[string]any{"query": map[string]any{"members": queryMembers}, "captures": captures})
		return err
	})
	return result, err
}
func (r *workflowRunner) interpretInput(store scvworkflow.Store, s *scvworkflow.Session, run scvworkflow.Run, input map[string]any, replay bool) (json.RawMessage, error) {
	c := input["corpus"].(map[string]any)
	if run.CorpusInstallation == nil || run.CorpusInstallation.Role != c["domain"] {
		return nil, fmt.Errorf("missing exact corpus installation")
	}
	corpus, err := scvcorpus.New(run.CorpusRoot, store.TOPSID, c["domain"].(string))
	if err != nil {
		return nil, err
	}
	queryInput := map[string]any{"snapshot_digest": c["snapshot_digest"], "member_ids": c["member_ids"], "selection": c["selection"], "query_time": input["query_time"], "max_age_seconds": c["max_age_seconds"]}
	var exported json.RawMessage
	if replay {
		exported, err = r.corpusExport(corpus, *run.CorpusInstallation, queryInput)
	} else {
		exported, err = retainedWorkflowSelection(corpus, queryInput)
	}
	if err != nil {
		return nil, err
	}
	value, err := scvworkflow.Decode(exported)
	if err != nil {
		return nil, err
	}
	query, ok := value["query"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid corpus export query")
	}
	members, ok := query["members"].([]any)
	if !ok {
		return nil, fmt.Errorf("invalid corpus export members")
	}
	selected := map[string]string{}
	for _, raw := range members {
		m, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid selected corpus member")
		}
		id, ok := m["member_id"].(string)
		if !ok {
			return nil, fmt.Errorf("missing selected member identity")
		}
		index, ok := m["selected"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("requested member %s has no capture under explicit selection", id)
		}
		digest, err := corpusDigest(index["capture_digest"], "selected capture digest")
		if err != nil {
			return nil, err
		}
		selected[id] = digest
	}
	profiles, bindings := []any{}, []any{}
	for _, raw := range input["profile_bindings"].([]any) {
		b := raw.(map[string]any)
		record, err := r.reference(s, b["profile_ref"].(string), "profile", replay)
		if err != nil {
			return nil, err
		}
		capture, ok := selected[b["member_id"].(string)]
		if !ok {
			return nil, fmt.Errorf("profile member missing from corpus selection")
		}
		profiles = append(profiles, record.Artifact)
		bindings = append(bindings, map[string]any{"profile_digest": record.ArtifactDigest, "capture_digest": capture})
	}
	return workflowRaw(map[string]any{"captures": value["captures"], "profiles": profiles, "bindings": bindings, "selection_policy": input["selection_policy"]})
}
func workflowResult(run scvworkflow.Run, validation string) (json.RawMessage, error) {
	status := "prepared"
	if run.Stages["interpret"].ResultRef != "" {
		status = "interpreted"
	}
	if run.Stages["evaluate"].ResultRef != "" {
		status = "evaluated"
	}
	if run.Complete {
		status = "complete"
	}
	var evaluation, reassessment any
	if ref := run.Stages["evaluate"].ResultRef; ref != "" {
		evaluation = ref
	}
	if ref := run.Stages["reassess"].ResultRef; ref != "" {
		reassessment = ref
	}
	return scvworkflow.Canonical(map[string]any{"protocol": "symphony.qxctl.scv-workflow-result.v1", "run": run, "status": status, "validation": validation, "evaluation_ref": evaluation, "reassessment_ref": reassessment})
}
