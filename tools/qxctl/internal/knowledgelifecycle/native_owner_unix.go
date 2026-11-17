//go:build darwin || linux

package knowledgelifecycle

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

type nativeOwnerTarget struct {
	receipt     receiptV2
	receiptPath string
	controller  string
	cleanup     func()
}

func (e *Executor) executeNativeOwner(result *ExecutionResult, action PlannedAction, component DesiredComponent, wanted bool, observed ObservedComponent, present bool) error {
	if len(e.nativeOwners[action.ComponentID]) == 0 {
		return fmt.Errorf("compatibility_blocked: exact retained native owner executable is not explicitly selected")
	}
	if e.executionScope == nil || !wanted || !safeAbsolutePath(component.InstallRoot) {
		return fmt.Errorf("authorization_denied: native owner dispatch requires an exact reviewed execution scope and installation root")
	}
	if !actionBeforeMatches(action, observed, present) {
		return fmt.Errorf("observation_retryable: native owner target changed after action preparation")
	}
	if len(action.ExpectedArtifactDigests) == 0 || len(action.ExpectedArtifactDigests) > 64 {
		return fmt.Errorf("integrity_fatal: native owner action must bind bounded exact receipts")
	}
	if action.Kind == "install" {
		if component.Presence != "present" || component.SelectedPackage == nil || len(action.ExpectedArtifactDigests) != 1 || action.ExpectedArtifactDigests[0] != component.SelectedPackage.ReceiptDigest {
			return fmt.Errorf("integrity_fatal: native install does not bind the exact desired package")
		}
	} else if component.Presence != "absent" || present && (observed.SelectedPackageDigest != nil || observed.Activation != "inactive" || observed.Docking != "undocked") {
		return fmt.Errorf("dependency_wait: native removal requires desired absence and verified deselection, inactivity and undocking")
	}
	// Validate every route and retain independent controller copies before the
	// first physical effect. Never execute a receipt-discovered path or argv.
	targets := make([]nativeOwnerTarget, 0, len(action.ExpectedArtifactDigests))
	defer func() {
		for _, target := range targets {
			target.cleanup()
		}
	}()
	for _, digest := range action.ExpectedArtifactDigests {
		identity := PackageIdentity{ReceiptProtocol: "symphony.knowledge.install-receipt.v2", ReceiptDigest: digest}
		if action.Kind == "install" {
			identity = *component.SelectedPackage
		}
		source, receipt, err := e.stagedReceipt(identity)
		if err != nil {
			return err
		}
		if err := e.validateNativeRetentionSources(source.root, e.nativeOwners[action.ComponentID]); err != nil {
			return err
		}
		if err := validateNativeOwnerReceipt(action.ComponentID, source, receipt); err != nil {
			return err
		}
		if err := validateStagedPackage(source.root, receipt); err != nil {
			return err
		}
		disposition := "retained"
		if action.Kind == "uninstall" {
			disposition = "release"
		}
		if !e.executionScope.AllowsOwnership(component.InstallRoot, action.ComponentID, digest, disposition) {
			return fmt.Errorf("authorization_denied: native effect lacks the exact scoped ownership effect")
		}
		for _, installed := range observed.Packages {
			if installed.ReceiptDigest == digest && (installed.InstallRoot != component.InstallRoot || installed.Integrity != "valid") {
				return fmt.Errorf("integrity_fatal: observed native package identity is not valid at the reviewed root")
			}
		}
		controller, cleanup, err := retainNativeController(e.nativeOwners[action.ComponentID], receipt.Files[0])
		if err != nil {
			return err
		}
		targets = append(targets, nativeOwnerTarget{receipt, source.relativePath, controller, cleanup})
	}
	owner, err := NewOwnershipStore(component.InstallRoot, e.stateRoot, e.topsID, e.profileID)
	if err != nil {
		return err
	}
	owner.executionScope = e.executionScope
	locked, err := lockInstallRoot(component.InstallRoot, false)
	if err != nil {
		return fmt.Errorf("dependency_wait: native owner root is not ready: %w", err)
	}
	defer locked.close()
	for _, target := range targets {
		if err := dispatchNativeOwner(locked, owner, action.Kind, target); err != nil {
			return err
		}
		result.AfterEvidenceDigests = append(result.AfterEvidenceDigests, target.receipt.ReceiptDigest)
	}
	return nil
}

