// Package stavprotocol implements protocol mechanics defined by knowledge/stav.
// It owns no runtime or governance authority.
package stavprotocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"unicode/utf16"
	"unicode/utf8"
)

// MaxSafeInteger is the largest integer interoperable without precision loss
// across I-JSON implementations.
const MaxSafeInteger uint64 = 1<<53 - 1

var (
	ErrInvalidJSON      = errors.New("stav: invalid strict JSON")
	ErrNonCanonicalJSON = errors.New("stav: input is not canonical JCS")
)

type jsonObject map[string]jsonValue
type jsonArray []jsonValue
type jsonValue any

type strictParser struct {
	b     []byte
	i     int
	depth int
}

const maxJSONDepth = 64

// Canonicalize parses a value using the strict STAV I-JSON profile and returns
// its RFC 8785 canonical form. Null, unsafe numeric forms, duplicate names,
// invalid Unicode, and trailing values are rejected.
func Canonicalize(input []byte) ([]byte, error) {
	v, err := parseStrict(input)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	writeCanonical(&out, v)
	return out.Bytes(), nil
}

// MarshalCanonical marshals v and then applies the strict STAV canonicalizer.
// Callers should normally use the typed Encode functions, which validate the
// domain model before serialization.
func MarshalCanonical(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("stav: marshal: %w", err)
	}
	return Canonicalize(b)
}

func parseStrict(input []byte) (jsonValue, error) {
	if !utf8.Valid(input) {
		return nil, fmt.Errorf("%w: invalid UTF-8", ErrInvalidJSON)
	}
	p := strictParser{b: input}
	p.space()
	v, err := p.value()
	if err != nil {
		return nil, err
	}
	p.space()
	if p.i != len(p.b) {
		return nil, fmt.Errorf("%w: trailing data", ErrInvalidJSON)
	}
	return v, nil
}

func (p *strictParser) value() (jsonValue, error) {
	if p.i >= len(p.b) {
		return nil, fmt.Errorf("%w: unexpected end", ErrInvalidJSON)
	}
	switch p.b[p.i] {
	case '{':
		return p.object()
	case '[':
		return p.array()
	case '"':
		return p.string()
	case 't':
		if p.consume("true") {
			return true, nil
		}
	case 'f':
		if p.consume("false") {
			return false, nil
		}
	case 'n':
		return nil, fmt.Errorf("%w: null is prohibited", ErrInvalidJSON)
	default:
		if p.b[p.i] == '-' || (p.b[p.i] >= '0' && p.b[p.i] <= '9') {
			return p.number()
		}
	}
	return nil, fmt.Errorf("%w: unexpected token", ErrInvalidJSON)
}

func (p *strictParser) object() (jsonValue, error) {
	if err := p.enter(); err != nil {
		return nil, err
	}
	defer p.leave()
	p.i++
	p.space()
	o := make(jsonObject)
	if p.take('}') {
		return o, nil
	}
	for {
		if p.i >= len(p.b) || p.b[p.i] != '"' {
			return nil, fmt.Errorf("%w: object name required", ErrInvalidJSON)
		}
		nameValue, err := p.string()
		if err != nil {
			return nil, err
		}
		name := nameValue.(string)
		if _, exists := o[name]; exists {
			return nil, fmt.Errorf("%w: duplicate object name", ErrInvalidJSON)
		}
		p.space()
		if !p.take(':') {
			return nil, fmt.Errorf("%w: missing object colon", ErrInvalidJSON)
		}
		p.space()
		v, err := p.value()
		if err != nil {
			return nil, err
		}
		o[name] = v
		p.space()
		if p.take('}') {
			return o, nil
		}
		if !p.take(',') {
			return nil, fmt.Errorf("%w: missing object comma", ErrInvalidJSON)
		}
		p.space()
	}
}

