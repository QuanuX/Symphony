//go:build !darwin && !linux

package packageretirement

type Delegation struct{}

func (*Delegation) Close() error { return nil }
func ReadDispatch(rootFD, lockFD, proofFD int, operation, componentID, requestedVersion, prefix string) (*Delegation, error) {
	if rootFD == -1 && lockFD == -1 && proofFD == -1 {
		return nil, nil
	}
	return nil, unavailable()
}
