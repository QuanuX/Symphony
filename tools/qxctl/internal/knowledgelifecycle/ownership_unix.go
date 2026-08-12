//go:build darwin || linux

package knowledgelifecycle

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func (s *OwnershipStore) snapshotOwnership() (OwnershipSnapshot, error) {
	var snapshot OwnershipSnapshot
	err := s.withOwnershipLock(false, func(_ int, exists bool, registry OwnershipRegistry) error {
		snapshot = ownershipSnapshot(exists, registry)
		return nil
	})
	return snapshot, err
}

func (s *OwnershipStore) withOwnershipLock(
	create bool,
	operation func(rootFD int, exists bool, registry OwnershipRegistry) error,
) error {
	locked, err := lockInstallRoot(s.installRoot, create)
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENOENT) {
		return operation(-1, false, OwnershipRegistry{})
	}
	if err != nil {
		return fmt.Errorf("lock shared installation root: %w", err)
	}
	defer locked.close()
	registry, exists, err := readOwnershipRegistryAt(int(locked.root.Fd()), s.installRoot)
	if err != nil {
		return err
	}
	if exists {
		if create {
			if err := ensureOwnershipFenceAt(int(locked.root.Fd())); err != nil {
				return fmt.Errorf("repair ownership compatibility fence: %w", err)
			}
		} else if err := validateOwnershipFenceAt(int(locked.root.Fd())); err != nil {
			return fmt.Errorf("validate ownership compatibility fence: %w", err)
		}
	} else if !create {
		fenced, err := ownershipFenceStateAt(int(locked.root.Fd()))
		if err != nil {
			return fmt.Errorf("validate orphaned ownership compatibility fence: %w", err)
		}
		if fenced {
			return fmt.Errorf("ownership compatibility fence exists without its registry; run ownership reconcile")
		}
	}
	return operation(int(locked.root.Fd()), exists, registry)
}

func validateOwnershipFenceAt(rootFD int) error {
	exists, err := ownershipFenceStateAt(rootFD)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("exact ownership compatibility fence is absent")
	}
	return nil
}

func ownershipFenceStateAt(rootFD int) (bool, error) {
	parent, leaf, err := openRelativeParent(rootFD, ownershipFenceRelative, false)
	if errors.Is(err, syscall.ENOENT) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer unix.Close(parent)
	matches, exists, err := bytesAtMatch(parent, leaf, []byte(ownershipFenceDocument), "", false)
	if err != nil {
		return exists, err
	}
	if exists && !matches {
		return true, fmt.Errorf("ownership compatibility fence is invalid")
	}
	return exists, nil
}

func ensureOwnershipFenceAt(rootFD int) error {
	digest := sha256.Sum256([]byte(ownershipFenceDocument))
	_, err := installBytesAt(rootFD, ownershipFenceRelative, []byte(ownershipFenceDocument), 0o644,
		"sha256:"+hex.EncodeToString(digest[:]), false)
	return err
}

func readOwnershipRegistryAt(rootFD int, installRoot string) (OwnershipRegistry, bool, error) {
	if rootFD < 0 {
		return OwnershipRegistry{}, false, nil
	}
	fd, err := unix.Openat(rootFD, ownershipRegistryFile,
		unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if errors.Is(err, syscall.ENOENT) {
		return OwnershipRegistry{}, false, nil
	}
	if err != nil {
		return OwnershipRegistry{}, false, fmt.Errorf("open ownership registry: %w", err)
	}
	file := os.NewFile(uintptr(fd), ownershipRegistryFile)
	defer file.Close()
	if err := validateTrustedRegular(fd); err != nil {
		return OwnershipRegistry{}, false, err
	}
	data, err := io.ReadAll(io.LimitReader(file, maxOwnershipBytes+1))
	if err != nil || len(data) > maxOwnershipBytes {
		return OwnershipRegistry{}, false, fmt.Errorf("ownership registry read failed or exceeded its bound")
	}
	var registry OwnershipRegistry
	if err := decodeExact(data, &registry); err != nil {
		return OwnershipRegistry{}, false, fmt.Errorf("decode ownership registry: %w", err)
	}
	if err := validateOwnershipRegistry(registry, installRoot); err != nil {
		return OwnershipRegistry{}, false, err
	}
	return registry, true, nil
}

func writeOwnershipRegistryAt(rootFD int, registry OwnershipRegistry) error {
	if rootFD < 0 {
		return fmt.Errorf("shared installation root is absent")
	}
	// The receipt-layout fence is committed first. Lifecycle clients that
	// predate root ownership already fail closed on its unsupported protocol,
	// so an interrupted first registry write cannot reopen legacy mutation.
	if err := ensureOwnershipFenceAt(rootFD); err != nil {
		return fmt.Errorf("commit ownership compatibility fence: %w", err)
	}
	data, err := json.Marshal(registry)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if len(data) > maxOwnershipBytes {
		return fmt.Errorf("ownership registry exceeds its bound")
	}
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return err
	}
	temporary := ".ownership.tmp-" + hex.EncodeToString(random)
	fd, err := unix.Openat(rootFD, temporary,
		unix.O_CREAT|unix.O_EXCL|unix.O_WRONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(fd), temporary)
	cleanup := func() {
		_ = file.Close()
		_ = unix.Unlinkat(rootFD, temporary, 0)
	}
	if err := validateTrustedRegular(fd); err != nil {
		cleanup()
		return err
	}
	if _, err := file.Write(data); err != nil {
		cleanup()
		return err
	}
	if err := file.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := file.Close(); err != nil {
		_ = unix.Unlinkat(rootFD, temporary, 0)
		return err
	}
	if err := unix.Renameat(rootFD, temporary, rootFD, ownershipRegistryFile); err != nil {
		_ = unix.Unlinkat(rootFD, temporary, 0)
		return err
	}
	return unix.Fsync(rootFD)
}

