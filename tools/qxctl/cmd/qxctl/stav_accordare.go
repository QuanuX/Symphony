package main

import (
	"context"
	"fmt"
	"time"

	accordareclient "github.com/QuanuX/Symphony/modules/accordare-stav-producer/client"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/spf13/cobra"
)

type stavAccordareOptions struct {
	topsID, scope string
	jsonOutput    bool
}

func newSTAVAccordareCommand() *cobra.Command {
	family := structural("accordare", fmt.Errorf("accordare subcommand is required: status or reconcile"))
	for _, operation := range []string{"status", "reconcile"} {
		options := stavAccordareOptions{scope: "user"}
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
		if operation == "reconcile" {
			interaction = "recover"
		}
		spec := commandSpec("stav.accordare."+operation, featureSTAV, interaction)
		spec.OutputProtocols = []string{"symphony.accordare.stav-producer.administration-result.v1"}
		spec.ResultValidationProtocols = []string{"symphony.accordare.stav-producer.administration-result.v1"}
		if operation == "reconcile" {
			spec.Mutability = "permission_backed_mutation"
			spec.AuthorityMode = "target_host_permission"
			spec.RecoveryCommandID = stringPointer("qxcmd:symphony:stav.accordare.reconcile")
		}
		commandregistry.Attach(leaf, spec)
		leaf.Flags().StringVar(&options.topsID, "tops-id", "", "immutable TOPS UUID")
		leaf.Flags().StringVar(&options.scope, "scope", "user", "producer scope: user or system")
		leaf.Flags().BoolVar(&options.jsonOutput, "json", false, "emit JSON")
		family.AddCommand(leaf)
	}
	return family
}

func runSTAVAccordare(operation string, options stavAccordareOptions) error {
	if options.topsID == "" || options.scope != "user" && options.scope != "system" {
		return fmt.Errorf("Accordare producer administration requires --tops-id and a user or system --scope")
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
		"pending": result.Pending, "reconciliation_needed": result.Pending > 0,
	}
	if options.jsonOutput {
		return printIndentedJSON(output)
	}
	fmt.Printf("Accordare STAV producer: operation=%s tops_id=%s pending=%d reconciliation_needed=%t\n", operation, options.topsID, result.Pending, result.Pending > 0)
	return nil
}
