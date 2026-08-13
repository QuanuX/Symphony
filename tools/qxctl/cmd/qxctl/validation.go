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
	warningStateID         string
	subjectID              string
	expectedPolicyDigest   string
	expectedBaselineDigest string
	expectedStateDigest    string
	defaultDisposition     string
	historicalPresentation string
	newPresentation        string
	rules                  []string
	rationale              string
	validUntil             string
	supersededBySubjectID  string
	classification         string
	filter                 validation.DisplayFilter
	jsonOutput             bool
}

type validationOutcomeError struct{ outcome string }

func (failure *validationOutcomeError) Error() string {
	return "validation evaluation outcome is " + failure.outcome
}

func newValidateCommand() *cobra.Command {
	command := structural("validate", fmt.Errorf("validate subcommand is required: scan, debug, root-summary, profile, baseline, or warning"))
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
			child.Flags().StringVar(&options.filter.SubjectID, "subject", "", "display only one exact stable warning subject ID")
			child.Flags().StringVar(&options.filter.Classification, "classification", "", "display only open, accepted, resolved, superseded, or muted warning evidence")
		}
		child.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		command.AddCommand(child)
	}
	rootSummaryOptions := validationOptions{version: "0.1.0-dev"}
	rootSummary := &cobra.Command{
		Use:  "root-summary",
		Args: usageOnlyArgs,
		RunE: func(*cobra.Command, []string) error { return runValidationRootSummary(rootSummaryOptions) },
	}
	registeredRootSummary(rootSummary)
	rootSummary.Flags().StringVar(&rootSummaryOptions.prefix, "prefix", "", "exact Symphony Validator installation prefix")
	rootSummary.Flags().StringVar(&rootSummaryOptions.version, "version", "0.1.0-dev", "exact installed validator version")
	rootSummary.Flags().StringVar(&rootSummaryOptions.repository, "repo", "", "Symphony repository path; defaults to the current repository")
	rootSummary.Flags().BoolVar(&rootSummaryOptions.jsonOutput, "json", false, "emit exact validated root-summary JSON")
	rootSummary.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	command.AddCommand(rootSummary)

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

	warning := structural("warning", fmt.Errorf("validate warning subcommand is required: status, list, show, sync, accept, reopen, supersede, mute, or unmute"))
	for _, operation := range []string{"status", "list", "show", "sync", "accept", "reopen", "supersede", "mute", "unmute"} {
		options := validationOptions{warningStateID: "default", version: "0.1.0-dev"}
		child := &cobra.Command{
			Use:  operation,
			Args: usageOnlyArgs,
			RunE: func(*cobra.Command, []string) error { return runValidationWarning(operation, options) },
		}
		if operation == "status" || operation == "list" || operation == "show" {
			interaction := map[string]string{"status": "query", "list": "discover", "show": "inspect"}[operation]
			registeredValidationWarning(child, "validate.warning."+operation, interaction, operation == "show", false, "")
		} else {
			recovery := map[string]string{
				"sync": "validate.warning.sync", "accept": "validate.warning.reopen",
				"reopen": "validate.warning.reopen", "supersede": "validate.warning.reopen",
				"mute": "validate.warning.unmute", "unmute": "validate.warning.mute",
			}[operation]
			registeredValidationWarning(child, "validate.warning."+operation, "configure", true, true, recovery)
		}
		addValidationStateFlags(child, &options)
		child.Flags().StringVar(&options.warningStateID, "warning-state-id", "default", "exact protected warning lifecycle state identity")
		if operation == "list" {
			child.Flags().StringVar(&options.classification, "classification", "", "optional open, accepted, resolved, superseded, or muted filter")
		}
		if operation == "show" || operation == "accept" || operation == "reopen" || operation == "supersede" || operation == "mute" || operation == "unmute" {
			child.Flags().StringVar(&options.subjectID, "subject-id", "", "exact stable warning subject digest")
		}
		if operation == "sync" || operation == "accept" || operation == "reopen" || operation == "supersede" || operation == "mute" || operation == "unmute" {
			child.Flags().StringVar(&options.expectedStateDigest, "expected-state-digest", "", "required prior warning state: absent or exact tagged SHA-256 digest")
		}
		if operation == "sync" {
			child.Flags().StringVar(&options.prefix, "prefix", "", "exact Symphony Validator installation prefix")
			child.Flags().StringVar(&options.version, "version", "0.1.0-dev", "exact installed validator version")
			child.Flags().StringVar(&options.repository, "repo", "", "Symphony repository path; defaults to the current repository")
		}
		if operation == "accept" || operation == "reopen" || operation == "supersede" || operation == "mute" || operation == "unmute" {
			child.Flags().StringVar(&options.rationale, "rationale", "", "required bounded administrative rationale")
		}
		if operation == "accept" {
			child.Flags().StringVar(&options.validUntil, "valid-until", "", "optional future STSC whole-second UTC acceptance expiry")
		}
		if operation == "supersede" {
			child.Flags().StringVar(&options.supersededBySubjectID, "superseded-by-subject-id", "", "distinct known replacement warning subject digest")
		}
		child.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		warning.AddCommand(child)
	}
	warning.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	command.AddCommand(warning)
	command.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	return command
}

