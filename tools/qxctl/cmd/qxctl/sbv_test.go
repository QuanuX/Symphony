package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
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
