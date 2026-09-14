package shvpublicationstate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
)

var hashPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

func cleanRoot(s string) bool {
	if !utf8.ValidString(s) || len(s) > 4096 || !filepath.IsAbs(s) || filepath.Clean(s) != s {
		return false
	}
	for _, c := range s {
		if c < 32 || c == 127 {
			return false
		}
	}
	return true
}
func sameValue(a, b any) bool {
	na, ae := normalized(a)
	nb, be := normalized(b)
	if ae != nil || be != nil {
		return false
	}
	x, e := knowledgeengine.SCVCanonical(na)
	y, f := knowledgeengine.SCVCanonical(nb)
	return e == nil && f == nil && bytes.Equal(x, y)
}

type retainedHead struct {
	Generation int64   `json:"generation"`
	Previous   *string `json:"previous_digest"`
	Digest     string  `json:"digest"`
	Definition struct {
		CatalogueID string `json:"catalogue_id"`
		TOPSID      string `json:"tops_id"`
	} `json:"definition"`
}
type retainedTransition struct {
	Protocol    string          `json:"protocol"`
	OperationID string          `json:"operation_id"`
	Expected    *string         `json:"expected_state_digest"`
	Head        json.RawMessage `json:"head"`
	Digest      string          `json:"digest"`
}
type retainedPlan struct {
	Protocol    string          `json:"protocol"`
	OperationID string          `json:"operation_id"`
	Expected    *string         `json:"expected_state_digest"`
	ChangeKind  string          `json:"change_kind"`
	Reason      string          `json:"reason"`
	Head        json.RawMessage `json:"head"`
	Digest      string          `json:"digest"`
}

