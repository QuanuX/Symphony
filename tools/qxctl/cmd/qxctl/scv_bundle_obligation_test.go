package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvworkflow"
)

// Original logical outcomes are the actual recorded C++ fixture. This fake
// owner admits only exact registered inputs/identities, isolating orchestration
// and immutable publication; installed acceptance supplies process evidence.
func bundledObligationFixture(t *testing.T, bundledBefore, bundledAfter bool) (*workflowRunner, scvworkflow.Store, map[string]any) {
	t.Helper()
	return bundledObligationFixtureAt(t, bundledBefore, bundledAfter, workflowStore(t))
}

func bundledObligationFixtureAt(t *testing.T, bundledBefore, bundledAfter bool, store scvworkflow.Store) (*workflowRunner, scvworkflow.Store, map[string]any) {
	t.Helper()
	fixture := logicalReferenceFixture(t)["operations"].([]any)[3].(map[string]any)
	logical := fixture["logical_input"].(map[string]any)
	selected := knowledgeengine.Installation{Role: "scv", Version: "0.10.0-dev", Prefix: store.Root, ReceiptDigest: "synthetic receipt", ExecutableDigest: "synthetic executable"}
	r := &workflowRunner{installation: selected}
	registered := map[string]json.RawMessage{}
	key := func(inst knowledgeengine.Installation, op string, input any) string {
		raw, err := knowledgeengine.SCVCanonical(map[string]any{"installation": inst, "operation": op, "input": input})
		if err != nil {
			t.Fatal(err)
		}
		return string(raw)
	}
	register := func(record scvworkflow.Record) {
		registered[key(record.Installation, record.Operation, record.Input)] = record.Artifact
	}
	refs := map[string]string{}
	for _, item := range []struct {
		name   string
		bundle bool
	}{{"before", bundledBefore}, {"after", bundledAfter}} {
		artifact := logical[item.name].(map[string]any)
		inst := knowledgeengine.Installation{Role: "scv", Version: "0.7.0-dev", Prefix: store.Root, ReceiptDigest: "synthetic original receipt", ExecutableDigest: "synthetic original executable"}
		var record scvworkflow.Record
		if item.bundle {
			inst.Version = "0.9.0-dev"
			record = logicalTestBundle(t, inst, "composition_explore", artifact["input"], artifact)
		} else {
			record = logicalTestDirect(t, inst, "composition_explore", artifact["input"], artifact)
		}
		register(record)
		refs[item.name] = logicalRetain(t, store, record)
	}
	followup := logicalTestBundle(t, selected, "composition_followup", logical, fixture["logical_result"])
	register(followup)
	r.owner = func(inst knowledgeengine.Installation, op string, input any) (json.RawMessage, error) {
		if value, ok := registered[key(inst, op, input)]; ok {
			return value, nil
		}
		return nil, fmt.Errorf("synthetic owner unavailable or exact registered input changed")
	}
	return r, store, map[string]any{"protocol": "symphony.qxctl.scv-obligation-link-request.v2", "transport": "bundle", "before_ref": refs["before"], "after_ref": refs["after"], "submissions": logical["submissions"]}
}

func TestSCVBundledObligationLinkRetainsMixedOriginalProvenance(t *testing.T) {
	for _, pair := range []struct{ a, b bool }{{false, false}, {false, true}, {true, false}, {true, true}} {
		t.Run(fmt.Sprintf("%t_%t", pair.a, pair.b), func(t *testing.T) {
			r, s, input := bundledObligationFixture(t, pair.a, pair.b)
			var calls []string
			owner := r.owner
			r.owner = func(inst knowledgeengine.Installation, op string, in any) (json.RawMessage, error) {
				calls = append(calls, inst.Version+":"+op)
				return owner(inst, op, in)
			}
			raw, err := r.bundleObligationLink(s, "retain", input)
			if err != nil {
				t.Fatal(err)
			}
			result := workflowValue(t, raw)
			link := result["link"].(map[string]any)
			if result["protocol"] != "symphony.qxctl.scv-obligation-link-result.v2" || result["validation"] != "owner_replayed" || link["transport"] != "bundle" {
				t.Fatal("wrong result semantics")
			}
			if result["followup_ref"] != link["followup"].(map[string]any)["record_ref"] {
				t.Fatal("follow-up reference mismatch")
			}
			for _, item := range []struct {
				name   string
				bundle bool
			}{{"before", pair.a}, {"after", pair.b}} {
				d := link[item.name].(map[string]any)
				if d["record_ref"] != input[item.name+"_ref"] || d["logical_operation"] != "composition_explore" || (d["transport"] != nil) != item.bundle {
					t.Fatal("source provenance or logical identity changed")
				}
			}
			followup := link["followup"].(map[string]any)
			if followup["transport"] == nil || followup["logical_operation"] != "composition_followup" {
				t.Fatal("follow-up did not retain its actual bundle operation")
			}
			expected := []string{}
			for _, bundle := range []bool{pair.a, pair.b} {
				if bundle {
					expected = append(expected, "0.9.0-dev:composition_bundle_evaluate")
				} else {
					expected = append(expected, "0.7.0-dev:composition_explore")
				}
			}
			expected = append(expected, "0.10.0-dev:composition_bundle_evaluate")
			if fmt.Sprint(calls) != fmt.Sprint(expected) {
				t.Fatal("original owner substitution", calls)
			}
			calls = nil
			r.installation = knowledgeengine.Installation{}
			shown, err := r.bundleObligationLink(s, "show", map[string]any{"link_ref": result["link_ref"]})
			if err != nil || !scvworkflow.Same(raw, shown) {
				t.Fatal("show lost original owners or relationship identity", err)
			}
			if fmt.Sprint(calls) != fmt.Sprint(expected) {
				t.Fatal("show substituted caller-selected owner", calls)
			}
		})
	}
}

