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
	for _, name := range []string{"candidate.tar", "candidate.json", "candidate.json.sig", "platform-evidence.json", "build-evidence.json", "dependency-map.json", "identity-catalog.json", "SHA256SUMS"} {
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
	for name, kind := range map[string]reflect.Type{"delivery-set-v1": reflect.TypeOf(dsIndex{}), "delivery-selection-v1": reflect.TypeOf(dsSelection{}), "delivery-definition-v1": reflect.TypeOf(dsDefinition{}), "immutable-files-v1": reflect.TypeOf(dsExecutableInventory{})} {
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
		if name == "immutable-files-v1" {
			check(defs["file"].(map[string]any), reflect.TypeOf(dsFile{}))
			continue
		}
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

func TestExecutableIdentityBindings(t *testing.T) {
	_, index := dsPrepared(t)
	a := index.Artifacts[0]
	a.Primary = "qxctl"
	a.Inventory = "inventory.json"
	a.Files = []string{a.Primary, a.Inventory}
	file := dsFile{Path: a.Primary, SHA256: strings.Repeat("1", 64), Size: 2, Mode: 0700}
	index.Files = []dsFile{file, {Path: a.Inventory, SHA256: strings.Repeat("2", 64), Size: 3, Mode: 0600}}
	sum := strings.Repeat("a", 64)
	build := dsExecutableBuild{Protocol: "symphony.artifact-build-identity.v1", Component: "qxctl", Version: a.Selection.Versions["qxctl"], Binary: "qxctl", Commit: index.Source.Commit, Tree: index.Source.Tree, Epoch: "123", DependencySHA: index.DependencySHA, IdentitySHA: index.IdentitySHA, ModSHA: sum, SumSHA: &sum, Recipe: "symphony.qxctl.reproducible-build.v1", GoVersion: "go1.26.5", GOOS: "darwin", GOARCH: index.Profile.Architecture, ExecutableSHA: file.SHA256, Builds: 2, Reproducibility: "verified_on_recorded_profile"}
	inventory := dsExecutableInventory{Protocol: "symphony.release.immutable-files.v1", Files: []dsFile{file}}
	if dsExecutableBinding(index, a, build, inventory) != nil {
		t.Fatal("valid identity rejected")
	}
	cases := map[string]func(*dsExecutableBuild){
		"protocol":        func(b *dsExecutableBuild) { b.Protocol = "other" },
		"component":       func(b *dsExecutableBuild) { b.Component = "sov-stdio-adapter" },
		"version":         func(b *dsExecutableBuild) { b.Version = "other" },
		"binary":          func(b *dsExecutableBuild) { b.Binary = "other" },
		"commit":          func(b *dsExecutableBuild) { b.Commit = strings.Repeat("b", 40) },
		"tree":            func(b *dsExecutableBuild) { b.Tree = strings.Repeat("c", 40) },
		"epoch":           func(b *dsExecutableBuild) { b.Epoch = "00123" },
		"negative-epoch":  func(b *dsExecutableBuild) { b.Epoch = "-1" },
		"dependency":      func(b *dsExecutableBuild) { b.DependencySHA = sum },
		"identities":      func(b *dsExecutableBuild) { b.IdentitySHA = sum },
		"module":          func(b *dsExecutableBuild) { b.ModSHA = "invalid" },
		"sum":             func(b *dsExecutableBuild) { bad := "invalid"; b.SumSHA = &bad },
		"recipe":          func(b *dsExecutableBuild) { b.Recipe = "symphony.go.candidate-build.v1" },
		"toolchain":       func(b *dsExecutableBuild) { b.GoVersion = "go1.27.1" },
		"os":              func(b *dsExecutableBuild) { b.GOOS = "linux" },
		"arch":            func(b *dsExecutableBuild) { b.GOARCH = "amd64" },
		"executable":      func(b *dsExecutableBuild) { b.ExecutableSHA = sum },
		"build-count":     func(b *dsExecutableBuild) { b.Builds = 1 },
		"reproducibility": func(b *dsExecutableBuild) { b.Reproducibility = "not_assessed" },
		"publication":     func(b *dsExecutableBuild) { b.Publication = true },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			bad := build
			change(&bad)
			if dsExecutableBinding(index, a, bad, inventory) == nil {
				t.Fatal("mismatch accepted")
			}
		})
	}
	for _, name := range []string{"missing", "extra", "digest", "size", "mode", "path", "protocol"} {
		t.Run("inventory-"+name, func(t *testing.T) {
			bad := dsExecutableInventory{Protocol: inventory.Protocol, Files: append([]dsFile{}, inventory.Files...)}
			switch name {
			case "missing":
				bad.Files = []dsFile{}
			case "extra":
				bad.Files = append(bad.Files, file)
			case "digest":
				bad.Files[0].SHA256 = sum
			case "size":
				bad.Files[0].Size++
			case "mode":
				bad.Files[0].Mode = 0600
			case "path":
				bad.Files[0].Path = "other"
			case "protocol":
				bad.Protocol = "other"
			}
			if dsExecutableBinding(index, a, build, bad) == nil {
				t.Fatal("inventory mismatch accepted")
			}
		})
	}
	info := dsQxctlInfo{Protocol: "symphony.qxctl.build-info.v1", Component: "qxctl", Version: build.Version, SourceState: "stamped", Commit: &build.Commit, Tree: &build.Tree, Epoch: &build.Epoch, DependencySHA: &build.DependencySHA, Recipe: &build.Recipe, GoVersion: build.GoVersion, GOOS: build.GOOS, GOARCH: build.GOARCH, ExecutableSHA: build.ExecutableSHA}
	if dsQxctlBinding(build, dsJSON(info)) != nil {
		t.Fatal("valid running identity rejected")
	}
	for _, name := range []string{"unstamped", "missing-commit", "wrong-tree", "wrong-binary", "wrong-toolchain"} {
		t.Run("running-"+name, func(t *testing.T) {
			bad := info
			switch name {
			case "unstamped":
				bad.SourceState = "unavailable"
			case "missing-commit":
				bad.Commit = nil
			case "wrong-tree":
				wrong := strings.Repeat("f", 40)
				bad.Tree = &wrong
			case "wrong-binary":
				bad.ExecutableSHA = sum
			case "wrong-toolchain":
				bad.GoVersion = "go1.27.1"
			}
			if dsQxctlBinding(build, dsJSON(bad)) == nil {
				t.Fatal("running identity mismatch accepted")
			}
		})
	}
}
func TestNativeDeliveryBindings(t *testing.T) {
	_, index := dsPrepared(t)
	manifest := "native/candidate.json"
	policy := strings.Repeat("f", 64)
	artifact := dsArtifact{Selection: dsChoice{ID: "native", Kind: "native", Adapter: "native-candidate-v1", Versions: map[string]string{"component": "1.0.0"}}, Primary: "native/candidate.tar", Manifest: &manifest, SignerPolicySHA: &policy}
	index.Files = []dsFile{{Path: artifact.Primary, SHA256: strings.Repeat("1", 64)}, {Path: manifest, SHA256: strings.Repeat("2", 64)}}
	host := dsNativeHost{Protocol: "symphony.release.host-preparation.v1", Status: "verified_extraction_only", Commit: index.Source.Commit, Tree: index.Source.Tree, ArtifactSHA: index.Files[0].SHA256, ManifestSHA: index.Files[1].SHA256, PolicySHA: policy, Profile: index.Profile.ID, FileCount: 1}
	if dsNativeBinding(index, artifact, host) != nil {
		t.Fatal("valid native binding rejected")
	}
	for name, change := range map[string]func(*dsNativeHost){
		"source":        func(h *dsNativeHost) { h.Commit = strings.Repeat("c", 40) },
		"tree":          func(h *dsNativeHost) { h.Tree = strings.Repeat("c", 40) },
		"artifact":      func(h *dsNativeHost) { h.ArtifactSHA = strings.Repeat("c", 64) },
		"manifest":      func(h *dsNativeHost) { h.ManifestSHA = strings.Repeat("c", 64) },
		"policy":        func(h *dsNativeHost) { h.PolicySHA = strings.Repeat("c", 64) },
		"profile":       func(h *dsNativeHost) { h.Profile = "other" },
		"empty-payload": func(h *dsNativeHost) { h.FileCount = 0 },
		"scope":         func(h *dsNativeHost) { h.Status = "passed" },
		"fresh-host":    func(h *dsNativeHost) { h.FreshHost = true },
		"provider":      func(h *dsNativeHost) { h.Provider = true },
		"release":       func(h *dsNativeHost) { h.Release = true },
		"publication":   func(h *dsNativeHost) { h.Publication = true },
	} {
		t.Run(name, func(t *testing.T) {
			bad := host
			change(&bad)
			if dsNativeBinding(index, artifact, bad) == nil {
				t.Fatal("wrong native binding accepted")
			}
		})
	}
	inventory := func() map[string]any {
		return map[string]any{
			"protocol": "symphony.release.candidate.v1", "publication_authorized": false,
			"source":                map[string]any{"commit": index.Source.Commit, "tree": index.Source.Tree},
			"platform":              map[string]any{"profile_id": index.Profile.ID, "os": index.Profile.OS, "architecture": index.Profile.Architecture},
			"dependency_map_sha256": index.DependencySHA, "identity_catalog_sha256": index.IdentitySHA,
			"components": []any{map[string]any{"component_id": "component", "version": "1.0.0"}},
		}
	}
	if dsNativeInventory(index, artifact, dsJSON(inventory())) != nil {
		t.Fatal("valid native selection rejected")
	}
	for _, name := range []string{"os", "architecture", "dependency", "identity", "version", "omitted-component", "extra-component", "duplicate-component"} {
		t.Run("inventory-"+name, func(t *testing.T) {
			bad := inventory()
			switch name {
			case "os", "architecture":
				bad["platform"].(map[string]any)[name] = "wrong"
			case "dependency":
				bad["dependency_map_sha256"] = strings.Repeat("c", 64)
			case "identity":
				bad["identity_catalog_sha256"] = strings.Repeat("c", 64)
			case "version":
				bad["components"].([]any)[0].(map[string]any)["version"] = "other"
			case "omitted-component":
				bad["components"] = []any{}
			case "extra-component":
				bad["components"] = append(bad["components"].([]any), map[string]any{"component_id": "extra", "version": "1.0.0"})
			case "duplicate-component":
				bad["components"] = append(bad["components"].([]any), bad["components"].([]any)[0])
			}
			if dsNativeInventory(index, artifact, dsJSON(bad)) == nil {
				t.Fatal("wrong inventory admitted")
			}
		})
	}
}
func TestNativeVerifierSourceBinding(t *testing.T) {
	work := t.TempDir()
	root := filepath.Join(work, "delivery")
	repo := filepath.Join(work, "verifier")
	os.Mkdir(root, 0700)
	os.Mkdir(repo, 0700)
	record := dsNativeTool{Protocol: "symphony.release.native-verifier-tool.v1", GoVersion: "go1.26.5", Sources: map[string]string{}}
	for _, name := range dsNativeSources {
		data := []byte("fixture trusted source " + name)
		dsPut(t, filepath.Join(repo, name), data)
		record.Sources[name] = dsHash(data)
	}
	tool := filepath.Join(work, "candidate-host")
	dsPut(t, tool, []byte("#!/bin/sh\nexit 0\n"))
	os.Chmod(tool, 0700)
	data, _ := os.ReadFile(tool)
	record.ToolSHA = dsHash(data)
	if dsNativeToolOK(root, repo, tool, record) != nil {
		t.Fatal("valid independent verifier refused")
	}
	for _, name := range []string{"source", "missing-source", "extra-source", "tool", "toolchain", "tool-in-payload", "source-in-payload"} {
		t.Run(name, func(t *testing.T) {
			raw := dsJSON(record)
			var bad dsNativeTool
			dsDecode(raw, &bad)
			selectedRepo, selectedTool := repo, tool
			switch name {
			case "source":
				bad.Sources[dsNativeSources[0]] = strings.Repeat("a", 64)
			case "missing-source":
				delete(bad.Sources, dsNativeSources[0])
			case "extra-source":
				bad.Sources["extra"] = strings.Repeat("a", 64)
			case "tool":
				bad.ToolSHA = strings.Repeat("a", 64)
			case "toolchain":
				bad.GoVersion = "go1.27.1"
			case "tool-in-payload":
				selectedTool = filepath.Join(root, "candidate-host")
			case "source-in-payload":
				selectedRepo = root
			}
			if dsNativeToolOK(root, selectedRepo, selectedTool, bad) == nil {
				t.Fatal("unbound verifier admitted")
			}
		})
	}
}
func dsOwnerFixture(t *testing.T) (dsOptions, dsIndex, dsOwnerInspection) {
	t.Helper()
	o, s, d := dsFixture(t)
	component := "secure-identity-access-governance"
	s.Artifacts[0] = dsChoice{ID: "cli", PackageID: component, Kind: "go", Adapter: "ssiag-receipt-v2", Versions: map[string]string{component: "0.1.0-dev"}, Dependencies: []string{}}
	a := &d.Artifacts[0]
	a.Selection = s.Artifacts[0]
	for _, p := range a.Files {
		if e := os.Remove(filepath.Join(o.root, p)); e != nil {
			t.Fatal(e)
		}
	}
	a.Primary = "ssiag/libexec/symphony/" + component + "/0.1.0-dev/symphony-ssiag"
	receipt := "ssiag/share/symphony/receipts/" + component + "/0.1.0-dev/install-receipt.json"
	a.Inventory = receipt
	a.Receipts = []string{receipt}
	a.Files = []string{a.Primary, receipt}
	dsPut(t, filepath.Join(o.root, a.Primary), []byte("inert owner-binding fixture"))
	if e := os.Chmod(filepath.Join(o.root, a.Primary), 0700); e != nil {
		t.Fatal(e)
	}
	dsPut(t, filepath.Join(o.root, receipt), []byte("inert receipt-binding fixture"))
	d.Files = append(append(d.Files[:5:5], a.Files...), d.Artifacts[1].Files...)
	key := dsKey(t, filepath.Dir(o.root), "owner-fixture-key")
	pub, e := os.ReadFile(key + ".pub")
	if e != nil {
		t.Fatal(e)
	}
	dsPut(t, o.policy, append([]byte("fixture namespaces=\"symphony-delivery-set\" "), pub...))
	dsPut(t, o.selection, dsJSON(s))
	dsPut(t, o.definition, dsJSON(d))
	if _, e = dsRun(o); e != nil {
		t.Fatal(e)
	}
	if b, e := exec.Command("ssh-keygen", "-Y", "sign", "-f", key, "-n", dsNamespace, filepath.Join(o.root, "delivery-set.json")).CombinedOutput(); e != nil {
		t.Fatalf("%s %v", b, e)
	}
	raw, e := os.ReadFile(filepath.Join(o.root, "delivery-set.json"))
	if e != nil {
		t.Fatal(e)
	}
	var index dsIndex
	if e = dsDecode(raw, &index); e != nil {
		t.Fatal(e)
	}
	binary, _ := dsObserve(o.root, a.Primary)
	binary.Path = strings.TrimPrefix(binary.Path, "ssiag/")
	rec, _ := dsObserve(o.root, receipt)
	rec.Path = strings.TrimPrefix(rec.Path, "ssiag/")
	inspection := dsOwnerInspection{Protocol: "symphony.release.owner-inspection.v1", Adapter: "ssiag-receipt-v2", Component: component, Version: "0.1.0-dev", OS: index.Profile.OS, Architecture: index.Profile.Architecture, Binary: binary, Receipt: rec, ReceiptDigest: "sha256:" + strings.Repeat("a", 64), Scope: "read_only_owner_receipt"}
	o.artifact = "cli"
	o.mode = "inspect-owner"
	o.ownerTool = filepath.Join(filepath.Dir(o.root), "fixture-inspector")
	o.ownerBuild = filepath.Join(filepath.Dir(o.root), "tool-build.json")
	// This inert test double tests dispatch/binding; real owner-reader suites
	// separately install, inspect and remove via their internal owner APIs.
	script := "#!/bin/sh\nprintf '%s\\n' '" + strings.TrimSpace(string(dsJSON(inspection))) + "'\n"
	dsPut(t, o.ownerTool, []byte(script))
	if e = os.Chmod(o.ownerTool, 0700); e != nil {
		t.Fatal(e)
	}
	tool, _ := dsObserve(filepath.Dir(o.ownerTool), filepath.Base(o.ownerTool))
	build := dsOwnerBuild{Protocol: "symphony.release.owner-tool-build.v1", Adapter: inspection.Adapter, Component: component, Version: inspection.Version, OS: index.Profile.OS, Architecture: index.Profile.Architecture, Source: index.Source, SourceArchiveSHA: strings.Repeat("f", 64), ToolSHA: tool.SHA256, GoVersion: "go1.26.5"}
	dsPut(t, o.ownerBuild, dsJSON(build))
	return o, index, inspection
}
func TestDeliveryOwnerBinding(t *testing.T) {
	o, index, valid := dsOwnerFixture(t)
	if e := dsOwnerBinding(index, index.Artifacts[0], valid); e != nil {
		t.Fatal(e)
	}
	cases := map[string]func(*dsOwnerInspection){
		"adapter":                  func(i *dsOwnerInspection) { i.Adapter = "stav-receipt-v2" },
		"component":                func(i *dsOwnerInspection) { i.Component = "other" },
		"version":                  func(i *dsOwnerInspection) { i.Version = "other" },
		"os":                       func(i *dsOwnerInspection) { i.OS = "linux" },
		"arch":                     func(i *dsOwnerInspection) { i.Architecture = "amd64" },
		"scope":                    func(i *dsOwnerInspection) { i.Scope = "provider" },
		"provider-claim":           func(i *dsOwnerInspection) { i.Provider = true },
		"fresh-host-claim":         func(i *dsOwnerInspection) { i.FreshHost = true },
		"release-claim":            func(i *dsOwnerInspection) { i.Release = true },
		"publication-claim":        func(i *dsOwnerInspection) { i.Publication = true },
		"binary-path":              func(i *dsOwnerInspection) { i.Binary.Path = "../program" },
		"binary-digest":            func(i *dsOwnerInspection) { i.Binary.SHA256 = strings.Repeat("b", 64) },
		"binary-mode":              func(i *dsOwnerInspection) { i.Binary.Mode = 0600 },
		"binary-size":              func(i *dsOwnerInspection) { i.Binary.Size++ },
		"receipt-path":             func(i *dsOwnerInspection) { i.Receipt.Path = "other/install-receipt.json" },
		"receipt-digest":           func(i *dsOwnerInspection) { i.Receipt.SHA256 = strings.Repeat("b", 64) },
		"receipt-mode":             func(i *dsOwnerInspection) { i.Receipt.Mode = 0644 },
		"receipt-canonical-digest": func(i *dsOwnerInspection) { i.ReceiptDigest = "sha256:bad" },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			bad := valid
			change(&bad)
			if dsOwnerBinding(index, index.Artifacts[0], bad) == nil {
				t.Fatal("mismatch accepted")
			}
		})
	}
	r, e := dsInspectOwner(o)
	if e != nil || !r.OwnerVerified || r.FreshHost || r.Release || r.Publication || r.ArtifactID != "cli" {
		t.Fatal(r, e)
	}
}
func TestDeliveryOwnerRefusesUntrustedOrChangedInputs(t *testing.T) {
	for _, name := range []string{"missing-signature", "wrong-tool", "wrong-source", "wrong-toolchain", "wrong-tool-version", "unhandled-adapter", "unknown-artifact", "duplicate-output", "payload-drift", "tool-in-payload", "missing-receipt"} {
		t.Run(name, func(t *testing.T) {
			o, index, _ := dsOwnerFixture(t)
			switch name {
			case "missing-signature":
				os.Remove(filepath.Join(o.root, "delivery-set.json.sig"))
			case "wrong-tool":
				dsPut(t, o.ownerTool, []byte("#!/bin/sh\nexit 0\n"))
			case "wrong-source", "wrong-toolchain", "wrong-tool-version":
				b, _ := os.ReadFile(o.ownerBuild)
				var build dsOwnerBuild
				dsDecode(b, &build)
				if name == "wrong-source" {
					build.Source.Commit = strings.Repeat("c", 40)
				}
				if name == "wrong-toolchain" {
					build.GoVersion = "go1.27.1"
				}
				if name == "wrong-tool-version" {
					build.Version = "other"
				}
				dsPut(t, o.ownerBuild, dsJSON(build))
			case "unhandled-adapter":
				o.artifact = "carrier"
			case "unknown-artifact":
				o.artifact = "absent"
			case "duplicate-output":
				o.output = filepath.Join(filepath.Dir(o.root), "exists.json")
				dsPut(t, o.output, []byte("preserve"))
			case "payload-drift":
				// A successful-looking owner cannot change payloads and retain the old proof.
				b, _ := os.ReadFile(o.ownerTool)
				script := append([]byte("#!/bin/sh\nprintf 'changed' > \"$2\"\n"), bytes.TrimPrefix(b, []byte("#!/bin/sh\n"))...)
				dsPut(t, o.ownerTool, script)
				os.Chmod(o.ownerTool, 0700)
				br, _ := os.ReadFile(o.ownerBuild)
				var build dsOwnerBuild
				dsDecode(br, &build)
				build.ToolSHA = dsHash(script)
				dsPut(t, o.ownerBuild, dsJSON(build))
			case "tool-in-payload":
				o.ownerTool = filepath.Join(o.root, index.Artifacts[0].Primary)
			case "missing-receipt":
				os.Remove(filepath.Join(o.root, index.Artifacts[0].Receipts[0]))
			}
			if _, e := dsInspectOwner(o); e == nil {
				t.Fatal("untrusted or stale owner proof accepted")
			}
		})
	}
}
