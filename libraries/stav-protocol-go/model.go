package stavprotocol

const (
	SchemaCandidate             = "symphony.stav.candidate.v1"
	SchemaEvent                 = "symphony.stav.event.v1"
	SchemaReceipt               = "symphony.stav.receipt.v1"
	SchemaQuery                 = "symphony.stav.query.v1"
	SchemaQueryPage             = "symphony.stav.query-page.v1"
	SchemaVerification          = "symphony.stav.verification.v1"
	SchemaAppendAuthorityConfig = "symphony.stav.append-authority.config.v1"
	SchemaAppendAuthorityStatus = "symphony.stav.append-authority.status.v1"

	SchemaLocalRequest  = "symphony.stav.local.request.v1"
	SchemaLocalResponse = "symphony.stav.local.response.v1"
)

const (
	LocalOperationAppend = "append"
	LocalOperationQuery  = "query"
	LocalOperationStatus = "status"
	LocalOperationVerify = "verify"
)

const (
	LocalDispositionSucceeded = "succeeded"
	LocalDispositionRejected  = "rejected"
)

const (
	ReasonReceiptCommitted           = "symphony.stav.receipt.committed"
	ReasonReceiptRejected            = "symphony.stav.receipt.rejected"
	ReasonReceiptIdempotencyConflict = "symphony.stav.receipt.idempotency-conflict"
	ReasonReceiptEventClassDenied    = "symphony.stav.receipt.event-class-denied"
	ReasonReceiptOperationDenied     = "symphony.stav.receipt.operation-denied"
	ReasonReceiptTOPSMismatch        = "symphony.stav.receipt.tops-mismatch"
	ReasonReceiptLedgerFull          = "symphony.stav.receipt.ledger-full"
	ReasonReceiptLedgerUnavailable   = "symphony.stav.receipt.ledger-unavailable"
	ReasonResponseSucceeded          = "symphony.stav.response.succeeded"
	ReasonResponseInvalidRequest     = "symphony.stav.response.invalid-request"
	ReasonResponseUnauthorizedPeer   = "symphony.stav.response.unauthorized-peer"
	ReasonResponseOperationDenied    = "symphony.stav.response.operation-denied"
	ReasonResponseLedgerUnavailable  = "symphony.stav.response.ledger-unavailable"
	ReasonResponseLedgerFull         = "symphony.stav.response.ledger-full"
	ReasonResponseInternalFailure    = "symphony.stav.response.internal-failure"
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

type PeerPermission struct {
	EventClass  string `json:"event_class"`
	OperationID string `json:"operation_id"`
}

type ProducerGrant struct {
	GID         uint64           `json:"gid"`
	Permissions []PeerPermission `json:"permissions"`
	Producer    SafeReference    `json:"producer"`
	Subject     SafeReference    `json:"subject"`
	UID         uint64           `json:"uid"`
}

type ReaderGrant struct {
	Classifications []string      `json:"classifications"`
	GID             uint64        `json:"gid"`
	Subject         SafeReference `json:"subject"`
	UID             uint64        `json:"uid"`
}

type AuthorityGrant struct {
	GID     uint64        `json:"gid"`
	Subject SafeReference `json:"subject"`
	UID     uint64        `json:"uid"`
}

type AppendAuthorityAuthentication struct {
	Authority AuthorityGrant  `json:"authority"`
	Mechanism string          `json:"mechanism"`
	Producers []ProducerGrant `json:"producers"`
	Readers   []ReaderGrant   `json:"readers"`
}

type AppendAuthorityLedger struct {
	Durability string `json:"durability"`
	MaxBytes   uint64 `json:"max_bytes"`
	Path       string `json:"path"`
	Recovery   string `json:"recovery"`
	Retention  string `json:"retention"`
	Rotation   string `json:"rotation"`
}

type AppendAuthorityListen struct {
	Address string `json:"address"`
	Network string `json:"network"`
}

type AppendAuthorityConfig struct {
	Authentication AppendAuthorityAuthentication `json:"authentication"`
	Ledger         AppendAuthorityLedger         `json:"ledger"`
	Listen         AppendAuthorityListen         `json:"listen"`
	Mode           string                        `json:"mode"`
	Schema         string                        `json:"schema"`
	TOPSID         string                        `json:"tops_id"`
}

type AppendAuthorityStatus struct {
	Events          uint64 `json:"events"`
	GenesisDigest   string `json:"genesis_digest"`
	LastEventDigest string `json:"last_event_digest,omitempty"`
	LedgerBytes     uint64 `json:"ledger_bytes"`
	LastSequence    uint64 `json:"last_sequence"`
	MaxLedgerBytes  uint64 `json:"max_ledger_bytes"`
	Mode            string `json:"mode"`
	Ready           bool   `json:"ready"`
	RecoveredTail   bool   `json:"recovered_tail"`
	Schema          string `json:"schema"`
	StorageState    string `json:"storage_state"`
	TOPSID          string `json:"tops_id"`
}

type VerifyRequest struct {
	AfterSequence   uint64  `json:"after_sequence"`
	ThroughSequence *uint64 `json:"through_sequence,omitempty"`
}

type LocalRequest struct {
	Candidate *Candidate     `json:"candidate,omitempty"`
	Operation string         `json:"operation"`
	Query     *Query         `json:"query,omitempty"`
	RequestID string         `json:"request_id"`
	Schema    string         `json:"schema"`
	TOPSID    string         `json:"tops_id"`
	Verify    *VerifyRequest `json:"verify,omitempty"`
}

type LocalResponse struct {
	Disposition  string                 `json:"disposition"`
	Operation    string                 `json:"operation"`
	Page         *QueryPage             `json:"page,omitempty"`
	ReasonCode   string                 `json:"reason_code"`
	Receipt      *Receipt               `json:"receipt,omitempty"`
	RequestID    string                 `json:"request_id"`
	Schema       string                 `json:"schema"`
	Status       *AppendAuthorityStatus `json:"status,omitempty"`
	TOPSID       string                 `json:"tops_id"`
	Verification *Verification          `json:"verification,omitempty"`
}
