package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func gateFixture(t *testing.T) (gatePlan, []byte, gatePacket, string) {
	t.Helper()
	root := t.TempDir()
	plan := gatePlan{Format: 1, Profiles: []string{"fixture-profile"}, Inputs: []string{"review"}, Scenarios: []string{"provider"}}
	raw, _ := json.Marshal(plan)
	packet := newPacket(plan, raw)
	source := strings.Repeat("a", 40)
	hash := strings.Repeat("b", 64)
	put := func(name string, b []byte) *evidenceRef {
		if e := os.WriteFile(filepath.Join(root, name), b, 0600); e != nil {
			t.Fatal(e)
		}
		return &evidenceRef{Path: name, SHA256: gateHash(b)}
	}
	evidence := put("observation.txt", []byte("synthetic gate test only; not operational evidence"))
	proof, _ := json.Marshal(map[string]any{"protocol": "symphony.release.candidate-verification.v1", "scope": "caller_supplied_payload", "profile_id": "fixture-profile", "source_commit": source, "artifact_sha256": hash, "inventory_sha256": hash, "allowed_signers_sha256": hash})
	record, _ := json.Marshal(map[string]any{"profile_id": "fixture-profile", "scenario": "provider", "source_commit": source, "artifact_sha256": hash, "result": "passed", "scope": "operational", "skips": 0, "evidence": []*evidenceRef{evidence}})
	packet.Inputs["review"] = evidence
	packet.Profiles["fixture-profile"] = profileEvidence{SourceCommit: source, ArtifactSHA256: hash, ManifestSHA256: hash, SignerPolicySHA256: hash, VerifiedPayload: put("proof.json", proof), FreshHostInventory: evidence, RuntimeNoticeReview: evidence, Scenarios: map[string]scenarioEvidence{"provider": {Result: "passed", Scope: "operational", Record: put("scenario.json", record)}}}
	return plan, raw, packet, root
}
func TestGateCompleteStructureNeverApprovesPublication(t *testing.T) {
	p, plan, packet, root := gateFixture(t)
	b, _ := json.Marshal(packet)
	r, e := checkPacket(p, plan, b, root)
	if e != nil || !r.EvidenceReady || r.ReleaseAccepted || r.Publication {
		t.Fatal(r, e)
	}
}
func TestGateCurrentPlanTemplateRemainsIncomplete(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	p, raw, e := loadGatePlan(filepath.Join(filepath.Dir(file), "plan.json"))
	if e != nil {
		t.Fatal(e)
	}
	packet := newPacket(p, raw)
	if len(packet.Profiles) != 4 || len(p.Scenarios) != 9 {
		t.Fatal("review changed acceptance matrix")
	}
	b, _ := json.Marshal(packet)
	r, e := checkPacket(p, raw, b, t.TempDir())
	if e != nil || r.EvidenceReady || r.ReleaseAccepted || len(r.Missing) == 0 {
		t.Fatal(r, e)
	}
}
func TestGateRefusesDriftAndUnsafeEvidence(t *testing.T) {
	for _, name := range []string{"plan-drift", "missing-profile", "extra-profile", "missing-scenario", "wrong-artifact", "changed-file", "symlink", "traversal", "unknown-field", "duplicate-field", "negative-skips", "empty-attachments"} {
		t.Run(name, func(t *testing.T) {
			p, plan, packet, root := gateFixture(t)
			q := packet.Profiles["fixture-profile"]
			switch name {
			case "plan-drift":
				packet.PlanSHA256 = strings.Repeat("c", 64)
			case "missing-profile":
				delete(packet.Profiles, "fixture-profile")
			case "extra-profile":
				packet.Profiles["unexpected"] = q
			case "missing-scenario":
				delete(q.Scenarios, "provider")
			case "wrong-artifact":
				q.ArtifactSHA256 = strings.Repeat("d", 64)
				packet.Profiles["fixture-profile"] = q
			case "changed-file":
				os.WriteFile(filepath.Join(root, "observation.txt"), []byte("changed"), 0600)
			case "symlink":
				os.Rename(filepath.Join(root, "observation.txt"), filepath.Join(root, "real.txt"))
				os.Symlink("real.txt", filepath.Join(root, "observation.txt"))
			case "traversal":
				packet.Inputs["review"].Path = "../escape"
			case "negative-skips":
				s := q.Scenarios["provider"]
				s.Skips = -1
				q.Scenarios["provider"] = s
			case "empty-attachments":
				b, _ := os.ReadFile(filepath.Join(root, "scenario.json"))
				var v map[string]any
				json.Unmarshal(b, &v)
				v["evidence"] = []any{}
				b, _ = json.Marshal(v)
				os.WriteFile(filepath.Join(root, "scenario.json"), b, 0600)
				s := q.Scenarios["provider"]
				s.Record.SHA256 = gateHash(b)
			}
			b, _ := json.Marshal(packet)
			if name == "unknown-field" {
				b = []byte(strings.Replace(string(b), "{", `{"unexpected":true,`, 1))
			}
			if name == "duplicate-field" {
				b = []byte(strings.Replace(string(b), "{", `{"format_version":1,`, 1))
			}
			if _, e := checkPacket(p, plan, b, root); e == nil {
				t.Fatal("invalid evidence admitted")
			}
		})
	}
}
func TestGateFixturesFailuresAndSkipsRemainIncomplete(t *testing.T) {
	for _, name := range []string{"fixture", "failed", "skip"} {
		t.Run(name, func(t *testing.T) {
			p, plan, packet, root := gateFixture(t)
			q := packet.Profiles["fixture-profile"]
			s := q.Scenarios["provider"]
			switch name {
			case "fixture":
				s.Scope = "fixture"
			case "failed":
				s.Result = "failed"
			case "skip":
				s.Skips = 1
			}
			b, _ := os.ReadFile(filepath.Join(root, s.Record.Path))
			var v map[string]any
			json.Unmarshal(b, &v)
			v["scope"] = s.Scope
			v["result"] = s.Result
			v["skips"] = s.Skips
			b, _ = json.Marshal(v)
			os.WriteFile(filepath.Join(root, s.Record.Path), b, 0600)
			s.Record.SHA256 = gateHash(b)
			q.Scenarios["provider"] = s
			packet.Profiles["fixture-profile"] = q
			b, _ = json.Marshal(packet)
			r, e := checkPacket(p, plan, b, root)
			if e != nil || r.EvidenceReady || len(r.Missing) == 0 {
				t.Fatal(r, e)
			}
		})
	}
}
func TestGateOutputsCannotOverwriteExistingEvidence(t *testing.T) {
	p := filepath.Join(t.TempDir(), "report.json")
	if e := writeNew(p, map[string]bool{"original": true}); e != nil {
		t.Fatal(e)
	}
	before, _ := os.ReadFile(p)
	if e := writeNew(p, nil); e == nil {
		t.Fatal("overwrote output")
	}
	after, _ := os.ReadFile(p)
	if string(before) != string(after) {
		t.Fatal("changed evidence")
	}
}

func TestGateRejectsDeepJSONAndEmptyAttachments(t *testing.T) {
	var value any
	deep := []byte(strings.Repeat("[", 100) + "0" + strings.Repeat("]", 100))
	if strictJSON(deep, &value) == nil {
		t.Fatal("unbounded nesting")
	}
	p, plan, packet, root := gateFixture(t)
	if e := os.WriteFile(filepath.Join(root, "empty.txt"), nil, 0600); e != nil {
		t.Fatal(e)
	}
	packet.Inputs["review"] = &evidenceRef{Path: "empty.txt", SHA256: gateHash(nil)}
	raw, _ := json.Marshal(packet)
	if _, e := checkPacket(p, plan, raw, root); e == nil {
		t.Fatal("empty evidence accepted")
	}
}
