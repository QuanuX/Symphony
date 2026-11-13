// delivery-acceptance composes the unchanged v1 gate with multi-artifact evidence.
// A ready packet is ready for independent review, never acceptance/publication.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
)

type daRequirements struct {
	Protocol     string                          `json:"protocol"`
	PlanSHA      string                          `json:"core_plan_sha256"`
	MappingSHA   string                          `json:"acceptance_mapping_sha256"`
	Profiles     map[string]daRequiredProfile    `json:"profiles"`
	Supplemental map[string]*map[string][]string `json:"supplemental"`
}
type daRequiredProfile struct {
	Artifacts *[]string            `json:"artifact_ids"`
	Scenarios map[string]*[]string `json:"scenario_artifacts"`
}
type daTarget struct {
	Root      string `json:"root"`
	Selection string `json:"selection"`
	Policy    string `json:"allowed_signers"`
	Signer    string `json:"signer"`
}
type daTargets struct {
	Protocol string               `json:"protocol"`
	Profiles map[string]*daTarget `json:"profiles"`
}
type daOwner struct {
	Report  evidenceRef   `json:"report"`
	Support []evidenceRef `json:"support"`
}
type daArtifactEvidence struct {
	Owner        *daOwner     `json:"owner"`
	Installation *evidenceRef `json:"installation"`
	Loader       *evidenceRef `json:"loader"`
	Trust        *evidenceRef `json:"trust"`
}
type daProfileEvidence struct {
	Index     *evidenceRef                  `json:"delivery_index"`
	Primary   *string                       `json:"primary_native_artifact_id"`
	Artifacts map[string]daArtifactEvidence `json:"artifacts"`
	Scenarios map[string]*evidenceRef       `json:"scenarios"`
}
type daPacket struct {
	Protocol        string                       `json:"protocol"`
	CoreSHA         string                       `json:"core_packet_sha256"`
	PlanSHA         string                       `json:"core_plan_sha256"`
	MappingSHA      string                       `json:"acceptance_mapping_sha256"`
	RequirementsSHA string                       `json:"artifact_requirements_sha256"`
	Profiles        map[string]daProfileEvidence `json:"profiles"`
	Criteria        map[string]*evidenceRef      `json:"paper_criteria"`
	Supplemental    map[string]*evidenceRef      `json:"supplemental"`
	Release         bool                         `json:"release_acceptance_complete"`
	Publication     bool                         `json:"publication_authorized"`
}

// Every observation carries actual file identities, not an artifact ID in a
// digest field. Identity maps are recomputed from the independently verified set.
type daIdentity struct {
	Primary   dsFile   `json:"primary"`
	Inventory dsFile   `json:"inventory"`
	Receipts  []dsFile `json:"receipts"`
	PolicySHA *string  `json:"artifact_signer_policy_sha256"`
}
type daBinding struct {
	Source    dsSource              `json:"source"`
	IndexSHA  string                `json:"delivery_index_sha256"`
	Artifacts map[string]daIdentity `json:"artifacts"`
}
type daObservation struct {
	Protocol    string               `json:"protocol"`
	ID          string               `json:"id"`
	Result      string               `json:"result"`
	Scope       string               `json:"scope"`
	Skips       int                  `json:"skips"`
	Bindings    map[string]daBinding `json:"bindings"`
	CoreCells   []string             `json:"core_cells"`
	Evidence    []evidenceRef        `json:"evidence"`
	Observer    string               `json:"observer"`
	ObservedAt  string               `json:"observed_at"`
	Expected    string               `json:"expected_outcome"`
	Actual      string               `json:"actual_outcome"`
	Release     bool                 `json:"release_acceptance_complete"`
	Publication bool                 `json:"publication_authorized"`
}
type daResult struct {
	Protocol         string            `json:"protocol"`
	State            string            `json:"state"`
	Ready            bool              `json:"evidence_ready_for_review"`
	PacketSHA        string            `json:"packet_sha256"`
	RequirementsSHA  string            `json:"artifact_requirements_sha256"`
	MappingSHA       string            `json:"acceptance_mapping_sha256"`
	Core             gateResult        `json:"core_v1_result"`
	Missing          []string          `json:"missing"`
	Indexes          map[string]string `json:"verified_delivery_indexes"`
	VerifiedEvidence int               `json:"verified_evidence_files"`
	Release          bool              `json:"release_acceptance_complete"`
	Publication      bool              `json:"publication_authorized"`
}
type daMapping struct {
	Profiles     []string
	Scenarios    []string
	Campaigns    map[string]string
	Criteria     map[string][]string
	Supplemental []string
	CF           map[string]bool
}
type daNativeProof struct {
	Protocol     string `json:"protocol"`
	Scope        string `json:"scope"`
	Publication  bool   `json:"publication_authorized"`
	VerifiedAt   string `json:"verified_at"`
	Commit       string `json:"source_commit"`
	Tree         string `json:"source_tree"`
	Profile      string `json:"profile_id"`
	ArtifactSHA  string `json:"artifact_sha256"`
	ManifestSHA  string `json:"inventory_sha256"`
	SignatureSHA string `json:"signature_sha256"`
	PolicySHA    string `json:"allowed_signers_sha256"`
	Scheme       string `json:"signature_scheme"`
	Namespace    string `json:"namespace"`
	Principal    string `json:"signer_principal"`
	Key          string `json:"observed_key_id"`
}
type daOptions struct{ mode, plan, mapping, requirements, core, packet, targets, report string }

