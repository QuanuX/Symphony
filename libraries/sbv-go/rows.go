package sbv

import (
	"context"
	"encoding/json"
	"math/big"
	"sort"
	"strconv"
	"strings"
)

// Table describes a consumer's exact shape and feature order. Native filters,
// source/time windows and column selection belong to result_select, reachable
// through Invoke; this reader performs no filtering or numeric transformation.
type Table struct {
	ArrayPointer string   `json:"array_pointer"`
	Columns      []Column `json:"columns"`
}
type Column struct {
	Name     string `json:"name"`
	Pointer  string `json:"pointer"` // Relative JSON Pointer within each row.
	Type     string `json:"type"`    // json, string, bool, int64, uint64, rational.
	Optional bool   `json:"optional"`
	Nullable bool   `json:"nullable"`
}
type Cell struct {
	Present bool            `json:"present"`
	Value   json.RawMessage `json:"value"`
}
type Row struct {
	Index uint64 `json:"index"`
	Cells []Cell `json:"cells"` // Caller-declared column order, not object order.
}

// Rows holds at most one source row plus stream state. Each delivered row and
// cell owns its bytes; callers may retain them independently of subsequent I/O.
// Rows are provisional until the returned completion and error confirm success.
func (c *Client) Rows(ctx context.Context, op string, request json.RawMessage, options StreamOptions, table Table, emit func(Row) error) (Completion, error) {
	t, err := newTable(table, emit)
	if err != nil {
		return Completion{}, err
	}
	done, err := c.Stream(ctx, op, request, options, t.push)
	if err == nil {
		err = ctx.Err()
	}
	if err == nil {
		err = t.finish()
	}
	if err != nil {
		done.Complete = false
	}
	return done, err
}

type cellNode struct {
	kind   string
	value  json.RawMessage
	object map[string]*cellNode
	array  []*cellNode
}
type tableReader struct {
	spec       Table
	emit       func(Row) error
	found      bool
	row        *cellNode
	rowPointer string
	index      uint64
	nodes      map[string]*cellNode
}

