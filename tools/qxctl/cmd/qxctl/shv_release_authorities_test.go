//go:build darwin || linux

package main

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	stav "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	stavclient "github.com/QuanuX/Symphony/modules/stav-append-authority/client"
)

// These are real private development-mode owner processes, with an exact
// synthetic policy grant and no provider, supervisor or installation mutation.
func serveSHVQualificationAuthorities(t *testing.T, evidence, resource string) func(string, string) any {
	t.Helper()
	// Short sockets remain below sockaddr_un bounds even with long CI work paths.
	root, err := os.MkdirTemp("/tmp", "shv-auth-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(root, "run"))
	t.Setenv("SYMPHONY_SSIAG_CONFIG", "")
	t.Setenv("SYMPHONY_SSIAG_SOCKET", filepath.Join(root, "ssiag.sock"))
	uid, gid := uint64(os.Geteuid()), uint64(os.Getegid())
	cfg := stav.AppendAuthorityConfig{
		Schema: stav.SchemaAppendAuthorityConfig, TOPSID: ssiagTestTOPSID, Mode: "user",
		Listen: stav.AppendAuthorityListen{Network: "unix", Address: filepath.Join(root, "stav.sock")},
		Ledger: stav.AppendAuthorityLedger{Path: filepath.Join(root, "audit", "ledger"), MaxBytes: 1048576, Durability: "fsync-before-receipt", Recovery: "preserve-incomplete-tail", Retention: "preserve_all", Rotation: "disabled"},
		Authentication: stav.AppendAuthorityAuthentication{Mechanism: "kernel-peer-credentials",
			Authority: stav.AuthorityGrant{UID: uid, GID: gid, Subject: stav.SafeReference{ID: "qualification-stav", Kind: "symphony.identity.service"}},
			Producers: []stav.ProducerGrant{{UID: uid, GID: gid, Subject: stav.SafeReference{ID: "qualification-ssiag", Kind: "symphony.identity.service"}, Producer: stav.SafeReference{ID: "qualification-ssiag", Kind: "symphony.stav.producer"}, Permissions: []stav.PeerPermission{{EventClass: "symphony.ssiag.policy.decision", OperationID: "symphony.ssiag.authorize"}}}},
			Readers:   []stav.ReaderGrant{{UID: uid, GID: gid, Subject: stav.SafeReference{ID: "qualification-reader", Kind: "symphony.identity.operator"}, Classifications: []string{"administrative_metadata"}}}}}
	encoded, err := stav.EncodeAppendAuthorityConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "stav.json")
	if err = os.WriteFile(configPath, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SYMPHONY_STAV_CONFIG", configPath)
	serviceConfig := map[string]any{
		"schema": "symphony.ssiag.config.v1", "mode": "user", "tops": map[string]any{"id": ssiagTestTOPSID, "name": "SHV private qualification"},
		"listen":         map[string]any{"network": "unix", "address": filepath.Join(os.Getenv("XDG_RUNTIME_DIR"), "symphony", ssiagTestTOPSID, "ssiag", "ssiag.sock")},
		"authentication": map[string]any{"mechanism": "unix_peer_credentials", "service": map[string]any{"id": "symphony.ssiag.service", "kind": "symphony.identity.service", "uid": uid, "gid": gid}, "subjects": []any{map[string]any{"id": "qualification-owner", "kind": "symphony.identity.owner", "uid": uid, "gid": gid}}},
		"authorization":  map[string]any{"default_effect": "deny", "max_capability_seconds": 900, "grants": []any{map[string]any{"id": "source-onboard", "subject_id": "qualification-owner", "authority_basis": "host_owner", "operation": "symphony.shv.source.onboard", "resource": resource, "audience": "qxctl", "scope": "tops:" + ssiagTestTOPSID}}}, "providers": []any{}}
	encoded, err = json.Marshal(serviceConfig)
	if err != nil {
		t.Fatal(err)
	}
	servicePath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "symphony", ssiagTestTOPSID, "ssiag", "config.json")
	if err = os.MkdirAll(filepath.Dir(servicePath), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(servicePath, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	start := func(name, binary, path, socket string) {
		t.Helper()
		if !filepath.IsAbs(binary) {
			t.Fatal("authority executable must be absolute")
		}
		log, err := os.OpenFile(filepath.Join(evidence, name+".log"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		command := exec.Command(binary, "serve", "--tops-id", ssiagTestTOPSID, "--scope", "user", "--config", path)
		command.Stdout = log
		command.Stderr = log
		if err = command.Start(); err != nil {
			log.Close()
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() { done <- command.Wait() }()
		exited := false
		t.Cleanup(func() {
			if !exited {
				_ = command.Process.Signal(syscall.SIGTERM)
				select {
				case err := <-done:
					if err != nil {
						t.Errorf("%s shutdown: %v", name, err)
					}
				case <-time.After(5 * time.Second):
					_ = command.Process.Kill()
					<-done
					t.Errorf("%s shutdown timeout", name)
				}
			}
			_ = log.Close()
		})
		for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
			select {
			case err := <-done:
				exited = true
				t.Fatalf("%s exited before readiness: %v (see evidence log)", name, err)
			default:
			}
			connection, err := net.DialTimeout("unix", socket, 50*time.Millisecond)
			if err == nil {
				connection.Close()
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("%s did not become ready", name)
	}
	start("stav", os.Getenv("SHV_RELEASE_STAV"), configPath, cfg.Listen.Address)
	start("ssiag", os.Getenv("SHV_RELEASE_SSIAG"), servicePath, os.Getenv("SYMPHONY_SSIAG_SOCKET"))
	client, err := stavclient.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return func(correlationID, requestID string) any {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		id, err := stav.GenerateUUIDv4()
		if err != nil {
			t.Fatal(err)
		}
		response, err := client.Do(ctx, stav.LocalRequest{Schema: stav.SchemaLocalRequest, TOPSID: ssiagTestTOPSID, RequestID: id, Operation: stav.LocalOperationQuery, Query: &stav.Query{Schema: stav.SchemaQuery, TOPSID: ssiagTestTOPSID, AfterSequence: 0, Limit: 10, EventClasses: []string{"symphony.ssiag.policy.decision"}, Outcomes: []string{"allowed"}, CorrelationID: correlationID, RequestID: requestID}})
		if err != nil || response.Page == nil || len(response.Page.Entries) != 1 {
			t.Fatalf("exact policy audit absent: %+v %v", response, err)
		}
		event := response.Page.Entries[0]
		if event.Projection.CorrelationID != correlationID || event.Projection.RequestID != requestID || event.Projection.Target.ID != resource || event.Projection.Outcome != "allowed" || event.VerificationState != "verified" {
			t.Fatalf("policy audit binding mismatch: %+v", event)
		}
		id, err = stav.GenerateUUIDv4()
		if err != nil {
			t.Fatal(err)
		}
		status, err := client.Do(ctx, stav.LocalRequest{Schema: stav.SchemaLocalRequest, TOPSID: ssiagTestTOPSID, RequestID: id, Operation: stav.LocalOperationStatus})
		if err != nil || status.Status == nil || status.Status.Events != 1 || !status.Status.Ready {
			t.Fatalf("replay duplicated policy audit: %+v %v", status, err)
		}
		return map[string]any{"policy_query": response, "authority_status": status, "source_write_receipt": nil, "mode": "private real user-scope development processes; no supervisor"}
	}
}
