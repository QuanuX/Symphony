package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeepWikiLinksAndFencedExamples(t *testing.T) {
	r := renderer{
		commit:     "abc123",
		lines:      map[string]int{"README.md": 7},
		paths:      map[string]bool{"README.md": true},
		numbers:    map[string]string{"1.1": repository + "/wiki/DeepWiki-Setup"},
		unresolved: make(map[string]bool),
	}
	lines := []string{
		"[README.md](README.md) [setup](#1.1) [README.md:2-3]()",
		"Sources: `README.md:4](), `README.md:5()`, [`README.md:6]()`, `README.md:7`",
		"The `evaluate()` function uses `evaluate`.",
		"```go\n// [README.md](README.md)\n```",
	}
	rendered, err := r.page(strings.Join(lines, "\n"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"/blob/abc123/README.md)", "/wiki/DeepWiki-Setup)", "?plain=1#L2-L3)", "?plain=1#L4)", "?plain=1#L5)", "?plain=1#L6)", "?plain=1#L7)", lines[2], lines[3]} {
		if !strings.Contains(rendered, want) {
			t.Errorf("missing %q in rendered page:\n%s", want, rendered)
		}
	}
}

func TestIncompleteSnapshotAndWikiOwnership(t *testing.T) {
	if _, err := parseTopics("- 1 Overview\n- 2 Install", "# Page: Overview\nOnly one page."); err == nil {
		t.Fatal("accepted a snapshot missing an indexed page")
	}
	block := beginIndex + "\nnew\n" + endIndex
	old := "Manual intro.\n" + beginIndex + "\nold\n" + endIndex + "\nManual footer.\n"
	got, err := indexBlock(old, block)
	if err != nil || got != "Manual intro.\n"+block+"\nManual footer.\n" {
		t.Fatalf("updating the index changed manual text: %q, %v", got, err)
	}
	dir := t.TempDir()
	name := "DeepWiki-Overview.md"
	if err := os.WriteFile(filepath.Join(dir, name), []byte("Editor-owned page."), 0644); err != nil {
		t.Fatal(err)
	}
	if err := publishLocal(dir, map[string]string{name: owned + "\nnew"}); err == nil {
		t.Fatal("overwrote editor-owned page")
	}
	got, err = readPage(dir, name)
	if err != nil || got != "Editor-owned page." {
		t.Fatalf("manual page content changed: %q, %v", got, err)
	}
}
