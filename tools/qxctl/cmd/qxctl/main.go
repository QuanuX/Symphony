package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/contracts"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/inventory"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgebinding"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/modules"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/repository"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/ssiagclient"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/status"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/stavclient"
	qxversion "github.com/QuanuX/Symphony/tools/qxctl/internal/version"
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
	fmt.Println("  validate scan --tops-id UUID --prefix PATH [--profile-id ID] [--baseline-id ID] [--json] Run full structured repository validation")
	fmt.Println("  validate debug --tops-id UUID --prefix PATH [--rule ID|--record ID|--path PATH|--delta CLASS] [--json] Filter display after a full scan")
	fmt.Println("  validate profile list|show|set|remove --tops-id UUID [...] Administer protected warning policy")
	fmt.Println("  validate baseline create|show|remove --tops-id UUID [...] Administer protected warning baselines")
	fmt.Println("  ssiag status --tops-id UUID [--scope user|system] [--json] Read safe SSIAG status")
	fmt.Println("  ssiag providers --tops-id UUID [--scope user|system] [--json] List safe provider metadata")
	fmt.Println("  ssiag doctor --tops-id UUID [--scope user|system] Verify local SSIAG availability")
	fmt.Println("  ssiag grants lifecycle --tops-id UUID --subject-id ID [--profile-id ID] [--json] Generate exact caller-neutral lifecycle grant input")
	fmt.Println("  ssiag policy status --tops-id UUID [--scope user|system] [--json] Read protected local policy metadata")
	fmt.Println("  ssiag policy propose --tops-id UUID --operation-id ID --expected-policy-digest DIGEST (--input FILE|--reset) [--json] Prepare a caller-neutral local policy proposal")
	fmt.Println("  ssiag policy apply --tops-id UUID --input FILE [--json] Audit and atomically apply an exact policy proposal")
	fmt.Println("  ssiag policy recover --tops-id UUID --operation-id ID (--expected-attempt-digest DIGEST|--discover) [--json] Recover a durable policy attempt")
	fmt.Println("  stav status --tops-id UUID [--scope user|system] [--json] Read authenticated STAV status")
	fmt.Println("  stav verify --tops-id UUID [--scope user|system] [--json] Verify the STAV digest chain")
	fmt.Println("  stav query --tops-id UUID [--scope user|system] [bounded filters] [--json] Query authorized STAV projections")
	fmt.Println("  stav doctor --tops-id UUID [--scope user|system] Run authenticated STAV diagnostics")
	fmt.Println("  knowledge engines list [--state-root PATH] [--json] List the user-default exact engine bindings")
	fmt.Println("  knowledge engines inspect ROLE [--state-root PATH] [--json] Inspect one exact engine binding")
	fmt.Println("  knowledge engines doctor [--state-root PATH] [--json] Verify every bound installation")
	fmt.Println("  knowledge engines bind ROLE --prefix PATH [--version VERSION] --expected-registry-digest STATE [--json] Bind an exact installation")
	fmt.Println("  knowledge engines unbind ROLE --expected-registry-digest DIGEST [--json] Remove one exact binding")
	fmt.Println("  knowledge reconcile compatibility [--state-root PATH] [--repo PATH] [--json] Negotiate coordinator compatibility")
	fmt.Println("  knowledge reconcile begin --operation-id ID --expected-journal-digest STATE --path FILE... [--json] Begin a durable context")
	fmt.Println("  knowledge reconcile status [--state-root PATH] [--repo PATH] [--json] Read durable context status")
	fmt.Println("  knowledge reconcile checkpoint --operation-id ID --expected-journal-digest DIGEST [--json] Record a durable checkpoint")
	fmt.Println("  knowledge reconcile close --operation-id ID --expected-journal-digest DIGEST [--json] Close a durable context")
	fmt.Println("  knowledge reconcile recover --operation-id ID (--expected-journal-digest DIGEST|--discover) [--json] Repair from durable evidence")
	fmt.Println("  knowledge session begin --tops-id UUID --operation-id ID --expected-journal-digest STATE [--context-ref REF...] [--json] Begin an authenticated authority epoch")
	fmt.Println("  knowledge session status --tops-id UUID [--json] Read authenticated session status")
	fmt.Println("  knowledge session checkpoint --tops-id UUID --operation-id ID --expected-journal-digest DIGEST [--context-ref REF...] [--json] Checkpoint an authenticated session")
	fmt.Println("  knowledge session close --tops-id UUID --operation-id ID --expected-journal-digest DIGEST [--json] Close an authenticated session")
	fmt.Println("  knowledge session recover --tops-id UUID --operation-id ID (--expected-journal-digest DIGEST|--discover) [--json] Recover authenticated session evidence")
	fmt.Println("  knowledge session transition --tops-id UUID --event login|refresh|logout --event-id ID [--recover] [--json] Converge an explicit host lifecycle event")
	fmt.Println("  knowledge session features begin|status|checkpoint|close|recover --tops-id UUID [...] Maintain persistent SSFV session evidence")
	fmt.Println("  knowledge lifecycle profile list --tops-id UUID [--profile-id ID] [--json] List protected lifecycle profiles")
	fmt.Println("  knowledge lifecycle profile show --tops-id UUID [--profile-id ID] [--json] Read one protected lifecycle profile")
	fmt.Println("  knowledge lifecycle profile set --tops-id UUID --input FILE --expected-profile-digest STATE [--profile-id ID] [--json] Commit exact desired profile intent")
	fmt.Println("  knowledge lifecycle profile remove --tops-id UUID --expected-profile-digest DIGEST [--profile-id ID] [--json] Remove one protected lifecycle profile")
	fmt.Println("  knowledge lifecycle ownership status|reconcile|adopt|release --tops-id UUID --root PATH [--profile-id ID] [...] Administer shared-root claims")
	fmt.Println("  knowledge lifecycle host install|update|status|reconcile|enable|disable|uninstall|run --tops-id UUID [--profile-id ID] [...] Administer the Linux report-only boot receptor")
	fmt.Println("  knowledge lifecycle observe --tops-id UUID [--profile-id ID | --root PATH...] [--json] Inventory fixed-layout receipts without execution")
	fmt.Println("  knowledge lifecycle report --tops-id UUID [--profile-id ID] [--prior-applied-state-digest DIGEST] [--json] Re-observe and produce a dynamic report-only plan")
	fmt.Println("  knowledge lifecycle boot --tops-id UUID --operation-id ID --expected-journal-digest STATE [--json] Durably record or replan report-only boot evidence")
	fmt.Println("  knowledge lifecycle status --tops-id UUID [--profile-id ID] [--json] Read durable lifecycle boot state")
	fmt.Println("  knowledge lifecycle recover --tops-id UUID --operation-id ID (--expected-journal-digest DIGEST|--discover) [--json] Repair uniquely linked lifecycle boot evidence")
	fmt.Println("  knowledge lifecycle apply --tops-id UUID --operation-id ID --source-journal-digest DIGEST --expected-apply-journal-digest STATE --expected-applied-state-digest STATE [--source-root PATH...] [--json] Converge exact apply-compatible lifecycle actions")
	fmt.Println("  knowledge lifecycle apply-status --tops-id UUID [--profile-id ID] [--json] Read durable apply-capable lifecycle state")
	fmt.Println("  knowledge lifecycle apply-recover --tops-id UUID --operation-id ID (--expected-apply-journal-digest DIGEST|--discover) [--json] Repair uniquely linked apply-capable lifecycle evidence")
	fmt.Println("  maestro inspect --prefix PATH --tops-id UUID --receptor-id ID [--version VERSION] [--json] Inspect an exact local receptor")
	fmt.Println("  maestro inventory --prefix PATH --tops-id UUID [--version VERSION] [--json] Read derived receptor inventory")
	fmt.Println("  maestro status --prefix PATH --tops-id UUID --receptor-id ID [--component-id ID] [--json] Read authenticated presence")
	fmt.Println("  maestro recover --prefix PATH --tops-id UUID --receptor-id ID --operation-id ID (--expected-registry-digest DIGEST|--discover) [--json] Repair unique presence evidence")
	fmt.Println("  skvi inspect --prefix PATH [--version VERSION] [--json] Inspect an exact installed SKVI engine")
	fmt.Println("  skvi check --prefix PATH [--version VERSION] [--json] Check canonical SKVI index truth")
	fmt.Println("  skvi propose --prefix PATH --input FILE [--version VERSION] [--json] Prepare a caller-declared proposal")
	fmt.Println("  skvi project --prefix PATH [--version VERSION] [--json] Build a disposable SKVI projection")
	fmt.Println("  sclv inspect --prefix PATH [--version VERSION] [--json] Inspect an exact installed SCLV engine")
	fmt.Println("  sclv check --prefix PATH [--version VERSION] [--json] Check canonical SCLV ledger truth")
	fmt.Println("  sclv propose --prefix PATH --input FILE [--version VERSION] [--json] Prepare a provider-neutral record proposal")
	fmt.Println("  sclv recover --prefix PATH --input FILE [--version VERSION] [--json] Reconcile ephemeral SCLV closure evidence")
	fmt.Println("  sclv project --prefix PATH [--version VERSION] [--json] Build a disposable SCLV projection")
	fmt.Println("  sclv evidence local-git --prefix PATH --input FILE [--version VERSION] [--json] Normalize receipt-owned local Git evidence")
	fmt.Println("  sclv evidence airgap --prefix PATH --input FILE [--version VERSION] [--json] Normalize receipt-owned air-gap evidence")
	fmt.Println("  sacv inspect --prefix PATH [--version VERSION] [--json] Inspect an exact installed SACV engine")
	fmt.Println("  sacv check --prefix PATH [--version VERSION] [--json] Check canonical SACV registry and API contracts")
	fmt.Println("  sacv diff --prefix PATH --input FILE [--version VERSION] [--json] Compare bounded OpenAPI revisions")
	fmt.Println("  sacv propose --prefix PATH --input FILE [--version VERSION] [--json] Prepare a caller-declared registry proposal")
	fmt.Println("  sacv project --prefix PATH [--version VERSION] [--json] Build a disposable SACV inventory")
	fmt.Println("  sodv inspect --prefix PATH [--version VERSION] [--json] Inspect an exact installed SODV engine")
	fmt.Println("  sodv check --prefix PATH [--version VERSION] [--json] Check canonical release transaction truth")
	fmt.Println("  sodv verify --prefix PATH --input FILE [--version VERSION] [--json] Verify caller-observed publication state")
	fmt.Println("  sodv propose --prefix PATH --input FILE [--version VERSION] [--json] Prepare a release-record proposal")
	fmt.Println("  sodv recover --prefix PATH --input FILE [--version VERSION] [--json] Reconcile an interrupted release journal")
	fmt.Println("  sodv project --prefix PATH [--version VERSION] [--json] Build a disposable release inventory")
	fmt.Println("  ssfv inspect --prefix PATH [--version VERSION] [--json] Inspect an exact installed SSFV engine")
	fmt.Println("  ssfv check --prefix PATH [--version VERSION] [freshness flags] [--json] Check canonical semantic-feature truth")
	fmt.Println("  ssfv diff --prefix PATH --input FILE [--version VERSION] [--json] Compare a semantic baseline with live truth")
	fmt.Println("  ssfv propose --prefix PATH --input FILE [--version VERSION] [--json] Prepare a caller-declared semantic proposal")
	fmt.Println("  ssfv graph --prefix PATH [--version VERSION] [--json] Build a disposable semantic-feature graph")
}