func (p *strictParser) array() (jsonValue, error) {
	if err := p.enter(); err != nil {
		return nil, err
	}
	defer p.leave()
	p.i++
	p.space()
	a := make(jsonArray, 0)
	if p.take(']') {
		return a, nil
	}
	for {
		v, err := p.value()
		if err != nil {
			return nil, err
		}
		a = append(a, v)
		p.space()
		if p.take(']') {
			return a, nil
		}
		if !p.take(',') {
			return nil, fmt.Errorf("%w: missing array comma", ErrInvalidJSON)
		}
		p.space()
	}
}

func (p *strictParser) string() (jsonValue, error) {
	start := p.i
	p.i++
	for p.i < len(p.b) {
		c := p.b[p.i]
		if c == '"' {
			p.i++
			var s string
			if err := json.Unmarshal(p.b[start:p.i], &s); err != nil {
				return nil, fmt.Errorf("%w: malformed string", ErrInvalidJSON)
			}
			if err := validateUnicodeString(s); err != nil {
				return nil, err
			}
			return s, nil
		}
		if c < 0x20 {
			return nil, fmt.Errorf("%w: unescaped control character", ErrInvalidJSON)
		}
		if c == '\\' {
			p.i++
			if p.i >= len(p.b) {
				return nil, fmt.Errorf("%w: truncated escape", ErrInvalidJSON)
			}
			switch p.b[p.i] {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
				p.i++
				continue
			case 'u':
				first, err := p.unicodeEscape()
				if err != nil {
					return nil, err
				}
				if first >= 0xD800 && first <= 0xDBFF {
					if p.i+1 >= len(p.b) || p.b[p.i] != '\\' || p.b[p.i+1] != 'u' {
						return nil, fmt.Errorf("%w: unpaired high surrogate", ErrInvalidJSON)
					}
					p.i++
					second, err := p.unicodeEscape()
					if err != nil || second < 0xDC00 || second > 0xDFFF {
						return nil, fmt.Errorf("%w: unpaired high surrogate", ErrInvalidJSON)
					}
					r := utf16.DecodeRune(rune(first), rune(second))
					if isNoncharacter(r) {
						return nil, fmt.Errorf("%w: Unicode noncharacter", ErrInvalidJSON)
					}
				} else if first >= 0xDC00 && first <= 0xDFFF {
					return nil, fmt.Errorf("%w: unpaired low surrogate", ErrInvalidJSON)
				} else if isNoncharacter(rune(first)) {
					return nil, fmt.Errorf("%w: Unicode noncharacter", ErrInvalidJSON)
				}
				continue
			default:
				return nil, fmt.Errorf("%w: invalid escape", ErrInvalidJSON)
			}
		}
		if c >= utf8.RuneSelf {
			r, size := utf8.DecodeRune(p.b[p.i:])
			if r == utf8.RuneError && size == 1 {
				return nil, fmt.Errorf("%w: invalid UTF-8", ErrInvalidJSON)
			}
			if isNoncharacter(r) {
				return nil, fmt.Errorf("%w: Unicode noncharacter", ErrInvalidJSON)
			}
			p.i += size
			continue
		}
		p.i++
	}
	return nil, fmt.Errorf("%w: unterminated string", ErrInvalidJSON)
}

func (p *strictParser) unicodeEscape() (uint16, error) {
	// p.i points at 'u'.
	if p.i+5 > len(p.b) {
		return 0, fmt.Errorf("%w: truncated Unicode escape", ErrInvalidJSON)
	}
	n, err := strconv.ParseUint(string(p.b[p.i+1:p.i+5]), 16, 16)
	if err != nil {
		return 0, fmt.Errorf("%w: malformed Unicode escape", ErrInvalidJSON)
	}
	p.i += 5
	return uint16(n), nil
}