func (e *Executor) validateNativeRetentionSources(source string, controllers []string) error {
	canonical, err := filepath.EvalSymlinks(source)
	if err != nil {
		return fmt.Errorf("integrity_fatal: staged native source is unavailable: %w", err)
	}
	for _, effect := range e.executionScope.Effects {
		for _, path := range append([]string{canonical}, controllers...) {
			relative, err := filepath.Rel(effect.InstallRoot, path)
			if err != nil || relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				return fmt.Errorf("integrity_fatal: native rollback sources and retained controllers must be outside every installation root in the reviewed scope")
			}
		}
	}
	return nil
}

func validateNativeOwnerReceipt(component string, candidate receiptCandidate, receipt receiptV2) error {
	binary := nativeOwnerBinary(component)
	if binary == "" || receipt.ComponentID != component || receipt.ModuleID != component || receipt.PackageID != component ||
		receipt.ComponentKind != "service" || receipt.VectorID != nil || receipt.EngineID != nil || receipt.InstallScope != "prefix" || receipt.PrefixMode != "installation_prefix" ||
		!safeToken(receipt.Version, 256) || len(receipt.Files) != 1 || receipt.Files[0].Kind != "executable" || receipt.Files[0].Size == 0 ||
		receipt.Files[0].Path != "libexec/symphony/"+component+"/"+receipt.Version+"/"+binary ||
		candidate.relativePath != "share/symphony/receipts/"+component+"/"+receipt.Version+"/install-receipt.json" ||
		!containsString(receipt.ProvidesCapabilities, "symphony.package-retirement.v1") || !containsString(receipt.ProvidesCapabilities, nativeOwnerDispatchCapability) ||
		len(receipt.PlatformRequirements) != 1 || receipt.PlatformRequirements[0].OS != platformOS() || receipt.PlatformRequirements[0].Architecture != platformArchitecture() || !receipt.PlatformRequirements[0].Critical {
		return fmt.Errorf("compatibility_blocked: receipt does not bind the qualified native service owner, platform and dispatch capability")
	}
	return nil
}

// Retain a fresh private copy before exec, so carrier replacement or cleanup
// cannot change which bytes execute. Package uninstall never executes the
// installed target itself. This does not confer publisher trust on a receipt.
func retainNativeController(paths []string, binary receiptV2File) (string, func(), error) {
	for _, path := range paths {
		parent, err := openOrCreateAbsoluteDirectory(filepath.Dir(path), false)
		if err != nil {
			return "", nil, fmt.Errorf("integrity_fatal: retained controller parent is unsafe: %w", err)
		}
		input, err := nativeOpenAt(parent, filepath.Base(path))
		unix.Close(parent)
		if err != nil {
			return "", nil, fmt.Errorf("integrity_fatal: retained controller is unsafe: %w", err)
		}
		var st unix.Stat_t
		err = unix.Fstat(int(input.Fd()), &st)
		if err != nil || st.Nlink != 1 || st.Mode&0o111 == 0 || st.Size < 1 || st.Size > maxObservedFileBytes("executable") {
			input.Close()
			return "", nil, fmt.Errorf("integrity_fatal: retained controller custody is invalid")
		}
		if uint64(st.Size) != binary.Size {
			input.Close()
			continue
		}
		directory, err := os.MkdirTemp("", "symphony-native-owner-")
		if err != nil {
			input.Close()
			return "", nil, err
		}
		directory, err = filepath.EvalSymlinks(directory)
		if err != nil {
			input.Close()
			return "", nil, err
		}
		cleanup := func() { _ = os.RemoveAll(directory) }
		controller := filepath.Join(directory, "owner")
		output, err := os.OpenFile(controller, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o500)
		if err != nil {
			input.Close()
			cleanup()
			return "", nil, err
		}
		hash := sha256.New()
		n, copyErr := io.Copy(io.MultiWriter(output, hash), io.LimitReader(input, int64(binary.Size)+1))
		input.Close()
		syncErr := output.Sync()
		closeErr := output.Close()
		if copyErr != nil || syncErr != nil || closeErr != nil {
			cleanup()
			return "", nil, fmt.Errorf("integrity_fatal: cannot retain native controller")
		}
		if uint64(n) != binary.Size || "sha256:"+hex.EncodeToString(hash.Sum(nil)) != binary.Digest {
			cleanup()
			continue
		}
		return controller, cleanup, nil
	}
	return "", nil, fmt.Errorf("integrity_fatal: no explicitly selected native controller matches the exact receipt bytes")
}

