package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func dsPut(t *testing.T, p string, b []byte) {
	t.Helper()
	if e := os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(p, b, 0600); e != nil {
		t.Fatal(e)
	}
}
func dsFixture(t *testing.T) (dsOptions, dsSelection, dsDefinition) {
	t.Helper()
	work := t.TempDir()
	root := filepath.Join(work, "delivery")
	if e := os.Mkdir(root, 0700); e != nil {
		t.Fatal(e)
	}
	source := dsSource{Commit: strings.Repeat("a", 40), Tree: strings.Repeat("b", 40)}
	choices := []dsChoice{
		{ID: "cli", PackageID: "qxctl", Kind: "go", Adapter: "qxctl-files-v1", Versions: map[string]string{"qxctl": "0.1.0-dev"}, Dependencies: []string{}},
		{ID: "carrier", PackageID: "sov-stdio-adapter", Kind: "go", Adapter: "sov-executable-v1", Versions: map[string]string{"sov-stdio-adapter": "0.2.0-dev"}, Dependencies: []string{"cli"}},
	}
	s := dsSelection{Protocol: "symphony.release.delivery-selection.v1", Source: source, Profile: dsProfile{ID: "macos15-arm64", OS: "macos", Architecture: "arm64"}, CapabilitySHA: strings.Repeat("c", 64), DependencySHA: strings.Repeat("d", 64), IdentitySHA: strings.Repeat("e", 64), Artifacts: choices}
	files := []string{"metadata/install.json", "metadata/remove.json", "metadata/notices.txt", "metadata/source.tar", "metadata/build.json"}
	for _, p := range files {
		dsPut(t, filepath.Join(root, p), []byte("synthetic fixture metadata; no provider or release acceptance\n"))
	}
	artifacts := []dsArtifact{}
	for _, choice := range choices {
		primary := choice.ID + "/program"
		inventory := choice.ID + "/inventory.json"
		dsPut(t, filepath.Join(root, primary), []byte("inert fixture, not an executable"))
		dsPut(t, filepath.Join(root, inventory), []byte("{\"fixture\":true}\n"))
		files = append(files, primary, inventory)
		artifacts = append(artifacts, dsArtifact{Selection: choice, Source: &source, Files: []string{primary, inventory}, Primary: primary, Inventory: inventory, Receipts: []string{}, InstallRecipe: "metadata/install.json", RemoveRecipe: "metadata/remove.json", Notices: []string{"metadata/notices.txt"}, SourceMaterial: []string{"metadata/source.tar"}, BuildEvidence: []string{"metadata/build.json"}})
	}
	definition := dsDefinition{SetID: "fixture-independent-versions", Files: files, Artifacts: artifacts}
	o := dsOptions{mode: "prepare", root: root, selection: filepath.Join(work, "selection.json"), definition: filepath.Join(work, "definition.json"), policy: filepath.Join(work, "allowed-signers"), profile: s.Profile.ID, signer: "fixture"}
	dsPut(t, o.selection, dsJSON(s))
	dsPut(t, o.definition, dsJSON(definition))
	dsPut(t, o.policy, []byte("fixture policy placeholder; signature tests replace with a disposable key\n"))
	return o, s, definition
}
func dsPrepared(t *testing.T) (dsOptions, dsIndex) {
	t.Helper()
	o, _, _ := dsFixture(t)
	r, e := dsRun(o)
	if e != nil || r.SignatureValid || r.OwnerVerified || r.Release || r.Publication || r.FreshHost || len(r.Missing) != 2 {
		t.Fatal(r, e)
	}
	b, e := os.ReadFile(filepath.Join(o.root, "delivery-set.json"))
	if e != nil {
		t.Fatal(e)
	}
	var index dsIndex
	if dsDecode(b, &index) != nil {
		t.Fatal("decode fixture")
	}
	o.mode = "validate"
	return o, index
}
func TestDeliveryPreparationBindsAllArtifactsWithoutGrantingAcceptance(t *testing.T) {
	o, index := dsPrepared(t)
	r, e := dsRun(o)
	if e != nil || r.Scope != "unsigned_bytes_only" || r.VerifiedFiles != 9 || r.VerifiedBytes == 0 || r.OwnerVerified || r.Release || r.Publication || r.SignatureValid || r.FreshHost {
		t.Fatal(r, e)
	}
	if index.Artifacts[0].Selection.Versions["qxctl"] == index.Artifacts[1].Selection.Versions["sov-stdio-adapter"] {
		t.Fatal("fixture must preserve independent versions")
	}
	o.mode = "prepare"
	before, _ := os.ReadFile(filepath.Join(o.root, "delivery-set.json"))
	if _, e = dsRun(o); e == nil {
		t.Fatal("overwrote index")
	}
	after, _ := os.ReadFile(filepath.Join(o.root, "delivery-set.json"))
	if !bytes.Equal(before, after) {
		t.Fatal("changed sealed index")
	}
}
func TestDeliveryStrictParserAndCoverageRefusals(t *testing.T) {
	names := []string{"unknown-field", "case-alias", "escaped-duplicate", "duplicate-field", "missing-field", "null-false", "null-count", "extra-json", "invalid-utf8", "too-deep", "extra-artifact", "missing-artifact", "duplicate-artifact", "wrong-source", "wrong-tree", "wrong-profile", "wrong-capability", "wrong-map", "wrong-identities", "wrong-selection", "wrong-policy", "publication", "file-traversal", "file-absolute", "file-control", "duplicate-file", "unsorted-files", "negative-size", "oversize", "unsafe-mode", "unknown-adapter", "adapter-kind", "changed-component-version", "changed-dependency", "extra-unreferenced-file", "missing-notice", "missing-source", "missing-build", "metadata-escape", "invented-native-receipt"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			o, index := dsPrepared(t)
			switch name {
			case "extra-artifact":
				index.Artifacts = append(index.Artifacts, index.Artifacts[0])
			case "missing-artifact":
				index.Artifacts = index.Artifacts[:1]
			case "duplicate-artifact":
				index.Artifacts[1] = index.Artifacts[0]
			case "wrong-source":
				index.Source.Commit = strings.Repeat("f", 40)
			case "wrong-tree":
				index.Source.Tree = strings.Repeat("f", 40)
			case "wrong-profile":
				index.Profile.Architecture = "amd64"
			case "wrong-capability":
				index.CapabilitySHA = strings.Repeat("f", 64)
			case "wrong-map":
				index.DependencySHA = strings.Repeat("f", 64)
			case "wrong-identities":
				index.IdentitySHA = strings.Repeat("f", 64)
			case "wrong-selection":
				index.SelectionSHA = strings.Repeat("f", 64)
			case "wrong-policy":
				index.PolicySHA = strings.Repeat("f", 64)
			case "publication":
				index.Publication = true
			case "file-traversal":
				index.Files[0].Path = "../outside"
			case "file-absolute":
				index.Files[0].Path = "/tmp/outside"
			case "file-control":
				index.Files[0].Path = "bad\nname"
			case "duplicate-file":
				index.Files = append(index.Files, index.Files[0])
			case "unsorted-files":
				index.Files[0], index.Files[1] = index.Files[1], index.Files[0]
			case "negative-size":
				index.Files[0].Size = -1
			case "oversize":
				index.Files[0].Size = dsPayloadLimit + 1
			case "unsafe-mode":
				index.Files[0].Mode = 0777
			case "unknown-adapter":
				index.Artifacts[0].Selection.Adapter = "run-arbitrary-command"
			case "adapter-kind":
				index.Artifacts[0].Selection.Kind = "native"
			case "changed-component-version":
				index.Artifacts[0].Selection.Versions["qxctl"] = "9.9.9"
			case "changed-dependency":
				index.Artifacts[1].Selection.Dependencies = []string{}
			case "extra-unreferenced-file":
				index.Files = append(index.Files, dsFile{Path: "unreferenced", SHA256: strings.Repeat("a", 64), Mode: 0600})
			case "missing-notice":
				index.Artifacts[0].Notices = []string{}
			case "missing-source":
				index.Artifacts[0].SourceMaterial = []string{}
			case "missing-build":
				index.Artifacts[0].BuildEvidence = []string{}
			case "metadata-escape":
				index.Artifacts[0].InstallRecipe = "../external.json"
			case "invented-native-receipt":
				index.Artifacts[0].Receipts = []string{"metadata/notices.txt"}
			}
			b := dsJSON(index)
			switch name {
			case "unknown-field":
				b = []byte(strings.Replace(string(b), "{", `{"unknown":false,`, 1))
			case "case-alias":
				b = []byte(strings.Replace(string(b), "{", `{"Publication_Authorized":false,`, 1))
			case "escaped-duplicate":
				b = []byte(strings.Replace(string(b), "{", `{"\u0070ublication_authorized":false,`, 1))
			case "duplicate-field":
				b = []byte(strings.Replace(string(b), "{", `{"publication_authorized":false,`, 1))
			case "missing-field":
				b = []byte(strings.Replace(string(b), `"publication_authorized": false`, `"unknown": false`, 1))
			case "null-false":
				b = []byte(strings.Replace(string(b), `"publication_authorized": false`, `"publication_authorized": null`, 1))
			case "null-count":
				b = []byte(strings.Replace(string(b), `"size": 17`, `"size": null`, 1))
				var m map[string]any
				json.Unmarshal(b, &m)
				m["files"].([]any)[0].(map[string]any)["size"] = nil
				b = dsJSON(m)
			case "extra-json":
				b = append(b, []byte("{}")...)
			case "invalid-utf8":
				b = append(b, byte(0xff))
			case "too-deep":
				b = []byte(strings.Repeat("[", 40) + "0" + strings.Repeat("]", 40))
			}
			dsPut(t, filepath.Join(o.root, "delivery-set.json"), b)
			if _, e := dsRun(o); e == nil {
				t.Fatal("admitted corrupted delivery set")
			}
		})
	}
}
func TestDeliverySelectionRefusesCyclesMissingDependenciesAndDuplicateOwners(t *testing.T) {
	for _, name := range []string{"cycle", "unknown-dependency", "duplicate-dependency", "duplicate-artifact", "unknown-kind", "empty-versions", "null-dependencies"} {
		t.Run(name, func(t *testing.T) {
			o, s, _ := dsFixture(t)
			switch name {
			case "cycle":
				s.Artifacts[0].Dependencies = []string{"carrier"}
			case "unknown-dependency":
				s.Artifacts[1].Dependencies = []string{"missing"}
			case "duplicate-dependency":
				s.Artifacts[1].Dependencies = []string{"cli", "cli"}
			case "duplicate-artifact":
				s.Artifacts[1] = s.Artifacts[0]
			case "unknown-kind":
				s.Artifacts[0].Kind = "script"
			case "empty-versions":
				s.Artifacts[0].Versions = map[string]string{}
			case "null-dependencies":
				s.Artifacts[0].Dependencies = nil
			}
			dsPut(t, o.selection, dsJSON(s))
			if _, e := dsRun(o); e == nil {
				t.Fatal("admitted invalid independent selection")
			}
			if _, e := os.Lstat(filepath.Join(o.root, "delivery-set.json")); !os.IsNotExist(e) {
				t.Fatal("invalid preparation left an index")
			}
		})
	}
}
func TestDeliveryFilesystemRefusals(t *testing.T) {
	for _, name := range []string{"changed-bytes", "missing-file", "extra-file", "symlink-file", "symlink-parent", "hardlink-file", "changed-mode", "writable-directory", "policy-in-root", "selection-in-root", "existing-report", "report-in-root", "wrong-expected-profile"} {
		t.Run(name, func(t *testing.T) {
			o, index := dsPrepared(t)
			p := filepath.Join(o.root, index.Files[0].Path)
			switch name {
			case "changed-bytes":
				dsPut(t, p, []byte("changed"))
			case "missing-file":
				if e := os.Remove(p); e != nil {
					t.Fatal(e)
				}
			case "extra-file":
				dsPut(t, filepath.Join(o.root, "extra"), []byte("extra"))
			case "symlink-file":
				if e := os.Rename(p, p+".real"); e != nil {
					t.Fatal(e)
				}
				if e := os.Symlink(p+".real", p); e != nil {
					t.Fatal(e)
				}
			case "symlink-parent":
				dir := filepath.Dir(p)
				if e := os.Rename(dir, dir+".real"); e != nil {
					t.Fatal(e)
				}
				if e := os.Symlink(dir+".real", dir); e != nil {
					t.Fatal(e)
				}
			case "hardlink-file":
				if e := os.Link(p, filepath.Join(filepath.Dir(o.root), "hardlink")); e != nil {
					t.Fatal(e)
				}
			case "changed-mode":
				if e := os.Chmod(p, 0644); e != nil {
					t.Fatal(e)
				}
			case "writable-directory":
				if e := os.Chmod(filepath.Dir(p), 0777); e != nil {
					t.Fatal(e)
				}
			case "policy-in-root":
				b, _ := os.ReadFile(o.policy)
				o.policy = filepath.Join(o.root, "policy")
				dsPut(t, o.policy, b)
			case "selection-in-root":
				b, _ := os.ReadFile(o.selection)
				o.selection = filepath.Join(o.root, "selection")
				dsPut(t, o.selection, b)
			case "existing-report":
				o.output = filepath.Join(filepath.Dir(o.root), "existing-report.json")
				dsPut(t, o.output, []byte("keep original"))
			case "report-in-root":
				o.output = filepath.Join(o.root, "report.json")
			case "wrong-expected-profile":
				o.profile = "ubuntu24.04-arm64"
			}
			if _, e := dsRun(o); e == nil {
				t.Fatal("unsafe file/identity admitted")
			}
			if name == "existing-report" {
				b, _ := os.ReadFile(o.output)
				if string(b) != "keep original" {
					t.Fatal("overwrote evidence")
				}
			}
		})
	}
}
func dsKey(t *testing.T, work, name string) string {
	t.Helper()
	key := filepath.Join(work, name)
	if b, e := exec.Command("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", key).CombinedOutput(); e != nil {
		t.Fatalf("ssh-keygen: %v %s", e, b)
	}
	return key
}
func TestDeliveryRealSignaturesAndNamespaceBinding(t *testing.T) {
	o, _, _ := dsFixture(t)
	work := filepath.Dir(o.root)
	key := dsKey(t, work, "fixture-key")
	pub, e := os.ReadFile(key + ".pub")
	if e != nil {
		t.Fatal(e)
	}
	dsPut(t, o.policy, append([]byte(`fixture namespaces="symphony-delivery-set" `), pub...))
	if _, e := dsRun(o); e != nil {
		t.Fatal(e)
	}
	indexPath := filepath.Join(o.root, "delivery-set.json")
	if b, e := exec.Command("ssh-keygen", "-Y", "sign", "-f", key, "-n", dsNamespace, indexPath).CombinedOutput(); e != nil {
		t.Fatalf("sign: %v %s", e, b)
	}
	o.mode = "verify"
	r, e := dsRun(o)
	if e != nil || !r.SignatureValid || r.Scope != "signed_bytes_only" || r.OwnerVerified || r.Release || r.Publication || r.FreshHost {
		t.Fatal(r, e)
	}
	o.signer = "wrong-principal"
	if _, e := dsRun(o); e == nil {
		t.Fatal("wrong signer accepted")
	}
	o.signer = "fixture"
	original, _ := os.ReadFile(indexPath)
	dsPut(t, indexPath, append(original, ' '))
	if _, e := dsRun(o); e == nil {
		t.Fatal("changed signed bytes accepted")
	}
	dsPut(t, indexPath, original)
	if e := os.Remove(indexPath + ".sig"); e != nil {
		t.Fatal(e)
	}
	if b, e := exec.Command("ssh-keygen", "-Y", "sign", "-f", key, "-n", "symphony-release", indexPath).CombinedOutput(); e != nil {
		t.Fatalf("sign: %v %s", e, b)
	}
	if _, e := dsRun(o); e == nil {
		t.Fatal("wrong namespace accepted")
	}
	if e := os.Remove(indexPath + ".sig"); e != nil {
		t.Fatal(e)
	}
	other := dsKey(t, work, "other-key")
	if b, e := exec.Command("ssh-keygen", "-Y", "sign", "-f", other, "-n", dsNamespace, indexPath).CombinedOutput(); e != nil {
		t.Fatalf("sign: %v %s", e, b)
	}
	if _, e := dsRun(o); e == nil {
		t.Fatal("wrong key accepted")
	}
}
func TestDeliveryNativeCandidateMustBindEntireCompanionSet(t *testing.T) {
	o, s, definition := dsFixture(t)
	a := &definition.Artifacts[0]
	choice := &s.Artifacts[0]
	choice.Kind = "native"
	choice.Adapter = "native-candidate-v1"
	a.Selection = *choice
	manifest := "native/candidate.json"
	signature := manifest + ".sig"
	policy := strings.Repeat("f", 64)
	a.Manifest = &manifest
	a.Signature = &signature
	a.SignerPolicySHA = &policy
	a.Primary = "native/candidate.tar"
	a.Inventory = manifest
	for _, p := range a.Files {
		os.Remove(filepath.Join(o.root, p))
	}
	definition.Files = definition.Files[:5]
	definition.Files = append(definition.Files, definition.Artifacts[1].Files...)
	a.Files = []string{}
	for _, name := range []string{"candidate.tar", "candidate.json", "candidate.json.sig", "platform-evidence.json", "build-evidence.json", "dependency-map.json", "identity-catalog.json"} {
		p := "native/" + name
		dsPut(t, filepath.Join(o.root, p), []byte("native adapter must independently verify these fixture bytes"))
		a.Files = append(a.Files, p)
		definition.Files = append(definition.Files, p)
	}
	dsPut(t, o.selection, dsJSON(s))
	dsPut(t, o.definition, dsJSON(definition))
	r, e := dsRun(o)
	if e != nil || r.OwnerVerified || len(r.Missing) != 2 {
		t.Fatal(r, e)
	}
	// Even re-hashing and re-signing a shortened index cannot hide a missing native companion.
	raw, _ := os.ReadFile(filepath.Join(o.root, "delivery-set.json"))
	var index dsIndex
	dsDecode(raw, &index)
	index.Artifacts[0].Files = index.Artifacts[0].Files[:6]
	dsPut(t, filepath.Join(o.root, "delivery-set.json"), dsJSON(index))
	o.mode = "validate"
	if _, e := dsRun(o); e == nil {
		t.Fatal("partial native companion set accepted")
	}
}

