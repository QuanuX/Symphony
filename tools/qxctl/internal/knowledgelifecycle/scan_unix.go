//go:build darwin || linux

package knowledgelifecycle

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

const (
	maxReceiptModules  = 4096
	maxReceiptVersions = 4096
)

func scanReceiptCandidates(roots []string) ([]receiptCandidate, error) {
	result := make([]receiptCandidate, 0)
	for _, root := range roots {
		rootCandidates, err := scanReceiptRoot(root)
		if err != nil {
			return nil, fmt.Errorf("scan configured root %s: %w", root, err)
		}
		result = append(result, rootCandidates...)
		if len(result) > 4096 {
			return nil, fmt.Errorf("receipt inventory exceeds 4096 entries")
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].root != result[j].root {
			return result[i].root < result[j].root
		}
		return result[i].relativePath < result[j].relativePath
	})
	return result, nil
}

func scanReceiptRoot(root string) ([]receiptCandidate, error) {
	info, err := os.Lstat(root)
	if errors.Is(err, os.ErrNotExist) {
		return []receiptCandidate{}, nil
	}
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("configured root is not a no-follow directory")
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil || filepath.Clean(resolved) != filepath.Clean(root) {
		return nil, fmt.Errorf("configured root changes through symbolic-link resolution")
	}
	rootFD, err := unix.Open(root, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
	if err != nil {
		return nil, err
	}
	if err := validateTrustedDirectory(rootFD); err != nil {
		_ = unix.Close(rootFD)
		return nil, err
	}
	receiptsFD, exists, err := openTrustedDirectoryPath(rootFD, []string{"share", "symphony", "receipts"})
	_ = unix.Close(rootFD)
	if err != nil || !exists {
		return nil, err
	}
	defer unix.Close(receiptsFD)
	modules, err := readDirectoryNames(receiptsFD, maxReceiptModules)
	if err != nil {
		return nil, err
	}
	result := make([]receiptCandidate, 0)
	for _, module := range modules {
		if strings.HasPrefix(module, ".") {
			continue
		}
		if !safeToken(module, 256) {
			return nil, fmt.Errorf("receipt module directory has an invalid name")
		}
		moduleFD, err := openTrustedDirectoryAt(receiptsFD, module)
		if err != nil {
			return nil, fmt.Errorf("open receipt module directory %q: %w", module, err)
		}
		versions, readErr := readDirectoryNames(moduleFD, maxReceiptVersions)
		if readErr != nil {
			_ = unix.Close(moduleFD)
			return nil, readErr
		}
		for _, version := range versions {
			if strings.HasPrefix(version, ".") {
				continue
			}
			if !safeVersion(version) {
				_ = unix.Close(moduleFD)
				return nil, fmt.Errorf("receipt version directory has an invalid name")
			}
			versionFD, err := openTrustedDirectoryAt(moduleFD, version)
			if err != nil {
				_ = unix.Close(moduleFD)
				return nil, fmt.Errorf("open receipt version directory %q: %w", version, err)
			}
			relative := filepath.ToSlash(filepath.Join("share", "symphony", "receipts", module, version, "install-receipt.json"))
			data, exists, readable := readReceiptAt(versionFD)
			_ = unix.Close(versionFD)
			if !exists {
				continue
			}
			result = append(result, receiptCandidate{
				root: root, relativePath: relative, module: module, version: version,
				data: data, readable: readable,
			})
		}
		_ = unix.Close(moduleFD)
	}
	return result, nil
}

func openTrustedDirectoryPath(start int, components []string) (int, bool, error) {
	current, err := unix.Dup(start)
	if err != nil {
		return -1, false, err
	}
	for _, component := range components {
		next, openErr := unix.Openat(current, component, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
		_ = unix.Close(current)
		if errors.Is(openErr, syscall.ENOENT) {
			return -1, false, nil
		}
		if openErr != nil {
			return -1, false, openErr
		}
		if err := validateTrustedDirectory(next); err != nil {
			_ = unix.Close(next)
			return -1, false, err
		}
		current = next
	}
	return current, true, nil
}

func openTrustedDirectoryAt(parent int, name string) (int, error) {
	fd, err := unix.Openat(parent, name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
	if err != nil {
		return -1, err
	}
	if err := validateTrustedDirectory(fd); err != nil {
		_ = unix.Close(fd)
		return -1, err
	}
	return fd, nil
}

func readDirectoryNames(fd, maximum int) ([]string, error) {
	duplicate, err := unix.Dup(fd)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(duplicate), "receipt-directory")
	defer file.Close()
	entries, err := file.ReadDir(maximum + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if len(entries) > maximum {
		return nil, fmt.Errorf("receipt directory exceeds %d entries", maximum)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}

func readReceiptAt(directory int) ([]byte, bool, bool) {
	fd, err := unix.Openat(directory, "install-receipt.json", unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if errors.Is(err, syscall.ENOENT) {
		return nil, false, false
	}
	if err != nil {
		return nil, true, false
	}
	file := os.NewFile(uintptr(fd), "install-receipt.json")
	defer file.Close()
	if err := validateTrustedRegular(fd); err != nil {
		return nil, true, false
	}
	data, err := io.ReadAll(io.LimitReader(file, maxReceiptBytes+1))
	if err != nil || len(data) == 0 || len(data) > maxReceiptBytes {
		return nil, true, false
	}
	return data, true, true
}

func validateTrustedDirectory(fd int) error {
	var status unix.Stat_t
	if err := unix.Fstat(fd, &status); err != nil {
		return err
	}
	if status.Mode&unix.S_IFMT != unix.S_IFDIR || (status.Uid != 0 && status.Uid != uint32(os.Geteuid())) || status.Mode&0o022 != 0 {
		return fmt.Errorf("installed directory is not trusted-owner controlled")
	}
	return nil
}

func validateTrustedRegular(fd int) error {
	var status unix.Stat_t
	if err := unix.Fstat(fd, &status); err != nil {
		return err
	}
	if status.Mode&unix.S_IFMT != unix.S_IFREG || (status.Uid != 0 && status.Uid != uint32(os.Geteuid())) || status.Mode&0o022 != 0 {
		return fmt.Errorf("installed file is not trusted-owner controlled")
	}
	return nil
}

func hashTrustedRelative(root, relative string, maximum int64) (string, uint64, error) {
	if !safeRelativePath(relative) {
		return "", 0, fmt.Errorf("installed path is unsafe")
	}
	components := strings.Split(relative, "/")
	current, err := unix.Open(root, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
	if err != nil {
		return "", 0, err
	}
	if err := validateTrustedDirectory(current); err != nil {
		_ = unix.Close(current)
		return "", 0, err
	}
	for _, component := range components[:len(components)-1] {
		next, openErr := unix.Openat(current, component, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
		_ = unix.Close(current)
		if openErr != nil {
			return "", 0, openErr
		}
		if err := validateTrustedDirectory(next); err != nil {
			_ = unix.Close(next)
			return "", 0, err
		}
		current = next
	}
	fd, err := unix.Openat(current, components[len(components)-1], unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	_ = unix.Close(current)
	if err != nil {
		return "", 0, err
	}
	file := os.NewFile(uintptr(fd), components[len(components)-1])
	defer file.Close()
	if err := validateTrustedRegular(fd); err != nil {
		return "", 0, err
	}
	info, err := file.Stat()
	if err != nil || info.Size() < 0 || info.Size() > maximum {
		return "", 0, fmt.Errorf("installed file exceeds its size bound")
	}
	hash := sha256.New()
	written, err := io.Copy(hash, io.LimitReader(file, maximum+1))
	if err != nil || written != info.Size() {
		return "", 0, fmt.Errorf("installed file changed while hashing")
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), uint64(written), nil
}

func hashRegularFile(path string, maximum int64) (string, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if err != nil {
		return "", err
	}
	file := os.NewFile(uintptr(fd), path)
	defer file.Close()
	var status unix.Stat_t
	if err := unix.Fstat(fd, &status); err != nil || status.Mode&unix.S_IFMT != unix.S_IFREG || status.Size < 0 || status.Size > maximum {
		return "", fmt.Errorf("executable is not a bounded regular file")
	}
	hash := sha256.New()
	written, err := io.Copy(hash, io.LimitReader(file, maximum+1))
	if err != nil || written != status.Size {
		return "", fmt.Errorf("executable changed while hashing")
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func kernelABI() string {
	var name unix.Utsname
	if err := unix.Uname(&name); err != nil {
		return "uname:unavailable"
	}
	bytes := make([]byte, 0, len(name.Release))
	for _, value := range name.Release {
		if value == 0 {
			break
		}
		bytes = append(bytes, byte(value))
	}
	digest := sha256.Sum256(bytes)
	return "uname:" + hex.EncodeToString(digest[:])
}
