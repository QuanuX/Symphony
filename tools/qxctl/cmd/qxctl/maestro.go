package main

import (
	"context"
	"fmt"
	"time"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgelifecycle"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/maestroclient"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/ssiagclient"
	"github.com/spf13/cobra"
)

type maestroOptions struct {
	prefix                 string
	version                string
	repository             string
	stateRoot              string
	topsID                 string
	receptorID             string
	componentID            string
	operationID            string
	expectedRegistryDigest string
	discover               bool
	scope                  string
	ttl                    time.Duration
	jsonOutput             bool
}

func newMaestroCommand() *cobra.Command {
	command := &cobra.Command{
		Use: "maestro", Args: usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error {
			return fmt.Errorf("maestro subcommand is required: inspect, inventory, status, or recover")
		},
	}
	for _, operation := range []string{"inspect", "inventory", "status", "recover"} {
		options := maestroOptions{version: "0.1.0-dev", scope: "user", ttl: 15 * time.Minute}
		child := &cobra.Command{
			Use: operation, Args: usageOnlyArgs,
			RunE: func(*cobra.Command, []string) error { return runMaestro(operation, options) },
		}
		child.Flags().StringVar(&options.prefix, "prefix", "", "exact Maestro installation prefix")
		child.Flags().StringVar(&options.version, "version", "0.1.0-dev", "exact Maestro version")
		child.Flags().StringVar(&options.repository, "repo", "", "Symphony repository path")
		child.Flags().StringVar(&options.topsID, "tops-id", "", "immutable TOPS UUID")
		child.Flags().StringVar(&options.receptorID, "receptor-id", "", "exact Maestro receptor identity")
		child.Flags().StringVar(&options.scope, "scope", "user", "SSIAG installation scope: user or system")
		child.Flags().BoolVar(&options.jsonOutput, "json", false, "emit result JSON")
		if operation != "inspect" {
			child.Flags().StringVar(&options.stateRoot, "state-root", "", "state root; defaults to XDG_STATE_HOME or ~/.local/state")
			child.Flags().DurationVar(&options.ttl, "ttl", 15*time.Minute, "requested Maestro capability lifetime")
		}
		if operation == "status" {
			child.Flags().StringVar(&options.componentID, "component-id", "", "optional exact component filter")
		}
		if operation == "recover" {
			child.Flags().StringVar(&options.operationID, "operation-id", "", "stable recovery idempotency token")
			child.Flags().StringVar(&options.expectedRegistryDigest, "expected-registry-digest", "", "exact recoverable registry digest")
			child.Flags().BoolVar(&options.discover, "discover", false, "recover a uniquely linked forward registry")
		}
		child.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		command.AddCommand(child)
	}
	command.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	return command
}

func runMaestro(operation string, options maestroOptions) error {
	if options.prefix == "" || options.topsID == "" {
		return fmt.Errorf("--prefix and --tops-id are required")
	}
	if operation != "inventory" && options.receptorID == "" {
		return fmt.Errorf("--receptor-id is required")
	}
	installation, err := knowledgeengine.InspectMaestroInstallation(options.prefix, options.version)
	if err != nil {
		return err
	}
	repositoryRoot, err := resolveKnowledgeRepository(options.repository)
	if err != nil {
		return err
	}
	ctx := context.Background()
	var result maestroclient.Result
	switch operation {
	case "inspect":
		result, err = maestroclient.Inspect(ctx, installation.Prefix, installation.Version,
			repositoryRoot, options.topsID, options.receptorID)
	case "status":
		stateRoot, stateErr := maestroStateRoot(options.stateRoot, options.topsID)
		if stateErr != nil {
			return stateErr
		}
		filter := options.componentID
		if filter == "" {
			filter = "all"
		}
		resource := maestroclient.Resource(options.topsID, options.receptorID, "status", filter, "none", "status")
		decision, authErr := authorizeMaestro(options, "status", resource)
		if authErr != nil {
			return authErr
		}
		result, err = maestroclient.Status(ctx, installation.Prefix, installation.Version,
			repositoryRoot, stateRoot, options.topsID, options.receptorID, options.componentID, decision)
	case "inventory":
		stateRoot, stateErr := maestroStateRoot(options.stateRoot, options.topsID)
		if stateErr != nil {
			return stateErr
		}
		decision, authErr := authorizeMaestroInventory(options)
		if authErr != nil {
			return authErr
		}
		inventory, inventoryErr := maestroclient.Inventory(
			ctx, installation.Prefix, installation.Version, repositoryRoot,
			stateRoot, options.topsID, decision)
		if inventoryErr != nil {
			return inventoryErr
		}
		if options.jsonOutput {
			return printIndentedJSON(inventory)
		}
		fmt.Printf("Maestro inventory: tops_id=%s receptors=%d docked_components=%d digest=%s derived=true canonical=false\n",
			inventory.TOPSID, inventory.Inventory.ReceptorCount,
			inventory.Inventory.DockedComponentCount, inventory.Inventory.InventoryDigest)
		return nil
	case "recover":
		if options.operationID == "" || options.discover == (options.expectedRegistryDigest != "") {
			return fmt.Errorf("--operation-id and exactly one of --expected-registry-digest or --discover are required")
		}
		expected := options.expectedRegistryDigest
		if options.discover {
			expected = "discover"
		}
		stateRoot, stateErr := maestroStateRoot(options.stateRoot, options.topsID)
		if stateErr != nil {
			return stateErr
		}
		resource := maestroclient.Resource(options.topsID, options.receptorID, "recover", "all", "none", expected)
		decision, authErr := authorizeMaestro(options, "recover", resource)
		if authErr != nil {
			return authErr
		}
		result, err = maestroclient.Recover(ctx, installation.Prefix, installation.Version,
			repositoryRoot, stateRoot, options.topsID, options.receptorID, options.operationID, expected, decision)
	default:
		return fmt.Errorf("unsupported Maestro operation")
	}
	if err != nil {
		return err
	}
	if options.jsonOutput {
		return printIndentedJSON(result)
	}
	fmt.Printf("Maestro %s: tops_id=%s receptor=%s outcome=%s changed=%t registry=%s execution_enabled=false canonical=false\n",
		operation, result.TOPSID, result.ReceptorID, result.Outcome, result.Changed, nullableString(result.RegistryDigest))
	return nil
}

