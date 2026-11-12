package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestRuntimeClosureRejectsMissingOrMisorderedDependency(t *testing.T) {
	base := component{Path: "foundation", Installable: true, Group: "bootstrap"}
	owner := component{Path: "owner", Installable: true, Group: "test", Dependencies: []dependency{{Path: "foundation", Scope: "runtime"}, {Path: "test-fixture", Scope: "test"}}}
	p := projection{Components: []component{base, owner}, Order: []string{"foundation", "owner"}}
	if got := selectGraph(p, "test"); len(got) != 2 || got[0] != "foundation" {
		t.Fatal(got)
	}
	for _, kind := range []string{"missing", "order", "cycle"} {
		t.Run(kind, func(t *testing.T) {
			q := p
			q.Components = append([]component{}, p.Components...)
			switch kind {
			case "missing":
				q.Components = q.Components[1:]
			case "order":
				q.Order = []string{"owner", "foundation"}
			case "cycle":
				q.Components[0].Dependencies = []dependency{{Path: "owner", Scope: "runtime"}}
			}
			defer func() {
				if recover() == nil {
					t.Fatal("invalid runtime graph accepted")
				}
			}()
			selectGraph(q, "test")
		})
	}
}
func TestConfiguredFileSetRejectsOmittedAndDuplicateHeaders(t *testing.T) {
	expected := []string{"lib/sdk.a", "include/operation.hpp", "receipt.json"}
	if !sameSet([]string{"receipt.json", "lib/sdk.a", "include/operation.hpp"}, expected) {
		t.Fatal("ordered-independent exact inventory rejected")
	}
	for _, bad := range [][]string{{"lib/sdk.a", "receipt.json"}, {"lib/sdk.a", "lib/sdk.a", "receipt.json"}} {
		if sameSet(bad, expected) {
			t.Fatal("omitted or duplicate owned header admitted")
		}
	}
}

// Descendants hold inherited output pipes and would write a late marker if
// they survived cleanup. Readiness removes command startup from cancellation.
func TestBoundedOutputCleansDescendants(t *testing.T) {
	if role := os.Getenv("SDK_DISTRIBUTION_TIMEOUT_FIXTURE"); role != "" {
		signal.Ignore(syscall.SIGTERM) // Exercise force-stop, not cooperative exit.
		if role == "child" {
			if err := os.WriteFile(os.Getenv("SDK_READY"), []byte("ready"), 0600); err != nil {
				panic(err)
			}
			time.Sleep(2 * time.Second)
			if err := os.WriteFile(os.Getenv("SDK_LATE"), []byte("descendant survived"), 0600); err != nil {
				panic(err)
			}
			time.Sleep(10 * time.Second)
			os.Exit(0)
		}
		child := exec.Command(os.Args[0], "-test.run=^TestBoundedOutputCleansDescendants$")
		child.Env = append(os.Environ(), "SDK_DISTRIBUTION_TIMEOUT_FIXTURE=child")
		child.Stdout, child.Stderr = os.Stdout, os.Stderr
		if err := child.Start(); err != nil {
			os.Exit(1)
		}
		if role == "early_exit" {
			os.Exit(0)
		}
		_ = child.Wait()
		os.Exit(0)
	}
	for _, role := range []string{"parent", "early_exit"} {
		t.Run(role, func(t *testing.T) {
			dir := t.TempDir()
			ready, late := filepath.Join(dir, "ready"), filepath.Join(dir, "late")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			env := append(os.Environ(), "SDK_DISTRIBUTION_TIMEOUT_FIXTURE="+role, "SDK_READY="+ready, "SDK_LATE="+late, "GORACE=atexit_sleep_ms=0")
			done := make(chan error, 1)
			go func() {
				_, err := boundedOutput(ctx, dir, env, os.Args[0], "-test.run=^TestBoundedOutputCleansDescendants$")
				done <- err
			}()
			deadline := time.Now().Add(3 * time.Second)
			for {
				if _, err := os.Stat(ready); err == nil {
					break
				}
				if time.Now().After(deadline) {
					cancel()
					<-done
					t.Fatal("descendant readiness not observed")
				}
				time.Sleep(5 * time.Millisecond)
			}
			if role == "parent" {
				cancel()
			}
			select {
			case err := <-done:
				if role == "parent" && !errors.Is(err, context.Canceled) {
					t.Fatalf("cancellation: %v", err)
				}
				if expectedCommandResult(false, err) {
					t.Fatal("runner cancellation or pipe failure counted as a production refusal")
				}
				if role == "early_exit" && !errors.Is(err, exec.ErrWaitDelay) {
					t.Fatalf("early exit: %v", err)
				}
			case <-time.After(2 * time.Second):
				cancel()
				<-done
				t.Fatal("cleanup did not bound descendant output")
			}
			time.Sleep(2100 * time.Millisecond)
			if _, err := os.Stat(late); !os.IsNotExist(err) {
				t.Fatal("descendant continued after cleanup")
			}
		})
	}
}

func TestRunnerSeparatesRefusalFromSetupAndProtectsStatusDescriptor(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := boundedOutput(ctx, t.TempDir(), nil, "/bin/sh", "-c", "exit 7")
	if !expectedCommandResult(false, err) {
		t.Fatalf("actual exit not classified: %v", err)
	}
	_, err = boundedOutput(ctx, t.TempDir(), nil, "/does/not/exist/qualification-command")
	if err == nil || expectedCommandResult(false, err) {
		t.Fatalf("startup failure counted as a refusal: %v", err)
	}
	// A real child cannot write or forge the supervisor's completion channel.
	_, err = boundedOutput(ctx, t.TempDir(), nil, "/bin/sh", "-c", "if { printf forged >&3; } 2>/dev/null; then exit 99; fi; exit 0")
	if err != nil {
		t.Fatalf("supervisor status descriptor escaped: %v", err)
	}
}

func TestCanonicalSDKCoverage(t *testing.T) {
	repo, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err = os.Stat(filepath.Join(repo, "cmake", "mvp-dependencies.json")); err == nil {
			break
		}
		parent := filepath.Dir(repo)
		if parent == repo {
			t.Fatal("repository not found")
		}
		repo = parent
	}
	var p projection
	var m manifest
	if err = json.Unmarshal(read(filepath.Join(repo, "cmake/mvp-dependencies.json")), &p); err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(read(filepath.Join(repo, "tests/native-sdk-distribution/manifest.json")), &m); err != nil {
		t.Fatal(err)
	}
	c := campaign{repo: repo, components: map[string]component{}, evidence: object{}}
	for _, component := range p.Components {
		c.components[component.Path] = component
	}
	c.covered(nil, m)
}