func pointer(s string) bool {
	if s == "" {
		return true
	}
	if !strings.HasPrefix(s, "/") {
		return false
	}
	for _, p := range strings.Split(s[1:], "/") {
		if _, e := unescape(p); e != nil {
			return false
		}
	}
	return true
}
func newTable(spec Table, emit func(Row) error) (*tableReader, error) {
	if !pointer(spec.ArrayPointer) || emit == nil {
		return nil, fail("shape", "array pointer and row callback required")
	}
	seen := map[string]bool{}
	for _, c := range spec.Columns {
		if c.Name == "" || seen[c.Name] || !pointer(c.Pointer) {
			return nil, fail("shape", "unique named columns and valid pointers required")
		}
		seen[c.Name] = true
		switch c.Type {
		case "json", "string", "bool", "int64", "uint64", "rational":
		default:
			return nil, fail("precision", "unsupported dtype; no implicit floating conversion")
		}
	}
	spec.Columns = append([]Column(nil), spec.Columns...)
	return &tableReader{spec: spec, emit: emit}, nil
}
func (t *tableReader) push(n Node) error {
	if t.row != nil && n.Pointer != t.rowPointer && !strings.HasPrefix(n.Pointer, t.rowPointer+"/") {
		if err := t.flush(); err != nil {
			return err
		}
	}
	if n.Pointer == t.spec.ArrayPointer {
		if n.Kind != "array" {
			return fail("shape", "selected table must be an array")
		}
		t.found = true
		return nil
	}
	prefix := t.spec.ArrayPointer + "/"
	if !strings.HasPrefix(n.Pointer, prefix) {
		return nil
	}
	tail := strings.TrimPrefix(n.Pointer, prefix)
	if !strings.Contains(tail, "/") {
		index, err := uintString(tail)
		if err != nil {
			return err
		}
		if !t.found {
			return fail("shape", "array header required")
		}
		t.rowPointer = n.Pointer
		t.index = index
		t.nodes = map[string]*cellNode{}
	}
	if t.nodes == nil {
		return fail("shape", "row root required")
	}
	cell := &cellNode{kind: n.Kind, value: append(json.RawMessage(nil), n.Value...)}
	if n.Kind == "object" {
		cell.object = map[string]*cellNode{}
	}
	if n.Pointer == t.rowPointer {
		t.row = cell
	} else {
		at := strings.LastIndex(n.Pointer, "/")
		p := t.nodes[n.Pointer[:at]]
		if p == nil {
			return fail("shape", "row parent missing")
		}
		key, err := unescape(n.Pointer[at+1:])
		if err != nil {
			return err
		}
		if p.kind == "object" {
			p.object[key] = cell
		} else if p.kind == "array" {
			p.array = append(p.array, cell)
		} else {
			return fail("shape", "row parent is scalar")
		}
	}
	t.nodes[n.Pointer] = cell
	return nil
}
func (t *tableReader) finish() error {
	if !t.found {
		return fail("shape", "selected array does not exist")
	}
	return t.flush()
}
func (t *tableReader) flush() error {
	if t.row == nil {
		return nil
	}
	out := Row{Index: t.index, Cells: make([]Cell, 0, len(t.spec.Columns))}
	for _, col := range t.spec.Columns {
		n := t.nodes[t.rowPointer+col.Pointer]
		if n == nil {
			if !col.Optional {
				return fail("shape", "required column missing: "+col.Name)
			}
			out.Cells = append(out.Cells, Cell{Value: json.RawMessage("null")})
			continue
		}
		raw := n.bytes()
		if err := columnType(raw, col); err != nil {
			return err
		}
		out.Cells = append(out.Cells, Cell{Present: true, Value: raw})
	}
	t.row = nil
	t.nodes = nil
	return t.emit(out)
}
func (n *cellNode) bytes() json.RawMessage {
	if n.kind == "scalar" {
		return append(json.RawMessage(nil), n.value...)
	}
	var b strings.Builder
	if n.kind == "array" {
		b.WriteByte('[')
		for i, c := range n.array {
			if i > 0 {
				b.WriteByte(',')
			}
			b.Write(c.bytes())
		}
		b.WriteByte(']')
	} else {
		keys := make([]string, 0, len(n.object))
		for k := range n.object {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(quote(k))
			b.WriteByte(':')
			b.Write(n.object[k].bytes())
		}
		b.WriteByte('}')
	}
	return json.RawMessage(b.String())
}
func columnType(raw json.RawMessage, c Column) error {
	bad := func() error { return fail("precision", "column "+c.Name+" does not match selected "+c.Type) }
	if string(raw) == "null" {
		if c.Nullable {
			return nil
		}
		return fail("shape", "non-null column is null: "+c.Name)
	}
	var s string
	switch c.Type {
	case "json":
		return nil
	case "bool":
		if string(raw) == "true" || string(raw) == "false" {
			return nil
		}
	case "string":
		if json.Unmarshal(raw, &s) == nil {
			return nil
		}
	case "uint64":
		if json.Unmarshal(raw, &s) == nil {
			if _, err := uintString(s); err == nil {
				return nil
			}
		}
	case "int64":
		if json.Unmarshal(raw, &s) == nil {
			v, err := strconv.ParseInt(s, 10, 64)
			if err == nil && strconv.FormatInt(v, 10) == s {
				return nil
			}
		}
	case "rational":
		m, err := object(raw)
		if err != nil || !exactFields(m, "numerator", "denominator") {
			return bad()
		}
		n, ok := new(big.Int).SetString(str(m, "numerator"), 10)
		d, good := new(big.Int).SetString(str(m, "denominator"), 10)
		if ok && good && n.String() == str(m, "numerator") && d.String() == str(m, "denominator") && d.Sign() > 0 {
			return nil
		}
	}
	return bad()
}
