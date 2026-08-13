package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/validation"
	"github.com/spf13/cobra"
)

type validationOptions struct {
	topsID                 string
	stateRoot              string
	prefix                 string
	version                string
	repository             string
	profileID              string
	baselineID             string
	expectedPolicyDigest   string
	expectedBaselineDigest string
	defaultDisposition     string
	historicalPresentation string
	newPresentation        string
	rules                  []string
	filter                 validation.DisplayFilter
	jsonOutput             bool
}

type validationOutcomeError struct{ outcome string }

func (failure *validationOutcomeError) Error() string {
	return "validation evaluation outcome is " + failure.outcome
}

func newValidateCommand() *cobra.Command {
	command := structural("validate", fmt.Errorf("validate subcommand is required: scan, debug, profile, or baseline"))
	for _, operation := range []string{"scan", "debug"} {
		options := validationOptions{version: "0.1.0-dev"}
		child := &cobra.Command{
			Use:  operation,
			Args: usageOnlyArgs,
			RunE: func(*cobra.Command, []string) error { return runValidationScan(operation, options) },
		}
		registered(child, "validate."+operation, featureValidation, "validate")
		addValidationExecutionFlags(child, &options)
		if operation == "debug" {
			child.Flags().StringVar(&options.filter.RuleID, "rule", "", "display only one exact rule ID after the full scan")
			child.Flags().StringVar(&options.filter.RecordID, "record", "", "display only one exact SCLV record ID after the full scan")
			child.Flags().StringVar(&options.filter.Path, "path", "", "display only one exact repository-relative path after the full scan")
			child.Flags().StringVar(&options.filter.Delta, "delta", "", "display only new, unchanged, or resolved warning evidence")
		}
		child.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		command.AddCommand(child)
	}

	profile := structural("profile", fmt.Errorf("validate profile subcommand is required: list, show, set, or remove"))
	for _, operation := range []string{"list", "show", "set", "remove"} {
		options := validationOptions{
			profileID: "default", defaultDisposition: "record",
			historicalPresentation: "summary", newPresentation: "full",
		}
		child := &cobra.Command{
			Use:  operation,
			Args: usageOnlyArgs,
			RunE: func(*cobra.Command, []string) error { return runValidationProfile(operation, options) },
		}
		if operation == "list" || operation == "show" {
			registered(child, "validate.profile."+operation, featureValidation, map[string]string{"list": "discover", "show": "inspect"}[operation])
		} else {
			registeredMutation(child, "validate.profile."+operation, featureValidation, "configure", "target_host_permission", "")
		}
		addValidationStateFlags(child, &options)
		if operation != "list" {
			child.Flags().StringVar(&options.profileID, "profile-id", "default", "exact validation profile identity")
		}
		if operation == "set" || operation == "remove" {
			child.Flags().StringVar(&options.expectedPolicyDigest, "expected-policy-digest", "", "required prior policy state: absent or exact tagged SHA-256 digest")
		}
		if operation == "set" {
			child.Flags().StringVar(&options.defaultDisposition, "warning", "record", "default warning disposition: record, review, or require")
			child.Flags().StringVar(&options.historicalPresentation, "historical", "summary", "unchanged warning presentation: full, summary, or count")
			child.Flags().StringVar(&options.newPresentation, "new", "full", "new warning presentation: full or summary")
			child.Flags().StringArrayVar(&options.rules, "rule", nil, "exact override RULE=record|review|require,full|summary|count; repeat as needed")
		}
		child.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		profile.AddCommand(child)
	}
	profile.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	command.AddCommand(profile)

	baseline := structural("baseline", fmt.Errorf("validate baseline subcommand is required: create, show, or remove"))
	for _, operation := range []string{"create", "show", "remove"} {
		options := validationOptions{baselineID: "default", version: "0.1.0-dev"}
		child := &cobra.Command{
			Use:  operation,
			Args: usageOnlyArgs,
			RunE: func(*cobra.Command, []string) error { return runValidationBaseline(operation, options) },
		}
		if operation == "show" {
			registered(child, "validate.baseline.show", featureValidation, "inspect")
		} else {
			registeredMutation(child, "validate.baseline."+operation, featureValidation, "configure", "target_host_permission", "")
		}
		addValidationStateFlags(child, &options)
		child.Flags().StringVar(&options.baselineID, "baseline-id", "default", "exact validation baseline identity")
		if operation == "create" {
			child.Flags().StringVar(&options.prefix, "prefix", "", "exact Symphony Validator installation prefix")
			child.Flags().StringVar(&options.version, "version", "0.1.0-dev", "exact installed validator version")
			child.Flags().StringVar(&options.repository, "repo", "", "Symphony repository path; defaults to the current repository")
		}
		if operation == "create" || operation == "remove" {
			child.Flags().StringVar(&options.expectedBaselineDigest, "expected-baseline-digest", "", "required prior baseline state: absent or exact tagged SHA-256 digest")
		}
		child.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		baseline.AddCommand(child)
	}
	baseline.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	command.AddCommand(baseline)
	command.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	return command
}

