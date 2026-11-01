// Executable qualification driver. All authoritative scientific and durable-job
// results come from the selected installed native owner, through real qxctl/SOV.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type object = map[string]any

var prefix, work, qxctl, adapter, self string
var serial int
var checks []string
var evidence = object{"format_version": 1, "status": "in_progress", "qualification": "sbv-mvp-installed-process", "live_data": false, "gpu_execution": false, "network_transport": false, "principal_boundary": "same_uid_private_owner_filesystem", "checkpoint_resume": false}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func check(ok bool, label string) {
	if !ok {
		panic("assertion: " + label)
	}
	checks = append(checks, label)
}
func encode(v any) []byte { b, e := json.MarshalIndent(v, "", "  "); must(e); return append(b, '\n') }
func decode(b []byte) object {
	var o object
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	must(d.Decode(&o))
	var more any
	if d.Decode(&more) != io.EOF {
		panic("trailing JSON")
	}
	return o
}
func read(path string) object  { b, e := os.ReadFile(path); must(e); return decode(b) }
func write(path string, v any) { must(os.WriteFile(path, encode(v), 0600)) }
func digest(path string) string {
	b, e := os.ReadFile(path)
	must(e)
	return fmt.Sprintf("%x", sha256.Sum256(b))
}
func copyObject(x object) object        { return decode(encode(x)) }
func obj(x any) object                  { return x.(map[string]any) }
func section(x object, name string) any { return obj(obj(x["sections"])[name])["data"] }
func path(name string) string           { return filepath.Join(work, "outputs", name) }
func input(v any) string {
	serial++
	p := path(fmt.Sprintf("%03d-input.json", serial))
	write(p, v)
	return p
}
func command(args []string, env []string) (object, error) {
	serial++
	stem := filepath.Join(work, "logs", fmt.Sprintf("%03d", serial))
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, qxctl, args...)
	cmd.Env = append(os.Environ(), env...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	b, e := cmd.Output()
	must(os.WriteFile(stem+"-stdout.json", b, 0600))
	must(os.WriteFile(stem+"-stderr.txt", stderr.Bytes(), 0600))
	if len(b) == 0 {
		panic(fmt.Sprintf("qxctl %v: %v: %s", args, e, stderr.String()))
	}
	result := decode(b)
	return result, e
}
func invoke(args ...string) object {
	r, e := command(args, nil)
	if e != nil {
		panic(fmt.Sprintf("qxctl %v: %v: %s", args, e, encode(r)))
	}
	return r
}
func sbvArgs(op string) []string {
	args := []string{"sbv"}
	args = append(args, strings.Split(op, "_")...)
	return append(args, "--prefix", prefix, "--version", "0.33.0-dev", "--json")
}
func sbv(op string, request object) object {
	args := sbvArgs(op)
	if strings.HasPrefix(op, "result_") {
		args = append(args, "--path", request["path"].(string), "--digest", request["expected_sha256"].(string))
		if op == "result_query" {
			args = append(args, "--pointer", request["pointer"].(string), "--limit", "1")
		}
	} else {
		args = append(args, "--input", input(request))
	}
	return invoke(args...)
}
func template(op, variant string) object {
	args := append(sbvArgs("template"), "--operation", op)
	if variant != "" {
		args = append(args, "--variant", variant)
	}
	return obj(invoke(args...)["template"])
}
func jobRequest(op string, a object) object {
	r := copyObject(a)
	r["protocol"] = "symphony.sbv." + strings.ReplaceAll(op, "_", "-") + "-input.v1"
	return r
}
func nativeJob(op string, a object) object { return sbv(op, jobRequest(op, a)) }
func createJob(name string, request object) (object, object) {
	dir := path(name)
	must(os.Mkdir(dir, 0700))
	c := template("job_create", "")
	c["directory"] = dir
	c["job_id"] = name
	if request != nil {
		c["request"] = request
	}
	obj(c["request"])["output_path"] = path(name + "-result.json")
	r := sbv("job_create", c)
	check(r["phase"] == "queued", name+" created queued")
	return object{"directory": dir, "job_id": name, "expected_plan_sha256": r["plan_sha256"]}, r
}
func publicResult(receipt object) object {
	return object{"path": receipt["path"], "expected_sha256": receipt["content_sha256"]}
}
func census() {
	source := path("fixture.dbn")
	check(fileSize(source) == 1200, "exact 15-record synthetic DBN source")
	run := template("run", "")
	run["source_path"] = source
	run["source_sha256"] = digest(source)
	run["output_path"] = path("census-run.json")
	run["criteria"] = object{"rule": "spaced_trades", "spacing_ns": "200", "min_trade_size": "1", "direction": "any", "max_signals": "2"}
	run["execution"] = object{"model": "none", "horizon_ns": "1100", "price_offsets_nanos": []any{}, "probability_numerator": "0", "probability_denominator": "1"}
	run["replay"] = object{"before_ns": "300", "after_ns": "1100", "retain_events": true}
	run["studies"] = []any{"signal_summary"}
	run["workers"] = "1"
	receipt := sbv("run", run)
	native := read(run["output_path"].(string))
	check(receipt["content_sha256"] == native["content_sha256"], "native census receipt binds result")
	signals := section(native, "signals").([]any)
	check(len(signals) == 2, "closed census contains exactly two signals")
	check(obj(signals[0])["source_ordinal"] == "3" && obj(signals[1])["source_ordinal"] == "5", "closed census identity and ordering")
	check(obj(section(native, "summary"))["additional_eligible_after_cap"] != "0", "selection cap is explicitly partial")
	check(reflect.DeepEqual(sbv("result_export", publicResult(receipt)), native), "CLI export preserves complete native typed result")
	query := publicResult(receipt)
	query["pointer"] = "/sections/census/data/census_sha256"
	sbv("result_query", query)
	evaluation := template("evaluate", "")
	evaluation["source_path"] = source
	evaluation["source_sha256"] = digest(source)
	evaluation["dataset"] = "GLBX.MDP3"
	evaluation["census"] = object{"path": run["output_path"], "expected_sha256": native["content_sha256"], "pointer": ""}
	evaluation["model"] = object{"protocol": "symphony.sbv.model-selection.v1", "id": "observed_trade_levels", "version": "1", "horizon_ns": "1100", "parameters": object{"levels_per_side": "1", "include_anchor": false, "thin_support": "unavailable"}}
	evaluation["replay"] = run["replay"]
	evaluation["studies"] = []any{"signal_summary", "model_summary"}
	evaluation["workers"] = "1"
	evaluation["output_path"] = path("evaluation.json")
	sbv("evaluate", evaluation)
	first := read(evaluation["output_path"].(string))
	check(reflect.DeepEqual(section(first, "signals"), section(native, "signals")) && reflect.DeepEqual(section(first, "census"), section(native, "census")), "second pass cannot replace the closed census")
	evaluation["census"] = object{"path": evaluation["output_path"], "expected_sha256": first["content_sha256"], "pointer": ""}
	evaluation["output_path"] = path("reevaluation.json")
	obj(obj(evaluation["model"])["parameters"])["include_anchor"] = true
	must(os.Remove(run["output_path"].(string)))
	sbv("evaluate", evaluation)
	second := read(evaluation["output_path"].(string))
	check(reflect.DeepEqual(section(first, "census"), section(second, "census")), "retained census is self-contained after ancestor removal")
	obj(evaluation["census"])["pointer"] = "/sections/signals/data"
	evaluation["output_path"] = path("invalid-census.json")
	_, err := command(append(sbvArgs("evaluate"), "--input", input(evaluation)), nil)
	check(err != nil && absent(evaluation["output_path"].(string)), "bare signal array refused without output publication")
	evidence["data"] = object{"source_sha256": digest(source), "source_bytes": 1200, "records": 15, "instrument_id": "5482", "dataset": "GLBX.MDP3", "selected_signals": 2, "workers": 1, "calibrated_market_evidence": false, "selection": "explicitly_capped_partial"}
}
func fileSize(p string) int64 { s, e := os.Stat(p); must(e); return s.Size() }
func absent(p string) bool    { _, e := os.Stat(p); return os.IsNotExist(e) }

