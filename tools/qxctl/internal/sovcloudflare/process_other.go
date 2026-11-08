//go:build !darwin && !linux

package sovcloudflare

import "os/exec"

// Receipt platform admission rejects unsupported platforms before dispatch.
func configure(cmd *exec.Cmd) {}
func cleanup(cmd *exec.Cmd)   {}
