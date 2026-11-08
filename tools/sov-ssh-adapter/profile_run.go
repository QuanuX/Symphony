//go:build darwin || linux

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/sov-ssh-adapter/auditclient"
	a "github.com/QuanuX/Symphony/tools/sov-ssh-adapter/connectionaudit"
	v1 "github.com/QuanuX/Symphony/tools/sov-ssh-adapter/contract"
	"github.com/QuanuX/Symphony/tools/sov-ssh-adapter/profile"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func runProfile(ctx context.Context, raw []byte, out io.Writer) int {
	in, err := profile.DecodeInput(raw)
	if err != nil || os.Geteuid() == 0 {
		return 2
	}
	remote, invocationDigest, err := profile.TargetCommand(in)
	if err != nil {
		return 2
	}
	host, _ := v1.PublicKeyDigest(in.Endpoint.Direct.HostPublicKey)
	key, _ := v1.PublicKeyDigest(in.Endpoint.Direct.ClientPublicKey)
	reply := profile.Reply{Protocol: profile.ReplyProtocol, EndpointSHA256: profile.Digest(in.Endpoint), InvocationSHA256: invocationDigest, HostKeySHA256: host, ClientKeySHA256: key, TransportOutcome: "not_dispatched", ReasonCode: "transport_profile.admission_refused", TargetResult: json.RawMessage("null")}
	emit := func() int {
		if json.NewEncoder(out).Encode(reply) != nil {
			return 1
		}
		return 0
	}
	s, err := profile.Open(in.Endpoint.Scope)
	if err != nil {
		return emit()
	}
	defer s.Close()
	var inv struct {
		RequestID string        `json:"request_id"`
		Adapter   v1.Executable `json:"adapter"`
		Selection struct {
			Resources struct {
				Deadline *string `json:"deadline_unix_ms"`
			} `json:"resources"`
		} `json:"selection"`
	}
	if json.Unmarshal(in.Invocation, &inv) != nil {
		return emit()
	}
	if inv.Selection.Resources.Deadline != nil {
		ms, e := strconv.ParseInt(*inv.Selection.Resources.Deadline, 10, 64)
		if e != nil || ms <= 0 || strconv.FormatInt(ms, 10) != *inv.Selection.Resources.Deadline {
			return emit()
		}
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, time.UnixMilli(ms))
		defer cancel()
	}
	dir, err := os.MkdirTemp("", "sov-ssh-v2-")
	if err != nil {
		return emit()
	}
	defer os.RemoveAll(dir)
	known, identity := filepath.Join(dir, "known_hosts"), filepath.Join(dir, "identity.pub")
	if os.WriteFile(known, []byte("sov-selected-target "+in.Endpoint.Direct.HostPublicKey+"\n"), 0600) != nil || os.WriteFile(identity, []byte(in.Endpoint.Direct.ClientPublicKey+"\n"), 0600) != nil {
		return emit()
	}
	var cmd *exec.Cmd
	var journal *profile.Journal
	limit, _ := strconv.Atoi(in.Endpoint.Snapshot.Profile.Configuration.Limits.CapturedBytes)
	var response = profileBuffer{limit: limit}
	defer func() {
		if cmd != nil && cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		if journal != nil {
			journal.Close()
		}
	}()
	err = s.With(in.Endpoint, func(carrier, ssh, helper string) error {
		self, e := os.Executable()
		if e != nil || self != carrier || inv.Adapter.Path != carrier || "sha256:"+inv.Adapter.SHA256 != in.Endpoint.Snapshot.Profile.Configuration.Adapter.ExecutableDigest {
			return fmt.Errorf("carrier substitution")
		}
		if e = profile.CheckSession(in.Endpoint); e != nil {
			if e.Error() == "transport_profile.interaction_required" {
				reply.ReasonCode = e.Error()
			}
			return e
		}
		proxy, e := profile.ProxyCommand(helper, in.Endpoint)
		if e != nil {
			return e
		}
		journal, e = profile.Begin(in.Endpoint, inv.RequestID, invocationDigest)
		if e != nil {
			return e
		}
		auditRequest := a.Request{Protocol: a.Protocol, RequestID: inv.RequestID, TOPSID: in.Endpoint.Scope.TOPSID, ProfileID: in.Endpoint.Snapshot.Profile.ProfileID, EndpointDigest: "sha256:" + profile.Digest(in.Endpoint), InvocationDigest: "sha256:" + invocationDigest}
		receipt, e := auditclient.Submit(ctx, auditRequest)
		if e != nil {
			return e
		}
		if e = journal.Record("audit_committed", receipt); e != nil {
			return e
		}
		argv := sshOptions(in.Endpoint.Direct, known, identity)
		for i, v := range argv {
			if v == "ProxyCommand=none" {
				argv[i] = "ProxyCommand=" + proxy
			}
		}
		cmd = exec.CommandContext(ctx, ssh, append(argv, remote)...)
		cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C", "HOME=" + in.Endpoint.SessionDirectory}
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.WaitDelay = 250 * time.Millisecond
		cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM) }
		cmd.Stdin = bytes.NewReader(in.Invocation)
		cmd.Stderr = io.Discard
		cmd.Stdout = &response
		if e = journal.Record("dispatch_intent", map[string]string{"endpoint_digest": reply.EndpointSHA256}); e != nil {
			return e
		}
		if e = ctx.Err(); e != nil {
			return e
		}
		if e = cmd.Start(); e != nil {
			_ = journal.Record("not_dispatched", nil)
			return e
		}
		reply.TransportOutcome = "response_lost"
		reply.ReasonCode = "transport_profile.response_uncertain"
		if e = journal.Record("started", map[string]int{"pid": cmd.Process.Pid}); e != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			_ = cmd.Wait()
			return e
		}
		return nil
	})
	if err != nil {
		return emit()
	}
	err = cmd.Wait()
	// Do not retain target bytes in the private dispatch record.
	if err == nil && v1.CheckJSON(response.Bytes()) == nil {
		reply.TransportOutcome = "response_received"
		reply.ReasonCode = "transport_profile.response_received"
		reply.TargetResult = append(json.RawMessage{}, response.Bytes()...)
	}
	if journal.Record(reply.TransportOutcome, map[string]string{"response_sha256": v1.Digest(response.Bytes())}) != nil {
		reply.TransportOutcome = "response_lost"
		reply.ReasonCode = "transport_profile.response_uncertain"
		reply.TargetResult = json.RawMessage("null")
	}
	return emit()
}

// protocolDiscriminator only selects the strict parser; it grants no admission.
func isProfileInput(raw []byte) bool {
	var p struct {
		Protocol string `json:"protocol"`
	}
	return json.Unmarshal(raw, &p) == nil && strings.EqualFold(p.Protocol, profile.InputProtocol)
}

type profileBuffer struct {
	bytes.Buffer
	limit int
}

func (b *profileBuffer) Write(p []byte) (int, error) {
	if len(p) > b.limit-b.Len() {
		return 0, fmt.Errorf("profile capture bound")
	}
	return b.Buffer.Write(p)
}
