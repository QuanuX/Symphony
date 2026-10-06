package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/spf13/cobra"
)

const featureSBVAdministration = "ssfv:symphony:qxctl-sbv-administration"

func newSBVCommand() *cobra.Command {
	root := structural("sbv", errUsageOnly)
	results := structural("result", errUsageOnly)
	datasets := structural("dataset", errUsageOnly)
	for _, op := range knowledgeengine.SBVOperations {
		leaf := strings.TrimPrefix(strings.TrimPrefix(op, "result_"), "dataset_")
		c := newSBVLeaf(op, strings.ReplaceAll(leaf, "_", "-"))
		if strings.HasPrefix(op, "result_") {
			results.AddCommand(c)
		} else if strings.HasPrefix(op, "dataset_") {
			datasets.AddCommand(c)
		} else {
			root.AddCommand(c)
		}
	}
	results.AddCommand(newSBVLeaf("result_export", "export"))
	root.AddCommand(results, datasets, newSBVLeaf("schema", "schema"), newSBVLeaf("template", "template"))
	return root
}
func sbvSafeError(err error) error {
	var p *knowledgeengine.ProcessError
	if errors.As(err, &p) && safeSCVEngineCode(p.Code) != nil {
		return &knowledgeengine.ProcessError{Code: p.Code, Message: "SBV refused the request; inspect a write destination before retrying"}
	}
	return fmt.Errorf("SBV command failed; verify the exact installation and input contract, and inspect a write destination before retrying")
}
func newSBVLeaf(op, leaf string) *cobra.Command {
	var prefix, version, input, operation, path, digest, pointer, cursor, format string
	var timeout, absoluteDeadline string
	var limit uint
	var machine bool
	resource := op == "schema" || op == "template"
	read := strings.HasPrefix(op, "result_") && op != "result_select"
	c := &cobra.Command{Use: leaf, Short: "Native SBV " + strings.ReplaceAll(op, "_", " "), Args: usageOnlyArgs, RunE: func(c *cobra.Command, _ []string) error {
		if format != "text" && format != "json" && format != "ndjson" {
			return errUsageOnly
		}
		if machine {
			if c.Flags().Changed("format") && format != "json" {
				return errUsageOnly
			}
			format = "json"
		}
		if resource {
			var raw json.RawMessage
			var err error
			if op == "schema" {
				raw, err = knowledgeengine.SBVSchema(prefix, version, operation)
			} else {
				_, raw, err = knowledgeengine.SBVResource(prefix, version, operation, true)
			}
			if err != nil {
				return sbvSafeError(err)
			}
			wrapper, err := json.Marshal(map[string]any{"protocol": "symphony.sbv." + op + ".v1", "operation": operation, op: json.RawMessage(raw)})
			if err != nil {
				return err
			}
			return renderSBV(c.OutOrStdout(), wrapper, format)
		}
		caller, stop := signal.NotifyContext(c.Context(), os.Interrupt)
		defer stop()
		ctx, cancel, err := sbvDeadlineContext(caller, timeout, absoluteDeadline, c.Flags().Changed("timeout"))
		if err != nil {
			return err
		}
		defer cancel()
		nativeOp := op
		if op == "result_export" {
			nativeOp = "result_inspect"
		}
		var raw []byte
		if read {
			if input != "" {
				return errUsageOnly
			}
			p := map[string]any{"protocol": "symphony.sbv." + strings.ReplaceAll(nativeOp, "_", "-") + "-input.v1", "path": path, "expected_sha256": digest}
			if op == "result_query" {
				p["pointer"] = pointer
				p["limit"] = strconv.FormatUint(uint64(limit), 10)
				p["cursor"] = cursor
			}
			raw, err = json.Marshal(p)
		} else if (op == "capabilities" || op == "catalogue") && input == "" {
			raw, err = json.Marshal(map[string]string{"protocol": "symphony.sbv." + op + "-input.v1"})
		} else {
			raw, err = knowledgeengine.ReadPayload(input)
		}
		if err != nil {
			return sbvSafeError(err)
		}
		cwd, err := os.Getwd()
		if err != nil {
			return sbvSafeError(err)
		}
		result, err := knowledgeengine.InvokeSBV(ctx, prefix, version, cwd, nativeOp, raw)
		if err != nil {
			return sbvSafeError(err)
		}
		out := []byte(result.Result)
		if op == "result_export" {
			out, err = knowledgeengine.ReadSBVExport(path, result.Result)
			if err != nil {
				return sbvSafeError(err)
			}
		}
		return renderSBV(c.OutOrStdout(), out, format)
	}}
	c.Flags().StringVar(&prefix, "prefix", "", "exact SBV installation prefix")
	c.Flags().StringVar(&version, "version", "", "exact SBV release")
	c.Flags().StringVar(&format, "format", "text", "text, json or lossless pointer ndjson")
	c.Flags().BoolVar(&machine, "json", false, "JSON output and structured failures")
	for _, f := range []string{"prefix", "version"} {
		_ = c.MarkFlagRequired(f)
	}
	if resource {
		c.Flags().StringVar(&operation, "operation", "", "exact operation, e.g. run or result_query")
		_ = c.MarkFlagRequired("operation")
	} else if read {
		c.Flags().StringVar(&path, "path", "", "absolute result artifact path")
		c.Flags().StringVar(&digest, "digest", "", "expected content SHA256; binds the immutable snapshot")
		_ = c.MarkFlagRequired("path")
		if op != "result_inspect" {
			_ = c.MarkFlagRequired("digest")
		}
		if op == "result_query" {
			c.Flags().StringVar(&pointer, "pointer", "", "JSON Pointer; object/array queries list immediate children")
			c.Flags().StringVar(&cursor, "cursor", "", "continuation cursor from the same snapshot and query")
			c.Flags().UintVar(&limit, "limit", 100, "maximum nodes, 1..256")
		}
	} else {
		c.Flags().StringVar(&input, "input", "", "bounded exact native input JSON")
		if op != "capabilities" && op != "catalogue" {
			_ = c.MarkFlagRequired("input")
		}
	}
	if !resource {
		c.Flags().StringVar(&timeout, "timeout", "none", "user timeout (e.g. 30m), none or 0; no default deadline")
		c.Flags().StringVar(&absoluteDeadline, "deadline-unix-ms", "", "user absolute Unix-millisecond deadline; exclusive with --timeout")
	}
	c.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	key := "sbv." + strings.ReplaceAll(op, "_", ".")
	interaction := "invoke"
	if resource || op == "capabilities" || op == "catalogue" {
		interaction = "discover"
	} else if read || op == "result_select" || op == "dataset_inspect" {
		interaction = "inspect"
	}
	spec := commandSpec(key, featureSBVAdministration, interaction)
	spec.FeatureBindings = append(spec.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:sbv-engine", Interaction: interaction})
	if op == "run" || op == "compose" || op == "evaluate" || op == "compose_joint" || op == "economics" || op == "book" || op == "liquidity" || op == "allocation_economics" || op == "analyze" || op == "compare" || op == "resample" || op == "experiment" || op == "split" || op == "dataset_load" || op == "dataset_release" || op == "dataset_execute" {
		spec.Mutability = "permission_backed_mutation"
		spec.AuthorityMode = "target_host_permission"
		spec.RecoveryCommandID = stringPointer("qxcmd:symphony:sbv.result.inspect")
	}
	if op == "dataset_load" || op == "dataset_release" {
		spec.RecoveryCommandID = stringPointer("qxcmd:symphony:sbv.dataset.inspect")
	}
	if op == "result_inspect" || op == "dataset_inspect" {
		spec.FeatureBindings = append(spec.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:sbv-engine", Interaction: "recover"}, commandregistry.FeatureBinding{FeatureID: featureSBVAdministration, Interaction: "recover"})
	}
	nativeOp := op
	if op == "result_export" {
		nativeOp = "result_inspect"
	}
	if !resource {
		slug := strings.ReplaceAll(nativeOp, "_", "-")
		spec.BackendOperationIDs = []string{"engop:symphony:sbv." + slug}
		spec.InputProtocols = []string{"symphony.sbv." + slug + "-input.v1"}
		spec.OutputProtocols = []string{"symphony.sbv." + slug + ".v1"}
		if op == "result_export" {
			spec.OutputProtocols = []string{"symphony.sbv.result.v1", "symphony.sbv.pointer-stream.v1"}
		}
		spec.ResultValidationProtocols = []string{"symphony.sbv." + slug + ".v1"}
	} else {
		spec.OutputProtocols = []string{"symphony.sbv." + op + ".v1"}
		spec.ResultValidationProtocols = spec.OutputProtocols
	}
	commandregistry.Attach(c, spec)
	return c
}

// No timer is introduced unless selected by the caller; parent cancellation survives.
func sbvDeadlineContext(parent context.Context, timeout, absolute string, timeoutSet bool) (context.Context, context.CancelFunc, error) {
	if absolute != "" {
		value, err := strconv.ParseInt(absolute, 10, 64)
		if timeoutSet || err != nil || value <= 0 || value == math.MaxInt64 || strconv.FormatInt(value, 10) != absolute {
			return nil, nil, fmt.Errorf("select one deadline: --timeout or a canonical positive --deadline-unix-ms")
		}
		ctx, cancel := context.WithDeadline(parent, time.UnixMilli(value))
		return ctx, cancel, nil
	}
	if timeout == "none" || timeout == "0" {
		ctx, cancel := context.WithCancel(parent)
		return ctx, cancel, nil
	}
	duration, err := time.ParseDuration(timeout)
	if err != nil || duration <= 0 {
		return nil, nil, fmt.Errorf("--timeout requires a positive duration, none or 0")
	}
	ctx, cancel := context.WithTimeout(parent, duration)
	return ctx, cancel, nil
}

// The frontend renders only the owner payload. A later GUI can use the same
// result file and queries without reinterpreting native financial calculations.
func renderSBV(out io.Writer, raw []byte, format string) error {
	if format == "json" {
		var b bytes.Buffer
		if err := json.Indent(&b, raw, "", "  "); err != nil {
			return err
		}
		b.WriteByte('\n')
		_, err := out.Write(b.Bytes())
		return err
	}
	var root any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&root); err != nil {
		return err
	}
	if format == "text" {
		b, err := json.MarshalIndent(root, "", "  ")
		if err != nil {
			return err
		}
		var safe strings.Builder
		safe.WriteString("SBV — complete selected data\n")
		for _, r := range string(b) {
			if r >= 127 {
				if r > 65535 {
					a, b := utf16.EncodeRune(r)
					fmt.Fprintf(&safe, "\\u%04x\\u%04x", a, b)
				} else {
					fmt.Fprintf(&safe, "\\u%04x", r)
				}
			} else {
				safe.WriteRune(r)
			}
		}
		safe.WriteByte('\n')
		_, err = io.WriteString(out, safe.String())
		return err
	}
	if format != "ndjson" {
		return errUsageOnly
	}
	encode := json.NewEncoder(out)
	encode.SetEscapeHTML(false)
	count := uint64(0)
	if err := encode.Encode(map[string]any{"event": "begin", "protocol": "symphony.sbv.pointer-stream.v1"}); err != nil {
		return err
	}
	var walk func(any, string) error
	walk = func(v any, p string) error {
		kind := "scalar"
		value := v
		switch v.(type) {
		case map[string]any:
			kind = "object"
			value = map[string]any{}
		case []any:
			kind = "array"
			value = []any{}
		}
		if err := encode.Encode(map[string]any{"event": "node", "pointer": p, "kind": kind, "value": value}); err != nil {
			return err
		}
		count++
		switch x := v.(type) {
		case map[string]any:
			keys := make([]string, 0, len(x))
			for k := range x {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				token := strings.ReplaceAll(strings.ReplaceAll(k, "~", "~0"), "/", "~1")
				if err := walk(x[k], p+"/"+token); err != nil {
					return err
				}
			}
		case []any:
			for i, child := range x {
				if err := walk(child, p+"/"+strconv.Itoa(i)); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(root, ""); err != nil {
		return err
	}
	return encode.Encode(map[string]any{"event": "end", "status": "complete", "nodes": strconv.FormatUint(count, 10)})
}
