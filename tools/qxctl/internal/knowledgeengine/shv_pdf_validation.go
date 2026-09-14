package knowledgeengine

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"
)

func validatePDFInput(p map[string]any) error {
	if !shvFields(p, "source_root", "source", "decoder_root", "decoder", "profile") || p["profile"] != "amd-69290-table8.v1" {
		return shvFail()
	}
	s := shvMap(p["source"])
	d := shvMap(p["decoder"])
	if !shvFields(s, "id", "path", "bytes", "digest") || !shvFields(d, "path", "digest") || !regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`).MatchString(shvText(s["id"])) {
		return shvFail()
	}
	for _, x := range []struct {
		root string
		ref  map[string]any
		max  int
	}{{shvText(p["source_root"]), s, 1048576}, {shvText(p["decoder_root"]), d, 67108864}} {
		if !filepath.IsAbs(x.root) {
			return shvFail()
		}
		raw, e := readTrustedNoFollowRelative(x.root, shvText(x.ref["path"]), int64(x.max))
		if e != nil {
			return e
		}
		if digestBytes(raw) != x.ref["digest"] {
			return shvFail()
		}
		if x.ref == nil {
			return shvFail()
		}
		if n, ok := x.ref["bytes"]; ok && !scvEqual(n, len(raw)) {
			return shvFail()
		}
	}
	return nil
}
func ValidateSHVPDFResult(op string, payload, result []byte) error {
	return ValidateSHVPDFResultVersion(op, payload, result, SHVPDFVersion)
}
func ValidateSHVPDFResultVersion(op string, payload, result []byte, version string) error {
	if version != SHVPDFVersion && version != SHVPDFGraphVersion {
		return shvFail()
	}
	p, e := shvObject(payload)
	if e != nil {
		return e
	}
	r, e := shvObject(result)
	if e != nil {
		return e
	}
	if op == "inspect" {
		return shvPDFDescriptor(p, r, version)
	}
	if op == "graph_project" || op == "graph_validate" {
		return validatePDFGraphResult(op, p, r, version)
	}
	if op != "extract" || validatePDFInput(p) != nil || !shvFields(r, "protocol", "profile", "source", "decoder", "page_index", "page_count", "text", "text_digest", "rows", "namespace", "namespace_equivalence", "documentary_lineages", "digest") {
		return shvFail()
	}
	if e = scvSeal(r, "digest"); e != nil {
		return e
	}
	if r["protocol"] != "symphony.shv.pdf-extraction.v1" || !scvEqual(r["source"], p["source"]) || !scvEqual(r["decoder"], p["decoder"]) || r["profile"] != p["profile"] || r["page_index"] != json.Number("12") || r["page_count"] != json.Number("16") || r["documentary_lineages"] != json.Number("1") || r["namespace"] != "AMD.OPN" || r["namespace_equivalence"] != "not_asserted" {
		return shvFail()
	}
	text := shvText(r["text"])
	if len(text) > 131072 || digestBytes([]byte(text)) != r["text_digest"] || !strings.Contains(text, "69290 Rev. 1.00 April 2026") || !strings.Contains(text, "Table 8. EPYC 7003 Series Portfolio APP Values") {
		return shvFail()
	}
	header := "OPN Model\nNumber\nAPP \n(Weighted TFLOPS)\n"
	if strings.Count(text, header) != 1 {
		return shvFail()
	}
	tail := strings.SplitN(text, header, 2)[1]
	end := strings.Index(tail, "[Public]")
	if end < 0 || strings.TrimSpace(tail[end+8:]) != "" {
		return shvFail()
	}
	rows := []any{}
	ids := map[string]bool{}
	models := map[string]bool{}
	pattern := regexp.MustCompile(`^(100-[0-9]{9}) ([0-9A-Z]{4,8}) ([0-9]+\.[0-9]+)$`)
	for _, line := range strings.Split(tail[:end], "\n") {
		if line == "" {
			continue
		}
		m := pattern.FindStringSubmatch(line)
		if m == nil || len(rows) >= 32 || ids[m[1]] || models[m[2]] {
			return shvFail()
		}
		ids[m[1]] = true
		models[m[2]] = true
		rows = append(rows, map[string]any{"opn": m[1], "model": m[2], "source_metric_text": m[3]})
	}
	if len(rows) != 28 || !scvEqual(rows, r["rows"]) {
		return shvFail()
	}
	return nil
}
