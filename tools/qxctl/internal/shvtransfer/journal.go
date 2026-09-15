// Package shvtransfer owns local orchestration evidence. C++ connectors remain
// the only writers of indexed intent, snapshot and projection semantics.
package shvtransfer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"path/filepath"
	"regexp"
)

var digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

const MaxBytes = 4 << 20
const Protocol = "symphony.qxctl.shv-store-transfer-intent.v1"

type Intent struct {
	Protocol   string          `json:"protocol"`
	SourceRoot string          `json:"source_root"`
	TargetRoot string          `json:"target_root"`
	Plan       json.RawMessage `json:"plan"`
	Digest     string          `json:"digest"`
}
type Event struct {
	Protocol       string `json:"protocol"`
	TransferDigest string `json:"transfer_digest"`
	Sequence       int    `json:"sequence"`
	Previous       string `json:"previous"`
	Phase          string `json:"phase"`
	OperationID    string `json:"operation_id"`
	ResultDigest   string `json:"result_digest"`
	Digest         string `json:"digest"`
}

func Seal(v any) ([]byte, error) {
	raw, err := knowledgeengine.SCVCanonical(v)
	if err != nil {
		return nil, err
	}
	m, err := Decode(raw)
	if err != nil {
		return nil, err
	}
	delete(m, "digest")
	d, err := knowledgeengine.SCVDigest(m)
	if err != nil {
		return nil, err
	}
	m["digest"] = d
	raw, err = knowledgeengine.SCVCanonical(m)
	if err != nil {
		return nil, err
	}
	return raw, knowledgeengine.ValidateSCVBundleText(raw)
}
func Decode(raw []byte) (map[string]any, error) {
	if err := knowledgeengine.ValidateSCVBundleText(raw); err != nil {
		return nil, err
	}
	var m map[string]any
	// The shared canonical parser bounds integers, text, depth and duplicate keys.
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	err := d.Decode(&m)
	return m, err
}
func ReadIntent(raw []byte) (Intent, error) {
	var i Intent
	if err := strict(raw, &i); err != nil {
		return i, err
	}
	if i.Protocol != Protocol || i.SourceRoot == i.TargetRoot || !filepath.IsAbs(i.SourceRoot) || filepath.Clean(i.SourceRoot) != i.SourceRoot || i.SourceRoot == "/" || !filepath.IsAbs(i.TargetRoot) || filepath.Clean(i.TargetRoot) != i.TargetRoot || i.TargetRoot == "/" {
		return i, fmt.Errorf("transfer identity differs")
	}
	p, err := Decode(i.Plan)
	if err != nil {
		return i, err
	}
	input, ok := p["input"].(map[string]any)
	if !ok {
		return i, fmt.Errorf("plan input required")
	}
	source, _ := input["source_connector"].(map[string]any)
	version, _ := source["Version"].(string)
	in, _ := knowledgeengine.SCVCanonical(input)
	if err = knowledgeengine.ValidateSHVStoreResultVersion("transfer_plan", in, i.Plan, version); err != nil {
		return i, err
	}
	if p["disposition"] != "ready" || input["target_root"] != i.TargetRoot {
		return i, fmt.Errorf("transfer requires ready exact target plan")
	}
	return i, nil
}
func strict(raw []byte, v any) error {
	m, err := Decode(raw)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(raw, v); err != nil {
		return err
	}
	round, err := knowledgeengine.SCVCanonical(v)
	if err != nil {
		return err
	}
	rm, err := Decode(round)
	if err != nil {
		return err
	}
	a, _ := knowledgeengine.SCVCanonical(m)
	b, _ := knowledgeengine.SCVCanonical(rm)
	if string(a) != string(b) {
		return fmt.Errorf("unknown or mistyped journal field")
	}
	sealed, err := Seal(v)
	if err != nil {
		return err
	}
	if string(a) != string(sealed) {
		return fmt.Errorf("journal seal differs")
	}
	return nil
}
func ReadEvent(raw []byte) (Event, error) { var e Event; err := strict(raw, &e); return e, err }
func Operations(i Intent) ([]string, map[string]string) {
	p, _ := Decode(i.Plan)
	ids := []string{}
	states := map[string]string{}
	for _, v := range p["selected"].([]any) {
		s := v.(map[string]any)["source"].(map[string]any)
		id := s["intent"].(map[string]any)["operation_id"].(string)
		ids = append(ids, id)
		states[id] = s["state"].(string)
	}
	return ids, states
}

// Next enforces a single adjacent immutable chain. No timestamp or discovery
// selects progress; unknown phases, gaps and reordered operations fail closed.
func Next(i Intent, events []Event) (string, string) {
	ids, states := Operations(i)
	if len(events) == 0 {
		return "prepare_pending", ids[0]
	}
	last := events[len(events)-1]
	switch last.Phase {
	case "prepare_pending":
		return "prepared", last.OperationID
	case "prepared":
		if states[last.OperationID] == "committed" {
			return "commit_pending", last.OperationID
		}
	case "commit_pending":
		return "committed", last.OperationID
	case "complete":
		return "", ""
	}
	for n, id := range ids {
		if id == last.OperationID {
			if n+1 == len(ids) {
				return "complete", ""
			}
			return "prepare_pending", ids[n+1]
		}
	}
	return "", ""
}
func ValidateChain(i Intent, events []Event) error {
	if len(events) > 65 {
		return fmt.Errorf("transfer progress exceeds bound")
	}
	for n, e := range events {
		phase, id := Next(i, events[:n])
		prev := i.Digest
		if n > 0 {
			prev = events[n-1].Digest
		}
		if e.Protocol != "symphony.qxctl.shv-store-transfer-event.v1" || e.TransferDigest != i.Digest || e.Sequence != n || e.Previous != prev || e.Phase != phase || e.OperationID != id || phase == "" {
			return fmt.Errorf("transfer event chain differs")
		}
		if (phase == "prepared" || phase == "committed") != (e.ResultDigest != "") {
			return fmt.Errorf("transfer event result identity differs")
		}
		if e.ResultDigest != "" && !digestPattern.MatchString(e.ResultDigest) {
			return fmt.Errorf("invalid result digest")
		}
	}
	return nil
}
