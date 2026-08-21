package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgebinding"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgelifecycle"
)

func TestLifecycleBindingAdapterSwitchesAndRollsBackExactReceiptV2(t *testing.T) {
	stateRoot := t.TempDir()
	adapter, err := newLifecycleBindingAdapter(stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	firstRoot, firstDigest := lifecycleBindingFixture(t, "0.1.0")
	secondRoot, secondDigest := lifecycleBindingFixture(t, "0.2.0")

	selectPackage := func(root, version, digest, expected string) string {
		t.Helper()
		desired := knowledgelifecycle.DesiredComponent{
			ComponentID: "skvi-engine", Presence: "present",
			SelectedPackage: &knowledgelifecycle.PackageIdentity{
				PackageID: "skvi-engine", Version: version,
				ReceiptProtocol: "symphony.knowledge.install-receipt.v2", ReceiptDigest: digest,
			},
		}
		before := lifecycleTestDigest("before-" + version)
		observed := knowledgelifecycle.ObservedComponent{
			ComponentID: "skvi-engine", ObservationDigest: before,
			Packages: []knowledgelifecycle.ObservedPackage{{
				PackageID: "skvi-engine", Version: version, InstallRoot: root,
				ReceiptProtocol: "symphony.knowledge.install-receipt.v2", ReceiptDigest: digest,
				Integrity: "valid", EntryPointsValidated: true,
			}},
		}
		var registryDigest *string
		if expected != "absent" {
			registryDigest = &expected
		}
		action := knowledgelifecycle.PlannedAction{
			ActionID:    "lifecycle-action:" + strings.Repeat(version[:1], 64),
			ComponentID: "skvi-engine", Kind: "select", Direction: "forward",
			ExpectedBeforeDigest: &before, TargetStateDigest: lifecycleTestDigest("target-" + version),
			ExpectedArtifactDigests: []string{digest}, ExpectedEvidence: []string{},
			PrerequisiteActionIDs: []string{}, Disposition: "ready", Blockers: []knowledgelifecycle.Blocker{},
		}
		outcome, _, evidence, err := adapter.ExecuteBinding(
			action, desired, true, observed, true,
			knowledgelifecycle.Observation{BindingRegistryDigest: registryDigest})
		if err != nil || outcome != "committed" || len(evidence) != 2 || evidence[1] != digest {
			t.Fatalf("exact binding selection failed: outcome=%s evidence=%v err=%v", outcome, evidence, err)
		}
		return evidence[0]
	}

	firstRegistry := selectPackage(firstRoot, "0.1.0", firstDigest, "absent")
	secondRegistry := selectPackage(secondRoot, "0.2.0", secondDigest, firstRegistry)
	rolledBackRegistry := selectPackage(firstRoot, "0.1.0", firstDigest, secondRegistry)
	if rolledBackRegistry == firstRegistry || rolledBackRegistry == secondRegistry {
		t.Fatal("forward/rollback binding generations did not produce linked state")
	}
	store, err := knowledgebinding.NewStore(stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.Snapshot()
	if err != nil || !snapshot.Exists || len(snapshot.Registry.Bindings) != 1 ||
		snapshot.Registry.Bindings[0].ReceiptDigest != firstDigest ||
		snapshot.Registry.PreviousRegistryDigest == nil || *snapshot.Registry.PreviousRegistryDigest != secondRegistry {
		t.Fatalf("rollback did not select exact prior package with predecessor evidence: %+v err=%v", snapshot, err)
	}
}

func TestLifecycleBindingAdapterCoversEveryEstablishedEngineRole(t *testing.T) {
	adapter, err := newLifecycleBindingAdapter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, componentID := range []string{
		"knowledge-session-coordinator", "skvi-engine", "sclv-engine", "sacv-engine",
		"sodv-engine", "ssfv-engine", "sav-engine", "sev-engine",
	} {
		if !adapter.Handles(componentID) {
			t.Errorf("lifecycle binding adapter does not cover %s", componentID)
		}
	}
}

func lifecycleBindingFixture(t *testing.T, version string) (string, string) {
	t.Helper()
	root := t.TempDir()
	base := "share/doc/symphony/skvi-engine/" + version + "/"
	license := "share/licenses/symphony-skvi-engine/" + version + "/"
	paths := []string{
		"libexec/symphony/skvi-engine/" + version + "/symphony-skvi",
		base + "INTENT.md", base + "MANIFEST.md", base + "INSTALL.md", base + "SKILL.md", base + "SPEC.md",
		license + "LICENSE-AGPL-3.0", license + "nlohmann-json-LICENSE.MIT",
	}
	files := make([]map[string]any, 0, len(paths))
	for _, relative := range paths {
		data := []byte(version + ":" + relative + "\n")
		mode := os.FileMode(0o644)
		kind := "regular"
		if strings.HasPrefix(relative, "libexec/") {
			mode, kind = 0o755, "executable"
		}
		absolute := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absolute, data, mode); err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(data)
		files = append(files, map[string]any{
			"path": relative, "kind": kind, "size": len(data),
			"digest": "sha256:" + hex.EncodeToString(digest[:]),
		})
	}
	binary := paths[0]
	platformOS := runtime.GOOS
	if platformOS == "darwin" {
		platformOS = "macos"
	}
	receipt := map[string]any{
		"protocol": "symphony.knowledge.install-receipt.v2", "format_version": 2,
		"component_id": "skvi-engine", "component_kind": "vector_engine",
		"module_id": "skvi-engine", "vector_id": "skvi", "engine_id": "symphony-skvi",
		"package_id": "skvi-engine", "version": version, "install_scope": "prefix",
		"prefix_mode": "installation_prefix", "files": files,
		"entry_points": []any{map[string]any{
			"entry_point_id": "symphony-skvi", "kind": "executable", "path": binary,
			"protocols": []string{"symphony.knowledge.engine-process.v1"},
		}},
		"provides_capabilities": []string{}, "requires_capabilities": []string{},
		"compatible_receptors": []string{"symphony.maestro.knowledge-engine.v1"},
		"platform_requirements": []any{map[string]any{
			"os": platformOS, "architecture": runtime.GOARCH, "kernel_abi": nil, "critical": true,
		}},
	}
	canonical, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(canonical)
	receiptDigest := "sha256:" + hex.EncodeToString(digest[:])
	receipt["receipt_digest"] = receiptDigest
	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	receiptPath := filepath.Join(root, "share/symphony/receipts/skvi-engine", version, "install-receipt.json")
	if err := os.MkdirAll(filepath.Dir(receiptPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(receiptPath, encoded, 0o644); err != nil {
		t.Fatal(err)
	}
	return root, receiptDigest
}
