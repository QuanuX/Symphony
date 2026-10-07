package main

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
	"time"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/spf13/cobra"
)

// Transport fixtures only; native and installed tests independently qualify
// logical traversal. These tests exercise the CLI's writer and exit boundary.
func cliBulkFixture(t *testing.T, format string, body []byte) (*knowledgeengine.SBVBulkExport, string) {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "export.json")
	ref := map[string]any{"manifest_path": filepath.Join(dir, "bundle", "manifest.json"), "manifest_sha256": strings.Repeat("b", 64), "content_sha256": strings.Repeat("a", 64)}
	suffix := "}\n"
	if format == "ndjson" {
		suffix = `{"content_sha256":"` + strings.Repeat("a", 64) + `","event":"end","manifest_sha256":"` + strings.Repeat("b", 64) + `","nodes":"6","status":"complete"}` + "\n"
	}
	data := append(append([]byte{}, body...), suffix...)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(data)
	request := map[string]any{"protocol": "symphony.sbv.bundle-export-input.v1", "reference": ref, "read_options": map[string]any{"max_page_bytes": nil, "cache_bytes": "0"}, "output_path": path, "format": format, "extensions": map[string]any{}}
	result := map[string]any{"protocol": "symphony.sbv.bundle-export.v1", "status": "complete", "reference": ref, "output_path": path, "format": format, "bytes": strconv.Itoa(len(data)), "file_sha256": hex.EncodeToString(h[:]), "completion_suffix": suffix, "source_authorship": "not_verified", "verification_extent": "full_logical_closure", "logical_body_bytes": "100", "logical_body_nodes": "9", "logical_body_values": "5", "exported_value_nodes": "6", "extensions": map[string]any{}, "io_stats": map[string]any{"files_read": "1", "bytes_read": "100", "cache_hits": "0", "maximum_file_bytes": "100"}}
	raw, _ := json.Marshal(result)
	x, err := knowledgeengine.OpenSBVBundleExport(request, raw)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = x.Close() })
	return x, path
}

func TestSBVBulkCommandFormatBeforeDispatch(t *testing.T) {
	for _, pair := range [][2]string{{"json", "json"}, {"json", "text"}, {"ndjson", "ndjson"}} {
		if !sbvBulkFormatCompatible(map[string]any{"format": pair[0]}, pair[1]) {
			t.Fatal(pair)
		}
	}
	for _, pair := range [][2]string{{"ndjson", "json"}, {"ndjson", "text"}, {"json", "ndjson"}} {
		if sbvBulkFormatCompatible(map[string]any{"format": pair[0]}, pair[1]) {
			t.Fatal(pair)
		}
		path := filepath.Join(t.TempDir(), "request.json")
		if err := os.WriteFile(path, []byte(`{"format":"`+pair[0]+`"}`), 0600); err != nil {
			t.Fatal(err)
		}
		command := newSBVCommand()
		command.SilenceErrors = true
		command.SilenceUsage = true
		command.SetOut(io.Discard)
		command.SetErr(io.Discard)
		command.SetArgs([]string{"bundle", "export", "--prefix", "/missing/installation", "--version", "0.22.0-dev", "--input", path, "--format", pair[1]})
		_, err := command.ExecuteC()
		if !errors.Is(err, errUsageOnly) || !strings.Contains(err.Error(), "before dispatch") {
			t.Fatalf("format reached installation: %v", err)
		}
	}
	root := newSBVCommand()
	for _, op := range []string{"import", "inspect", "query", "verify", "export"} {
		command, _, err := root.Find([]string{"bundle", op})
		if err != nil || command.Name() != op {
			t.Fatal(op, err)
		}
		if command.Flags().Lookup("input") == nil {
			t.Fatal("missing request control", op)
		}
		if (command.Flags().Lookup("receipt-only") != nil) != (op == "export") {
			t.Fatal("receipt-only exposure", op)
		}
	}
}

type cliBulkWriter struct {
	bytes.Buffer
	remaining int
	failure   error
	cancel    context.CancelFunc
	calls     int
}

