package main

import (
	"errors"
	"fmt"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var errUsageOnly = errors.New("print qxctl usage")

type ssiagOptions struct {
	topsID     string
	scope      string
	jsonOutput bool
}

type stavOptions struct {
	topsID          string
	scope           string
	jsonOutput      bool
	query           stavprotocol.Query
	throughSequence optionalUint64
	verifyAfter     uint64
	verifyThrough   optionalUint64
}

type skviOptions struct {
	prefix              string
	version             string
	repository          string
	input               string
	expectedIndexDigest string
	jsonOutput          bool
}

type sclvOptions struct {
	prefix               string
	version              string
	repository           string
	input                string
	expectedLedgerDigest string
	jsonOutput           bool
}

type sacvOptions struct {
	prefix                 string
	version                string
	repository             string
	input                  string
	expectedRegistryDigest string
	jsonOutput             bool
}

type sodvOptions struct {
	prefix               string
	version              string
	repository           string
	input                string
	expectedLedgerDigest string
	jsonOutput           bool
}

func execute(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 1
	}
	if !knownTopLevel(args[0]) {
		fmt.Printf("unknown command: %s\n", args[0])
		printUsage()
		return 1
	}
	if err := validateLegacySubcommand(args); err != nil {
		fmt.Printf("%s failed: %v\n", failurePrefix(args), err)
		return 1
	}

	root, err := newRootCommand()
	if err != nil {
		fmt.Printf("qxctl failed: %v\n", err)
		return 1
	}
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		if errors.Is(err, errUsageOnly) {
			printUsage()
			return 1
		}
		fmt.Printf("%s failed: %v\n", failurePrefix(args), err)
		return 1
	}
	return 0
}

func executeCommand(args []string) error {
	if err := validateLegacySubcommand(args); err != nil {
		return err
	}
	root, err := newRootCommand()
	if err != nil {
		return err
	}
	root.SetArgs(args)
	return root.Execute()
}

