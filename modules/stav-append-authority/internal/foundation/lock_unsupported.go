//go:build !darwin && !linux

package foundation

import "fmt"

type lifecycleLease struct{}

func acquireLifecycleLease(string) (*lifecycleLease, error) {
	return nil, fmt.Errorf("foundational lifecycle locking is unsupported on this platform")
}

func (*lifecycleLease) Close() error { return nil }
