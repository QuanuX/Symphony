package sovcloudflare

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strconv"
)

// matches implements the deliberately small schema vocabulary used by the
// receipt-owned declaration. It is not advertised as a general JSON Schema
// engine; native semantic validation remains authoritative.
func matches(value any, schema any) bool {
	s, ok := schema.(map[string]any)
	if !ok {
		return false
	}
	if choices, ok := s["oneOf"].([]any); ok {
		count := 0
		for _, choice := range choices {
			if matches(value, choice) {
				count++
			}
		}
		return count == 1
	}
	if exact, ok := s["const"]; ok && !bytes.Equal(Canonical(exact), Canonical(value)) {
		return false
	}
	if choices, ok := s["enum"].([]any); ok {
		found := false
		for _, choice := range choices {
			if bytes.Equal(Canonical(choice), Canonical(value)) {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	limit := func(key string, fallback int64) int64 {
		if n, ok := s[key].(json.Number); ok {
			v, err := strconv.ParseInt(string(n), 10, 64)
			if err == nil {
				return v
			}
		}
		return fallback
	}
	switch s["type"] {
	case nil:
		return true
	case "null":
		return value == nil
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "string":
		v, ok := value.(string)
		if !ok || int64(len(v)) > limit("maxLength", 65536) {
			return false
		}
		if pattern, ok := s["pattern"].(string); ok {
			re, err := regexp.Compile(pattern)
			return err == nil && re.MatchString(v)
		}
		return true
	case "integer":
		n, ok := value.(json.Number)
		if !ok {
			return false
		}
		v, err := strconv.ParseInt(string(n), 10, 64)
		return err == nil && v >= limit("minimum", -9007199254740991) && v <= limit("maximum", 9007199254740991)
	case "array":
		a, ok := value.([]any)
		if !ok || int64(len(a)) < limit("minItems", 0) || int64(len(a)) > limit("maxItems", 500000) {
			return false
		}
		if item, ok := s["items"]; ok {
			for _, v := range a {
				if !matches(v, item) {
					return false
				}
			}
		}
		return true
	case "object":
		m, ok := value.(map[string]any)
		if !ok {
			return false
		}
		properties, has := s["properties"].(map[string]any)
		if !has {
			return true
		}
		if required, ok := s["required"].([]any); ok {
			for _, k := range required {
				key, ok := k.(string)
				if !ok {
					return false
				}
				if _, ok = m[key]; !ok {
					return false
				}
			}
		}
		for k, v := range m {
			sub, ok := properties[k]
			if !ok || !matches(v, sub) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
