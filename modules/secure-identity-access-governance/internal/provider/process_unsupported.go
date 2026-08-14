//go:build !darwin && !linux

package provider

import "os/exec"

func configureProviderProcess(*exec.Cmd) {}
func terminateProviderProcess(*exec.Cmd) {}
