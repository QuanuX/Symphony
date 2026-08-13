package foundationlifecycle

const (
	AdapterProtocol     = "symphony.foundation.lifecycle-adapter.v1"
	CommandProtocol     = "symphony.foundation.lifecycle-command.v1"
	ObservationProtocol = "symphony.foundation.lifecycle-observation.v1"
	PlanProtocol        = "symphony.foundation.lifecycle-plan.v1"
	AttemptProtocol     = "symphony.foundation.lifecycle-attempt.v1"
	ResultProtocol      = "symphony.foundation.lifecycle-result.v1"
)

type Command struct {
	Protocol              string  `json:"protocol"`
	FormatVersion         uint64  `json:"format_version"`
	Operation             string  `json:"operation"`
	Component             string  `json:"component"`
	Surface               string  `json:"surface"`
	Scope                 string  `json:"scope"`
	TOPSID                string  `json:"tops_id"`
	OperationID           *string `json:"operation_id"`
	RequestID             *string `json:"request_id"`
	CorrelationID         *string `json:"correlation_id"`
	ExpectedStateDigest   *string `json:"expected_state_digest"`
	ExpectedAttemptDigest *string `json:"expected_attempt_digest"`
	Intent                *Intent `json:"intent"`
	Plan                  *Plan   `json:"plan"`
	Discover              bool    `json:"discover"`
	RequestedAt           string  `json:"requested_at"`
	DeadlineAt            string  `json:"deadline_at"`
}

type Intent struct {
	DesiredState string  `json:"desired_state"`
	TOPSName     *string `json:"tops_name"`
	ServiceUID   *uint32 `json:"service_uid"`
	ServiceGID   *uint32 `json:"service_gid"`
	AuthorityUID *uint32 `json:"authority_uid"`
	AuthorityGID *uint32 `json:"authority_gid"`
	AuditMode    string  `json:"audit_mode"`
	TTLSeconds   uint64  `json:"ttl_seconds"`
}

type Plan struct {
	Protocol            string  `json:"protocol"`
	FormatVersion       uint64  `json:"format_version"`
	Component           string  `json:"component"`
	Surface             string  `json:"surface"`
	Scope               string  `json:"scope"`
	TOPSID              string  `json:"tops_id"`
	OperationID         string  `json:"operation_id"`
	RequestID           string  `json:"request_id"`
	CorrelationID       string  `json:"correlation_id"`
	ExpectedStateDigest string  `json:"expected_state_digest"`
	DesiredState        string  `json:"desired_state"`
	TOPSName            *string `json:"tops_name"`
	ServiceUID          *uint32 `json:"service_uid"`
	ServiceGID          *uint32 `json:"service_gid"`
	AuthorityUID        *uint32 `json:"authority_uid"`
	AuthorityGID        *uint32 `json:"authority_gid"`
	AuditMode           string  `json:"audit_mode"`
	CreatedAt           string  `json:"created_at"`
	ExpiresAt           string  `json:"expires_at"`
	PlanDigest          string  `json:"plan_digest"`
}

type InstallationObservation struct {
	State                 string  `json:"state"`
	BinaryPath            *string `json:"binary_path"`
	BinaryDigest          *string `json:"binary_digest"`
	InstallEvidenceDigest *string `json:"install_evidence_digest"`
	ReceiptDigest         *string `json:"receipt_digest"`
	Legacy                bool    `json:"legacy"`
}

type EnrollmentObservation struct {
	State         string  `json:"state"`
	RecordPath    *string `json:"record_path"`
	RecordDigest  *string `json:"record_digest"`
	ConfigPath    *string `json:"config_path"`
	ConfigDigest  *string `json:"config_digest"`
	UID           *uint32 `json:"uid"`
	GID           *uint32 `json:"gid"`
	DataPreserved bool    `json:"data_preserved"`
}

type SupervisorObservation struct {
	Manager              *string `json:"manager"`
	ManagerState         string  `json:"manager_state"`
	DescriptorState      string  `json:"descriptor_state"`
	DescriptorPath       *string `json:"descriptor_path"`
	DescriptorDigest     *string `json:"descriptor_digest"`
	Enablement           string  `json:"enablement"`
	ProcessState         string  `json:"process_state"`
	EndpointState        string  `json:"endpoint_state"`
	ActivationGeneration *string `json:"activation_generation"`
	PackageReceiptDigest *string `json:"package_receipt_digest"`
}

