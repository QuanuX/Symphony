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
	args[3] = strings.Repeat("0", 64)
	if code := run(args, strings.NewReader(""), &out); code != 2 {
		t.Fatal("changed executable accepted")
	}
	if code := run(append(args, "extra"), strings.NewReader(""), &out); code != 2 {
		t.Fatal("extra args accepted")
	}
}
