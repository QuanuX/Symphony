package main

import (
	"context"
	"fmt"
	"time"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/ssiagclient"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type ssiagProviderBindingOptions struct {
	topsID              string
	scope               string
	providerName        string
	installationID      string
	expectedStateDigest string
	reason              string
	planDigest          string
	operationID         string
	jsonOutput          bool
}

func newSSIAGProviderInstallationsCommand() (*cobra.Command, error) {
	mapper := viper.New()
	command := &cobra.Command{
		Use: "installations <provider-name>",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("SSIAG provider installations requires exactly one provider name")
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			return runSSIAGProviderBinding("installations", ssiagProviderBindingOptions{
				providerName: args[0], topsID: mapper.GetString("tops-id"), scope: mapper.GetString("scope"),
				jsonOutput: mapper.GetBool("json"),
			})
		},
	}
	spec := commandSpec("ssiag.provider.installations", featureSSIAG, "discover")
	spec.BackendOperationIDs = []string{"engop:symphony:ssiag.provider.installations.list"}
	spec.OutputProtocols = []string{ssiagclient.ProviderInstallationInventoryProtocol}
	spec.ResultValidationProtocols = []string{ssiagclient.ProviderInstallationInventoryProtocol}
	commandregistry.Attach(command, spec)
	if err := addSSIAGProviderBindingMappedFlags(command, mapper); err != nil {
		return nil, fmt.Errorf("bind SSIAG provider installations flags: %w", err)
	}
	return command, nil
}

func newSSIAGProviderBindingCommand() (*cobra.Command, error) {
	command := structural("binding", fmt.Errorf("SSIAG provider binding subcommand is required: status, plan, apply, apply-status, or recover"))
	for _, operation := range []string{"status", "plan", "apply", "apply-status", "recover"} {
		child, err := newSSIAGProviderBindingLeaf(operation)
		if err != nil {
			return nil, err
		}
		command.AddCommand(child)
	}
	return command, nil
}

func newSSIAGProviderBindingLeaf(operation string) (*cobra.Command, error) {
	mapper := viper.New()
	command := &cobra.Command{
		Use: operation + " <provider-name>",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("SSIAG provider binding %s requires exactly one provider name", operation)
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			return runSSIAGProviderBinding(operation, ssiagProviderBindingOptions{
				providerName: args[0], topsID: mapper.GetString("tops-id"), scope: mapper.GetString("scope"),
				installationID:      mapper.GetString("installation-id"),
				expectedStateDigest: mapper.GetString("expected-state-digest"), reason: mapper.GetString("reason"),
				planDigest: mapper.GetString("plan-digest"), operationID: mapper.GetString("operation-id"),
				jsonOutput: mapper.GetBool("json"),
			})
		},
	}
	interaction := map[string]string{"status": "query", "plan": "propose", "apply": "apply", "apply-status": "query", "recover": "recover"}[operation]
	spec := commandSpec("ssiag.provider.binding."+operation, featureSSIAG, interaction)
	backendOperation := operation
	if operation == "status" {
		backendOperation = "observe"
	}
	spec.BackendOperationIDs = []string{"engop:symphony:ssiag.provider.binding." + backendOperation}
	spec.OutputProtocols = []string{map[string]string{
		"status":       ssiagclient.ProviderBindingStatusProtocol,
		"plan":         ssiagclient.ProviderBindingPlanProtocol,
		"apply":        ssiagclient.ProviderBindingResultProtocol,
		"apply-status": ssiagclient.ProviderBindingResultProtocol,
		"recover":      ssiagclient.ProviderBindingResultProtocol,
	}[operation]}
	spec.ResultValidationProtocols = append([]string(nil), spec.OutputProtocols...)
	switch operation {
	case "plan":
		spec.Mutability = "proposal_only"
		spec.AuthorityMode = "ssiag"
		spec.InputProtocols = []string{ssiagclient.ProviderBindingPlanRequestProtocol}
	case "apply":
		spec.Mutability = "permission_backed_mutation"
		spec.AuthorityMode = "ssiag"
		spec.InputProtocols = []string{ssiagclient.ProviderBindingApplyRequestProtocol}
		spec.RecoveryCommandID = stringPointer("qxcmd:symphony:ssiag.provider.binding.recover")
	case "apply-status":
		spec.AuthorityMode = "ssiag"
	case "recover":
		spec.Mutability = "permission_backed_mutation"
		spec.AuthorityMode = "ssiag"
		spec.InputProtocols = []string{ssiagclient.ProviderBindingRecoveryRequestProtocol}
		spec.RecoveryCommandID = stringPointer("qxcmd:symphony:ssiag.provider.binding.recover")
	}
	commandregistry.Attach(command, spec)
	if err := addSSIAGProviderBindingMappedFlags(command, mapper); err != nil {
		return nil, fmt.Errorf("bind SSIAG provider binding %s flags: %w", operation, err)
	}
	switch operation {
	case "plan":
		command.Flags().String("installation-id", "", "opaque exact installation ID, or not_applicable to unbind while preserving material")
		command.Flags().String("expected-state-digest", "", "exact binding state digest, or absent for initial state")
		command.Flags().String("reason", "", "bounded administrative reason; never include secret material")
	case "apply":
		command.Flags().String("plan-digest", "", "exact SSIAG-issued provider binding plan digest")
		command.Flags().String("expected-state-digest", "", "exact binding state digest, or absent for initial state")
	case "apply-status":
		command.Flags().String("operation-id", "", "exact SSIAG-issued apply operation UUID")
	case "recover":
		command.Flags().String("expected-state-digest", "", "exact binding state digest, or absent for initial state")
		command.Flags().String("reason", "", "bounded administrative recovery reason; never include secret material")
	}
	for _, key := range []string{"installation-id", "expected-state-digest", "reason", "plan-digest", "operation-id"} {
		if flag := command.Flags().Lookup(key); flag != nil {
			if err := mapper.BindPFlag(key, flag); err != nil {
				return nil, fmt.Errorf("bind %s: %w", key, err)
			}
		}
	}
	return command, nil
}