func authorizeMaestroInventory(options maestroOptions) (ssiagclient.AuthorizationDecision, error) {
	client, err := ssiagclient.NewForTOPS(options.scope, options.topsID, 4*time.Second)
	if err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if _, err := requireSSIAGStatus(ctx, client, options.topsID, options.scope); err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	requestID, err := randomUUID()
	if err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	correlationID, err := randomUUID()
	if err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	now := time.Now().UTC().Truncate(time.Second)
	request := ssiagclient.AuthorizationRequest{
		Schema: "symphony.ssiag.authorization-request.v1", RequestID: requestID,
		CorrelationID: correlationID, Operation: "symphony.maestro.receptor-inventory.read",
		Resource: maestroclient.InventoryResource(options.topsID), Audience: "qxctl",
		Scope: "tops:" + options.topsID, RequestedAt: now,
		RequestedExpiresAt: now.Add(options.ttl).UTC().Truncate(time.Second),
	}
	decision, err := client.Authorize(ctx, request)
	if err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	if err := validateSessionAuthorization(decision, request, options.topsID); err != nil {
		return ssiagclient.AuthorizationDecision{}, fmt.Errorf("SSIAG Maestro inventory authorization rejected: %w", err)
	}
	return decision, nil
}

func maestroStateRoot(value, topsID string) (string, error) {
	store, err := knowledgelifecycle.NewStore(value, topsID)
	if err != nil {
		return "", err
	}
	return store.StateRoot(), nil
}

func authorizeMaestro(options maestroOptions, operation, resource string) (ssiagclient.AuthorizationDecision, error) {
	client, err := ssiagclient.NewForTOPS(options.scope, options.topsID, 4*time.Second)
	if err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if _, err := requireSSIAGStatus(ctx, client, options.topsID, options.scope); err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	requestID, err := randomUUID()
	if err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	correlationID, err := randomUUID()
	if err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	now := time.Now().UTC().Truncate(time.Second)
	request := ssiagclient.AuthorizationRequest{
		Schema: "symphony.ssiag.authorization-request.v1", RequestID: requestID,
		CorrelationID: correlationID, Operation: "symphony.maestro.docking." + operation,
		Resource: resource, Audience: "qxctl", Scope: "tops:" + options.topsID,
		RequestedAt: now, RequestedExpiresAt: now.Add(options.ttl).UTC().Truncate(time.Second),
	}
	decision, err := client.Authorize(ctx, request)
	if err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	if err := validateSessionAuthorization(decision, request, options.topsID); err != nil {
		return ssiagclient.AuthorizationDecision{}, fmt.Errorf("SSIAG Maestro authorization rejected: %w", err)
	}
	return decision, nil
}

func nullableString(value *string) string {
	if value == nil {
		return "absent"
	}
	return *value
}
