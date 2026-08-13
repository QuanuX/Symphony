//go:build darwin || linux

package foundationlifecycle

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

type attemptStore struct {
	directory string
	file      *os.File
}

func openAttemptStore(root, topsID, surface string, exclusive bool) (*attemptStore, error) {
	directory := filepath.Join(root, topsID, surface)
	if err := ensurePrivatePath(directory); err != nil {
		return nil, err
	}
	lockPath := filepath.Join(directory, "lifecycle.lock")
	fd, err := unix.Open(lockPath, unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open SSIAG lifecycle lock: %w", err)
	}
	file := os.NewFile(uintptr(fd), lockPath)
	fail := func(value error) (*attemptStore, error) { _ = file.Close(); return nil, value }
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil || stat.Mode&unix.S_IFMT != unix.S_IFREG || stat.Uid != uint32(os.Geteuid()) {
		return fail(fmt.Errorf("SSIAG lifecycle lock is unsafe"))
	}
	mode := unix.LOCK_SH
	if exclusive {
		mode = unix.LOCK_EX
	}
	if err := unix.Flock(fd, mode); err != nil {
		return fail(fmt.Errorf("lock SSIAG lifecycle state: %w", err))
	}
	return &attemptStore{directory: directory, file: file}, nil
}

func (store *attemptStore) close() error {
	if store == nil || store.file == nil {
		return nil
	}
	fd := int(store.file.Fd())
	err := errors.Join(unix.Flock(fd, unix.LOCK_UN), store.file.Close())
	store.file = nil
	return err
}

func (store *attemptStore) read() (Attempt, bool, error) {
	path := filepath.Join(store.directory, "attempt.json")
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if errors.Is(err, syscall.ENOENT) {
		return Attempt{}, false, nil
	}
	if err != nil {
		return Attempt{}, false, fmt.Errorf("open SSIAG lifecycle attempt: %w", err)
	}
	file := os.NewFile(uintptr(fd), path)
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxRequestBytes {
		return Attempt{}, false, fmt.Errorf("SSIAG lifecycle attempt is unsafe or unbounded")
	}
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var attempt Attempt
	if err := decoder.Decode(&attempt); err != nil {
		return Attempt{}, false, fmt.Errorf("decode SSIAG lifecycle attempt: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return Attempt{}, false, fmt.Errorf("SSIAG lifecycle attempt contains trailing data")
	}
	if err := validateAttempt(attempt); err != nil {
		return Attempt{}, false, err
	}
	return attempt, true, nil
}

func (store *attemptStore) readPlan() (Plan, error) {
	path := filepath.Join(store.directory, "plan.json")
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return Plan{}, fmt.Errorf("open SSIAG lifecycle recovery plan: %w", err)
	}
	file := os.NewFile(uintptr(fd), path)
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxRequestBytes {
		return Plan{}, fmt.Errorf("SSIAG lifecycle recovery plan is unsafe")
	}
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var plan Plan
	if err := decoder.Decode(&plan); err != nil {
		return Plan{}, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return Plan{}, fmt.Errorf("SSIAG lifecycle recovery plan contains trailing data")
	}
	if err := plan.validate(); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

func (store *attemptStore) writePlan(plan Plan) error {
	data, err := json.Marshal(plan)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	temp, err := os.CreateTemp(store.directory, ".plan-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, filepath.Join(store.directory, "plan.json")); err != nil {
		return err
	}
	dir, err := os.Open(store.directory)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func (store *attemptStore) write(attempt *Attempt) error {
	var err error
	attempt.AttemptDigest, err = digestWithout(*attempt, "attempt_digest")
	if err != nil {
		return err
	}
	data, err := json.Marshal(*attempt)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	temp, err := os.CreateTemp(store.directory, ".attempt-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, filepath.Join(store.directory, "attempt.json")); err != nil {
		return err
	}
	dir, err := os.Open(store.directory)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func ensurePrivatePath(path string) error {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		return fmt.Errorf("SSIAG lifecycle path must be absolute")
	}
	var missing []string
	for current := path; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err == nil {
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("unsafe SSIAG lifecycle directory component")
			}
			break
		}
		if !os.IsNotExist(err) {
			return err
		}
		missing = append(missing, current)
		if filepath.Dir(current) == current {
			return fmt.Errorf("no safe SSIAG lifecycle directory ancestor")
		}
	}
	for index := len(missing) - 1; index >= 0; index-- {
		if err := os.Mkdir(missing[index], 0o700); err != nil && !os.IsExist(err) {
			return err
		}
		if err := os.Chmod(missing[index], 0o700); err != nil {
			return err
		}
	}
	return nil
}

func validateAttempt(attempt Attempt) error {
	if attempt.Protocol != AttemptProtocol || attempt.FormatVersion != 1 || attempt.Component != "ssiag" ||
		!oneOf(attempt.Surface, "enrollment", "supervisor") || !oneOf(attempt.Scope, "user", "system") || !validTOPSID(attempt.TOPSID) ||
		!validToken(attempt.OperationID) || !validTOPSID(attempt.RequestID) || !validTOPSID(attempt.CorrelationID) ||
		!oneOf(attempt.Phase, "prepared", "mutating", "observing", "mutation_closed", "reconciling_audit", "closed", "recovery_required") ||
		!validDigest(attempt.PlanDigest) || !validDigestOrAbsent(attempt.PriorStateDigest) || !validDigest(attempt.BinaryDigest) || !validDigest(attempt.InstallEvidenceDigest) ||
		!oneOf(attempt.AuditState, "pending", "committed", "audit_deferred", "reconciled") || !validSTSC(attempt.StartedAt) || !validSTSC(attempt.UpdatedAt) || !validDigest(attempt.AttemptDigest) {
		return fmt.Errorf("SSIAG lifecycle attempt identity is invalid")
	}
	want, err := digestWithout(attempt, "attempt_digest")
	if err != nil || want != attempt.AttemptDigest {
		return fmt.Errorf("SSIAG lifecycle attempt digest mismatch")
	}
	return nil
}
