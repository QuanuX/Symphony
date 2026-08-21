//go:build !darwin && !linux

package peer

import "fmt"

func fromFD(int) (Credentials, error) {
	return Credentials{}, fmt.Errorf("kernel peer credentials are unsupported")
}