func addSSIAGProviderBindingMappedFlags(command *cobra.Command, mapper *viper.Viper) error {
	command.Flags().String("tops-id", "", "immutable TOPS UUID")
	command.Flags().String("scope", "user", "SSIAG scope: user or system")
	command.Flags().Bool("json", false, "emit JSON")
	for _, key := range []string{"tops-id", "scope", "json"} {
		if err := mapper.BindPFlag(key, command.Flags().Lookup(key)); err != nil {
			return err
		}
	}
	return mapper.BindEnv("tops-id", "SYMPHONY_SSIAG_TOPS_ID")
}

func runSSIAGProviderBinding(operation string, options ssiagProviderBindingOptions) error {
	if options.topsID == "" {
		return fmt.Errorf("--tops-id or SYMPHONY_SSIAG_TOPS_ID is required")
	}
	if err := ssiagclient.ValidateProviderName(options.providerName); err != nil {
		return err
	}
	if options.scope != "user" && options.scope != "system" {
		return fmt.Errorf("unsupported SSIAG scope %q", options.scope)
	}
	if err := validateSSIAGProviderBindingOptions(operation, options); err != nil {
		return err
	}
	commandContext, commandCancel := context.WithTimeout(context.Background(), ssiagProviderBindingEndToEndBudget)
	defer commandCancel()
	client, err := ssiagclient.NewForTOPS(options.scope, options.topsID, ssiagProviderBindingOperationBudget)
	if err != nil {
		return err
	}
	statusContext, statusCancel := context.WithTimeout(commandContext, ssiagProviderStatusBudget)
	_, err = requireSSIAGStatus(statusContext, client, options.topsID, options.scope)
	statusCancel()
	if err != nil {
		return err
	}
	operationContext, operationCancel := context.WithTimeout(commandContext, ssiagProviderBindingOperationBudget)
	defer operationCancel()
	switch operation {
	case "installations":
		result, err := client.ProviderInstallations(operationContext, options.providerName)
		if err != nil {
			return err
		}
		if err := requireProviderBindingIdentity(result.TOPSID, result.Scope, options); err != nil {
			return err
		}
		if options.jsonOutput {
			return printSSIAGJSON(result)
		}
		fmt.Printf("SSIAG provider installations: name=%s kind=%s count=%d state_digest=%s operational=false\n",
			result.ProviderName, result.ProviderKind, len(result.Installations), result.InventoryDigest)
		return nil
	case "status":
		result, err := client.ProviderBindingStatus(operationContext, options.providerName)
		if err != nil {
			return err
		}
		if err := requireProviderBindingIdentity(result.TOPSID, result.Scope, options); err != nil {
			return err
		}
		if options.jsonOutput {
			return printSSIAGJSON(result)
		}
		fmt.Printf("SSIAG provider binding: name=%s state=%s generation=%d installation=%s previous=%s recovery_required=%t state_digest=%s operational=false\n",
			result.ProviderName, result.BindingState, result.Generation, result.InstallationID,
			result.PreviousInstallationID, result.RecoveryRequired, result.StateDigest)
		return nil
	case "plan":
		result, err := client.PlanProviderBinding(operationContext, options.providerName, ssiagclient.ProviderBindingPlanRequest{
			InstallationID: options.installationID, ExpectedStateDigest: options.expectedStateDigest, Reason: options.reason,
		})
		if err != nil {
			return err
		}
		if err := requireProviderBindingIdentity(result.TOPSID, result.Scope, options); err != nil {
			return err
		}
		if options.jsonOutput {
			return printSSIAGJSON(result)
		}
		fmt.Printf("SSIAG provider binding plan: id=%s desired=%s applicable=%t changed=%t recovery_required=%t actions=%d digest=%s operational=false\n",
			result.PlanID, result.DesiredState, result.Applicable, result.Changed, result.RecoveryRequired,
			len(result.Actions), result.PlanDigest)
		return nil
	case "apply":
		result, err := client.ApplyProviderBinding(operationContext, options.providerName, ssiagclient.ProviderBindingApplyRequest{
			PlanDigest: options.planDigest, ExpectedStateDigest: options.expectedStateDigest,
		})
		return printSSIAGProviderBindingResult(result, err, options)
	case "apply-status":
		result, err := client.ProviderBindingApplyStatus(operationContext, options.providerName, options.operationID)
		return printSSIAGProviderBindingResult(result, err, options)
	case "recover":
		result, err := client.RecoverProviderBinding(operationContext, options.providerName, ssiagclient.ProviderBindingRecoveryRequest{
			ExpectedStateDigest: options.expectedStateDigest, Reason: options.reason,
		})
		return printSSIAGProviderBindingResult(result, err, options)
	default:
		return fmt.Errorf("unknown SSIAG provider binding operation %q", operation)
	}
}

