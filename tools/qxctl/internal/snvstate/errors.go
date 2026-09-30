package snvstate

// BoundaryError names an explicitly detected storage/publication condition.
// Causes remain available to local recovery code; the CLI emits only admitted
// fixed codes and fixed messages. An opaque failure is never policy denial.
type BoundaryError struct {
	Code  string
	Cause error
}

func (e *BoundaryError) Error() string       { return "SNV operation refused" }
func (e *BoundaryError) Unwrap() error       { return e.Cause }
func Refusal(code string, cause error) error { return &BoundaryError{Code: code, Cause: cause} }
