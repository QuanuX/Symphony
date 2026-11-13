// delivery-set binds a complete, explicitly selected multi-artifact delivery.
// This release tool does not install payloads, access providers, or establish
// publisher authority or release acceptance. Explicit owner modes invoke trusted
// readers; inspect-executable runs only an authenticated executable's fixed
// version query in a private empty home after complete signature/byte checks.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"debug/buildinfo"
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
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
)

const dsMetadataLimit = 16 << 20
const dsPayloadLimit int64 = 16 << 30
const dsNamespace = "symphony-delivery-set"

var dsError = errors.New("delivery-set refused; preserve private evidence and inspect the selected inputs")

type dsSource struct {
	Commit string `json:"commit"`
	Tree   string `json:"tree"`
}
type dsProfile struct {
	ID           string `json:"profile_id"`
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
}
type dsChoice struct {
	ID           string            `json:"id"`
	PackageID    string            `json:"package_id"`
	Kind         string            `json:"kind"`
	Adapter      string            `json:"verification_adapter"`
	Versions     map[string]string `json:"component_versions"`
	Dependencies []string          `json:"dependencies"`
}
type dsSelection struct {
	Protocol      string     `json:"protocol"`
	Source        dsSource   `json:"source"`
	Profile       dsProfile  `json:"profile"`
	CapabilitySHA string     `json:"capability_manifest_sha256"`
	DependencySHA string     `json:"dependency_map_sha256"`
	IdentitySHA   string     `json:"identity_catalog_sha256"`
	Artifacts     []dsChoice `json:"artifacts"`
}
type dsFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Mode   uint32 `json:"mode"`
}
type dsVendor struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Provenance string `json:"provenance_path"`
}
type dsArtifact struct {
	Selection       dsChoice  `json:"selection"`
	Source          *dsSource `json:"source"`
	Vendor          *dsVendor `json:"vendor"`
	Files           []string  `json:"file_paths"`
	Primary         string    `json:"primary_path"`
	Inventory       string    `json:"payload_inventory_path"`
	Manifest        *string   `json:"manifest_path"`
	Signature       *string   `json:"signature_path"`
	SignerPolicySHA *string   `json:"artifact_signer_policy_sha256"`
	Receipts        []string  `json:"receipt_paths"`
	InstallRecipe   string    `json:"install_recipe_path"`
	RemoveRecipe    string    `json:"remove_recipe_path"`
	Notices         []string  `json:"notice_paths"`
	SourceMaterial  []string  `json:"source_material_paths"`
	BuildEvidence   []string  `json:"build_evidence_paths"`
}
type dsDefinition struct {
	SetID     string       `json:"set_id"`
	Files     []string     `json:"file_paths"`
	Artifacts []dsArtifact `json:"artifacts"`
}
type dsIndex struct {
	Protocol      string       `json:"protocol"`
	SetID         string       `json:"set_id"`
	Source        dsSource     `json:"source"`
	Profile       dsProfile    `json:"profile"`
	CapabilitySHA string       `json:"capability_manifest_sha256"`
	SelectionSHA  string       `json:"package_selection_sha256"`
	DependencySHA string       `json:"dependency_map_sha256"`
	IdentitySHA   string       `json:"identity_catalog_sha256"`
	PolicySHA     string       `json:"index_signer_policy_sha256"`
	Files         []dsFile     `json:"files"`
	Artifacts     []dsArtifact `json:"artifacts"`
	Publication   bool         `json:"publication_authorized"`
}
type dsResult struct {
	Protocol       string    `json:"protocol"`
	Scope          string    `json:"scope"`
	IndexSHA       string    `json:"index_sha256"`
	SelectionSHA   string    `json:"package_selection_sha256"`
	PolicySHA      string    `json:"index_signer_policy_sha256"`
	Source         dsSource  `json:"source"`
	Profile        dsProfile `json:"profile"`
	VerifiedFiles  int       `json:"verified_files"`
	VerifiedBytes  int64     `json:"verified_bytes"`
	SignatureValid bool      `json:"index_signature_verified"`
	OwnerVerified  bool      `json:"owner_verification_complete"`
	FreshHost      bool      `json:"fresh_host_verified"`
	Release        bool      `json:"release_acceptance_complete"`
	Publication    bool      `json:"publication_authorized"`
	Missing        []string  `json:"pending_owner_adapters"`
}

func dsHash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func dsHex(s string, bytes int) bool {
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == bytes && s == strings.ToLower(s)
}
func dsToken(s string) bool {
	if len(s) == 0 || len(s) > 128 {
		return false
	}
	for _, c := range s {
		if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789._-", c) {
			return false
		}
	}
	return s != "." && s != ".."
}
func dsSigner(s string) bool {
	if len(s) == 0 || len(s) > 256 {
		return false
	}
	for _, c := range s {
		if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789._-@+", c) {
			return false
		}
	}
	return true
}
func dsPath(s string) bool {
	if !utf8.ValidString(s) || len(s) == 0 || len(s) > 2048 || path.IsAbs(s) || path.Clean(s) != s || strings.ContainsAny(s, "\\:;[]") {
		return false
	}
	for _, part := range strings.Split(s, "/") {
		if part == "." || part == ".." || part == "" {
			return false
		}
	}
	for _, c := range s {
		if c < 32 || c == 127 {
			return false
		}
	}
	return s != "delivery-set.json" && s != "delivery-set.json.sig"
}

// Reject missing fields as well as duplicate/unknown fields and primitive nulls.
// Pointer fields must be explicitly null when inapplicable; lists use [].
func dsShape(v any, t reflect.Type) bool {
	if t.Kind() == reflect.Pointer {
		return v == nil || dsShape(v, t.Elem())
	}
	if v == nil {
		return false
	}
	switch t.Kind() {
	case reflect.Struct:
		m, ok := v.(map[string]any)
		if !ok || len(m) != t.NumField() {
			return false
		}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			key := strings.Split(f.Tag.Get("json"), ",")[0]
			value, exists := m[key]
			if !exists || !dsShape(value, f.Type) {
				return false
			}
		}
	case reflect.Slice:
		a, ok := v.([]any)
		if !ok {
			return false
		}
		for _, value := range a {
			if !dsShape(value, t.Elem()) {
				return false
			}
		}
	case reflect.Map:
		m, ok := v.(map[string]any)
		if !ok {
			return false
		}
		for _, value := range m {
			if !dsShape(value, t.Elem()) {
				return false
			}
		}
	}
	return true
}
func dsDecode(b []byte, out any) error {
	if len(b) == 0 || len(b) > dsMetadataLimit || !utf8.Valid(b) {
		return dsError
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	nodes := 0
	var walk func(int) error
	walk = func(depth int) error {
		nodes++
		if depth > 32 || nodes > 200000 {
			return dsError
		}
		token, e := d.Token()
		if e != nil {
			return dsError
		}
		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '{':
				seen := map[string]bool{}
				for d.More() {
					k, e := d.Token()
					if e != nil {
						return dsError
					}
					key, ok := k.(string)
					if !ok || seen[key] {
						return dsError
					}
					seen[key] = true
					if walk(depth+1) != nil {
						return dsError
					}
				}
			case '[':
				for d.More() {
					if walk(depth+1) != nil {
						return dsError
					}
				}
			default:
				return dsError
			}
			if _, e := d.Token(); e != nil {
				return dsError
			}
		}
		return nil
	}
	if walk(0) != nil {
		return dsError
	}
	if _, e := d.Token(); e != io.EOF {
		return dsError
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.DisallowUnknownFields()
	if decoder.Decode(out) != nil {
		return dsError
	}
	var shape any
	if json.Unmarshal(b, &shape) != nil || !dsShape(shape, reflect.TypeOf(out).Elem()) {
		return dsError
	}
	return nil
}