func daSorted[T any](m map[string]T) []string {
	r := make([]string, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	sort.Strings(r)
	return r
}
func daSetEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := map[string]bool{}
	for _, s := range a {
		if seen[s] {
			return false
		}
		seen[s] = true
	}
	for _, s := range b {
		if !seen[s] {
			return false
		}
		delete(seen, s)
	}
	return len(seen) == 0
}
func daRead(p string) ([]byte, error) {
	if !filepath.IsAbs(p) || filepath.Clean(p) != p {
		return nil, dsError
	}
	root, e := dsRoot(filepath.Dir(p))
	if e != nil {
		return nil, e
	}
	f, e := dsObserve(root, filepath.Base(p))
	if e != nil || f.Size > dsMetadataLimit {
		return nil, dsError
	}
	b, e := dsRead(filepath.Join(root, filepath.Base(p)), dsMetadataLimit)
	if e != nil || dsHash(b) != f.SHA256 {
		return nil, dsError
	}
	return b, nil
}
func daLoadMapping(raw []byte, p gatePlan) (daMapping, error) {
	m := daMapping{Campaigns: map[string]string{}, Criteria: map[string][]string{}, CF: map[string]bool{}}
	var all map[string]json.RawMessage
	if len(raw) > dsMetadataLimit || strictJSON(raw, &all) != nil {
		return m, dsError
	}
	var campaigns []struct {
		ID       string `json:"id"`
		Scenario string `json:"scenario"`
	}
	var criteria []struct {
		ID        string   `json:"criterion_id"`
		Campaigns []string `json:"campaign_ids"`
	}
	var cf, additional []struct {
		ID string `json:"id"`
	}
	var cells []struct {
		ID       string `json:"id"`
		Profile  string `json:"profile"`
		Scenario string `json:"scenario"`
		Campaign string `json:"campaign_id"`
	}
	for k, v := range map[string]any{"required_profiles": &m.Profiles, "required_scenarios": &m.Scenarios, "campaigns": &campaigns, "paper_criteria": &criteria, "cloudflare_operation_cells": &cf, "additional_campaigns": &additional, "core_cells": &cells} {
		if json.Unmarshal(all[k], v) != nil {
			return m, dsError
		}
	}
	if len(m.Profiles) != 4 || len(m.Scenarios) != 9 || !daSetEqual(m.Profiles, p.Profiles) || !daSetEqual(m.Scenarios, p.Scenarios) || len(campaigns) != 9 || len(criteria) != 58 || len(cf) != 23 || len(additional) != 8 || len(cells) != 36 {
		return m, dsError
	}
	scenarios := []string{}
	for _, c := range campaigns {
		if !dsToken(c.ID) || m.Campaigns[c.ID] != "" {
			return m, dsError
		}
		m.Campaigns[c.ID] = c.Scenario
		scenarios = append(scenarios, c.Scenario)
	}
	if !daSetEqual(scenarios, m.Scenarios) {
		return m, dsError
	}
	seen := map[string]bool{}
	for _, c := range cells {
		if c.ID != c.Profile+"::"+c.Scenario || seen[c.ID] || m.Campaigns[c.Campaign] != c.Scenario || !daContains(m.Profiles, c.Profile) {
			return m, dsError
		}
		seen[c.ID] = true
	}
	for _, c := range criteria {
		if !dsToken(c.ID) || m.Criteria[c.ID] != nil || len(c.Campaigns) == 0 || !daUnique(c.Campaigns) {
			return m, dsError
		}
		for _, id := range c.Campaigns {
			if m.Campaigns[id] == "" {
				return m, dsError
			}
		}
		m.Criteria[c.ID] = c.Campaigns
	}
	for _, c := range cf {
		if !strings.HasPrefix(c.ID, "CF::") || !dsToken(strings.TrimPrefix(c.ID, "CF::")) {
			return m, dsError
		}
		m.CF[c.ID] = true
		m.Supplemental = append(m.Supplemental, c.ID)
	}
	for _, c := range additional {
		if !dsToken(c.ID) {
			return m, dsError
		}
		m.Supplemental = append(m.Supplemental, c.ID)
	}
	if !daUnique(m.Supplemental) {
		return m, dsError
	}
	return m, nil
}
func daContains(a []string, s string) bool {
	for _, v := range a {
		if v == s {
			return true
		}
	}
	return false
}
func daUnique(a []string) bool {
	s := map[string]bool{}
	for _, v := range a {
		if s[v] {
			return false
		}
		s[v] = true
	}
	return true
}
func daNewRequirements(p gatePlan, plan, mapping []byte, m daMapping) daRequirements {
	r := daRequirements{Protocol: "symphony.release.artifact-requirements.v1", PlanSHA: dsHash(plan), MappingSHA: dsHash(mapping), Profiles: map[string]daRequiredProfile{}, Supplemental: map[string]*map[string][]string{}}
	for _, id := range p.Profiles {
		s := map[string]*[]string{}
		for _, v := range p.Scenarios {
			s[v] = nil
		}
		r.Profiles[id] = daRequiredProfile{Scenarios: s}
	}
	for _, id := range m.Supplemental {
		r.Supplemental[id] = nil
	}
	return r
}
func daRequirementsOK(r daRequirements, p gatePlan, plan, mapping []byte, m daMapping) error {
	if r.Protocol != "symphony.release.artifact-requirements.v1" || r.PlanSHA != dsHash(plan) || r.MappingSHA != dsHash(mapping) || len(r.Profiles) < 4 || len(r.Profiles) > 16 || !exactKeys(r.Supplemental, m.Supplemental) {
		return dsError
	}
	for _, id := range p.Profiles {
		if _, ok := r.Profiles[id]; !ok {
			return dsError
		}
	}
	validIDs := func(ids []string) bool {
		if len(ids) == 0 || len(ids) > 4096 || !daUnique(ids) {
			return false
		}
		for _, id := range ids {
			if !dsToken(id) {
				return false
			}
		}
		return true
	}
	for id, v := range r.Profiles {
		expected := []string{}
		if daContains(p.Profiles, id) {
			expected = p.Scenarios
		}
		if !dsToken(id) || !exactKeys(v.Scenarios, expected) {
			return dsError
		}
		if v.Artifacts != nil && !validIDs(*v.Artifacts) {
			return dsError
		}
		for _, ids := range v.Scenarios {
			if ids == nil {
				continue
			}
			if v.Artifacts == nil || !validIDs(*ids) {
				return dsError
			}
			for _, a := range *ids {
				if !daContains(*v.Artifacts, a) {
					return dsError
				}
			}
		}
	}
	for _, v := range r.Supplemental {
		if v == nil {
			continue
		}
		if len(*v) == 0 {
			return dsError
		}
		for profile, ids := range *v {
			q, ok := r.Profiles[profile]
			if !ok || q.Artifacts == nil || !validIDs(ids) {
				return dsError
			}
			for _, id := range ids {
				if !daContains(*q.Artifacts, id) {
					return dsError
				}
			}
		}
	}
	return nil
}
func daNewPacket(p gatePlan, plan, mapping, requirements, core []byte, r daRequirements, m daMapping) daPacket {
	v := daPacket{Protocol: "symphony.release.delivery-acceptance.v1", CoreSHA: dsHash(core), PlanSHA: dsHash(plan), MappingSHA: dsHash(mapping), RequirementsSHA: dsHash(requirements), Profiles: map[string]daProfileEvidence{}, Criteria: map[string]*evidenceRef{}, Supplemental: map[string]*evidenceRef{}}
	for id, q := range r.Profiles {
		a := map[string]daArtifactEvidence{}
		if q.Artifacts != nil {
			for _, k := range *q.Artifacts {
				a[k] = daArtifactEvidence{}
			}
		}
		s := map[string]*evidenceRef{}
		for k := range q.Scenarios {
			s[k] = nil
		}
		v.Profiles[id] = daProfileEvidence{Artifacts: a, Scenarios: s}
	}
	for id := range m.Criteria {
		v.Criteria[id] = nil
	}
	for _, id := range m.Supplemental {
		v.Supplemental[id] = nil
	}
	return v
}
func daIdentityFor(index dsIndex, a dsArtifact) daIdentity {
	files := map[string]dsFile{}
	for _, f := range index.Files {
		files[f.Path] = f
	}
	v := daIdentity{Primary: files[a.Primary], Inventory: files[a.Inventory], PolicySHA: a.SignerPolicySHA, Receipts: []dsFile{}}
	for _, p := range a.Receipts {
		v.Receipts = append(v.Receipts, files[p])
	}
	return v
}

