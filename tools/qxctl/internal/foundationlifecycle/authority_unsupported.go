//go:build !darwin && !linux

package foundationlifecycle

func hasSystemAuthority() bool { return false }

func effectiveIdentity() (uint32, uint32) { return 0, 0 }

func foundationPlatformSupported() bool { return false }
