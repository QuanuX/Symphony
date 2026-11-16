package main

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
)

func TestCandidateClosureOutputPreservesSelectedSource(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(base, "source")
	if err := os.Mkdir(repo, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "source-alias")
	if err := os.Symlink(repo, link); err != nil {
		t.Fatal(err)
	}
	for _, output := range []string{"relative.json", filepath.Join(repo, "evidence.json"), filepath.Join(link, "evidence.json")} {
		if _, err := gcOutputPath(repo, output); err == nil {
			t.Fatalf("unsafe output accepted: %s", output)
		}
	}
	output := filepath.Join(base, "evidence.json")
	if path, err := gcOutputPath(repo, output); err != nil || path != output {
		t.Fatalf("separate new output refused: %s %v", path, err)
	}
	if err := os.WriteFile(output, []byte("retained evidence"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := gcOutputPath(repo, output); err == nil {
		t.Fatal("existing evidence would be overwritten")
	}
}

func gcFixtureProof() gcDownload {
	owner := strings.Repeat("a", 40)
	sum := "h1:" + base64.StdEncoding.EncodeToString(make([]byte, 32))
	return gcDownload{Path: gcAccordareModule, Version: "v0.1.0-dev.0.20261116044713-" + owner[:12], Sum: sum, GoModSum: sum, Origin: &gcOrigin{VCS: "git", URL: "https://github.com/QuanuX/Symphony", Subdir: "modules/accordare-stav-producer", Hash: owner}}
}
func TestCandidateClosureExactPseudoOrigin(t *testing.T) {
	valid := gcFixtureProof()
	if err := gcValidateProof(valid, strings.Repeat("a", 40)); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"old_tag", "different_commit", "wrong_module", "wrong_subdir", "missing_sum", "download_error"} {
		t.Run(name, func(t *testing.T) {
			p := gcFixtureProof()
			switch name {
			case "old_tag":
				p.Version = "v0.1.0-dev"
			case "different_commit":
				p.Origin.Hash = strings.Repeat("b", 40)
			case "wrong_module":
				p.Path = "example.com/other"
			case "wrong_subdir":
				p.Origin.Subdir = "modules/stav-append-authority"
			case "missing_sum":
				p.Sum = ""
			case "download_error":
				p.Error = "not resolved"
			}
			if err := gcValidateProof(p, strings.Repeat("a", 40)); err == nil {
				t.Fatal("invalid candidate origin accepted")
			}
		})
	}
}
func TestCandidateClosureRequiresReviewedOwnerSelection(t *testing.T) {
	p := gcFixtureProof()
	owner := strings.Repeat("a", 40)
	selected := gcSelection{Protocol: "symphony.release.go-owner-source.v1", Component: "accordare-stav-producer", Module: p.Path, Version: p.Version, Commit: owner, Sum: p.Sum, GoModSum: p.GoModSum, Origin: p.Origin}
	if err := gcValidateSelection(selected, p, owner); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"different_commit", "different_version", "different_sum", "missing_origin", "publication_claim"} {
		t.Run(name, func(t *testing.T) {
			altered := selected
			switch name {
			case "different_commit":
				altered.Commit = strings.Repeat("b", 40)
			case "different_version":
				altered.Version = "v0.1.0-dev"
			case "different_sum":
				altered.Sum = "h1:other"
			case "missing_origin":
				altered.Origin = nil
			case "publication_claim":
				altered.Publication = true
			}
			if err := gcValidateSelection(altered, p, owner); err == nil {
				t.Fatal("unreviewed source selection accepted")
			}
		})
	}
}
func TestCandidateClosureRejectsOldOrReplacedReader(t *testing.T) {
	proof := gcFixtureProof()
	module := "github.com/QuanuX/Symphony/tools/qxctl"
	makeInfo := func() *debug.BuildInfo {
		return &debug.BuildInfo{GoVersion: "go1.26.5", Main: debug.Module{Path: module}, Deps: []*debug.Module{{Path: proof.Path, Version: proof.Version, Sum: proof.Sum}}, Settings: []debug.BuildSetting{{Key: "CGO_ENABLED", Value: "0"}, {Key: "-trimpath", Value: "true"}}}
	}
	sums := map[string]string{proof.Path + " " + proof.Version: proof.Sum, proof.Path + " v0.1.0-dev": proof.Sum}
	if deps, err := gcCheckDependencies(makeInfo(), module, sums, proof); err != nil || len(deps) != 1 {
		t.Fatalf("valid closure: %v", err)
	}
	for _, name := range []string{"old_reader", "replace", "missing", "wrong_sum", "wrong_toolchain", "cgo", "duplicate"} {
		t.Run(name, func(t *testing.T) {
			info := makeInfo()
			switch name {
			case "old_reader":
				info.Deps[0].Version = "v0.1.0-dev"
			case "replace":
				info.Deps[0].Replace = &debug.Module{Path: "/private/source"}
			case "missing":
				info.Deps = nil
			case "wrong_sum":
				info.Deps[0].Sum = "h1:" + base64.StdEncoding.EncodeToString([]byte(strings.Repeat("b", 32)))
			case "wrong_toolchain":
				info.GoVersion = "go1.27.1"
			case "cgo":
				info.Settings[0].Value = "1"
			case "duplicate":
				info.Deps = append(info.Deps, info.Deps[0])
			}
			if _, err := gcCheckDependencies(info, module, sums, proof); err == nil {
				t.Fatal("unsafe closure accepted")
			}
		})
	}
}
