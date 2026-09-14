package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/spf13/cobra"
)

const shvComparisonProtocol = "symphony.qxctl.shv-refresh-comparison.v1"

var comparisonEndpointKeys = []string{"bundle_path", "source_root", "state_root", "tops_id", "source_id", "source_prefix", "source_version", "kernel_prefix", "kernel_version"}

func comparisonTemplate() map[string]any {
	endpoint := func() map[string]any {
		m := map[string]any{}
		for _, k := range comparisonEndpointKeys {
			m[k] = nil
		}
		return m
	}
	return map[string]any{"previous": endpoint(), "current": endpoint()}
}
func newSHVRefreshCompareCommand() *cobra.Command {
	var input string
	c := &cobra.Command{Use: "compare", Args: usageOnlyArgs, Short: "Compare two exact natively replayed bundles without causal inference", RunE: func(*cobra.Command, []string) error {
		raw, e := knowledgeengine.ReadPayload(input)
		if e != nil {
			return e
		}
		out, e := compareSHVRefresh(raw)
		if e != nil {
			return e
		}
		return printIndentedJSON(out)
	}}
	c.Flags().StringVar(&input, "input", "", "explicit previous/current bundle files and exact engine/source selections")
	_ = c.MarkFlagRequired("input")
	c.Flags().Bool("json", false, "emit structured comparison")
	c.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	s := commandSpec("shv.refresh.compare", featureSHVAdministration, "invoke")
	s.Mutability = "read_only"
	s.InputProtocols = []string{"symphony.qxctl.shv-refresh-comparison-input.v1"}
	s.OutputProtocols = []string{shvComparisonProtocol}
	s.ResultValidationProtocols = s.OutputProtocols
	s.BackendOperationIDs = []string{"engop:symphony:shv-source.capture.import", "engop:symphony:shv-source.graph.project", "engop:symphony:shv.catalogue.build", "engop:symphony:shv.coverage.plan", "engop:symphony:shv.evaluate", "engop:symphony:shv.graph.project"}
	s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:shv-source-engine", Interaction: "invoke"}, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:shv-engine", Interaction: "invoke"})
	commandregistry.Attach(c, s)
	return c
}
func comparisonEndpoint(raw json.RawMessage) (shvRefreshOptions, error) {
	m, e := refreshObject(raw, comparisonEndpointKeys...)
	if e != nil {
		return shvRefreshOptions{}, e
	}
	for _, k := range comparisonEndpointKeys {
		if refreshString(m[k]) == "" {
			return shvRefreshOptions{}, fmt.Errorf("comparison endpoint requires %s", k)
		}
	}
	return shvRefreshOptions{sourcePrefix: refreshString(m["source_prefix"]), sourceVersion: refreshString(m["source_version"]), prefix: refreshString(m["kernel_prefix"]), version: refreshString(m["kernel_version"]), stateRoot: refreshString(m["state_root"]), topsID: refreshString(m["tops_id"]), sourceID: refreshString(m["source_id"]), sourceRoot: refreshString(m["source_root"]), input: refreshString(m["bundle_path"])}, nil
}
func compareSHVRefresh(raw json.RawMessage) (json.RawMessage, error) {
	m, e := refreshObject(raw, "previous", "current")
	if e != nil {
		return nil, e
	}
	previous, e := comparisonEndpoint(m["previous"])
	if e != nil {
		return nil, e
	}
	current, e := comparisonEndpoint(m["current"])
	if e != nil {
		return nil, e
	}
	if previous.topsID != current.topsID || previous.sourceID != current.sourceID {
		return nil, fmt.Errorf("comparison requires same TOPS and logical source; cross-source identity is not inferred")
	}
	bundles := []json.RawMessage{}
	proofs := []json.RawMessage{}
	for _, o := range []shvRefreshOptions{previous, current} {
		b, e := knowledgeengine.ReadPayload(o.input)
		if e != nil {
			return nil, e
		}
		var proof json.RawMessage
		e = emitSHVRefresh("verify", o, b, func(result json.RawMessage) error { proof = append(json.RawMessage(nil), result...); return nil })
		if e != nil {
			return nil, e
		}
		bundles = append(bundles, b)
		proofs = append(proofs, proof)
	}
	out, e := diffSHVRefresh(bundles[0], bundles[1], proofs)
	if e != nil {
		return nil, e
	}
	if e = refreshReplayBound(out); e != nil {
		return nil, e
	}
	return out, nil
}