func runValidationRootSummary(options validationOptions) error {
	if options.prefix == "" {
		return fmt.Errorf("--prefix is required")
	}
	summary, err := validation.RunRootSummary(context.Background(), options.prefix, options.version, options.repository)
	if err != nil {
		return err
	}
	if options.jsonOutput {
		return printIndentedJSON(summary)
	}
	fmt.Printf("Repository root summary: catalog=%s features=%d nested=%d owner_scopes=%d commands=%d expectations=%d publications=%d digest=%s\n",
		summary.SSFV.CatalogState, summary.SSFV.RegisteredFeatures, summary.SSFV.NestedFeatures,
		summary.SSFV.RegisteredOwnerScopes, summary.QXCTL.RegisteredCommands,
		summary.FeatureAdministration.Expectations, len(summary.PublishedSourceVersions), summary.SummaryDigest)
	return nil
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
	command.Flags().StringVar(&options.warningStateID, "warning-state-id", "default", "protected warning lifecycle state identity; an absent state treats current warnings as open")
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
	var warningState *validation.WarningState
	if options.warningStateID != "" {
		snapshot, err := store.WarningState(options.warningStateID)
		if err != nil {
			return err
		}
		if snapshot.Exists {
			warningState = &snapshot.State
		}
	}
	filter := validation.DisplayFilter{}
	if operation == "debug" {
		filter = options.filter
	}
	projection, err := validation.EvaluateWithWarningState(raw, policy, baseline, warningState, filter, time.Now(), operation == "debug")
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
		classification := projection.WarningClassifications[finding.OccurrenceID]
		fmt.Printf("Validation evidence: category=%s classification=%s scope=%s rule=%s subject=%s occurrence=%s detail=%q\n",
			finding.Category, classification, finding.Scope, finding.RuleID, finding.SubjectID, finding.OccurrenceID, finding.Detail)
	}
	for _, occurrence := range projection.Historical {
		finding := occurrence.Finding
		fmt.Printf("Validation historical warning: classification=resolved rule=%s subject=%s occurrence=%s last_observed_at=%s evidence_digests=%s detail=%q\n",
			finding.RuleID, finding.SubjectID, finding.OccurrenceID, occurrence.LastObservedAt,
			strings.Join(occurrence.EvidenceDigests, ","), finding.Detail)
	}
	if len(evaluation.ResolvedWarningOccurrenceIDs) != 0 {
		fmt.Printf("Validation resolved warnings: %s\n", strings.Join(evaluation.ResolvedWarningOccurrenceIDs, ","))
	}
}

