package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/validation"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var errUsageOnly = errors.New("print qxctl usage")

const cliErrorProtocol = "symphony.qxctl.error.v1"

type cliErrorEnvelope struct {
	Protocol  string         `json:"protocol"`
	Outcome   string         `json:"outcome"`
	CommandID *string        `json:"command_id"`
	Error     cliErrorDetail `json:"error"`
	ExitCode  int            `json:"exit_code"`
}

type cliErrorDetail struct {
	Code       string  `json:"code"`
	Message    string  `json:"message"`
	EngineCode *string `json:"engine_code"`
}

// Already-emitted validated evidence owns its output and specialized exit
// status. A root diagnostic must never turn that evidence into a second JSON
// document or replace a nonzero validated status with a generic status.
func finishCommandError(root, command *cobra.Command, args []string, err error) int {
	if errors.Is(err, errUsageOnly) {
		if scvJSONRequested(root, args) {
			writeCLIError(command, err, 1)
		} else {
			printCommandHelp(root)
		}
		return 1
	}
	var validationFailure *validationOutcomeError
	if errors.As(err, &validationFailure) {
		return 1
	}
	var exactEvidenceExit *exactEvidenceExitError
	if errors.As(err, &exactEvidenceExit) {
		return boundedCLIExit(exactEvidenceExit.code)
	}
	status := 1
	var validatorExit *validation.ValidatorExitError
	if errors.As(err, &validatorExit) {
		status = boundedCLIExit(validatorExit.ExitCode)
	}
	var engineError *knowledgeengine.ProcessError
	if spec, specErr := commandregistry.Spec(command); specErr == nil &&
		strings.HasPrefix(spec.CommandID, "qxcmd:symphony:snv.") && errors.As(err, &engineError) && engineError != nil {
		status = snvEngineErrorExit(engineError.Code)
	}
	if scvJSONRequested(root, args) {
		writeCLIError(command, err, status)
	} else {
		fmt.Printf("%s failed: %v\n", failurePrefix(args), err)
	}
	return status
}

func boundedCLIExit(status int) int {
	if status > 0 && status <= 125 {
		return status
	}
	return 1
}

func writeCLIError(command *cobra.Command, err error, status int) {
	detail := cliErrorDetail{Code: "command_failed", Message: "The command could not complete; no successful result is available."}
	var commandID *string
	if spec, specErr := commandregistry.Spec(command); specErr == nil {
		commandID = &spec.CommandID
	} else if command != nil {
		detail = cliErrorDetail{Code: "invalid_arguments", Message: "Select a command and supply valid arguments; use its --help for accepted flags."}
	}
	if errors.Is(err, errUsageOnly) {
		detail = cliErrorDetail{Code: "invalid_arguments", Message: "Select a command and supply valid arguments; use its --help for accepted flags."}
	}
	var engineError *knowledgeengine.ProcessError
	if errors.As(err, &engineError) && engineError != nil {
		detail = cliErrorDetail{Code: "engine_rejected", Message: "The selected engine rejected the request.", EngineCode: safeSCVEngineCode(engineError.Code)}
		if detail.EngineCode != nil && strings.HasPrefix(engineError.Code, "snv.") {
			// Admitted SNV codes include local authority/storage boundaries;
			// they do not all describe a rejection by the native reducer.
			detail.Code = "operation_refused"
			detail.Message = "The SNV workflow was refused; inspect its exact state or retained attempt."
		}
	}
	// Only fixed messages, registered command identity, and allowlisted engine
	// codes cross this boundary. In particular, never serialize err.Error().
	_ = json.NewEncoder(os.Stdout).Encode(cliErrorEnvelope{
		Protocol: cliErrorProtocol, Outcome: "error", CommandID: commandID,
		Error: detail, ExitCode: status,
	})
}

