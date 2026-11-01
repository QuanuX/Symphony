package main

import (
	"strings"
	"testing"
)

func TestNativeEnrollmentOwnershipAdmission(t *testing.T) {
	for _, tc := range []struct {
		name, uid, gid, errorText string
	}{
		{"zero", "0", "0", "invalid TOPS ID"},
		{"ordinary", "501", "20", "invalid TOPS ID"},
		{"maximum usable", "4294967294", "4294967294", "invalid TOPS ID"},
		{"reserved UID", "4294967295", "20", "is reserved"},
		{"reserved GID", "501", "4294967295", "is reserved"},
		{"UID overflow", "4294967296", "20", "invalid authority UID"},
		{"GID overflow", "501", "4294967296", "invalid authority GID"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// An invalid TOPS selector stops admitted identities before any path
			// resolution or mutation. No real system enrollment is attempted.
			err := runEnroll([]string{"--scope", "system", "--tops-id", "not-a-tops-id", "--authority-uid", tc.uid, "--authority-gid", tc.gid})
			if err == nil || !strings.Contains(err.Error(), tc.errorText) {
				t.Fatalf("unexpected identity admission: got %v, want %q", err, tc.errorText)
			}
		})
	}
}
