package knowledgeengine

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestReceiptV2StreamsSDKArtifactBeyondJSONBound(t *testing.T) {
	prefix := t.TempDir()
	receiptPath, receipt := createInstalledV2Fixture(t, skviSpec, prefix, "0.1.0-dev")
	relative := "lib/symphony/skvi-engine/0.1.0-dev/libsymphony-skvi.a"
	path := filepath.Join(prefix, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	contents := bytes.Repeat([]byte("a"), maxResponseBytes+1)
	if err := os.WriteFile(path, contents, 0644); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(contents)
	receipt.Files = append(receipt.Files, receiptV2File{Path: relative, Kind: "regular", Size: uint64(len(contents)), Digest: "sha256:" + hex.EncodeToString(hash[:])})
	writeReceiptV2Fixture(t, receiptPath, &receipt)
	if _, err := InspectInstallation("skvi", prefix, "0.1.0-dev"); err != nil {
		t.Fatalf("valid SDK artifact above the JSON byte bound was rejected: %v", err)
	}
	contents[len(contents)-1] = 'b'
	if err := os.WriteFile(path, contents, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectInstallation("skvi", prefix, "0.1.0-dev"); err == nil {
		t.Fatal("streamed artifact digest drift was accepted")
	}
	contents[len(contents)-1] = 'a'
	if err := os.WriteFile(path, contents, 0644); err != nil {
		t.Fatal(err)
	}
	receipt.Files[len(receipt.Files)-1].Size++
	writeReceiptV2Fixture(t, receiptPath, &receipt)
	if _, err := InspectInstallation("skvi", prefix, "0.1.0-dev"); err == nil {
		t.Fatal("resealed receipt with a false artifact size was accepted")
	}
	receipt.Files[len(receipt.Files)-1].Size--
	writeReceiptV2Fixture(t, receiptPath, &receipt)
	if err := os.Chmod(path, 0666); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectInstallation("skvi", prefix, "0.1.0-dev"); err == nil {
		t.Fatal("writable SDK artifact was accepted")
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "archive")
	if err := os.Rename(path, outside); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, path); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectInstallation("skvi", prefix, "0.1.0-dev"); err == nil {
		t.Fatal("symlinked SDK artifact was accepted")
	}
}

func TestReceiptV2ArtifactHasIndependentExplicitByteBound(t *testing.T) {
	prefix := t.TempDir()
	receiptPath, receipt := createInstalledV2Fixture(t, skviSpec, prefix, "0.1.0-dev")
	relative := "lib/oversized-sdk.a"
	path := filepath.Join(prefix, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = file.Truncate(maxReceiptV2ArtifactBytes + 1); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	receipt.Files = append(receipt.Files, receiptV2File{Path: relative, Kind: "regular", Size: uint64(maxReceiptV2ArtifactBytes + 1), Digest: "sha256:" + string(bytes.Repeat([]byte("0"), 64))})
	writeReceiptV2Fixture(t, receiptPath, &receipt)
	if _, err := InspectInstallation("skvi", prefix, "0.1.0-dev"); err == nil {
		t.Fatal("artifact above the explicit 64 MiB package bound was accepted")
	}
}