func TestDeliveryContractSchemasMatchRequiredGoFields(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
	for name, kind := range map[string]reflect.Type{"delivery-set-v1": reflect.TypeOf(dsIndex{}), "delivery-selection-v1": reflect.TypeOf(dsSelection{}), "delivery-definition-v1": reflect.TypeOf(dsDefinition{})} {
		b, e := os.ReadFile(filepath.Join(root, "cmake/release", name+".schema.json"))
		if e != nil {
			t.Fatal(e)
		}
		var schema map[string]any
		if json.Unmarshal(b, &schema) != nil {
			t.Fatal("invalid schema")
		}
		check := func(shape map[string]any, typ reflect.Type) {
			required := map[string]bool{}
			for _, k := range shape["required"].([]any) {
				required[k.(string)] = true
			}
			properties := shape["properties"].(map[string]any)
			if shape["additionalProperties"] != false || len(required) != typ.NumField() || len(properties) != typ.NumField() {
				t.Fatal("schema field coverage drift", name, typ)
			}
			for i := 0; i < typ.NumField(); i++ {
				key := typ.Field(i).Tag.Get("json")
				if !required[key] || properties[key] == nil {
					t.Fatal("schema missing field", key)
				}
			}
		}
		check(schema, kind)
		defs := schema["$defs"].(map[string]any)
		for key, typ := range map[string]reflect.Type{"source": reflect.TypeOf(dsSource{}), "profile": reflect.TypeOf(dsProfile{}), "choice": reflect.TypeOf(dsChoice{}), "vendor": reflect.TypeOf(dsVendor{}), "file": reflect.TypeOf(dsFile{}), "artifact": reflect.TypeOf(dsArtifact{})} {
			check(defs[key].(map[string]any), typ)
		}
	}
}

