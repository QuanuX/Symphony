package profilewire

import (
	"runtime"
	"strings"
)

var Operations = []string{"list", "inspect", "schema", "template", "plan", "apply", "status", "recover"}

// Template contains synthetic references, never discovered identities or credentials.
func Template() Request {
	id, op := "33333333-3333-4333-8333-333333333333", "44444444-4444-4444-8444-444444444444"
	digest := "sha256:" + strings.Repeat("0", 64)
	pkg := PackageRef{PackageID: "66666666-6666-4666-8666-666666666666", Version: Version, ManifestDigest: digest, ReceiptDigest: digest, ExecutableDigest: digest}
	config := Configuration{Name: "example", TransportKind: "cloudflare-access", Platform: runtime.GOOS + "-" + runtime.GOARCH, ExecutionProfile: "cloudflare-access-user-session.v1", Adapter: pkg, SSH: pkg, Helper: pkg, Applications: []string{"access.example.invalid"}, Endpoints: []EndpointConstraint{{Host: "ssh.example.invalid", Port: "22", User: "example", HostKeyDigest: digest}}, AuthenticationMode: "user_session", SessionRef: "session:example", Limits: Limits{ControlBytes: "8388608", CapturedBytes: "1048576", CleanupSeconds: "5", CallerTimeout: true}, Compatibility: []string{"symphony.sov.ssh-endpoint.v2", "symphony.sov.ssh-invoke-input.v2", "symphony.sov.ssh-carrier-result.v2"}}
	permitted := []Principal{{UID: "501", GID: "20"}}
	return Request{Protocol: RequestProtocol, RequestID: "77777777-7777-4777-8777-777777777777", Operation: "plan", Scope: Scope{HostID: "11111111-1111-4111-8111-111111111111", TOPSID: "22222222-2222-4222-8222-222222222222"}, ProfileID: &id, OperationID: &op, Change: &Change{Action: "create", Configuration: &config, Permitted: &permitted}}
}
func ValidateResult(r Result, input Request) error {
	if r.Protocol != ResultProtocol || r.RequestID != input.RequestID || r.Operation != input.Operation || r.Scope != input.Scope || r.Profiles == nil {
		return ErrBinding
	}
	switch r.Code {
	case "transport_profile.completed", "transport_profile.recovery_required", "transport_profile.rejected", "transport_profile.root_required", "transport_profile.invalid_binding", "transport_profile.invalid_document", "transport_profile.generation_exhausted", "transport_profile.untrusted_store", "transport_profile.stale_state", "transport_profile.name_reserved", "transport_profile.busy", "transport_profile.attempt_exists", "transport_profile.audit_unavailable", "transport_profile.storage_unavailable":
	default:
		return ErrBinding
	}
	if r.Completed && r.Code != "transport_profile.completed" && r.Code != "transport_profile.recovery_required" {
		return ErrBinding
	}
	if !r.Completed && r.Code == "transport_profile.completed" {
		return ErrBinding
	}
	if r.NextAction != "none" && r.NextAction != "inspect" && r.NextAction != "recover" {
		return ErrBinding
	}
	for _, s := range r.Profiles {
		if ValidateSnapshot(s) != nil || s.Profile.Scope != input.Scope {
			return ErrBinding
		}
	}
	if r.Snapshot != nil && (ValidateSnapshot(*r.Snapshot) != nil || input.ProfileID == nil || r.Snapshot.Profile.ProfileID != *input.ProfileID || r.Snapshot.Profile.Scope != input.Scope) {
		return ErrBinding
	}
	if r.Plan != nil {
		p := r.Plan
		if input.Operation != "plan" || input.ProfileID == nil || input.OperationID == nil || p.Protocol != PlanProtocol || p.Scope != input.Scope || p.ProfileID != *input.ProfileID || p.OperationID != *input.OperationID || p.Digest != Digest(PlanProtocol, *p) || !ValidDigest(p.AdministratorDigest) || !Same(p.Change, input.Change) || ValidateSnapshot(p.Desired) != nil || p.Desired.Profile.Scope != input.Scope || p.Desired.Profile.ProfileID != p.ProfileID {
			return ErrBinding
		}
	}
	if r.Completed && input.Operation == "plan" && r.Plan == nil {
		return ErrBinding
	}
	if r.Attempt != nil {
		a := r.Attempt
		if !ValidRequestID(a.OperationID) || !ValidDigest(a.PlanDigest) || a.PendingDigest != nil && !ValidDigest(*a.PendingDigest) || a.AuditIntentDigest != nil && !ValidDigest(*a.AuditIntentDigest) || a.AuditOutcomeDigest != nil && !ValidDigest(*a.AuditOutcomeDigest) {
			return ErrBinding
		}
		if a.Stage != "prepared" && a.Stage != "admitted" && a.Stage != "published" && a.Stage != "settled" {
			return ErrBinding
		}
		if a.Outcome != "pending" && a.Outcome != "published" && a.Outcome != "aborted" {
			return ErrBinding
		}
	}
	if r.Completed && (input.Operation == "apply" || input.Operation == "recover") {
		if r.Attempt == nil || r.Attempt.Stage != "settled" || r.Attempt.PendingDigest != nil || r.Attempt.AuditIntentDigest == nil || r.Attempt.AuditOutcomeDigest == nil {
			return ErrBinding
		}
		op, digest := "", ""
		if input.Plan != nil {
			op, digest = input.Plan.OperationID, input.Plan.Digest
			if r.Attempt.Outcome != "published" {
				return ErrBinding
			}
		} else {
			op, digest = input.Recovery.OperationID, input.Recovery.PlanDigest
		}
		if r.Attempt.OperationID != op || r.Attempt.PlanDigest != digest {
			return ErrBinding
		}
	}
	return nil
}
