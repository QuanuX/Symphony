package main

import (
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvtransfer"
	"github.com/spf13/cobra"
)

func newSCVGraphIndexTransferCommand(action string) *cobra.Command {
	o := graphIndexOptions{}
	var targetRoot, expected string
	c := &cobra.Command{Use: action, Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error {
		if action == "transfer-status" {
			var out json.RawMessage
			err := scvtransfer.With(targetRoot, false, nil, func(j *scvtransfer.Journal) error {
				if j.Intent.Digest != expected {
					return fmt.Errorf("expected transfer identity differs")
				}
				return transferOutput(j, nil, nil, false, &out)
			})
			if err != nil {
				return err
			}
			return outputSCV(o.scv, out)
		}
		input, err := graphIndexInput(o)
		if err != nil {
			return err
		}
		source, err := newGraphIndexRunner(action, o)
		if err != nil {
			return err
		}
		runner, err := newTransferRunner(source, input)
		if err != nil {
			return err
		}
		out, err := runner.execute(action == "transfer")
		if err != nil {
			return err
		}
		return outputSCV(o.scv, out)
	}}
	s := commandSpec("scv.graph-index."+action, featureSCVAdministration, "invoke")
	s.Mutability = "evidence_only"
	s.InputProtocols = []string{"symphony.qxctl.scv-index-transfer-input.v1"}
	s.OutputProtocols = []string{"symphony.qxctl.scv-index-transfer-result.v1"}
	s.ResultValidationProtocols = s.OutputProtocols
	if action == "transfer-status" {
		c.Flags().BoolVar(&o.scv.jsonOutput, "json", false, "structured output")
		c.Flags().StringVar(&targetRoot, "target-root", "", "exact existing target root")
		c.Flags().StringVar(&expected, "transfer-digest", "", "exact durable transfer identity")
		_ = c.MarkFlagRequired("target-root")
		_ = c.MarkFlagRequired("transfer-digest")
		s = commandSpec("scv.graph-index."+action, featureSCVAdministration, "inspect")
		s.Mutability = "read_only"
		s.InputProtocols = []string{}
		s.OutputProtocols = []string{"symphony.qxctl.scv-index-transfer-result.v1"}
		s.ResultValidationProtocols = s.OutputProtocols
	} else {
		scvRetainedReadFlags(c, &o.scv)
		c.Flags().StringVar(&o.backend, "backend", "duckdb", "selected SQL backend")
		c.Flags().StringVar(&o.connectorPrefix, "connector-prefix", "", "exact 0.2.0-dev source reader installation")
		c.Flags().StringVar(&o.connectorVersion, "connector-version", "", "exact source reader version")
		c.Flags().StringVar(&o.root, "index-root", "", "existing source root")
		c.Flags().StringVar(&o.scv.topsID, "tops-id", "", "exact TOPS UUID")
		c.Flags().StringVar(&o.namespace, "namespace", "", "exact namespace")
		for _, f := range []string{"connector-prefix", "connector-version", "index-root", "tops-id", "namespace"} {
			_ = c.MarkFlagRequired(f)
		}
		if action == "transfer-recover" {
			s = commandSpec("scv.graph-index."+action, featureSCVAdministration, "recover")
			s.Mutability = "evidence_only"
			s.InputProtocols = []string{"symphony.qxctl.scv-index-transfer-input.v1"}
			s.OutputProtocols = []string{"symphony.qxctl.scv-index-transfer-result.v1"}
			s.ResultValidationProtocols = s.OutputProtocols
		}
		s.BackendOperationIDs = []string{"engop:symphony:scv.graph-index.inspect", "engop:symphony:scv.graph-index.inventory", "engop:symphony:scv.graph-index.transfer.plan", "engop:symphony:scv.graph-index.prepare", "engop:symphony:scv.graph-index.commit"}
		for _, interaction := range []string{"inspect", "query", "invoke"} {
			s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: graphIndexFeature, Interaction: interaction})
		}
		for _, domain := range knowledgeengine.SCVDomains() {
			s.BackendOperationIDs = append(s.BackendOperationIDs, "engop:symphony:"+domain+".graph.query")
			s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:" + domain + "-engine", Interaction: "query"})
		}
		s.RecoveryCommandID = stringPointer("qxcmd:symphony:scv.graph-index.transfer-recover")
	}
	commandregistry.Attach(c, s)
	return c
}
