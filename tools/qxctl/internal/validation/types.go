package validation

import "encoding/json"

const (
	ResultProtocol       = "symphony.validation.result.v1"
	PolicyProtocol       = "symphony.validation.policy.v1"
	BaselineProtocol     = "symphony.validation.baseline.v1"
	WarningStateProtocol = "symphony.validation.warning-state.v1"
	RootSummaryProtocol  = "symphony.repository.root-summary.v1"
)

type Finding struct {
	Attributes   map[string]string `json:"attributes"`
	Category     string            `json:"category"`
	Detail       string            `json:"detail"`
	OccurrenceID string            `json:"occurrence_id"`
	RuleID       string            `json:"rule_id"`
	Scope        string            `json:"scope"`
	SubjectID    string            `json:"subject_id"`
}

type Summary struct {
	Other     uint64 `json:"other"`
	Pass      uint64 `json:"pass"`
	Total     uint64 `json:"total"`
	Violation uint64 `json:"violation"`
	Warning   uint64 `json:"warning"`
}

type Evidence struct {
	EvidenceDigest           string    `json:"evidence_digest"`
	ExitCode                 int       `json:"exit_code"`
	Findings                 []Finding `json:"findings"`
	Outcome                  string    `json:"outcome"`
	RepositoryIdentityDigest string    `json:"repository_identity_digest"`
	Summary                  Summary   `json:"summary"`
	ValidatorID              string    `json:"validator_id"`
	ValidatorVersion         string    `json:"validator_version"`
}

type Evaluation struct {
	BaselineDigest                *string  `json:"baseline_digest"`
	BaselineID                    *string  `json:"baseline_id"`
	DisplayedOccurrenceIDs        []string `json:"displayed_occurrence_ids"`
	EvaluationDigest              string   `json:"evaluation_digest"`
	NewWarningOccurrenceIDs       []string `json:"new_warning_occurrence_ids"`
	Outcome                       string   `json:"outcome"`
	PolicyDigest                  *string  `json:"policy_digest"`
	ProfileID                     *string  `json:"profile_id"`
	RequiredRuleIDs               []string `json:"required_rule_ids"`
	ResolvedWarningOccurrenceIDs  []string `json:"resolved_warning_occurrence_ids"`
	ReviewRuleIDs                 []string `json:"review_rule_ids"`
	UnchangedWarningOccurrenceIDs []string `json:"unchanged_warning_occurrence_ids"`
}

type Result struct {
	Evaluation    *Evaluation `json:"evaluation"`
	Evidence      Evidence    `json:"evidence"`
	FormatVersion uint64      `json:"format_version"`
	Protocol      string      `json:"protocol"`
	ResultDigest  string      `json:"result_digest"`
}

type RulePolicy struct {
	Disposition  string `json:"disposition"`
	Presentation string `json:"presentation"`
	RuleID       string `json:"rule_id"`
}

type Policy struct {
	Canonical                 bool         `json:"canonical"`
	DefaultWarningDisposition string       `json:"default_warning_disposition"`
	FormatVersion             uint64       `json:"format_version"`
	Generation                uint64       `json:"generation"`
	HistoricalPresentation    string       `json:"historical_presentation"`
	NewPresentation           string       `json:"new_presentation"`
	PolicyDigest              string       `json:"policy_digest"`
	PreviousPolicyDigest      *string      `json:"previous_policy_digest"`
	ProfileID                 string       `json:"profile_id"`
	Protocol                  string       `json:"protocol"`
	Rules                     []RulePolicy `json:"rules"`
	TOPSID                    string       `json:"tops_id"`
	UpdatedAt                 string       `json:"updated_at"`
}

type PolicyConfig struct {
	DefaultWarningDisposition string
	HistoricalPresentation    string
	NewPresentation           string
	ProfileID                 string
	Rules                     []RulePolicy
}

type Baseline struct {
	BaselineDigest           string   `json:"baseline_digest"`
	BaselineID               string   `json:"baseline_id"`
	Canonical                bool     `json:"canonical"`
	CreatedAt                string   `json:"created_at"`
	EvidenceDigest           string   `json:"evidence_digest"`
	FormatVersion            uint64   `json:"format_version"`
	Protocol                 string   `json:"protocol"`
	RepositoryIdentityDigest string   `json:"repository_identity_digest"`
	TOPSID                   string   `json:"tops_id"`
	ValidatorID              string   `json:"validator_id"`
	ValidatorVersion         string   `json:"validator_version"`
	WarningOccurrenceIDs     []string `json:"warning_occurrence_ids"`
	WarningSubjectIDs        []string `json:"warning_subject_ids"`
}

type PolicySnapshot struct {
	Exists bool   `json:"exists"`
	Policy Policy `json:"policy"`
}

type BaselineSnapshot struct {
	Exists   bool     `json:"exists"`
	Baseline Baseline `json:"baseline"`
}

// WarningState is a protected, noncanonical lifecycle projection over immutable
// validator warnings. It is intentionally side-by-side with the exact v1 raw
// result, policy, and baseline protocols: older clients can continue to decode
// those objects without accepting new fields.
type WarningState struct {
	Canonical                bool                `json:"canonical"`
	FormatVersion            uint64              `json:"format_version"`
	Generation               uint64              `json:"generation"`
	LastEvidenceDigest       string              `json:"last_evidence_digest"`
	LastSync                 WarningSync         `json:"last_sync"`
	PreviousStateDigest      *string             `json:"previous_state_digest"`
	Protocol                 string              `json:"protocol"`
	RepositoryIdentityDigest string              `json:"repository_identity_digest"`
	StateDigest              string              `json:"state_digest"`
	StateID                  string              `json:"state_id"`
	Subjects                 []WarningSubject    `json:"subjects"`
	TOPSID                   string              `json:"tops_id"`
	Transitions              []WarningTransition `json:"transitions"`
	UpdatedAt                string              `json:"updated_at"`
	ValidatorID              string              `json:"validator_id"`
	ValidatorVersion         string              `json:"validator_version"`
}