func ownershipInstallAllowedAt(rootFD int, installRoot string, owner *OwnershipStore, componentID, receiptDigest string) error {
	if err := validateOwnershipFenceAt(rootFD); err != nil {
		return fmt.Errorf("dependency_wait: ownership compatibility fence is not ready: %w", err)
	}
	registry, exists, err := readOwnershipRegistryAt(rootFD, installRoot)
	if err != nil {
		return err
	}
	if !exists || registry.EnforcementState != "enforced" {
		return fmt.Errorf("dependency_wait: shared-root ownership adoption is incomplete")
	}
	for _, claim := range registry.Claims {
		if claim.ReceiptDigest == receiptDigest && owner.ownsClaim(claim) &&
			claim.ComponentID == componentID && claim.Disposition == "retained" {
			return nil
		}
	}
	return fmt.Errorf("dependency_wait: exact retained shared-root claim is absent")
}

func ownershipReclaimAllowedAt(rootFD int, installRoot string, owner *OwnershipStore, componentID, receiptDigest string) (OwnershipRegistry, error) {
	if err := validateOwnershipFenceAt(rootFD); err != nil {
		return OwnershipRegistry{}, fmt.Errorf("dependency_wait: ownership compatibility fence is not ready: %w", err)
	}
	registry, exists, err := readOwnershipRegistryAt(rootFD, installRoot)
	if err != nil {
		return OwnershipRegistry{}, err
	}
	if !exists || registry.EnforcementState != "enforced" {
		return OwnershipRegistry{}, fmt.Errorf("dependency_wait: shared-root ownership adoption is incomplete")
	}
	allowed := containsString(registry.ReleasedReceiptDigests, receiptDigest)
	for _, claim := range registry.Claims {
		if claim.ReceiptDigest != receiptDigest {
			continue
		}
		if owner.ownsClaim(claim) && claim.ComponentID == componentID && claim.Disposition == "retiring" {
			allowed = true
			continue
		}
		// A retiring claim is positive evidence that another control domain has
		// also released retention. Once every claim for the exact receipt is
		// retiring, any one of the releasing domains may perform the single
		// serialized physical reclamation on behalf of all of them.
		if claim.ClaimKind == "profile" && claim.Disposition == "retiring" {
			continue
		}
		return OwnershipRegistry{}, fmt.Errorf("dependency_wait: receipt remains claimed by %s", claim.ClaimID)
	}
	if !allowed {
		return OwnershipRegistry{}, fmt.Errorf("dependency_wait: exact shared-root release evidence is absent")
	}
	return registry, nil
}

func commitOwnershipReclaimedAt(rootFD int, registry OwnershipRegistry, owner *OwnershipStore, receiptDigest string) error {
	next := make([]OwnershipClaim, 0, len(registry.Claims))
	for _, claim := range registry.Claims {
		if claim.ReceiptDigest == receiptDigest && claim.ClaimKind == "profile" && claim.Disposition == "retiring" {
			continue
		}
		next = append(next, claim)
	}
	registry.Claims = next
	registry.ObservedReceiptDigests = removeDigest(registry.ObservedReceiptDigests, receiptDigest)
	registry.ReleasedReceiptDigests = removeDigest(registry.ReleasedReceiptDigests, receiptDigest)
	if registry.Generation >= 9007199254740991 {
		return fmt.Errorf("ownership registry generation is exhausted")
	}
	registry.Generation++
	registry.PreviousOwnershipRegistryDigest = stringPointer(registry.OwnershipRegistryDigest)
	registry.UpdatedAt = owner.now().UTC().Truncate(time.Second).Format(time.RFC3339)
	if err := finalizeOwnershipRegistry(&registry); err != nil {
		return err
	}
	return writeOwnershipRegistryAt(rootFD, registry)
}
