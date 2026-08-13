//go:build darwin || linux

package foundationlifecycle

import "os"

func hasSystemAuthority() bool { return os.Geteuid() == 0 }

func effectiveIdentity() (uint32, uint32) { return uint32(os.Geteuid()), uint32(os.Getegid()) }

func foundationPlatformSupported() bool { return true }