func newRootCommand() (*cobra.Command, error) {
	root := &cobra.Command{
		Use:           "qxctl",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error {
			return errUsageOnly
		},
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.SetHelpCommand(&cobra.Command{Use: "__help", Hidden: true})
	root.SetHelpFunc(func(*cobra.Command, []string) { printUsage() })
	root.SetUsageFunc(func(*cobra.Command) error {
		printUsage()
		return nil
	})
	root.Version = version.Version
	root.SetVersionTemplate("qxctl version {{.Version}}\n")

	root.AddCommand(
		operationCommand("doctor", runDoctor),
		operationCommand("contracts", runContracts),
		newInventoryCommand(),
		jsonOperationCommand("status", runStatus),
		newModulesCommand(),
		newModuleCommand(),
	)

	ssiag, err := newSSIAGCommand()
	if err != nil {
		return nil, err
	}
	stav := newSTAVCommand()
	root.AddCommand(ssiag, stav, newSKVICommand(), newSCLVCommand(), newSACVCommand(), newSODVCommand())
	return root, nil
}

func newSODVCommand() *cobra.Command {
	command := &cobra.Command{
		Use:  "sodv",
		Args: usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error {
			return fmt.Errorf("SODV subcommand is required: inspect, check, verify, propose, recover, or project")
		},
	}
	for _, operation := range []string{"inspect", "check", "verify", "propose", "recover", "project"} {
		options := sodvOptions{version: "0.1.0-dev"}
		child := &cobra.Command{
			Use:  operation,
			Args: usageOnlyArgs,
			RunE: func(*cobra.Command, []string) error { return runSODV(operation, options) },
		}
		child.Flags().StringVar(&options.prefix, "prefix", "", "exact SODV installation prefix")
		child.Flags().StringVar(&options.version, "version", "0.1.0-dev", "exact installed SODV engine version")
		child.Flags().StringVar(&options.repository, "repo", "", "Symphony repository path; defaults to the current repository")
		child.Flags().BoolVar(&options.jsonOutput, "json", false, "emit operation result JSON")
		if operation == "check" {
			child.Flags().StringVar(&options.expectedLedgerDigest, "expected-ledger-digest", "", "optional expected tagged SHA-256 release-ledger digest")
		}
		if operation == "verify" || operation == "propose" || operation == "recover" {
			child.Flags().StringVar(&options.input, "input", "", "no-follow JSON operation payload file")
		}
		child.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		command.AddCommand(child)
	}
	command.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	return command
}

func newSACVCommand() *cobra.Command {
	command := &cobra.Command{
		Use:  "sacv",
		Args: usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error {
			return fmt.Errorf("SACV subcommand is required: inspect, check, diff, propose, or project")
		},
	}
	for _, operation := range []string{"inspect", "check", "diff", "propose", "project"} {
		options := sacvOptions{version: "0.1.0-dev"}
		child := &cobra.Command{
			Use:  operation,
			Args: usageOnlyArgs,
			RunE: func(*cobra.Command, []string) error { return runSACV(operation, options) },
		}
		child.Flags().StringVar(&options.prefix, "prefix", "", "exact SACV installation prefix")
		child.Flags().StringVar(&options.version, "version", "0.1.0-dev", "exact installed SACV engine version")
		child.Flags().StringVar(&options.repository, "repo", "", "Symphony repository path; defaults to the current repository")
		child.Flags().BoolVar(&options.jsonOutput, "json", false, "emit operation result JSON")
		if operation == "check" {
			child.Flags().StringVar(&options.expectedRegistryDigest, "expected-registry-digest", "", "optional expected tagged SHA-256 registry digest")
		}
		if operation == "diff" || operation == "propose" {
			child.Flags().StringVar(&options.input, "input", "", "no-follow JSON operation payload file")
		}
		child.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		command.AddCommand(child)
	}
	command.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	return command
}

func newSCLVCommand() *cobra.Command {
	command := &cobra.Command{
		Use:  "sclv",
		Args: usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error {
			return fmt.Errorf("SCLV subcommand is required: inspect, check, propose, recover, or project")
		},
	}
	for _, operation := range []string{"inspect", "check", "propose", "recover", "project"} {
		options := sclvOptions{version: "0.1.0-dev"}
		child := &cobra.Command{
			Use:  operation,
			Args: usageOnlyArgs,
			RunE: func(*cobra.Command, []string) error { return runSCLV(operation, options) },
		}
		child.Flags().StringVar(&options.prefix, "prefix", "", "exact SCLV installation prefix")
		child.Flags().StringVar(&options.version, "version", "0.1.0-dev", "exact installed SCLV engine version")
		child.Flags().StringVar(&options.repository, "repo", "", "Symphony repository path; defaults to the current repository")
		child.Flags().BoolVar(&options.jsonOutput, "json", false, "emit operation result JSON")
		if operation == "check" {
			child.Flags().StringVar(&options.expectedLedgerDigest, "expected-ledger-digest", "", "optional expected tagged SHA-256 ledger digest")
		}
		if operation == "propose" || operation == "recover" {
			child.Flags().StringVar(&options.input, "input", "", "no-follow JSON operation payload file")
		}
		child.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		command.AddCommand(child)
	}
	command.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	return command
}

func newSKVICommand() *cobra.Command {
	command := &cobra.Command{
		Use:  "skvi",
		Args: usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error {
			return fmt.Errorf("SKVI subcommand is required: inspect, check, propose, or project")
		},
	}
	for _, operation := range []string{"inspect", "check", "propose", "project"} {
		options := skviOptions{version: "0.1.0-dev"}
		child := &cobra.Command{
			Use:  operation,
			Args: usageOnlyArgs,
			RunE: func(*cobra.Command, []string) error { return runSKVI(operation, options) },
		}
		child.Flags().StringVar(&options.prefix, "prefix", "", "exact SKVI installation prefix")
		child.Flags().StringVar(&options.version, "version", "0.1.0-dev", "exact installed SKVI engine version")
		child.Flags().StringVar(&options.repository, "repo", "", "Symphony repository path; defaults to the current repository")
		child.Flags().BoolVar(&options.jsonOutput, "json", false, "emit operation result JSON")
		if operation == "check" {
			child.Flags().StringVar(&options.expectedIndexDigest, "expected-index-digest", "", "optional expected tagged SHA-256 index digest")
		}
		if operation == "propose" {
			child.Flags().StringVar(&options.input, "input", "", "no-follow JSON proposal payload file")
		}
		child.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		command.AddCommand(child)
	}
	command.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	return command
}

func operationCommand(use string, run func() error) *cobra.Command {
	command := &cobra.Command{
		Use:  use,
		Args: usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error { return run() },
	}
	command.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	return command
}

func jsonOperationCommand(use string, run func(bool) error) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use:  use,
		Args: usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error { return run(jsonOutput) },
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit JSON")
	command.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	return command
}

