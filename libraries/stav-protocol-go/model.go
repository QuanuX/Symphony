package stavprotocol

const (
	SchemaCandidate    = "symphony.stav.candidate.v1"
	SchemaEvent        = "symphony.stav.event.v1"
	SchemaReceipt      = "symphony.stav.receipt.v1"
	SchemaQuery        = "symphony.stav.query.v1"
	SchemaQueryPage    = "symphony.stav.query-page.v1"
	SchemaVerification = "symphony.stav.verification.v1"

	SchemaLocalRequest  = "symphony.stav.local.request.v1"
	SchemaLocalResponse = "symphony.stav.local.response.v1"
)

const (
	MaxCandidateBytes = 61_440
	MaxEventBytes     = 65_536
	MaxRequestBytes   = 65_536
	MaxResponseBytes  = 4_194_304
	MaxQueryLimit     = 1_000
)

type SafeReference struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}

type TROG struct {
	ID         string `json:"id,omitempty"`
	ReasonCode string `json:"reason_code,omitempty"`
	State      string `json:"state"`
}

type Authentication struct {
	MethodID   string `json:"method_id,omitempty"`
	ReasonCode string `json:"reason_code,omitempty"`
	State      string `json:"state"`
}

type Configuration struct {
	NewDigest      string `json:"new_digest,omitempty"`
	PreviousDigest string `json:"previous_digest,omitempty"`
	ReasonCode     string `json:"reason_code,omitempty"`
	State          string `json:"state"`
}

type Topology struct {
	TOPSID string `json:"tops_id"`
	TROG   TROG   `json:"trog"`
}

type Correlation struct {
	CorrelationID string `json:"correlation_id"`
	RequestID     string `json:"request_id"`
}

type Operation struct {
	EventClass  string        `json:"event_class"`
	OperationID string        `json:"operation_id"`
	Target      SafeReference `json:"target"`
}

type Result struct {
	IntentID   string `json:"intent_id"`
	Outcome    string `json:"outcome"`
	ReasonCode string `json:"reason_code"`
}

type Redaction struct {
	Classification string `json:"classification"`
}

type CandidateActor struct {
	Authentication Authentication `json:"authentication"`
	Principal      SafeReference  `json:"principal"`
}

type Candidate struct {
	Actor         CandidateActor `json:"actor"`
	Configuration Configuration  `json:"configuration"`
	Correlation   Correlation    `json:"correlation"`
	Operation     Operation      `json:"operation"`
	Redaction     Redaction      `json:"redaction"`
	Result        Result         `json:"result"`
	Schema        string         `json:"schema"`
	Topology      Topology       `json:"topology"`
}

type EventActor struct {
	Authentication Authentication `json:"authentication"`
	Principal      SafeReference  `json:"principal"`
	Producer       SafeReference  `json:"producer"`
}

type EventIdentity struct {
	EventID string `json:"event_id"`
	Schema  string `json:"schema"`
}

type EventOrdering struct {
	Sequence  uint64 `json:"sequence"`
	Timestamp string `json:"timestamp"`
}

type EventIntegrity struct {
	PrecedingEventDigest string `json:"preceding_event_digest"`
}

type Event struct {
	Actor         EventActor     `json:"actor"`
	Configuration Configuration  `json:"configuration"`
	Correlation   Correlation    `json:"correlation"`
	Identity      EventIdentity  `json:"identity"`
	Integrity     EventIntegrity `json:"integrity"`
	Operation     Operation      `json:"operation"`
	Ordering      EventOrdering  `json:"ordering"`
	Redaction     Redaction      `json:"redaction"`
	Result        Result         `json:"result"`
	Topology      Topology       `json:"topology"`
}

type CommitResult struct {
	EventDigest string `json:"event_digest,omitempty"`
	EventID     string `json:"event_id,omitempty"`
	ReasonCode  string `json:"reason_code,omitempty"`
	Sequence    uint64 `json:"sequence,omitempty"`
	State       string `json:"state"`
	Timestamp   string `json:"timestamp,omitempty"`
}

type Receipt struct {
	CandidateDigest string       `json:"candidate_digest"`
	Commit          CommitResult `json:"commit"`
	Disposition     string       `json:"disposition"`
	ReasonCode      string       `json:"reason_code"`
	RequestID       string       `json:"request_id"`
	Schema          string       `json:"schema"`
	TOPSID          string       `json:"tops_id"`
}

type Query struct {
	AfterSequence   uint64   `json:"after_sequence"`
	CorrelationID   string   `json:"correlation_id,omitempty"`
	EventClasses    []string `json:"event_classes"`
	FromTime        string   `json:"from_time,omitempty"`
	Limit           uint64   `json:"limit"`
	Outcomes        []string `json:"outcomes"`
	RequestID       string   `json:"request_id,omitempty"`
	Schema          string   `json:"schema"`
	ThroughSequence *uint64  `json:"through_sequence,omitempty"`
	ThroughTime     string   `json:"through_time,omitempty"`
	TOPSID          string   `json:"tops_id"`
}

type QueryProjection struct {
	Actor         SafeReference `json:"actor"`
	CorrelationID string        `json:"correlation_id"`
	EventClass    string        `json:"event_class"`
	OperationID   string        `json:"operation_id"`
	Outcome       string        `json:"outcome"`
	ReasonCode    string        `json:"reason_code"`
	RequestID     string        `json:"request_id"`
	Target        SafeReference `json:"target"`
	Timestamp     string        `json:"timestamp"`
}

type QueryEntry struct {
	EventDigest       string          `json:"event_digest"`
	EventID           string          `json:"event_id"`
	Projection        QueryProjection `json:"projection"`
	RedactionState    string          `json:"redaction_state"`
	Sequence          uint64          `json:"sequence"`
	VerificationState string          `json:"verification_state"`
}

type QueryNext struct {
	AfterSequence uint64 `json:"after_sequence,omitempty"`
	State         string `json:"state"`
}

type QueryPage struct {
	AfterSequence uint64       `json:"after_sequence"`
	Entries       []QueryEntry `json:"entries"`
	Next          QueryNext    `json:"next"`
	Schema        string       `json:"schema"`
	TOPSID        string       `json:"tops_id"`
}

type VerificationResult struct {
	AtSequence uint64 `json:"at_sequence,omitempty"`
	ReasonCode string `json:"reason_code,omitempty"`
	State      string `json:"state"`
}

type Verification struct {
	AfterSequence   uint64             `json:"after_sequence"`
	EventsChecked   uint64             `json:"events_checked"`
	Result          VerificationResult `json:"result"`
	Schema          string             `json:"schema"`
	ThroughSequence uint64             `json:"through_sequence"`
	TOPSID          string             `json:"tops_id"`
}
