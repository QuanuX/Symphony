// Package scvworkflow retains exact owner-validated artifacts and resumable
// caller compositions. It neither interprets evidence nor selects authority heads.
package scvworkflow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvcorpus"
)

const MaxBytes = 4 << 20
const MaxInputBytes = 1 << 20
const MaxList = 128

type Store struct{ Root, TOPSID string }
type Record struct {
	Protocol       string                       `json:"protocol"`
	Kind           string                       `json:"kind"`
	Operation      string                       `json:"operation"`
	Installation   knowledgeengine.Installation `json:"installation"`
	Input          json.RawMessage              `json:"input"`
	Artifact       json.RawMessage              `json:"artifact"`
	ArtifactDigest string                       `json:"artifact_digest"`
	Digest         string                       `json:"digest"`
}
type Checkpoint struct {
	InputRef  string `json:"input_ref"`
	ResultRef string `json:"result_ref"`
}
type Run struct {
	Protocol           string                        `json:"protocol"`
	OperationID        string                        `json:"operation_id"`
	Installation       knowledgeengine.Installation  `json:"installation"`
	CorpusInstallation *knowledgeengine.Installation `json:"corpus_installation"`
	CorpusRoot         string                        `json:"corpus_root"`
	Request            json.RawMessage               `json:"request"`
	IntentDigest       string                        `json:"intent_digest"`
	Stages             map[string]Checkpoint         `json:"stages"`
	Complete           bool                          `json:"complete"`
	Digest             string                        `json:"digest"`
}