func (w *cliBulkWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.cancel != nil {
		defer w.cancel()
	}
	if len(p) >= w.remaining {
		n, _ := w.Buffer.Write(p[:w.remaining])
		w.remaining = 0
		return n, w.failure
	}
	n, e := w.Buffer.Write(p)
	w.remaining -= n
	return n, e
}

func TestSBVBulkRenderingPreservesOriginalWriterFailure(t *testing.T) {
	x, _ := cliBulkFixture(t, "json", []byte(`{"v":"exact"`))
	sentinel := errors.New("private writer diagnostic")
	for _, limit := range []int{0, 1, 5} {
		w := &cliBulkWriter{remaining: limit, failure: sentinel}
		var diagnostic bytes.Buffer
		command := &cobra.Command{}
		command.SetOut(w)
		command.SetErr(&diagnostic)
		err := renderSBVBulk(command, context.Background(), x, "json")
		if !errors.Is(err, sentinel) {
			t.Fatal("lost writer identity", err)
		}
		var exit *exactEvidenceExitError
		if limit == 0 {
			if errors.As(err, &exit) || strings.Contains(err.Error(), sentinel.Error()) {
				t.Fatal("unstarted output leaked error or claimed evidence", err)
			}
		} else if !errors.As(err, &exit) || exit.code != 1 || finishCommandError(nil, nil, nil, err) != 1 {
			t.Fatal("partial output not handled", err)
		}
		if w.Len() != limit || bytes.HasSuffix(w.Bytes(), []byte("}\n")) {
			t.Fatal("partial output changed", w.String())
		}
	}
}

func TestSBVBulkCancellationStartsNoFurtherDataWrite(t *testing.T) {
	x, _ := cliBulkFixture(t, "ndjson", []byte("{\"event\":\"begin\"}\n{\"event\":\"node\"}\n"))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := &cliBulkWriter{remaining: 100000, cancel: cancel}
	command := &cobra.Command{}
	command.SetOut(w)
	command.SetErr(io.Discard)
	err := renderSBVBulk(command, ctx, x, "ndjson")
	if !errors.Is(err, context.Canceled) || w.calls != 1 || strings.Contains(w.String(), `"event":"end"`) {
		t.Fatal("write after cancellation", w.calls, err, w.String())
	}
}

func TestSBVBulkDamagedNDJSONHasErrorOnlyTerminal(t *testing.T) {
	x, path := cliBulkFixture(t, "ndjson", []byte("{\"event\":\"begin\"}\n{\"event\":\"node\"}\n"))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data[10] = 'x'
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	var out, diagnostic bytes.Buffer
	command := &cobra.Command{}
	command.SetOut(&out)
	command.SetErr(&diagnostic)
	err = renderSBVBulk(command, context.Background(), x, "ndjson")
	var status *exactEvidenceExitError
	if !errors.As(err, &status) || status.code != 1 || !strings.Contains(out.String(), `"status":"error"`) || strings.Contains(out.String(), `"status":"complete"`) {
		t.Fatal(out.String(), err)
	}
}

func TestSBVBulkPipeWriteDeadline(t *testing.T) {
	x, _ := cliBulkFixture(t, "json", []byte(`{"value":"`+strings.Repeat("x", 1024*1024)+`"`))
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	if err = w.SetWriteDeadline(time.Time{}); err != nil {
		t.Skip("host pipe does not support deadlines")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	command := &cobra.Command{}
	command.SetOut(w)
	command.SetErr(io.Discard)
	// A watchdog closes the test-owned pipe only to keep regressions from
	// hanging the suite. The product never creates a background data writer.
	watchdog := time.AfterFunc(2*time.Second, func() { _ = w.Close() })
	defer watchdog.Stop()
	start := time.Now()
	err = renderSBVBulk(command, ctx, x, "json")
	if err == nil || time.Since(start) > time.Second {
		t.Fatal("pipe write did not respect chosen deadline", err, time.Since(start))
	}
}
