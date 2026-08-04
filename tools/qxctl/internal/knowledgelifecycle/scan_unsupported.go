//go:build !darwin && !linux

package knowledgelifecycle

import "fmt"

func scanReceiptCandidates([]string) ([]receiptCandidate, error) {
	return nil, fmt.Errorf("configured-root observation is unsupported on this operating system")
}

func hashTrustedRelative(string, string, int64) (string, uint64, error) {
	return "", 0, fmt.Errorf("configured-root observation is unsupported on this operating system")
}

func hashRegularFile(string, int64) (string, error) {
	return "", fmt.Errorf("configured-root observation is unsupported on this operating system")
}

func kernelABI() string { return "unsupported" }