func TestDeliveryCMakeSignerUsesValidatedIndexAndIndependentPolicy(t *testing.T) {
	o, _, _ := dsFixture(t)
	work := filepath.Dir(o.root)
	key := dsKey(t, work, "cmake-key")
	pub, e := os.ReadFile(key + ".pub")
	if e != nil {
		t.Fatal(e)
	}
	o.signer = "publisher@example.invalid"
	dsPut(t, o.policy, append([]byte(o.signer+` namespaces="symphony-delivery-set" `), pub...))
	if _, e := dsRun(o); e != nil {
		t.Fatal(e)
	}
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
	tool := filepath.Join(work, "delivery-set")
	build := exec.Command(filepath.Join(runtime.GOROOT(), "bin/go"), "build", "-trimpath", "-o", tool, filepath.Join(filepath.Dir(file), "delivery-set.go"))
	if b, e := build.CombinedOutput(); e != nil {
		t.Fatalf("build release tool: %v %s", e, b)
	}
	args := []string{"-DRELEASE_DELIVERY_TOOL=" + tool, "-DRELEASE_DELIVERY_ROOT=" + o.root, "-DRELEASE_DELIVERY_SELECTION=" + o.selection, "-DRELEASE_ALLOWED_SIGNERS=" + o.policy, "-DRELEASE_SIGN_KEY=" + key, "-DRELEASE_SIGNER=" + o.signer, "-DRELEASE_EXPECTED_PROFILE=" + o.profile, "-P", filepath.Join(root, "cmake/release/SignDeliverySet.cmake")}
	if b, e := exec.Command("cmake", args...).CombinedOutput(); e != nil {
		t.Fatalf("sign helper: %v %s", e, b)
	}
	o.mode = "verify"
	r, e := dsRun(o)
	if e != nil || !r.SignatureValid || r.OwnerVerified || r.Publication {
		t.Fatal(r, e)
	}
	before, _ := os.ReadFile(filepath.Join(o.root, "delivery-set.json.sig"))
	if b, e := exec.Command("cmake", args...).CombinedOutput(); e == nil || !bytes.Contains(b, []byte("signature already exists")) {
		t.Fatal("signature overwrite not refused", e, string(b))
	}
	after, _ := os.ReadFile(filepath.Join(o.root, "delivery-set.json.sig"))
	if !bytes.Equal(before, after) {
		t.Fatal("existing signature changed")
	}
}
