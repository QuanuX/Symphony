package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"runtime"
	"testing"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/version"
)

func TestVersionJSONBindsRunningExecutable(t *testing.T) {
	output := captureStdout(t, func() error {
		return executeCommand([]string{"version", "--json"})
	})
	var info version.Info
	if err := json.Unmarshal([]byte(output), &info); err != nil {
		t.Fatal(err)
	}
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	if info.Protocol != version.BuildInfoProtocol || info.ComponentID != "qxctl" || info.Version != version.Version ||
		info.SourceState != "unavailable" || info.ExecutableSHA256 != hex.EncodeToString(digest[:]) ||
		info.GoVersion != runtime.Version() || info.GOOS != runtime.GOOS || info.GOARCH != runtime.GOARCH {
		t.Fatalf("incorrect running identity: %+v", info)
	}
}
