package validation

import "encoding/json"

const (
	ResultProtocol   = "symphony.validation.result.v1"
	PolicyProtocol   = "symphony.validation.policy.v1"
	BaselineProtocol = "symphony.validation.baseline.v1"
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
	Result    Result
	RawJSON   json.RawMessage
	Displayed []Finding
}
