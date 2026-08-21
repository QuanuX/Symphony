package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
)

func TestAccordareGrantIsClosedIdempotentAndRemovable(t *testing.T) {
	cfg := testSTAVGrantConfig()
	desired := stavprotocol.ProducerGrant{
		UID: 101, GID: 102,
		Producer:    stavprotocol.SafeReference{ID: accordareProducerID, Kind: "symphony.stav.producer"},
		Subject:     stavprotocol.SafeReference{ID: "symphony.accordare.stav.producer", Kind: "symphony.identity.service"},
		Permissions: append([]stavprotocol.PeerPermission(nil), accordarePermissions...),
	}
	installed, changed, err := setAccordareGrant(cfg, desired, true)
	if err != nil || !changed || len(installed.Authentication.Producers) != 1 {
		t.Fatalf("install exact grant failed: changed=%t err=%v config=%+v", changed, err, installed)
	}
	for index, permission := range installed.Authentication.Producers[0].Permissions {
		if permission != accordarePermissions[index] {
			t.Fatalf("permission escaped closed vocabulary: %+v", permission)
		}
	}
	if _, changed, err := setAccordareGrant(installed, desired, true); err != nil || changed {
		t.Fatalf("exact replay was not idempotent: changed=%t err=%v", changed, err)
	}
	drifted := desired
	drifted.Permissions = append(drifted.Permissions, stavprotocol.PeerPermission{EventClass: "symphony.other.event", OperationID: "symphony.other.operation"})
	if _, _, err := setAccordareGrant(installed, drifted, true); err == nil {
		t.Fatal("differing replacement bypassed explicit removal")
	}
	removed, changed, err := setAccordareGrant(installed, desired, false)
	if err != nil || !changed || len(removed.Authentication.Producers) != 0 {
		t.Fatalf("remove exact grant failed: changed=%t err=%v", changed, err)
	}
}

func TestGrantAttemptRecoveryAcceptsOnlyRecordedSide(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "attempt.json")
	previous := "sha256:" + strings.Repeat("1", 64)
	next := "sha256:" + strings.Repeat("2", 64)
	attempt := grantAttempt{Schema: "symphony.stav.grant-attempt.v1", Operation: "install", OperationID: "grant-test", TOPSID: ssiagTestTOPSID, PreviousConfigDigest: previous, NewConfigDigest: next}
	data, _ := json.Marshal(attempt)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := recoverGrantAttempt(path, ssiagTestTOPSID, next); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatal("closed recovery marker remained visible")
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := recoverGrantAttempt(path, ssiagTestTOPSID, "sha256:"+strings.Repeat("3", 64)); err == nil {
		t.Fatal("unrelated configuration state was treated as recoverable")
	}
}

func testSTAVGrantConfig() stavprotocol.AppendAuthorityConfig {
	return stavprotocol.AppendAuthorityConfig{
		Schema: stavprotocol.SchemaAppendAuthorityConfig, TOPSID: ssiagTestTOPSID, Mode: "user",
		Listen:         stavprotocol.AppendAuthorityListen{Network: "unix", Address: "/tmp/stav-test.sock"},
		Ledger:         stavprotocol.AppendAuthorityLedger{Durability: "fsync-before-receipt", MaxBytes: 1 << 20, Path: "/tmp/stav-test.log", Recovery: "preserve-incomplete-tail", Retention: "preserve_all", Rotation: "disabled"},
		Authentication: stavprotocol.AppendAuthorityAuthentication{Mechanism: "kernel-peer-credentials", Authority: stavprotocol.AuthorityGrant{UID: 1, GID: 1, Subject: stavprotocol.SafeReference{ID: "stav-authority", Kind: "symphony.identity.service"}}, Producers: []stavprotocol.ProducerGrant{}, Readers: []stavprotocol.ReaderGrant{}},
	}
}
