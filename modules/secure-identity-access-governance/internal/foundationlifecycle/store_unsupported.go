//go:build !darwin && !linux

package foundationlifecycle

import "fmt"

type attemptStore struct{}

func openAttemptStore(string, string, string, bool) (*attemptStore, error) {
	return nil, fmt.Errorf("SSIAG foundational lifecycle is unsupported on this platform")
}
func (*attemptStore) close() error { return nil }
func (*attemptStore) read() (Attempt, bool, error) {
	return Attempt{}, false, fmt.Errorf("unsupported")
}
func (*attemptStore) write(*Attempt) error    { return fmt.Errorf("unsupported") }
func (*attemptStore) readPlan() (Plan, error) { return Plan{}, fmt.Errorf("unsupported") }
func (*attemptStore) writePlan(Plan) error    { return fmt.Errorf("unsupported") }
