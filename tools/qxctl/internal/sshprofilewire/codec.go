package sshprofilewire

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"unicode/utf8"
)

var ErrDocument = errors.New("transport_profile.invalid_document")

// CheckJSON bounds control documents and rejects duplicate names, invalid
// Unicode and numeric aliases before encoding/json can normalize them away.
func CheckJSON(raw []byte) error {
	if len(raw) == 0 || len(raw) > MaxDocumentBytes || !utf8.Valid(raw) || !validEscapes(raw) {
		return ErrDocument
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var walk func(int) error
	walk = func(depth int) error {
		if depth > 64 {
			return ErrDocument
		}
		t, err := d.Token()
		if err != nil {
			return ErrDocument
		}
		if n, ok := t.(json.Number); ok {
			v, e := strconv.ParseInt(string(n), 10, 64)
			if e != nil || v < -9007199254740991 || v > 9007199254740991 || strconv.FormatInt(v, 10) != string(n) {
				return ErrDocument
			}
		}
		if p, ok := t.(json.Delim); ok {
			switch p {
			case '{':
				seen := map[string]bool{}
				for d.More() {
					t, e := d.Token()
					if e != nil {
						return ErrDocument
					}
					k, ok := t.(string)
					if !ok || seen[k] {
						return ErrDocument
					}
					seen[k] = true
					if walk(depth+1) != nil {
						return ErrDocument
					}
				}
			case '[':
				for d.More() {
					if walk(depth+1) != nil {
						return ErrDocument
					}
				}
			default:
				return ErrDocument
			}
			if _, err = d.Token(); err != nil {
				return ErrDocument
			}
		}
		return nil
	}
	if walk(0) != nil {
		return ErrDocument
	}
	if _, err := d.Token(); err != io.EOF {
		return ErrDocument
	}
	return nil
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
		n, e := strconv.ParseUint(string(raw[i+1:i+5]), 16, 16)
		if e != nil {
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
			low, e := strconv.ParseUint(string(raw[i+3:i+7]), 16, 16)
			if e != nil || low < 0xdc00 || low > 0xdfff {
				return false
			}
			i += 6
		}
	}
	return true
}

// Canonical uses the declared Go encoding: sorted object keys, standard Go
// JSON escaping, array order and explicit nulls. It does not claim RFC 8785.
func Canonical(value any) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, ErrDocument
	}
	var normalized any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if d.Decode(&normalized) != nil {
		return nil, ErrDocument
	}
	return json.Marshal(normalized)
}

func Decode(raw []byte, target any) error {
	if CheckJSON(raw) != nil {
		return ErrDocument
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(target) != nil {
		return ErrDocument
	}
	var original any
	d = json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if d.Decode(&original) != nil {
		return ErrDocument
	}
	a, err := Canonical(original)
	if err != nil {
		return ErrDocument
	}
	b, err := Canonical(target)
	if err != nil || !bytes.Equal(a, b) {
		return ErrDocument
	}
	return nil
}

// Digest excludes only the object's own digest member; nested identities are
// retained. Generation counters are strings and never pass through floats.
func Digest(protocol string, value any) string {
	raw, err := Canonical(value)
	if err != nil {
		return ""
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var object map[string]any
	if d.Decode(&object) != nil || object == nil {
		return ""
	}
	delete(object, "digest")
	body, err := json.Marshal(object)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(append([]byte(protocol+"\x00"), body...))
	const digits = "0123456789abcdef"
	result := make([]byte, 71)
	copy(result, "sha256:")
	for i, b := range sum {
		result[7+2*i] = digits[b>>4]
		result[8+2*i] = digits[b&15]
	}
	return string(result)
}
