package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/shvjob"
	"github.com/spf13/cobra"
	"os"
	"regexp"
)

const jobProtocol = "symphony.qxctl.shv-materialization.v1"

var jobToken = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)

type jobOptions struct {
	root, id, prefix, version, input string
	budget                           int
}

func newSHVMaterializationCommand() *cobra.Command {
	root := structural("materialization", fmt.Errorf("materialization operation required"))
	for _, op := range []string{"prepare", "run", "resume", "status", "export", "schema", "template"} {
		o := jobOptions{}
		c := &cobra.Command{Use: op, Args: usageOnlyArgs, Short: "Manage private resumable evidence without catalogue publication", RunE: func(*cobra.Command, []string) error { return runSHVJob(op, o) }}
		c.Flags().Bool("json", false, "emit structured evidence")
		if op != "schema" && op != "template" {
			c.Flags().StringVar(&o.root, "job-root", "", "explicit private job root")
			c.Flags().StringVar(&o.id, "job-id", "", "stable caller job ID")
			_ = c.MarkFlagRequired("job-root")
			_ = c.MarkFlagRequired("job-id")
		}
		if op == "prepare" {
			c.Flags().StringVar(&o.prefix, "prefix", "", "exact partition installation")
			c.Flags().StringVar(&o.version, "version", "", "exact partition version")
			c.Flags().StringVar(&o.input, "input", "", "task plan JSON")
			for _, f := range []string{"prefix", "version", "input"} {
				_ = c.MarkFlagRequired(f)
			}
		}
		if op == "run" || op == "resume" {
			c.Flags().IntVar(&o.budget, "max-tasks", 0, "pending task budget 1..8")
			_ = c.MarkFlagRequired("max-tasks")
		}
		c.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		interaction := "invoke"
		if op == "status" {
			interaction = "inspect"
		}
		if op == "resume" {
			interaction = "recover"
		}
		if op == "schema" || op == "template" {
			interaction = "discover"
		}
		s := commandSpec("shv.materialization."+op, featureSHVAdministration, interaction)
		s.Mutability = "read_only"
		if op == "prepare" || op == "run" || op == "resume" {
			s.Mutability = "evidence_only"
		}
		s.OutputProtocols = []string{"symphony.qxctl.shv-materialization-result.v1"}
		if op == "schema" || op == "template" {
			s.OutputProtocols = []string{"symphony.qxctl.shv-materialization-" + op + ".v1"}
		}
		if op == "prepare" {
			s.InputProtocols = []string{"symphony.qxctl.shv-materialization-plan.v1"}
		}
		if op == "run" || op == "resume" || op == "export" {
			s.BackendOperationIDs = []string{"engop:symphony:shv-partition.partition.build", "engop:symphony:shv-source.capture.import", "engop:symphony:shv-source.graph.project", "engop:symphony:shv.catalogue.build", "engop:symphony:shv.coverage.plan", "engop:symphony:shv.evaluate", "engop:symphony:shv.graph.project"}
			if op == "export" {
				s.BackendOperationIDs = append(s.BackendOperationIDs, "engop:symphony:shv-partition.manifest.build")
			}
			for _, f := range []string{"shv-partition-engine", "shv-source-engine", "shv-engine"} {
				s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:" + f, Interaction: "invoke"})
			}
		}
		if op == "prepare" {
			s.BackendOperationIDs = []string{"engop:symphony:shv-partition.manifest.build"}
			s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:shv-partition-engine", Interaction: "invoke"})
		}
		s.ResultValidationProtocols = s.OutputProtocols
		commandregistry.Attach(c, s)
		root.AddCommand(c)
	}
	root.AddCommand(newSHVRelocationCommand())
	return root
}
func jobDecode(raw json.RawMessage, keys ...string) (map[string]json.RawMessage, error) {
	return refreshObject(raw, keys...)
}
func jobSeal(m map[string]json.RawMessage) (json.RawMessage, error) {
	v := map[string]any{}
	for k, x := range m {
		v[k] = x
	}
	return sealSHVActivation(v)
}
func jobMarshal(v any) json.RawMessage { r, _ := json.Marshal(v); return r }
func jobBundle(raw json.RawMessage, o shvRefreshOptions) (map[string]json.RawMessage, error) {
	b, e := jobDecode(raw, "protocol", "tops_id", "source_id", "source", "source_installation", "kernel_installation", "request", "captures", "catalogue", "coverage", "evaluation", "source_graph", "catalogue_graph", "digest")
	if e != nil {
		return nil, e
	}
	sealed, e := jobSeal(b)
	if e != nil || !refreshEqual(raw, sealed) || refreshString(b["protocol"]) != shvRefreshProtocol || refreshString(b["tops_id"]) != o.topsID || refreshString(b["source_id"]) != o.sourceID {
		return nil, fmt.Errorf("job bundle identity differs")
	}
	for _, pair := range []struct{ key, prefix, version string }{{"source_installation", o.sourcePrefix, o.sourceVersion}, {"kernel_installation", o.prefix, o.version}} {
		var i knowledgeengine.Installation
		if json.Unmarshal(b[pair.key], &i) != nil || i.Prefix != pair.prefix || i.Version != pair.version {
			return nil, fmt.Errorf("job endpoint differs from pinned bundle")
		}
	}
	return b, nil
}
func jobValidate(raw json.RawMessage, id string) (map[string]json.RawMessage, []map[string]json.RawMessage, error) {
	var header map[string]json.RawMessage
	if e := json.Unmarshal(raw, &header); e != nil {
		return nil, nil, e
	}
	keys := []string{"protocol", "job_id", "installation", "tasks", "required_references", "digest"}
	derived := refreshString(header["protocol"]) == derivedJobProtocol
	if derived {
		keys = append(keys, "origin")
	}
	j, e := jobDecode(raw, keys...)
	if e != nil {
		return nil, nil, e
	}
	if (!derived && refreshString(j["protocol"]) != jobProtocol) || refreshString(j["job_id"]) != id {
		return nil, nil, fmt.Errorf("job identity differs")
	}
	var tasks []map[string]json.RawMessage
	if json.Unmarshal(j["tasks"], &tasks) != nil || len(tasks) < 1 || len(tasks) > 8 {
		return nil, nil, fmt.Errorf("job tasks outside bounds")
	}
	if _, e = jobDecode(j["installation"], "Role", "ModuleID", "EngineID", "Version", "Prefix", "ReceiptPath", "ReceiptDigest", "ReceiptProtocol", "ExecutablePath", "ExecutableDigest"); e != nil {
		return nil, nil, e
	}
	var selected knowledgeengine.Installation
	if e = json.Unmarshal(j["installation"], &selected); e != nil {
		return nil, nil, e
	}
	if _, e = knowledgeengine.ExpectedSHVPartitionVersion("manifest_build", jobMarshal(map[string]any{"entries": []any{}, "required_references": j["required_references"]}), selected.Version); e != nil {
		return nil, nil, e
	}
	seen := map[string]bool{}
	for _, t := range tasks {
		if _, e = jobDecode(jobMarshal(t), "task_id", "endpoint", "bundle", "completion"); e != nil {
			return nil, nil, e
		}
		tid := refreshString(t["task_id"])
		if !jobToken.MatchString(tid) || seen[tid] {
			return nil, nil, fmt.Errorf("invalid/duplicate job task")
		}
		seen[tid] = true
		o, e := comparisonEndpoint(t["endpoint"])
		if e != nil {
			return nil, nil, e
		}
		b, e := jobBundle(t["bundle"], o)
		if e != nil {
			return nil, nil, e
		}
		if string(t["completion"]) != "null" {
			c, e := jobDecode(t["completion"], "partition", "replay")
			if e != nil {
				return nil, nil, e
			}
			p, e := partitionFromBundle(b)
			if e != nil {
				return nil, nil, e
			}
			if e = knowledgeengine.ValidateSHVPartitionResultVersion("partition_build", p, c["partition"], selected.Version); e != nil {
				return nil, nil, e
			}
			proof, e := jobDecode(c["replay"], "protocol", "bundle_digest", "selected_source_digest", "current_source_digest", "source_is_current", "valid", "digest")
			if e != nil {
				return nil, nil, e
			}
			sealed, e := jobSeal(proof)
			var source map[string]json.RawMessage
			_ = json.Unmarshal(b["source"], &source)
			if e != nil || !refreshEqual(sealed, c["replay"]) || refreshString(proof["protocol"]) != "symphony.qxctl.shv-refresh-verification.v1" || !refreshEqual(proof["bundle_digest"], b["digest"]) || !refreshEqual(proof["selected_source_digest"], source["digest"]) || string(proof["valid"]) != "true" || !regexp.MustCompile(`^sha256:[0-9a-f]{64}$`).MatchString(refreshString(proof["current_source_digest"])) || (string(proof["source_is_current"]) != "true" && string(proof["source_is_current"]) != "false") || (string(proof["source_is_current"]) == "true") != refreshEqual(proof["current_source_digest"], proof["selected_source_digest"]) {
				return nil, nil, fmt.Errorf("retained replay correspondence differs")
			}
		}
	}
	if derived {
		if e = validateJobOrigin(j["origin"], tasks); e != nil {
			return nil, nil, e
		}
	}
	return j, tasks, nil
}
func jobSummary(j map[string]json.RawMessage, tasks []map[string]json.RawMessage, op string, processed int, manifest any, proofs any) (json.RawMessage, error) {
	rows := []any{}
	done := 0
	for _, t := range tasks {
		var pd any
		if string(t["completion"]) != "null" {
			var c map[string]json.RawMessage
			_ = json.Unmarshal(t["completion"], &c)
			var p map[string]json.RawMessage
			_ = json.Unmarshal(c["partition"], &p)
			pd = p["digest"]
			done++
		}
		rows = append(rows, map[string]any{"task_id": t["task_id"], "partition_digest": pd})
	}
	return sealSHVActivation(map[string]any{"protocol": "symphony.qxctl.shv-materialization-result.v1", "operation": op, "job_id": j["job_id"], "checkpoint_digest": j["digest"], "tasks": rows, "completed": done, "pending": len(tasks) - done, "processed": processed, "dependency_validation": map[bool]string{true: "sequential_replay", false: "dependencies_not_replayed"}[op == "export"], "manifest": manifest, "replays": proofs, "catalogue_published": false})
}
func jobMaterialize(t map[string]json.RawMessage, inst knowledgeengine.Installation) (json.RawMessage, error) {
	o, e := comparisonEndpoint(t["endpoint"])
	if e != nil {
		return nil, e
	}
	var proof json.RawMessage
	if e = emitSHVRefresh("verify", o, t["bundle"], func(r json.RawMessage) error { proof = r; return nil }); e != nil {
		return nil, e
	}
	b, e := jobBundle(t["bundle"], o)
	if e != nil {
		return nil, e
	}
	input, e := partitionFromBundle(b)
	if e != nil {
		return nil, e
	}
	cwd, e := os.Getwd()
	if e != nil {
		return nil, e
	}
	r, e := knowledgeengine.InvokeSHVPartition(context.Background(), inst.Prefix, inst.Version, cwd, "partition_build", input)
	if e != nil {
		return nil, e
	}
	return jobMarshal(map[string]any{"partition": r.Result, "replay": proof}), nil
}
func runSHVJob(op string, o jobOptions) error {
	if op == "schema" || op == "template" {
		return jobDiscovery(op)
	}
	store, e := shvjob.New(o.root, o.id)
	if e != nil {
		return e
	}
	var proposed json.RawMessage
	var preparedInstallation knowledgeengine.Installation
	if op == "prepare" {
		inst, e := knowledgeengine.InspectSHVPartition(o.prefix, o.version)
		if e != nil {
			return e
		}
		preparedInstallation = inst
		raw, e := knowledgeengine.ReadPayload(o.input)
		if e != nil {
			return e
		}
		p, e := jobDecode(raw, "tasks", "required_references")
		if e != nil {
			return e
		}
		var ts []map[string]json.RawMessage
		if json.Unmarshal(p["tasks"], &ts) != nil || len(ts) < 1 || len(ts) > 8 {
			return fmt.Errorf("tasks outside bounds")
		}
		for _, t := range ts {
			if _, e = jobDecode(jobMarshal(t), "task_id", "endpoint"); e != nil {
				return e
			}
			ep, e := comparisonEndpoint(t["endpoint"])
			if e != nil {
				return e
			}
			bundle, e := knowledgeengine.ReadPayload(ep.input)
			if e != nil {
				return e
			}
			if _, e = jobBundle(bundle, ep); e != nil {
				return e
			}
			t["bundle"] = bundle
			t["completion"] = json.RawMessage("null")
		}
		proposed, e = sealSHVActivation(map[string]any{"protocol": jobProtocol, "job_id": o.id, "installation": inst, "tasks": ts, "required_references": p["required_references"]})
		if e != nil {
			return e
		}
		if len(proposed) > 3*(1<<18) {
			return fmt.Errorf("job exceeds checkpoint reserve")
		}
		if _, _, e = jobValidate(proposed, o.id); e != nil {
			return e
		}
		projection := []map[string]json.RawMessage{}
		for _, t := range ts {
			ep, _ := comparisonEndpoint(t["endpoint"])
			b, e := jobBundle(t["bundle"], ep)
			if e != nil {
				return e
			}
			pi, e := partitionFromBundle(b)
			if e != nil {
				return e
			}
			part, e := knowledgeengine.ExpectedSHVPartitionVersion("partition_build", pi, inst.Version)
			if e != nil {
				return e
			}
			var source map[string]json.RawMessage
			_ = json.Unmarshal(b["source"], &source)
			proof, e := sealSHVActivation(map[string]any{"protocol": "symphony.qxctl.shv-refresh-verification.v1", "bundle_digest": b["digest"], "selected_source_digest": source["digest"], "current_source_digest": source["digest"], "source_is_current": false, "valid": true})
			if e != nil {
				return e
			}
			copyTask := map[string]json.RawMessage{}
			for k, v := range t {
				copyTask[k] = v
			}
			copyTask["completion"] = jobMarshal(map[string]any{"partition": part, "replay": proof})
			projection = append(projection, copyTask)
		}
		projected, e := sealSHVActivation(map[string]any{"protocol": jobProtocol, "job_id": o.id, "installation": inst, "tasks": projection, "required_references": p["required_references"]})
		if e != nil {
			return e
		}
		if e = knowledgeengine.ValidateSCVBundleText(projected); e != nil {
			return fmt.Errorf("completed job would exceed checkpoint bounds: %w", e)
		}
		cwd, e := os.Getwd()
		if e != nil {
			return e
		}
		_, e = knowledgeengine.InvokeSHVPartition(context.Background(), inst.Prefix, inst.Version, cwd, "manifest_build", jobMarshal(map[string]any{"entries": []any{}, "required_references": p["required_references"]}))
		if e != nil {
			return e
		}
	}
	if (op == "run" || op == "resume") && (o.budget < 1 || o.budget > 8) {
		return fmt.Errorf("max-tasks must be 1..8")
	}
	var output json.RawMessage
	e = store.WithLock(func(tx *shvjob.Transaction) error {
		raw := tx.Read()
		if op == "prepare" {
			if raw == nil {
				if e := tx.Save(proposed, func() error {
					now, err := knowledgeengine.InspectSHVPartition(preparedInstallation.Prefix, preparedInstallation.Version)
					if err != nil {
						return err
					}
					if now != preparedInstallation {
						return fmt.Errorf("job partition installation changed during preparation")
					}
					return nil
				}); e != nil {
					return e
				}
				raw = proposed
			} else {
				old, tasks, e := jobValidate(raw, o.id)
				if e != nil {
					return e
				}
				for _, t := range tasks {
					t["completion"] = json.RawMessage("null")
				}
				old["tasks"] = jobMarshal(tasks)
				base, e := jobSeal(old)
				if e != nil || !refreshEqual(base, proposed) {
					return fmt.Errorf("job ID already binds different intent")
				}
			}
		}
		if raw == nil {
			return fmt.Errorf("unknown materialization job")
		}
		j, tasks, e := jobValidate(raw, o.id)
		if e != nil {
			return e
		}
		var inst knowledgeengine.Installation
		if e = json.Unmarshal(j["installation"], &inst); e != nil {
			return e
		}
		guard := func() error {
			now, e := knowledgeengine.InspectSHVPartition(inst.Prefix, inst.Version)
			if e != nil {
				return e
			}
			if !refreshEqual(now, j["installation"]) {
				return fmt.Errorf("job partition installation changed")
			}
			return nil
		}
		processed := 0
		if op == "run" || op == "resume" {
			if e = guard(); e != nil {
				return e
			}
			for _, t := range tasks {
				if string(t["completion"]) != "null" {
					continue
				}
				if processed >= o.budget {
					break
				}
				completion, e := jobMaterialize(t, inst)
				if e != nil {
					return e
				}
				t["completion"] = completion
				j["tasks"] = jobMarshal(tasks)
				raw, e = jobSeal(j)
				if e != nil {
					return e
				}
				validated, _, err := jobValidate(raw, o.id)
				if err != nil {
					return err
				}
				if e = tx.Save(raw, guard); e != nil {
					return e
				}
				j = validated
				processed++
			}
		}
		var manifest any
		var proofs any
		if op == "export" {
			if e = guard(); e != nil {
				return e
			}
			entries := []any{}
			seen := map[string]bool{}
			replays := []any{}
			for _, t := range tasks {
				if string(t["completion"]) == "null" {
					return fmt.Errorf("job still has pending tasks")
				}
				fresh, e := jobMaterialize(t, inst)
				if e != nil {
					return e
				}
				var f, c map[string]json.RawMessage
				_ = json.Unmarshal(fresh, &f)
				_ = json.Unmarshal(t["completion"], &c)
				if !refreshEqual(f["partition"], c["partition"]) {
					return fmt.Errorf("retained partition differs on replay")
				}
				var part map[string]json.RawMessage
				_ = json.Unmarshal(f["partition"], &part)
				id := refreshString(part["digest"])
				if !seen[id] {
					entries = append(entries, map[string]any{"partition_digest": id, "partition": f["partition"]})
					seen[id] = true
				}
				replays = append(replays, map[string]any{"task_id": t["task_id"], "replay": f["replay"]})
			}
			cwd, e := os.Getwd()
			if e != nil {
				return e
			}
			r, e := knowledgeengine.InvokeSHVPartition(context.Background(), inst.Prefix, inst.Version, cwd, "manifest_build", jobMarshal(map[string]any{"entries": entries, "required_references": j["required_references"]}))
			if e != nil {
				return e
			}
			if e = guard(); e != nil {
				return e
			}
			manifest = r.Result
			proofs = replays
		}
		output, e = jobSummary(j, tasks, op, processed, manifest, proofs)
		return e
	})
	if e != nil {
		return e
	}
	return printIndentedJSON(output)
}
