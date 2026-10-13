package sovremote

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
)

type Executable struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type Endpoint struct {
	TargetQXCTL         Executable `json:"target_qxctl"`
	ConfigurationPath   string     `json:"configuration_path"`
	ConfigurationSHA256 string     `json:"configuration_sha256"`
}
type Invocation struct {
	Protocol           string          `json:"protocol"`
	Selection          Selection       `json:"selection"`
	ExpectedPlanSHA256 string          `json:"expected_plan_sha256"`
	RequestID          string          `json:"request_id"`
	CorrelationID      string          `json:"correlation_id"`
	Operation          string          `json:"operation"`
	Request            json.RawMessage `json:"request"`
	Adapter            Executable      `json:"adapter"`
	Endpoint           Endpoint        `json:"endpoint"`
}
type Configuration struct {
	Protocol                   string     `json:"protocol"`
	Target                     Target     `json:"target"`
	Job                        Job        `json:"job"`
	Runtime                    Runtime    `json:"runtime"`
	SBVPrefix                  string     `json:"sbv_prefix"`
	AuthorityMode              string     `json:"authority_mode"`
	Authority                  Authority  `json:"authority"`
	AuthorizationWindowSeconds *string    `json:"authorization_window_seconds"`
	AllowedArtifacts           []Artifact `json:"allowed_artifacts"`
}
type Artifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type ConfigureInput struct {
	Protocol       string        `json:"protocol"`
	Path           string        `json:"path"`
	ExpectedSHA256 *string       `json:"expected_sha256"`
	Configuration  Configuration `json:"configuration"`
}
type ConfigurationResult struct {
	Protocol      string        `json:"protocol"`
	Path          string        `json:"path"`
	SHA256        string        `json:"sha256"`
	Configuration Configuration `json:"configuration"`
}
type TargetResult struct {
	Protocol                    string          `json:"protocol"`
	RequestID                   string          `json:"request_id"`
	InvocationSHA256            string          `json:"invocation_sha256"`
	ConfigurationSHA256         string          `json:"configuration_sha256"`
	Target                      Target          `json:"target"`
	AuthorityMode               string          `json:"authority_mode"`
	AuthorizationDecisionSHA256 *string         `json:"authorization_decision_sha256"`
	Operation                   string          `json:"operation"`
	DispatchStarted             bool            `json:"dispatch_started"`
	Status                      string          `json:"status"`
	ErrorCode                   *string         `json:"error_code"`
	Result                      json.RawMessage `json:"result"`
}
type InvocationResult struct {
	Protocol           string        `json:"protocol"`
	RequestID          string        `json:"request_id"`
	InvocationSHA256   string        `json:"invocation_sha256"`
	PlanSHA256         string        `json:"plan_sha256"`
	TransportOutcome   string        `json:"transport_outcome"`
	AutomaticRetry     bool          `json:"automatic_retry"`
	OriginVerification string        `json:"origin_verification"`
	TargetResult       *TargetResult `json:"target_result"`
}

func Digest(raw []byte) string { d := sha256.Sum256(raw); return fmt.Sprintf("%x", d) }
func canonical(v any) ([]byte, error) {
	b, e := json.Marshal(v)
	if e != nil {
		return nil, e
	}
	var m any
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	if e = d.Decode(&m); e != nil {
		return nil, e
	}
	return json.Marshal(m)
}
func InvocationDigest(in Invocation) (string, error) {
	b, e := canonical(in)
	if e != nil {
		return "", e
	}
	return Digest(append([]byte("symphony.sov.invocation.v1\x00"), b...)), nil
}
func DecodeInvocation(raw []byte) (Invocation, error) {
	var in Invocation
	if e := decode(raw, &in); e != nil {
		return in, e
	}
	p, e := build(in.Selection)
	if e != nil {
		return in, e
	}
	if in.Protocol != "symphony.sov.remote-invoke-input.v1" || p.PlanSHA256 != in.ExpectedPlanSHA256 || !uuid.MatchString(in.RequestID) || !uuid.MatchString(in.CorrelationID) || !sha.MatchString(in.Adapter.SHA256) || !filepath.IsAbs(in.Adapter.Path) || !filepath.IsAbs(in.Endpoint.TargetQXCTL.Path) || in.Endpoint.TargetQXCTL.SHA256 != in.Selection.Runtime.QXCTLSHA256 || !filepath.IsAbs(in.Endpoint.ConfigurationPath) || !sha.MatchString(in.Endpoint.ConfigurationSHA256) || in.Adapter.SHA256 != in.Selection.Transport.AdapterSHA256 {
		return in, fmt.Errorf("invalid SOV invocation identity")
	}
	if in.Selection.Transport.Arrangement != "controller" {
		return in, fmt.Errorf("selected arrangement is not implemented by this invocation profile")
	}
	// No resource broker is present. Native calculation resources are already in
	// the immutable job. Never silently accept a placement constraint we cannot apply.
	r := in.Selection.Resources
	if r.Threads != nil || r.MemoryBytes != nil || len(r.Devices) != 0 || r.ObservationSHA256 != nil {
		return in, fmt.Errorf("additional placement constraints require a selected resource adapter")
	}
	if e = validateDeadline(in.Selection.Resources.DeadlineUnixMS); e != nil {
		return in, e
	}
	if !operationAllowed(in.Operation) {
		return in, fmt.Errorf("unsupported SOV target operation")
	}
	if e = knowledgeengine.ValidateJSONObject(in.Request, MaxDocumentBytes); e != nil {
		return in, fmt.Errorf("invalid native request")
	}
	return in, nil
}
func operationAllowed(op string) bool {
	switch op {
	case "job_inspect", "job_execute", "job_cancel", "job_reconcile", "result_inspect", "result_query", "bundle_inspect", "bundle_query", "bundle_verify":
		return true
	}
	return false
}

