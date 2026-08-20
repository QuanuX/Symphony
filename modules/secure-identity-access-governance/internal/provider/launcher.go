package provider

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	ssiagpaths "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
)

type processLauncher struct{}

func (processLauncher) ObserveReadiness(ctx context.Context, declaration ExecutableTrust) (AdapterReadinessObservation, error) {
	executable, cleanup, err := stageVerifiedExecutable(declaration)
	if err != nil {
		return AdapterReadinessObservation{}, err
	}
	defer cleanup()
	command := exec.CommandContext(ctx, executable, "readiness")
	command.Dir = "/"
	command.Env = []string{"PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C"}
	stdout := &boundedBuffer{limit: maximumControlBytes}
	stderr := &boundedBuffer{limit: 4096}
	command.Stdout, command.Stderr = stdout, stderr
	configureProviderProcess(command)
	command.WaitDelay = 250 * time.Millisecond
	if err := command.Run(); err != nil {
		terminateProviderProcess(command)
		if ctx.Err() != nil {
			return AdapterReadinessObservation{}, ctx.Err()
		}
		return AdapterReadinessObservation{}, fmt.Errorf("provider readiness process failed")
	}
	if stdout.overflow || stdout.Len() == 0 || validateJSONMembers(stdout.Bytes()) != nil || validateReadinessShape(stdout.Bytes()) != nil {
		return AdapterReadinessObservation{}, fmt.Errorf("provider readiness response is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
	decoder.DisallowUnknownFields()
	var observation AdapterReadinessObservation
	if err := decoder.Decode(&observation); err != nil {
		return AdapterReadinessObservation{}, fmt.Errorf("decode provider readiness response: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return AdapterReadinessObservation{}, fmt.Errorf("provider emitted more than one readiness response")
	}
	return observation, nil
}

func (processLauncher) Exchange(ctx context.Context, declaration ExecutableTrust, request ControlRequest) (ControlResponse, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return ControlResponse{}, fmt.Errorf("encode provider control request: %w", err)
	}
	if len(payload)+1 > maximumControlBytes {
		return ControlResponse{}, fmt.Errorf("provider control request exceeds limit")
	}
	payload = append(payload, '\n')
	executable, cleanup, err := stageVerifiedExecutable(declaration)
	if err != nil {
		return ControlResponse{}, err
	}
	defer cleanup()
	command := exec.CommandContext(ctx, executable, "serve")
	command.Dir = "/"
	command.Env = []string{"PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C"}
	command.Stdin = bytes.NewReader(payload)
	stdout := &boundedBuffer{limit: maximumControlBytes}
	stderr := &boundedBuffer{limit: 4096}
	command.Stdout, command.Stderr = stdout, stderr
	configureProviderProcess(command)
	command.WaitDelay = 250 * time.Millisecond
	if err := command.Run(); err != nil {
		terminateProviderProcess(command)
		if ctx.Err() != nil {
			return ControlResponse{}, ctx.Err()
		}
		return ControlResponse{}, fmt.Errorf("provider control process failed")
	}
	if stdout.overflow {
		return ControlResponse{}, fmt.Errorf("provider control response exceeds limit")
	}
	responsePayload := stdout.Bytes()
	if len(responsePayload) == 0 || len(responsePayload) > maximumControlBytes || validateJSONMembers(responsePayload) != nil || validateControlResponseShape(responsePayload) != nil {
		return ControlResponse{}, fmt.Errorf("provider control response is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(responsePayload))
	decoder.DisallowUnknownFields()
	var response ControlResponse
	if err := decoder.Decode(&response); err != nil {
		return ControlResponse{}, fmt.Errorf("decode provider control response: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return ControlResponse{}, fmt.Errorf("provider emitted more than one response")
	}
	return response, nil
}

// stageVerifiedExecutable closes the receipt-check-to-exec replacement gap.
// The child executes a private copy of the exact bytes whose digest the
// protected trust declaration selected, not a path that can be swapped after
// validation.
func stageVerifiedExecutable(declaration ExecutableTrust) (string, func(), error) {
	if strings.Contains(filepath.ToSlash(declaration.ExecutablePath), "/"+macOSKeychainBundleName+"/Contents/MacOS/") {
		scope := ssiagpaths.ScopeUser
		if declaration.OwnerUID == 0 {
			scope = ssiagpaths.ScopeSystem
		}
		pkg, err := inspectAdapterPackage(declaration, scope)
		if err != nil {
			return "", func() {}, fmt.Errorf("verify provider bundle for staging: %w", err)
		}
		return stageVerifiedBundle(declaration, pkg)
	}
	source, err := openProviderFile(declaration.ExecutablePath)
	if err != nil {
		return "", func() {}, fmt.Errorf("open verified provider executable: %w", err)
	}
	defer source.Close()
	info, err := source.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maximumProviderExecutableBytes {
		return "", func() {}, fmt.Errorf("provider executable cannot be safely staged")
	}
	directory, err := os.MkdirTemp("", "symphony-ssiag-provider-")
	if err != nil {
		return "", func() {}, fmt.Errorf("create provider staging directory: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(directory) }
	if err := os.Chmod(directory, 0o700); err != nil {
		cleanup()
		return "", func() {}, err
	}
	targetPath := filepath.Join(directory, "adapter")
	target, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o500)
	if err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("create staged provider executable: %w", err)
	}
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(target, hash), io.LimitReader(source, maximumProviderExecutableBytes+1))
	syncErr, closeErr := target.Sync(), target.Close()
	actual := "sha256:" + hex.EncodeToString(hash.Sum(nil))
	if copyErr != nil || syncErr != nil || closeErr != nil || written != info.Size() || actual != declaration.ExecutableDigest {
		cleanup()
		return "", func() {}, fmt.Errorf("provider executable changed during staging")
	}
	return targetPath, cleanup, nil
}

func stageVerifiedBundle(declaration ExecutableTrust, pkg adapterPackage) (string, func(), error) {
	directory, err := os.MkdirTemp("", "symphony-ssiag-provider-bundle-")
	if err != nil {
		return "", func() {}, fmt.Errorf("create provider bundle staging directory: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(directory) }
	if err := os.Chmod(directory, 0o700); err != nil {
		cleanup()
		return "", func() {}, err
	}
	var total int64
	for _, file := range pkg.receipt.Files {
		if file.Size > 64<<20 || total > 256<<20-int64(file.Size) {
			cleanup()
			return "", func() {}, fmt.Errorf("provider bundle exceeds staging bound")
		}
		total += int64(file.Size)
		sourcePath := filepath.Join(pkg.prefix, filepath.FromSlash(file.Path))
		targetPath := filepath.Join(directory, filepath.FromSlash(file.Path))
		if !pathWithinPrefix(directory, targetPath) {
			cleanup()
			return "", func() {}, fmt.Errorf("provider bundle staging path escapes boundary")
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
			cleanup()
			return "", func() {}, fmt.Errorf("create staged provider bundle directory: %w", err)
		}
		if err := copyVerifiedProviderFile(sourcePath, targetPath, file); err != nil {
			cleanup()
			return "", func() {}, err
		}
	}
	targetExecutable := filepath.Join(directory, filepath.FromSlash(pkg.relative))
	if !pathWithinPrefix(directory, targetExecutable) {
		cleanup()
		return "", func() {}, fmt.Errorf("staged provider entry point escapes boundary")
	}
	info, err := os.Lstat(targetExecutable)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		cleanup()
		return "", func() {}, fmt.Errorf("staged provider bundle entry point is invalid")
	}
	digest, err := digestPath(targetExecutable)
	if err != nil || digest != declaration.ExecutableDigest {
		cleanup()
		return "", func() {}, fmt.Errorf("staged provider bundle entry point changed")
	}
	return targetExecutable, cleanup, nil
}

func copyVerifiedProviderFile(sourcePath, targetPath string, evidence receiptFile) error {
	source, err := openProviderFile(sourcePath)
	if err != nil {
		return fmt.Errorf("open verified provider bundle file: %w", err)
	}
	defer source.Close()
	info, err := source.Stat()
	if err != nil || !info.Mode().IsRegular() || uint64(info.Size()) != evidence.Size {
		return fmt.Errorf("provider bundle file changed before staging")
	}
	mode := os.FileMode(0o400)
	if evidence.Kind == "executable" {
		mode = 0o500
	}
	target, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return fmt.Errorf("create staged provider bundle file: %w", err)
	}
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(target, hash), io.LimitReader(source, int64(evidence.Size)+1))
	syncErr, closeErr := target.Sync(), target.Close()
	actual := "sha256:" + hex.EncodeToString(hash.Sum(nil))
	if copyErr != nil || syncErr != nil || closeErr != nil || written != int64(evidence.Size) || actual != evidence.Digest {
		_ = os.Remove(targetPath)
		return fmt.Errorf("provider bundle file changed during staging")
	}
	return nil
}

type boundedBuffer struct {
	bytes.Buffer
	limit    int
	overflow bool
}

func (b *boundedBuffer) Write(payload []byte) (int, error) {
	remaining := b.limit - b.Len()
	if remaining <= 0 {
		b.overflow = true
		return len(payload), nil
	}
	if len(payload) > remaining {
		_, _ = b.Buffer.Write(payload[:remaining])
		b.overflow = true
		return len(payload), nil
	}
	return b.Buffer.Write(payload)
}