func runKnowledgeEngines(operation string, options knowledgeEngineOptions) error {
	store, err := knowledgebinding.NewStore(options.stateRoot)
	if err != nil {
		return err
	}
	switch operation {
	case "list":
		snapshot, err := store.Snapshot()
		if err != nil {
			return err
		}
		if options.jsonOutput {
			return printIndentedJSON(snapshot)
		}
		if !snapshot.Exists {
			fmt.Println("Knowledge engine bindings: absent profile=user-default canonical=false")
			return nil
		}
		fmt.Printf(
			"Knowledge engine bindings: profile=%s generation=%d digest=%s canonical=false\n",
			snapshot.Registry.ProfileID, snapshot.Registry.Generation, snapshot.Registry.RegistryDigest,
		)
		for _, binding := range snapshot.Registry.Bindings {
			fmt.Printf(
				"Knowledge engine binding: role=%s engine=%s version=%s state=%s prefix=%s\n",
				binding.Role, binding.EngineID, binding.Version, binding.State, binding.Prefix,
			)
		}
		return nil
	case "inspect":
		snapshot, err := store.Snapshot()
		if err != nil {
			return err
		}
		if !snapshot.Exists {
			return fmt.Errorf("knowledge engine binding registry is absent")
		}
		for _, binding := range snapshot.Registry.Bindings {
			if binding.Role != options.role {
				continue
			}
			if options.jsonOutput {
				return printIndentedJSON(map[string]any{
					"registry_digest": snapshot.Registry.RegistryDigest,
					"binding":         binding,
					"canonical":       false,
				})
			}
			fmt.Printf(
				"Knowledge engine binding: role=%s module=%s engine=%s version=%s state=%s receipt_digest=%s executable_digest=%s canonical=false\n",
				binding.Role, binding.ModuleID, binding.EngineID, binding.Version, binding.State,
				binding.ReceiptDigest, binding.ExecutableDigest,
			)
			return nil
		}
		return fmt.Errorf("knowledge engine role %q is not bound", options.role)
	case "doctor":
		report, err := store.Doctor()
		if err != nil {
			return err
		}
		if options.jsonOutput {
			if err := printIndentedJSON(report); err != nil {
				return err
			}
		} else {
			fmt.Printf(
				"Knowledge engine binding doctor: healthy=%t registry_exists=%t checks=%d\n",
				report.Healthy, report.RegistryExists, len(report.Results),
			)
			for _, result := range report.Results {
				fmt.Printf(
					"Knowledge engine binding check: role=%s healthy=%t code=%s message=%q\n",
					result.Role, result.Healthy, result.Code, result.Message,
				)
			}
		}
		if !report.Healthy {
			return fmt.Errorf("knowledge engine binding doctor reported unhealthy state")
		}
		return nil
	case "bind":
		if options.prefix == "" {
			return fmt.Errorf("--prefix is required")
		}
		if options.expectedRegistryDigest == "" {
			return fmt.Errorf("--expected-registry-digest is required")
		}
		registry, changed, err := store.Bind(
			options.role, options.prefix, options.version, options.expectedRegistryDigest)
		if err != nil {
			return err
		}
		if options.jsonOutput {
			return printIndentedJSON(map[string]any{"changed": changed, "registry": registry})
		}
		fmt.Printf(
			"Knowledge engine binding: operation=bind role=%s changed=%t generation=%d digest=%s canonical=false\n",
			options.role, changed, registry.Generation, registry.RegistryDigest,
		)
		return nil
	case "unbind":
		if options.expectedRegistryDigest == "" {
			return fmt.Errorf("--expected-registry-digest is required")
		}
		registry, changed, err := store.Unbind(options.role, options.expectedRegistryDigest)
		if err != nil {
			return err
		}
		if options.jsonOutput {
			return printIndentedJSON(map[string]any{"changed": changed, "registry": registry})
		}
		fmt.Printf(
			"Knowledge engine binding: operation=unbind role=%s changed=%t generation=%d digest=%s canonical=false\n",
			options.role, changed, registry.Generation, registry.RegistryDigest,
		)
		return nil
	default:
		return fmt.Errorf("unsupported knowledge engines operation")
	}
}

