package sbv

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

type Reference struct {
	ManifestPath   string `json:"manifest_path,omitempty"`
	ManifestSHA256 string `json:"manifest_sha256,omitempty"`
	ContentSHA256  string `json:"content_sha256"`
}

func (r Reference) valid() bool {
	return r.ManifestPath != "" && validSHA(r.ManifestSHA256) && validSHA(r.ContentSHA256)
}

type StreamOptions struct {
	Reference      Reference
	MaxRecordBytes uint64 // Zero means no caller-selected record limit.
}
type Node struct {
	Pointer string          `json:"pointer"`
	Kind    string          `json:"kind"`
	Value   json.RawMessage `json:"value"` // Owned copy, valid after callback/stream return.
}
type Completion struct {
	Complete      bool      `json:"complete"`
	Nodes         uint64    `json:"nodes"`
	Reference     Reference `json:"reference"`
	ContentSHA256 string    `json:"content_sha256"`
	Protocol      string    `json:"protocol"`
}
type frame struct {
	pointer, id, kind, last string
	seen, hashed, children  uint64
}

// ReadStream verifies preorder structure, exact scalars, source correspondence
// and the native logical SHA256 while retaining open ancestor addresses and the
// largest record. Full ancestor pointers consume space proportional to their
// combined lengths; deeply nested paths can cost more than a depth-only stack.
// It never converts a scientific numeric string to a floating point number.
// emit is synchronous; its blocking time belongs to the caller. Reader I/O can
// only be interrupted by its owner. Client.Stream cancels its qxctl subprocess.
func ReadStream(ctx context.Context, input io.Reader, options StreamOptions, emit func(Node) error) (result Completion, err error) {
	if !validSHA(options.Reference.ContentSHA256) {
		return result, fail("identity", "expected content SHA256 required")
	}
	if emit == nil {
		return result, fail("shape", "node callback required")
	}
	reader := bufio.NewReader(input)
	read := func() (map[string]json.RawMessage, error) {
		var raw []byte
		for {
			if e := ctx.Err(); e != nil {
				return nil, e
			}
			part, e := reader.ReadSlice('\n')
			if options.MaxRecordBytes != 0 && (uint64(len(raw)) > options.MaxRecordBytes || uint64(len(part)) > options.MaxRecordBytes-uint64(len(raw))) {
				return nil, fail("resource", "record exceeds caller-selected limit")
			}
			raw = append(raw, part...)
			if e == bufio.ErrBufferFull {
				continue
			}
			if e != nil {
				if e == io.EOF && len(raw) != 0 {
					return nil, fail("incomplete", "record has no final newline")
				}
				return nil, e
			}
			return object(raw)
		}
	}
	begin, err := read()
	if err != nil {
		return result, err
	}
	result.Protocol = str(begin, "protocol")
	result.Reference = options.Reference
	bundle := result.Protocol == "symphony.sbv.node-stream.v1"
	var expected uint64
	if bundle {
		if !exactFields(begin, "event", "protocol", "reference", "logical_protocol", "order", "source_authorship", "expected_value_nodes") || str(begin, "logical_protocol") != "symphony.sbv.result.v1" || str(begin, "order") != "canonical_preorder_values.v1" || str(begin, "source_authorship") != "not_verified" {
			return result, fail("shape", "node stream header mismatch")
		}
		var ref Reference
		refFields, refErr := object(begin["reference"])
		if refErr != nil || !exactFields(refFields, "manifest_path", "manifest_sha256", "content_sha256") || json.Unmarshal(begin["reference"], &ref) != nil || !ref.valid() || ref != options.Reference {
			return result, fail("identity", "node stream reference mismatch")
		}
		expected, err = uintString(str(begin, "expected_value_nodes"))
		if err != nil {
			return result, err
		}
	} else if result.Protocol != "symphony.sbv.pointer-stream.v1" || !exactFields(begin, "event", "protocol") || options.Reference.ManifestSHA256 != "" {
		return result, fail("shape", "pointer stream header mismatch")
	}
	if str(begin, "event") != "begin" {
		return result, fail("incomplete", "begin record required")
	}
	h := sha256.New()
	write := func(s string) { _, _ = io.WriteString(h, s) }
	stack := []frame{}
	closeFrame := func() error {
		f := stack[len(stack)-1]
		if bundle && f.seen != f.children {
			return fail("shape", "container child count mismatch")
		}
		if f.kind == "object" {
			write("}")
		} else {
			write("]")
		}
		stack = stack[:len(stack)-1]
		return nil
	}
	seenDigest := false
	for {
		m, e := read()
		if e != nil {
			return result, &Error{Code: "incomplete", Detail: "stream ended before completion", Cause: e}
		}
		if str(m, "event") == "end" {
			fields := []string{"event", "status", "nodes"}
			if bundle {
				fields = append(fields, "content_sha256", "manifest_sha256")
			}
			count, e := uintString(str(m, "nodes"))
			if !exactFields(m, fields...) || str(m, "status") != "complete" || e != nil || count != result.Nodes || count == 0 || bundle && count != expected {
				return result, fail("incomplete", "completion count/status mismatch")
			}
			if bundle && (str(m, "content_sha256") != options.Reference.ContentSHA256 || str(m, "manifest_sha256") != options.Reference.ManifestSHA256) {
				return result, fail("identity", "completion identity mismatch")
			}
			for len(stack) > 0 {
				if e = closeFrame(); e != nil {
					return result, e
				}
			}
			result.ContentSHA256 = hex.EncodeToString(h.Sum(nil))
			if !seenDigest || result.ContentSHA256 != options.Reference.ContentSHA256 {
				return result, fail("identity", "logical content SHA256 mismatch")
			}
			if _, e = read(); e != io.EOF {
				return result, fail("incomplete", "trailing data after completion")
			}
			if e = ctx.Err(); e != nil {
				return result, e
			}
			result.Complete = true
			return result, nil
		}
		fields := []string{"event", "pointer", "kind", "value"}
		if bundle {
			fields = []string{"event", "node_id", "parent_id", "edge", "kind", "value", "children"}
		}
		if str(m, "event") != "node" || !exactFields(m, fields...) {
			return result, fail("shape", "node record mismatch")
		}
		n := Node{Kind: str(m, "kind"), Value: append(json.RawMessage(nil), m["value"]...)}
		var edge, parent string
		var children uint64
		if bundle {
			id, e := uintString(str(m, "node_id"))
			if e != nil || id != result.Nodes {
				return result, fail("shape", "nonconsecutive node id")
			}
			children, e = uintString(str(m, "children"))
			if e != nil {
				return result, e
			}
			if result.Nodes == 0 {
				if string(m["parent_id"]) != "null" || string(m["edge"]) != "null" {
					return result, fail("shape", "root node address mismatch")
				}
			} else {
				pid := str(m, "parent_id")
				for len(stack) > 0 && stack[len(stack)-1].id != pid {
					if e = closeFrame(); e != nil {
						return result, e
					}
				}
				if len(stack) == 0 || json.Unmarshal(m["edge"], &edge) != nil {
					return result, fail("shape", "parent/edge mismatch")
				}
				parent = stack[len(stack)-1].pointer
				n.Pointer = parent + "/" + escape(edge)
			}
		} else {
			if json.Unmarshal(m["pointer"], &n.Pointer) != nil {
				return result, fail("shape", "pointer required")
			}
			if result.Nodes != 0 {
				at := strings.LastIndex(n.Pointer, "/")
				if at < 0 {
					return result, fail("shape", "duplicate root or invalid pointer")
				}
				parent = n.Pointer[:at]
				edge, e = unescape(n.Pointer[at+1:])
				if e != nil {
					return result, e
				}
				for len(stack) > 0 && stack[len(stack)-1].pointer != parent {
					if e = closeFrame(); e != nil {
						return result, e
					}
				}
				if len(stack) == 0 {
					return result, fail("shape", "parent is not an open container")
				}
			} else if n.Pointer != "" {
				return result, fail("shape", "first node must be root")
			}
		}
		virtual := n.Pointer == "/content_sha256"
		if result.Nodes == 0 {
			if n.Kind != "object" {
				return result, fail("shape", "result root must be an object")
			}
		} else {
			f := &stack[len(stack)-1]
			if bundle && f.seen >= f.children {
				return result, fail("shape", "too many children")
			}
			if f.kind == "array" {
				index, e := uintString(edge)
				if e != nil || index != f.seen {
					return result, fail("shape", "array index/order mismatch")
				}
			} else if !virtual {
				if f.hashed > 0 && edge <= f.last {
					return result, fail("shape", "object key/order mismatch")
				}
				f.last = edge
			}
			if f.seen == ^uint64(0) {
				return result, fail("representation", "child counter overflow")
			}
			f.seen++
			if !virtual {
				if f.hashed > 0 {
					write(",")
				}
				if f.kind == "object" {
					write(quote(edge))
					write(":")
				}
				f.hashed++
			}
		}
		if virtual {
			if seenDigest || n.Kind != "scalar" || str(m, "value") != options.Reference.ContentSHA256 || children != 0 {
				return result, fail("identity", "root digest value mismatch")
			}
			seenDigest = true
		} else {
			switch n.Kind {
			case "object", "array":
				v := bytes.TrimSpace(n.Value)
				shell := "{}"
				if n.Kind == "array" {
					shell = "[]"
				}
				if string(v) != shell {
					return result, fail("shape", "container value must be an empty shell")
				}
				write(shell[:1])
				stack = append(stack, frame{pointer: n.Pointer, id: strconv.FormatUint(result.Nodes, 10), kind: n.Kind, children: children})
			case "scalar":
				if children != 0 {
					return result, fail("shape", "scalar cannot have children")
				}
				value, e := scalar(n.Value)
				if e != nil {
					return result, e
				}
				write(value)
			default:
				return result, fail("shape", "unknown node kind")
			}
		}
		if result.Nodes == ^uint64(0) {
			return result, fail("representation", "node counter overflow")
		}
		result.Nodes++
		if e = emit(n); e != nil {
			return result, e
		}
	}
}

