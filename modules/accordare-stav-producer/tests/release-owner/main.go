// Release qualification harness only. Reads the existing owner's receipt contract;
// never installs, enrolls, starts a service, or executes the inspected payload.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/packageinstall"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/version"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"syscall"
)

var errRefused = errors.New("owner inspection refused")

const component = "accordare-stav-producer"

type fileEvidence struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Mode   uint32 `json:"mode"`
}
type result struct {
	Protocol      string       `json:"protocol"`
	Adapter       string       `json:"verification_adapter"`
	Component     string       `json:"component_id"`
	Version       string       `json:"version"`
	OS            string       `json:"os"`
	Architecture  string       `json:"architecture"`
	Binary        fileEvidence `json:"binary"`
	Receipt       fileEvidence `json:"receipt"`
	ReceiptDigest string       `json:"receipt_digest"`
	Scope         string       `json:"scope"`
	Provider      bool         `json:"provider_qualified"`
	FreshHost     bool         `json:"fresh_host_qualified"`
	Release       bool         `json:"release_accepted"`
	Publication   bool         `json:"publication_authorized"`
}

func observe(name string, limit int64) (fileEvidence, error) {
	var out fileEvidence
	if !filepath.IsAbs(name) || filepath.Clean(name) != name {
		return out, errRefused
	}
	resolved, err := filepath.EvalSymlinks(name)
	if err != nil || resolved != name {
		return out, errRefused
	}
	before, err := os.Lstat(name)
	if err != nil || !before.Mode().IsRegular() || before.Mode().Perm()&0022 != 0 || before.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 || before.Size() > limit {
		return out, errRefused
	}
	stat, ok := before.Sys().(*syscall.Stat_t)
	if !ok || stat.Nlink != 1 || stat.Uid != uint32(os.Geteuid()) {
		return out, errRefused
	}
	f, err := os.Open(name)
	if err != nil {
		return out, errRefused
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(before, opened) {
		return out, errRefused
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, limit+1))
	after, e := os.Lstat(name)
	if err != nil || e != nil || n != before.Size() || n > limit || !os.SameFile(before, after) || before.Mode() != after.Mode() || !before.ModTime().Equal(after.ModTime()) {
		return out, errRefused
	}
	return fileEvidence{name, hex.EncodeToString(h.Sum(nil)), n, uint32(before.Mode().Perm())}, nil
}
func owner(prefix, binary, receipt string) (string, string, error) {
	r, err := packageinstall.Inspect(prefix, version.Version)
	if err != nil || r.Binary != binary || r.Receipt != receipt {
		return "", "", errRefused
	}
	// The owner validates the file digest; observe it independently for this report.
	f, err := observe(binary, 16<<30)
	if err != nil {
		return "", "", errRefused
	}
	return "sha256:" + f.SHA256, r.ReceiptDigest, nil
}
func inspect(binary string) (result, error) {
	var out result
	suffix := filepath.Join("libexec", "symphony", component, version.Version, "symphony-accordare-stav-producer")
	if !strings.HasSuffix(binary, string(filepath.Separator)+suffix) {
		return out, errRefused
	}
	prefix := strings.TrimSuffix(binary, string(filepath.Separator)+suffix)
	if prefix == "" {
		return out, errRefused
	}
	receipt := filepath.Join(prefix, "share", "symphony", "receipts", component, version.Version, "install-receipt.json")
	b, err := observe(binary, 16<<30)
	if err != nil || b.Mode&0111 == 0 {
		return out, errRefused
	}
	r, err := observe(receipt, 16<<20)
	if err != nil {
		return out, errRefused
	}
	binaryDigest, receiptDigest, err := owner(prefix, binary, receipt)
	if err != nil || binaryDigest != "sha256:"+b.SHA256 || !strings.HasPrefix(receiptDigest, "sha256:") || len(receiptDigest) != 71 {
		return out, errRefused
	}
	b2, e1 := observe(binary, 16<<30)
	r2, e2 := observe(receipt, 16<<20)
	if e1 != nil || e2 != nil || !reflect.DeepEqual(b, b2) || !reflect.DeepEqual(r, r2) {
		return out, errRefused
	}
	b.Path, _ = filepath.Rel(prefix, b.Path)
	r.Path, _ = filepath.Rel(prefix, r.Path)
	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "macos"
	}
	out = result{Protocol: "symphony.release.owner-inspection.v1", Adapter: "accordare-receipt-v2", Component: component, Version: version.Version, OS: osName, Architecture: runtime.GOARCH, Binary: b, Receipt: r, ReceiptDigest: receiptDigest, Scope: "read_only_owner_receipt"}
	return out, nil
}
func main() {
	binary := flag.String("binary", "", "absolute canonical installed executable; private quiescent prefix")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, errRefused)
		os.Exit(1)
	}
	r, err := inspect(*binary)
	if err != nil {
		fmt.Fprintln(os.Stderr, errRefused)
		os.Exit(1)
	}
	if json.NewEncoder(os.Stdout).Encode(r) != nil {
		os.Exit(1)
	}
}
