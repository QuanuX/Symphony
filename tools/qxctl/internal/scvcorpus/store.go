// Package scvcorpus retains immutable evidence and resumable job bookkeeping.
// It owns no selected source, corpus or graph head and grants no authority.
package scvcorpus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"regexp"
	"strings"
	"unicode"
)

const MaxMembers = 128
const MaxArtifactBytes = 1 << 20
const maxJobBytes = 2 << 20

type Store struct{ Root, TOPSID, Domain string }
type Checkpoint struct {
	CaptureDigest string `json:"capture_digest"`
	IndexDigest   string `json:"index_digest"`
}
type Job struct {
	Protocol       string                       `json:"protocol"`
	OperationID    string                       `json:"operation_id"`
	Mode           string                       `json:"mode"`
	Installation   knowledgeengine.Installation `json:"installation"`
	Input          json.RawMessage              `json:"input"`
	IntentDigest   string                       `json:"intent_digest"`
	Completed      map[string]Checkpoint        `json:"completed"`
	SnapshotTime   string                       `json:"snapshot_time"`
	SnapshotDigest string                       `json:"snapshot_digest"`
	Digest         string                       `json:"digest"`
}

var digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
var topsPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func ValidDigest(s string) bool { return digestPattern.MatchString(s) }
func ValidID(s string) bool {
	if len(s) == 0 || len(s) > 128 {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
func New(root, tops, domain string) (Store, error) {
	if !topsPattern.MatchString(tops) {
		return Store{}, fmt.Errorf("corpus store requires exact lowercase TOPS UUID")
	}
	valid := false
	for _, d := range knowledgeengine.SCVDomains() {
		valid = valid || d == domain
	}
	if !valid {
		return Store{}, fmt.Errorf("invalid corpus domain")
	}
	return Store{Root: root, TOPSID: tops, Domain: domain}, nil
}
func Decode(raw []byte) (map[string]any, error) {
	if err := knowledgeengine.ValidateJSONObject(raw, MaxArtifactBytes); err != nil {
		return nil, err
	}
	var result map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	err := decoder.Decode(&result)
	return result, err
}
func canonical(value any) ([]byte, error) {
	raw, err := knowledgeengine.SCVCanonical(value)
	if err != nil {
		return nil, err
	}
	var decoded any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if err = d.Decode(&decoded); err != nil {
		return nil, err
	}
	return knowledgeengine.SCVCanonical(decoded)
}
func Same(a, b any) bool {
	x, xe := canonical(a)
	y, ye := canonical(b)
	return xe == nil && ye == nil && bytes.Equal(x, y)
}
func objectDigest(raw []byte) (string, error) {
	value, err := Decode(raw)
	if err != nil {
		return "", err
	}
	expected, ok := value["digest"].(string)
	if !ok || !ValidDigest(expected) {
		return "", fmt.Errorf("artifact lacks exact digest")
	}
	delete(value, "digest")
	actual, err := knowledgeengine.SCVDigest(value)
	if err != nil || actual != expected {
		return "", fmt.Errorf("artifact digest mismatch %s", expected)
	}
	return expected, nil
}
func protocol(kind string) string {
	return map[string]string{"captures": "symphony.scv.capture.v1", "indexes": "symphony.scv.capture-index.v1", "snapshots": "symphony.scv.corpus.v1"}[kind]
}
func validateArtifact(kind, digest string, raw []byte) error {
	actual, err := objectDigest(raw)
	if err != nil {
		return fmt.Errorf("%s %s: %w", kind, digest, err)
	}
	if actual != digest {
		return fmt.Errorf("%s requested digest %s differs from bytes", kind, digest)
	}
	value, _ := Decode(raw)
	if value["protocol"] != protocol(kind) || protocol(kind) == "" {
		return fmt.Errorf("artifact protocol mismatch %s", digest)
	}
	return nil
}
func jobIntent(j Job) any {
	return map[string]any{"protocol": "symphony.qxctl.scv-corpus-intent.v1", "operation_id": j.OperationID, "mode": j.Mode, "installation": j.Installation, "input": j.Input}
}
func sealJob(j Job) (Job, []byte, error) {
	j.Digest = ""
	raw, err := canonical(j)
	if err != nil {
		return j, nil, err
	}
	var value map[string]any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	_ = d.Decode(&value)
	delete(value, "digest")
	j.Digest, err = knowledgeengine.SCVDigest(value)
	if err != nil {
		return j, nil, err
	}
	raw, err = canonical(j)
	if len(raw) > maxJobBytes {
		return j, nil, fmt.Errorf("corpus job exceeds byte bound")
	}
	return j, raw, err
}
func NewJob(operation, mode string, installation knowledgeengine.Installation, input json.RawMessage) (Job, error) {
	if !ValidID(operation) || (mode != "acquire" && mode != "import") {
		return Job{}, fmt.Errorf("invalid corpus operation identity/mode")
	}
	if _, err := Decode(input); err != nil {
		return Job{}, err
	}
	j := Job{Protocol: "symphony.qxctl.scv-corpus-job.v1", OperationID: operation, Mode: mode, Installation: installation, Input: append(json.RawMessage(nil), input...), Completed: map[string]Checkpoint{}}
	raw, err := canonical(jobIntent(j))
	if err != nil {
		return j, err
	}
	var value any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	_ = d.Decode(&value)
	j.IntentDigest, err = knowledgeengine.SCVDigest(value)
	return j, err
}
func validateJob(raw []byte, operation string) (Job, error) {
	if err := knowledgeengine.ValidateJSONObject(raw, maxJobBytes); err != nil {
		return Job{}, err
	}
	if len(raw) > maxJobBytes {
		return Job{}, fmt.Errorf("corpus job exceeds bound")
	}
	var j Job
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&j); err != nil {
		return j, err
	}
	if j.Protocol != "symphony.qxctl.scv-corpus-job.v1" || j.OperationID != operation || j.Completed == nil || len(j.Completed) > MaxMembers {
		return j, fmt.Errorf("invalid corpus job identity/checkpoints")
	}
	expected, err := NewJob(j.OperationID, j.Mode, j.Installation, j.Input)
	if err != nil || expected.IntentDigest != j.IntentDigest {
		return j, fmt.Errorf("corpus intent digest mismatch")
	}
	sealed, _, err := sealJob(j)
	if err != nil || sealed.Digest != j.Digest {
		return j, fmt.Errorf("corpus job digest mismatch")
	}
	input, err := Decode(j.Input)
	if err != nil {
		return j, err
	}
	if input["operation_id"] != j.OperationID {
		return j, fmt.Errorf("corpus journal and pinned input operation IDs differ")
	}
	members, ok := input["members"].([]any)
	if !ok || len(members) > MaxMembers {
		return j, fmt.Errorf("invalid pinned corpus members")
	}
	allowed := map[string]bool{}
	for _, raw := range members {
		m, ok := raw.(map[string]any)
		if !ok {
			return j, fmt.Errorf("invalid pinned member")
		}
		id, _ := m["member_id"].(string)
		if !ValidID(id) || allowed[id] {
			return j, fmt.Errorf("invalid pinned member identity")
		}
		allowed[id] = true
	}
	for id, c := range j.Completed {
		if !allowed[id] || !ValidDigest(c.CaptureDigest) || !ValidDigest(c.IndexDigest) {
			return j, fmt.Errorf("invalid corpus checkpoint")
		}
	}
	if j.SnapshotDigest != "" && (!ValidDigest(j.SnapshotDigest) || len(j.Completed) != len(members) || j.SnapshotTime == "") {
		return j, fmt.Errorf("invalid completed corpus job")
	}
	return j, nil
}
func keyForOperation(operation string) string {
	digest, _ := knowledgeengine.SCVDigest(map[string]any{"operation_id": operation})
	return strings.TrimPrefix(digest, "sha256:")
}
