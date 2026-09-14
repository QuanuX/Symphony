package main

import (
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/shvtransfer"
	"github.com/spf13/cobra"
)

func newSHVStoreTransferCommand(action string) *cobra.Command {
	var targetRoot, expected, prefix, version, sourceRoot, input, backend string
	c := &cobra.Command{Use: action, Short: "Copy caller-selected structural evidence or inspect retained transfer progress", Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error {
		if action == "transfer-status" {
			var out json.RawMessage
			e := shvtransfer.With(targetRoot, false, nil, func(j *shvtransfer.Journal) error {
				if j.Intent.Digest != expected {
					return fmt.Errorf("expected transfer identity differs")
				}
				return shvTransferOutput(j, nil, nil, false, &out)
			})
			if e != nil {
				return e
			}
			return printIndentedJSON(out)
		}
		if backend != "duckdb" {
			return fmt.Errorf("unsupported graph store backend")
		}
		raw, e := knowledgeengine.ReadPayload(input)
		if e != nil {
			return e
		}
		p, e := shvtransfer.Decode(raw)
		if e != nil {
			return e
		}
		source, e := newSHVStoreRunner(sourceRoot, prefix, version)
		if e != nil {
			return e
		}
		runner, e := newSHVTransferRunner(source, p)
		if e != nil {
			return e
		}
		out, e := runner.execute(true)
		if e != nil {
			return e
		}
		return printIndentedJSON(out)
	}}
	c.Flags().Bool("json", false, "structured output")
	s := commandSpec("shv.graph.store."+action, featureSHVAdministration, "invoke")
	s.Mutability = "evidence_only"
	s.TargetScope = "local"
	s.InputProtocols = []string{"symphony.qxctl.shv-store-transfer-input.v1"}
	s.OutputProtocols = []string{"symphony.qxctl.shv-store-transfer-result.v1"}
	if action == "transfer-status" {
		c.Flags().StringVar(&targetRoot, "target-root", "", "exact existing target root")
		c.Flags().StringVar(&expected, "transfer-digest", "", "exact durable transfer identity")
		c.MarkFlagRequired("target-root")
		c.MarkFlagRequired("transfer-digest")
		s = commandSpec("shv.graph.store."+action, featureSHVAdministration, "inspect")
		s.Mutability = "read_only"
		s.TargetScope = "local"
		s.OutputProtocols = []string{"symphony.qxctl.shv-store-transfer-result.v1"}
	} else {
		c.Flags().StringVar(&prefix, "connector-prefix", "", "exact source reader installation")
		c.Flags().StringVar(&version, "connector-version", "", "exact source reader version")
		c.Flags().StringVar(&sourceRoot, "store-root", "", "existing private source database directory")
		c.Flags().StringVar(&input, "input", "", "plan and expected_plan_digest; repeat identical input to recover")
		c.Flags().StringVar(&backend, "backend", "duckdb", "explicit storage backend")
		for _, f := range []string{"connector-prefix", "connector-version", "store-root", "input"} {
			c.MarkFlagRequired(f)
		}
		s.BackendOperationIDs = []string{"engop:symphony:shv.graph-store.inspect", "engop:symphony:shv.graph-store.inventory", "engop:symphony:shv.graph-store.transfer.plan", "engop:symphony:shv.graph-store.prepare", "engop:symphony:shv.graph-store.commit"}
		for _, i := range []string{"inspect", "query", "invoke", "recover"} {
			s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:shv-graph-duckdb-connector", Interaction: i})
		}
	}
	s.ResultValidationProtocols = s.OutputProtocols
	commandregistry.Attach(c, s)
	return c
}
