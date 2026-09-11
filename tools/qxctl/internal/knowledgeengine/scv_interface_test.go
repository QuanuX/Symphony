package knowledgeengine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSCVGeneratedInterfacePreservesExactHistoricalAdmissions(t *testing.T) {
	for i, count := range []int{13, 17, 20, 21, 22, 26, 26} {
		version := []string{"0.1.0-dev", "0.2.0-dev", "0.3.0-dev", "0.4.0-dev", "0.5.0-dev", "0.6.0-dev", "0.7.0-dev"}[i]
		if scvInterfaceOperationCount(version) != count {
			t.Fatalf("%s operation surface changed", version)
		}
		expectedCompanions := []int{1, 2, 3, 4, 5, 8, 9}[i]
		if len(scvInterfaceCompanions(version)) != expectedCompanions {
			t.Fatal("companion history changed", version)
		}
		for _, domain := range SCVDomains() {
			if !SCVDomainSupported(version, domain) {
				t.Fatal("historical supplied domain unavailable", version, domain)
			}
		}
	}
	for _, version := range []string{"latest", "0.8.0-dev", "0.6.0", ""} {
		if SCVOperationSupported(version, "inspect") || SCVSupports(version, "interface") || SCVArtifactSupported(version, "knowledge_interpret") || SCVDomainSupported(version, "scv") {
			t.Fatal("unselected exact release admitted", version)
		}
	}
	for _, item := range []struct{ op, kind, version string }{{"knowledge_interpret", "knowledge", "0.3.0-dev"}, {"profile_prepare", "profile", "0.4.0-dev"}, {"provider_onboard", "provider", "0.5.0-dev"}, {"provider_coverage", "coverage", "0.5.0-dev"}, {"provider_pack_prepare", "provider_pack", "0.6.0-dev"}, {"provider_pack_evaluate", "provider_pack_evaluation", "0.6.0-dev"}, {"composition_explore", "composition", "0.6.0-dev"}, {"composition_reassess", "composition_reassessment", "0.6.0-dev"}} {
		if SCVArtifactKind(item.op) != item.kind || SCVArtifactMinimumRelease(item.op) != item.version || !SCVArtifactSupported(item.version, item.op) {
			t.Fatal("retained artifact admission changed", item)
		}
	}
	if SCVArtifactSupported("0.4.0-dev", "provider_onboard") || SCVOperationSupported("0.5.0-dev", "composition_explore") || SCVSupports("0.3.0-dev", "schema") || SCVSupports("0.5.0-dev", "interface") {
		t.Fatal("new surface retroactively admitted")
	}
	if !SCVSupports("0.3.0-dev", "workflow") || !SCVSupports("0.3.0-dev", "artifact") || !SCVSupports("0.2.0-dev", "corpus") || !SCVSupports("0.4.0-dev", "schema") {
		t.Fatal("old surface removed")
	}
}

func TestSCVGeneratedMetadataReturnsIndependentCopies(t *testing.T) {
	domains := SCVDomains()
	domains[0] = "changed"
	companions := scvInterfaceCompanions("0.6.0-dev")
	companions[0] = "changed"
	ops := SCVArtifactOperations()
	ops[0] = "changed"
	metadata, ok := SCVOperationMetadata("profile_prepare")
	if !ok {
		t.Fatal("missing metadata")
	}
	metadata.Interactions[0] = "apply"
	if SCVDomains()[0] != "scv" || scvInterfaceCompanions("0.6.0-dev")[0] != "SOURCE-KNOWLEDGE.md" || SCVArtifactOperations()[0] == "changed" || SCVOperationInteraction("profile_prepare") != "propose" {
		t.Fatal("caller mutated generated interface state")
	}
	for _, op := range []string{"provider_pack_prepare", "provider_pack_evaluate", "composition_explore", "composition_reassess"} {
		metadata, ok := SCVOperationMetadata(op)
		if !ok || metadata.ExpectedState || metadata.InputProtocol == "" || metadata.OutputProtocol == "" {
			t.Fatal("incomplete or authority-bearing metadata", op)
		}
	}
}

