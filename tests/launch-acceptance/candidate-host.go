// candidate-host verifies and extracts an explicitly selected candidate into a
// new private prefix. It never runs a downloaded program or claims acceptance.
package main

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const maxInventory = 16 << 20
const maxPayload = int64(16 << 30)

var errCandidate = errors.New("candidate host preparation refused; preserve private work for diagnosis")

type fileEntry struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}
type candidate struct {
	Protocol              string                        `json:"protocol"`
	PublicationAuthorized bool                          `json:"publication_authorized"`
	Source                struct{ Commit, Tree string } `json:"source"`
	Platform              struct {
		Profile      string `json:"profile_id"`
		OS           string `json:"os"`
		Architecture string `json:"architecture"`
	} `json:"platform"`
	Artifact struct {
		Filename string `json:"filename"`
		SHA256   string `json:"sha256"`
		Size     int64  `json:"size"`
	} `json:"artifact"`
	Files []fileEntry `json:"files"`
}

func safePath(s string) bool {
	if s == "" || s == "." || path.Clean(s) != s || path.IsAbs(s) || strings.ContainsAny(s, "\\:;[]") {
		return false
	}
	for _, part := range strings.Split(s, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	for _, r := range s {
		if r < 32 || r == 127 {
			return false
		}
	}
	return true
}
func regular(path string, maximum int64) (*os.File, error) {
	info, e := os.Lstat(path)
	if e != nil || !info.Mode().IsRegular() || info.Size() > maximum {
		return nil, errCandidate
	}
	f, e := os.Open(path)
	if e != nil {
		return nil, errCandidate
	}
	actual, e := f.Stat()
	if e != nil || !os.SameFile(info, actual) {
		f.Close()
		return nil, errCandidate
	}
	return f, nil
}
func readBounded(path string, limit int64) ([]byte, error) {
	f, e := regular(path, limit)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, limit+1))
	if e != nil || int64(len(b)) > limit {
		return nil, errCandidate
	}
	return b, nil
}
func digest(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

// A private quiescent work directory and a fully verified inventory are required.
// Even a signed archive cannot create links, devices, duplicates or unlisted files.
func extract(archive *os.File, root string, inv candidate) error {
	if len(inv.Files) == 0 || len(inv.Files) > 65536 || inv.Artifact.Size < 0 || inv.Artifact.Size > maxPayload {
		return errCandidate
	}
	entries := map[string]fileEntry{}
	var total int64
	for _, f := range inv.Files {
		decoded, e := hex.DecodeString(f.SHA256)
		if !safePath(f.Path) || e != nil || len(decoded) != 32 || f.SHA256 != strings.ToLower(f.SHA256) || f.Size < 0 || f.Size > maxPayload-total {
			return errCandidate
		}
		if _, ok := entries[f.Path]; ok {
			return errCandidate
		}
		entries[f.Path] = f
		total += f.Size
	}
	h := sha256.New()
	n, e := io.Copy(h, io.LimitReader(archive, maxPayload+1))
	if e != nil || n != inv.Artifact.Size || hex.EncodeToString(h.Sum(nil)) != inv.Artifact.SHA256 {
		return errCandidate
	}
	if _, e = archive.Seek(0, 0); e != nil {
		return errCandidate
	}
	if e = os.Mkdir(root, 0700); e != nil {
		return errCandidate
	}
	owner, e := os.OpenRoot(root)
	if e != nil {
		return errCandidate
	}
	defer owner.Close()
	reader := tar.NewReader(archive)
	seen := map[string]bool{}
	for {
		header, e := reader.Next()
		if e == io.EOF {
			break
		}
		if e != nil {
			return errCandidate
		}
		f, ok := entries[header.Name]
		if !ok || seen[header.Name] || !safePath(header.Name) || header.Typeflag != tar.TypeReg || header.Size != f.Size || header.Mode < 0 || header.Mode&^int64(0777) != 0 || header.Mode&0022 != 0 {
			return errCandidate
		}
		parent := path.Dir(header.Name)
		if parent != "." {
			if e = owner.MkdirAll(parent, 0700); e != nil {
				return errCandidate
			}
		}
		output, e := owner.OpenFile(header.Name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			return errCandidate
		}
		h := sha256.New()
		n, e := io.Copy(io.MultiWriter(output, h), reader)
		if e == nil && (n != f.Size || hex.EncodeToString(h.Sum(nil)) != f.SHA256) {
			e = errCandidate
		}
		if e == nil {
			e = output.Chmod(os.FileMode(header.Mode))
		}
		if e == nil {
			e = output.Sync()
		}
		closeErr := output.Close()
		if e != nil || closeErr != nil {
			return errCandidate
		}
		seen[header.Name] = true
	}
	if len(seen) != len(entries) {
		return errCandidate
	}
	padding := make([]byte, 32768)
	for {
		n, e := archive.Read(padding)
		for _, b := range padding[:n] {
			if b != 0 {
				return errCandidate
			}
		}
		if e == io.EOF {
			break
		}
		if e != nil {
			return errCandidate
		}
	}
	return nil
}

type options struct{ repository, candidate, signers, signer, profile, work string }

func prepare(o options) error {
	for _, p := range []string{o.repository, o.candidate, o.signers, o.work} {
		if !filepath.IsAbs(p) || filepath.Clean(p) != p {
			return errCandidate
		}
	}
	if o.signer == "" || o.profile == "" {
		return errCandidate
	}
	// Policy is outside the candidate; CMake independently enforces this boundary.
	repo, e := filepath.EvalSymlinks(o.repository)
	if e != nil {
		return errCandidate
	}
	verifier := filepath.Join(repo, "cmake/release/VerifyCandidate.cmake")
	parent, e := filepath.EvalSymlinks(filepath.Dir(o.work))
	if e != nil {
		return errCandidate
	}
	o.work = filepath.Join(parent, filepath.Base(o.work))
	candidateRoot, e := filepath.EvalSymlinks(o.candidate)
	if e != nil {
		return errCandidate
	}
	for _, root := range []string{repo, candidateRoot} {
		rel, e := filepath.Rel(root, o.work)
		if e != nil || rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return errCandidate
		}
	}
	if _, e = os.Lstat(o.work); !os.IsNotExist(e) {
		return errCandidate
	}
	if e = os.Mkdir(o.work, 0700); e != nil {
		return errCandidate
	}
	verify := func(unpacked bool) error {
		args := []string{"-DRELEASE_CANDIDATE=" + o.candidate, "-DRELEASE_ALLOWED_SIGNERS=" + o.signers, "-DRELEASE_SIGNER=" + o.signer, "-DRELEASE_EXPECTED_PROFILE=" + o.profile}
		name := "downloaded-bytes"
		if unpacked {
			name = "unpacked"
			args = append(args, "-DRELEASE_UNPACKED_ROOT="+filepath.Join(o.work, "payload"))
		}
		args = append(args, "-DRELEASE_VERIFICATION_OUTPUT="+filepath.Join(o.work, name+".json"), "-P", verifier)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		log, e := os.OpenFile(filepath.Join(o.work, name+".log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			return errCandidate
		}
		defer log.Close()
		cmd := exec.CommandContext(ctx, "cmake", args...)
		cmd.Stdout = log
		cmd.Stderr = log
		if cmd.Run() != nil {
			return errCandidate
		}
		return nil
	}
	if e = verify(false); e != nil {
		return e
	}
	raw, e := readBounded(filepath.Join(o.candidate, "candidate.json"), maxInventory)
	if e != nil {
		return e
	}
	proof, e := readBounded(filepath.Join(o.work, "downloaded-bytes.json"), maxInventory)
	if e != nil {
		return e
	}
	var verified struct {
		Inventory string `json:"inventory_sha256"`
		Artifact  string `json:"artifact_sha256"`
	}
	if json.Unmarshal(proof, &verified) != nil || verified.Inventory != digest(raw) {
		return errCandidate
	}
	var inv candidate
	if json.Unmarshal(raw, &inv) != nil || inv.Protocol != "symphony.release.candidate.v1" || inv.PublicationAuthorized || inv.Platform.Profile != o.profile || inv.Artifact.Filename != "candidate.tar" || inv.Artifact.SHA256 != verified.Artifact {
		return errCandidate
	}
	archive, e := regular(filepath.Join(o.candidate, "candidate.tar"), maxPayload)
	if e != nil {
		return e
	}
	defer archive.Close()
	if e = extract(archive, filepath.Join(o.work, "payload"), inv); e != nil {
		return e
	}
	if e = verify(true); e != nil {
		return e
	}
	second, e := readBounded(filepath.Join(o.work, "unpacked.json"), maxInventory)
	if e != nil {
		return e
	}
	var reverified struct {
		Inventory string `json:"inventory_sha256"`
		Policy    string `json:"allowed_signers_sha256"`
	}
	var original struct {
		Policy string `json:"allowed_signers_sha256"`
	}
	if json.Unmarshal(second, &reverified) != nil || json.Unmarshal(proof, &original) != nil || reverified.Inventory != verified.Inventory || reverified.Policy != original.Policy {
		return errCandidate
	}
	host, e := exec.Command("uname", "-srm").Output()
	if e != nil {
		return errCandidate
	}
	result := map[string]any{"protocol": "symphony.release.host-preparation.v1", "status": "verified_extraction_only", "publication_authorized": false, "release_acceptance_complete": false, "fresh_host_verified": false, "provider_qualification_complete": false, "source_commit": inv.Source.Commit, "source_tree": inv.Source.Tree, "artifact_sha256": inv.Artifact.SHA256, "inventory_sha256": verified.Inventory, "allowed_signers_sha256": original.Policy, "profile_id": o.profile, "host_os": runtime.GOOS, "host_arch": runtime.GOARCH, "host_kernel": strings.TrimSpace(string(host)), "prepared_at": time.Now().UTC().Format(time.RFC3339Nano), "payload_file_count": len(inv.Files), "remaining": "fresh host inventory, OS/runtime/loader qualification, approved scenario execution and provider evidence"}
	data, e := json.MarshalIndent(result, "", "  ")
	if e != nil {
		return errCandidate
	}
	return os.WriteFile(filepath.Join(o.work, "host-preparation.json"), append(data, '\n'), 0600)
}
func main() {
	var o options
	flag.StringVar(&o.repository, "repository", "", "reviewed verification-tool checkout")
	flag.StringVar(&o.candidate, "candidate", "", "retrieved candidate directory")
	flag.StringVar(&o.signers, "allowed-signers", "", "independently obtained signer policy")
	flag.StringVar(&o.signer, "signer", "", "selected publisher principal")
	flag.StringVar(&o.profile, "profile", "", "exact selected candidate profile")
	flag.StringVar(&o.work, "work", "", "new private work directory")
	flag.Parse()
	if flag.NArg() != 0 || prepare(o) != nil {
		fmt.Fprintln(os.Stderr, errCandidate)
		os.Exit(1)
	}
	fmt.Println("Verified extraction completed; acceptance and publication remain pending")
}