func newInventoryCommand() *cobra.Command {
	command := jsonOperationCommand("inventory", runInventory)
	command.AddCommand(jsonOperationCommand("digest", runInventoryDigest))
	return command
}

func newModulesCommand() *cobra.Command {
	command := operationCommand("modules", runModules)
	command.AddCommand(
		operationCommand("check", runModulesCheck),
		jsonOperationCommand("metadata", runModulesMetadata),
	)
	return command
}

func newModuleCommand() *cobra.Command {
	command := &cobra.Command{
		Use:  "module",
		Args: usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error { return errUsageOnly },
	}
	command.AddCommand(
		namedModuleCommand("inspect", runModuleInspect),
		namedModuleCommand("check", runModuleCheck),
		namedModuleMetadataCommand(),
	)
	command.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	return command
}

func namedModuleCommand(use string, run func(string) error) *cobra.Command {
	command := &cobra.Command{
		Use:  use + " <module-name>",
		Args: exactOneUsageArg,
		RunE: func(_ *cobra.Command, args []string) error { return run(args[0]) },
	}
	command.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	return command
}

func namedModuleMetadataCommand() *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use:  "metadata <module-name>",
		Args: exactOneUsageArg,
		RunE: func(_ *cobra.Command, args []string) error {
			return runModuleMetadata(args[0], jsonOutput)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit JSON")
	command.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	return command
}

func newSSIAGCommand() (*cobra.Command, error) {
	command := &cobra.Command{
		Use:  "ssiag",
		Args: usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error {
			return fmt.Errorf("SSIAG subcommand is required: status, providers, or doctor")
		},
	}
	for _, subcommand := range []string{"status", "providers", "doctor"} {
		child, err := newSSIAGLeaf(subcommand)
		if err != nil {
			return nil, err
		}
		command.AddCommand(child)
	}
	return command, nil
}

func newSSIAGLeaf(subcommand string) (*cobra.Command, error) {
	mapper := viper.New()
	command := &cobra.Command{
		Use: subcommand,
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("unexpected SSIAG arguments: %v", args)
			}
			return nil
		},
		RunE: func(*cobra.Command, []string) error {
			return runSSIAG(subcommand, ssiagOptions{
				topsID:     mapper.GetString("tops-id"),
				scope:      mapper.GetString("scope"),
				jsonOutput: mapper.GetBool("json"),
			})
		},
	}
	command.Flags().String("tops-id", "", "immutable TOPS UUID")
	command.Flags().String("scope", "user", "SSIAG scope: user or system")
	command.Flags().Bool("json", false, "emit JSON")
	for _, key := range []string{"tops-id", "scope", "json"} {
		if err := mapper.BindPFlag(key, command.Flags().Lookup(key)); err != nil {
			return nil, fmt.Errorf("bind SSIAG %s flag: %w", key, err)
		}
	}
	if err := mapper.BindEnv("tops-id", "SYMPHONY_SSIAG_TOPS_ID"); err != nil {
		return nil, fmt.Errorf("bind SSIAG TOPS environment: %w", err)
	}
	return command, nil
}

func newSTAVCommand() *cobra.Command {
	command := &cobra.Command{
		Use:  "stav",
		Args: usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error {
			return fmt.Errorf("STAV subcommand is required: status, verify, query, or doctor")
		},
	}
	command.AddCommand(&cobra.Command{
		Use:                "append",
		DisableFlagParsing: true,
		RunE: func(*cobra.Command, []string) error {
			return fmt.Errorf("qxctl stav append is prohibited; qxctl never submits arbitrary events or edits ledgers")
		},
	})
	for _, subcommand := range []string{"status", "verify", "query", "doctor"} {
		command.AddCommand(newSTAVLeaf(subcommand))
	}
	return command
}