type remote struct{ selection, endpoint object }

func setupRemote(a object, carrier string) remote {
	s := obj(invoke("sov", "remote", "template", "--operation", "plan", "--json")["template"])
	s["operation_id"] = "sbv-mvp-qualification"
	s["job"] = object{"directory": a["directory"], "job_id": a["job_id"], "plan_sha256": a["expected_plan_sha256"]}
	s["target"] = object{"tops_id": "fixture-tops", "node_id": "fixture-node", "habitat_id": "fixture-habitat", "identity_sha256": digest(path("source-commit.txt"))}
	s["transport"] = object{"arrangement": "controller", "family": "ipc", "adapter_id": "qualification-stdio", "adapter_version": "0.1.0-dev", "adapter_sha256": digest(carrier), "endpoint_ref": "private-owner-process"}
	receipt := read(filepath.Join(prefix, "share/symphony/receipts/sbv-engine/0.33.0-dev/install-receipt.json"))
	s["runtime"] = object{"qxctl_version": "0.1.0-dev", "qxctl_sha256": digest(qxctl), "sbv_version": "0.33.0-dev", "sbv_installation_sha256": strings.TrimPrefix(receipt["receipt_digest"].(string), "sha256:")}
	config := obj(invoke("sov", "remote", "template", "--operation", "target_configure", "--json")["template"])
	cfg := obj(config["configuration"])
	cfg["target"] = s["target"]
	cfg["job"] = s["job"]
	cfg["runtime"] = s["runtime"]
	cfg["authority"] = s["authority"]
	cfg["sbv_prefix"] = prefix
	config["path"] = filepath.Join(a["directory"].(string), "target-configuration.json")
	result := invoke("sov", "target", "configure", "--input", input(config), "--json")
	return remote{s, object{"target_qxctl": object{"path": qxctl, "sha256": digest(qxctl)}, "configuration_path": config["path"], "configuration_sha256": result["sha256"]}}
}

