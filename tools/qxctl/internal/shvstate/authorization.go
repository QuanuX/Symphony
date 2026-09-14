package shvstate

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/ssiagclient"
)

// These are retained-evidence consistency checks, not authentication. The CLI
// must obtain the decision through authenticated SSIAG and validate its request.
// Historic decisions remain valid evidence after expiry; publication does not.
func canonicalAuthorization(raw json.RawMessage, s Store, a Attempt) (json.RawMessage, error) {
	var d ssiagclient.AuthorizationDecision
	if decode(raw, &d) != nil {
		return nil, fmt.Errorf("invalid SSIAG decision shape")
	}
	var object any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if dec.Decode(&object) != nil || !sameValue(object, d) {
		return nil, fmt.Errorf("SSIAG decision lacks exact required shape")
	}
	var p retainedPlan
	if json.Unmarshal(a.Intent.Plan, &p) != nil {
		return nil, fmt.Errorf("invalid SHV plan")
	}
	action := map[string]string{"onboard": "symphony.shv.source.onboard", "relocation": "symphony.shv.source.relocation", "authority_change": "symphony.shv.source.authority-change"}[p.ChangeKind]
	if action == "" || d.Schema != "symphony.ssiag.authorization-decision.v1" || d.Effect != "allow" || d.ReasonCode != "symphony.ssiag.policy.exact-grant" || d.DecisionID == "" || stavprotocol.ValidateRequestUUID(d.RequestID) != nil || d.CorrelationID != a.CorrelationID || d.TOPSID != s.TOPSID || d.Target.Operation != action || d.Target.Resource != resource(s.TOPSID, s.SourceID) || d.Target.Audience != "qxctl" || d.Target.Scope != "tops:"+s.TOPSID || d.CallerClassUsed || d.CanonicalApply || d.AuthorityBasis == nil || d.Capability == nil || d.ExpiresAt == nil || !hashPattern.MatchString(d.PolicyDigest) || !hashPattern.MatchString(d.ConfigDigest) {
		return nil, fmt.Errorf("SSIAG decision does not bind retained SHV authority target")
	}
	c := d.Capability
	basis := *d.AuthorityBasis
	if c.Protocol != "symphony.ssiag.capability.v1" || c.TOPSID != d.TOPSID || c.Subject != d.Subject || c.Subject.ID == "" || c.Subject.Kind == "" || c.Subject.Authority != "unix_peer_credentials" || c.Target != d.Target || c.RequestID != d.RequestID || c.CorrelationID != a.CorrelationID || c.AuthorityBasis != basis || (basis != "host_owner" && basis != "granted_permission") || c.GrantID == "" || c.PolicyDigest != d.PolicyDigest || c.ConfigDigest != d.ConfigDigest || c.Transferable || c.CanonicalApply || c.IssuedAt != d.DecidedAt || c.ExpiresAt != *d.ExpiresAt || !c.ExpiresAt.After(c.IssuedAt) || c.IssuedAt.Location() != time.UTC || c.ExpiresAt.Location() != time.UTC {
		return nil, fmt.Errorf("SSIAG retained capability correspondence mismatch")
	}
	binding := capabilityBinding(*c)
	if c.BindingDigest != binding || c.CapabilityID != "ssiag-capability:"+strings.TrimPrefix(binding, "sha256:") {
		return nil, fmt.Errorf("SSIAG retained capability binding mismatch")
	}
	normalizedDecision, e := normalized(d)
	if e != nil {
		return nil, e
	}
	result, e := knowledgeengine.SCVCanonical(normalizedDecision)
	return result, e
}
func capabilityBinding(c ssiagclient.Capability) string {
	joined := strings.Join([]string{c.Protocol, c.Subject.ID, c.Subject.Kind, c.Subject.Authority, c.TOPSID, c.Target.Operation, c.Target.Resource, c.Target.Audience, c.Target.Scope, c.AuthorityBasis, c.GrantID, c.RequestID, c.CorrelationID, c.IssuedAt.UTC().Format(time.RFC3339), c.ExpiresAt.UTC().Format(time.RFC3339), c.PolicyDigest, c.ConfigDigest, "transferable=false", "canonical_apply=false"}, "\n")
	h := sha256.Sum256([]byte(joined))
	return "sha256:" + hex.EncodeToString(h[:])
}
func authorizationFresh(raw json.RawMessage) error {
	var d ssiagclient.AuthorizationDecision
	if json.Unmarshal(raw, &d) != nil || d.ExpiresAt == nil || d.Capability == nil || !d.ExpiresAt.After(time.Now().UTC()) || !d.Capability.ExpiresAt.After(time.Now().UTC()) {
		return fmt.Errorf("SHV source authorization expired before publication")
	}
	return nil
}
