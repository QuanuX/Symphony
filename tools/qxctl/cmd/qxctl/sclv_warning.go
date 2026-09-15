package main

import (
	"fmt"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/validation"
	"github.com/spf13/cobra"
)

// SCLV warning administration uses the common host-local lifecycle. It does not
// invoke the SCLV engine, change its ledger, or add a detector exception.
func newSCLVWarningCommand() *cobra.Command {
	command := structural("warning", fmt.Errorf("SCLV warning subcommand is required: list, show, acknowledge, or reopen"))
	for _, operation := range []string{"list", "show", "acknowledge", "reopen"} {
		options := validationOptions{warningStateID: "default"}
		child := &cobra.Command{
			Use:   operation,
			Short: "Administer exact historical-reference warnings in local validation state",
			Args:  usageOnlyArgs,
			RunE: func(*cobra.Command, []string) error {
				return runSCLVWarning(operation, options)
			},
		}
		mutation := operation == "acknowledge" || operation == "reopen"
		interaction := map[string]string{"list": "discover", "show": "inspect", "acknowledge": "configure", "reopen": "configure"}[operation]
		recovery := ""
		if mutation {
			recovery = "sclv.warning.reopen"
		}
		registeredValidationWarning(child, "sclv.warning."+operation, interaction, operation != "list", mutation, recovery)
		addValidationStateFlags(child, &options)
		child.Flags().StringVar(&options.warningStateID, "warning-state-id", "default", "exact local state populated by validate warning sync")
		if operation == "list" {
			child.Flags().StringVar(&options.classification, "classification", "", "optional open, accepted, resolved, superseded, or muted filter")
		} else {
			child.Flags().StringVar(&options.subjectID, "subject-id", "", "exact historical-reference warning subject digest from list")
		}
		if mutation {
			child.Flags().StringVar(&options.expectedStateDigest, "expected-state-digest", "", "required exact prior warning-state digest")
			child.Flags().StringVar(&options.rationale, "rationale", "", "required explanation of the intended manual behavior or reopening")
		}
		if operation == "acknowledge" {
			child.Flags().StringVar(&options.validUntil, "valid-until", "", "optional future STSC whole-second UTC acceptance expiry")
		}
		child.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		command.AddCommand(child)
	}
	command.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	return command
}

func runSCLVWarning(operation string, options validationOptions) error {
	switch operation {
	case "acknowledge":
		operation = "accept"
	case "list", "show", "reopen":
	default:
		return fmt.Errorf("unsupported SCLV warning operation")
	}
	return runValidationWarningFiltered(operation, options, isSCLVHistoricalWarning)
}

func isSCLVHistoricalWarning(subject validation.WarningSubject) bool {
	switch subject.RuleID {
	case "sclv_reference.historical_path_absent", "sclv_reference.historical_path_not_file", "sclv_skvi_reference.historical":
	default:
		return false
	}
	if len(subject.Occurrences) == 0 {
		return false
	}
	for _, occurrence := range subject.Occurrences {
		finding := occurrence.Finding
		if finding.Category != "warning" || finding.RuleID != subject.RuleID ||
			finding.Attributes["record_id"] == "" || finding.Attributes["path"] == "" {
			return false
		}
	}
	return true
}
