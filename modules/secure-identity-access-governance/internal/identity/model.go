package identity

import "time"

type Subject struct {
	ID         string            `json:"id"`
	Kind       string            `json:"kind"`
	Authority  string            `json:"authority"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

type ProofSummary struct {
	Type         string    `json:"type"`
	Issuer       string    `json:"issuer"`
	Audience     string    `json:"audience"`
	IssuedAt     time.Time `json:"issued_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	UserPresence bool      `json:"user_presence"`
	UserVerified bool      `json:"user_verified"`
}