// validateDeadline rejects unrepresentable or elapsed explicit deadlines.
// Omitting the value never installs a default calculation deadline.
func validateDeadline(value *string) error {
	if value == nil {
		return nil
	}
	ms, e := strconv.ParseInt(*value, 10, 64)
	if e != nil || ms <= 0 {
		return fmt.Errorf("deadline cannot be represented by this process adapter")
	}
	// Context timers are created in Run/Serve, after admission validation.
	if time.UnixMilli(ms).Before(time.Now()) {
		return fmt.Errorf("caller deadline elapsed")
	}
	return nil
}
func withDeadline(parent context.Context, value *string) (context.Context, context.CancelFunc) {
	if value == nil {
		return context.WithCancel(parent)
	}
	ms, _ := strconv.ParseInt(*value, 10, 64)
	return context.WithDeadline(parent, time.UnixMilli(ms))
}

// StageExecutable copies a selected executable into a private one-use directory
// before execution. The bytes executed are the bytes whose digest was checked.
func StageExecutable(selected Executable) (string, func(), error) {
	if !filepath.IsAbs(selected.Path) || !sha.MatchString(selected.SHA256) {
		return "", nil, fmt.Errorf("invalid executable selection")
	}
	f, e := openExecutable(selected.Path)
	if e != nil {
		return "", nil, e
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil || !st.Mode().IsRegular() {
		return "", nil, fmt.Errorf("executable is not a regular file")
	}
	dir, e := os.MkdirTemp("", "sov-executable-")
	if e != nil {
		return "", nil, e
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	dst := filepath.Join(dir, "executable")
	o, e := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0700)
	if e != nil {
		cleanup()
		return "", nil, e
	}
	h := sha256.New()
	_, e = io.Copy(io.MultiWriter(o, h), f)
	closeErr := o.Close()
	if e != nil || closeErr != nil || fmt.Sprintf("%x", h.Sum(nil)) != selected.SHA256 {
		cleanup()
		return "", nil, fmt.Errorf("executable digest mismatch")
	}
	return dst, cleanup, nil
}

type boundedBuffer struct {
	bytes.Buffer
	overflow bool
}

func (b *boundedBuffer) Write(v []byte) (int, error) {
	if int64(b.Len()+len(v)) > MaxDocumentBytes {
		b.overflow = true
		return 0, fmt.Errorf("SOV control response exceeds bound")
	}
	return b.Buffer.Write(v)
}

func Run(ctx context.Context, raw []byte) (InvocationResult, error) {
	in, e := DecodeInvocation(raw)
	if e != nil {
		return InvocationResult{}, e
	}
	digest, e := InvocationDigest(in)
	if e != nil {
		return InvocationResult{}, e
	}
	out := InvocationResult{Protocol: "symphony.sov.remote-invoke.v1", RequestID: in.RequestID, InvocationSHA256: digest, PlanSHA256: in.ExpectedPlanSHA256, TransportOutcome: "not_dispatched", OriginVerification: "selected_adapter_assertion"}
	executable, cleanup, e := StageExecutable(in.Adapter)
	if e != nil {
		return out, e
	}
	defer cleanup()
	ctx, cancel := withDeadline(ctx, in.Selection.Resources.DeadlineUnixMS)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "--target", in.Endpoint.TargetQXCTL.Path, "--target-sha256", in.Endpoint.TargetQXCTL.SHA256, "--configuration", in.Endpoint.ConfigurationPath, "--configuration-sha256", in.Endpoint.ConfigurationSHA256)
	processCancellation(cmd)
	request, e := json.Marshal(in)
	if e != nil {
		return out, e
	}
	cmd.Stdin = bytes.NewReader(request)
	var response boundedBuffer
	cmd.Stdout = &response
	cmd.Stderr = io.Discard
	if e = cmd.Start(); e != nil {
		return out, fmt.Errorf("SOV adapter could not start")
	}
	defer cleanupProcess(cmd)
	out.TransportOutcome = "response_lost"
	if e = cmd.Wait(); e != nil || response.overflow {
		return out, nil
	}
	var target TargetResult
	if decode(response.Bytes(), &target) != nil || validateTargetResult(in, digest, target) != nil {
		return out, nil
	}
	out.TransportOutcome = "response_received"
	out.TargetResult = &target
	return out, nil
}
func validateTargetResult(in Invocation, digest string, r TargetResult) error {
	if r.Protocol != "symphony.sov.target-result.v1" || r.RequestID != in.RequestID || r.InvocationSHA256 != digest || r.ConfigurationSHA256 != in.Endpoint.ConfigurationSHA256 || r.Target != in.Selection.Target || r.Operation != in.Operation || (r.AuthorityMode != "owner_filesystem" && r.AuthorityMode != "ssiag") {
		return fmt.Errorf("target result mismatch")
	}
	if (r.AuthorityMode == "ssiag") != (r.AuthorizationDecisionSHA256 != nil) || !nullableSHA(r.AuthorizationDecisionSHA256) {
		return fmt.Errorf("authority evidence mismatch")
	}
	if r.Status == "completed" {
		if !r.DispatchStarted || r.ErrorCode != nil || bytes.Equal(r.Result, []byte("null")) {
			return fmt.Errorf("missing native result")
		}
		if e := knowledgeengine.ValidateSBVOperationResult(in.Operation, in.Request, r.Result); e != nil {
			return e
		}
	} else if r.Status != "native_error" || !r.DispatchStarted || r.ErrorCode == nil || !safeNativeError(*r.ErrorCode) || !bytes.Equal(r.Result, []byte("null")) {
		return fmt.Errorf("invalid native failure")
	}
	return nil
}

