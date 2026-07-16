package policy

import (
	"context"
	"time"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/credential"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/identity"
)

type Request struct {
	ID          string
	Subject     identity.Subject
	Reference   credential.Reference
	Operation   string
	Audience    string
	RequestedAt time.Time
}

type Decision struct {
	Allowed bool
	Reason  string
	Expires time.Time
}

type Evaluator interface {
	Evaluate(context.Context, Request) Decision
}

// DenyAll is the only policy supplied by the scaffold.
type DenyAll struct{}

func (DenyAll) Evaluate(_ context.Context, _ Request) Decision {
	return Decision{Allowed: false, Reason: "policy.deny_by_default"}
}
