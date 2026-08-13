package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/foundationlifecycle"
	"github.com/spf13/cobra"
)

type foundationLifecycleOptions struct {
	prefix                string
	version               string
	topsID                string
	scope                 string
	operationID           string
	expectedStateDigest   string
	expectedAttemptDigest string
	planPath              string
	desiredState          string
	topsName              string
	serviceUID            string
	serviceGID            string
	authorityUID          string
	authorityGID          string
	auditMode             string
	ttl                   time.Duration
	discover              bool
	jsonOutput            bool
}

func newFoundationLifecycleFamily(component, surface, wrapperFeature string) *cobra.Command {
	family := structural(surface, fmt.Errorf("%s %s subcommand is required: status, plan, apply, apply-status, or recover", component, surface))
	for _, leaf := range []string{"status", "plan", "apply", "apply-status", "recover"} {
		options := foundationLifecycleOptions{scope: "user", auditMode: "ordinary", ttl: 5 * time.Minute}
		operation := map[string]string{"status": "observe", "apply-status": "apply_status"}[leaf]
		if operation == "" {
			operation = leaf
		}
		child := &cobra.Command{
			Use:  leaf,
			Args: usageOnlyArgs,
			RunE: func(*cobra.Command, []string) error {
				return runFoundationLifecycle(component, surface, operation, options)
			},
		}
		registeredFoundationLifecycle(child, component, surface, leaf, wrapperFeature)
		addFoundationLifecycleCommonFlags(child, &options)
		switch leaf {
		case "plan":
			child.Flags().StringVar(&options.operationID, "operation-id", "", "stable idempotency identity for the planned mutation")
			child.Flags().StringVar(&options.expectedStateDigest, "expected-state-digest", "", "exact observed stable state: absent or tagged SHA-256 digest")
			child.Flags().StringVar(&options.desiredState, "desired-state", "", desiredStateHelp(surface))
			child.Flags().StringVar(&options.auditMode, "audit-mode", "ordinary", "audit posture: ordinary or audit_deferred")
			child.Flags().DurationVar(&options.ttl, "ttl", 5*time.Minute, "plan validity, greater than zero and at most 10m")
			if component == "ssiag" && surface == "enrollment" {
				child.Flags().StringVar(&options.topsName, "tops-name", "", "mutable TOPS display name, required when desired state is enrolled")
				child.Flags().StringVar(&options.serviceUID, "service-uid", "", "exact system-scope SSIAG service UID")
				child.Flags().StringVar(&options.serviceGID, "service-gid", "", "exact system-scope SSIAG service GID")
			}
			if component == "stav" && surface == "enrollment" {
				child.Flags().StringVar(&options.authorityUID, "authority-uid", "", "exact system-scope STAV authority UID")
				child.Flags().StringVar(&options.authorityGID, "authority-gid", "", "exact system-scope STAV authority GID")
			}
		case "apply":
			child.Flags().StringVar(&options.planPath, "plan", "", "bounded no-follow foundation lifecycle plan JSON")
			child.Flags().StringVar(&options.expectedAttemptDigest, "expected-attempt-digest", "", "prior protected attempt state: absent or exact tagged SHA-256 digest")
		case "apply-status":
			child.Flags().StringVar(&options.operationID, "operation-id", "", "optional exact operation identity to query")
		case "recover":
			child.Flags().StringVar(&options.operationID, "operation-id", "", "stable identity of the mutation to recover")
			child.Flags().StringVar(&options.expectedAttemptDigest, "expected-attempt-digest", "", "exact protected attempt digest to recover")
			child.Flags().BoolVar(&options.discover, "discover", false, "recover only from one uniquely validated local attempt")
		}
		child.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		family.AddCommand(child)
	}
	family.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	return family
}

func addFoundationLifecycleCommonFlags(command *cobra.Command, options *foundationLifecycleOptions) {
	command.Flags().StringVar(&options.prefix, "prefix", "", "exact receipt-v2 installation prefix")
	command.Flags().StringVar(&options.version, "version", "", "exact installed adapter version; omitted only when one version matches")
	command.Flags().StringVar(&options.topsID, "tops-id", "", "immutable TOPS UUID")
	command.Flags().StringVar(&options.scope, "scope", "user", "local installation scope: user or system")
	command.Flags().BoolVar(&options.jsonOutput, "json", false, "emit validated foundational lifecycle result JSON")
}

