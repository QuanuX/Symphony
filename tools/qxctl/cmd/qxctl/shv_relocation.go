package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/shvjob"
	"github.com/spf13/cobra"
	"path/filepath"
)

const derivedJobProtocol = "symphony.qxctl.shv-materialization.v2"

//go:embed shv_relocation.schema.json
var relocationSchema json.RawMessage

func relocationRoot(s string) bool { return filepath.IsAbs(s) && filepath.Clean(s) == s && s != "/" }
func validateJobOrigin(raw json.RawMessage, tasks []map[string]json.RawMessage) error {
	o, e := jobDecode(raw, "parent_job_root", "parent_job_id", "parent_checkpoint_digest", "source_roots")
	if e != nil {
		return e
	}
	if !relocationRoot(refreshString(o["parent_job_root"])) || !jobToken.MatchString(refreshString(o["parent_job_id"])) || !relocationHash(refreshString(o["parent_checkpoint_digest"])) {
		return fmt.Errorf("invalid relocation parent")
	}
	var roots []map[string]json.RawMessage
	if json.Unmarshal(o["source_roots"], &roots) != nil || len(roots) != len(tasks) {
		return fmt.Errorf("relocation task lineage differs")
	}
	for i, r := range roots {
		if _, e = jobDecode(jobMarshal(r), "task_id", "from", "to"); e != nil {
			return e
		}
		ep, e := comparisonEndpoint(tasks[i]["endpoint"])
		if e != nil {
			return e
		}
		if !refreshEqual(r["task_id"], tasks[i]["task_id"]) || !relocationRoot(refreshString(r["from"])) || !relocationRoot(refreshString(r["to"])) || refreshString(r["to"]) != ep.sourceRoot || string(tasks[i]["completion"]) == "null" {
			return fmt.Errorf("relocation lineage correspondence differs")
		}
	}
	return nil
}
func relocationHash(s string) bool {
	if len(s) != 71 || s[:7] != "sha256:" {
		return false
	}
	for _, c := range s[7:] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func newSHVRelocationCommand() *cobra.Command {
	root := structural("relocation", fmt.Errorf("relocation operation required"))
	for _, op := range []string{"run", "schema", "template"} {
		var fromRoot, fromID, toRoot, toID, input string
		c := &cobra.Command{Use: op, Args: usageOnlyArgs, Short: "Replay relocated source data into a separate derived job", RunE: func(*cobra.Command, []string) error {
			if op != "run" {
				m := map[string]any{"protocol": "symphony.qxctl.shv-relocation-" + op + ".v1", "origin": "qxctl_embedded"}
				if op == "schema" {
					m["schema"] = relocationSchema
				} else {
					m["template"] = map[string]any{"tasks": []any{map[string]any{"task_id": nil, "source_root": nil}}}
					m["status"] = "unanswered_template_not_validated_input"
				}
				r, e := sealSHVActivation(m)
				if e != nil {
					return e
				}
				return printIndentedJSON(r)
			}
			return runSHVRelocation(fromRoot, fromID, toRoot, toID, input)
		}}
		c.Flags().Bool("json", false, "emit structured evidence")
		if op == "run" {
			for _, f := range []struct {
				name string
				p    *string
			}{{"job-root", &fromRoot}, {"job-id", &fromID}, {"target-job-root", &toRoot}, {"target-job-id", &toID}, {"input", &input}} {
				c.Flags().StringVar(f.p, f.name, "", "explicit relocation selection")
				_ = c.MarkFlagRequired(f.name)
			}
		}
		c.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
		interaction := "discover"
		if op == "run" {
			interaction = "invoke"
		}
		s := commandSpec("shv.materialization.relocation."+op, featureSHVAdministration, interaction)
		s.Mutability = "read_only"
		protocol := "symphony.qxctl.shv-relocation-" + op + ".v1"
		if op == "run" {
			protocol = "symphony.qxctl.shv-relocation.v1"
			s.Mutability = "evidence_only"
			s.InputProtocols = []string{"symphony.qxctl.shv-relocation-input.v1"}
			s.BackendOperationIDs = []string{"engop:symphony:shv-partition.partition.build", "engop:symphony:shv-source.capture.import", "engop:symphony:shv-source.graph.project", "engop:symphony:shv.catalogue.build", "engop:symphony:shv.coverage.plan", "engop:symphony:shv.evaluate", "engop:symphony:shv.graph.project"}
			for _, f := range []string{"shv-partition-engine", "shv-source-engine", "shv-engine"} {
				s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:" + f, Interaction: "invoke"})
			}
		}
		s.OutputProtocols = []string{protocol}
		s.ResultValidationProtocols = s.OutputProtocols
		commandregistry.Attach(c, s)
		root.AddCommand(c)
	}
	return root
}
func runSHVRelocation(fromRoot, fromID, toRoot, toID, input string) error {
	from, e := shvjob.New(fromRoot, fromID)
	if e != nil {
		return e
	}
	to, e := shvjob.New(toRoot, toID)
	if e != nil {
		return e
	}
	if from == to {
		return fmt.Errorf("relocation requires a separate target job")
	}
	raw, e := knowledgeengine.ReadPayload(input)
	if e != nil {
		return e
	}
	request, e := jobDecode(raw, "tasks")
	if e != nil {
		return e
	}
	var moves []map[string]json.RawMessage
	if json.Unmarshal(request["tasks"], &moves) != nil || len(moves) < 1 || len(moves) > 8 {
		return fmt.Errorf("relocation tasks outside bounds")
	}
	var output json.RawMessage
	e = from.WithLock(func(parent *shvjob.Transaction) error {
		j, tasks, e := jobValidate(parent.Read(), fromID)
		if e != nil {
			return e
		}
		if len(moves) != len(tasks) {
			return fmt.Errorf("explicit relocation required for every task in original order")
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
			if now != inst {
				return fmt.Errorf("relocation partition installation changed")
			}
			return nil
		}
		if e = guard(); e != nil {
			return e
		}
		lineage := []any{}
		proofs := []any{}
		for i, t := range tasks {
			move := moves[i]
			if _, e = jobDecode(jobMarshal(move), "task_id", "source_root"); e != nil {
				return e
			}
			destination := refreshString(move["source_root"])
			if !refreshEqual(move["task_id"], t["task_id"]) || !relocationRoot(destination) {
				return fmt.Errorf("relocation task identity or root differs")
			}
			ep, e := jobDecode(t["endpoint"], comparisonEndpointKeys...)
			if e != nil {
				return e
			}
			lineage = append(lineage, map[string]any{"task_id": t["task_id"], "from": ep["source_root"], "to": destination})
			ep["source_root"] = jobMarshal(destination)
			t["endpoint"] = jobMarshal(ep)
			completion, e := jobMaterialize(t, inst)
			if e != nil {
				return e
			}
			var fresh map[string]json.RawMessage
			_ = json.Unmarshal(completion, &fresh)
			if string(t["completion"]) != "null" {
				var old map[string]json.RawMessage
				_ = json.Unmarshal(t["completion"], &old)
				if !refreshEqual(old["partition"], fresh["partition"]) {
					return fmt.Errorf("relocated partition identity differs")
				}
			}
			t["completion"] = completion
			proofs = append(proofs, map[string]any{"task_id": t["task_id"], "replay": fresh["replay"]})
		}
		origin := map[string]any{"parent_job_root": fromRoot, "parent_job_id": fromID, "parent_checkpoint_digest": j["digest"], "source_roots": lineage}
		proposed, e := sealSHVActivation(map[string]any{"protocol": derivedJobProtocol, "job_id": toID, "installation": inst, "tasks": tasks, "required_references": j["required_references"], "origin": origin})
		if e != nil {
			return e
		}
		if _, _, e = jobValidate(proposed, toID); e != nil {
			return e
		}
		var saved json.RawMessage
		e = to.WithLock(func(target *shvjob.Transaction) error {
			saved = target.Read()
			if saved == nil {
				if e := target.Save(proposed, guard); e != nil {
					return e
				}
				saved = proposed
				return nil
			}
			// A retry may observe a newer source head. Compare immutable derivation intent,
			// including partition identity, but preserve the original stored replay proof.
			old, oldTasks, e := jobValidate(saved, toID)
			if e != nil {
				return e
			}
			if refreshString(old["protocol"]) != derivedJobProtocol {
				return fmt.Errorf("target job is not this derivation")
			}
			next, nextTasks, e := jobValidate(proposed, toID)
			if e != nil {
				return e
			}
			for _, list := range [][]map[string]json.RawMessage{oldTasks, nextTasks} {
				for _, t := range list {
					var c map[string]json.RawMessage
					_ = json.Unmarshal(t["completion"], &c)
					c["replay"] = json.RawMessage("null")
					t["completion"] = jobMarshal(c)
				}
			}
			old["tasks"] = jobMarshal(oldTasks)
			next["tasks"] = jobMarshal(nextTasks)
			a, e := jobSeal(old)
			if e != nil {
				return e
			}
			b, e := jobSeal(next)
			if e != nil {
				return e
			}
			if !refreshEqual(a, b) {
				return fmt.Errorf("target job already binds different derivation")
			}
			return guard()
		})
		if e != nil {
			return e
		}
		var child map[string]json.RawMessage
		_ = json.Unmarshal(saved, &child)
		output, e = sealSHVActivation(map[string]any{"protocol": "symphony.qxctl.shv-relocation.v1", "job_id": toID, "checkpoint_digest": child["digest"], "origin": origin, "replays": proofs, "dependency_validation": "sequential_replay", "catalogue_published": false})
		return e
	})
	if e != nil {
		return e
	}
	return printIndentedJSON(output)
}