// Engine code syntax alone is insufficient: an otherwise valid identifier
// could contain caller data. Unknown/new codes remain null until admitted here.
func safeSCVEngineCode(code string) *string {
	switch code {
	case "sbv.contract", "sbv.pointer", "sbv.failure", "sbv.outcome_uncertain", "sqav.request.invalid", "sqav.request.rejected", "sqav.request.limit", "sqav.request.unavailable", "request.invalid_deadline", "request.deadline_expired", "argument.count", "argument.unsupported", "internal.failure", "operation.unsupported",
		"request.deadline", "request.deadline_exceeded", "corpus.invalid", "interpretation.invalid", "coverage.invalid", "pack.invalid", "composition.invalid",
		"knowledge.invalid", "knowledge.evaluation_limit",
		"scv.fields", "scv.type", "scv.bounds", "scv.identity", "scv.locator", "scv.digest",
		"scv.domain", "scv.duplicate", "scv.identity_change", "scv.generation", "scv.protocol",
		"scv.stale_state", "scv.plan_mismatch", "scv.time", "scv.capture_size", "scv.capture_encoding",
		"scv.capture_digest", "scv.completeness":
		return &code
	case "invalid_input", "invalid_payload", "unsupported_protocol", "capacity_exceeded",
		"reference_mismatch", "reference_conflict", "conflict", "unsupported_operation", "deadline_exceeded",
		"scnv.invalid_input", "scnv.capacity_exceeded", "scnv.lineage_conflict",
		"scnv.expected_evidence_conflict", "scnv.snapshot_required",
		"snv.invalid_input", "snv.deadline_exceeded", "snv.unsupported_owner", "snv.unsupported_mode",
		"snv.unsupported_operation", "snv.snapshot_conflict", "snv.state_conflict", "snv.intent_conflict",
		"snv.capacity_exceeded", "snv.authority_denied", "snv.authority_unavailable", "snv.authority_expired",
		"snv.recovery_required", "snv.state_busy", "snv.unsafe_state", "snv.authority_conflict", "snv.installation_drift", "snv.candidate_drift", "snv.evidence_missing", "invocation.arguments", "descriptor.input", "deadline.exceeded":
		return &code
	}
	return nil
}

// These are reviewed domain categories, not a copy of an untrusted process
// status. Preserve existing owners' CLI behavior; this mapping belongs to SNV.
func snvEngineErrorExit(code string) int {
	switch code {
	case "reference_mismatch", "reference_conflict", "conflict", "scnv.lineage_conflict",
		"scnv.expected_evidence_conflict", "snv.snapshot_conflict", "snv.state_conflict", "snv.intent_conflict", "snv.authority_conflict", "snv.installation_drift", "snv.candidate_drift":
		return 4
	case "deadline_exceeded", "snv.deadline_exceeded", "request.deadline_expired", "deadline.exceeded", "snv.authority_unavailable", "snv.state_busy":
		return 3
	case "invalid_input", "invalid_payload", "unsupported_protocol", "capacity_exceeded", "unsupported_operation",
		"scnv.invalid_input", "scnv.capacity_exceeded", "scnv.snapshot_required",
		"snv.invalid_input", "snv.unsupported_owner", "snv.unsupported_mode", "snv.unsupported_operation",
		"operation.unsupported", "invocation.arguments", "descriptor.input", "snv.capacity_exceeded", "snv.evidence_missing", "snv.unsafe_state":
		return 2
	case "snv.authority_denied", "snv.authority_expired":
		return 5
	case "snv.recovery_required":
		return 6
	default:
		return 1
	}
}

// Parsing can fail before Cobra reaches --json. Inspect output intent without
// executing handlers or reparsing their flag values. The actual SCV tree tells
// us which flags consume the next token, including a value spelled "--json".
// This does not make misplaced/unknown flags valid or change Cobra's grammar.
func scvJSONRequested(root *cobra.Command, args []string) bool {
	if len(args) == 0 || (args[0] != "scv" && args[0] != "shv" && args[0] != "sqv" && args[0] != "snv" && args[0] != "sbv") {
		return false
	}
	consumesValue := map[string]bool{}
	var visit func(*cobra.Command)
	visit = func(command *cobra.Command) {
		command.Flags().VisitAll(func(flag *pflag.Flag) {
			consumesValue["--"+flag.Name] = flag.NoOptDefVal == ""
			if flag.Shorthand != "" {
				consumesValue["-"+flag.Shorthand] = flag.NoOptDefVal == ""
			}
		})
		for _, child := range command.Commands() {
			visit(child)
		}
	}
	if root != nil {
		for _, command := range root.Commands() {
			if command.Name() == args[0] {
				visit(command)
			}
		}
	}
	requested := false
	for index := 1; index < len(args); index++ {
		arg := args[index]
		if arg == "--" {
			break
		}
		if args[0] == "sbv" && ((arg == "--format" && index+1 < len(args) && (args[index+1] == "json" || args[index+1] == "ndjson")) || arg == "--format=json" || arg == "--format=ndjson") {
			requested = true
		} else if arg == "--json" {
			requested = true
		} else if value, found := strings.CutPrefix(arg, "--json="); found {
			parsed, err := strconv.ParseBool(value)
			// An invalid explicit JSON boolean still requests a machine parse
			// failure. Valid false forms explicitly restore human output.
			requested = err != nil || parsed
		} else if consumesValue[arg] {
			index++
		}
	}
	return requested
}