func TestSCVOwnerInterfaceRejectsChangedOrResealedDeclaration(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "knowledge", "scv", "OWNER-INTERFACE.json"))
	if err != nil {
		t.Fatal(err)
	}
	definition, err := scvObject(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = scvValidateOwnerInterface("0.7.0-dev", definition); err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []func(map[string]any){func(d map[string]any) { d["current_release"] = "0.5.0-dev" }, func(d map[string]any) { d["unknown"] = true }, func(d map[string]any) { d["operations"].([]any)[0].(map[string]any)["mutability"] = "proposal_only" }, func(d map[string]any) { d["domains"].([]any)[0].(map[string]any)["name"] = "scev-caller-added" }} {
		changed, err := scvObject(raw)
		if err != nil {
			t.Fatal(err)
		}
		mutation(changed)
		if _, err = scvValidateOwnerInterface("0.7.0-dev", changed); err == nil {
			t.Fatal("changed interface admitted even though its digest can be recomputed")
		}
	}
	if _, err = scvValidateOwnerInterface("0.5.0-dev", definition); err == nil {
		t.Fatal("interface discovery backported to unselected old owner")
	}
}

func TestInstalledSCVOwnerInterfaceAndSchemasWithoutCheckout(t *testing.T) {
	prefix := os.Getenv("SYMPHONY_SCV_COMPOSITION_PREFIX")
	if prefix == "" {
		t.Skip("requires exact .6 packages")
	}
	t.Chdir(t.TempDir())
	for _, domain := range SCVDomains() {
		t.Run(domain, func(t *testing.T) {
			result, err := SCVOwnerInterface(domain, prefix, "0.6.0-dev")
			if err != nil {
				t.Fatal(err)
			}
			if result["scope"] != "interface_metadata_only" || result["receipt_validation"] != "exact_owned_file_verified" || result["domain"] != domain {
				t.Fatal("wrong resource identity")
			}
			if result["definition_digest"] != scvInterfaceDefinitionDigests["0.6.0-dev"] {
				t.Fatal("wrong canonical interface definition")
			}
			schema, err := SCVSchemaDiscovery(domain, prefix, "0.6.0-dev", "show", "symphony.qxctl.scv-owner-interface.v1")
			if err != nil {
				t.Fatal(err)
			}
			if len(schema["owner_companions"].([]any)) != 8 {
				t.Fatal(".6 companion inventory incomplete")
			}
			if _, err = SCVOwnerInterface(domain, prefix, "0.5.0-dev"); err == nil {
				t.Fatal("old release substituted")
			}
		})
	}
}

func TestInstalledSCVOwnerInterfaceRejectsAlteredOwnedBytes(t *testing.T) {
	prefix := os.Getenv("SYMPHONY_SCV_COMPOSITION_PREFIX")
	if prefix == "" {
		t.Skip("requires exact .6 package")
	}
	installed, err := InspectSCVDomain("scv", prefix, "0.6.0-dev")
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
	if _, err = SCVOwnerInterface("scv", copyPrefix, "0.6.0-dev"); err != nil {
		t.Fatal(err)
	}
	definitionPath := filepath.Join(copyPrefix, "share", "symphony", "contracts", "scv-engine", "0.6.0-dev", "OWNER-INTERFACE.json")
	// Preserve valid JSON and semantic values: the receipt still binds exact bytes.
	data, err := os.ReadFile(definitionPath)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(definitionPath, append(data, ' '), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err = SCVOwnerInterface("scv", copyPrefix, "0.6.0-dev"); err == nil {
		t.Fatal("changed receipt-owned manifest bytes accepted")
	}
	definition, err := scvObject(data)
	if err != nil {
		t.Fatal(err)
	}
	definition["operations"].([]any)[0].(map[string]any)["mutability"] = "proposal_only"
	changed, err := SCVCanonical(definition)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(definitionPath, changed, 0644); err != nil {
		t.Fatal(err)
	}
	for i := range receipt.Files {
		if filepath.Base(receipt.Files[i].Path) == "OWNER-INTERFACE.json" {
			receipt.Files[i].Size = uint64(len(changed))
			receipt.Files[i].Digest = digestBytes(changed)
		}
	}
	writeReceiptV2Fixture(t, receiptPath, &receipt)
	if _, err = InspectSCVDomain("scv", copyPrefix, "0.6.0-dev"); err != nil {
		t.Fatal("fixture is not a valid resealed installation", err)
	}
	if _, err = SCVOwnerInterface("scv", copyPrefix, "0.6.0-dev"); err == nil {
		t.Fatal("resealed changed declaration widened compiled admission")
	}
}
