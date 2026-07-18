//go:build !darwin && !linux

package ssiagclient

import (
	"fmt"
	"net"
)

type Credentials struct {
	PID int32
	UID uint32
	GID uint32
}

func getPeerCredentials(_ net.Conn) (Credentials, error) {
	return Credentials{}, fmt.Errorf("kernel Unix peer credentials are unsupported on this platform")
}
