// acceptance-gates prepares and checks a bounded evidence packet for human review.
// It never executes provider operations or grants release/publication approval.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const gateLimit = 16 << 20

var gateError = errors.New("invalid acceptance evidence packet")

type gatePlan struct {
	Format    int      `json:"format_version"`
	Profiles  []string `json:"required_profiles"`
	Inputs    []string `json:"required_inputs"`
	Scenarios []string `json:"required_scenarios"`
}
type evidenceRef struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type scenarioEvidence struct {
	Result string       `json:"result"`
	Scope  string       `json:"scope"`
	Skips  int          `json:"skips"`
	Record *evidenceRef `json:"record"`
}
type profileEvidence struct {
	SourceCommit        string                      `json:"source_commit"`
	ArtifactSHA256      string                      `json:"artifact_sha256"`
	ManifestSHA256      string                      `json:"manifest_sha256"`
	SignerPolicySHA256  string                      `json:"signer_policy_sha256"`
	VerifiedPayload     *evidenceRef                `json:"verified_payload"`
	FreshHostInventory  *evidenceRef                `json:"fresh_host_inventory"`
	RuntimeNoticeReview *evidenceRef                `json:"runtime_notice_review"`
	Scenarios           map[string]scenarioEvidence `json:"scenarios"`
}
type gatePacket struct {
	Format     int                        `json:"format_version"`
	PlanSHA256 string                     `json:"plan_sha256"`
	Inputs     map[string]*evidenceRef    `json:"inputs"`
	Profiles   map[string]profileEvidence `json:"profiles"`
}
type gateResult struct {
	State           string   `json:"state"`
	EvidenceReady   bool     `json:"evidence_ready_for_review"`
	ReleaseAccepted bool     `json:"release_acceptance_complete"`
	Publication     bool     `json:"publication_authorized"`
	PacketSHA256    string   `json:"packet_sha256"`
	PlanSHA256      string   `json:"plan_sha256"`
	Missing         []string `json:"missing"`
	VerifiedFiles   int      `json:"verified_evidence_files"`
}

