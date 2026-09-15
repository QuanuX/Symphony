package knowledgeengine

import (
	"encoding/json"
	"regexp"
	"sort"
)

var partitionHash = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

func partHash(v any) bool { return partitionHash.MatchString(shvText(v)) }
func partNumber(v any, lo, hi int64) (int64, bool) {
	n, ok := v.(json.Number)
	if !ok {
		return 0, false
	}
	i, e := n.Int64()
	return i, e == nil && i >= lo && i <= hi
}
func partArray(v any, max int) ([]any, bool) { a, ok := v.([]any); return a, ok && len(a) <= max }
func partEngine(v any, source bool, version string) bool {
	m := shvMap(v)
	if !shvFields(m, "engine_id", "version", "executable_digest") || !partHash(m["executable_digest"]) {
		return false
	}
	if source {
		return m["engine_id"] == "symphony-shv-source" && (m["version"] == "0.1.0-dev" || (version == "0.4.0-dev" && m["version"] == "0.2.0-dev"))
	}
	return m["engine_id"] == "symphony-shv" && (m["version"] == "0.1.0-dev" || m["version"] == "0.2.0-dev" || m["version"] == "0.3.0-dev" || (version == "0.4.0-dev" && m["version"] == "0.4.0-dev"))
}
func partBuild(p map[string]any) (map[string]any, error) {
	return partBuildVersion(p, SHVPartitionVersion)
}
func partBuildVersion(p map[string]any, version string) (map[string]any, error) {
	if !shvFields(p, "dependencies", "subject_ids") {
		return nil, shvFail()
	}
	d := shvMap(p["dependencies"])
	if !shvFields(d, "source_revision_digest", "source_engine", "kernel_engine", "captures", "mapping_digest", "catalogue_digest") || !partEngine(d["source_engine"], true, version) || !partEngine(d["kernel_engine"], false, version) {
		return nil, shvFail()
	}
	for _, k := range []string{"source_revision_digest", "mapping_digest", "catalogue_digest"} {
		if !partHash(d[k]) {
			return nil, shvFail()
		}
	}
	captures, ok := partArray(d["captures"], 8)
	if !ok || len(captures) == 0 {
		return nil, shvFail()
	}
	ids, digests := map[string]bool{}, map[string]bool{}
	var bytes int64
	for _, v := range captures {
		c := shvMap(v)
		n, ok := partNumber(c["bytes"], 0, 1048576)
		if !shvFields(c, "capture_id", "capture_digest", "content_digest", "bytes") || !shvID(c["capture_id"]) || !partHash(c["capture_digest"]) || !partHash(c["content_digest"]) || !ok || ids[shvText(c["capture_id"])] || digests[shvText(c["capture_digest"])] {
			return nil, shvFail()
		}
		ids[shvText(c["capture_id"])] = true
		digests[shvText(c["capture_digest"])] = true
		bytes += n
	}
	if bytes > 4194304 {
		return nil, shvFail()
	}
	captures = append([]any{}, captures...)
	sort.Slice(captures, func(i, j int) bool {
		return shvText(shvMap(captures[i])["capture_id"]) < shvText(shvMap(captures[j])["capture_id"])
	})
	subjects, ok := partArray(p["subject_ids"], 32)
	if !ok || !shvStrings(subjects, false) {
		return nil, shvFail()
	}
	for _, v := range subjects {
		if !shvID(v) {
			return nil, shvFail()
		}
	}
	subjects = append([]any{}, subjects...)
	sort.Slice(subjects, func(i, j int) bool { return shvText(subjects[i]) < shvText(subjects[j]) })
	deps := map[string]any{}
	for k, v := range d {
		deps[k] = v
	}
	deps["captures"] = captures
	return shvSealNew(map[string]any{"protocol": "symphony.shv.partition.v1", "dependencies": deps, "subject_ids": subjects}), nil
}
func partValidate(v any, version string) bool {
	p := shvMap(v)
	if !shvFields(p, "protocol", "dependencies", "subject_ids", "digest") {
		return false
	}
	want, e := partBuildVersion(map[string]any{"dependencies": p["dependencies"], "subject_ids": p["subject_ids"]}, version)
	return e == nil && scvEqual(want, p)
}
func partRefs(v any) ([]any, bool) {
	a, ok := partArray(v, 128)
	if !ok {
		return nil, false
	}
	seen := map[string]bool{}
	for _, x := range a {
		r := shvMap(x)
		if !shvFields(r, "partition_digest", "subject_id") || !partHash(r["partition_digest"]) || !shvID(r["subject_id"]) {
			return nil, false
		}
		key := shvText(r["partition_digest"]) + "/" + shvText(r["subject_id"])
		if seen[key] {
			return nil, false
		}
		seen[key] = true
	}
	return a, true
}
func partStatus(entries []any, v any) string {
	r := shvMap(v)
	for _, x := range entries {
		e := shvMap(x)
		if e["partition_digest"] != r["partition_digest"] {
			continue
		}
		if e["partition"] == nil {
			return "missing_partition"
		}
		for _, id := range shvList(shvMap(e["partition"])["subject_ids"]) {
			if id == r["subject_id"] {
				return "found"
			}
		}
		return "missing_subject"
	}
	return "unlisted_partition"
}
func partManifest(p map[string]any) (map[string]any, error) {
	return partManifestVersion(p, SHVPartitionVersion)
}
func partManifestVersion(p map[string]any, version string) (map[string]any, error) {
	if !shvFields(p, "entries", "required_references") {
		return nil, shvFail()
	}
	entries, ok := partArray(p["entries"], 64)
	if !ok {
		return nil, shvFail()
	}
	seen := map[string]bool{}
	loaded := 0
	for _, v := range entries {
		e := shvMap(v)
		d := shvText(e["partition_digest"])
		if !shvFields(e, "partition_digest", "partition") || !partHash(d) || seen[d] {
			return nil, shvFail()
		}
		seen[d] = true
		if e["partition"] != nil {
			if !partValidate(e["partition"], version) || shvMap(e["partition"])["digest"] != d {
				return nil, shvFail()
			}
			loaded++
		}
	}
	refs, ok := partRefs(p["required_references"])
	if !ok {
		return nil, shvFail()
	}
	statuses := []any{}
	for _, r := range refs {
		statuses = append(statuses, map[string]any{"reference": r, "status": partStatus(entries, r)})
	}
	return shvSealNew(map[string]any{"protocol": "symphony.shv.partition-manifest.v1", "entries": entries, "required_references": refs, "reference_statuses": statuses, "loaded_count": loaded, "missing_count": len(entries) - loaded, "complete_inventory": loaded == len(entries)}), nil
}
func partQuery(p map[string]any) (map[string]any, error) {
	return partQueryVersion(p, SHVPartitionVersion)
}
func partQueryVersion(p map[string]any, version string) (map[string]any, error) {
	if !shvFields(p, "manifest", "selection", "limit", "cursor") {
		return nil, shvFail()
	}
	m := shvMap(p["manifest"])
	if !shvFields(m, "protocol", "entries", "required_references", "reference_statuses", "loaded_count", "missing_count", "complete_inventory", "digest") {
		return nil, shvFail()
	}
	want, e := partManifestVersion(map[string]any{"entries": m["entries"], "required_references": m["required_references"]}, version)
	if e != nil || !scvEqual(want, m) {
		return nil, shvFail()
	}
	selection, ok := partRefs(p["selection"])
	if !ok {
		return nil, shvFail()
	}
	limit, ok := partNumber(p["limit"], 1, 32)
	if !ok {
		return nil, shvFail()
	}
	sd, _ := SCVDigest(map[string]any{"selection": selection})
	var offset int64
	if p["cursor"] != nil {
		c := shvMap(p["cursor"])
		if !shvFields(c, "manifest_digest", "selection_digest", "offset") || c["manifest_digest"] != m["digest"] || c["selection_digest"] != sd {
			return nil, shvFail()
		}
		offset, ok = partNumber(c["offset"], 1, int64(len(selection))-1)
		if !ok {
			return nil, shvFail()
		}
	}
	end := offset + limit
	if end > int64(len(selection)) {
		end = int64(len(selection))
	}
	rows := []any{}
	for _, r := range selection[offset:end] {
		rows = append(rows, map[string]any{"reference": r, "status": partStatus(shvList(m["entries"]), r)})
	}
	var next any
	if end < int64(len(selection)) {
		next = map[string]any{"manifest_digest": m["digest"], "selection_digest": sd, "offset": end}
	}
	return shvSealNew(map[string]any{"protocol": "symphony.shv.partition-query.v1", "manifest_digest": m["digest"], "selection_digest": sd, "offset": offset, "rows": rows, "total_selected": len(selection), "next_cursor": next}), nil
}
func shvPartitionExpected(op string, p map[string]any) (map[string]any, error) {
	return shvPartitionExpectedVersion(op, p, SHVPartitionVersion)
}
func shvPartitionExpectedVersion(op string, p map[string]any, version string) (map[string]any, error) {
	if shvPartitionInterfaceAdmission[version] == nil {
		return nil, shvFail()
	}
	switch op {
	case "partition_build":
		return partBuildVersion(p, version)
	case "manifest_build":
		return partManifestVersion(p, version)
	case "manifest_query":
		return partQueryVersion(p, version)
	}
	return nil, shvFail()
}
func ValidateSHVPartitionResult(op string, input, result []byte) error {
	return ValidateSHVPartitionResultVersion(op, input, result, SHVPartitionVersion)
}
func ValidateSHVPartitionResultVersion(op string, input, result []byte, version string) error {
	if shvPartitionInterfaceAdmission[version] == nil {
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
		return shvPartitionDescriptor(p, r, version)
	}
	want, e := shvPartitionExpectedVersion(op, p, version)
	if e != nil {
		return e
	}
	if !scvEqual(want, r) {
		return shvFail()
	}
	return nil
}

// ExpectedSHVPartition is independent reference-contract rederivation for
// bounded checkpoint admission. It does not execute or authenticate an engine.
func ExpectedSHVPartition(op string, input []byte) (json.RawMessage, error) {
	return ExpectedSHVPartitionVersion(op, input, SHVPartitionVersion)
}
func ExpectedSHVPartitionVersion(op string, input []byte, version string) (json.RawMessage, error) {
	p, e := shvObject(input)
	if e != nil {
		return nil, e
	}
	want, e := shvPartitionExpectedVersion(op, p, version)
	if e != nil {
		return nil, e
	}
	return json.Marshal(want)
}
