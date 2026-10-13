//go:build darwin || linux

package sovremote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
)

func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "--target" {
		raw, _ := io.ReadAll(os.Stdin)
		in, e := DecodeInvocation(raw)
		if e != nil {
			os.Exit(2)
		}
		digest, _ := InvocationDigest(in)
		code := "native_operation_failed"
		r := TargetResult{Protocol: "symphony.sov.target-result.v1", RequestID: in.RequestID, InvocationSHA256: digest, ConfigurationSHA256: in.Endpoint.ConfigurationSHA256, Target: in.Selection.Target, AuthorityMode: "owner_filesystem", Operation: in.Operation, DispatchStarted: true, Status: "native_error", ErrorCode: &code, Result: json.RawMessage("null")}
		switch os.Getenv("SOV_TEST_RESPONSE") {
		case "ignore_cancel":
			signal.Ignore(syscall.SIGTERM)
			_ = os.WriteFile(os.Getenv("SOV_TEST_READY"), []byte(strconv.Itoa(os.Getpid())), 0600)
			for {
				time.Sleep(time.Hour)
			}
		case "drop":
			os.Exit(0)
		case "mismatch":
			r.RequestID = "00000000-0000-0000-0000-000000000000"
		case "oversize":
			fmt.Print(strings.Repeat("x", int(MaxDocumentBytes)+1))
			os.Exit(0)
		}
		_ = json.NewEncoder(os.Stdout).Encode(r)
		os.Exit(0)
	}
	os.Exit(m.Run())
}
func testInvocation(t *testing.T) Invocation {
	t.Helper()
	v, e := Template("invoke")
	if e != nil {
		t.Fatal(e)
	}
	return v.(Invocation)
}
func replan(t *testing.T, in *Invocation) {
	t.Helper()
	p, e := build(in.Selection)
	if e != nil {
		t.Fatal(e)
	}
	in.ExpectedPlanSHA256 = p.PlanSHA256
}
func privateDir(t *testing.T) string {
	t.Helper()
	p, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	if e = os.Chmod(p, 0700); e != nil {
		t.Fatal(e)
	}
	return p
}
func testConfiguration(t *testing.T) (ConfigureInput, ConfigurationResult) {
	t.Helper()
	v, _ := Template("target_configure")
	in := v.(ConfigureInput)
	in.Path = filepath.Join(privateDir(t), "configuration.json")
	out, e := Configure(encoded(t, in))
	if e != nil {
		t.Fatal(e)
	}
	return in, out
}
func TestInvocationSelectionAndDigest(t *testing.T) {
	in := testInvocation(t)
	if _, e := DecodeInvocation(encoded(t, in)); e != nil {
		t.Fatal(e)
	}
	a, _ := InvocationDigest(in)
	var request map[string]any
	_ = json.Unmarshal(in.Request, &request)
	pretty, _ := json.MarshalIndent(request, "", " ")
	in.Request = pretty
	b, _ := InvocationDigest(in)
	if a != b {
		t.Fatal("digest depends on whitespace")
	}
	for name, mutate := range map[string]func(*Invocation){"plan": func(x *Invocation) { x.ExpectedPlanSHA256 = strings.Repeat("a", 64) }, "adapter": func(x *Invocation) { x.Adapter.SHA256 = strings.Repeat("a", 64) }, "target": func(x *Invocation) { x.Endpoint.TargetQXCTL.SHA256 = strings.Repeat("a", 64) }, "operation": func(x *Invocation) { x.Operation = "run" }, "request": func(x *Invocation) { x.Request = json.RawMessage("null") }, "uuid": func(x *Invocation) { x.RequestID = "bad" }, "placement": func(x *Invocation) { x.Selection.Resources.Threads = ptr("4"); replan(t, x) }, "past_deadline": func(x *Invocation) { x.Selection.Resources.DeadlineUnixMS = ptr("1"); replan(t, x) }, "arrangement": func(x *Invocation) { x.Selection.Transport.Arrangement = "staged"; replan(t, x) }} {
		t.Run(name, func(t *testing.T) {
			v := testInvocation(t)
			mutate(&v)
			if _, e := DecodeInvocation(encoded(t, v)); e == nil {
				t.Fatal("bad invocation admitted")
			}
		})
	}
	ctx, cancel := withDeadline(context.Background(), nil)
	defer cancel()
	if _, ok := ctx.Deadline(); ok {
		t.Fatal("implicit deadline")
	}
	future := fmt.Sprint(time.Now().Add(time.Hour).UnixMilli())
	ctx, cancel = withDeadline(context.Background(), &future)
	defer cancel()
	if _, ok := ctx.Deadline(); !ok {
		t.Fatal("selected deadline lost")
	}
}
func TestConfigurationCASAndAuthority(t *testing.T) {
	in, out := testConfiguration(t)
	if _, e := Configure(encoded(t, in)); e == nil {
		t.Fatal("overwrote without CAS")
	}
	in.ExpectedSHA256 = ptr(strings.Repeat("a", 64))
	if _, e := Configure(encoded(t, in)); e == nil {
		t.Fatal("stale CAS")
	}
	in.ExpectedSHA256 = &out.SHA256
	in.Configuration.AuthorityMode = "ssiag"
	if _, e := Configure(encoded(t, in)); e == nil {
		t.Fatal("implicit authorization freshness")
	}
	in.Configuration.AuthorizationWindowSeconds = ptr("60")
	next, e := Configure(encoded(t, in))
	if e != nil || next.SHA256 == out.SHA256 {
		t.Fatal(e)
	}
	if _, _, e := ReadConfiguration(in.Path, out.SHA256); e == nil {
		t.Fatal("stale read")
	}
	if e := RemoveConfiguration(in.Path, out.SHA256); e == nil {
		t.Fatal("stale removal")
	}
	if e := RemoveConfiguration(in.Path, next.SHA256); e != nil {
		t.Fatal(e)
	}
	if _, _, e := ReadConfiguration(in.Path, ""); !os.IsNotExist(e) {
		t.Fatal("configuration retained", e)
	}
	if _, e := os.Stat(in.Path + ".lock"); e != nil {
		t.Fatal("stable lock missing")
	}
}
func TestConfigurationRejectsUnsafeFiles(t *testing.T) {
	for _, kind := range []string{"public_file", "symlink", "hardlink", "public_parent", "parent_symlink"} {
		t.Run(kind, func(t *testing.T) {
			in, out := testConfiguration(t)
			path := in.Path
			switch kind {
			case "public_file":
				_ = os.Chmod(path, 0644)
			case "symlink":
				_ = os.Rename(path, path+".real")
				_ = os.Symlink(path+".real", path)
			case "hardlink":
				_ = os.Link(path, path+".link")
			case "public_parent":
				_ = os.Chmod(filepath.Dir(path), 0755)
			case "parent_symlink":
				link := filepath.Join(privateDir(t), "link")
				_ = os.Symlink(filepath.Dir(path), link)
				path = filepath.Join(link, filepath.Base(path))
			}
			if _, _, e := ReadConfiguration(path, out.SHA256); e == nil {
				t.Fatal("unsafe file accepted")
			}
		})
	}
}
func TestOneShotProcessOutcomes(t *testing.T) {
	self, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(self)
	if e != nil {
		t.Fatal(e)
	}
	in := testInvocation(t)
	in.Adapter = Executable{self, Digest(b)}
	in.Selection.Transport.AdapterSHA256 = in.Adapter.SHA256
	replan(t, &in)
	for _, mode := range []string{"valid", "drop", "mismatch", "oversize"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("SOV_TEST_RESPONSE", mode)
			out, e := Run(context.Background(), encoded(t, in))
			if e != nil {
				t.Fatal(e)
			}
			expected := "response_lost"
			if mode == "valid" {
				expected = "response_received"
			}
			if out.TransportOutcome != expected || out.AutomaticRetry {
				t.Fatal(out)
			}
		})
	}
	in.Adapter.SHA256 = strings.Repeat("f", 64)
	in.Selection.Transport.AdapterSHA256 = in.Adapter.SHA256
	replan(t, &in)
	out, e := Run(context.Background(), encoded(t, in))
	if e == nil || out.TransportOutcome != "not_dispatched" {
		t.Fatal("unverified bytes executed")
	}
}
func TestTargetInstalledAuthorityAndBinding(t *testing.T) {
	prefix := os.Getenv("SYMPHONY_SBV_TEST_PREFIX")
	if prefix == "" {
		t.Skip("select installed native SBV")
	}
	root := privateDir(t)
	directory := filepath.Join(root, "job")
	if e := os.Mkdir(directory, 0700); e != nil {
		t.Fatal(e)
	}
	_, raw, e := knowledgeengine.SBVTemplateVariant(prefix, knowledgeengine.SBVAdministrationInterfaceVersion, "job_create", "")
	if e != nil {
		t.Fatal(e)
	}
	var create map[string]any
	_ = json.Unmarshal(raw, &create)
	create["directory"] = directory
	create["request"].(map[string]any)["output_path"] = filepath.Join(root, "result.json")
	queued, e := knowledgeengine.InvokeSBV(context.Background(), prefix, knowledgeengine.SBVAdministrationInterfaceVersion, root, "job_create", encoded(t, create))
	if e != nil {
		t.Fatal(e)
	}
	var state map[string]any
	_ = json.Unmarshal(queued.Result, &state)
	in := testInvocation(t)
	in.Selection.Job = Job{directory, create["job_id"].(string), state["plan_sha256"].(string)}
	self, _ := os.Executable()
	selfBytes, _ := os.ReadFile(self)
	in.Selection.Runtime.QXCTLSHA256 = Digest(selfBytes)
	in.Endpoint.TargetQXCTL = Executable{self, Digest(selfBytes)}
	inst, e := knowledgeengine.InspectSBV(prefix, knowledgeengine.SBVAdministrationInterfaceVersion)
	if e != nil {
		t.Fatal(e)
	}
	in.Selection.Runtime.SBVInstallationSHA256 = strings.TrimPrefix(inst.ReceiptDigest, "sha256:")
	c := Configuration{Protocol: "symphony.sov.target-configuration.v1", Target: in.Selection.Target, Job: in.Selection.Job, Runtime: in.Selection.Runtime, SBVPrefix: prefix, AuthorityMode: "ssiag", Authority: in.Selection.Authority, AuthorizationWindowSeconds: ptr("60"), AllowedArtifacts: []Artifact{}}
	config := ConfigureInput{Protocol: "symphony.sov.target-configure-input.v1", Path: filepath.Join(root, "configuration.json"), Configuration: c}
	configured, e := Configure(encoded(t, config))
	if e != nil {
		t.Fatal(e)
	}
	in.Endpoint.ConfigurationPath = config.Path
	in.Endpoint.ConfigurationSHA256 = configured.SHA256
	in.Operation = "job_execute"
	in.Request = encoded(t, map[string]any{"protocol": "symphony.sbv.job-execute-input.v1", "directory": directory, "job_id": c.Job.ID, "expected_plan_sha256": c.Job.PlanSHA256})
	replan(t, &in)
	denied := 0
	deny := func(context.Context, Configuration, Invocation) (string, error) {
		denied++
		return "", fmt.Errorf("denied")
	}
	if _, e := Serve(context.Background(), encoded(t, in), config.Path, configured.SHA256, c.Runtime.QXCTLVersion, deny); e == nil || denied != 1 {
		t.Fatal("authority bypass", e)
	}
	if _, e := os.Stat(filepath.Join(directory, "claim.json")); !os.IsNotExist(e) {
		t.Fatal("denied execution claimed job")
	}
	allow := func(context.Context, Configuration, Invocation) (string, error) { return strings.Repeat("c", 64), nil }
	out, e := Serve(context.Background(), encoded(t, in), config.Path, configured.SHA256, c.Runtime.QXCTLVersion, allow)
	if e != nil {
		t.Fatal(e)
	}
	digest, _ := InvocationDigest(in)
	if out.Status != "completed" {
		t.Fatal("native dispatch failed", out)
	}
	if e = validateTargetResult(in, digest, out); e != nil {
		t.Fatal(e)
	}
	again, e := Serve(context.Background(), encoded(t, in), config.Path, configured.SHA256, c.Runtime.QXCTLVersion, allow)
	if e != nil || !bytes.Equal(out.Result, again.Result) {
		t.Fatal("duplicate execution altered result", e)
	}
	in.Selection.Job.ID = "retargeted"
	replan(t, &in)
	if _, e := Serve(context.Background(), encoded(t, in), config.Path, configured.SHA256, c.Runtime.QXCTLVersion, allow); e == nil {
		t.Fatal("target binding lost")
	}
}

