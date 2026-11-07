package knowledgeengine

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// This bounded schema subset is used only with compiled-digest owner resources.
// Open objects carry native SDK values without reproducing provider semantics.
// Transport JSON still forbids floats, unsafe integers and duplicate members.
func srrapivTransportShape(schema, value any, depth int) bool {
	if depth > maxJSONDepth {
		return false
	}
	if boolean, ok := schema.(bool); ok {
		return boolean
	}
	s, ok := schema.(map[string]any)
	if !ok {
		return false
	}
	for k := range s {
		switch k {
		case "type", "const", "enum", "anyOf", "properties", "required", "additionalProperties", "pattern", "maxLength", "minLength", "items", "minItems", "maxItems", "description", "format", "minimum", "maximum":
		default:
			return false
		}
	}
	if c, ok := s["const"]; ok && !reflect.DeepEqual(c, value) {
		return false
	}
	if alternatives, ok := s["anyOf"]; ok {
		variants, ok := alternatives.([]any)
		if !ok || len(variants) == 0 {
			return false
		}
		matched := false
		for _, variant := range variants {
			matched = srrapivTransportShape(variant, value, depth+1) || matched
		}
		if !matched {
			return false
		}
	}
	if enumeration, ok := s["enum"]; ok {
		options, ok := enumeration.([]any)
		if !ok {
			return false
		}
		matched := false
		for _, option := range options {
			matched = reflect.DeepEqual(option, value) || matched
		}
		if !matched {
			return false
		}
	}
	count := func(key string, n int) bool {
		v, present := s[key]
		if !present {
			return true
		}
		bound, ok := v.(int64)
		if !ok || bound < 0 {
			return false
		}
		if strings.HasPrefix(key, "min") {
			return int64(n) >= bound
		}
		return int64(n) <= bound
	}
	switch s["type"] {
	case nil:
	case "null":
		return value == nil
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "integer", "number":
		n, ok := value.(int64)
		if !ok {
			return false
		}
		for _, key := range []string{"minimum", "maximum"} {
			if v, exists := s[key]; exists {
				bound, ok := v.(int64)
				if !ok || (key == "minimum" && n < bound) || (key == "maximum" && n > bound) {
					return false
				}
			}
		}
	case "string":
		str, ok := value.(string)
		if !ok || !count("minLength", utf8.RuneCountInString(str)) || !count("maxLength", utf8.RuneCountInString(str)) {
			return false
		}
		if pattern, exists := s["pattern"]; exists {
			p, ok := pattern.(string)
			if !ok {
				return false
			}
			re, err := regexp.Compile(p)
			if err != nil || !re.MatchString(str) {
				return false
			}
		}
		if format, exists := s["format"]; exists {
			n, err := strconv.ParseUint(str, 10, 64)
			if format != "uint64-decimal" || err != nil || strconv.FormatUint(n, 10) != str {
				return false
			}
		}
	case "array":
		a, ok := value.([]any)
		if !ok || !count("minItems", len(a)) || !count("maxItems", len(a)) {
			return false
		}
		items, hasItems := s["items"]
		if hasItems {
			for _, element := range a {
				if !srrapivTransportShape(items, element, depth+1) {
					return false
				}
			}
		}
	case "object":
		object, ok := value.(map[string]any)
		if !ok {
			return false
		}
		properties := map[string]any{}
		if v, exists := s["properties"]; exists {
			properties, ok = v.(map[string]any)
			if !ok {
				return false
			}
		}
		if v, exists := s["required"]; exists {
			required, ok := v.([]any)
			if !ok {
				return false
			}
			for _, member := range required {
				key, ok := member.(string)
				if !ok {
					return false
				}
				if _, exists := object[key]; !exists {
					return false
				}
			}
		}
		additional, exists := s["additionalProperties"]
		if !exists {
			additional = true
		}
		for key, element := range object {
			rule, exists := properties[key]
			if !exists {
				rule = additional
			}
			if !srrapivTransportShape(rule, element, depth+1) {
				return false
			}
		}
	default:
		return false
	}
	return true
}
