//go:build darwin || linux

package main

import (
	"context"
	"encoding/json"
	"fmt"
	c "github.com/QuanuX/Symphony/tools/sov-transport-profile-admin/contract"
	"github.com/QuanuX/Symphony/tools/sov-transport-profile-admin/internal/auditclient"
	"github.com/QuanuX/Symphony/tools/sov-transport-profile-admin/internal/store"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() { os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }
func run(args []string, in io.Reader, out, diagnostic io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(diagnostic, "transport_profile.invalid_command")
		return 2
	}
	switch args[0] {
	case "--version":
		fmt.Fprintln(out, "sov-transport-profile-admin version "+c.Version)
		return 0
	case "schema":
		fmt.Fprint(out, c.SchemaJSON)
		return 0
	case "template":
		return emit(out, c.Template())
	case "query", "administer":
	default:
		fmt.Fprintln(diagnostic, "transport_profile.invalid_command")
		return 2
	}
	// Even a directly invoked root executable must be the exact protected install.
	// SUDO_* and all other environment claims are irrelevant to authority.
	privileged := args[0] == "administer"
	if privileged != (os.Geteuid() == 0) {
		fmt.Fprintln(diagnostic, "transport_profile.invalid_privilege")
		return 3
	}
	self, e := os.Executable()
	if e != nil || self != store.HelperPath() {
		fmt.Fprintln(diagnostic, "transport_profile.protected_install_required")
		return 3
	}
	if _, e = store.AdministratorIdentity(); e != nil {
		fmt.Fprintln(diagnostic, "transport_profile.protected_install_required")
		return 3
	}
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(signalCtx, 30*time.Second)
	defer cancel()
	type read struct {
		data []byte
		err  error
	}
	done := make(chan read, 1)
	go func() { b, e := io.ReadAll(io.LimitReader(in, c.MaxDocumentBytes+1)); done <- read{b, e} }()
	var input c.Request
	select {
	case r := <-done:
		if r.err != nil || c.Decode(r.data, &input) != nil || c.ValidateRequest(input) != nil {
			fmt.Fprintln(diagnostic, "transport_profile.invalid_document")
			return 2
		}
	case <-ctx.Done():
		fmt.Fprintln(diagnostic, "transport_profile.deadline")
		return 4
	}
	if privileged != (input.Operation == "apply" || input.Operation == "recover") {
		fmt.Fprintln(diagnostic, "transport_profile.invalid_command")
		return 2
	}
	s, e := store.Open(input.Scope, auditclient.Submit)
	if e != nil {
		fmt.Fprintln(diagnostic, "transport_profile.protected_store_unavailable")
		return 3
	}
	// The process owns all store descriptors until exit. A deadline may interrupt
	// an active write; its prepared journal requires explicit recovery.
	finished := make(chan c.Result, 1)
	go func() { finished <- s.Handle(ctx, input) }()
	var result c.Result
	select {
	case result = <-finished:
		s.Close()
	case <-ctx.Done():
		fmt.Fprintln(diagnostic, "transport_profile.outcome_unknown_inspect_status")
		return 4
	}
	if emit(out, result) != 0 {
		return 4
	}
	if !result.Completed {
		return 5
	}
	return 0
}
func emit(out io.Writer, value any) int {
	data, e := json.Marshal(value)
	if e != nil || len(data) > c.MaxDocumentBytes {
		return 4
	}
	if _, e = out.Write(append(data, '\n')); e != nil {
		return 4
	}
	return 0
}
