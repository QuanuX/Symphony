//go:build !darwin && !linux

package knowledgelifecycle

import "fmt"

func (e *Executor) executeNativeOwner(_ *ExecutionResult, _ PlannedAction, _ DesiredComponent, _ bool, _ ObservedComponent, _ bool) error {
	return fmt.Errorf("compatibility_blocked: native owner dispatch requires a qualified Unix host")
}
