//go:build darwin || linux

package knowledgelifecycle

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

const packageMutationLock = ".symphony-lifecycle.lock"

type installLock struct {
	root *os.File
	lock *os.File
}

func (l *installLock) close() {
	if l == nil {
		return
	}
	if l.lock != nil {
		_ = unix.Flock(int(l.lock.Fd()), unix.LOCK_UN)
		_ = l.lock.Close()
	}
	if l.root != nil {
		_ = l.root.Close()
	}
}

func installReceiptV2(sourceRoot, targetRoot, receiptPath string, receipt receiptV2) error {
	if sourceRoot == targetRoot || !safeAbsolutePath(sourceRoot) || !safeAbsolutePath(targetRoot) ||
		!safeRelativePath(receiptPath) {
		return fmt.Errorf("integrity_fatal: staged and target package roots are invalid")
	}
	if err := validateStagedPackage(sourceRoot, receipt); err != nil {
		return err
	}
	locked, err := lockInstallRoot(targetRoot, true)
	if err != nil {
		return fmt.Errorf("integrity_fatal: lock package target: %w", err)
	}
	defer locked.close()

	created := make([]string, 0, len(receipt.Files)+1)
	rollback := func() {
		for index := len(created) - 1; index >= 0; index-- {
			_ = unlinkRelative(int(locked.root.Fd()), created[index])
		}
	}
	for _, owned := range receipt.Files {
		source, err := openRelativeRegular(sourceRoot, owned.Path)
		if err != nil {
			rollback()
			return fmt.Errorf("integrity_fatal: staged file %s is unavailable: %w", owned.Path, err)
		}
		wasCreated, copyErr := installFileAt(
			int(locked.root.Fd()), owned.Path, source, owned.Size, owned.Digest, owned.Kind == "executable")
		_ = source.Close()
		if copyErr != nil {
			rollback()
			return fmt.Errorf("integrity_fatal: install owned file %s: %w", owned.Path, copyErr)
		}
		if wasCreated {
			created = append(created, owned.Path)
		}
	}
	receiptCreated, err := installBytesAt(
		int(locked.root.Fd()), receiptPath, mustJSON(receipt), 0o644, receipt.ReceiptDigest, true)
	if err != nil {
		rollback()
		return fmt.Errorf("integrity_fatal: install immutable receipt: %w", err)
	}
	if receiptCreated {
		created = append(created, receiptPath)
	}
	if err := unix.Fsync(int(locked.root.Fd())); err != nil {
		return fmt.Errorf("integrity_fatal: synchronize package root: %w", err)
	}
	return nil
}

func uninstallReceiptV2(targetRoot, sourceRoot, receiptPath string, receipt receiptV2) error {
	if sourceRoot == targetRoot || !safeAbsolutePath(sourceRoot) || !safeAbsolutePath(targetRoot) ||
		!safeRelativePath(receiptPath) {
		return fmt.Errorf("integrity_fatal: staged rollback and target package roots are invalid")
	}
	if err := validateStagedPackage(sourceRoot, receipt); err != nil {
		return err
	}
	locked, err := lockInstallRoot(targetRoot, false)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("integrity_fatal: lock package target: %w", err)
	}
	defer locked.close()

	// Validate every remaining owned file before deleting any of them. Missing
	// files are accepted only as forward-resumable evidence from an interrupted
	// prior attempt; a conflicting file always stops the operation.
	for _, owned := range receipt.Files {
		matches, exists, err := relativeFileMatches(int(locked.root.Fd()), owned.Path, owned.Size, owned.Digest)
		if err != nil || exists && !matches {
			return fmt.Errorf("integrity_fatal: installed file %s no longer matches its rollback proof", owned.Path)
		}
	}
	receiptMatches, receiptExists, err := relativeBytesMatch(
		int(locked.root.Fd()), receiptPath, mustJSON(receipt), receipt.ReceiptDigest, true)
	if err != nil || receiptExists && !receiptMatches {
		return fmt.Errorf("integrity_fatal: installed receipt no longer matches rollback proof")
	}

	for _, owned := range receipt.Files {
		if err := unlinkRelative(int(locked.root.Fd()), owned.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("integrity_fatal: remove owned file %s: %w", owned.Path, err)
		}
	}
	// The receipt is the commit marker: it is removed only after all owned files.
	if err := unlinkRelative(int(locked.root.Fd()), receiptPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("integrity_fatal: remove immutable receipt: %w", err)
	}
	if err := unix.Fsync(int(locked.root.Fd())); err != nil {
		return fmt.Errorf("integrity_fatal: synchronize package root: %w", err)
	}
	return nil
}

func validateStagedPackage(root string, receipt receiptV2) error {
	for _, owned := range receipt.Files {
		digest, size, err := hashTrustedRelative(root, owned.Path, maxObservedFileBytes(owned.Kind))
		if err != nil || size != owned.Size || digest != owned.Digest {
			return fmt.Errorf("integrity_fatal: staged file %s does not match immutable receipt", owned.Path)
		}
	}
	return nil
}

