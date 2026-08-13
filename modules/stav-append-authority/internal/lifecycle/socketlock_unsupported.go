//go:build !darwin && !linux

package lifecycle

import "fmt"

type socketLifecycleLease struct{}

func acquireSocketLifecycleLease(string) (*socketLifecycleLease, error) {
	return nil, fmt.Errorf("STAV socket lifecycle locking is unsupported on this platform")
}

func (*socketLifecycleLease) Close() error { return nil }
