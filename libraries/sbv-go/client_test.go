package sbv

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "sbv" && os.Getenv("SBV_ADAPTER_HELPER") == "1" {
		var input map[string]json.RawMessage
		for i, a := range os.Args {
			if a == "--input" {
				raw, _ := os.ReadFile(os.Args[i+1])
				_ = json.Unmarshal(raw, &input)
			}
		}
		switch str(input, "test") {
		case "reject":
			fmt.Print(`{"protocol":"symphony.qxctl.error.v1","outcome":"error","error":{"code":"engine_rejected"}}`)
			os.Exit(1)
		case "cancel":
			ch := make(chan os.Signal, 1)
			signal.Notify(ch, syscall.SIGTERM)
			_ = os.WriteFile(str(input, "started"), []byte("ready"), 0600)
			<-ch
			os.Exit(130)
		case "stream":
			raw, _ := fixture(false)
			_, _ = os.Stdout.Write(raw)
			os.Exit(0)
		default:
			_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"protocol": "test.qxctl.v1", "args": os.Args[1:], "request": input})
			os.Exit(0)
		}
	}
	os.Exit(m.Run())
}
func testClient(t *testing.T) *Client {
	t.Helper()
	t.Setenv("SBV_ADAPTER_HELPER", "1")
	exe, _ := os.Executable()
	raw, _ := os.ReadFile(exe)
	sha := sha256.Sum256(raw)
	dir, _ := filepath.EvalSymlinks(t.TempDir())
	_ = os.Chmod(dir, 0700)
	c, err := New(Config{Executable: exe, ExecutableSHA256: hex.EncodeToString(sha[:]), Prefix: dir, Version: NativeVersion, WorkDir: dir, RequestDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func TestClientAllOperationsAndResourceSelection(t *testing.T) {
	c := testClient(t)
	for _, op := range Operations() {
		args, err := route(op)
		if err != nil || args[0] != "sbv" {
			t.Fatal(op, err)
		}
	}
	raw, err := c.Invoke(context.Background(), "queue_scenarios", json.RawMessage(`{"test":"copy","value":"18446744073709551615"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "queue-scenarios") || !strings.Contains(string(raw), "18446744073709551615") {
		t.Fatal(string(raw))
	}
	if _, err = c.Resource(context.Background(), "evaluate", "template", "partitioned"); err != nil {
		t.Fatal(err)
	}
	if _, err = c.Invoke(context.Background(), "result_query", json.RawMessage(`{"protocol":"symphony.sbv.result-query-input.v1","path":"/caller/result","expected_sha256":"abc","pointer":"/a~1b","cursor":"","limit":"7"}`)); err != nil {
		t.Fatal(err)
	}
	files, _ := filepath.Glob(filepath.Join(c.config.RequestDir, "sbv-request-*"))
	if len(files) != 0 {
		t.Fatal("request files leaked", files)
	}
	if _, err = c.Invoke(context.Background(), "run; rm", json.RawMessage(`{}`)); err == nil {
		t.Fatal("unknown command")
	}
	if _, err = c.Invoke(context.Background(), "result_query", json.RawMessage(`{}`)); err == nil {
		t.Fatal("shape")
	}
}
func TestClientExactSelectionAndTypedFailure(t *testing.T) {
	c := testClient(t)
	config := c.config
	config.Version = "latest"
	if _, err := New(config); err == nil {
		t.Fatal("unversioned installation")
	}
	config = c.config
	config.ExecutableSHA256 = strings.Repeat("a", 64)
	if _, err := New(config); err == nil {
		t.Fatal("wrong executable identity")
	}
	_, err := c.Invoke(context.Background(), "evaluate", json.RawMessage(`{"test":"reject"}`))
	var failure *Error
	if !errors.As(err, &failure) || failure.Code != "process" || !strings.Contains(string(failure.Response), "engine_rejected") {
		t.Fatal(err)
	}
}
func TestClientCancellationReapsChild(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := filepath.Join(c.config.WorkDir, "started")
	p, _ := json.Marshal(map[string]string{"test": "cancel", "started": started})
	done := make(chan error, 1)
	go func() { _, err := c.Invoke(ctx, "experiment", p); done <- err }()
	limit := time.NewTimer(10 * time.Second)
	defer limit.Stop()
	tick := time.NewTicker(5 * time.Millisecond)
	defer tick.Stop()
wait:
	for {
		select {
		case <-limit.C:
			t.Fatal("test fixture failed to start")
		case <-tick.C:
			if _, err := os.Stat(started); err == nil {
				break wait
			}
		}
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal("cancelled cause not retained", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("child was not reaped")
	}
	files, _ := filepath.Glob(filepath.Join(c.config.RequestDir, "sbv-request-*"))
	if len(files) != 0 {
		t.Fatal("request leaked")
	}
}