// Result reads require an explicit owner projection; a job receipt is not a
// controller permission to read arbitrary target files.
func (r *remote) projectResult(receipt object) {
	cfg := read(r.endpoint["configuration_path"].(string))
	cfg["allowed_artifacts"] = []any{object{"path": receipt["path"], "sha256": receipt["content_sha256"]}}
	updated := invoke("sov", "target", "configure", "--input", input(object{
		"protocol": "symphony.sov.target-configure-input.v1", "path": r.endpoint["configuration_path"],
		"expected_sha256": r.endpoint["configuration_sha256"], "configuration": cfg,
	}), "--json")
	r.endpoint["configuration_sha256"] = updated["sha256"]
}

func (r remote) request(op string, a object, carrier string) object {
	s := copyObject(r.selection)
	obj(s["transport"])["adapter_sha256"] = digest(carrier)
	p := invoke("sov", "remote", "plan", "--input", input(s), "--json")
	serial++
	id := fmt.Sprintf("00000000-0000-4000-8000-%012d", serial)
	return object{"protocol": "symphony.sov.remote-invoke-input.v1", "selection": s, "expected_plan_sha256": p["plan_sha256"], "request_id": id, "correlation_id": "00000000-0000-4000-8000-000000000002", "operation": op, "request": jobRequest(op, a), "adapter": object{"path": carrier, "sha256": digest(carrier)}, "endpoint": r.endpoint}
}
func (r remote) call(op string, a object) object {
	result, e := command([]string{"sov", "remote", "invoke", "--input", input(r.request(op, a, adapter)), "--json"}, nil)
	check(e == nil && result["transport_outcome"] == "response_received" && result["automatic_retry"] == false, "remote "+op+" exact response without retry")
	target := obj(result["target_result"])
	check(target["status"] == "completed" && target["authority_mode"] == "owner_filesystem", "target owner boundary "+op)
	return obj(target["result"])
}
func durableCompletion() {
	a, _ := createJob("lost-reply", nil)
	r := setupRemote(a, adapter)
	// This explicit fault carrier forwards through the production stdio adapter,
	// waits for its real reply, saves that reply, then drops it at the controller.
	lost, err := command([]string{"sov", "remote", "invoke", "--input", input(r.request("job_execute", a, self)), "--json"}, []string{"SBV_QUALIFICATION_REAL_ADAPTER=" + adapter, "SBV_QUALIFICATION_DROPPED_REPLY=" + path("dropped-target-reply.json")})
	check(err != nil && lost["transport_outcome"] == "response_lost" && lost["automatic_retry"] == false && lost["target_result"] == nil, "lost completed response remains indeterminate at controller")
	target := read(path("dropped-target-reply.json"))
	check(target["status"] == "completed" && obj(target["result"])["phase"] == "completed", "fault injected after actual native completion")
	observed := r.call("job_inspect", a)
	check(observed["phase"] == "completed", "recovery observes original completion")
	before := snapshotFiles(a["directory"].(string))
	duplicate := r.call("job_execute", a)
	check(reflect.DeepEqual(observed["result_receipt"], duplicate["result_receipt"]) && observed["generation"] == duplicate["generation"], "duplicate execute returns original generation and receipt")
	check(reflect.DeepEqual(before, snapshotFiles(a["directory"].(string))), "duplicate execution does not rewrite durable job files")
	a["expected_generation"] = observed["generation"]
	reconciled := r.call("job_reconcile", a)
	check(reconciled["generation"] == observed["generation"], "completed reconciliation is idempotent")
	a["request_id"] = "late-cancel"
	a["reason"] = "fixture completion won"
	late := r.call("job_cancel", a)
	check(late["phase"] == "completed" && late["cancellation_requested"] == false, "late cancellation cannot rewrite completion")
	receipt := obj(observed["result_receipt"])
	original := read(receipt["path"].(string))
	check(reflect.DeepEqual(sbv("result_export", publicResult(receipt)), original), "job result export preserves authoritative native result")
	r.projectResult(receipt)
	readRequest := publicResult(receipt)
	remoteInspect := r.call("result_inspect", readRequest)
	localInspect := sbv("result_inspect", readRequest)
	check(reflect.DeepEqual(remoteInspect, localInspect), "remote result inspection preserves exact owner result")
	readRequest["pointer"] = "/sections"
	readRequest["limit"] = "1"
	readRequest["cursor"] = ""
	remoteQuery := r.call("result_query", readRequest)
	localQuery := sbv("result_query", readRequest)
	check(reflect.DeepEqual(remoteQuery, localQuery), "remote query preserves complete native typed page")
	evidence["completed_job_receipt"] = receipt
}
func snapshotFiles(root string) object {
	out := object{}
	must(filepath.WalkDir(root, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if !d.IsDir() {
			out[strings.TrimPrefix(p, root+"/")] = digest(p)
		}
		return nil
	}))
	return out
}

