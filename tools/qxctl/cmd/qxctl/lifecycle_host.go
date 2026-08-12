package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgelifecycle"
)

func runKnowledgeLifecycleHost(operation string, options knowledgeLifecycleOptions) error {
	if options.topsID == "" {
		return fmt.Errorf("--tops-id is required")
	}
	if options.scope != "system" {
		return fmt.Errorf("lifecycle host integration requires --scope system")
	}
	if options.stateRoot == "" {
		options.stateRoot = "/var/lib"
	}
	if options.ttl <= 0 || options.ttl > 24*time.Hour {
		return fmt.Errorf("--ttl must be greater than zero and no more than 24h")
	}
	if operation == "run" {
		return runKnowledgeLifecycleHostBoot(options)
	}
	if err := authorizeKnowledgeLifecycle(options, "host."+operation,
		lifecycleResource(options.topsID, options.profileID, "host")); err != nil {
		return err
	}
	store, err := lifecycleStore(options)
	if err != nil {
		return err
	}
	admin, err := knowledgelifecycle.NewHostAdmin(store)
	if err != nil {
		return err
	}
	var result knowledgelifecycle.HostIntegrationResult
	switch operation {
	case "install", "update":
		if options.expectedHostDigest == "" {
			return fmt.Errorf("--expected-host-digest is required")
		}
		if options.expectedHostDigest != "absent" && !validTaggedDigest(options.expectedHostDigest) {
			return fmt.Errorf("--expected-host-digest must be absent or an exact tagged SHA-256 digest")
		}
		if options.recoveryMode != "strict" && options.recoveryMode != "discover" {
			return fmt.Errorf("--recovery-mode must be strict or discover")
		}
		repository, err := resolveKnowledgeRepository(options.repository)
		if err != nil {
			return err
		}
		integrationRoot := options.integrationRoot
		if integrationRoot == "" {
			integrationRoot = filepath.Join(
				"/var/lib/symphony", options.topsID, "qxctl", "lifecycle-host",
				knowledgelifecycle.HostProfileKey(options.profileID))
		}
		integrationRoot, err = filepath.Abs(integrationRoot)
		if err != nil || filepath.Clean(integrationRoot) != integrationRoot {
			return fmt.Errorf("--integration-root must be an absolute normalized path")
		}
		result, err = admin.Provision(knowledgelifecycle.HostProvisionInput{
			Operation: operation, ProfileID: options.profileID, RepositoryRoot: repository,
			IntegrationRoot: integrationRoot, RecoveryMode: options.recoveryMode,
			DesiredEnabled: !options.hostDisabled, ExpectedDigest: options.expectedHostDigest,
			Now: time.Now(),
		})
	case "status":
		result, err = admin.Status(options.profileID)
	case "reconcile":
		result, err = admin.Reconcile(options.profileID, time.Now())
	case "enable", "disable":
		if !validTaggedDigest(options.expectedHostDigest) {
			return fmt.Errorf("--expected-host-digest must be an exact tagged SHA-256 digest")
		}
		result, err = admin.SetEnabled(options.profileID, options.expectedHostDigest, operation == "enable", time.Now())
	case "uninstall":
		if !validTaggedDigest(options.expectedHostDigest) {
			return fmt.Errorf("--expected-host-digest must be an exact tagged SHA-256 digest")
		}
		result, err = admin.Uninstall(options.profileID, options.expectedHostDigest, time.Now())
	default:
		return fmt.Errorf("unknown lifecycle host operation %q", operation)
	}
	if err != nil {
		return err
	}
	return printLifecycleHostResult(result, options.jsonOutput)
}

