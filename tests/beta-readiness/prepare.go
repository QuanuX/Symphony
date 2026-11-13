// Source-bound, explicitly incomplete W12 preparation. Never deploy or publish.
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
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"unicode/utf8"
)

type betaObject = map[string]any

type betaRefusal struct{ err error }

var betaInputs = []string{
	"tests/public-beta/package-plan.json", "tests/public-beta/acceptance-mapping.json",
	"tests/public-beta/capability-groups.json", "tests/public-beta/owners.json",
	"tests/public-beta/execution/authorization.json", "cmake/mvp-dependencies.json",
	"cmake/mvp-release-identities.json", "cmake/mvp-platform-profiles.json",
	"tests/launch-acceptance/plan.json",
}

func betaNeed(ok bool, message string) {
	if !ok {
		panic(betaRefusal{errors.New(message)})
	}
}
func betaMust[T any](value T, err error) T {
	if err != nil {
		panic(betaRefusal{err})
	}
	return value
}
func betaGuard(fn func() betaObject) (value betaObject, err error) {
	defer func() {
		if p := recover(); p != nil {
			if r, ok := p.(betaRefusal); ok {
				err = r.err
			} else {
				panic(p)
			}
		}
	}()
	return fn(), nil
}
func betaObj(value any) betaObject {
	v, ok := value.(map[string]any)
	betaNeed(ok, "JSON object required")
	return v
}
func betaArray(value any) []any {
	v, ok := value.([]any)
	betaNeed(ok, "JSON array required")
	return v
}
func betaString(value any) string {
	v, ok := value.(string)
	betaNeed(ok, "JSON string required")
	return v
}
func betaField(row betaObject, key string) any {
	v, ok := row[key]
	betaNeed(ok, "missing source field: "+key)
	return v
}
func betaStrings(value any) []string {
	out := []string{}
	for _, item := range betaArray(value) {
		out = append(out, betaString(item))
	}
	return out
}
func betaSet(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		out[value] = true
	}
	return out
}
func betaPick(row betaObject, keys ...string) betaObject {
	out := betaObject{}
	for _, key := range keys {
		out[key] = betaField(row, key)
	}
	return out
}
func betaUnique(value any, key string) map[string]betaObject {
	out := map[string]betaObject{}
	for _, item := range betaArray(value) {
		row := betaObj(item)
		id := betaString(betaField(row, key))
		_, exists := out[id]
		betaNeed(!exists, "duplicate "+key)
		out[id] = row
	}
	return out
}
func betaEntry(rows map[string]betaObject, key string) betaObject {
	row, ok := rows[key]
	betaNeed(ok, "missing source entry: "+key)
	return row
}
func betaFlags(row betaObject) betaObject {
	row["release_acceptance_complete"] = false
	row["publication_authorized"] = false
	return row
}