type WarningSync struct {
	EvidenceDigest               string   `json:"evidence_digest"`
	KnownSubjectNewOccurrenceIDs []string `json:"known_subject_new_occurrence_ids"`
	NewSubjectIDs                []string `json:"new_subject_ids"`
	ObservedOccurrenceIDs        []string `json:"observed_occurrence_ids"`
	ReopenedSubjectIDs           []string `json:"reopened_subject_ids"`
	ResolvedSubjectIDs           []string `json:"resolved_subject_ids"`
	SyncDigest                   string   `json:"sync_digest"`
	UnchangedOccurrenceIDs       []string `json:"unchanged_occurrence_ids"`
}

type WarningSubject struct {
	Classification        string              `json:"classification"`
	FirstObservedAt       string              `json:"first_observed_at"`
	LastObservedAt        string              `json:"last_observed_at"`
	Muted                 bool                `json:"muted_presentation_only"`
	OccurrenceIDs         []string            `json:"occurrence_ids"`
	Occurrences           []WarningOccurrence `json:"occurrences"`
	Rationale             *string             `json:"rationale"`
	RuleID                string              `json:"rule_id"`
	SubjectID             string              `json:"subject_id"`
	SupersededBySubjectID *string             `json:"superseded_by_subject_id"`
	ValidUntil            *string             `json:"valid_until"`
}

type WarningOccurrence struct {
	Active          bool     `json:"active"`
	EvidenceDigests []string `json:"evidence_digests"`
	Finding         Finding  `json:"finding"`
	FirstObservedAt string   `json:"first_observed_at"`
	LastObservedAt  string   `json:"last_observed_at"`
	OccurrenceID    string   `json:"occurrence_id"`
}

type WarningTransition struct {
	At                       string  `json:"at"`
	FromClassification       *string `json:"from_classification"`
	Operation                string  `json:"operation"`
	OccurrenceID             *string `json:"occurrence_id"`
	PreviousTransitionDigest *string `json:"previous_transition_digest"`
	Rationale                *string `json:"rationale"`
	Sequence                 uint64  `json:"sequence"`
	SubjectID                string  `json:"subject_id"`
	SupersededBySubjectID    *string `json:"superseded_by_subject_id"`
	ToClassification         string  `json:"to_classification"`
	TransitionDigest         string  `json:"transition_digest"`
	ValidUntil               *string `json:"valid_until"`
}

type WarningStateSnapshot struct {
	Exists bool         `json:"exists"`
	State  WarningState `json:"state"`
}

type WarningStateSummary struct {
	Accepted    int    `json:"accepted"`
	Generation  uint64 `json:"generation"`
	Muted       int    `json:"muted_presentation_only"`
	Open        int    `json:"open"`
	Resolved    int    `json:"resolved"`
	StateDigest string `json:"state_digest"`
	StateID     string `json:"state_id"`
	Superseded  int    `json:"superseded"`
}

type WarningStateList struct {
	Canonical bool                  `json:"canonical"`
	States    []WarningStateSummary `json:"states"`
	TOPSID    string                `json:"tops_id"`
}

type WarningMutation struct {
	Classification        string
	ExpectedStateDigest   string
	Muted                 *bool
	Rationale             string
	StateID               string
	SubjectID             string
	SupersededBySubjectID string
	ValidUntil            string
}

type RootSummary struct {
	FeatureAdministration   RootSummaryAdministration `json:"feature_administration"`
	FormatVersion           uint64                    `json:"format_version"`
	Protocol                string                    `json:"protocol"`
	PublishedSourceVersions []RootSummaryPublication  `json:"published_source_versions"`
	QXCTL                   RootSummaryQXCTL          `json:"qxctl"`
	SSFV                    RootSummarySSFV           `json:"ssfv"`
	SummaryDigest           string                    `json:"summary_digest"`
}

type RootSummaryAdministration struct {
	Exemptions   uint64 `json:"exemptions"`
	Expectations uint64 `json:"expectations"`
	Prohibited   uint64 `json:"prohibited"`
	Required     uint64 `json:"required"`
	Unreviewed   uint64 `json:"unreviewed"`
}

type RootSummaryPublication struct {
	Coordinate string `json:"coordinate"`
	Revision   string `json:"revision"`
	Tag        string `json:"tag"`
	Version    string `json:"version"`
}

type RootSummaryQXCTL struct {
	RegisteredCommands uint64 `json:"registered_commands"`
}

type RootSummarySSFV struct {
	CatalogState            string   `json:"catalog_state"`
	NestedFeatures          uint64   `json:"nested_features"`
	RegisteredFeatures      uint64   `json:"registered_features"`
	RegisteredOwnerFeatures []string `json:"registered_owner_features"`
	RegisteredOwnerScopes   uint64   `json:"registered_owner_scopes"`
}

type PolicySummary struct {
	DefaultWarningDisposition string `json:"default_warning_disposition"`
	Generation                uint64 `json:"generation"`
	PolicyDigest              string `json:"policy_digest"`
	ProfileID                 string `json:"profile_id"`
	RuleCount                 int    `json:"rule_count"`
}

type PolicyList struct {
	Canonical bool            `json:"canonical"`
	Policies  []PolicySummary `json:"policies"`
	TOPSID    string          `json:"tops_id"`
}

type Projection struct {
	Result                 Result
	RawJSON                json.RawMessage
	Displayed              []Finding
	Historical             []WarningOccurrence
	WarningClassifications map[string]string
}
