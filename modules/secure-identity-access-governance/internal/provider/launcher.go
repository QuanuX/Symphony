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
	"time"
)

type processLauncher struct{}

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