func addValidationStateFlags(command *cobra.Command, options *validationOptions) {
	command.Flags().StringVar(&options.topsID, "tops-id", "", "immutable TOPS UUID")
	command.Flags().StringVar(&options.stateRoot, "state-root", "", "state root; defaults to XDG_STATE_HOME or ~/.local/state")
	command.Flags().BoolVar(&options.jsonOutput, "json", false, "emit JSON")
}

func addValidationExecutionFlags(command *cobra.Command, options *validationOptions) {
	addValidationStateFlags(command, options)
	command.Flags().StringVar(&options.prefix, "prefix", "", "exact Symphony Validator installation prefix")
	command.Flags().StringVar(&options.version, "version", "0.1.0-dev", "exact installed validator version")
	command.Flags().StringVar(&options.repository, "repo", "", "Symphony repository path; defaults to the current repository")
	command.Flags().StringVar(&options.profileID, "profile-id", "", "optional protected validation profile identity")
	command.Flags().StringVar(&options.baselineID, "baseline-id", "", "optional protected validation baseline identity")
}

func runValidationScan(operation string, options validationOptions) error {
	if options.topsID == "" || options.prefix == "" {
		return fmt.Errorf("--tops-id and --prefix are required")
	}
	store, err := validation.NewStore(options.stateRoot, options.topsID)
	if err != nil {
		return err
	}
	raw, err := validation.Run(context.Background(), options.prefix, options.version, options.repository)
	if err != nil {
		return err
	}
	var policy *validation.Policy
	if options.profileID != "" {
		snapshot, err := store.Policy(options.profileID)
		if err != nil {
			return err
		}
		if !snapshot.Exists {
			return fmt.Errorf("validation profile %q is absent", options.profileID)
		}
		policy = &snapshot.Policy
	}
	var baseline *validation.Baseline
	if options.baselineID != "" {
		snapshot, err := store.Baseline(options.baselineID)
		if err != nil {
			return err
		}
		if !snapshot.Exists {
			return fmt.Errorf("validation baseline %q is absent", options.baselineID)
		}
		baseline = &snapshot.Baseline
	}
	filter := validation.DisplayFilter{}
	if operation == "debug" {
		filter = options.filter
	}
	projection, err := validation.Evaluate(raw, policy, baseline, filter)
	if err != nil {
		return err
	}
	if options.jsonOutput {
		if err := printIndentedJSON(projection.Result); err != nil {
			return err
		}
	} else {
		printValidationProjection(projection)
	}
	if projection.Result.Evaluation.Outcome != "pass" {
		return &validationOutcomeError{outcome: projection.Result.Evaluation.Outcome}
	}
	return nil
}

