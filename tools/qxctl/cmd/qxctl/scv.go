package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgebinding"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvstate"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/ssiagclient"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const featureSCVAdministration = "ssfv:symphony:qxctl.scv-administration"

type scvOptions struct {
	domain, prefix, version, repository, input, topsID, stateRoot, sourceID, operationID string
	jsonOutput                                                                           bool
}
type scvOperation struct{ leaf, operation, interaction, mutability string }

var scvOperations = []scvOperation{
	{"inspect", "inspect", "inspect", "read_only"},
	{"provider-onboard", "provider_onboard", "discover", "proposal_only"},
	{"source-plan", "source_plan", "propose", "proposal_only"},
	{"source-check", "source_status", "inspect", "read_only"},
	{"capture-import", "capture_import", "invoke", "evidence_only"},
	{"capture-compare", "capture_compare", "validate", "evidence_only"},
	{"interpret", "knowledge_interpret", "invoke", "evidence_only"},
	{"graph", "graph_build", "invoke", "evidence_only"},
	{"query", "graph_query", "query", "read_only"},
	{"diff", "graph_diff", "validate", "evidence_only"},
	{"explain", "graph_explain", "query", "read_only"},
	{"evaluate", "graph_evaluate", "validate", "evidence_only"},
}

func attachSCV(command *cobra.Command, key, operation, interaction, mutability string, protected bool) {
	spec := commandSpec(key, featureSCVAdministration, interaction)
	spec.Mutability = mutability
	for _, domain := range knowledgeengine.SCVDomains() {
		spec.BackendOperationIDs = append(spec.BackendOperationIDs, "engop:symphony:"+domain+"."+strings.ReplaceAll(operation, "_", "."))
		spec.FeatureBindings = append(spec.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:" + domain + "-engine", Interaction: interaction})
	}
	spec.InputProtocols = []string{"symphony.scv." + strings.ReplaceAll(operation, "_", "-") + "-input.v1"}
	switch key {
	case "scv.source.propose":
		spec.InputProtocols = []string{"symphony.qxctl.scv-source-proposal-input.v1"}
	case "scv.source.apply":
		spec.InputProtocols = []string{"symphony.scv.source-plan.v1"}
	case "scv.source.status", "scv.source.recover":
		spec.InputProtocols = []string{}
	case "scv.acquire":
		spec.InputProtocols = []string{"symphony.qxctl.scv-acquire-input.v1"}
		for _, domain := range knowledgeengine.SCVDomains() {
			spec.BackendOperationIDs = append(spec.BackendOperationIDs, "engop:symphony:"+domain+".source.status")
			spec.FeatureBindings = append(spec.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:" + domain + "-engine", Interaction: "inspect"})
		}
	}
	output, _ := knowledgeengine.SCVResultProtocol(operation)
	if protected && (strings.HasSuffix(key, ".apply") || strings.HasSuffix(key, ".recover")) {
		spec.AuthorityMode = "target_host_permission"
		spec.RecoveryCommandID = stringPointer("qxcmd:symphony:scv.source.recover")
		spec.FeatureBindings = append(spec.FeatureBindings, commandregistry.FeatureBinding{FeatureID: backendFeatureSSIAG, Interaction: "invoke"})
	}
	if protected {
		output = "symphony.qxctl.scv-source-result.v1"
	}
	if key == "scv.source.propose" {
		output = "symphony.scv.source-plan.v1"
	}
	spec.OutputProtocols = []string{output}
	spec.ResultValidationProtocols = []string{output}
	commandregistry.Attach(command, spec)
}

func scvFlags(command *cobra.Command, options *scvOptions) {
	command.Flags().StringVar(&options.domain, "domain", "scv", "exact installed domain: scv, schv, scev, schv-aws, schv-azure, schv-do, schv-gcp, scev-cf")
	command.Flags().StringVar(&options.prefix, "prefix", "", "exact receipt-v2 installation prefix")
	command.Flags().StringVar(&options.version, "version", "0.1.0-dev", "exact installed engine version; never latest")
	command.Flags().StringVar(&options.repository, "repo", "", "operation working directory; defaults to current directory")
	command.Flags().StringVar(&options.input, "input", "", "bounded no-follow JSON payload file")
	command.Flags().BoolVar(&options.jsonOutput, "json", false, "emit validated JSON")
	command.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
}

func newSCVCommand() *cobra.Command {
	command := structural("scv", fmt.Errorf("SCV subcommand is required"))
	for _, operation := range scvOperations {
		options := scvOptions{}
		child := &cobra.Command{Use: operation.leaf, Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error { return runSCV(operation.operation, options) }}
		attachSCV(child, "scv."+operation.leaf, operation.operation, operation.interaction, operation.mutability, false)
		scvFlags(child, &options)
		command.AddCommand(child)
	}
	options := scvOptions{}
	acquire := &cobra.Command{Use: "acquire", Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error { return runSCVAcquire(options) }}
	attachSCV(acquire, "scv.acquire", "capture_import", "invoke", "evidence_only", false)
	scvFlags(acquire, &options)
	command.AddCommand(acquire)
	source := structural("source", fmt.Errorf("source subcommand is required: propose, apply, status, recover"))
	for _, leaf := range []string{"propose", "apply", "status", "recover"} {
		options := scvOptions{}
		child := &cobra.Command{Use: leaf, Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error { return runSCVSource(leaf, options) }}
		op := map[string]string{"propose": "source_plan", "apply": "source_apply", "status": "source_status", "recover": "source_apply"}[leaf]
		interaction := map[string]string{"propose": "propose", "apply": "apply", "status": "inspect", "recover": "recover"}[leaf]
		mutability := "read_only"
		if leaf == "propose" {
			mutability = "proposal_only"
		}
		if leaf == "apply" || leaf == "recover" {
			mutability = "permission_backed_mutation"
		}
		attachSCV(child, "scv.source."+leaf, op, interaction, mutability, true)
		scvFlags(child, &options)
		child.Flags().StringVar(&options.topsID, "tops-id", "", "immutable TOPS UUID for this user-owned source store")
		child.Flags().StringVar(&options.stateRoot, "state-root", "", "owned local state root; defaults to XDG_STATE_HOME or ~/.local/state")
		child.Flags().StringVar(&options.sourceID, "source-id", "", "exact source identity")
		child.Flags().StringVar(&options.operationID, "operation-id", "", "exact stable operation identity for status or recovery")
		source.AddCommand(child)
	}
	command.AddCommand(source)
	command.AddCommand(newSCVProjectionCommand())
	return command
}

func scvWorkingDirectory(options scvOptions) (string, error) {
	start := options.repository
	if start == "" {
		var err error
		start, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	return filepath.Abs(start)
}
func scvInput(options scvOptions) (map[string]any, error) {
	raw, err := knowledgeengine.ReadPayload(options.input)
	if err != nil {
		return nil, err
	}
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&object); err != nil {
		return nil, err
	}
	return object, nil
}
func invokeSCV(options scvOptions, operation string, input any) (json.RawMessage, error) {
	cwd, err := scvWorkingDirectory(options)
	if err != nil {
		return nil, err
	}
	raw, err := knowledgeengine.SCVCanonical(input)
	if err != nil {
		return nil, err
	}
	response, err := knowledgeengine.InvokeSCVDomain(context.Background(), options.domain, options.prefix, options.version, cwd, operation, raw)
	return response.Result, err
}
func runSCV(operation string, options scvOptions) error {
	var input any = map[string]any{}
	if operation != "inspect" {
		var err error
		input, err = scvInput(options)
		if err != nil {
			return err
		}
	} else if options.input != "" {
		return fmt.Errorf("inspect does not accept --input")
	}
	result, err := invokeSCV(options, operation, input)
	if err != nil {
		return err
	}
	return outputSCV(options, result)
}
func outputSCV(options scvOptions, result json.RawMessage) error {
	// JSON is also the lossless human-readable default: no invented summary
	// drops partial evidence, source revision, or qualification fields.
	return printIndentedJSON(result)
}

func sourceResult(operation string, tx *scvstate.Transaction, result json.RawMessage, operationID string) (json.RawMessage, error) {
	var attempt any
	if operationID != "" {
		value, ok := tx.Attempt(operationID)
		if ok {
			attempt = value
		}
	}
	snapshot := tx.Snapshot()
	return knowledgeengine.SCVCanonical(map[string]any{"protocol": "symphony.qxctl.scv-source-result.v1", "operation": operation,
		"tops_id": snapshot.TOPSID, "domain": snapshot.Domain, "source_id": snapshot.SourceID, "state_digest": snapshot.StateDigest,
		"head_operation_id": snapshot.HeadOperationID, "owner_result": result, "attempt": attempt,
		"canonical_apply_enabled": false, "authorization_audit": "ssiag_policy_decision_only", "source_write_stav_receipt": nil})
}

func runSCVSource(operation string, options scvOptions) error {
	if options.stateRoot == "" {
		var err error
		options.stateRoot, err = knowledgebinding.DefaultStateRoot()
		if err != nil {
			return err
		}
	}
	store, err := scvstate.New(options.stateRoot, options.topsID, options.domain, options.sourceID)
	if err != nil {
		return err
	}
	var input map[string]any
	if operation == "propose" || operation == "apply" {
		input, err = scvInput(options)
		if err != nil {
			return err
		}
	} else if options.input != "" {
		return fmt.Errorf("status/recover use stored exact intent, not --input")
	}
	if operation == "recover" && !validSessionToken(options.operationID) {
		return fmt.Errorf("recover requires --operation-id")
	}
	var output json.RawMessage
	err = store.WithLock(func(tx *scvstate.Transaction) error {
		if operation == "status" {
			result, err := invokeSCV(options, "source_status", map[string]any{"source": tx.Current()})
			if err != nil {
				return err
			}
			output, err = sourceResult(operation, tx, result, options.operationID)
			return err
		}
		if operation == "propose" {
			if len(input) != 3 || input["operation_id"] == nil || input["desired"] == nil || input["reason"] == nil {
				return fmt.Errorf("source propose requires exactly operation_id, desired and reason; current comes from protected state")
			}
			desired, ok := input["desired"].(map[string]any)
			if !ok || desired["source_id"] != options.sourceID {
				return fmt.Errorf("desired source does not match --source-id")
			}
			input["current"] = tx.Current()
			result, err := invokeSCV(options, "source_plan", input)
			if err != nil {
				return err
			}
			// A proposal remains portable: --json emits the owner's exact Plan,
			// which is accepted as --input by source apply without rewriting.
			output = result
			return nil
		}
		installed, err := knowledgeengine.InspectSCVDomain(options.domain, options.prefix, options.version)
		if err != nil {
			return err
		}
		var plan json.RawMessage
		if operation == "recover" {
			prior, ok := tx.Attempt(options.operationID)
			if !ok {
				return fmt.Errorf("no stored operation matches --operation-id")
			}
			if prior.Intent.Installation != installed {
				return fmt.Errorf("recovery requires the exact original receipt and executable")
			}
			plan = prior.Intent.Plan
		} else {
			plan, err = knowledgeengine.SCVCanonical(input)
			if err != nil {
				return err
			}
		}
		var binding struct {
			OperationID string `json:"operation_id"`
			ChangeKind  string `json:"change_kind"`
			Source      struct {
				SourceID string `json:"source_id"`
			} `json:"source"`
		}
		if err := json.Unmarshal(plan, &binding); err != nil {
			return err
		}
		if binding.Source.SourceID != options.sourceID || !validSessionToken(binding.OperationID) {
			return fmt.Errorf("plan source or operation identity mismatch")
		}
		if options.operationID != "" && options.operationID != binding.OperationID {
			return fmt.Errorf("plan operation differs from --operation-id")
		}
		// Lost-response replay checks the retained exact plan before invoking a
		// reducer against a later head. It never repeats a committed write.
		if prior, ok := tx.Attempt(binding.OperationID); ok && prior.Status == "committed" {
			if !scvSameJSON(prior.Intent.Plan, plan) || prior.Intent.Installation != installed {
				return fmt.Errorf("operation_id binds a different committed intent")
			}
			output, err = sourceResult(operation, tx, prior.Intent.Transition, binding.OperationID)
			return err
		}
		transition, err := invokeSCV(options, "source_apply", map[string]any{"plan": plan, "current": tx.Current()})
		if err != nil {
			return err
		}
		// Reinspect after invocation to reject a receipt swap before journaling.
		rechecked, err := knowledgeengine.InspectSCVDomain(options.domain, options.prefix, options.version)
		if err != nil || rechecked != installed {
			return fmt.Errorf("source owner installation changed during reduction")
		}
		intent, err := scvstate.NewIntent(binding.OperationID, plan, transition, installed)
		if err != nil {
			return err
		}
		if _, err := tx.Prepare(intent); err != nil {
			return err
		}
		decision, err := authorizeSCVSource(options, binding.OperationID, binding.ChangeKind)
		if err != nil {
			return fmt.Errorf("source intent retained; authorization failed before source mutation: %w", err)
		}
		evidence, err := knowledgeengine.SCVCanonical(decision)
		if err != nil {
			return err
		}
		if err := tx.Commit(binding.OperationID, evidence, scvAuthorizationFreshness(decision)); err != nil {
			return err
		}
		output, err = sourceResult(operation, tx, transition, binding.OperationID)
		return err
	})
	if err != nil {
		return err
	}
	return outputSCV(options, output)
}

func scvSameJSON(a, b []byte) bool {
	var x, y any
	for _, item := range []struct {
		data   []byte
		target *any
	}{{a, &x}, {b, &y}} {
		d := json.NewDecoder(bytes.NewReader(item.data))
		d.UseNumber()
		if d.Decode(item.target) != nil {
			return false
		}
	}
	xb, _ := knowledgeengine.SCVCanonical(x)
	yb, _ := knowledgeengine.SCVCanonical(y)
	return bytes.Equal(xb, yb)
}
func scvSourceResource(options scvOptions) string {
	digest, _ := knowledgeengine.SCVDigest(map[string]any{"tops_id": options.topsID, "domain": options.domain, "source_id": options.sourceID})
	return "symphony.scv.source:" + strings.TrimPrefix(digest, "sha256:")
}
func authorizeSCVSource(options scvOptions, operationID, kind string) (ssiagclient.AuthorizationDecision, error) {
	if kind != "onboard" && kind != "relocate" && kind != "authority_change" && kind != "revise" {
		return ssiagclient.AuthorizationDecision{}, fmt.Errorf("unsupported source change kind")
	}
	return authorizeSCVRequest(options.topsID, operationID, "symphony.scv.source."+strings.ReplaceAll(kind, "_", "-"), scvSourceResource(options))
}

func authorizeSCVRequest(topsID, operationID, permissionOperation, resource string) (ssiagclient.AuthorizationDecision, error) {
	client, err := ssiagclient.NewForTOPS("user", topsID, 4*time.Second)
	if err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if _, err := requireSSIAGStatus(ctx, client, topsID, "user"); err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	requestID, err := randomUUID()
	if err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	now := time.Now().UTC().Truncate(time.Second)
	request := ssiagclient.AuthorizationRequest{Schema: "symphony.ssiag.authorization-request.v1", RequestID: requestID, CorrelationID: operationID,
		Operation: permissionOperation, Resource: resource, Audience: "qxctl", Scope: "tops:" + topsID,
		RequestedAt: now, RequestedExpiresAt: now.Add(time.Minute)}
	decision, err := client.Authorize(ctx, request)
	if err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	if err := validateSessionAuthorization(decision, request, topsID); err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	return decision, nil
}

// The decision was authenticated and fully validated before this closure is
// created. Check both exact expiry bounds again immediately before publication.
func scvAuthorizationFreshness(decision ssiagclient.AuthorizationDecision) func() error {
	return func() error {
		now := time.Now().UTC()
		if decision.ExpiresAt == nil || decision.Capability == nil || !decision.ExpiresAt.After(now) || !decision.Capability.ExpiresAt.After(now) {
			return fmt.Errorf("SSIAG authorization expired before selection publication")
		}
		return nil
	}
}
