//go:build !symphony_snv_faults || (!darwin && !linux)

package snvstate

// Fault barriers are absent from production builds.
func barrier(string) {}
