//go:build !darwin && !linux

package peerauth

import "fmt"

func credentialsFromFD(int) (Credentials, error) {
	return Credentials{}, fmt.Errorf("stav peer authentication: unsupported operating system")
}
