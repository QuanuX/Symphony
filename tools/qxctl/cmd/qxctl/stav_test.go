package main

import (
	"strings"
	"testing"
)

const stavTestTOPSID = "01234567-89ab-4def-8123-456789abcdef"

func TestSTAVReadCommandsRemainGated(t *testing.T) {
	for _, command := range []string{"status", "verify", "query", "doctor"} {
		err := runSTAV([]string{command, "--tops-id", stavTestTOPSID})
		if err == nil || !strings.Contains(err.Error(), "reserved but unavailable") {
			t.Fatalf("%s gate error = %v", command, err)
		}
	}
}

func TestSTAVAppendIsProhibited(t *testing.T) {
	err := runSTAV([]string{"append", "--tops-id", stavTestTOPSID})
	if err == nil || !strings.Contains(err.Error(), "prohibited") {
		t.Fatalf("append error = %v", err)
	}
}

func TestSTAVRequiresCanonicalTOPSID(t *testing.T) {
	if err := runSTAV([]string{"status"}); err == nil || !strings.Contains(err.Error(), "--tops-id is required") {
		t.Fatalf("missing TOPS ID error = %v", err)
	}
	if err := runSTAV([]string{"status", "--tops-id", "INVALID"}); err == nil || !strings.Contains(err.Error(), "invalid TOPS ID") {
		t.Fatalf("invalid TOPS ID error = %v", err)
	}
}

func TestSTAVRejectsUnknownQueryFilters(t *testing.T) {
	err := runSTAV([]string{"query", "--tops-id", stavTestTOPSID, "--actor", "ssiag"})
	if err == nil {
		t.Fatal("unratified query filter unexpectedly accepted")
	}
}

func TestSTAVQueryAcceptsRatifiedScopeAndBoundedFiltersBeforeGate(t *testing.T) {
	err := runSTAV([]string{
		"query", "--tops-id", stavTestTOPSID, "--scope", "system",
		"--after-sequence", "4", "--through-sequence", "9",
		"--event-class", "symphony.stav.fixture.event",
		"--outcome", "allowed", "--limit", "10",
	})
	if err == nil || !strings.Contains(err.Error(), "reserved but unavailable") {
		t.Fatalf("query gate error = %v", err)
	}
}

func TestSTAVQueryRejectsUnsafeInteger(t *testing.T) {
	err := runSTAV([]string{"query", "--tops-id", stavTestTOPSID, "--after-sequence", "9007199254740992"})
	if err == nil || !strings.Contains(err.Error(), "safe range") {
		t.Fatalf("unsafe integer error = %v", err)
	}
}
