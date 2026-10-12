package sbv

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

// Independently enumerated preorder fixture; the digest covers these exact
// native compact bytes, not the adapter's canonicalization implementation.
const logical = `{"a":[{"empty":[],"flag":true,"n":"18446744073709551615","null":null,"r":{"denominator":"3","numerator":"-2"},"s":"<>&é\n"},{"n":"0"}],"z":{}}`

func fixture(bundle bool) ([]byte, Reference) {
	h := sha256.Sum256([]byte(logical))
	ref := Reference{ContentSHA256: hex.EncodeToString(h[:])}
	type f struct {
		p, k, v, parent, edge string
		children              int
	}
	rows := []f{{"", "object", "{}", "", "", 3}, {"/content_sha256", "scalar", fmt.Sprintf("%q", ref.ContentSHA256), "0", "content_sha256", 0}, {"/a", "array", "[]", "0", "a", 2}, {"/a/0", "object", "{}", "2", "0", 6}, {"/a/0/empty", "array", "[]", "3", "empty", 0}, {"/a/0/flag", "scalar", "true", "3", "flag", 0}, {"/a/0/n", "scalar", `"18446744073709551615"`, "3", "n", 0}, {"/a/0/null", "scalar", "null", "3", "null", 0}, {"/a/0/r", "object", "{}", "3", "r", 2}, {"/a/0/r/denominator", "scalar", `"3"`, "8", "denominator", 0}, {"/a/0/r/numerator", "scalar", `"-2"`, "8", "numerator", 0}, {"/a/0/s", "scalar", `"<>&é\n"`, "3", "s", 0}, {"/a/1", "object", "{}", "2", "1", 1}, {"/a/1/n", "scalar", `"0"`, "12", "n", 0}, {"/z", "object", "{}", "0", "z", 0}}
	// The legacy stream sorts the virtual digest alongside actual root keys.
	if !bundle {
		rows = append(append(append([]f{}, rows[0]), rows[2:14]...), rows[1], rows[14])
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if bundle {
		ref.ManifestPath = "/selected/manifest.json"
		ref.ManifestSHA256 = strings.Repeat("b", 64)
		_ = enc.Encode(map[string]any{"event": "begin", "protocol": "symphony.sbv.node-stream.v1", "reference": ref, "logical_protocol": "symphony.sbv.result.v1", "order": "canonical_preorder_values.v1", "source_authorship": "not_verified", "expected_value_nodes": fmt.Sprint(len(rows))})
	} else {
		_ = enc.Encode(map[string]any{"event": "begin", "protocol": "symphony.sbv.pointer-stream.v1"})
	}
	for i, r := range rows {
		m := map[string]any{"event": "node", "kind": r.k, "value": json.RawMessage(r.v)}
		if bundle {
			m["node_id"] = fmt.Sprint(i)
			m["children"] = fmt.Sprint(r.children)
			if i == 0 {
				m["parent_id"] = nil
				m["edge"] = nil
			} else {
				m["parent_id"] = r.parent
				m["edge"] = r.edge
			}
		} else {
			m["pointer"] = r.p
		}
		_ = enc.Encode(m)
	}
	end := map[string]any{"event": "end", "status": "complete", "nodes": fmt.Sprint(len(rows))}
	if bundle {
		end["content_sha256"] = ref.ContentSHA256
		end["manifest_sha256"] = ref.ManifestSHA256
	}
	_ = enc.Encode(end)
	return b.Bytes(), ref
}
func TestStreamExactFormatsAndLifetime(t *testing.T) {
	for _, bundle := range []bool{false, true} {
		t.Run(fmt.Sprint(bundle), func(t *testing.T) {
			raw, ref := fixture(bundle)
			var retained []Node
			done, err := ReadStream(context.Background(), bytes.NewReader(raw), StreamOptions{Reference: ref}, func(n Node) error { retained = append(retained, n); return nil })
			if err != nil || !done.Complete || done.Nodes != 15 || done.ContentSHA256 != ref.ContentSHA256 {
				t.Fatalf("%+v %v", done, err)
			}
			for i := range raw {
				raw[i] = 'x'
			}
			for _, n := range retained {
				if n.Pointer == "/a/0/n" && string(n.Value) != `"18446744073709551615"` {
					t.Fatal("lost precision/ownership")
				}
			}
		})
	}
}
func TestStreamRejectsCorruption(t *testing.T) {
	raw, ref := fixture(true)
	s := string(raw)
	for name, input := range map[string]string{
		"truncated":       s[:strings.LastIndex(strings.TrimSuffix(s, "\n"), "\n")+1],
		"missing-newline": strings.TrimSuffix(s, "\n"),
		"extra":           s + "{}\n",
		"numeric":         strings.Replace(s, `"value":"18446744073709551615"`, `"value":18446744073709551615`, 1),
		"float":           strings.Replace(s, `"value":"18446744073709551615"`, `"value":1.5`, 1),
		"duplicate":       strings.Replace(s, `"event":"begin"`, `"event":"begin","event":"begin"`, 1),
		"node-id":         strings.Replace(s, `"node_id":"2"`, `"node_id":"3"`, 1),
		"parent":          strings.Replace(s, `"parent_id":"3"`, `"parent_id":"0"`, 1),
		"children":        strings.Replace(s, `"children":"6"`, `"children":"7"`, 1),
		"hash":            strings.Replace(s, `"value":"0"`, `"value":"1"`, 1),
		"surrogate":       strings.Replace(s, `"value":"0"`, `"value":"\ud800"`, 1),
		"status":          strings.Replace(s, `"status":"complete"`, `"status":"error"`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			d, e := ReadStream(context.Background(), strings.NewReader(input), StreamOptions{Reference: ref}, func(Node) error { return nil })
			if e == nil || d.Complete {
				t.Fatalf("accepted %s", name)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := ReadStream(ctx, bytes.NewReader(raw), StreamOptions{Reference: ref}, func(Node) error { return nil }); e == nil {
		t.Fatal("cancel ignored")
	}
	if _, e := ReadStream(context.Background(), bytes.NewReader(raw), StreamOptions{Reference: ref, MaxRecordBytes: 12}, func(Node) error { return nil }); e == nil {
		t.Fatal("limit ignored")
	}
	want := errors.New("consumer stopped")
	if _, e := ReadStream(context.Background(), bytes.NewReader(raw), StreamOptions{Reference: ref}, func(Node) error { return want }); !errors.Is(e, want) {
		t.Fatal(e)
	}
}
func TestLegacyStructuralRejections(t *testing.T) {
	raw, ref := fixture(false)
	for _, pair := range [][2]string{{`"pointer":"/a/1"`, `"pointer":"/a/2"`}, {`"pointer":"/z"`, `"pointer":"/a"`}, {`"pointer":"/z"`, `"pointer":"/~3"`}, {`"value":{}`, `"value":{"hidden":true}`}} {
		input := bytes.Replace(raw, []byte(pair[0]), []byte(pair[1]), 1)
		if _, err := ReadStream(context.Background(), bytes.NewReader(input), StreamOptions{Reference: ref}, func(Node) error { return nil }); err == nil {
			t.Fatal(pair)
		}
	}
}
func TestRowsShapeOrderMissingAndLifetime(t *testing.T) {
	var rows []Row
	table := Table{ArrayPointer: "/a", Columns: []Column{{Name: "quantity", Pointer: "/n", Type: "uint64"}, {Name: "weight", Pointer: "/r", Type: "rational", Optional: true}, {Name: "missing", Pointer: "/absent", Type: "string", Optional: true}, {Name: "null", Pointer: "/null", Type: "json", Optional: true, Nullable: true}}}
	r, err := newTable(table, func(row Row) error { rows = append(rows, row); return nil })
	if err != nil {
		t.Fatal(err)
	}
	raw, ref := fixture(true)
	done, err := ReadStream(context.Background(), bytes.NewReader(raw), StreamOptions{Reference: ref}, r.push)
	if err != nil || !done.Complete {
		t.Fatal(err)
	}
	if err = r.finish(); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || string(rows[0].Cells[0].Value) != `"18446744073709551615"` || string(rows[1].Cells[0].Value) != `"0"` || rows[0].Cells[2].Present || !rows[0].Cells[3].Present || rows[1].Cells[3].Present {
		t.Fatalf("%+v", rows)
	}
	for _, col := range []Column{{Name: "q", Pointer: "/n", Type: "int64"}, {Name: "absent", Pointer: "/absent", Type: "string"}, {Name: "null", Pointer: "/null", Type: "json"}} {
		r, _ := newTable(Table{ArrayPointer: "/a", Columns: []Column{col}}, func(Row) error { return nil })
		if _, err = ReadStream(context.Background(), bytes.NewReader(raw), StreamOptions{Reference: ref}, r.push); err == nil {
			err = r.finish()
		}
		if err == nil {
			t.Fatal("invalid row admitted", col)
		}
	}
	if _, err = newTable(Table{Columns: []Column{{Name: "x", Type: "float64"}}}, func(Row) error { return nil }); err == nil {
		t.Fatal("implicit precision conversion")
	}
}
func TestUnicodeAndEscapedPointers(t *testing.T) {
	for _, raw := range []string{`{"a":"\ud800"}`, `{"a":"\udc00"}`, "{\"a\":\"\xff\"}", `{"a":1,"a":2}`} {
		if _, e := object([]byte(raw)); e == nil {
			t.Fatal("bad JSON admitted", raw)
		}
	}
	for _, s := range []string{"/a~b", "é", string(rune(0x2028)), "<>&\x00"} {
		if got, e := unescape(escape(s)); e != nil || got != s {
			t.Fatal(s)
		}
		var decoded string
		if e := json.Unmarshal([]byte(quote(s)), &decoded); e != nil || decoded != s {
			t.Fatal(s)
		}
	}
	if _, e := object([]byte(`{"a":"\ud83d\ude00"}`)); e != nil {
		t.Fatal(e)
	}
	if e := strictJSON([]byte(`{} {}`)); e == nil {
		t.Fatal("trailing object accepted")
	}
}
func TestNoAggregateStreamLimit(t *testing.T) {
	// Many rows exceed scanner defaults without accumulating the complete tree.
	const count = 25000
	var body bytes.Buffer
	body.WriteString(`{"rows":[`)
	for i := 0; i < count; i++ {
		if i != 0 {
			body.WriteByte(',')
		}
		body.WriteString(`"18446744073709551615"`)
	}
	body.WriteString(`]}`)
	h := sha256.Sum256(body.Bytes())
	ref := Reference{ContentSHA256: hex.EncodeToString(h[:])}
	reader, writer := io.Pipe()
	go func() {
		defer writer.Close()
		enc := json.NewEncoder(writer)
		_ = enc.Encode(map[string]any{"event": "begin", "protocol": "symphony.sbv.pointer-stream.v1"})
		emit := func(p, k string, v any) {
			_ = enc.Encode(map[string]any{"event": "node", "pointer": p, "kind": k, "value": v})
		}
		emit("", "object", map[string]any{})
		emit("/content_sha256", "scalar", ref.ContentSHA256)
		emit("/rows", "array", []any{})
		for i := 0; i < count; i++ {
			emit(fmt.Sprint("/rows/", i), "scalar", "18446744073709551615")
		}
		_ = enc.Encode(map[string]string{"event": "end", "status": "complete", "nodes": fmt.Sprint(count + 3)})
	}()
	done, err := ReadStream(context.Background(), reader, StreamOptions{Reference: ref}, func(Node) error { return nil })
	_ = reader.Close()
	if err != nil || done.Nodes != count+3 {
		t.Fatal(done, err)
	}
}

func TestEmptySelectionShapeAndExactDtypes(t *testing.T) {
	if e := columnType(json.RawMessage(`"18446744073709551616"`), Column{Name: "v", Type: "uint64"}); e == nil {
		t.Fatal("overflow admitted")
	}
	for _, raw := range []string{`{"numerator":"1","denominator":"0"}`, `{"numerator":"01","denominator":"2"}`, `{"numerator":"1","denominator":"-2"}`} {
		if e := columnType(json.RawMessage(raw), Column{Name: "v", Type: "rational"}); e == nil {
			t.Fatal("bad rational shape", raw)
		}
	}
	if e := columnType(json.RawMessage(`{"numerator":"184467440737095516160000","denominator":"3"}`), Column{Name: "v", Type: "rational"}); e != nil {
		t.Fatal(e)
	}
	r, _ := newTable(Table{ArrayPointer: "/missing"}, func(Row) error { return nil })
	if r.finish() == nil {
		t.Fatal("absent table treated as empty")
	}
	r, _ = newTable(Table{ArrayPointer: "/empty", Columns: []Column{}}, func(Row) error { t.Fatal("empty array produced row"); return nil })
	if e := r.push(Node{Pointer: "/empty", Kind: "array", Value: json.RawMessage(`[]`)}); e != nil {
		t.Fatal(e)
	}
	if e := r.finish(); e != nil {
		t.Fatal(e)
	}
}
