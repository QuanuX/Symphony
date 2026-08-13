package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/invariantregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/validation"
	"github.com/spf13/cobra"
)

type knowledgeInvariantOptions struct {
	repository  string
	invariantID string
	prefix      string
	version     string
	jsonOutput  bool
}

// runInvariantValidator is replaceable only by same-package tests. Production
// always uses the exact receipt-validated validator client and its complete,
// digest-bound symphony.validation.result.v1 evidence.
var runInvariantValidator = validation.Run

type exactEvidenceExitError struct {
	code int
}

func (failure *exactEvidenceExitError) Error() string {
	return fmt.Sprintf("validated evidence reports exit status %d", failure.code)
}

func newKnowledgeInvariantCommand() *cobra.Command {
	command := structural("invariant", fmt.Errorf("knowledge invariant subcommand is required: status, list, show, or check"))
	for _, operation := range []string{"status", "list", "show", "check"} {
		options := knowledgeInvariantOptions{version: "0.1.0-dev"}
		child := &cobra.Command{
			Use:  operation,
			Args: usageOnlyArgs,
			RunE: func(*cobra.Command, []string) error {
				return runKnowledgeInvariant(operation, options)
			},
		}
		if operation == "check" {
			registeredInvariantCheck(child)
		} else {
			registeredInvariantQuery(child, "knowledge.invariant."+operation, map[string]string{
				"status": "query", "list": "discover", "show": "inspect",
			}[operation])
		}
		child.Flags().StringVar(&options.repository, "repo", "", "Symphony repository path; defaults to the current repository")
		child.Flags().BoolVar(&options.jsonOutput, "json", false, "emit structured JSON")
		if operation == "show" {
			child.Flags().StringVar(&options.invariantID, "invariant-id", "", "exact stable invariant:symphony: identity")
		}
		if operation == "check" {
			child.Flags().StringVar(&options.prefix, "prefix", "", "exact Symphony Validator installation prefix")
			child.Flags().StringVar(&options.version, "version", "0.1.0-dev", "exact installed validator version")
		}
		child.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		command.AddCommand(child)
	}
	command.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	return command
}

func runKnowledgeInvariant(operation string, options knowledgeInvariantOptions) error {
	if operation == "check" {
		if options.prefix == "" {
			return fmt.Errorf("--prefix is required")
		}
		result, err := runInvariantValidator(
			context.Background(), options.prefix, options.version, options.repository,
		)
		if err != nil {
			return err
		}
		if options.jsonOutput {
			if err := printIndentedJSON(result); err != nil {
				return err
			}
		} else {
			printInvariantCheck(result)
		}
		if result.Evidence.ExitCode != 0 {
			return &exactEvidenceExitError{code: result.Evidence.ExitCode}
		}
		return nil
	}
	if operation == "show" && options.invariantID == "" {
		return fmt.Errorf("--invariant-id is required")
	}

	registry, err := invariantregistry.Load(options.repository)
	if err != nil {
		return err
	}
	switch operation {
	case "status":
		projection, err := invariantregistry.Status(registry)
		if err != nil {
			return err
		}
		if options.jsonOutput {
			return printIndentedJSON(projection)
		}
		fmt.Printf(
			"Knowledge invariant registry: digest=%s invariants=%d adapters=%d catalog_scope=%s catalog_complete=%t forward_gate=%s consumer_check=%s semantic_validity=%s\n",
			projection.RegistryDigest, projection.InvariantCount, projection.AdapterCount,
			projection.CatalogScope, projection.CatalogComplete, projection.ForwardGate,
			projection.ConsumerCheck, projection.SemanticValidity,
		)
		return nil
	case "list":
		projection, err := invariantregistry.List(registry)
		if err != nil {
			return err
		}
		if options.jsonOutput {
			return printIndentedJSON(projection)
		}
		fmt.Printf(
			"Knowledge invariants: digest=%s count=%d consumer_check=%s semantic_validity=%s\n",
			projection.RegistryDigest, len(projection.Invariants), projection.ConsumerCheck, projection.SemanticValidity,
		)
		for _, invariant := range projection.Invariants {
			fmt.Printf(
				"Knowledge invariant: id=%s owner=%s contract=%s ipc=%t status=%s title=%q\n",
				invariant.InvariantID, invariant.OwnerComponent, invariant.OwnerContract,
				invariant.IPCBoundary, invariant.Status, invariant.Title,
			)
		}
		return nil
	case "show":
		projection, err := invariantregistry.Show(registry, options.invariantID)
		if err != nil {
			return err
		}
		if options.jsonOutput {
			return printIndentedJSON(projection)
		}
		invariant := projection.Invariant
		fmt.Printf(
			"Knowledge invariant: id=%s title=%q owner=%s contract=%s ipc=%t status=%s digest=%s consumer_check=%s semantic_validity=%s\n",
			invariant.InvariantID, invariant.Title, invariant.OwnerComponent, invariant.OwnerContract,
			invariant.IPCBoundary, invariant.Status, projection.RegistryDigest,
			projection.ConsumerCheck, projection.SemanticValidity,
		)
		fmt.Printf("Knowledge invariant statement: %s\n", invariant.Statement)
		return nil
	default:
		return fmt.Errorf("unsupported knowledge invariant operation")
	}
}

func printInvariantCheck(result validation.Result) {
	invariantFindings := 0
	for _, finding := range result.Evidence.Findings {
		if strings.HasPrefix(finding.RuleID, "invariant_ownership.") {
			invariantFindings++
		}
	}
	fmt.Printf(
		"Knowledge invariant check: outcome=%s exit=%d validator=%s version=%s invariant_findings=%d evidence=%s result=%s\n",
		result.Evidence.Outcome, result.Evidence.ExitCode, result.Evidence.ValidatorID,
		result.Evidence.ValidatorVersion, invariantFindings, result.Evidence.EvidenceDigest, result.ResultDigest,
	)
	for _, finding := range result.Evidence.Findings {
		if finding.Category == "pass" {
			continue
		}
		fmt.Printf(
			"Knowledge invariant validation evidence: category=%s scope=%s rule=%s subject=%s occurrence=%s detail=%q\n",
			finding.Category, finding.Scope, finding.RuleID, finding.SubjectID, finding.OccurrenceID, finding.Detail,
		)
	}
}
