package knowledgeengine

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func sbvBundleFixture(t *testing.T, data []byte, format string) (map[string]any, map[string]any, string) {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "export.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	reference := map[string]any{"manifest_path": filepath.Join(dir, "bundle", "manifest.json"), "manifest_sha256": strings.Repeat("b", 64), "content_sha256": strings.Repeat("a", 64)}
	request := map[string]any{"protocol": "symphony.sbv.bundle-export-input.v1", "reference": reference, "read_options": map[string]any{"max_page_bytes": nil, "cache_bytes": "0"}, "output_path": path, "format": format, "extensions": map[string]any{}}
	hash := sha256.Sum256(data)
	suffix := "}\n"
	if format == "ndjson" {
		suffix = `{"content_sha256":"` + strings.Repeat("a", 64) + `","event":"end","manifest_sha256":"` + strings.Repeat("b", 64) + `","nodes":"6","status":"complete"}` + "\n"
	}
	result := map[string]any{"protocol": "symphony.sbv.bundle-export.v1", "status": "complete", "reference": reference, "output_path": path, "format": format,
		"bytes": strconv.Itoa(len(data)), "file_sha256": hex.EncodeToString(hash[:]), "completion_suffix": suffix, "source_authorship": "not_verified", "verification_extent": "full_logical_closure",
		"logical_body_bytes": "100", "logical_body_nodes": "9", "logical_body_values": "5", "exported_value_nodes": "6", "extensions": map[string]any{},
		"io_stats": map[string]any{"files_read": "1", "bytes_read": "100", "cache_hits": "0", "maximum_file_bytes": "100"}}
	return request, result, path
}

func sbvOpenFixture(t *testing.T, request, result map[string]any) *SBVBulkExport {
	t.Helper()
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	x, err := OpenSBVBundleExport(request, raw)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = x.Close() })
	return x
}