type Observation struct {
	Protocol            string                  `json:"protocol"`
	FormatVersion       uint64                  `json:"format_version"`
	Component           string                  `json:"component"`
	Surface             string                  `json:"surface"`
	Scope               string                  `json:"scope"`
	TOPSID              string                  `json:"tops_id"`
	Installation        InstallationObservation `json:"installation"`
	Enrollment          EnrollmentObservation   `json:"enrollment"`
	Supervisor          SupervisorObservation   `json:"supervisor"`
	RecoveryRequired    bool                    `json:"recovery_required"`
	ActiveAttemptDigest *string                 `json:"active_attempt_digest"`
	ObservedAt          string                  `json:"observed_at"`
	StableStateDigest   string                  `json:"stable_state_digest"`
	ObservationDigest   string                  `json:"observation_digest"`
}

type Attempt struct {
	Protocol              string  `json:"protocol"`
	FormatVersion         uint64  `json:"format_version"`
	Component             string  `json:"component"`
	Surface               string  `json:"surface"`
	Scope                 string  `json:"scope"`
	TOPSID                string  `json:"tops_id"`
	OperationID           string  `json:"operation_id"`
	RequestID             string  `json:"request_id"`
	CorrelationID         string  `json:"correlation_id"`
	Phase                 string  `json:"phase"`
	PlanDigest            string  `json:"plan_digest"`
	PriorStateDigest      string  `json:"prior_state_digest"`
	DesiredState          string  `json:"desired_state"`
	BinaryDigest          string  `json:"binary_digest"`
	InstallEvidenceDigest string  `json:"install_evidence_digest"`
	AuditState            string  `json:"audit_state"`
	AuditReceiptDigest    *string `json:"audit_receipt_digest"`
	StartedAt             string  `json:"started_at"`
	UpdatedAt             string  `json:"updated_at"`
	CompletedAt           *string `json:"completed_at"`
	PredecessorDigest     *string `json:"predecessor_digest"`
	ResultDigest          *string `json:"result_digest"`
	AttemptDigest         string  `json:"attempt_digest"`
}

type ErrorResult struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Result struct {
	Protocol               string       `json:"protocol"`
	FormatVersion          uint64       `json:"format_version"`
	Operation              string       `json:"operation"`
	Component              string       `json:"component"`
	Surface                string       `json:"surface"`
	Scope                  string       `json:"scope"`
	TOPSID                 string       `json:"tops_id"`
	OperationID            *string      `json:"operation_id"`
	Disposition            string       `json:"disposition"`
	DesiredState           *string      `json:"desired_state"`
	Observation            Observation  `json:"observation"`
	Plan                   *Plan        `json:"plan"`
	Changed                bool         `json:"changed"`
	Replayed               bool         `json:"replayed"`
	Recovered              bool         `json:"recovered"`
	RecoveryRequired       bool         `json:"recovery_required"`
	ReconciliationRequired bool         `json:"reconciliation_required"`
	AttemptDigest          *string      `json:"attempt_digest"`
	AuditState             string       `json:"audit_state"`
	AuditReceiptDigest     *string      `json:"audit_receipt_digest"`
	StartedAt              *string      `json:"started_at"`
	CompletedAt            string       `json:"completed_at"`
	Error                  *ErrorResult `json:"error"`
	ReadOnly               bool         `json:"read_only"`
	Canonical              bool         `json:"canonical"`
	ResultDigest           string       `json:"result_digest"`
}

type Compatibility struct {
	ConfigReadMajors  []uint64 `json:"config_read_majors"`
	ConfigWriteMajor  uint64   `json:"config_write_major"`
	RuntimeReadMajors []uint64 `json:"runtime_read_majors"`
	RuntimeWriteMajor uint64   `json:"runtime_write_major"`
	StateReadMajors   []uint64 `json:"state_read_majors"`
	StateWriteMajor   uint64   `json:"state_write_major"`
	RollbackReadable  bool     `json:"rollback_readable"`
}

type Limits struct {
	RequestBytes  uint64 `json:"request_bytes"`
	ResponseBytes uint64 `json:"response_bytes"`
	DeadlineMS    uint64 `json:"deadline_ms"`
	JSONDepth     uint64 `json:"json_depth"`
	JSONValues    uint64 `json:"json_values"`
}

type AdapterDescriptor struct {
	Protocol              string        `json:"protocol"`
	FormatVersion         uint64        `json:"format_version"`
	Component             string        `json:"component"`
	AdapterVersion        string        `json:"adapter_version"`
	BinaryPath            string        `json:"binary_path"`
	BinaryDigest          string        `json:"binary_digest"`
	InstallEvidenceDigest string        `json:"install_evidence_digest"`
	Operations            []string      `json:"operations"`
	SupportedScopes       []string      `json:"supported_scopes"`
	SupportedManagers     []string      `json:"supported_managers"`
	Compatibility         Compatibility `json:"compatibility"`
	Limits                Limits        `json:"limits"`
	CanonicalApplyEnabled bool          `json:"canonical_apply_enabled"`
	NetworkListener       bool          `json:"network_listener"`
	DescriptorDigest      string        `json:"descriptor_digest"`
}