func TestSCVBundledObligationLinkRecoversBothPublicationBoundaries(t *testing.T) {
	for _, boundary := range []string{"obligation_result_retained", "obligation_link_retained"} {
		t.Run(boundary, func(t *testing.T) {
			r, s, input := bundledObligationFixture(t, false, true)
			fired := false
			r.afterCheckpoint = func(point string) error {
				if point == boundary && !fired {
					fired = true
					return errors.New("synthetic interruption after durable publication")
				}
				return nil
			}
			if _, err := r.bundleObligationLink(s, "retain", input); err == nil || !fired {
				t.Fatal("missing interruption", err)
			}
			r.afterCheckpoint = nil
			a, err := r.bundleObligationLink(s, "retain", input)
			if err != nil {
				t.Fatal(err)
			}
			b, err := r.bundleObligationLink(s, "retain", input)
			if err != nil || !scvworkflow.Same(a, b) {
				t.Fatal("recovery or immutable reuse changed identity", err)
			}
		})
	}
}

func TestSCVBundledObligationLinkRejectsResealedDescriptorAndCorrespondence(t *testing.T) {
	r, s, input := bundledObligationFixture(t, false, true)
	raw, err := r.bundleObligationLink(s, "retain", input)
	if err != nil {
		t.Fatal(err)
	}
	original := workflowValue(t, raw)["link"].(map[string]any)
	mutations := map[string]func(map[string]any){
		"root":           func(m map[string]any) { m["root"] = "/different-root" },
		"tops":           func(m map[string]any) { m["tops_id"] = "11111111-2222-3333-4444-555555555555" },
		"transport":      func(m map[string]any) { m["transport"] = "inline" },
		"logical_digest": func(m map[string]any) { d := m["after"].(map[string]any); d["logical_digest"] = d["artifact_digest"] },
		"transport_root": func(m map[string]any) {
			d := m["after"].(map[string]any)
			d["transport"].(map[string]any)["result_root_digest"] = d["logical_digest"]
		},
		"descriptor_extra":            func(m map[string]any) { m["before"].(map[string]any)["verified"] = true },
		"matched_before_substitution": func(m map[string]any) { m["before"] = m["after"] },
		"submissions":                 func(m map[string]any) { m["submissions_digest"] = m["before"].(map[string]any)["logical_digest"] },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			link := workflowClone(t, original)
			mutate(link)
			wrapper, e := scvworkflow.Seal(map[string]any{"protocol": "symphony.qxctl.scv-workflow-payload.v1", "input": link})
			if e != nil {
				t.Fatal(e)
			}
			var ref string
			e = s.With("", true, func(session *scvworkflow.Session, _ *scvworkflow.Run) error {
				var err error
				ref, err = session.Put("payloads", wrapper)
				return err
			})
			if e != nil {
				t.Fatal(e)
			}
			if _, e = r.bundleObligationLink(s, "show", map[string]any{"link_ref": ref}); e == nil {
				t.Fatal("accepted altered descriptor or source correspondence")
			}
		})
	}
}

func TestSCVBundledObligationLinkPreservesExplicitRequestAndOriginalOwners(t *testing.T) {
	r, s, input := bundledObligationFixture(t, false, true)
	for _, change := range []string{"protocol", "transport", "extra"} {
		t.Run(change, func(t *testing.T) {
			value := workflowClone(t, input)
			value[change] = "changed"
			if _, err := r.bundleObligationLink(s, "retain", value); err == nil {
				t.Fatal("accepted implicit or altered v2 request")
			}
		})
	}
	raw, err := r.bundleObligationLink(s, "retain", input)
	if err != nil {
		t.Fatal(err)
	}
	owner := r.owner
	r.owner = func(inst knowledgeengine.Installation, op string, in any) (json.RawMessage, error) {
		if inst.Version == "0.7.0-dev" {
			return nil, errors.New("original legacy owner unavailable")
		}
		return owner(inst, op, in)
	}
	if _, err = r.bundleObligationLink(s, "show", map[string]any{"link_ref": workflowValue(t, raw)["link_ref"]}); err == nil {
		t.Fatal("show substituted unavailable original owner")
	}
	r.owner = owner
	r.installation.Version = "0.9.0-dev"
	if _, err = r.bundleObligationLink(s, "retain", input); err == nil {
		t.Fatal("earlier adapter release accepted new relationship request")
	}
}

