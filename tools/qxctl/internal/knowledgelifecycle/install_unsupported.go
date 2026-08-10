//go:build !darwin && !linux

package knowledgelifecycle

import "fmt"

func installReceiptV2(string, string, string, receiptV2) error {
	return fmt.Errorf("compatibility_blocked: native package installation is unsupported; use WSL or a remote supported TOPS node")
}

func uninstallReceiptV2(string, string, string, receiptV2) error {
	return fmt.Errorf("compatibility_blocked: native package removal is unsupported; use WSL or a remote supported TOPS node")
}
