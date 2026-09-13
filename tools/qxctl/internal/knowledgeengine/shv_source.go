package knowledgeengine

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

var shvMarkup = regexp.MustCompile(`(?s)<!--.*?-->|<(?:[^>"']|"[^"]*"|'[^']*')*>`)
var shvTagName = regexp.MustCompile(`^</?([^\t\r\n\f />]+)`)
var shvIgnored = regexp.MustCompile(`(?is)<(script|style)\b(?:[^>"']|"[^"]*"|'[^']*')*>`)
var shvEntities = map[string]string{"amp": "&", "lt": "<", "gt": ">", "quot": "\"", "apos": "'", "nbsp": " ", "reg": "®", "trade": "™", "copy": "©", "ndash": "–", "mdash": "—", "times": "×", "micro": "µ"}

func shvNormalize(s string) (string, error) {
	var out strings.Builder
	for len(s) > 0 {
		i := strings.IndexByte(s, '&')
		if i < 0 {
			out.WriteString(s)
			break
		}
		out.WriteString(s[:i])
		s = s[i+1:]
		end := strings.IndexByte(s, ';')
		if end < 0 || end > 15 {
			return "", shvFail()
		}
		entity := s[:end]
		s = s[end+1:]
		if strings.HasPrefix(entity, "#") {
			digits, base := entity[1:], 10
			if strings.HasPrefix(digits, "x") || strings.HasPrefix(digits, "X") {
				digits, base = digits[1:], 16
			}
			if digits == "" || strings.HasPrefix(digits, "+") || strings.HasPrefix(digits, "-") {
				return "", shvFail()
			}
			n, e := strconv.ParseUint(digits, base, 32)
			if e != nil || n == 0 || n > 0x10ffff || (n >= 0xd800 && n <= 0xdfff) {
				return "", shvFail()
			}
			out.WriteRune(rune(n))
		} else {
			x, ok := shvEntities[entity]
			if !ok {
				return "", shvFail()
			}
			out.WriteString(x)
		}
	}
	s = out.String()
	out.Reset()
	space := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\r' || r == '\n' || r == '\f' {
			space = out.Len() > 0
			continue
		}
		if r < 32 || r == 127 {
			return "", shvFail()
		}
		if space {
			out.WriteByte(' ')
		}
		out.WriteRune(r)
		space = false
	}
	return out.String(), nil
}

