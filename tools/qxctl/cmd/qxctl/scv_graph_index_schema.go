package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvtransfer"
	"github.com/spf13/cobra"
	"sort"
)

func graphIndexSchemaEntries(version string) map[string]string {
	m := map[string]string{
		"symphony.qxctl.scv-graph-index-import-input.v1": "QxctlImportInput", "symphony.qxctl.scv-graph-index-query-input.v1": "QxctlQueryInput", "symphony.qxctl.scv-graph-index-export-input.v1": "QxctlExportInput", "symphony.qxctl.scv-graph-index-result.v1": "QxctlResult",
		"symphony.qxctl.scv-index-transfer-input.v1": "TransferInput", "symphony.qxctl.scv-index-transfer-result.v1": "TransferResult", "symphony.qxctl.scv-index-transfer-intent.v1": "TransferIntent", "symphony.qxctl.scv-index-transfer-event.v1": "TransferEvent", "symphony.qxctl.scv-index-schema.v1": "SchemaResult",
	}
	if version == knowledgeengine.SCVGraphIndexPlanningVersion {
		m["symphony.qxctl.scv-graph-index-inventory-input.v1"] = "QxctlInventoryInput"
		m["symphony.qxctl.scv-graph-index-transfer-plan-input.v1"] = "QxctlTransferPlanInput"
		m["symphony.qxctl.scv-graph-index-maintenance-result.v1"] = "QxctlMaintenanceResult"
	}
	return m
}

var graphIndexDiscoverySchema = json.RawMessage(`{"$schema":"https://json-schema.org/draft/2020-12/schema","$defs":{"SchemaResult":{"type":"object","additionalProperties":false,"required":["protocol","installation","entries","schema","digest"],"properties":{"protocol":{"const":"symphony.qxctl.scv-index-schema.v1"},"installation":{"type":"object"},"entries":{"type":"array","items":{"type":"object","additionalProperties":false,"required":["protocol","origin","fragment","schema_digest"],"properties":{"protocol":{"type":"string"},"origin":{"enum":["connector_receipt","qxctl_embedded"]},"fragment":{"type":"string"},"schema_digest":{"type":"string","pattern":"^sha256:[0-9a-f]{64}$"}}}},"schema":{"type":["object","null"]},"digest":{"type":"string","pattern":"^sha256:[0-9a-f]{64}$"}}}}}`)

func graphIndexSchemaResult(inst knowledgeengine.Installation, native json.RawMessage, protocol string) (json.RawMessage, error) {
	nativeDoc, err := scvtransfer.Decode(native)
	if err != nil {
		return nil, err
	}
	transferDoc, err := scvtransfer.Decode(scvtransfer.Schema())
	if err != nil {
		return nil, err
	}
	discoveryDoc, err := scvtransfer.Decode(graphIndexDiscoverySchema)
	if err != nil {
		return nil, err
	}
	entries := graphIndexSchemaEntries(inst.Version)
	ids := []string{}
	for id := range entries {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	rows := []any{}
	var selected any
	for _, id := range ids {
		def := entries[id]
		doc := nativeDoc
		raw := []byte(native)
		origin := "connector_receipt"
		if def == "SchemaResult" {
			doc = discoveryDoc
			raw = graphIndexDiscoverySchema
			origin = "qxctl_embedded"
		} else if _, ok := transferDoc["$defs"].(map[string]any)[def]; ok && len(def) >= 8 && def[:8] == "Transfer" {
			doc = transferDoc
			raw = scvtransfer.Schema()
			origin = "qxctl_embedded"
		}
		if _, ok := doc["$defs"].(map[string]any)[def]; !ok {
			return nil, fmt.Errorf("selected schema lacks %s", def)
		}
		if protocol != "" && protocol != id {
			continue
		}
		if protocol == id {
			selected = doc
		}
		rows = append(rows, map[string]any{"protocol": id, "origin": origin, "fragment": "#/$defs/" + def, "schema_digest": fmt.Sprintf("sha256:%x", sha256.Sum256(raw))})
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("protocol not provided by selected connector/CLI schema profile")
	}
	return scvtransfer.Seal(map[string]any{"protocol": "symphony.qxctl.scv-index-schema.v1", "installation": inst, "entries": rows, "schema": selected})
}
func newSCVGraphIndexSchemaCommand() *cobra.Command {
	var prefix, version, protocol string
	c := &cobra.Command{Use: "schema", Short: "Discover exact connector and CLI transfer schemas", Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error {
		inst, raw, err := knowledgeengine.SCVGraphIndexSchema(prefix, version)
		if err != nil {
			return err
		}
		result, err := graphIndexSchemaResult(inst, raw, protocol)
		if err != nil {
			return err
		}
		return printIndentedJSON(json.RawMessage(result))
	}}
	c.Flags().StringVar(&prefix, "connector-prefix", "", "exact installed connector prefix")
	c.Flags().StringVar(&version, "connector-version", "", "exact installed version")
	c.Flags().StringVar(&protocol, "protocol", "", "show this exact protocol; omit to list")
	c.Flags().Bool("json", false, "emit structured JSON")
	_ = c.MarkFlagRequired("connector-prefix")
	_ = c.MarkFlagRequired("connector-version")
	spec := commandSpec("scv.graph-index.schema", featureSCVAdministration, "discover")
	spec.Mutability = "read_only"
	spec.OutputProtocols = []string{"symphony.qxctl.scv-index-schema.v1"}
	spec.ResultValidationProtocols = spec.OutputProtocols
	commandregistry.Attach(c, spec)
	return c
}