func betaConstruction(packages, dependency, identities, plan betaObject) (betaObject, error) {
	return betaGuard(func() betaObject {
		recipes := betaUnique(betaField(packages, "package_recipes"), "id")
		native := betaUnique(betaField(dependency, "native_components"), "path")
		goOwners := betaUnique(betaField(identities, "go_artifacts"), "path")
		templates := betaObj(betaField(packages, "build_templates"))
		profiles := betaObject{}
		for _, profile := range betaStrings(betaField(plan, "required_profiles")) {
			group := betaEntry(recipes, "PKG-CORE-"+profile)
			selected := betaStrings(betaField(group, "content_recipe_ids"))
			selectedSet := betaSet(selected)
			betaNeed(len(selectedSet) == len(selected), "duplicate selected recipe")
			selectedNative := map[string]betaObject{}
			selectedGo := []any{}
			for _, recipeID := range selected {
				r := betaEntry(recipes, recipeID)
				betaNeed(betaSet(betaStrings(betaField(r, "candidate_profiles")))[profile], "recipe not declared on profile")
				roots := betaStrings(betaField(r, "source_roots"))
				switch betaString(betaField(r, "kind")) {
				case "native_receipt_unit":
					betaNeed(len(roots) == 1, "native recipe root mismatch")
					root := roots[0]
					n := betaEntry(native, root)
					betaNeed(reflect.DeepEqual(betaField(r, "proposed_package_version"), betaField(n, "package_release_version")) &&
						reflect.DeepEqual(betaField(r, "project_version"), betaField(n, "project_version")) &&
						reflect.DeepEqual(betaField(r, "installed_options"), betaField(n, "installed_options")) &&
						reflect.DeepEqual(betaField(r, "build_targets"), betaField(n, "artifact_targets")) &&
						betaField(n, "installable") == true, "recipe drift from current native owner: "+recipeID)
					for _, dep := range betaStrings(betaField(n, "dependency_paths_in_build_order")) {
						betaNeed(selectedSet["PKG-NATIVE-"+betaString(betaField(betaEntry(native, dep), "component_id"))], "missing native dependency closure")
					}
					selectedNative[root] = betaObject{"recipe_id": recipeID, "component_id": betaField(n, "component_id"),
						"source_root": root, "version": betaField(n, "package_release_version"),
						"installed_options": betaField(n, "installed_options"), "build_targets": betaField(n, "artifact_targets"),
						"dependency_source_roots":  betaField(n, "dependency_paths_in_build_order"),
						"external_dependency_refs": betaField(r, "external_dependency_refs"),
						"recipe_template":          betaField(templates, betaString(betaField(r, "build_template")))}
				case "go_command":
					betaNeed(len(roots) == 1, "Go recipe root mismatch")
					g := betaEntry(goOwners, roots[0])
					betaNeed(reflect.DeepEqual(betaField(r, "proposed_package_version"), betaField(g, "package_version")), "recipe drift from current Go owner")
					for _, key := range []string{"binary_name", "main_package", "version_symbol", "source_runtime_version"} {
						betaNeed(reflect.DeepEqual(betaField(r, key), betaField(g, key)), "recipe drift from current Go owner")
					}
					selectedGo = append(selectedGo, betaPick(r, "id", "source_roots", "binary_name", "main_package", "proposed_package_version", "version_symbol", "build_template", "installation_contract"))
				default:
					betaNeed(false, "unsupported core recipe kind")
				}
			}
			betaNeed(len(selectedNative) == 51 && len(selectedGo) == 6, "approved 51 native / 6 Go engineering selection changed")
			ordered := []any{}
			seen := map[string]bool{}
			for _, path := range betaStrings(betaField(dependency, "native_build_order")) {
				row, selected := selectedNative[path]
				if !selected {
					continue
				}
				betaNeed(!seen[path], "build order omits or duplicates selected unit")
				for _, dep := range betaStrings(betaField(betaEntry(native, path), "dependency_paths_in_build_order")) {
					betaNeed(seen[dep], "non-topological native build order")
				}
				seen[path] = true
				ordered = append(ordered, row)
			}
			betaNeed(len(seen) == len(selectedNative), "build order omits or duplicates selected unit")
			profiles[profile] = betaObject{"native_build_order": ordered, "go_commands": selectedGo,
				"optional_recipe_ids": betaField(group, "separately_selectable_recipe_ids"), "final_stage": nil,
				"finalizer_and_loader_policy": nil, "toolchain_and_dependency_inputs": nil,
				"payload_review": nil, "candidate_identity": nil, "retrieved_anaconda_identity": nil}
		}
		return betaFlags(betaObject{"protocol": "symphony.public-beta.construction-worklist.v1", "scope": "source_plan_only",
			"profiles": profiles, "go_build_templates": betaPick(templates, "GO-IDENTITIES", "QXCTL-REPRODUCIBLE"),
			"external_dependency_requirements": betaField(dependency, "external_dependencies"),
			"supplemental_cloudflare":          betaEntry(recipes, "PKG-CF-INTEL"), "source_material": betaEntry(recipes, "PKG-SOURCE"),
			"construction_note": "No final stage or build is implied. Select exact inputs, finalize before receipt sealing, review actual notices/runtime/source closure, then sign with independent policy."})
	})
}