func TestSCVBundledObligationLinkConcurrentIdenticalPublication(t *testing.T) {
	r, s, input := bundledObligationFixture(t, true, false)
	var wg sync.WaitGroup
	results := make(chan json.RawMessage, 2)
	failures := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			raw, err := r.bundleObligationLink(s, "retain", input)
			if err != nil {
				failures <- err
			} else {
				results <- raw
			}
		}()
	}
	wg.Wait()
	close(results)
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
	var first json.RawMessage
	count := 0
	for raw := range results {
		if first == nil {
			first = raw
		} else if !scvworkflow.Same(first, raw) {
			t.Fatal("concurrent publication produced different relationships")
		}
		count++
	}
	if count != 2 {
		t.Fatal("missing successful publication")
	}
}

func TestSCVBundledObligationLinkRejectsPoisonedExistingPayload(t *testing.T) {
	root := filepath.Join(t.TempDir(), "�")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	store, err := scvworkflow.New(root, ssiagTestTOPSID)
	if err != nil {
		t.Fatal(err)
	}
	r, s, input := bundledObligationFixtureAt(t, false, true, store)
	first, err := r.bundleObligationLink(s, "retain", input)
	if err != nil {
		t.Fatal(err)
	}
	ref := workflowValue(t, first)["link_ref"].(string)
	var original []byte
	if err = s.With("", false, func(session *scvworkflow.Session, _ *scvworkflow.Run) error {
		var e error
		original, e = session.Get("payloads", ref)
		return e
	}); err != nil {
		t.Fatal(err)
	}
	poisoned := bytes.Replace(original, []byte("�"), []byte(`\ud800`), 1)
	if bytes.Equal(original, poisoned) {
		t.Fatal("missing free Unicode root fixture")
	}
	if digest, e := scvworkflow.Digest(poisoned); e != nil || digest != ref {
		t.Fatal("expected legacy-normalized digest", e)
	}
	path := filepath.Join(s.Root, "symphony", "qxctl", "scv", "workflows-v1", s.TOPSID, "payloads", strings.TrimPrefix(ref, "sha256:")+".json")
	// Deliberate corruption only inside this private store. Publication must
	// neither overwrite it nor report success based on the proposed good bytes.
	if err = os.WriteFile(path, poisoned, 0600); err != nil {
		t.Fatal(err)
	}
	reported := false
	r.afterCheckpoint = func(point string) error {
		if point == "obligation_link_retained" {
			reported = true
		}
		return nil
	}
	if _, err = r.bundleObligationLink(s, "retain", input); err == nil || reported {
		t.Fatal("claimed publication of a malformed retained payload", err)
	}
	if _, err = r.bundleObligationLink(s, "show", map[string]any{"link_ref": ref}); err == nil {
		t.Fatal("show accepted poisoned payload")
	}
	if err = os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	r.afterCheckpoint = nil
	again, err := r.bundleObligationLink(s, "retain", input)
	if err != nil || !scvworkflow.Same(first, again) {
		t.Fatal("explicitly restored original object did not recover", err)
	}
}

func TestSCVBundledObligationLinkRejectsDirectFollowupBeforeInvocation(t *testing.T) {
	r, s, input := bundledObligationFixture(t, false, true)
	raw, err := r.bundleObligationLink(s, "retain", input)
	if err != nil {
		t.Fatal(err)
	}
	link := workflowValue(t, raw)["link"].(map[string]any)
	fixture := logicalReferenceFixture(t)["operations"].([]any)[3].(map[string]any)
	legacy := logicalTestDirect(t, knowledgeengine.Installation{Role: "scv", Version: "0.8.0-dev"}, "composition_followup", fixture["logical_input"], fixture["logical_result"])
	logicalRetain(t, s, legacy)
	view, err := projectLogicalRecord(legacy, "composition_followup")
	if err != nil {
		t.Fatal(err)
	}
	link["followup"] = view.Descriptor
	wrapper, err := scvworkflow.Seal(map[string]any{"protocol": "symphony.qxctl.scv-workflow-payload.v1", "input": link})
	if err != nil {
		t.Fatal(err)
	}
	var ref string
	if err = s.With("", true, func(session *scvworkflow.Session, _ *scvworkflow.Run) error {
		var e error
		ref, e = session.Put("payloads", wrapper)
		return e
	}); err != nil {
		t.Fatal(err)
	}
	owner := r.owner
	invoked := false
	r.owner = func(inst knowledgeengine.Installation, op string, in any) (json.RawMessage, error) {
		if op == "composition_followup" {
			invoked = true
			return legacy.Artifact, nil
		}
		return owner(inst, op, in)
	}
	if _, err = r.bundleObligationLink(s, "show", map[string]any{"link_ref": ref}); err == nil || invoked {
		t.Fatal("unsupported direct follow-up reached native invocation", err)
	}
}
