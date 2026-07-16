//go:build !darwin && !linux

package peerauth

import "fmt"

func credentialsFromFD(_ int) (Credentials, error) {
	return Credentials{}, fmt.Errorf("kernel Unix peer credentials are unsupported on this platform")
}
