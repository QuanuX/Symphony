// Read-only candidate closure verification. This is not release authorization.
package main

import (
	"bytes"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"
)

const gcAccordareModule = "github.com/QuanuX/Symphony/modules/accordare-stav-producer"

var gcGitID = regexp.MustCompile(`^[0-9a-f]{40}$`)
var gcSHA = regexp.MustCompile(`^[0-9a-f]{64}$`)
var gcPseudo = regexp.MustCompile(`^v0\.1\.0-dev\.0\.[0-9]{14}-[0-9a-f]{12}$`)

type gcOrigin struct{ VCS, URL, Subdir, Hash, Ref string }
type gcDownload struct {
	Path, Version, Sum, GoModSum string
	Origin                       *gcOrigin
	Error                        string
}
type gcSelection struct {
	Protocol    string    `json:"protocol"`
	Component   string    `json:"component_id"`
	Module      string    `json:"module"`
	Version     string    `json:"version"`
	Commit      string    `json:"source_commit"`
	Sum         string    `json:"module_sum"`
	GoModSum    string    `json:"go_mod_sum"`
	Origin      *gcOrigin `json:"origin"`
	Released    bool      `json:"public_release_completed"`
	Publication bool      `json:"publication_authorized"`
}
type gcIdentity struct {
	Protocol      string  `json:"protocol"`
	Component     string  `json:"component_id"`
	Version       string  `json:"version"`
	BinaryName    string  `json:"binary_name"`
	Commit        string  `json:"source_commit"`
	Tree          string  `json:"source_tree"`
	DependencyMap string  `json:"dependency_map_sha256"`
	Catalog       string  `json:"release_identity_catalog_sha256"`
	GoMod         string  `json:"go_mod_sha256"`
	GoSum         *string `json:"go_sum_sha256"`
	Recipe        string  `json:"build_recipe"`
	GoVersion     string  `json:"go_version"`
	OS            string  `json:"goos"`
	Arch          string  `json:"goarch"`
	Executable    string  `json:"executable_sha256"`
	Publication   bool    `json:"publication_authorized"`
}
type gcArtifact struct {
	Component string `json:"component_id"`
	Path      string `json:"path"`
	Main      string `json:"main_package"`
	Binary    string `json:"binary_name"`
	Version   string `json:"package_version"`
	Recipe    string `json:"recipe"`
}
type gcCatalog struct {
	Artifacts []gcArtifact `json:"go_artifacts"`
}
type gcDependency struct {
	Path    string `json:"module"`
	Version string `json:"version"`
	Sum     string `json:"sum"`
}
type gcUnit struct {
	Component     string         `json:"component_id"`
	NativeVersion string         `json:"native_version"`
	Executable    string         `json:"executable_sha256"`
	Size          int64          `json:"executable_size"`
	Identity      string         `json:"artifact_identity_sha256"`
	Module        string         `json:"main_module"`
	GoMod         string         `json:"go_mod_sha256"`
	GoSum         *string        `json:"go_sum_sha256"`
	Recipe        string         `json:"build_recipe"`
	Dependencies  []gcDependency `json:"linked_go_modules"`
}
type gcResult struct {
	Protocol               string   `json:"protocol"`
	Commit                 string   `json:"source_commit"`
	Tree                   string   `json:"source_tree"`
	OS                     string   `json:"goos"`
	Arch                   string   `json:"goarch"`
	GoVersion              string   `json:"go_version"`
	OwnerSource            string   `json:"accordare_owner_source_commit"`
	OwnerModuleVersion     string   `json:"accordare_go_module_version"`
	OwnerModuleSum         string   `json:"accordare_go_module_sum"`
	OwnerGoModSum          string   `json:"accordare_go_mod_sum"`
	DownloadEvidence       string   `json:"module_download_evidence_sha256"`
	OwnerSelection         string   `json:"owner_source_selection_sha256"`
	DependencyMap          string   `json:"dependency_map_sha256"`
	Catalog                string   `json:"release_identity_catalog_sha256"`
	Units                  []gcUnit `json:"units"`
	Status                 string   `json:"status"`
	PublicReleaseCompleted bool     `json:"public_release_completed"`
	ReleaseAcceptance      bool     `json:"release_acceptance_complete"`
	Publication            bool     `json:"publication_authorized"`
}

