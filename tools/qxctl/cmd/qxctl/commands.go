package main

import (
	"errors"
	"fmt"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/ssiagclient"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/validation"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var errUsageOnly = errors.New("print qxctl usage")

type ssiagOptions struct {
	topsID          string
	scope           string
	providerName    string
	profileID       string
	subjectID       string
	authorityBasis  string
	input           string
	expectedPolicy  string
	operationID     string
	expectedAttempt string
	reset           bool
	discover        bool
	ttl             time.Duration
	jsonOutput      bool
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

type ssfvOptions struct {
	prefix                  string
	version                 string
	repository              string
	input                   string
	baseline                string
	freshness               string
	expectedNamespaceDigest string
	expectedRegistryDigest  string
	jsonOutput              bool
}

type knowledgeEngineOptions struct {
	role                   string
	stateRoot              string
	prefix                 string
	version                string
	expectedRegistryDigest string
	jsonOutput             bool
}

type knowledgeReconcileOptions struct {
	stateRoot             string
	repository            string
	operationID           string
	expectedJournalDigest string
	paths                 []string
	discover              bool
	jsonOutput            bool
}

type knowledgeSessionOptions struct {
	topsID                string
	scope                 string
	stateRoot             string
	repository            string
	event                 string
	eventID               string
	operationID           string
	expectedJournalDigest string
	contextRefs           []string
	discover              bool
	recoverTransition     bool
	ttl                   time.Duration
	jsonOutput            bool
}

type knowledgeSSFVMaintenanceOptions struct {
	topsID                string
	scope                 string
	stateRoot             string
	repository            string
	operationID           string
	expectedJournalDigest string
	discover              bool
	ttl                   time.Duration
	maestroPrefix         string
	maestroVersion        string
	jsonOutput            bool
}

type knowledgeLifecycleOptions struct {
	topsID                  string
	scope                   string
	stateRoot               string
	repository              string
	profileID               string
	input                   string
	expectedProfileDigest   string
	configuredRoots         []string
	priorAppliedStateDigest string
	operationID             string
	expectedJournalDigest   string
	expectedApplyDigest     string
	expectedAppliedDigest   string
	sourceJournalDigest     string
	stagedRoots             []string
	ownershipRoot           string
	expectedOwnershipDigest string
	receiptDigest           string
	expectedHostDigest      string
	integrationRoot         string
	recoveryMode            string
	hostDisabled            bool
	bootResultSink          *validatedLifecycleBootResult
	maestroPrefix           string
	maestroVersion          string
	maestroReceptorIDs      []string
	maxActions              uint64
	discover                bool
	ttl                     time.Duration
	jsonOutput              bool
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
		var validationFailure *validationOutcomeError
		if errors.As(err, &validationFailure) {
			return 1
		}
		var exactEvidenceExit *exactEvidenceExitError
		if errors.As(err, &exactEvidenceExit) {
			if exactEvidenceExit.code > 0 && exactEvidenceExit.code <= 125 {
				return exactEvidenceExit.code
			}
			return 1
		}
		var validatorExit *validation.ValidatorExitError
		if errors.As(err, &validatorExit) {
			fmt.Printf("%s failed: %v\n", failurePrefix(args), err)
			if validatorExit.ExitCode > 0 && validatorExit.ExitCode <= 125 {
				return validatorExit.ExitCode
			}
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
	root := structural("qxctl", errUsageOnly)
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.CompletionOptions.DisableDefaultCmd = true
	root.SetHelpCommand(commandregistry.Internal(&cobra.Command{Use: "__help"}))
	root.SetHelpFunc(func(*cobra.Command, []string) { printUsage() })
	root.SetUsageFunc(func(*cobra.Command) error {
		printUsage()
		return nil
	})
	root.Version = version.Version
	root.SetVersionTemplate("qxctl version {{.Version}}\n")

	root.AddCommand(
		operationCommand("doctor", featureQXCTL, "validate", runDoctor),
		operationCommand("contracts", featureQXCTL, "validate", runContracts),
		newInventoryCommand(),
		jsonOperationCommand("status", featureQXCTL, "query", runStatus),
		newModulesCommand(),
		newModuleCommand(),
		newCommandsCommand(root),
	)

	ssiag, err := newSSIAGCommand()
	if err != nil {
		return nil, err
	}
	stav := newSTAVCommand()
	root.AddCommand(
		ssiag, stav, newKnowledgeCommand(), newSKVICommand(), newSCLVCommand(),
		newSACVCommand(), newSODVCommand(), newSSFVCommand(), newMaestroCommand(),
		newValidateCommand(),
	)
	if err := commandregistry.Validate(root); err != nil {
		return nil, fmt.Errorf("qxctl command registry parity failed: %w", err)
	}
	return root, nil
}

func newKnowledgeCommand() *cobra.Command {
	command := structural("knowledge", fmt.Errorf("knowledge subcommand is required: invariant, engines, reconcile, session, or lifecycle"))
	command.AddCommand(newKnowledgeInvariantCommand())
	engines := structural("engines", fmt.Errorf("knowledge engines subcommand is required: list, inspect, doctor, bind, or unbind"))
	for _, operation := range []string{"list", "inspect", "doctor", "bind", "unbind"} {
		options := knowledgeEngineOptions{version: "0.1.0-dev"}
		child := &cobra.Command{
			Use: operation,
			Args: func(_ *cobra.Command, args []string) error {
				if operation == "inspect" || operation == "bind" || operation == "unbind" {
					if len(args) != 1 {
						return errUsageOnly
					}
					options.role = args[0]
					return nil
				}
				return usageOnlyArgs(nil, args)
			},
			RunE: func(*cobra.Command, []string) error {
				return runKnowledgeEngines(operation, options)
			},
		}
		if operation == "bind" || operation == "unbind" {
			registeredMutation(child, "knowledge.engines."+operation, featureBindings, "configure", "target_host_permission", "")
		} else {
			registered(child, "knowledge.engines."+operation, featureBindings, map[string]string{
				"list": "discover", "inspect": "inspect", "doctor": "validate",
			}[operation])
		}
		child.Flags().StringVar(
			&options.stateRoot, "state-root", "",
			"user state root; defaults to XDG_STATE_HOME or ~/.local/state",
		)
		child.Flags().BoolVar(&options.jsonOutput, "json", false, "emit JSON")
		if operation == "bind" {
			child.Flags().StringVar(&options.prefix, "prefix", "", "exact knowledge engine installation prefix")
			child.Flags().StringVar(&options.version, "version", "0.1.0-dev", "exact installed engine version")
		}
		if operation == "bind" || operation == "unbind" {
			child.Flags().StringVar(
				&options.expectedRegistryDigest, "expected-registry-digest", "",
				"required prior registry state: absent or exact tagged SHA-256 digest",
			)
		}
		child.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		engines.AddCommand(child)
	}
	engines.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	command.AddCommand(engines)

	reconcile := structural("reconcile", fmt.Errorf("knowledge reconcile subcommand is required: compatibility, begin, status, checkpoint, close, or recover"))
	for _, operation := range []string{"compatibility", "begin", "status", "checkpoint", "close", "recover"} {
		options := knowledgeReconcileOptions{}
		child := &cobra.Command{
			Use:  operation,
			Args: usageOnlyArgs,
			RunE: func(*cobra.Command, []string) error {
				return runKnowledgeReconcile(operation, options)
			},
		}
		interaction := "lifecycle"
		if operation == "compatibility" || operation == "status" {
			interaction = "query"
		} else if operation == "recover" {
			interaction = "recover"
		}
		if operation == "compatibility" || operation == "status" {
			registered(child, "knowledge.reconcile."+operation, featureQXCTL, interaction)
		} else {
			registeredMutation(child, "knowledge.reconcile."+operation, featureQXCTL, interaction, "target_host_permission", "knowledge.reconcile.recover")
		}
		child.Flags().StringVar(
			&options.stateRoot, "state-root", "",
			"user state root; defaults to XDG_STATE_HOME or ~/.local/state",
		)
		child.Flags().StringVar(
			&options.repository, "repo", "",
			"Symphony repository path; defaults to the current repository",
		)
		child.Flags().BoolVar(&options.jsonOutput, "json", false, "emit operation result JSON")
		if operation == "begin" || operation == "checkpoint" ||
			operation == "close" || operation == "recover" {
			child.Flags().StringVar(
				&options.operationID, "operation-id", "",
				"stable idempotency token for this mutation",
			)
			child.Flags().StringVar(
				&options.expectedJournalDigest, "expected-journal-digest", "",
				"required prior journal state: absent or exact tagged SHA-256 digest",
			)
		}
		if operation == "begin" {
			child.Flags().StringSliceVar(
				&options.paths, "path", nil,
				"repository-relative file to reconcile; repeat for additional files",
			)
		}
		if operation == "recover" {
			child.Flags().BoolVar(
				&options.discover, "discover", false,
				"recover from uniquely validated local evidence when the head is unavailable",
			)
		}
		child.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		reconcile.AddCommand(child)
	}
	reconcile.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	command.AddCommand(reconcile)

	session := structural("session", fmt.Errorf("knowledge session subcommand is required: begin, status, checkpoint, close, recover, transition, or features"))
	for _, operation := range []string{"begin", "status", "checkpoint", "close", "recover"} {
		options := knowledgeSessionOptions{scope: "user", ttl: 15 * time.Minute}
		child := &cobra.Command{
			Use:  operation,
			Args: usageOnlyArgs,
			RunE: func(*cobra.Command, []string) error {
				return runKnowledgeSession(operation, options)
			},
		}
		if operation == "status" {
			registered(child, "knowledge.session.status", featureSessions, "query")
		} else {
			interaction := "lifecycle"
			if operation == "recover" {
				interaction = "recover"
			}
			registeredMutation(child, "knowledge.session."+operation, featureSessions, interaction, "ssiag", "knowledge.session.recover")
		}
		child.Flags().StringVar(&options.topsID, "tops-id", "", "immutable TOPS UUID")
		child.Flags().StringVar(&options.scope, "scope", "user", "SSIAG installation scope: user or system")
		child.Flags().StringVar(&options.stateRoot, "state-root", "", "user state root; defaults to XDG_STATE_HOME or ~/.local/state")
		child.Flags().StringVar(&options.repository, "repo", "", "Symphony repository path; defaults to the current repository")
		child.Flags().DurationVar(&options.ttl, "ttl", 15*time.Minute, "requested authority-epoch lifetime")
		child.Flags().BoolVar(&options.jsonOutput, "json", false, "emit operation result JSON")
		if operation != "status" {
			child.Flags().StringVar(&options.operationID, "operation-id", "", "stable idempotency token for this mutation")
			child.Flags().StringVar(&options.expectedJournalDigest, "expected-journal-digest", "", "required prior session state: absent or exact tagged SHA-256 digest")
		}
		if operation == "begin" || operation == "checkpoint" {
			child.Flags().StringSliceVar(&options.contextRefs, "context-ref", nil, "reconciliation context reference to attach; repeat as needed")
		}
		if operation == "recover" {
			child.Flags().BoolVar(&options.discover, "discover", false, "recover from uniquely validated local evidence when the head is unavailable")
		}
		child.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		session.AddCommand(child)
	}
	transitionOptions := knowledgeSessionOptions{scope: "user", ttl: 15 * time.Minute}
	transition := &cobra.Command{
		Use:  "transition",
		Args: usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error {
			return runKnowledgeSessionTransition(transitionOptions)
		},
	}
	registeredMutation(transition, "knowledge.session.transition", featureSessions, "lifecycle", "ssiag", "knowledge.session.recover")
	transition.Flags().StringVar(&transitionOptions.topsID, "tops-id", "", "immutable TOPS UUID")
	transition.Flags().StringVar(&transitionOptions.scope, "scope", "user", "SSIAG installation scope: user or system")
	transition.Flags().StringVar(&transitionOptions.stateRoot, "state-root", "", "user state root; defaults to XDG_STATE_HOME or ~/.local/state")
	transition.Flags().StringVar(&transitionOptions.repository, "repo", "", "Symphony repository path; defaults to the current repository")
	transition.Flags().StringVar(&transitionOptions.event, "event", "", "explicit lifecycle event: login, refresh, or logout")
	transition.Flags().StringVar(&transitionOptions.eventID, "event-id", "", "stable idempotency token for the host lifecycle event")
	transition.Flags().StringSliceVar(&transitionOptions.contextRefs, "context-ref", nil, "reconciliation context reference to attach; repeat as needed")
	transition.Flags().BoolVar(&transitionOptions.recoverTransition, "recover", false, "attempt discovery recovery before transition when status evidence is damaged")
	transition.Flags().DurationVar(&transitionOptions.ttl, "ttl", 15*time.Minute, "requested authority-epoch lifetime")
	transition.Flags().BoolVar(&transitionOptions.jsonOutput, "json", false, "emit transition result JSON")
	transition.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	session.AddCommand(transition)

	features := structural("features", fmt.Errorf("knowledge session features subcommand is required: begin, status, checkpoint, close, or recover"))
	for _, operation := range []string{"begin", "status", "checkpoint", "close", "recover"} {
		options := knowledgeSSFVMaintenanceOptions{
			scope: "user", ttl: 15 * time.Minute, maestroVersion: "0.1.0-dev",
		}
		child := &cobra.Command{
			Use:  operation,
			Args: usageOnlyArgs,
			RunE: func(*cobra.Command, []string) error {
				return runKnowledgeSSFVMaintenance(operation, options)
			},
		}
		if operation == "status" {
			registered(child, "knowledge.session.features.status", featureSessions, "query")
		} else {
			interaction := "lifecycle"
			if operation == "recover" {
				interaction = "recover"
			}
			registeredMutation(child, "knowledge.session.features."+operation, featureSessions, interaction, "ssiag", "knowledge.session.features.recover")
		}
		child.Flags().StringVar(&options.topsID, "tops-id", "", "immutable TOPS UUID")
		child.Flags().StringVar(&options.scope, "scope", "user", "SSIAG installation scope: user or system")
		child.Flags().StringVar(&options.stateRoot, "state-root", "", "state root; defaults to XDG_STATE_HOME or ~/.local/state")
		child.Flags().StringVar(&options.repository, "repo", "", "Symphony repository path; defaults to the current repository")
		child.Flags().DurationVar(&options.ttl, "ttl", 15*time.Minute, "requested maintenance authorization lifetime")
		child.Flags().BoolVar(&options.jsonOutput, "json", false, "emit operation result JSON")
		if operation != "status" {
			child.Flags().StringVar(&options.operationID, "operation-id", "", "stable idempotency token for this maintenance mutation")
			child.Flags().StringVar(&options.expectedJournalDigest, "expected-journal-digest", "", "required prior maintenance state: absent or exact tagged SHA-256 digest")
		}
		if operation == "begin" || operation == "checkpoint" || operation == "close" {
			child.Flags().StringVar(&options.maestroPrefix, "maestro-prefix", "", "optional exact Maestro installation prefix for derived receptor inventory evidence")
			child.Flags().StringVar(&options.maestroVersion, "maestro-version", "0.1.0-dev", "exact Maestro version when inventory evidence is enabled")
		}
		if operation == "recover" {
			child.Flags().BoolVar(&options.discover, "discover", false, "recover from uniquely validated digest-linked local maintenance evidence")
		}
		child.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		features.AddCommand(child)
	}
	features.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	session.AddCommand(features)
	session.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	command.AddCommand(session)

	lifecycle := structural("lifecycle", fmt.Errorf("knowledge lifecycle subcommand is required: profile, ownership, host, observe, report, boot, status, recover, apply, apply-status, or apply-recover"))
	profile := structural("profile", fmt.Errorf("knowledge lifecycle profile subcommand is required: list, show, set, or remove"))
	for _, operation := range []string{"list", "show", "set", "remove"} {
		options := knowledgeLifecycleOptions{scope: "user", profileID: "default", ttl: 15 * time.Minute}
		child := &cobra.Command{
			Use:  operation,
			Args: usageOnlyArgs,
			RunE: func(*cobra.Command, []string) error {
				return runKnowledgeLifecycleProfile(operation, options)
			},
		}
		if operation == "list" || operation == "show" {
			registered(child, "knowledge.lifecycle.profile."+operation, featureLifecycle, map[string]string{"list": "discover", "show": "inspect"}[operation])
		} else {
			registeredMutation(child, "knowledge.lifecycle.profile."+operation, featureLifecycle, "configure", "ssiag", "knowledge.lifecycle.recover")
		}
		addKnowledgeLifecycleCommonFlags(child, &options)
		if operation == "set" {
			child.Flags().StringVar(&options.input, "input", "", "bounded no-follow lifecycle profile input JSON")
		}
		if operation == "set" || operation == "remove" {
			child.Flags().StringVar(
				&options.expectedProfileDigest, "expected-profile-digest", "",
				"required prior profile state: absent or exact tagged SHA-256 digest",
			)
		}
		child.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		profile.AddCommand(child)
	}
	profile.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	lifecycle.AddCommand(profile)

	ownership := structural("ownership", fmt.Errorf("knowledge lifecycle ownership subcommand is required: status, reconcile, adopt, or release"))
	for _, operation := range []string{"status", "reconcile", "adopt", "release"} {
		options := knowledgeLifecycleOptions{scope: "user", profileID: "default", ttl: 15 * time.Minute}
		child := &cobra.Command{
			Use:  operation,
			Args: usageOnlyArgs,
			RunE: func(*cobra.Command, []string) error {
				return runKnowledgeLifecycleOwnership(operation, options)
			},
		}
		if operation == "status" {
			registered(child, "knowledge.lifecycle.ownership.status", featureLifecycle, "query")
		} else {
			registeredMutation(child, "knowledge.lifecycle.ownership."+operation, featureLifecycle, "lifecycle", "ssiag", "knowledge.lifecycle.recover")
		}
		addKnowledgeLifecycleCommonFlags(child, &options)
		child.Flags().StringVar(&options.ownershipRoot, "root", "", "exact configured shared installation root")
		if operation == "adopt" || operation == "release" {
			child.Flags().StringVar(&options.expectedOwnershipDigest, "expected-ownership-registry-digest", "", "exact ownership registry compare-and-swap digest")
		}
		if operation == "release" {
			child.Flags().StringVar(&options.receiptDigest, "receipt-digest", "", "exact legacy-preserve receipt digest to release")
		}
		child.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		ownership.AddCommand(child)
	}
	ownership.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	lifecycle.AddCommand(ownership)

	host := structural("host", fmt.Errorf("knowledge lifecycle host subcommand is required: install, update, status, reconcile, enable, disable, uninstall, or run"))
	for _, operation := range []string{"install", "update", "status", "reconcile", "enable", "disable", "uninstall", "run"} {
		options := knowledgeLifecycleOptions{scope: "system", profileID: "default", ttl: 15 * time.Minute, recoveryMode: "discover"}
		child := &cobra.Command{
			Use:  operation,
			Args: usageOnlyArgs,
			RunE: func(*cobra.Command, []string) error {
				return runKnowledgeLifecycleHost(operation, options)
			},
		}
		if operation == "status" {
			registered(child, "knowledge.lifecycle.host.status", featureHost, "query")
		} else {
			recovery := "knowledge.lifecycle.host.reconcile"
			if operation == "run" {
				recovery = "knowledge.lifecycle.recover"
			}
			registeredMutation(child, "knowledge.lifecycle.host."+operation, featureHost, "lifecycle", "target_host_permission", recovery)
		}
		addKnowledgeLifecycleHostFlags(child, &options)
		if operation == "install" || operation == "update" {
			child.Flags().StringVar(&options.integrationRoot, "integration-root", "", "exact protected systemd integration root; defaults per TOPS/profile")
			child.Flags().StringVar(&options.recoveryMode, "recovery-mode", "discover", "boot evidence recovery: strict or discover")
			child.Flags().BoolVar(&options.hostDisabled, "disabled", false, "install the unit without enabling it for host boot")
		}
		if operation == "install" || operation == "update" || operation == "enable" || operation == "disable" || operation == "uninstall" {
			child.Flags().StringVar(&options.expectedHostDigest, "expected-host-digest", "", "required prior host integration state: absent or exact tagged SHA-256 digest")
		}
		child.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		host.AddCommand(child)
	}
	host.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	lifecycle.AddCommand(host)

	observeOptions := knowledgeLifecycleOptions{scope: "user", profileID: "default", ttl: 15 * time.Minute}
	observe := &cobra.Command{
		Use:  "observe",
		Args: usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error {
			return runKnowledgeLifecycleObserve(observeOptions)
		},
	}
	registered(observe, "knowledge.lifecycle.observe", featureLifecycle, "inspect")
	addKnowledgeLifecycleCommonFlags(observe, &observeOptions)
	observe.Flags().StringSliceVar(
		&observeOptions.configuredRoots, "root", nil,
		"explicit trusted installation root for bootstrap observation; repeat as needed",
	)
	observe.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	lifecycle.AddCommand(observe)

	reportOptions := knowledgeLifecycleOptions{scope: "user", profileID: "default", ttl: 15 * time.Minute}
	report := &cobra.Command{
		Use:  "report",
		Args: usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error {
			return runKnowledgeLifecycleReport(reportOptions)
		},
	}
	registered(report, "knowledge.lifecycle.report", featureLifecycle, "query")
	addKnowledgeLifecycleCommonFlags(report, &reportOptions)
	report.Flags().StringVar(
		&reportOptions.priorAppliedStateDigest, "prior-applied-state-digest", "",
		"optional exact tagged SHA-256 digest of the last applied-state evidence",
	)
	report.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	lifecycle.AddCommand(report)

	bootOptions := knowledgeLifecycleOptions{scope: "user", profileID: "default", ttl: 15 * time.Minute}
	boot := &cobra.Command{
		Use:  "boot",
		Args: usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error {
			return runKnowledgeLifecycleBoot(bootOptions)
		},
	}
	registeredMutation(boot, "knowledge.lifecycle.boot", featureLifecycle, "lifecycle", "ssiag", "knowledge.lifecycle.recover")
	addKnowledgeLifecycleCommonFlags(boot, &bootOptions)
	boot.Flags().StringVar(&bootOptions.operationID, "operation-id", "", "stable idempotency token for this durable boot observation")
	boot.Flags().StringVar(&bootOptions.expectedJournalDigest, "expected-journal-digest", "", "required prior journal state: absent or exact tagged SHA-256 digest")
	boot.Flags().StringVar(&bootOptions.priorAppliedStateDigest, "prior-applied-state-digest", "", "optional exact tagged SHA-256 digest of the last applied-state evidence")
	boot.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	lifecycle.AddCommand(boot)

	statusOptions := knowledgeLifecycleOptions{scope: "user", profileID: "default", ttl: 15 * time.Minute}
	status := &cobra.Command{
		Use:  "status",
		Args: usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error {
			return runKnowledgeLifecycleBootState("lifecycle_boot_status", statusOptions)
		},
	}
	registered(status, "knowledge.lifecycle.status", featureLifecycle, "query")
	addKnowledgeLifecycleCommonFlags(status, &statusOptions)
	status.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	lifecycle.AddCommand(status)

	recoverOptions := knowledgeLifecycleOptions{scope: "user", profileID: "default", ttl: 15 * time.Minute}
	recover := &cobra.Command{
		Use:  "recover",
		Args: usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error {
			return runKnowledgeLifecycleBootState("lifecycle_boot_recover", recoverOptions)
		},
	}
	registeredMutation(recover, "knowledge.lifecycle.recover", featureLifecycle, "recover", "ssiag", "knowledge.lifecycle.recover")
	addKnowledgeLifecycleCommonFlags(recover, &recoverOptions)
	recover.Flags().StringVar(&recoverOptions.operationID, "operation-id", "", "stable idempotency token for this recovery mutation")
	recover.Flags().StringVar(&recoverOptions.expectedJournalDigest, "expected-journal-digest", "", "exact recoverable journal digest")
	recover.Flags().BoolVar(&recoverOptions.discover, "discover", false, "recover from uniquely validated digest-linked local evidence")
	recover.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	lifecycle.AddCommand(recover)

	applyOptions := knowledgeLifecycleOptions{scope: "user", profileID: "default", ttl: 15 * time.Minute, maxActions: 4096}
	apply := &cobra.Command{
		Use:  "apply",
		Args: usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error {
			return runKnowledgeLifecycleApply(applyOptions)
		},
	}
	registeredMutation(apply, "knowledge.lifecycle.apply", featureLifecycle, "apply", "ssiag", "knowledge.lifecycle.apply-recover")
	addKnowledgeLifecycleCommonFlags(apply, &applyOptions)
	apply.Flags().StringVar(&applyOptions.operationID, "operation-id", "", "stable base idempotency token for this apply transaction")
	apply.Flags().StringVar(&applyOptions.sourceJournalDigest, "source-journal-digest", "", "exact apply-compatible report journal digest")
	apply.Flags().StringVar(&applyOptions.expectedApplyDigest, "expected-apply-journal-digest", "", "required prior apply journal state: absent or exact tagged SHA-256 digest")
	apply.Flags().StringVar(&applyOptions.expectedAppliedDigest, "expected-applied-state-digest", "", "required prior applied state: absent or exact tagged SHA-256 digest")
	apply.Flags().StringSliceVar(&applyOptions.stagedRoots, "source-root", nil, "trusted staged v2 package root; repeat as needed")
	apply.Flags().Uint64Var(&applyOptions.maxActions, "max-actions", 4096, "bounded actions to converge in this invocation")
	apply.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	lifecycle.AddCommand(apply)

	applyStatusOptions := knowledgeLifecycleOptions{scope: "user", profileID: "default", ttl: 15 * time.Minute}
	applyStatus := &cobra.Command{
		Use:  "apply-status",
		Args: usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error {
			return runKnowledgeLifecycleApplyState("lifecycle_apply_status", applyStatusOptions)
		},
	}
	registered(applyStatus, "knowledge.lifecycle.apply-status", featureLifecycle, "query")
	addKnowledgeLifecycleCommonFlags(applyStatus, &applyStatusOptions)
	applyStatus.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	lifecycle.AddCommand(applyStatus)

	applyRecoverOptions := knowledgeLifecycleOptions{scope: "user", profileID: "default", ttl: 15 * time.Minute}
	applyRecover := &cobra.Command{
		Use:  "apply-recover",
		Args: usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error {
			return runKnowledgeLifecycleApplyState("lifecycle_apply_recover", applyRecoverOptions)
		},
	}
	registeredMutation(applyRecover, "knowledge.lifecycle.apply-recover", featureLifecycle, "recover", "ssiag", "knowledge.lifecycle.apply-recover")
	addKnowledgeLifecycleCommonFlags(applyRecover, &applyRecoverOptions)
	applyRecover.Flags().StringVar(&applyRecoverOptions.operationID, "operation-id", "", "stable idempotency token for apply recovery")
	applyRecover.Flags().StringVar(&applyRecoverOptions.expectedApplyDigest, "expected-apply-journal-digest", "", "exact recoverable apply journal digest")
	applyRecover.Flags().BoolVar(&applyRecoverOptions.discover, "discover", false, "recover from uniquely validated digest-linked local apply evidence")
	applyRecover.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	lifecycle.AddCommand(applyRecover)
	lifecycle.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	command.AddCommand(lifecycle)
	command.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	return command
}

func addKnowledgeLifecycleCommonFlags(command *cobra.Command, options *knowledgeLifecycleOptions) {
	if options.maestroVersion == "" {
		options.maestroVersion = "0.1.0-dev"
	}
	command.Flags().StringVar(&options.topsID, "tops-id", "", "immutable TOPS UUID")
	command.Flags().StringVar(&options.scope, "scope", "user", "SSIAG installation scope: user or system")
	command.Flags().StringVar(&options.stateRoot, "state-root", "", "state root; defaults to XDG_STATE_HOME or ~/.local/state")
	command.Flags().StringVar(&options.repository, "repo", "", "Symphony repository path; defaults to the current repository")
	command.Flags().StringVar(&options.maestroPrefix, "maestro-prefix", "", "exact Maestro installation prefix for docking observation and apply")
	command.Flags().StringVar(&options.maestroVersion, "maestro-version", "0.1.0-dev", "exact Maestro version")
	command.Flags().StringSliceVar(&options.maestroReceptorIDs, "maestro-receptor-id", nil, "exact receptor to observe or mutate; repeat for receptor-switch recovery")
	command.Flags().StringVar(&options.profileID, "profile-id", "default", "exact lifecycle profile identity")
	command.Flags().DurationVar(&options.ttl, "ttl", 15*time.Minute, "requested lifecycle authorization lifetime")
	command.Flags().BoolVar(&options.jsonOutput, "json", false, "emit JSON")
}

func addKnowledgeLifecycleHostFlags(command *cobra.Command, options *knowledgeLifecycleOptions) {
	command.Flags().StringVar(&options.topsID, "tops-id", "", "immutable TOPS UUID")
	command.Flags().StringVar(&options.scope, "scope", "system", "SSIAG installation scope; host integration requires system")
	command.Flags().StringVar(&options.stateRoot, "state-root", "", "state root; Linux host integration defaults to /var/lib")
	command.Flags().StringVar(&options.repository, "repo", "", "Symphony repository path; defaults to the current repository")
	command.Flags().StringVar(&options.profileID, "profile-id", "default", "exact lifecycle profile identity")
	command.Flags().DurationVar(&options.ttl, "ttl", 15*time.Minute, "requested lifecycle authorization lifetime")
	command.Flags().BoolVar(&options.jsonOutput, "json", false, "emit JSON")
}

func newSSFVCommand() *cobra.Command {
	command := structural("ssfv", fmt.Errorf("SSFV subcommand is required: inspect, check, diff, propose, graph, or administration-check"))
	for _, operation := range []string{"inspect", "check", "diff", "propose", "graph", "administration-check"} {
		options := ssfvOptions{version: "0.1.0-dev", freshness: "disabled"}
		child := &cobra.Command{
			Use:  operation,
			Args: usageOnlyArgs,
			RunE: func(command *cobra.Command, _ []string) error {
				if operation == "check" && options.baseline != "" &&
					!command.Flags().Changed("freshness") {
					options.freshness = "report"
				}
				return runSSFV(operation, options)
			},
		}
		featureID := map[string]string{
			"inspect": featureSSFV, "check": featureSSFVSnapshot,
			"diff": featureSSFVComparison, "propose": featureSSFVProposal,
			"graph":                featureSSFVProjection,
			"administration-check": featureAdministrationAssurance,
		}[operation]
		if operation == "propose" {
			registeredProposal(child, "ssfv.propose", featureID)
		} else if operation == "administration-check" {
			registeredEvidence(child, "ssfv.administration-check", featureID, "validate")
		} else {
			interaction := map[string]string{"inspect": "inspect", "check": "validate", "diff": "validate", "graph": "query"}[operation]
			registered(child, "ssfv."+operation, featureID, interaction)
		}
		if spec, err := commandregistry.Spec(child); err == nil {
			spec.BackendOperationIDs = []string{"engop:symphony:ssfv." + operation}
			if operation == "administration-check" {
				spec.InputProtocols = []string{"symphony.knowledge.administration-coverage-input.v1"}
				spec.OutputProtocols = []string{"symphony.knowledge.administration-coverage-result.v1"}
				spec.ResultValidationProtocols = []string{"symphony.knowledge.administration-coverage-result.v1"}
			}
			commandregistry.Attach(child, spec)
		}
		child.Flags().StringVar(&options.prefix, "prefix", "", "exact SSFV installation prefix")
		child.Flags().StringVar(&options.version, "version", "0.1.0-dev", "exact installed SSFV engine version")
		child.Flags().StringVar(&options.repository, "repo", "", "Symphony repository path; defaults to the current repository")
		child.Flags().BoolVar(&options.jsonOutput, "json", false, "emit operation result JSON")
		if operation == "check" {
			child.Flags().StringVar(&options.expectedNamespaceDigest, "expected-namespace-digest", "", "optional expected tagged SHA-256 namespace digest")
			child.Flags().StringVar(&options.expectedRegistryDigest, "expected-registry-digest", "", "optional expected tagged SHA-256 registry digest")
			child.Flags().StringVar(&options.baseline, "baseline", "", "bounded no-follow semantic snapshot JSON file")
			child.Flags().StringVar(&options.freshness, "freshness", "disabled", "semantic freshness mode: disabled, report, or require")
		}
		if operation == "diff" || operation == "propose" || operation == "administration-check" {
			child.Flags().StringVar(&options.input, "input", "", "no-follow JSON operation payload file")
		}
		child.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		command.AddCommand(child)
	}
	command.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	return command
}

func newSODVCommand() *cobra.Command {
	command := structural("sodv", fmt.Errorf("SODV subcommand is required: inspect, check, verify, propose, recover, or project"))
	for _, operation := range []string{"inspect", "check", "verify", "propose", "recover", "project"} {
		options := sodvOptions{version: "0.1.0-dev"}
		child := &cobra.Command{
			Use:  operation,
			Args: usageOnlyArgs,
			RunE: func(*cobra.Command, []string) error { return runSODV(operation, options) },
		}
		featureID := map[string]string{
			"inspect": featureSODV, "check": featureSODVLedger,
			"verify": featureSODVVerification, "propose": featureSODVProposal,
			"recover": featureSODVRecovery, "project": featureSODVProjection,
		}[operation]
		if operation == "propose" {
			registeredProposal(child, "sodv.propose", featureID)
		} else if operation == "recover" {
			registeredEvidence(child, "sodv.recover", featureID, "recover")
		} else {
			interaction := map[string]string{"inspect": "inspect", "check": "validate", "verify": "validate", "recover": "recover", "project": "query"}[operation]
			registered(child, "sodv."+operation, featureID, interaction)
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
	command := structural("sacv", fmt.Errorf("SACV subcommand is required: inspect, check, diff, propose, or project"))
	for _, operation := range []string{"inspect", "check", "diff", "propose", "project"} {
		options := sacvOptions{version: "0.1.0-dev"}
		child := &cobra.Command{
			Use:  operation,
			Args: usageOnlyArgs,
			RunE: func(*cobra.Command, []string) error { return runSACV(operation, options) },
		}
		featureID := map[string]string{
			"inspect": featureSACV, "check": featureSACVConformance,
			"diff": featureSACVCompatibility, "propose": featureSACVProposal,
			"project": featureSACVProjection,
		}[operation]
		if operation == "propose" {
			registeredProposal(child, "sacv.propose", featureID)
		} else {
			interaction := map[string]string{"inspect": "inspect", "check": "validate", "diff": "validate", "project": "query"}[operation]
			registered(child, "sacv."+operation, featureID, interaction)
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
	command := structural("sclv", fmt.Errorf("SCLV subcommand is required: inspect, check, propose, recover, project, or evidence"))
	for _, operation := range []string{"inspect", "check", "propose", "recover", "project"} {
		options := sclvOptions{version: "0.1.0-dev"}
		child := &cobra.Command{
			Use:  operation,
			Args: usageOnlyArgs,
			RunE: func(*cobra.Command, []string) error { return runSCLV(operation, options) },
		}
		featureID := map[string]string{
			"inspect": featureSCLV, "check": featureSCLVAssurance,
			"propose": featureSCLVProposal, "recover": featureSCLVRecovery,
			"project": featureSCLVProjection,
		}[operation]
		if operation == "propose" {
			registeredProposal(child, "sclv.propose", featureID)
		} else if operation == "recover" {
			registeredEvidence(child, "sclv.recover", featureID, "recover")
		} else {
			interaction := map[string]string{"inspect": "inspect", "check": "validate", "recover": "recover", "project": "query"}[operation]
			registered(child, "sclv."+operation, featureID, interaction)
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
	evidence := structural("evidence", fmt.Errorf("SCLV evidence adapter is required: local-git or airgap"))
	for _, adapter := range []string{"local-git", "airgap"} {
		options := sclvOptions{version: "0.1.0-dev"}
		child := &cobra.Command{
			Use:  adapter,
			Args: usageOnlyArgs,
			RunE: func(*cobra.Command, []string) error { return runSCLVEvidence(adapter, options) },
		}
		registeredEvidence(child, "sclv.evidence."+adapter, featureSCLVEvidence, "invoke")
		child.Flags().StringVar(&options.prefix, "prefix", "", "exact SCLV installation prefix")
		child.Flags().StringVar(&options.version, "version", "0.1.0-dev", "exact installed SCLV package version")
		child.Flags().StringVar(&options.repository, "repo", "", "Symphony repository path; defaults to the current repository")
		child.Flags().StringVar(&options.input, "input", "", "no-follow JSON provider-evidence input file")
		child.Flags().BoolVar(&options.jsonOutput, "json", false, "emit normalized evidence JSON")
		child.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		evidence.AddCommand(child)
	}
	evidence.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	command.AddCommand(evidence)
	command.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	return command
}

func newSKVICommand() *cobra.Command {
	command := structural("skvi", fmt.Errorf("SKVI subcommand is required: inspect, check, propose, or project"))
	for _, operation := range []string{"inspect", "check", "propose", "project"} {
		options := skviOptions{version: "0.1.0-dev"}
		child := &cobra.Command{
			Use:  operation,
			Args: usageOnlyArgs,
			RunE: func(*cobra.Command, []string) error { return runSKVI(operation, options) },
		}
		featureID := map[string]string{
			"inspect": featureSKVI, "check": featureSKVIAssurance,
			"propose": featureSKVIProposal, "project": featureSKVIProjection,
		}[operation]
		if operation == "propose" {
			registeredProposal(child, "skvi.propose", featureID)
		} else {
			interaction := map[string]string{"inspect": "inspect", "check": "validate", "project": "query"}[operation]
			registered(child, "skvi."+operation, featureID, interaction)
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

func operationCommand(use, featureID, interaction string, run func() error) *cobra.Command {
	command := &cobra.Command{
		Use:  use,
		Args: usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error { return run() },
	}
	registered(command, use, featureID, interaction)
	command.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	return command
}

func jsonOperationCommand(use, featureID, interaction string, run func(bool) error) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use:  use,
		Args: usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error { return run(jsonOutput) },
	}
	registered(command, use, featureID, interaction)
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit JSON")
	command.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	return command
}

func newInventoryCommand() *cobra.Command {
	command := jsonOperationCommand("inventory", featureQXCTL, "query", runInventory)
	digest := jsonOperationCommand("digest", featureQXCTL, "query", runInventoryDigest)
	registered(digest, "inventory.digest", featureQXCTL, "query")
	command.AddCommand(digest)
	return command
}

func newModulesCommand() *cobra.Command {
	command := operationCommand("modules", featureQXCTL, "discover", runModules)
	command.AddCommand(
		operationCommand("check", featureQXCTL, "validate", runModulesCheck),
		jsonOperationCommand("metadata", featureQXCTL, "inspect", runModulesMetadata),
	)
	for _, child := range command.Commands() {
		registered(child, "modules."+child.Name(), featureQXCTL, map[string]string{"check": "validate", "metadata": "inspect"}[child.Name()])
	}
	return command
}

func newModuleCommand() *cobra.Command {
	command := structural("module", errUsageOnly)
	command.AddCommand(
		namedModuleCommand("inspect", "inspect", runModuleInspect),
		namedModuleCommand("check", "validate", runModuleCheck),
		namedModuleMetadataCommand(),
	)
	command.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	return command
}

func namedModuleCommand(use, interaction string, run func(string) error) *cobra.Command {
	command := &cobra.Command{
		Use:  use + " <module-name>",
		Args: exactOneUsageArg,
		RunE: func(_ *cobra.Command, args []string) error { return run(args[0]) },
	}
	registered(command, "module."+use, featureQXCTL, interaction)
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
	registered(command, "module.metadata", featureQXCTL, "inspect")
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit JSON")
	command.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	return command
}

func newSSIAGCommand() (*cobra.Command, error) {
	command := structural("ssiag", fmt.Errorf("SSIAG subcommand is required: status, providers, provider, doctor, grants, policy, enrollment, or supervisor"))
	for _, subcommand := range []string{"status", "providers", "doctor"} {
		child, err := newSSIAGLeaf(subcommand)
		if err != nil {
			return nil, err
		}
		command.AddCommand(child)
	}
	providerCommand, err := newSSIAGProviderCommand()
	if err != nil {
		return nil, err
	}
	command.AddCommand(providerCommand)
	grants := structural("grants", fmt.Errorf("SSIAG grants subcommand is required: lifecycle"))
	grantOptions := ssiagOptions{scope: "user", profileID: "default", authorityBasis: "host_owner"}
	lifecycle := &cobra.Command{
		Use:  "lifecycle",
		Args: usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error {
			return runSSIAGLifecycleGrantPlan(grantOptions)
		},
	}
	registeredProposal(lifecycle, "ssiag.grants.lifecycle", featureSSIAG)
	lifecycle.Flags().StringVar(&grantOptions.topsID, "tops-id", "", "immutable TOPS UUID")
	lifecycle.Flags().StringVar(&grantOptions.scope, "scope", "user", "SSIAG scope: user or system")
	lifecycle.Flags().StringVar(&grantOptions.profileID, "profile-id", "default", "exact lifecycle profile identity")
	lifecycle.Flags().StringVar(&grantOptions.subjectID, "subject-id", "", "configured canonical SSIAG subject identity")
	lifecycle.Flags().StringVar(&grantOptions.authorityBasis, "authority-basis", "host_owner", "host_owner or granted_permission")
	lifecycle.Flags().BoolVar(&grantOptions.jsonOutput, "json", false, "emit JSON")
	grants.AddCommand(lifecycle)
	command.AddCommand(grants)
	command.AddCommand(newSSIAGPolicyCommand())
	command.AddCommand(
		newFoundationLifecycleFamily("ssiag", "enrollment", featureSSIAG),
		newFoundationLifecycleFamily("ssiag", "supervisor", featureSSIAG),
	)
	return command, nil
}

func newSSIAGProviderCommand() (*cobra.Command, error) {
	command := structural("provider", fmt.Errorf("SSIAG provider subcommand is required: show, verify, readiness, installations, or binding"))
	for _, operation := range []string{"show", "verify", "readiness"} {
		mapper := viper.New()
		operation := operation
		child := &cobra.Command{
			Use: operation + " <provider-name>",
			Args: func(_ *cobra.Command, args []string) error {
				if len(args) != 1 {
					return fmt.Errorf("SSIAG provider %s requires exactly one provider name", operation)
				}
				return nil
			},
			RunE: func(_ *cobra.Command, args []string) error {
				return runSSIAGProvider(operation, ssiagOptions{
					providerName: args[0], topsID: mapper.GetString("tops-id"),
					scope: mapper.GetString("scope"), jsonOutput: mapper.GetBool("json"),
					authorityBasis: mapper.GetString("authority-basis"),
				})
			},
		}
		spec := commandSpec("ssiag.provider."+operation, featureSSIAG, map[string]string{"show": "inspect", "verify": "validate", "readiness": "validate"}[operation])
		if operation == "verify" {
			spec.BackendOperationIDs = []string{
				"engop:symphony:ssiag.provider.metadata-handshake",
				"engop:symphony:ssiag.provider.trust.verify",
			}
		} else if operation == "readiness" {
			spec.BackendOperationIDs = []string{
				"engop:symphony:ssiag.provider.readiness.observe",
				"engop:symphony:ssiag.macos-keychain-provider.readiness.observe",
			}
		} else {
			spec.BackendOperationIDs = []string{"engop:symphony:ssiag.provider.trust.show"}
		}
		outputProtocol := "symphony.ssiag.provider-trust-result.v1"
		if operation == "readiness" {
			outputProtocol = ssiagclient.ProviderReadinessResultProtocol
		}
		spec.OutputProtocols = []string{outputProtocol}
		spec.ResultValidationProtocols = []string{outputProtocol}
		if operation == "verify" || operation == "readiness" {
			spec.Mutability = "evidence_only"
			spec.AuthorityMode = "ssiag"
			spec.InputProtocols = []string{map[string]string{
				"verify":    "symphony.ssiag.provider-trust-verification-request.v1",
				"readiness": ssiagclient.ProviderReadinessObservationRequestProtocol,
			}[operation]}
		}
		commandregistry.Attach(child, spec)
		child.Flags().String("tops-id", "", "immutable TOPS UUID")
		child.Flags().String("scope", "user", "SSIAG scope: user or system")
		child.Flags().Bool("json", false, "emit JSON")
		if operation == "verify" || operation == "readiness" {
			child.Flags().String("authority-basis", "host_owner", "host_owner or granted_permission")
		}
		for _, key := range []string{"tops-id", "scope", "json"} {
			if err := mapper.BindPFlag(key, child.Flags().Lookup(key)); err != nil {
				return nil, fmt.Errorf("bind SSIAG provider %s %s flag: %w", operation, key, err)
			}
		}
		if operation == "verify" || operation == "readiness" {
			if err := mapper.BindPFlag("authority-basis", child.Flags().Lookup("authority-basis")); err != nil {
				return nil, fmt.Errorf("bind SSIAG provider %s authority flag: %w", operation, err)
			}
		}
		if err := mapper.BindEnv("tops-id", "SYMPHONY_SSIAG_TOPS_ID"); err != nil {
			return nil, fmt.Errorf("bind SSIAG provider TOPS environment: %w", err)
		}
		command.AddCommand(child)
	}
	installations, err := newSSIAGProviderInstallationsCommand()
	if err != nil {
		return nil, err
	}
	binding, err := newSSIAGProviderBindingCommand()
	if err != nil {
		return nil, err
	}
	command.AddCommand(installations, binding)
	return command, nil
}

func newSSIAGPolicyCommand() *cobra.Command {
	command := structural("policy", fmt.Errorf("SSIAG policy subcommand is required: status, propose, apply, or recover"))
	statusOptions := ssiagOptions{scope: "user"}
	status := &cobra.Command{Use: "status", Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error {
		return runSSIAGPolicy("status", statusOptions)
	}}
	registered(status, "ssiag.policy.status", featureSSIAG, "query")
	addSSIAGPolicyCommonFlags(status, &statusOptions)
	command.AddCommand(status)

	proposalOptions := ssiagOptions{scope: "user", authorityBasis: "host_owner", ttl: 5 * time.Minute}
	propose := &cobra.Command{Use: "propose", Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error {
		return runSSIAGPolicy("propose", proposalOptions)
	}}
	registeredProposal(propose, "ssiag.policy.propose", featureSSIAG)
	addSSIAGPolicyCommonFlags(propose, &proposalOptions)
	propose.Flags().StringVar(&proposalOptions.input, "input", "", "bounded authorization policy JSON")
	propose.Flags().StringVar(&proposalOptions.expectedPolicy, "expected-policy-digest", "", "exact current policy digest")
	propose.Flags().StringVar(&proposalOptions.operationID, "operation-id", "", "stable retry/recovery operation identity")
	propose.Flags().StringVar(&proposalOptions.authorityBasis, "authority-basis", "host_owner", "host_owner or granted_permission")
	propose.Flags().BoolVar(&proposalOptions.reset, "reset", false, "restore the enrolled config policy")
	propose.Flags().DurationVar(&proposalOptions.ttl, "ttl", 5*time.Minute, "proposal lifetime, at most 10m")
	command.AddCommand(propose)

	applyOptions := ssiagOptions{scope: "user"}
	apply := &cobra.Command{Use: "apply", Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error {
		return runSSIAGPolicy("apply", applyOptions)
	}}
	registeredMutation(apply, "ssiag.policy.apply", featureSSIAG, "apply", "ssiag", "ssiag.policy.recover")
	addSSIAGPolicyCommonFlags(apply, &applyOptions)
	apply.Flags().StringVar(&applyOptions.input, "input", "", "bounded SSIAG policy proposal JSON")
	command.AddCommand(apply)

	recoverOptions := ssiagOptions{scope: "user"}
	recover := &cobra.Command{Use: "recover", Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error {
		return runSSIAGPolicy("recover", recoverOptions)
	}}
	registeredMutation(recover, "ssiag.policy.recover", featureSSIAG, "recover", "ssiag", "ssiag.policy.recover")
	addSSIAGPolicyCommonFlags(recover, &recoverOptions)
	recover.Flags().StringVar(&recoverOptions.operationID, "operation-id", "", "stable operation identity")
	recover.Flags().StringVar(&recoverOptions.expectedAttempt, "expected-attempt-digest", "", "exact pending attempt digest")
	recover.Flags().BoolVar(&recoverOptions.discover, "discover", false, "discover the unique pending attempt for the operation")
	command.AddCommand(recover)
	return command
}

func addSSIAGPolicyCommonFlags(command *cobra.Command, options *ssiagOptions) {
	command.Flags().StringVar(&options.topsID, "tops-id", "", "immutable TOPS UUID")
	command.Flags().StringVar(&options.scope, "scope", "user", "SSIAG scope: user or system")
	command.Flags().BoolVar(&options.jsonOutput, "json", false, "emit JSON")
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
	registered(command, "ssiag."+subcommand, featureSSIAG, map[string]string{"status": "query", "providers": "discover", "doctor": "validate"}[subcommand])
	if subcommand == "status" || subcommand == "providers" {
		spec, _ := commandregistry.Spec(command)
		protocol := map[string]string{"status": "symphony.ssiag.status.v1", "providers": "symphony.ssiag.providers.v1"}[subcommand]
		spec.OutputProtocols = []string{protocol}
		spec.ResultValidationProtocols = []string{protocol}
		commandregistry.Attach(command, spec)
	}
	if subcommand == "doctor" {
		spec, _ := commandregistry.Spec(command)
		unsupported := false
		spec.JSONOutput = &unsupported
		commandregistry.Attach(command, spec)
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
	command := structural("stav", fmt.Errorf("STAV subcommand is required: status, verify, query, doctor, enrollment, or supervisor"))
	appendCommand := &cobra.Command{
		Use:                "append",
		Hidden:             true,
		DisableFlagParsing: true,
		RunE: func(*cobra.Command, []string) error {
			return fmt.Errorf("qxctl stav append is prohibited; qxctl never submits arbitrary events or edits ledgers")
		},
	}
	appendSpec := commandSpec("stav.append", featureSTAV, "apply")
	appendSpec.Mutability = "prohibited"
	command.AddCommand(commandregistry.Attach(appendCommand, appendSpec))
	for _, subcommand := range []string{"status", "verify", "query", "doctor"} {
		command.AddCommand(newSTAVLeaf(subcommand))
	}
	command.AddCommand(
		newFoundationLifecycleFamily("stav", "enrollment", featureSTAV),
		newFoundationLifecycleFamily("stav", "supervisor", featureSTAV),
	)
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
	registered(command, "stav."+subcommand, featureSTAV, map[string]string{"status": "query", "verify": "validate", "query": "query", "doctor": "validate"}[subcommand])
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
	case "--help", "--version", "doctor", "contracts", "commands", "inventory", "status", "modules", "module", "ssiag", "stav", "knowledge", "skvi", "sclv", "sacv", "sodv", "ssfv", "maestro", "validate":
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
	case "ssiag", "stav":
		if len(args) > 2 && (args[1] == "enrollment" || args[1] == "supervisor") {
			switch args[2] {
			case "status", "plan", "apply", "apply-status", "recover":
				return args[0] + " " + args[1] + " " + args[2]
			}
		}
	case "knowledge":
		if len(args) > 2 && args[1] == "invariant" {
			switch args[2] {
			case "status", "list", "show", "check":
				return "knowledge invariant " + args[2]
			}
		}
		if len(args) > 2 && args[1] == "engines" {
			switch args[2] {
			case "list", "inspect", "doctor", "bind", "unbind":
				return "knowledge engines " + args[2]
			}
		}
		if len(args) > 2 && args[1] == "reconcile" {
			switch args[2] {
			case "compatibility", "begin", "status", "checkpoint", "close", "recover":
				return "knowledge reconcile " + args[2]
			}
		}
		if len(args) > 2 && args[1] == "session" {
			switch args[2] {
			case "begin", "status", "checkpoint", "close", "recover", "transition":
				return "knowledge session " + args[2]
			}
		}
		if len(args) > 2 && args[1] == "lifecycle" {
			if args[2] == "observe" || args[2] == "report" || args[2] == "boot" ||
				args[2] == "status" || args[2] == "recover" || args[2] == "apply" ||
				args[2] == "apply-status" || args[2] == "apply-recover" {
				return "knowledge lifecycle " + args[2]
			}
			if len(args) > 3 && args[2] == "profile" {
				switch args[3] {
				case "list", "show", "set", "remove":
					return "knowledge lifecycle profile " + args[3]
				}
			}
			if len(args) > 3 && args[2] == "ownership" {
				switch args[3] {
				case "status", "reconcile", "adopt", "release":
					return "knowledge lifecycle ownership " + args[3]
				}
			}
			if len(args) > 3 && args[2] == "host" {
				switch args[3] {
				case "install", "update", "status", "reconcile", "enable", "disable", "uninstall", "run":
					return "knowledge lifecycle host " + args[3]
				}
			}
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
			case "evidence":
				if len(args) > 2 && (args[2] == "local-git" || args[2] == "airgap") {
					return "sclv evidence " + args[2]
				}
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
	case "ssfv":
		if len(args) > 1 {
			switch args[1] {
			case "inspect", "check", "diff", "propose", "graph":
				return "ssfv " + args[1]
			}
		}
	case "maestro":
		if len(args) > 1 {
			switch args[1] {
			case "inspect", "status", "recover":
				return "maestro " + args[1]
			}
		}
	case "validate":
		if len(args) > 1 && (args[1] == "scan" || args[1] == "debug" || args[1] == "root-summary") {
			return "validate " + args[1]
		}
		if len(args) > 2 && (args[1] == "profile" || args[1] == "baseline" || args[1] == "warning") {
			return "validate " + args[1] + " " + args[2]
		}
	}
	return args[0]
}

func validateLegacySubcommand(args []string) error {
	if len(args) < 2 || args[0] != "stav" {
		return nil
	}
	switch args[1] {
	case "status", "verify", "query", "doctor", "append", "enrollment", "supervisor", "--help", "-h":
		return nil
	default:
		return fmt.Errorf("unknown STAV subcommand %q", args[1])
	}
}
