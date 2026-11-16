//go:build !darwin && !linux

package packageretirement

import "fmt"

type Lease struct{}

func unavailable() error {
	return fmt.Errorf("native package retirement is unsupported on this platform")
}
func PrepareInstall(string, Identity, bool, ...*Delegation) (*Lease, error) {
	return nil, unavailable()
}
func AcquireUse(string, Identity) (*Lease, error)                      { return nil, unavailable() }
func BeginRetirement(string, Identity, ...*Delegation) (*Lease, error) { return nil, unavailable() }
func Inspect(string, string, string) (Identity, string, error)         { return Identity{}, "", unavailable() }
func (*Lease) Activate() error                                         { return unavailable() }
func (*Lease) FinishRetirement() error                                 { return unavailable() }
func (*Lease) CheckRoot() error                                        { return unavailable() }
func (*Lease) VerifyFile(string, string, uint64) error                 { return unavailable() }
func (*Lease) RemoveFile(string, string, uint64) error                 { return unavailable() }
func (*Lease) State() string                                           { return "unsupported" }
func (*Lease) Close() error                                            { return nil }
func (*Lease) ReadFile(string, int64) ([]byte, error)                  { return nil, unavailable() }

func AcquireRecoveryUse(string, Identity) (*Lease, error) { return nil, unavailable() }
