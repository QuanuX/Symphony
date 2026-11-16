package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeDispatchRejectsIncompleteDescriptorsBeforeMutation(t *testing.T) {
	for _, operation := range []string{"install", "uninstall"} {
		for _, flags := range [][]string{{"--lifecycle-root-fd", "3"}, {"--lifecycle-root-fd", "3", "--lifecycle-lock-fd", "3", "--lifecycle-proof-fd", "5"}} {
			t.Run(operation+strings.Join(flags, "_"), func(t *testing.T) {
				prefix := filepath.Join(t.TempDir(), "absent-prefix")
				args := append([]string{"--prefix", prefix}, flags...)
				err := runPackage(append([]string{operation}, args...))
				if err == nil || !strings.Contains(err.Error(), "descriptor flags must be supplied together and distinct") {
					t.Fatalf("invalid delegation reached owner mutation: %v", err)
				}
				if _, err := os.Stat(prefix); !os.IsNotExist(err) {
					t.Fatalf("invalid delegation created installation root: %v", err)
				}
			})
		}
	}
}
