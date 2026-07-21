package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/contracts"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/inventory"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/modules"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/repository"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/ssiagclient"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/status"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/stavclient"
)

func main() {
	os.Exit(execute(os.Args[1:]))
}

func printUsage() {
	fmt.Println("qxctl - Symphony administrative spine")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  qxctl <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  --help                            Print concise usage")
	fmt.Println("  --version                         Print version")
	fmt.Println("  doctor                            Perform local repository/admin-spine checks")
	fmt.Println("  contracts                         Verify first runtime-set module contract surfaces")
	fmt.Println("  modules                           List deterministic runtime modules")
	fmt.Println("  modules check                     Verify contract shape for all modules")
	fmt.Println("  modules metadata [--json]         Extract contract metadata for all modules")
	fmt.Println("  module inspect <module-name>      Inspect a specific runtime module")
	fmt.Println("  module check <module-name>        Verify contract shape for a module")
	fmt.Println("  module metadata <module-name> [--json] Extract contract metadata for a module")
	fmt.Println("  inventory [--json]                Emit deterministic runtime inventory snapshot")
	fmt.Println("  inventory digest [--json]         Emit deterministic runtime inventory SHA-256 digest")
	fmt.Println("  status [--json]                   Report consolidated administrative status")
	fmt.Println("  ssiag status --tops-id UUID [--scope user|system] [--json] Read safe SSIAG status")
	fmt.Println("  ssiag providers --tops-id UUID [--scope user|system] [--json] List safe provider metadata")
	fmt.Println("  ssiag doctor --tops-id UUID [--scope user|system] Verify local SSIAG availability")
	fmt.Println("  stav status --tops-id UUID [--scope user|system] [--json] Read authenticated STAV status")
	fmt.Println("  stav verify --tops-id UUID [--scope user|system] [--json] Verify the STAV digest chain")
	fmt.Println("  stav query --tops-id UUID [--scope user|system] [bounded filters] [--json] Query authorized STAV projections")
	fmt.Println("  stav doctor --tops-id UUID [--scope user|system] Run authenticated STAV diagnostics")
	fmt.Println("  skvi inspect --prefix PATH [--version VERSION] [--json] Inspect an exact installed SKVI engine")
	fmt.Println("  skvi check --prefix PATH [--version VERSION] [--json] Check canonical SKVI index truth")
	fmt.Println("  skvi propose --prefix PATH --input FILE [--version VERSION] [--json] Prepare a caller-declared proposal")
	fmt.Println("  skvi project --prefix PATH [--version VERSION] [--json] Build a disposable SKVI projection")
	fmt.Println("  sclv inspect --prefix PATH [--version VERSION] [--json] Inspect an exact installed SCLV engine")
	fmt.Println("  sclv check --prefix PATH [--version VERSION] [--json] Check canonical SCLV ledger truth")
	fmt.Println("  sclv propose --prefix PATH --input FILE [--version VERSION] [--json] Prepare a provider-neutral record proposal")
	fmt.Println("  sclv recover --prefix PATH --input FILE [--version VERSION] [--json] Reconcile ephemeral SCLV closure evidence")
	fmt.Println("  sclv project --prefix PATH [--version VERSION] [--json] Build a disposable SCLV projection")
}

func runSKVI(operation string, options skviOptions) error {
	if options.prefix == "" {
		return fmt.Errorf("--prefix is required")
	}
	start := options.repository
	if start == "" {
		var err error
		start, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("could not get current working directory: %w", err)
		}
	}
	start, err := filepath.Abs(start)
	if err != nil {
		return fmt.Errorf("resolve repository path: %w", err)
	}
	info, err := os.Lstat(start)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("--repo must identify a no-follow directory")
	}
	repoRoot, err := repository.FindRoot(start)
	if err != nil {
		return fmt.Errorf("could not find Symphony repository root: %w", err)
	}

	var payload []byte
	switch operation {
	case "inspect":
		payload = []byte(`{}`)
	case "check":
		expected := any(nil)
		if options.expectedIndexDigest != "" {
			expected = options.expectedIndexDigest
		}
		payload, err = json.Marshal(map[string]any{"expected_index_digest": expected})
	case "propose":
		payload, err = knowledgeengine.ReadPayload(options.input)
	case "project":
		payload = []byte(`{"format":"json"}`)
	default:
		return fmt.Errorf("unsupported SKVI operation")
	}
	if err != nil {
		return err
	}
	response, err := knowledgeengine.Invoke(
		context.Background(), options.prefix, options.version, repoRoot, operation, payload)
	if err != nil {
		return err
	}
	checkValid, err := validateSKVIResult(operation, response.Result)
	if err != nil {
		return err
	}
	if options.jsonOutput {
		var output bytes.Buffer
		if err := json.Indent(&output, response.Result, "", "  "); err != nil {
			return fmt.Errorf("format SKVI result: %w", err)
		}
		fmt.Println(output.String())
		if !checkValid {
			return fmt.Errorf("SKVI index check reported violations")
		}
		return nil
	}
	return printSKVIResult(operation, response.Result)
}

