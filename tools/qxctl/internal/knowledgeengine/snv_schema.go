package knowledgeengine

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// snvTransportShape is the finite JSON Schema subset in compiled owner
// resources. It resolves only local definitions and checks transport shape;
// identity, materiality, connectivity and naming reducers remain native.
func snvTransportShape(schema, value any, document map[string]any, depth int) bool {
	rule, okay := schema.(map[string]any)
	if !okay || depth > 64 {
		return false
	}
	if _, present := rule["$defs"]; present {
		document = rule
	}
	for key := range rule {
		switch key {
		case "$schema", "$id", "$ref", "$defs", "title", "description", "type", "const", "enum", "anyOf", "oneOf", "allOf", "if", "then", "else", "not", "properties", "required", "additionalProperties", "pattern", "format", "maxLength", "minLength", "items", "maxItems", "minItems", "uniqueItems", "minimum", "maximum", "maxProperties", "minProperties":
		default:
			return false
		}
	}
	if reference, present := rule["$ref"]; present {
		text, okay := reference.(string)
		if !okay || !strings.HasPrefix(text, "#/$defs/") {
			return false
		}
		defs, okay := document["$defs"].(map[string]any)
		if !okay {
			return false
		}
		target, present := defs[strings.TrimPrefix(text, "#/$defs/")]
		if !present || !snvTransportShape(target, value, document, depth+1) {
			return false
		}
	}
	if constant, present := rule["const"]; present && !reflect.DeepEqual(constant, value) {
		return false
	}
	if options, present := rule["enum"]; present {
		items, okay := options.([]any)
		if !okay {
			return false
		}
		matched := false
		for _, item := range items {
			if reflect.DeepEqual(item, value) {
				matched = true
			}
		}
		if !matched {
			return false
		}
	}
	for _, keyword := range []string{"anyOf", "oneOf", "allOf"} {
		if variants, present := rule[keyword]; present {
			options, okay := variants.([]any)
			if !okay || len(options) == 0 {
				return false
			}
			matches := 0
			for _, option := range options {
				if snvTransportShape(option, value, document, depth+1) {
					matches++
				}
			}
			if keyword == "anyOf" && matches == 0 || keyword == "oneOf" && matches != 1 || keyword == "allOf" && matches != len(options) {
				return false
			}
		}
	}
	if condition, present := rule["if"]; present {
		branch := "else"
		if snvTransportShape(condition, value, document, depth+1) {
			branch = "then"
		}
		if option, present := rule[branch]; present && !snvTransportShape(option, value, document, depth+1) {
			return false
		}
	}
	if negative, present := rule["not"]; present && snvTransportShape(negative, value, document, depth+1) {
		return false
	}
	count := func(minimum, maximum string, n int) bool {
		if limit, present := rule[minimum]; present {
			value, okay := limit.(int64)
			if !okay || value < 0 || int64(n) < value {
				return false
			}
		}
		if limit, present := rule[maximum]; present {
			value, okay := limit.(int64)
			if !okay || value < 0 || int64(n) > value {
				return false
			}
		}
		return true
	}
	kind, present := rule["type"]
	if present {
		names := []any{kind}
		if union, okay := kind.([]any); okay {
			names = union
		}
		if len(names) == 0 || len(names) > 6 {
			return false
		}
		matched, seen := false, map[string]bool{}
		for _, item := range names {
			name, okay := item.(string)
			if !okay || seen[name] {
				return false
			}
			seen[name] = true
			match, admitted := snvTransportType(name, value)
			if !admitted {
				return false
			}
			matched = matched || match
		}
		if !matched {
			return false
		}
	}
	switch item := value.(type) {
	case int64:
		if minimum, present := rule["minimum"]; present {
			n, okay := minimum.(int64)
			if !okay || item < n {
				return false
			}
		}
		if maximum, present := rule["maximum"]; present {
			n, okay := maximum.(int64)
			if !okay || item > n {
				return false
			}
		}
	case string:
		if !count("minLength", "maxLength", utf8.RuneCountInString(item)) {
			return false
		}
		if pattern, present := rule["pattern"]; present {
			text, okay := pattern.(string)
			if !okay {
				return false
			}
			converted, err := snvSchemaPattern(text)
			if err != nil {
				return false
			}
			matcher, err := regexp.Compile(converted)
			if err != nil || !matcher.MatchString(item) {
				return false
			}
		}
		if format, present := rule["format"]; present {
			if format != "date-time" {
				return false
			}
			if _, err := time.Parse(time.RFC3339, item); err != nil {
				return false
			}
		}
	case []any:
		if !count("minItems", "maxItems", len(item)) {
			return false
		}
		if shape, present := rule["items"]; present {
			for _, value := range item {
				if !snvTransportShape(shape, value, document, depth+1) {
					return false
				}
			}
		}
		if unique, present := rule["uniqueItems"]; present {
			enabled, okay := unique.(bool)
			if !okay {
				return false
			}
			if enabled {
				seen := map[string]bool{}
				for _, value := range item {
					raw, err := SCVCanonical(value)
					if err != nil || seen[string(raw)] {
						return false
					}
					seen[string(raw)] = true
				}
			}
		}
	case map[string]any:
		if !count("minProperties", "maxProperties", len(item)) {
			return false
		}
		if required, present := rule["required"]; present {
			names, okay := required.([]any)
			if !okay {
				return false
			}
			for _, name := range names {
				key, okay := name.(string)
				if !okay {
					return false
				}
				if _, present := item[key]; !present {
					return false
				}
			}
		}
		properties := map[string]any{}
		if value, present := rule["properties"]; present {
			var okay bool
			properties, okay = value.(map[string]any)
			if !okay {
				return false
			}
		}
		for key, value := range item {
			if shape, present := properties[key]; present {
				if !snvTransportShape(shape, value, document, depth+1) {
					return false
				}
				continue
			}
			if additional, present := rule["additionalProperties"]; present {
				switch setting := additional.(type) {
				case bool:
					if !setting {
						return false
					}
				case map[string]any:
					if !snvTransportShape(setting, value, document, depth+1) {
						return false
					}
				default:
					return false
				}
			}
		}
	}
	return true
}

func snvTransportType(name string, value any) (matched, admitted bool) {
	switch name {
	case "object":
		_, matched = value.(map[string]any)
	case "array":
		_, matched = value.([]any)
	case "integer":
		_, matched = value.(int64)
	case "string":
		_, matched = value.(string)
	case "boolean":
		_, matched = value.(bool)
	case "null":
		matched = value == nil
	default:
		return false, false
	}
	return matched, true
}

// JSON Schema's ECMA Unicode escapes are translated before Go RE2 parsing.
// External regexes and references are never accepted as schema resources.
func snvSchemaPattern(pattern string) (string, error) {
	var result strings.Builder
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == '\\' && i+1 < len(pattern) {
			if pattern[i+1] == 'u' {
				if i+6 > len(pattern) {
					return "", fmt.Errorf("Incomplete schema Unicode escape")
				}
				cp, err := strconv.ParseUint(pattern[i+2:i+6], 16, 16)
				if err != nil || cp >= 0xd800 && cp <= 0xdfff {
					return "", fmt.Errorf("Unsupported schema Unicode escape")
				}
				result.WriteRune(rune(cp))
				i += 5
				continue
			}
			result.WriteByte(pattern[i])
			i++
			result.WriteByte(pattern[i])
			continue
		}
		result.WriteByte(pattern[i])
	}
	return result.String(), nil
}
