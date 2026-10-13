//go:build darwin || linux

// The independently selected SOV local-process adapter relays one bounded
// control request. It neither interprets the scientific payload nor grants access.
package main

import (
	"context"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"syscall"
	"time"
)

const Version = "0.1.0-dev"

func main() { os.Exit(run(os.Args[1:], os.Stdin, os.Stdout)) }
func run(args []string, in io.Reader, out io.Writer) int {
	f := flag.NewFlagSet("sov-stdio-adapter", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	target := f.String("target", "", "")
	digest := f.String("target-sha256", "", "")
	configuration := f.String("configuration", "", "")
	configurationSHA := f.String("configuration-sha256", "", "")
	if f.Parse(args) != nil || f.NArg() != 0 || !filepath.IsAbs(*target) || !filepath.IsAbs(*configuration) || !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(*digest) || !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(*configurationSHA) {
		return 2
	}
	src, e := os.OpenFile(*target, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if e != nil {
		return 2
	}
	defer src.Close()
	st, e := src.Stat()
	if e != nil || !st.Mode().IsRegular() {
		return 2
	}
	dir, e := os.MkdirTemp("", "sov-target-")
	if e != nil {
		return 2
	}
	defer os.RemoveAll(dir)
	staged := filepath.Join(dir, "qxctl")
	dst, e := os.OpenFile(staged, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0700)
	if e != nil {
		return 2
	}
	h := sha256.New()
	_, e = io.Copy(io.MultiWriter(dst, h), src)
	closeErr := dst.Close()
	if e != nil || closeErr != nil || fmt.Sprintf("%x", h.Sum(nil)) != *digest {
		return 2
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	cmd := exec.CommandContext(ctx, staged, "sov", "target", "invoke", "--configuration", *configuration, "--configuration-sha256", *configurationSHA, "--json")
	// Cleanup bound applies only after cancellation or process exit.
	cmd.WaitDelay = 250 * time.Millisecond
	defer func() {
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
	}()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM) }
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = io.Discard
	if cmd.Run() != nil {
		return 1
	}
	return 0
}