func printValidationProjection(projection validation.Projection) {
	result := projection.Result
	evaluation := result.Evaluation
	fmt.Printf("Validation: outcome=%s validator=%s version=%s evidence=%s result=%s\n",
		evaluation.Outcome, result.Evidence.ValidatorID, result.Evidence.ValidatorVersion,
		result.Evidence.EvidenceDigest, result.ResultDigest)
	fmt.Printf("Validation findings: pass=%d warning=%d violation=%d other=%d new=%d unchanged=%d resolved=%d displayed=%d\n",
		result.Evidence.Summary.Pass, result.Evidence.Summary.Warning,
		result.Evidence.Summary.Violation, result.Evidence.Summary.Other,
		len(evaluation.NewWarningOccurrenceIDs), len(evaluation.UnchangedWarningOccurrenceIDs),
		len(evaluation.ResolvedWarningOccurrenceIDs), len(projection.Displayed))
	for _, finding := range projection.Displayed {
		fmt.Printf("Validation evidence: category=%s scope=%s rule=%s occurrence=%s detail=%q\n",
			finding.Category, finding.Scope, finding.RuleID, finding.OccurrenceID, finding.Detail)
	}
	if len(evaluation.ResolvedWarningOccurrenceIDs) != 0 {
		fmt.Printf("Validation resolved warnings: %s\n", strings.Join(evaluation.ResolvedWarningOccurrenceIDs, ","))
	}
}

func runValidationProfile(operation string, options validationOptions) error {
	if options.topsID == "" {
		return fmt.Errorf("--tops-id is required")
	}
	store, err := validation.NewStore(options.stateRoot, options.topsID)
	if err != nil {
		return err
	}
	switch operation {
	case "list":
		result, err := store.ListPolicies()
		if err != nil {
			return err
		}
		if options.jsonOutput {
			return printIndentedJSON(result)
		}
		fmt.Printf("Validation profiles: tops_id=%s count=%d canonical=false\n", result.TOPSID, len(result.Policies))
		for _, policy := range result.Policies {
			fmt.Printf("Validation profile: id=%s generation=%d warning=%s rules=%d digest=%s\n",
				policy.ProfileID, policy.Generation, policy.DefaultWarningDisposition, policy.RuleCount, policy.PolicyDigest)
		}
		return nil
	case "show":
		snapshot, err := store.Policy(options.profileID)
		if err != nil {
			return err
		}
		if !snapshot.Exists {
			return fmt.Errorf("validation profile %q is absent", options.profileID)
		}
		if options.jsonOutput {
			return printIndentedJSON(snapshot.Policy)
		}
		fmt.Printf("Validation profile: id=%s generation=%d warning=%s historical=%s new=%s rules=%d digest=%s canonical=false\n",
			snapshot.Policy.ProfileID, snapshot.Policy.Generation,
			snapshot.Policy.DefaultWarningDisposition, snapshot.Policy.HistoricalPresentation,
			snapshot.Policy.NewPresentation, len(snapshot.Policy.Rules), snapshot.Policy.PolicyDigest)
		return nil
	case "set":
		if options.expectedPolicyDigest == "" {
			return fmt.Errorf("--expected-policy-digest is required")
		}
		rules, err := parseValidationRules(options.rules)
		if err != nil {
			return err
		}
		policy, changed, err := store.SetPolicy(validation.PolicyConfig{
			ProfileID: options.profileID, DefaultWarningDisposition: options.defaultDisposition,
			HistoricalPresentation: options.historicalPresentation,
			NewPresentation:        options.newPresentation, Rules: rules,
		}, options.expectedPolicyDigest, time.Now())
		if err != nil {
			return err
		}
		if options.jsonOutput {
			return printIndentedJSON(map[string]any{"changed": changed, "policy": policy})
		}
		fmt.Printf("Validation profile: operation=set id=%s changed=%t generation=%d digest=%s canonical=false\n",
			policy.ProfileID, changed, policy.Generation, policy.PolicyDigest)
		return nil
	case "remove":
		if options.expectedPolicyDigest == "" {
			return fmt.Errorf("--expected-policy-digest is required")
		}
		changed, err := store.RemovePolicy(options.profileID, options.expectedPolicyDigest)
		if err != nil {
			return err
		}
		if options.jsonOutput {
			return printIndentedJSON(map[string]any{"canonical": false, "changed": changed, "profile_id": options.profileID})
		}
		fmt.Printf("Validation profile: operation=remove id=%s changed=%t canonical=false\n", options.profileID, changed)
		return nil
	default:
		return fmt.Errorf("unsupported validation profile operation")
	}
}

