package main

import (
	"strings"
	"testing"
)

const stavTestTOPSID = "01234567-89ab-4def-8123-456789abcdef"

func TestSTAVReadCommandsRequireEnrollment(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for _, command := range []string{"status", "verify", "query", "doctor"} {
		err := executeCommand([]string{"stav", command, "--tops-id", stavTestTOPSID})
		if err == nil || !strings.Contains(err.Error(), "STAV config") {
			t.Fatalf("%s enrollment error = %v", command, err)
		}
	}
}

func TestSTAVAppendIsProhibited(t *testing.T) {
	err := executeCommand([]string{"stav", "append", "--tops-id", stavTestTOPSID})
	if err == nil || !strings.Contains(err.Error(), "prohibited") {
		t.Fatalf("append error = %v", err)
	}
}

func TestSTAVRequiresCanonicalTOPSID(t *testing.T) {
	if err := executeCommand([]string{"stav", "status"}); err == nil || !strings.Contains(err.Error(), "--tops-id is required") {
		t.Fatalf("missing TOPS ID error = %v", err)
	}
	if err := executeCommand([]string{"stav", "status", "--tops-id", "INVALID"}); err == nil || !strings.Contains(err.Error(), "invalid TOPS ID") {
		t.Fatalf("invalid TOPS ID error = %v", err)
	}
}

func TestSTAVRejectsUnknownQueryFilters(t *testing.T) {
	err := executeCommand([]string{"stav", "query", "--tops-id", stavTestTOPSID, "--actor", "ssiag"})
	if err == nil {
		t.Fatal("unratified query filter unexpectedly accepted")
	}
}

func TestSTAVQueryAcceptsRatifiedScopeAndBoundedFiltersBeforeConnection(t *testing.T) {
	err := executeCommand([]string{
		"stav", "query", "--tops-id", stavTestTOPSID, "--scope", "system",
		"--after-sequence", "4", "--through-sequence", "9",
		"--event-class", "symphony.stav.fixture.event",
		"--outcome", "allowed", "--limit", "10",
	})
	if err == nil || !strings.Contains(err.Error(), "STAV config") {
		t.Fatalf("query enrollment error = %v", err)
	}
}

func TestSTAVQueryRejectsUnsafeInteger(t *testing.T) {
	err := executeCommand([]string{"stav", "query", "--tops-id", stavTestTOPSID, "--after-sequence", "9007199254740992"})
	if err == nil || !strings.Contains(err.Error(), "safe range") {
		t.Fatalf("unsafe integer error = %v", err)
	}
}