func newSTAVLeaf(subcommand string) *cobra.Command {
	options := stavOptions{scope: "user"}
	command := &cobra.Command{
		Use: subcommand,
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("unexpected STAV arguments: %v", args)
			}
			return nil
		},
		RunE: func(*cobra.Command, []string) error { return runSTAV(subcommand, options) },
	}
	command.Flags().StringVar(&options.topsID, "tops-id", "", "immutable TOPS UUID")
	command.Flags().StringVar(&options.scope, "scope", "user", "STAV scope: user or system")
	if subcommand != "doctor" {
		command.Flags().BoolVar(&options.jsonOutput, "json", false, "emit JSON")
	}
	if subcommand == "query" {
		options.query = stavprotocol.Query{
			Schema:       stavprotocol.SchemaQuery,
			EventClasses: make([]string, 0),
			Outcomes:     make([]string, 0),
			Limit:        100,
		}
		command.Flags().Uint64Var(&options.query.AfterSequence, "after-sequence", 0, "exclusive sequence cursor")
		command.Flags().Var(&options.throughSequence, "through-sequence", "optional inclusive sequence ceiling")
		command.Flags().StringVar(&options.query.FromTime, "from-time", "", "optional inclusive UTC timestamp")
		command.Flags().StringVar(&options.query.ThroughTime, "through-time", "", "optional inclusive UTC timestamp")
		command.Flags().StringArrayVar(&options.query.EventClasses, "event-class", []string{}, "registered event class; repeat up to 16 times")
		command.Flags().StringArrayVar(&options.query.Outcomes, "outcome", []string{}, "generic outcome; repeat up to 5 times")
		command.Flags().StringVar(&options.query.CorrelationID, "correlation-id", "", "optional correlation UUID")
		command.Flags().StringVar(&options.query.RequestID, "request-id", "", "optional request UUID")
		command.Flags().Uint64Var(&options.query.Limit, "limit", 100, "page size from 1 through 1000")
	}
	if subcommand == "verify" {
		command.Flags().Uint64Var(&options.verifyAfter, "after-sequence", 0, "exclusive verification cursor")
		command.Flags().Var(&options.verifyThrough, "through-sequence", "optional inclusive verification ceiling")
	}
	return command
}

func usageOnlyArgs(_ *cobra.Command, args []string) error {
	if len(args) != 0 {
		return errUsageOnly
	}
	return nil
}

func exactOneUsageArg(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errUsageOnly
	}
	return nil
}

func knownTopLevel(value string) bool {
	switch value {
	case "--help", "--version", "doctor", "contracts", "inventory", "status", "modules", "module", "ssiag", "stav", "skvi", "sclv", "sacv", "sodv":
		return true
	default:
		return false
	}
}

func failurePrefix(args []string) string {
	if len(args) == 0 {
		return "qxctl"
	}
	switch args[0] {
	case "inventory":
		if len(args) > 1 && args[1] == "digest" {
			return "inventory digest"
		}
	case "modules":
		if len(args) > 1 && (args[1] == "check" || args[1] == "metadata") {
			return "modules " + args[1]
		}
	case "module":
		if len(args) > 1 && (args[1] == "inspect" || args[1] == "check" || args[1] == "metadata") {
			return "module " + args[1]
		}
	case "skvi":
		if len(args) > 1 {
			switch args[1] {
			case "inspect", "check", "propose", "project":
				return "skvi " + args[1]
			}
		}
	case "sclv":
		if len(args) > 1 {
			switch args[1] {
			case "inspect", "check", "propose", "recover", "project":
				return "sclv " + args[1]
			}
		}
	case "sacv":
		if len(args) > 1 {
			switch args[1] {
			case "inspect", "check", "diff", "propose", "project":
				return "sacv " + args[1]
			}
		}
	case "sodv":
		if len(args) > 1 {
			switch args[1] {
			case "inspect", "check", "verify", "propose", "recover", "project":
				return "sodv " + args[1]
			}
		}
	}
	return args[0]
}

func validateLegacySubcommand(args []string) error {
	if len(args) < 2 || args[0] != "stav" {
		return nil
	}
	switch args[1] {
	case "status", "verify", "query", "doctor", "append", "--help", "-h":
		return nil
	default:
		return fmt.Errorf("unknown STAV subcommand %q", args[1])
	}
}
