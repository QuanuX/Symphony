// Package sbv transports exact SBV requests through a caller-selected qxctl.
// C++ owns scientific admission and calculation. This package owns no dataset,
// installation, research policy or result publication lifecycle.
package sbv

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

const Version = "0.1.4-dev"
const NativeVersion = "0.30.0-dev"

// Error preserves a typed boundary and the original qxctl machine failure.
// A failure never proves that a requested write had no effect. No retry occurs.
type Error struct {
	Code     string
	Detail   string
	Response json.RawMessage
	Cause    error
}

func (e *Error) Error() string       { return "sbv.adapter." + e.Code + ": " + e.Detail }
func (e *Error) Unwrap() error       { return e.Cause }
func fail(code, detail string) error { return &Error{Code: code, Detail: detail} }

type Config struct {
	Executable       string // Absolute selected qxctl binary, checked before every call.
	ExecutableSHA256 string
	Prefix           string // Exact existing native installation; qxctl verifies its receipt.
	Version          string
	WorkDir          string
	RequestDir       string // Existing private directory for temporary mode-0600 requests.
}
type Client struct{ config Config }

func New(config Config) (*Client, error) {
	if config.Version != NativeVersion || !validSHA(config.ExecutableSHA256) {
		return nil, fail("compatibility", "select the supported exact native release and qxctl SHA256")
	}
	for _, path := range []string{config.Executable, config.Prefix, config.WorkDir, config.RequestDir} {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path {
			return nil, fail("path", "absolute clean paths required")
		}
	}
	info, err := os.Stat(config.RequestDir)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return nil, fail("path", "request directory must exist and be private")
	}
	c := &Client{config: config}
	if err = c.verifyExecutable(); err != nil {
		return nil, err
	}
	return c, nil
}

// Operations returns a copy of the exact native administration surface.
func Operations() []string {
	return strings.Fields("capabilities run compose result_inspect result_query evaluate catalogue compose_joint economics book liquidity queue_scenarios account_simulate allocation_economics result_select backend_plan live_plan analyze compare resample experiment split fit predict source_retain source_export generate_census provider_inspect compose_economics research_history dataset_load dataset_inspect dataset_execute dataset_release dataset_jobs dataset_cancel dataset_configure dataset_stop bundle_import bundle_inspect bundle_query bundle_verify bundle_export")
}

func (c *Client) verifyExecutable() error {
	f, err := os.Open(c.config.Executable)
	if err != nil {
		return &Error{Code: "installation", Detail: "cannot open selected qxctl", Cause: err}
	}
	defer f.Close()
	i, err := f.Stat()
	if err != nil || !i.Mode().IsRegular() {
		return fail("installation", "qxctl must be a regular file")
	}
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return err
	}
	if hex.EncodeToString(h.Sum(nil)) != c.config.ExecutableSHA256 {
		return fail("installation", "qxctl bytes differ from selected SHA256")
	}
	return nil
}

func route(op string) ([]string, error) {
	for _, v := range append(Operations(), "result_export") {
		if v != op {
			continue
		}
		for _, prefix := range []string{"result_", "dataset_", "bundle_"} {
			if strings.HasPrefix(op, prefix) {
				return []string{"sbv", strings.TrimSuffix(prefix, "_"), strings.ReplaceAll(strings.TrimPrefix(op, prefix), "_", "-")}, nil
			}
		}
		return []string{"sbv", strings.ReplaceAll(op, "_", "-")}, nil
	}
	return nil, fail("operation", "operation is not in the selected native contract")
}

func (c *Client) command(ctx context.Context, args []string) (*exec.Cmd, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := c.verifyExecutable(); err != nil {
		return nil, err
	}
	args = append(args, "--prefix", c.config.Prefix, "--version", c.config.Version)
	cmd := exec.CommandContext(ctx, c.config.Executable, args...)
	cmd.Dir = c.config.WorkDir
	// qxctl's SIGTERM handler cancels its native process group. No default
	// deadline, detached worker or kill-after policy is introduced here.
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	return cmd, nil
}

func (c *Client) input(op string, request json.RawMessage) ([]string, func(), error) {
	args, err := route(op)
	if err != nil {
		return nil, func() {}, err
	}
	m, err := object(request)
	if err != nil {
		return nil, func() {}, err
	}
	if op == "result_inspect" || op == "result_query" || op == "result_export" {
		native := op
		if op == "result_export" {
			native = "result_inspect"
		}
		fields := []string{"protocol", "path", "expected_sha256"}
		if op == "result_query" {
			fields = append(fields, "pointer", "limit", "cursor")
		}
		if !exactFields(m, fields...) || str(m, "protocol") != "symphony.sbv."+strings.ReplaceAll(native, "_", "-")+"-input.v1" {
			return nil, func() {}, fail("shape", "result access request shape mismatch")
		}
		for _, pair := range [][2]string{{"path", "--path"}, {"expected_sha256", "--digest"}, {"pointer", "--pointer"}, {"limit", "--limit"}, {"cursor", "--cursor"}} {
			if value, ok := m[pair[0]]; ok {
				var v string
				if json.Unmarshal(value, &v) != nil {
					return nil, func() {}, fail("shape", "result access fields must be strings")
				}
				args = append(args, pair[1], v)
			}
		}
		return args, func() {}, nil
	}
	f, err := os.CreateTemp(c.config.RequestDir, "sbv-request-*.json")
	if err != nil {
		return nil, func() {}, err
	}
	cleanup := func() { _ = os.Remove(f.Name()) }
	_, err = f.Write(request)
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		cleanup()
		return nil, func() {}, errors.Join(err, closeErr)
	}
	return append(args, "--input", f.Name()), cleanup, nil
}

