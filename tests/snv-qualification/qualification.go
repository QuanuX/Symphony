// Standalone, standard-library-only qualification of exact installed SNV owners.
// All retained inputs are repository fixtures or finite public observer outputs.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"
)

type object = map[string]any

type campaign struct {
	repo, prefix, work, qxctl string
	commands                  []object
}

func check(ok bool, message string) {
	if !ok {
		panic(message)
	}
}
func must(err error) {
	if err != nil {
		panic(err)
	}
}
func read(path string) []byte { b, err := os.ReadFile(path); must(err); return b }
func digest(b []byte) string  { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func decode(b []byte) object {
	var v object
	must(json.Unmarshal(b, &v))
	check(v != nil, "expected JSON object")
	return v
}
func (c *campaign) write(name string, v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	must(err)
	path := filepath.Join(c.work, "outputs", name+".json")
	must(os.WriteFile(path, append(b, '\n'), 0600))
	return path
}
func (c *campaign) call(label string, args ...string) object {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	argv := append([]string{"snv"}, args...)
	argv = append(argv, "--prefix", c.prefix, "--version", "0.1.0-dev", "--json")
	command := exec.CommandContext(ctx, c.qxctl, argv...)
	command.Dir = filepath.Join(c.work, "tmp")
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	output := filepath.Join(c.work, "outputs", label+".json")
	must(os.WriteFile(output, stdout.Bytes(), 0600))
	must(os.WriteFile(filepath.Join(c.work, "logs", label+".stderr.log"), stderr.Bytes(), 0600))
	if err != nil {
		panic(fmt.Sprintf("%s: %v; see retained stdout/stderr", label, err))
	}
	v := decode(stdout.Bytes())
	c.commands = append(c.commands, object{"label": label, "arguments": argv, "output": "outputs/" + label + ".json", "sha256": digest(stdout.Bytes()), "success": true})
	return v
}
func owner(v object, name string) {
	check(v["owner"] == name && v["owner_version"] == "0.1.0-dev", "exact installed owner identity: "+name)
}
func noAuthority(v object) {
	check(v["head"] == nil && v["head_write_stav_receipt"] == nil && v["canonical_apply_enabled"] == false,
		"private evidence or status must not claim selected head or STAV head receipt")
}
func tree(root string) map[string]string {
	result := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if errors.Is(err, os.ErrNotExist) && path == root {
			return nil
		}
		if err != nil {
			return err
		}
		rel, e := filepath.Rel(root, path)
		if e != nil {
			return e
		}
		info, e := d.Info()
		if e != nil {
			return e
		}
		if d.IsDir() {
			result[rel] = "directory:" + info.Mode().String()
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unexpected non-regular private fixture: %s", rel)
		}
		result[rel] = info.Mode().String() + ":" + digest(read(path))
		return nil
	})
	must(err)
	return result
}

