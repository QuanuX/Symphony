package main

import (
	"strings"
	"testing"
)

func TestNativeEnrollmentRejectsReservedOwnership(t *testing.T) {
	for _, tc := range []struct {
		name, uid, gid, errorText string
	}{
		{"reserved UID", "4294967295", "20", "is reserved"},
		{"reserved GID", "501", "4294967295", "is reserved"},
		{"UID overflow", "4294967296", "20", "invalid service-uid"},
		{"GID overflow", "501", "4294967296", "invalid service-gid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := runEnroll([]string{"--scope", "system", "--tops-id", "01234567-89ab-4def-8123-456789abcdef", "--service-uid", tc.uid, "--service-gid", tc.gid})
			if err == nil || !strings.Contains(err.Error(), tc.errorText) {
				t.Fatalf("unsafe identity reached lifecycle observation: %v", err)
			}
		})
	}
}
