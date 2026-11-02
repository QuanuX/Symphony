package foundationaudit

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func TestEndpointConfigurationFIFORefusedBeforeDial(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	t.Setenv("XDG_RUNTIME_DIR", "")
	request := testRequest()
	path, _, err := ssiagPaths(request.Scope, request.TOPSID)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err = syscall.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := SubmitToOwner(context.Background(), request); done <- err }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("FIFO config admitted")
		}
	case <-time.After(time.Second):
		t.Fatal("configuration read blocked before type validation")
	}
}

func TestBridgeClientAuthenticatesKernelPeerBeforeHTTP(t *testing.T) {
	for _, mismatch := range []bool{false, true} {
		t.Run(map[bool]string{false: "exact-peer", true: "wrong-gid"}[mismatch], func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
			t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
			t.Setenv("XDG_RUNTIME_DIR", "")
			request := testRequest()
			path, canonicalSocket, err := ssiagPaths(request.Scope, request.TOPSID)
			if err != nil {
				t.Fatal(err)
			}
			directory, err := os.MkdirTemp("/tmp", "f-audit-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.RemoveAll(directory) })
			socket := filepath.Join(directory, "s")
			t.Setenv("SYMPHONY_SSIAG_SOCKET", socket)
			uid, gid := uint32(os.Geteuid()), uint32(os.Getegid())
			if mismatch {
				gid++
			}
			cfg := map[string]any{"schema": "symphony.ssiag.config.v1", "mode": "user", "tops": map[string]any{"id": request.TOPSID}, "listen": map[string]any{"network": "unix", "address": canonicalSocket}, "authentication": map[string]any{"mechanism": "unix_peer_credentials", "service": map[string]any{"id": "symphony.ssiag.service", "kind": "symphony.identity.service", "uid": uid, "gid": gid}}}
			data, _ := json.Marshal(cfg)
			if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			listener, err := net.Listen("unix", socket)
			if err != nil {
				t.Fatal(err)
			}
			var calls atomic.Int32
			server := &http.Server{ReadHeaderTimeout: time.Second, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				response := committed(t, request)
				data, _ := json.Marshal(response)
				w.Header().Set("Content-Type", "application/json")
				w.Write(data)
			})}
			done := make(chan error, 1)
			go func() { done <- server.Serve(listener) }()
			response, err := SubmitToOwner(context.Background(), request)
			server.Close()
			<-done
			if mismatch {
				if err == nil || calls.Load() != 0 {
					t.Fatalf("wrong kernel peer received HTTP: calls=%d err=%v", calls.Load(), err)
				}
				return
			}
			if err != nil || calls.Load() != 1 || response.Validate(request, uid) != nil {
				t.Fatalf("exact authenticated response rejected: %v", err)
			}
		})
	}
}
