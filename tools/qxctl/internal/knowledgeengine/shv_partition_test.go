package knowledgeengine

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSHVPartitionCorrespondence(t *testing.T) {
	h := "sha256:" + strings.Repeat("1", 64)
	raw := []byte(`{"dependencies":{"source_revision_digest":"` + h + `","source_engine":{"engine_id":"symphony-shv-source","version":"0.1.0-dev","executable_digest":"` + h + `"},"kernel_engine":{"engine_id":"symphony-shv","version":"0.2.0-dev","executable_digest":"` + h + `"},"captures":[{"capture_id":"body","capture_digest":"` + h + `","content_digest":"` + h + `","bytes":3}],"mapping_digest":"` + h + `","catalogue_digest":"` + h + `"},"subject_ids":["b","a"]}`)
	p, e := shvObject(raw)
	if e != nil {
		t.Fatal(e)
	}
	part, e := partBuild(p)
	if e != nil {
		t.Fatal(e)
	}
	encoded, _ := json.Marshal(part)
	if e = ValidateSHVPartitionResult("partition_build", raw, encoded); e != nil {
		t.Fatal(e)
	}
	part["subject_ids"] = []any{"forged"}
	delete(part, "digest")
	part = shvSealNew(part)
	encoded, _ = json.Marshal(part)
	if ValidateSHVPartitionResult("partition_build", raw, encoded) == nil {
		t.Fatal("resealed false subject accepted")
	}
	part, _ = partBuild(p)
	input := map[string]any{"entries": []any{map[string]any{"partition_digest": part["digest"], "partition": part}, map[string]any{"partition_digest": "sha256:" + strings.Repeat("2", 64), "partition": nil}}, "required_references": []any{}}
	m, e := partManifest(input)
	if e != nil {
		t.Fatal(e)
	}
	m["complete_inventory"] = true
	delete(m, "digest")
	m = shvSealNew(m)
	i, _ := json.Marshal(input)
	r, _ := json.Marshal(m)
	if ValidateSHVPartitionResult("manifest_build", i, r) == nil {
		t.Fatal("false completeness accepted")
	}
	for _, op := range []string{"partition_build", "manifest_build", "manifest_query"} {
		if _, e := shvPartitionExpected(op, map[string]any{}); e == nil {
			t.Fatal(op)
		}
	}
}
func TestSHVPartitionCursorBindings(t *testing.T) {
	m, _ := partManifest(map[string]any{"entries": []any{}, "required_references": []any{}})
	mraw, _ := json.Marshal(m)
	m, _ = shvObject(mraw)
	h := "sha256:" + strings.Repeat("1", 64)
	selection := []any{map[string]any{"partition_digest": h, "subject_id": "a"}, map[string]any{"partition_digest": h, "subject_id": "b"}}
	input := map[string]any{"manifest": m, "selection": selection, "limit": json.Number("1"), "cursor": nil}
	q, e := partQuery(input)
	if e != nil {
		t.Fatal(e)
	}
	c := shvMap(q["next_cursor"])
	c["offset"] = json.Number("1")
	input["cursor"] = c
	c["selection_digest"] = h
	if _, e = partQuery(input); e == nil {
		t.Fatal("foreign cursor accepted")
	}
}