func nativeOpenAt(rootFD int, relative string) (*os.File, error) {
	parent, leaf, err := openRelativeParent(rootFD, relative, false)
	if err != nil {
		return nil, err
	}
	defer unix.Close(parent)
	fd, err := unix.Openat(parent, leaf, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	if err := validateTrustedRegular(fd); err != nil {
		unix.Close(fd)
		return nil, err
	}
	var stat unix.Stat_t
	if unix.Fstat(fd, &stat) != nil || stat.Nlink != 1 {
		unix.Close(fd)
		return nil, fmt.Errorf("native owner evidence must have independent file custody")
	}
	return os.NewFile(uintptr(fd), relative), nil
}

func nativeReadAt(rootFD int, relative string, bound int64) ([]byte, error) {
	file, err := nativeOpenAt(rootFD, relative)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, bound+1))
	if err != nil || int64(len(data)) > bound {
		return nil, fmt.Errorf("native evidence exceeds its read bound")
	}
	return data, nil
}

func dispatchNativeOwner(locked *installLock, owner *OwnershipStore, operation string, target nativeOwnerTarget) error {
	rootFD := int(locked.root.Fd())
	if err := validateNativeMutationLock(locked); err != nil {
		return err
	}
	var registry OwnershipRegistry
	var err error
	if operation == "install" {
		err = ownershipInstallAllowedAt(rootFD, owner.installRoot, owner, target.receipt.ComponentID, target.receipt.ReceiptDigest)
		if err == nil {
			registry, _, err = readOwnershipRegistryAt(rootFD, owner.installRoot)
		}
	} else {
		registry, err = ownershipReclaimAllowedAt(rootFD, owner.installRoot, owner, target.receipt.ComponentID, target.receipt.ReceiptDigest)
	}
	if err != nil {
		return err
	}
	// A durable tombstone allows explicit reconciliation after a lost result;
	// absence alone never proves that a native retirement completed.
	if operation == "uninstall" && verifyNativeOwnerOutcome(locked, owner.installRoot, target, "retired") == nil {
		return commitOwnershipReclaimedAt(rootFD, registry, owner, target.receipt.ReceiptDigest)
	}
	proof, err := makeNativeDispatchProof(locked, owner, operation, target, registry)
	if err != nil {
		return err
	}
	data, err := json.Marshal(proof)
	if err != nil {
		return err
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		return err
	}
	defer reader.Close()
	// The proof is bounded below PIPE_BUF, and the write end is closed before
	// launching. Native validation cannot block waiting for a trailing writer.
	if len(data) > 4096 {
		writer.Close()
		return fmt.Errorf("integrity_fatal: dispatch proof exceeds its bound")
	}
	_, err = writer.Write(data)
	writer.Close()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, target.controller, nativeOwnerArguments(operation, target.receipt.ComponentID, owner.installRoot, target.receipt.Version)...)
	cmd.ExtraFiles = []*os.File{locked.root, locked.lock, reader}
	// Inherit the actual caller HOME/XDG environment. A staging HOME here
	// would conceal references and could make native removal unsafe.
	var output nativeBoundedOutput
	cmd.Stdout, cmd.Stderr = &output, &output
	cmd.WaitDelay = 2 * time.Second
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("critical_state_unknown: native owner did not confirm completion; preserve the attempt and re-observe before recovery: %v (%s)", err, strings.TrimSpace(output.String()))
	}
	phase := "open"
	if operation == "uninstall" {
		phase = "retired"
	}
	if err := verifyNativeOwnerOutcome(locked, owner.installRoot, target, phase); err != nil {
		return fmt.Errorf("critical_state_unknown: native owner outcome failed independent verification: %w", err)
	}
	if operation == "uninstall" {
		if err := validateNativeMutationLock(locked); err != nil {
			return err
		}
		return commitOwnershipReclaimedAt(rootFD, registry, owner, target.receipt.ReceiptDigest)
	}
	return nil
}