type cleanupProbe struct{}

func barrier(name string, kill, failCleanup bool) {
	req := template("account_simulate", "native_feedback")
	lib := obj(obj(req["feedback"])["provider"])["library"].(map[string]any)
	lib["path"] = filepath.Join(work, "fixtures", "launch-feedback.so")
	lib["expected_sha256"] = digest(lib["path"].(string))
	entered, release, pidfile := path(name+"-entered"), path(name+"-release"), path(name+"-pid")
	obj(obj(req["streams"].([]any)[0])["events"].([]any)[0])["context"] = object{"mode": "job_barrier", "entered": entered, "release": release, "pid_marker": pidfile}
	a, _ := createJob(name, req)
	r := setupRemote(a, adapter)
	request := r.request("job_execute", a, adapter)
	in := input(request)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, qxctl, "sov", "remote", "invoke", "--input", in, "--json")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		if err == syscall.ESRCH {
			return os.ErrProcessDone
		}
		return err
	}
	cmd.WaitDelay = 3 * time.Second
	stdout, e := os.OpenFile(path(name+"-executor.json"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	must(e)
	defer stdout.Close()
	stderr, e := os.OpenFile(filepath.Join(work, "logs", name+"-executor.stderr"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	must(e)
	defer stderr.Close()
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	must(cmd.Start())
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	finished := false
	defer func() {
		// Unblock the known test provider first. SIGTERM then follows the real
		// controller/adapter/target cancellation path; those owners reap their
		// children. Keep an isolated controller group and a bounded final reap
		// even when a qualification assertion fails before the normal wait.
		_ = os.WriteFile(release, []byte("cleanup\n"), 0600)
		cancel()
		if !finished {
			select {
			case <-done:
				finished = true
			case <-time.After(6 * time.Second):
				_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
				select {
				case <-done:
					finished = true
				case <-time.After(3 * time.Second):
					fmt.Fprintln(os.Stderr, "controller reap exceeded cleanup bound")
				}
			}
		}
		write(path(name+"-cleanup.json"), object{"controller_reaped": finished, "controller_pid": cmd.Process.Pid})
	}()
	deadline := time.Now().Add(20 * time.Second)
	for absent(entered) {
		select {
		case err := <-done:
			finished = true
			panic(fmt.Sprintf("barrier exited before entry: %v", err))
		default:
		}
		if time.Now().After(deadline) {
			panic("barrier entry timeout")
		}
		time.Sleep(10 * time.Millisecond)
	}
	running := r.call("job_inspect", a)
	check(running["phase"] == "running" && running["execution_lock_held"] == true, "native "+name+" admitted and running")
	firstPID, e := os.ReadFile(pidfile)
	must(e)
	pid, e := strconv.Atoi(strings.TrimSpace(string(firstPID)))
	must(e)
	check(pid > 1 && pid != os.Getpid(), "test fixture identifies separate native worker")
	duplicate := r.call("job_execute", a)
	currentPID, e := os.ReadFile(pidfile)
	must(e)
	afterDuplicate := r.call("job_inspect", a)
	check(bytes.Equal(firstPID, currentPID) && duplicate["generation"] == running["generation"] && duplicate["phase"] == "running" && afterDuplicate["execution_lock_held"] == true && afterDuplicate["generation"] == running["generation"], "active duplicate preserves exact blocked worker and generation")
	if failCleanup {
		panic(cleanupProbe{})
	}
	if kill {
		// The private provider marker is written by this invocation's blocked native
		// process. Its still-held job lock and unchanged marker bind this test kill.
		must(syscall.Kill(pid, syscall.SIGKILL))
	} else {
		a["expected_generation"] = running["generation"]
		a["request_id"] = "cooperative-cancel"
		a["reason"] = "qualification caller cancellation"
		cancellation := r.call("job_cancel", a)
		check(cancellation["cancellation_requested"] == true, "running cancellation is durable")
		delete(a, "expected_generation")
		delete(a, "request_id")
		delete(a, "reason")
	}
	select {
	case <-done:
		finished = true
	case <-time.After(25 * time.Second):
		panic("worker did not finish after selected interruption")
	}
	after := r.call("job_inspect", a)
	if kill {
		check(after["execution_lock_held"] == false && after["admitted"] == true, "killed worker leaves admitted orphan claim")
		a["expected_generation"] = after["generation"]
		unknown := r.call("job_reconcile", a)
		check(unknown["phase"] == "unknown" && unknown["result_receipt"] == nil && unknown["automatic_retry"] == false, "orphan reconciles unknown without adopting output")
		delete(a, "expected_generation")
		before := snapshotFiles(a["directory"].(string))
		again := r.call("job_execute", a)
		check(again["phase"] == "unknown" && again["generation"] == unknown["generation"] && reflect.DeepEqual(before, snapshotFiles(a["directory"].(string))), "unknown execute cannot restart or rewrite journal")
	} else {
		check(after["phase"] == "cancelled" && after["cancellation_observed"] == true, "cooperative worker cancellation is observed terminal state")
	}
	check(absent(path(name+"-result.json")), name+" publishes no result")
}
func checkFailureCleanup() {
	observed := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(cleanupProbe); !ok {
					panic(r)
				}
				observed = true
			}
		}()
		barrier("cleanup-failure", false, true)
	}()
	check(observed, "deliberate assertion failure exercises cleanup path")
	check(read(path("cleanup-failure-cleanup.json"))["controller_reaped"] == true, "failure cleanup reaps controller")
	raw, err := os.ReadFile(path("cleanup-failure-pid"))
	must(err)
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	must(err)
	check(syscall.Kill(pid, 0) == syscall.ESRCH, "failure cleanup leaves no blocked native worker")
}