func dsRead(p string, maximum int64) ([]byte, error) {
	info, e := os.Lstat(p)
	if e != nil || !info.Mode().IsRegular() || info.Size() > maximum || info.Size() < 0 {
		return nil, dsError
	}
	f, e := os.Open(p)
	if e != nil {
		return nil, dsError
	}
	defer f.Close()
	actual, e := f.Stat()
	if e != nil || !os.SameFile(info, actual) {
		return nil, dsError
	}
	b, e := io.ReadAll(io.LimitReader(f, maximum+1))
	if e != nil || int64(len(b)) > maximum {
		return nil, dsError
	}
	after, e := os.Lstat(p)
	if e != nil || !after.Mode().IsRegular() || !os.SameFile(info, after) || after.Size() != int64(len(b)) {
		return nil, dsError
	}
	return b, nil
}
func dsContained(root, relative string) (string, error) {
	if !dsPath(relative) {
		return "", dsError
	}
	p := root
	parts := strings.Split(relative, "/")
	for i, part := range parts {
		p = filepath.Join(p, part)
		info, e := os.Lstat(p)
		if e != nil || info.Mode()&os.ModeSymlink != 0 || (i < len(parts)-1 && !info.IsDir()) {
			return "", dsError
		}
	}
	return p, nil
}
func dsObserve(root, relative string) (dsFile, error) {
	var result dsFile
	p, e := dsContained(root, relative)
	if e != nil {
		return result, e
	}
	info, e := os.Lstat(p)
	if e != nil || !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > dsPayloadLimit || info.Mode().Perm()&0022 != 0 || info.Mode()&os.ModeType != 0 || info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
		return result, dsError
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); !ok || stat.Nlink != 1 {
		return result, dsError
	}
	f, e := os.Open(p)
	if e != nil {
		return result, dsError
	}
	defer f.Close()
	actual, e := f.Stat()
	if e != nil || !os.SameFile(info, actual) {
		return result, dsError
	}
	h := sha256.New()
	n, e := io.Copy(h, io.LimitReader(f, info.Size()+1))
	after, statErr := os.Lstat(p)
	if e != nil || statErr != nil || n != info.Size() || !after.Mode().IsRegular() || !os.SameFile(info, after) || after.Size() != n || after.Mode() != info.Mode() || !after.ModTime().Equal(info.ModTime()) {
		return result, dsError
	}
	return dsFile{Path: relative, SHA256: hex.EncodeToString(h.Sum(nil)), Size: n, Mode: uint32(info.Mode().Perm())}, nil
}
func dsIndependent(root, p string) ([]byte, error) {
	if !filepath.IsAbs(p) {
		return nil, dsError
	}
	actual, e := filepath.EvalSymlinks(p)
	if e != nil {
		return nil, dsError
	}
	rel, e := filepath.Rel(root, actual)
	if e != nil || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return nil, dsError
	}
	return dsRead(p, dsMetadataLimit)
}
func dsRoot(root string) (string, error) {
	if !filepath.IsAbs(root) {
		return "", dsError
	}
	info, e := os.Lstat(root)
	if e != nil || !info.IsDir() || info.Mode().Perm()&0022 != 0 {
		return "", dsError
	}
	return filepath.EvalSymlinks(root)
}
func dsSourceOK(s dsSource) bool { return dsHex(s.Commit, 20) && dsHex(s.Tree, 20) }
func dsProfileOK(p dsProfile) bool {
	return dsToken(p.ID) && (p.OS == "macos" || p.OS == "linux") && (p.Architecture == "amd64" || p.Architecture == "arm64")
}
func dsChoiceOK(c dsChoice) bool {
	adapters := map[string]string{"native-candidate-v1": "native", "ssiag-receipt-v2": "go", "stav-receipt-v2": "go", "accordare-receipt-v2": "go", "qxctl-files-v1": "go", "sov-executable-v1": "go", "swift-metadata-v2": "swift", "cloudflare-bundle-v1": "vendor", "source-material-v1": "source"}
	if !dsToken(c.ID) || !dsToken(c.PackageID) || adapters[c.Adapter] != c.Kind || c.Kind == "" || len(c.Versions) == 0 || len(c.Versions) > 256 || len(c.Dependencies) > 4096 {
		return false
	}
	for component, version := range c.Versions {
		if !dsToken(component) || !dsToken(version) {
			return false
		}
	}
	seen := map[string]bool{}
	for _, dep := range c.Dependencies {
		if !dsToken(dep) || dep == c.ID || seen[dep] {
			return false
		}
		seen[dep] = true
	}
	return true
}
func dsSelectionOK(selection dsSelection) bool {
	if selection.Protocol != "symphony.release.delivery-selection.v1" || !dsSourceOK(selection.Source) || !dsProfileOK(selection.Profile) || !dsHex(selection.CapabilitySHA, 32) || !dsHex(selection.DependencySHA, 32) || !dsHex(selection.IdentitySHA, 32) || len(selection.Artifacts) == 0 || len(selection.Artifacts) > 4096 {
		return false
	}
	choices := map[string]dsChoice{}
	for _, c := range selection.Artifacts {
		if !dsChoiceOK(c) {
			return false
		}
		if _, exists := choices[c.ID]; exists {
			return false
		}
		choices[c.ID] = c
	}
	visited := map[string]int{}
	var visit func(string) bool
	visit = func(id string) bool {
		if visited[id] == 1 {
			return false
		}
		if visited[id] == 2 {
			return true
		}
		c, exists := choices[id]
		if !exists {
			return false
		}
		visited[id] = 1
		for _, dep := range c.Dependencies {
			if !visit(dep) {
				return false
			}
		}
		visited[id] = 2
		return true
	}
	for id := range choices {
		if !visit(id) {
			return false
		}
	}
	return true
}
func dsValidate(index dsIndex, selection dsSelection, selectionRaw, policy []byte, expectedProfile string) error {
	if !dsSelectionOK(selection) || index.Protocol != "symphony.release.delivery-set.v1" || !dsToken(index.SetID) || index.Publication || index.Source != selection.Source || index.Profile != selection.Profile || index.Profile.ID != expectedProfile || index.CapabilitySHA != selection.CapabilitySHA || index.DependencySHA != selection.DependencySHA || index.IdentitySHA != selection.IdentitySHA || index.SelectionSHA != dsHash(selectionRaw) || index.PolicySHA != dsHash(policy) || len(policy) == 0 || len(index.Artifacts) != len(selection.Artifacts) || len(index.Files) == 0 || len(index.Files) > 65536 {
		return dsError
	}
	files := map[string]dsFile{}
	total := int64(0)
	previous := ""
	for _, f := range index.Files {
		if !dsPath(f.Path) || f.Path <= previous || !dsHex(f.SHA256, 32) || f.Size < 0 || f.Size > dsPayloadLimit-total || f.Mode > 0777 || f.Mode&0022 != 0 {
			return dsError
		}
		previous = f.Path
		total += f.Size
		files[f.Path] = f
	}
	choices := map[string]dsChoice{}
	for _, c := range selection.Artifacts {
		choices[c.ID] = c
	}
	seenArtifacts := map[string]bool{}
	assigned := map[string]bool{}
	referenced := map[string]bool{}
	for _, a := range index.Artifacts {
		choice, exists := choices[a.Selection.ID]
		if !exists || seenArtifacts[a.Selection.ID] || !reflect.DeepEqual(choice, a.Selection) || len(a.Files) == 0 {
			return dsError
		}
		seenArtifacts[a.Selection.ID] = true
		if choice.Kind == "vendor" {
			if a.Source != nil || a.Vendor == nil || !dsToken(a.Vendor.Name) || !dsToken(a.Vendor.Version) {
				return dsError
			}
		} else if a.Source == nil || *a.Source != index.Source || a.Vendor != nil {
			return dsError
		}
		local := map[string]bool{}
		for _, p := range a.Files {
			if _, exists := files[p]; !exists || assigned[p] {
				return dsError
			}
			local[p] = true
			assigned[p] = true
			referenced[p] = true
		}
		if !local[a.Primary] || !local[a.Inventory] {
			return dsError
		}
		if choice.Kind == "native" {
			if a.Manifest == nil || a.Signature == nil || a.SignerPolicySHA == nil || !dsHex(*a.SignerPolicySHA, 32) || !local[*a.Manifest] || !local[*a.Signature] || a.Inventory != *a.Manifest {
				return dsError
			}
			directory := path.Dir(a.Primary)
			if path.Base(a.Primary) != "candidate.tar" || *a.Manifest != path.Join(directory, "candidate.json") || *a.Signature != path.Join(directory, "candidate.json.sig") || len(a.Files) != 8 {
				return dsError
			}
			for _, name := range []string{"candidate.tar", "candidate.json", "candidate.json.sig", "platform-evidence.json", "build-evidence.json", "dependency-map.json", "identity-catalog.json", "SHA256SUMS"} {
				if !local[path.Join(directory, name)] {
					return dsError
				}
			}
		} else if a.Manifest != nil || a.Signature != nil || a.SignerPolicySHA != nil {
			return dsError
		}
		refs := []string{a.InstallRecipe, a.RemoveRecipe}
		if len(a.Notices) == 0 || len(a.SourceMaterial) == 0 || len(a.BuildEvidence) == 0 {
			return dsError
		}
		refs = append(refs, a.Notices...)
		refs = append(refs, a.BuildEvidence...)
		for _, receipt := range a.Receipts {
			if !local[receipt] {
				return dsError
			}
		}
		if a.Vendor != nil {
			refs = append(refs, a.Vendor.Provenance)
		}
		for _, p := range refs {
			f, exists := files[p]
			if !exists || f.Size > dsMetadataLimit {
				return dsError
			}
			referenced[p] = true
		}
		for _, p := range a.SourceMaterial {
			if _, exists := files[p]; !exists {
				return dsError
			}
			referenced[p] = true
		}
	}
	if len(referenced) != len(files) {
		return dsError
	}
	return nil
}
func dsCheckFiles(root string, index dsIndex) (int64, error) {
	expected := map[string]dsFile{}
	total := int64(0)
	for _, f := range index.Files {
		observed, e := dsObserve(root, f.Path)
		if e != nil || observed != f {
			return 0, dsError
		}
		expected[f.Path] = f
		total += f.Size
	}
	seen := 0
	entries := 0
	e := filepath.WalkDir(root, func(p string, entry os.DirEntry, err error) error {
		entries++
		if entries > 131072 {
			return dsError
		}
		if err != nil {
			return dsError
		}
		if p == root {
			return nil
		}
		rel, e := filepath.Rel(root, p)
		if e != nil {
			return dsError
		}
		rel = filepath.ToSlash(rel)
		info, e := entry.Info()
		if e != nil || info.Mode()&os.ModeSymlink != 0 {
			return dsError
		}
		if entry.IsDir() {
			if info.Mode().Perm()&0022 != 0 {
				return dsError
			}
			return nil
		}
		if rel == "delivery-set.json" || rel == "delivery-set.json.sig" {
			if !info.Mode().IsRegular() || info.Size() > dsMetadataLimit {
				return dsError
			}
			return nil
		}
		if _, ok := expected[rel]; !ok {
			return dsError
		}
		seen++
		return nil
	})
	if e != nil || seen != len(expected) {
		return 0, dsError
	}
	return total, nil
}
func dsWriteNew(p string, b []byte) error {
	if !filepath.IsAbs(p) {
		return dsError
	}
	f, e := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return dsError
	}
	n, writeErr := f.Write(b)
	syncErr := f.Sync()
	closeErr := f.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil || n != len(b) {
		return dsError
	}
	return nil
}
func dsJSON(v any) []byte { b, _ := json.MarshalIndent(v, "", "  "); return append(b, '\n') }
func dsPrepare(root string, definition dsDefinition, selection dsSelection, selectionRaw, policy []byte) ([]byte, error) {
	index := dsIndex{Protocol: "symphony.release.delivery-set.v1", SetID: definition.SetID, Source: selection.Source, Profile: selection.Profile, CapabilitySHA: selection.CapabilitySHA, SelectionSHA: dsHash(selectionRaw), DependencySHA: selection.DependencySHA, IdentitySHA: selection.IdentitySHA, PolicySHA: dsHash(policy), Files: []dsFile{}, Artifacts: definition.Artifacts}
	if len(definition.Files) == 0 || len(definition.Files) > 65536 {
		return nil, dsError
	}
	for _, p := range definition.Files {
		f, e := dsObserve(root, p)
		if e != nil {
			return nil, e
		}
		index.Files = append(index.Files, f)
	}
	sort.Slice(index.Files, func(i, j int) bool { return index.Files[i].Path < index.Files[j].Path })
	if dsValidate(index, selection, selectionRaw, policy, selection.Profile.ID) != nil {
		return nil, dsError
	}
	if _, e := dsCheckFiles(root, index); e != nil {
		return nil, e
	}
	return dsJSON(index), nil
}

