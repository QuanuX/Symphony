package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvcorpus"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvtransport"
	"github.com/spf13/cobra"
	"sort"
	"strings"
	"time"
)

const corpusBudget = 120 * time.Second
const corpusMaxAncestry = 128
const corpusMaxVerifiedObjects = 4096

func newSCVCorpusCommand() *cobra.Command {
	group := structural("corpus", fmt.Errorf("corpus subcommand is required: acquire, import, recover, inspect, query, export, diff"))
	for _, leaf := range []string{"acquire", "import", "recover", "inspect", "query", "export", "diff"} {
		options := scvOptions{}
		var root string
		child := &cobra.Command{Use: leaf, Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error { return runSCVCorpus(leaf, options, root) }}
		scvFlagsVersion(child, &options, "0.2.0-dev")
		child.Flags().StringVar(&root, "corpus-root", "", "explicit owned local immutable evidence root")
		child.Flags().StringVar(&options.topsID, "tops-id", "", "exact TOPS UUID for local evidence namespace")
		interaction := map[string]string{"acquire": "invoke", "import": "invoke", "recover": "recover", "inspect": "inspect", "query": "query", "export": "query", "diff": "validate"}[leaf]
		spec := commandSpec("scv.corpus."+leaf, featureSCVAdministration, interaction)
		spec.Mutability = "read_only"
		if leaf == "acquire" || leaf == "import" || leaf == "recover" || leaf == "diff" {
			spec.Mutability = "evidence_only"
		}
		spec.InputProtocols = []string{"symphony.qxctl.scv-corpus-" + leaf + "-input.v1"}
		output := "symphony.qxctl.scv-corpus-result.v1"
		switch leaf {
		case "inspect":
			output = "symphony.scv.corpus.v1"
		case "query":
			output = "symphony.scv.corpus-query.v1"
		case "export":
			output = "symphony.qxctl.scv-corpus-export.v1"
		case "diff":
			output = "symphony.scv.corpus-diff.v1"
		}
		spec.OutputProtocols = []string{output}
		spec.ResultValidationProtocols = spec.OutputProtocols
		bindings := map[string]string{"capture_index": "invoke", "corpus_build": "invoke"}
		if leaf == "acquire" || leaf == "recover" {
			bindings["source_status"] = "inspect"
			bindings["capture_import"] = "invoke"
		}
		if leaf == "export" || leaf == "query" {
			bindings["corpus_query"] = "query"
		}
		if leaf == "diff" {
			bindings["corpus_diff"] = "validate"
		}
		ops := []string{}
		for op := range bindings {
			ops = append(ops, op)
		}
		sort.Strings(ops)
		for _, domain := range knowledgeengine.SCVDomains() {
			seen := map[string]bool{}
			for _, op := range ops {
				spec.BackendOperationIDs = append(spec.BackendOperationIDs, "engop:symphony:"+domain+"."+strings.ReplaceAll(op, "_", "."))
				if !seen[bindings[op]] {
					spec.FeatureBindings = append(spec.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:" + domain + "-engine", Interaction: bindings[op]})
					seen[bindings[op]] = true
				}
			}
		}
		if leaf == "acquire" || leaf == "import" || leaf == "recover" {
			spec.RecoveryCommandID = stringPointer("qxcmd:symphony:scv.corpus.recover")
		}
		commandregistry.Attach(child, spec)
		group.AddCommand(child)
	}
	return group
}

type corpusRunner struct {
	options      scvOptions
	installation knowledgeengine.Installation
	owner        func(string, any) (json.RawMessage, error)
	acquire      func(context.Context, scvtransport.Input) (scvtransport.CaptureInput, error)
	now          func() time.Time
	budget       time.Duration
}

func newCorpusRunner(options scvOptions) (*corpusRunner, error) {
	if options.version != "0.2.0-dev" && options.version != "0.3.0-dev" && options.version != "0.4.0-dev" {
		return nil, fmt.Errorf("corpus commands require exact supported 0.2.0-dev, 0.3.0-dev or 0.4.0-dev engine")
	}
	installed, err := knowledgeengine.InspectSCVDomain(options.domain, options.prefix, options.version)
	if err != nil {
		return nil, err
	}
	runner := &corpusRunner{options: options, installation: installed, acquire: scvtransport.Acquire, now: time.Now, budget: corpusBudget}
	runner.owner = func(op string, input any) (json.RawMessage, error) {
		current, err := knowledgeengine.InspectSCVDomain(options.domain, options.prefix, options.version)
		if err != nil || current != installed {
			return nil, fmt.Errorf("exact corpus owner installation changed")
		}
		return invokeSCV(options, op, input)
	}
	return runner, nil
}
func runSCVCorpus(operation string, options scvOptions, root string) error {
	if root == "" {
		return fmt.Errorf("--corpus-root is required; no implicit cache or selected head")
	}
	store, err := scvcorpus.New(root, options.topsID, options.domain)
	if err != nil {
		return err
	}
	input, err := scvInput(options)
	if err != nil {
		return err
	}
	runner, err := newCorpusRunner(options)
	if err != nil {
		return err
	}
	result, err := runner.execute(store, operation, input)
	if err != nil {
		return err
	}
	return outputSCV(options, result)
}
func exactCorpusFields(value map[string]any, names ...string) error {
	if len(value) != len(names) {
		return fmt.Errorf("unexpected corpus input fields")
	}
	for _, name := range names {
		if _, ok := value[name]; !ok {
			return fmt.Errorf("missing corpus input field %s", name)
		}
	}
	return nil
}
func corpusString(value any, label string) (string, error) {
	s, ok := value.(string)
	if !ok || !scvcorpus.ValidID(s) {
		return "", fmt.Errorf("%s must be a nonempty bounded opaque ID", label)
	}
	return s, nil
}
func corpusDigest(value any, label string) (string, error) {
	s, ok := value.(string)
	if !ok || !scvcorpus.ValidDigest(s) {
		return "", fmt.Errorf("%s must be an exact digest", label)
	}
	return s, nil
}
func corpusObject(value any) (map[string]any, error) {
	raw, err := knowledgeengine.SCVCanonical(value)
	if err != nil {
		return nil, err
	}
	return scvcorpus.Decode(raw)
}
func corpusRaw(value any) (json.RawMessage, error) {
	raw, err := knowledgeengine.SCVCanonical(value)
	if err != nil {
		return nil, err
	}
	if err = knowledgeengine.ValidateJSONObject(raw, scvcorpus.MaxArtifactBytes); err != nil {
		return nil, err
	}
	return raw, nil
}

func (r *corpusRunner) validateInput(mode string, input map[string]any) error {
	if err := exactCorpusFields(input, "operation_id", "corpus_id", "previous_snapshot_digest", "members"); err != nil {
		return err
	}
	if _, err := corpusString(input["operation_id"], "operation_id"); err != nil {
		return err
	}
	if _, err := corpusString(input["corpus_id"], "corpus_id"); err != nil {
		return err
	}
	if input["previous_snapshot_digest"] != nil {
		if _, err := corpusDigest(input["previous_snapshot_digest"], "previous_snapshot_digest"); err != nil {
			return err
		}
	}
	members, ok := input["members"].([]any)
	if !ok || len(members) > scvcorpus.MaxMembers {
		return fmt.Errorf("corpus members must be an array of at most 128")
	}
	seen, tuples := map[string]bool{}, map[string]bool{}
	sourceOwners := map[string]string{}
	for _, raw := range members {
		member, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid corpus member")
		}
		id, err := corpusString(member["member_id"], "member_id")
		if err != nil || seen[id] {
			return fmt.Errorf("invalid or duplicate corpus member ID")
		}
		seen[id] = true
		var source map[string]any
		var locator string
		if mode == "acquire" {
			if err = exactCorpusFields(member, "member_id", "source", "locator_id"); err != nil {
				return err
			}
			source, ok = member["source"].(map[string]any)
			if !ok {
				return fmt.Errorf("member source must be an object")
			}
			locator, err = corpusString(member["locator_id"], "locator_id")
			if err != nil {
				return err
			}
			if _, err = r.owner("source_status", map[string]any{"source": source}); err != nil {
				return fmt.Errorf("member %s source: %w", id, err)
			}
			locators, ok := source["locators"].([]any)
			if !ok {
				return fmt.Errorf("member lacks source locators")
			}
			found := false
			for _, raw := range locators {
				item, ok := raw.(map[string]any)
				if ok && item["locator_id"] == locator {
					uri, ok := item["uri"].(string)
					if !ok {
						return fmt.Errorf("invalid source URI")
					}
					if _, err = scvtransport.ValidateURL(uri); err != nil {
						return fmt.Errorf("member %s: %w", id, err)
					}
					found = true
				}
			}
			if !found {
				return fmt.Errorf("member %s locator is absent", id)
			}
		} else {
			if err = exactCorpusFields(member, "member_id", "capture"); err != nil {
				return err
			}
			capture, ok := member["capture"].(map[string]any)
			if !ok {
				return fmt.Errorf("member capture must be an object")
			}
			if _, err = r.owner("capture_index", map[string]any{"capture": capture}); err != nil {
				return fmt.Errorf("member %s capture: %w", id, err)
			}
			source, ok = capture["source"].(map[string]any)
			if !ok {
				return fmt.Errorf("capture source missing")
			}
			locator, _ = capture["locator_id"].(string)
		}
		tuple, _ := knowledgeengine.SCVDigest(map[string]any{"family_id": source["family_id"], "provider_id": source["provider_id"], "source_id": source["source_id"], "locator_id": locator})
		sourceID, _ := source["source_id"].(string)
		owner, _ := knowledgeengine.SCVDigest(map[string]any{"family_id": source["family_id"], "provider_id": source["provider_id"]})
		if previous, ok := sourceOwners[sourceID]; ok && previous != owner {
			return fmt.Errorf("source identity has conflicting provider/family owners")
		}
		sourceOwners[sourceID] = owner
		if tuples[tuple] {
			return fmt.Errorf("duplicate corpus source/locator tuple")
		}
		tuples[tuple] = true
	}
	sort.Slice(members, func(i, j int) bool {
		return members[i].(map[string]any)["member_id"].(string) < members[j].(map[string]any)["member_id"].(string)
	})
	return nil
}

