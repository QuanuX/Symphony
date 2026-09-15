package knowledgeengine

import (
	"encoding/json"
	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"path/filepath"
)

func pubPath(v any) bool {
	s := shvText(v)
	return shvBoundedText(v, 4096) && filepath.IsAbs(s) && filepath.Clean(s) == s
}
func pubInstall(v any, module, engine string, versions ...string) bool {
	x := shvMap(v)
	if !shvFields(x, "Role", "ModuleID", "EngineID", "Version", "Prefix", "ReceiptPath", "ReceiptDigest", "ReceiptProtocol", "ExecutablePath", "ExecutableDigest") {
		return false
	}
	p := shvText(x["Prefix"])
	vstr := shvText(x["Version"])
	found := false
	for _, v := range versions {
		found = found || v == vstr
	}
	return found && x["Role"] == module && x["ModuleID"] == module && x["EngineID"] == engine && x["ReceiptProtocol"] == receiptProtocolV2 && pubPath(p) && pubPath(x["ReceiptPath"]) && pubPath(x["ExecutablePath"]) && partHash(x["ReceiptDigest"]) && partHash(x["ExecutableDigest"]) && x["ReceiptPath"] == p+"/share/symphony/receipts/"+module+"/"+vstr+"/install-receipt.json" && x["ExecutablePath"] == p+"/libexec/symphony/"+module+"/"+vstr+"/"+engine
}
func pubDefinition(d map[string]any) error { return pubDefinitionVersion(d, SHVPublicationVersion) }
func pubDefinitionVersion(d map[string]any, version string) error {
	partVersions := []string{"0.2.0-dev"}
	sourceVersions := []string{"0.1.0-dev"}
	kernelVersions := []string{"0.1.0-dev", "0.2.0-dev", "0.3.0-dev"}
	if version == "0.4.0-dev" {
		partVersions = append(partVersions, "0.3.0-dev", "0.4.0-dev")
		sourceVersions = append(sourceVersions, "0.2.0-dev")
		kernelVersions = append(kernelVersions, "0.4.0-dev")
	}
	if !shvFields(d, "catalogue_id", "tops_id", "manifest", "policy", "partition_installation", "members") || !shvID(d["catalogue_id"]) || stavprotocol.ValidateTOPSID(shvText(d["tops_id"])) != nil || !pubInstall(d["partition_installation"], "shv-partition-engine", "symphony-shv-partition", partVersions...) {
		return shvFail()
	}
	m := shvMap(d["manifest"])
	expected, e := partManifestVersion(map[string]any{"entries": m["entries"], "required_references": m["required_references"]}, shvText(shvMap(d["partition_installation"])["Version"]))
	if e != nil || !scvEqual(expected, m) {
		return shvFail()
	}
	entries, ok := partArray(m["entries"], 8)
	if !ok {
		return shvFail()
	}
	policy := shvMap(d["policy"])
	if !shvFields(policy, "missing_partitions", "missing_references") {
		return shvFail()
	}
	for _, k := range []string{"missing_partitions", "missing_references"} {
		if policy[k] != "allow" && policy[k] != "reject" {
			return shvFail()
		}
	}
	if policy["missing_partitions"] == "reject" && m["missing_count"] != json.Number("0") {
		return shvFail()
	}
	if policy["missing_references"] == "reject" {
		for _, v := range shvList(m["reference_statuses"]) {
			if shvMap(v)["status"] != "found" {
				return shvFail()
			}
		}
	}
	loaded := map[string]map[string]any{}
	for _, v := range entries {
		x := shvMap(v)
		if x["partition"] != nil {
			loaded[shvText(x["partition_digest"])] = shvMap(x["partition"])
		}
	}
	members, ok := partArray(d["members"], 8)
	if !ok || len(members) != len(loaded) {
		return shvFail()
	}
	seen := map[string]bool{}
	for _, v := range members {
		x := shvMap(v)
		pd := shvText(x["partition_digest"])
		if !shvFields(x, "partition_digest", "endpoint", "store", "bundle_digest", "graph_digest", "source_installation", "kernel_installation", "store_installation") || loaded[pd] == nil || seen[pd] || !partHash(x["bundle_digest"]) || !partHash(x["graph_digest"]) {
			return shvFail()
		}
		seen[pd] = true
		if !pubInstall(x["source_installation"], "shv-source-engine", "symphony-shv-source", sourceVersions...) || !pubInstall(x["kernel_installation"], "shv-engine", "symphony-shv", kernelVersions...) || pubStoreInstallation(x["store_installation"], version) != nil {
			return shvFail()
		}
		ep := shvMap(x["endpoint"])
		if !shvFields(ep, "bundle_path", "source_root", "state_root", "tops_id", "source_id", "source_prefix", "source_version", "kernel_prefix", "kernel_version") || !shvID(ep["source_id"]) || ep["tops_id"] != d["tops_id"] {
			return shvFail()
		}
		for _, k := range []string{"bundle_path", "source_root", "state_root", "source_prefix", "kernel_prefix"} {
			if !pubPath(ep[k]) {
				return shvFail()
			}
		}
		for _, pair := range [][2]string{{"source", "source_installation"}, {"kernel", "kernel_installation"}} {
			i := shvMap(x[pair[1]])
			if ep[pair[0]+"_prefix"] != i["Prefix"] || ep[pair[0]+"_version"] != i["Version"] || !scvEqual(shvMap(loaded[pd]["dependencies"])[pair[0]+"_engine"], map[string]any{"engine_id": i["EngineID"], "version": i["Version"], "executable_digest": i["ExecutableDigest"]}) {
				return shvFail()
			}
		}
		s := shvMap(x["store"])
		i := shvMap(x["store_installation"])
		if !shvFields(s, "root", "tops_id", "namespace", "snapshot_digest", "prefix", "version") || !pubPath(s["root"]) || !pubPath(s["prefix"]) || s["tops_id"] != d["tops_id"] || !graphIndexNamespace.MatchString(shvText(s["namespace"])) || !partHash(s["snapshot_digest"]) || s["prefix"] != i["Prefix"] || s["version"] != i["Version"] {
			return shvFail()
		}
	}
	return nil
}
func pubHead(h map[string]any) error { return pubHeadVersion(h, SHVPublicationVersion) }
func pubHeadVersion(h map[string]any, version string) error {
	if !shvFields(h, "protocol", "definition", "generation", "previous_digest", "digest") || h["protocol"] != "symphony.shv.publication-head.v1" || shvSealed(h) != nil {
		return shvFail()
	}
	g, ok := partNumber(h["generation"], 1, 32)
	if !ok || (g == 1 && h["previous_digest"] != nil) || (g > 1 && !partHash(h["previous_digest"])) {
		return shvFail()
	}
	return pubDefinitionVersion(shvMap(h["definition"]), version)
}
func pubPlan(p map[string]any) (map[string]any, error) {
	return pubPlanVersion(p, SHVPublicationVersion)
}
func pubPlanVersion(p map[string]any, version string) (map[string]any, error) {
	if !shvFields(p, "operation_id", "current", "desired", "reason") || !shvID(p["operation_id"]) || !shvBoundedText(p["reason"], 4096) || pubDefinitionVersion(shvMap(p["desired"]), version) != nil {
		return nil, shvFail()
	}
	var previous any
	generation := int64(1)
	d := shvMap(p["desired"])
	if p["current"] != nil {
		c := shvMap(p["current"])
		if pubHeadVersion(c, version) != nil {
			return nil, shvFail()
		}
		cd := shvMap(c["definition"])
		if cd["catalogue_id"] != d["catalogue_id"] || cd["tops_id"] != d["tops_id"] || scvEqual(cd, d) {
			return nil, shvFail()
		}
		g, ok := partNumber(c["generation"], 1, 31)
		if !ok {
			return nil, shvFail()
		}
		generation = g + 1
		previous = c["digest"]
	}
	return shvSealNew(map[string]any{"protocol": "symphony.shv.publication-plan.v1", "operation_id": p["operation_id"], "expected_state_digest": previous, "change_kind": "publish", "reason": p["reason"], "head": shvSealNew(map[string]any{"protocol": "symphony.shv.publication-head.v1", "definition": d, "generation": generation, "previous_digest": previous})}), nil
}
func ValidateSHVPublicationResult(op string, input, result []byte) error {
	return ValidateSHVPublicationResultVersion(op, input, result, SHVPublicationVersion)
}
func ValidateSHVPublicationResultVersion(op string, input, result []byte, version string) error {
	if shvPublicationAdmission[version] == nil {
		return shvFail()
	}
	p, e := shvObject(input)
	if e != nil {
		return e
	}
	r, e := shvObject(result)
	if e != nil {
		return e
	}
	if op == "inspect" {
		return shvPublicationDescriptor(p, r, version)
	}
	var want map[string]any
	switch op {
	case "publication_plan":
		want, e = pubPlanVersion(p, version)
	case "publication_reduce":
		if !shvFields(p, "current", "plan") {
			return shvFail()
		}
		plan := shvMap(p["plan"])
		if !shvFields(plan, "protocol", "operation_id", "expected_state_digest", "change_kind", "reason", "head", "digest") {
			return shvFail()
		}
		expected, err := pubPlanVersion(map[string]any{"operation_id": plan["operation_id"], "current": p["current"], "desired": shvMap(plan["head"])["definition"], "reason": plan["reason"]}, version)
		if err != nil || !scvEqual(expected, plan) {
			return shvFail()
		}
		want = shvSealNew(map[string]any{"protocol": "symphony.shv.publication-transition.v1", "operation_id": plan["operation_id"], "expected_state_digest": plan["expected_state_digest"], "head": plan["head"]})
	case "publication_status":
		a, ok := partArray(p["history"], 32)
		if !shvFields(p, "history") || !ok || len(a) == 0 {
			return shvFail()
		}
		ds := []any{}
		var previous, scope any
		for j, v := range a {
			h := shvMap(v)
			d := shvMap(h["definition"])
			if pubHeadVersion(h, version) != nil || !scvEqual(h["generation"], j+1) || !scvEqual(h["previous_digest"], previous) {
				return shvFail()
			}
			key := []any{d["tops_id"], d["catalogue_id"]}
			if scope != nil && !scvEqual(scope, key) {
				return shvFail()
			}
			scope = key
			previous = h["digest"]
			ds = append(ds, previous)
		}
		want = shvSealNew(map[string]any{"protocol": "symphony.shv.publication-status.v1", "head": a[len(a)-1], "history_digests": ds})
	default:
		return shvFail()
	}
	if e != nil || !scvEqual(want, r) {
		return shvFail()
	}
	return nil
}

func pubStoreInstallation(v any, version string) error {
	writer := shvText(shvMap(v)["Version"])
	admitted := writer == SHVStoreVersion || ((version == SHVPublicationTransferVersion || version == "0.3.0-dev" || version == "0.4.0-dev") && (writer == SHVStoreInventoryVersion || writer == SHVStoreTransferVersion)) || (version == "0.4.0-dev" && writer == "0.4.0-dev")
	if !admitted {
		return shvFail()
	}
	return storeInstallationVersion(v, writer)
}