// Invoke returns copied JSON. Status recovery_required/unavailable is retained
// verbatim; successful transport does not promote it to successful publication.
// For bundle_export this method returns the receipt; Stream reads the data.
func (c *Client) Invoke(ctx context.Context, op string, request json.RawMessage) (json.RawMessage, error) {
	args, cleanup, err := c.input(op, request)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	if op == "bundle_export" {
		args = append(args, "--receipt-only")
	}
	return c.capture(ctx, append(args, "--format", "json"))
}

func (c *Client) Resource(ctx context.Context, op, kind, variant string) (json.RawMessage, error) {
	if _, err := route(op); err != nil {
		return nil, err
	}
	if kind != "schema" && kind != "template" || kind == "schema" && variant != "" {
		return nil, fail("shape", "select schema or template and an applicable variant")
	}
	args := []string{"sbv", kind, "--operation", op, "--format", "json"}
	if variant != "" {
		args = append(args, "--variant", variant)
	}
	return c.capture(ctx, args)
}

func (c *Client) capture(ctx context.Context, args []string) (json.RawMessage, error) {
	cmd, err := c.command(ctx, args)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return nil, &Error{Code: "process", Detail: "qxctl did not complete; reconcile write destinations before retry", Response: append(json.RawMessage(nil), out.Bytes()...), Cause: errors.Join(err, ctx.Err())}
	}
	if _, err = object(out.Bytes()); err != nil {
		return nil, err
	}
	return append(json.RawMessage(nil), out.Bytes()...), nil
}

// Stream exports a complete native artifact through qxctl. Nodes delivered to
// emit are provisional until this returns a complete receipt AND a nil error.
// Callback errors cancel qxctl, wait for it and do not retry or remove artifacts.
func (c *Client) Stream(ctx context.Context, op string, request json.RawMessage, options StreamOptions, emit func(Node) error) (Completion, error) {
	if op != "result_export" && op != "bundle_export" {
		return Completion{}, fail("operation", "stream requires result_export or bundle_export")
	}
	m, err := object(request)
	if err != nil {
		return Completion{}, err
	}
	if op == "bundle_export" {
		if str(m, "format") != "ndjson" {
			return Completion{}, fail("shape", "bundle stream requires native ndjson format")
		}
		var reference Reference
		if json.Unmarshal(m["reference"], &reference) != nil || !reference.valid() {
			return Completion{}, fail("identity", "bundle reference required")
		}
		options.Reference = reference
	} else {
		options.Reference = Reference{ContentSHA256: str(m, "expected_sha256")}
	}
	if !validSHA(options.Reference.ContentSHA256) {
		return Completion{}, fail("identity", "exact content SHA256 required")
	}
	caller := ctx
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	args, cleanup, err := c.input(op, request)
	if err != nil {
		return Completion{}, err
	}
	defer cleanup()
	cmd, err := c.command(ctx, append(args, "--format", "ndjson"))
	if err != nil {
		return Completion{}, err
	}
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return Completion{}, err
	}
	if err = cmd.Start(); err != nil {
		return Completion{}, err
	}
	completion, parseErr := ReadStream(ctx, pipe, options, emit)
	if parseErr != nil {
		cancel()
		_ = pipe.Close()
	}
	waitErr := cmd.Wait()
	// CommandContext may report its internally cancelled context even after a
	// successful child exit. Cleanup of a parse/callback failure is not evidence
	// that the caller cancelled its operation.
	if parseErr != nil && caller.Err() == nil && errors.Is(waitErr, context.Canceled) {
		waitErr = nil
	}
	if parseErr != nil || waitErr != nil {
		completion.Complete = false
		return completion, &Error{Code: "stream", Detail: "export incomplete; emitted values are provisional", Cause: errors.Join(parseErr, waitErr, caller.Err())}
	}
	return completion, nil
}

func validSHA(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil && strings.ToLower(s) == s
}
func str(m map[string]json.RawMessage, k string) string {
	var s string
	_ = json.Unmarshal(m[k], &s)
	return s
}
func exactFields(m map[string]json.RawMessage, keys ...string) bool {
	if len(m) != len(keys) {
		return false
	}
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			return false
		}
	}
	return true
}

// Reject duplicate keys before a typed decoder can silently replace evidence.
func object(raw []byte) (map[string]json.RawMessage, error) {
	if err := strictJSON(raw); err != nil {
		return nil, err
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil || m == nil {
		return nil, fail("shape", "JSON object required")
	}
	return m, nil
}
func strictJSON(raw []byte) error {
	if !validUnicodeJSON(raw) {
		return fail("precision", "invalid UTF-8 or unpaired JSON surrogate")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var read func() error
	read = func() error {
		t, err := d.Token()
		if err != nil {
			return err
		}
		switch t {
		case json.Delim('{'):
			seen := map[string]bool{}
			for d.More() {
				k, e := d.Token()
				if e != nil {
					return e
				}
				s, ok := k.(string)
				if !ok || seen[s] {
					return fail("shape", "duplicate or invalid object key")
				}
				seen[s] = true
				if e = read(); e != nil {
					return e
				}
			}
			_, err = d.Token()
		case json.Delim('['):
			for d.More() {
				if e := read(); e != nil {
					return e
				}
			}
			_, err = d.Token()
		}
		return err
	}
	if err := read(); err != nil {
		return &Error{Code: "shape", Detail: "invalid JSON", Cause: err}
	}
	if _, err := d.Token(); err != io.EOF {
		return fail("shape", fmt.Sprintf("trailing JSON: %v", err))
	}
	return nil
}
