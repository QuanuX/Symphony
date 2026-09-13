package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
)

func TestSCVSchemaCatalogCoversCurrentCommandProtocolsAndReferences(t *testing.T) {
	base := filepath.Join("..", "..", "..", "..", "knowledge", "scv", "schemas", "v1")
	read := func(file string) map[string]any {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join(base, file))
		if err != nil {
			t.Fatal(err)
		}
		var value map[string]any
		if err = json.Unmarshal(raw, &value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	resolve := func(document map[string]any, fragment string) any {
		t.Helper()
		var value any = document
		if fragment == "" {
			return value
		}
		if !strings.HasPrefix(fragment, "#/") {
			t.Fatal("unsupported fragment", fragment)
		}
		for _, part := range strings.Split(fragment[2:], "/") {
			m, ok := value.(map[string]any)
			if !ok {
				t.Fatal("nonobject pointer", fragment)
			}
			part = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
			value, ok = m[part]
			if !ok {
				t.Fatal("unresolved pointer", fragment)
			}
		}
		return value
	}
	catalog := read("schema-catalog.json")
	seen := map[string]bool{}
	for _, raw := range catalog["entries"].([]any) {
		entry := raw.(map[string]any)
		id := entry["protocol"].(string)
		if seen[id] {
			t.Fatal("duplicate catalog protocol", id)
		}
		seen[id] = true
		schema := read(entry["file"].(string))
		resolve(schema, entry["fragment"].(string))
		if entry["kind"] == "input" {
			if _, ok := entry["template"].(map[string]any); !ok {
				t.Fatal("input template absent", id)
			}
		}
	}
	// Optional connector and CLI-owned orchestration protocols have separate
	// exact schema discovery; they do not belong to a frozen SCV engine release.
	for protocol := range graphIndexSchemaEntries("0.2.0-dev") {
		seen[protocol] = true
	}
	root, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := commandregistry.BuildExpected(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range manifest.Commands {
		if !strings.HasPrefix(command.CommandID, "qxcmd:symphony:scv.") {
			continue
		}
		for _, list := range [][]string{command.InputProtocols, command.OutputProtocols, command.ResultValidationProtocols} {
			for _, protocol := range list {
				if !seen[protocol] {
					t.Errorf("%s has undiscoverable protocol %s", command.CommandID, protocol)
				}
			}
		}
	}
	files, err := filepath.Glob(filepath.Join(base, "*.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		document := read(filepath.Base(file))
		var walk func(any)
		walk = func(value any) {
			switch v := value.(type) {
			case map[string]any:
				if ref, ok := v["$ref"].(string); ok {
					parts := strings.SplitN(ref, "#", 2)
					other := document
					if parts[0] != "" {
						other = read(parts[0])
					}
					fragment := ""
					if len(parts) == 2 && parts[1] != "" {
						fragment = "#" + parts[1]
					}
					resolve(other, fragment)
				}
				for _, nested := range v {
					walk(nested)
				}
			case []any:
				for _, nested := range v {
					walk(nested)
				}
			}
		}
		walk(document)
	}
}

func TestSCVSchemaGrammarExposesOnlySupportedFlags(t *testing.T) {
	group := newSCVSchemaCommand()
	for _, command := range group.Commands() {
		if command.Flags().Lookup("input") != nil || command.Flags().Lookup("repo") != nil {
			t.Fatal("unsupported schema flags advertised")
		}
		if command.Name() == "list" && command.Flags().Lookup("protocol") != nil {
			t.Fatal("list does not select one protocol")
		}
	}
}
