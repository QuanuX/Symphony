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
	case "argument.count", "argument.unsupported", "internal.failure", "operation.unsupported",
		"request.deadline", "request.deadline_exceeded", "corpus.invalid", "interpretation.invalid", "coverage.invalid", "pack.invalid", "composition.invalid",
		"knowledge.invalid", "knowledge.evaluation_limit",
		"scv.fields", "scv.type", "scv.bounds", "scv.identity", "scv.locator", "scv.digest",
		"scv.domain", "scv.duplicate", "scv.identity_change", "scv.generation", "scv.protocol",
		"scv.stale_state", "scv.plan_mismatch", "scv.time", "scv.capture_size", "scv.capture_encoding",
		"scv.capture_digest", "scv.completeness":
		return &code
	}
	return nil
}

// Parsing can fail before Cobra reaches --json. Inspect output intent without
// executing handlers or reparsing their flag values. The actual SCV tree tells
// us which flags consume the next token, including a value spelled "--json".
// This does not make misplaced/unknown flags valid or change Cobra's grammar.
func scvJSONRequested(root *cobra.Command, args []string) bool {
	activation := len(args) >= 3 && args[0] == "shv" && args[1] == "source" && args[2] == "activation"
	refresh := len(args) >= 2 && args[0] == "shv" && args[1] == "refresh"
	if len(args) == 0 || (args[0] != "scv" && !activation && !refresh) {
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
		if arg == "--json" {
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