func gcHash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func gcOutputPath(repo, output string) (string, error) {
	if !filepath.IsAbs(output) || filepath.Clean(output) != output {
		return "", fmt.Errorf("candidate evidence output must be a new absolute file outside source")
	}
	root, err := filepath.EvalSymlinks(repo)
	if err != nil {
		return "", err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return "", err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(output))
	if err != nil {
		return "", err
	}
	canonical := filepath.Join(parent, filepath.Base(output))
	relative, err := filepath.Rel(root, canonical)
	if err != nil || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("candidate evidence output would modify selected source")
	}
	if _, err := os.Lstat(canonical); !os.IsNotExist(err) {
		return "", fmt.Errorf("candidate evidence output must not already exist")
	}
	return canonical, nil
}
func gcRead(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > limit {
		return nil, fmt.Errorf("unsafe or unbounded candidate evidence")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, fmt.Errorf("candidate evidence read failed")
	}
	return data, nil
}
func gcJSON(data []byte, out any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	if err := d.Decode(out); err != nil {
		return err
	}
	if d.Decode(&struct{}{}) != io.EOF {
		return fmt.Errorf("trailing JSON evidence")
	}
	return nil
}
func gcGit(repo string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"--no-replace-objects", "-C", repo}, args...)...)
	data, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("candidate source Git check failed: %w", err)
	}
	return data, nil
}
func gcSource(repo, commit, path string) ([]byte, error) { return gcGit(repo, "show", commit+":"+path) }
func gcValidSum(sum string) bool {
	if !strings.HasPrefix(sum, "h1:") {
		return false
	}
	b, e := base64.StdEncoding.DecodeString(sum[3:])
	return e == nil && len(b) == 32
}
func gcValidateProof(proof gcDownload, owner string) error {
	if !gcGitID.MatchString(owner) || proof.Path != gcAccordareModule || !gcPseudo.MatchString(proof.Version) || !strings.HasSuffix(proof.Version, "-"+owner[:12]) || !gcValidSum(proof.Sum) || !gcValidSum(proof.GoModSum) || proof.Error != "" || proof.Origin == nil {
		return fmt.Errorf("Accordare dependency must be a genuine exact-commit pseudo-version with canonical sums")
	}
	if proof.Origin.VCS != "git" || proof.Origin.URL != "https://github.com/QuanuX/Symphony" && proof.Origin.URL != "https://github.com/QuanuX/Symphony.git" || proof.Origin.Subdir != "modules/accordare-stav-producer" || proof.Origin.Hash != owner {
		return fmt.Errorf("Accordare module origin differs from the selected owner source")
	}
	return nil
}
func gcValidateSelection(selected gcSelection, proof gcDownload, owner string) error {
	if selected.Protocol != "symphony.release.go-owner-source.v1" || selected.Component != "accordare-stav-producer" || selected.Module != proof.Path || selected.Version != proof.Version || selected.Commit != owner || selected.Sum != proof.Sum || selected.GoModSum != proof.GoModSum || selected.Origin == nil || proof.Origin == nil || *selected.Origin != *proof.Origin || selected.Released || selected.Publication {
		return fmt.Errorf("public module resolution differs from the committed candidate owner source")
	}
	return nil
}
func gcSums(data []byte) map[string]string {
	result := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		f := strings.Fields(line)
		if len(f) == 3 {
			result[f[0]+" "+f[1]] = f[2]
		}
	}
	return result
}
func gcRequireVersion(data []byte, module string) string {
	for _, line := range strings.Split(string(data), "\n") {
		f := strings.Fields(strings.SplitN(line, "//", 2)[0])
		for i := 0; i+1 < len(f); i++ {
			if f[i] == module {
				return f[i+1]
			}
		}
	}
	return ""
}
func gcCheckDependencies(info *debug.BuildInfo, module string, sums map[string]string, proof gcDownload) ([]gcDependency, error) {
	if info.GoVersion != "go1.26.5" || info.Main.Path != module || info.Main.Replace != nil {
		return nil, fmt.Errorf("Go main module/toolchain differs")
	}
	settings := map[string]string{}
	for _, s := range info.Settings {
		settings[s.Key] = s.Value
	}
	if settings["CGO_ENABLED"] != "0" || settings["-trimpath"] != "true" {
		return nil, fmt.Errorf("candidate must use the isolated no-CGO trimpath recipe")
	}
	if len(info.Deps) > 256 {
		return nil, fmt.Errorf("linked module set exceeds bound")
	}
	linked := make([]gcDependency, 0, len(info.Deps))
	seen := map[string]bool{}
	for _, d := range info.Deps {
		if d.Replace != nil || seen[d.Path] || d.Version == "" || !gcValidSum(d.Sum) || sums[d.Path+" "+d.Version] != d.Sum {
			return nil, fmt.Errorf("unverified, replaced or duplicate linked dependency %s", d.Path)
		}
		seen[d.Path] = true
		if d.Path == gcAccordareModule && (d.Version != proof.Version || d.Sum != proof.Sum) {
			return nil, fmt.Errorf("qxctl linked the wrong Accordare owner reader")
		}
		linked = append(linked, gcDependency{d.Path, d.Version, d.Sum})
	}
	if module == "github.com/QuanuX/Symphony/tools/qxctl" && !seen[gcAccordareModule] {
		return nil, fmt.Errorf("qxctl omits the source-bound Accordare client")
	}
	sort.Slice(linked, func(i, j int) bool { return linked[i].Path < linked[j].Path })
	return linked, nil
}

