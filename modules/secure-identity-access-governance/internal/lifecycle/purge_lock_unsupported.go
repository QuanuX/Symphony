//go:build !darwin && !linux

package lifecycle

import "fmt"

type purgeSocketLease struct{}

func acquirePurgeSocketLease(string) (*purgeSocketLease, error) {
	return nil, fmt.Errorf("SSIAG purge lifecycle locking is unsupported on this platform")
}

func (*purgeSocketLease) Close() error { return nil }