// These are structural differences in independently replayed owner artifacts,
// not new hardware semantics, compatibility conclusions or causal attribution.
func diffSHVRefresh(previous, current json.RawMessage, proofs []json.RawMessage) (json.RawMessage, error) {
	var a, b map[string]json.RawMessage
	if e := json.Unmarshal(previous, &a); e != nil {
		return nil, e
	}
	if e := json.Unmarshal(current, &b); e != nil {
		return nil, e
	}
	dimensions := []any{}
	add := func(name string, x, y any) error {
		// Wrapper permits hashing arrays/scalars using the same canonical JSON rules.
		dx, e := knowledgeengine.SCVDigest(map[string]any{"value": x})
		if e != nil {
			return e
		}
		dy, e := knowledgeengine.SCVDigest(map[string]any{"value": y})
		if e != nil {
			return e
		}
		dimensions = append(dimensions, map[string]any{"dimension": name, "changed": dx != dy, "previous_digest": dx, "current_digest": dy})
		return nil
	}
	// Normalize RawMessages before digesting; declaration/input key order is not identity.
	normalize := func(raw json.RawMessage) any {
		var v any
		d := json.NewDecoder(bytes.NewReader(raw))
		d.UseNumber()
		_ = d.Decode(&v)
		return v
	}
	for _, k := range []string{"source", "source_installation", "kernel_installation"} {
		if e := add(k, normalize(a[k]), normalize(b[k])); e != nil {
			return nil, e
		}
	}
	var ar, br map[string]json.RawMessage
	_ = json.Unmarshal(a["request"], &ar)
	_ = json.Unmarshal(b["request"], &br)
	for _, k := range []string{"mapping", "profile", "subject_ids", "requirements"} {
		if e := add(k, normalize(ar[k]), normalize(br[k])); e != nil {
			return nil, e
		}
	}
	captureParts := func(raw json.RawMessage) ([]string, []any, []any) {
		var captures []map[string]json.RawMessage
		_ = json.Unmarshal(raw, &captures)
		bodies := []string{}
		manifests := []any{}
		observations := []any{}
		for _, c := range captures {
			var manifest map[string]json.RawMessage
			_ = json.Unmarshal(c["manifest"], &manifest)
			bodies = append(bodies, refreshString(manifest["digest"])+":"+string(manifest["bytes"]))
			manifests = append(manifests, normalize(c["manifest"]))
			delete(c, "source")
			delete(c, "manifest")
			delete(c, "digest")
			v, _ := json.Marshal(c)
			observations = append(observations, normalize(v))
		}
		sort.Strings(bodies)
		return bodies, manifests, observations
	}
	ab, am, ao := captureParts(a["captures"])
	bb, bm, bo := captureParts(b["captures"])
	for _, v := range []struct {
		name string
		x, y any
	}{{"capture_bodies", ab, bb}, {"capture_manifests", am, bm}, {"capture_observations", ao, bo}} {
		if e := add(v.name, v.x, v.y); e != nil {
			return nil, e
		}
	}
	for _, k := range []string{"catalogue", "coverage", "evaluation", "source_graph", "catalogue_graph"} {
		if e := add(k, normalize(a[k]), normalize(b[k])); e != nil {
			return nil, e
		}
	}
	subjects, assertions := comparisonCatalogueRows(a["catalogue"], b["catalogue"])
	return sealSHVActivation(map[string]any{
		"protocol": shvComparisonProtocol, "tops_id": a["tops_id"], "source_id": a["source_id"],
		"previous_bundle_digest": a["digest"], "current_bundle_digest": b["digest"],
		"same_bundle":     refreshEqual(previous, current),
		"previous_replay": proofs[0], "current_replay": proofs[1],
		"observation_scope": "sequential_source_replays", "causal_attribution": "not_inferred",
		"dimensions": dimensions, "subject_changes": subjects, "assertion_changes": assertions,
	})
}

func comparisonCatalogueRows(previous, current json.RawMessage) ([]any, []any) {
	type rows struct {
		metadata   map[string]json.RawMessage
		assertions map[string]json.RawMessage
	}
	read := func(raw json.RawMessage) rows {
		var cat struct {
			Subjects []map[string]json.RawMessage `json:"subjects"`
		}
		_ = json.Unmarshal(raw, &cat)
		r := rows{map[string]json.RawMessage{}, map[string]json.RawMessage{}}
		for _, subject := range cat.Subjects {
			id := refreshString(subject["id"])
			var assertions []map[string]json.RawMessage
			_ = json.Unmarshal(subject["assertions"], &assertions)
			for _, assertion := range assertions {
				key := id + "/" + refreshString(assertion["predicate"])
				r.assertions[key], _ = json.Marshal(assertion)
			}
			delete(subject, "assertions")
			r.metadata[id], _ = json.Marshal(subject)
		}
		return r
	}
	a, b := read(previous), read(current)
	changed := func(left, right map[string]json.RawMessage, assertion bool) []any {
		keys := map[string]bool{}
		for k := range left {
			keys[k] = true
		}
		for k := range right {
			keys[k] = true
		}
		ordered := []string{}
		for k := range keys {
			ordered = append(ordered, k)
		}
		sort.Strings(ordered)
		out := []any{}
		for _, key := range ordered {
			x, xok := left[key]
			y, yok := right[key]
			if xok && yok && refreshEqual(x, y) {
				continue
			}
			kind := "changed"
			if !xok {
				kind = "added"
			}
			if !yok {
				kind = "removed"
			}
			row := map[string]any{"kind": kind, "previous": x, "current": y}
			if assertion {
				parts := strings.SplitN(key, "/", 2)
				row["subject_id"] = parts[0]
				row["predicate"] = parts[1]
			} else {
				row["subject_id"] = key
			}
			out = append(out, row)
		}
		return out
	}
	return changed(a.metadata, b.metadata, false), changed(a.assertions, b.assertions, true)
}
