package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type betaFixture struct {
	packages, dependencies, identities, plan, mapping betaObject
}

func betaTestFixture(t *testing.T) betaFixture {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source location unavailable")
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	read := func(path string) betaObject { return betaRead(filepath.Join(root, path)) }
	return betaFixture{read("tests/public-beta/package-plan.json"), read("cmake/mvp-dependencies.json"),
		read("cmake/mvp-release-identities.json"), read("tests/launch-acceptance/plan.json"), read("tests/public-beta/acceptance-mapping.json")}
}
func (f betaFixture) construction() (betaObject, error) {
	return betaConstruction(f.packages, f.dependencies, f.identities, f.plan)
}
func (f betaFixture) workorders() (betaObject, error) { return betaWorkorders(f.mapping, f.plan) }
func (f betaFixture) recipe(id string) betaObject {
	return betaEntry(betaUnique(f.packages["package_recipes"], "id"), id)
}
func (f betaFixture) core() betaObject {
	return f.recipe("PKG-CORE-" + betaStrings(f.plan["required_profiles"])[0])
}
func (f betaFixture) native() betaObject {
	return f.recipe("PKG-NATIVE-knowledge-vector-engine-cpp")
}
func betaRefused(t *testing.T, err error, fragment string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), fragment) {
		t.Fatalf("expected refusal containing %q; got %v", fragment, err)
	}
}
func betaCheck(t *testing.T, ok bool, message string) {
	t.Helper()
	if !ok {
		t.Fatal(message)
	}
}
func TestCompleteSelectionRemainsPlanOnly(t *testing.T) {
	out, err := betaTestFixture(t).construction()
	if err != nil {
		t.Fatal(err)
	}
	profiles := betaObj(out["profiles"])
	betaCheck(t, len(profiles) == 4, "all four profiles retained")
	for _, value := range profiles {
		row := betaObj(value)
		betaCheck(t, len(betaArray(row["native_build_order"])) == 51, "51 native units retained")
		betaCheck(t, len(betaArray(row["go_commands"])) == 6, "six Go commands retained")
		betaCheck(t, row["candidate_identity"] == nil && row["retrieved_anaconda_identity"] == nil, "candidate/retrieval remain pending")
	}
	betaCheck(t, out["release_acceptance_complete"] == false && out["publication_authorized"] == false, "authority remains false")
}
func TestMissingCoreComponent(t *testing.T) {
	f := betaTestFixture(t)
	core := f.core()
	ids := betaArray(core["content_recipe_ids"])
	core["content_recipe_ids"] = ids[:len(ids)-1]
	_, err := f.construction()
	betaRefused(t, err, "selection changed")
}
func TestDuplicateCoreComponent(t *testing.T) {
	f := betaTestFixture(t)
	core := f.core()
	ids := betaArray(core["content_recipe_ids"])
	core["content_recipe_ids"] = append(ids, ids[0])
	_, err := f.construction()
	betaRefused(t, err, "duplicate selected recipe")
}
func TestMissingDependency(t *testing.T) {
	f := betaTestFixture(t)
	core := f.core()
	ids := []any{}
	for _, id := range betaArray(core["content_recipe_ids"]) {
		if id != "PKG-NATIVE-knowledge-vector-engine-cpp" {
			ids = append(ids, id)
		}
	}
	core["content_recipe_ids"] = ids
	_, err := f.construction()
	betaRefused(t, err, "missing native dependency")
}
func TestVersionDrift(t *testing.T) {
	f := betaTestFixture(t)
	f.native()["proposed_package_version"] = "9.0.0"
	_, err := f.construction()
	betaRefused(t, err, "current native owner")
}
func TestInstalledOptionDrift(t *testing.T) {
	f := betaTestFixture(t)
	f.native()["installed_options"] = betaObject{"SOURCE_FALLBACK": "ON"}
	_, err := f.construction()
	betaRefused(t, err, "current native owner")
}
func TestPrivateTargetSubstitution(t *testing.T) {
	f := betaTestFixture(t)
	f.native()["build_targets"] = []any{"replacement"}
	_, err := f.construction()
	betaRefused(t, err, "current native owner")
}
func TestGoVersionDrift(t *testing.T) {
	f := betaTestFixture(t)
	f.recipe("PKG-GO-qxctl")["proposed_package_version"] = "9.0.0"
	_, err := f.construction()
	betaRefused(t, err, "current Go owner")
}
func TestGoEntrypointSubstitution(t *testing.T) {
	f := betaTestFixture(t)
	f.recipe("PKG-GO-qxctl")["main_package"] = "./replacement"
	_, err := f.construction()
	betaRefused(t, err, "current Go owner")
}
func TestProfileOmission(t *testing.T) {
	f := betaTestFixture(t)
	f.native()["candidate_profiles"] = []any{}
	_, err := f.construction()
	betaRefused(t, err, "not declared on profile")
}
func TestOrderInversion(t *testing.T) {
	f := betaTestFixture(t)
	order := betaArray(f.dependencies["native_build_order"])
	for left, right := 0, len(order)-1; left < right; left, right = left+1, right-1 {
		order[left], order[right] = order[right], order[left]
	}
	_, err := f.construction()
	betaRefused(t, err, "non-topological")
}
func TestOrderOmission(t *testing.T) {
	f := betaTestFixture(t)
	f.dependencies["native_build_order"] = betaArray(f.dependencies["native_build_order"])[1:]
	_, err := f.construction()
	betaRefused(t, err, "native build order")
}
func TestDuplicateRecipe(t *testing.T) {
	f := betaTestFixture(t)
	f.packages["package_recipes"] = append(betaArray(f.packages["package_recipes"]), f.native())
	_, err := f.construction()
	betaRefused(t, err, "duplicate id")
}
func TestAllRequirementsStayPending(t *testing.T) {
	out, err := betaTestFixture(t).workorders()
	if err != nil {
		t.Fatal(err)
	}
	for kind, count := range map[string]int{"core": 36, "paper": 58, "supplemental": 31} {
		rows := betaArray(out[kind])
		betaCheck(t, len(rows) == count, "full "+kind+" coverage")
		for _, value := range rows {
			row := betaObj(value)
			betaCheck(t, row["acceptance_status"] == "pending" && row["final_artifact_bindings"] == nil && row["observed_result"] == nil, "every requirement remains pending")
		}
	}
	betaCheck(t, out["release_acceptance_complete"] == false && out["publication_authorized"] == false, "authority remains false")
}
func TestCriterionDeletion(t *testing.T) {
	f := betaTestFixture(t)
	rows := betaArray(f.mapping["paper_criteria"])
	f.mapping["paper_criteria"] = rows[:len(rows)-1]
	_, err := f.workorders()
	betaRefused(t, err, "missing retained criteria")
}
func TestCriterionDuplication(t *testing.T) {
	f := betaTestFixture(t)
	rows := betaArray(f.mapping["paper_criteria"])
	rows[1] = rows[0]
	_, err := f.workorders()
	betaRefused(t, err, "duplicate criterion_id")
}
func TestCloudflareOperationDeletion(t *testing.T) {
	f := betaTestFixture(t)
	rows := betaArray(f.mapping["cloudflare_operation_cells"])
	f.mapping["cloudflare_operation_cells"] = rows[:len(rows)-1]
	_, err := f.workorders()
	betaRefused(t, err, "missing retained criteria or supplemental work")
}
func TestCoordinatedProfileScopeReduction(t *testing.T) {
	f := betaTestFixture(t)
	rows := betaArray(f.plan["required_profiles"])
	f.plan["required_profiles"] = rows[:len(rows)-1]
	f.mapping["required_profiles"] = f.plan["required_profiles"]
	_, err := f.workorders()
	betaRefused(t, err, "four unique profiles")
}
func TestCoordinatedScenarioScopeReduction(t *testing.T) {
	f := betaTestFixture(t)
	rows := betaArray(f.plan["required_scenarios"])
	removed := rows[len(rows)-1]
	f.plan["required_scenarios"] = rows[:len(rows)-1]
	f.mapping["required_scenarios"] = f.plan["required_scenarios"]
	campaigns := []any{}
	for _, value := range betaArray(f.mapping["campaigns"]) {
		if betaObj(value)["scenario"] != removed {
			campaigns = append(campaigns, value)
		}
	}
	f.mapping["campaigns"] = campaigns
	_, err := f.workorders()
	betaRefused(t, err, "nine unique scenarios")
}
func TestScenarioDeletion(t *testing.T) {
	f := betaTestFixture(t)
	rows := betaArray(f.mapping["campaigns"])
	f.mapping["campaigns"] = rows[:len(rows)-1]
	_, err := f.workorders()
	betaRefused(t, err, "missing scenario work order")
}