type corpusVerifier struct {
	runner    *corpusRunner
	session   *scvcorpus.Session
	snapshots map[string]json.RawMessage
	indexes   map[string]json.RawMessage
	visiting  map[string]bool
	objects   int
}

func (r *corpusRunner) verifier(s *scvcorpus.Session) *corpusVerifier {
	return &corpusVerifier{runner: r, session: s, snapshots: map[string]json.RawMessage{}, indexes: map[string]json.RawMessage{}, visiting: map[string]bool{}}
}
func (v *corpusVerifier) index(value any) (json.RawMessage, error) {
	index, err := corpusObject(value)
	if err != nil {
		return nil, err
	}
	digest, err := corpusDigest(index["digest"], "index digest")
	if err != nil {
		return nil, err
	}
	if raw, ok := v.indexes[digest]; ok {
		if !scvcorpus.SameRaw(raw, mustCorpusRaw(index)) {
			return nil, fmt.Errorf("embedded index differs %s", digest)
		}
		return raw, nil
	}
	v.objects++
	if v.objects > corpusMaxVerifiedObjects {
		return nil, fmt.Errorf("corpus verification exceeds 4096 unique object pairs")
	}
	stored, err := v.session.Get("indexes", digest)
	if err != nil {
		return nil, err
	}
	if !scvcorpus.SameRaw(stored, mustCorpusRaw(index)) {
		return nil, fmt.Errorf("embedded index differs from retained %s", digest)
	}
	captureDigest, err := corpusDigest(index["capture_digest"], "capture digest")
	if err != nil {
		return nil, err
	}
	capture, err := v.session.Get("captures", captureDigest)
	if err != nil {
		return nil, err
	}
	regenerated, err := v.runner.owner("capture_index", map[string]any{"capture": json.RawMessage(capture)})
	if err != nil {
		return nil, fmt.Errorf("retained capture %s: %w", captureDigest, err)
	}
	if !scvcorpus.SameRaw(regenerated, stored) {
		return nil, fmt.Errorf("capture/index projection mismatch %s", digest)
	}
	v.indexes[digest] = stored
	return stored, nil
}
func mustCorpusRaw(value any) []byte { raw, _ := knowledgeengine.SCVCanonical(value); return raw }
func (v *corpusVerifier) snapshot(digest string, depth int) (json.RawMessage, error) {
	if raw, ok := v.snapshots[digest]; ok {
		// A cached root may be reused as a deeper predecessor in a diff. Its
		// previously verified full chain still consumes the caller's depth budget.
		value, err := scvcorpus.Decode(raw)
		if err != nil {
			return nil, err
		}
		generation, ok := value["generation"].(json.Number)
		if !ok {
			return nil, fmt.Errorf("invalid cached corpus generation")
		}
		height, err := generation.Int64()
		if err != nil || height < 1 || height+int64(depth) > corpusMaxAncestry {
			return nil, fmt.Errorf("corpus ancestry exceeds 128 at cached %s", digest)
		}
		return raw, nil
	}
	if depth >= corpusMaxAncestry || v.visiting[digest] {
		return nil, fmt.Errorf("corpus ancestry exceeds 128 or cycles at %s", digest)
	}
	v.visiting[digest] = true
	defer delete(v.visiting, digest)
	raw, err := v.session.Get("snapshots", digest)
	if err != nil {
		return nil, err
	}
	snapshot, err := scvcorpus.Decode(raw)
	if err != nil {
		return nil, err
	}
	var previous any
	if snapshot["parent_digest"] != nil {
		parent, err := corpusDigest(snapshot["parent_digest"], "parent digest")
		if err != nil {
			return nil, err
		}
		previous, err = v.snapshot(parent, depth+1)
		if err != nil {
			return nil, err
		}
	}
	members, ok := snapshot["members"].([]any)
	if !ok || len(members) > scvcorpus.MaxMembers {
		return nil, fmt.Errorf("invalid retained corpus member count")
	}
	attempts := []any{}
	for _, raw := range members {
		member, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid retained corpus member")
		}
		latest, err := v.index(member["latest_attempt"])
		if err != nil {
			return nil, err
		}
		if member["last_complete"] != nil {
			if _, err = v.index(member["last_complete"]); err != nil {
				return nil, err
			}
		}
		attempts = append(attempts, map[string]any{"member_id": member["member_id"], "capture": latest})
	}
	rebuilt, err := v.runner.owner("corpus_build", map[string]any{"corpus_id": snapshot["corpus_id"], "previous": previous, "snapshot_time": snapshot["snapshot_time"], "attempts": attempts})
	if err != nil {
		return nil, fmt.Errorf("retained snapshot %s: %w", digest, err)
	}
	if !scvcorpus.SameRaw(rebuilt, raw) {
		return nil, fmt.Errorf("snapshot cannot be reconstructed %s", digest)
	}
	v.snapshots[digest] = raw
	return raw, nil
}