func runKnowledgeReconcile(operation string, options knowledgeReconcileOptions) error {
	if operation == "begin" || operation == "checkpoint" ||
		operation == "close" || operation == "recover" {
		if options.operationID == "" {
			return fmt.Errorf("--operation-id is required")
		}
	}
	expected := options.expectedJournalDigest
	if operation == "begin" || operation == "checkpoint" || operation == "close" {
		if expected == "" {
			return fmt.Errorf("--expected-journal-digest is required")
		}
	}
	if operation == "begin" && len(options.paths) == 0 {
		return fmt.Errorf("at least one --path is required")
	}
	if operation == "recover" {
		if options.discover && expected != "" {
			return fmt.Errorf("--discover and --expected-journal-digest are mutually exclusive")
		}
		if options.discover {
			expected = "discover"
		}
		if expected == "" {
			return fmt.Errorf("--expected-journal-digest or --discover is required")
		}
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

	store, err := knowledgebinding.NewStore(options.stateRoot)
	if err != nil {
		return err
	}
	snapshot, err := store.Snapshot()
	if err != nil {
		return err
	}
	if !snapshot.Exists {
		return fmt.Errorf("knowledge engine binding registry is absent")
	}

	type inventoryEntry struct {
		Role             string `json:"role"`
		ModuleID         string `json:"module_id"`
		EngineID         string `json:"engine_id"`
		Version          string `json:"version"`
		ReceiptDigest    string `json:"receipt_digest"`
		ExecutableDigest string `json:"executable_digest"`
	}
	inventoryEntries := make([]inventoryEntry, 0, len(snapshot.Registry.Bindings))
	var coordinator *knowledgebinding.Binding
	for index := range snapshot.Registry.Bindings {
		binding := &snapshot.Registry.Bindings[index]
		installed, inspectErr := knowledgeengine.InspectInstallation(
			binding.Role, binding.Prefix, binding.Version)
		if inspectErr != nil {
			return fmt.Errorf("bound %s installation is unavailable: %w", binding.Role, inspectErr)
		}
		if installed.ModuleID != binding.ModuleID || installed.EngineID != binding.EngineID ||
			installed.ReceiptPath != binding.ReceiptPath ||
			installed.ReceiptDigest != binding.ReceiptDigest ||
			installed.ExecutablePath != binding.ExecutablePath ||
			installed.ExecutableDigest != binding.ExecutableDigest {
			return fmt.Errorf("bound %s installation no longer matches its content-addressed identity", binding.Role)
		}
		inventoryEntries = append(inventoryEntries, inventoryEntry{
			Role: binding.Role, ModuleID: binding.ModuleID, EngineID: binding.EngineID,
			Version: binding.Version, ReceiptDigest: binding.ReceiptDigest,
			ExecutableDigest: binding.ExecutableDigest,
		})
		if binding.Role == "coordinator" {
			coordinator = binding
		}
	}
	if coordinator == nil {
		return fmt.Errorf("knowledge-session coordinator is not bound")
	}

	var operationID any
	var expectedDigest any
	if options.operationID != "" {
		operationID = options.operationID
	}
	if expected != "" {
		expectedDigest = expected
	}
	paths := options.paths
	if paths == nil {
		paths = []string{}
	}
	payload, err := json.Marshal(map[string]any{
		"protocol":                "symphony.knowledge.reconciliation-command.v1",
		"operation":               operation,
		"state_root":              store.StateRoot(),
		"operation_id":            operationID,
		"expected_journal_digest": expectedDigest,
		"paths":                   paths,
		"binding_registry_digest": snapshot.Registry.RegistryDigest,
		"engine_inventory":        inventoryEntries,
		"client": map[string]any{
			"client_id":              "qxctl",
			"client_version":         strings.ReplaceAll(qxversion.Version, " ", "-"),
			"process_protocols":      []string{"symphony.knowledge.engine-process.v1"},
			"journal_read_versions":  []uint64{1},
			"journal_write_versions": []uint64{1},
			"capabilities": []string{
				"atomic-head-v1",
				"content-snapshot-v1",
				"discovery-recovery-v1",
				"dual-slot-journal-v1",
				"expected-state-cas-v1",
				"idempotent-operation-v1",
				"nonblocking-lock-v1",
				"opaque-extension-preservation-v1",
				"recovery-forward-v1",
			},
		},
	})
	if err != nil {
		return fmt.Errorf("encode reconciliation request: %w", err)
	}
	response, err := knowledgeengine.InvokeCoordinator(
		context.Background(),
		coordinator.Prefix,
		coordinator.Version,
		repoRoot,
		operation,
		payload,
	)
	if err != nil {
		return err
	}
	result, err := validateReconciliationResult(response.Result, operation)
	if err != nil {
		return err
	}
	if options.jsonOutput {
		var display any
		if err := json.Unmarshal(response.Result, &display); err != nil {
			return fmt.Errorf("decode reconciliation result: %w", err)
		}
		return printIndentedJSON(display)
	}
	digest := "absent"
	if result.JournalDigest != nil {
		digest = *result.JournalDigest
	}
	fmt.Printf(
		"Knowledge reconciliation: operation=%s compatibility=%s present=%t changed=%t recovered=%t read_only=%t digest=%s canonical=false\n",
		result.Operation, result.Mode, *result.JournalPresent,
		*result.Changed, *result.Recovered, *result.ReadOnly, digest,
	)
	return nil
}

type sessionInvocation struct {
	Result sessionResult
	Raw    json.RawMessage
}

func runKnowledgeSession(operation string, options knowledgeSessionOptions) error {
	invocation, err := executeKnowledgeSession(operation, options)
	if err != nil {
		return err
	}
	if options.jsonOutput {
		var display any
		if err := json.Unmarshal(invocation.Raw, &display); err != nil {
			return fmt.Errorf("decode authenticated session result: %w", err)
		}
		return printIndentedJSON(display)
	}
	digest := "absent"
	if invocation.Result.JournalDigest != nil {
		digest = *invocation.Result.JournalDigest
	}
	fmt.Printf(
		"Knowledge session: operation=%s state=%s present=%t changed=%t recovered=%t read_only=%t digest=%s canonical=false\n",
		operation, invocation.Result.EffectiveState, *invocation.Result.JournalPresent,
		*invocation.Result.Changed, *invocation.Result.Recovered, *invocation.Result.ReadOnly, digest,
	)
	return nil
}

func executeKnowledgeSession(operation string, options knowledgeSessionOptions) (sessionInvocation, error) {
	if options.topsID == "" {
		return sessionInvocation{}, fmt.Errorf("--tops-id is required")
	}
	if options.scope != "user" && options.scope != "system" {
		return sessionInvocation{}, fmt.Errorf("--scope must be user or system")
	}
	if options.ttl <= 0 || options.ttl > 24*time.Hour {
		return sessionInvocation{}, fmt.Errorf("--ttl must be greater than zero and no more than 24h")
	}
	if operation != "status" && options.operationID == "" {
		return sessionInvocation{}, fmt.Errorf("--operation-id is required")
	}
	expected := options.expectedJournalDigest
	if operation == "begin" || operation == "checkpoint" || operation == "close" {
		if expected == "" {
			return sessionInvocation{}, fmt.Errorf("--expected-journal-digest is required")
		}
	}
	if operation == "recover" {
		if options.discover && expected != "" {
			return sessionInvocation{}, fmt.Errorf("--discover and --expected-journal-digest are mutually exclusive")
		}
		if options.discover {
			expected = "discover"
		}
		if expected == "" {
			return sessionInvocation{}, fmt.Errorf("--expected-journal-digest or --discover is required")
		}
	}

	repoRoot, err := resolveKnowledgeRepository(options.repository)
	if err != nil {
		return sessionInvocation{}, err
	}
	store, err := knowledgebinding.NewStore(options.stateRoot)
	if err != nil {
		return sessionInvocation{}, err
	}
	bindingSnapshot, err := store.Snapshot()
	if err != nil {
		return sessionInvocation{}, err
	}
	if !bindingSnapshot.Exists {
		return sessionInvocation{}, fmt.Errorf("knowledge engine binding registry is absent")
	}
	var coordinator *knowledgebinding.Binding
	for index := range bindingSnapshot.Registry.Bindings {
		if bindingSnapshot.Registry.Bindings[index].Role == "coordinator" {
			coordinator = &bindingSnapshot.Registry.Bindings[index]
			break
		}
	}
	if coordinator == nil {
		return sessionInvocation{}, fmt.Errorf("knowledge-session coordinator is not bound")
	}
	installed, err := knowledgeengine.InspectInstallation("coordinator", coordinator.Prefix, coordinator.Version)
	if err != nil {
		return sessionInvocation{}, fmt.Errorf("bound coordinator installation is unavailable: %w", err)
	}
	if installed.ModuleID != coordinator.ModuleID || installed.EngineID != coordinator.EngineID ||
		installed.ReceiptDigest != coordinator.ReceiptDigest || installed.ExecutableDigest != coordinator.ExecutableDigest {
		return sessionInvocation{}, fmt.Errorf("bound coordinator installation no longer matches its content-addressed identity")
	}

	ssiag, err := ssiagclient.NewForTOPS(options.scope, options.topsID, 4*time.Second)
	if err != nil {
		return sessionInvocation{}, err
	}
	authorizationContext, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if _, err := requireSSIAGStatus(authorizationContext, ssiag, options.topsID, options.scope); err != nil {
		return sessionInvocation{}, err
	}
	requestID, err := randomUUID()
	if err != nil {
		return sessionInvocation{}, err
	}
	correlationID, err := randomUUID()
	if err != nil {
		return sessionInvocation{}, err
	}
	now := time.Now().UTC().Truncate(time.Second)
	authorizationRequest := ssiagclient.AuthorizationRequest{
		Schema: "symphony.ssiag.authorization-request.v1", RequestID: requestID,
		CorrelationID: correlationID, Operation: "symphony.knowledge.session." + operation,
		Resource: sessionRepositoryResource(repoRoot), Audience: "qxctl", Scope: "tops:" + options.topsID,
		RequestedAt: now, RequestedExpiresAt: now.Add(options.ttl).UTC().Truncate(time.Second),
	}
	decision, err := ssiag.Authorize(authorizationContext, authorizationRequest)
	if err != nil {
		return sessionInvocation{}, err
	}
	if err := validateSessionAuthorization(decision, authorizationRequest, options.topsID); err != nil {
		return sessionInvocation{}, err
	}

	var operationID any
	var expectedDigest any
	if options.operationID != "" {
		operationID = options.operationID
	}
	if expected != "" {
		expectedDigest = expected
	}
	contextRefs := options.contextRefs
	if contextRefs == nil {
		contextRefs = []string{}
	}
	engineOperation := "session_" + operation
	payload, err := json.Marshal(map[string]any{
		"protocol":                "symphony.knowledge.session-command.v1",
		"operation":               engineOperation,
		"state_root":              store.StateRoot(),
		"operation_id":            operationID,
		"expected_journal_digest": expectedDigest,
		"repository_root":         repoRoot,
		"context_refs":            contextRefs,
		"authorization_decision":  decision,
		"client": map[string]any{
			"client_id": "qxctl", "client_version": strings.ReplaceAll(qxversion.Version, " ", "-"),
			"process_protocols":     []string{"symphony.knowledge.engine-process.v1"},
			"journal_read_versions": []uint64{1}, "journal_write_versions": []uint64{1},
			"capabilities": []string{
				"atomic-head-v1", "authority-epoch-v1", "discovery-recovery-v1",
				"dual-slot-journal-v1", "expected-state-cas-v1", "idempotent-operation-v1",
				"nonblocking-lock-v1", "opaque-extension-preservation-v1",
				"recovery-forward-v1", "ssiag-capability-binding-v1",
			},
		},
	})
	if err != nil {
		return sessionInvocation{}, fmt.Errorf("encode authenticated session request: %w", err)
	}
	response, err := knowledgeengine.InvokeCoordinator(
		context.Background(), coordinator.Prefix, coordinator.Version, repoRoot, engineOperation, payload)
	if err != nil {
		return sessionInvocation{}, err
	}
	result, err := validateSessionResult(response.Result, engineOperation)
	if err != nil {
		return sessionInvocation{}, err
	}
	return sessionInvocation{Result: result, Raw: append(json.RawMessage(nil), response.Result...)}, nil
}

const sessionTransitionProtocol = "symphony.knowledge.session-transition-result.v1"

type sessionTransitionStep struct {
	Operation      string  `json:"operation"`
	OperationID    *string `json:"operation_id"`
	EffectiveState string  `json:"effective_state"`
	JournalDigest  *string `json:"journal_digest"`
	Changed        bool    `json:"changed"`
	Recovered      bool    `json:"recovered"`
}

type sessionTransitionReport struct {
	Protocol              string                  `json:"protocol"`
	Event                 string                  `json:"event"`
	EventID               string                  `json:"event_id"`
	Disposition           string                  `json:"disposition"`
	InitialState          string                  `json:"initial_state"`
	FinalState            string                  `json:"final_state"`
	FinalJournalDigest    *string                 `json:"final_journal_digest"`
	Steps                 []sessionTransitionStep `json:"steps"`
	Recovered             bool                    `json:"recovered"`
	CanonicalApplyEnabled bool                    `json:"canonical_apply_enabled"`
	Canonical             bool                    `json:"canonical"`
	ResultDigest          string                  `json:"result_digest"`
}

type sessionTransitionInvoker func(string, knowledgeSessionOptions) (sessionInvocation, error)

func runKnowledgeSessionTransition(options knowledgeSessionOptions) error {
	report, err := performKnowledgeSessionTransition(options, executeKnowledgeSession)
	if err != nil {
		return err
	}
	if options.jsonOutput {
		return printIndentedJSON(report)
	}
	digest := "absent"
	if report.FinalJournalDigest != nil {
		digest = *report.FinalJournalDigest
	}
	fmt.Printf(
		"Knowledge session transition: event=%s disposition=%s initial=%s final=%s steps=%d recovered=%t digest=%s canonical=false\n",
		report.Event, report.Disposition, report.InitialState, report.FinalState,
		len(report.Steps), report.Recovered, digest,
	)
	return nil
}

func performKnowledgeSessionTransition(
	options knowledgeSessionOptions,
	invoke sessionTransitionInvoker,
) (sessionTransitionReport, error) {
	if options.event != "login" && options.event != "refresh" && options.event != "logout" {
		return sessionTransitionReport{}, fmt.Errorf("--event must be login, refresh, or logout")
	}
	if !validSessionToken(options.eventID) || len(options.eventID) > 224 {
		return sessionTransitionReport{}, fmt.Errorf("--event-id must be a stable token of no more than 224 characters")
	}
	for _, reference := range options.contextRefs {
		if !validSessionToken(reference) {
			return sessionTransitionReport{}, fmt.Errorf("--context-ref contains an invalid token")
		}
	}
	report := sessionTransitionReport{
		Protocol: sessionTransitionProtocol, Event: options.event, EventID: options.eventID,
		InitialState: "unknown", FinalState: "unknown", Steps: make([]sessionTransitionStep, 0, 5),
		CanonicalApplyEnabled: false, Canonical: false,
	}
	statusOptions := transitionCallOptions(options, "", "", nil, false)
	current, err := invoke("status", statusOptions)
	if err != nil {
		if !options.recoverTransition || !recoverableSessionStatusError(err) {
			return sessionTransitionReport{}, err
		}
		recoveryID := options.eventID + ":recover"
		recoveryOptions := transitionCallOptions(options, recoveryID, "", nil, true)
		recovered, recoverErr := invoke("recover", recoveryOptions)
		if recoverErr != nil {
			return sessionTransitionReport{}, fmt.Errorf("session transition discovery recovery failed: %w", recoverErr)
		}
		appendSessionTransitionStep(&report, "recover", recoveryID, recovered)
		report.Recovered = report.Recovered || *recovered.Result.Recovered
		current, err = invoke("status", statusOptions)
		if err != nil {
			return sessionTransitionReport{}, fmt.Errorf("session status remained unavailable after recovery: %w", err)
		}
	}
	report.InitialState = current.Result.EffectiveState
	appendSessionTransitionStep(&report, "status", "", current)

	completionIDs := []string{}
	switch options.event {
	case "login":
		completionIDs = []string{options.eventID + ":begin"}
	case "refresh":
		completionIDs = []string{options.eventID + ":checkpoint", options.eventID + ":begin"}
	case "logout":
		completionIDs = []string{options.eventID + ":close"}
	}
	alreadyApplied, err := sessionContainsOperation(current, completionIDs)
	if err != nil {
		return sessionTransitionReport{}, err
	}
	if alreadyApplied {
		report.Disposition = "already_applied"
		setSessionTransitionFinal(&report, current)
		return finalizeSessionTransitionReport(report)
	}

	switch options.event {
	case "login":
		if *current.Result.JournalPresent &&
			(current.Result.EffectiveState == "open" || current.Result.EffectiveState == "expired") {
			current, err = invokeSessionTransitionMutation(
				invoke, options, "close", options.eventID+":close-prior", *current.Result.JournalDigest, nil)
			if err != nil {
				return sessionTransitionReport{}, err
			}
			appendSessionTransitionStep(&report, "close", options.eventID+":close-prior", current)
		}
		expected := "absent"
		if *current.Result.JournalPresent {
			expected = *current.Result.JournalDigest
		}
		current, err = invokeSessionTransitionMutation(
			invoke, options, "begin", options.eventID+":begin", expected, options.contextRefs)
		if err != nil {
			return sessionTransitionReport{}, err
		}
		appendSessionTransitionStep(&report, "begin", options.eventID+":begin", current)
		report.Disposition = "begun"
	case "refresh":
		if !*current.Result.JournalPresent {
			return sessionTransitionReport{}, fmt.Errorf("refresh requires an open authority epoch; submit a login transition")
		}
		if current.Result.EffectiveState == "closed" {
			closeCompleted, closeEvidenceErr := sessionContainsOperation(
				current, []string{options.eventID + ":close"})
			if closeEvidenceErr != nil {
				return sessionTransitionReport{}, closeEvidenceErr
			}
			if !closeCompleted {
				return sessionTransitionReport{}, fmt.Errorf("refresh requires an open authority epoch; submit a login transition")
			}
			beginID := options.eventID + ":begin"
			current, err = invokeSessionTransitionMutation(
				invoke, options, "begin", beginID, *current.Result.JournalDigest, options.contextRefs)
			if err != nil {
				return sessionTransitionReport{}, err
			}
			appendSessionTransitionStep(&report, "begin", beginID, current)
			report.Disposition = "reauthenticated"
			break
		}
		if current.Result.EffectiveState == "open" {
			checkpointID := options.eventID + ":checkpoint"
			checkpoint, checkpointErr := invokeSessionTransitionMutation(
				invoke, options, "checkpoint", checkpointID, *current.Result.JournalDigest, options.contextRefs)
			if checkpointErr == nil {
				current = checkpoint
				appendSessionTransitionStep(&report, "checkpoint", checkpointID, current)
				report.Disposition = "refreshed"
				break
			}
			if !sessionReauthenticationRequired(checkpointErr) {
				return sessionTransitionReport{}, checkpointErr
			}
		}
		current, err = rotateSessionTransition(&report, invoke, options, current)
		if err != nil {
			return sessionTransitionReport{}, err
		}
		report.Disposition = "reauthenticated"
	case "logout":
		if !*current.Result.JournalPresent || current.Result.EffectiveState == "closed" {
			report.Disposition = "no_change"
			break
		}
		current, err = invokeSessionTransitionMutation(
			invoke, options, "close", options.eventID+":close", *current.Result.JournalDigest, nil)
		if err != nil {
			return sessionTransitionReport{}, err
		}
		appendSessionTransitionStep(&report, "close", options.eventID+":close", current)
		report.Disposition = "closed"
	}
	setSessionTransitionFinal(&report, current)
	return finalizeSessionTransitionReport(report)
}

func transitionCallOptions(
	base knowledgeSessionOptions,
	operationID string,
	expected string,
	contexts []string,
	discover bool,
) knowledgeSessionOptions {
	base.operationID = operationID
	base.expectedJournalDigest = expected
	base.contextRefs = contexts
	base.discover = discover
	base.jsonOutput = false
	return base
}

func invokeSessionTransitionMutation(
	invoke sessionTransitionInvoker,
	base knowledgeSessionOptions,
	operation string,
	operationID string,
	expected string,
	contexts []string,
) (sessionInvocation, error) {
	result, err := invoke(operation, transitionCallOptions(base, operationID, expected, contexts, false))
	if err != nil {
		return sessionInvocation{}, fmt.Errorf("session transition %s failed: %w", operation, err)
	}
	return result, nil
}

func rotateSessionTransition(
	report *sessionTransitionReport,
	invoke sessionTransitionInvoker,
	options knowledgeSessionOptions,
	current sessionInvocation,
) (sessionInvocation, error) {
	closeID := options.eventID + ":close"
	closed, err := invokeSessionTransitionMutation(
		invoke, options, "close", closeID, *current.Result.JournalDigest, nil)
	if err != nil {
		return sessionInvocation{}, err
	}
	appendSessionTransitionStep(report, "close", closeID, closed)
	beginID := options.eventID + ":begin"
	begun, err := invokeSessionTransitionMutation(
		invoke, options, "begin", beginID, *closed.Result.JournalDigest, options.contextRefs)
	if err != nil {
		return sessionInvocation{}, err
	}
	appendSessionTransitionStep(report, "begin", beginID, begun)
	return begun, nil
}

func appendSessionTransitionStep(
	report *sessionTransitionReport,
	operation string,
	operationID string,
	invocation sessionInvocation,
) {
	var identity *string
	if operationID != "" {
		value := operationID
		identity = &value
	}
	var digest *string
	if invocation.Result.JournalDigest != nil {
		value := *invocation.Result.JournalDigest
		digest = &value
	}
	report.Steps = append(report.Steps, sessionTransitionStep{
		Operation: operation, OperationID: identity,
		EffectiveState: invocation.Result.EffectiveState, JournalDigest: digest,
		Changed: *invocation.Result.Changed, Recovered: *invocation.Result.Recovered,
	})
}

func setSessionTransitionFinal(report *sessionTransitionReport, invocation sessionInvocation) {
	report.FinalState = invocation.Result.EffectiveState
	if invocation.Result.JournalDigest == nil {
		report.FinalJournalDigest = nil
		return
	}
	value := *invocation.Result.JournalDigest
	report.FinalJournalDigest = &value
}

func sessionContainsOperation(invocation sessionInvocation, candidates []string) (bool, error) {
	if !*invocation.Result.JournalPresent {
		return false, nil
	}
	var journal struct {
		Checkpoints []struct {
			OperationID string `json:"operation_id"`
		} `json:"checkpoints"`
	}
	if err := json.Unmarshal(invocation.Result.Journal, &journal); err != nil || journal.Checkpoints == nil {
		return false, fmt.Errorf("authenticated session journal lacks checkpoint evidence")
	}
	wanted := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		wanted[candidate] = struct{}{}
	}
	for _, checkpoint := range journal.Checkpoints {
		if _, ok := wanted[checkpoint.OperationID]; ok {
			return true, nil
		}
	}
	return false, nil
}