// Validate only the explicitly selected field set. An unobserved field remains
// null, while observed zero is a valid value; refresh need not change its value.
func observer(v object, fields []string, observation string, operatingSystem string) error {
	if v["owner"] != "local-observer" || v["owner_version"] != "0.1.0-dev" || v["observation_id"] != observation || v["acquisition_route"] != "native_fixed_sources" {
		return errors.New("observer identity/acquisition mismatch")
	}
	if v["node_ref"] != nil || v["physical_inventory_complete"] != false || v["allocation_verified"] != false || v["node_association"] != "caller_supplied_unverified" {
		return errors.New("observer invented identity/inventory/entitlement")
	}
	subjects, ok := v["subject_ids"].([]any)
	if !ok || len(subjects) != 0 {
		return errors.New("observer assigned subject identity")
	}
	items, ok := v["fields"].([]any)
	if !ok || len(items) != len(fields) {
		return errors.New("observer field selection mismatch")
	}
	selected := map[string]bool{}
	for _, f := range fields {
		selected[f] = true
	}
	observed := 0
	for _, item := range items {
		f, ok := item.(map[string]any)
		if !ok {
			return errors.New("malformed field")
		}
		name, ok := f["field"].(string)
		if !ok || !selected[name] {
			return errors.New("duplicate or unselected field")
		}
		delete(selected, name)
		status, ok := f["status"].(string)
		if !ok {
			return errors.New("missing field status")
		}
		switch status {
		case "observed", "unavailable", "permission_denied", "unsupported_source", "too_large", "io_error", "malformed":
		default:
			return errors.New("unknown field status")
		}
		if status == "observed" {
			if f["value"] == nil {
				return errors.New("observed field lacks value")
			}
			observed++
		} else if f["value"] != nil {
			return errors.New("unobserved field became a value")
		}
	}
	switch v["status"] {
	case "complete":
		if observed != len(fields) {
			return errors.New("complete cannot conceal unseen fields")
		}
	case "partial", "changed_during_capture", "unavailable":
	default:
		return errors.New("unknown capture status")
	}
	platform, ok := v["platform"].(map[string]any)
	if !ok {
		return errors.New("missing platform")
	}
	if operatingSystem == "linux" {
		if platform["os"] != "linux" {
			return errors.New("Linux observer must identify the native platform")
		}
	} else if operatingSystem == "darwin" {
		if platform["os"] != nil || v["status"] != "unavailable" || observed != 0 {
			return errors.New("macOS must report native observer unavailable")
		}
	} else {
		return errors.New("unsupported qualification host")
	}
	capture, ok := v["capture"].(map[string]any)
	if !ok || capture["atomic_inventory"] != false {
		return errors.New("missing bounded capture")
	}
	started, a := capture["started_unix_ms"].(float64)
	finished, b := capture["finished_unix_ms"].(float64)
	if !a || !b || finished < started {
		return errors.New("invalid observation interval")
	}
	return nil
}
func (c *campaign) run() {
	input := decode(read(filepath.Join(c.repo, "modules/snv-engine/tests/fixtures/snv_inspect.json")))
	bundle := input["bundle"].(map[string]any)
	artifacts := bundle["artifacts"].([]any)
	routes := map[string][]string{"sniv": {"identity", "validate"}, "snrv": {"resources", "validate"}, "sciv": {"clusters", "validate"}, "scnv": {"names", "validate"}}
	operations := map[string]string{"sniv": "identity_validate", "snrv": "resources_validate", "sciv": "sciv_validate", "scnv": "names_validate", "snv": "snv_inspect", "local-observer": "observe"}
	for _, name := range []string{"sniv", "snrv", "sciv", "scnv", "snv", "local-observer"} {
		for _, kind := range []string{"schema", "template"} {
			result := c.call(name+"-"+kind, kind, "--owner", name, "--operation", operations[name])
			check(result["protocol"] == "symphony.qxctl.snv-"+kind+".v1" && result["owner"] == name && result["operation"] == operations[name] && result[kind] != nil, "exact installed resource")
		}
	}
	check(len(artifacts) == 4, "four supplied original artifacts required")
	seen := map[string]bool{}
	for _, artifact := range artifacts {
		a := artifact.(map[string]any)
		name := a["owner"].(string)
		check(!seen[name] && routes[name] != nil, "duplicate/unknown owner")
		seen[name] = true
		check(a["operation"] == operations[name], "original operation binding")
		path := filepath.Join(c.work, "outputs", name+"-input.json")
		must(os.WriteFile(path, []byte(a["source_utf8"].(string)), 0600))
		args := append(append([]string{}, routes[name]...), "--input", path)
		owner(c.call(name+"-replay", args...), name)
	}
	supplied := c.write("supplied-input", input)
	owner(c.call("snv-replay", "inspect", "--input", supplied), "symphony-snv")
	resolve := filepath.Join(c.repo, "modules/scnv-engine/tests/fixtures/names_resolve.json")
	owner(c.call("names-resolve", "names", "resolve", "--input", resolve), "scnv")
	// Evidence preparation/commit retains only these supplied fixtures in a fresh
	// private root. It neither selects a view nor invokes protected state apply.
	root := filepath.Join(c.work, "tmp", "evidence")
	absent := filepath.Join(c.work, "tmp", "absent-view")
	viewArgs := []string{"--state-root", absent, "--tops-id", "00000000-0000-4000-8000-000000000019", "--view-id", "qualification-view"}
	state := c.call("absent-state-status", append([]string{"state", "status"}, viewArgs...)...)
	noAuthority(state)
	_, err := os.Lstat(absent)
	check(errors.Is(err, os.ErrNotExist), "read-only status created absent root")
	evidenceInput := c.write("evidence-input", object{"protocol": "symphony.snv.evidence-plan-input.v1", "operation_id": "qualification-evidence", "mode": "bundle", "bundle": bundle, "export_records": nil})
	prepared := c.call("evidence-prepare", "evidence", "prepare", "--input", evidenceInput, "--state-root", root)
	noAuthority(prepared)
	check(prepared["attempt"].(map[string]any)["status"] == "prepared", "retention prepared")
	args := []string{"evidence", "commit", "--state-root", root, "--operation-id", "qualification-evidence"}
	committed := c.call("evidence-commit", args...)
	noAuthority(committed)
	check(committed["attempt"].(map[string]any)["status"] == "committed", "retention committed")
	retained := tree(root)
	retried := c.call("evidence-repeat-commit", args...)
	check(reflect.DeepEqual(committed, retried) && reflect.DeepEqual(retained, tree(root)), "repeat retention changed evidence")
	status := c.call("evidence-status", "evidence", "status", "--state-root", root, "--operation-id", "qualification-evidence")
	noAuthority(status)
	check(status["journal_digest"] == committed["journal_digest"] && reflect.DeepEqual(retained, tree(root)), "read-only retention status changed evidence")
	// Pure planning resolves the retained bundle, and must preserve every private
	// state byte. No selected-view authority is constructed by this harness.
	planInput := c.write("state-plan-input", object{"protocol": "symphony.snv.state-plan-input.v1", "view": object{"tops_id": "00000000-0000-4000-8000-000000000019", "view_id": "qualification-view"}, "operation_id": "qualification-select", "expected_state_digest": nil, "prior_head": nil, "change_kind": "select", "bundle": bundle, "reason": "supplied-evidence qualification", "migration": nil})
	proposal := c.call("state-plan", "state", "plan", "--input", planInput, "--state-root", root, "--tops-id", "00000000-0000-4000-8000-000000000019", "--view-id", "qualification-view")
	check(proposal["protocol"] == "symphony.qxctl.snv-state-proposal.v1" && proposal["plan"] != nil && reflect.DeepEqual(retained, tree(root)), "pure plan changed retained evidence")
	allFields := []string{"cpu_present", "cpu_online", "memory_total", "memory_available"}
	request := object{"protocol": "symphony.snv.local-observe-input.v1", "node_ref": nil, "profile": "linux-proc-sysfs-v1", "fields": allFields}
	native := []object{}
	for _, id := range []string{"native-before", "native-refresh"} {
		request["observation_id"] = id
		path := c.write(id+"-input", request)
		result := c.call(id, "observe", "--input", path)
		must(observer(result, allFields, id, runtime.GOOS))
		check(runtime.GOOS != "linux" || result["status"] != "unavailable", "Linux qualification requires actual observed proc/sysfs evidence; unavailable is retained but does not qualify Linux")
		native = append(native, result)
	}
	before := native[0]["capture"].(map[string]any)
	after := native[1]["capture"].(map[string]any)
	check(after["started_unix_ms"].(float64) >= before["finished_unix_ms"].(float64), "refresh reused capture interval")
	check(decode(read(filepath.Join(c.work, "outputs", "native-before.json")))["observation_id"] == "native-before", "refresh overwrote prior capture")
	request["observation_id"] = "native-selected-field"
	request["fields"] = []string{"memory_available"}
	result := c.call("native-selected-field", "observe", "--input", c.write("native-selected-field-input", request))
	must(observer(result, []string{"memory_available"}, "native-selected-field", runtime.GOOS))
	for _, command := range c.commands {
		check(digest(read(filepath.Join(c.work, command["output"].(string)))) == command["sha256"], "later call changed a retained earlier output")
	}
	receipts := map[string]string{}
	for _, id := range []string{"knowledge-vector-engine-cpp", "snv-common-cpp", "sniv-engine", "snrv-engine", "sciv-engine", "scnv-engine", "snv-engine", "snv-local-observer"} {
		version := "0.1.0-dev"
		if id == "knowledge-vector-engine-cpp" {
			version = "0.2.0-dev"
		}
		path := filepath.Join("share", "symphony", "receipts", id, version, "install-receipt.json")
		receipts[path] = digest(read(filepath.Join(c.prefix, path)))
	}
	sdk := decode(read(filepath.Join(c.work, "logs", "sdk-replay.log")))
	check(sdk["semantic_engines_replayed"] == float64(5), "five installed SDK replay proof missing")
	git := exec.Command("git", "-C", c.repo, "rev-parse", "HEAD")
	head, err := git.Output()
	must(err)
	sourceFiles := map[string]string{}
	for _, path := range []string{
		"tests/snv-qualification/run.sh", "tests/snv-qualification/qualify.cmake", "tests/snv-qualification/qualification.go", "tests/snv-qualification/qualification_test.go",
		"tests/snv-qualification/sdk-consumer/CMakeLists.txt", "tests/snv-qualification/sdk-consumer/consumer.cpp",
		"tests/mvp-dependencies/common.cmake", "tests/mvp-dependencies/prove-imports.cmake", "tests/mvp-dependencies/verify-receipt.cmake", "cmake/mvp-dependencies.json",
		"tests/release-qualification/go-runner.go", "tests/release-qualification/manifest.json",
		"modules/snv-engine/tests/fixtures/snv_inspect.json", "modules/scnv-engine/tests/fixtures/names_resolve.json",
	} {
		sourceFiles[path] = digest(read(filepath.Join(c.repo, path)))
	}
	sourceTree, err := exec.Command("git", "-C", c.repo, "rev-parse", "HEAD^{tree}").Output()
	must(err)
	dirty, err := exec.Command("git", "-C", c.repo, "status", "--porcelain").Output()
	must(err)
	pending := object{"schema": "symphony-test.snv-qualification.v1", "complete": false, "source_head": strings.TrimSpace(string(head)), "source_tree": strings.TrimSpace(string(sourceTree)), "source_dirty": len(dirty) != 0, "source_file_sha256": sourceFiles, "host": object{"goos": runtime.GOOS, "goarch": runtime.GOARCH, "details": "logs/host-info.log", "details_sha256": digest(read(filepath.Join(c.work, "logs", "host-info.log")))}, "selected_prefix": c.prefix, "qxctl": object{"path": c.qxctl, "sha256": digest(read(c.qxctl))}, "receipt_sha256": receipts, "commands": c.commands, "sdk_semantic_owners": 5, "observer": object{"native_status_before": native[0]["status"], "native_status_after": native[1]["status"], "native_acquisition": "native_fixed_sources", "synthetic_partial_unseen_refresh": "logs/sdk-replay.log", "identity_or_provisioning_authority": false}, "boundaries": []string{"receipts verify installed bytes, not publisher authenticity", "native observations cover this process kernel/proc/sysfs exposure, not bare metal or cgroup entitlement", "macOS unavailable output is not Linux runtime coverage", "SDK supplied reader is synthetic, separate from native host", "private fixture retention and pure planning do not select a view or activate SSIAG/STAV", "completion requires installed Go gates and post-use receipt verification"}}
	b, err := json.MarshalIndent(pending, "", "  ")
	must(err)
	must(os.WriteFile(filepath.Join(c.work, "evidence.pending.json"), append(b, '\n'), 0600))
	fmt.Printf("PASS %d actual qxctl commands; five installed SDK owners; separate native and injected observer evidence\n", len(c.commands))
}
func main() {
	c := campaign{}
	flag.StringVar(&c.repo, "repo", "", "repository")
	flag.StringVar(&c.prefix, "prefix", "", "installed prefix")
	flag.StringVar(&c.work, "work", "", "private workspace")
	flag.StringVar(&c.qxctl, "qxctl", "", "exact qxctl")
	flag.Parse()
	defer func() {
		if p := recover(); p != nil {
			fmt.Fprintln(os.Stderr, p)
			os.Exit(1)
		}
	}()
	for _, path := range []string{c.repo, c.prefix, c.work, c.qxctl} {
		check(filepath.IsAbs(path), "absolute paths required")
	}
	c.run()
}