func (r *corpusRunner) execute(store scvcorpus.Store, operation string, input map[string]any) (json.RawMessage, error) {
	if operation == "acquire" || operation == "import" || operation == "recover" {
		return r.job(store, operation, input)
	}
	var result json.RawMessage
	err := store.WithRead(func(s *scvcorpus.Session, _ *scvcorpus.Job) error {
		verifier := r.verifier(s)
		if operation == "diff" {
			if err := exactCorpusFields(input, "before_snapshot_digest", "after_snapshot_digest"); err != nil {
				return err
			}
			before, err := corpusDigest(input["before_snapshot_digest"], "before snapshot")
			if err != nil {
				return err
			}
			after, err := corpusDigest(input["after_snapshot_digest"], "after snapshot")
			if err != nil {
				return err
			}
			a, err := verifier.snapshot(before, 0)
			if err != nil {
				return err
			}
			b, err := verifier.snapshot(after, 0)
			if err != nil {
				return err
			}
			result, err = r.owner("corpus_diff", map[string]any{"before": a, "after": b})
			return err
		}
		if operation == "inspect" {
			if err := exactCorpusFields(input, "snapshot_digest"); err != nil {
				return err
			}
		} else if operation == "export" || operation == "query" {
			if err := exactCorpusFields(input, "snapshot_digest", "member_ids", "selection", "query_time", "max_age_seconds"); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("unknown corpus operation")
		}
		digest, err := corpusDigest(input["snapshot_digest"], "snapshot digest")
		if err != nil {
			return err
		}
		snapshot, err := verifier.snapshot(digest, 0)
		if err != nil {
			return err
		}
		if operation == "inspect" {
			result = snapshot
			return nil
		}
		query, err := r.owner("corpus_query", map[string]any{"corpus": snapshot, "member_ids": input["member_ids"], "selection": input["selection"], "query_time": input["query_time"], "max_age_seconds": input["max_age_seconds"]})
		if err != nil {
			return err
		}
		if operation == "query" {
			result = query
			return nil
		}
		value, err := scvcorpus.Decode(query)
		if err != nil {
			return err
		}
		members, ok := value["members"].([]any)
		if !ok {
			return fmt.Errorf("invalid owner query members")
		}
		captures := []any{}
		for _, raw := range members {
			member := raw.(map[string]any)
			if member["selected"] == nil {
				continue
			}
			index, err := corpusObject(member["selected"])
			if err != nil {
				return err
			}
			if _, err = verifier.index(index); err != nil {
				return err
			}
			capture, err := s.Get("captures", index["capture_digest"].(string))
			if err != nil {
				return err
			}
			captures = append(captures, json.RawMessage(capture))
			if len(captures) > 16 {
				return fmt.Errorf("export exceeds 16 captures; select a smaller explicit member subset")
			}
		}
		// The capture-bearing input itself must fit the existing process frame and
		// value limits. Added interpretation claims/policy remain independently bounded.
		materialized, err := knowledgeengine.SCVCanonical(map[string]any{"captures": captures})
		if err != nil {
			return err
		}
		if err = knowledgeengine.ValidateJSONObject(materialized, scvcorpus.MaxArtifactBytes); err != nil {
			return fmt.Errorf("export capture input exceeds process bounds: %w", err)
		}
		result, err = knowledgeengine.SCVCanonical(map[string]any{"protocol": "symphony.qxctl.scv-corpus-export.v1", "query": query, "captures": captures})
		if err != nil {
			return err
		}
		return knowledgeengine.ValidateJSONObject(result, 4<<20)
	})
	return result, err
}