func recoverableSessionStatusError(err error) bool {
	var processError *knowledgeengine.ProcessError
	if !errors.As(err, &processError) {
		return false
	}
	switch processError.Code {
	case "session.head_invalid", "session.head_missing", "session.head_slot_mismatch",
		"session.journal_invalid", "session.recovery_required", "session.state_json_invalid":
		return true
	default:
		return false
	}
}

func sessionReauthenticationRequired(err error) bool {
	var processError *knowledgeengine.ProcessError
	return errors.As(err, &processError) && processError.Code == "session.reauthentication_required"
}

func finalizeSessionTransitionReport(report sessionTransitionReport) (sessionTransitionReport, error) {
	input := map[string]any{
		"protocol": report.Protocol, "event": report.Event, "event_id": report.EventID,
		"disposition": report.Disposition, "initial_state": report.InitialState,
		"final_state": report.FinalState, "final_journal_digest": report.FinalJournalDigest,
		"steps": report.Steps, "recovered": report.Recovered,
		"canonical_apply_enabled": report.CanonicalApplyEnabled, "canonical": report.Canonical,
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		return sessionTransitionReport{}, fmt.Errorf("encode session transition digest input: %w", err)
	}
	digest := sha256.Sum256(encoded)
	report.ResultDigest = "sha256:" + hex.EncodeToString(digest[:])
	return report, nil
}

func sessionRepositoryResource(repositoryRoot string) string {
	digest := sha256.Sum256([]byte("repository-root:" + repositoryRoot))
	return "symphony.knowledge.repository:" + hex.EncodeToString(digest[:])
}

func resolveKnowledgeRepository(start string) (string, error) {
	if start == "" {
		var err error
		start, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("could not get current working directory: %w", err)
		}
	}
	absolute, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve repository path: %w", err)
	}
	info, err := os.Lstat(absolute)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", fmt.Errorf("--repo must identify a no-follow directory")
	}
	root, err := repository.FindRoot(absolute)
	if err != nil {
		return "", fmt.Errorf("could not find Symphony repository root: %w", err)
	}
	return root, nil
}

func randomUUID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate request identity: %w", err)
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		value[0:4], value[4:6], value[6:8], value[8:10], value[10:16]), nil
}

func validateSessionAuthorization(decision ssiagclient.AuthorizationDecision, request ssiagclient.AuthorizationRequest, topsID string) error {
	if decision.Schema != "symphony.ssiag.authorization-decision.v1" || decision.Effect != "allow" ||
		decision.ReasonCode != "symphony.ssiag.policy.exact-grant" || !validSessionToken(decision.DecisionID) ||
		decision.RequestID != request.RequestID || decision.CorrelationID != request.CorrelationID ||
		decision.TOPSID != topsID || decision.Target.Operation != request.Operation ||
		decision.Target.Resource != request.Resource || decision.Target.Audience != request.Audience ||
		decision.Target.Scope != request.Scope || decision.CallerClassUsed || decision.CanonicalApply ||
		decision.AuthorityBasis == nil || decision.Capability == nil || decision.ExpiresAt == nil {
		return fmt.Errorf("SSIAG authorization decision does not bind the requested session authority")
	}
	capability := decision.Capability
	basis := *decision.AuthorityBasis
	binding := capability.BindingDigest
	current := time.Now().UTC()
	if capability.Protocol != "symphony.ssiag.capability.v1" || capability.TOPSID != topsID ||
		!validSessionToken(capability.Subject.ID) || !validSessionToken(capability.Subject.Kind) ||
		(capability.Subject.Authority != "unix_peer_credentials") ||
		(basis != "host_owner" && basis != "granted_permission") ||
		!validSessionToken(capability.CapabilityID) || !validSessionToken(capability.GrantID) ||
		capability.Subject != decision.Subject || capability.Target != decision.Target ||
		capability.RequestID != decision.RequestID || capability.CorrelationID != decision.CorrelationID ||
		capability.AuthorityBasis != basis || capability.PolicyDigest != decision.PolicyDigest ||
		capability.ConfigDigest != decision.ConfigDigest || capability.Transferable || capability.CanonicalApply ||
		!validTaggedDigest(capability.PolicyDigest) || !validTaggedDigest(capability.ConfigDigest) ||
		!validTaggedDigest(binding) || sessionCapabilityBinding(*capability) != binding ||
		capability.CapabilityID != "ssiag-capability:"+strings.TrimPrefix(binding, "sha256:") ||
		!capability.IssuedAt.Equal(decision.DecidedAt) || !capability.ExpiresAt.Equal(*decision.ExpiresAt) ||
		capability.IssuedAt.Location() != time.UTC || capability.ExpiresAt.Location() != time.UTC ||
		decision.DecidedAt.Location() != time.UTC || decision.ExpiresAt.Location() != time.UTC ||
		capability.IssuedAt.Before(request.RequestedAt.Add(-30*time.Second)) ||
		capability.IssuedAt.After(current.Add(30*time.Second)) ||
		!capability.ExpiresAt.After(current) || capability.ExpiresAt.After(request.RequestedExpiresAt) ||
		!capability.ExpiresAt.After(capability.IssuedAt) {
		return fmt.Errorf("SSIAG capability evidence is inconsistent or expired")
	}
	return nil
}

func sessionCapabilityBinding(capability ssiagclient.Capability) string {
	joined := strings.Join([]string{
		capability.Protocol, capability.Subject.ID, capability.Subject.Kind, capability.Subject.Authority,
		capability.TOPSID, capability.Target.Operation, capability.Target.Resource,
		capability.Target.Audience, capability.Target.Scope, capability.AuthorityBasis,
		capability.GrantID, capability.RequestID, capability.CorrelationID,
		capability.IssuedAt.UTC().Format(time.RFC3339), capability.ExpiresAt.UTC().Format(time.RFC3339),
		capability.PolicyDigest, capability.ConfigDigest, "transferable=false", "canonical_apply=false",
	}, "\n")
	digest := sha256.Sum256([]byte(joined))
	return "sha256:" + hex.EncodeToString(digest[:])
}

func validSessionToken(value string) bool {
	if value == "" || len(value) > 256 || strings.TrimSpace(value) != value {
		return false
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') || strings.ContainsRune("._:-", character) {
			continue
		}
		return false
	}
	return true
}

type sessionResult struct {
	Protocol       string          `json:"protocol"`
	Operation      string          `json:"operation"`
	Compatibility  json.RawMessage `json:"compatibility"`
	JournalPresent *bool           `json:"journal_present"`
	Journal        json.RawMessage `json:"journal"`
	JournalDigest  *string         `json:"journal_digest"`
	EffectiveState string          `json:"effective_state"`
	Changed        *bool           `json:"changed"`
	Recovered      *bool           `json:"recovered"`
	RepairActions  []string        `json:"repair_actions"`
	ReadOnly       *bool           `json:"read_only"`
	ApplyEnabled   *bool           `json:"canonical_apply_enabled"`
	Canonical      *bool           `json:"canonical"`
}

func validateSessionResult(raw json.RawMessage, operation string) (sessionResult, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return sessionResult{}, fmt.Errorf("decode authenticated session result: %w", err)
	}
	required := []string{
		"protocol", "operation", "compatibility", "journal_present", "journal", "journal_digest",
		"effective_state", "changed", "recovered", "repair_actions", "read_only",
		"canonical_apply_enabled", "canonical",
	}
	if len(fields) != len(required) {
		return sessionResult{}, fmt.Errorf("authenticated session result has an invalid field set")
	}
	for _, field := range required {
		if _, ok := fields[field]; !ok {
			return sessionResult{}, fmt.Errorf("authenticated session result is incomplete")
		}
	}
	var result sessionResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return sessionResult{}, fmt.Errorf("decode authenticated session result: %w", err)
	}
	if result.Protocol != "symphony.knowledge.session-result.v1" || result.Operation != operation ||
		result.JournalPresent == nil || result.Changed == nil || result.Recovered == nil || result.ReadOnly == nil ||
		!explicitFalse(result.ApplyEnabled) || !explicitFalse(result.Canonical) || result.RepairActions == nil ||
		(result.EffectiveState != "absent" && result.EffectiveState != "open" &&
			result.EffectiveState != "closed" && result.EffectiveState != "expired") {
		return sessionResult{}, fmt.Errorf("authenticated session result violates the v1 identity contract")
	}
	journalNull := bytes.Equal(bytes.TrimSpace(result.Journal), []byte("null"))
	if !*result.JournalPresent {
		if !journalNull || result.JournalDigest != nil || result.EffectiveState != "absent" {
			return sessionResult{}, fmt.Errorf("absent authenticated session result is inconsistent")
		}
	} else {
		if journalNull || result.JournalDigest == nil || !validTaggedDigest(*result.JournalDigest) || result.EffectiveState == "absent" {
			return sessionResult{}, fmt.Errorf("present authenticated session result is incomplete")
		}
		var journal struct {
			Protocol      string `json:"protocol"`
			JournalDigest string `json:"journal_digest"`
			Canonical     *bool  `json:"canonical"`
		}
		if err := json.Unmarshal(result.Journal, &journal); err != nil ||
			journal.Protocol != "symphony.knowledge.session-journal.v1" ||
			journal.JournalDigest != *result.JournalDigest || !explicitFalse(journal.Canonical) {
			return sessionResult{}, fmt.Errorf("authenticated session journal identity is invalid")
		}
	}
	if *result.ReadOnly != (operation == "session_status") ||
		(operation != "session_recover" && *result.Recovered) ||
		(operation == "session_recover" && *result.Recovered != *result.Changed) {
		return sessionResult{}, fmt.Errorf("authenticated session mutation flags are inconsistent")
	}
	return result, nil
}

type reconciliationResult struct {
	Protocol       string          `json:"protocol"`
	Operation      string          `json:"operation"`
	Compatibility  json.RawMessage `json:"compatibility"`
	JournalPresent *bool           `json:"journal_present"`
	Journal        json.RawMessage `json:"journal"`
	JournalDigest  *string         `json:"journal_digest"`
	Changed        *bool           `json:"changed"`
	Recovered      *bool           `json:"recovered"`
	RepairActions  []string        `json:"repair_actions"`
	ReadOnly       *bool           `json:"read_only"`
	ApplyEnabled   *bool           `json:"canonical_apply_enabled"`
	Canonical      *bool           `json:"canonical"`
	Mode           string          `json:"-"`
}

