//go:build darwin || linux

package main

import (
	"bytes"
	"context"
	"github.com/QuanuX/Symphony/tools/sov-ssh-adapter/contract"
	"io"
	"os/exec"
	"strings"
	"testing"
)

type unreadVersionInput struct{ t *testing.T }

func (r unreadVersionInput) Read([]byte) (int, error) {
	r.t.Fatal("version inspection read transport input")
	return 0, io.EOF
}

func TestVersionWithoutTransportInput(t *testing.T) {
	var out bytes.Buffer
	if code := run(context.Background(), []string{"--version"}, unreadVersionInput{t}, &out); code != 0 || out.String() != "sov-ssh-adapter version 0.1.0-dev\n" {
		t.Fatalf("version failed: code=%d output=%q", code, out.String())
	}
	for _, args := range [][]string{{"--version", "extra"}, {"--version=true"}} {
		out.Reset()
		if code := run(context.Background(), args, unreadVersionInput{t}, &out); code != 2 || out.Len() != 0 {
			t.Fatalf("accepted non-version arguments: %v, %d, %q", args, code, out.String())
		}
	}
}

func TestSSHOptionsExactAndNoninteractive(t *testing.T) {
	client, err := exec.LookPath("ssh")
	if err != nil {
		t.Skip("OpenSSH client unavailable")
	}
	e := contract.Endpoint{Host: "example.invalid", Port: "2222", User: "example", AgentSocket: "/private/agent with space.sock"}
	a := append([]string{"-G"}, sshOptions(e, "/private/known hosts", "/private/identity file")...)
	b, err := exec.Command(client, a...).Output()
	if err != nil {
		t.Fatal("OpenSSH rejected selected profile", err)
	}
	values := map[string]string{}
	for _, line := range strings.Split(string(b), "\n") {
		k, v, ok := strings.Cut(line, " ")
		if ok {
			values[k] = v
		}
	}
	for k, want := range map[string]string{"hostname": "example.invalid", "user": "example", "port": "2222", "batchmode": "yes", "identitiesonly": "yes", "stricthostkeychecking": "true", "forwardagent": "no", "forwardx11": "no", "permitlocalcommand": "no", "controlmaster": "false", "requesttty": "false", "hostkeyalias": "sov-selected-target", "identityagent": "/private/agent with space.sock", "identityfile": "/private/identity file", "userknownhostsfile": "/private/known hosts", "globalknownhostsfile": "/dev/null", "connectionattempts": "1", "connecttimeout": "none"} {
		if values[k] != want {
			t.Errorf("%s=%q; want %q", k, values[k], want)
		}
	}
	s := "9"
	e.ConnectTimeoutSeconds = &s
	a = append([]string{"-G"}, sshOptions(e, "/private/known", "/private/key")...)
	b, err = exec.Command(client, a...).Output()
	if err != nil || !bytes.Contains(b, []byte("connecttimeout 9\n")) {
		t.Fatal("caller connection timeout lost", err)
	}
}
func TestCarrierRejectsInvalidControlBeforeExecution(t *testing.T) {
	for _, raw := range []string{"{}", `{"protocol":"x","protocol":"y"}`, strings.Repeat("x", int(contract.MaxDocumentBytes)+1)} {
		var out bytes.Buffer
		if code := run(context.Background(), nil, strings.NewReader(raw), &out); code != 2 || out.Len() != 0 {
			t.Fatal("invalid carrier control dispatched", code)
		}
	}
	var b boundedBuffer
	if _, err := b.Write(make([]byte, int(contract.MaxDocumentBytes)+1)); err == nil || b.Len() != 0 {
		t.Fatal("unbounded reply")
	}
}

func TestProtectedSystemPathRefusesCallerWritableTree(t *testing.T) {
	if err := protectedSystemPath("/usr/bin/ssh"); err != nil {
		t.Fatal(err)
	}
	if err := protectedSystemPath(t.TempDir() + "/client"); err == nil {
		t.Fatal("caller path accepted as protected system client")
	}
}

func TestSessionModeNeverFallsBackToSSH(t *testing.T) {
	for _, raw := range []string{`{}`, `{"protocol":"symphony.sov.ssh-invoke-input.v1"}`, `{"protocol":"symphony.sov.transport-session-request.v1","token":"SECRET"}`} {
		var out bytes.Buffer
		if status := run(context.Background(), []string{"--session"}, strings.NewReader(raw), &out); status != 2 || out.Len() != 0 {
			t.Fatalf("invalid session escaped closed mode: %d", status)
		}
	}
}
