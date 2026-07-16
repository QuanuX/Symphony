package credential

import "time"

type Reference struct {
	Provider string `json:"provider"`
	Name     string `json:"name"`
	Version  string `json:"version,omitempty"`
	Type     string `json:"type"`
}

type Lease struct {
	ID         string    `json:"id"`
	SubjectID  string    `json:"subject_id"`
	Reference  Reference `json:"reference"`
	Operations []string  `json:"operations"`
	IssuedAt   time.Time `json:"issued_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}