func validateReconciliationResult(raw json.RawMessage, operation string) (reconciliationResult, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return reconciliationResult{}, fmt.Errorf("decode reconciliation result: %w", err)
	}
	required := map[string]struct{}{
		"protocol": {}, "operation": {}, "compatibility": {}, "journal_present": {},
		"journal": {}, "journal_digest": {}, "changed": {}, "recovered": {},
		"repair_actions": {}, "read_only": {}, "canonical_apply_enabled": {}, "canonical": {},
	}
	if len(fields) != len(required) {
		return reconciliationResult{}, fmt.Errorf("reconciliation result has an invalid field set")
	}
	for field := range fields {
		if _, ok := required[field]; !ok {
			return reconciliationResult{}, fmt.Errorf("reconciliation result contains unknown field %q", field)
		}
	}
	var result reconciliationResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return reconciliationResult{}, fmt.Errorf("decode reconciliation result: %w", err)
	}
	if result.Protocol != "symphony.knowledge.reconciliation-result.v1" ||
		result.Operation != operation || result.JournalPresent == nil ||
		result.Changed == nil || result.Recovered == nil || result.ReadOnly == nil ||
		!explicitFalse(result.ApplyEnabled) || !explicitFalse(result.Canonical) ||
		result.RepairActions == nil || len(result.RepairActions) > 32 {
		return reconciliationResult{}, fmt.Errorf("reconciliation result violates the v1 identity contract")
	}
	for _, action := range result.RepairActions {
		if action == "" || len(action) > 4096 || strings.IndexFunc(action, func(character rune) bool {
			return character < 0x20 || character == 0x7f
		}) >= 0 {
			return reconciliationResult{}, fmt.Errorf("reconciliation result contains unsafe repair guidance")
		}
	}
	var compatibilityFields map[string]json.RawMessage
	if err := json.Unmarshal(result.Compatibility, &compatibilityFields); err != nil ||
		len(compatibilityFields) != 7 {
		return reconciliationResult{}, fmt.Errorf("reconciliation compatibility result is invalid")
	}
	for _, field := range []string{
		"mode", "process_protocol", "journal_read_version", "journal_write_version",
		"shared_capabilities", "missing_capabilities", "reasons",
	} {
		if _, ok := compatibilityFields[field]; !ok {
			return reconciliationResult{}, fmt.Errorf("reconciliation compatibility result is incomplete")
		}
	}
	var compatibility struct {
		Mode                string   `json:"mode"`
		ProcessProtocol     *string  `json:"process_protocol"`
		JournalReadVersion  *uint64  `json:"journal_read_version"`
		JournalWriteVersion *uint64  `json:"journal_write_version"`
		SharedCapabilities  []string `json:"shared_capabilities"`
		MissingCapabilities []string `json:"missing_capabilities"`
		Reasons             []string `json:"reasons"`
	}
	if err := json.Unmarshal(result.Compatibility, &compatibility); err != nil ||
		(compatibility.Mode != "full" && compatibility.Mode != "read_only" &&
			compatibility.Mode != "migration_required" && compatibility.Mode != "unsupported") ||
		compatibility.SharedCapabilities == nil ||
		compatibility.MissingCapabilities == nil ||
		compatibility.Reasons == nil || len(compatibility.Reasons) == 0 ||
		len(compatibility.Reasons) > 32 {
		return reconciliationResult{}, fmt.Errorf("reconciliation compatibility mode is invalid")
	}
	for _, reason := range compatibility.Reasons {
		if reason == "" || len(reason) > 4096 || strings.IndexFunc(reason, func(character rune) bool {
			return character < 0x20 || character == 0x7f
		}) >= 0 {
			return reconciliationResult{}, fmt.Errorf("reconciliation compatibility reason is unsafe")
		}
	}
	if compatibility.Mode == "full" &&
		(compatibility.ProcessProtocol == nil ||
			*compatibility.ProcessProtocol != "symphony.knowledge.engine-process.v1" ||
			compatibility.JournalReadVersion == nil ||
			*compatibility.JournalReadVersion != 1 ||
			compatibility.JournalWriteVersion == nil ||
			*compatibility.JournalWriteVersion != 1 ||
			len(compatibility.MissingCapabilities) != 0) {
		return reconciliationResult{}, fmt.Errorf("full reconciliation compatibility evidence is inconsistent")
	}
	result.Mode = compatibility.Mode
	journalIsNull := bytes.Equal(bytes.TrimSpace(result.Journal), []byte("null"))
	if !*result.JournalPresent {
		if !journalIsNull || result.JournalDigest != nil {
			return reconciliationResult{}, fmt.Errorf("absent reconciliation journal result is inconsistent")
		}
	} else {
		if journalIsNull || result.JournalDigest == nil || !validTaggedDigest(*result.JournalDigest) {
			return reconciliationResult{}, fmt.Errorf("present reconciliation journal result is incomplete")
		}
		var journal struct {
			Protocol      string `json:"protocol"`
			JournalDigest string `json:"journal_digest"`
			Canonical     *bool  `json:"canonical"`
		}
		if err := json.Unmarshal(result.Journal, &journal); err != nil ||
			journal.Protocol != "symphony.knowledge.reconciliation-journal.v1" ||
			journal.JournalDigest != *result.JournalDigest ||
			!explicitFalse(journal.Canonical) {
			return reconciliationResult{}, fmt.Errorf("reconciliation journal identity is invalid")
		}
	}
	isReadOperation := operation == "compatibility" || operation == "status"
	if *result.ReadOnly != isReadOperation ||
		(!isReadOperation && compatibility.Mode != "full") ||
		(operation != "recover" && *result.Recovered) ||
		(operation == "recover" && *result.Recovered != *result.Changed) {
		return reconciliationResult{}, fmt.Errorf("reconciliation result mutation flags are inconsistent")
	}
	return result, nil
}

func printIndentedJSON(value any) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("format JSON output: %w", err)
	}
	fmt.Println(string(encoded))
	return nil
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
				CallerDeclaredOperation  *bool `json:"caller_declared_operation"`
				EngineDecidedDomainTruth *bool `json:"engine_decided_domain_truth"`
				Ratified                 *bool `json:"ratified"`
			} `json:"authority"`
			Operations []json.RawMessage `json:"operations"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.Protocol != "symphony.knowledge.proposal.v1" ||
			value.ModuleID != "skvi-engine" || value.EngineID != "symphony-skvi" || value.VectorID != "skvi" ||
			value.ProposalID == "" || !validTaggedDigest(value.ProposalDigest) || len(value.Operations) != 1 ||
			!explicitTrue(value.Authority.CallerDeclaredOperation) ||
			!explicitFalse(value.Authority.EngineDecidedDomainTruth) ||
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

func runSACV(operation string, options sacvOptions) error {
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
		if options.expectedRegistryDigest != "" {
			expected = options.expectedRegistryDigest
		}
		payload, err = json.Marshal(map[string]any{"expected_registry_digest": expected})
	case "diff", "propose":
		payload, err = knowledgeengine.ReadPayload(options.input)
	case "project":
		payload = []byte(`{"format":"json"}`)
	default:
		return fmt.Errorf("unsupported SACV operation")
	}
	if err != nil {
		return err
	}
	response, err := knowledgeengine.InvokeSACV(
		context.Background(), options.prefix, options.version, repoRoot, operation, payload)
	if err != nil {
		return err
	}
	checkValid, err := validateSACVResult(operation, response.Result)
	if err != nil {
		return err
	}
	if options.jsonOutput {
		var output bytes.Buffer
		if err := json.Indent(&output, response.Result, "", "  "); err != nil {
			return fmt.Errorf("format SACV result: %w", err)
		}
		fmt.Println(output.String())
		if !checkValid {
			return fmt.Errorf("SACV registry check reported violations")
		}
		return nil
	}
	return printSACVResult(operation, response.Result)
}

func validateSACVResult(operation string, result json.RawMessage) (bool, error) {
	switch operation {
	case "inspect":
		var value struct {
			Readiness              string `json:"readiness"`
			EmptyRegistryValid     *bool  `json:"empty_registry_valid"`
			EngineDecidesOwnership *bool  `json:"engine_decides_ownership"`
			CanonicalApplyEnabled  *bool  `json:"canonical_apply_enabled"`
			ParserFormats          struct {
				JSON string `json:"json"`
				YAML string `json:"yaml"`
			} `json:"parser_formats"`
			Descriptor struct {
				EngineID               string `json:"engine_id"`
				OpenAPITarget          string `json:"openapi_target"`
				CanonicalApplyEnabled  *bool  `json:"canonical_apply_enabled"`
				SessionMutationEnabled *bool  `json:"session_mutation_enabled"`
				NetworkListener        *bool  `json:"network_listener"`
			} `json:"descriptor"`
		}
		if err := json.Unmarshal(result, &value); err != nil ||
			value.Readiness != "read_check_diff_propose_project" ||
			value.ParserFormats.JSON != "implemented" || value.ParserFormats.YAML != "fail_closed_unavailable" ||
			value.Descriptor.EngineID != "symphony-sacv" || value.Descriptor.OpenAPITarget != "3.2.0" ||
			!explicitTrue(value.EmptyRegistryValid) || !explicitFalse(value.EngineDecidesOwnership) ||
			!explicitFalse(value.CanonicalApplyEnabled) ||
			!explicitFalse(value.Descriptor.CanonicalApplyEnabled) ||
			!explicitFalse(value.Descriptor.SessionMutationEnabled) ||
			!explicitFalse(value.Descriptor.NetworkListener) {
			return false, fmt.Errorf("SACV inspect result violates the implemented safety contract")
		}
		return true, nil
	case "check":
		var value struct {
			Protocol              string `json:"protocol"`
			ReadOnly              *bool  `json:"read_only"`
			CanonicalApplyEnabled *bool  `json:"canonical_apply_enabled"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.Protocol != "symphony.sacv.check-result.v1" ||
			!explicitTrue(value.ReadOnly) || !explicitFalse(value.CanonicalApplyEnabled) {
			return false, fmt.Errorf("SACV check result violates the implemented safety contract")
		}
		return sacvCheckValid(result)
	case "diff":
		var value struct {
			Protocol     string            `json:"protocol"`
			State        string            `json:"state"`
			Changes      []json.RawMessage `json:"changes"`
			ReadOnly     *bool             `json:"read_only"`
			Noncanonical *bool             `json:"noncanonical"`
			ResultDigest string            `json:"result_digest"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.Protocol != "symphony.sacv.diff-result.v1" ||
			(value.State != "identical" && value.State != "compatible_additive" &&
				value.State != "breaking" && value.State != "review_required") ||
			value.Changes == nil || !explicitTrue(value.ReadOnly) || !explicitTrue(value.Noncanonical) ||
			!validTaggedDigest(value.ResultDigest) {
			return false, fmt.Errorf("SACV diff result violates the implemented safety contract")
		}
		return true, nil
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
				CallerDeclaredOperation  *bool `json:"caller_declared_operation"`
				EngineDecidedDomainTruth *bool `json:"engine_decided_domain_truth"`
				Ratified                 *bool `json:"ratified"`
			} `json:"authority"`
			WriteSet []struct {
				TargetPath string `json:"target_path"`
			} `json:"write_set"`
			Operations []json.RawMessage `json:"operations"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.Protocol != "symphony.knowledge.proposal.v1" ||
			value.ModuleID != "sacv-engine" || value.EngineID != "symphony-sacv" || value.VectorID != "sacv" ||
			value.ProposalID == "" || !validTaggedDigest(value.ProposalDigest) || len(value.Operations) != 1 ||
			len(value.WriteSet) != 1 || value.WriteSet[0].TargetPath != "knowledge/sacv/REGISTRY.md" ||
			!explicitTrue(value.Authority.CallerDeclaredOperation) ||
			!explicitFalse(value.Authority.EngineDecidedDomainTruth) ||
			!explicitFalse(value.Authority.Ratified) || !explicitFalse(value.CanonicalApplyEnabled) {
			return false, fmt.Errorf("SACV proposal result violates the implemented safety contract")
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
		if err := json.Unmarshal(result, &value); err != nil || value.Protocol != "symphony.sacv.projection.v1" ||
			value.ModuleID != "sacv-engine" || value.EngineID != "symphony-sacv" || value.VectorID != "sacv" ||
			value.EntryCount == nil || value.Entries == nil || *value.EntryCount != uint64(len(value.Entries)) ||
			!validTaggedDigest(value.ProjectionDigest) ||
			!explicitTrue(value.Noncanonical) || !explicitTrue(value.Rebuildable) {
			return false, fmt.Errorf("SACV projection result violates the implemented safety contract")
		}
		return true, nil
	default:
		return false, fmt.Errorf("unsupported SACV operation")
	}
}

