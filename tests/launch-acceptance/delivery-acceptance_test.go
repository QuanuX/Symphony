package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type daFixtureData struct {
	p             gatePlan
	plan, mapping []byte
	m             daMapping
	r             daRequirements
	v1            gatePacket
	v             daPacket
	targets       daTargets
	root          string
	indexes       map[string]dsIndex
}

func daPut(t *testing.T, root, name string, v any) *evidenceRef {
	t.Helper()
	b, ok := v.([]byte)
	if !ok {
		b = dsJSON(v)
	}
	dsPut(t, filepath.Join(root, name), b)
	return &evidenceRef{Path: name, SHA256: dsHash(b)}
}
func daInputs(t *testing.T) daFixtureData {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Dir(file)
	p, plan, e := loadGatePlan(filepath.Join(root, "plan.json"))
	if e != nil {
		t.Fatal(e)
	}
	mapping, e := os.ReadFile(filepath.Join(root, "../../docs/public-beta/acceptance-mapping.json"))
	if e != nil {
		t.Fatal(e)
	}
	m, e := daLoadMapping(mapping, p)
	if e != nil {
		t.Fatal(e)
	}
	r := daNewRequirements(p, plan, mapping, m)
	core := newPacket(p, plan)
	targets := daTargets{Protocol: "symphony.release.delivery-targets.v1", Profiles: map[string]*daTarget{}}
	for _, id := range p.Profiles {
		targets.Profiles[id] = nil
	}
	return daFixtureData{p: p, plan: plan, mapping: mapping, m: m, r: r, v1: core, v: daNewPacket(p, plan, mapping, dsJSON(r), dsJSON(core), r, m), targets: targets, root: t.TempDir(), indexes: map[string]dsIndex{}}
}
func (f *daFixtureData) check() (daResult, error) {
	return daCheck(f.p, f.plan, f.mapping, dsJSON(f.r), dsJSON(f.v1), dsJSON(f.v), f.m, f.r, f.targets, f.root, f.root)
}
func TestDeliveryAcceptancePendingRetainsWholeMatrix(t *testing.T) {
	f := daInputs(t)
	if daRequirementsOK(f.r, f.p, f.plan, f.mapping, f.m) != nil {
		t.Fatal("template requirements")
	}
	r, e := f.check()
	if e != nil || r.Ready || r.Core.EvidenceReady || r.Release || r.Publication || len(f.v.Criteria) != 58 || len(f.v.Supplemental) != 31 || len(f.v.Profiles) != 4 || len(r.Missing) < 120 {
		t.Fatal(r, e)
	}
}
func daSignedNativeSet(t *testing.T, profile string) (daTarget, dsIndex) {
	t.Helper()
	work := t.TempDir()
	root := filepath.Join(work, "delivery")
	if e := os.Mkdir(root, 0700); e != nil {
		t.Fatal(e)
	}
	target := dsProfile{ID: profile, OS: "linux", Architecture: "amd64"}
	if strings.HasPrefix(profile, "macos") {
		target.OS = "macos"
	}
	if strings.HasSuffix(profile, "arm64") {
		target.Architecture = "arm64"
	}
	source := dsSource{Commit: strings.Repeat("a", 40), Tree: strings.Repeat("b", 40)}
	selection := dsSelection{Protocol: "symphony.release.delivery-selection.v1", Source: source, Profile: target, CapabilitySHA: strings.Repeat("c", 64), DependencySHA: strings.Repeat("d", 64), IdentitySHA: strings.Repeat("e", 64), Artifacts: []dsChoice{}}
	d := dsDefinition{SetID: "synthetic-acceptance-fixture", Files: []string{}, Artifacts: []dsArtifact{}}
	for _, id := range []string{"base", "extra"} {
		dep := []string{}
		if id == "extra" {
			dep = []string{"base"}
		}
		choice := dsChoice{ID: id, PackageID: id, Kind: "native", Adapter: "native-candidate-v1", Versions: map[string]string{"fixture-" + id: "1.0.0"}, Dependencies: dep}
		if profile == "macos-intel-supplemental" && id == "extra" {
			choice.Versions = map[string]string{"sov-cloudflare-adapter": "fixture-only"}
		}
		manifest, sig, policy := id+"/candidate.json", id+"/candidate.json.sig", strings.Repeat("f", 64)
		a := dsArtifact{Selection: choice, Source: &source, Files: []string{}, Primary: id + "/candidate.tar", Inventory: manifest, Manifest: &manifest, Signature: &sig, SignerPolicySHA: &policy, Receipts: []string{}, InstallRecipe: id + "/build-evidence.json", RemoveRecipe: id + "/build-evidence.json", Notices: []string{id + "/platform-evidence.json"}, SourceMaterial: []string{id + "/dependency-map.json"}, BuildEvidence: []string{id + "/build-evidence.json"}}
		for _, name := range []string{"candidate.tar", "candidate.json", "candidate.json.sig", "platform-evidence.json", "build-evidence.json", "dependency-map.json", "identity-catalog.json", "SHA256SUMS"} {
			p := id + "/" + name
			dsPut(t, filepath.Join(root, p), []byte("Synthetic acceptance binding fixture; not a native candidate: "+p))
			a.Files = append(a.Files, p)
			d.Files = append(d.Files, p)
		}
		d.Artifacts = append(d.Artifacts, a)
		selection.Artifacts = append(selection.Artifacts, choice)
	}
	key := dsKey(t, work, "fixture-key")
	pub, e := os.ReadFile(key + ".pub")
	if e != nil {
		t.Fatal(e)
	}
	o := dsOptions{mode: "prepare", root: root, selection: filepath.Join(work, "selection.json"), definition: filepath.Join(work, "definition.json"), policy: filepath.Join(work, "allowed-signers"), profile: profile, signer: "fixture"}
	dsPut(t, o.policy, append([]byte("fixture namespaces=\"symphony-delivery-set\" "), pub...))
	dsPut(t, o.selection, dsJSON(selection))
	dsPut(t, o.definition, dsJSON(d))
	if _, e = dsRun(o); e != nil {
		t.Fatal(e)
	}
	if b, e := exec.Command("ssh-keygen", "-Y", "sign", "-f", key, "-n", dsNamespace, filepath.Join(root, "delivery-set.json")).CombinedOutput(); e != nil {
		t.Fatal(string(b), e)
	}
	raw, e := os.ReadFile(filepath.Join(root, "delivery-set.json"))
	if e != nil {
		t.Fatal(e)
	}
	var index dsIndex
	if dsDecode(raw, &index) != nil {
		t.Fatal("fixture index")
	}
	return daTarget{Root: root, Selection: o.selection, Policy: o.policy, Signer: "fixture"}, index
}
func daCompleteFixture(t *testing.T) daFixtureData {
	t.Helper()
	f := daInputs(t)
	ids := []string{"base", "extra"}
	extra := "macos-intel-supplemental"
	f.r.Profiles[extra] = daRequiredProfile{Artifacts: &ids, Scenarios: map[string]*[]string{}}
	for _, profile := range f.p.Profiles {
		q := f.r.Profiles[profile]
		q.Artifacts = &ids
		for scenario := range q.Scenarios {
			q.Scenarios[scenario] = &ids
		}
		f.r.Profiles[profile] = q
	}
	for id := range f.r.Supplemental {
		required := map[string][]string{extra: ids}
		f.r.Supplemental[id] = &required
	}
	f.v = daNewPacket(f.p, f.plan, f.mapping, dsJSON(f.r), dsJSON(f.v1), f.r, f.m)
	c := daChecker{root: f.root, indexes: f.indexes, result: daResult{Indexes: map[string]string{}}}
	observation := func(id string, required map[string][]string, cells []string) *evidenceRef {
		b, ready, e := c.bindings(required)
		if e != nil || !ready {
			t.Fatal(e)
		}
		attachment := daPut(t, f.root, "synthetic.txt", []byte("Synthetic checker test only: not operational, live, platform or release evidence"))
		// The checker validates structure; this deliberate assertion is a fixture
		// testing review readiness. Independent human review must reject it as real.
		o := daObservation{Protocol: "symphony.release.delivery-observation.v1", ID: id, Result: "passed", Scope: "operational", Bindings: b, CoreCells: cells, Evidence: []evidenceRef{*attachment}, Observer: "synthetic-test", ObservedAt: "2026-11-13T00:00:00Z", Expected: "synthetic binding match", Actual: "synthetic binding match"}
		return daPut(t, f.root, strings.ReplaceAll(id, ":", "_")+".json", o)
	}
	for _, profile := range daSorted(f.r.Profiles) {
		target, index := daSignedNativeSet(t, profile)
		f.targets.Profiles[profile] = &target
		f.indexes[profile] = index
		raw, _ := os.ReadFile(filepath.Join(target.Root, "delivery-set.json"))
		c.result.Indexes[profile] = dsHash(raw)
		pv := f.v.Profiles[profile]
		pv.Index = daPut(t, f.root, profile+"-index.json", raw)
		base := "base"
		if daContains(f.p.Profiles, profile) {
			pv.Primary = &base
		}
		for _, a := range index.Artifacts {
			identity := daIdentityFor(index, a)
			files := map[string]dsFile{}
			for _, file := range index.Files {
				files[file.Path] = file
			}
			h := dsNativeHost{Protocol: "symphony.release.host-preparation.v1", Status: "verified_extraction_only", Commit: index.Source.Commit, Tree: index.Source.Tree, ArtifactSHA: identity.Primary.SHA256, ManifestSHA: identity.Inventory.SHA256, PolicySHA: *a.SignerPolicySHA, Profile: profile, FileCount: 1}
			tool := dsNativeTool{Protocol: "symphony.release.native-verifier-tool.v1", ToolSHA: strings.Repeat("1", 64), GoVersion: "go1.26.5", Sources: map[string]string{}}
			for _, p := range dsNativeSources {
				tool.Sources[p] = strings.Repeat("2", 64)
			}
			proof := daNativeProof{Protocol: "symphony.release.candidate-verification.v1", Scope: "caller_supplied_payload", VerifiedAt: "2026-11-13T00:00:00Z", Commit: index.Source.Commit, Tree: index.Source.Tree, Profile: profile, ArtifactSHA: h.ArtifactSHA, ManifestSHA: h.ManifestSHA, SignatureSHA: files[*a.Signature].SHA256, PolicySHA: h.PolicySHA, Scheme: "sshsig", Namespace: "symphony-release", Principal: "fixture", Key: "SHA256:synthetic"}
			prefix := profile + "-" + a.Selection.ID
			hr := daPut(t, f.root, prefix+"-host.json", h)
			tr := daPut(t, f.root, prefix+"-tool.json", tool)
			pr := daPut(t, f.root, prefix+"-proof.json", proof)
			owner := dsNativeResult{Protocol: "symphony.release.delivery-native-verification.v1", Scope: "selected_native_extraction_and_receipts", IndexSHA: dsHash(raw), ArtifactID: a.Selection.ID, Source: index.Source, Profile: index.Profile, ArtifactSHA: h.ArtifactSHA, ManifestSHA: h.ManifestSHA, PolicySHA: h.PolicySHA, ToolSHA: tool.ToolSHA, ToolRecordSHA: tr.SHA256, ToolSources: tool.Sources, HostReportSHA: hr.SHA256, PayloadReportSHA: pr.SHA256, PayloadFileCount: 1, OwnerVerified: true}
			pv.Artifacts[a.Selection.ID] = daArtifactEvidence{Owner: &daOwner{Report: *daPut(t, f.root, prefix+"-owner.json", owner), Support: []evidenceRef{*hr, *tr, *pr}}}
			if a.Selection.ID == "base" && daContains(f.p.Profiles, profile) {
				evidence := daPut(t, f.root, "synthetic.txt", []byte("Synthetic checker test only: not operational, live, platform or release evidence"))
				q := f.v1.Profiles[profile]
				q.SourceCommit = h.Commit
				q.ArtifactSHA256 = h.ArtifactSHA
				q.ManifestSHA256 = h.ManifestSHA
				q.SignerPolicySHA256 = h.PolicySHA
				q.VerifiedPayload = pr
				q.FreshHostInventory = evidence
				q.RuntimeNoticeReview = evidence
				for scenario := range q.Scenarios {
					record := map[string]any{"profile_id": profile, "scenario": scenario, "source_commit": h.Commit, "artifact_sha256": h.ArtifactSHA, "result": "passed", "scope": "operational", "skips": 0, "evidence": []*evidenceRef{evidence}}
					q.Scenarios[scenario] = scenarioEvidence{Result: "passed", Scope: "operational", Record: daPut(t, f.root, profile+"-v1-"+scenario+".json", record)}
				}
				f.v1.Profiles[profile] = q
			}
		}
		f.v.Profiles[profile] = pv
	}
	for _, profile := range daSorted(f.r.Profiles) {
		pv := f.v.Profiles[profile]
		for id, a := range pv.Artifacts {
			required := map[string][]string{profile: {id}}
			if id == "extra" {
				required[profile] = ids
			}
			a.Installation = observation(profile+":"+id+":installation", required, []string{})
			a.Loader = observation(profile+":"+id+":loader", required, []string{})
			a.Trust = observation(profile+":"+id+":trust", required, []string{})
			pv.Artifacts[id] = a
		}
		for scenario := range pv.Scenarios {
			label := profile + "::" + scenario
			pv.Scenarios[scenario] = observation(label, map[string][]string{profile: ids}, []string{label})
		}
		f.v.Profiles[profile] = pv
	}
	for id, campaigns := range f.m.Criteria {
		required := map[string][]string{}
		cells := []string{}
		for _, profile := range f.p.Profiles {
			required[profile] = ids
			for _, campaign := range campaigns {
				cells = append(cells, profile+"::"+f.m.Campaigns[campaign])
			}
		}
		f.v.Criteria[id] = observation(id, required, cells)
	}
	for id, required := range f.r.Supplemental {
		f.v.Supplemental[id] = observation(id, *required, []string{})
	}
	evidence := daPut(t, f.root, "synthetic.txt", []byte("Synthetic checker test only: not operational, live, platform or release evidence"))
	for id := range f.v1.Inputs {
		f.v1.Inputs[id] = evidence
	}
	f.v.CoreSHA = dsHash(dsJSON(f.v1))
	return f
}
func TestDeliveryAcceptanceCompleteStructureNeverApprovesRelease(t *testing.T) {
	f := daCompleteFixture(t)
	if e := daRequirementsOK(f.r, f.p, f.plan, f.mapping, f.m); e != nil {
		t.Fatal(e)
	}
	r, e := f.check()
	if e != nil || !r.Ready || !r.Core.EvidenceReady || r.Release || r.Publication || len(r.Indexes) != 5 {
		t.Fatal(r, e)
	}
}
func TestDeliveryAcceptanceMutationRefusals(t *testing.T) {
	// Reuse immutable signed fixture bytes; each mutation gets its own packet and
	// restores any changed observation. No parallel mutation of shared fixtures.
	f := daCompleteFixture(t)
	original := dsJSON(f.v)
	originalCore := dsJSON(f.v1)
	originalRequirements := dsJSON(f.r)
	for _, name := range []string{"missing-profile", "missing-scenario", "missing-secondary", "extra-artifact", "missing-criterion", "missing-supplement", "wrong-core-hash", "wrong-plan-hash", "wrong-map-hash", "wrong-requirements-hash", "acceptance-claim", "publication-claim", "primary-is-secondary-digest", "primary-is-artifact-id", "primary-selects-other-artifact", "owner-wrong-artifact", "owner-wrong-tree", "owner-wrong-index", "owner-publish", "owner-support-missing", "owner-native-proof-wrong", "scenario-wrong-artifact", "scenario-missing-artifact", "scenario-wrong-index", "scenario-wrong-cell", "criterion-wrong-cells", "empty-evidence", "evidence-traversal", "evidence-symlink", "evidence-hardlink", "evidence-group-write", "observation-null-flag", "observation-case-alias", "requirement-empty-list", "requirement-unknown-artifact", "requirement-duplicate", "missing-target-profile", "unsigned-target", "different-source"} {
		t.Run(name, func(t *testing.T) {
			if dsDecode(original, &f.v) != nil || dsDecode(originalCore, &f.v1) != nil || dsDecode(originalRequirements, &f.r) != nil {
				t.Fatal("reset")
			}
			profile := f.p.Profiles[0]
			pv := f.v.Profiles[profile]
			scenario := f.p.Scenarios[0]
			var changedPath string
			var before []byte
			change := func(ref *evidenceRef, fn func(map[string]any)) {
				changedPath = filepath.Join(f.root, ref.Path)
				before, _ = os.ReadFile(changedPath)
				var v map[string]any
				json.Unmarshal(before, &v)
				fn(v)
				b := dsJSON(v)
				dsPut(t, changedPath, b)
				ref.SHA256 = dsHash(b)
			}
			defer func() {
				if changedPath != "" {
					os.Remove(changedPath)
					dsPut(t, changedPath, before)
				}
			}()
			switch name {
			case "missing-profile":
				delete(f.v.Profiles, profile)
			case "missing-scenario":
				delete(pv.Scenarios, scenario)
			case "missing-secondary":
				delete(pv.Artifacts, "extra")
			case "extra-artifact":
				pv.Artifacts["invented"] = daArtifactEvidence{}
			case "missing-criterion":
				delete(f.v.Criteria, "UE1")
			case "missing-supplement":
				delete(f.v.Supplemental, f.m.Supplemental[0])
			case "wrong-core-hash":
				f.v.CoreSHA = strings.Repeat("0", 64)
			case "wrong-plan-hash":
				f.v.PlanSHA = strings.Repeat("0", 64)
			case "wrong-map-hash":
				f.v.MappingSHA = strings.Repeat("0", 64)
			case "wrong-requirements-hash":
				f.v.RequirementsSHA = strings.Repeat("0", 64)
			case "acceptance-claim":
				f.v.Release = true
			case "publication-claim":
				f.v.Publication = true
			case "primary-selects-other-artifact":
				other := "extra"
				pv.Primary = &other
				f.v.Profiles[profile] = pv
			case "primary-is-secondary-digest":
				q := f.v1.Profiles[profile]
				q.ArtifactSHA256 = strings.Repeat("9", 64)
				f.v1.Profiles[profile] = q
				f.v.CoreSHA = dsHash(dsJSON(f.v1))
			case "primary-is-artifact-id":
				q := f.v1.Profiles[profile]
				q.ArtifactSHA256 = "base"
				f.v1.Profiles[profile] = q
				f.v.CoreSHA = dsHash(dsJSON(f.v1))
			case "owner-support-missing":
				a := pv.Artifacts["base"]
				a.Owner.Support = []evidenceRef{}
			case "owner-wrong-artifact", "owner-wrong-tree", "owner-wrong-index", "owner-publish":
				ref := &pv.Artifacts["base"].Owner.Report
				change(ref, func(m map[string]any) {
					switch name {
					case "owner-wrong-artifact":
						m["artifact_id"] = "extra"
					case "owner-wrong-tree":
						m["source"].(map[string]any)["tree"] = strings.Repeat("9", 40)
					case "owner-wrong-index":
						m["index_sha256"] = strings.Repeat("9", 64)
					case "owner-publish":
						m["publication_authorized"] = true
					}
				})
			case "owner-native-proof-wrong":
				a := pv.Artifacts["base"]
				ref := &a.Owner.Support[2]
				change(ref, func(m map[string]any) { m["artifact_sha256"] = strings.Repeat("9", 64) })
			case "scenario-wrong-artifact", "scenario-missing-artifact", "scenario-wrong-index", "scenario-wrong-cell", "empty-evidence", "observation-null-flag", "observation-case-alias":
				change(pv.Scenarios[scenario], func(m map[string]any) {
					binding := m["bindings"].(map[string]any)[profile].(map[string]any)
					switch name {
					case "scenario-wrong-artifact":
						binding["artifacts"].(map[string]any)["base"].(map[string]any)["primary"].(map[string]any)["sha256"] = strings.Repeat("9", 64)
					case "scenario-missing-artifact":
						delete(binding["artifacts"].(map[string]any), "extra")
					case "scenario-wrong-index":
						binding["delivery_index_sha256"] = strings.Repeat("9", 64)
					case "scenario-wrong-cell":
						m["core_cells"] = []string{}
					case "empty-evidence":
						m["evidence"] = []any{}
					case "observation-null-flag":
						m["publication_authorized"] = nil
					case "observation-case-alias":
						m["Publication_authorized"] = false
						delete(m, "publication_authorized")
					}
				})
			case "criterion-wrong-cells":
				change(f.v.Criteria["UE1"], func(m map[string]any) { m["core_cells"] = []string{} })
			case "evidence-traversal":
				pv.Scenarios[scenario].Path = "../escape"
			case "evidence-symlink", "evidence-hardlink", "evidence-group-write":
				ref := pv.Scenarios[scenario]
				changedPath = filepath.Join(f.root, ref.Path)
				before, _ = os.ReadFile(changedPath)
				other := filepath.Join(t.TempDir(), "other")
				dsPut(t, other, before)
				os.Remove(changedPath)
				switch name {
				case "evidence-symlink":
					os.Symlink(other, changedPath)
				case "evidence-hardlink":
					os.Link(other, changedPath)
				case "evidence-group-write":
					os.WriteFile(changedPath, before, 0600)
					if e := os.Chmod(changedPath, 0660); e != nil {
						t.Fatal(e)
					}
				}
			case "requirement-empty-list", "requirement-unknown-artifact", "requirement-duplicate":
				q := f.r.Profiles[profile]
				ids := []string{}
				if name == "requirement-unknown-artifact" {
					ids = []string{"missing"}
				}
				if name == "requirement-duplicate" {
					ids = []string{"base", "base"}
				}
				q.Scenarios[scenario] = &ids
				f.r.Profiles[profile] = q
				if daRequirementsOK(f.r, f.p, f.plan, f.mapping, f.m) == nil {
					t.Fatal("requirements accepted")
				}
				return
			case "missing-target-profile":
				saved := f.targets.Profiles[profile]
				delete(f.targets.Profiles, profile)
				defer func() { f.targets.Profiles[profile] = saved }()
			case "unsigned-target":
				sig := filepath.Join(f.targets.Profiles[profile].Root, "delivery-set.json.sig")
				b, _ := os.ReadFile(sig)
				os.Remove(sig)
				defer dsPut(t, sig, b)
			case "different-source":
				index := f.indexes[profile]
				index.Source.Tree = strings.Repeat("9", 40)
				p := filepath.Join(f.targets.Profiles[profile].Root, "delivery-set.json")
				b, _ := os.ReadFile(p)
				dsPut(t, p, dsJSON(index))
				defer dsPut(t, p, b)
			}
			if _, e := f.check(); e == nil {
				t.Fatal("unsafe/mismatched evidence admitted")
			}
		})
	}
}
func TestDeliveryAcceptanceIncompleteCannotUpgradeCore(t *testing.T) {
	f := daCompleteFixture(t)
	original := dsJSON(f.v)
	originalCore := dsJSON(f.v1)
	for _, name := range []string{"missing-owner", "missing-installation", "missing-loader", "missing-trust", "missing-criterion", "missing-supplemental", "missing-target", "core-pending", "fixture", "failed", "skipped"} {
		t.Run(name, func(t *testing.T) {
			dsDecode(original, &f.v)
			dsDecode(originalCore, &f.v1)
			profile := f.p.Profiles[0]
			pv := f.v.Profiles[profile]
			a := pv.Artifacts["extra"]
			var path string
			var before []byte
			switch name {
			case "missing-owner":
				a.Owner = nil
			case "missing-installation":
				a.Installation = nil
			case "missing-loader":
				a.Loader = nil
			case "missing-trust":
				a.Trust = nil
			case "missing-criterion":
				f.v.Criteria["UE1"] = nil
			case "missing-supplemental":
				f.v.Supplemental[f.m.Supplemental[0]] = nil
			case "missing-target":
				saved := f.targets.Profiles[profile]
				f.targets.Profiles[profile] = nil
				defer func() { f.targets.Profiles[profile] = saved }()
			case "core-pending":
				f.v1.Inputs[f.p.Inputs[0]] = nil
				f.v.CoreSHA = dsHash(dsJSON(f.v1))
			default:
				ref := pv.Scenarios[f.p.Scenarios[0]]
				path = filepath.Join(f.root, ref.Path)
				before, _ = os.ReadFile(path)
				var o daObservation
				dsDecode(before, &o)
				if name == "fixture" {
					o.Scope = "fixture"
				}
				if name == "failed" {
					o.Result = "failed"
				}
				if name == "skipped" {
					o.Skips = 1
				}
				b := dsJSON(o)
				dsPut(t, path, b)
				ref.SHA256 = dsHash(b)
				defer dsPut(t, path, before)
			}
			pv.Artifacts["extra"] = a
			r, e := f.check()
			if e != nil || r.Ready || r.Release || r.Publication || len(r.Missing) == 0 {
				t.Fatal(r, e)
			}
		})
	}
}
func TestDeliveryAcceptanceActualOwnerReportShape(t *testing.T) {
	o, index, inspection := dsOwnerFixture(t)
	result, e := dsInspectOwner(o)
	if e != nil {
		t.Fatal(e)
	}
	root := t.TempDir()
	build, e := os.ReadFile(o.ownerBuild)
	if e != nil {
		t.Fatal(e)
	}
	owner := daOwner{Report: *daPut(t, root, "owner.json", result), Support: []evidenceRef{*daPut(t, root, "build.json", build), *daPut(t, root, "inspection.json", inspection)}}
	c := daChecker{root: root, indexes: map[string]dsIndex{index.Profile.ID: index}, result: daResult{Indexes: map[string]string{index.Profile.ID: result.IndexSHA}}}
	if e = c.owner(index.Profile.ID, o.artifact, &owner); e != nil {
		t.Fatal(e)
	}
	result.Inspection.ReceiptDigest = "sha256:" + strings.Repeat("0", 64)
	owner.Report = *daPut(t, root, "mismatch.json", result)
	if c.owner(index.Profile.ID, o.artifact, &owner) == nil {
		t.Fatal("embedded/raw owner mismatch admitted")
	}
}
func TestDeliveryAcceptanceCLIInitializationAndImmutableOutputs(t *testing.T) {
	f := daInputs(t)
	work := f.root
	plan := filepath.Join(work, "plan.json")
	mapping := filepath.Join(work, "mapping.json")
	requirements := filepath.Join(work, "requirements.json")
	core := filepath.Join(work, "core.json")
	packet := filepath.Join(work, "packet.json")
	targets := filepath.Join(work, "targets.json")
	report := filepath.Join(work, "report.json")
	dsPut(t, plan, f.plan)
	dsPut(t, mapping, f.mapping)
	o := daOptions{mode: "init-requirements", plan: plan, mapping: mapping, requirements: requirements}
	if _, e := daRun(o); e != nil {
		t.Fatal(e)
	}
	if _, e := daRun(o); e == nil {
		t.Fatal("requirements overwritten")
	}
	dsPut(t, core, dsJSON(f.v1))
	o.mode = "init"
	o.core = core
	o.packet = packet
	if _, e := daRun(o); e != nil {
		t.Fatal(e)
	}
	if _, e := daRun(o); e == nil {
		t.Fatal("packet overwritten")
	}
	dsPut(t, targets, dsJSON(f.targets))
	o.mode = "check"
	o.targets = targets
	o.report = report
	r, e := daRun(o)
	if e != nil || r.Ready || r.Release || r.Publication {
		t.Fatal(r, e)
	}
	if _, e := daRun(o); e == nil {
		t.Fatal("report overwritten")
	}
}
func TestDeliveryAcceptanceMappingCannotDropRequiredCells(t *testing.T) {
	f := daInputs(t)
	for _, key := range []string{"required_profiles", "required_scenarios", "campaigns", "core_cells", "paper_criteria", "cloudflare_operation_cells", "additional_campaigns"} {
		t.Run(key, func(t *testing.T) {
			var m map[string]any
			json.Unmarshal(f.mapping, &m)
			a := m[key].([]any)
			m[key] = a[:len(a)-1]
			if _, e := daLoadMapping(dsJSON(m), f.p); e == nil {
				t.Fatal("coverage silently reduced")
			}
		})
	}
}