func (p *strictParser) number() (jsonValue, error) {
	start := p.i
	if p.b[p.i] == '-' {
		return nil, fmt.Errorf("%w: negative numbers are prohibited", ErrInvalidJSON)
	}
	if p.b[p.i] == '0' {
		p.i++
		if p.i < len(p.b) && p.b[p.i] >= '0' && p.b[p.i] <= '9' {
			return nil, fmt.Errorf("%w: leading zero", ErrInvalidJSON)
		}
	} else {
		for p.i < len(p.b) && p.b[p.i] >= '0' && p.b[p.i] <= '9' {
			p.i++
		}
	}
	if p.i < len(p.b) && (p.b[p.i] == '.' || p.b[p.i] == 'e' || p.b[p.i] == 'E') {
		return nil, fmt.Errorf("%w: non-integer number", ErrInvalidJSON)
	}
	n, err := strconv.ParseUint(string(p.b[start:p.i]), 10, 64)
	if err != nil || n > MaxSafeInteger {
		return nil, fmt.Errorf("%w: unsafe integer", ErrInvalidJSON)
	}
	return n, nil
}

func (p *strictParser) space() {
	for p.i < len(p.b) {
		switch p.b[p.i] {
		case ' ', '\t', '\n', '\r':
			p.i++
		default:
			return
		}
	}
}

func (p *strictParser) take(want byte) bool {
	if p.i < len(p.b) && p.b[p.i] == want {
		p.i++
		return true
	}
	return false
}

func (p *strictParser) consume(want string) bool {
	if len(p.b)-p.i >= len(want) && string(p.b[p.i:p.i+len(want)]) == want {
		p.i += len(want)
		return true
	}
	return false
}

func (p *strictParser) enter() error {
	p.depth++
	if p.depth > maxJSONDepth {
		p.depth--
		return fmt.Errorf("%w: nesting depth exceeds limit", ErrInvalidJSON)
	}
	return nil
}

func (p *strictParser) leave() { p.depth-- }

func validateUnicodeString(s string) error {
	for _, r := range s {
		if isNoncharacter(r) {
			return fmt.Errorf("%w: Unicode noncharacter", ErrInvalidJSON)
		}
	}
	return nil
}

func isNoncharacter(r rune) bool {
	return r >= 0xFDD0 && r <= 0xFDEF || r >= 0 && (r&0xFFFF == 0xFFFE || r&0xFFFF == 0xFFFF)
}

func writeCanonical(out *bytes.Buffer, v jsonValue) {
	switch x := v.(type) {
	case jsonObject:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return utf16Less(keys[i], keys[j]) })
		out.WriteByte('{')
		for i, k := range keys {
			if i != 0 {
				out.WriteByte(',')
			}
			writeCanonicalString(out, k)
			out.WriteByte(':')
			writeCanonical(out, x[k])
		}
		out.WriteByte('}')
	case jsonArray:
		out.WriteByte('[')
		for i, item := range x {
			if i != 0 {
				out.WriteByte(',')
			}
			writeCanonical(out, item)
		}
		out.WriteByte(']')
	case string:
		writeCanonicalString(out, x)
	case uint64:
		out.WriteString(strconv.FormatUint(x, 10))
	case bool:
		if x {
			out.WriteString("true")
		} else {
			out.WriteString("false")
		}
	}
}

func utf16Less(a, b string) bool {
	aa := utf16.Encode([]rune(a))
	bb := utf16.Encode([]rune(b))
	for i := 0; i < len(aa) && i < len(bb); i++ {
		if aa[i] != bb[i] {
			return aa[i] < bb[i]
		}
	}
	return len(aa) < len(bb)
}

func writeCanonicalString(out *bytes.Buffer, s string) {
	const hex = "0123456789abcdef"
	out.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"', '\\':
			out.WriteByte('\\')
			out.WriteRune(r)
		case '\b':
			out.WriteString("\\b")
		case '\t':
			out.WriteString("\\t")
		case '\n':
			out.WriteString("\\n")
		case '\f':
			out.WriteString("\\f")
		case '\r':
			out.WriteString("\\r")
		default:
			if r < 0x20 {
				out.WriteString("\\u00")
				out.WriteByte(hex[byte(r)>>4])
				out.WriteByte(hex[byte(r)&0x0f])
			} else {
				out.WriteRune(r)
			}
		}
	}
	out.WriteByte('"')
}