func safeNativeError(code string) bool {
	switch code {
	case "native_operation_failed", "sbv.contract", "sbv.pointer", "sbv.failure", "sbv.outcome_uncertain", "sbv.cancelled", "deadline.exceeded", "request.invalid_deadline", "request.deadline_expired", "argument.count", "argument.unsupported", "internal.failure", "operation.unsupported":
		return true
	}
	return false
}

func ValidateConfiguration(c Configuration) error {
	if c.Protocol != "symphony.sov.target-configuration.v1" || !filepath.IsAbs(c.SBVPrefix) || c.Runtime.SBVVersion != knowledgeengine.SBVAdministrationInterfaceVersion || !sha.MatchString(c.Runtime.SBVInstallationSHA256) || !sha.MatchString(c.Runtime.QXCTLSHA256) || !exactQXCTLVersion(c.Runtime.QXCTLVersion) || !sha.MatchString(c.Target.IdentitySHA256) || !named(c.Target.TOPS) || !named(c.Target.Node) || !named(c.Target.Habitat) || !filepath.IsAbs(c.Job.Directory) || filepath.Clean(c.Job.Directory) != c.Job.Directory || c.Job.Directory == "/" || !jobID.MatchString(c.Job.ID) || !sha.MatchString(c.Job.PlanSHA256) || c.AllowedArtifacts == nil {
		return fmt.Errorf("invalid target configuration")
	}
	if c.AuthorityMode != "owner_filesystem" && c.AuthorityMode != "ssiag" {
		return fmt.Errorf("unsupported target authority")
	}
	if !named(c.Authority.Operation) || !named(c.Authority.Resource) || !named(c.Authority.Audience) || !named(c.Authority.Scope) || !nullableSHA(c.Authority.PolicySHA256) {
		return fmt.Errorf("invalid target authority selection")
	}
	if c.AuthorityMode == "ssiag" {
		if c.AuthorizationWindowSeconds == nil {
			return fmt.Errorf("select authorization freshness window")
		}
		n, e := strconv.ParseUint(*c.AuthorizationWindowSeconds, 10, 32)
		if e != nil || n == 0 || strconv.FormatUint(n, 10) != *c.AuthorizationWindowSeconds {
			return fmt.Errorf("invalid authority freshness window")
		}
	} else if c.AuthorizationWindowSeconds != nil || c.Authority.PolicySHA256 != nil {
		return fmt.Errorf("filesystem mode carries no SSIAG authority")
	}
	for _, a := range c.AllowedArtifacts {
		if !filepath.IsAbs(a.Path) || !sha.MatchString(a.SHA256) {
			return fmt.Errorf("invalid artifact projection")
		}
	}
	return nil
}

