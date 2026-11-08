package profilewire

import (
	"bytes"
	"errors"
	"math"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var ErrBinding = errors.New("transport_profile.invalid_binding")
var uuid = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
var tagged = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
var name = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)
var version = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[a-z0-9.]+)?$`)
var hostname = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9.-]{0,251}[a-z0-9])?$`)
var user = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,63}$`)
var session = regexp.MustCompile(`^session:[a-zA-Z0-9][a-zA-Z0-9._:-]{0,127}$`)

func ValidUUID(s string) bool   { return uuid.MatchString(s) }
func ValidDigest(s string) bool { return tagged.MatchString(s) }
func ValidScope(s Scope) bool   { return ValidUUID(s.HostID) && ValidUUID(s.TOPSID) }
func Decimal(s string, max uint64, zero bool) bool {
	n, e := strconv.ParseUint(s, 10, 64)
	return e == nil && strconv.FormatUint(n, 10) == s && n <= max && (zero || n > 0)
}
func ValidPath(s string) bool {
	return s != "" && len(s) <= 4096 && !path.IsAbs(s) && path.Clean(s) == s && s != "." && s != ".." && !strings.HasPrefix(s, "../") && !strings.ContainsAny(s, "\\\x00\r\n\t")
}
func next(s string) (string, error) {
	if !Decimal(s, math.MaxUint64, false) {
		return "", ErrBinding
	}
	n, _ := strconv.ParseUint(s, 10, 64)
	if n == math.MaxUint64 {
		return "", errors.New("transport_profile.generation_exhausted")
	}
	return strconv.FormatUint(n+1, 10), nil
}
func validPackage(p PackageRef) bool {
	return ValidUUID(p.PackageID) && version.MatchString(p.Version) && ValidDigest(p.ManifestDigest) && ValidDigest(p.ReceiptDigest) && ValidDigest(p.ExecutableDigest)
}
func ValidPrincipals(p []Principal) bool {
	if len(p) == 0 {
		return false
	}
	seen := map[Principal]bool{}
	for _, v := range p {
		// Reject POSIX ownership sentinel as well as root. Supplementary groups
		// and environment claims do not confer profile authority.
		if !Decimal(v.UID, math.MaxUint32-1, false) || !Decimal(v.GID, math.MaxUint32-1, false) || seen[v] {
			return false
		}
		seen[v] = true
	}
	return true
}
func ValidConfiguration(c Configuration) bool {
	if !name.MatchString(c.Name) || c.TransportKind != "cloudflare-access" || c.ExecutionProfile != "cloudflare-access-user-session.v1" || c.AuthenticationMode != "user_session" || !session.MatchString(c.SessionRef) {
		return false
	}
	switch c.Platform {
	case "darwin-amd64", "darwin-arm64", "linux-amd64", "linux-arm64":
	default:
		return false
	}
	if !validPackage(c.Adapter) || !validPackage(c.SSH) || !validPackage(c.Helper) {
		return false
	}
	if c.Limits.ControlBytes != "8388608" || !Decimal(c.Limits.CapturedBytes, MaxDocumentBytes, false) || c.Limits.CleanupSeconds != "5" || !c.Limits.CallerTimeout {
		return false
	}
	if len(c.Applications) == 0 || len(c.Endpoints) == 0 {
		return false
	}
	seen := map[string]bool{}
	for _, h := range c.Applications {
		if !validHostname(h) || seen[h] {
			return false
		}
		seen[h] = true
	}
	seen = map[string]bool{}
	for _, e := range c.Endpoints {
		if !validHostname(e.Host) || !Decimal(e.Port, 65535, false) || !user.MatchString(e.User) || !ValidDigest(e.HostKeyDigest) {
			return false
		}
		key := e.Host + ":" + e.Port + ":" + e.User + ":" + e.HostKeyDigest
		if seen[key] {
			return false
		}
		seen[key] = true
	}
	want := []string{"symphony.sov.ssh-endpoint.v2", "symphony.sov.ssh-invoke-input.v2", "symphony.sov.ssh-carrier-result.v2"}
	a, _ := Canonical(c.Compatibility)
	b, _ := Canonical(want)
	return bytes.Equal(a, b)
}
func ValidateSnapshot(s Snapshot) error {
	p, q := s.Profile, s.Policy
	if s.Protocol != SnapshotProtocol || p.Protocol != ProfileProtocol || q.Protocol != PolicyProtocol || !ValidScope(p.Scope) || q.Scope != p.Scope || !ValidUUID(p.ProfileID) || q.ProfileID != p.ProfileID || !Decimal(p.Generation, math.MaxUint64, false) || !Decimal(q.Generation, math.MaxUint64, false) || q.ProfileGeneration != p.Generation || p.Policy.Generation != q.Generation {
		return ErrBinding
	}
	if p.State != "enabled" && p.State != "disabled" && p.State != "removed" {
		return ErrBinding
	}
	if !ValidConfiguration(p.Configuration) || !ValidPrincipals(q.Permitted) || q.Digest != Digest(PolicyProtocol, q) || p.Policy.Digest != q.Digest || p.Digest != Digest(ProfileProtocol, p) || s.Digest != Digest(SnapshotProtocol, s) {
		return ErrBinding
	}
	return nil
}
func SealSnapshot(s *Snapshot) {
	s.Policy.Digest = Digest(PolicyProtocol, s.Policy)
	s.Profile.Policy = PolicyRef{Generation: s.Policy.Generation, Digest: s.Policy.Digest}
	s.Profile.Digest = Digest(ProfileProtocol, s.Profile)
	s.Digest = Digest(SnapshotProtocol, s)
}
func Prior(s *Snapshot) *Expected {
	if s == nil {
		return nil
	}
	return &Expected{ProfileGeneration: s.Profile.Generation, PolicyGeneration: s.Policy.Generation, SnapshotDigest: s.Digest}
}
func Same(a, b any) bool {
	x, e := Canonical(a)
	y, f := Canonical(b)
	return e == nil && f == nil && bytes.Equal(x, y)
}

func Desired(scope Scope, id string, current *Snapshot, change Change) (Snapshot, error) {
	if !ValidScope(scope) || !ValidUUID(id) {
		return Snapshot{}, ErrBinding
	}
	var result Snapshot
	if current == nil {
		if change.Action != "create" || change.Configuration == nil || change.Permitted == nil {
			return result, ErrBinding
		}
		result = Snapshot{Protocol: SnapshotProtocol, Profile: Profile{Protocol: ProfileProtocol, Scope: scope, ProfileID: id, Generation: "1", State: "disabled", Configuration: *change.Configuration}, Policy: Policy{Protocol: PolicyProtocol, Scope: scope, ProfileID: id, Generation: "1", ProfileGeneration: "1", Permitted: *change.Permitted}}
	} else {
		if ValidateSnapshot(*current) != nil || current.Profile.Scope != scope || current.Profile.ProfileID != id || current.Profile.State == "removed" {
			return result, ErrBinding
		}
		result = *current
		pg, e := next(result.Profile.Generation)
		if e != nil {
			return result, e
		}
		qg, e := next(result.Policy.Generation)
		if e != nil {
			return result, e
		}
		result.Profile.Generation = pg
		result.Policy.Generation = qg
		result.Policy.ProfileGeneration = pg
		switch change.Action {
		case "replace":
			if change.Configuration == nil || change.Permitted != nil || change.Configuration.Name != result.Profile.Configuration.Name {
				return result, ErrBinding
			}
			result.Profile.Configuration = *change.Configuration
		case "policy":
			if change.Configuration != nil || change.Permitted == nil {
				return result, ErrBinding
			}
			result.Policy.Permitted = *change.Permitted
		case "enable", "disable", "remove":
			if change.Configuration != nil || change.Permitted != nil {
				return result, ErrBinding
			}
			if change.Action == "enable" {
				if result.Profile.State != "disabled" {
					return result, ErrBinding
				}
				result.Profile.State = "enabled"
			}
			if change.Action == "disable" {
				if result.Profile.State != "enabled" {
					return result, ErrBinding
				}
				result.Profile.State = "disabled"
			}
			if change.Action == "remove" {
				result.Profile.State = "removed"
			}
		default:
			return result, ErrBinding
		}
	}
	SealSnapshot(&result)
	return result, ValidateSnapshot(result)
}
func MakePlan(scope Scope, id, operation, administrator string, current *Snapshot, change Change, now time.Time) (Plan, error) {
	if !ValidRequestID(operation) || !ValidDigest(administrator) {
		return Plan{}, ErrBinding
	}
	desired, e := Desired(scope, id, current, change)
	if e != nil {
		return Plan{}, e
	}
	now = now.UTC().Truncate(time.Second)
	p := Plan{Protocol: PlanProtocol, AdministratorDigest: administrator, OperationID: operation, Scope: scope, ProfileID: id, Expected: Prior(current), Change: change, Desired: desired, CreatedAt: now.Format(time.RFC3339), ExpiresAt: now.Add(5 * time.Minute).Format(time.RFC3339)}
	p.Digest = Digest(PlanProtocol, p)
	return p, nil
}
func ValidatePlan(p Plan, current *Snapshot, now time.Time, allowExpired bool) error {
	created, e := time.Parse(time.RFC3339, p.CreatedAt)
	expires, f := time.Parse(time.RFC3339, p.ExpiresAt)
	if p.Protocol != PlanProtocol || !ValidDigest(p.AdministratorDigest) || !ValidRequestID(p.OperationID) || e != nil || f != nil || created.UTC().Format(time.RFC3339) != p.CreatedAt || expires.UTC().Format(time.RFC3339) != p.ExpiresAt || !expires.Equal(created.Add(5*time.Minute)) || created.After(now.Add(30*time.Second)) || (!allowExpired && !expires.After(now)) || !Same(p.Expected, Prior(current)) || p.Digest != Digest(PlanProtocol, p) {
		return ErrBinding
	}
	want, e := Desired(p.Scope, p.ProfileID, current, p.Change)
	if e != nil || !Same(want, p.Desired) {
		return ErrBinding
	}
	return nil
}
func ValidateRequest(r Request) error {
	if r.Protocol != RequestProtocol || !ValidRequestID(r.RequestID) || !ValidScope(r.Scope) {
		return ErrBinding
	}
	if r.ProfileID != nil && !ValidUUID(*r.ProfileID) || r.OperationID != nil && !ValidRequestID(*r.OperationID) {
		return ErrBinding
	}
	switch r.Operation {
	case "list":
		if r.ProfileID != nil || r.OperationID != nil || r.Change != nil || r.Plan != nil || r.Recovery != nil {
			return ErrBinding
		}
	case "inspect":
		if r.ProfileID == nil || r.OperationID != nil || r.Change != nil || r.Plan != nil || r.Recovery != nil {
			return ErrBinding
		}
	case "plan":
		if r.ProfileID == nil || r.OperationID == nil || r.Change == nil || r.Plan != nil || r.Recovery != nil {
			return ErrBinding
		}
	case "apply":
		if r.ProfileID != nil || r.OperationID != nil || r.Change != nil || r.Plan == nil || r.Recovery != nil || r.Plan.Scope != r.Scope {
			return ErrBinding
		}
	case "status":
		if r.ProfileID != nil || r.OperationID == nil || r.Change != nil || r.Plan != nil || r.Recovery != nil {
			return ErrBinding
		}
	case "recover":
		if r.ProfileID != nil || r.OperationID != nil || r.Change != nil || r.Plan != nil || r.Recovery == nil || !ValidRequestID(r.Recovery.OperationID) || !ValidDigest(r.Recovery.PlanDigest) || !ValidDigest(r.Recovery.ExpectedPendingDigest) || (r.Recovery.Action != "complete" && r.Recovery.Action != "abort") {
			return ErrBinding
		}
	default:
		return ErrBinding
	}
	return nil
}

func ValidRequestID(s string) bool { return ValidUUID(s) && (s[14] == '4' || s[14] == '7') }

func validHostname(h string) bool {
	if !hostname.MatchString(h) {
		return false
	}
	for _, label := range strings.Split(h, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
	}
	return true
}