func betaWorkorders(mapping, plan betaObject) (betaObject, error) {
	return betaGuard(func() betaObject {
		profiles := betaStrings(betaField(plan, "required_profiles"))
		scenarios := betaStrings(betaField(plan, "required_scenarios"))
		betaNeed(len(profiles) == 4 && len(betaSet(profiles)) == 4 && len(scenarios) == 9 && len(betaSet(scenarios)) == 9,
			"four unique profiles and nine unique scenarios required")
		betaNeed(betaField(mapping, "core_cell_count") == json.Number("36") && betaField(mapping, "paper_criterion_count") == json.Number("58") &&
			reflect.DeepEqual(betaSet(betaStrings(betaField(mapping, "required_profiles"))), betaSet(profiles)) &&
			reflect.DeepEqual(betaSet(betaStrings(betaField(mapping, "required_scenarios"))), betaSet(scenarios)), "acceptance scope drift")
		campaigns := betaUnique(betaField(mapping, "campaigns"), "scenario")
		betaNeed(len(campaigns) == len(scenarios), "missing scenario work order")
		for _, scenario := range scenarios {
			betaEntry(campaigns, scenario)
		}
		criteriaRows := betaArray(betaField(mapping, "paper_criteria"))
		criteria := betaUnique(criteriaRows, "criterion_id")
		extra := append(append([]any{}, betaArray(betaField(mapping, "cloudflare_operation_cells"))...), betaArray(betaField(mapping, "additional_campaigns"))...)
		betaUnique(extra, "id")
		betaNeed(len(criteria) == 58 && len(extra) == 31, "missing retained criteria or supplemental work")
		pending := func(row betaObject) betaObject {
			for _, field := range []string{"final_artifact_bindings", "operator", "independent_reviewer", "observed_result"} {
				row[field] = nil
			}
			row["evidence"] = []any{}
			row["acceptance_status"] = "pending"
			return row
		}
		core, paper, supplemental := []any{}, []any{}, []any{}
		for _, profile := range profiles {
			for _, scenario := range scenarios {
				core = append(core, pending(betaObject{"id": profile + "::" + scenario, "profile": profile, "source_campaign": betaEntry(campaigns, scenario)}))
			}
		}
		for _, item := range criteriaRows {
			row := betaObj(item)
			paper = append(paper, pending(betaObject{"id": betaField(row, "criterion_id"), "source_criterion": row}))
		}
		for _, item := range extra {
			row := betaObj(item)
			supplemental = append(supplemental, pending(betaObject{"id": betaField(row, "id"), "source_requirement": row}))
		}
		return betaFlags(betaObject{"protocol": "symphony.public-beta.campaign-workorders.v1", "core": core, "paper": paper, "supplemental": supplemental})
	})
}

