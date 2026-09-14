//go:build !symphony_shv_transfer_fault_test

package main

// Production has no environment-controlled interruption behavior.
func shvTransferBarrier(string) error { return nil }
