// delivery-set binds a complete, explicitly selected multi-artifact delivery.
// This release tool does not install or execute payloads, access providers, or
// establish publisher authority, owner qualification, or release acceptance.
package main

import (
	"bytes"
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
	"reflect"
	"sort"
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
			if path.Base(a.Primary) != "candidate.tar" || *a.Manifest != path.Join(directory, "candidate.json") || *a.Signature != path.Join(directory, "candidate.json.sig") || len(a.Files) != 7 {
				return dsError
			}
			for _, name := range []string{"candidate.tar", "candidate.json", "candidate.json.sig", "platform-evidence.json", "build-evidence.json", "dependency-map.json", "identity-catalog.json"} {
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

type dsOptions struct{ mode, root, selection, policy, profile, signer, definition, output string }

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
func main() {
	var o dsOptions
	flag.StringVar(&o.mode, "mode", "verify", "prepare, validate unsigned bytes, or verify signature and bytes")
	flag.StringVar(&o.root, "root", "", "absolute private delivery-set directory")
	flag.StringVar(&o.selection, "selection", "", "independent exact artifact selection JSON outside delivery root")
	flag.StringVar(&o.policy, "allowed-signers", "", "independent allowed-signers policy outside delivery root")
	flag.StringVar(&o.profile, "profile", "", "explicit expected target profile")
	flag.StringVar(&o.signer, "signer", "", "expected signer principal for verification")
	flag.StringVar(&o.definition, "definition", "", "independent preparation definition outside delivery root")
	flag.StringVar(&o.output, "report", "", "optional new absolute result file, preferably outside delivery root")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, dsError)
		os.Exit(1)
	}
	r, e := dsRun(o)
	if e != nil {
		fmt.Fprintln(os.Stderr, dsError)
		os.Exit(1)
	}
	fmt.Print(string(dsJSON(r)))
}
