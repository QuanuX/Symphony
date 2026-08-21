package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	accordareclient "github.com/QuanuX/Symphony/modules/accordare-stav-producer/client"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/spf13/cobra"
)

type stavAccordareOptions struct {
	topsID, scope, prefix, version string
	force, noStart                 bool
	jsonOutput                     bool
}

func newSTAVAccordareCommand() *cobra.Command {
	family := structural("accordare", fmt.Errorf("accordare subcommand is required: status, reconcile, supervisor-install, or supervisor-uninstall"))
	for _, operation := range []string{"status", "reconcile", "supervisor-install", "supervisor-uninstall"} {
		options := stavAccordareOptions{scope: "user", version: "0.1.0-dev"}
		leaf := &cobra.Command{
			Use: operation,
			Args: func(_ *cobra.Command, args []string) error {
				if len(args) != 0 {
					return fmt.Errorf("unexpected Accordare producer arguments: %v", args)
				}
				return nil
			},
			RunE: func(_ *cobra.Command, _ []string) error { return runSTAVAccordare(operation, options) },
		}
		interaction := "query"
		if operation == "reconcile" || strings.HasPrefix(operation, "supervisor-") {
			interaction = "recover"
			if strings.HasPrefix(operation, "supervisor-") {
				interaction = "apply"
			}
		}
		spec := commandSpec("stav.accordare."+operation, featureSTAV, interaction)
		spec.OutputProtocols = []string{"symphony.accordare.stav-producer.administration-result.v1"}
		spec.ResultValidationProtocols = []string{"symphony.accordare.stav-producer.administration-result.v1"}
		if strings.HasPrefix(operation, "supervisor-") {
			spec.OutputProtocols = []string{"symphony.accordare.stav-producer.supervisor-result.v1"}
			spec.ResultValidationProtocols = []string{"symphony.accordare.stav-producer.supervisor-result.v1"}
		}
		if operation == "reconcile" || strings.HasPrefix(operation, "supervisor-") {
			spec.Mutability = "permission_backed_mutation"
			spec.AuthorityMode = "target_host_permission"
			if operation == "reconcile" {
				spec.RecoveryCommandID = stringPointer("qxcmd:symphony:stav.accordare.reconcile")
			} else if operation == "supervisor-install" {
				spec.RecoveryCommandID = stringPointer("qxcmd:symphony:stav.accordare.supervisor-uninstall")
			} else {
				spec.RecoveryCommandID = stringPointer("qxcmd:symphony:stav.accordare.supervisor-install")
			}
		}
		commandregistry.Attach(leaf, spec)
		leaf.Flags().StringVar(&options.topsID, "tops-id", "", "immutable TOPS UUID")
		leaf.Flags().StringVar(&options.scope, "scope", "user", "producer scope: user or system")
		leaf.Flags().BoolVar(&options.jsonOutput, "json", false, "emit JSON")
		if strings.HasPrefix(operation, "supervisor-") {
			leaf.Flags().StringVar(&options.prefix, "prefix", "", "exact Accordare producer installation prefix")
			leaf.Flags().StringVar(&options.version, "version", "0.1.0-dev", "exact installed producer version")
			leaf.Flags().BoolVar(&options.force, "force", false, "replace or remove a drifted descriptor")
			leaf.Flags().BoolVar(&options.noStart, "no-start", false, "change the descriptor without native manager mutation")
		}
		family.AddCommand(leaf)
	}
	return family
}

func runSTAVAccordare(operation string, options stavAccordareOptions) error {
	if options.topsID == "" || options.scope != "user" && options.scope != "system" {
		return fmt.Errorf("Accordare producer administration requires --tops-id and a user or system --scope")
	}
	if strings.HasPrefix(operation, "supervisor-") {
		return runSTAVAccordareSupervisor(operation, options)
	}
	path, err := accordareclient.ConfigPath(options.scope, options.topsID)
	if err != nil {
		return err
	}
	producerClient, err := accordareclient.NewFromConfig(path)
	if err != nil {
		return fmt.Errorf("load exact Accordare producer enrollment: %w", err)
	}
	requestID, err := randomUUID()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	var result accordareclient.AuditResult
	if operation == "status" {
		result, err = producerClient.Status(ctx, requestID, options.topsID)
	} else {
		result, err = producerClient.Reconcile(ctx, requestID, options.topsID)
	}
	if err != nil {
		return err
	}
	if result.Disposition != "succeeded" {
		return fmt.Errorf("Accordare producer rejected %s: %s", operation, result.ReasonCode)
	}
	output := map[string]any{
		"protocol":  "symphony.accordare.stav-producer.administration-result.v1",
		"operation": operation, "tops_id": options.topsID, "scope": options.scope,
		"disposition": result.Disposition, "reason_code": result.ReasonCode,
		"pending": result.Pending, "intent_pending": result.IntentPending, "append_pending": result.AppendPending,
		"reconciliation_needed": result.Pending > 0,
	}
	if options.jsonOutput {
		return printIndentedJSON(output)
	}
	fmt.Printf("Accordare STAV producer: operation=%s tops_id=%s pending=%d intent_pending=%d append_pending=%d reconciliation_needed=%t\n", operation, options.topsID, result.Pending, result.IntentPending, result.AppendPending, result.Pending > 0)
	return nil
}

func runSTAVAccordareSupervisor(operation string, options stavAccordareOptions) error {
	if options.prefix == "" {
		return fmt.Errorf("Accordare supervisor administration requires --prefix")
	}
	installation, err := accordareclient.VerifyInstallation(options.prefix, options.version)
	if err != nil {
		return fmt.Errorf("verify exact Accordare producer installation: %w", err)
	}
	configPath, err := accordareclient.ConfigPath(options.scope, options.topsID)
	if err != nil {
		return err
	}
	verb := strings.TrimPrefix(operation, "supervisor-")
	args := []string{"supervisor", verb, "--tops-id", options.topsID, "--scope", options.scope, "--config", configPath, "--binary", installation.Binary}
	if options.force {
		args = append(args, "--force")
	}
	if options.noStart {
		args = append(args, "--no-start")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, installation.Binary, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("Accordare supervisor %s failed: %s", verb, strings.TrimSpace(string(output)))
	}
	result := map[string]any{"protocol": "symphony.accordare.stav-producer.supervisor-result.v1", "operation": verb, "tops_id": options.topsID, "scope": options.scope, "version": installation.Version, "receipt_digest": installation.ReceiptDigest, "output": strings.TrimSpace(string(output))}
	if options.jsonOutput {
		return printIndentedJSON(result)
	}
	fmt.Println(result["output"])
	return nil
}
