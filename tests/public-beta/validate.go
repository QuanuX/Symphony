// Validate the frozen preparation dossier; never candidate or release acceptance.
// This host-only Go tool uses local Git and does not run JSON-supplied commands.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"unicode/utf8"
)

type object = map[string]any

func require(ok bool, message string) {
	if !ok {
		panic(message)
	}
}
func get(o object, key string) any { v, ok := o[key]; require(ok, "missing field: "+key); return v }
func obj(v any) object             { o, ok := v.(map[string]any); require(ok, "JSON object required"); return o }
func list(v any) []any             { a, ok := v.([]any); require(ok, "JSON array required"); return a }
func str(v any) string             { s, ok := v.(string); require(ok, "JSON string required"); return s }
func count(v any) int {
	n, ok := v.(json.Number)
	require(ok, "JSON number required")
	i, e := n.Int64()
	require(e == nil && i >= 0, "nonnegative integer required")
	return int(i)
}
func eq(a, b any) bool    { return reflect.DeepEqual(a, b) }
func isFalse(v any) bool  { b, ok := v.(bool); return ok && !b }
func isTrue(v any) bool   { b, ok := v.(bool); return ok && b }
func sha(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func read(path string) []byte {
	b, e := os.ReadFile(path)
	if e != nil {
		panic(e)
	}
	return b
}
func file(path string) bool { s, e := os.Stat(path); return e == nil && s.Mode().IsRegular() }
func texts(v any) []string {
	a := list(v)
	s := make([]string, len(a))
	for i, x := range a {
		s[i] = str(x)
	}
	return s
}
func set(values []string) map[string]bool {
	s := map[string]bool{}
	for _, x := range values {
		s[x] = true
	}
	return s
}
func unique(values []string, label string) map[string]bool {
	s := set(values)
	require(len(s) == len(values), "duplicate "+label)
	return s
}
func subset(a, b map[string]bool) bool {
	for x := range a {
		if !b[x] {
			return false
		}
	}
	return true
}
func keys(o object) []string {
	s := make([]string, 0, len(o))
	for k := range o {
		s = append(s, k)
	}
	return s
}
func ids(rows []any, key string) []string {
	s := make([]string, len(rows))
	for i, r := range rows {
		s[i] = str(get(obj(r), key))
	}
	return s
}
func optionalStrings(o object, key string) []string {
	v, ok := o[key]
	if !ok {
		return nil
	}
	return texts(v)
}
func nonempty(v any) bool {
	switch x := v.(type) {
	case string:
		return x != ""
	case []any:
		return len(x) > 0
	case map[string]any:
		return len(x) > 0
	case bool:
		return x
	case json.Number:
		n, err := x.Float64()
		return err != nil || n != 0
	default:
		return v != nil
	}
}

// The default encoding/json object parser accepts duplicate keys. Keep the
// original dossier's duplicate-key refusal at every nested object boundary.
func parseJSON(data []byte) any {
	require(utf8.Valid(data), "invalid UTF-8 JSON")
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	nodes := 0
	var value func(int) any
	value = func(depth int) any {
		nodes++
		require(depth <= 64 && nodes <= 1000000, "JSON structure exceeds bound")
		t, e := d.Token()
		if e != nil {
			panic(e)
		}
		delim, ok := t.(json.Delim)
		if !ok {
			return t
		}
		switch delim {
		case '{':
			out := object{}
			for d.More() {
				k, e := d.Token()
				if e != nil {
					panic(e)
				}
				name := str(k)
				_, seen := out[name]
				require(!seen, "duplicate JSON key: "+name)
				out[name] = value(depth + 1)
			}
			close, e := d.Token()
			require(e == nil && close == json.Delim('}'), "invalid JSON object")
			return out
		case '[':
			out := []any{}
			for d.More() {
				out = append(out, value(depth+1))
			}
			close, e := d.Token()
			require(e == nil && close == json.Delim(']'), "invalid JSON array")
			return out
		default:
			panic("unexpected JSON delimiter")
		}
	}
	result := value(0)
	_, e := d.Token()
	require(e == io.EOF, "trailing JSON data")
	return result
}

func validate(repo, folder string, historical bool) (result object, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			result = nil
			err = fmt.Errorf("%v", failure)
		}
	}()
	git := func(args ...string) []byte {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		b, e := cmd.Output()
		if e != nil {
			panic(fmt.Sprintf("git %s: %v: %s", args[0], e, strings.TrimSpace(stderr.String())))
		}
		return b
	}
	load := func(name string) object { return obj(parseJSON(read(filepath.Join(folder, name)))) }
	d, inventory, mapping := load("dossier.json"), load("source-inventory.json"), load("acceptance-mapping.json")
	paper, owners, capabilities, packages := load("paper-criteria.json"), load("owners.json"), load("capability-groups.json"), load("package-plan.json")
	base := str(get(d, "source_commit"))
	decoded, e := hex.DecodeString(base)
	require(e == nil && len(decoded) == 20 && strings.ToLower(base) == base, "invalid source commit")
	require(count(get(d, "format_version")) == 1, "unsupported dossier format")
	require(get(d, "state") == "prepared_for_manual_review", "not a preparation snapshot")
	for _, f := range []string{"scope_approved", "implementation_continuation_approved", "release_acceptance_complete", "publication_authorized"} {
		require(isFalse(get(d, f)), "preparation must not assert "+f)
	}
	freeze := obj(get(d, "feature_freeze"))
	require(isTrue(get(freeze, "in_force")), "feature freeze must remain active")
	require(isFalse(get(freeze, "new_features_authorized")), "new feature authority is not granted")
	require(strings.TrimSpace(string(git("rev-parse", base+"^{tree}"))) == get(d, "source_tree"), "source tree mismatch")
	git("merge-base", "--is-ancestor", base, "HEAD")
	invref := obj(get(d, "source_inventory"))
	require(get(invref, "path") == "source-inventory.json", "unexpected inventory path")
	require(sha(read(filepath.Join(folder, "source-inventory.json"))) == get(invref, "sha256"), "inventory digest mismatch")
	require(get(inventory, "source_commit") == base && get(inventory, "source_tree") == get(d, "source_tree"), "inventory source binding mismatch")
	for _, data := range []object{mapping, owners, capabilities, packages} {
		require(count(get(data, "format_version")) == 1 && get(data, "source_commit") == base, "companion source/version mismatch")
	}
	const tracking = ".github/RELEASE-GATES.md"
	for path, digest := range obj(get(d, "source_fingerprints")) {
		require(!filepath.IsAbs(path) && !set(strings.Split(filepath.ToSlash(path), "/"))[".."], "unsafe source fingerprint path")
		require(sha(git("show", base+":"+path)) == digest, "historical fingerprint mismatch: "+path)
		if !historical && path != tracking {
			require(sha(read(filepath.Join(repo, path))) == digest, "working source drift: "+path)
		}
	}
	if !historical {
		changed := string(git("diff", "--name-only", base, "--")) + string(git("ls-files", "--others", "--exclude-standard"))
		for _, path := range strings.Split(strings.TrimSpace(changed), "\n") {
			if path != "" {
				require(strings.HasPrefix(path, "tests/public-beta/") || path == tracking, "source outside dossier changed; regenerate/review source binding: "+path)
			}
		}
	}
	sourceJSON := func(path string) object { return obj(parseJSON(git("show", base+":"+path))) }
	sourceMap, plan, cloudflare := sourceJSON("cmake/mvp-dependencies.json"), sourceJSON("tests/launch-acceptance/plan.json"), sourceJSON("tests/cloudflare-release/support-matrix.json")
	for _, k := range []string{"component_roots", "native_components", "native_build_order", "go_modules", "swift_provider", "external_dependencies"} {
		require(eq(get(inventory, k), get(sourceMap, k)), "inventory projection drift: "+k)
	}
	for k, path := range map[string]string{"dependency_map_sha256": "cmake/mvp-dependencies.json", "acceptance_plan_sha256": "tests/launch-acceptance/plan.json", "platform_profile_catalog_sha256": "cmake/mvp-platform-profiles.json"} {
		require(get(inventory, k) == sha(git("show", base+":"+path)), "inventory binding mismatch: "+k)
	}
	require(eq(get(inventory, "platform_profile_catalog"), sourceJSON("cmake/mvp-platform-profiles.json")), "platform inventory drift")
	for _, k := range []string{"selected_components", "promised_provider_operations", "reviewer", "reviewed_at"} {
		require(get(inventory, k) == nil, "raw inventory approval changed: "+k)
	}
	for _, data := range []object{inventory, mapping} {
		for _, f := range []string{"release_acceptance_complete", "publication_authorized"} {
			require(isFalse(get(data, f)), "companion asserts "+f)
		}
		for _, k := range []string{"required_profiles", "required_scenarios", "report_fields"} {
			require(eq(get(data, k), get(plan, k)), "core acceptance contract changed: "+k)
		}
		inputs := obj(get(data, "required_inputs"))
		require(eq(set(keys(inputs)), set(texts(get(plan, "required_inputs")))), "required input lost")
		for _, v := range inputs {
			require(v == nil, "raw input supplied without review")
		}
	}
	roles := list(get(owners, "roles"))
	roleIDs := unique(ids(roles, "id"), "owner role")
	named := map[string]string{}
	for _, r := range roles {
		row := obj(r)
		if get(row, "person") != nil {
			named[str(get(row, "id"))] = str(get(row, "person"))
		}
	}
	require(eq(named, map[string]string{"release_approver": "@quantDIY", "rollback_primary": "@quantDIY"}), "unexpected named assignment")
	decisions := list(get(d, "decisions"))
	decisionIDs := unique(ids(decisions, "id"), "decision")
	expectedDecisions := map[string]bool{}
	for i := 1; i <= 7; i++ {
		expectedDecisions[fmt.Sprintf("D%02d", i)] = true
	}
	require(eq(decisionIDs, expectedDecisions), "decision coverage changed")
	for _, x := range decisions {
		r := obj(x)
		require(get(r, "state") == "pending_manual_review" && get(r, "approved_by") == nil && get(r, "approved_at") == nil, "preparation asserts decision approval")
		require(roleIDs[str(get(r, "decision_owner_role"))], "unknown decision owner")
		for _, path := range texts(get(r, "references")) {
			require(file(filepath.Join(folder, path)), "missing decision reference: "+path)
		}
	}
	roots := set(ids(list(get(inventory, "component_roots")), "path"))
	groups := list(get(capabilities, "groups"))
	unique(ids(groups, "id"), "capability group")
	assigned := map[string]int{}
	for _, g := range groups {
		for _, p := range texts(get(obj(g), "component_roots")) {
			assigned[p]++
		}
	}
	assignedSet := map[string]bool{}
	for p, n := range assigned {
		assignedSet[p] = true
		require(n == 1, "component root assigned more than once")
	}
	require(eq(assignedSet, roots), "capability roots missing or unknown")
	scenarios := set(texts(get(plan, "required_scenarios")))
	for _, g := range groups {
		r := obj(g)
		for _, k := range []string{"operations", "platform_claims", "implementation_state", "qualification_state", "proposed_selection", "freeze_disposition", "evidence_paths"} {
			require(nonempty(get(r, k)), "capability group lacks "+k+": "+str(get(r, "id")))
		}
		selected := obj(get(r, "proposed_selection"))
		ss := texts(get(r, "acceptance_scenarios"))
		if len(ss) == 0 {
			require(get(selected, "state") == "proposed_build_source_only" || get(selected, "state") == "proposed_no_binary_proposal_visible", "runtime group lacks acceptance scenarios")
		}
		for _, s := range ss {
			require(scenarios[s], "unknown capability scenario")
		}
		require(get(selected, "approval") == "unapproved", "capability approval invented")
		for _, path := range texts(get(r, "evidence_paths")) {
			require(file(filepath.Join(repo, path)), "missing capability evidence: "+path)
		}
	}
	require(isFalse(get(capabilities, "capability_selection_approved")) && isFalse(get(capabilities, "publication_authorized")), "capability approval asserted")
	require(get(packages, "approval_state") == "pending_human_approval", "package selection approval invented")
	require(isFalse(get(packages, "publication_authorized")) && isFalse(get(packages, "real_build_execution_authorized_by_this_document")), "package plan grants execution/publication")
	require(get(packages, "candidate_source_commit") == nil && get(packages, "candidate_source_tree") == nil, "candidate identity fabricated")
	recipes := list(get(packages, "package_recipes"))
	recipeIDs := unique(ids(recipes, "id"), "package recipe")
	inputs := list(get(packages, "open_inputs"))
	inputIDs := unique(ids(inputs, "id"), "package input")
	trust := list(get(packages, "trust_requirements"))
	trustIDs := unique(ids(trust, "id"), "trust requirement")
	dispositions := list(get(packages, "source_root_dispositions"))
	require(eq(unique(ids(dispositions, "source_root"), "package source root"), roots), "package root coverage mismatch")
	for _, v := range dispositions {
		r := obj(v)
		if get(r, "proposed_recipe_id") != nil {
			require(recipeIDs[str(get(r, "proposed_recipe_id"))], "unknown root recipe")
		}
	}
	for _, v := range inputs {
		r := obj(v)
		require(decisionIDs[str(get(r, "decision_id"))] && get(r, "state") == "open" && get(r, "value") == nil, "package input silently supplied")
	}
	for _, v := range trust {
		require(subset(set(texts(get(obj(v), "input_ids"))), inputIDs), "unknown trust input")
	}
	byRecipe := map[string]object{}
	for _, v := range recipes {
		r := obj(v)
		byRecipe[str(get(r, "id"))] = r
		require(get(r, "approval_state") == "pending_human_approval" && len(list(get(r, "supported_profiles"))) == 0, "recipe claims acceptance")
		require(subset(set(texts(get(r, "source_roots"))), roots), "unknown recipe root")
		require(subset(set(texts(get(r, "trust_requirements"))), trustIDs) && subset(set(texts(get(r, "open_input_ids"))), inputIDs), "recipe trust/input mismatch")
		for _, v := range obj(get(r, "observed_artifacts")) {
			require(v == nil, "fabricated candidate artifact")
		}
		require(subset(set(optionalStrings(r, "dependency_recipe_ids")), recipeIDs), "unknown dependency recipe")
		for _, f := range []string{"content_recipe_ids", "separately_selectable_recipe_ids"} {
			require(subset(set(optionalStrings(r, f)), recipeIDs), "unknown recipe reference: "+f)
		}
	}
	for _, v := range recipes {
		r := obj(v)
		if get(r, "kind") == "delivery_set_not_single_native_archive" {
			contents := unique(texts(get(r, "content_recipe_ids")), "delivery content")
			for member := range contents {
				require(subset(set(optionalStrings(byRecipe[member], "dependency_recipe_ids")), contents), "delivery set omits native dependency closure")
			}
		}
	}
	nativeRows := list(get(inventory, "native_components"))
	nativeByRoot := map[string]object{}
	for _, v := range nativeRows {
		r := obj(v)
		if isTrue(get(r, "installable")) {
			nativeByRoot[str(get(r, "path"))] = r
		}
	}
	recipeRoots := map[string]bool{}
	for _, v := range recipes {
		r := obj(v)
		if get(r, "kind") == "native_receipt_unit" {
			rs := texts(get(r, "source_roots"))
			require(len(rs) > 0, "native receipt root missing")
			recipeRoots[rs[0]] = true
			c, ok := nativeByRoot[rs[0]]
			require(ok, "native receipt unit omitted or invented")
			require(get(r, "proposed_package_version") == get(c, "package_release_version") && eq(get(r, "sdk_exports"), get(c, "exported_packages")), "native version/SDK export drift")
		}
	}
	nativeRoots := map[string]bool{}
	for p := range nativeByRoot {
		nativeRoots[p] = true
	}
	require(eq(recipeRoots, nativeRoots), "native receipt unit omitted or invented")
	campaigns := list(get(mapping, "campaigns"))
	campaignIDs := unique(ids(campaigns, "id"), "campaign")
	require(eq(set(ids(campaigns, "scenario")), scenarios), "campaign coverage mismatch")
	for _, v := range campaigns {
		r := obj(v)
		require(roleIDs[str(get(r, "owner_role"))], "unknown campaign owner")
		require(get(r, "execution_status") == "not_run" && get(r, "acceptance_status") == "pending", "prepared campaign asserts result")
		require(subset(set(texts(get(r, "required_decisions"))), decisionIDs), "unknown campaign decision")
		for _, p := range texts(get(r, "existing_harnesses")) {
			require(file(filepath.Join(repo, p)), "missing harness reference: "+p)
		}
	}
	cells := list(get(mapping, "core_cells"))
	unique(ids(cells, "id"), "core cell")
	expected := map[string]bool{}
	for _, p := range texts(get(plan, "required_profiles")) {
		for s := range scenarios {
			expected[p+"::"+s] = true
		}
	}
	actual := map[string]bool{}
	for _, v := range cells {
		r := obj(v)
		actual[str(get(r, "profile"))+"::"+str(get(r, "scenario"))] = true
	}
	require(len(cells) == 36 && len(expected) == 36 && count(get(mapping, "core_cell_count")) == 36 && eq(actual, expected), "36 core profile/scenario cells must be preserved")
	for _, v := range cells {
		r := obj(v)
		require(get(r, "id") == str(get(r, "profile"))+"::"+str(get(r, "scenario")), "core cell ID mismatch")
		require(isTrue(get(r, "mandatory")) && get(r, "candidate_sha256") == nil && get(r, "skips") == nil, "core cell fabricates qualification")
	}
	requiredCriteria := map[string]bool{}
	for i := 1; i <= 13; i++ {
		for j := 1; j <= 4; j++ {
			requiredCriteria[fmt.Sprintf("C%02d.%d", i, j)] = true
		}
	}
	for i := 1; i <= 6; i++ {
		requiredCriteria[fmt.Sprintf("UE%d", i)] = true
	}
	criteria := list(get(paper, "criteria"))
	mapped := list(get(mapping, "paper_criteria"))
	paperIDs := unique(ids(criteria, "criterion_id"), "paper criterion")
	mappedIDs := unique(ids(mapped, "criterion_id"), "mapped criterion")
	require(eq(paperIDs, requiredCriteria) && eq(mappedIDs, requiredCriteria) && count(get(mapping, "paper_criterion_count")) == 58, "58 criterion coverage mismatch")
	paperByID := map[string]object{}
	for _, v := range criteria {
		r := obj(v)
		paperByID[str(get(r, "criterion_id"))] = r
	}
	for _, v := range mapped {
		r := obj(v)
		require(get(r, "criterion_specific_expected_outcome") == get(paperByID[str(get(r, "criterion_id"))], "paper_criterion_text"), "criterion text changed or omitted")
		require(roleIDs[str(get(r, "owner_role"))] && subset(set(texts(get(r, "campaign_ids"))), campaignIDs), "criterion owner/campaign mismatch")
		require(get(r, "scope_exclusion_decision") == nil && get(r, "applicability") == "proposed_pending_manual_scope_review", "criterion silently waived")
	}
	for _, v := range list(get(paper, "categories")) {
		r := obj(v)
		require(get(r, "current_accepted_score") == nil && get(r, "current_score_status") == "not_reassessed", "invented acceptance score")
	}
	cfCells := list(get(mapping, "cloudflare_operation_cells"))
	unique(ids(cfCells, "operation"), "Cloudflare operation")
	cfByOperation := map[string]object{}
	cfOperations := map[string]bool{}
	for _, v := range list(get(cloudflare, "resource_operations")) {
		r := obj(v)
		name := str(get(r, "operation"))
		cfByOperation[name] = r
		cfOperations[name] = true
	}
	require(eq(set(ids(cfCells, "operation")), cfOperations) && len(cfCells) == 23, "Cloudflare operation coverage changed")
	platforms := set(texts(get(cloudflare, "admitted_vendor_runtime_platforms")))
	for _, v := range cfCells {
		r := obj(v)
		s := cfByOperation[str(get(r, "operation"))]
		require(platforms[str(get(r, "runtime_platform"))], "unadmitted Cloudflare runtime")
		require(get(r, "http_method") == get(s, "method") && get(r, "provider_path_template") == get(s, "path"), "Cloudflare operation binding drift")
		require(get(r, "implementation_at_baseline") == get(s, "implementation") && get(r, "live_at_baseline") == get(s, "live"), "Cloudflare evidence status drift")
		require(get(r, "exact_host_profile") == nil && get(r, "host_profile_decision") == "D02", "invented Cloudflare host admission")
	}
	for _, rows := range [][]any{cells, mapped, cfCells, list(get(mapping, "additional_campaigns"))} {
		for _, v := range rows {
			r := obj(v)
			require(get(r, "execution_status") == "not_run" && get(r, "review_status") == "not_accepted" && len(list(get(r, "evidence"))) == 0, "prepared mapping asserts acceptance evidence")
			if _, ok := r["campaign_ids"]; ok {
				require(subset(set(texts(get(r, "campaign_ids"))), campaignIDs), "unknown supplemental campaign")
			}
		}
	}
	status := "valid_preparation_only"
	if historical {
		status = "valid_historical_preparation_only"
	}
	return object{"status": status, "source_commit": base, "current_product_source_verified": !historical, "component_roots": len(roots), "native_components": len(nativeRows), "capability_groups": len(groups), "core_cells": len(cells), "paper_criteria": len(requiredCriteria), "cloudflare_operations": len(cfCells), "scope_approved": false, "release_acceptance_complete": false, "publication_authorized": false}, nil
}

func main() {
	repo := flag.String("repo", "", "source repository; defaults to the current Git checkout")
	folder := flag.String("dossier", "", "alternate dossier copy; defaults to source tests/public-beta")
	historical := flag.Bool("historical", false, "validate preserved preparation against its Git source, not current product files")
	flag.Parse()
	fail := func(err error) { fmt.Fprintln(os.Stderr, "INVALID preparation dossier:", err); os.Exit(1) }
	if flag.NArg() != 0 {
		fail(fmt.Errorf("unexpected positional arguments"))
	}
	if *repo == "" {
		b, e := exec.Command("git", "rev-parse", "--show-toplevel").Output()
		if e != nil {
			fail(e)
		}
		*repo = strings.TrimSpace(string(b))
	}
	absolute, e := filepath.Abs(*repo)
	if e != nil {
		fail(e)
	}
	*repo = absolute
	if *folder == "" {
		*folder = filepath.Join(*repo, "tests/public-beta")
	}
	result, e := validate(*repo, *folder, *historical)
	if e != nil {
		fail(e)
	}
	out, e := json.MarshalIndent(result, "", "  ")
	if e != nil {
		fail(e)
	}
	fmt.Println(string(out))
}