func lockInstallRoot(root string, create bool) (*installLock, error) {
	fd, err := openOrCreateAbsoluteDirectory(root, create)
	if err != nil {
		return nil, err
	}
	rootFile := os.NewFile(uintptr(fd), root)
	lockFD, err := unix.Openat(fd, packageMutationLock,
		unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		_ = rootFile.Close()
		return nil, err
	}
	lockFile := os.NewFile(uintptr(lockFD), packageMutationLock)
	if err := validateTrustedRegular(lockFD); err != nil {
		_ = lockFile.Close()
		_ = rootFile.Close()
		return nil, err
	}
	if err := unix.Flock(lockFD, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = lockFile.Close()
		_ = rootFile.Close()
		return nil, fmt.Errorf("package root is busy: %w", err)
	}
	return &installLock{root: rootFile, lock: lockFile}, nil
}

func openOrCreateAbsoluteDirectory(path string, create bool) (int, error) {
	clean := filepath.Clean(path)
	if !safeAbsolutePath(clean) || clean == "/" {
		return -1, fmt.Errorf("unsafe install root")
	}
	current, err := unix.Open("/", unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
	if err != nil {
		return -1, err
	}
	for _, component := range strings.Split(strings.TrimPrefix(clean, "/"), "/") {
		next, openErr := unix.Openat(current, component, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
		if errors.Is(openErr, syscall.ENOENT) && create {
			if makeErr := unix.Mkdirat(current, component, 0o755); makeErr != nil && !errors.Is(makeErr, syscall.EEXIST) {
				_ = unix.Close(current)
				return -1, makeErr
			}
			if syncErr := unix.Fsync(current); syncErr != nil {
				_ = unix.Close(current)
				return -1, syncErr
			}
			next, openErr = unix.Openat(current, component, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
		}
		_ = unix.Close(current)
		if openErr != nil {
			return -1, openErr
		}
		if err := validateTrustedDirectory(next); err != nil {
			_ = unix.Close(next)
			return -1, err
		}
		current = next
	}
	return current, nil
}

func openRelativeRegular(root, relative string) (*os.File, error) {
	rootFD, err := unix.Open(root, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
	if err != nil {
		return nil, err
	}
	if err := validateTrustedDirectory(rootFD); err != nil {
		_ = unix.Close(rootFD)
		return nil, err
	}
	parent, leaf, err := openRelativeParent(rootFD, relative, false)
	_ = unix.Close(rootFD)
	if err != nil {
		return nil, err
	}
	defer unix.Close(parent)
	fd, err := unix.Openat(parent, leaf, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	if err := validateTrustedRegular(fd); err != nil {
		_ = unix.Close(fd)
		return nil, err
	}
	return os.NewFile(uintptr(fd), relative), nil
}

func openRelativeParent(rootFD int, relative string, create bool) (int, string, error) {
	if !safeRelativePath(relative) {
		return -1, "", fmt.Errorf("unsafe relative path")
	}
	components := strings.Split(relative, "/")
	current, err := unix.Dup(rootFD)
	if err != nil {
		return -1, "", err
	}
	for _, component := range components[:len(components)-1] {
		next, openErr := unix.Openat(current, component, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
		if errors.Is(openErr, syscall.ENOENT) && create {
			if makeErr := unix.Mkdirat(current, component, 0o755); makeErr != nil && !errors.Is(makeErr, syscall.EEXIST) {
				_ = unix.Close(current)
				return -1, "", makeErr
			}
			if syncErr := unix.Fsync(current); syncErr != nil {
				_ = unix.Close(current)
				return -1, "", syncErr
			}
			next, openErr = unix.Openat(current, component, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
		}
		_ = unix.Close(current)
		if openErr != nil {
			return -1, "", openErr
		}
		if err := validateTrustedDirectory(next); err != nil {
			_ = unix.Close(next)
			return -1, "", err
		}
		current = next
	}
	return current, components[len(components)-1], nil
}

func installFileAt(rootFD int, relative string, source *os.File, size uint64, digest string, executable bool) (bool, error) {
	mode := uint32(0o644)
	if executable {
		mode = 0o755
	}
	parent, leaf, err := openRelativeParent(rootFD, relative, true)
	if err != nil {
		return false, err
	}
	defer unix.Close(parent)
	if matches, exists, err := fileAtMatches(parent, leaf, size, digest); err != nil || exists {
		if err != nil {
			return false, err
		}
		if !matches {
			return false, fmt.Errorf("target already exists with different content")
		}
		return false, nil
	}
	return writeTempAndLink(parent, leaf, mode, size, digest, source, nil)
}

func installBytesAt(rootFD int, relative string, data []byte, mode uint32, digest string, receiptDigest bool) (bool, error) {
	parent, leaf, err := openRelativeParent(rootFD, relative, true)
	if err != nil {
		return false, err
	}
	defer unix.Close(parent)
	matches, exists, err := bytesAtMatch(parent, leaf, data, digest, receiptDigest)
	if err != nil {
		return false, err
	}
	if exists {
		if !matches {
			return false, fmt.Errorf("target already exists with different content")
		}
		return false, nil
	}
	return writeTempAndLink(parent, leaf, mode, uint64(len(data)), digest, nil, data)
}

func writeTempAndLink(parent int, leaf string, mode uint32, expectedSize uint64, expectedDigest string, source *os.File, data []byte) (bool, error) {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return false, err
	}
	temporary := "." + leaf + ".tmp-" + hex.EncodeToString(randomBytes)
	fd, err := unix.Openat(parent, temporary, unix.O_CREAT|unix.O_EXCL|unix.O_WRONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, mode)
	if err != nil {
		return false, err
	}
	file := os.NewFile(uintptr(fd), temporary)
	cleanup := func() {
		_ = file.Close()
		_ = unix.Unlinkat(parent, temporary, 0)
	}
	hash := sha256.New()
	writer := io.MultiWriter(file, hash)
	var written int64
	if source != nil {
		if _, err := source.Seek(0, io.SeekStart); err != nil {
			cleanup()
			return false, err
		}
		written, err = io.Copy(writer, io.LimitReader(source, int64(expectedSize)+1))
		if err != nil {
			cleanup()
			return false, err
		}
	} else {
		count, err := writer.Write(data)
		written = int64(count)
		if err != nil {
			cleanup()
			return false, err
		}
	}
	actualDigest := "sha256:" + hex.EncodeToString(hash.Sum(nil))
	if uint64(written) != expectedSize || source != nil && actualDigest != expectedDigest {
		cleanup()
		return false, fmt.Errorf("source changed while copying")
	}
	if err := file.Sync(); err != nil {
		cleanup()
		return false, err
	}
	if err := file.Close(); err != nil {
		_ = unix.Unlinkat(parent, temporary, 0)
		return false, err
	}
	if err := unix.Linkat(parent, temporary, parent, leaf, 0); err != nil {
		_ = unix.Unlinkat(parent, temporary, 0)
		if errors.Is(err, syscall.EEXIST) {
			return false, fmt.Errorf("target appeared concurrently")
		}
		return false, err
	}
	if err := unix.Unlinkat(parent, temporary, 0); err != nil {
		return true, err
	}
	if err := unix.Fsync(parent); err != nil {
		return true, err
	}
	return true, nil
}

func fileAtMatches(parent int, leaf string, size uint64, digest string) (bool, bool, error) {
	fd, err := unix.Openat(parent, leaf, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if errors.Is(err, syscall.ENOENT) {
		return false, false, nil
	}
	if err != nil {
		return false, true, err
	}
	file := os.NewFile(uintptr(fd), leaf)
	defer file.Close()
	if err := validateTrustedRegular(fd); err != nil {
		return false, true, err
	}
	hash := sha256.New()
	written, err := io.Copy(hash, io.LimitReader(file, int64(size)+1))
	if err != nil {
		return false, true, err
	}
	actual := "sha256:" + hex.EncodeToString(hash.Sum(nil))
	return uint64(written) == size && actual == digest, true, nil
}

func bytesAtMatch(parent int, leaf string, data []byte, digest string, receiptDigest bool) (bool, bool, error) {
	fd, err := unix.Openat(parent, leaf, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if errors.Is(err, syscall.ENOENT) {
		return false, false, nil
	}
	if err != nil {
		return false, true, err
	}
	file := os.NewFile(uintptr(fd), leaf)
	defer file.Close()
	if err := validateTrustedRegular(fd); err != nil {
		return false, true, err
	}
	actual, err := io.ReadAll(io.LimitReader(file, int64(len(data))+1))
	if err != nil {
		return false, true, err
	}
	if receiptDigest {
		var decoded receiptV2
		if decodeExact(actual, &decoded) != nil {
			return false, true, nil
		}
		return decoded.ReceiptDigest == digest && string(actual) == string(data), true, nil
	}
	return string(actual) == string(data), true, nil
}

func relativeFileMatches(rootFD int, relative string, size uint64, digest string) (bool, bool, error) {
	parent, leaf, err := openRelativeParent(rootFD, relative, false)
	if errors.Is(err, syscall.ENOENT) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	defer unix.Close(parent)
	return fileAtMatches(parent, leaf, size, digest)
}

func relativeBytesMatch(rootFD int, relative string, data []byte, digest string, receiptDigest bool) (bool, bool, error) {
	parent, leaf, err := openRelativeParent(rootFD, relative, false)
	if errors.Is(err, syscall.ENOENT) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	defer unix.Close(parent)
	return bytesAtMatch(parent, leaf, data, digest, receiptDigest)
}

func unlinkRelative(rootFD int, relative string) error {
	parent, leaf, err := openRelativeParent(rootFD, relative, false)
	if errors.Is(err, syscall.ENOENT) {
		return os.ErrNotExist
	}
	if err != nil {
		return err
	}
	defer unix.Close(parent)
	if err := unix.Unlinkat(parent, leaf, 0); err != nil {
		if errors.Is(err, syscall.ENOENT) {
			return os.ErrNotExist
		}
		return err
	}
	return unix.Fsync(parent)
}
