// This standard-library-only runner retains Go test events and refuses missing
// selected tests or unclassified skips. It does not install or activate services.
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type module struct {
	Path     string   `json:"path"`
	Packages []string `json:"packages"`
}
type optionalTest struct {
	Module         string `json:"module"`
	Package        string `json:"package"`
	Test           string `json:"test"`
	Classification string `json:"classification"`
}
type suite struct {
	Group             string   `json:"group"`
	Module            string   `json:"module"`
	Package           string   `json:"package"`
	Tests             []string `json:"tests"`
	PrefixEnvironment []string `json:"prefix_environment"`
}
type manifest struct {
	GoModules []module       `json:"go_modules"`
	Optional  []optionalTest `json:"optional_go_tests"`
	Installed []suite        `json:"installed_go_suites"`
	Required  []suite        `json:"required_go_tests"`
}
type event struct {
	Action  string
	Package string
	Test    string
	Output  string
}
type result struct {
	Package        string `json:"package"`
	Test           string `json:"test"`
	Action         string `json:"action"`
	Classification string `json:"classification,omitempty"`
}
type summary struct {
	Module          string   `json:"module"`
	Mode            string   `json:"mode"`
	Passed          int      `json:"top_level_passed"`
	Subtests        int      `json:"subtests_passed"`
	Skipped         int      `json:"top_level_skipped"`
	SkippedSubtests int      `json:"subtests_skipped"`
	NoTestPackages  []string `json:"packages_without_tests"`
	Results         []result `json:"results"`
	Required        []string `json:"required_tests"`
	Error           string   `json:"error,omitempty"`
}

func packageName(moduleID, relative string) string {
	if relative == "." {
		return moduleID
	}
	return moduleID + "/" + strings.TrimPrefix(relative, "./")
}

func evaluate(modulePath, moduleID, mode string, events []event, optional []optionalTest, required []suite) (summary, error) {
	s := summary{Module: modulePath, Mode: mode, NoTestPackages: []string{}, Results: []result{}, Required: []string{}}
	allowed := map[string]string{}
	for _, item := range optional {
		if item.Module == modulePath {
			allowed[packageName(moduleID, item.Package)+"/"+item.Test] = item.Classification
		}
	}
	passed := map[string]bool{}
	var problems []string
	for _, e := range events {
		if e.Action != "pass" && e.Action != "skip" && e.Action != "fail" {
			continue
		}
		if e.Test == "" {
			if e.Action == "skip" {
				s.NoTestPackages = append(s.NoTestPackages, e.Package)
			}
			if e.Action == "fail" {
				problems = append(problems, "package failed: "+e.Package)
			}
			continue
		}
		item := result{Package: e.Package, Test: e.Test, Action: e.Action}
		key := e.Package + "/" + e.Test
		subtest := strings.Contains(e.Test, "/")
		switch e.Action {
		case "pass":
			passed[key] = true
			if subtest {
				s.Subtests++
			} else {
				s.Passed++
			}
		case "skip":
			if subtest {
				s.SkippedSubtests++
			} else {
				s.Skipped++
			}
			item.Classification = allowed[e.Package+"/"+strings.Split(e.Test, "/")[0]]
			if mode == "installed" || item.Classification == "" {
				problems = append(problems, "unqualified skip: "+key)
			}
		case "fail":
			problems = append(problems, "test failed: "+key)
		}
		s.Results = append(s.Results, item)
	}
	for _, selected := range required {
		if selected.Module != modulePath {
			continue
		}
		for _, name := range selected.Tests {
			key := packageName(moduleID, selected.Package) + "/" + name
			s.Required = append(s.Required, key)
			if !passed[key] {
				problems = append(problems, "required test did not pass: "+key)
			}
		}
	}
	if s.Passed == 0 {
		problems = append(problems, "no top-level test executed")
	}
	if len(problems) != 0 {
		s.Error = strings.Join(problems, "; ")
		return s, errors.New(s.Error)
	}
	return s, nil
}

func moduleID(directory string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(directory, "go.mod"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			return fields[1], nil
		}
	}
	return "", errors.New("module declaration missing")
}

func manifests(directory string) (map[string]string, error) {
	result := map[string]string{}
	for _, name := range []string{"go.mod", "go.sum"} {
		raw, err := os.ReadFile(filepath.Join(directory, name))
		if errors.Is(err, os.ErrNotExist) && name == "go.sum" {
			result[name] = "absent"
			continue
		}
		if err != nil {
			return nil, err
		}
		result[name] = fmt.Sprintf("%x", sha256.Sum256(raw))
	}
	return result, nil
}

