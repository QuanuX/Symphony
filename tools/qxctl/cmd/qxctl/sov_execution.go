package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/sovremote"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/ssiagclient"
	qxversion "github.com/QuanuX/Symphony/tools/qxctl/internal/version"
	"github.com/spf13/cobra"
)

func newSOVInvoke() *cobra.Command {
	var input string
	var machine bool
	c := &cobra.Command{Use: "invoke", Short: "Invoke an exact SOV process adapter once; reconcile ambiguous outcomes", Args: usageOnlyArgs, RunE: func(c *cobra.Command, _ []string) error {
		raw, e := knowledgeengine.ReadPayload(input)
		if e != nil {
			return fmt.Errorf("invalid SOV invocation input")
		}
		ctx, stop := sbvSignalContext(c.Context())
		defer stop()
		result, e := sovremote.Run(ctx, raw)
		if result.Protocol == "" {
			return fmt.Errorf("SOV invocation rejected before dispatch")
		}
		b, encodeErr := json.Marshal(result)
		if encodeErr != nil {
			return encodeErr
		}
		format := "text"
		if machine {
			format = "json"
		}
		if writeErr := renderCompleteData(c.OutOrStdout(), b, format, "SOV remote"); writeErr != nil {
			return writeErr
		}
		if e != nil || result.TransportOutcome != "response_received" || result.TargetResult.Status != "completed" {
			return &exactEvidenceExitError{code: 5}
		}
		return nil
	}}
	c.Flags().StringVar(&input, "input", "", "exact invocation JSON")
	_ = c.MarkFlagRequired("input")
	c.Flags().BoolVar(&machine, "json", false, "complete portable JSON")
	s := commandSpec("sov.remote.invoke", featureSOVExecution, "invoke")
	s.Mutability = "permission_backed_mutation"
	s.AuthorityMode = "target_host_permission"
	s.TargetScope = "local"
	s.RecoveryCommandID = stringPointer("qxcmd:symphony:sov.remote.invoke")
	s.InputProtocols = []string{"symphony.sov.remote-invoke-input.v1"}
	s.OutputProtocols = []string{"symphony.sov.remote-invoke.v1"}
	s.ResultValidationProtocols = s.OutputProtocols
	commandregistry.Attach(c, s)
	return c
}

const featureSOVExecution = "ssfv:symphony:qxctl-sov-process-execution"

