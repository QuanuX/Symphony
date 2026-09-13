package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/spf13/cobra"
)

func newSCVBundleCommand() *cobra.Command {
	group := structural("bundle", fmt.Errorf("bundle subcommand is required: pack, inspect, evaluate, expand, workflow, obligations"))
	for _, leaf := range []string{"pack", "inspect", "evaluate", "expand"} {
		options := scvOptions{}
		command := &cobra.Command{Use: leaf, Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error { return runSCVBundle(leaf, options) }}
		scvFlagsVersion(command, &options, "0.9.0-dev")
		if leaf == "inspect" {
			attachSCV(command, "scv.composition.bundle.inspect", "bundle_inspect", "inspect", "read_only", false)
		} else if leaf == "evaluate" {
			attachSCV(command, "scv.composition.bundle.evaluate", "composition_bundle_evaluate", "validate", "evidence_only", false)
		} else {
			operation, interaction, ownerInteraction := "bundle_inspect", "invoke", "inspect"
			inputs := []string{"symphony.qxctl.scv.bundle-pack-input.v1"}
			outputs := []string{"symphony.scv.composition-bundle-evaluate-input.v1"}
			if leaf == "expand" {
				operation, interaction, ownerInteraction = "composition_bundle_evaluate", "query", "validate"
				inputs = []string{"symphony.scv.composition-bundle-evaluate-input.v1"}
				outputs = []string{"symphony.scv.composition-exploration.v1", "symphony.scv.composition-reassessment.v1", "symphony.scv.composition-obligations.v1", "symphony.scv.composition-followup.v1"}
				sort.Strings(outputs)
			}
			spec := commandSpec("scv.composition.bundle."+leaf, featureSCVAdministration, interaction)
			spec.Mutability = "evidence_only"
			spec.InputProtocols, spec.OutputProtocols, spec.ResultValidationProtocols = inputs, outputs, outputs
			for _, domain := range knowledgeengine.SCVDomains() {
				suffix := "bundle.inspect"
				if operation == "composition_bundle_evaluate" {
					suffix = "composition.bundle.evaluate"
				}
				spec.BackendOperationIDs = append(spec.BackendOperationIDs, "engop:symphony:"+domain+"."+suffix)
				spec.FeatureBindings = append(spec.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:" + domain + "-engine", Interaction: ownerInteraction})
			}
			commandregistry.Attach(command, spec)
		}
		group.AddCommand(command)
	}
	group.AddCommand(newSCVBundleWorkflowCommand(), newSCVBundleObligationsCommand())
	return group
}

func packSCVBundle(raw []byte, options scvOptions) (json.RawMessage, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	if len(envelope) != 2 || envelope["operation"] == nil || envelope["input"] == nil {
		return nil, fmt.Errorf("bundle packing requires exactly operation and input")
	}
	var operation string
	if err := json.Unmarshal(envelope["operation"], &operation); err != nil {
		return nil, err
	}
	switch operation {
	case "composition_explore", "composition_reassess", "composition_obligations", "composition_followup":
	default:
		return nil, fmt.Errorf("bundle packing requires a supported composition operation")
	}
	bundle, err := knowledgeengine.SCVBundleEncode(envelope["input"])
	if err != nil {
		return nil, err
	}
	return knowledgeengine.SCVCanonical(map[string]any{"operation": operation, "owner": map[string]any{"domain": options.domain, "version": options.version}, "bundle": json.RawMessage(bundle)})
}

func runSCVBundle(action string, options scvOptions) error {
	if action == "pack" {
		raw, err := knowledgeengine.ReadSCVBundlePackPayload(options.input)
		if err != nil {
			return err
		}
		packed, err := packSCVBundle(raw, options)
		if err != nil {
			return err
		}
		if _, err = invokeSCV(options, "bundle_inspect", packed); err != nil {
			return err
		}
		// Emit ready-to-read transport bytes: indentation can push a valid
		// invocation over the unchanged one-MiB input-file limit.
		_, err = fmt.Fprintln(os.Stdout, string(packed))
		return err
	}
	if action == "inspect" || action == "evaluate" || action == "expand" {
		input, err := knowledgeengine.ReadPayload(options.input)
		if err != nil {
			return err
		}
		if err = knowledgeengine.ValidateSCVBundleText(input); err != nil {
			return err
		}
		operation := "composition_bundle_evaluate"
		if action == "inspect" {
			operation = "bundle_inspect"
		}
		// Keep the original validated bytes until the native process boundary.
		// Decoding here first would silently replace lone Unicode surrogates.
		raw, err := invokeSCV(options, operation, json.RawMessage(input))
		if err != nil {
			return err
		}
		if action != "expand" {
			return outputSCV(options, raw)
		}
		var result map[string]json.RawMessage
		decoder := json.NewDecoder(bytes.NewReader(raw))
		if err = decoder.Decode(&result); err != nil {
			return err
		}
		expanded, err := knowledgeengine.SCVBundleDecode(result["result_bundle"])
		if err != nil {
			return err
		}
		return outputSCV(options, expanded)
	}
	return fmt.Errorf("unsupported bundle action")
}

// Only new bundle artifact imports need this raw-text guard. The retained
// import runner subsequently uses its established exact owner replay/storage.
func scvArtifactInput(operation string, options scvOptions) (map[string]any, error) {
	if operation != "import" {
		return scvInput(options)
	}
	raw, err := knowledgeengine.ReadPayload(options.input)
	if err != nil {
		return nil, err
	}
	var envelope map[string]json.RawMessage
	if err = json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	var nativeOperation string
	// A wrong operation type is rejected later by the established import path.
	_ = json.Unmarshal(envelope["operation"], &nativeOperation)
	if nativeOperation == "bundle_inspect" || nativeOperation == "composition_bundle_evaluate" {
		if err = knowledgeengine.ValidateSCVBundleText(raw); err != nil {
			return nil, err
		}
	}
	var input map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	err = decoder.Decode(&input)
	return input, err
}