type nativeBoundedOutput struct{ bytes.Buffer }

func (b *nativeBoundedOutput) Write(p []byte) (int, error) {
	n := len(p)
	if remaining := 4096 - b.Len(); remaining > 0 {
		if len(p) > remaining {
			p = p[:remaining]
		}
		_, _ = b.Buffer.Write(p)
	}
	return n, nil
}

func makeNativeDispatchProof(locked *installLock, owner *OwnershipStore, operation string, target nativeOwnerTarget, registry OwnershipRegistry) (nativeDispatchProof, error) {
	var root, lock unix.Stat_t
	if err := unix.Fstat(int(locked.root.Fd()), &root); err != nil {
		return nativeDispatchProof{}, err
	}
	if err := unix.Fstat(int(locked.lock.Fd()), &lock); err != nil {
		return nativeDispatchProof{}, err
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nativeDispatchProof{}, err
	}
	return nativeDispatchProof{
		Protocol: nativeOwnerDispatchCapability, FormatVersion: 1, Operation: operation,
		ComponentID: target.receipt.ComponentID, Version: target.receipt.Version, InstallRoot: owner.installRoot,
		ReceiptDigest: target.receipt.ReceiptDigest, BinaryDigest: target.receipt.Files[0].Digest,
		ControlDomainID: owner.controlDomainID, TOPSID: owner.topsID, ProfileID: owner.profileID,
		OwnershipRegistryDigest: registry.OwnershipRegistryDigest, OwnershipGeneration: registry.Generation,
		RootDevice: uint64(root.Dev), RootInode: root.Ino, LockDevice: uint64(lock.Dev), LockInode: lock.Ino, Nonce: hex.EncodeToString(nonce),
	}, nil
}

type nativeRetirementIdentity struct {
	Owner           string `json:"owner"`
	Version         string `json:"version"`
	ReceiptDigest   string `json:"receipt_digest"`
	BinaryDigest    string `json:"binary_digest"`
	BinaryRelative  string `json:"binary_relative"`
	ReceiptRelative string `json:"receipt_relative"`
	BinarySize      uint64 `json:"binary_size"`
}
type nativeRetirementState struct {
	Protocol        string                   `json:"protocol"`
	Identity        nativeRetirementIdentity `json:"identity"`
	Epoch           string                   `json:"epoch"`
	RootDevice      uint64                   `json:"root_device"`
	RootInode       uint64                   `json:"root_inode"`
	Phase           string                   `json:"phase"`
	GuardDevice     uint64                   `json:"guard_device"`
	GuardInode      uint64                   `json:"guard_inode"`
	AdmissionDevice uint64                   `json:"admission_device"`
	AdmissionInode  uint64                   `json:"admission_inode"`
	UseDevice       uint64                   `json:"use_device"`
	UseInode        uint64                   `json:"use_inode"`
}

