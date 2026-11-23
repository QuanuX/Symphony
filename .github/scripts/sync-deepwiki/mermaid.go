package main

import (
	"encoding/json"
	"regexp"
	"strings"
)

var (
	doubledNodeLabel = regexp.MustCompile(`\[""([^"\r\n]+)""\]`)
	flowNodeLabel    = regexp.MustCompile(`\["([^"\\\r\n]|\\.|" / ")*"\]`)
	labelEscapes     = strings.NewReplacer(`" / "`, " / ", `\"`, "#quot;")
	sequenceClassDef = regexp.MustCompile(`(?m)^([ \t]*)(classDef default [^\r\n]+)$`)
)

// DeepWiki occasionally emits flowchart labels and titles using a syntax
// GitHub's Mermaid parser cannot read. Keep the text and diagram topology.
func normalizeMermaid(diagram string) string {
	declaration, body, _ := strings.Cut(diagram, "\n")
	var frontmatter string
	if quoted, ok := strings.CutPrefix(strings.TrimSpace(declaration), "title "); ok {
		var title string
		if strings.HasPrefix(quoted, `"`) && json.Unmarshal([]byte(quoted), &title) == nil {
			frontmatter = "---\ntitle: " + quoted + "\n---\n"
			diagram = body
			declaration, _, _ = strings.Cut(body, "\n")
		}
	}

	declaration = strings.TrimSpace(declaration)
	switch {
	case strings.HasPrefix(declaration, "graph "), strings.HasPrefix(declaration, "flowchart "):
		diagram = doubledNodeLabel.ReplaceAllString(diagram, `["${1}"]`)
		diagram = flowNodeLabel.ReplaceAllStringFunc(diagram, labelEscapes.Replace)
	case declaration == "sequenceDiagram":
		// Sequence diagrams do not support flowchart class definitions.
		diagram = sequenceClassDef.ReplaceAllString(diagram, `${1}%% ${2}`)
	}
	return frontmatter + diagram
}