func runKnowledgeLifecycleHostBoot(options knowledgeLifecycleOptions) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("native lifecycle host integration is Linux-only; use WSL or a remote Linux TOPS node")
	}
	store, err := lifecycleStore(options)
	if err != nil {
		return err
	}
	snapshot, err := store.HostSnapshot(options.profileID)
	if err != nil {
		return err
	}
	if !snapshot.Exists || snapshot.Integration.State != "installed" || !snapshot.Integration.DesiredEnabled {
		return fmt.Errorf("enabled lifecycle host integration is absent")
	}
	record := snapshot.Integration
	if err := authorizeKnowledgeLifecycle(options, "host.run",
		lifecycleResource(options.topsID, options.profileID, record.IntegrationDigest)); err != nil {
		return err
	}
	currentDigest, err := knowledgelifecycle.DigestCurrentExecutable()
	if err != nil {
		return err
	}
	executorAccepted := currentDigest == record.ActiveExecutor.Digest
	for _, fallback := range record.FallbackExecutors {
		executorAccepted = executorAccepted || currentDigest == fallback.Digest
	}
	if !executorAccepted {
		return fmt.Errorf("running qxctl is outside the descriptor's content-addressed compatibility set")
	}
	options.repository = record.RepositoryRoot
	bootID, err := linuxBootID()
	if err != nil {
		return err
	}
	operationID := "host-boot:" + bootID
	status, statusErr := invokeKnowledgeLifecycleBootState("lifecycle_boot_status", options)
	if statusErr != nil && record.RecoveryMode == "discover" {
		recovery := options
		recovery.operationID = operationID + ".recover"
		recovery.discover = true
		if _, err := invokeKnowledgeLifecycleBootState("lifecycle_boot_recover", recovery); err != nil {
			return fmt.Errorf("lifecycle host boot status failed (%v) and discovery recovery failed: %w", statusErr, err)
		}
		status, statusErr = invokeKnowledgeLifecycleBootState("lifecycle_boot_status", options)
	}
	if statusErr != nil {
		return statusErr
	}
	if status.JournalPresent && status.OperationID == operationID {
		if options.jsonOutput {
			return printIndentedJSON(map[string]any{
				"protocol": "symphony.knowledge.lifecycle-host-boot-result.v1", "tops_id": options.topsID,
				"profile_id": options.profileID, "boot_id": bootID, "journal_digest": status.JournalDigest,
				"changed": false, "replayed": true, "apply_authorized": false, "canonical": false,
			})
		}
		fmt.Printf("Knowledge lifecycle host boot: profile=%s boot_id=%s journal_digest=%s changed=false replayed=true apply_authorized=false canonical=false\n",
			options.profileID, bootID, status.JournalDigest)
		return nil
	}
	coordinator, err := exactBoundCoordinator(options.stateRoot)
	if err != nil {
		return err
	}
	applyStatus, err := invokeLifecycleApplyState(
		options, coordinator.Prefix, coordinator.Version, record.RepositoryRoot,
		"lifecycle_apply_status", "", "")
	if err != nil {
		return err
	}
	options.operationID = operationID
	options.expectedJournalDigest = "absent"
	if status.JournalPresent {
		options.expectedJournalDigest = status.JournalDigest
	}
	options.priorAppliedStateDigest = applyStatus.AppliedDigest
	var bootResult validatedLifecycleBootResult
	options.bootResultSink = &bootResult
	if err := runKnowledgeLifecycleBoot(options); err != nil {
		return err
	}
	output := map[string]any{
		"protocol": "symphony.knowledge.lifecycle-host-boot-result.v1", "tops_id": options.topsID,
		"profile_id": options.profileID, "boot_id": bootID, "journal_digest": bootResult.JournalDigest,
		"changed": bootResult.Changed, "replayed": false, "apply_authorized": false, "canonical": false,
	}
	if options.jsonOutput {
		return printIndentedJSON(output)
	}
	fmt.Printf("Knowledge lifecycle host boot: profile=%s boot_id=%s journal_digest=%s changed=%t replayed=false apply_authorized=false canonical=false\n",
		options.profileID, bootID, bootResult.JournalDigest, bootResult.Changed)
	return nil
}

func printLifecycleHostResult(result knowledgelifecycle.HostIntegrationResult, jsonOutput bool) error {
	if jsonOutput {
		return printIndentedJSON(result)
	}
	digest := "absent"
	generation := uint64(0)
	enabled := false
	state := "absent"
	if result.Integration != nil {
		digest = result.Integration.IntegrationDigest
		generation = result.Integration.Generation
		enabled = result.Integration.DesiredEnabled
		state = result.Integration.State
	}
	fmt.Printf("Knowledge lifecycle host %s: profile=%s present=%t state=%s enabled=%t generation=%d digest=%s drift=%d changed=%t recovered=%t apply_authorized=false canonical=false\n",
		result.Operation, result.ProfileID, result.Present, state, enabled, generation, digest,
		len(result.Drift), result.Changed, result.Recovered)
	for _, drift := range result.Drift {
		fmt.Printf("Host drift: code=%s detail=%s\n", drift.Code, drift.Detail)
	}
	return nil
}

func linuxBootID() (string, error) {
	const path = "/proc/sys/kernel/random/boot_id"
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > 128 {
		return "", fmt.Errorf("Linux boot identity is unavailable")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	value := strings.TrimSpace(string(data))
	if len(value) != 36 || strings.ToLower(value) != value {
		return "", fmt.Errorf("Linux boot identity is invalid")
	}
	for index, character := range value {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if character != '-' {
				return "", fmt.Errorf("Linux boot identity is invalid")
			}
			continue
		}
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return "", fmt.Errorf("Linux boot identity is invalid")
		}
	}
	return value, nil
}