func runValidationWarning(operation string, options validationOptions) error {
	if options.topsID == "" {
		return fmt.Errorf("--tops-id is required")
	}
	store, err := validation.NewStore(options.stateRoot, options.topsID)
	if err != nil {
		return err
	}
	if operation == "status" {
		states, err := store.ListWarningStates()
		if err != nil {
			return err
		}
		if options.jsonOutput {
			return printIndentedJSON(states)
		}
		fmt.Printf("Validation warning states: tops_id=%s count=%d canonical=false\n", states.TOPSID, len(states.States))
		for _, state := range states.States {
			fmt.Printf("Validation warning state: id=%s generation=%d open=%d accepted=%d resolved=%d superseded=%d muted=%d digest=%s\n",
				state.StateID, state.Generation, state.Open, state.Accepted, state.Resolved, state.Superseded, state.Muted, state.StateDigest)
		}
		return nil
	}
	if operation == "sync" {
		if options.prefix == "" || options.expectedStateDigest == "" {
			return fmt.Errorf("--prefix and --expected-state-digest are required")
		}
		raw, err := validation.Run(context.Background(), options.prefix, options.version, options.repository)
		if err != nil {
			return err
		}
		state, changed, err := store.SyncWarningState(options.warningStateID, options.expectedStateDigest, raw, time.Now())
		if err != nil {
			return err
		}
		if options.jsonOutput {
			if err := printIndentedJSON(state); err != nil {
				return err
			}
		} else {
			fmt.Printf("Validation warning sync: id=%s changed=%t generation=%d new_subjects=%d known_subject_new_occurrences=%d resolved=%d digest=%s canonical=false\n",
				state.StateID, changed, state.Generation, len(state.LastSync.NewSubjectIDs),
				len(state.LastSync.KnownSubjectNewOccurrenceIDs), len(state.LastSync.ResolvedSubjectIDs), state.StateDigest)
		}
		if raw.Evidence.Summary.Violation != 0 {
			return &validationOutcomeError{outcome: "failed"}
		}
		return nil
	}
	snapshot, err := store.WarningState(options.warningStateID)
	if err != nil {
		return err
	}
	if !snapshot.Exists {
		return fmt.Errorf("validation warning state %q is absent", options.warningStateID)
	}
	if operation == "list" {
		subjects := make([]validation.WarningSubject, 0, len(snapshot.State.Subjects))
		for _, subject := range snapshot.State.Subjects {
			if options.classification == "muted" && !subject.Muted {
				continue
			}
			if options.classification != "" && options.classification != "muted" && subject.Classification != options.classification {
				continue
			}
			subjects = append(subjects, subject)
		}
		if options.classification != "" && options.classification != "muted" && options.classification != "open" &&
			options.classification != "accepted" && options.classification != "resolved" && options.classification != "superseded" {
			return fmt.Errorf("--classification must be open, accepted, resolved, superseded, or muted")
		}
		if options.jsonOutput {
			return printIndentedJSON(map[string]any{
				"canonical": false, "state_digest": snapshot.State.StateDigest,
				"state_id": snapshot.State.StateID, "subjects": subjects, "tops_id": snapshot.State.TOPSID,
			})
		}
		fmt.Printf("Validation warning subjects: state_id=%s count=%d digest=%s canonical=false\n",
			snapshot.State.StateID, len(subjects), snapshot.State.StateDigest)
		for _, subject := range subjects {
			fmt.Printf("Validation warning subject: classification=%s muted=%t rule=%s subject=%s occurrences=%d last_observed_at=%s\n",
				subject.Classification, subject.Muted, subject.RuleID, subject.SubjectID, len(subject.Occurrences), subject.LastObservedAt)
		}
		return nil
	}
	if operation == "show" {
		if options.subjectID == "" {
			return fmt.Errorf("--subject-id is required")
		}
		index := sort.Search(len(snapshot.State.Subjects), func(i int) bool { return snapshot.State.Subjects[i].SubjectID >= options.subjectID })
		if index == len(snapshot.State.Subjects) || snapshot.State.Subjects[index].SubjectID != options.subjectID {
			return fmt.Errorf("warning subject %q is unknown", options.subjectID)
		}
		subject := snapshot.State.Subjects[index]
		transitions := make([]validation.WarningTransition, 0)
		for _, transition := range snapshot.State.Transitions {
			if transition.SubjectID == subject.SubjectID {
				transitions = append(transitions, transition)
			}
		}
		if options.jsonOutput {
			return printIndentedJSON(snapshot.State)
		}
		fmt.Printf("Validation warning subject: classification=%s muted=%t rule=%s subject=%s occurrences=%d transitions=%d state_digest=%s\n",
			subject.Classification, subject.Muted, subject.RuleID, subject.SubjectID,
			len(subject.Occurrences), len(transitions), snapshot.State.StateDigest)
		for _, occurrence := range subject.Occurrences {
			fmt.Printf("Validation warning occurrence: active=%t occurrence=%s first_observed_at=%s last_observed_at=%s evidence_digests=%s detail=%q\n",
				occurrence.Active, occurrence.OccurrenceID, occurrence.FirstObservedAt, occurrence.LastObservedAt,
				strings.Join(occurrence.EvidenceDigests, ","), occurrence.Finding.Detail)
		}
		return nil
	}
	if options.subjectID == "" || options.expectedStateDigest == "" || options.rationale == "" {
		return fmt.Errorf("--subject-id, --expected-state-digest, and --rationale are required")
	}
	mutation := validation.WarningMutation{
		ExpectedStateDigest: options.expectedStateDigest, Rationale: options.rationale,
		StateID: options.warningStateID, SubjectID: options.subjectID,
	}
	switch operation {
	case "accept":
		mutation.Classification = "accepted"
		mutation.ValidUntil = options.validUntil
	case "reopen":
		mutation.Classification = "open"
	case "supersede":
		mutation.Classification = "superseded"
		mutation.SupersededBySubjectID = options.supersededBySubjectID
	case "mute", "unmute":
		muted := operation == "mute"
		mutation.Muted = &muted
	default:
		return fmt.Errorf("unsupported validation warning operation")
	}
	state, changed, err := store.MutateWarning(mutation, time.Now())
	if err != nil {
		return err
	}
	if options.jsonOutput {
		return printIndentedJSON(state)
	}
	fmt.Printf("Validation warning mutation: operation=%s state_id=%s subject=%s changed=%t generation=%d digest=%s canonical=false\n",
		operation, state.StateID, options.subjectID, changed, state.Generation, state.StateDigest)
	return nil
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