type daChecker struct {
	root         string
	result       daResult
	indexes      map[string]dsIndex
	targets      daTargets
	requirements daRequirements
	seen         map[string]string
	bytes        int64
}

func (c *daChecker) missing(label string) { c.result.Missing = append(c.result.Missing, label) }
func (c *daChecker) read(label string, ref *evidenceRef) ([]byte, error) {
	if ref == nil {
		c.missing(label)
		return nil, nil
	}
	if !dsHex(ref.SHA256, 32) {
		return nil, dsError
	}
	p, e := dsContained(c.root, ref.Path)
	if e != nil {
		return nil, e
	}
	b, e := daRead(p)
	if e != nil || len(b) == 0 || dsHash(b) != ref.SHA256 {
		return nil, dsError
	}
	if c.seen == nil {
		c.seen = map[string]string{}
	}
	if old, ok := c.seen[ref.Path]; ok && old != ref.SHA256 {
		return nil, dsError
	}
	if _, ok := c.seen[ref.Path]; !ok {
		c.bytes += int64(len(b))
		if len(c.seen) >= 10000 || c.bytes > 512<<20 {
			return nil, dsError
		}
		c.seen[ref.Path] = ref.SHA256
	}
	c.result.VerifiedEvidence++
	return b, nil
}
func daArtifact(index dsIndex, id string) (dsArtifact, bool) {
	for _, a := range index.Artifacts {
		if a.Selection.ID == id {
			return a, true
		}
	}
	return dsArtifact{}, false
}
func (c *daChecker) bindings(required map[string][]string) (map[string]daBinding, bool, error) {
	result := map[string]daBinding{}
	ready := true
	for profile, ids := range required {
		index, ok := c.indexes[profile]
		if !ok {
			ready = false
			continue
		}
		b := daBinding{Source: index.Source, IndexSHA: c.result.Indexes[profile], Artifacts: map[string]daIdentity{}}
		for _, id := range ids {
			a, ok := daArtifact(index, id)
			if !ok {
				return nil, false, dsError
			}
			for _, dep := range a.Selection.Dependencies {
				if !daContains(ids, dep) {
					return nil, false, dsError
				}
			}
			b.Artifacts[id] = daIdentityFor(index, a)
		}
		result[profile] = b
	}
	return result, ready, nil
}
func (c *daChecker) observation(label string, ref *evidenceRef, required map[string][]string, cells []string) error {
	raw, e := c.read(label, ref)
	if e != nil || raw == nil {
		return e
	}
	var o daObservation
	if dsDecode(raw, &o) != nil || o.Protocol != "symphony.release.delivery-observation.v1" || o.ID != label || o.Release || o.Publication || o.Skips < 0 || !daSetEqual(o.CoreCells, cells) || len(o.Evidence) == 0 || len(o.Evidence) > 128 || strings.TrimSpace(o.Observer) == "" || strings.TrimSpace(o.Expected) == "" || strings.TrimSpace(o.Actual) == "" {
		return dsError
	}
	if _, e := time.Parse(time.RFC3339, o.ObservedAt); e != nil {
		return dsError
	}
	if !daContains([]string{"pending", "failed", "passed"}, o.Result) || !daContains([]string{"unqualified", "fixture", "operational"}, o.Scope) {
		return dsError
	}
	bindings, ready, e := c.bindings(required)
	if e != nil {
		return e
	}
	if ready {
		if !reflect.DeepEqual(o.Bindings, bindings) {
			return dsError
		}
	} else {
		c.missing(label + ":verified_deliveries")
	}
	for i, ref := range o.Evidence {
		if _, e = c.read(fmt.Sprintf("%s:evidence:%d", label, i), &ref); e != nil {
			return e
		}
	}
	if o.Result != "passed" || o.Scope != "operational" || o.Skips != 0 {
		c.missing(label + ":operational_no_skip_pass")
	}
	return nil
}