func gateHash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func validHex(s string, n int) bool {
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == n && s == strings.ToLower(s)
}
func strictJSON(b []byte, out any) error {
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	nodes := 0
	var walk func(int) error
	walk = func(depth int) error {
		nodes++
		if depth > 64 || nodes > 200000 {
			return gateError
		}
		t, e := d.Token()
		if e != nil {
			return gateError
		}
		delim, ok := t.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				key, e := d.Token()
				if e != nil {
					return gateError
				}
				s, ok := key.(string)
				if !ok || seen[s] {
					return gateError
				}
				seen[s] = true
				if e = walk(depth + 1); e != nil {
					return e
				}
			}
		case '[':
			for d.More() {
				if e = walk(depth + 1); e != nil {
					return e
				}
			}
		default:
			return gateError
		}
		_, e = d.Token()
		return e
	}
	if e := walk(0); e != nil {
		return gateError
	}
	if _, e := d.Token(); e != io.EOF {
		return gateError
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.DisallowUnknownFields()
	if decoder.Decode(out) != nil {
		return gateError
	}
	return nil
}
func bounded(path string) ([]byte, error) {
	info, e := os.Lstat(path)
	if e != nil || !info.Mode().IsRegular() || info.Size() > gateLimit {
		return nil, gateError
	}
	f, e := os.Open(path)
	if e != nil {
		return nil, gateError
	}
	defer f.Close()
	actual, e := f.Stat()
	if e != nil || !os.SameFile(info, actual) {
		return nil, gateError
	}
	b, e := io.ReadAll(io.LimitReader(f, gateLimit+1))
	if e != nil || len(b) > gateLimit {
		return nil, gateError
	}
	return b, nil
}
func loadGatePlan(path string) (gatePlan, []byte, error) {
	b, e := bounded(path)
	if e != nil {
		return gatePlan{}, nil, e
	}
	// The plan contains other descriptive fields; its entire bytes are bound below.
	var all map[string]json.RawMessage
	if strictJSON(b, &all) != nil {
		return gatePlan{}, nil, gateError
	}
	var p gatePlan
	for name, target := range map[string]any{"format_version": &p.Format, "required_profiles": &p.Profiles, "required_inputs": &p.Inputs, "required_scenarios": &p.Scenarios} {
		if json.Unmarshal(all[name], target) != nil {
			return p, nil, gateError
		}
	}
	if p.Format != 1 {
		return p, nil, gateError
	}
	for _, values := range [][]string{p.Profiles, p.Inputs, p.Scenarios} {
		seen := map[string]bool{}
		if len(values) == 0 || len(values) > 128 {
			return p, nil, gateError
		}
		for _, v := range values {
			if v == "" || seen[v] {
				return p, nil, gateError
			}
			seen[v] = true
		}
	}
	return p, b, nil
}
func newPacket(p gatePlan, raw []byte) gatePacket {
	v := gatePacket{Format: 1, PlanSHA256: gateHash(raw), Inputs: map[string]*evidenceRef{}, Profiles: map[string]profileEvidence{}}
	for _, key := range p.Inputs {
		v.Inputs[key] = nil
	}
	for _, key := range p.Profiles {
		q := profileEvidence{Scenarios: map[string]scenarioEvidence{}}
		for _, s := range p.Scenarios {
			q.Scenarios[s] = scenarioEvidence{Result: "pending", Scope: "unqualified"}
		}
		v.Profiles[key] = q
	}
	return v
}
func exactKeys[T any](m map[string]T, expected []string) bool {
	if len(m) != len(expected) {
		return false
	}
	for _, k := range expected {
		if _, ok := m[k]; !ok {
			return false
		}
	}
	return true
}
func checkPacket(p gatePlan, planRaw, raw []byte, root string) (gateResult, error) {
	result := gateResult{State: "incomplete", Missing: []string{}, PlanSHA256: gateHash(planRaw), PacketSHA256: gateHash(raw)}
	var packet gatePacket
	if strictJSON(raw, &packet) != nil || packet.Format != 1 || packet.PlanSHA256 != result.PlanSHA256 || !exactKeys(packet.Inputs, p.Inputs) || !exactKeys(packet.Profiles, p.Profiles) {
		return result, gateError
	}
	verify := func(label string, ref *evidenceRef) ([]byte, error) {
		if ref == nil {
			result.Missing = append(result.Missing, label)
			return nil, nil
		}
		if !validHex(ref.SHA256, 32) || ref.Path == "" || filepath.IsAbs(ref.Path) || filepath.ToSlash(filepath.Clean(ref.Path)) != ref.Path || strings.ContainsAny(ref.Path, "\\:") {
			return nil, gateError
		}
		parts := strings.Split(ref.Path, "/")
		current := root
		for _, part := range parts {
			if part == "." || part == ".." || part == "" {
				return nil, gateError
			}
			current = filepath.Join(current, part)
			info, e := os.Lstat(current)
			if e != nil || info.Mode()&os.ModeSymlink != 0 {
				return nil, gateError
			}
		}
		b, e := bounded(current)
		if e != nil || len(b) == 0 || gateHash(b) != ref.SHA256 {
			return nil, gateError
		}
		result.VerifiedFiles++
		return b, nil
	}
	for _, key := range p.Inputs {
		if _, e := verify("input:"+key, packet.Inputs[key]); e != nil {
			return result, e
		}
	}
	selectedSource := ""
	for _, key := range p.Profiles {
		q := packet.Profiles[key]
		if q.SourceCommit != "" {
			if selectedSource != "" && q.SourceCommit != selectedSource {
				return result, gateError
			}
			selectedSource = q.SourceCommit
		}
		if !exactKeys(q.Scenarios, p.Scenarios) {
			return result, gateError
		}
		for _, binding := range []struct {
			name, value string
			size        int
		}{{"source_commit", q.SourceCommit, 20}, {"artifact_sha256", q.ArtifactSHA256, 32}, {"manifest_sha256", q.ManifestSHA256, 32}, {"signer_policy_sha256", q.SignerPolicySHA256, 32}} {
			if binding.value == "" {
				result.Missing = append(result.Missing, key+":"+binding.name)
			} else if !validHex(binding.value, binding.size) {
				return result, gateError
			}
		}
		proof, e := verify(key+":verified_payload", q.VerifiedPayload)
		if e != nil {
			return result, e
		}
		if proof != nil {
			var fields map[string]json.RawMessage
			if strictJSON(proof, &fields) != nil {
				return result, gateError
			}
			for name, want := range map[string]string{"protocol": "symphony.release.candidate-verification.v1", "scope": "caller_supplied_payload", "profile_id": key, "source_commit": q.SourceCommit, "artifact_sha256": q.ArtifactSHA256, "inventory_sha256": q.ManifestSHA256, "allowed_signers_sha256": q.SignerPolicySHA256} {
				var got string
				if json.Unmarshal(fields[name], &got) != nil || want == "" || got != want {
					return result, gateError
				}
			}
		}
		for _, item := range []struct {
			name string
			ref  *evidenceRef
		}{{"fresh_host_inventory", q.FreshHostInventory}, {"runtime_notice_review", q.RuntimeNoticeReview}} {
			if _, e = verify(key+":"+item.name, item.ref); e != nil {
				return result, e
			}
		}
		for _, scenario := range p.Scenarios {
			s := q.Scenarios[scenario]
			label := key + ":" + scenario
			switch s.Result {
			case "pending", "failed", "passed":
			default:
				return result, gateError
			}
			switch s.Scope {
			case "unqualified", "fixture", "operational":
			default:
				return result, gateError
			}
			if s.Skips < 0 {
				return result, gateError
			}
			b, e := verify(label+":record", s.Record)
			if e != nil {
				return result, e
			}
			if s.Result != "passed" || s.Scope != "operational" || s.Skips != 0 {
				result.Missing = append(result.Missing, label+":operational_no_skip_pass")
			}
			if b != nil {
				var record struct {
					Profile  string         `json:"profile_id"`
					Scenario string         `json:"scenario"`
					Source   string         `json:"source_commit"`
					Artifact string         `json:"artifact_sha256"`
					Result   string         `json:"result"`
					Scope    string         `json:"scope"`
					Skips    int            `json:"skips"`
					Evidence []*evidenceRef `json:"evidence"`
				}
				if strictJSON(b, &record) != nil || record.Profile != key || record.Scenario != scenario || record.Source != q.SourceCommit || record.Artifact != q.ArtifactSHA256 || record.Result != s.Result || record.Scope != s.Scope || record.Skips != s.Skips || len(record.Evidence) == 0 || len(record.Evidence) > 128 {
					return result, gateError
				}
				for i, ref := range record.Evidence {
					if _, e = verify(fmt.Sprintf("%s:evidence:%d", label, i), ref); e != nil {
						return result, e
					}
				}
			}
		}
	}
	if len(result.Missing) == 0 {
		result.State = "evidence_ready_for_review"
		result.EvidenceReady = true
	}
	return result, nil
}
func writeNew(path string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return e
	}
	defer f.Close()
	_, e = f.Write(append(b, '\n'))
	return e
}
func main() {
	plan := flag.String("plan", "", "reviewed acceptance plan")
	packet := flag.String("packet", "", "new template or existing evidence packet")
	output := flag.String("report", "", "new report file for check")
	mode := flag.String("mode", "check", "init or check")
	flag.Parse()
	fail := func() { fmt.Fprintln(os.Stderr, gateError); os.Exit(1) }
	if flag.NArg() != 0 || !filepath.IsAbs(*plan) || !filepath.IsAbs(*packet) {
		fail()
	}
	p, raw, e := loadGatePlan(*plan)
	if e != nil {
		fail()
	}
	if *mode == "init" {
		if *output != "" || writeNew(*packet, newPacket(p, raw)) != nil {
			fail()
		}
		fmt.Println("Prepared unqualified acceptance packet")
		return
	}
	if *mode != "check" || !filepath.IsAbs(*output) {
		fail()
	}
	data, e := bounded(*packet)
	if e != nil {
		fail()
	}
	result, e := checkPacket(p, raw, data, filepath.Dir(*packet))
	if e != nil {
		fail()
	}
	if writeNew(*output, result) != nil {
		fail()
	}
	if !result.EvidenceReady {
		fmt.Println("Acceptance inputs/evidence incomplete")
		os.Exit(2)
	}
	fmt.Println("Evidence packet ready for independent review; acceptance and publication remain unapproved")
}
