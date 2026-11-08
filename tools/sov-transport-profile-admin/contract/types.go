// Package contract owns only the protected transport profile wire contract.
// It grants no privilege and implements no connection or provider operation.
package contract

const Version = "0.1.0-dev"
const MaxDocumentBytes = 8 << 20
const ProfileProtocol = "symphony.sov.ssh-transport-profile.v1"
const PolicyProtocol = "symphony.sov.ssh-transport-policy.v1"
const SnapshotProtocol = "symphony.sov.transport-profile-snapshot.v1"
const PlanProtocol = "symphony.sov.transport-profile-plan.v1"
const RequestProtocol = "symphony.sov.transport-profile-request.v1"
const ResultProtocol = "symphony.sov.transport-profile-result.v1"
const PackageProtocol = "symphony.sov.transport-package.v1"

type Scope struct {
	HostID string `json:"host_id"`
	TOPSID string `json:"tops_id"`
}
type Principal struct {
	UID string `json:"uid"`
	GID string `json:"gid"`
}
type PackageRef struct {
	PackageID        string `json:"package_id"`
	Version          string `json:"version"`
	ManifestDigest   string `json:"manifest_digest"`
	ReceiptDigest    string `json:"receipt_digest"`
	ExecutableDigest string `json:"executable_digest"`
}
type EndpointConstraint struct {
	Host          string `json:"host"`
	Port          string `json:"port"`
	User          string `json:"user"`
	HostKeyDigest string `json:"host_key_digest"`
}
type Limits struct {
	ControlBytes   string `json:"control_bytes"`
	CapturedBytes  string `json:"captured_bytes"`
	CleanupSeconds string `json:"cleanup_seconds"`
	CallerTimeout  bool   `json:"caller_timeout"`
}
type Configuration struct {
	Name               string               `json:"name"`
	TransportKind      string               `json:"transport_kind"`
	Platform           string               `json:"platform"`
	ExecutionProfile   string               `json:"execution_profile"`
	Adapter            PackageRef           `json:"adapter"`
	SSH                PackageRef           `json:"ssh"`
	Helper             PackageRef           `json:"helper"`
	Applications       []string             `json:"applications"`
	Endpoints          []EndpointConstraint `json:"endpoints"`
	AuthenticationMode string               `json:"authentication_mode"`
	SessionRef         string               `json:"session_ref"`
	Limits             Limits               `json:"limits"`
	Compatibility      []string             `json:"compatibility"`
}
type PolicyRef struct {
	Generation string `json:"generation"`
	Digest     string `json:"digest"`
}
type Profile struct {
	Protocol      string        `json:"protocol"`
	Scope         Scope         `json:"scope"`
	ProfileID     string        `json:"profile_id"`
	Generation    string        `json:"generation"`
	State         string        `json:"state"`
	Policy        PolicyRef     `json:"policy"`
	Configuration Configuration `json:"configuration"`
	Digest        string        `json:"digest"`
}
type Policy struct {
	Protocol          string      `json:"protocol"`
	Scope             Scope       `json:"scope"`
	ProfileID         string      `json:"profile_id"`
	ProfileGeneration string      `json:"profile_generation"`
	Generation        string      `json:"generation"`
	Permitted         []Principal `json:"permitted"`
	Digest            string      `json:"digest"`
}
type Snapshot struct {
	Protocol string  `json:"protocol"`
	Profile  Profile `json:"profile"`
	Policy   Policy  `json:"policy"`
	Digest   string  `json:"digest"`
}
type Expected struct {
	ProfileGeneration string `json:"profile_generation"`
	PolicyGeneration  string `json:"policy_generation"`
	SnapshotDigest    string `json:"snapshot_digest"`
}
type Change struct {
	Action        string         `json:"action"`
	Configuration *Configuration `json:"configuration"`
	Permitted     *[]Principal   `json:"permitted"`
}
type Plan struct {
	Protocol            string    `json:"protocol"`
	AdministratorDigest string    `json:"administrator_digest"`
	OperationID         string    `json:"operation_id"`
	Scope               Scope     `json:"scope"`
	ProfileID           string    `json:"profile_id"`
	Expected            *Expected `json:"expected"`
	Change              Change    `json:"change"`
	Desired             Snapshot  `json:"desired"`
	CreatedAt           string    `json:"created_at"`
	ExpiresAt           string    `json:"expires_at"`
	Digest              string    `json:"digest"`
}
type Recovery struct {
	OperationID           string `json:"operation_id"`
	PlanDigest            string `json:"plan_digest"`
	ExpectedPendingDigest string `json:"expected_pending_digest"`
	Action                string `json:"action"`
}
type Request struct {
	Protocol    string    `json:"protocol"`
	RequestID   string    `json:"request_id"`
	Operation   string    `json:"operation"`
	Scope       Scope     `json:"scope"`
	ProfileID   *string   `json:"profile_id"`
	OperationID *string   `json:"operation_id"`
	Change      *Change   `json:"change"`
	Plan        *Plan     `json:"plan"`
	Recovery    *Recovery `json:"recovery"`
}
type Attempt struct {
	OperationID        string  `json:"operation_id"`
	PlanDigest         string  `json:"plan_digest"`
	Stage              string  `json:"stage"`
	PendingDigest      *string `json:"pending_digest"`
	Outcome            string  `json:"outcome"`
	AuditIntentDigest  *string `json:"audit_intent_digest"`
	AuditOutcomeDigest *string `json:"audit_outcome_digest"`
}
type Result struct {
	Protocol   string     `json:"protocol"`
	RequestID  string     `json:"request_id"`
	Operation  string     `json:"operation"`
	Scope      Scope      `json:"scope"`
	Completed  bool       `json:"completed"`
	Code       string     `json:"code"`
	Profiles   []Snapshot `json:"profiles"`
	Snapshot   *Snapshot  `json:"snapshot"`
	Plan       *Plan      `json:"plan"`
	Attempt    *Attempt   `json:"attempt"`
	NextAction string     `json:"next_action"`
}

// Package manifests are installed separately by a host administrator. Paths
// are relative to a fixed protected package root, never caller-selected paths.
type PackageFile struct {
	Path       string `json:"path"`
	Digest     string `json:"digest"`
	Size       string `json:"size"`
	Executable bool   `json:"executable"`
}
type PackageManifest struct {
	Protocol            string        `json:"protocol"`
	PackageID           string        `json:"package_id"`
	Version             string        `json:"version"`
	Role                string        `json:"role"`
	Platform            string        `json:"platform"`
	Executable          string        `json:"executable"`
	ReceiptPath         string        `json:"receipt_path"`
	ReceiptDigest       string        `json:"receipt_digest"`
	Files               []PackageFile `json:"files"`
	QualificationRef    string        `json:"qualification_ref"`
	QualificationPath   string        `json:"qualification_path"`
	ReceiptComponentID  string        `json:"receipt_component_id"`
	ReceiptVersion      string        `json:"receipt_version"`
	ReceiptEntryPointID string        `json:"receipt_entry_point_id"`
	Digest              string        `json:"digest"`
}