func (r *corpusRunner) job(store scvcorpus.Store, operation string, input map[string]any) (json.RawMessage, error) {
	var proposed scvcorpus.Job
	var err error
	if operation == "recover" {
		if err = exactCorpusFields(input, "operation_id"); err != nil {
			return nil, err
		}
	} else {
		if err = r.validateInput(operation, input); err != nil {
			return nil, err
		}
		raw, err := corpusRaw(input)
		if err != nil {
			return nil, err
		}
		proposed, err = scvcorpus.NewJob(input["operation_id"].(string), operation, r.installation, raw)
		if err != nil {
			return nil, err
		}
	}
	operationID, err := corpusString(input["operation_id"], "operation_id")
	if err != nil {
		return nil, err
	}
	var result json.RawMessage
	err = store.With(operationID, func(s *scvcorpus.Session, existing *scvcorpus.Job) error {
		job := proposed
		if existing != nil {
			job = *existing
			if job.Installation != r.installation {
				return fmt.Errorf("corpus recovery requires original exact installation")
			}
			if operation != "recover" && job.IntentDigest != proposed.IntentDigest {
				return fmt.Errorf("operation ID already binds a different exact corpus intent")
			}
		} else if operation == "recover" {
			return fmt.Errorf("corpus operation not found")
		}
		pinned, err := scvcorpus.Decode(job.Input)
		if err != nil {
			return err
		}
		if err = r.validateInput(job.Mode, pinned); err != nil {
			return err
		}
		verifier := r.verifier(s)
		var previous any
		if pinned["previous_snapshot_digest"] != nil {
			previous, err = verifier.snapshot(pinned["previous_snapshot_digest"].(string), 1)
			if err != nil {
				return err
			}
			previousValue, err := corpusObject(previous)
			if err != nil {
				return err
			}
			if err = corpusPredecessorBindsInput(job.Mode, pinned, previousValue); err != nil {
				return err
			}
			if existing == nil {
				if err = corpusObservationNotFuture(previousValue["snapshot_time"], r.now()); err != nil {
					return fmt.Errorf("predecessor: %w", err)
				}
			}
		}
		if existing == nil {
			if job.Mode == "import" {
				for _, raw := range pinned["members"].([]any) {
					member := raw.(map[string]any)
					capture := member["capture"].(map[string]any)
					if err = corpusObservationNotFuture(capture["observed_at"], r.now()); err != nil {
						return fmt.Errorf("member %s: %w", member["member_id"], err)
					}
				}
			}
			job, err = s.Save(job)
			if err != nil {
				return err
			}
		}
		// The 120s scheduling/network deadline excludes preflight validation and
		// one bounded owner-validation/checkpoint tail for the final retrieval.
		ctx, cancel := context.WithTimeout(context.Background(), r.budget)
		defer cancel()
		members := pinned["members"].([]any)
		for _, raw := range members {
			member := raw.(map[string]any)
			id := member["member_id"].(string)
			if checkpoint, ok := job.Completed[id]; ok {
				index, err := s.Get("indexes", checkpoint.IndexDigest)
				if err != nil {
					return err
				}
				value, err := scvcorpus.Decode(index)
				if err != nil {
					return err
				}
				if value["capture_digest"] != checkpoint.CaptureDigest {
					return fmt.Errorf("checkpoint capture mismatch")
				}
				if _, err = verifier.index(value); err != nil {
					return err
				}
				if err = checkpointBindsMember(job.Mode, member, value); err != nil {
					return err
				}
				continue
			}
			if ctx.Err() != nil {
				return fmt.Errorf("corpus operation %s retained %d/%d members; scheduling budget exhausted, recover exact operation", operationID, len(job.Completed), len(members))
			}
			var capture json.RawMessage
			if job.Mode == "import" {
				capture, err = corpusRaw(member["capture"])
			} else {
				source, encodeErr := corpusRaw(member["source"])
				if encodeErr != nil {
					return encodeErr
				}
				acquired, acquireErr := r.acquire(ctx, scvtransport.Input{Source: source, LocatorID: member["locator_id"].(string)})
				if acquireErr != nil {
					return fmt.Errorf("corpus member %s remains unfinished: %w", id, acquireErr)
				}
				capture, err = r.owner("capture_import", acquired)
			}
			if err != nil {
				return err
			}
			index, err := r.owner("capture_index", map[string]any{"capture": capture})
			if err != nil {
				return err
			}
			captureDigest, err := s.Put("captures", capture)
			if err != nil {
				return err
			}
			indexDigest, err := s.Put("indexes", index)
			if err != nil {
				return err
			}
			completed := make(map[string]scvcorpus.Checkpoint, len(job.Completed)+1)
			for k, v := range job.Completed {
				completed[k] = v
			}
			completed[id] = scvcorpus.Checkpoint{CaptureDigest: captureDigest, IndexDigest: indexDigest}
			next := job
			next.Completed = completed
			job, err = s.Save(next)
			if err != nil {
				return err
			}
		}
		attempts := []any{}
		for _, raw := range members {
			member := raw.(map[string]any)
			checkpoint := job.Completed[member["member_id"].(string)]
			index, err := s.Get("indexes", checkpoint.IndexDigest)
			if err != nil {
				return err
			}
			value, err := scvcorpus.Decode(index)
			if err != nil {
				return err
			}
			if _, err = verifier.index(value); err != nil {
				return err
			}
			attempts = append(attempts, map[string]any{"member_id": member["member_id"], "capture": json.RawMessage(index)})
		}
		if job.SnapshotTime == "" {
			next := job
			now := r.now().UTC().Truncate(time.Second)
			for _, raw := range attempts {
				attempt := raw.(map[string]any)
				index, err := corpusObject(attempt["capture"])
				if err != nil {
					return err
				}
				if err = corpusObservationNotFuture(index["observed_at"], now); err != nil {
					return fmt.Errorf("attempts retained; recover after clock ordering is restored: %w", err)
				}
			}
			if previous != nil {
				value, err := corpusObject(previous)
				if err != nil {
					return err
				}
				if err = corpusObservationNotFuture(value["snapshot_time"], now); err != nil {
					return fmt.Errorf("attempts retained; recover after clock ordering is restored: %w", err)
				}
			}
			next.SnapshotTime = now.Format(time.RFC3339)
			job, err = s.Save(next)
			if err != nil {
				return err
			}
		}
		snapshot, err := r.owner("corpus_build", map[string]any{"corpus_id": pinned["corpus_id"], "previous": previous, "snapshot_time": job.SnapshotTime, "attempts": attempts})
		if err != nil {
			return fmt.Errorf("corpus attempts retained; snapshot build failed: %w", err)
		}
		if job.SnapshotDigest != "" {
			retained, err := verifier.snapshot(job.SnapshotDigest, 0)
			if err != nil {
				return err
			}
			if !scvcorpus.SameRaw(snapshot, retained) {
				return fmt.Errorf("completed snapshot does not bind exact retained job intent/checkpoints")
			}
			result, err = corpusJobResult(job, retained)
			return err
		}
		digest, err := s.Put("snapshots", snapshot)
		if err != nil {
			return err
		}
		next := job
		next.SnapshotDigest = digest
		job, err = s.Save(next)
		if err != nil {
			return err
		}
		result, err = corpusJobResult(job, snapshot)
		return err
	})
	return result, err
}
func checkpointBindsMember(mode string, member, index map[string]any) error {
	if mode == "import" {
		capture, ok := member["capture"].(map[string]any)
		if !ok || !scvcorpus.Same(index["capture_digest"], capture["digest"]) {
			return fmt.Errorf("checkpoint changed pinned imported capture")
		}
	} else {
		source, ok := member["source"].(map[string]any)
		if !ok || !scvcorpus.Same(index["source_digest"], source["digest"]) || !scvcorpus.Same(index["locator_id"], member["locator_id"]) {
			return fmt.Errorf("checkpoint changed pinned source/locator")
		}
	}
	return nil
}
func corpusJobResult(job scvcorpus.Job, snapshot json.RawMessage) (json.RawMessage, error) {
	return knowledgeengine.SCVCanonical(map[string]any{"protocol": "symphony.qxctl.scv-corpus-result.v1", "operation_id": job.OperationID, "intent_digest": job.IntentDigest, "snapshot_digest": job.SnapshotDigest, "snapshot": snapshot, "completed_members": len(job.Completed)})
}

