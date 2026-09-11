package knowledgeengine

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// SCVSchemaDiscovery reads only receipt-owned resources from the explicitly
// selected installation. It requires neither a checkout nor network access.
func SCVSchemaDiscovery(domain, prefix, version, action, protocol string) (map[string]any, error) {
	if version != "0.4.0-dev" {
		return nil, fmt.Errorf("packaged SCV schema discovery requires exact 0.4.0-dev")
	}
	if action != "list" && action != "show" && action != "template" {
		return nil, fmt.Errorf("unsupported schema discovery action")
	}
	if (action == "list" && protocol != "") || (action != "list" && protocol == "") {
		return nil, fmt.Errorf("show/template require --protocol; list does not accept it")
	}
	installed, err := InspectSCVDomain(domain, prefix, version)
	if err != nil {
		return nil, err
	}
	receiptRelative := filepath.ToSlash(filepath.Join("share", "symphony", "receipts", installed.ModuleID, version, "install-receipt.json"))
	raw, err := readTrustedNoFollowRelative(installed.Prefix, receiptRelative, maxReceiptBytes)
	if err != nil {
		return nil, err
	}
	var receipt receiptV2
	if err = decodeExact(raw, &receipt); err != nil || receipt.ReceiptDigest != installed.ReceiptDigest {
		return nil, fmt.Errorf("selected schema receipt changed")
	}
	owned := map[string]receiptV2File{}
	for _, f := range receipt.Files {
		owned[f.Path] = f
	}
	base := "share/symphony/schemas/" + installed.ModuleID + "/" + version + "/"
	read := func(name string) (map[string]any, string, error) {
		if name == "" || len(name) > 128 || strings.ContainsAny(name, "/\\") || (name != "schema-catalog.json" && !strings.HasSuffix(name, ".schema.json")) {
			return nil, "", fmt.Errorf("invalid packaged schema filename")
		}
		path := base + name
		file, found := owned[path]
		if !found || file.Kind != "regular" {
			return nil, "", fmt.Errorf("schema is not owned by selected receipt: %s", name)
		}
		data, err := readTrustedNoFollowRelative(installed.Prefix, path, maxRequestBytes)
		if err != nil {
			return nil, "", err
		}
		if uint64(len(data)) != file.Size || digestBytes(data) != file.Digest {
			return nil, "", fmt.Errorf("packaged schema changed: %s", name)
		}
		object, err := scvObject(data)
		return object, file.Digest, err
	}
	catalog, catalogDigest, err := read("schema-catalog.json")
	if err != nil {
		return nil, err
	}
	if catalog["protocol"] != "symphony.scv.schema-catalog.v1" || catalog["engine_version"] != version {
		return nil, fmt.Errorf("schema catalog identity mismatch")
	}
	entries, ok := catalog["entries"].([]any)
	if !ok || len(entries) == 0 || len(entries) > 256 {
		return nil, fmt.Errorf("invalid schema catalog entries")
	}
	index := map[string]map[string]any{}
	for _, raw := range entries {
		e, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid schema catalog entry")
		}
		id, ok := e["protocol"].(string)
		if !ok || id == "" || index[id] != nil {
			return nil, fmt.Errorf("invalid schema protocol identity")
		}
		index[id] = e
	}
	result := map[string]any{"protocol": "symphony.qxctl.scv-schema-" + action + ".v1", "domain": domain, "engine_version": version,
		"installation": installed, "catalog_digest": catalogDigest, "limits": catalog["limits"], "limitations": catalog["limitations"]}
	companions := []any{}
	for _, name := range []string{"SOURCE-KNOWLEDGE.md", "CORPUS.md", "INTERPRETATION.md", "AGENT-WORKFLOWS.md"} {
		relative := "share/doc/symphony/" + installed.ModuleID + "/" + version + "/" + name
		file, ok := owned[relative]
		if !ok || file.Kind != "regular" {
			return nil, fmt.Errorf("missing packaged owner companion %s", name)
		}
		companions = append(companions, map[string]any{"path": filepath.Join(installed.Prefix, filepath.FromSlash(relative)), "digest": file.Digest})
	}
	result["owner_companions"] = companions
	if action == "list" {
		out := []any{}
		ids := []string{}
		for id := range index {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			e := index[id]
			copy := map[string]any{}
			for key, value := range e {
				if key != "template" && key != "required_inputs" {
					copy[key] = value
				}
			}
			_, copy["template_available"] = e["template"]
			out = append(out, copy)
		}
		result["entries"] = out
	} else {
		entry := index[protocol]
		if entry == nil {
			return nil, fmt.Errorf("protocol is not in selected installed schema catalog: %s", protocol)
		}
		file, fok := entry["file"].(string)
		fragment, pok := entry["fragment"].(string)
		if !fok || !pok {
			return nil, fmt.Errorf("invalid schema catalog reference")
		}
		document, digest, err := read(file)
		if err != nil {
			return nil, err
		}
		if _, err = scvSchemaFragment(document, fragment); err != nil {
			return nil, err
		}
		result["requested_protocol"] = protocol
		result["schema_file"] = file
		result["schema_fragment"] = fragment
		result["schema_digest"] = digest
		if action == "template" {
			template, exists := entry["template"]
			if !exists {
				return nil, fmt.Errorf("selected protocol has no input authoring template")
			}
			result["input_template"] = template
			result["required_inputs"] = entry["required_inputs"]
			result["template_is_evidence"] = false
		} else {
			result["document"] = document
			references := map[string]any{}
			seen := map[string]bool{}
			var visit func(string, map[string]any) error
			visit = func(name string, doc map[string]any) error {
				if seen[name] {
					return nil
				}
				seen[name] = true
				if len(seen) > 64 {
					return fmt.Errorf("schema reference closure exceeds 64 documents")
				}
				var walk func(any) error
				walk = func(value any) error {
					switch v := value.(type) {
					case map[string]any:
						if rawRef, present := v["$ref"]; present {
							ref, ok := rawRef.(string)
							if !ok {
								return fmt.Errorf("invalid schema reference")
							}
							parts := strings.SplitN(ref, "#", 2)
							target := parts[0]
							pointer := ""
							if len(parts) == 2 && parts[1] != "" {
								pointer = "#" + parts[1]
							}
							other := doc
							if target != "" {
								var err error
								other, _, err = read(target)
								if err != nil {
									return err
								}
							}
							if _, err := scvSchemaFragment(other, pointer); err != nil {
								return err
							}
							if target != "" && target != file {
								references[target] = other
								if err := visit(target, other); err != nil {
									return err
								}
							}
						}
						for _, nested := range v {
							if err := walk(nested); err != nil {
								return err
							}
						}
					case []any:
						for _, nested := range v {
							if err := walk(nested); err != nil {
								return err
							}
						}
					}
					return nil
				}
				return walk(doc)
			}
			if err = visit(file, document); err != nil {
				return nil, err
			}
			result["reference_documents"] = references
		}
	}
	current, err := InspectSCVDomain(domain, prefix, version)
	if err != nil || current != installed {
		return nil, fmt.Errorf("schema installation changed during discovery")
	}
	digest, err := SCVDigest(result)
	if err != nil {
		return nil, err
	}
	result["digest"] = digest
	encoded, err := SCVCanonical(result)
	if err != nil {
		return nil, err
	}
	if len(encoded) > 4<<20 {
		return nil, fmt.Errorf("schema discovery output exceeds 4 MiB")
	}
	return result, nil
}

func scvSchemaFragment(document map[string]any, fragment string) (any, error) {
	if fragment == "" {
		return document, nil
	}
	if !strings.HasPrefix(fragment, "#/") {
		return nil, fmt.Errorf("unsupported schema fragment")
	}
	var value any = document
	for _, part := range strings.Split(fragment[2:], "/") {
		part = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
		object, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("schema fragment does not resolve")
		}
		value, ok = object[part]
		if !ok {
			return nil, fmt.Errorf("schema fragment is absent")
		}
	}
	return value, nil
}