func runValidationBaseline(operation string, options validationOptions) error {
	if options.topsID == "" {
		return fmt.Errorf("--tops-id is required")
	}
	store, err := validation.NewStore(options.stateRoot, options.topsID)
	if err != nil {
		return err
	}
	switch operation {
	case "create":
		if options.prefix == "" || options.expectedBaselineDigest == "" {
			return fmt.Errorf("--prefix and --expected-baseline-digest are required")
		}
		raw, err := validation.Run(context.Background(), options.prefix, options.version, options.repository)
		if err != nil {
			return err
		}
		baseline, changed, err := store.CreateBaseline(options.baselineID, options.expectedBaselineDigest, raw, time.Now())
		if err != nil {
			return err
		}
		if options.jsonOutput {
			return printIndentedJSON(map[string]any{"baseline": baseline, "changed": changed})
		}
		fmt.Printf("Validation baseline: operation=create id=%s changed=%t warnings=%d digest=%s canonical=false\n",
			baseline.BaselineID, changed, len(baseline.WarningOccurrenceIDs), baseline.BaselineDigest)
		return nil
	case "show":
		snapshot, err := store.Baseline(options.baselineID)
		if err != nil {
			return err
		}
		if !snapshot.Exists {
			return fmt.Errorf("validation baseline %q is absent", options.baselineID)
		}
		if options.jsonOutput {
			return printIndentedJSON(snapshot.Baseline)
		}
		fmt.Printf("Validation baseline: id=%s validator=%s version=%s warnings=%d created_at=%s digest=%s canonical=false\n",
			snapshot.Baseline.BaselineID, snapshot.Baseline.ValidatorID, snapshot.Baseline.ValidatorVersion,
			len(snapshot.Baseline.WarningOccurrenceIDs), snapshot.Baseline.CreatedAt, snapshot.Baseline.BaselineDigest)
		return nil
	case "remove":
		if options.expectedBaselineDigest == "" {
			return fmt.Errorf("--expected-baseline-digest is required")
		}
		changed, err := store.RemoveBaseline(options.baselineID, options.expectedBaselineDigest)
		if err != nil {
			return err
		}
		if options.jsonOutput {
			return printIndentedJSON(map[string]any{"baseline_id": options.baselineID, "canonical": false, "changed": changed})
		}
		fmt.Printf("Validation baseline: operation=remove id=%s changed=%t canonical=false\n", options.baselineID, changed)
		return nil
	default:
		return fmt.Errorf("unsupported validation baseline operation")
	}
}

func parseValidationRules(values []string) ([]validation.RulePolicy, error) {
	rules := make([]validation.RulePolicy, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		identity, settings, ok := strings.Cut(value, "=")
		if !ok {
			return nil, fmt.Errorf("validation rule override must use RULE=DISPOSITION,PRESENTATION")
		}
		disposition, presentation, ok := strings.Cut(settings, ",")
		if !ok || identity == "" || disposition == "" || presentation == "" || seen[identity] {
			return nil, fmt.Errorf("validation rule override is malformed or duplicated")
		}
		seen[identity] = true
		rules = append(rules, validation.RulePolicy{RuleID: identity, Disposition: disposition, Presentation: presentation})
	}
	sort.Slice(rules, func(i, j int) bool { return rules[i].RuleID < rules[j].RuleID })
	return rules, nil
}