func queuedCancel() {
	a, q := createJob("queued-cancel", nil)
	a["expected_generation"] = q["generation"]
	a["request_id"] = "queued-cancel"
	a["reason"] = "qualification caller cancellation"
	r := nativeJob("job_cancel", a)
	check(r["phase"] == "cancelled", "queued cancellation is terminal")
	delete(a, "expected_generation")
	delete(a, "request_id")
	delete(a, "reason")
	r = nativeJob("job_execute", a)
	check(r["phase"] == "cancelled" && absent(path("queued-cancel-result.json")), "cancelled queued work is never executed")
}
func dropReply() int {
	real := os.Getenv("SBV_QUALIFICATION_REAL_ADAPTER")
	out := os.Getenv("SBV_QUALIFICATION_DROPPED_REPLY")
	if !filepath.IsAbs(real) || !filepath.IsAbs(out) {
		return 71
	}
	cmd := exec.Command(real, os.Args[1:]...)
	cmd.Stdin = os.Stdin
	b, e := cmd.Output()
	if e != nil {
		return 72
	}
	// Deliberately no stdout: the production controller must preserve uncertainty.
	if e = os.WriteFile(out, b, 0600); e != nil {
		return 74
	}
	return 73
}
func main() {
	if len(os.Args) > 1 && os.Args[1] == "--target" {
		os.Exit(dropReply())
	}
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: qualify PREFIX WORK")
		os.Exit(2)
	}
	prefix, work = os.Args[1], os.Args[2]
	qxctl = filepath.Join(work, "bin", "qxctl")
	adapter = filepath.Join(work, "bin", "sov-stdio-adapter")
	self = filepath.Join(work, "bin", "qualify")
	failed := false
	defer func() {
		if r := recover(); r != nil {
			evidence["status"] = "failed"
			evidence["failure"] = fmt.Sprint(r)
			fmt.Fprintln(os.Stderr, r)
			failed = true
		}
		evidence["checks"] = checks
		evidence["check_count"] = len(checks)
		evidence["captured_at"] = time.Now().UTC().Format(time.RFC3339)
		write(filepath.Join(work, "evidence.json"), evidence)
		if failed {
			os.Exit(1)
		}
	}()
	evidence["host"] = object{"os": runtime.GOOS, "architecture": runtime.GOARCH, "uid": os.Getuid()}
	evidence["executables"] = object{"qxctl": digest(qxctl), "sov_stdio_adapter": digest(adapter), "native": digest(filepath.Join(prefix, "libexec/symphony/sbv-engine/0.33.0-dev/symphony-sbv")), "fault_carrier": digest(self), "feedback_fixture": digest(filepath.Join(work, "fixtures/launch-feedback.so"))}
	evidence["receipt"] = read(filepath.Join(prefix, "share/symphony/receipts/sbv-engine/0.33.0-dev/install-receipt.json"))["receipt_digest"]
	evidence["harness_inputs"] = snapshotFiles(filepath.Join(work, "source"))
	evidence["module_cache_seed_used"] = os.Getenv("SBV_QUALIFICATION_MODULE_SEED") != ""
	evidence["go_dependency_verification"] = "go mod verify before and after isolated readonly builds"
	evidence["go_module_inputs"] = object{"qxctl_go_mod": digest(filepath.Join(work, "qxctl-source", "go.mod")), "qxctl_go_sum": digest(filepath.Join(work, "qxctl-source", "go.sum")), "adapter_go_mod": digest(filepath.Join(work, "adapter-source", "go.mod"))}
	evidence["profile_observation_files"] = []string{"outputs/host.txt", "outputs/os-release.txt", "outputs/compiler-version.txt", "outputs/cmake-version.txt", "outputs/go-version.txt", "outputs/source-commit.txt", "outputs/source-working-diff.txt"}
	census()
	queuedCancel()
	durableCompletion()
	barrier("running-cancel", false, false)
	barrier("worker-loss", true, false)
	checkFailureCleanup()
	evidence["status"] = "passed"
	fmt.Printf("SBV installed boundary: %d assertions passed; no skipped cases\n", len(checks))
}