func betaSHA(path string) string {
	file := betaMust(os.Open(path))
	defer file.Close()
	hash := sha256.New()
	betaMust(io.Copy(hash, file))
	return hex.EncodeToString(hash.Sum(nil))
}
func betaRead(path string) betaObject {
	info := betaMust(os.Lstat(path))
	betaNeed(info.Mode().IsRegular() && info.Size() <= 16*1024*1024, "bounded regular source input required: "+path)
	data := betaMust(os.ReadFile(path))
	betaNeed(utf8.Valid(data), "source JSON must be valid UTF-8")
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var row betaObject
	betaMust(struct{}{}, decoder.Decode(&row))
	betaNeed(row != nil, "JSON object required")
	var trailing any
	betaNeed(decoder.Decode(&trailing) == io.EOF, "trailing JSON input")
	return row
}
func betaOutput(command string, args []string, dir string, env []string) string {
	cmd := exec.Command(command, args...)
	cmd.Dir, cmd.Env = dir, env
	out := betaMust(cmd.Output())
	return strings.TrimSpace(string(out))
}
func betaSource(repo string) betaObject {
	git := func(args ...string) string {
		return betaOutput("git", append([]string{"-C", repo}, args...), repo, nil)
	}
	betaNeed(git("status", "--porcelain", "--untracked-files=all") == "", "clean committed source required; retain changes before preparing review")
	return betaObject{"commit": git("rev-parse", "HEAD"), "tree": git("rev-parse", "HEAD^{tree}")}
}
func betaEnvironment(overrides map[string]string) []string {
	values := map[string]string{}
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[key] = value
		}
	}
	for key, value := range overrides {
		values[key] = value
	}
	out := []string{}
	for key, value := range values {
		out = append(out, key+"="+value)
	}
	return out
}
func betaWriteNew(path string, value any) {
	file := betaMust(os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600))
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	betaMust(struct{}{}, encoder.Encode(value))
	betaMust(struct{}{}, file.Close())
}
func betaInside(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

type betaArguments struct{ repo, work, goDriver string }

func betaPrepare(args betaArguments) (betaObject, error) {
	return betaGuard(func() betaObject {
		repo := betaMust(filepath.EvalSymlinks(betaMust(filepath.Abs(args.repo))))
		initial := betaSource(repo)
		values, hashes := map[string]betaObject{}, map[string]string{}
		for _, path := range betaInputs {
			values[path], hashes[path] = betaRead(filepath.Join(repo, path)), betaSHA(filepath.Join(repo, path))
		}
		auth := values[betaInputs[4]]
		betaNeed(betaField(auth, "state") == "implementation_authorized" && betaField(auth, "feature_freeze_in_force") == true &&
			betaField(auth, "new_product_features_authorized") == false && betaField(auth, "publication_authorized") == false,
			"expected frozen engineering authorization")
		plan := values[betaInputs[len(betaInputs)-1]]
		builds := betaMust(betaConstruction(values[betaInputs[0]], values[betaInputs[5]], values[betaInputs[6]], plan))
		campaigns := betaMust(betaWorkorders(values[betaInputs[1]], plan))
		_, existsErr := os.Lstat(args.work)
		betaNeed(filepath.IsAbs(args.work) && errors.Is(existsErr, os.ErrNotExist), "new absolute work directory required")
		parent := betaMust(filepath.EvalSymlinks(filepath.Dir(args.work)))
		work := filepath.Join(parent, filepath.Base(args.work))
		betaNeed(!betaInside(repo, work), "review directory must be outside source")
		goDriver := betaMust(filepath.Abs(args.goDriver))
		overrides := map[string]string{"GOWORK": "off", "GOENV": "off", "GOTOOLCHAIN": "local", "GOPROXY": "off", "GOSUMDB": "off"}
		env := betaEnvironment(overrides)
		betaNeed(strings.HasPrefix(betaOutput(goDriver, []string{"version"}, repo, env), "go version go1.26.5 "), "exact Go 1.26.5 required")
		betaMust(struct{}{}, os.Mkdir(work, 0700))
		for _, name := range []string{"logs", "tools"} {
			betaMust(struct{}{}, os.Mkdir(filepath.Join(work, name), 0700))
		}
		overrides["GOCACHE"], overrides["GOMODCACHE"] = filepath.Join(work, "go-cache"), filepath.Join(work, "go-modules")
		env = betaEnvironment(overrides)
		write := func(name string, value any) { betaWriteNew(filepath.Join(work, name), value) }
		run := func(label string, command []string, expected int) {
			log := betaMust(os.OpenFile(filepath.Join(work, "logs", label+".log"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600))
			defer log.Close()
			cmd := exec.Command(command[0], command[1:]...)
			cmd.Dir, cmd.Env, cmd.Stdout, cmd.Stderr = repo, env, log, log
			err := cmd.Run()
			status := 0
			if err != nil {
				var exit *exec.ExitError
				betaNeed(errors.As(err, &exit), label+": process unavailable")
				status = exit.ExitCode()
			}
			betaNeed(status == expected, fmt.Sprintf("%s: unexpected exit %d", label, status))
			betaMust(struct{}{}, log.Close())
		}
		for _, check := range []string{"mvp-dependencies", "release-qualification"} {
			run(check, []string{"cmake", "-P", filepath.Join(repo, "tests", check, "verify.cmake")}, 0)
		}
		prefix := filepath.Join(repo, "tests/launch-acceptance")
		for _, build := range []struct {
			name  string
			files []string
		}{{"acceptance-gates", []string{"acceptance-gates.go", "acceptance-gates-main.go"}},
			{"delivery-acceptance", []string{"delivery-set.go", "acceptance-gates.go", "delivery-acceptance.go"}}} {
			command := []string{goDriver, "build", "-trimpath", "-o", filepath.Join(work, "tools", build.name)}
			for _, file := range build.files {
				command = append(command, filepath.Join(prefix, file))
			}
			run("build-"+build.name, command, 0)
		}
		core, companion := filepath.Join(work, "tools/acceptance-gates"), filepath.Join(work, "tools/delivery-acceptance")
		for _, item := range []struct {
			name  string
			value any
		}{{"core-plan.json", plan}, {"acceptance-mapping.json", values[betaInputs[1]]}, {"construction.json", builds}, {"workorders.json", campaigns}} {
			write(item.name, item.value)
		}
		path := func(name string) string { return filepath.Join(work, name) }
		planPath, mappingPath := path("core-plan.json"), path("acceptance-mapping.json")
		run("init-core", []string{core, "-mode", "init", "-plan", planPath, "-packet", path("core-packet.json")}, 0)
		run("init-requirements", []string{companion, "-mode", "init-requirements", "-plan", planPath,
			"-mapping", mappingPath, "-requirements", path("requirements.json")}, 0)
		base := []string{companion, "-plan", planPath, "-mapping", mappingPath, "-requirements", path("requirements.json"),
			"-core-packet", path("core-packet.json"), "-packet", path("companion-packet.json")}
		run("init-companion", append(append([]string{}, base...), "-mode", "init"), 0)
		targets := betaObject{}
		for _, profile := range betaStrings(betaField(plan, "required_profiles")) {
			targets[profile] = nil
		}
		write("targets.json", betaObject{"protocol": "symphony.release.delivery-targets.v1", "profiles": targets})
		run("check-core", []string{core, "-mode", "check", "-plan", planPath, "-packet", path("core-packet.json"), "-report", path("core-report.json")}, 2)
		run("check-companion", append(append([]string{}, base...), "-mode", "check", "-targets", path("targets.json"), "-report", path("companion-report.json")), 2)
		betaNeed(reflect.DeepEqual(betaSource(repo), initial), "source changed during preparation")
		for _, name := range betaInputs {
			betaNeed(hashes[name] == betaSHA(filepath.Join(repo, name)), "source changed during preparation")
		}
		files := map[string]string{}
		for _, directory := range []string{work, path("logs"), path("tools")} {
			for _, entry := range betaMust(os.ReadDir(directory)) {
				file := filepath.Join(directory, entry.Name())
				if betaMust(entry.Info()).Mode().IsRegular() {
					files[filepath.ToSlash(betaMust(filepath.Rel(work, file)))] = betaSHA(file)
				}
			}
		}
		write("review.json", betaFlags(betaObject{"protocol": "symphony.public-beta.w12-review.v1", "source": initial,
			"state": "incomplete_no_go_for_public_beta", "human_release_decision": nil,
			"distribution_channel": betaField(auth, "distribution_channel"), "end_W12_inputs": betaField(auth, "deferred_to_end_of_W12"),
			"scope":                "local engineering and qualification preparation; no final candidate or operational acceptance",
			"source_inputs_sha256": hashes, "go_driver_sha256": betaSHA(goDriver), "files_sha256": files,
			"counts": betaObject{"profiles": 4, "native_units_per_core": 51, "go_commands_per_core": 6,
				"core_cells": len(betaArray(campaigns["core"])), "paper_criteria": len(betaArray(campaigns["paper"])), "supplemental": len(betaArray(campaigns["supplemental"]))}}))
		return betaFlags(betaObject{"review": path("review.json"), "state": "incomplete_no_go_for_public_beta"})
	})
}

func betaMain(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("beta-readiness", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repo := flags.String("repo", "", "clean committed Symphony checkout")
	work := flags.String("work", "", "new absolute private review directory outside source")
	goDriver := flags.String("go", "", "exact Go 1.26.5 executable")
	if err := flags.Parse(args); err != nil {
		return 1
	}
	if flags.NArg() != 0 || *repo == "" || *work == "" || *goDriver == "" {
		fmt.Fprintln(stderr, "beta review refused: --repo, --work and --go are required")
		return 1
	}
	result, err := betaPrepare(betaArguments{*repo, *work, *goDriver})
	if err != nil {
		fmt.Fprintln(stderr, "beta review refused: "+err.Error())
		return 1
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(stderr, "beta review refused: "+err.Error())
		return 1
	}
	return 2
}
func main() { os.Exit(betaMain(os.Args[1:], os.Stdout, os.Stderr)) }
