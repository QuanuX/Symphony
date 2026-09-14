package knowledgeengine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestSHVPDFResultBoundary(t *testing.T) {
	root := t.TempDir()
	source := []byte("synthetic source identity; validation test does not decode PDF")
	decoder := []byte("synthetic decoder identity; never executed")
	for name, raw := range map[string][]byte{"source.pdf": source, "decoder": decoder} {
		if e := os.WriteFile(filepath.Join(root, name), raw, 0600); e != nil {
			t.Fatal(e)
		}
	}
	p := map[string]any{"source_root": root, "decoder_root": root, "source": map[string]any{"id": "synthetic", "path": "source.pdf", "bytes": len(source), "digest": digestBytes(source)}, "decoder": map[string]any{"path": "decoder", "digest": digestBytes(decoder)}, "profile": "amd-69290-table8.v1"}
	text := "69290 Rev. 1.00 April 2026\nTable 8. EPYC 7003 Series Portfolio APP Values\nOPN Model\nNumber\nAPP \n(Weighted TFLOPS)\n"
	rows := []any{}
	for i := 0; i < 28; i++ {
		opn := fmt.Sprintf("100-000000%03d", 300+i)
		model := fmt.Sprint(7000 + i)
		text += opn + " " + model + " 0.1000\n"
		rows = append(rows, map[string]any{"opn": opn, "model": model, "source_metric_text": "0.1000"})
	}
	text += "[Public]"
	r := map[string]any{"protocol": "symphony.shv.pdf-extraction.v1", "profile": p["profile"], "source": p["source"], "decoder": p["decoder"], "page_index": 12, "page_count": 16, "text": text, "text_digest": digestBytes([]byte(text)), "rows": rows, "namespace": "AMD.OPN", "namespace_equivalence": "not_asserted", "documentary_lineages": 1}
	seal := func(v map[string]any) []byte {
		delete(v, "digest")
		b, _ := json.Marshal(v)
		v["digest"] = digestBytes(b)
		b, _ = json.Marshal(v)
		return b
	}
	input, _ := json.Marshal(p)
	good := seal(r)
	if e := ValidateSHVPDFResult("extract", input, good); e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"row", "namespace", "source", "lineage", "extra", "text"} {
		t.Run(name, func(t *testing.T) {
			v, _ := shvObject(good)
			switch name {
			case "row":
				shvMap(v["rows"].([]any)[0])["opn"] = "100-000000999"
			case "namespace":
				v["namespace"] = "Product ID Tray"
			case "source":
				shvMap(v["source"])["digest"] = digestBytes([]byte("other"))
			case "lineage":
				v["documentary_lineages"] = 2
			case "extra":
				v["unexpected"] = true
			case "text":
				v["text"] = text + "unexpected"
				v["text_digest"] = digestBytes([]byte(v["text"].(string)))
			}
			if ValidateSHVPDFResult("extract", input, seal(v)) == nil {
				t.Fatal("accepted resealed mismatch")
			}
		})
	}
	if e := os.WriteFile(filepath.Join(root, "source.pdf"), []byte("changed"), 0600); e != nil {
		t.Fatal(e)
	}
	if ValidateSHVPDFResult("extract", input, good) == nil {
		t.Fatal("accepted changed original")
	}
}
