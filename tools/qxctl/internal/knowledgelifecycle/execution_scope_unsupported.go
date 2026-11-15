//go:build !darwin && !linux

package knowledgelifecycle

import "fmt"

func ReadExecutionScope(string) (ExecutionScope, error) {
	return ExecutionScope{}, fmt.Errorf("execution scope reading is supported only on Linux and macOS")
}