type Authorizer func(context.Context, Configuration, Invocation) (string, error)

// Serve is invoked explicitly under the target's OS identity. Configuration is
// private and owned by that identity. It is never supplied as controller authority.
func Serve(ctx context.Context, raw []byte, configurationPath, expectedSHA, clientVersion string, authorize Authorizer) (TargetResult, error) {
	in, e := DecodeInvocation(raw)
	if e != nil {
		return TargetResult{}, e
	}
	if configurationPath != in.Endpoint.ConfigurationPath || expectedSHA != in.Endpoint.ConfigurationSHA256 {
		return TargetResult{}, fmt.Errorf("endpoint selection mismatch")
	}
	c, _, e := ReadConfiguration(configurationPath, expectedSHA)
	if e != nil {
		return TargetResult{}, e
	}
	s := in.Selection
	if c.Target != s.Target || c.Job != s.Job || c.Runtime != s.Runtime || !reflect.DeepEqual(c.Authority, s.Authority) {
		return TargetResult{}, fmt.Errorf("target selection mismatch")
	}
	self, e := os.Executable()
	if e != nil {
		return TargetResult{}, e
	}
	b, e := os.ReadFile(self)
	if e != nil || Digest(b) != s.Runtime.QXCTLSHA256 || clientVersion != s.Runtime.QXCTLVersion {
		return TargetResult{}, fmt.Errorf("target qxctl identity mismatch")
	}
	inst, e := knowledgeengine.InspectSBV(c.SBVPrefix, c.Runtime.SBVVersion)
	if e != nil || strings.TrimPrefix(inst.ReceiptDigest, "sha256:") != c.Runtime.SBVInstallationSHA256 {
		return TargetResult{}, fmt.Errorf("target native installation mismatch")
	}
	var request map[string]any
	_ = json.Unmarshal(in.Request, &request)
	if strings.HasPrefix(in.Operation, "job_") {
		if request["directory"] != c.Job.Directory || request["job_id"] != c.Job.ID || request["expected_plan_sha256"] != c.Job.PlanSHA256 {
			return TargetResult{}, fmt.Errorf("native job binding mismatch")
		}
	} else {
		path, digest := request["path"], request["expected_sha256"]
		if strings.HasPrefix(in.Operation, "bundle_") {
			reference, _ := request["reference"].(map[string]any)
			path, digest = reference["manifest_path"], reference["manifest_sha256"]
		}
		matched := false
		for _, a := range c.AllowedArtifacts {
			if path == a.Path && digest == a.SHA256 {
				matched = true
			}
		}
		if !matched {
			return TargetResult{}, fmt.Errorf("artifact outside selected projection")
		}
	}
	ctx, cancel := withDeadline(ctx, s.Resources.DeadlineUnixMS)
	defer cancel()
	digest, _ := InvocationDigest(in)
	out := TargetResult{Protocol: "symphony.sov.target-result.v1", RequestID: in.RequestID, InvocationSHA256: digest, ConfigurationSHA256: expectedSHA, Target: c.Target, AuthorityMode: c.AuthorityMode, Operation: in.Operation, Status: "native_error", Result: json.RawMessage("null")}
	admission := func(actual knowledgeengine.Installation) error {
		if actual != inst {
			return fmt.Errorf("target native installation changed")
		}
		if _, _, e := ReadConfiguration(configurationPath, expectedSHA); e != nil {
			return e
		}
		if c.AuthorityMode == "ssiag" {
			if authorize == nil {
				return fmt.Errorf("target SSIAG authority unavailable")
			}
			d, e := authorize(ctx, c, in)
			if e != nil || !sha.MatchString(d) {
				return fmt.Errorf("target authority denied or unavailable")
			}
			out.AuthorizationDecisionSHA256 = &d
		}
		if _, _, e := ReadConfiguration(configurationPath, expectedSHA); e != nil {
			return e
		}
		if e := ctx.Err(); e != nil {
			return e
		}
		out.DispatchStarted = true
		return nil
	}
	r, e := knowledgeengine.InvokeSBVWithAdmission(ctx, c.SBVPrefix, c.Runtime.SBVVersion, c.Job.Directory, in.Operation, in.Request, admission)
	if e != nil && !out.DispatchStarted {
		return TargetResult{}, e
	}
	if e != nil {
		code := "native_operation_failed"
		var native *knowledgeengine.ProcessError
		if errors.As(e, &native) && native != nil && safeNativeError(native.Code) {
			code = native.Code
		}
		out.ErrorCode = &code
		return out, nil
	}
	out.Status = "completed"
	out.Result = r.Result
	return out, nil
}
