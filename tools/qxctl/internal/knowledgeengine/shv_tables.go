package knowledgeengine

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

var shvScopedSelector = regexp.MustCompile(`^[a-z][a-z0-9-]*#[^#]+$`)

func shvTableSelector(v any) bool {
	s := shvText(v)
	if !shvBoundedText(s, 128) || !shvScopedSelector.MatchString(s) {
		return false
	}
	switch strings.SplitN(s, "#", 2)[0] {
	case "table", "tr", "th", "td", "h1", "dt", "dd":
		return false
	}
	return true
}
func shvMappingShape(m map[string]any, version string) bool {
	if m["interpretation_profile"] == "pdf_opn.v1" {
		return (version == SHVDocumentVersion || version == SHVKernelInterfaceVersion) && shvFields(m, "id", "manufacturer", "model", "hardware_class", "source_id", "heading_section", "interpretation_profile", "document", "fields") && m["heading_section"] == "table8" && shvFields(shvMap(m["document"]), "decoder_root", "extraction")
	}

	if _, tagged := m["interpretation_profile"]; tagged {
		return (version == SHVTableVersion || (version == SHVDocumentVersion || version == SHVKernelInterfaceVersion)) && m["interpretation_profile"] == "scoped_tables.v1" && shvFields(m, "id", "manufacturer", "model", "hardware_class", "source_id", "heading_section", "interpretation_profile", "fields") && (len(shvList(m["fields"])) == 0 || shvTableSelector(m["heading_section"]))
	}
	return shvFields(m, "id", "manufacturer", "model", "hardware_class", "source_id", "heading_section", "field_section", "fields") && shvBoundedText(m["field_section"], 128) && m["heading_section"] != m["field_section"]
}
func shvFieldShape(f, m map[string]any, version string) bool {
	if m["interpretation_profile"] == "pdf_opn.v1" {
		return (version == SHVDocumentVersion || version == SHVKernelInterfaceVersion) && shvFields(f, "predicate", "label", "next_label", "value_type", "qualifier") && f["label"] == "OPN" && f["next_label"] == "Model" && f["value_type"] == "string" && f["qualifier"] == "issuer=AMD;namespace=opn;profile=1"
	}

	if m["interpretation_profile"] == "scoped_tables.v1" {
		if (version != SHVTableVersion && (version != SHVDocumentVersion && version != SHVKernelInterfaceVersion)) || !shvTableSelector(f["section"]) || f["section"] == m["heading_section"] {
			return false
		}
		if f["predicate"] == "model_introduction" && f["value_type"] != "date" && f["value_type"] != "quarter_20yy" {
			return false
		}
		if f["value_type"] == "table_rows" {
			return shvFields(f, "predicate", "section", "columns", "value_type", "qualifier") && shvTableColumns(f["columns"])
		}
		return shvFields(f, "predicate", "section", "label", "next_label", "value_type", "qualifier") && shvBoundedText(f["label"], 256) && (f["next_label"] == nil || shvBoundedText(f["next_label"], 256))
	}
	if f["predicate"] == "model_introduction" && f["value_type"] != "date" {
		return false
	}
	return shvFields(f, "predicate", "label", "next_label", "value_type", "qualifier") && shvBoundedText(f["label"], 256) && shvBoundedText(f["next_label"], 256)
}
func shvTableColumns(v any) bool {
	a, ok := v.([]any)
	if !ok || len(a) < 1 || len(a) > 16 || !shvStrings(a, false) {
		return false
	}
	for _, v := range a {
		if !shvBoundedText(v, 256) {
			return false
		}
	}
	return true
}
func shvQuarter(text string) (map[string]any, error) {
	if !regexp.MustCompile(`^Q[1-4]'[0-9][0-9]$`).MatchString(text) {
		return nil, shvFail()
	}
	year := 2000 + int(text[3]-'0')*10 + int(text[4]-'0')
	month := time.Month(1 + int(text[1]-'1')*3)
	from := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	through := from.AddDate(0, 3, -1)
	return map[string]any{"precision": "quarter", "source_text": text, "from": from.Format("2006-01-02"), "through": through.Format("2006-01-02")}, nil
}
func shvStructuredValue(v any, kind string, field map[string]any) bool {
	m := shvMap(v)
	if kind == "quarter_20yy" {
		q, e := shvQuarter(shvText(m["source_text"]))
		return e == nil && scvEqual(q, m)
	}
	if kind != "table_rows" || !shvFields(m, "columns", "rows") || !scvEqual(m["columns"], field["columns"]) || !shvTableColumns(m["columns"]) {
		return false
	}
	rows, ok := m["rows"].([]any)
	if !ok || len(rows) < 1 || len(rows) > 32 {
		return false
	}
	seen := map[string]bool{}
	for _, v := range rows {
		row, ok := v.([]any)
		if !ok || len(row) != len(shvList(m["columns"])) {
			return false
		}
		for _, cell := range row {
			s, ok := cell.(string)
			if !ok || len(s) > 4096 || !utf8.ValidString(s) {
				return false
			}
			for _, r := range s {
				if r < 32 || r == 127 {
					return false
				}
			}
		}
		key := shvText(row[0])
		if key == "" || seen[key] {
			return false
		}
		seen[key] = true
	}
	return true
}

