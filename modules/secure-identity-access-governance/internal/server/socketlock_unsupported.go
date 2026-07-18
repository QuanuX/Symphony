//go:build !darwin && !linux

package server

import "fmt"

type socketLease struct{}

func acquireSocketLease(string) (*socketLease, error) {
	return nil, fmt.Errorf("SSIAG socket lifecycle locking is unsupported on this platform")
}

func (*socketLease) Close() error { return nil }
