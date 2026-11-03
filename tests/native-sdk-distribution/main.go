// Independent distribution qualification. All mutations target a private copy
// of completed installed artifacts; configured uninstallers retain ownership.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"syscall"
	"time"
)

type object = map[string]any
type dependency struct{ Path, Scope string }
type exported struct {
	Package string
	Targets []string
}
type consumer struct{ Path string }
type component struct {
	Path         string
	ID           string `json:"component_id"`
	Version      string `json:"package_release_version"`
	Group        string `json:"qualification_group"`
	Installable  bool
	Dependencies []dependency `json:"first_party_sdk_dependencies"`
	Exports      []exported   `json:"exported_packages"`
	Consumers    []consumer   `json:"installed_sdk_consumer_checks"`
}
type projection struct {
	Components []component `json:"native_components"`
	Order      []string    `json:"native_build_order"`
}
type coverage struct {
	Path, Coverage string
	Exports        []string
}
type sourceConsumer struct{ Path, Coverage, Group string }
type manifest struct {
	Separate   []struct{ Path, Workflow, Reason string } `json:"separate_campaigns"`
	Sources    []sourceConsumer                          `json:"source_consumers"`
	SDKs       []coverage                                `json:"sdk_producers"`
	Exclusions []struct{ Path, Reason string }           `json:"explicit_external_exclusions"`
}
type owned struct {
	Path, Digest, Kind string
	Size               int64
}
type receipt struct {
	Files  []owned
	Digest string `json:"receipt_digest"`
}
type fileState struct {
	SHA256 string `json:"sha256"`
	Mode   uint32 `json:"mode"`
}
type campaign struct {
	repo, group, installed, work, prior string
	checks                              []string
	evidence                            object
	components                          map[string]component
	raw                                 map[string]json.RawMessage
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func require(ok bool, message string) {
	if !ok {
		panic(message)
	}
}
func read(p string) []byte  { b, e := os.ReadFile(p); must(e); return b }
func hash(b []byte) string  { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func encode(v any) []byte   { b, e := json.MarshalIndent(v, "", "  "); must(e); return append(b, '\n') }
func write(p string, v any) { must(os.WriteFile(p, encode(v), 0600)) }
func decode(b []byte) object {
	var v object
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	must(d.Decode(&v))
	require(v != nil, "object required")
	return v
}
func (c *campaign) check(ok bool, label string) {
	require(ok, label)
	c.checks = append(c.checks, label)
}

// The supervisor retains the process-group identity until the caller has
// drained output or reached its bound. The actual command may exit early;
// cleanup still occurs before the supervisor is reaped, so its PID cannot be
// reused by an unrelated process group.
func init() {
	if len(os.Args) < 3 || os.Args[1] != "--sdk-distribution-supervisor" {
		return
	}
	signal.Notify(make(chan os.Signal, 1), syscall.SIGTERM)
	status := os.NewFile(3, "command-status")
	syscall.CloseOnExec(3) // Only this supervisor may report command completion.
	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	result := commandStatus{ExitCode: -1}
	if err := cmd.Run(); err != nil {
		result.Error = err.Error()
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			result.ExitCode = exit.ExitCode()
		}
	} else {
		result.ExitCode = 0
	}
	_ = os.Stdout.Close()
	_ = os.Stderr.Close()
	_ = json.NewEncoder(status).Encode(result)
	_ = status.Close()
	for {
		time.Sleep(time.Hour)
	}
}

type commandStatus struct {
	ExitCode int
	Error    string
}
type commandExitError struct {
	Code   int
	Detail string
}

func (e *commandExitError) Error() string {
	return fmt.Sprintf("command exit %d: %s", e.Code, e.Detail)
}
func expectedCommandResult(success bool, err error) bool {
	if success {
		return err == nil
	}
	var exit *commandExitError
	return errors.As(err, &exit) && exit.Code > 0
}

type commandOutput struct {
	Bytes []byte
	Err   error
}

func boundedOutput(ctx context.Context, dir string, env []string, args ...string) ([]byte, error) {
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	outputReader, outputWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer outputReader.Close()
	defer outputWriter.Close()
	statusReader, statusWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer statusReader.Close()
	defer statusWriter.Close()
	cmd := exec.Command(executable, append([]string{"--sdk-distribution-supervisor"}, args...)...)
	cmd.Dir, cmd.Env = dir, env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout, cmd.Stderr = outputWriter, outputWriter
	cmd.ExtraFiles = []*os.File{statusWriter}
	if err = cmd.Start(); err != nil {
		return nil, err
	}
	_ = outputWriter.Close()
	_ = statusWriter.Close()
	output := make(chan commandOutput, 1)
	go func() { b, e := io.ReadAll(outputReader); output <- commandOutput{b, e} }()
	status := make(chan commandStatus, 1)
	go func() {
		result := commandStatus{ExitCode: -1}
		if e := json.NewDecoder(io.LimitReader(statusReader, 8192)).Decode(&result); e != nil {
			result.ExitCode = -1
			result.Error = "supervisor status: " + e.Error()
		}
		status <- result
	}()
	var result commandOutput
	var reported commandStatus
	var haveOutput, haveStatus bool
	var drain <-chan time.Time
	var timer *time.Timer
	for err == nil && !(haveOutput && haveStatus) {
		select {
		case result = <-output:
			haveOutput = true
			output = nil
		case reported = <-status:
			haveStatus = true
			status = nil
			timer = time.NewTimer(time.Second)
			drain = timer.C
		case <-drain:
			err = exec.ErrWaitDelay
		case <-ctx.Done():
			err = ctx.Err()
		}
	}
	if timer != nil {
		timer.Stop()
	}
	if err != nil {
		// Give qxctl its bounded opportunity to reap its own native process group.
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		time.Sleep(250 * time.Millisecond)
	}
	// No Wait has run yet: even an unexpectedly exited supervisor is unreaped,
	// so the process-group identity cannot have been reused here.
	killErr := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	waitErr := cmd.Wait()
	_ = statusReader.Close()
	if !haveOutput {
		// A child which deliberately escapes this group cannot keep our pipe wait
		// open indefinitely; ordinary qxctl cleanup is exercised by its owner gate.
		_ = outputReader.Close()
		result = <-output
	}
	if err != nil {
		return result.Bytes, err
	}
	if killErr != nil && killErr != syscall.ESRCH {
		return result.Bytes, killErr
	}
	if waitErr == nil {
		return result.Bytes, fmt.Errorf("supervisor exited without owned cleanup")
	}
	if result.Err != nil {
		return result.Bytes, result.Err
	}
	if reported.ExitCode < 0 {
		return result.Bytes, fmt.Errorf("runner: %s", reported.Error)
	}
	if reported.ExitCode > 0 {
		return result.Bytes, &commandExitError{reported.ExitCode, reported.Error}
	}
	if reported.Error != "" {
		return result.Bytes, fmt.Errorf("runner status: %s", reported.Error)
	}
	return result.Bytes, nil
}
func (c *campaign) run(label, dir string, success bool, args ...string) []byte {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	b, e := boundedOutput(ctx, dir, nil, args...)
	must(os.WriteFile(filepath.Join(c.work, "logs", label+".log"), b, 0600))
	require(ctx.Err() == nil, label+": execution bound exceeded")
	require(expectedCommandResult(success, e), fmt.Sprintf("%s: unexpected command result %v (see retained log)", label, e))
	return b
}
func regular(p string) os.FileInfo {
	s, e := os.Lstat(p)
	must(e)
	require(s.Mode().IsRegular(), "expected regular file: "+p)
	return s
}
func copyFile(src, dst string) {
	s := regular(src)
	must(os.MkdirAll(filepath.Dir(dst), 0700))
	f, e := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, s.Mode().Perm())
	must(e)
	in, e := os.Open(src)
	must(e)
	_, e = io.Copy(f, in)
	must(e)
	must(in.Close())
	must(f.Close())
	must(os.Chmod(dst, s.Mode().Perm()))
}
func copyTree(src, dst string) {
	must(filepath.WalkDir(src, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		r, e := filepath.Rel(src, p)
		if e != nil {
			return e
		}
		target := filepath.Join(dst, r)
		if d.IsDir() {
			return os.MkdirAll(target, 0700)
		}
		copyFile(p, target)
		return nil
	}))
}
func snapshot(root string) map[string]fileState {
	out := map[string]fileState{}
	must(filepath.WalkDir(root, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		s := regular(p)
		r, e := filepath.Rel(root, p)
		if e != nil {
			return e
		}
		out[filepath.ToSlash(r)] = fileState{hash(read(p)), uint32(s.Mode().Perm())}
		return nil
	}))
	return out
}
func receiptPath(x component) string {
	return "share/symphony/receipts/" + x.ID + "/" + x.Version + "/install-receipt.json"
}
func sameSet(a, b []string) bool {
	a = append([]string{}, a...)
	b = append([]string{}, b...)
	sort.Strings(a)
	sort.Strings(b)
	return reflect.DeepEqual(a, b)
}
func selectGraph(p projection, group string) []string {
	byPath := map[string]component{}
	for _, x := range p.Components {
		require(byPath[x.Path].Path == "", "duplicate mapped root")
		byPath[x.Path] = x
	}
	selected := map[string]bool{}
	visiting := map[string]bool{}
	var visit func(string)
	visit = func(path string) {
		require(!visiting[path], "runtime SDK cycle")
		if selected[path] {
			return
		}
		x, ok := byPath[path]
		require(ok && x.Installable, "missing installed SDK dependency: "+path)
		visiting[path] = true
		for _, d := range x.Dependencies {
			if d.Scope == "runtime" {
				visit(d.Path)
			}
		}
		delete(visiting, path)
		selected[path] = true
	}
	for _, x := range p.Components {
		if x.Group == group && x.Installable {
			visit(x.Path)
		}
	}
	var ordered []string
	seen := map[string]bool{}
	for _, path := range p.Order {
		if !selected[path] {
			continue
		}
		require(!seen[path], "duplicate topological root")
		for _, d := range byPath[path].Dependencies {
			if d.Scope == "runtime" {
				require(seen[d.Path], "runtime SDK order is not topological")
			}
		}
		seen[path] = true
		ordered = append(ordered, path)
	}
	require(len(ordered) > 0 && len(ordered) == len(selected), "incomplete selected runtime graph")
	return ordered
}
func (c *campaign) verifyReceipt(x component, prefix, label string, success bool) {
	c.run(label, c.work, success, "cmake", "-DMVP_REPOSITORY="+c.repo, "-DMVP_COMPONENT_FILE="+filepath.Join(c.work, "metadata", x.ID+".json"), "-DMVP_SELECTED_PREFIX="+prefix, "-P", filepath.Join(c.repo, "tests/mvp-dependencies/verify-receipt.cmake"))
}
func (c *campaign) completed(label string) {
	b := read(filepath.Join(c.installed, "completed-steps.txt"))
	require(strings.Contains("\n"+string(b), "\n"+label+"\n"), "required prior gate missing: "+label)
	path := filepath.Join(c.installed, "logs", label+".log")
	regular(path)
	if c.evidence["prerequisite_logs"] == nil {
		c.evidence["prerequisite_logs"] = map[string]string{}
	}
	c.evidence["prerequisite_logs"].(map[string]string)[label] = hash(read(path))
}
func (c *campaign) covered(selected []string, m manifest) {
	all := map[string]coverage{}
	for _, r := range m.SDKs {
		require(all[r.Path].Path == "", "duplicate SDK coverage row")
		all[r.Path] = r
	}
	count := 0
	for _, x := range c.components {
		if len(x.Exports) == 0 {
			continue
		}
		count++
		r, ok := all[x.Path]
		require(ok, "unclassified shipped SDK: "+x.Path)
		var packages []string
		for _, p := range x.Exports {
			packages = append(packages, p.Package)
		}
		require(sameSet(packages, r.Exports), "SDK export coverage drift: "+x.Path)
	}
	require(count == len(all), "stale SDK coverage row")
	discovered := map[string]bool{}
	for _, parent := range []string{"modules", "libraries"} {
		matches, e := filepath.Glob(filepath.Join(c.repo, parent, "*", "tests", "sdk-consumer", "CMakeLists.txt"))
		must(e)
		for _, p := range matches {
			r, e := filepath.Rel(c.repo, p)
			must(e)
			discovered[filepath.ToSlash(r)] = true
		}
	}
	mapped := map[string]bool{}
	excluded := map[string]bool{}
	for _, e := range m.Exclusions {
		require(e.Reason != "", "unexplained external exclusion")
		regular(filepath.Join(c.repo, e.Path))
		excluded[e.Path] = true
	}
	for _, row := range m.Separate {
		require(row.Reason != "", "unexplained separate campaign")
		regular(filepath.Join(c.repo, row.Path))
		regular(filepath.Join(c.repo, row.Workflow))
		excluded[row.Path] = true
	}
	for _, x := range c.components {
		for _, check := range x.Consumers {
			if strings.HasSuffix(check.Path, "/tests/sdk-consumer/CMakeLists.txt") {
				mapped[check.Path] = true
			} else {
				require(excluded[check.Path], "unclassified special SDK consumer: "+check.Path)
			}
		}
	}
	require(reflect.DeepEqual(mapped, discovered), "shipped SDK consumer inventory differs from copied-consumer map")
	for _, path := range selected {
		x := c.components[path]
		if len(x.Exports) == 0 {
			continue
		}
		r := all[path]
		switch r.Coverage {
		case "mvp_copied_consumer":
			require(len(x.Consumers) > 0, "missing shipped consumer")
			for _, label := range []string{"configure", "build", "run"} {
				c.completed(x.ID + "-consumer-" + label)
			}
			for _, item := range x.Consumers {
				if !strings.HasSuffix(item.Path, "/tests/sdk-consumer/CMakeLists.txt") {
					continue
				}
				source := filepath.Dir(filepath.Join(c.repo, item.Path))
				copied := filepath.Join(c.installed, "consumers", x.ID)
				must(filepath.WalkDir(source, func(p string, d os.DirEntry, e error) error {
					if e != nil {
						return e
					}
					if d.IsDir() {
						return nil
					}
					relative, e := filepath.Rel(source, p)
					if e != nil {
						return e
					}
					require(hash(read(p)) == hash(read(filepath.Join(copied, relative))), "copied public consumer source drift")
					return nil
				}))
			}
		case "mvp_c_abi_cpp_interop":
			for _, label := range []string{"sbv-consumer-configure", "sbv-consumer-build", "sbv-consumer-tests", "sbv-consumer-loader-proof"} {
				c.completed(label)
			}
		case "snv_campaign":
			require(c.prior != "", "knowledge campaign required")
			e := decode(read(filepath.Join(c.prior, "evidence.json")))
			require(e["complete"] == true && e["sdk_semantic_owners"] == json.Number("5"), "SNV installed campaign incomplete")
			hashes := e["receipt_sha256"].(map[string]any)
			require(hashes[receiptPath(x)] == hash(read(filepath.Join(c.installed, "prefix", receiptPath(x)))), "SNV campaign selected another installed SDK")
		case "mechanics_consumer": // Executed below against the selected original prefix.
		default:
			panic("unknown SDK coverage kind: " + r.Coverage)
		}
		c.check(true, "bound installed consumer coverage: "+x.ID)
	}
	sourceExamples := map[string]bool{}
	for _, parent := range []string{"modules", "libraries"} {
		for _, leaf := range []string{"main.go", "main.cpp", "main.c"} {
			matches, err := filepath.Glob(filepath.Join(c.repo, parent, "*", "examples", "*", leaf))
			must(err)
			for _, path := range matches {
				relative, err := filepath.Rel(c.repo, filepath.Dir(path))
				must(err)
				sourceExamples[filepath.ToSlash(relative)] = true
			}
		}
	}
	declaredExamples := map[string]bool{}
	for _, row := range m.Sources {
		require(row.Path == "libraries/sbv-go/examples/roundtrip" && row.Coverage == "copied_source_against_exact_native_artifacts" && row.Group == "quantitative", "unimplemented source consumer qualification: "+row.Path)
		declaredExamples[row.Path] = true
	}
	require(reflect.DeepEqual(sourceExamples, declaredExamples), "shipped source example inventory changed")
	c.evidence["sdk_coverage_manifest_sha256"] = hash(read(filepath.Join(c.repo, "tests/native-sdk-distribution/manifest.json")))
	c.evidence["external_exclusions"] = m.Exclusions
	c.evidence["separate_campaigns"] = m.Separate
}
func (c *campaign) mechanics(selected []string) {
	common := false
	for _, p := range selected {
		common = common || p == "libraries/snv-common-cpp"
	}
	dir := filepath.Join(c.work, "mechanics")
	copyTree(filepath.Join(c.repo, "tests/native-sdk-distribution/mechanics"), dir)
	deps := []object{{"target": "Symphony::KnowledgeVectorEngine", "scope": "runtime"}}
	if common {
		deps = append(deps, object{"target": "Symphony::SnvCommon", "scope": "runtime"})
	}
	metadata := filepath.Join(c.work, "metadata/mechanics.json")
	write(metadata, object{"project_version": nil, "artifact_targets": []string{"consumer"}, "first_party_sdk_dependencies": deps})
	c.run("mechanics-configure", c.work, true, "cmake", "-S", dir, "-B", dir+"/build", "-DCMAKE_BUILD_TYPE=Release", "-DCMAKE_POSITION_INDEPENDENT_CODE=ON", "-DCMAKE_FIND_USE_PACKAGE_REGISTRY=OFF", "-DCMAKE_FIND_USE_SYSTEM_PACKAGE_REGISTRY=OFF", "-DCMAKE_PREFIX_PATH="+filepath.Join(c.installed, "prefix"), fmt.Sprintf("-DSDK_INCLUDE_SNV_COMMON=%t", common), "-DCMAKE_PROJECT_TOP_LEVEL_INCLUDES="+filepath.Join(c.repo, "tests/mvp-dependencies/prove-imports.cmake"), "-DMVP_REPOSITORY="+c.repo, "-DMVP_SELECTED_PREFIX="+filepath.Join(c.installed, "prefix"), "-DMVP_COMPONENT_FILE="+metadata, "-DMVP_PROOF_LOG="+filepath.Join(c.work, "logs/mechanics-imports.log"))
	c.run("mechanics-build", c.work, true, "cmake", "--build", dir+"/build", "--parallel", "2")
	c.run("mechanics-run", c.work, true, dir+"/build/consumer")
	c.check(true, "independent foundation/common headers, exports and runtime")
}
func (c *campaign) sourceAdapter() {
	if c.group != "quantitative" {
		return
	}
	require(c.prior != "", "completed quantitative campaign required")
	prior := decode(read(filepath.Join(c.prior, "evidence.json")))
	require(prior["status"] == "passed", "SBV native artifact campaign incomplete")
	qxctl := filepath.Join(c.prior, "bin/qxctl")
	execs := prior["executables"].(map[string]any)
	require(execs["qxctl"] == hash(read(qxctl)), "SBV campaign qxctl drift")
	native := filepath.Join(c.installed, "prefix/libexec/symphony/sbv-engine/0.33.0-dev/symphony-sbv")
	require(execs["native"] == hash(read(native)), "SBV campaign native owner drift")
	module := filepath.Join(c.work, "sbv-go")
	copyTree(filepath.Join(c.repo, "libraries/sbv-go"), module)
	binary := filepath.Join(c.work, "source-roundtrip")
	c.run("source-adapter-build", module, true, "go", "build", "-mod=readonly", "-trimpath", "-o", binary, "./examples/roundtrip")
	requestDir := filepath.Join(c.work, "tmp/requests")
	must(os.Mkdir(requestDir, 0700))
	cfg := filepath.Join(c.work, "metadata/sbv-go.json")
	write(cfg, object{"Executable": qxctl, "ExecutableSHA256": hash(read(qxctl)), "Prefix": filepath.Join(c.installed, "prefix"), "Version": "0.33.0-dev", "WorkDir": filepath.Join(c.work, "tmp"), "RequestDir": requestDir})
	output := filepath.Join(c.work, "source-roundtrip-output")
	c.run("source-adapter-run", c.work, true, binary, cfg, filepath.Join(c.prior, "outputs/evaluation.json"), output)
	result := decode(read(filepath.Join(output, "report.json")))
	require(result["status"] == "passed" && result["provider_requests"] == "0" && result["resume_executed"] == "0", "source adapter roundtrip incomplete")
	c.evidence["source_adapter"] = object{"module": "libraries/sbv-go", "example": "examples/roundtrip", "binary_sha256": hash(read(binary)), "qxctl_sha256": execs["qxctl"], "native_sha256": execs["native"], "prior_evidence_sha256": hash(read(filepath.Join(c.prior, "evidence.json"))), "report": result}
	c.check(true, "copied Go source consumer executes against exact installed SBV without provider calls")
}
func (c *campaign) detached(x component, r receipt) string {
	src := filepath.Join(c.installed, "build", x.ID)
	dst := filepath.Join(c.work, "uninstallers", x.ID)
	must(os.MkdirAll(dst, 0700))
	files := []string{"uninstall.cmake", "SymphonyUninstallReceiptV2.cmake", "SymphonyReceiptV2Support.cmake"}
	identities, e := filepath.Glob(filepath.Join(src, "symphony-receipt-v2-identity-*.cmake"))
	must(e)
	require(len(identities) == 1, "exact configured identity helper required: "+x.ID)
	files = append(files, filepath.Base(identities[0]))
	for _, name := range files {
		copyFile(filepath.Join(src, name), filepath.Join(dst, name))
	}
	output := filepath.Join(dst, "configured-files.txt")
	c.run(x.ID+"-configured-inventory", c.work, true, "cmake", "-DSDK_UNINSTALLER="+filepath.Join(dst, "uninstall.cmake"), "-DSDK_INVENTORY_OUTPUT="+output, "-P", filepath.Join(c.repo, "tests/native-sdk-distribution/inspect-uninstaller.cmake"))
	configured := strings.Split(string(read(output)), ";")
	expected := []string{}
	for _, f := range r.Files {
		expected = append(expected, f.Path)
	}
	expected = append(expected, receiptPath(x))
	require(sameSet(configured, expected), "configured uninstaller does not own exact final receipt file set: "+x.ID)
	c.check(true, "detached configured inventory matches final receipt: "+x.ID)
	return filepath.Join(dst, "uninstall.cmake")
}
func (c *campaign) remove(x component, prefix, script, label string, success bool) {
	c.run(x.ID+"-"+label, c.work, success, "cmake", "-DINSTALL_PREFIX="+prefix, "-P", script)
}
func (c *campaign) refusal(x component, prefix, script, relative, label string, mutate func([]byte) []byte, receiptReject bool) {
	p := filepath.Join(prefix, relative)
	s := regular(p)
	original := read(p)
	must(os.WriteFile(p, mutate(append([]byte{}, original...)), s.Mode().Perm()))
	before := snapshot(prefix)
	c.remove(x, prefix, script, label, false)
	c.check(reflect.DeepEqual(before, snapshot(prefix)), x.ID+": "+label+" preserves every remaining byte")
	if receiptReject {
		c.verifyReceipt(x, prefix, x.ID+"-"+label+"-admission", false)
	}
	must(os.WriteFile(p, original, s.Mode().Perm()))
}
func (c *campaign) qualify() {
	var p projection
	raw := read(filepath.Join(c.repo, "cmake/mvp-dependencies.json"))
	must(json.Unmarshal(raw, &p))
	c.components = map[string]component{}
	c.raw = map[string]json.RawMessage{}
	var mapped struct {
		Components []json.RawMessage `json:"native_components"`
	}
	must(json.Unmarshal(raw, &mapped))
	for i, x := range p.Components {
		c.components[x.Path] = x
		c.raw[x.Path] = mapped.Components[i]
	}
	selected := selectGraph(p, c.group)
	actual := strings.Split(strings.TrimSpace(string(read(filepath.Join(c.installed, "selected-components.txt")))), "\n")
	require(len(actual) > 0 && actual[0] == c.group && reflect.DeepEqual(actual[1:], selected), "selected installation is not the exact group runtime DAG")
	inputs := map[string]string{}
	for _, relative := range []string{"tests/native-sdk-distribution/run.sh", "tests/native-sdk-distribution/main.go", "tests/native-sdk-distribution/main_test.go", "tests/native-sdk-distribution/manifest.json", "tests/native-sdk-distribution/inspect-uninstaller.cmake", "tests/native-sdk-distribution/mechanics/CMakeLists.txt", "tests/native-sdk-distribution/mechanics/main.cpp"} {
		inputs[relative] = hash(read(filepath.Join(c.repo, relative)))
	}
	c.evidence["harness_input_sha256"] = inputs
	c.evidence["selected_runtime_dag"] = selected
	c.evidence["projection_sha256"] = hash(raw)
	var m manifest
	must(json.Unmarshal(read(filepath.Join(c.repo, "tests/native-sdk-distribution/manifest.json")), &m))
	c.covered(selected, m)
	originalPrefix := filepath.Join(c.installed, "prefix")
	original := snapshot(originalPrefix)
	c.evidence["original_files"] = original
	receipts := map[string]receipt{}
	scripts := map[string]string{}
	for _, path := range selected {
		x := c.components[path]
		for _, step := range []string{"configure", "build", "install", "receipt"} {
			c.completed(x.ID + "-" + step)
		}
		must(os.WriteFile(filepath.Join(c.work, "metadata", x.ID+".json"), c.raw[path], 0600))
		c.verifyReceipt(x, originalPrefix, x.ID+"-receipt-before", true)
		var r receipt
		must(json.Unmarshal(read(filepath.Join(originalPrefix, receiptPath(x))), &r))
		require(len(r.Files) > 0, "empty receipt")
		receipts[path] = r
		scripts[path] = c.detached(x, r)
	}
	c.evidence["detached_uninstaller_files"] = snapshot(filepath.Join(c.work, "uninstallers"))
	c.mechanics(selected)
	c.sourceAdapter()
	copyPrefix := filepath.Join(c.work, "disposable-prefix")
	copyTree(originalPrefix, copyPrefix)
	must(os.MkdirAll(filepath.Join(copyPrefix, "unrelated/nested"), 0700))
	must(os.WriteFile(filepath.Join(copyPrefix, "unrelated/nested/user-evidence.txt"), []byte("caller evidence survives every package removal\n"), 0600))
	expected := snapshot(copyPrefix)
	for i := len(selected) - 1; i >= 0; i-- {
		path := selected[i]
		x := c.components[path]
		r := receipts[path]
		script := scripts[path]
		appendTamper := func(b []byte) []byte { return append(b, []byte("\nintentional qualification tamper\n")...) }
		c.refusal(x, copyPrefix, script, r.Files[0].Path, "owned-byte-tamper", appendTamper, true)
		if len(x.Exports) > 0 {
			header, export := "", ""
			for _, f := range r.Files {
				if header == "" && (strings.HasSuffix(f.Path, ".hpp") || strings.HasSuffix(f.Path, ".h")) {
					header = f.Path
				}
				if export == "" && strings.HasSuffix(f.Path, "Config.cmake") {
					export = f.Path
				}
			}
			require(header != "" && export != "", "SDK lacks receipt-owned public headers/export: "+x.ID)
			c.refusal(x, copyPrefix, script, header, "header-tamper", appendTamper, true)
			c.refusal(x, copyPrefix, script, export, "export-tamper", appendTamper, true)
		}
		c.refusal(x, copyPrefix, script, receiptPath(x), "receipt-digest-tamper", func(b []byte) []byte {
			v := decode(b)
			v["receipt_digest"] = "sha256:" + strings.Repeat("0", 64)
			return encode(v)
		}, true)
		c.refusal(x, copyPrefix, script, receiptPath(x), "resealed-identity-tamper", func(b []byte) []byte {
			v := decode(b)
			v["component_kind"] = "qualification-forged-kind"
			delete(v, "receipt_digest")
			canonical, e := json.Marshal(v)
			must(e)
			v["receipt_digest"] = "sha256:" + hash(canonical)
			return encode(v)
		}, false)
		c.check(reflect.DeepEqual(expected, snapshot(copyPrefix)), x.ID+": tamper restoration is exact")
		// Missing owned bytes are an explicitly supported interrupted-removal case.
		// The consumer/receipt gate must reject that incomplete package; the exact
		// configured uninstaller then finishes removing only its remaining files.
		must(os.Remove(filepath.Join(copyPrefix, r.Files[0].Path)))
		c.verifyReceipt(x, copyPrefix, x.ID+"-partial-removal-admission", false)
		c.remove(x, copyPrefix, script, "resume-removal", true)
		for _, f := range r.Files {
			delete(expected, f.Path)
		}
		delete(expected, receiptPath(x))
		c.check(reflect.DeepEqual(expected, snapshot(copyPrefix)), x.ID+": removal preserves unrelated packages and caller evidence")
		c.remove(x, copyPrefix, script, "repeat-removal", true)
		c.check(reflect.DeepEqual(expected, snapshot(copyPrefix)), x.ID+": repeated removal is idempotent")
	}
	c.check(len(expected) == 1, "reverse DAG removed all packages and preserved only unrelated evidence")
	c.check(reflect.DeepEqual(original, snapshot(originalPrefix)), "original installed graph was never mutated")
	c.evidence["remaining_disposable_files"] = expected
	c.evidence["status"] = "passed"
}
func main() {
	c := campaign{evidence: object{"format_version": 1, "status": "in_progress", "publication": false, "activation": false, "mutation_scope": "private copy of completed final installed graph"}}
	flag.StringVar(&c.repo, "repo", "", "source")
	flag.StringVar(&c.group, "group", "", "group")
	flag.StringVar(&c.installed, "installed", "", "completed MVP workspace")
	flag.StringVar(&c.work, "work", "", "private work")
	flag.StringVar(&c.prior, "campaign", "", "completed selected native campaign")
	flag.Parse()
	defer func() {
		failed := false
		if p := recover(); p != nil {
			c.evidence["status"] = "failed"
			c.evidence["failure"] = fmt.Sprint(p)
			fmt.Fprintln(os.Stderr, p)
			failed = true
		}
		c.evidence["checks"] = c.checks
		c.evidence["check_count"] = len(c.checks)
		c.evidence["group"] = c.group
		c.evidence["captured_at"] = time.Now().UTC().Format(time.RFC3339)
		logs := snapshot(filepath.Join(c.work, "logs"))
		delete(logs, "qualification.log")
		c.evidence["logs"] = logs
		write(filepath.Join(c.work, "evidence.json"), c.evidence)
		if failed {
			os.Exit(1)
		}
	}()
	for _, path := range []string{c.repo, c.installed, c.work} {
		require(filepath.IsAbs(path), "absolute paths required")
	}
	c.qualify()
	fmt.Printf("PASS installed distribution: %d assertions; original prefix preserved; no activation\n", len(c.checks))
}
