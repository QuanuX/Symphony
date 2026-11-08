// Package sovcloudflare selects and invokes the independent SOV native owner.
// Provider semantics and installation state remain in that owner.
package sovcloudflare

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
)

const Limit = 8 * 1024 * 1024
const RequestProtocol = "symphony.sov.cloudflare-request.v1"
const ResultProtocol = "symphony.sov.cloudflare-result.v1"
const OwnerProtocol = "symphony.sov.cloudflare-owner.v1"

// Decode preserves integers and rejects duplicate keys, trailing values and
// non-UTF8. JSON is deliberately narrower than arbitrary provider JSON.
func Decode(raw []byte) (any, error) {
	if len(raw) >= Limit || !utf8.Valid(raw) || !validEscapes(raw) {
		return nil, fmt.Errorf("invalid control frame")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var read func(int) (any, error)
	read = func(depth int) (any, error) {
		if depth > 96 {
			return nil, fmt.Errorf("control nesting exceeded")
		}
		t, err := d.Token()
		if err != nil {
			return nil, err
		}
		if delim, ok := t.(json.Delim); ok {
			switch delim {
			case '{':
				m := map[string]any{}
				for d.More() {
					k, err := d.Token()
					if err != nil {
						return nil, err
					}
					key, ok := k.(string)
					if !ok {
						return nil, fmt.Errorf("invalid key")
					}
					if _, exists := m[key]; exists {
						return nil, fmt.Errorf("duplicate key")
					}
					v, err := read(depth + 1)
					if err != nil {
						return nil, err
					}
					m[key] = v
				}
				end, err := d.Token()
				if err != nil || end != json.Delim('}') {
					return nil, fmt.Errorf("invalid object")
				}
				return m, nil
			case '[':
				a := []any{}
				for d.More() {
					v, err := read(depth + 1)
					if err != nil {
						return nil, err
					}
					a = append(a, v)
				}
				end, err := d.Token()
				if err != nil || end != json.Delim(']') {
					return nil, fmt.Errorf("invalid array")
				}
				return a, nil
			default:
				return nil, fmt.Errorf("invalid delimiter")
			}
		}
		if n, ok := t.(json.Number); ok {
			s := string(n)
			if strings.ContainsAny(s, ".eE") || s == "-0" {
				return nil, fmt.Errorf("noncanonical integer")
			}
			if strings.HasPrefix(s, "-") {
				_, err = strconv.ParseInt(s, 10, 64)
			} else {
				_, err = strconv.ParseUint(s, 10, 64)
			}
			if err != nil {
				return nil, fmt.Errorf("integer overflow")
			}
			number, parseErr := strconv.ParseInt(s, 10, 64)
			if parseErr != nil || number < -9007199254740991 || number > 9007199254740991 {
				return nil, fmt.Errorf("integer outside shared exact range")
			}
		}
		return t, nil
	}
	v, err := read(0)
	if err != nil {
		return nil, err
	}
	if _, err = d.Token(); err != io.EOF {
		return nil, fmt.Errorf("trailing control data")
	}
	return v, nil
}

func validEscapes(raw []byte) bool {
	for i := 0; i < len(raw); i++ {
		if raw[i] != '\\' {
			continue
		}
		i++
		if i >= len(raw) {
			return false
		}
		if raw[i] != 'u' {
			continue
		}
		if i+4 >= len(raw) {
			return false
		}
		n, err := strconv.ParseUint(string(raw[i+1:i+5]), 16, 16)
		if err != nil {
			return false
		}
		i += 4
		if n >= 0xdc00 && n <= 0xdfff {
			return false
		}
		if n >= 0xd800 && n <= 0xdbff {
			if i+6 >= len(raw) || raw[i+1] != '\\' || raw[i+2] != 'u' {
				return false
			}
			low, err := strconv.ParseUint(string(raw[i+3:i+7]), 16, 16)
			if err != nil || low < 0xdc00 || low > 0xdfff {
				return false
			}
			i += 6
		}
	}
	return true
}

// Canonical uses sorted UTF8 keys, literal UTF8 strings with only JSON-required
// escapes, integer decimal values, explicit nulls and no whitespace. It is not
// RFC8785; the native implementation uses the same restricted value domain.
func Canonical(v any) []byte {
	var out bytes.Buffer
	var emit func(any)
	quote := func(s string) {
		out.WriteByte('"')
		for _, r := range s {
			switch r {
			case '"', '\\':
				out.WriteByte('\\')
				out.WriteRune(r)
			case '\b':
				out.WriteString(`\b`)
			case '\f':
				out.WriteString(`\f`)
			case '\n':
				out.WriteString(`\n`)
			case '\r':
				out.WriteString(`\r`)
			case '\t':
				out.WriteString(`\t`)
			default:
				if r < 32 {
					fmt.Fprintf(&out, `\u%04x`, r)
				} else {
					out.WriteRune(r)
				}
			}
		}
		out.WriteByte('"')
	}
	emit = func(v any) {
		switch x := v.(type) {
		case nil:
			out.WriteString("null")
		case bool:
			if x {
				out.WriteString("true")
			} else {
				out.WriteString("false")
			}
		case string:
			quote(x)
		case json.Number:
			out.WriteString(string(x))
		case []any:
			out.WriteByte('[')
			for i, v := range x {
				if i > 0 {
					out.WriteByte(',')
				}
				emit(v)
			}
			out.WriteByte(']')
		case map[string]any:
			keys := make([]string, 0, len(x))
			for k := range x {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			out.WriteByte('{')
			for i, k := range keys {
				if i > 0 {
					out.WriteByte(',')
				}
				quote(k)
				out.WriteByte(':')
				emit(x[k])
			}
			out.WriteByte('}')
		default:
			panic("non-control value")
		}
	}
	emit(v)
	return out.Bytes()
}
func Digest(v any, domain string) string {
	sum := sha256.Sum256(append([]byte(domain+"\x00"), Canonical(v)...))
	return fmt.Sprintf("sha256:%x", sum)
}
func uuid() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

type bounded struct {
	bytes.Buffer
	overflow bool
}

func (b *bounded) Write(p []byte) (int, error) {
	if b.Len()+len(p) >= Limit {
		b.overflow = true
		return 0, fmt.Errorf("result limit")
	}
	return b.Buffer.Write(p)
}

func Run(ctx context.Context, prefix, version, operation string, raw []byte) ([]byte, bool, error) {
	input, err := Decode(raw)
	if err != nil {
		return nil, false, fmt.Errorf("invalid Cloudflare input")
	}
	v, e := "sov", "symphony-sov-cloudflare-adapter"
	rel := "libexec/symphony/sov-cloudflare-adapter/" + version + "/" + e
	entry, err := knowledgeengine.InspectReceiptV2EntryPoint(prefix, version, knowledgeengine.ReceiptV2EntryPointSpec{Label: "sov-cloudflare-adapter", ComponentID: "sov-cloudflare-adapter", ComponentKind: "adapter", ModuleID: "sov-cloudflare-adapter", PackageID: "sov-cloudflare-adapter", VectorID: &v, EngineID: &e, EntryPointID: e, EntryPointKind: "executable", EntryPointRelativePath: rel, RequiredProtocols: []string{RequestProtocol}})
	if err != nil {
		return nil, false, fmt.Errorf("exact Cloudflare adapter installation unavailable")
	}
	ownerBytes, err := os.ReadFile(filepath.Join(entry.Prefix, "share/symphony/schemas/sov-cloudflare-adapter", version, "owner.json"))
	if err != nil {
		return nil, false, fmt.Errorf("Cloudflare owner declaration unavailable")
	}
	owner, err := Decode(ownerBytes)
	if err != nil {
		return nil, false, fmt.Errorf("invalid owner declaration")
	}
	known := false
	for _, o := range Operations {
		if o == operation {
			known = true
		}
	}
	if !known {
		return nil, false, fmt.Errorf("unsupported Cloudflare command")
	}
	planRef := any(nil)
	if operation == "installation.apply" || operation == "installation.reconcile" {
		if m, ok := input.(map[string]any); ok {
			planRef = m["plan_digest"]
		}
	}
	request := map[string]any{"protocol": RequestProtocol, "request_id": uuid(), "correlation_id": uuid(), "operation": operation, "descriptor_ref": Digest(owner, OwnerProtocol), "installation_ref": map[string]any{"version": version, "executable_digest": entry.ExecutableDigest, "receipt_digest": entry.ReceiptDigest}, "context_ref": nil, "plan_ref": planRef, "input": input}
	requestBytes := Canonical(request)
	if len(requestBytes) >= Limit {
		return nil, false, fmt.Errorf("request exceeds control bound")
	}
	// Execute a private verified snapshot, not a mutable path after verification.
	scratch, err := os.MkdirTemp("", "symphony-cloudflare-")
	if err != nil {
		return nil, false, err
	}
	defer os.RemoveAll(scratch)
	program, err := os.ReadFile(entry.ExecutablePath)
	if err != nil {
		return nil, false, err
	}
	h := sha256.Sum256(program)
	if fmt.Sprintf("sha256:%x", h) != entry.ExecutableDigest {
		return nil, false, fmt.Errorf("adapter changed")
	}
	staged := filepath.Join(scratch, e)
	if err = os.WriteFile(staged, program, 0700); err != nil {
		return nil, false, err
	}
	cmd := exec.CommandContext(ctx, staged)
	configure(cmd)
	cmd.Dir = scratch
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C"}
	cmd.Stdin = bytes.NewReader(requestBytes)
	cmd.Stderr = io.Discard
	var output bounded
	cmd.Stdout = &output
	if err = cmd.Start(); err != nil {
		return nil, false, fmt.Errorf("Cloudflare adapter did not start")
	}
	defer cleanup(cmd)
	runErr := cmd.Wait()
	result, decodeErr := Decode(output.Bytes())
	if output.overflow || decodeErr != nil {
		return lost(request), false, nil
	}
	m, ok := result.(map[string]any)
	if !ok || len(m) != 19 || m["protocol"] != ResultProtocol || m["request_id"] != request["request_id"] || m["correlation_id"] != request["correlation_id"] || m["operation"] != operation || m["request_ref"] != Digest(request, RequestProtocol) || m["descriptor_ref"] != request["descriptor_ref"] || !bytes.Equal(Canonical(m["installation_ref"]), Canonical(request["installation_ref"])) || !bytes.Equal(Canonical(m["plan_ref"]), Canonical(planRef)) || m["context_ref"] != nil {
		return lost(request), false, nil
	}
	if !validResult(m) {
		return lost(request), false, nil
	}
	declaration, ok := owner.(map[string]any)
	if !ok || !matches(result, declaration["result_schema"]) {
		return lost(request), false, nil
	}
	if m["outcome"] == "completed" {
		matched := false
		if operations, ok := declaration["operations"].([]any); ok {
			for _, op := range operations {
				if d, ok := op.(map[string]any); ok && d["operation"] == operation {
					matched = matches(m["data"], d["data_schema"])
				}
			}
		}
		if !matched || m["error_code"] != nil {
			return lost(request), false, nil
		}
	}
	completed := m["outcome"] == "completed"
	if (runErr == nil) != completed {
		return lost(request), false, nil
	}
	return Canonical(result), completed, nil
}

func validResult(m map[string]any) bool {
	allowed := map[string][]string{"dispatch_state": {"not_dispatched", "dispatch_started", "response_received"}, "outcome": {"completed", "rejected", "aborted", "failed", "unknown"}, "effect_state": {"not_applicable", "not_applied", "applied_verified", "partial_verified", "unresolved"}}
	for k, values := range allowed {
		found := false
		for _, v := range values {
			if m[k] == v {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	for _, k := range []string{"error_code", "authority_ref", "audit_ref", "evidence", "artifacts", "next_action", "data"} {
		if _, ok := m[k]; !ok {
			return false
		}
	}
	return true
}
func lost(request map[string]any) []byte {
	mutation := request["operation"] == "installation.apply" || request["operation"] == "installation.reconcile"
	outcome, effect := "failed", "not_applied"
	if mutation {
		outcome, effect = "unknown", "unresolved"
	}
	return Canonical(map[string]any{"protocol": ResultProtocol, "request_id": request["request_id"], "correlation_id": request["correlation_id"], "operation": request["operation"], "request_ref": Digest(request, RequestProtocol), "descriptor_ref": request["descriptor_ref"], "installation_ref": request["installation_ref"], "context_ref": nil, "plan_ref": request["plan_ref"], "dispatch_state": "dispatch_started", "outcome": outcome, "effect_state": effect, "error_code": "local_failure", "authority_ref": nil, "audit_ref": nil, "evidence": []any{}, "artifacts": []any{}, "next_action": "reconcile", "data": nil})
}
