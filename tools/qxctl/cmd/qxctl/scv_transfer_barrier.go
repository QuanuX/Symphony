//go:build !symphony_transfer_fault_test

package main

// Production has no environment-controlled interruption behavior.
func scvTransferBarrier(string) error { return nil }
