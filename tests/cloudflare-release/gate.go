// I6 local evidence reducer. This does not authorize or qualify live operations.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
)

type suite struct {
	ID       string   `json:"id"`
	Format   string   `json:"format"`
	Required []string `json:"required_tests"`
	Cases    []string `json:"acceptance_coverage"`
}
type manifest struct {
	Version int      `json:"format_version"`
	Suites  []suite  `json:"suites"`
	Open    []string `json:"open_requirements"`
}
type artifact struct {
	Suite  string `json:"suite"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type selection struct {
	Revision  string     `json:"source_revision"`
	Platform  string     `json:"platform"`
	Artifacts []artifact `json:"artifacts"`
}
type result struct {
	ID      string   `json:"suite"`
	Status  string   `json:"status"`
	Passed  int      `json:"passed_test_events"`
	Failed  int      `json:"failed_test_events"`
	Skipped []string `json:"skipped_tests"`
	Missing []string `json:"missing_tests"`
	Digest  string   `json:"evidence_sha256,omitempty"`
}
type report struct {
	Protocol  string   `json:"protocol"`
	Revision  string   `json:"source_revision"`
	Platform  string   `json:"platform"`
	Local     string   `json:"local_suites"`
	Milestone string   `json:"first_milestone"`
	Live      string   `json:"live_qualification"`
	Evidence  string   `json:"evidence_authority"`
	Suites    []result `json:"suites"`
	Open      []string `json:"open_requirements"`
}

func strict(raw []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return errors.New("trailing JSON")
	}
	return nil
}
func bounded(path string) ([]byte, error) {
	f, e := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	s, e := f.Stat()
	if e != nil || !s.Mode().IsRegular() || s.Size() > 64<<20 {
		return nil, errors.New("regular bounded evidence required")
	}
	b, e := io.ReadAll(io.LimitReader(f, (64<<20)+1))
	if len(b) > 64<<20 {
		return nil, errors.New("oversized evidence")
	}
	return b, e
}
func evidence(root, relative string) ([]byte, error) {
	if filepath.IsAbs(relative) || filepath.Clean(relative) != relative || relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, errors.New("relative evidence path required")
	}
	base, e := filepath.EvalSymlinks(root)
	if e != nil {
		return nil, e
	}
	p := filepath.Join(base, relative)
	real, e := filepath.EvalSymlinks(p)
	if e != nil {
		return nil, e
	}
	if real != p {
		return nil, errors.New("symlink evidence refused")
	}
	return bounded(real)
}
func parseSuite(s suite, raw []byte) (result, error) {
	r := result{ID: s.ID, Status: "passed", Skipped: []string{}, Missing: []string{}}
	names := map[string]bool{}
	if s.Format == "go-json" {
		d := json.NewDecoder(bytes.NewReader(raw))
		packages := map[string]string{}
		tests := map[string]string{}
		for {
			var event struct{ Action, Package, Test string }
			err := d.Decode(&event)
			if err == io.EOF {
				break
			}
			if err != nil {
				return r, err
			}
			if event.Package == "" {
				return r, errors.New("Go event without package")
			}
			key := event.Package + "/" + event.Test
			if event.Test == "" {
				switch event.Action {
				case "start":
					if _, ok := packages[event.Package]; ok {
						return r, errors.New("duplicate Go package")
					}
					packages[event.Package] = "running"
				case "pass", "fail", "skip":
					if packages[event.Package] != "running" {
						return r, errors.New("Go package terminal without start")
					}
					packages[event.Package] = event.Action
				}
			}
			if event.Test != "" {
				switch event.Action {
				case "run":
					if _, ok := tests[key]; ok {
						return r, errors.New("duplicate Go test")
					}
					tests[key] = "running"
				case "pass", "fail", "skip":
					if tests[key] != "running" {
						return r, errors.New("Go test terminal without run")
					}
					tests[key] = event.Action
					names[event.Test] = true
					if event.Action == "pass" {
						r.Passed++
					} else if event.Action == "fail" {
						r.Failed++
					} else {
						r.Skipped = append(r.Skipped, key)
					}
				}
			}
		}
		if len(packages) == 0 {
			return r, errors.New("empty Go report")
		}
		for p, status := range packages {
			switch status {
			case "running":
				return r, errors.New("incomplete Go package")
			case "fail":
				r.Failed++
			case "skip":
				r.Skipped = append(r.Skipped, p)
			}
		}
		for _, status := range tests {
			if status == "running" {
				return r, errors.New("incomplete Go test")
			}
		}
	} else if s.Format == "ctest-junit" {
		type test struct {
			Name    string    `xml:"name,attr"`
			Status  string    `xml:"status,attr"`
			Failure *struct{} `xml:"failure"`
			Error   *struct{} `xml:"error"`
			Skipped *struct{} `xml:"skipped"`
		}
		var doc struct {
			XMLName xml.Name `xml:"testsuite"`
			Tests   []test   `xml:"testcase"`
		}
		if e := xml.Unmarshal(raw, &doc); e != nil {
			return r, e
		}
		if len(doc.Tests) == 0 {
			return r, errors.New("empty CTest report")
		}
		for _, t := range doc.Tests {
			if t.Name == "" || names[t.Name] {
				return r, errors.New("missing or duplicate native test")
			}
			names[t.Name] = true
			if t.Failure != nil || t.Error != nil {
				r.Failed++
			} else if t.Skipped != nil || t.Status == "notrun" || t.Status == "disabled" {
				r.Skipped = append(r.Skipped, t.Name)
			} else if t.Status == "run" {
				r.Passed++
			} else {
				return r, errors.New("unknown native test status")
			}
		}
	} else {
		return r, errors.New("unknown suite format")
	}
	for _, name := range s.Required {
		if !names[name] {
			r.Missing = append(r.Missing, name)
		}
	}
	if r.Failed > 0 {
		r.Status = "failed"
	} else if len(r.Skipped) > 0 || len(r.Missing) > 0 || r.Passed == 0 {
		r.Status = "incomplete"
	}
	sort.Strings(r.Skipped)
	sort.Strings(r.Missing)
	return r, nil
}
func evaluate(m manifest, in selection, root string) (report, error) {
	r := report{Protocol: "symphony.cloudflare-local-qualification.v1", Revision: in.Revision, Platform: in.Platform, Local: "passed", Milestone: "open", Live: "not_evaluated", Evidence: "caller-selected local logs; not an independent live attestation", Suites: []result{}, Open: m.Open}
	if m.Version != 1 || len(m.Suites) == 0 || len(m.Open) == 0 || !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(in.Revision) || !regexp.MustCompile(`^(darwin|linux)-(amd64|arm64)$`).MatchString(in.Platform) {
		return r, errors.New("invalid qualification selection")
	}
	selected := map[string]artifact{}
	known := map[string]bool{}
	for _, s := range m.Suites {
		if s.ID == "" || known[s.ID] || len(s.Required) == 0 {
			return r, errors.New("invalid suite inventory")
		}
		known[s.ID] = true
	}
	for _, a := range in.Artifacts {
		if !known[a.Suite] || selected[a.Suite].Suite != "" {
			return r, errors.New("unknown or duplicate suite evidence")
		}
		selected[a.Suite] = a
	}
	for _, s := range m.Suites {
		a, ok := selected[s.ID]
		if !ok {
			r.Suites = append(r.Suites, result{ID: s.ID, Status: "missing", Missing: s.Required, Skipped: []string{}})
			if r.Local != "failed" {
				r.Local = "incomplete"
			}
			continue
		}
		raw, e := evidence(root, a.Path)
		if e != nil {
			return r, fmt.Errorf("%s: %w", s.ID, e)
		}
		h := sha256.Sum256(raw)
		if hex.EncodeToString(h[:]) != a.SHA256 {
			return r, fmt.Errorf("%s: evidence digest mismatch", s.ID)
		}
		v, e := parseSuite(s, raw)
		if e != nil {
			return r, fmt.Errorf("%s: %w", s.ID, e)
		}
		v.Digest = a.SHA256
		r.Suites = append(r.Suites, v)
		if v.Status == "failed" {
			r.Local = "failed"
		} else if v.Status != "passed" && r.Local != "failed" {
			r.Local = "incomplete"
		}
	}
	return r, nil
}
func main() {
	manifestPath := flag.String("manifest", "tests/cloudflare-release/manifest.json", "reviewed local suite inventory")
	selectionPath := flag.String("selection", "", "source/platform and evidence hashes")
	root := flag.String("evidence", "", "local evidence directory")
	flag.Parse()
	fail := func(e error) { fmt.Fprintln(os.Stderr, e); os.Exit(1) }
	var m manifest
	var in selection
	b, e := bounded(*manifestPath)
	if e != nil {
		fail(e)
	}
	if e = strict(b, &m); e != nil {
		fail(e)
	}
	b, e = bounded(*selectionPath)
	if e != nil {
		fail(e)
	}
	if e = strict(b, &in); e != nil {
		fail(e)
	}
	out, e := evaluate(m, in, *root)
	if e != nil {
		fail(e)
	}
	b, e = json.MarshalIndent(out, "", "  ")
	if e != nil {
		fail(e)
	}
	fmt.Println(string(b))
	// Exit zero means all selected local suites passed, never I6 release approval.
	if out.Local != "passed" {
		os.Exit(2)
	}
}
