package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/spf13/cobra"
)

const graphIndexFeature = "ssfv:symphony:scv-graph-duckdb-connector"

type graphIndexOptions struct {
	scv                                                                      scvOptions
	backend, connectorPrefix, connectorVersion, root, namespace, operationID string
}

func newSCVGraphIndexCommand() *cobra.Command {
	group := structural("graph-index", fmt.Errorf("graph-index subcommand is required: inspect, import, status, recover, query, export"))
	for _, action := range []string{"inspect", "import", "status", "recover", "query", "export"} {
		o := graphIndexOptions{}
		child := &cobra.Command{Use: action, Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error { return runSCVGraphIndex(action, o) }}
		if action == "import" {
			scvRetainedReadFlags(child, &o.scv)
			child.Flags().StringVar(&o.scv.domain, "domain", "scv", "exact SCV validating owner domain")
			child.Flags().StringVar(&o.scv.prefix, "prefix", "", "exact receipt-v2 SCV owner installation prefix")
			child.Flags().StringVar(&o.scv.version, "version", "", "exact installed SCV owner version; never latest")
			for _, flag := range []string{"domain", "prefix", "version"} {
				_ = child.MarkFlagRequired(flag)
			}
		} else {
			scvRetainedReadFlags(child, &o.scv)
		}
		child.Flags().StringVar(&o.backend, "backend", "duckdb", "selected optional SQL index backend (user-selected default duckdb)")
		child.Flags().StringVar(&o.connectorPrefix, "connector-prefix", "", "exact receipt-owned connector installation prefix")
		child.Flags().StringVar(&o.connectorVersion, "connector-version", "", "exact connector version, currently 0.1.0-dev; never latest")
		_ = child.MarkFlagRequired("connector-prefix")
		_ = child.MarkFlagRequired("connector-version")
		if action != "inspect" {
			child.Flags().StringVar(&o.root, "index-root", "", "existing private index root; fixed database filename owned by connector")
			child.Flags().StringVar(&o.scv.topsID, "tops-id", "", "exact TOPS UUID for index namespace")
			child.Flags().StringVar(&o.namespace, "namespace", "", "explicit logical graph index namespace")
			for _, flag := range []string{"index-root", "tops-id", "namespace"} {
				_ = child.MarkFlagRequired(flag)
			}
		}
		if action == "status" || action == "recover" {
			child.Flags().StringVar(&o.operationID, "operation-id", "", "exact retained index operation identity")
			_ = child.MarkFlagRequired("operation-id")
		}
		interaction := map[string]string{"inspect": "inspect", "import": "invoke", "status": "inspect", "recover": "recover", "query": "query", "export": "query"}[action]
		spec := commandSpec("scv.graph-index."+action, featureSCVAdministration, interaction)
		spec.Mutability = "evidence_only" // DuckDB may physically recover even during logical reads.
		if action == "inspect" {
			spec.Mutability = "read_only"
		}
		spec.InputProtocols = []string{}
		if action == "import" || action == "query" || action == "export" {
			spec.InputProtocols = []string{"symphony.qxctl.scv-graph-index-" + action + "-input.v1"}
		}
		spec.OutputProtocols = []string{"symphony.qxctl.scv-graph-index-result.v1"}
		spec.ResultValidationProtocols = spec.OutputProtocols
		ops := map[string][]string{"inspect": {"inspect"}, "import": {"prepare", "commit"}, "status": {"status"}, "recover": {"status", "commit"}, "query": {"query"}, "export": {"export"}}[action]
		seen := map[string]bool{}
		for _, op := range ops {
			spec.BackendOperationIDs = append(spec.BackendOperationIDs, "engop:symphony:scv.graph-index."+op)
			binding := map[string]string{"inspect": "inspect", "prepare": "invoke", "commit": "invoke", "status": "inspect", "query": "query", "export": "query"}[op]
			if !seen[binding] {
				spec.FeatureBindings = append(spec.FeatureBindings, commandregistry.FeatureBinding{FeatureID: graphIndexFeature, Interaction: binding})
				seen[binding] = true
			}
		}
		if action == "recover" {
			spec.FeatureBindings = append(spec.FeatureBindings, commandregistry.FeatureBinding{FeatureID: graphIndexFeature, Interaction: "recover"})
		}
		if action == "import" || action == "recover" || action == "query" || action == "export" {
			for _, domain := range knowledgeengine.SCVDomains() {
				spec.BackendOperationIDs = append(spec.BackendOperationIDs, "engop:symphony:"+domain+".graph.query")
				spec.FeatureBindings = append(spec.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:" + domain + "-engine", Interaction: "query"})
			}
		}
		if action == "import" || action == "recover" {
			spec.RecoveryCommandID = stringPointer("qxcmd:symphony:scv.graph-index.recover")
		}
		commandregistry.Attach(child, spec)
		group.AddCommand(child)
	}
	return group
}

type graphIndexRunner struct {
	options      graphIndexOptions
	connector    knowledgeengine.Installation
	inspectOwner func(string, string, string) (knowledgeengine.Installation, error)
	owner        func(knowledgeengine.Installation, json.RawMessage, string) (json.RawMessage, error)
	invoke       func(string, map[string]any) (json.RawMessage, error)
	afterPrepare func() error
}

func newGraphIndexRunner(action string, o graphIndexOptions) (*graphIndexRunner, error) {
	if o.scv.repository != "" {
		return nil, fmt.Errorf("graph-index does not accept --repo; the operation working directory is --index-root")
	}
	if o.backend != "duckdb" || o.connectorPrefix == "" || o.connectorVersion != knowledgeengine.SCVGraphIndexConnectorVersion {
		return nil, fmt.Errorf("graph index requires selected duckdb backend and exact connector prefix/version")
	}
	cwd := ""
	if action != "inspect" {
		if !filepath.IsAbs(o.root) || filepath.Clean(o.root) != o.root || o.root == "/" {
			return nil, fmt.Errorf("--index-root must be an existing clean absolute private directory")
		}
		resolved, err := filepath.EvalSymlinks(o.root)
		if err != nil || resolved != o.root {
			return nil, fmt.Errorf("index root cannot contain symlinks")
		}
		info, err := os.Lstat(o.root)
		if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 {
			return nil, fmt.Errorf("index root must be an existing 0700 directory")
		}
		if stavprotocol.ValidateTOPSID(o.scv.topsID) != nil || !regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`).MatchString(o.namespace) {
			return nil, fmt.Errorf("index TOPS or namespace is invalid")
		}
		cwd = o.root
	} else {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}
	connector, err := knowledgeengine.InspectSCVGraphIndexConnector(o.backend, o.connectorPrefix, o.connectorVersion)
	if err != nil {
		return nil, err
	}
	r := &graphIndexRunner{options: o, connector: connector, inspectOwner: knowledgeengine.InspectSCVDomain}
	r.invoke = func(operation string, payload map[string]any) (json.RawMessage, error) {
		raw, err := knowledgeengine.SCVCanonical(payload)
		if err != nil {
			return nil, err
		}
		current, err := knowledgeengine.InspectSCVGraphIndexConnector(o.backend, o.connectorPrefix, o.connectorVersion)
		if err != nil || current != connector {
			return nil, fmt.Errorf("selected connector installation changed")
		}
		response, err := knowledgeengine.InvokeSCVGraphIndexConnector(context.Background(), o.backend, o.connectorPrefix, o.connectorVersion, cwd, operation, raw)
		return response.Result, err
	}
	r.owner = func(inst knowledgeengine.Installation, graph json.RawMessage, queryTime string) (json.RawMessage, error) {
		payload, err := knowledgeengine.SCVCanonical(map[string]any{"graph": graph, "query_time": queryTime})
		if err != nil {
			return nil, err
		}
		response, err := knowledgeengine.InvokeSCVDomain(context.Background(), inst.Role, inst.Prefix, inst.Version, cwd, "graph_query", payload)
		return response.Result, err
	}
	return r, nil
}

func graphIndexInput(o graphIndexOptions) (map[string]any, error) {
	raw, err := knowledgeengine.ReadPayload(o.scv.input)
	if err != nil {
		return nil, err
	}
	if err = knowledgeengine.ValidateSCVBundleText(raw); err != nil {
		return nil, err
	}
	var input map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	err = decoder.Decode(&input)
	return input, err
}
func graphIndexExact(input map[string]any, fields ...string) bool {
	if input == nil || len(input) != len(fields) {
		return false
	}
	for _, key := range fields {
		if _, ok := input[key]; !ok {
			return false
		}
	}
	return true
}
func graphIndexQueryTime(value any) (string, error) {
	s, ok := value.(string)
	if !ok || len(s) != 20 {
		return "", fmt.Errorf("explicit canonical UTC query_time is required")
	}
	t, err := time.Parse("2006-01-02T15:04:05Z", s)
	if err != nil || t.Format("2006-01-02T15:04:05Z") != s {
		return "", fmt.Errorf("invalid query_time")
	}
	return s, nil
}
func (r *graphIndexRunner) validateOwner(inst knowledgeengine.Installation, graph json.RawMessage, queryTime string) (json.RawMessage, error) {
	current, err := r.inspectOwner(inst.Role, inst.Prefix, inst.Version)
	if err != nil || current != inst {
		return nil, fmt.Errorf("graph index requires original exact SCV owner installation")
	}
	result, err := r.owner(inst, graph, queryTime)
	if err != nil {
		return nil, err
	}
	current, err = r.inspectOwner(inst.Role, inst.Prefix, inst.Version)
	if err != nil || current != inst {
		return nil, fmt.Errorf("graph index owner installation changed during validation")
	}
	return result, nil
}

func graphIndexOutput(action string, connector, owner json.RawMessage) (json.RawMessage, error) {
	if owner == nil {
		owner = json.RawMessage("null")
	}
	value := map[string]any{"protocol": "symphony.qxctl.scv-graph-index-result.v1", "operation": action, "connector_result": connector, "owner_evaluation": owner}
	digest, err := knowledgeengine.SCVDigest(value)
	if err != nil {
		return nil, err
	}
	value["digest"] = digest
	raw, err := knowledgeengine.SCVCanonical(value)
	if err != nil {
		return nil, err
	}
	if err = knowledgeengine.ValidateJSONObject(raw, 4<<20); err != nil {
		return nil, fmt.Errorf("graph index result envelope exceeds existing bounds: %w", err)
	}
	if err = knowledgeengine.ValidateSCVBundleUnicode(raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (r *graphIndexRunner) execute(action string, input map[string]any) (json.RawMessage, error) {
	o := r.options
	if action == "inspect" {
		if input != nil || o.scv.input != "" {
			return nil, fmt.Errorf("graph-index inspect does not accept input")
		}
		result, err := r.invoke("inspect", map[string]any{})
		if err != nil {
			return nil, err
		}
		return graphIndexOutput(action, result, nil)
	}
	context := func() map[string]any { return map[string]any{"tops_id": o.scv.topsID, "namespace": o.namespace} }
	var native, evaluation json.RawMessage
	var err error
	if action == "import" {
		if !graphIndexExact(input, "operation_id", "graph", "query_time") || o.scv.domain == "" || o.scv.prefix == "" || o.scv.version == "" {
			return nil, fmt.Errorf("graph-index import requires exact operation_id/graph/query_time and explicit SCV owner")
		}
		queryTime, err := graphIndexQueryTime(input["query_time"])
		if err != nil {
			return nil, err
		}
		inst, err := r.inspectOwner(o.scv.domain, o.scv.prefix, o.scv.version)
		if err != nil {
			return nil, err
		}
		graph, err := knowledgeengine.SCVCanonical(input["graph"])
		if err != nil {
			return nil, err
		}
		evaluation, err = r.validateOwner(inst, graph, queryTime)
		if err != nil {
			return nil, err
		}
		payload := context()
		for key, value := range input {
			payload[key] = value
		}
		payload["owner"] = inst
		payload["connector"] = r.connector
		native, err = r.invoke("prepare", payload)
		if err != nil {
			return nil, err
		}
		ref, err := knowledgeengine.SCVGraphIndexReference(native)
		if err != nil {
			return nil, err
		}
		if ref.Owner != inst || ref.Connector != r.connector {
			return nil, fmt.Errorf("prepared graph changed selected installation")
		}
		if _, err = graphIndexOutput(action, native, evaluation); err != nil {
			return nil, err
		}
		if r.afterPrepare != nil {
			if err = r.afterPrepare(); err != nil {
				return nil, err
			}
		}
		current, err := r.inspectOwner(inst.Role, inst.Prefix, inst.Version)
		if err != nil || current != inst {
			return nil, fmt.Errorf("graph index owner changed before commit; intent retained")
		}
		commit := context()
		commit["operation_id"] = input["operation_id"]
		commit["expected_intent_digest"] = ref.IntentDigest
		native, err = r.invoke("commit", commit)
		if err != nil {
			return nil, err
		}
		current, err = r.inspectOwner(inst.Role, inst.Prefix, inst.Version)
		if err != nil || current != inst {
			return nil, fmt.Errorf("graph index retained; owner changed before returning validation")
		}
	} else if action == "status" || action == "recover" {
		if input != nil || o.scv.input != "" || o.operationID == "" {
			return nil, fmt.Errorf("graph-index status/recover requires --operation-id and no input")
		}
		payload := context()
		payload["operation_id"] = o.operationID
		native, err = r.invoke("status", payload)
		if err != nil {
			return nil, err
		}
		if action == "recover" {
			ref, err := knowledgeengine.SCVGraphIndexReference(native)
			if err != nil {
				return nil, err
			}
			evaluation, err = r.validateOwner(ref.Owner, ref.Graph, ref.QueryTime)
			if err != nil {
				return nil, err
			}
			if _, err = graphIndexOutput(action, native, evaluation); err != nil {
				return nil, err
			}
			payload["expected_intent_digest"] = ref.IntentDigest
			native, err = r.invoke("commit", payload)
			if err != nil {
				return nil, err
			}
			current, err := r.inspectOwner(ref.Owner.Role, ref.Owner.Prefix, ref.Owner.Version)
			if err != nil || current != ref.Owner {
				return nil, fmt.Errorf("graph index retained; owner changed before returning validation")
			}
		}
	} else if action == "query" || action == "export" {
		fields := []string{"snapshot_digest", "query_time"}
		if action == "query" {
			fields = []string{"snapshot_digest", "kind", "filters", "cursor", "limit", "query_time"}
		}
		if !graphIndexExact(input, fields...) {
			return nil, fmt.Errorf("graph-index %s input has missing or unknown fields", action)
		}
		queryTime, err := graphIndexQueryTime(input["query_time"])
		if err != nil {
			return nil, err
		}
		payload := context()
		for key, value := range input {
			if key != "query_time" {
				payload[key] = value
			}
		}
		native, err = r.invoke(action, payload)
		if err != nil {
			return nil, err
		}
		ref, err := knowledgeengine.SCVGraphIndexReference(native)
		if err != nil {
			return nil, err
		}
		evaluation, err = r.validateOwner(ref.Owner, ref.Graph, queryTime)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unsupported graph-index route")
	}
	return graphIndexOutput(action, native, evaluation)
}

func runSCVGraphIndex(action string, o graphIndexOptions) error {
	var input map[string]any
	var err error
	if action == "import" || action == "query" || action == "export" {
		input, err = graphIndexInput(o)
		if err != nil {
			return err
		}
	}
	r, err := newGraphIndexRunner(action, o)
	if err != nil {
		return err
	}
	result, err := r.execute(action, input)
	if err != nil {
		return err
	}
	return outputSCV(o.scv, result)
}