func run(repo, work, mode, prefix string, selected module, tests []suite, optional []optionalTest) error {
	directory := filepath.Join(repo, filepath.FromSlash(selected.Path))
	before, err := manifests(directory)
	if err != nil {
		return err
	}
	id, err := moduleID(directory)
	if err != nil {
		return err
	}
	label := strings.ReplaceAll(selected.Path, "/", "-")
	if mode == "installed" {
		// A module may have multiple package or test selections. Bind each set
		// of evidence files to its complete selection while retaining O_EXCL.
		selection, err := json.Marshal(tests)
		if err != nil {
			return err
		}
		label += fmt.Sprintf("-%x", sha256.Sum256(selection))
	}
	stdout, err := os.OpenFile(filepath.Join(work, label+".jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer stdout.Close()
	stderr, err := os.OpenFile(filepath.Join(work, label+".stderr.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer stderr.Close()
	arguments := []string{"test", "-mod=readonly", "-race", "-count=1", "-json"}
	if mode == "installed" {
		var names []string
		for _, item := range tests {
			for _, name := range item.Tests {
				names = append(names, regexp.QuoteMeta(name))
			}
		}
		arguments = append(arguments, "-run", "^("+strings.Join(names, "|")+")$")
	}
	arguments = append(arguments, selected.Packages...)
	command := exec.Command("go", arguments...)
	command.Dir = directory
	command.Env = append(os.Environ(), "GOWORK=off", "GOENV=off", "GOTOOLCHAIN=local")
	if mode == "installed" {
		for _, item := range tests {
			for _, variable := range item.PrefixEnvironment {
				command.Env = append(command.Env, variable+"="+prefix)
			}
		}
	}
	command.Stderr = stderr
	pipe, err := command.StdoutPipe()
	if err != nil {
		return err
	}
	if err = command.Start(); err != nil {
		return err
	}
	var events []event
	var decodeErr error
	scanner := bufio.NewScanner(pipe)
	scanner.Buffer(make([]byte, 65536), 8<<20)
	for scanner.Scan() {
		line := scanner.Bytes()
		if _, err := stdout.Write(append(append([]byte{}, line...), '\n')); err != nil {
			decodeErr = err
			break
		}
		var e event
		if err := json.Unmarshal(line, &e); err != nil {
			decodeErr = err
			break
		}
		if e.Action == "pass" || e.Action == "skip" || e.Action == "fail" {
			events = append(events, e)
		}
	}
	if scanner.Err() != nil {
		decodeErr = scanner.Err()
	}
	if decodeErr != nil {
		_ = command.Process.Kill()
	}
	waitErr := command.Wait()
	s, evaluationErr := evaluate(selected.Path, id, mode, events, optional, tests)
	after, manifestErr := manifests(directory)
	if manifestErr == nil {
		for name, digest := range before {
			if after[name] != digest {
				manifestErr = fmt.Errorf("readonly test changed %s", name)
				break
			}
		}
	}
	if manifestErr != nil {
		s.Error += "; manifest preservation: " + manifestErr.Error()
	}
	if decodeErr != nil {
		s.Error += "; invalid/truncated event stream: " + decodeErr.Error()
	}
	if waitErr != nil {
		s.Error += "; go test: " + waitErr.Error()
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(work, label+"-summary.json"), append(raw, '\n'), 0600); err != nil {
		return err
	}
	if decodeErr != nil || waitErr != nil || evaluationErr != nil || manifestErr != nil {
		return fmt.Errorf("%s: %s (see %s)", selected.Path, s.Error, work)
	}
	fmt.Printf("PASS %s: %d top-level tests, %d subtests; %d explicitly classified top-level skips, %d skipped subtests\n", selected.Path, s.Passed, s.Subtests, s.Skipped, s.SkippedSubtests)
	return nil
}

func main() {
	repo := flag.String("repo", "", "source repository")
	manifestPath := flag.String("manifest", "", "qualification manifest")
	work := flag.String("work", "", "fresh absolute evidence directory")
	mode := flag.String("mode", "full", "full or installed")
	group := flag.String("group", "", "installed group")
	prefix := flag.String("prefix", "", "exact installed prefix")
	flag.Parse()
	fail := func(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
	if !filepath.IsAbs(*repo) || !filepath.IsAbs(*work) || (*mode != "full" && *mode != "installed") {
		fail(errors.New("absolute repository/work paths and full/installed mode required"))
	}
	if *mode == "installed" && !filepath.IsAbs(*prefix) {
		fail(errors.New("installed mode requires an absolute prefix"))
	}
	if err := os.Mkdir(*work, 0700); err != nil {
		fail(err)
	}
	raw, err := os.ReadFile(*manifestPath)
	if err != nil {
		fail(err)
	}
	var m manifest
	if err = json.Unmarshal(raw, &m); err != nil {
		fail(err)
	}
	version, err := exec.Command("go", "env", "GOVERSION").Output()
	if err != nil || strings.TrimSpace(string(version)) != "go1.26.5" {
		fail(errors.New("exact Go1.26.5 required"))
	}
	if *mode == "full" {
		var failures []string
		for _, selected := range m.GoModules {
			if err = run(*repo, *work, *mode, "", selected, m.Required, m.Optional); err != nil {
				failures = append(failures, err.Error())
			}
		}
		if len(failures) != 0 {
			fail(errors.New(strings.Join(failures, "\n")))
		}
	} else {
		found := false
		for _, selected := range m.Installed {
			if selected.Group != *group {
				continue
			}
			found = true
			if err = run(*repo, *work, *mode, *prefix, module{Path: selected.Module, Packages: []string{selected.Package}}, []suite{selected}, m.Optional); err != nil {
				fail(err)
			}
		}
		if !found {
			fail(errors.New("no declared installed suite for group"))
		}
	}
}
