//go:build !darwin && !linux

package server

import "fmt"

type socketLease struct{}

func acquireSocketLease(string) (*socketLease, error) {
	return nil, fmt.Errorf("Accordare socket lifecycle locks are unsupported")
}

func (*socketLease) Close() error { return nil }
