//go:build darwin || linux

package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/shvstate"
)

// This is a bounded retained-document workflow, not a vendor crawler. Its Unix
// authorization uses private real SSIAG/STAV processes and records their policy
// audit. It does not claim a source-write receipt or hardware qualification.
func TestInstalledSHVBoundedCatalogueWorkflow(t *testing.T) {
	prefix, storePrefix, requestPath := os.Getenv("SHV_RELEASE_PREFIX"), os.Getenv("SHV_RELEASE_STORE_PREFIX"), os.Getenv("SHV_RELEASE_PDF_REQUEST")
	if prefix == "" || storePrefix == "" || requestPath == "" || os.Getenv("SHV_RELEASE_SSIAG") == "" || os.Getenv("SHV_RELEASE_STAV") == "" {
		t.Skip("requires exact installed SHV owners/store and pinned PDFium/AMD request via SHV_RELEASE_PREFIX, SHV_RELEASE_STORE_PREFIX, SHV_RELEASE_PDF_REQUEST, SHV_RELEASE_SSIAG, SHV_RELEASE_STAV")
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	evidence := os.Getenv("SHV_RELEASE_EVIDENCE")
	if evidence == "" {
		evidence = root
	}
	if err = os.MkdirAll(evidence, 0700); err != nil {
		t.Fatal(err)
	}
	write := func(name string, v any) string {
		t.Helper()
		b, e := knowledgeengine.SCVCanonical(v)
		if e != nil {
			t.Fatal(e)
		}
		p := filepath.Join(evidence, name+".json")
		if e = os.WriteFile(p, b, 0600); e != nil {
			t.Fatal(e)
		}
		return p
	}
	object := func(v any) map[string]any {
		t.Helper()
		m, ok := v.(map[string]any)
		if !ok {
			t.Fatalf("expected object, got %T", v)
		}
		return m
	}
	read := func(path string) map[string]any {
		t.Helper()
		b, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		var v map[string]any
		d := json.NewDecoder(strings.NewReader(string(b)))
		d.UseNumber()
		if e = d.Decode(&v); e != nil {
			t.Fatal(e)
		}
		return v
	}
	seal := func(v map[string]any) map[string]any {
		t.Helper()
		delete(v, "digest")
		d, e := knowledgeengine.SCVDigest(v)
		if e != nil {
			t.Fatal(e)
		}
		v["digest"] = d
		return v
	}
	calls, refusals := 0, 0
	run := func(route string, body any, good bool, flags ...string) map[string]any {
		t.Helper()
		calls++
		args := strings.Fields("shv " + route)
		args = append(args, flags...)
		args = append(args, "--json")
		if body != nil {
			args = append(args, "--input", write(fmt.Sprintf("%03d-request", calls), body))
		}
		out, status := invokeCLI(t, args...)
		if err := os.WriteFile(filepath.Join(evidence, fmt.Sprintf("%03d-output.txt", calls)), []byte(out), 0600); err != nil {
			t.Fatal(err)
		}
		if (status == 0) != good {
			t.Fatalf("%s: status=%d output=%s", route, status, out)
		}
		var v map[string]any
		d := json.NewDecoder(strings.NewReader(out))
		d.UseNumber()
		if e := d.Decode(&v); e != nil {
			t.Fatalf("%s: %v %s", route, e, out)
		}
		write(fmt.Sprintf("%03d-response", calls), v)
		if !good {
			refusals++
			return v
		}
		if result, ok := v["result"].(map[string]any); ok {
			return result
		}
		return v
	}
	owner := func(version string) []string { return []string{"--prefix", prefix, "--version", version} }
	connector, err := knowledgeengine.InspectSHVStore(storePrefix, "0.4.0-dev")
	if err != nil {
		t.Fatalf("exact installed connector admission: %v", err)
	}
	write("connector-installation", connector)
	request := read(requestPath)
	source := object(request["source"])
	original, err := os.ReadFile(filepath.Join(request["source_root"].(string), source["path"].(string)))
	if err != nil {
		t.Fatal(err)
	}
	sources := filepath.Join(root, "sources")
	if err = os.Mkdir(sources, 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(sources, source["path"].(string)), original, 0600); err != nil {
		t.Fatal(err)
	}
	request["source_root"] = sources
	extraction := run("pdf extract", request, true, owner("0.3.0-dev")...)
	if len(extraction["rows"].([]any)) != 28 || extraction["documentary_lineages"] != json.Number("1") || extraction["namespace_equivalence"] != "not_asserted" {
		t.Fatal("PDF evidence scope changed")
	}
	graph := run("pdf graph project", request, true, owner("0.3.0-dev")...)
	run("pdf graph validate", map[string]any{"request": request, "graph": graph}, true, owner("0.3.0-dev")...)
	wrong := read(write("decoder-negative", request))
	object(wrong["decoder"])["digest"] = "sha256:" + strings.Repeat("0", 64)
	run("pdf extract", wrong, false, owner("0.3.0-dev")...)
	state := filepath.Join(root, "source-state")
	sourceID := "amd-69290-r1"
	subjectID := "pdf-7313"
	activation := append(owner("0.2.0-dev"), "--tops-id", ssiagTestTOPSID, "--source-id", sourceID, "--state-root", state)
	uri := "https://www.amd.com/content/dam/amd/en/documents/legal/epyc-app-export-control.pdf"
	definition := map[string]any{"source_id": sourceID, "publisher": "AMD", "authority_role": "publication 69290 revision 1.00, Table 8; documentary OPN only", "subject_ids": []string{subjectID}, "locators": []any{map[string]any{"id": "amd-pdf", "uri": uri, "format": "opaque"}}}
	plan := run("source activation propose", map[string]any{"operation_id": "onboard", "desired": definition, "reason": "explicit bounded qualification source"}, true, activation...)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "absent-config"))
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(root, "absent-runtime"))
	t.Setenv("SYMPHONY_SSIAG_SOCKET", "")
	run("source activation apply", plan, false, activation...)
	store, err := shvstate.New(state, ssiagTestTOPSID, sourceID)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.WithLock(func(tx *shvstate.Transaction) error {
		a, ok := tx.Attempt("onboard")
		if string(tx.Current()) != "null" || !ok || a.Status != "prepared" {
			t.Fatal("denied activation changed source or lost intent")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	audit := serveSHVQualificationAuthorities(t, evidence, shvActivationResource(shvActivationOptions{topsID: ssiagTestTOPSID, sourceID: sourceID}))
	recovered := run("source activation recover", nil, true, append(activation, "--operation-id", "onboard")...)
	if object(recovered["attempt"])["status"] != "committed" {
		t.Fatal("authorized retained source was not committed")
	}
	run("source activation apply", plan, true, activation...)
	authorization := object(object(recovered["attempt"])["authorization"])
	write("protected-activation-audit", audit(authorization["correlation_id"].(string), authorization["request_id"].(string)))
	status := run("source activation status", nil, true, activation...)
	revision := object(status["source"])["digest"]
	qualifier := "issuer=AMD;namespace=opn;profile=1"
	fields := []any{map[string]any{"predicate": "oem_identifier", "label": "OPN", "next_label": "Model", "value_type": "string", "qualifier": qualifier}}
	mapping := []any{map[string]any{"id": subjectID, "manufacturer": "AMD", "model": "7313", "hardware_class": "cpu", "source_id": source["id"], "heading_section": "table8", "interpretation_profile": "pdf_opn.v1", "document": map[string]any{"decoder_root": request["decoder_root"], "extraction": extraction}, "fields": fields}}
	profile := run("profile compile", map[string]any{"id": "amd-opn", "revision": "1", "hardware_class": "cpu", "metrics": []any{map[string]any{"predicate": "oem_identifier", "value_type": "string", "qualifier": qualifier, "required": true, "description": "Documentary AMD OPN; no tray-ID equivalence", "extensions": map[string]any{}}}, "extensions": map[string]any{}}, true, owner("0.3.0-dev")...)
	diagnosis := run("mapping diagnose", map[string]any{"profile": profile, "mapping": mapping}, true, owner("0.3.0-dev")...)
	if diagnosis["evidence_scope"] != "mapping_declarations_only" {
		t.Fatal("profile claimed source authority")
	}
	manifest := map[string]any{}
	for k, v := range source {
		manifest[k] = v
	}
	manifest["format"] = "opaque"
	coverage := seal(map[string]any{"protocol": "symphony.shv.coverage-profile.v1", "as_of": "2026-11-01", "selector": map[string]any{"op": "all"}})
	refreshInput := map[string]any{"expected_source_digest": revision, "captures": []any{map[string]any{"locator_id": "amd-pdf", "resolved_uri": uri, "redirect_chain": []any{}, "observed_at": time.Now().UTC().Truncate(time.Second).Format(time.RFC3339), "upstream_revision": map[string]any{"scheme": "amd-publication-revision", "value": "69290:1.00"}, "manifest": manifest, "completeness": "complete", "issues": []any{}}}, "mapping": mapping, "profile": coverage, "subject_ids": []any{}, "requirements": []any{map[string]any{"id": "match", "predicate": "oem_identifier", "operator": "eq", "value": "100-000000329", "qualifier": qualifier}, map[string]any{"id": "namespace", "predicate": "oem_identifier", "operator": "eq", "value": "100-000000329", "qualifier": "issuer=AMD;namespace=product-id-tray;profile=1"}}}
	refreshFlags := append(owner("0.4.0-dev"), "--source-prefix", prefix, "--source-version", "0.2.0-dev", "--state-root", state, "--tops-id", ssiagTestTOPSID, "--source-id", sourceID, "--source-root", sources)
	bundle := run("refresh build", refreshInput, true, refreshFlags...)
	run("refresh verify", bundle, true, refreshFlags...)
	findings := object(bundle["evaluation"])["findings"].([]any)
	if object(findings[0])["status"] != "supported" || object(findings[1])["status"] != "unresolved" {
		t.Fatal("namespace evidence was silently equated")
	}
	if object(object(bundle["coverage"])["counts"])["included"] != json.Number("1") {
		t.Fatal("selected coverage changed")
	}
	endpoint := map[string]any{"bundle_path": write("refresh-bundle", bundle), "source_root": sources, "state_root": state, "tops_id": ssiagTestTOPSID, "source_id": sourceID, "source_prefix": prefix, "source_version": "0.2.0-dev", "kernel_prefix": prefix, "kernel_version": "0.4.0-dev"}
	partitionResult := run("partition from-refresh", endpoint, true, owner("0.4.0-dev")...)
	partition := object(partitionResult["partition"])
	partitionManifest := run("partition manifest", map[string]any{"entries": []any{map[string]any{"partition_digest": partition["digest"], "partition": partition}}, "required_references": []any{}}, true, owner("0.4.0-dev")...)
	graphRoot := filepath.Join(root, "graph")
	if err = os.Mkdir(graphRoot, 0700); err != nil {
		t.Fatal(err)
	}
	graphFlags := []string{"--connector-prefix", storePrefix, "--connector-version", "0.4.0-dev", "--store-root", graphRoot}
	prepared := run("graph store prepare", map[string]any{"tops_id": ssiagTestTOPSID, "namespace": "qualification", "operation_id": "graph-1", "graph": bundle["catalogue_graph"]}, true, graphFlags...)
	committed := run("graph store commit", map[string]any{"tops_id": ssiagTestTOPSID, "namespace": "qualification", "operation_id": "graph-1", "expected_intent_digest": object(prepared["intent"])["digest"]}, true, graphFlags...)
	storeSelection := map[string]any{"root": graphRoot, "tops_id": ssiagTestTOPSID, "namespace": "qualification", "snapshot_digest": committed["snapshot_digest"], "prefix": storePrefix, "version": "0.4.0-dev"}
	pubFlags := append(owner("0.4.0-dev"), "--tops-id", ssiagTestTOPSID, "--catalogue-id", "bounded-amd", "--state-root", filepath.Join(root, "publication"))
	proposal := map[string]any{"operation_id": "preview", "reason": "bounded documentary catalogue preview only", "desired": map[string]any{"manifest": partitionManifest, "policy": map[string]any{"missing_partitions": "reject", "missing_references": "reject"}, "partition_prefix": prefix, "partition_version": "0.4.0-dev", "members": []any{map[string]any{"partition_digest": partition["digest"], "endpoint": endpoint, "store": storeSelection}}}}
	preview := run("catalogue publication plan", proposal, true, pubFlags...)
	if len(object(object(preview["head"])["definition"])["members"].([]any)) != 1 {
		t.Fatal("preview lost selected partition")
	}
	pubStatus := run("catalogue publication status", nil, true, pubFlags...)
	if pubStatus["head"] != nil {
		t.Fatal("preview published a catalogue")
	}
	if err = os.WriteFile(filepath.Join(sources, source["path"].(string)), []byte("changed retained source"), 0600); err != nil {
		t.Fatal(err)
	}
	run("refresh verify", bundle, false, refreshFlags...)
	run("catalogue publication plan", proposal, false, pubFlags...)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	executableBytes, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	write("summary", map[string]any{"cli_execution_mode": "current qxctl Go test helper subprocess", "cli_executable_sha256": fmt.Sprintf("%x", sha256.Sum256(executableBytes)), "status": "passed", "calls": calls, "expected_refusals": refusals, "skips": 0, "selected_models": []string{"7313"}, "decoded_rows": 28, "documentary_lineages": 1, "source_sha256": source["digest"], "decoder": request["decoder"], "source_revision": revision, "partition_digest": partition["digest"], "preview_digest": preview["digest"], "publication_applied": false, "authorization_evidence": "real private SSIAG/STAV policy-decision commitment; no source-write receipt", "physical_hardware_tested": false})
	t.Logf("SHV bounded workflow: %d calls, %d expected refusals, 0 skips", calls, refusals)
}
