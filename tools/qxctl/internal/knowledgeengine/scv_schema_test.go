package knowledgeengine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSCVSchemaFragmentRejectsUnresolvableReference(t *testing.T) {
	doc := map[string]any{"$defs": map[string]any{"a/b": map[string]any{"type": "object"}}}
	if _, err := scvSchemaFragment(doc, "#/$defs/a~1b"); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"https://remote.invalid/schema", "#/$defs/missing", "#/$defs/a~1b/type/x"} {
		if _, err := scvSchemaFragment(doc, bad); err == nil {
			t.Fatal("unresolved pointer accepted", bad)
		}
	}
}
func TestInstalledSCVSchemaDiscoveryWithoutCheckout(t *testing.T) {
	prefix := os.Getenv("SYMPHONY_SCV_AGENT_PREFIX")
	if prefix == "" {
		t.Skip("requires exact .4 packages")
	}
	t.Chdir(t.TempDir())
	for _, domain := range SCVDomains() {
		t.Run(domain, func(t *testing.T) {
			result, err := SCVSchemaDiscovery(domain, prefix, "0.4.0-dev", "list", "")
			if err != nil {
				t.Fatal(err)
			}
			if result["engine_version"] != "0.4.0-dev" {
				t.Fatal("wrong version")
			}
			entries := result["entries"].([]any)
			if len(entries) < 60 {
				t.Fatal("incomplete protocol catalog")
			}
			protocol := "symphony.scv.profile-prepare-input.v1"
			shown, err := SCVSchemaDiscovery(domain, prefix, "0.4.0-dev", "show", protocol)
			if err != nil {
				t.Fatal(err)
			}
			if shown["requested_protocol"] != protocol {
				t.Fatal("schema protocol mismatch")
			}
			prepared, err := SCVSchemaDiscovery(domain, prefix, "0.4.0-dev", "template", protocol)
			if err != nil {
				t.Fatal(err)
			}
			if prepared["template_is_evidence"] != false {
				t.Fatal("template represented as evidence")
			}
			draft := prepared["input_template"].(map[string]any)["profile"].(map[string]any)
			if draft["provider_id"] != nil || draft["rationale"] != nil {
				t.Fatal("invented profile values")
			}
			for _, c := range result["owner_companions"].([]any) {
				path := c.(map[string]any)["path"].(string)
				if !filepath.IsAbs(path) {
					t.Fatal("companion path not installed absolute")
				}
				if _, err := os.Stat(path); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := SCVSchemaDiscovery(domain, prefix, "0.3.0-dev", "list", ""); err == nil {
				t.Fatal("silently substituted .4 schemas for old package")
			}
			if _, err := SCVSchemaDiscovery(domain, prefix, "0.4.0-dev", "show", "not-an-admitted-protocol"); err == nil {
				t.Fatal("unknown protocol accepted")
			}
		})
	}
}

func TestInstalledSCVSchemaDiscoveryRejectsChangedOwnedBytes(t *testing.T) {
	prefix := os.Getenv("SYMPHONY_SCV_AGENT_PREFIX")
	if prefix == "" {
		t.Skip("requires exact .4 package")
	}
	installed, err := InspectSCVDomain("scv", prefix, "0.4.0-dev")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(installed.ReceiptPath)
	if err != nil {
		t.Fatal(err)
	}
	var receipt receiptV2
	if err = decodeExact(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	copyPrefix := t.TempDir()
	for _, file := range receipt.Files {
		data, err := os.ReadFile(filepath.Join(installed.Prefix, filepath.FromSlash(file.Path)))
		if err != nil {
			t.Fatal(err)
		}
		destination := filepath.Join(copyPrefix, filepath.FromSlash(file.Path))
		if err = os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
			t.Fatal(err)
		}
		mode := os.FileMode(0644)
		if file.Kind == "executable" {
			mode = 0755
		}
		if err = os.WriteFile(destination, data, mode); err != nil {
			t.Fatal(err)
		}
	}
	relative, err := filepath.Rel(installed.Prefix, installed.ReceiptPath)
	if err != nil {
		t.Fatal(err)
	}
	receiptPath := filepath.Join(copyPrefix, relative)
	if err = os.MkdirAll(filepath.Dir(receiptPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(receiptPath, raw, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err = SCVSchemaDiscovery("scv", copyPrefix, "0.4.0-dev", "list", ""); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(copyPrefix, "share/symphony/schemas/scv-engine/0.4.0-dev/schema-catalog.json")
	if err = os.WriteFile(target, []byte(`{"protocol":"forged"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err = SCVSchemaDiscovery("scv", copyPrefix, "0.4.0-dev", "list", ""); err == nil {
		t.Fatal("changed receipt-owned catalog was accepted")
	}
}
