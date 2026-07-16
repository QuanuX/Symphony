package stavprotocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// EncodeCandidate validates and canonically encodes a candidate.
func EncodeCandidate(v Candidate) ([]byte, error) {
	return encodeTyped(v, MaxCandidateBytes, v.Validate)
}

// DecodeCandidate requires canonical wire bytes and decodes a candidate.
func DecodeCandidate(data []byte) (Candidate, error) {
	return decodeTyped[Candidate](data, MaxCandidateBytes, func(v Candidate) error { return v.Validate() })
}

func EncodeEvent(v Event) ([]byte, error) {
	return encodeTyped(v, MaxEventBytes, v.Validate)
}

func DecodeEvent(data []byte) (Event, error) {
	return decodeTyped[Event](data, MaxEventBytes, func(v Event) error { return v.Validate() })
}

func EncodeReceipt(v Receipt) ([]byte, error) {
	return encodeTyped(v, MaxResponseBytes, v.Validate)
}

func DecodeReceipt(data []byte) (Receipt, error) {
	return decodeTyped[Receipt](data, MaxResponseBytes, func(v Receipt) error { return v.Validate() })
}

func EncodeQuery(v Query) ([]byte, error) {
	return encodeTyped(v, MaxRequestBytes, v.Validate)
}

func DecodeQuery(data []byte) (Query, error) {
	return decodeTyped[Query](data, MaxRequestBytes, func(v Query) error { return v.Validate() })
}

func EncodeQueryPage(v QueryPage) ([]byte, error) {
	return encodeTyped(v, MaxResponseBytes, v.Validate)
}

func DecodeQueryPage(data []byte) (QueryPage, error) {
	return decodeTyped[QueryPage](data, MaxResponseBytes, func(v QueryPage) error { return v.Validate() })
}

func EncodeVerification(v Verification) ([]byte, error) {
	return encodeTyped(v, MaxResponseBytes, v.Validate)
}

func DecodeVerification(data []byte) (Verification, error) {
	return decodeTyped[Verification](data, MaxResponseBytes, func(v Verification) error { return v.Validate() })
}

func encodeTyped[T any](v T, max int, validate func() error) ([]byte, error) {
	if err := validate(); err != nil {
		return nil, err
	}
	b, err := MarshalCanonical(v)
	if err != nil {
		return nil, err
	}
	if len(b) > max {
		return nil, fmt.Errorf("stav: canonical document exceeds size limit")
	}
	return b, nil
}

func decodeTyped[T any](data []byte, max int, validate func(T) error) (T, error) {
	var zero T
	if len(data) == 0 || len(data) > max {
		return zero, fmt.Errorf("stav: canonical document size out of range")
	}
	ast, err := parseStrict(data)
	if err != nil {
		return zero, err
	}
	var canonical bytes.Buffer
	writeCanonical(&canonical, ast)
	if !bytes.Equal(data, canonical.Bytes()) {
		return zero, ErrNonCanonicalJSON
	}
	t := reflect.TypeOf(zero)
	if err := validateExactShape(ast, t); err != nil {
		return zero, err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var out T
	if err := dec.Decode(&out); err != nil {
		return zero, fmt.Errorf("stav: typed decode: %w", err)
	}
	if err := validate(out); err != nil {
		return zero, err
	}
	return out, nil
}

func validateExactShape(v jsonValue, t reflect.Type) error {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Struct:
		o, ok := v.(jsonObject)
		if !ok {
			return fmt.Errorf("stav: object required")
		}
		type jsonField struct {
			typeOf   reflect.Type
			required bool
		}
		fields := make(map[string]jsonField)
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" {
				continue
			}
			tag := f.Tag.Get("json")
			parts := strings.Split(tag, ",")
			name := parts[0]
			if name == "-" {
				continue
			}
			if name == "" {
				name = f.Name
			}
			required := true
			for _, option := range parts[1:] {
				if option == "omitempty" {
					required = false
				}
			}
			fields[name] = jsonField{typeOf: f.Type, required: required}
		}
		seen := make(map[string]struct{}, len(o))
		for name, child := range o {
			field, exists := fields[name]
			if !exists {
				return fmt.Errorf("stav: unknown or case-mismatched member")
			}
			seen[name] = struct{}{}
			if err := validateExactShape(child, field.typeOf); err != nil {
				return err
			}
		}
		for name, field := range fields {
			if field.required {
				if _, exists := seen[name]; !exists {
					return fmt.Errorf("stav: required member %q missing", name)
				}
			}
		}
	case reflect.Slice, reflect.Array:
		a, ok := v.(jsonArray)
		if !ok {
			return fmt.Errorf("stav: array required")
		}
		for _, child := range a {
			if err := validateExactShape(child, t.Elem()); err != nil {
				return err
			}
		}
	case reflect.String:
		if _, ok := v.(string); !ok {
			return fmt.Errorf("stav: string required")
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if _, ok := v.(uint64); !ok {
			return fmt.Errorf("stav: safe integer required")
		}
	case reflect.Bool:
		if _, ok := v.(bool); !ok {
			return fmt.Errorf("stav: boolean required")
		}
	default:
		return fmt.Errorf("stav: unsupported typed field")
	}
	return nil
}