// Exercise the actual command admission before any tool build or output mutation.
func betaAdmissionRepo(t *testing.T) (string, string) {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	source := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	base := t.TempDir()
	repo := filepath.Join(base, "repo")
	if err := os.Mkdir(repo, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range betaInputs {
		path := filepath.Join(repo, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}, {"-c", "user.name=Beta Test", "-c", "user.email=beta@example.invalid", "-c", "commit.gpgsign=false", "commit", "-qm", "fixture"}} {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("fixture git: %v: %s", err, output)
		}
	}
	return repo, filepath.Join(base, "review")
}
func TestDirtySourceRefusesBeforeOutputCreation(t *testing.T) {
	repo, work := betaAdmissionRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "uncommitted"), []byte("pending change"), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	status := betaMain([]string{"--repo", repo, "--work", work, "--go", "/unused-before-source-admission"}, &stdout, &stderr)
	betaCheck(t, status == 1 && strings.Contains(stderr.String(), "clean committed source required"), "dirty source must refuse")
	_, err := os.Lstat(work)
	betaCheck(t, errors.Is(err, os.ErrNotExist), "dirty source must not create output")
}
func TestExistingOutputRefusesWithoutChangingBytes(t *testing.T) {
	repo, work := betaAdmissionRepo(t)
	if err := os.Mkdir(work, 0700); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(work, "keep.json")
	original := []byte("{\"retain\":true}\n")
	if err := os.WriteFile(sentinel, original, 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	status := betaMain([]string{"--repo", repo, "--work", work, "--go", "/unused-after-output-refusal"}, &stdout, &stderr)
	betaCheck(t, status == 1 && strings.Contains(stderr.String(), "new absolute work directory required"), "existing output must refuse")
	actual, err := os.ReadFile(sentinel)
	if err != nil {
		t.Fatal(err)
	}
	betaCheck(t, bytes.Equal(original, actual), "existing output bytes unchanged")
	entries, err := os.ReadDir(work)
	if err != nil {
		t.Fatal(err)
	}
	betaCheck(t, len(entries) == 1, "no partial tool or packet created")
}

func TestInvalidUTF8SourceRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.json")
	if err := os.WriteFile(path, []byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'}, 0600); err != nil {
		t.Fatal(err)
	}
	_, err := betaGuard(func() betaObject { return betaRead(path) })
	betaRefused(t, err, "valid UTF-8")
}
