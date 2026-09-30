package knowledgeengine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The copied receipt is a local test fixture, not publisher authentication.
// Every owned resource remains the real admitted SNIV resource. Only the
// executable and its fixture receipt digest change, so refusal must occur at
// the actual process-response boundary rather than installation admission.
func TestSNVInstalledAdversarialTransport(t *testing.T) {
	prefix := os.Getenv("SYMPHONY_SNV_TEST_PREFIX")
	if prefix == "" {
		prefix = os.Getenv("SNV_TEST_PREFIX")
	}
	if prefix == "" {
		t.Skip("SYMPHONY_SNV_TEST_PREFIX selects an actual installed SNIV package")
	}
	installed, err := InspectSNV(prefix, "0.1.0-dev", "sniv")
	if err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(filepath.Join(snvRepository(t), "modules/sniv-engine/tests/fixtures/identity_validate.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := InvokeSNV(context.Background(), prefix, "0.1.0-dev", t.TempDir(), "sniv", "identity_validate", payload); err != nil {
		t.Fatal("real admitted baseline", err)
	}
	testBinary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ mode, reason string }{
		{"valid", ""},
		{"extra_stdout", "invalid SNIV engine response"},
		{"malformed", "invalid SNIV engine response"},
		{"oversized", "engine response exceeds"},
		{"wrong_request_id", "engine response identity mismatch"},
		{"wrong_correlation_id", "engine response identity mismatch"},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			copied := snvAdversarialInstallation(t, prefix, installed, testBinary, tc.mode)
			if _, err := InspectSNV(copied, "0.1.0-dev", "sniv"); err != nil {
				t.Fatal("test executable was not receipt admitted", err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			response, err := InvokeSNV(ctx, copied, "0.1.0-dev", t.TempDir(), "sniv", "identity_validate", payload)
			if tc.reason == "" {
				if err != nil || response.Outcome != "ok" {
					t.Fatal("unchanged delegated response refused", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.reason) {
				t.Fatalf("%s did not refuse at response boundary: %v", tc.mode, err)
			}
			t.Logf("receipt-admitted actual child refused: %s", err)
		})
	}
}

func snvAdversarialInstallation(t *testing.T, prefix string, installed Installation, testBinary, mode string) string {
	t.Helper()
	raw, err := os.ReadFile(installed.ReceiptPath)
	if err != nil {
		t.Fatal(err)
	}
	var receipt receiptV2
	if err := decodeExact(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	copied := t.TempDir()
	for _, owned := range receipt.Files {
		input, err := os.Open(filepath.Join(prefix, filepath.FromSlash(owned.Path)))
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(copied, filepath.FromSlash(owned.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			input.Close()
			t.Fatal(err)
		}
		permissions := os.FileMode(0644)
		if owned.Kind == "executable" {
			permissions = 0755
		}
		output, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, permissions)
		if err != nil {
			input.Close()
			t.Fatal(err)
		}
		_, copyErr := io.Copy(output, input)
		closeErr := output.Close()
		input.Close()
		if copyErr != nil || closeErr != nil {
			t.Fatal(copyErr, closeErr)
		}
	}
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }
	wrapper := []byte("#!/bin/sh\nexport SYMPHONY_SNV_ADVERSARIAL_MODE=" + quote(mode) +
		"\nexport SYMPHONY_SNV_ADVERSARIAL_REAL=" + quote(installed.ExecutablePath) +
		"\nexec " + quote(testBinary) + " -test.run='^TestSNVAdversarialChildProcess$'\n")
	relative := "libexec/symphony/sniv-engine/0.1.0-dev/symphony-sniv"
	if err := os.WriteFile(filepath.Join(copied, filepath.FromSlash(relative)), wrapper, 0755); err != nil {
		t.Fatal(err)
	}
	for i := range receipt.Files {
		if receipt.Files[i].Path == relative {
			receipt.Files[i].Size = uint64(len(wrapper))
			receipt.Files[i].Digest = digestBytes(wrapper)
		}
	}
	writeReceiptV2Fixture(t, filepath.Join(copied, "share/symphony/receipts/sniv-engine/0.1.0-dev/install-receipt.json"), &receipt)
	return copied
}

// A test-only child delegates the unchanged request to the immutable native
// executable before perturbing its output. Wrong IDs receive a recomputed,
// self-consistent envelope seal to isolate correspondence from digest failure.
func TestSNVAdversarialChildProcess(t *testing.T) {
	mode := os.Getenv("SYMPHONY_SNV_ADVERSARIAL_MODE")
	if mode == "" {
		t.Skip("actual child helper invoked by the installed transport gate")
	}
	input, err := io.ReadAll(io.LimitReader(os.Stdin, maxRequestBytes+1))
	if err != nil || len(input) > maxRequestBytes {
		os.Exit(91)
	}
	command := exec.Command(os.Getenv("SYMPHONY_SNV_ADVERSARIAL_REAL"))
	command.Stdin = bytes.NewReader(input)
	raw, err := command.Output()
	if err != nil {
		os.Exit(92)
	}
	switch mode {
	case "valid":
	case "extra_stdout":
		raw = append(raw, []byte("\n{}\n")...)
	case "malformed":
		raw = []byte("{\n")
	case "oversized":
		raw = bytes.Repeat([]byte("x"), maxResponseBytes+1)
	case "wrong_request_id", "wrong_correlation_id":
		object, err := ParseSNVObject(raw)
		if err != nil {
			os.Exit(93)
		}
		field := strings.TrimPrefix(mode, "wrong_")
		object[field] = "other-adversarial-request"
		delete(object, "response_digest")
		canonical, err := SCVCanonical(object)
		if err != nil {
			os.Exit(94)
		}
		object["response_digest"] = digestBytes(canonical)
		raw, err = SCVCanonical(object)
		if err != nil || !json.Valid(raw) {
			os.Exit(95)
		}
	default:
		os.Exit(96)
	}
	if _, err := os.Stdout.Write(raw); err != nil {
		fmt.Fprintln(os.Stderr, "test child output refused")
		os.Exit(97)
	}
	os.Exit(0)
}
