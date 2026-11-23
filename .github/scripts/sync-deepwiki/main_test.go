package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOverviewMigratesToWikiHome(t *testing.T) {
	repo, wiki, input := t.TempDir(), t.TempDir(), t.TempDir()
	write := func(dir, file, data string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, file), []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
	gitRun := func(args ...string) string {
		t.Helper()
		out, err := git(repo, args...)
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(out)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	gitRun("init")
	gitRun("config", "user.name", "Wiki Test")
	gitRun("config", "user.email", "wiki@example.invalid")
	write(repo, "README.md", "Symphony\n")
	gitRun("add", "README.md")
	gitRun("commit", "-m", "indexed source")
	pin := gitRun("rev-parse", "HEAD")
	gitRun("commit", "--allow-empty", "-m", "later revision")

	const index = "- 1 Overview\n  - 1.1 Setup"
	const overview = "# Overview\n\n<details><summary>Relevant source files</summary>\n\n[README.md](README.md)\n\n</details>\n\nSymphony architecture.\n\n```mermaid\ngraph TD\nA-->B\n```"
	const contents = "# Page: Overview\n" + overview + "\n# Page: Setup\n# Setup\n\n[Overview](#1)"
	const envelope = `{"id":1,"result":{"content":[{"type":"text","text":%q}]}}`
	write(input, "structure.sse", fmt.Sprintf(envelope, index))
	write(input, "contents.sse", fmt.Sprintf(envelope, contents))
	marker := fmt.Sprintf("<!-- DeepWiki snapshot %x citation-commit %s -->", sha256.Sum256([]byte(index+"\x00"+contents)), pin)
	write(wiki, "Home.md", "Editor's note.\n"+beginIndex+"\n"+marker+"\n## Old index\n"+endIndex+"\nContact the maintainers.\n")

	snapshot := func() map[string]string {
		t.Helper()
		if err := run(wiki, repo, input, ""); err != nil {
			t.Fatal(err)
		}
		pages := make(map[string]string)
		for _, name := range []string{"Home.md", "_Sidebar.md", "DeepWiki-Overview.md", "DeepWiki-Setup.md"} {
			data, err := os.ReadFile(filepath.Join(wiki, name))
			if err != nil {
				t.Fatal(err)
			}
			pages[name] = string(data)
		}
		return pages
	}
	pages := snapshot()
	home := pages["Home.md"]
	rendered := strings.Replace(overview, "(README.md)", "("+repository+"/blob/"+pin+"/README.md)", 1)
	if !strings.Contains(home, rendered) || !strings.Contains(pages["DeepWiki-Overview.md"], rendered) {
		t.Fatal("Home and the existing Overview URL must both show the full rendered overview")
	}
	if !strings.HasPrefix(home, "Editor's note.\n") || !strings.HasSuffix(home, "Contact the maintainers.\n") ||
		!strings.Contains(home, marker) || strings.Contains(home, "Old index") || strings.Contains(home, "[1.1 Setup]") {
		t.Fatalf("Home migration lost manual text or source metadata, or retained the index:\n%s", home)
	}
	if !strings.Contains(pages["_Sidebar.md"], "[1 Overview]("+repository+"/wiki/Home)") ||
		!strings.Contains(pages["DeepWiki-Setup.md"], "[Overview]("+repository+"/wiki/Home)") {
		t.Fatal("the sidebar and cross-topic links must open the wiki home for Overview")
	}
	for name, repeat := range snapshot() {
		if repeat != pages[name] {
			t.Errorf("%s changed when replaying the same snapshot", name)
		}
	}
}

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