func validateSKVIResult(operation string, result json.RawMessage) (bool, error) {
	switch operation {
	case "inspect":
		var value struct {
			Readiness               string `json:"readiness"`
			CanonicalApplyEnabled   *bool  `json:"canonical_apply_enabled"`
			EngineDecidesMembership *bool  `json:"engine_decides_membership"`
			Descriptor              struct {
				EngineID               string `json:"engine_id"`
				CanonicalApplyEnabled  *bool  `json:"canonical_apply_enabled"`
				SessionMutationEnabled *bool  `json:"session_mutation_enabled"`
				NetworkListener        *bool  `json:"network_listener"`
			} `json:"descriptor"`
		}
		if err := json.Unmarshal(result, &value); err != nil ||
			value.Readiness != "read_check_propose_project" || value.Descriptor.EngineID != "symphony-skvi" ||
			!explicitFalse(value.CanonicalApplyEnabled) || !explicitFalse(value.EngineDecidesMembership) ||
			!explicitFalse(value.Descriptor.CanonicalApplyEnabled) ||
			!explicitFalse(value.Descriptor.SessionMutationEnabled) ||
			!explicitFalse(value.Descriptor.NetworkListener) {
			return false, fmt.Errorf("SKVI inspect result violates the implemented safety contract")
		}
		return true, nil
	case "check":
		var value struct {
			Protocol              string `json:"protocol"`
			ReadOnly              *bool  `json:"read_only"`
			CanonicalApplyEnabled *bool  `json:"canonical_apply_enabled"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.Protocol != "symphony.skvi.check-result.v1" ||
			!explicitTrue(value.ReadOnly) || !explicitFalse(value.CanonicalApplyEnabled) {
			return false, fmt.Errorf("SKVI check result violates the implemented safety contract")
		}
		return skviCheckValid(result)
	case "propose":
		var value struct {
			Protocol              string `json:"protocol"`
			ModuleID              string `json:"module_id"`
			EngineID              string `json:"engine_id"`
			VectorID              string `json:"vector_id"`
			ProposalID            string `json:"proposal_id"`
			ProposalDigest        string `json:"proposal_digest"`
			CanonicalApplyEnabled *bool  `json:"canonical_apply_enabled"`
			Authority             struct {
				CallerDeclaredOperation *bool `json:"caller_declared_operation"`
				EngineDecidedMembership *bool `json:"engine_decided_membership"`
				Ratified                *bool `json:"ratified"`
			} `json:"authority"`
			Operations []json.RawMessage `json:"operations"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.Protocol != "symphony.knowledge.proposal.v1" ||
			value.ModuleID != "skvi-engine" || value.EngineID != "symphony-skvi" || value.VectorID != "skvi" ||
			value.ProposalID == "" || !validTaggedDigest(value.ProposalDigest) || len(value.Operations) != 1 ||
			!explicitTrue(value.Authority.CallerDeclaredOperation) ||
			!explicitFalse(value.Authority.EngineDecidedMembership) ||
			!explicitFalse(value.Authority.Ratified) || !explicitFalse(value.CanonicalApplyEnabled) {
			return false, fmt.Errorf("SKVI proposal result violates the implemented safety contract")
		}
		return true, nil
	case "project":
		var value struct {
			Protocol         string            `json:"protocol"`
			ModuleID         string            `json:"module_id"`
			EngineID         string            `json:"engine_id"`
			VectorID         string            `json:"vector_id"`
			EntryCount       *uint64           `json:"entry_count"`
			Entries          []json.RawMessage `json:"entries"`
			ProjectionDigest string            `json:"projection_digest"`
			Noncanonical     *bool             `json:"noncanonical"`
			Rebuildable      *bool             `json:"rebuildable"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.Protocol != "symphony.skvi.projection.v1" ||
			value.ModuleID != "skvi-engine" || value.EngineID != "symphony-skvi" || value.VectorID != "skvi" ||
			value.EntryCount == nil || value.Entries == nil || *value.EntryCount != uint64(len(value.Entries)) ||
			!validTaggedDigest(value.ProjectionDigest) ||
			!explicitTrue(value.Noncanonical) || !explicitTrue(value.Rebuildable) {
			return false, fmt.Errorf("SKVI projection result violates the implemented safety contract")
		}
		return true, nil
	default:
		return false, fmt.Errorf("unsupported SKVI operation")
	}
}

func explicitFalse(value *bool) bool { return value != nil && !*value }

func explicitTrue(value *bool) bool { return value != nil && *value }

func validTaggedDigest(value string) bool {
	if len(value) != 71 || value[:7] != "sha256:" {
		return false
	}
	for _, character := range value[7:] {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func skviCheckValid(result json.RawMessage) (bool, error) {
	var value struct {
		Summary struct {
			Violation uint64 `json:"violation"`
			State     string `json:"state"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(result, &value); err != nil || value.Summary.State == "" {
		return false, fmt.Errorf("SKVI check result is incomplete")
	}
	return value.Summary.State == "valid" && value.Summary.Violation == 0, nil
}

func printSKVIResult(operation string, result json.RawMessage) error {
	switch operation {
	case "inspect":
		var value struct {
			Readiness             string `json:"readiness"`
			CanonicalApplyEnabled bool   `json:"canonical_apply_enabled"`
			Descriptor            struct {
				EngineID      string `json:"engine_id"`
				EngineVersion string `json:"engine_version"`
			} `json:"descriptor"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.Descriptor.EngineID == "" || value.Readiness == "" {
			return fmt.Errorf("SKVI inspect result is incomplete")
		}
		fmt.Printf("SKVI: engine=%s version=%s readiness=%s apply=%t\n",
			value.Descriptor.EngineID, value.Descriptor.EngineVersion,
			value.Readiness, value.CanonicalApplyEnabled)
		return nil
	case "check":
		var value struct {
			EntriesChecked       uint64 `json:"entries_checked"`
			RelationshipsChecked uint64 `json:"relationships_checked"`
			Index                struct {
				Digest string `json:"digest"`
			} `json:"index"`
			Summary struct {
				Pass      uint64 `json:"pass"`
				Warning   uint64 `json:"warning"`
				Violation uint64 `json:"violation"`
				State     string `json:"state"`
			} `json:"summary"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.Summary.State == "" || value.Index.Digest == "" {
			return fmt.Errorf("SKVI check result is incomplete")
		}
		fmt.Printf("SKVI check: state=%s entries=%d relationships=%d pass=%d warning=%d violation=%d index_digest=%s\n",
			value.Summary.State, value.EntriesChecked, value.RelationshipsChecked,
			value.Summary.Pass, value.Summary.Warning, value.Summary.Violation, value.Index.Digest)
		if value.Summary.State != "valid" || value.Summary.Violation != 0 {
			return fmt.Errorf("SKVI index check reported violations")
		}
		return nil
	case "propose":
		var value struct {
			ProposalID            string `json:"proposal_id"`
			ProposalDigest        string `json:"proposal_digest"`
			CanonicalApplyEnabled bool   `json:"canonical_apply_enabled"`
			Authority             struct {
				Ratified bool `json:"ratified"`
			} `json:"authority"`
			Operations []struct {
				Type       string `json:"type"`
				TargetPath string `json:"target_path"`
			} `json:"operations"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.ProposalID == "" || len(value.Operations) != 1 {
			return fmt.Errorf("SKVI proposal result is incomplete")
		}
		fmt.Printf("SKVI proposal: id=%s digest=%s operation=%s target=%s ratified=%t apply=%t\n",
			value.ProposalID, value.ProposalDigest, value.Operations[0].Type,
			value.Operations[0].TargetPath, value.Authority.Ratified, value.CanonicalApplyEnabled)
		return nil
	case "project":
		var value struct {
			EntryCount       uint64 `json:"entry_count"`
			ProjectionDigest string `json:"projection_digest"`
			Noncanonical     bool   `json:"noncanonical"`
			Rebuildable      bool   `json:"rebuildable"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.ProjectionDigest == "" {
			return fmt.Errorf("SKVI projection result is incomplete")
		}
		fmt.Printf("SKVI projection: entries=%d digest=%s noncanonical=%t rebuildable=%t\n",
			value.EntryCount, value.ProjectionDigest, value.Noncanonical, value.Rebuildable)
		return nil
	default:
		return fmt.Errorf("unsupported SKVI result")
	}
}

func runSCLV(operation string, options sclvOptions) error {
	if options.prefix == "" {
		return fmt.Errorf("--prefix is required")
	}
	start := options.repository
	if start == "" {
		var err error
		start, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("could not get current working directory: %w", err)
		}
	}
	start, err := filepath.Abs(start)
	if err != nil {
		return fmt.Errorf("resolve repository path: %w", err)
	}
	info, err := os.Lstat(start)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("--repo must identify a no-follow directory")
	}
	repoRoot, err := repository.FindRoot(start)
	if err != nil {
		return fmt.Errorf("could not find Symphony repository root: %w", err)
	}

	var payload []byte
	switch operation {
	case "inspect":
		payload = []byte(`{}`)
	case "check":
		expected := any(nil)
		if options.expectedLedgerDigest != "" {
			expected = options.expectedLedgerDigest
		}
		payload, err = json.Marshal(map[string]any{"expected_ledger_digest": expected})
	case "propose", "recover":
		payload, err = knowledgeengine.ReadPayload(options.input)
	case "project":
		payload = []byte(`{"format":"json"}`)
	default:
		return fmt.Errorf("unsupported SCLV operation")
	}
	if err != nil {
		return err
	}
	response, err := knowledgeengine.InvokeSCLV(
		context.Background(), options.prefix, options.version, repoRoot, operation, payload)
	if err != nil {
		return err
	}
	checkValid, err := validateSCLVResult(operation, response.Result)
	if err != nil {
		return err
	}
	if options.jsonOutput {
		var output bytes.Buffer
		if err := json.Indent(&output, response.Result, "", "  "); err != nil {
			return fmt.Errorf("format SCLV result: %w", err)
		}
		fmt.Println(output.String())
		if !checkValid {
			return fmt.Errorf("SCLV ledger check reported violations")
		}
		return nil
	}
	return printSCLVResult(operation, response.Result)
}

func validateSCLVResult(operation string, result json.RawMessage) (bool, error) {
	switch operation {
	case "inspect":
		var value struct {
			ReadOnly              *bool    `json:"read_only"`
			CanonicalApplyEnabled *bool    `json:"canonical_apply_enabled"`
			EvidenceAdapters      []string `json:"evidence_adapters"`
			Descriptor            struct {
				EngineID               string `json:"engine_id"`
				CanonicalApplyEnabled  *bool  `json:"canonical_apply_enabled"`
				SessionMutationEnabled *bool  `json:"session_mutation_enabled"`
				NetworkListener        *bool  `json:"network_listener"`
			} `json:"descriptor"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.Descriptor.EngineID != "symphony-sclv" ||
			!explicitTrue(value.ReadOnly) || !explicitFalse(value.CanonicalApplyEnabled) ||
			!explicitFalse(value.Descriptor.CanonicalApplyEnabled) ||
			!explicitFalse(value.Descriptor.SessionMutationEnabled) ||
			!explicitFalse(value.Descriptor.NetworkListener) || len(value.EvidenceAdapters) != 2 ||
			value.EvidenceAdapters[0] != "symphony-sclv-evidence-local-git" ||
			value.EvidenceAdapters[1] != "symphony-sclv-evidence-airgap" {
			return false, fmt.Errorf("SCLV inspect result violates the implemented safety contract")
		}
		return true, nil
	case "check":
		var value struct {
			Protocol              string `json:"protocol"`
			ReadOnly              *bool  `json:"read_only"`
			CanonicalApplyEnabled *bool  `json:"canonical_apply_enabled"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.Protocol != "symphony.sclv.check-result.v1" ||
			!explicitTrue(value.ReadOnly) || !explicitFalse(value.CanonicalApplyEnabled) {
			return false, fmt.Errorf("SCLV check result violates the implemented safety contract")
		}
		return sclvCheckValid(result)
	case "propose":
		var value struct {
			Protocol              string `json:"protocol"`
			ModuleID              string `json:"module_id"`
			EngineID              string `json:"engine_id"`
			VectorID              string `json:"vector_id"`
			ProposalID            string `json:"proposal_id"`
			ProposalDigest        string `json:"proposal_digest"`
			CanonicalApplyEnabled *bool  `json:"canonical_apply_enabled"`
			WriteSet              []struct {
				TargetPath string `json:"target_path"`
			} `json:"write_set"`
			Operations []struct {
				Type       string `json:"type"`
				TargetPath string `json:"target_path"`
			} `json:"operations"`
			Authority struct {
				CallerDeclaredOperation *bool `json:"caller_declared_operation"`
				EngineDecidedMembership *bool `json:"engine_decided_membership"`
				Ratified                *bool `json:"ratified"`
			} `json:"authority"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.Protocol != "symphony.knowledge.proposal.v1" ||
			value.ModuleID != "sclv-engine" || value.EngineID != "symphony-sclv" || value.VectorID != "sclv" ||
			value.ProposalID == "" || !validTaggedDigest(value.ProposalDigest) ||
			len(value.WriteSet) != 1 || value.WriteSet[0].TargetPath != "knowledge/sclv/CHANGELOG.md" ||
			len(value.Operations) != 1 || value.Operations[0].Type != "append_record_v3" ||
			value.Operations[0].TargetPath != "knowledge/sclv/CHANGELOG.md" ||
			!explicitTrue(value.Authority.CallerDeclaredOperation) ||
			!explicitFalse(value.Authority.EngineDecidedMembership) ||
			!explicitFalse(value.Authority.Ratified) || !explicitFalse(value.CanonicalApplyEnabled) {
			return false, fmt.Errorf("SCLV proposal result violates the implemented safety contract")
		}
		return true, nil
	case "recover":
		var value struct {
			Protocol              string          `json:"protocol"`
			Action                string          `json:"action"`
			JournalMutated        *bool           `json:"journal_mutated"`
			CanonicalApplyEnabled *bool           `json:"canonical_apply_enabled"`
			DeleteRecommended     *bool           `json:"delete_recommended"`
			Proposal              json.RawMessage `json:"proposal"`
			ResultDigest          string          `json:"result_digest"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.Protocol != "symphony.sclv.recovery-result.v1" ||
			!explicitFalse(value.JournalMutated) || !explicitFalse(value.CanonicalApplyEnabled) ||
			value.DeleteRecommended == nil || !validTaggedDigest(value.ResultDigest) {
			return false, fmt.Errorf("SCLV recovery result violates the implemented safety contract")
		}
		switch value.Action {
		case "resume":
			if *value.DeleteRecommended {
				return false, fmt.Errorf("SCLV resumable recovery recommended journal deletion")
			}
			if len(value.Proposal) == 0 || string(value.Proposal) != "null" {
				return false, fmt.Errorf("SCLV recovery result contains an unexpected proposal")
			}
		case "abandon", "no_op":
			if !*value.DeleteRecommended {
				return false, fmt.Errorf("SCLV terminal recovery omitted its deletion recommendation")
			}
			if len(value.Proposal) == 0 || string(value.Proposal) != "null" {
				return false, fmt.Errorf("SCLV recovery result contains an unexpected proposal")
			}
		case "propose_late_recovery":
			if *value.DeleteRecommended {
				return false, fmt.Errorf("SCLV late recovery recommended deletion before proposal completion")
			}
			if len(value.Proposal) == 0 || string(value.Proposal) == "null" {
				return false, fmt.Errorf("SCLV late recovery omitted its proposal")
			}
			if _, err := validateSCLVResult("propose", value.Proposal); err != nil {
				return false, fmt.Errorf("SCLV late-recovery proposal is invalid: %w", err)
			}
		default:
			return false, fmt.Errorf("SCLV recovery result has an unknown action")
		}
		return true, nil
	case "project":
		var value struct {
			Protocol         string            `json:"protocol"`
			ModuleID         string            `json:"module_id"`
			EngineID         string            `json:"engine_id"`
			VectorID         string            `json:"vector_id"`
			RecordCount      *uint64           `json:"record_count"`
			Records          []json.RawMessage `json:"records"`
			ProjectionDigest string            `json:"projection_digest"`
			Noncanonical     *bool             `json:"noncanonical"`
			Rebuildable      *bool             `json:"rebuildable"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.Protocol != "symphony.sclv.projection.v1" ||
			value.ModuleID != "sclv-engine" || value.EngineID != "symphony-sclv" || value.VectorID != "sclv" ||
			value.RecordCount == nil || value.Records == nil || *value.RecordCount != uint64(len(value.Records)) ||
			!validTaggedDigest(value.ProjectionDigest) ||
			!explicitTrue(value.Noncanonical) || !explicitTrue(value.Rebuildable) {
			return false, fmt.Errorf("SCLV projection result violates the implemented safety contract")
		}
		return true, nil
	default:
		return false, fmt.Errorf("unsupported SCLV operation")
	}
}

func sclvCheckValid(result json.RawMessage) (bool, error) {
	var value struct {
		Summary struct {
			Violation uint64 `json:"violation"`
			State     string `json:"state"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(result, &value); err != nil || value.Summary.State == "" {
		return false, fmt.Errorf("SCLV check result is incomplete")
	}
	return value.Summary.State == "valid" && value.Summary.Violation == 0, nil
}

func printSCLVResult(operation string, result json.RawMessage) error {
	switch operation {
	case "inspect":
		var value struct {
			ReadOnly   bool `json:"read_only"`
			Descriptor struct {
				EngineID      string `json:"engine_id"`
				EngineVersion string `json:"engine_version"`
				ThermalPath   string `json:"thermal_path"`
			} `json:"descriptor"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.Descriptor.EngineID == "" {
			return fmt.Errorf("SCLV inspect result is incomplete")
		}
		fmt.Printf("SCLV: engine=%s version=%s thermal=%s read_only=%t apply=false\n",
			value.Descriptor.EngineID, value.Descriptor.EngineVersion, value.Descriptor.ThermalPath, value.ReadOnly)
		return nil
	case "check":
		var value struct {
			RecordsChecked uint64 `json:"records_checked"`
			Ledger         struct {
				Digest string `json:"digest"`
			} `json:"ledger"`
			Summary struct {
				Pass      uint64 `json:"pass"`
				Warning   uint64 `json:"warning"`
				Violation uint64 `json:"violation"`
				State     string `json:"state"`
			} `json:"summary"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.Summary.State == "" || value.Ledger.Digest == "" {
			return fmt.Errorf("SCLV check result is incomplete")
		}
		fmt.Printf("SCLV check: state=%s records=%d pass=%d warning=%d violation=%d ledger_digest=%s\n",
			value.Summary.State, value.RecordsChecked, value.Summary.Pass,
			value.Summary.Warning, value.Summary.Violation, value.Ledger.Digest)
		if value.Summary.State != "valid" || value.Summary.Violation != 0 {
			return fmt.Errorf("SCLV ledger check reported violations")
		}
		return nil
	case "propose":
		var value struct {
			ProposalID            string `json:"proposal_id"`
			ProposalDigest        string `json:"proposal_digest"`
			CanonicalApplyEnabled bool   `json:"canonical_apply_enabled"`
			Authority             struct {
				Ratified bool `json:"ratified"`
			} `json:"authority"`
			Operations []struct {
				Type       string `json:"type"`
				TargetPath string `json:"target_path"`
			} `json:"operations"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.ProposalID == "" || len(value.Operations) != 1 {
			return fmt.Errorf("SCLV proposal result is incomplete")
		}
		fmt.Printf("SCLV proposal: id=%s digest=%s operation=%s target=%s ratified=%t apply=%t\n",
			value.ProposalID, value.ProposalDigest, value.Operations[0].Type,
			value.Operations[0].TargetPath, value.Authority.Ratified, value.CanonicalApplyEnabled)
		return nil
	case "recover":
		var value struct {
			Action            string `json:"action"`
			JournalDigest     string `json:"journal_digest"`
			DeleteRecommended bool   `json:"delete_recommended"`
			JournalMutated    bool   `json:"journal_mutated"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.Action == "" || value.JournalDigest == "" {
			return fmt.Errorf("SCLV recovery result is incomplete")
		}
		fmt.Printf("SCLV recovery: action=%s journal_digest=%s journal_mutated=%t delete_recommended=%t apply=false\n",
			value.Action, value.JournalDigest, value.JournalMutated, value.DeleteRecommended)
		return nil
	case "project":
		var value struct {
			RecordCount      uint64 `json:"record_count"`
			ProjectionDigest string `json:"projection_digest"`
			Noncanonical     bool   `json:"noncanonical"`
			Rebuildable      bool   `json:"rebuildable"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.ProjectionDigest == "" {
			return fmt.Errorf("SCLV projection result is incomplete")
		}
		fmt.Printf("SCLV projection: records=%d digest=%s noncanonical=%t rebuildable=%t\n",
			value.RecordCount, value.ProjectionDigest, value.Noncanonical, value.Rebuildable)
		return nil
	default:
		return fmt.Errorf("unsupported SCLV result")
	}
}

func runSTAV(subcommand string, options stavOptions) error {
	topsID := &options.topsID
	scope := &options.scope
	jsonOutput := options.jsonOutput
	query := options.query
	throughSequence := options.throughSequence
	verifyAfter := options.verifyAfter
	verifyThrough := options.verifyThrough
	if *topsID == "" {
		return fmt.Errorf("--tops-id is required")
	}
	if _, err := stavclient.SocketForTOPS(*scope, *topsID); err != nil {
		return err
	}
	if subcommand == "query" {
		query.TOPSID = *topsID
		if throughSequence.set {
			value := throughSequence.value
			query.ThroughSequence = &value
		}
		if _, err := stavprotocol.EncodeQuery(query); err != nil {
			return fmt.Errorf("invalid bounded STAV query: %w", err)
		}
	}
	if subcommand == "verify" && verifyThrough.set && verifyThrough.value <= verifyAfter {
		return fmt.Errorf("verification through-sequence must follow after-sequence")
	}
	client, err := stavclient.NewForTOPS(*scope, *topsID)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	requestID, err := stavprotocol.GenerateUUIDv4()
	if err != nil {
		return err
	}
	request := stavprotocol.LocalRequest{
		Operation: subcommand,
		RequestID: requestID,
		Schema:    stavprotocol.SchemaLocalRequest,
		TOPSID:    *topsID,
	}
	switch subcommand {
	case "query":
		request.Query = &query
	case "verify":
		verify := stavprotocol.VerifyRequest{AfterSequence: verifyAfter}
		if verifyThrough.set {
			value := verifyThrough.value
			verify.ThroughSequence = &value
		}
		request.Verify = &verify
	case "doctor":
		request.Operation = stavprotocol.LocalOperationStatus
	}
	response, err := client.Do(ctx, request)
	if err != nil {
		return err
	}
	if response.Disposition != stavprotocol.LocalDispositionSucceeded {
		return fmt.Errorf("STAV %s rejected: %s", subcommand, response.ReasonCode)
	}
	switch subcommand {
	case "status":
		if jsonOutput {
			return printSTAVJSON(response.Status)
		}
		fmt.Printf("STAV: ready=%t tops_id=%s mode=%s events=%d ledger_bytes=%d storage=%s\n", response.Status.Ready, response.Status.TOPSID, response.Status.Mode, response.Status.Events, response.Status.LedgerBytes, response.Status.StorageState)
		return nil
	case "verify":
		if jsonOutput {
			return printSTAVJSON(response.Verification)
		}
		fmt.Printf("STAV verification: state=%s tops_id=%s after=%d through=%d checked=%d\n", response.Verification.Result.State, response.Verification.TOPSID, response.Verification.AfterSequence, response.Verification.ThroughSequence, response.Verification.EventsChecked)
		if response.Verification.Result.State != "verified" {
			return fmt.Errorf("STAV verification failed at sequence %d: %s", response.Verification.Result.AtSequence, response.Verification.Result.ReasonCode)
		}
		return nil
	case "query":
		if jsonOutput {
			return printSTAVJSON(response.Page)
		}
		for _, entry := range response.Page.Entries {
			fmt.Printf("STAV event: sequence=%d timestamp=%s class=%s operation=%s outcome=%s reason=%s request_id=%s\n", entry.Sequence, entry.Projection.Timestamp, entry.Projection.EventClass, entry.Projection.OperationID, entry.Projection.Outcome, entry.Projection.ReasonCode, entry.Projection.RequestID)
		}
		fmt.Printf("STAV query: entries=%d next=%s\n", len(response.Page.Entries), response.Page.Next.State)
		return nil
	case "doctor":
		if !response.Status.Ready {
			return fmt.Errorf("STAV append authority is not ready")
		}
		verifyID, err := stavprotocol.GenerateUUIDv4()
		if err != nil {
			return err
		}
		verificationResponse, err := client.Do(ctx, stavprotocol.LocalRequest{
			Operation: stavprotocol.LocalOperationVerify,
			RequestID: verifyID,
			Schema:    stavprotocol.SchemaLocalRequest,
			TOPSID:    *topsID,
			Verify:    &stavprotocol.VerifyRequest{AfterSequence: 0},
		})
		if err != nil {
			return err
		}
		if verificationResponse.Disposition != stavprotocol.LocalDispositionSucceeded || verificationResponse.Verification.Result.State != "verified" {
			return fmt.Errorf("STAV doctor chain verification failed")
		}
		fmt.Printf("STAV doctor: tops_id=%s ready=true events=%d storage=%s chain=verified endpoint=authenticated\n", response.Status.TOPSID, response.Status.Events, response.Status.StorageState)
		fmt.Println("STAV doctor: checks passed")
		return nil
	}
	return fmt.Errorf("unsupported STAV command")
}

func printSTAVJSON(value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

type optionalUint64 struct {
	set   bool
	value uint64
}

func (v *optionalUint64) String() string {
	if !v.set {
		return ""
	}
	return strconv.FormatUint(v.value, 10)
}

func (*optionalUint64) Type() string { return "uint64" }

func (v *optionalUint64) Set(value string) error {
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid non-negative integer: %w", err)
	}
	v.set = true
	v.value = parsed
	return nil
}

func runSSIAG(subcommand string, options ssiagOptions) error {
	jsonOutput := &options.jsonOutput
	scope := &options.scope
	topsID := &options.topsID
	if *topsID == "" {
		return fmt.Errorf("--tops-id or SYMPHONY_SSIAG_TOPS_ID is required")
	}
	client, err := ssiagclient.NewForTOPS(*scope, *topsID, 4*time.Second)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	switch subcommand {
	case "status":
		status, err := requireSSIAGStatus(ctx, client, *topsID, *scope)
		if err != nil {
			return err
		}
		if *jsonOutput {
			return printSSIAGJSON(status)
		}
		fmt.Printf("SSIAG: %s version=%s ready=%t tops_id=%s tops_name=%q mode=%s providers=%d\n", status.Name, status.Version, status.Ready, status.TOPSID, status.TOPSName, status.Mode, status.ProviderCount)
		return nil
	case "providers":
		if _, err := requireSSIAGStatus(ctx, client, *topsID, *scope); err != nil {
			return err
		}
		providers, err := client.Providers(ctx)
		if err != nil {
			return err
		}
		if *jsonOutput {
			return printSSIAGJSON(providers)
		}
		if len(providers.Providers) == 0 {
			fmt.Println("SSIAG providers: none declared")
			return nil
		}
		for _, provider := range providers.Providers {
			fmt.Printf("SSIAG provider: %s kind=%s status=%s\n", provider.Name, provider.Kind, provider.Status)
		}
		return nil
	case "doctor":
		if *jsonOutput {
			return fmt.Errorf("SSIAG doctor does not support --json")
		}
		status, err := requireSSIAGStatus(ctx, client, *topsID, *scope)
		if err != nil {
			return err
		}
		providers, err := client.Providers(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("SSIAG doctor: schema=%s tops_id=%s ready=true providers=%d\n", status.Schema, status.TOPSID, len(providers.Providers))
		fmt.Println("SSIAG doctor: checks passed")
		return nil
	default:
		return fmt.Errorf("unknown SSIAG subcommand %q", subcommand)
	}
}

func requireSSIAGStatus(ctx context.Context, client *ssiagclient.Client, topsID, scope string) (ssiagclient.Status, error) {
	status, err := client.Status(ctx)
	if err != nil {
		return ssiagclient.Status{}, err
	}
	if status.TOPSID != topsID {
		return ssiagclient.Status{}, fmt.Errorf("SSIAG response TOPS ID does not match requested identity")
	}
	if status.Mode != scope {
		return ssiagclient.Status{}, fmt.Errorf("SSIAG response mode does not match requested scope")
	}
	if !status.Ready {
		return ssiagclient.Status{}, fmt.Errorf("SSIAG is not ready")
	}
	return status, nil
}

func printSSIAGJSON(value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func runDoctor() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("could not find Symphony repository root: %w", err)
	}
	fmt.Printf("found repository root: %s\n", repoRoot)

	expectedModules := []string{
		"node-troll",
		"bus-troll",
		"hotpath-runtime",
	}

	for _, mod := range expectedModules {
		modPath := filepath.Join(repoRoot, "modules", mod)
		if !repository.IsDir(modPath) {
			return fmt.Errorf("missing expected module directory: modules/%s", mod)
		}
		fmt.Printf("verified module exists: modules/%s\n", mod)
	}

	validatorPath := filepath.Join(repoRoot, "tools", "symphony-validator")
	if !repository.IsDir(validatorPath) {
		return fmt.Errorf("missing validator directory: tools/symphony-validator")
	}
	fmt.Println("verified validator exists: tools/symphony-validator")

	fmt.Println("doctor checks passed")
	return nil
}

func runModules() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("could not find Symphony repository root: %w", err)
	}

	output, err := modules.List(repoRoot)
	for _, line := range output {
		fmt.Println(line)
	}
	if err != nil {
		fmt.Println("modules: checks failed")
		return err
	}
	return nil
}

func runModulesCheck() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("could not find Symphony repository root: %w", err)
	}

	output, err := modules.CheckAll(repoRoot)
	for _, line := range output {
		fmt.Println(line)
	}
	if err != nil {
		fmt.Println("modules check: checks failed")
		return err
	}
	return nil
}

func runModuleInspect(moduleName string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("could not find Symphony repository root: %w", err)
	}

	output, err := modules.Inspect(repoRoot, moduleName)
	for _, line := range output {
		fmt.Println(line)
	}
	if err != nil {
		fmt.Println("inspection: checks failed")
		return err
	}
	return nil
}

func runModuleCheck(moduleName string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("could not find Symphony repository root: %w", err)
	}

	output, err := modules.Check(repoRoot, moduleName)
	for _, line := range output {
		fmt.Println(line)
	}
	if err != nil {
		fmt.Println("module check: checks failed")
		return err
	}
	return nil
}

func runContracts() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("could not find Symphony repository root: %w", err)
	}

	output, err := contracts.Verify(repoRoot)
	for _, line := range output {
		fmt.Println(line)
	}
	if err != nil {
		fmt.Println("contracts: checks failed")
		return err
	}
	return nil
}

func runModuleMetadata(moduleName string, jsonOutput bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("could not find Symphony repository root: %w", err)
	}

	if jsonOutput {
		outputBytes, err := modules.MetadataJSON(repoRoot, moduleName)
		if err != nil {
			fmt.Printf("module metadata failed: %v\n", err)
			return err
		}
		fmt.Println(string(outputBytes))
		return nil
	}

	output, err := modules.Metadata(repoRoot, moduleName)
	for _, line := range output {
		fmt.Println(line)
	}
	if err != nil {
		fmt.Println("module metadata: checks failed")
		return err
	}
	return nil
}

func runModulesMetadata(jsonOutput bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("could not find Symphony repository root: %w", err)
	}

	if jsonOutput {
		outputBytes, err := modules.MetadataAllJSON(repoRoot)
		if err != nil {
			fmt.Printf("modules metadata failed: %v\n", err)
			return err
		}
		fmt.Println(string(outputBytes))
		return nil
	}

	output, err := modules.MetadataAll(repoRoot)
	for _, line := range output {
		fmt.Println(line)
	}
	if err != nil {
		fmt.Println("modules metadata: checks failed")
		return err
	}
	return nil
}

func runInventory(jsonOutput bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("could not find Symphony repository root: %w", err)
	}

	if jsonOutput {
		outputBytes, err := inventory.SnapshotJSON(repoRoot)
		if err != nil {
			return err
		}
		fmt.Println(string(outputBytes))
		return nil
	}

	output, err := inventory.Snapshot(repoRoot)
	if err != nil {
		return err
	}
	for _, line := range output {
		fmt.Println(line)
	}
	return nil
}

func runInventoryDigest(jsonOutput bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("could not find Symphony repository root: %w", err)
	}

	if jsonOutput {
		outputBytes, err := inventory.DigestJSON(repoRoot)
		if err != nil {
			return err
		}
		fmt.Println(string(outputBytes))
		return nil
	}

	output, err := inventory.Digest(repoRoot)
	if err != nil {
		return err
	}
	for _, line := range output {
		fmt.Println(line)
	}
	return nil
}

func runStatus(jsonOutput bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("could not find Symphony repository root: %w", err)
	}

	if jsonOutput {
		outputBytes, err := status.ReportJSON(repoRoot)
		if err != nil {
			return err
		}
		fmt.Println(string(outputBytes))
		return nil
	}

	output, err := status.Report(repoRoot)
	if err != nil {
		return err
	}
	for _, line := range output {
		fmt.Println(line)
	}
	return nil
}