type shvTableCell struct{ kind, text string }
type shvTableRow []shvTableCell

// This scanner consumes a finite HTML grammar; it does not apply browser repair.
// Scope identity, table/cell pairing, and source byte replay are independent of
// the native interpreter. Other hardware modes remain separate complete rows.
func shvHTMLTable(raw, model, headingSection, fieldSection string) ([]shvTableRow, error) {
	if !utf8.ValidString(raw) || strings.ContainsRune(raw, 0) || headingSection == fieldSection || !shvTableSelector(headingSection) || !shvTableSelector(fieldSection) {
		return nil, shvFail()
	}
	type scope struct {
		tag, id      string
		depth, count int
	}
	h, f := strings.SplitN(headingSection, "#", 2), strings.SplitN(fieldSection, "#", 2)
	hs, fs := scope{tag: h[0], id: h[1]}, scope{tag: f[0], id: f[1]}
	scopes := []*scope{&hs, &fs}
	rows := []shvTableRow{}
	row := shvTableRow{}
	tableOpen, rowOpen := false, false
	tables, heads := 0, 0
	active, heading := "", ""
	group := ""
	var buf strings.Builder
	pos := 0
	for pos < len(raw) {
		relative := shvMarkup.FindStringIndex(raw[pos:])
		if relative == nil {
			if strings.ContainsRune(raw[pos:], '<') {
				return nil, shvFail()
			}
			if active != "" {
				buf.WriteString(raw[pos:])
			} else if tableOpen && strings.Trim(raw[pos:], " \t\r\n\f") != "" {
				return nil, shvFail()
			}
			break
		}
		start, end := pos+relative[0], pos+relative[1]
		text := raw[pos:start]
		if strings.ContainsRune(text, '<') {
			return nil, shvFail()
		}
		if tableOpen && active == "" && strings.Trim(text, " \t\r\n\f") != "" {
			return nil, shvFail()
		}
		if active != "" {
			buf.WriteString(text)
			if buf.Len() > 65536 {
				return nil, shvFail()
			}
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
		if tableOpen && closing && strings.Trim(tag[len(parts[0]):len(tag)-1], " \t\r\n\f") != "" {
			return nil, shvFail()
		}
		if !closing && (name == "script" || name == "style") {
			match := regexp.MustCompile(`(?i)</` + name + `[\t\n\r\f />]`).FindStringIndex(raw[pos:])
			if match == nil {
				return nil, shvFail()
			}
			last := strings.IndexByte(raw[pos+match[0]:], '>')
			if last < 0 {
				return nil, shvFail()
			}
			pos += match[0] + last + 1
			continue
		}
		if !closing {
			id, has, e := shvHTMLID(tag, len(parts[0]))
			if e != nil {
				return nil, e
			}
			if has && id == hs.id && name == hs.tag && fs.depth > 0 {
				return nil, shvFail()
			}
			for _, s := range scopes {
				if s.depth > 0 && name == s.tag && !self {
					s.depth++
				}
				if has && id == s.id && name == s.tag {
					s.count++
					if s.count != 1 || self {
						return nil, shvFail()
					}
					s.depth = 1
				}
			}
		}
		if tableOpen && active == "" && name != "table" && name != "tr" && name != "th" && name != "td" && name != "thead" && name != "tbody" && name != "tfoot" {
			return nil, shvFail()
		}
		selectedHeading := name == "h1" && hs.depth > 0 && fs.depth == 0
		if selectedHeading {
			if !closing {
				if active != "" || self {
					return nil, shvFail()
				}
				active = "h1"
				buf.Reset()
			} else {
				if active != "h1" || buf.Len() > 65536 {
					return nil, shvFail()
				}
				value, e := shvNormalize(buf.String())
				if e != nil {
					return nil, e
				}
				heading = value
				heads++
				active = ""
				buf.Reset()
			}
		} else if fs.depth > 0 && (name == "table" || name == "tr" || name == "th" || name == "td" || name == "thead" || name == "tbody" || name == "tfoot") {
			switch name {
			case "table":
				if closing {
					if !tableOpen || rowOpen || active != "" || group != "" {
						return nil, shvFail()
					}
					tableOpen = false
				} else {
					if self || tableOpen || rowOpen || active != "" || tables > 0 {
						return nil, shvFail()
					}
					tables++
					tableOpen = true
				}
			case "thead", "tbody", "tfoot":
				if !tableOpen || rowOpen || active != "" || self {
					return nil, shvFail()
				}
				if closing {
					if group != name {
						return nil, shvFail()
					}
					group = ""
				} else {
					if group != "" {
						return nil, shvFail()
					}
					group = name
				}
			case "tr":
				if closing {
					if !tableOpen || !rowOpen || active != "" || len(row) == 0 {
						return nil, shvFail()
					}
					rows = append(rows, row)
					if len(rows) > 256 {
						return nil, shvFail()
					}
					rowOpen = false
				} else {
					if self || !tableOpen || rowOpen || active != "" {
						return nil, shvFail()
					}
					rowOpen = true
					row = shvTableRow{}
				}
			case "th", "td":
				if closing {
					if !tableOpen || !rowOpen || active != name || buf.Len() > 65536 {
						return nil, shvFail()
					}
					value, e := shvNormalize(buf.String())
					if e != nil || len(value) > 4096 {
						return nil, shvFail()
					}
					row = append(row, shvTableCell{name, value})
					if len(row) > 16 {
						return nil, shvFail()
					}
					active = ""
					buf.Reset()
				} else {
					if self || !tableOpen || !rowOpen || active != "" {
						return nil, shvFail()
					}
					span, e := shvHTMLSpan(tag, len(parts[0]))
					if e != nil || span {
						return nil, shvFail()
					}
					active = name
					buf.Reset()
				}
			}
		} else if active != "" {
			buf.WriteByte(' ')
		}
		if buf.Len() > 65536 {
			return nil, shvFail()
		}
		if closing {
			for _, s := range scopes {
				if s.depth > 0 && name == s.tag {
					s.depth--
					if s.depth == 0 && (active != "" || rowOpen || tableOpen || group != "") {
						return nil, shvFail()
					}
				}
			}
		}
	}
	if hs.depth != 0 || fs.depth != 0 || hs.count != 1 || fs.count != 1 || heads != 1 || heading != model || active != "" || tableOpen || rowOpen || tables != 1 || len(rows) == 0 || group != "" {
		return nil, shvFail()
	}
	return rows, nil
}
func shvTableValue(rows []shvTableRow, f map[string]any) (any, error) {
	kind := shvText(f["value_type"])
	if kind == "table_rows" {
		columns := shvList(f["columns"])
		if !shvTableColumns(columns) || len(rows) < 2 || len(rows) > 33 || len(rows[0]) != len(columns) {
			return nil, shvFail()
		}
		for i, c := range rows[0] {
			if c.kind != "th" || c.text != columns[i] {
				return nil, shvFail()
			}
		}
		values := []any{}
		for _, row := range rows[1:] {
			if len(row) != len(columns) {
				return nil, shvFail()
			}
			v := []any{}
			for _, c := range row {
				if c.kind != "td" {
					return nil, shvFail()
				}
				v = append(v, c.text)
			}
			values = append(values, v)
		}
		out := map[string]any{"columns": columns, "rows": values}
		if !shvStructuredValue(out, kind, f) {
			return nil, shvFail()
		}
		return out, nil
	}
	seen := map[string]bool{}
	index := -1
	for i, row := range rows {
		if len(row) != 2 || row[0].kind != "th" || row[1].kind != "td" || row[0].text == "" || seen[row[0].text] {
			return nil, shvFail()
		}
		seen[row[0].text] = true
		if row[0].text == f["label"] {
			index = i
		}
	}
	if index < 0 || (f["next_label"] == nil && index != len(rows)-1) || (f["next_label"] != nil && (index+1 >= len(rows) || rows[index+1][0].text != f["next_label"])) {
		return nil, shvFail()
	}
	text := rows[index][1].text
	if kind == "quarter_20yy" {
		return shvQuarter(text)
	}
	return shvSourceValue(text, kind)
}
func shvValidateTableSubject(raw string, m, s map[string]any) error {
	actual := map[string]map[string]any{}
	for _, v := range shvList(s["assertions"]) {
		a := shvMap(v)
		actual[shvText(a["predicate"])] = a
	}
	for _, v := range shvList(m["fields"]) {
		f := shvMap(v)
		rows, e := shvHTMLTable(raw, shvText(m["model"]), shvText(m["heading_section"]), shvText(f["section"]))
		if e != nil {
			return e
		}
		want, e := shvTableValue(rows, f)
		if e != nil {
			return e
		}
		if !scvEqual(want, actual[shvText(f["predicate"])]["value"]) {
			return fmt.Errorf("SHV table interpretation differs from retained source")
		}
	}
	return nil
}

func shvHTMLSpan(tag string, pos int) (bool, error) {
	end := len(tag) - 1

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
			return false, shvFail()
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
				return false, shvFail()
			}
			if tag[pos] == '\'' || tag[pos] == '"' {
				quote := tag[pos]
				pos++
				start = pos
				for pos < end && tag[pos] != quote {
					pos++
				}
				if pos == end {
					return false, shvFail()
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
		_ = value
		if name == "rowspan" || name == "colspan" {
			has = true
		}
	}
	return has, nil
}
