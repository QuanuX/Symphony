package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/spf13/cobra"
)

//go:embed shv_activation.schema.json
var shvActivationSchema json.RawMessage

func newSHVActivationDiscovery(template bool) *cobra.Command {
	var prefix, version, op string
	leaf, protocol := "schema", "symphony.qxctl.shv-source-activation-schema.v1"
	if template {
		leaf, protocol = "template", "symphony.qxctl.shv-source-activation-template.v1"
	}
	c := &cobra.Command{Use: leaf, Args: usageOnlyArgs, Short: "Discover CLI-owned protected source contracts", RunE: func(*cobra.Command, []string) error {
		installation, e := knowledgeengine.InspectSHVSource(prefix, version)
		if e != nil {
			return e
		}
		result := map[string]any{"protocol": protocol, "installation": installation}
		if !template {
			result["origin"] = "qxctl_embedded"
			result["schema"] = shvActivationSchema
		} else {
			if op != "propose" && op != "apply" {
				return fmt.Errorf("template operation must be propose or apply")
			}
			var value any
			if op == "propose" {
				value = map[string]any{"operation_id": nil, "desired": nil, "reason": nil}
			} else {
				value = map[string]any{"protocol": "symphony.shv.source-plan.v1", "operation_id": nil, "expected_state_digest": nil, "change_kind": nil, "reason": nil, "source": nil, "digest": nil}
			}
			result["operation"] = op
			result["template"] = value
			result["status"] = "unanswered_template_not_validated_input"
		}
		raw, e := sealSHVActivation(result)
		if e != nil {
			return e
		}
		return printIndentedJSON(raw)
	}}
	c.Flags().StringVar(&prefix, "prefix", "", "exact installed SHV source engine prefix")
	c.Flags().StringVar(&version, "version", "", "exact source engine version")
	c.Flags().Bool("json", false, "emit structured JSON")
	_ = c.MarkFlagRequired("prefix")
	_ = c.MarkFlagRequired("version")
	if template {
		c.Flags().StringVar(&op, "operation", "", "propose or apply")
		_ = c.MarkFlagRequired("operation")
	}
	c.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	s := commandSpec("shv.source.activation."+leaf, featureSHVAdministration, "discover")
	s.Mutability = "read_only"
	s.OutputProtocols = []string{protocol}
	s.ResultValidationProtocols = s.OutputProtocols
	commandregistry.Attach(c, s)
	return c
}
