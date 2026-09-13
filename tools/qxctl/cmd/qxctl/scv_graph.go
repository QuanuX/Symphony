package main

import (
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgebinding"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvgraph"
	"github.com/spf13/cobra"
	"strings"
	"time"
)

func newSCVProjectionCommand() *cobra.Command {
	group := structural("projection", fmt.Errorf("projection subcommand is required: select, status, recover"))
	for _, leaf := range []string{"select", "status", "recover"} {
		options := scvOptions{}
		var graphID string
		child := &cobra.Command{Use: leaf, Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error { return runSCVProjection(leaf, options, graphID) }}
		scvFlags(child, &options)
		child.Flags().StringVar(&options.topsID, "tops-id", "", "immutable TOPS UUID for graph selection")
		child.Flags().StringVar(&options.stateRoot, "state-root", "", "owned local graph state root")
		child.Flags().StringVar(&options.operationID, "operation-id", "", "stable graph selection operation identity")
		child.Flags().StringVar(&graphID, "graph-id", "", "stable installation-local graph selection identity")
		interaction := map[string]string{"select": "configure", "status": "inspect", "recover": "recover"}[leaf]
		spec := commandSpec("scv.projection."+leaf, featureSCVAdministration, interaction)
		spec.InputProtocols = []string{}
		if leaf == "select" {
			spec.InputProtocols = []string{"symphony.qxctl.scv-projection-input.v1"}
		}
		spec.OutputProtocols = []string{"symphony.qxctl.scv-projection-result.v1"}
		spec.ResultValidationProtocols = spec.OutputProtocols
		for _, domain := range knowledgeengine.SCVDomains() {
			spec.BackendOperationIDs = append(spec.BackendOperationIDs, "engop:symphony:"+domain+".graph.query")
			spec.FeatureBindings = append(spec.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:" + domain + "-engine", Interaction: "query"})
		}
		if leaf != "status" {
			spec.Mutability = "permission_backed_mutation"
			spec.AuthorityMode = "target_host_permission"
			spec.RecoveryCommandID = stringPointer("qxcmd:symphony:scv.projection.recover")
			spec.FeatureBindings = append(spec.FeatureBindings, commandregistry.FeatureBinding{FeatureID: backendFeatureSSIAG, Interaction: "invoke"})
		}
		commandregistry.Attach(child, spec)
		group.AddCommand(child)
	}
	return group
}

func graphResource(options scvOptions, graphID string) string {
	digest, _ := knowledgeengine.SCVDigest(map[string]any{"tops_id": options.topsID, "domain": options.domain, "graph_id": graphID})
	return "symphony.scv.graph:" + strings.TrimPrefix(digest, "sha256:")
}
func runSCVProjection(operation string, options scvOptions, graphID string) error {
	if options.stateRoot == "" {
		var err error
		options.stateRoot, err = knowledgebinding.DefaultStateRoot()
		if err != nil {
			return err
		}
	}
	store, err := scvgraph.New(options.stateRoot, options.topsID, options.domain, graphID)
	if err != nil {
		return err
	}
	if operation != "select" && options.input != "" {
		return fmt.Errorf("projection status/recover reads retained evidence, not --input")
	}
	var selected scvgraph.Document
	if operation == "status" {
		selected, err = store.Inspect()
		if err != nil {
			return err
		}
	} else {
		installed, err := knowledgeengine.InspectSCVDomain(options.domain, options.prefix, options.version)
		if err != nil {
			return err
		}
		var intent scvgraph.Intent
		if operation == "recover" {
			current, err := store.Inspect()
			if err != nil {
				return err
			}
			attempt, ok := current.Operations[options.operationID]
			if !ok {
				return fmt.Errorf("exact graph operation was not found")
			}
			if attempt.Intent.Installation != installed {
				return fmt.Errorf("graph recovery requires the original exact installation")
			}
			intent = attempt.Intent
		} else {
			input, err := scvInput(options)
			if err != nil {
				return err
			}
			if len(input) != 3 {
				return fmt.Errorf("projection select requires graph, expected_graph_digest and expected_generation")
			}
			graph, ok := input["graph"]
			if !ok {
				return fmt.Errorf("graph is required")
			}
			value, ok := input["expected_graph_digest"]
			if !ok {
				return fmt.Errorf("expected_graph_digest is required, null for absent head")
			}
			var expected *string
			generation, ok := input["expected_generation"].(json.Number)
			if !ok {
				return fmt.Errorf("expected_generation must be an exact integer")
			}
			generationValue, err := generation.Int64()
			if err != nil || generationValue < 0 || generationValue >= 128 {
				return fmt.Errorf("expected_generation must be within retained revision bounds")
			}
			if value != nil {
				d, ok := value.(string)
				if !ok {
					return fmt.Errorf("expected_graph_digest must be null or a digest")
				}
				expected = &d
			}
			raw, err := knowledgeengine.SCVCanonical(graph)
			if err != nil {
				return err
			}
			intent, err = scvgraph.NewIntent(options.operationID, expected, int(generationValue), raw, installed)
			if err != nil {
				return err
			}
		}
		queryTime := time.Now().UTC().Truncate(time.Second).Format(time.RFC3339)
		// The C++ owner revalidates the graph's exact captures, claims, native
		// structure and digest; the storage adapter never interprets them.
		if _, err := invokeSCV(options, "graph_query", map[string]any{"graph": intent.Graph, "query_time": queryTime}); err != nil {
			return err
		}
		rechecked, err := knowledgeengine.InspectSCVDomain(options.domain, options.prefix, options.version)
		if err != nil || rechecked != installed {
			return fmt.Errorf("graph owner installation changed during validation")
		}
		selected, err = store.Select(intent, func(bound scvgraph.Intent, correlationID string) (scvgraph.Authorization, error) {
			decision, err := authorizeSCVRequest(options.topsID, correlationID, "symphony.scv.graph.select", graphResource(options, graphID))
			if err != nil {
				return scvgraph.Authorization{}, err
			}
			evidence, err := knowledgeengine.SCVCanonical(decision)
			if err != nil || decision.ExpiresAt == nil || decision.Capability == nil {
				return scvgraph.Authorization{}, fmt.Errorf("verified graph authority evidence missing")
			}
			expires := *decision.ExpiresAt
			if decision.Capability.ExpiresAt.Before(expires) {
				expires = decision.Capability.ExpiresAt
			}
			return scvgraph.Authorization{Evidence: evidence, ValidUntil: expires}, nil
		})
		if err != nil {
			return fmt.Errorf("graph selection incomplete; inspect or recover exact operation: %w", err)
		}
	}
	var evaluation json.RawMessage = json.RawMessage("null")
	if selected.GraphDigest != nil {
		evaluation, err = invokeSCV(options, "graph_query", map[string]any{"graph": selected.SelectedGraph(), "query_time": time.Now().UTC().Truncate(time.Second).Format(time.RFC3339)})
		if err != nil {
			return fmt.Errorf("selected graph is retained; current evaluation unavailable: %w", err)
		}
	}
	var attempt any
	if options.operationID != "" {
		if found, ok := selected.Operations[options.operationID]; ok {
			attempt = found
		}
	}
	result, err := knowledgeengine.SCVCanonical(map[string]any{"protocol": "symphony.qxctl.scv-projection-result.v1",
		"operation": operation, "tops_id": options.topsID, "domain": options.domain, "graph_id": graphID,
		"generation": selected.Generation, "graph_digest": selected.GraphDigest, "head_operation_id": selected.HeadOperationID,
		"graph": selected.SelectedGraph(), "current_evaluation": evaluation, "attempt": attempt,
		"canonical_apply_enabled": false, "authorization_audit": "ssiag_policy_decision_only"})
	if err != nil {
		return err
	}
	return outputSCV(options, result)
}