func sacvCheckValid(result json.RawMessage) (bool, error) {
	var value struct {
		Summary struct {
			Violation uint64 `json:"violation"`
			State     string `json:"state"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(result, &value); err != nil || value.Summary.State == "" {
		return false, fmt.Errorf("SACV check result is incomplete")
	}
	return value.Summary.State == "valid" && value.Summary.Violation == 0, nil
}

func printSACVResult(operation string, result json.RawMessage) error {
	switch operation {
	case "inspect":
		var value struct {
			Readiness     string `json:"readiness"`
			ParserFormats struct {
				JSON string `json:"json"`
				YAML string `json:"yaml"`
			} `json:"parser_formats"`
			Descriptor struct {
				EngineID      string `json:"engine_id"`
				EngineVersion string `json:"engine_version"`
				OpenAPITarget string `json:"openapi_target"`
			} `json:"descriptor"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.Descriptor.EngineID == "" || value.Readiness == "" {
			return fmt.Errorf("SACV inspect result is incomplete")
		}
		fmt.Printf("SACV: engine=%s version=%s readiness=%s openapi=%s json=%s yaml=%s\n",
			value.Descriptor.EngineID, value.Descriptor.EngineVersion, value.Readiness,
			value.Descriptor.OpenAPITarget, value.ParserFormats.JSON, value.ParserFormats.YAML)
		return nil
	case "check":
		var value struct {
			EntriesChecked    uint64 `json:"entries_checked"`
			DocumentsChecked  uint64 `json:"documents_checked"`
			OperationsChecked uint64 `json:"operations_checked"`
			Registry          struct {
				Digest string `json:"digest"`
			} `json:"registry"`
			Summary struct {
				Pass      uint64 `json:"pass"`
				Warning   uint64 `json:"warning"`
				Violation uint64 `json:"violation"`
				State     string `json:"state"`
			} `json:"summary"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.Summary.State == "" || value.Registry.Digest == "" {
			return fmt.Errorf("SACV check result is incomplete")
		}
		fmt.Printf("SACV check: state=%s entries=%d documents=%d operations=%d pass=%d warning=%d violation=%d registry_digest=%s\n",
			value.Summary.State, value.EntriesChecked, value.DocumentsChecked, value.OperationsChecked,
			value.Summary.Pass, value.Summary.Warning, value.Summary.Violation, value.Registry.Digest)
		if value.Summary.State != "valid" || value.Summary.Violation != 0 {
			return fmt.Errorf("SACV registry check reported violations")
		}
		return nil
	case "diff":
		var value struct {
			State        string `json:"state"`
			ResultDigest string `json:"result_digest"`
			Summary      struct {
				Additive uint64 `json:"additive"`
				Breaking uint64 `json:"breaking"`
				Review   uint64 `json:"review_required"`
			} `json:"summary"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.State == "" || value.ResultDigest == "" {
			return fmt.Errorf("SACV diff result is incomplete")
		}
		fmt.Printf("SACV diff: state=%s additive=%d breaking=%d review_required=%d digest=%s noncanonical=true\n",
			value.State, value.Summary.Additive, value.Summary.Breaking, value.Summary.Review, value.ResultDigest)
		return nil
	case "propose":
		var value struct {
			ProposalID     string `json:"proposal_id"`
			ProposalDigest string `json:"proposal_digest"`
			Authority      struct {
				Ratified bool `json:"ratified"`
			} `json:"authority"`
			Operations []struct {
				Type       string `json:"type"`
				TargetPath string `json:"target_path"`
			} `json:"operations"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.ProposalID == "" || len(value.Operations) != 1 {
			return fmt.Errorf("SACV proposal result is incomplete")
		}
		fmt.Printf("SACV proposal: id=%s digest=%s operation=%s target=%s ratified=%t apply=false\n",
			value.ProposalID, value.ProposalDigest, value.Operations[0].Type,
			value.Operations[0].TargetPath, value.Authority.Ratified)
		return nil
	case "project":
		var value struct {
			EntryCount       uint64 `json:"entry_count"`
			ProjectionDigest string `json:"projection_digest"`
			Noncanonical     bool   `json:"noncanonical"`
			Rebuildable      bool   `json:"rebuildable"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.ProjectionDigest == "" {
			return fmt.Errorf("SACV projection result is incomplete")
		}
		fmt.Printf("SACV projection: entries=%d digest=%s noncanonical=%t rebuildable=%t\n",
			value.EntryCount, value.ProjectionDigest, value.Noncanonical, value.Rebuildable)
		return nil
	default:
		return fmt.Errorf("unsupported SACV result")
	}
}

func runSODV(operation string, options sodvOptions) error {
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
	case "verify", "propose", "recover":
		payload, err = knowledgeengine.ReadPayload(options.input)
	case "project":
		payload = []byte(`{"format":"json"}`)
	default:
		return fmt.Errorf("unsupported SODV operation")
	}
	if err != nil {
		return err
	}
	response, err := knowledgeengine.InvokeSODV(
		context.Background(), options.prefix, options.version, repoRoot, operation, payload)
	if err != nil {
		return err
	}
	checkValid, err := validateSODVResult(operation, response.Result)
	if err != nil {
		return err
	}
	if options.jsonOutput {
		var output bytes.Buffer
		if err := json.Indent(&output, response.Result, "", "  "); err != nil {
			return fmt.Errorf("format SODV result: %w", err)
		}
		fmt.Println(output.String())
		if !checkValid {
			return fmt.Errorf("SODV release-ledger check reported violations")
		}
		return nil
	}
	return printSODVResult(operation, response.Result)
}

func validateSODVResult(operation string, result json.RawMessage) (bool, error) {
	switch operation {
	case "inspect":
		var value struct {
			EngineDecidesReleaseTruth          *bool `json:"engine_decides_release_truth"`
			CallerSuppliesExternalObservations *bool `json:"caller_supplies_external_observations"`
			NetworkAccess                      *bool `json:"network_access"`
			CanonicalApplyEnabled              *bool `json:"canonical_apply_enabled"`
			Descriptor                         struct {
				EngineID              string `json:"engine_id"`
				Language              string `json:"language"`
				ThermalPath           string `json:"thermal_path"`
				ProviderInput         string `json:"provider_input"`
				NetworkAccess         *bool  `json:"network_access"`
				CanonicalApplyEnabled *bool  `json:"canonical_apply_enabled"`
				NetworkListener       *bool  `json:"network_listener"`
			} `json:"descriptor"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.Descriptor.EngineID != "symphony-sodv" ||
			value.Descriptor.Language != "C++26" || value.Descriptor.ThermalPath != "freezing" ||
			value.Descriptor.ProviderInput != "caller_supplied" ||
			!explicitFalse(value.EngineDecidesReleaseTruth) ||
			!explicitTrue(value.CallerSuppliesExternalObservations) || !explicitFalse(value.NetworkAccess) ||
			!explicitFalse(value.CanonicalApplyEnabled) || !explicitFalse(value.Descriptor.NetworkAccess) ||
			!explicitFalse(value.Descriptor.CanonicalApplyEnabled) || !explicitFalse(value.Descriptor.NetworkListener) {
			return false, fmt.Errorf("SODV inspect result violates the implemented safety contract")
		}
		return true, nil
	case "check":
		var value struct {
			Protocol              string `json:"protocol"`
			ReadOnly              *bool  `json:"read_only"`
			CanonicalApplyEnabled *bool  `json:"canonical_apply_enabled"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.Protocol != "symphony.sodv.check-result.v1" ||
			!explicitTrue(value.ReadOnly) || !explicitFalse(value.CanonicalApplyEnabled) {
			return false, fmt.Errorf("SODV check result violates the implemented safety contract")
		}
		return sodvCheckValid(result)
	case "verify":
		var value struct {
			Protocol                 string `json:"protocol"`
			VerificationState        string `json:"verification_state"`
			ReadOnly                 *bool  `json:"read_only"`
			Noncanonical             *bool  `json:"noncanonical"`
			EngineDeclaresCompletion *bool  `json:"engine_declares_completion"`
			ResultDigest             string `json:"result_digest"`
		}
		states := map[string]bool{"authorized_unpublished": true, "published_waiting_evidence": true,
			"completion_candidate": true, "verified_completed": true, "blocked_mismatch": true}
		if err := json.Unmarshal(result, &value); err != nil || value.Protocol != "symphony.sodv.verify-result.v1" ||
			!states[value.VerificationState] || !explicitTrue(value.ReadOnly) || !explicitTrue(value.Noncanonical) ||
			!explicitFalse(value.EngineDeclaresCompletion) || !validTaggedDigest(value.ResultDigest) {
			return false, fmt.Errorf("SODV verify result violates the implemented safety contract")
		}
		return true, nil
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
				CallerDeclaredOperation  *bool `json:"caller_declared_operation"`
				EngineDecidedDomainTruth *bool `json:"engine_decided_domain_truth"`
				Ratified                 *bool `json:"ratified"`
			} `json:"authority"`
			WriteSet []struct {
				TargetPath string `json:"target_path"`
			} `json:"write_set"`
			Operations []json.RawMessage `json:"operations"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.Protocol != "symphony.knowledge.proposal.v1" ||
			value.ModuleID != "sodv-engine" || value.EngineID != "symphony-sodv" || value.VectorID != "sodv" ||
			value.ProposalID == "" || !validTaggedDigest(value.ProposalDigest) || len(value.Operations) != 1 ||
			len(value.WriteSet) != 1 || value.WriteSet[0].TargetPath != "knowledge/sodv/RELEASES.md" ||
			!explicitTrue(value.Authority.CallerDeclaredOperation) ||
			!explicitFalse(value.Authority.EngineDecidedDomainTruth) || !explicitFalse(value.Authority.Ratified) ||
			!explicitFalse(value.CanonicalApplyEnabled) {
			return false, fmt.Errorf("SODV proposal result violates the implemented safety contract")
		}
		return true, nil
	case "recover":
		var value struct {
			Protocol              string          `json:"protocol"`
			Action                string          `json:"action"`
			Verification          json.RawMessage `json:"verification"`
			JournalMutated        *bool           `json:"journal_mutated"`
			CanonicalApplyEnabled *bool           `json:"canonical_apply_enabled"`
			ResultDigest          string          `json:"result_digest"`
		}
		actions := map[string]bool{"resume_authorized_publication": true, "await_public_evidence": true,
			"completion_proposal_required": true, "propose_forward_completion": true,
			"no_op_completed": true, "fail_closed_review": true}
		if err := json.Unmarshal(result, &value); err != nil || value.Protocol != "symphony.sodv.recovery-result.v1" ||
			!actions[value.Action] || !explicitFalse(value.JournalMutated) || !explicitFalse(value.CanonicalApplyEnabled) ||
			!validTaggedDigest(value.ResultDigest) {
			return false, fmt.Errorf("SODV recovery result violates the implemented safety contract")
		}
		if _, err := validateSODVResult("verify", value.Verification); err != nil {
			return false, fmt.Errorf("SODV recovery verification is unsafe: %w", err)
		}
		return true, nil
	case "project":
		var value struct {
			Protocol         string            `json:"protocol"`
			ModuleID         string            `json:"module_id"`
			EngineID         string            `json:"engine_id"`
			VectorID         string            `json:"vector_id"`
			RecordCount      *uint64           `json:"record_count"`
			TransactionCount *uint64           `json:"transaction_count"`
			Records          []json.RawMessage `json:"records"`
			Transactions     []json.RawMessage `json:"transactions"`
			ProjectionDigest string            `json:"projection_digest"`
			Noncanonical     *bool             `json:"noncanonical"`
			Rebuildable      *bool             `json:"rebuildable"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.Protocol != "symphony.sodv.projection.v1" ||
			value.ModuleID != "sodv-engine" || value.EngineID != "symphony-sodv" || value.VectorID != "sodv" ||
			value.RecordCount == nil || value.TransactionCount == nil || value.Records == nil || value.Transactions == nil ||
			*value.RecordCount != uint64(len(value.Records)) || *value.TransactionCount != uint64(len(value.Transactions)) ||
			!validTaggedDigest(value.ProjectionDigest) || !explicitTrue(value.Noncanonical) || !explicitTrue(value.Rebuildable) {
			return false, fmt.Errorf("SODV projection result violates the implemented safety contract")
		}
		return true, nil
	default:
		return false, fmt.Errorf("unsupported SODV operation")
	}
}

func sodvCheckValid(result json.RawMessage) (bool, error) {
	var value struct {
		Summary struct {
			Violations uint64 `json:"violations"`
			State      string `json:"state"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(result, &value); err != nil || value.Summary.State == "" {
		return false, fmt.Errorf("SODV check result is incomplete")
	}
	return value.Summary.State == "valid" && value.Summary.Violations == 0, nil
}

func printSODVResult(operation string, result json.RawMessage) error {
	switch operation {
	case "inspect":
		var value struct {
			CanonicalLedger string `json:"canonical_ledger"`
			Descriptor      struct {
				EngineID      string `json:"engine_id"`
				EngineVersion string `json:"engine_version"`
				ThermalPath   string `json:"thermal_path"`
			} `json:"descriptor"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.Descriptor.EngineID == "" || value.CanonicalLedger == "" {
			return fmt.Errorf("SODV inspect result is incomplete")
		}
		fmt.Printf("SODV: engine=%s version=%s thermal=%s ledger=%s provider_input=caller_supplied apply=false\n",
			value.Descriptor.EngineID, value.Descriptor.EngineVersion, value.Descriptor.ThermalPath, value.CanonicalLedger)
		return nil
	case "check":
		var value struct {
			RecordsChecked      uint64 `json:"records_checked"`
			TransactionsChecked uint64 `json:"transactions_checked"`
			Ledger              struct {
				Digest string `json:"digest"`
			} `json:"ledger"`
			Summary struct {
				Pass       uint64 `json:"passes"`
				Warnings   uint64 `json:"warnings"`
				Violations uint64 `json:"violations"`
				State      string `json:"state"`
			} `json:"summary"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.Summary.State == "" || value.Ledger.Digest == "" {
			return fmt.Errorf("SODV check result is incomplete")
		}
		fmt.Printf("SODV check: state=%s records=%d transactions=%d pass=%d warnings=%d violations=%d ledger_digest=%s\n",
			value.Summary.State, value.RecordsChecked, value.TransactionsChecked, value.Summary.Pass,
			value.Summary.Warnings, value.Summary.Violations, value.Ledger.Digest)
		if value.Summary.State != "valid" || value.Summary.Violations != 0 {
			return fmt.Errorf("SODV release-ledger check reported violations")
		}
		return nil
	case "verify":
		var value struct {
			AuthorizationRecordID string `json:"authorization_record_id"`
			VerificationState     string `json:"verification_state"`
			ResultDigest          string `json:"result_digest"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.AuthorizationRecordID == "" || value.ResultDigest == "" {
			return fmt.Errorf("SODV verify result is incomplete")
		}
		fmt.Printf("SODV verify: authorization=%s state=%s digest=%s noncanonical=true declares_completion=false\n",
			value.AuthorizationRecordID, value.VerificationState, value.ResultDigest)
		return nil
	case "propose":
		var value struct {
			ProposalID     string `json:"proposal_id"`
			ProposalDigest string `json:"proposal_digest"`
			Operations     []struct {
				Type       string `json:"type"`
				TargetPath string `json:"target_path"`
			} `json:"operations"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.ProposalID == "" || len(value.Operations) != 1 {
			return fmt.Errorf("SODV proposal result is incomplete")
		}
		fmt.Printf("SODV proposal: id=%s digest=%s operation=%s target=%s ratified=false apply=false\n",
			value.ProposalID, value.ProposalDigest, value.Operations[0].Type, value.Operations[0].TargetPath)
		return nil
	case "recover":
		var value struct {
			Action            string `json:"action"`
			DeleteRecommended bool   `json:"delete_recommended"`
			ResultDigest      string `json:"result_digest"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.Action == "" || value.ResultDigest == "" {
			return fmt.Errorf("SODV recovery result is incomplete")
		}
		fmt.Printf("SODV recovery: action=%s delete_recommended=%t digest=%s journal_mutated=false apply=false\n",
			value.Action, value.DeleteRecommended, value.ResultDigest)
		return nil
	case "project":
		var value struct {
			RecordCount      uint64 `json:"record_count"`
			TransactionCount uint64 `json:"transaction_count"`
			ProjectionDigest string `json:"projection_digest"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.ProjectionDigest == "" {
			return fmt.Errorf("SODV projection result is incomplete")
		}
		fmt.Printf("SODV projection: records=%d transactions=%d digest=%s noncanonical=true rebuildable=true\n",
			value.RecordCount, value.TransactionCount, value.ProjectionDigest)
		return nil
	default:
		return fmt.Errorf("unsupported SODV result")
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

type sclvEvidenceResult struct {
	Protocol          string          `json:"protocol"`
	AdapterID         string          `json:"adapter_id"`
	AdapterVersion    string          `json:"adapter_version"`
	ProviderNamespace string          `json:"provider_namespace"`
	EvidenceKind      string          `json:"evidence_kind"`
	ObservedAt        string          `json:"observed_at"`
	SourceReference   string          `json:"source_reference"`
	Repository        json.RawMessage `json:"repository"`
	ChangeRequest     json.RawMessage `json:"change_request"`
	Ratification      json.RawMessage `json:"ratification"`
	EvidenceDigest    string          `json:"evidence_digest"`
}

func runSCLVEvidence(adapter string, options sclvOptions) error {
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
	payload, err := knowledgeengine.ReadPayload(options.input)
	if err != nil {
		return err
	}
	response, err := knowledgeengine.InvokeSCLVEvidence(
		context.Background(), options.prefix, options.version, repoRoot, adapter, payload)
	if err != nil {
		return err
	}
	result, err := validateSCLVEvidenceResult(adapter, options.version, response.Result)
	if err != nil {
		return err
	}
	if options.jsonOutput {
		var output bytes.Buffer
		if err := json.Indent(&output, response.Result, "", "  "); err != nil {
			return fmt.Errorf("format SCLV provider evidence: %w", err)
		}
		fmt.Println(output.String())
		return nil
	}
	var repositoryEvidence struct {
		RevisionScheme string `json:"revision_scheme"`
		RevisionValue  string `json:"revision_value"`
		TreeDigest     string `json:"tree_digest"`
	}
	var ratification struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(result.Repository, &repositoryEvidence); err != nil {
		return fmt.Errorf("decode validated SCLV repository evidence: %w", err)
	}
	if err := json.Unmarshal(result.Ratification, &ratification); err != nil {
		return fmt.Errorf("decode validated SCLV ratification evidence: %w", err)
	}
	fmt.Printf(
		"SCLV evidence: adapter=%s provider=%s kind=%s revision=%s:%s tree_digest=%s evidence_digest=%s ratification=%s truth_decided=false permission_decided=false\n",
		adapter, result.ProviderNamespace, result.EvidenceKind, repositoryEvidence.RevisionScheme,
		repositoryEvidence.RevisionValue, repositoryEvidence.TreeDigest, result.EvidenceDigest,
		ratification.State,
	)
	return nil
}

func validateSCLVEvidenceResult(adapter, version string, raw json.RawMessage) (sclvEvidenceResult, error) {
	required := []string{
		"protocol", "adapter_id", "adapter_version", "provider_namespace", "evidence_kind",
		"observed_at", "source_reference", "repository", "change_request", "ratification",
		"evidence_digest",
	}
	if _, err := exactJSONObject(raw, required); err != nil {
		return sclvEvidenceResult{}, fmt.Errorf("SCLV provider evidence field set is invalid: %w", err)
	}
	var result sclvEvidenceResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return sclvEvidenceResult{}, fmt.Errorf("decode SCLV provider evidence: %w", err)
	}
	expectedID, expectedProvider, expectedKind := "", "", ""
	switch adapter {
	case "local-git":
		expectedID, expectedProvider, expectedKind = "symphony-sclv-evidence-local-git", "local-git", "revision"
	case "airgap":
		expectedID, expectedProvider, expectedKind = "symphony-sclv-evidence-airgap", "airgap", "combined"
	default:
		return sclvEvidenceResult{}, fmt.Errorf("unsupported SCLV evidence adapter")
	}
	if result.Protocol != "symphony.knowledge.provider-evidence.v1" || result.AdapterID != expectedID ||
		result.AdapterVersion != version || result.ProviderNamespace != expectedProvider ||
		result.EvidenceKind != expectedKind || !validEvidenceUTC(result.ObservedAt) ||
		!validEvidenceText(result.SourceReference, 4096) || !validTaggedDigest(result.EvidenceDigest) {
		return sclvEvidenceResult{}, fmt.Errorf("SCLV provider evidence identity is invalid")
	}

	if _, err := exactJSONObject(result.Repository, []string{"revision_scheme", "revision_value", "tree_digest"}); err != nil {
		return sclvEvidenceResult{}, fmt.Errorf("SCLV repository evidence is invalid: %w", err)
	}
	var repositoryEvidence struct {
		RevisionScheme string `json:"revision_scheme"`
		RevisionValue  string `json:"revision_value"`
		TreeDigest     string `json:"tree_digest"`
	}
	if err := json.Unmarshal(result.Repository, &repositoryEvidence); err != nil ||
		!validGitRevision(repositoryEvidence.RevisionScheme, repositoryEvidence.RevisionValue) ||
		!validTaggedDigest(repositoryEvidence.TreeDigest) {
		return sclvEvidenceResult{}, fmt.Errorf("SCLV repository evidence values are invalid")
	}
	if err := validateSCLVChangeRequest(result.ChangeRequest, adapter == "local-git"); err != nil {
		return sclvEvidenceResult{}, err
	}
	if err := validateSCLVRatification(result.Ratification, adapter == "airgap"); err != nil {
		return sclvEvidenceResult{}, err
	}

	var canonical map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&canonical); err != nil {
		return sclvEvidenceResult{}, fmt.Errorf("decode SCLV evidence for digest verification: %w", err)
	}
	delete(canonical, "evidence_digest")
	encoded, err := marshalDigestCanonical(canonical)
	if err != nil {
		return sclvEvidenceResult{}, fmt.Errorf("encode SCLV evidence for digest verification: %w", err)
	}
	digest := sha256.Sum256(encoded)
	if "sha256:"+hex.EncodeToString(digest[:]) != result.EvidenceDigest {
		return sclvEvidenceResult{}, fmt.Errorf("SCLV provider evidence digest mismatch")
	}
	return result, nil
}

func validateSCLVChangeRequest(raw json.RawMessage, requireAbsent bool) error {
	if _, err := exactJSONObject(raw, []string{"state", "provider", "id", "reference", "absence_reason"}); err != nil {
		return fmt.Errorf("SCLV change-request evidence field set is invalid: %w", err)
	}
	var value struct {
		State         string `json:"state"`
		Provider      string `json:"provider"`
		ID            string `json:"id"`
		Reference     string `json:"reference"`
		AbsenceReason string `json:"absence_reason"`
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return fmt.Errorf("decode SCLV change-request evidence: %w", err)
	}
	if value.State == "present" && !requireAbsent {
		if !validEvidenceToken(value.Provider, 128) || value.ID == "not_applicable" ||
			value.Reference == "not_applicable" || !validEvidenceText(value.ID, 4096) ||
			!validEvidenceText(value.Reference, 4096) || value.AbsenceReason != "not_applicable" {
			return fmt.Errorf("SCLV present change-request evidence is inconsistent")
		}
		return nil
	}
	if value.State != "not_applicable" || value.Provider != "not_applicable" ||
		value.ID != "not_applicable" || value.Reference != "not_applicable" ||
		value.AbsenceReason == "not_applicable" || !validEvidenceText(value.AbsenceReason, 4096) {
		return fmt.Errorf("SCLV absent change-request evidence is inconsistent")
	}
	return nil
}

func validateSCLVRatification(raw json.RawMessage, requireAsserted bool) error {
	if _, err := exactJSONObject(raw, []string{
		"state", "subject", "effective_permission", "method", "evidence_reference",
		"evidence_digest", "absence_reason",
	}); err != nil {
		return fmt.Errorf("SCLV ratification evidence field set is invalid: %w", err)
	}
	var value struct {
		State               string `json:"state"`
		Subject             string `json:"subject"`
		EffectivePermission string `json:"effective_permission"`
		Method              string `json:"method"`
		EvidenceReference   string `json:"evidence_reference"`
		EvidenceDigest      string `json:"evidence_digest"`
		AbsenceReason       string `json:"absence_reason"`
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return fmt.Errorf("decode SCLV ratification evidence: %w", err)
	}
	if value.State == "asserted" && requireAsserted {
		if value.Subject == "not_applicable" || value.EffectivePermission == "not_applicable" ||
			value.Method == "not_applicable" || value.EvidenceReference == "not_applicable" ||
			!validEvidenceText(value.Subject, 4096) || !validEvidenceText(value.EffectivePermission, 4096) ||
			!validEvidenceText(value.Method, 4096) || !validEvidenceText(value.EvidenceReference, 4096) ||
			!validTaggedDigest(value.EvidenceDigest) || value.AbsenceReason != "not_applicable" {
			return fmt.Errorf("SCLV asserted ratification evidence is inconsistent")
		}
		return nil
	}
	if value.State != "not_asserted" || requireAsserted || value.Subject != "not_applicable" ||
		value.EffectivePermission != "not_applicable" || value.Method != "not_applicable" ||
		value.EvidenceReference != "not_applicable" || value.EvidenceDigest != "not_applicable" ||
		value.AbsenceReason == "not_applicable" || !validEvidenceText(value.AbsenceReason, 4096) {
		return fmt.Errorf("SCLV absent ratification evidence is inconsistent")
	}
	return nil
}

func exactJSONObject(raw json.RawMessage, required []string) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil || len(fields) != len(required) {
		return nil, fmt.Errorf("object is incomplete or contains unknown fields")
	}
	for _, field := range required {
		if _, ok := fields[field]; !ok {
			return nil, fmt.Errorf("required field %q is absent", field)
		}
	}
	return fields, nil
}

func marshalDigestCanonical(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buffer.Bytes(), []byte("\n")), nil
}

func validEvidenceUTC(value string) bool {
	if len(value) != len("2006-01-02T15:04:05Z") {
		return false
	}
	parsed, err := time.Parse("2006-01-02T15:04:05Z", value)
	return err == nil && parsed.UTC().Format("2006-01-02T15:04:05Z") == value
}

func validGitRevision(scheme, revision string) bool {
	length := 0
	switch scheme {
	case "git-sha1":
		length = 40
	case "git-sha256":
		length = 64
	default:
		return false
	}
	if len(revision) != length {
		return false
	}
	for _, character := range revision {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func validEvidenceToken(value string, maximum int) bool {
	if value == "" || len(value) > maximum {
		return false
	}
	for _, character := range value {
		if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') &&
			(character < '0' || character > '9') && character != '.' && character != '_' &&
			character != ':' && character != '-' {
			return false
		}
	}
	return true
}

func validEvidenceText(value string, maximum int) bool {
	if value == "" || len(value) > maximum {
		return false
	}
	return strings.IndexFunc(value, func(character rune) bool {
		return character < 0x20 || character == 0x7f
	}) < 0
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
				CallerDeclaredOperation  *bool `json:"caller_declared_operation"`
				EngineDecidedDomainTruth *bool `json:"engine_decided_domain_truth"`
				Ratified                 *bool `json:"ratified"`
			} `json:"authority"`
		}
		if err := json.Unmarshal(result, &value); err != nil || value.Protocol != "symphony.knowledge.proposal.v1" ||
			value.ModuleID != "sclv-engine" || value.EngineID != "symphony-sclv" || value.VectorID != "sclv" ||
			value.ProposalID == "" || !validTaggedDigest(value.ProposalDigest) ||
			len(value.WriteSet) != 1 || value.WriteSet[0].TargetPath != "knowledge/sclv/CHANGELOG.md" ||
			len(value.Operations) != 1 || value.Operations[0].Type != "append_record_v3" ||
			value.Operations[0].TargetPath != "knowledge/sclv/CHANGELOG.md" ||
			!explicitTrue(value.Authority.CallerDeclaredOperation) ||
			!explicitFalse(value.Authority.EngineDecidedDomainTruth) ||
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

func runSSIAGPolicy(operation string, options ssiagOptions) error {
	if options.topsID == "" {
		return fmt.Errorf("--tops-id is required")
	}
	client, err := ssiagclient.NewForTOPS(options.scope, options.topsID, 8*time.Second)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if _, err := requireSSIAGStatus(ctx, client, options.topsID, options.scope); err != nil {
		return err
	}
	switch operation {
	case "status":
		result, err := client.PolicyStatus(ctx)
		if err != nil {
			return err
		}
		return printSSIAGPolicyResult(result, options.topsID, options.jsonOutput)
	case "propose":
		if options.operationID == "" || options.expectedPolicy == "" {
			return fmt.Errorf("--operation-id and --expected-policy-digest are required")
		}
		if options.reset == (options.input != "") {
			return fmt.Errorf("exactly one of --input or --reset is required")
		}
		if options.ttl <= 0 || options.ttl > 10*time.Minute {
			return fmt.Errorf("--ttl must be greater than zero and at most 10m")
		}
		var desired *ssiagclient.AuthorizationPolicy
		change := "reset"
		if options.input != "" {
			var policyValue ssiagclient.AuthorizationPolicy
			if err := ssiagclient.ReadBoundedJSON(options.input, &policyValue); err != nil {
				return err
			}
			desired = &policyValue
			change = "replace"
		}
		requestID, err := stavprotocol.GenerateUUIDv4()
		if err != nil {
			return err
		}
		correlationID, err := stavprotocol.GenerateUUIDv4()
		if err != nil {
			return err
		}
		now := time.Now().UTC().Truncate(time.Second)
		proposal, err := client.ProposePolicy(ctx, ssiagclient.PolicyProposalRequest{
			Protocol: "symphony.ssiag.policy-proposal-request.v1", OperationID: options.operationID,
			RequestID: requestID, CorrelationID: correlationID, AuthorityBasis: options.authorityBasis,
			ExpectedPolicyDigest: options.expectedPolicy, Change: change, DesiredPolicy: desired,
			RequestedAt: now, ExpiresAt: now.Add(options.ttl).UTC().Truncate(time.Second),
		})
		if err != nil {
			return err
		}
		if proposal.TOPSID != options.topsID || proposal.CallerClassUsed || proposal.Canonical || proposal.Applied {
			return fmt.Errorf("SSIAG returned an invalid policy proposal binding")
		}
		if options.jsonOutput {
			return printSSIAGJSON(proposal)
		}
		fmt.Printf("SSIAG policy proposal: tops_id=%s operation_id=%s change=%s expected=%s desired=%s expires_at=%s digest=%s caller_class_used=false canonical=false applied=false\n",
			proposal.TOPSID, proposal.OperationID, proposal.Change, proposal.ExpectedPolicyDigest,
			proposal.DesiredPolicyDigest, proposal.ExpiresAt.Format(time.RFC3339), proposal.ProposalDigest)
		return nil
	case "apply":
		if options.input == "" {
			return fmt.Errorf("--input is required")
		}
		var proposal ssiagclient.PolicyProposal
		if err := ssiagclient.ReadBoundedJSON(options.input, &proposal); err != nil {
			return err
		}
		if proposal.TOPSID != options.topsID {
			return fmt.Errorf("SSIAG policy proposal TOPS ID does not match --tops-id")
		}
		result, err := client.ApplyPolicy(ctx, ssiagclient.PolicyApplyRequest{
			Protocol: "symphony.ssiag.policy-apply-request.v1", Proposal: proposal,
		})
		if err != nil {
			return err
		}
		return printSSIAGPolicyResult(result, options.topsID, options.jsonOutput)
	case "recover":
		if options.operationID == "" {
			return fmt.Errorf("--operation-id is required")
		}
		if options.discover == (options.expectedAttempt != "") {
			return fmt.Errorf("exactly one of --expected-attempt-digest or --discover is required")
		}
		result, err := client.RecoverPolicy(ctx, ssiagclient.PolicyRecoveryRequest{
			Protocol: "symphony.ssiag.policy-recovery-request.v1", OperationID: options.operationID,
			ExpectedAttemptDigest: options.expectedAttempt, Discover: options.discover,
		})
		if err != nil {
			return err
		}
		return printSSIAGPolicyResult(result, options.topsID, options.jsonOutput)
	default:
		return fmt.Errorf("unknown SSIAG policy operation %q", operation)
	}
}

func printSSIAGPolicyResult(result ssiagclient.PolicyResult, topsID string, jsonOutput bool) error {
	if result.TOPSID != topsID || result.CallerClassUsed || result.Canonical {
		return fmt.Errorf("SSIAG returned an invalid policy result binding")
	}
	if jsonOutput {
		return printSSIAGJSON(result)
	}
	fmt.Printf("SSIAG policy %s: tops_id=%s source=%s generation=%d policy_digest=%s state_digest=%s recovery_required=%t changed=%t recovered=%t caller_class_used=false canonical=false\n",
		result.Operation, result.TOPSID, result.Source, result.Generation, result.PolicyDigest,
		result.StateDigest, result.RecoveryRequired, result.Changed, result.Recovered)
	if result.RecoveryRequired {
		fmt.Printf("SSIAG policy recovery: attempt_digest=%s\n", result.AttemptDigest)
	}
	return nil
}

type ssiagLifecycleGrant struct {
	ID             string `json:"id"`
	SubjectID      string `json:"subject_id"`
	AuthorityBasis string `json:"authority_basis"`
	Operation      string `json:"operation"`
	Resource       string `json:"resource"`
	Audience       string `json:"audience"`
	Scope          string `json:"scope"`
}

type ssiagLifecycleGrantPlan struct {
	Protocol        string                `json:"protocol"`
	FormatVersion   uint64                `json:"format_version"`
	TOPSID          string                `json:"tops_id"`
	ProfileID       string                `json:"profile_id"`
	SubjectID       string                `json:"subject_id"`
	AuthorityBasis  string                `json:"authority_basis"`
	Resource        string                `json:"resource"`
	CatalogResource string                `json:"catalog_resource"`
	Audience        string                `json:"audience"`
	Scope           string                `json:"scope"`
	Grants          []ssiagLifecycleGrant `json:"grants"`
	ApplyEnabled    bool                  `json:"apply_enabled"`
	Canonical       bool                  `json:"canonical"`
	PlanDigest      string                `json:"plan_digest"`
}

func runSSIAGLifecycleGrantPlan(options ssiagOptions) error {
	if options.topsID == "" || options.subjectID == "" {
		return fmt.Errorf("--tops-id and --subject-id are required")
	}
	if _, err := ssiagclient.ConfigForTOPS(options.scope, options.topsID); err != nil {
		return err
	}
	if !validSessionToken(options.profileID) || !validSessionToken(options.subjectID) ||
		(options.authorityBasis != "host_owner" && options.authorityBasis != "granted_permission") {
		return fmt.Errorf("lifecycle grant identity or authority basis is invalid")
	}
	permissions := []string{
		"apply.close", "apply.finalize", "apply.prepare", "apply.recover", "apply.status",
		"boot", "boot.recover", "boot.status", "observe",
		"host.disable", "host.enable", "host.install", "host.reconcile", "host.run", "host.status", "host.uninstall", "host.update",
		"ownership.adopt", "ownership.reconcile", "ownership.release", "ownership.status",
		"profile.list", "profile.remove", "profile.set", "profile.show", "report",
	}
	resource := lifecycleResource(options.topsID, options.profileID, "")
	catalogResource := lifecycleCatalogResource(options.topsID)
	grants := make([]ssiagLifecycleGrant, 0, len(permissions))
	for _, permission := range permissions {
		operation := "symphony.knowledge.lifecycle." + permission
		grantResource := resource
		if permission == "profile.list" {
			grantResource = catalogResource
		}
		scope := "tops:" + options.topsID
		idDigest := sha256.Sum256([]byte(options.subjectID + "\n" + options.authorityBasis + "\n" +
			operation + "\n" + grantResource + "\nqxctl\n" + scope))
		grants = append(grants, ssiagLifecycleGrant{
			ID:        "ssiag-grant:lifecycle:" + hex.EncodeToString(idDigest[:12]),
			SubjectID: options.subjectID, AuthorityBasis: options.authorityBasis,
			Operation: operation, Resource: grantResource, Audience: "qxctl", Scope: scope,
		})
	}
	plan := ssiagLifecycleGrantPlan{
		Protocol: "symphony.ssiag.lifecycle-grant-plan.v1", FormatVersion: 1,
		TOPSID: options.topsID, ProfileID: options.profileID, SubjectID: options.subjectID,
		AuthorityBasis: options.authorityBasis, Resource: resource, CatalogResource: catalogResource, Audience: "qxctl",
		Scope: "tops:" + options.topsID, Grants: grants, ApplyEnabled: false, Canonical: false,
	}
	encoded, err := json.Marshal(plan)
	if err != nil {
		return err
	}
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if err := decoder.Decode(&object); err != nil {
		return err
	}
	delete(object, "plan_digest")
	canonical, err := json.Marshal(object)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(canonical)
	plan.PlanDigest = "sha256:" + hex.EncodeToString(digest[:])
	if options.jsonOutput {
		return printIndentedJSON(plan)
	}
	fmt.Printf("SSIAG lifecycle grant plan: tops_id=%s profile=%s subject=%s grants=%d resource=%s digest=%s apply_enabled=false canonical=false\n",
		plan.TOPSID, plan.ProfileID, plan.SubjectID, len(plan.Grants), plan.Resource, plan.PlanDigest)
	return nil
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