func TestProcessCancellationReturnsAmbiguousOutcome(t *testing.T) {
	self, _ := os.Executable()
	b, _ := os.ReadFile(self)
	in := testInvocation(t)
	in.Adapter = Executable{self, Digest(b)}
	in.Selection.Transport.AdapterSHA256 = in.Adapter.SHA256
	replan(t, &in)
	ready := filepath.Join(privateDir(t), "ready")
	t.Setenv("SOV_TEST_RESPONSE", "ignore_cancel")
	t.Setenv("SOV_TEST_READY", ready)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan InvocationResult, 1)
	go func() { out, _ := Run(ctx, encoded(t, in)); done <- out }()
	end := time.Now().Add(10 * time.Second)
	for {
		if _, e := os.Stat(ready); e == nil {
			break
		}
		if time.Now().After(end) {
			t.Fatal("helper did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	pidBytes, _ := os.ReadFile(ready)
	pid, _ := strconv.Atoi(string(pidBytes))
	defer syscall.Kill(-pid, syscall.SIGKILL)
	cancel()
	select {
	case out := <-done:
		if out.TransportOutcome != "response_lost" || out.AutomaticRetry {
			t.Fatal(out)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("cancellation blocked behind selected adapter")
	}
}
func TestMalformedRemoteNativePairCannotPanic(t *testing.T) {
	for _, op := range []string{"job_inspect", "job_execute", "job_cancel", "job_reconcile", "result_inspect", "result_query", "bundle_inspect", "bundle_query", "bundle_verify"} {
		t.Run(op, func(t *testing.T) {
			for _, raw := range []string{`{}`, `{"protocol":"symphony.sbv.result-query-input.v1","pointer":"","expected_sha256":"` + strings.Repeat("a", 64) + `"}`} {
				if e := knowledgeengine.ValidateSBVOperationResult(op, []byte(raw), []byte(`{"protocol":"symphony.sbv.result-query.v1","engine_version":"0.33.0-dev","pointer":"","content_sha256":"`+strings.Repeat("a", 64)+`","query_sha256":"`+strings.Repeat("a", 64)+`","offset":"0","total":"0","nodes":[],"complete":true,"next_cursor":""}`)); e == nil {
					t.Fatal("malformed pair accepted")
				}
			}
		})
	}
}

func TestFIFORefusedWithoutBlocking(t *testing.T) {
	dir := privateDir(t)
	path := filepath.Join(dir, "fifo")
	if e := syscall.Mkfifo(path, 0600); e != nil {
		t.Fatal(e)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, _, e := ReadConfiguration(path, ""); e == nil {
			t.Error("FIFO configuration accepted")
		}
		if _, cleanup, e := StageExecutable(Executable{path, strings.Repeat("a", 64)}); e == nil {
			cleanup()
			t.Error("FIFO executable accepted")
		}
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("special file blocked inspection")
	}
}
