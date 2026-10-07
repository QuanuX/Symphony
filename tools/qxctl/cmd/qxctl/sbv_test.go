package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSBVRenderingPreservesAllNodes(t *testing.T) {
	raw := []byte(`{"sections":{"user/key~name":["9007199254740993","\u001b]8;;evil\u0007","\u202e",{},[]]},"content_sha256":"example"}`)
	var text bytes.Buffer
	if err := renderSBV(&text, raw, "text"); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"\x1b", "\x07", "\u202e"} {
		if strings.Contains(text.String(), bad) {
			t.Fatal("unescaped control")
		}
	}
	if !strings.Contains(text.String(), `\u202e`) || !strings.Contains(text.String(), "9007199254740993") {
		t.Fatal("lost exact value")
	}
	var stream bytes.Buffer
	if err := renderSBV(&stream, raw, "ndjson"); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(stream.String()), "\n")
	if len(lines) != 11 {
		t.Fatalf("unexpected tree size %d", len(lines))
	}
	nodes := map[string]any{}
	for i, line := range lines {
		var row map[string]any
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatal(err)
		}
		if i == 0 && row["event"] != "begin" {
			t.Fatal("missing begin")
		}
		if i == len(lines)-1 {
			if row["event"] != "end" || row["status"] != "complete" || row["nodes"] != "9" {
				t.Fatal(row)
			}
		} else if i > 0 {
			nodes[row["pointer"].(string)] = row["value"]
		}
	}
	if nodes["/sections/user~1key~0name/0"] != "9007199254740993" || nodes["/sections/user~1key~0name/1"] != "\x1b]8;;evil\x07" {
		t.Fatal(nodes)
	}
	if m, ok := nodes["/sections/user~1key~0name/3"].(map[string]any); !ok || len(m) != 0 {
		t.Fatal("empty object lost")
	}
	if a, ok := nodes["/sections/user~1key~0name/4"].([]any); !ok || len(a) != 0 {
		t.Fatal("empty array lost")
	}
}

type brokenSBVWriter struct {
	remaining int
	bytes.Buffer
}

func TestSBVRecoveryRenderingPreservesEvidenceAndExit(t *testing.T) {
	// Rendering fixture only; closed contract admission is tested in the consumer.
	raw := []byte(`{"protocol":"symphony.sbv.source-export.v1","status":"recovery_required","code":"sbv.source_export_unpublished","recovery":{"binary_publication_confirmed":true,"path":"/user/selected.dbn"}}`)
	for _, format := range []string{"json", "text", "ndjson"} {
		t.Run(format, func(t *testing.T) {
			var got, want bytes.Buffer
			if err := renderSBV(&want, raw, format); err != nil {
				t.Fatal(err)
			}
			err := renderSBVInvocation(&got, raw, format)
			var status *exactEvidenceExitError
			if !errors.As(err, &status) || status.code != 5 {
				t.Fatalf("lost recovery exit: %v", err)
			}
			if got.String() != want.String() {
				t.Fatal("recovery evidence changed during rendering")
			}
			if exit := finishCommandError(nil, nil, nil, err); exit != 5 {
				t.Fatalf("root changed exit to %d", exit)
			}
		})
	}
	w := &brokenSBVWriter{}
	var status *exactEvidenceExitError
	if err := renderSBVInvocation(w, raw, "json"); err == nil || errors.As(err, &status) {
		t.Fatal("failed output claimed complete evidence")
	}
	var completed bytes.Buffer
	if err := renderSBVInvocation(&completed, []byte(`{"status":"complete"}`), "json"); err != nil {
		t.Fatal(err)
	}
}

func (w *brokenSBVWriter) Write(p []byte) (int, error) {
	if w.remaining == 0 {
		return 0, errors.New("closed")
	}
	w.remaining--
	return w.Buffer.Write(p)
}
func TestSBVStreamFailureHasNoCompleteFrame(t *testing.T) {
	w := &brokenSBVWriter{remaining: 2}
	if renderSBV(w, []byte(`{"a":["b"]}`), "ndjson") == nil {
		t.Fatal("write failure ignored")
	}
	if strings.Contains(w.String(), `"complete"`) {
		t.Fatal("false completion")
	}
}

func TestSBVUserDeadlineFlags(t *testing.T) {
	for _, text := range []string{"none", "0"} {
		ctx, cancel, err := sbvDeadlineContext(context.Background(), text, "", true)
		if err != nil {
			t.Fatal(err)
		}
		if _, set := ctx.Deadline(); set {
			t.Fatal("implicit deadline")
		}
		cancel()
	}
	future := time.Now().Add(48 * time.Hour).UnixMilli()
	ctx, cancel, err := sbvDeadlineContext(context.Background(), "none", strconv.FormatInt(future, 10), false)
	if err != nil {
		t.Fatal(err)
	}
	d, ok := ctx.Deadline()
	if !ok || d.UnixMilli() != future {
		t.Fatal("changed user deadline")
	}
	cancel()
	ctx, cancel, err = sbvDeadlineContext(context.Background(), "48h", "", true)
	if err != nil {
		t.Fatal(err)
	}
	d, _ = ctx.Deadline()
	if time.Until(d) < 47*time.Hour {
		t.Fatal("timeout clamped")
	}
	cancel()
	for _, pair := range [][2]string{{"-1s", ""}, {"garbage", ""}, {"none", "01"}, {"none", "9223372036854775807"}, {"1h", strconv.FormatInt(future, 10)}} {
		if _, _, err := sbvDeadlineContext(context.Background(), pair[0], pair[1], true); err == nil {
			t.Fatal("invalid selection admitted", pair)
		}
	}
	parent, stop := context.WithCancel(context.Background())
	ctx, cancel, err = sbvDeadlineContext(parent, "none", "", false)
	if err != nil {
		t.Fatal(err)
	}
	stop()
	if ctx.Err() != context.Canceled {
		t.Fatal("parent cancellation lost")
	}
	cancel()
}