func validateSSIAGProviderBindingOptions(operation string, options ssiagProviderBindingOptions) error {
	switch operation {
	case "installations", "status":
		return nil
	case "plan":
		if options.installationID == "" || options.expectedStateDigest == "" || options.reason == "" {
			return fmt.Errorf("--installation-id, --expected-state-digest, and --reason are required")
		}
	case "apply":
		if options.planDigest == "" || options.expectedStateDigest == "" {
			return fmt.Errorf("--plan-digest and --expected-state-digest are required")
		}
	case "apply-status":
		if options.operationID == "" {
			return fmt.Errorf("--operation-id is required")
		}
	case "recover":
		if options.expectedStateDigest == "" || options.reason == "" {
			return fmt.Errorf("--expected-state-digest and --reason are required")
		}
	default:
		return fmt.Errorf("unknown SSIAG provider binding operation %q", operation)
	}
	return nil
}

func requireProviderBindingIdentity(topsID, scope string, options ssiagProviderBindingOptions) error {
	if topsID != options.topsID || scope != options.scope {
		return fmt.Errorf("SSIAG provider binding evidence does not match requested TOPS and scope")
	}
	return nil
}

func printSSIAGProviderBindingResult(result ssiagclient.ProviderBindingResult, err error, options ssiagProviderBindingOptions) error {
	if err != nil {
		return err
	}
	if err := requireProviderBindingIdentity(result.TOPSID, result.Scope, options); err != nil {
		return err
	}
	if options.jsonOutput {
		return printSSIAGJSON(result)
	}
	fmt.Printf("SSIAG provider binding %s: operation_id=%s state=%s generation=%d installation=%s previous=%s attempt=%s changed=%t recovered=%t recovery_required=%t state_digest=%s operational=false\n",
		result.Operation, result.OperationID, result.BindingState, result.Generation, result.InstallationID,
		result.PreviousInstallationID, result.AttemptState, result.Changed, result.Recovered,
		result.RecoveryRequired, result.StateDigest)
	return nil
}

const (
	ssiagProviderBindingOperationBudget = 12 * time.Second
	ssiagProviderBindingEndToEndBudget  = 15 * time.Second
)