func uintString(s string) (uint64, error) {
	n, e := strconv.ParseUint(s, 10, 64)
	if e != nil || strconv.FormatUint(n, 10) != s {
		return 0, fail("precision", "canonical uint64 string required")
	}
	return n, nil
}
func escape(s string) string { return strings.ReplaceAll(strings.ReplaceAll(s, "~", "~0"), "/", "~1") }
func unescape(s string) (string, error) {
	for i := 0; i < len(s); i++ {
		if s[i] == '~' {
			if i+1 >= len(s) || (s[i+1] != '0' && s[i+1] != '1') {
				return "", fail("shape", "invalid JSON Pointer escape")
			}
			i++
		}
	}
	return strings.ReplaceAll(strings.ReplaceAll(s, "~1", "/"), "~0", "~"), nil
}
func scalar(raw []byte) (string, error) {
	raw = bytes.TrimSpace(raw)
	if bytes.Equal(raw, []byte("null")) || bytes.Equal(raw, []byte("true")) || bytes.Equal(raw, []byte("false")) {
		return string(raw), nil
	}
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return "", fail("precision", "portable scalar must be an exact string, boolean or null")
	}
	return quote(s), nil
}

// Match native compact UTF-8 JSON, including literal HTML and U+2028/U+2029.
func quote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\b':
			b.WriteString("\\b")
		case '\f':
			b.WriteString("\\f")
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		case '\t':
			b.WriteString("\\t")
		default:
			if r < 32 {
				const h = "0123456789abcdef"
				b.WriteString("\\u00")
				b.WriteByte(h[r>>4])
				b.WriteByte(h[r&15])
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

func validUnicodeJSON(raw []byte) bool {
	if !utf8.Valid(raw) {
		return false
	}
	in := false
	for i := 0; i < len(raw); i++ {
		if raw[i] == '"' {
			in = !in
			continue
		}
		if !in || raw[i] != '\\' {
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
		n, e := strconv.ParseUint(string(raw[i+1:i+5]), 16, 16)
		if e != nil {
			return false
		}
		i += 4
		if n >= 0xdc00 && n <= 0xdfff {
			return false
		}
		if n < 0xd800 || n > 0xdbff {
			continue
		}
		if i+6 >= len(raw) || raw[i+1] != '\\' || raw[i+2] != 'u' {
			return false
		}
		m, e := strconv.ParseUint(string(raw[i+3:i+7]), 16, 16)
		if e != nil || m < 0xdc00 || m > 0xdfff {
			return false
		}
		i += 6
	}
	return true
}