func corpusObservationNotFuture(value any, now time.Time) error {
	text, ok := value.(string)
	observed, err := time.Parse("2006-01-02T15:04:05Z", text)
	if !ok || err != nil || len(text) != 20 {
		return fmt.Errorf("invalid evidence timestamp")
	}
	if observed.After(now.UTC().Truncate(time.Second)) {
		return fmt.Errorf("evidence timestamp is ahead of local snapshot clock")
	}
	return nil
}

func corpusPredecessorBindsInput(mode string, input, previous map[string]any) error {
	if previous["corpus_id"] != input["corpus_id"] {
		return fmt.Errorf("explicit predecessor belongs to a different corpus ID")
	}
	old := map[string]map[string]any{}
	for _, raw := range previous["members"].([]any) {
		member := raw.(map[string]any)
		old[member["member_id"].(string)] = member["latest_attempt"].(map[string]any)
	}
	for _, raw := range input["members"].([]any) {
		member := raw.(map[string]any)
		prior, retained := old[member["member_id"].(string)]
		if !retained {
			continue
		}
		var source map[string]any
		locator := member["locator_id"]
		if mode == "import" {
			capture := member["capture"].(map[string]any)
			source = capture["source"].(map[string]any)
			locator = capture["locator_id"]
		} else {
			source = member["source"].(map[string]any)
		}
		for _, field := range []string{"family_id", "provider_id", "source_id"} {
			if !scvcorpus.Same(prior[field], source[field]) {
				return fmt.Errorf("retained member ID cannot change source/locator tuple")
			}
		}
		if !scvcorpus.Same(prior["locator_id"], locator) {
			return fmt.Errorf("retained member ID cannot change source/locator tuple")
		}
	}
	return nil
}