func runFoundationLifecycle(component, surface, operation string, options foundationLifecycleOptions) error {
	if options.prefix == "" || options.topsID == "" {
		return fmt.Errorf("--prefix and --tops-id are required")
	}
	request := foundationlifecycle.Options{
		Component: component, Surface: surface, Operation: operation,
		Prefix: options.prefix, Version: options.version,
		Scope: options.scope, TOPSID: options.topsID, Discover: options.discover,
	}
	if options.operationID != "" {
		request.OperationID = stringPointer(options.operationID)
	}
	if options.expectedStateDigest != "" {
		request.ExpectedStateDigest = stringPointer(options.expectedStateDigest)
	}
	if options.expectedAttemptDigest != "" {
		request.ExpectedAttempt = stringPointer(options.expectedAttemptDigest)
	}
	if operation == "plan" {
		if options.operationID == "" || options.expectedStateDigest == "" || options.desiredState == "" {
			return fmt.Errorf("--operation-id, --expected-state-digest, and --desired-state are required")
		}
		if options.ttl <= 0 || options.ttl > 10*time.Minute || options.ttl%time.Second != 0 {
			return fmt.Errorf("--ttl must be a whole-second duration greater than zero and at most 10m")
		}
		intent, err := foundationLifecycleIntent(component, surface, options)
		if err != nil {
			return err
		}
		request.Intent = &intent
	}
	if operation == "apply" {
		if options.planPath == "" || options.expectedAttemptDigest == "" {
			return fmt.Errorf("--plan and --expected-attempt-digest are required")
		}
		plan, err := foundationlifecycle.ReadPlan(options.planPath)
		if err != nil {
			return err
		}
		if plan.Component != component || plan.Surface != surface || plan.Scope != options.scope || plan.TOPSID != options.topsID {
			return fmt.Errorf("foundation lifecycle plan does not match the selected command target")
		}
		request.Plan = &plan
		request.OperationID = stringPointer(plan.OperationID)
		request.ExpectedStateDigest = stringPointer(plan.ExpectedStateDigest)
	}
	if operation == "recover" {
		if options.operationID == "" {
			return fmt.Errorf("--operation-id is required")
		}
		if options.discover == (options.expectedAttemptDigest != "") {
			return fmt.Errorf("exactly one of --expected-attempt-digest or --discover is required")
		}
	}
	result, err := foundationlifecycle.Invoke(context.Background(), request)
	if options.jsonOutput {
		if result.Protocol != "" {
			if printErr := printIndentedJSON(result); printErr != nil {
				return printErr
			}
		}
		return err
	}
	if result.Protocol == "" {
		return err
	}
	state := result.Observation.Enrollment.State
	if surface == "supervisor" {
		state = fmt.Sprintf("manager:%s,descriptor:%s,process:%s,endpoint:%s",
			result.Observation.Supervisor.ManagerState,
			result.Observation.Supervisor.DescriptorState,
			result.Observation.Supervisor.ProcessState,
			result.Observation.Supervisor.EndpointState,
		)
	}
	fmt.Printf(
		"%s %s lifecycle: operation=%s disposition=%s state=%s changed=%t replayed=%t recovered=%t recovery_required=%t reconciliation_required=%t audit_state=%s state_digest=%s canonical=false\n",
		component, surface, operation, result.Disposition, state,
		result.Changed, result.Replayed, result.Recovered, result.RecoveryRequired,
		result.ReconciliationRequired, result.AuditState, result.Observation.StableStateDigest,
	)
	return err
}

func foundationLifecycleIntent(component, surface string, options foundationLifecycleOptions) (foundationlifecycle.Intent, error) {
	intent := foundationlifecycle.Intent{
		DesiredState: options.desiredState, AuditMode: options.auditMode,
		TTLSeconds: uint64(options.ttl / time.Second),
	}
	if options.auditMode != "ordinary" && options.auditMode != "audit_deferred" {
		return intent, fmt.Errorf("--audit-mode must be ordinary or audit_deferred")
	}
	if surface == "supervisor" {
		if options.desiredState != "native_running" && options.desiredState != "native_installed_stopped" && options.desiredState != "absent_stopped" {
			return intent, fmt.Errorf("supervisor --desired-state must be native_running, native_installed_stopped, or absent_stopped")
		}
		return intent, nil
	}
	if options.desiredState != "enrolled" && options.desiredState != "unenrolled_preserved" {
		return intent, fmt.Errorf("enrollment --desired-state must be enrolled or unenrolled_preserved")
	}
	if component == "ssiag" {
		if options.desiredState == "enrolled" && options.topsName == "" {
			return intent, fmt.Errorf("--tops-name is required for SSIAG enrollment")
		}
		if options.topsName != "" {
			if options.desiredState != "enrolled" {
				return intent, fmt.Errorf("--tops-name is valid only when desired state is enrolled")
			}
			intent.TOPSName = stringPointer(options.topsName)
		}
		uid, gid, err := pairedUint32(options.serviceUID, options.serviceGID, "service")
		if err != nil {
			return intent, err
		}
		intent.ServiceUID, intent.ServiceGID = uid, gid
		if options.scope == "user" && uid != nil {
			return intent, fmt.Errorf("user-scope SSIAG enrollment derives service identity and rejects UID/GID overrides")
		}
		if options.scope == "system" && options.desiredState == "enrolled" && uid == nil {
			return intent, fmt.Errorf("system-scope SSIAG enrollment requires --service-uid and --service-gid")
		}
	} else {
		uid, gid, err := pairedUint32(options.authorityUID, options.authorityGID, "authority")
		if err != nil {
			return intent, err
		}
		intent.AuthorityUID, intent.AuthorityGID = uid, gid
		if options.scope == "user" && uid != nil {
			return intent, fmt.Errorf("user-scope STAV enrollment derives authority identity and rejects UID/GID overrides")
		}
		if options.scope == "system" && options.desiredState == "enrolled" && uid == nil {
			return intent, fmt.Errorf("system-scope STAV enrollment requires --authority-uid and --authority-gid")
		}
	}
	return intent, nil
}

func pairedUint32(uidValue, gidValue, label string) (*uint32, *uint32, error) {
	if (uidValue == "") != (gidValue == "") {
		return nil, nil, fmt.Errorf("--%s-uid and --%s-gid must be supplied together", label, label)
	}
	if uidValue == "" {
		return nil, nil, nil
	}
	uid64, err := strconv.ParseUint(uidValue, 10, 32)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid --%s-uid", label)
	}
	gid64, err := strconv.ParseUint(gidValue, 10, 32)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid --%s-gid", label)
	}
	uid, gid := uint32(uid64), uint32(gid64)
	return &uid, &gid, nil
}

func desiredStateHelp(surface string) string {
	if surface == "enrollment" {
		return "desired state: enrolled or unenrolled_preserved"
	}
	return "desired state: native_running, native_installed_stopped, or absent_stopped"
}