type dsBoundedOutput struct{ bytes.Buffer }

func (b *dsBoundedOutput) Write(p []byte) (int, error) {
	if len(p) > 65536-b.Len() {
		return 0, dsError
	}
	return b.Buffer.Write(p)
}
func dsVerifySignature(index, signature, policy []byte, signer string) error {
	if !dsSigner(signer) || len(signature) == 0 || len(signature) > dsMetadataLimit {
		return dsError
	}
	work, e := os.MkdirTemp("", "symphony-delivery-signature-")
	if e != nil {
		return dsError
	}
	defer os.RemoveAll(work)
	policyFile := filepath.Join(work, "allowed-signers")
	signatureFile := filepath.Join(work, "index.sig")
	if dsWriteNew(policyFile, policy) != nil || dsWriteNew(signatureFile, signature) != nil {
		return dsError
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "ssh-keygen", "-Y", "verify", "-f", policyFile, "-I", signer, "-n", dsNamespace, "-s", signatureFile)
	command.Stdin = bytes.NewReader(index)
	var output dsBoundedOutput
	command.Stdout = &output
	command.Stderr = &output
	if command.Run() != nil {
		return dsError
	}
	return nil
}

type dsOptions struct{ mode, root, selection, policy, profile, signer, definition, output, artifact, ownerTool, ownerBuild, ownerRepository, ownerWork, nativePolicy, nativeSigner string }

