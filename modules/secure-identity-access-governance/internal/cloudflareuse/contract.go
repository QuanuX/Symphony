// Package cloudflareuse owns the nonsecret admission envelope. Credential bytes
// never enter SSIAG's Go process. A native provider holds the admitted connection
// until its exact one-shot use and cleanup finish.
package cloudflareuse

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	stav "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"io"
	"regexp"
	"strconv"
	"time"
	"unicode/utf8"
)

const Protocol = "symphony.ssiag.cloudflare-use.v1"
const Endpoint = "/v1/cloudflare/use"

type Request struct {
	Protocol        string  `json:"protocol"`
	RequestID       string  `json:"request_id"`
	TOPSID          string  `json:"tops_id"`
	PlanDigest      string  `json:"plan_digest"`
	AdapterDigest   string  `json:"adapter_digest"`
	ToolchainDigest string  `json:"toolchain_digest"`
	CredentialRef   string  `json:"credential_ref"`
	ResultRef       *string `json:"result_ref"`
	Operation       string  `json:"operation"`
	AccountID       string  `json:"account_id"`
	ZoneID          *string `json:"zone_id"`
	ResourceID      *string `json:"resource_id"`
	DeadlineUnixMS  string  `json:"deadline_unix_ms"`
}
type Completion struct {
	Protocol      string `json:"protocol"`
	RequestDigest string `json:"request_digest"`
	Outcome       string `json:"outcome"`
}

var hash = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
var id = regexp.MustCompile(`^[a-f0-9]{32}$`)
var resource = regexp.MustCompile(`^([a-f0-9]{32}|[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})$`)
var ref = regexp.MustCompile(`^ssiag-native:[a-z0-9][a-z0-9_-]{0,63}$`)
var operations = map[string]bool{}

func init() {
	for _, group := range []string{"tunnels", "dns.records", "zero-trust.access.applications", "zero-trust.access.policies"} {
		for _, verb := range []string{"list", "get", "create", "update", "delete"} {
			operations[group+"."+verb] = true
		}
	}
	delete(operations, "tunnels.update")
	for _, op := range []string{"tunnels.edit", "tunnels.config.get", "tunnels.config.update", "tunnels.token.get", "connector.start"} {
		operations[op] = true
	}
}
func (r Request) Deadline() time.Time {
	n, _ := strconv.ParseInt(r.DeadlineUnixMS, 10, 64)
	return time.UnixMilli(n)
}
func (r Request) Validate(now time.Time) error {
	if r.Protocol != Protocol || stav.ValidateRequestUUID(r.RequestID) != nil || stav.ValidateTOPSID(r.TOPSID) != nil || !hash.MatchString(r.PlanDigest) || !hash.MatchString(r.AdapterDigest) || !hash.MatchString(r.ToolchainDigest) || !ref.MatchString(r.CredentialRef) || !operations[r.Operation] || !id.MatchString(r.AccountID) || (r.ZoneID != nil && !id.MatchString(*r.ZoneID)) || (r.ResourceID != nil && !resource.MatchString(*r.ResourceID)) {
		return fmt.Errorf("cloudflare_use.invalid")
	}
	if r.ResultRef != nil && (!ref.MatchString(*r.ResultRef) || *r.ResultRef == r.CredentialRef || (r.Operation != "tunnels.token.get" && r.Operation != "tunnels.create" && r.Operation != "tunnels.get")) {
		return fmt.Errorf("cloudflare_use.result_binding")
	}
	n, e := strconv.ParseInt(r.DeadlineUnixMS, 10, 64)
	if e != nil || strconv.FormatInt(n, 10) != r.DeadlineUnixMS || !r.Deadline().After(now) || r.Deadline().Sub(now) > 60*time.Second {
		return fmt.Errorf("cloudflare_use.expired")
	}
	return nil
}
func (r Request) Resource() string {
	zone, object := "account", "collection"
	if r.ZoneID != nil {
		zone = *r.ZoneID
	}
	if r.ResourceID != nil {
		object = *r.ResourceID
	}
	return "cloudflare:" + r.AccountID + ":" + zone + ":" + object
}
func Hash(v any) string {
	raw, _ := json.Marshal(v)
	b, _ := canonical(raw)
	return fmt.Sprintf("sha256:%x", sha256.Sum256(append([]byte(Protocol+"\x00"), b...)))
}
func Decode(b []byte, v any) error {
	if len(b) > 32768 {
		return fmt.Errorf("cloudflare_use.limit")
	}
	normalized, e := canonical(b)
	if e != nil {
		return e
	}
	d := json.NewDecoder(bytes.NewReader(normalized))
	d.DisallowUnknownFields()
	if e = d.Decode(v); e != nil {
		return e
	}
	raw, e := json.Marshal(v)
	if e != nil {
		return e
	}
	typed, e := canonical(raw)
	if e != nil || !bytes.Equal(typed, normalized) {
		return fmt.Errorf("cloudflare_use.fields")
	}
	return nil
}

func OutcomeID(r Request) string {
	b := sha256.Sum256([]byte("symphony.ssiag.cloudflare-outcome.v1\x00" + Hash(r)))
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// This flat control envelope permits explicit null selections, unlike STAV
// candidate documents. Only strings/null are admitted and duplicate keys fail.
func canonical(b []byte) ([]byte, error) {
	if !utf8.Valid(b) {
		return nil, fmt.Errorf("invalid UTF-8")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	t, e := d.Token()
	if e != nil || t != json.Delim('{') {
		return nil, fmt.Errorf("object required")
	}
	m := map[string]any{}
	for d.More() {
		k, e := d.Token()
		if e != nil {
			return nil, e
		}
		key, ok := k.(string)
		if !ok {
			return nil, fmt.Errorf("key")
		}
		if _, ok = m[key]; ok {
			return nil, fmt.Errorf("duplicate")
		}
		v, e := d.Token()
		if e != nil {
			return nil, e
		}
		if v != nil {
			if _, ok := v.(string); !ok {
				return nil, fmt.Errorf("string required")
			}
		}
		m[key] = v
	}
	if t, e = d.Token(); e != nil || t != json.Delim('}') {
		return nil, fmt.Errorf("object end")
	}
	if _, e = d.Token(); e != io.EOF {
		return nil, fmt.Errorf("trailing value")
	}
	return json.Marshal(m)
}