func New(root, tops string) (Store, error) {
	if _, err := scvcorpus.New(root, tops, "scv"); err != nil {
		return Store{}, err
	}
	return Store{root, tops}, nil
}
func Decode(raw []byte) (map[string]any, error) {
	if err := knowledgeengine.ValidateJSONObject(raw, MaxBytes); err != nil {
		return nil, err
	}
	var result map[string]any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	err := d.Decode(&result)
	return result, err
}
func Canonical(v any) ([]byte, error) {
	raw, err := knowledgeengine.SCVCanonical(v)
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
func Seal(v any) ([]byte, error) {
	raw, err := Canonical(v)
	if err != nil {
		return nil, err
	}
	m, err := Decode(raw)
	if err != nil {
		return nil, err
	}
	delete(m, "digest")
	m["digest"], err = knowledgeengine.SCVDigest(m)
	if err != nil {
		return nil, err
	}
	return Canonical(m)
}
func Digest(raw []byte) (string, error) {
	m, err := Decode(raw)
	if err != nil {
		return "", err
	}
	digest, ok := m["digest"].(string)
	if !ok || !scvcorpus.ValidDigest(digest) {
		return "", fmt.Errorf("missing exact workflow artifact digest")
	}
	delete(m, "digest")
	actual, err := knowledgeengine.SCVDigest(m)
	if err != nil || actual != digest {
		return "", fmt.Errorf("workflow artifact digest mismatch")
	}
	return digest, nil
}
func Kind(operation string) string {
	return knowledgeengine.SCVArtifactKind(operation)
}
func NewRecord(operation string, installed knowledgeengine.Installation, input, result json.RawMessage) (Record, error) {
	r := Record{Protocol: "symphony.qxctl.scv-artifact.v1", Kind: Kind(operation), Operation: operation, Installation: installed, Input: input, Artifact: result}
	var err error
	r.ArtifactDigest, err = Digest(result)
	if err != nil {
		return r, err
	}
	raw, err := Seal(r)
	if err != nil {
		return r, err
	}
	return ReadRecord(raw)
}
func ReadRecord(raw []byte) (Record, error) {
	var r Record
	if _, err := Digest(raw); err != nil {
		return r, err
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&r); err != nil {
		return r, err
	}
	protocol, ok := knowledgeengine.SCVResultProtocol(r.Operation)
	if r.Protocol != "symphony.qxctl.scv-artifact.v1" || r.Kind == "" || r.Kind != Kind(r.Operation) || !ok {
		return r, fmt.Errorf("invalid retained artifact identity")
	}
	if !knowledgeengine.SCVArtifactSupported(r.Installation.Version, r.Operation) {
		return r, fmt.Errorf("retained artifact is unsupported by the exact recorded installation")
	}
	if err := knowledgeengine.ValidateJSONObject(r.Input, MaxInputBytes); err != nil {
		return r, err
	}
	artifact, err := Decode(r.Artifact)
	if err != nil {
		return r, err
	}
	digest, err := Digest(r.Artifact)
	if err != nil || digest != r.ArtifactDigest || artifact["protocol"] != protocol {
		return r, fmt.Errorf("retained native artifact identity mismatch")
	}
	return r, nil
}
func NewRun(operation string, installed knowledgeengine.Installation, corpusOwner *knowledgeengine.Installation, corpusRoot string, request json.RawMessage) (Run, error) {
	r := Run{Protocol: "symphony.qxctl.scv-workflow-run.v1", OperationID: operation, Installation: installed, CorpusInstallation: corpusOwner, CorpusRoot: corpusRoot, Request: request, Stages: map[string]Checkpoint{}}
	if !scvcorpus.ValidID(operation) {
		return r, fmt.Errorf("invalid workflow operation ID")
	}
	if err := knowledgeengine.ValidateJSONObject(request, MaxInputBytes); err != nil {
		return r, err
	}
	intent := map[string]any{"protocol": "symphony.qxctl.scv-workflow-intent.v1", "operation_id": operation, "installation": installed, "corpus_installation": corpusOwner, "corpus_root": corpusRoot, "request": request}
	raw, err := Canonical(intent)
	if err != nil {
		return r, err
	}
	m, err := Decode(raw)
	if err != nil {
		return r, err
	}
	r.IntentDigest, err = knowledgeengine.SCVDigest(m)
	return r, err
}
func ReadRun(raw []byte, operation string) (Run, error) {
	var r Run
	if _, err := Digest(raw); err != nil {
		return r, err
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&r); err != nil {
		return r, err
	}
	if r.Protocol != "symphony.qxctl.scv-workflow-run.v1" || r.OperationID != operation || r.Stages == nil || len(r.Stages) > 3 {
		return r, fmt.Errorf("invalid workflow run identity/checkpoints")
	}
	expected, err := NewRun(operation, r.Installation, r.CorpusInstallation, r.CorpusRoot, r.Request)
	if err != nil || expected.IntentDigest != r.IntentDigest {
		return r, fmt.Errorf("workflow intent digest mismatch")
	}
	m, err := Decode(r.Request)
	if err != nil || m["operation_id"] != operation {
		return r, fmt.Errorf("workflow request operation identity mismatch")
	}
	if m["corpus"] != nil {
		if r.CorpusInstallation == nil || r.CorpusRoot == "" {
			return r, fmt.Errorf("workflow corpus installation/root missing")
		}
		if _, ok := r.Stages["evaluate"]; ok && r.Stages["interpret"].ResultRef == "" {
			return r, fmt.Errorf("corpus workflow evaluation lacks interpretation")
		}
	} else {
		if r.CorpusInstallation != nil {
			return r, fmt.Errorf("unexpected corpus installation")
		}
		if _, ok := r.Stages["interpret"]; ok {
			return r, fmt.Errorf("interpretation checkpoint without corpus")
		}
	}
	if m["prior_evaluation_ref"] == nil {
		if _, ok := r.Stages["reassess"]; ok {
			return r, fmt.Errorf("unrequested reassessment checkpoint")
		}
	}
	for name, c := range r.Stages {
		if name != "interpret" && name != "evaluate" && name != "reassess" {
			return r, fmt.Errorf("unknown workflow checkpoint")
		}
		if !scvcorpus.ValidDigest(c.InputRef) || (c.ResultRef != "" && !scvcorpus.ValidDigest(c.ResultRef)) {
			return r, fmt.Errorf("invalid workflow checkpoint reference")
		}
	}
	if c, ok := r.Stages["reassess"]; ok && (r.Stages["evaluate"].ResultRef == "" || c.InputRef == "") {
		return r, fmt.Errorf("reassessment precedes evaluation")
	}
	if _, ok := r.Stages["evaluate"]; ok {
		if c, present := r.Stages["interpret"]; present && c.ResultRef == "" {
			return r, fmt.Errorf("evaluation precedes interpretation")
		}
	}
	if r.Complete {
		if r.Stages["evaluate"].ResultRef == "" {
			return r, fmt.Errorf("completed workflow lacks evaluation")
		}
		if m["prior_evaluation_ref"] != nil && r.Stages["reassess"].ResultRef == "" {
			return r, fmt.Errorf("completed workflow lacks requested reassessment")
		}
	}
	return r, nil
}
func keyForOperation(operation string) string {
	d, _ := knowledgeengine.SCVDigest(map[string]any{"operation_id": operation})
	return d[len("sha256:"):]
}
