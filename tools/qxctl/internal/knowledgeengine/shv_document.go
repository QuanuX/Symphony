package knowledgeengine

import "encoding/json"

func validateSHVDocumentSubject(p, m, s map[string]any, version string) error {
	if version != SHVDocumentVersion {
		return shvFail()
	}
	d := shvMap(m["document"])
	x := shvMap(d["extraction"])
	var source map[string]any
	for _, v := range shvList(p["sources"]) {
		candidate := shvMap(v)
		if candidate["id"] == m["source_id"] {
			if candidate["format"] != "opaque" {
				return shvFail()
			}
			source = map[string]any{"id": candidate["id"], "path": candidate["path"], "bytes": candidate["bytes"], "digest": candidate["digest"]}
		}
	}
	if !scvEqual(source, x["source"]) {
		return shvFail()
	}
	input, _ := json.Marshal(map[string]any{"source_root": p["source_root"], "source": source, "decoder_root": d["decoder_root"], "decoder": x["decoder"], "profile": x["profile"]})
	raw, _ := json.Marshal(x)
	if ValidateSHVPDFResult("extract", input, raw) != nil {
		return shvFail()
	}
	var row map[string]any
	for _, v := range shvList(x["rows"]) {
		c := shvMap(v)
		if c["model"] == m["model"] {
			row = c
		}
	}
	if row == nil {
		return shvFail()
	}
	for _, v := range shvList(s["assertions"]) {
		a := shvMap(v)
		if a["value"] != row["opn"] || a["qualifier"] != "issuer=AMD;namespace=opn;profile=1" {
			return shvFail()
		}
	}
	return nil
}
