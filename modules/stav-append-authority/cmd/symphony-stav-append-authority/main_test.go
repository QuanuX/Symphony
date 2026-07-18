package main

import (
	"strings"
	"testing"
)

const commandTestTOPSID = "01234567-89ab-4def-8123-456789abcdef"

func TestSystemEnrollmentRequiresExplicitAuthorityIdentity(t *testing.T) {
	err := runEnroll([]string{"--scope", "system", "--tops-id", commandTestTOPSID})
	if err == nil || !strings.Contains(err.Error(), "requires explicit --authority-uid and --authority-gid") {
		t.Fatalf("expected explicit system identity error, got %v", err)
	}
}

func TestEnrollmentRequiresAuthorityIdentityPair(t *testing.T) {
	err := runEnroll([]string{"--scope", "system", "--tops-id", commandTestTOPSID, "--authority-uid", "123"})
	if err == nil || !strings.Contains(err.Error(), "must be supplied together") {
		t.Fatalf("expected authority identity pair error, got %v", err)
	}
}

func TestUserEnrollmentRejectsAuthorityIdentityOverride(t *testing.T) {
	err := runEnroll([]string{
		"--scope", "user",
		"--tops-id", commandTestTOPSID,
		"--authority-uid", "123",
		"--authority-gid", "456",
	})
	if err == nil || !strings.Contains(err.Error(), "does not accept an override") {
		t.Fatalf("expected user identity override error, got %v", err)
	}
}

func TestSystemEnrollmentRejectsInvalidAuthorityIdentity(t *testing.T) {
	err := runEnroll([]string{
		"--scope", "system",
		"--tops-id", commandTestTOPSID,
		"--authority-uid", "not-a-number",
		"--authority-gid", "456",
	})
	if err == nil || !strings.Contains(err.Error(), "invalid authority UID") {
		t.Fatalf("expected invalid authority UID error, got %v", err)
	}
}