func decode(raw []byte, v any) error {
	if e := knowledgeengine.ValidateSCVBundleText(raw); e != nil {
		return e
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return e
	}
	var trailing any
	if e := d.Decode(&trailing); e != io.EOF {
		return fmt.Errorf("trailing JSON in SHV retained object")
	}
	return nil
}
func validateIntentShape(i Intent) error {
	if !tokenPattern.MatchString(i.OperationID) {
		return fmt.Errorf("invalid SHV operation identity")
	}
	digest, e := seal(i)
	if e != nil || digest != i.Digest {
		return fmt.Errorf("SHV intent seal mismatch")
	}
	x := i.Installation
	expectedReceipt := filepath.Join(x.Prefix, "share/symphony/receipts/shv-publication-engine/0.1.0-dev/install-receipt.json")
	expectedExecutable := filepath.Join(x.Prefix, "libexec/symphony/shv-publication-engine/0.1.0-dev/symphony-shv-publication")
	if x.Role != "shv-publication-engine" || x.ModuleID != "shv-publication-engine" || x.EngineID != "symphony-shv-publication" || x.Version != knowledgeengine.SHVPublicationVersion || !cleanRoot(x.Prefix) || x.ReceiptPath != expectedReceipt || x.ExecutablePath != expectedExecutable || x.ReceiptProtocol != "symphony.knowledge.install-receipt.v2" || !hashPattern.MatchString(x.ReceiptDigest) || !hashPattern.MatchString(x.ExecutableDigest) {
		return fmt.Errorf("SHV intent lacks exact historical owner installation")
	}
	var p retainedPlan
	var r retainedTransition
	if decode(i.Plan, &p) != nil || decode(i.Transition, &r) != nil || p.Protocol != "symphony.shv.publication-plan.v1" || r.Protocol != "symphony.shv.publication-transition.v1" || p.OperationID != i.OperationID || r.OperationID != i.OperationID || !sameDigest(p.Expected, r.Expected) || !sameJSON(p.Head, r.Head) {
		return fmt.Errorf("SHV retained intent binding mismatch")
	}
	for _, v := range []struct {
		raw    json.RawMessage
		digest string
	}{{i.Plan, p.Digest}, {i.Transition, r.Digest}} {
		var m map[string]any
		dec := json.NewDecoder(bytes.NewReader(v.raw))
		dec.UseNumber()
		if dec.Decode(&m) != nil {
			return fmt.Errorf("invalid SHV retained object")
		}
		delete(m, "digest")
		d, e := knowledgeengine.SCVDigest(m)
		if e != nil || d != v.digest {
			return fmt.Errorf("SHV retained object seal mismatch")
		}
	}
	return nil
}
func validateDocument(d Document, s Store) error {
	if d.Protocol != protocol || d.TOPSID != s.TOPSID || d.CatalogueID != s.CatalogueID || d.Operations == nil || len(d.Operations) > maxOperations {
		return fmt.Errorf("SHV store identity or bounds mismatch")
	}
	digest, e := seal(d)
	if e != nil || digest != d.Digest {
		return fmt.Errorf("SHV store seal mismatch")
	}
	committed := map[string]json.RawMessage{}
	byGeneration := map[int64]string{}
	correlations := map[string]bool{}
	for id, a := range d.Operations {
		if id != a.Intent.OperationID || validateIntentShape(a.Intent) != nil {
			return fmt.Errorf("invalid retained SHV intent")
		}
		if stavprotocol.ValidateRequestUUID(a.CorrelationID) != nil || len(a.CorrelationID) != 36 || a.CorrelationID[14] != '4' {
			return fmt.Errorf("SHV correlation must be durable UUIDv4")
		}
		if correlations[a.CorrelationID] {
			return fmt.Errorf("duplicate SHV authorization correlation")
		}
		correlations[a.CorrelationID] = true
		if a.Status != "prepared" && a.Status != "authorized" && a.Status != "committed" {
			return fmt.Errorf("invalid SHV attempt status")
		}
		if a.PriorAuthorizations == nil || len(a.PriorAuthorizations) > 64 {
			return fmt.Errorf("invalid SHV authorization history bounds")
		}
		if a.Status == "prepared" {
			if !bytes.Equal(a.Authorization, []byte("null")) || len(a.PriorAuthorizations) != 0 {
				return fmt.Errorf("prepared SHV attempt has authorization evidence")
			}
		} else {
			if _, e := canonicalAuthorization(a.Authorization, s, a); e != nil {
				return e
			}
		}
		for _, prior := range a.PriorAuthorizations {
			if _, e := canonicalAuthorization(prior, s, a); e != nil {
				return e
			}
		}
		var r retainedTransition
		_ = json.Unmarshal(a.Intent.Transition, &r)
		var head retainedHead
		if json.Unmarshal(r.Head, &head) != nil || head.Definition.CatalogueID != s.CatalogueID || head.Definition.TOPSID != s.TOPSID || !hashPattern.MatchString(head.Digest) || !sameDigest(head.Previous, r.Expected) {
			return fmt.Errorf("retained SHV head identity/predecessor mismatch")
		}
		if a.Status == "committed" {
			if _, exists := committed[head.Digest]; exists {
				return fmt.Errorf("duplicate committed SHV head")
			}
			if _, exists := byGeneration[head.Generation]; exists {
				return fmt.Errorf("committed SHV chain forks")
			}
			committed[head.Digest] = r.Head
			byGeneration[head.Generation] = head.Digest
		}
	}
	// Every attempt, including abandoned non-head branches, is independently
	// reduced against its exact retained committed predecessor. A seal is not proof.
	for _, a := range d.Operations {
		var r retainedTransition
		_ = json.Unmarshal(a.Intent.Transition, &r)
		current := json.RawMessage("null")
		if r.Expected != nil {
			var ok bool
			current, ok = committed[*r.Expected]
			if !ok {
				return fmt.Errorf("SHV attempt predecessor is not committed")
			}
		}
		input, e := knowledgeengine.SCVCanonical(map[string]any{"current": current, "plan": a.Intent.Plan})
		if e != nil {
			return e
		}
		if e = knowledgeengine.ValidateSHVPublicationResult("publication_reduce", input, a.Intent.Transition); e != nil {
			return fmt.Errorf("retained SHV transition replay: %w", e)
		}
	}
	if len(committed) == 0 {
		if d.StateDigest != nil || d.HeadOperationID != nil || !bytes.Equal(d.Head, []byte("null")) {
			return fmt.Errorf("absent SHV head mismatch")
		}
		return nil
	}
	if len(committed) > 32 || d.StateDigest == nil || d.HeadOperationID == nil {
		return fmt.Errorf("SHV committed head missing or chain too long")
	}
	for generation := int64(1); generation <= int64(len(committed)); generation++ {
		digest, ok := byGeneration[generation]
		if !ok {
			return fmt.Errorf("SHV history is not contiguous")
		}
		var head retainedHead
		_ = json.Unmarshal(committed[digest], &head)
		if generation == 1 {
			if head.Previous != nil {
				return fmt.Errorf("SHV initial predecessor mismatch")
			}
		} else if head.Previous == nil || *head.Previous != byGeneration[generation-1] {
			return fmt.Errorf("SHV committed history forks")
		}
	}
	if *d.StateDigest != byGeneration[int64(len(committed))] || !sameJSON(d.Head, committed[*d.StateDigest]) {
		return fmt.Errorf("SHV selected head is not terminal committed revision")
	}
	head, ok := d.Operations[*d.HeadOperationID]
	var transition retainedTransition
	_ = json.Unmarshal(head.Intent.Transition, &transition)
	if !ok || head.Status != "committed" || !sameJSON(transition.Head, d.Head) {
		return fmt.Errorf("SHV head operation mismatch")
	}
	return validateHistory(d)
}
func documentHistory(d Document) ([]json.RawMessage, error) {
	type entry struct {
		generation int64
		head       json.RawMessage
	}
	entries := []entry{}
	for _, a := range d.Operations {
		if a.Status != "committed" {
			continue
		}
		var r retainedTransition
		var s retainedHead
		if e := json.Unmarshal(a.Intent.Transition, &r); e != nil {
			return nil, e
		}
		if e := json.Unmarshal(r.Head, &s); e != nil {
			return nil, e
		}
		entries = append(entries, entry{s.Generation, append(json.RawMessage(nil), r.Head...)})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].generation < entries[j].generation })
	out := []json.RawMessage{}
	for _, e := range entries {
		out = append(out, e.head)
	}
	return out, nil
}
func resource(tops, head string) string {
	d, _ := knowledgeengine.SCVDigest(map[string]any{"tops_id": tops, "owner_engine_id": "symphony-shv-publication", "catalogue_id": head})
	return "symphony.shv.catalogue:" + strings.TrimPrefix(d, "sha256:")
}

func validateHistory(d Document) error {
	history, e := documentHistory(d)
	if e != nil {
		return e
	}
	return validateHistoryRaw(history)
}
func validateHistoryRaw(history []json.RawMessage) error {
	if len(history) == 0 {
		return nil
	}
	input, e := knowledgeengine.SCVCanonical(map[string]any{"history": history})
	if e != nil {
		return e
	}
	if len(input) > maxHistoryPayloadBytes {
		return fmt.Errorf("SHV committed history exceeds native head_status request budget")
	}
	digests := []string{}
	for _, v := range history {
		var s retainedHead
		if e := json.Unmarshal(v, &s); e != nil {
			return e
		}
		digests = append(digests, s.Digest)
	}
	result := map[string]any{"protocol": "symphony.shv.publication-status.v1", "head": history[len(history)-1], "history_digests": digests}
	digest, e := knowledgeengine.SCVDigest(result)
	if e != nil {
		return e
	}
	result["digest"] = digest
	output, e := knowledgeengine.SCVCanonical(result)
	if e != nil {
		return e
	}
	return knowledgeengine.ValidateSHVPublicationResult("publication_status", input, output)
}