func dsRun(options dsOptions) (dsResult, error) {
	var result dsResult
	root, e := dsRoot(options.root)
	if e != nil {
		return result, e
	}
	if options.output != "" {
		if !filepath.IsAbs(options.output) {
			return result, dsError
		}
		parent, e := filepath.EvalSymlinks(filepath.Dir(options.output))
		if e != nil {
			return result, dsError
		}
		rel, e := filepath.Rel(root, filepath.Join(parent, filepath.Base(options.output)))
		if e != nil || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
			return result, dsError
		}
		if _, e := os.Lstat(options.output); !os.IsNotExist(e) {
			return result, dsError
		}
	}
	selectionRaw, e := dsIndependent(root, options.selection)
	if e != nil {
		return result, e
	}
	policy, e := dsIndependent(root, options.policy)
	if e != nil {
		return result, e
	}
	var selection dsSelection
	if dsDecode(selectionRaw, &selection) != nil || !dsSelectionOK(selection) || selection.Profile.ID != options.profile {
		return result, dsError
	}
	var raw []byte
	if options.mode == "prepare" {
		definitionRaw, e := dsIndependent(root, options.definition)
		if e != nil {
			return result, e
		}
		var definition dsDefinition
		if dsDecode(definitionRaw, &definition) != nil {
			return result, dsError
		}
		raw, e = dsPrepare(root, definition, selection, selectionRaw, policy)
		if e != nil {
			return result, e
		}
		for _, name := range []string{"delivery-set.json", "delivery-set.json.sig"} {
			if _, e := os.Lstat(filepath.Join(root, name)); !os.IsNotExist(e) {
				return result, dsError
			}
		}
		if dsWriteNew(filepath.Join(root, "delivery-set.json"), raw) != nil {
			return result, dsError
		}
	} else if options.mode == "validate" || options.mode == "verify" {
		raw, e = dsRead(filepath.Join(root, "delivery-set.json"), dsMetadataLimit)
		if e != nil {
			return result, e
		}
	} else {
		return result, dsError
	}
	var index dsIndex
	if dsDecode(raw, &index) != nil || dsValidate(index, selection, selectionRaw, policy, options.profile) != nil {
		return result, dsError
	}
	if options.mode == "verify" {
		signature, e := dsRead(filepath.Join(root, "delivery-set.json.sig"), dsMetadataLimit)
		if e != nil || dsVerifySignature(raw, signature, policy, options.signer) != nil {
			return result, dsError
		}
	}
	total, e := dsCheckFiles(root, index)
	if e != nil {
		return result, e
	}
	// A stable second observation prevents a changed index from inheriting this result.
	after, e := dsRead(filepath.Join(root, "delivery-set.json"), dsMetadataLimit)
	if e != nil || !bytes.Equal(raw, after) {
		return result, dsError
	}
	scope := "unsigned_bytes_only"
	if options.mode == "verify" {
		scope = "signed_bytes_only"
	}
	pending := []string{}
	for _, artifact := range index.Artifacts {
		pending = append(pending, artifact.Selection.ID+":"+artifact.Selection.Adapter)
	}
	result = dsResult{Protocol: "symphony.release.delivery-verification.v1", Scope: scope, IndexSHA: dsHash(raw), SelectionSHA: dsHash(selectionRaw), PolicySHA: dsHash(policy), Source: index.Source, Profile: index.Profile, VerifiedFiles: len(index.Files), VerifiedBytes: total, SignatureValid: options.mode == "verify", Missing: pending}
	if options.output != "" {
		if dsWriteNew(options.output, dsJSON(result)) != nil {
			return dsResult{}, dsError
		}
	}
	return result, nil
}

// dsOwnerBuild is supplied by the operator independently of the downloaded set.
// It binds a trusted, reviewed harness build. An artifact cannot select a program.
type dsOwnerBuild struct {
	Protocol         string   `json:"protocol"`
	Adapter          string   `json:"verification_adapter"`
	Component        string   `json:"component_id"`
	Version          string   `json:"version"`
	OS               string   `json:"os"`
	Architecture     string   `json:"architecture"`
	Source           dsSource `json:"source"`
	SourceArchiveSHA string   `json:"source_archive_sha256"`
	ToolSHA          string   `json:"tool_sha256"`
	GoVersion        string   `json:"go_version"`
}
type dsOwnerInspection struct {
	Protocol      string `json:"protocol"`
	Adapter       string `json:"verification_adapter"`
	Component     string `json:"component_id"`
	Version       string `json:"version"`
	OS            string `json:"os"`
	Architecture  string `json:"architecture"`
	Binary        dsFile `json:"binary"`
	Receipt       dsFile `json:"receipt"`
	ReceiptDigest string `json:"receipt_digest"`
	Scope         string `json:"scope"`
	Provider      bool   `json:"provider_qualified"`
	FreshHost     bool   `json:"fresh_host_qualified"`
	Release       bool   `json:"release_accepted"`
	Publication   bool   `json:"publication_authorized"`
}
type dsOwnerResult struct {
	Protocol      string            `json:"protocol"`
	Scope         string            `json:"scope"`
	IndexSHA      string            `json:"index_sha256"`
	ArtifactID    string            `json:"artifact_id"`
	Source        dsSource          `json:"source"`
	Profile       dsProfile         `json:"profile"`
	ToolBuildSHA  string            `json:"tool_build_sha256"`
	ToolSHA       string            `json:"tool_sha256"`
	InspectionSHA string            `json:"inspection_sha256"`
	Inspection    dsOwnerInspection `json:"inspection"`
	OwnerVerified bool              `json:"selected_artifact_owner_verified"`
	FreshHost     bool              `json:"fresh_host_verified"`
	Release       bool              `json:"release_acceptance_complete"`
	Publication   bool              `json:"publication_authorized"`
}

