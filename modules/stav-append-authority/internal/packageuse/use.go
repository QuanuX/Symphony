// Package packageuse binds the existing native launch paths to the package
// owner's retirement guard. Development paths remain outside this guarantee.
package packageuse

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/packageretirement"
	stavpaths "github.com/QuanuX/Symphony/modules/stav-append-authority/internal/paths"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/version"
)

const Owner = "stav-append-authority"

// Acquire must precede service initialization or reference creation. The
// returned release is held until all work using the executable has stopped.
func Acquire(executable string) (func(), error) { return acquire(executable, "use") }

// AcquireMaintenance excludes service startup/use during destructive native
// maintenance. It does not reopen an already-retiring installation.
func AcquireMaintenance(executable string) (func(), error) { return acquire(executable, "maintenance") }

// AcquireRecovery retains existing controller custody while allowing stop and
// recovery through closed admission; it cannot admit a new service or reference.
func AcquireRecovery(executable string) (func(), error) { return acquire(executable, "recovery") }

func acquire(executable string, mode string) (func(), error) {
	if !filepath.IsAbs(executable) || filepath.Clean(executable) != executable {
		return nil, fmt.Errorf("package use requires an exact absolute executable path")
	}
	marker := string(filepath.Separator) + filepath.Join("libexec", "symphony", Owner) + string(filepath.Separator)
	suffix := marker + filepath.Join(version.Version, stavpaths.BinaryName)
	if !strings.HasSuffix(executable, suffix) {
		if strings.Contains(executable, marker) {
			return nil, fmt.Errorf("STAV managed executable version does not match this owner")
		}
		return func() {}, nil // Existing development/legacy execution is unqualified.
	}
	prefix := strings.TrimSuffix(executable, suffix)
	identity, phase, err := packageretirement.Inspect(prefix, Owner, version.Version)
	if err != nil {
		return nil, fmt.Errorf("STAV managed package has no valid retirement guard: %w", err)
	}
	if filepath.Join(prefix, filepath.FromSlash(identity.BinaryRelative)) != executable {
		return nil, fmt.Errorf("STAV retirement guard names a different executable")
	}
	var lease *packageretirement.Lease
	if mode == "recovery" {
		lease, err = packageretirement.AcquireRecoveryUse(prefix, identity)
	} else if mode == "use" {
		lease, err = packageretirement.AcquireUse(prefix, identity)
	} else if phase == "open" {
		lease, err = packageretirement.PrepareInstall(prefix, identity, false)
	} else if phase == "retiring" {
		lease, err = packageretirement.BeginRetirement(prefix, identity)
	} else {
		return nil, fmt.Errorf("STAV package is not available for maintenance")
	}
	if err != nil {
		return nil, err
	}
	if err := lease.VerifyFile(identity.BinaryRelative, identity.BinaryDigest, identity.BinarySize); err != nil {
		lease.Close()
		return nil, err
	}
	receipt, err := lease.ReadFile(identity.ReceiptRelative, 1024*1024)
	if err == nil {
		err = verifyReceiptIdentity(receipt, identity.ReceiptDigest)
	}
	if err != nil {
		lease.Close()
		return nil, err
	}
	return func() { lease.Close() }, nil
}

// The guard digest binds every semantic receipt field. Whitespace is irrelevant;
// altered capabilities or ownership cannot acquire admission through a valid path.
func verifyReceiptIdentity(data []byte, expected string) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var receipt map[string]any
	if err := decoder.Decode(&receipt); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return fmt.Errorf("STAV receipt has trailing JSON")
	}
	if receipt["receipt_digest"] != expected {
		return fmt.Errorf("STAV receipt identity differs from retirement guard")
	}
	delete(receipt, "receipt_digest")
	canonical, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(canonical)
	if "sha256:"+hex.EncodeToString(sum[:]) != expected {
		return fmt.Errorf("STAV receipt content differs from retirement guard")
	}
	return nil
}
