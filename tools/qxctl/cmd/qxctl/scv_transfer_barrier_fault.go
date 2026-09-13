//go:build symphony_transfer_fault_test

package main

import (
	"os"
	"syscall"
)

// Test-only CLI. Never installed or represented as the production executable.
func scvTransferBarrier(phase string) error {
	if phase == os.Getenv("SYMPHONY_TRANSFER_TEST_STOP") {
		return syscall.Kill(os.Getpid(), syscall.SIGSTOP)
	}
	return nil
}