func dsOwnerComponent(adapter string) string {
	switch adapter {
	case "ssiag-receipt-v2":
		return "secure-identity-access-governance"
	case "stav-receipt-v2":
		return "stav-append-authority"
	case "accordare-receipt-v2":
		return "accordare-stav-producer"
	}
	return ""
}
func dsOwnerBinding(index dsIndex, artifact dsArtifact, inspection dsOwnerInspection) error {
	component := dsOwnerComponent(artifact.Selection.Adapter)
	if component == "" || artifact.Selection.Kind != "go" || artifact.Selection.PackageID != component || len(artifact.Receipts) != 1 ||
		inspection.Protocol != "symphony.release.owner-inspection.v1" || inspection.Adapter != artifact.Selection.Adapter ||
		inspection.Component != component || inspection.Version != artifact.Selection.Versions[component] ||
		inspection.OS != index.Profile.OS || inspection.Architecture != index.Profile.Architecture ||
		inspection.Scope != "read_only_owner_receipt" || inspection.Provider || inspection.FreshHost || inspection.Release || inspection.Publication ||
		!strings.HasPrefix(inspection.ReceiptDigest, "sha256:") || !dsHex(strings.TrimPrefix(inspection.ReceiptDigest, "sha256:"), 32) {
		return dsError
	}
	files := map[string]dsFile{}
	for _, f := range index.Files {
		files[f.Path] = f
	}
	// A bundle may contain its owner prefix below a staging directory. Derive only
	// that prefix from the independently inspected owner's exact relative path.
	if !dsPath(inspection.Binary.Path) || !dsPath(inspection.Receipt.Path) ||
		!(artifact.Primary == inspection.Binary.Path || strings.HasSuffix(artifact.Primary, "/"+inspection.Binary.Path)) {
		return dsError
	}
	prefix := strings.TrimSuffix(artifact.Primary, inspection.Binary.Path)
	if prefix+inspection.Receipt.Path != artifact.Receipts[0] {
		return dsError
	}
	for _, binding := range []struct {
		name     string
		observed dsFile
	}{{artifact.Primary, inspection.Binary}, {artifact.Receipts[0], inspection.Receipt}} {
		f, ok := files[binding.name]
		observed := binding.observed
		observed.Path = binding.name
		if !ok || f != observed {
			return dsError
		}
	}
	return nil
}
func dsInspectOwner(options dsOptions) (dsOwnerResult, error) {
	var result dsOwnerResult
	if options.ownerRepository != "" || options.ownerWork != "" || options.nativePolicy != "" || options.nativeSigner != "" {
		return result, dsError
	}
	output := options.output
	options.output = ""
	options.mode = "verify"
	verified, e := dsRun(options)
	if e != nil {
		return result, e
	}
	root, e := dsRoot(options.root)
	if e != nil {
		return result, e
	}
	raw, e := dsRead(filepath.Join(root, "delivery-set.json"), dsMetadataLimit)
	if e != nil || dsHash(raw) != verified.IndexSHA {
		return result, dsError
	}
	var index dsIndex
	if dsDecode(raw, &index) != nil {
		return result, dsError
	}
	var artifact dsArtifact
	found := false
	for _, a := range index.Artifacts {
		if a.Selection.ID == options.artifact {
			artifact = a
			found = true
		}
	}
	component := dsOwnerComponent(artifact.Selection.Adapter)
	if !found || component == "" {
		return result, dsError
	}
	buildRaw, e := dsIndependent(root, options.ownerBuild)
	if e != nil {
		return result, e
	}
	var build dsOwnerBuild
	if dsDecode(buildRaw, &build) != nil || build.Protocol != "symphony.release.owner-tool-build.v1" ||
		build.Adapter != artifact.Selection.Adapter || build.Component != component ||
		build.Version != artifact.Selection.Versions[component] || build.OS != index.Profile.OS || build.Architecture != index.Profile.Architecture ||
		build.Source != index.Source || !dsHex(build.SourceArchiveSHA, 32) || !dsHex(build.ToolSHA, 32) || build.GoVersion != "go1.26.5" {
		return result, dsError
	}
	// Both the trusted executable and build record must be outside the delivery.
	if !filepath.IsAbs(options.ownerTool) || filepath.Clean(options.ownerTool) != options.ownerTool {
		return result, dsError
	}
	toolParent, e := dsRoot(filepath.Dir(options.ownerTool))
	if e != nil {
		return result, e
	}
	rel, e := filepath.Rel(root, filepath.Join(toolParent, filepath.Base(options.ownerTool)))
	if e != nil || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return result, dsError
	}
	tool, e := dsObserve(toolParent, filepath.Base(options.ownerTool))
	if e != nil || tool.SHA256 != build.ToolSHA || tool.Mode&0111 == 0 {
		return result, dsError
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, filepath.Join(toolParent, filepath.Base(options.ownerTool)), "-binary", filepath.Join(root, filepath.FromSlash(artifact.Primary)))
	command.Env = []string{"PATH=/usr/bin:/bin", "LANG=C"}
	var stdout, stderr dsBoundedOutput
	command.Stdout = &stdout
	command.Stderr = &stderr
	if command.Run() != nil {
		return result, dsError
	}
	inspectionRaw := stdout.Bytes()
	var inspection dsOwnerInspection
	if dsDecode(inspectionRaw, &inspection) != nil || dsOwnerBinding(index, artifact, inspection) != nil {
		return result, dsError
	}
	after, e := dsObserve(toolParent, filepath.Base(options.ownerTool))
	if e != nil || after != tool {
		return result, dsError
	}
	afterBuild, e := dsIndependent(root, options.ownerBuild)
	if e != nil || !bytes.Equal(buildRaw, afterBuild) {
		return result, dsError
	}
	// Reverify the entire signed set after the owner reader; no stale byte proof.
	again, e := dsRun(options)
	if e != nil || again.IndexSHA != verified.IndexSHA {
		return result, dsError
	}
	result = dsOwnerResult{Protocol: "symphony.release.delivery-owner-verification.v1", Scope: "selected_owner_receipt_only", IndexSHA: verified.IndexSHA, ArtifactID: artifact.Selection.ID, Source: index.Source, Profile: index.Profile, ToolBuildSHA: dsHash(buildRaw), ToolSHA: tool.SHA256, InspectionSHA: dsHash(inspectionRaw), Inspection: inspection, OwnerVerified: true}
	if output != "" {
		if !filepath.IsAbs(output) {
			return dsOwnerResult{}, dsError
		}
		parent, e := dsRoot(filepath.Dir(output))
		if e != nil {
			return dsOwnerResult{}, e
		}
		rel, e := filepath.Rel(root, filepath.Join(parent, filepath.Base(output)))
		if e != nil || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) || dsWriteNew(output, dsJSON(result)) != nil {
			return dsOwnerResult{}, dsError
		}
	}
	return result, nil
}

var dsNativeSources = []string{
	"tests/launch-acceptance/candidate-host.go",
	"cmake/release/VerifyCandidate.cmake",
	"cmake/release/Support.cmake",
	"cmake/release/StrictJSON.cmake",
	"cmake/SymphonyReceiptV2Support.cmake",
}

type dsNativeTool struct {
	Protocol  string            `json:"protocol"`
	ToolSHA   string            `json:"tool_sha256"`
	GoVersion string            `json:"go_version"`
	Sources   map[string]string `json:"source_files_sha256"`
}
type dsNativeHost struct {
	Protocol    string `json:"protocol"`
	Status      string `json:"status"`
	Publication bool   `json:"publication_authorized"`
	Release     bool   `json:"release_acceptance_complete"`
	FreshHost   bool   `json:"fresh_host_verified"`
	Provider    bool   `json:"provider_qualification_complete"`
	Commit      string `json:"source_commit"`
	Tree        string `json:"source_tree"`
	ArtifactSHA string `json:"artifact_sha256"`
	ManifestSHA string `json:"inventory_sha256"`
	PolicySHA   string `json:"allowed_signers_sha256"`
	Profile     string `json:"profile_id"`
	HostOS      string `json:"host_os"`
	HostArch    string `json:"host_arch"`
	Kernel      string `json:"host_kernel"`
	PreparedAt  string `json:"prepared_at"`
	FileCount   int    `json:"payload_file_count"`
	Remaining   string `json:"remaining"`
}
type dsNativeResult struct {
	Protocol         string            `json:"protocol"`
	Scope            string            `json:"scope"`
	IndexSHA         string            `json:"index_sha256"`
	ArtifactID       string            `json:"artifact_id"`
	Source           dsSource          `json:"source"`
	Profile          dsProfile         `json:"profile"`
	ArtifactSHA      string            `json:"artifact_sha256"`
	ManifestSHA      string            `json:"manifest_sha256"`
	PolicySHA        string            `json:"artifact_signer_policy_sha256"`
	ToolSHA          string            `json:"tool_sha256"`
	ToolRecordSHA    string            `json:"tool_record_sha256"`
	ToolSources      map[string]string `json:"tool_source_files_sha256"`
	HostReportSHA    string            `json:"host_preparation_sha256"`
	PayloadReportSHA string            `json:"payload_verification_sha256"`
	PayloadFileCount int               `json:"payload_file_count"`
	OwnerVerified    bool              `json:"selected_artifact_owner_verified"`
	FreshHost        bool              `json:"fresh_host_verified"`
	Release          bool              `json:"release_acceptance_complete"`
	Publication      bool              `json:"publication_authorized"`
}

