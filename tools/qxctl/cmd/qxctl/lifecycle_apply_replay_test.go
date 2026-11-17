package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgelifecycle"
)

// The native fixture performs and replays a real scoped close. The Go boundary
// must accept timestamp-only refresh without rewriting the durable journal.
func TestScopedNativeProducedCloseReplay(t *testing.T) {
	path := os.Getenv("SYMPHONY_SCOPED_APPLY_FIXTURE")
	if path == "" {
		t.Skip("native-produced qualification fixture not requested")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Replay struct {
			Scope   json.RawMessage                 `json:"scope"`
			Desired knowledgelifecycle.DesiredState `json:"desired"`
			Before  knowledgelifecycle.Observation  `json:"observation_before"`
			After   knowledgelifecycle.Observation  `json:"observation_after"`
			Prior   json.RawMessage                 `json:"prior_result"`
			Result  json.RawMessage                 `json:"result"`
			Source  string                          `json:"source_report_journal_digest"`
		} `json:"close_replay"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	record := fixture.Replay
	scope, err := knowledgelifecycle.DecodeExecutionScope(record.Scope)
	if err != nil {
		t.Fatal(err)
	}
	options := knowledgeLifecycleOptions{
		executionScope: &scope, profileID: scope.ProfileID, topsID: scope.TOPSID,
		sourceJournalDigest: record.Source,
	}
	prior, err := validateLifecycleApplyResult(record.Prior, "lifecycle_apply_close", scope.ProfileID,
		scope.TOPSID, record.Desired.DesiredStateDigest, record.Before.ObservationDigest, "")
	if err != nil {
		t.Fatal(err)
	}
	stable, err := knowledgelifecycle.StableInventoryDigest(record.After)
	if err != nil {
		t.Fatal(err)
	}
	beforeStable, err := knowledgelifecycle.StableInventoryDigest(record.Before)
	if err != nil || beforeStable != stable || record.Before.ObservationDigest == record.After.ObservationDigest {
		t.Fatalf("native fixture does not isolate timestamp-only refresh: %v", err)
	}
	if _, err := validateLifecycleApplyResult(record.Result, "lifecycle_apply_close", scope.ProfileID,
		scope.TOPSID, record.Desired.DesiredStateDigest, record.After.ObservationDigest, ""); err == nil {
		t.Fatal("general result validation relaxed the observation binding")
	}
	validate := func(raw json.RawMessage, opts knowledgeLifecycleOptions, operation, journal, applied, profile, inventory string,
		observed knowledgelifecycle.Observation) (validatedLifecycleApplyResult, error) {
		return validateLifecycleApplyMutationResult(raw, opts, operation, journal, applied, profile, inventory,
			record.Desired, observed, "")
	}
	result, err := validate(record.Result, options, "lifecycle_apply_close", prior.JournalDigest, "absent",
		scope.ProfileDigest, stable, record.After)
	if err != nil {
		t.Fatalf("native completed scoped close replay was rejected: %v", err)
	}
	if result.Changed || result.Recovered || !result.ScopeComplete || result.ProfileConverged ||
		result.JournalDigest != prior.JournalDigest || result.AppliedDigest != prior.AppliedDigest || result.Plan == nil ||
		!lifecycleJSONEqual(result.Raw.(map[string]any)["journal"], prior.Raw.(map[string]any)["journal"]) {
		t.Fatal("completed replay changed or weakened the durable completion evidence")
	}
	for _, test := range []struct {
		name                      string
		journal, applied, profile string
		inventory, source         string
	}{
		{"stale journal", lifecycleTestDigest("stale"), "absent", scope.ProfileDigest, stable, record.Source},
		{"absent journal", "absent", "absent", scope.ProfileDigest, stable, record.Source},
		{"stale applied state", prior.JournalDigest, lifecycleTestDigest("stale"), scope.ProfileDigest, stable, record.Source},
		{"different profile", prior.JournalDigest, "absent", lifecycleTestDigest("other"), stable, record.Source},
		{"different inventory", prior.JournalDigest, "absent", scope.ProfileDigest, lifecycleTestDigest("other"), record.Source},
		{"different source", prior.JournalDigest, "absent", scope.ProfileDigest, stable, lifecycleTestDigest("other")},
	} {
		t.Run(test.name, func(t *testing.T) {
			opts := options
			opts.sourceJournalDigest = test.source
			if _, err := validate(record.Result, opts, "lifecycle_apply_close", test.journal, test.applied,
				test.profile, test.inventory, record.After); err == nil {
				t.Fatal("completed replay accepted different requested durable evidence")
			}
		})
	}
	t.Run("observation inventory drift", func(t *testing.T) {
		observed := record.After
		drift := lifecycleTestDigest("new-registry")
		observed.BindingRegistryDigest = &drift
		if _, err := validate(record.Result, options, "lifecycle_apply_close", prior.JournalDigest, "absent",
			scope.ProfileDigest, stable, observed); err == nil {
			t.Fatal("completed replay accepted observation changes hidden behind an old digest")
		}
	})
	t.Run("different scope", func(t *testing.T) {
		other := scope
		other.ScopeDigest = lifecycleTestDigest("other-scope")
		opts := options
		opts.executionScope = &other
		if _, err := validate(record.Result, opts, "lifecycle_apply_close", prior.JournalDigest, "absent",
			scope.ProfileDigest, stable, record.After); err == nil {
			t.Fatal("completed replay accepted another execution scope")
		}
	})
	for _, name := range []string{"changed", "incomplete", "missing plan", "stale plan", "repair actions", "prepare"} {
		t.Run(name, func(t *testing.T) {
			var changed map[string]any
			if err := json.Unmarshal(record.Result, &changed); err != nil {
				t.Fatal(err)
			}
			operation := "lifecycle_apply_close"
			switch name {
			case "changed":
				changed["changed"] = true
			case "incomplete":
				changed["scope_complete"] = false
			case "missing plan":
				changed["plan"] = nil
			case "stale plan":
				plan := changed["plan"].(map[string]any)
				plan["observation_digest"] = record.Before.ObservationDigest
				plan["plan_digest"] = lifecyclePlanDigest(t, plan)
			case "repair actions":
				changed["repair_actions"] = []any{"unexpected repair"}
			case "prepare":
				operation = "lifecycle_apply_prepare"
				changed["operation"] = operation
			}
			raw, err := json.Marshal(changed)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := validate(raw, options, operation, prior.JournalDigest, "absent",
				scope.ProfileDigest, stable, record.After); err == nil {
				t.Fatal("completed close replay exception escaped its exact result boundary")
			}
		})
	}
}