func verifyNativeOwnerOutcome(locked *installLock, root string, target nativeOwnerTarget, phase string) error {
	if err := validateNativeMutationLock(locked); err != nil {
		return err
	}
	rootFD := int(locked.root.Fd())
	current, err := openOrCreateAbsoluteDirectory(root, false)
	if err != nil {
		return err
	}
	defer unix.Close(current)
	var pinned, named unix.Stat_t
	if unix.Fstat(rootFD, &pinned) != nil || unix.Fstat(current, &named) != nil || pinned.Dev != named.Dev || pinned.Ino != named.Ino {
		return fmt.Errorf("native installation root changed")
	}
	guard := ".symphony-package-retirement/" + target.receipt.ComponentID + "/" + target.receipt.Version
	data, err := nativeReadAt(rootFD, guard+"/state.json", 8192)
	if err != nil {
		return err
	}
	var state nativeRetirementState
	if err := decodeExact(data, &state); err != nil {
		return err
	}
	binary := target.receipt.Files[0]
	expected := nativeRetirementIdentity{target.receipt.ComponentID, target.receipt.Version, target.receipt.ReceiptDigest, binary.Digest, binary.Path, target.receiptPath, binary.Size}
	epoch, err := hex.DecodeString(state.Epoch)
	if err != nil || len(epoch) != 16 || state.Protocol != "symphony.package-retirement-state.v1" || state.Identity != expected || state.Phase != phase || state.RootDevice != uint64(pinned.Dev) || state.RootInode != pinned.Ino {
		return fmt.Errorf("native retirement evidence differs from the exact target")
	}
	parent, _, err := openRelativeParent(rootFD, guard+"/state.json", false)
	if err != nil {
		return err
	}
	defer unix.Close(parent)
	var directory unix.Stat_t
	if unix.Fstat(parent, &directory) != nil || uint64(directory.Dev) != state.GuardDevice || directory.Ino != state.GuardInode {
		return fmt.Errorf("retirement guard directory changed")
	}
	for _, lock := range []struct {
		name          string
		device, inode uint64
	}{{"admission.lock", state.AdmissionDevice, state.AdmissionInode}, {"use.lock", state.UseDevice, state.UseInode}} {
		file, err := nativeOpenAt(parent, lock.name)
		if err != nil {
			return err
		}
		var stat unix.Stat_t
		err = unix.Fstat(int(file.Fd()), &stat)
		file.Close()
		if err != nil || uint64(stat.Dev) != lock.device || stat.Ino != lock.inode || stat.Nlink != 1 {
			return fmt.Errorf("retirement guard lock changed")
		}
	}
	match, exists, err := nativePayloadMatches(rootFD, binary)
	if err != nil {
		return err
	}
	receiptData, receiptErr := nativeReadAt(rootFD, target.receiptPath, maxReceiptBytes)
	if phase == "retired" {
		if exists || !errors.Is(receiptErr, os.ErrNotExist) {
			return fmt.Errorf("native retirement retained package bytes or lacks exact absence evidence")
		}
		return nil
	}
	if !exists || !match || receiptErr != nil {
		return fmt.Errorf("native installation bytes or receipt differ")
	}
	var receipt receiptV2
	if decodeExact(receiptData, &receipt) != nil || !bytes.Equal(mustJSON(receipt), mustJSON(target.receipt)) {
		return fmt.Errorf("native installed receipt differs from the reviewed receipt")
	}
	return nil
}

func nativePayloadMatches(rootFD int, binary receiptV2File) (bool, bool, error) {
	file, err := nativeOpenAt(rootFD, binary.Path)
	if errors.Is(err, os.ErrNotExist) {
		return false, false, nil
	}
	if err != nil {
		return false, true, err
	}
	defer file.Close()
	hash := sha256.New()
	n, err := io.Copy(hash, io.LimitReader(file, int64(binary.Size)+1))
	return uint64(n) == binary.Size && "sha256:"+hex.EncodeToString(hash.Sum(nil)) == binary.Digest, true, err
}

func validateNativeMutationLock(locked *installLock) error {
	file, err := nativeOpenAt(int(locked.root.Fd()), packageMutationLock)
	if err != nil {
		return fmt.Errorf("integrity_fatal: native root lock is unavailable: %w", err)
	}
	defer file.Close()
	var pinned, named unix.Stat_t
	if unix.Fstat(int(locked.lock.Fd()), &pinned) != nil || unix.Fstat(int(file.Fd()), &named) != nil || pinned.Dev != named.Dev || pinned.Ino != named.Ino || pinned.Nlink != 1 || named.Nlink != 1 || pinned.Mode&0o077 != 0 {
		return fmt.Errorf("integrity_fatal: native root lock custody changed")
	}
	return nil
}
