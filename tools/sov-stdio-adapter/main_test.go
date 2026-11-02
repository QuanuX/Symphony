//go:build darwin || linux

package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "sov" {
		if len(os.Args) != 9 || os.Args[2] != "target" || os.Args[3] != "invoke" {
			os.Exit(3)
		}
		_, _ = io.Copy(os.Stdout, os.Stdin)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

type unreadVersionInput struct{ t *testing.T }

func (r unreadVersionInput) Read([]byte) (int, error) {
	r.t.Fatal("version inspection read transport input")
	return 0, io.EOF
}

func TestVersionWithoutTransportInput(t *testing.T) {
	var out bytes.Buffer
	if code := run([]string{"--version"}, unreadVersionInput{t}, &out); code != 0 || out.String() != "sov-stdio-adapter version 0.1.0-dev\n" {
		t.Fatalf("version failed: code=%d output=%q", code, out.String())
	}
	for _, args := range [][]string{{"--version", "extra"}, {"--version=true"}} {
		out.Reset()
		if code := run(args, unreadVersionInput{t}, &out); code != 2 || out.Len() != 0 {
			t.Fatalf("accepted non-version arguments: %v, %d, %q", args, code, out.String())
		}
	}
}

func TestRelayAndExactTarget(t *testing.T) {
	self, _ := os.Executable()
	b, e := os.ReadFile(self)
	if e != nil {
		t.Fatal(e)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(b))
	args := []string{"--target", self, "--target-sha256", digest, "--configuration", "/private/selected.json", "--configuration-sha256", strings.Repeat("a", 64)}
	var out bytes.Buffer
	if code := run(args, strings.NewReader(`{"opaque":"complete"}`), &out); code != 0 || out.String() != `{"opaque":"complete"}` {
		t.Fatal(code, out.String())
	}
	out.Reset()
	if code := run(append(append([]string{}, args...), "--self-sha256", digest), strings.NewReader(`{"self":"bound"}`), &out); code != 0 || out.String() != `{"self":"bound"}` {
		t.Fatal("exact bootstrap self check failed", code, out.String())
	}
	args[3] = strings.Repeat("0", 64)
	if code := run(append(append([]string{}, args...), "--self-sha256", strings.Repeat("0", 64)), strings.NewReader(""), &out); code != 2 {
		t.Fatal("bootstrap drift admitted")
	}
	if code := run(args, strings.NewReader(""), &out); code != 2 {
		t.Fatal("changed executable accepted")
	}
	if code := run(append(args, "extra"), strings.NewReader(""), &out); code != 2 {
		t.Fatal("extra args accepted")
	}
}