func newSOVTarget() *cobra.Command {
	root := structural("target", errUsageOnly)
	for _, op := range []string{"configure", "inspect", "remove", "invoke"} {
		root.AddCommand(newSOVTargetLeaf(op))
	}
	return root
}
func newSOVTargetLeaf(op string) *cobra.Command {
	var input, path, digest string
	var machine bool
	c := &cobra.Command{Use: op, Short: "Explicit local SOV target " + op, Args: usageOnlyArgs, RunE: func(c *cobra.Command, _ []string) error {
		var result any
		var e error
		switch op {
		case "configure":
			var raw []byte
			raw, e = knowledgeengine.ReadPayload(input)
			if e == nil {
				result, e = sovremote.Configure(raw)
			}
		case "inspect":
			var config sovremote.Configuration
			var sha string
			config, sha, e = sovremote.ReadConfiguration(path, digest)
			if e == nil {
				result = sovremote.ConfigurationResult{Protocol: "symphony.sov.target-configuration-result.v1", Path: path, SHA256: sha, Configuration: config}
			}
		case "remove":
			e = sovremote.RemoveConfiguration(path, digest)
			result = map[string]any{"protocol": "symphony.sov.target-remove.v1", "path": path, "expected_sha256": digest, "removed": e == nil}
		case "invoke":
			var raw []byte
			raw, e = io.ReadAll(io.LimitReader(c.InOrStdin(), sovremote.MaxDocumentBytes+1))
			if e == nil && int64(len(raw)) > sovremote.MaxDocumentBytes {
				e = fmt.Errorf("control input too large")
			}
			if e == nil {
				ctx, stop := sbvSignalContext(c.Context())
				defer stop()
				result, e = sovremote.Serve(ctx, raw, path, digest, qxversion.Version, authorizeSOVTarget)
			}
		}
		if e != nil {
			return fmt.Errorf("SOV target %s refused the exact selection; inspect configuration and native job before retrying", op)
		}
		b, e := json.Marshal(result)
		if e != nil {
			return e
		}
		format := "text"
		if machine {
			format = "json"
		}
		return renderCompleteData(c.OutOrStdout(), b, format, "SOV target")
	}}
	c.Flags().BoolVar(&machine, "json", false, "complete portable JSON")
	if op == "configure" {
		c.Flags().StringVar(&input, "input", "", "exact configuration/CAS input JSON")
		_ = c.MarkFlagRequired("input")
	} else {
		c.Flags().StringVar(&path, "configuration", "", "private target configuration file")
		c.Flags().StringVar(&digest, "configuration-sha256", "", "expected exact configuration bytes digest")
		_ = c.MarkFlagRequired("configuration")
		if op != "inspect" {
			_ = c.MarkFlagRequired("configuration-sha256")
		}
	}
	s := commandSpec("sov.target."+op, featureSOVExecution, "inspect")
	if op != "inspect" {
		s.FeatureBindings[0].Interaction = "invoke"
		s.Mutability = "permission_backed_mutation"
		s.AuthorityMode = "target_host_permission"
		s.RecoveryCommandID = stringPointer("qxcmd:symphony:sov.target.inspect")
	}
	switch op {
	case "configure":
		s.InputProtocols = []string{"symphony.sov.target-configure-input.v1"}
		s.OutputProtocols = []string{"symphony.sov.target-configuration-result.v1"}
	case "inspect":
		s.OutputProtocols = []string{"symphony.sov.target-configuration-result.v1"}
	case "remove":
		s.OutputProtocols = []string{"symphony.sov.target-remove.v1"}
	case "invoke":
		s.InputProtocols = []string{"symphony.sov.remote-invoke-input.v1"}
		s.OutputProtocols = []string{"symphony.sov.target-result.v1"}
	}
	s.ResultValidationProtocols = s.OutputProtocols
	commandregistry.Attach(c, s)
	return c
}

func authorizeSOVTarget(ctx context.Context, c sovremote.Configuration, in sovremote.Invocation) (string, error) {
	// The authenticated local SSIAG peer sees the executing target OS identity.
	// A decision received from the controller is never accepted as authority.
	client, e := ssiagclient.NewForTOPS("user", c.Target.TOPS, 0)
	if e != nil {
		return "", e
	}
	if _, e = requireSSIAGStatus(ctx, client, c.Target.TOPS, "user"); e != nil {
		return "", e
	}
	seconds, e := strconv.ParseUint(*c.AuthorizationWindowSeconds, 10, 32)
	if e != nil {
		return "", e
	}
	now := time.Now().UTC().Truncate(time.Second)
	request := ssiagclient.AuthorizationRequest{Schema: "symphony.ssiag.authorization-request.v1", RequestID: in.RequestID, CorrelationID: in.CorrelationID, Operation: c.Authority.Operation + "." + in.Operation, Resource: c.Authority.Resource, Audience: c.Authority.Audience, Scope: c.Authority.Scope, RequestedAt: now, RequestedExpiresAt: now.Add(time.Duration(seconds) * time.Second)}
	decision, e := client.Authorize(ctx, request)
	if e != nil {
		return "", e
	}
	if e = validateSessionAuthorization(decision, request, c.Target.TOPS); e != nil {
		return "", e
	}
	if c.Authority.PolicySHA256 != nil && strings.TrimPrefix(decision.PolicyDigest, "sha256:") != *c.Authority.PolicySHA256 {
		return "", fmt.Errorf("target policy changed")
	}
	if e = knowledgeAuthorizationFreshness(decision)(); e != nil {
		return "", e
	}
	b, e := json.Marshal(decision)
	if e != nil {
		return "", e
	}
	return sovremote.Digest(b), nil
}