func gcVerify(repo, buildRoot, qxctlRoot, proofPath, ownerCommit string) (gcResult, error) {
	var result gcResult
	proofBytes, err := gcRead(proofPath, 1<<20)
	if err != nil {
		return result, err
	}
	var proof gcDownload
	if err = gcJSON(proofBytes, &proof); err != nil {
		return result, err
	}
	if err = gcValidateProof(proof, ownerCommit); err != nil {
		return result, err
	}
	head, err := gcGit(repo, "rev-parse", "HEAD")
	if err != nil {
		return result, err
	}
	commit := strings.TrimSpace(string(head))
	if !gcGitID.MatchString(commit) {
		return result, fmt.Errorf("invalid source commit")
	}
	dirty, err := gcGit(repo, "status", "--porcelain", "--untracked-files=normal")
	if err != nil || len(dirty) != 0 {
		return result, fmt.Errorf("candidate checkout must be clean")
	}
	treeRaw, err := gcGit(repo, "rev-parse", commit+"^{tree}")
	if err != nil {
		return result, err
	}
	tree := strings.TrimSpace(string(treeRaw))
	selectionBytes, err := gcSource(repo, commit, "cmake/mvp-go-owner-closure.json")
	if err != nil {
		return result, err
	}
	var selected gcSelection
	if err = gcJSON(selectionBytes, &selected); err != nil {
		return result, err
	}
	if err = gcValidateSelection(selected, proof, ownerCommit); err != nil {
		return result, err
	}
	if _, err = gcGit(repo, "merge-base", "--is-ancestor", ownerCommit, commit); err != nil {
		return result, fmt.Errorf("owner dependency commit is not in the candidate ancestry")
	}
	timestamp, err := gcGit(repo, "show", "-s", "--format=%ct", ownerCommit)
	if err != nil {
		return result, err
	}
	seconds, err := strconv.ParseInt(strings.TrimSpace(string(timestamp)), 10, 64)
	if err != nil || !strings.Contains(proof.Version, ".0."+time.Unix(seconds, 0).UTC().Format("20060102150405")+"-") {
		return result, fmt.Errorf("pseudo-version timestamp differs from the exact owner commit")
	}
	if _, err = gcGit(repo, "diff", "--quiet", ownerCommit, commit, "--", "modules/accordare-stav-producer"); err != nil {
		return result, fmt.Errorf("Accordare native source changed after its selected client dependency")
	}
	catalogBytes, err := gcSource(repo, commit, "cmake/mvp-release-identities.json")
	if err != nil {
		return result, err
	}
	var catalog gcCatalog
	if err = gcJSON(catalogBytes, &catalog); err != nil {
		return result, err
	}
	mapBytes, err := gcSource(repo, commit, "cmake/mvp-dependencies.json")
	if err != nil {
		return result, err
	}
	paths := map[string]string{"secure-identity-access-governance": "modules/secure-identity-access-governance", "stav-append-authority": "modules/stav-append-authority", "accordare-stav-producer": "modules/accordare-stav-producer", "qxctl": "tools/qxctl", "sov-ssh-adapter": "tools/sov-ssh-adapter", "sov-stdio-adapter": "tools/sov-stdio-adapter", "sov-transport-profile-admin": "tools/sov-transport-profile-admin"}
	if len(catalog.Artifacts) != len(paths) {
		return result, fmt.Errorf("candidate must enumerate all seven Go commands")
	}
	result = gcResult{Protocol: "symphony.release.go-candidate-closure.v1", Commit: commit, Tree: tree, GoVersion: "go1.26.5", OwnerSource: ownerCommit, OwnerModuleVersion: proof.Version, OwnerModuleSum: proof.Sum, OwnerGoModSum: proof.GoModSum, DownloadEvidence: gcHash(proofBytes), OwnerSelection: gcHash(selectionBytes), DependencyMap: gcHash(mapBytes), Catalog: gcHash(catalogBytes), Status: "source_bound_candidate_closure_verified"}
	seen := map[string]bool{}
	for _, a := range catalog.Artifacts {
		if paths[a.Component] != a.Path || seen[a.Component] || filepath.Base(a.Binary) != a.Binary || a.Binary == "" {
			return result, fmt.Errorf("invalid candidate component owner")
		}
		seen[a.Component] = true
		identityPath := filepath.Join(buildRoot, "identities", a.Component+".json")
		binary := filepath.Join(buildRoot, a.Component, "bin", a.Binary)
		if a.Component == "qxctl" {
			identityPath = filepath.Join(qxctlRoot, "artifact-identity.json")
			binary = filepath.Join(qxctlRoot, "first/bin/qxctl")
		}
		identityBytes, err := gcRead(identityPath, 1<<20)
		if err != nil {
			return result, err
		}
		var id gcIdentity
		if err = gcJSON(identityBytes, &id); err != nil {
			return result, err
		}
		expectedRecipe := "symphony.go.candidate-build.v1"
		if a.Component == "qxctl" {
			expectedRecipe = "symphony.qxctl.reproducible-build.v1"
		}
		if id.Protocol != "symphony.artifact-build-identity.v1" || id.Component != a.Component || id.Version != a.Version || id.BinaryName != a.Binary || id.Commit != commit || id.Tree != tree || id.DependencyMap != result.DependencyMap || id.Catalog != result.Catalog || id.GoVersion != "go1.26.5" || id.Recipe != expectedRecipe || id.Publication || !gcSHA.MatchString(id.Executable) {
			return result, fmt.Errorf("candidate build identity differs for %s", a.Component)
		}
		if id.OS != "darwin" && id.OS != "linux" || id.Arch != "amd64" && id.Arch != "arm64" {
			return result, fmt.Errorf("unsupported candidate platform")
		}
		if result.OS == "" {
			result.OS, result.Arch = id.OS, id.Arch
		}
		if id.OS != result.OS || id.Arch != result.Arch {
			return result, fmt.Errorf("mixed candidate platforms")
		}
		mod, err := gcSource(repo, commit, a.Path+"/go.mod")
		if err != nil {
			return result, err
		}
		if gcHash(mod) != id.GoMod {
			return result, fmt.Errorf("candidate module metadata differs")
		}
		var sums []byte
		if id.GoSum != nil {
			sums, err = gcSource(repo, commit, a.Path+"/go.sum")
			if err != nil || gcHash(sums) != *id.GoSum {
				return result, fmt.Errorf("candidate checksums differ")
			}
		} else if a.Component != "sov-stdio-adapter" {
			return result, fmt.Errorf("required candidate checksums absent")
		}
		if a.Component == "qxctl" && (gcRequireVersion(mod, gcAccordareModule) != proof.Version || gcSums(sums)[proof.Path+" "+proof.Version] != proof.Sum || gcSums(sums)[proof.Path+" "+proof.Version+"/go.mod"] != proof.GoModSum) {
			return result, fmt.Errorf("qxctl source did not pin the exact canonical owner dependency")
		}
		data, err := gcRead(binary, 256<<20)
		if err != nil {
			return result, err
		}
		if gcHash(data) != id.Executable {
			return result, fmt.Errorf("candidate executable changed")
		}
		info, err := buildinfo.Read(bytes.NewReader(data))
		if err != nil {
			return result, err
		}
		module := "github.com/QuanuX/Symphony/" + a.Path
		expectedMain := module
		if a.Main != "." {
			expectedMain += "/" + strings.TrimPrefix(a.Main, "./")
		}
		if info.Path != expectedMain {
			return result, fmt.Errorf("candidate binary main package differs")
		}
		deps, err := gcCheckDependencies(info, module, gcSums(sums), proof)
		if err != nil {
			return result, fmt.Errorf("%s: %w", a.Component, err)
		}
		settings := map[string]string{}
		for _, setting := range info.Settings {
			settings[setting.Key] = setting.Value
		}
		if settings["GOOS"] != id.OS || settings["GOARCH"] != id.Arch {
			return result, fmt.Errorf("binary platform differs from identity")
		}
		result.Units = append(result.Units, gcUnit{a.Component, a.Version, id.Executable, int64(len(data)), gcHash(identityBytes), module, id.GoMod, id.GoSum, id.Recipe, deps})
	}
	sort.Slice(result.Units, func(i, j int) bool { return result.Units[i].Component < result.Units[j].Component })
	end, err := gcGit(repo, "status", "--porcelain", "--untracked-files=normal")
	if err != nil || len(end) != 0 {
		return result, fmt.Errorf("candidate source changed during verification")
	}
	endHead, err := gcGit(repo, "rev-parse", "HEAD")
	if err != nil || strings.TrimSpace(string(endHead)) != commit {
		return result, fmt.Errorf("candidate source moved during verification")
	}
	return result, nil
}