func TestSBVBulkExactBytesAndSafeText(t *testing.T) {
	// Several admitted-size scalars cross the stream's 64 KiB I/O boundary.
	value := map[string]any{"unknown": "<>&\u2028\u2029\x1b]52;c;anything\a\r\b\u0085\u202eβ😀\\u2028", "rows": []any{nil, "0", false, strings.Repeat("x", 60000), strings.Repeat("y", 60000), strings.Repeat("z", 60000)}}
	raw, err := sbvNativeCanonical(value)
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	request, receipt, _ := sbvBundleFixture(t, raw, "json")
	x := sbvOpenFixture(t, request, receipt)
	var out bytes.Buffer
	p, err := x.Stream(context.Background(), &out, "json")
	if err != nil || !p.Complete || !bytes.Equal(out.Bytes(), raw) || p.SourceBytes != uint64(len(raw)) {
		t.Fatalf("JSON parity: %+v %v", p, err)
	}
	out.Reset()
	p, err = x.Stream(context.Background(), &out, "text")
	if err != nil || !p.Complete {
		t.Fatalf("text stream: %+v %v", p, err)
	}
	body := bytes.SplitN(out.Bytes(), []byte("\n"), 2)[1]
	for _, b := range body {
		if b >= 127 || (b < 32 && b != '\n') {
			t.Fatal("executable terminal control/rune leaked")
		}
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	expected, _ := json.Marshal(value)
	actual, _ := json.Marshal(decoded)
	if !bytes.Equal(expected, actual) {
		t.Fatal("text changed exact field meanings")
	}
}

func TestSBVBulkUnicodeEverySplit(t *testing.T) {
	raw := []byte(`{"value":"α😀` + "\u2028\u2029\u0085\u202e" + `\\u2028\u001b\n"}`)
	var expected any
	if err := json.Unmarshal(raw, &expected); err != nil {
		t.Fatal(err)
	}
	for cut := 0; cut <= len(raw); cut++ {
		var out bytes.Buffer
		s := sbvSafeBulkText{write: func(p []byte) error { _, err := out.Write(p); return err }}
		if err := s.push(raw[:cut]); err != nil {
			t.Fatal(cut, err)
		}
		if err := s.push(raw[cut:]); err != nil || len(s.pending) != 0 {
			t.Fatal(cut, err)
		}
		var value any
		if err := json.Unmarshal(out.Bytes(), &value); err != nil {
			t.Fatal(cut, err)
		}
		a, _ := json.Marshal(value)
		b, _ := json.Marshal(expected)
		if !bytes.Equal(a, b) {
			t.Fatal("split changed values", cut)
		}
	}
	for _, raw := range [][]byte{{0xff}, {'"', 0x1b, '"'}} {
		s := sbvSafeBulkText{write: func([]byte) error { return nil }}
		if s.push(raw) == nil {
			t.Fatal("invalid/control byte admitted")
		}
	}
}

type sbvCutWriter struct {
	bytes.Buffer
	remaining int
	err       error
	short     bool
	cancel    context.CancelFunc
}

func (w *sbvCutWriter) Write(p []byte) (int, error) {
	if w.cancel != nil {
		defer w.cancel()
	}
	if w.short {
		return len(p) - 1, nil
	}
	if len(p) >= w.remaining {
		n, _ := w.Buffer.Write(p[:w.remaining])
		w.remaining = 0
		return n, w.err
	}
	n, err := w.Buffer.Write(p)
	w.remaining -= n
	return n, err
}

func TestSBVBulkCompletionFailureBoundaries(t *testing.T) {
	raw := []byte("{\"value\":\"exact\"}\n")
	request, receipt, path := sbvBundleFixture(t, raw, "json")
	x := sbvOpenFixture(t, request, receipt)
	sentinel := errors.New("selected writer failure")
	for _, cut := range []int{0, 1, len(raw) - 2, len(raw) - 1, len(raw)} {
		w := &sbvCutWriter{remaining: cut, err: sentinel}
		progress, err := x.Stream(context.Background(), w, "json")
		if !errors.Is(err, sentinel) || progress.Complete || !progress.OutputFailure || progress.OutputBytes != uint64(cut) {
			t.Fatal(cut, progress, err)
		}
	}
	if p, err := x.Stream(context.Background(), &sbvCutWriter{short: true}, "json"); !errors.Is(err, io.ErrShortWrite) || p.Complete {
		t.Fatal("short writer", p, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	w := &sbvCutWriter{remaining: len(raw) + 1, cancel: cancel}
	progress, err := x.Stream(ctx, w, "json")
	if !errors.Is(err, context.Canceled) || progress.Complete || bytes.HasSuffix(w.Bytes(), []byte("}\n")) {
		t.Fatal("completion released after body callback cancelled", progress, err)
	}
	for _, changed := range [][]byte{[]byte("{\"value\":\"wrong\"}\n"), raw[:len(raw)-1], append(append([]byte{}, raw...), 'x')} {
		if err := os.WriteFile(path, changed, 0600); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		progress, err := x.Stream(context.Background(), &out, "json")
		if err == nil || progress.Complete || bytes.HasSuffix(out.Bytes(), []byte("}\n")) {
			t.Fatal("changed export completed", progress, err)
		}
	}
}

func TestSBVBulkNDJSONAndNoFollow(t *testing.T) {
	end := `{"content_sha256":"` + strings.Repeat("a", 64) + `","event":"end","manifest_sha256":"` + strings.Repeat("b", 64) + `","nodes":"6","status":"complete"}` + "\n"
	raw := []byte("{\"event\":\"begin\"}\n{\"event\":\"node\"}\n" + end)
	request, receipt, path := sbvBundleFixture(t, raw, "ndjson")
	x := sbvOpenFixture(t, request, receipt)
	var out bytes.Buffer
	p, err := x.Stream(context.Background(), &out, "ndjson")
	if err != nil || !p.Complete || p.CompleteRecords != 3 || !bytes.Equal(out.Bytes(), raw) {
		t.Fatal(p, err)
	}
	if _, err := x.Stream(context.Background(), io.Discard, "json"); err == nil {
		t.Fatal("format mismatch admitted")
	}
	if err := os.Rename(path, path+".original"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(path+".original", path); err != nil {
		t.Fatal(err)
	}
	rawReceipt, _ := json.Marshal(receipt)
	if _, err := OpenSBVBundleExport(request, rawReceipt); err == nil {
		t.Fatal("symlink export admitted")
	}
}