func dsOutside(root, p string) bool {
	if !filepath.IsAbs(p) || filepath.Clean(p) != p {
		return false
	}
	parent, e := filepath.EvalSymlinks(filepath.Dir(p))
	if e != nil {
		return false
	}
	rel, e := filepath.Rel(root, filepath.Join(parent, filepath.Base(p)))
	return e == nil && (rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
func dsNativeToolOK(root, repository, tool string, record dsNativeTool) error {
	if record.Protocol != "symphony.release.native-verifier-tool.v1" || record.GoVersion != "go1.26.5" ||
		!dsHex(record.ToolSHA, 32) || len(record.Sources) != len(dsNativeSources) ||
		!dsOutside(root, repository) || !dsOutside(root, tool) {
		return dsError
	}
	for _, p := range dsNativeSources {
		expected, ok := record.Sources[p]
		observed, e := dsObserve(repository, p)
		if !ok || !dsHex(expected, 32) || e != nil || observed.SHA256 != expected {
			return dsError
		}
	}
	observed, e := dsObserve(filepath.Dir(tool), filepath.Base(tool))
	if e != nil || observed.Mode&0111 == 0 || observed.SHA256 != record.ToolSHA {
		return dsError
	}
	return nil
}
func dsNativeInventory(index dsIndex, artifact dsArtifact, raw []byte) error {
	var manifest map[string]any
	if dsDecode(raw, &manifest) != nil {
		return dsError
	}
	source, _ := manifest["source"].(map[string]any)
	platform, _ := manifest["platform"].(map[string]any)
	if manifest["protocol"] != "symphony.release.candidate.v1" || manifest["publication_authorized"] != false ||
		source["commit"] != index.Source.Commit || source["tree"] != index.Source.Tree || platform["profile_id"] != index.Profile.ID ||
		platform["os"] != index.Profile.OS || platform["architecture"] != index.Profile.Architecture ||
		manifest["dependency_map_sha256"] != index.DependencySHA || manifest["identity_catalog_sha256"] != index.IdentitySHA {
		return dsError
	}
	rows, ok := manifest["components"].([]any)
	if !ok || len(rows) != len(artifact.Selection.Versions) {
		return dsError
	}
	seen := map[string]bool{}
	for _, row := range rows {
		component, ok := row.(map[string]any)
		if !ok {
			return dsError
		}
		id, ok := component["component_id"].(string)
		version, versionOK := component["version"].(string)
		if !ok || !versionOK || seen[id] || artifact.Selection.Versions[id] != version {
			return dsError
		}
		seen[id] = true
	}
	return nil
}
func dsNativeBinding(index dsIndex, artifact dsArtifact, host dsNativeHost) error {
	files := map[string]dsFile{}
	for _, f := range index.Files {
		files[f.Path] = f
	}
	if artifact.Selection.Kind != "native" || artifact.Selection.Adapter != "native-candidate-v1" ||
		artifact.Manifest == nil || artifact.SignerPolicySHA == nil ||
		host.Protocol != "symphony.release.host-preparation.v1" || host.Status != "verified_extraction_only" ||
		host.Publication || host.Release || host.FreshHost || host.Provider || host.FileCount <= 0 || host.FileCount > 65536 ||
		host.Commit != index.Source.Commit || host.Tree != index.Source.Tree || host.Profile != index.Profile.ID ||
		host.ArtifactSHA != files[artifact.Primary].SHA256 || host.ManifestSHA != files[*artifact.Manifest].SHA256 || host.PolicySHA != *artifact.SignerPolicySHA {
		return dsError
	}
	return nil
}
func dsInspectNative(options dsOptions) (dsNativeResult, error) {
	var result dsNativeResult
	options.mode = "verify"
	output := options.output
	options.output = ""
	verified, e := dsRun(options)
	if e != nil {
		return result, e
	}
	root, e := dsRoot(options.root)
	if e != nil {
		return result, e
	}
	raw, e := dsRead(filepath.Join(root, "delivery-set.json"), dsMetadataLimit)
	if e != nil || dsHash(raw) != verified.IndexSHA {
		return result, dsError
	}
	var index dsIndex
	if dsDecode(raw, &index) != nil {
		return result, dsError
	}
	var artifact dsArtifact
	for _, a := range index.Artifacts {
		if a.Selection.ID == options.artifact {
			artifact = a
		}
	}
	if artifact.Selection.Kind != "native" || artifact.Selection.Adapter != "native-candidate-v1" || artifact.SignerPolicySHA == nil {
		return result, dsError
	}
	policy, e := dsIndependent(root, options.nativePolicy)
	if e != nil || dsHash(policy) != *artifact.SignerPolicySHA || !dsSigner(options.nativeSigner) {
		return result, dsError
	}
	buildRaw, e := dsIndependent(root, options.ownerBuild)
	if e != nil {
		return result, e
	}
	var build dsNativeTool
	if dsDecode(buildRaw, &build) != nil {
		return result, dsError
	}
	repository, e := dsRoot(options.ownerRepository)
	if e != nil || dsNativeToolOK(root, repository, options.ownerTool, build) != nil {
		return result, dsError
	}
	if !dsOutside(root, options.ownerWork) || !dsOutside(repository, options.ownerWork) {
		return result, dsError
	}
	if _, e = os.Lstat(options.ownerWork); !os.IsNotExist(e) {
		return result, dsError
	}
	if output != "" && (!dsOutside(root, output) || output == options.ownerWork) {
		return result, dsError
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	// The only program is the independent reviewed candidate-host tool; all script
	// names are fixed in that tool and hash-bound above. No archive commands run.
	command := exec.CommandContext(ctx, options.ownerTool, "-repository", repository, "-candidate", filepath.Dir(filepath.Join(root, artifact.Primary)),
		"-allowed-signers", options.nativePolicy, "-signer", options.nativeSigner, "-profile", options.profile, "-work", options.ownerWork)
	command.Env = []string{"PATH=" + os.Getenv("PATH"), "LANG=C"}
	var stdout, stderr dsBoundedOutput
	command.Stdout = &stdout
	command.Stderr = &stderr
	if command.Run() != nil {
		return result, dsError
	}
	hostRaw, e := dsRead(filepath.Join(options.ownerWork, "host-preparation.json"), dsMetadataLimit)
	if e != nil {
		return result, e
	}
	var host dsNativeHost
	if dsDecode(hostRaw, &host) != nil || dsNativeBinding(index, artifact, host) != nil {
		return result, dsError
	}
	payloadRaw, e := dsRead(filepath.Join(options.ownerWork, "unpacked.json"), dsMetadataLimit)
	if e != nil {
		return result, e
	}
	// candidate-host already runs VerifyCandidate against the exact extracted
	// receipt closure. Bind its output to the same manifest/policy/profile again.
	var proof map[string]any
	if dsDecode(payloadRaw, &proof) != nil {
		return result, dsError
	}
	for field, want := range map[string]string{"protocol": "symphony.release.candidate-verification.v1", "scope": "caller_supplied_payload",
		"source_commit": host.Commit, "source_tree": host.Tree, "profile_id": host.Profile, "artifact_sha256": host.ArtifactSHA,
		"inventory_sha256": host.ManifestSHA, "allowed_signers_sha256": host.PolicySHA} {
		if proof[field] != want {
			return result, dsError
		}
	}
	if proof["publication_authorized"] != false || proof["signature_scheme"] != "sshsig" || proof["namespace"] != "symphony-release" || proof["signer_principal"] != options.nativeSigner {
		return result, dsError
	}
	manifestRaw, e := dsRead(filepath.Join(root, *artifact.Manifest), dsMetadataLimit)
	if e != nil || dsHash(manifestRaw) != host.ManifestSHA || dsNativeInventory(index, artifact, manifestRaw) != nil {
		return result, dsError
	}
	checksums, e := dsRead(filepath.Join(root, path.Dir(artifact.Primary), "SHA256SUMS"), dsMetadataLimit)
	if e != nil || string(checksums) != fmt.Sprintf("%s  candidate.tar\n%s  candidate.json\n", host.ArtifactSHA, host.ManifestSHA) {
		return result, dsError
	}
	if dsNativeToolOK(root, repository, options.ownerTool, build) != nil {
		return result, dsError
	}
	afterBuild, e := dsIndependent(root, options.ownerBuild)
	if e != nil || !bytes.Equal(afterBuild, buildRaw) {
		return result, dsError
	}
	afterPolicy, e := dsIndependent(root, options.nativePolicy)
	if e != nil || !bytes.Equal(afterPolicy, policy) {
		return result, dsError
	}
	again, e := dsRun(options)
	if e != nil || again.IndexSHA != verified.IndexSHA {
		return result, dsError
	}
	result = dsNativeResult{Protocol: "symphony.release.delivery-native-verification.v1", Scope: "selected_native_extraction_and_receipts",
		IndexSHA: verified.IndexSHA, ArtifactID: artifact.Selection.ID, Source: index.Source, Profile: index.Profile, ArtifactSHA: host.ArtifactSHA,
		ManifestSHA: host.ManifestSHA, PolicySHA: host.PolicySHA, ToolSHA: build.ToolSHA, ToolRecordSHA: dsHash(buildRaw), ToolSources: build.Sources,
		HostReportSHA: dsHash(hostRaw), PayloadReportSHA: dsHash(payloadRaw), PayloadFileCount: host.FileCount, OwnerVerified: true}
	if output != "" && dsWriteNew(output, dsJSON(result)) != nil {
		return dsNativeResult{}, dsError
	}
	return result, nil
}

type dsExecutableBuild struct {
	Protocol        string  `json:"protocol"`
	Component       string  `json:"component_id"`
	Version         string  `json:"version"`
	Binary          string  `json:"binary_name"`
	Commit          string  `json:"source_commit"`
	Tree            string  `json:"source_tree"`
	Epoch           string  `json:"source_date_epoch"`
	DependencySHA   string  `json:"dependency_map_sha256"`
	IdentitySHA     string  `json:"release_identity_catalog_sha256"`
	ModSHA          string  `json:"go_mod_sha256"`
	SumSHA          *string `json:"go_sum_sha256"`
	Recipe          string  `json:"build_recipe"`
	GoVersion       string  `json:"go_version"`
	GOOS            string  `json:"goos"`
	GOARCH          string  `json:"goarch"`
	ExecutableSHA   string  `json:"executable_sha256"`
	Builds          int     `json:"isolated_builds"`
	Reproducibility string  `json:"reproducibility"`
	Publication     bool    `json:"publication_authorized"`
}
type dsExecutableInventory struct {
	Protocol string   `json:"protocol"`
	Files    []dsFile `json:"files"`
}
type dsQxctlInfo struct {
	Protocol      string  `json:"protocol"`
	Component     string  `json:"component_id"`
	Version       string  `json:"version"`
	SourceState   string  `json:"source_state"`
	Commit        *string `json:"source_commit"`
	Tree          *string `json:"source_tree"`
	Epoch         *string `json:"source_date_epoch"`
	DependencySHA *string `json:"dependency_map_sha256"`
	Recipe        *string `json:"build_recipe"`
	GoVersion     string  `json:"go_version"`
	GOOS          string  `json:"goos"`
	GOARCH        string  `json:"goarch"`
	ExecutableSHA string  `json:"executable_sha256"`
}
type dsExecutableResult struct {
	Protocol         string    `json:"protocol"`
	Scope            string    `json:"scope"`
	IndexSHA         string    `json:"index_sha256"`
	ArtifactID       string    `json:"artifact_id"`
	Source           dsSource  `json:"source"`
	Profile          dsProfile `json:"profile"`
	BuildIdentitySHA string    `json:"build_identity_sha256"`
	InventorySHA     string    `json:"payload_inventory_sha256"`
	ExecutableSHA    string    `json:"executable_sha256"`
	VersionOutputSHA string    `json:"version_output_sha256"`
	VersionOutput    string    `json:"version_output"`
	Component        string    `json:"component_id"`
	Version          string    `json:"version"`
	OwnerVerified    bool      `json:"selected_artifact_owner_verified"`
	FreshHost        bool      `json:"fresh_host_verified"`
	Release          bool      `json:"release_acceptance_complete"`
	Publication      bool      `json:"publication_authorized"`
}

func dsExecutableComponent(a dsArtifact) string {
	component := a.Selection.PackageID
	if (component == "qxctl" && a.Selection.Adapter == "qxctl-files-v1") ||
		((component == "sov-stdio-adapter" || component == "sov-ssh-adapter") && a.Selection.Adapter == "sov-executable-v1") {
		return component
	}
	return ""
}
func dsExecutableBinding(index dsIndex, a dsArtifact, build dsExecutableBuild, inventory dsExecutableInventory) error {
	component := dsExecutableComponent(a)
	osName := index.Profile.OS
	if osName == "macos" {
		osName = "darwin"
	}
	epoch, e := strconv.ParseInt(build.Epoch, 10, 64)
	files := map[string]dsFile{}
	for _, f := range index.Files {
		files[f.Path] = f
	}
	if component == "" || a.Selection.Kind != "go" || len(a.Selection.Versions) != 1 || len(a.Receipts) != 0 ||
		build.Protocol != "symphony.artifact-build-identity.v1" || build.Component != component || build.Binary != component ||
		build.Version != a.Selection.Versions[component] || build.Commit != index.Source.Commit || build.Tree != index.Source.Tree ||
		build.DependencySHA != index.DependencySHA || build.IdentitySHA != index.IdentitySHA ||
		!dsHex(build.ModSHA, 32) || (build.SumSHA != nil && !dsHex(*build.SumSHA, 32)) ||
		e != nil || epoch < 0 || strconv.FormatInt(epoch, 10) != build.Epoch || build.GoVersion != "go1.26.5" ||
		build.GOOS != osName || build.GOARCH != index.Profile.Architecture || build.ExecutableSHA != files[a.Primary].SHA256 || build.Publication {
		return dsError
	}
	if component == "qxctl" {
		if build.Recipe != "symphony.qxctl.reproducible-build.v1" || build.Builds != 2 || build.Reproducibility != "verified_on_recorded_profile" {
			return dsError
		}
	} else if build.Recipe != "symphony.go.candidate-build.v1" || build.Builds != 1 || build.Reproducibility != "not_assessed" {
		return dsError
	}
	if inventory.Protocol != "symphony.release.immutable-files.v1" || len(inventory.Files) != len(a.Files)-1 || len(inventory.Files) == 0 {
		return dsError
	}
	owned := map[string]bool{}
	for _, p := range a.Files {
		if p != a.Inventory {
			owned[p] = true
		}
	}
	for _, f := range inventory.Files {
		if !owned[f.Path] || files[f.Path] != f {
			return dsError
		}
		delete(owned, f.Path)
	}
	if len(owned) != 0 {
		return dsError
	}
	return nil
}
func dsQxctlBinding(build dsExecutableBuild, raw []byte) error {
	var info dsQxctlInfo
	if dsDecode(raw, &info) != nil || info.Protocol != "symphony.qxctl.build-info.v1" || info.Component != "qxctl" ||
		info.Version != build.Version || info.SourceState != "stamped" || info.GoVersion != build.GoVersion ||
		info.GOOS != build.GOOS || info.GOARCH != build.GOARCH || info.ExecutableSHA != build.ExecutableSHA {
		return dsError
	}
	for _, pair := range []struct {
		actual   *string
		expected string
	}{{info.Commit, build.Commit}, {info.Tree, build.Tree}, {info.Epoch, build.Epoch}, {info.DependencySHA, build.DependencySHA}, {info.Recipe, build.Recipe}} {
		if pair.actual == nil || *pair.actual != pair.expected {
			return dsError
		}
	}
	return nil
}
func dsInspectExecutable(options dsOptions) (dsExecutableResult, error) {
	var result dsExecutableResult
	if options.ownerTool != "" || options.ownerBuild != "" || options.ownerRepository != "" || options.ownerWork != "" || options.nativePolicy != "" || options.nativeSigner != "" {
		return result, dsError
	}
	output := options.output
	options.output = ""
	options.mode = "verify"
	verified, e := dsRun(options)
	if e != nil {
		return result, e
	}
	root, e := dsRoot(options.root)
	if e != nil {
		return result, e
	}
	if output != "" {
		if !dsOutside(root, output) {
			return result, dsError
		}
		if _, e = os.Lstat(output); !os.IsNotExist(e) {
			return result, dsError
		}
	}
	raw, e := dsRead(filepath.Join(root, "delivery-set.json"), dsMetadataLimit)
	if e != nil || dsHash(raw) != verified.IndexSHA {
		return result, dsError
	}
	var index dsIndex
	if dsDecode(raw, &index) != nil {
		return result, dsError
	}
	var a dsArtifact
	for _, artifact := range index.Artifacts {
		if artifact.Selection.ID == options.artifact {
			a = artifact
		}
	}
	component := dsExecutableComponent(a)
	if component == "" || len(a.BuildEvidence) != 1 {
		return result, dsError
	}
	buildRaw, e := dsRead(filepath.Join(root, a.BuildEvidence[0]), dsMetadataLimit)
	if e != nil {
		return result, e
	}
	var build dsExecutableBuild
	if dsDecode(buildRaw, &build) != nil {
		return result, dsError
	}
	inventoryRaw, e := dsRead(filepath.Join(root, a.Inventory), dsMetadataLimit)
	if e != nil {
		return result, e
	}
	var inventory dsExecutableInventory
	if dsDecode(inventoryRaw, &inventory) != nil || dsExecutableBinding(index, a, build, inventory) != nil {
		return result, dsError
	}
	if runtime.GOOS != build.GOOS || runtime.GOARCH != build.GOARCH {
		return result, dsError
	}
	binary := filepath.Join(root, a.Primary)
	observed, e := dsObserve(root, a.Primary)
	if e != nil || observed.Mode&0111 == 0 {
		return result, dsError
	}
	info, e := buildinfo.ReadFile(binary)
	expectedPath := "github.com/QuanuX/Symphony/tools/" + component
	if component == "qxctl" {
		expectedPath += "/cmd/qxctl"
	}
	if e != nil || info.GoVersion != build.GoVersion || info.Path != expectedPath || info.Main.Replace != nil {
		return result, dsError
	}
	settings := map[string]string{}
	for _, setting := range info.Settings {
		if _, duplicate := settings[setting.Key]; duplicate {
			return result, dsError
		}
		settings[setting.Key] = setting.Value
	}
	if settings["GOOS"] != build.GOOS || settings["GOARCH"] != build.GOARCH || settings["CGO_ENABLED"] != "0" || settings["-trimpath"] != "true" {
		return result, dsError
	}
	for _, dependency := range info.Deps {
		if dependency.Replace != nil {
			return result, dsError
		}
	}
	work, e := os.MkdirTemp("", "symphony-version-inspection-")
	if e != nil {
		return result, e
	}
	defer os.RemoveAll(work)
	args := []string{"--version"}
	if component == "qxctl" {
		args = []string{"version", "--json"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, binary, args...)
	command.Dir = work
	command.Env = []string{"PATH=/usr/bin:/bin", "LANG=C", "HOME=" + work, "XDG_CONFIG_HOME=" + filepath.Join(work, "config"), "XDG_STATE_HOME=" + filepath.Join(work, "state"), "XDG_RUNTIME_DIR=" + filepath.Join(work, "runtime")}
	var stdout, stderr dsBoundedOutput
	command.Stdout = &stdout
	command.Stderr = &stderr
	if command.Run() != nil {
		return result, dsError
	}
	if component == "qxctl" {
		if dsQxctlBinding(build, stdout.Bytes()) != nil {
			return result, dsError
		}
	} else if string(stdout.Bytes()) != component+" version "+build.Version+"\n" {
		return result, dsError
	}
	entries, e := os.ReadDir(work)
	if e != nil || len(entries) != 0 {
		return result, dsError
	}
	again, e := dsRun(options)
	if e != nil || again.IndexSHA != verified.IndexSHA {
		return result, dsError
	}
	result = dsExecutableResult{Protocol: "symphony.release.delivery-executable-verification.v1", Scope: "selected_executable_identity_only", IndexSHA: verified.IndexSHA, ArtifactID: a.Selection.ID,
		Source: index.Source, Profile: index.Profile, BuildIdentitySHA: dsHash(buildRaw), InventorySHA: dsHash(inventoryRaw), ExecutableSHA: build.ExecutableSHA, VersionOutputSHA: dsHash(stdout.Bytes()), VersionOutput: stdout.String(), Component: component, Version: build.Version, OwnerVerified: true}
	if output != "" && dsWriteNew(output, dsJSON(result)) != nil {
		return dsExecutableResult{}, dsError
	}
	return result, nil
}
func main() {
	var o dsOptions
	flag.StringVar(&o.mode, "mode", "verify", "prepare, validate, verify, inspect-owner, inspect-native, or inspect-executable")
	flag.StringVar(&o.root, "root", "", "absolute private delivery-set directory")
	flag.StringVar(&o.selection, "selection", "", "independent exact artifact selection JSON outside delivery root")
	flag.StringVar(&o.policy, "allowed-signers", "", "independent allowed-signers policy outside delivery root")
	flag.StringVar(&o.profile, "profile", "", "explicit expected target profile")
	flag.StringVar(&o.signer, "signer", "", "expected signer principal for verification")
	flag.StringVar(&o.definition, "definition", "", "independent preparation definition outside delivery root")
	flag.StringVar(&o.output, "report", "", "optional new absolute result file, preferably outside delivery root")
	flag.StringVar(&o.artifact, "artifact", "", "explicit artifact ID for inspect-owner")
	flag.StringVar(&o.ownerTool, "owner-tool", "", "independent trusted read-only owner harness for inspect-owner")
	flag.StringVar(&o.ownerBuild, "owner-build", "", "independent exact owner harness build record for inspect-owner")
	flag.StringVar(&o.ownerRepository, "owner-repository", "", "independent reviewed native verifier source checkout")
	flag.StringVar(&o.ownerWork, "owner-work", "", "new private native extraction directory outside source and delivery")
	flag.StringVar(&o.nativePolicy, "native-allowed-signers", "", "independent native artifact signer policy")
	flag.StringVar(&o.nativeSigner, "native-signer", "", "independent native artifact signer principal")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, dsError)
		os.Exit(1)
	}
	var r any
	var e error
	if o.mode == "inspect-owner" {
		r, e = dsInspectOwner(o)
	} else if o.mode == "inspect-native" {
		r, e = dsInspectNative(o)
	} else if o.mode == "inspect-executable" {
		r, e = dsInspectExecutable(o)
	} else {
		if o.artifact != "" || o.ownerTool != "" || o.ownerBuild != "" || o.ownerRepository != "" || o.ownerWork != "" || o.nativePolicy != "" || o.nativeSigner != "" {
			fmt.Fprintln(os.Stderr, dsError)
			os.Exit(1)
		}
		r, e = dsRun(o)
	}
	if e != nil {
		fmt.Fprintln(os.Stderr, dsError)
		os.Exit(1)
	}
	fmt.Print(string(dsJSON(r)))
}
