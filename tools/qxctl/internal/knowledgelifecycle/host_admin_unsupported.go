//go:build !darwin && !linux

package knowledgelifecycle

import (
	"fmt"
	"time"
)

type HostDrift struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

type HostIntegrationResult struct {
	Protocol        string           `json:"protocol"`
	Operation       string           `json:"operation"`
	TOPSID          string           `json:"tops_id"`
	ProfileID       string           `json:"profile_id"`
	Present         bool             `json:"present"`
	Integration     *HostIntegration `json:"integration"`
	Drift           []HostDrift      `json:"drift"`
	RepairActions   []string         `json:"repair_actions"`
	Changed         bool             `json:"changed"`
	Recovered       bool             `json:"recovered"`
	ApplyAuthorized bool             `json:"apply_authorized"`
	Canonical       bool             `json:"canonical"`
}
type HostAdmin struct{}
type HostProvisionInput struct {
	Operation       string
	ProfileID       string
	RepositoryRoot  string
	IntegrationRoot string
	RecoveryMode    string
	DesiredEnabled  bool
	ExpectedDigest  string
	Now             time.Time
}

func NewHostAdmin(*Store) (*HostAdmin, error) {
	return nil, fmt.Errorf("native lifecycle host integration is Linux-only; use WSL or a remote Linux TOPS node")
}

func (*HostAdmin) Provision(HostProvisionInput) (HostIntegrationResult, error) {
	return HostIntegrationResult{}, fmt.Errorf("native lifecycle host integration is Linux-only")
}
func (*HostAdmin) Status(string) (HostIntegrationResult, error) {
	return HostIntegrationResult{}, fmt.Errorf("native lifecycle host integration is Linux-only")
}
func (*HostAdmin) Reconcile(string, time.Time) (HostIntegrationResult, error) {
	return HostIntegrationResult{}, fmt.Errorf("native lifecycle host integration is Linux-only")
}
func (*HostAdmin) SetEnabled(string, string, bool, time.Time) (HostIntegrationResult, error) {
	return HostIntegrationResult{}, fmt.Errorf("native lifecycle host integration is Linux-only")
}
func (*HostAdmin) Uninstall(string, string, time.Time) (HostIntegrationResult, error) {
	return HostIntegrationResult{}, fmt.Errorf("native lifecycle host integration is Linux-only")
}