// A finite, nonexecuting token pass independently checks the owner's H1 and
// adjacent DT/DD profile. It deliberately does not use browser error repair.
func shvHTMLPairs(raw, model, headingSection, fieldSection string) ([][2]string, error) {
	if strings.ContainsRune(raw, 0) || !utf8.ValidString(raw) || headingSection == fieldSection || headingSection == "" || fieldSection == "" {
		return nil, shvFail()
	}
	type scope struct {
		id, tag, matchTag string
		depth, count      int
	}
	selectorPattern := regexp.MustCompile(`^[a-z][a-z0-9-]*#[^#]+$`)
	if !selectorPattern.MatchString(headingSection) || !selectorPattern.MatchString(fieldSection) {
		return nil, shvFail()
	}
	h, f := strings.SplitN(headingSection, "#", 2), strings.SplitN(fieldSection, "#", 2)
	headingScope, fieldScope := scope{id: h[1], matchTag: h[0]}, scope{id: f[1], matchTag: f[0]}
	scopes := []*scope{&headingScope, &fieldScope}
	pairs := [][2]string{}
	active, term, heading := "", "", ""
	var buf strings.Builder
	heads := 0
	waiting := false
	pos := 0
	for pos < len(raw) {
		relative := shvMarkup.FindStringIndex(raw[pos:])
		if relative == nil {
			if strings.ContainsRune(raw[pos:], '<') {
				return nil, shvFail()
			}
			if active != "" {
				buf.WriteString(raw[pos:])
			}
			break
		}
		start, end := pos+relative[0], pos+relative[1]
		text := raw[pos:start]
		if strings.ContainsRune(text, '<') {
			return nil, shvFail()
		}
		if active != "" {
			buf.WriteString(text)
		}
		tag := raw[start:end]
		pos = end
		if strings.HasPrefix(tag, "<!--") {
			continue
		}
		parts := shvTagName.FindStringSubmatch(tag)
		if len(parts) < 2 {
			continue
		}
		name := strings.ToLower(parts[1])
		if strings.HasPrefix(name, "!") || strings.HasPrefix(name, "?") {
			continue
		}
		closing := strings.HasPrefix(tag, "</")
		self := strings.HasSuffix(tag, "/>")
		if !closing && (name == "script" || name == "style") {
			closePattern := regexp.MustCompile(`(?i)</` + name + `[\t\n\r\f />]`)
			close := closePattern.FindStringIndex(raw[pos:])
			if close == nil {
				return nil, shvFail()
			}
			last := strings.IndexByte(raw[pos+close[0]:], '>')
			if last < 0 {
				return nil, shvFail()
			}
			pos += close[0] + last + 1
			continue
		}
		if !closing {
			elementID, hasID, e := shvHTMLID(tag, len(parts[0]))
			if e != nil {
				return nil, e
			}
			if hasID && elementID == headingScope.id && name == headingScope.matchTag && fieldScope.depth > 0 {
				return nil, shvFail()
			}
			for _, s := range scopes {
				if s.depth > 0 && name == s.tag && !self {
					s.depth++
				}
				if hasID && elementID == s.id && name == s.matchTag {
					s.count++
					if s.count != 1 || self || name == "h1" || name == "dt" || name == "dd" {
						return nil, shvFail()
					}
					s.tag = name
					s.depth = 1
				}
			}

		}
		selected := (name == "h1" && headingScope.depth > 0 && fieldScope.depth == 0) || ((name == "dt" || name == "dd") && fieldScope.depth > 0)
		if selected {
			if !closing {
				if active != "" || self || (name == "dt" && waiting) || (name == "dd" && !waiting) {
					return nil, shvFail()
				}
				active = name
				buf.Reset()
			} else {
				if active != name || buf.Len() > 65536 {
					return nil, shvFail()
				}
				value, e := shvNormalize(buf.String())
				if e != nil {
					return nil, e
				}
				active = ""
				buf.Reset()
				switch name {
				case "h1":
					heading = value
					heads++
				case "dt":
					if value == "" {
						return nil, shvFail()
					}
					term = value
					waiting = true
				case "dd":
					pairs = append(pairs, [2]string{term, value})
					waiting = false
					if len(pairs) > 256 {
						return nil, shvFail()
					}
				}
			}
		} else if active != "" {
			buf.WriteByte(' ')
		}
		if closing {
			for _, s := range scopes {
				if s.depth > 0 && name == s.tag {
					s.depth--
					if s.depth == 0 && (active != "" || waiting) {
						return nil, shvFail()
					}
				}
			}
		}
	}
	if headingScope.depth != 0 || fieldScope.depth != 0 || headingScope.count != 1 || fieldScope.count != 1 || active != "" || waiting || heads != 1 || heading != model {
		return nil, shvFail()
	}
	seen := map[string]bool{}
	for _, p := range pairs {
		if seen[p[0]] {
			return nil, shvFail()
		}
		seen[p[0]] = true
	}
	return pairs, nil
}
func shvHTMLID(tag string, pos int) (string, bool, error) {
	end := len(tag) - 1
	id := ""
	has := false
	white := func(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' }
	for pos < end {
		for pos < end && (white(tag[pos]) || tag[pos] == '/') {
			pos++
		}
		if pos == end {
			break
		}
		start := pos
		for pos < end && !white(tag[pos]) && tag[pos] != '=' && tag[pos] != '/' {
			pos++
		}
		name := strings.ToLower(tag[start:pos])
		if name == "" {
			return "", false, shvFail()
		}
		for pos < end && white(tag[pos]) {
			pos++
		}
		value := ""
		if pos < end && tag[pos] == '=' {
			pos++
			for pos < end && white(tag[pos]) {
				pos++
			}
			if pos == end {
				return "", false, shvFail()
			}
			if tag[pos] == '\'' || tag[pos] == '"' {
				quote := tag[pos]
				pos++
				start = pos
				for pos < end && tag[pos] != quote {
					pos++
				}
				if pos == end {
					return "", false, shvFail()
				}
				value = tag[start:pos]
				pos++
			} else {
				start = pos
				for pos < end && !white(tag[pos]) {
					pos++
				}
				value = tag[start:pos]
			}
		}
		if name == "id" {
			if has {
				return "", false, shvFail()
			}
			id = value
			has = true
		}
	}
	return id, has, nil
}

func shvSourceValue(text, kind string) (any, error) {
	if text == "" || len(text) > 4096 {
		return nil, shvFail()
	}
	switch kind {
	case "string":
		return text, nil
	case "integer":
		n, e := strconv.ParseInt(text, 10, 64)
		if e != nil || strconv.FormatInt(n, 10) != text || n < -9007199254740991 || n > 9007199254740991 {
			return nil, shvFail()
		}
		return json.Number(text), nil
	case "date":
		if len(text) != 10 || text[2] != '/' || text[5] != '/' {
			return nil, shvFail()
		}
		date := text[6:] + "-" + text[:2] + "-" + text[3:5]
		if !shvDate(date) {
			return nil, shvFail()
		}
		return date, nil
	case "tokens":
		tokens := strings.Split(text, "/")
		if len(tokens) > 32 {
			return nil, shvFail()
		}
		seen := map[string]bool{}
		for i, s := range tokens {
			tokens[i] = strings.Trim(s, " ")
			if tokens[i] == "" || len(tokens[i]) > 128 || seen[tokens[i]] {
				return nil, shvFail()
			}
			seen[tokens[i]] = true
		}
		sort.Strings(tokens)
		out := []any{}
		for _, s := range tokens {
			out = append(out, s)
		}
		return out, nil
	}
	return nil, shvFail()
}
func shvValidateBuild(p, r map[string]any) error {
	if !shvFields(p, "source_root", "sources", "subjects") || (!filepath.IsAbs(shvText(p["source_root"])) || filepath.Clean(shvText(p["source_root"])) != shvText(p["source_root"])) || !scvEqual(p["sources"], r["sources"]) || !scvEqual(p["subjects"], r["mapping"]) {
		return shvFail()
	}
	if e := shvCatalogue(r); e != nil {
		return e
	}
	rawSources := map[string]string{}
	formats := map[string]string{}
	for _, v := range shvList(r["sources"]) {
		s := shvMap(v)
		raw, e := readNoFollowRelative("/", strings.TrimPrefix(filepath.Join(shvText(p["source_root"]), shvText(s["path"])), "/"), 1<<20)
		if e != nil {
			return e
		}
		n, _ := s["bytes"].(json.Number).Int64()
		if int64(len(raw)) != n || digestBytes(raw) != s["digest"] {
			return fmt.Errorf("SHV consumed source bytes differ from declared identity")
		}
		rawSources[shvText(s["id"])] = string(raw)
		formats[shvText(s["id"])] = shvText(s["format"])
	}
	maps := map[string]map[string]any{}
	for _, v := range shvList(p["subjects"]) {
		m := shvMap(v)
		maps[shvText(m["id"])] = m
	}
	for _, v := range shvList(r["subjects"]) {
		s := shvMap(v)
		m := maps[shvText(s["id"])]
		fields := shvList(m["fields"])
		if len(fields) == 0 {
			continue
		}
		sid := shvText(m["source_id"])
		if formats[sid] != "html" {
			return shvFail()
		}
		pairs, e := shvHTMLPairs(rawSources[sid], shvText(m["model"]), shvText(m["heading_section"]), shvText(m["field_section"]))
		if e != nil {
			return e
		}
		actual := map[string]map[string]any{}
		for _, av := range shvList(s["assertions"]) {
			a := shvMap(av)
			actual[shvText(a["predicate"])] = a
		}
		for _, fv := range fields {
			f := shvMap(fv)
			found := false
			for i, pair := range pairs {
				if pair[0] != f["label"] {
					continue
				}
				if i+1 >= len(pairs) || pairs[i+1][0] != f["next_label"] {
					return shvFail()
				}
				want, e := shvSourceValue(pair[1], shvText(f["value_type"]))
				if e != nil {
					return e
				}
				if !scvEqual(actual[shvText(f["predicate"])]["value"], want) {
					return fmt.Errorf("SHV interpreted value differs from retained source")
				}
				found = true
			}
			if !found {
				return shvFail()
			}
		}
	}
	return nil
}
