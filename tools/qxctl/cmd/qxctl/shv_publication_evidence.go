package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"os"
)

// Rebuild the publication candidate from explicit retained references. Neither a
// caller-provided seal nor a stored graph substitutes for original owner replay.
func publicationEvidence(raw json.RawMessage, tops, catalogue string) (json.RawMessage, error) {
	in, e := refreshObject(raw, "manifest", "policy", "partition_prefix", "partition_version", "members")
	if e != nil {
		return nil, e
	}
	prefix, version := refreshString(in["partition_prefix"]), refreshString(in["partition_version"])
	if version != "0.2.0-dev" {
		return nil, fmt.Errorf("publication requires exact partition contract 0.2.0-dev")
	}
	pi, e := knowledgeengine.InspectSHVPartition(prefix, version)
	if e != nil {
		return nil, e
	}
	cwd, e := os.Getwd()
	if e != nil {
		return nil, e
	}
	var manifest map[string]json.RawMessage
	if e = json.Unmarshal(in["manifest"], &manifest); e != nil {
		return nil, e
	}
	payload, e := knowledgeengine.SCVCanonical(map[string]any{"entries": manifest["entries"], "required_references": manifest["required_references"]})
	if e != nil {
		return nil, e
	}
	built, e := knowledgeengine.InvokeSHVPartition(context.Background(), prefix, version, cwd, "manifest_build", payload)
	if e != nil {
		return nil, e
	}
	if !scvSameJSON(in["manifest"], built.Result) {
		return nil, fmt.Errorf("publication manifest differs from native replay")
	}
	var members []json.RawMessage
	if e = json.Unmarshal(in["members"], &members); e != nil || members == nil || len(members) > 8 {
		return nil, fmt.Errorf("publication members must be a bounded array")
	}
	out := []any{}
	var entries []map[string]json.RawMessage
	if e = json.Unmarshal(manifest["entries"], &entries); e != nil || len(entries) > 8 {
		return nil, fmt.Errorf("publication manifest exceeds eight-partition profile")
	}
	loaded := map[string]json.RawMessage{}
	for _, entry := range entries {
		if string(entry["partition"]) != "null" {
			loaded[refreshString(entry["partition_digest"])] = entry["partition"]
		}
	}
	if len(members) != len(loaded) {
		return nil, fmt.Errorf("one replay/store binding is required for each loaded partition")
	}
	seen := map[string]bool{}
	for _, member := range members {
		m, e := refreshObject(member, "partition_digest", "endpoint", "store")
		if e != nil {
			return nil, e
		}
		pd := refreshString(m["partition_digest"])
		if loaded[pd] == nil || seen[pd] {
			return nil, fmt.Errorf("duplicate or unlisted publication member")
		}
		seen[pd] = true
		endpoint, e := comparisonEndpoint(m["endpoint"])
		if e != nil {
			return nil, e
		}
		if endpoint.topsID != tops {
			return nil, fmt.Errorf("publication source TOPS differs")
		}
		body, e := knowledgeengine.ReadPayload(endpoint.input)
		if e != nil {
			return nil, e
		}
		if e = emitSHVRefresh("verify", endpoint, body, func(json.RawMessage) error { return nil }); e != nil {
			return nil, e
		}
		var bundle map[string]json.RawMessage
		if e = json.Unmarshal(body, &bundle); e != nil {
			return nil, e
		}
		pin, e := partitionFromBundle(bundle)
		if e != nil {
			return nil, e
		}
		part, e := knowledgeengine.InvokeSHVPartition(context.Background(), prefix, version, cwd, "partition_build", pin)
		if e != nil {
			return nil, e
		}
		if !scvSameJSON(part.Result, loaded[pd]) {
			return nil, fmt.Errorf("publication partition differs from replayed refresh")
		}
		st, e := refreshObject(m["store"], "root", "tops_id", "namespace", "snapshot_digest", "prefix", "version")
		if e != nil {
			return nil, e
		}
		if refreshString(st["tops_id"]) != tops {
			return nil, fmt.Errorf("publication store TOPS differs")
		}
		sp, sv := refreshString(st["prefix"]), refreshString(st["version"])
		si, e := knowledgeengine.InspectSHVStore(sp, sv)
		if e != nil {
			return nil, e
		}
		p, e := knowledgeengine.SCVCanonical(map[string]any{"tops_id": st["tops_id"], "namespace": st["namespace"], "snapshot_digest": st["snapshot_digest"]})
		if e != nil {
			return nil, e
		}
		exported, e := knowledgeengine.InvokeSHVStore(context.Background(), sp, sv, refreshString(st["root"]), "export", p)
		if e != nil {
			return nil, e
		}
		var result map[string]json.RawMessage
		json.Unmarshal(exported.Result, &result)
		var snapshot map[string]json.RawMessage
		json.Unmarshal(result["snapshot"], &snapshot)
		if !scvSameJSON(snapshot["graph"], bundle["catalogue_graph"]) {
			return nil, fmt.Errorf("durable graph differs from replayed catalogue graph")
		}
		var graph map[string]json.RawMessage
		json.Unmarshal(bundle["catalogue_graph"], &graph)
		out = append(out, map[string]any{"partition_digest": m["partition_digest"], "endpoint": m["endpoint"], "store": m["store"], "bundle_digest": bundle["digest"], "graph_digest": graph["digest"], "source_installation": bundle["source_installation"], "kernel_installation": bundle["kernel_installation"], "store_installation": si})
	}
	after, e := knowledgeengine.InspectSHVPartition(prefix, version)
	if e != nil || after != pi {
		return nil, fmt.Errorf("publication partition installation changed")
	}
	return knowledgeengine.SCVCanonical(map[string]any{"catalogue_id": catalogue, "tops_id": tops, "manifest": in["manifest"], "policy": in["policy"], "partition_installation": pi, "members": out})
}
func replayPublicationDefinition(definition json.RawMessage, tops, catalogue string) error {
	var d map[string]json.RawMessage
	if e := json.Unmarshal(definition, &d); e != nil {
		return e
	}
	var pi knowledgeengine.Installation
	if e := json.Unmarshal(d["partition_installation"], &pi); e != nil {
		return e
	}
	var members []map[string]json.RawMessage
	if e := json.Unmarshal(d["members"], &members); e != nil {
		return e
	}
	inputMembers := []any{}
	for _, m := range members {
		inputMembers = append(inputMembers, map[string]any{"partition_digest": m["partition_digest"], "endpoint": m["endpoint"], "store": m["store"]})
	}
	raw, e := knowledgeengine.SCVCanonical(map[string]any{"manifest": d["manifest"], "policy": d["policy"], "partition_prefix": pi.Prefix, "partition_version": pi.Version, "members": inputMembers})
	if e != nil {
		return e
	}
	rebuilt, e := publicationEvidence(raw, tops, catalogue)
	if e != nil {
		return e
	}
	if !scvSameJSON(rebuilt, definition) {
		return fmt.Errorf("publication evidence or exact owner installation changed")
	}
	return nil
}
