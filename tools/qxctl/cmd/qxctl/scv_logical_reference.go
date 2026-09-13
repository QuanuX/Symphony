package main

import (
	"encoding/json"
	"fmt"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvworkflow"
)

// logicalArtifactReference preserves a record's original provenance while
// exposing its independently checked logical documents to the v2 adapters.
// A projection is not an assertion that the original executable was invoked.
type logicalArtifactReference struct {
	Record     scvworkflow.Record
	Input      json.RawMessage
	Artifact   json.RawMessage
	Descriptor map[string]any
}

func supportedLogicalComposition(operation string) bool {
	switch operation {
	case "composition_explore", "composition_reassess", "composition_obligations", "composition_followup":
		return true
	default:
		return false
	}
}

func projectLogicalRecord(record scvworkflow.Record, requiredOperation string) (logicalArtifactReference, error) {
	var reference logicalArtifactReference
	if !supportedLogicalComposition(requiredOperation) {
		return reference, fmt.Errorf("unsupported required logical operation")
	}
	input, artifact := record.Input, record.Artifact
	var transport any
	if record.Operation == "composition_bundle_evaluate" {
		// Reuse the documents produced by this call's complete consumer check.
		// There is no cached record, installation decision or native replay here.
		checked, err := knowledgeengine.ValidateSCVBundleEvaluation(input, artifact)
		if err != nil {
			return reference, err
		}
		if checked.Operation != requiredOperation {
			return reference, fmt.Errorf("bundled record has a different logical operation")
		}
		if checked.Domain != record.Installation.Role || checked.Version != record.Installation.Version {
			return reference, fmt.Errorf("bundled record owner differs from its retained installation")
		}
		protocol, _ := knowledgeengine.SCVResultProtocol(requiredOperation)
		return logicalArtifactReference{Record: record, Input: checked.Input, Artifact: checked.Result, Descriptor: map[string]any{
			"record_ref": record.Digest, "logical_operation": requiredOperation, "logical_protocol": protocol,
			"logical_digest": checked.NativeResultDigest, "artifact_digest": record.ArtifactDigest,
			"transport": map[string]any{"result_bundle_digest": checked.ResultBundleDigest, "result_root_digest": checked.ResultRootDigest},
		}}, nil
	} else {
		if record.Operation != requiredOperation {
			return reference, fmt.Errorf("retained record has a different logical operation")
		}
		if err := knowledgeengine.ValidateSCVResult(record.Operation, input, artifact); err != nil {
			return reference, err
		}
		// Canonicalize the logical view, not the retained record or its seal.
		value, err := scvworkflow.Decode(input)
		if err != nil {
			return reference, err
		}
		input, err = knowledgeengine.SCVCanonical(value)
		if err != nil {
			return reference, err
		}
		value, err = scvworkflow.Decode(artifact)
		if err != nil {
			return reference, err
		}
		artifact, err = knowledgeengine.SCVCanonical(value)
		if err != nil {
			return reference, err
		}
	}
	value, err := scvworkflow.Decode(artifact)
	if err != nil {
		return reference, err
	}
	protocol, _ := knowledgeengine.SCVResultProtocol(requiredOperation)
	if value["protocol"] != protocol || value["domain"] != record.Installation.Role {
		return reference, fmt.Errorf("logical result protocol or original domain mismatch")
	}
	logicalDigest, err := scvworkflow.Digest(artifact)
	if err != nil {
		return reference, err
	}
	reference = logicalArtifactReference{Record: record, Input: input, Artifact: artifact, Descriptor: map[string]any{
		"record_ref": record.Digest, "logical_operation": requiredOperation, "logical_protocol": protocol,
		"logical_digest": logicalDigest, "artifact_digest": record.ArtifactDigest, "transport": transport,
	}}
	return reference, nil
}

func (r *workflowRunner) resolveLogicalReference(s *scvworkflow.Session, ref, requiredOperation string, replay bool) (logicalArtifactReference, error) {
	var reference logicalArtifactReference
	raw, err := s.Get("records", ref)
	if err != nil {
		return reference, err
	}
	// The store has already applied its exact record/child budgets. Preserve
	// raw Unicode before any generic decoder can normalize invalid escapes.
	if err = knowledgeengine.ValidateSCVBundleUnicode(raw); err != nil {
		return reference, err
	}
	record, err := scvworkflow.ReadRecord(raw)
	if err != nil {
		return reference, err
	}
	if record.Digest != ref {
		return reference, fmt.Errorf("logical reference differs from its retained record")
	}
	reference, err = projectLogicalRecord(record, requiredOperation)
	if err != nil {
		return reference, err
	}
	if replay {
		if r.owner == nil {
			return reference, fmt.Errorf("original owner invocation is unavailable")
		}
		if _, err = r.replay(record); err != nil {
			return reference, err
		}
	}
	return reference, nil
}

func bundleLogicalInput(inst knowledgeengine.Installation, logicalOperation string, input any) (json.RawMessage, error) {
	if !supportedLogicalComposition(logicalOperation) || !knowledgeengine.SCVArtifactSupported(inst.Version, "composition_bundle_evaluate") || !knowledgeengine.SCVDomainSupported(inst.Version, inst.Role) {
		return nil, fmt.Errorf("selected installation does not support bundled logical evaluation")
	}
	raw, err := knowledgeengine.SCVCanonical(input)
	if err != nil {
		return nil, err
	}
	// Encode enforces the independent four-MiB/32768-value logical budget.
	bundle, err := knowledgeengine.SCVBundleEncode(raw)
	if err != nil {
		return nil, err
	}
	payload, err := knowledgeengine.SCVCanonical(map[string]any{"operation": logicalOperation, "owner": map[string]any{"domain": inst.Role, "version": inst.Version}, "bundle": json.RawMessage(bundle)})
	if err != nil {
		return nil, err
	}
	// The complete transport envelope still needs to fit the old process input.
	if err = knowledgeengine.ValidateSCVBundleText(payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (r *workflowRunner) bundledLogicalRecord(inst knowledgeengine.Installation, logicalOperation string, input any) (scvworkflow.Record, error) {
	var record scvworkflow.Record
	payload, err := bundleLogicalInput(inst, logicalOperation, input)
	if err != nil {
		return record, err
	}
	if r.owner == nil {
		return record, fmt.Errorf("selected bundle owner invocation is unavailable")
	}
	result, err := r.owner(inst, "composition_bundle_evaluate", json.RawMessage(payload))
	if err != nil {
		return record, err
	}
	if err = knowledgeengine.ValidateSCVResult("composition_bundle_evaluate", payload, result); err != nil {
		return record, err
	}
	return scvworkflow.NewRecord("composition_bundle_evaluate", inst, payload, result)
}