// Owner reports remain scoped byte/receipt inspections. They cannot replace any
// final-host installation, loader, trust, scenario or independent review record.
func (c *daChecker) owner(profile, id string, record *daOwner) error {
	label := profile + ":" + id + ":owner"
	if record == nil {
		c.missing(label)
		return nil
	}
	raw, e := c.read(label, &record.Report)
	if e != nil {
		return e
	}
	support := map[string][]byte{}
	for i, ref := range record.Support {
		if i >= 16 {
			return dsError
		}
		b, e := c.read(label+":support", &ref)
		if e != nil {
			return e
		}
		if support[ref.SHA256] != nil {
			return dsError
		}
		support[ref.SHA256] = b
	}
	index, ok := c.indexes[profile]
	if !ok {
		c.missing(label + ":verified_delivery")
		return nil
	}
	a, ok := daArtifact(index, id)
	if !ok {
		return dsError
	}
	files := map[string]dsFile{}
	for _, f := range index.Files {
		files[f.Path] = f
	}
	switch {
	case dsOwnerComponent(a.Selection.Adapter) != "":
		var r dsOwnerResult
		var b dsOwnerBuild
		var i dsOwnerInspection
		if dsDecode(raw, &r) != nil || r.Protocol != "symphony.release.delivery-owner-verification.v1" || r.Scope != "selected_owner_receipt_only" || r.IndexSHA != c.result.Indexes[profile] || r.ArtifactID != id || r.Source != index.Source || r.Profile != index.Profile || !r.OwnerVerified || r.FreshHost || r.Release || r.Publication || dsOwnerBinding(index, a, r.Inspection) != nil || dsDecode(support[r.ToolBuildSHA], &b) != nil || dsDecode(support[r.InspectionSHA], &i) != nil || !reflect.DeepEqual(i, r.Inspection) {
			return dsError
		}
		if b.Protocol != "symphony.release.owner-tool-build.v1" || b.Adapter != a.Selection.Adapter || b.Component != dsOwnerComponent(a.Selection.Adapter) || b.Version != a.Selection.Versions[b.Component] || b.Source != index.Source || b.OS != index.Profile.OS || b.Architecture != index.Profile.Architecture || b.GoVersion != "go1.26.5" || b.ToolSHA != r.ToolSHA || !dsHex(b.ToolSHA, 32) || !dsHex(b.SourceArchiveSHA, 32) {
			return dsError
		}
	case a.Selection.Adapter == "native-candidate-v1":
		var r dsNativeResult
		var h dsNativeHost
		var t dsNativeTool
		if dsDecode(raw, &r) != nil || r.Protocol != "symphony.release.delivery-native-verification.v1" || r.Scope != "selected_native_extraction_and_receipts" || r.IndexSHA != c.result.Indexes[profile] || r.ArtifactID != id || r.Source != index.Source || r.Profile != index.Profile || !r.OwnerVerified || r.FreshHost || r.Release || r.Publication || a.Manifest == nil || a.SignerPolicySHA == nil || r.ArtifactSHA != files[a.Primary].SHA256 || r.ManifestSHA != files[*a.Manifest].SHA256 || r.PolicySHA != *a.SignerPolicySHA || dsDecode(support[r.HostReportSHA], &h) != nil || dsNativeBinding(index, a, h) != nil || r.PayloadFileCount != h.FileCount || dsDecode(support[r.ToolRecordSHA], &t) != nil || support[r.PayloadReportSHA] == nil {
			return dsError
		}
		if t.Protocol != "symphony.release.native-verifier-tool.v1" || t.GoVersion != "go1.26.5" || t.ToolSHA != r.ToolSHA || !dsHex(t.ToolSHA, 32) || !reflect.DeepEqual(t.Sources, r.ToolSources) || !exactKeys(t.Sources, dsNativeSources) {
			return dsError
		}
		for _, sha := range t.Sources {
			if !dsHex(sha, 32) {
				return dsError
			}
		}
		var proof daNativeProof
		if dsDecode(support[r.PayloadReportSHA], &proof) != nil || proof.Protocol != "symphony.release.candidate-verification.v1" || proof.Scope != "caller_supplied_payload" || proof.Publication || proof.Commit != index.Source.Commit || proof.Tree != index.Source.Tree || proof.Profile != profile || proof.ArtifactSHA != r.ArtifactSHA || proof.ManifestSHA != r.ManifestSHA || proof.PolicySHA != r.PolicySHA || a.Signature == nil || proof.SignatureSHA != files[*a.Signature].SHA256 || proof.Scheme != "sshsig" || proof.Namespace != "symphony-release" || !dsSigner(proof.Principal) || !strings.HasPrefix(proof.Key, "SHA256:") {
			return dsError
		}
		if _, e := time.Parse(time.RFC3339, proof.VerifiedAt); e != nil {
			return dsError
		}
	case dsExecutableComponent(a) != "":
		var r dsExecutableResult
		var b dsExecutableBuild
		var inv dsExecutableInventory
		target := c.targets.Profiles[profile]
		if target == nil || len(a.BuildEvidence) != 1 {
			return dsError
		}
		buildRaw, e := dsRead(filepath.Join(target.Root, filepath.FromSlash(a.BuildEvidence[0])), dsMetadataLimit)
		if e != nil {
			return e
		}
		invRaw, e := dsRead(filepath.Join(target.Root, filepath.FromSlash(a.Inventory)), dsMetadataLimit)
		if e != nil {
			return e
		}
		if dsDecode(raw, &r) != nil || dsDecode(buildRaw, &b) != nil || dsDecode(invRaw, &inv) != nil || dsExecutableBinding(index, a, b, inv) != nil || r.Protocol != "symphony.release.delivery-executable-verification.v1" || r.Scope != "selected_executable_identity_only" || r.IndexSHA != c.result.Indexes[profile] || r.ArtifactID != id || r.Source != index.Source || r.Profile != index.Profile || !r.OwnerVerified || r.FreshHost || r.Release || r.Publication || r.BuildIdentitySHA != dsHash(buildRaw) || r.InventorySHA != dsHash(invRaw) || r.ExecutableSHA != b.ExecutableSHA || r.Component != b.Component || r.Version != b.Version || r.VersionOutputSHA != dsHash([]byte(r.VersionOutput)) {
			return dsError
		}
		if r.Component == "qxctl" {
			if dsQxctlBinding(b, []byte(r.VersionOutput)) != nil {
				return dsError
			}
		} else if r.VersionOutput != r.Component+" version "+r.Version+"\n" {
			return dsError
		}
	default:
		c.missing(label + ":unsupported_owner_adapter")
	}
	return nil
}
func daCheck(p gatePlan, plan, mapping, requirements, core, packet []byte, m daMapping, r daRequirements, targets daTargets, root, coreRoot string) (daResult, error) {
	result := daResult{Protocol: "symphony.release.delivery-acceptance-result.v1", State: "incomplete", PacketSHA: dsHash(packet), RequirementsSHA: dsHash(requirements), MappingSHA: dsHash(mapping), Missing: []string{}, Indexes: map[string]string{}}
	var v daPacket
	var v1 gatePacket
	if dsDecode(packet, &v) != nil || dsDecode(core, &v1) != nil || v.Protocol != "symphony.release.delivery-acceptance.v1" || v.CoreSHA != dsHash(core) || v.PlanSHA != dsHash(plan) || v.MappingSHA != dsHash(mapping) || v.RequirementsSHA != dsHash(requirements) || v.Release || v.Publication || !exactKeys(v.Profiles, daSorted(r.Profiles)) || !exactKeys(v.Criteria, daSorted(m.Criteria)) || !exactKeys(v.Supplemental, m.Supplemental) || targets.Protocol != "symphony.release.delivery-targets.v1" || !exactKeys(targets.Profiles, daSorted(r.Profiles)) {
		return result, dsError
	}
	var e error
	result.Core, e = checkPacket(p, plan, core, coreRoot)
	if e != nil {
		return result, e
	}
	c := daChecker{root: root, result: result, indexes: map[string]dsIndex{}, targets: targets, requirements: r}
	if !result.Core.EvidenceReady {
		c.missing("core_v1:incomplete")
	}
	source := dsSource{}
	for _, profile := range daSorted(r.Profiles) {
		q := r.Profiles[profile]
		pv := v.Profiles[profile]
		ids := []string{}
		if q.Artifacts != nil {
			ids = *q.Artifacts
		} else {
			c.missing(profile + ":artifact_requirements")
		}
		if !exactKeys(pv.Artifacts, ids) || !exactKeys(pv.Scenarios, daSorted(q.Scenarios)) {
			return result, dsError
		}
		raw, e := c.read(profile+":delivery_index", pv.Index)
		if e != nil {
			return result, e
		}
		target := targets.Profiles[profile]
		if target == nil {
			c.missing(profile + ":independent_delivery_inputs")
			continue
		}
		verified, e := dsRun(dsOptions{mode: "verify", root: target.Root, selection: target.Selection, policy: target.Policy, profile: profile, signer: target.Signer})
		if e != nil {
			return result, e
		}
		indexRaw, e := dsRead(filepath.Join(target.Root, "delivery-set.json"), dsMetadataLimit)
		if e != nil || dsHash(indexRaw) != verified.IndexSHA {
			return result, dsError
		}
		var index dsIndex
		if dsDecode(indexRaw, &index) != nil {
			return result, dsError
		}
		if source != (dsSource{}) && index.Source != source {
			return result, dsError
		}
		source = index.Source
		actual := []string{}
		for _, a := range index.Artifacts {
			actual = append(actual, a.Selection.ID)
		}
		if q.Artifacts != nil && !daSetEqual(ids, actual) {
			return result, dsError
		}
		if raw != nil && dsHash(raw) != verified.IndexSHA {
			return result, dsError
		}
		c.indexes[profile] = index
		c.result.Indexes[profile] = verified.IndexSHA
		if daContains(p.Profiles, profile) {
			if pv.Primary == nil {
				c.missing(profile + ":primary_native_artifact")
			} else {
				a, ok := daArtifact(index, *pv.Primary)
				if !ok || a.Selection.Adapter != "native-candidate-v1" || a.Manifest == nil || a.SignerPolicySHA == nil {
					return result, dsError
				}
				files := map[string]dsFile{}
				for _, f := range index.Files {
					files[f.Path] = f
				}
				coreProfile := v1.Profiles[profile]
				for _, pair := range [][2]string{{coreProfile.SourceCommit, index.Source.Commit}, {coreProfile.ArtifactSHA256, files[a.Primary].SHA256}, {coreProfile.ManifestSHA256, files[*a.Manifest].SHA256}, {coreProfile.SignerPolicySHA256, *a.SignerPolicySHA}} {
					if pair[0] != "" && pair[0] != pair[1] {
						return result, dsError
					}
				}
			}
		} else if pv.Primary != nil {
			return result, dsError
		}
	}
	for _, profile := range daSorted(r.Profiles) {
		q := r.Profiles[profile]
		pv := v.Profiles[profile]
		for _, id := range daSorted(pv.Artifacts) {
			a := pv.Artifacts[id]
			if e = c.owner(profile, id, a.Owner); e != nil {
				return result, e
			}
			// Single-artifact lifecycle observations include that artifact's complete
			// dependency closure, derived from the signed set (no inferred dependency).
			ids := []string{id}
			if index, ok := c.indexes[profile]; ok {
				seen := map[string]bool{id: true}
				for n := 0; n < len(ids); n++ {
					artifact, found := daArtifact(index, ids[n])
					if !found {
						return result, dsError
					}
					for _, dep := range artifact.Selection.Dependencies {
						if !seen[dep] {
							seen[dep] = true
							ids = append(ids, dep)
						}
					}
				}
			}
			for _, item := range []struct {
				name string
				ref  *evidenceRef
			}{{"installation", a.Installation}, {"loader", a.Loader}, {"trust", a.Trust}} {
				if e = c.observation(profile+":"+id+":"+item.name, item.ref, map[string][]string{profile: ids}, []string{}); e != nil {
					return result, e
				}
			}
		}
		for _, scenario := range daSorted(q.Scenarios) {
			ids := q.Scenarios[scenario]
			label := profile + "::" + scenario
			if ids == nil {
				c.missing(label + ":artifact_requirements")
				if pv.Scenarios[scenario] != nil {
					return result, dsError
				}
				continue
			}
			if e = c.observation(label, pv.Scenarios[scenario], map[string][]string{profile: *ids}, []string{label}); e != nil {
				return result, e
			}
		}
	}
	for _, id := range daSorted(m.Criteria) {
		required := map[string][]string{}
		cells := []string{}
		pending := false
		for _, profile := range p.Profiles {
			seen := map[string]bool{}
			for _, campaign := range m.Criteria[id] {
				scenario := m.Campaigns[campaign]
				cells = append(cells, profile+"::"+scenario)
				ids := r.Profiles[profile].Scenarios[scenario]
				if ids == nil {
					pending = true
					continue
				}
				for _, a := range *ids {
					seen[a] = true
				}
			}
			required[profile] = daSorted(seen)
		}
		if pending {
			c.missing(id + ":artifact_requirements")
			if v.Criteria[id] != nil {
				return result, dsError
			}
			continue
		}
		if e = c.observation(id, v.Criteria[id], required, cells); e != nil {
			return result, e
		}
	}
	for _, id := range m.Supplemental {
		required := r.Supplemental[id]
		if required == nil {
			c.missing(id + ":artifact_requirements")
			if v.Supplemental[id] != nil {
				return result, dsError
			}
			continue
		}
		if m.CF[id] {
			for profile, ids := range *required {
				index, ok := c.indexes[profile]
				if !ok {
					continue
				}
				if index.Profile.OS != "macos" || index.Profile.Architecture != "amd64" {
					return result, dsError
				}
				found := false
				for _, aid := range ids {
					a, _ := daArtifact(index, aid)
					if a.Selection.Versions["sov-cloudflare-adapter"] != "" {
						found = true
					}
				}
				if !found {
					return result, dsError
				}
			}
		}
		if e = c.observation(id, v.Supplemental[id], *required, []string{}); e != nil {
			return result, e
		}
	}
	// Reverify every retained delivery after the evidence walk. These checks run
	// in private quiescent directories; they are not a hostile same-UID sandbox.
	for profile, target := range targets.Profiles {
		if target == nil {
			continue
		}
		again, e := dsRun(dsOptions{mode: "verify", root: target.Root, selection: target.Selection, policy: target.Policy, profile: profile, signer: target.Signer})
		if e != nil || again.IndexSHA != c.result.Indexes[profile] {
			return result, dsError
		}
	}
	for path, sha := range c.seen {
		p, e := dsContained(c.root, path)
		if e != nil {
			return result, e
		}
		raw, e := daRead(p)
		if e != nil || dsHash(raw) != sha {
			return result, dsError
		}
	}
	if len(c.result.Missing) == 0 {
		c.result.State = "evidence_ready_for_review"
		c.result.Ready = true
	}
	sort.Strings(c.result.Missing)
	return c.result, nil
}
func daRun(o daOptions) (daResult, error) {
	var result daResult
	plan, e := daRead(o.plan)
	if e != nil {
		return result, e
	}
	p, _, e := loadGatePlan(o.plan)
	if e != nil {
		return result, e
	}
	mapping, e := daRead(o.mapping)
	if e != nil {
		return result, e
	}
	m, e := daLoadMapping(mapping, p)
	if e != nil {
		return result, e
	}
	if o.mode == "init-requirements" {
		if o.core != "" || o.packet != "" || o.targets != "" || o.report != "" {
			return result, dsError
		}
		return result, dsWriteNew(o.requirements, dsJSON(daNewRequirements(p, plan, mapping, m)))
	}
	requirements, e := daRead(o.requirements)
	if e != nil {
		return result, e
	}
	var r daRequirements
	if dsDecode(requirements, &r) != nil || daRequirementsOK(r, p, plan, mapping, m) != nil {
		return result, dsError
	}
	core, e := daRead(o.core)
	if e != nil {
		return result, e
	}
	if o.mode == "init" {
		if o.targets != "" || o.report != "" {
			return result, dsError
		}
		if _, e := checkPacket(p, plan, core, filepath.Dir(o.core)); e != nil {
			return result, e
		}
		return result, dsWriteNew(o.packet, dsJSON(daNewPacket(p, plan, mapping, requirements, core, r, m)))
	}
	if o.mode != "check" {
		return result, dsError
	}
	raw, e := daRead(o.packet)
	if e != nil {
		return result, e
	}
	root, e := dsRoot(filepath.Dir(o.packet))
	if e != nil {
		return result, e
	}
	targetRaw, e := daRead(o.targets)
	if e != nil {
		return result, e
	}
	var targets daTargets
	if dsDecode(targetRaw, &targets) != nil {
		return result, dsError
	}
	for _, target := range targets.Profiles {
		if target == nil {
			continue
		}
		deliveryRoot, e := dsRoot(target.Root)
		if e != nil {
			return result, e
		}
		for _, path := range []string{o.plan, o.mapping, o.requirements, o.core, o.packet, o.targets, o.report} {
			if !dsOutside(deliveryRoot, path) {
				return result, dsError
			}
		}
	}
	result, e = daCheck(p, plan, mapping, requirements, core, raw, m, r, targets, root, filepath.Dir(o.core))
	if e != nil {
		return result, e
	}
	for path, b := range map[string][]byte{o.plan: plan, o.mapping: mapping, o.requirements: requirements, o.core: core, o.packet: raw, o.targets: targetRaw} {
		after, e := daRead(path)
		if e != nil || dsHash(after) != dsHash(b) {
			return result, dsError
		}
	}
	if e = dsWriteNew(o.report, dsJSON(result)); e != nil {
		return result, e
	}
	return result, nil
}
func main() {
	var o daOptions
	flag.StringVar(&o.mode, "mode", "check", "init-requirements, init, or check")
	flag.StringVar(&o.plan, "plan", "", "independent original v1 plan")
	flag.StringVar(&o.mapping, "mapping", "", "independent complete source-bound acceptance mapping")
	flag.StringVar(&o.requirements, "requirements", "", "independent artifact-to-scenario requirements, outside payload")
	flag.StringVar(&o.core, "core-packet", "", "existing v1 packet; remains authoritative")
	flag.StringVar(&o.packet, "packet", "", "new template or existing companion packet")
	flag.StringVar(&o.targets, "targets", "", "independent retained delivery roots, selections and trust policies")
	flag.StringVar(&o.report, "report", "", "new report path for check")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, dsError)
		os.Exit(1)
	}
	r, e := daRun(o)
	if e != nil {
		fmt.Fprintln(os.Stderr, dsError)
		os.Exit(1)
	}
	if o.mode != "check" {
		fmt.Println("Prepared unqualified delivery acceptance input")
		return
	}
	fmt.Print(string(dsJSON(r)))
	if !r.Ready {
		os.Exit(2)
	}
}
