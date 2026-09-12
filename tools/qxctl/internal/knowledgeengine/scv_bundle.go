package knowledgeengine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

const scvBundleProtocol = "symphony.scv.evidence-bundle.v1"
const scvBundleMaxObjects = 2048
const scvBundleMaxReferences = 8192
const scvBundleMaxSteps = 16384
const scvBundleMaterializedBytes = 64 << 20

var scvBundleInspectionLimitations = []string{
	"bundle validation establishes exact bounded reconstruction only; it does not validate composition semantics or authenticate evidence producers",
	"all dependencies are explicit and complete; no ambient source, graph, filesystem or network lookup occurs",
}
var scvBundleEvaluationLimitations = []string{
	"bundled execution preserves the existing composition owner's meanings and caller criteria; it does not certify deployment or authenticate submitted provenance",
	"content references reduce repeated transport; expanded values and graph work remain explicitly bounded",
	"historical installation identity belongs to retained owner records; a transport content digest is not a receipt",
}

func scvBundleOperation(op string) bool {
	switch op {
	case "composition_explore", "composition_reassess", "composition_obligations", "composition_followup":
		return true
	}
	return false
}

// The new codec uses nlohmann's integral canonical representation: in particular,
// JSON -0 becomes 0. Source strings and decimal strings are never normalized.
func scvBundleNormalize(value any) (any, error) {
	switch v := value.(type) {
	case json.Number:
		n, err := strconv.ParseInt(v.String(), 10, 64)
		if err != nil || n < -9007199254740991 || n > 9007199254740991 {
			return nil, fmt.Errorf("invalid bundle integer")
		}
		return n, nil
	case map[string]any:
		for key, child := range v {
			normalized, err := scvBundleNormalize(child)
			if err != nil {
				return nil, err
			}
			v[key] = normalized
		}
	case []any:
		for i, child := range v {
			normalized, err := scvBundleNormalize(child)
			if err != nil {
				return nil, err
			}
			v[i] = normalized
		}
	}
	return value, nil
}

// encoding/json replaces lone escaped UTF-16 surrogates. The native parser
// rejects them, so reject such wire spellings rather than hashing replacement text.
func scvBundleUnicode(raw []byte) error {
	for i := 0; i < len(raw); i++ {
		if raw[i] != '"' {
			continue
		}
		i++
		for ; i < len(raw) && raw[i] != '"'; i++ {
			if raw[i] != '\\' {
				continue
			}
			i++
			if i >= len(raw) || raw[i] != 'u' {
				continue
			}
			if i+4 >= len(raw) {
				return fmt.Errorf("invalid bundle Unicode escape")
			}
			unit, err := strconv.ParseUint(string(raw[i+1:i+5]), 16, 16)
			if err != nil {
				return err
			}
			i += 4
			if unit >= 0xDC00 && unit <= 0xDFFF {
				return fmt.Errorf("unpaired bundle Unicode surrogate")
			}
			if unit >= 0xD800 && unit <= 0xDBFF {
				if i+6 >= len(raw) || raw[i+1] != '\\' || raw[i+2] != 'u' {
					return fmt.Errorf("unpaired bundle Unicode surrogate")
				}
				low, err := strconv.ParseUint(string(raw[i+3:i+7]), 16, 16)
				if err != nil || low < 0xDC00 || low > 0xDFFF {
					return fmt.Errorf("unpaired bundle Unicode surrogate")
				}
				i += 6
			}
		}
	}
	return nil
}

func scvBundleObject(raw []byte) (map[string]any, error) {
	if err := validateJSONObject(raw, maxResponseBytes); err != nil {
		return nil, err
	}
	if err := scvBundleUnicode(raw); err != nil {
		return nil, err
	}
	var value map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	normalized, err := scvBundleNormalize(value)
	if err != nil {
		return nil, err
	}
	return normalized.(map[string]any), nil
}

func scvBundleKeys(value map[string]any) []string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// SCVBundleEncode mechanically encodes one bounded logical JSON object. This
// validates transport only; callers still invoke the exact native operation.
func SCVBundleEncode(input json.RawMessage) (json.RawMessage, error) {
	root, err := scvBundleObject(input)
	if err != nil {
		return nil, err
	}
	nodes := map[string]map[string]any{}
	var walk func(any) (map[string]any, error)
	walk = func(value any) (map[string]any, error) {
		var node map[string]any
		switch v := value.(type) {
		case map[string]any:
			members := map[string]any{}
			for _, key := range scvBundleKeys(v) {
				child, err := walk(v[key])
				if err != nil {
					return nil, err
				}
				members[key] = child
			}
			node = map[string]any{"kind": "object", "members": members}
		case []any:
			items := []any{}
			for _, v := range v {
				child, err := walk(v)
				if err != nil {
					return nil, err
				}
				items = append(items, child)
			}
			node = map[string]any{"kind": "array", "items": items}
		default:
			return map[string]any{"scalar": value}, nil
		}
		id, err := SCVDigest(value)
		if err != nil {
			return nil, err
		}
		node["digest"] = id
		if _, exists := nodes[id]; !exists {
			if len(nodes) >= scvBundleMaxObjects {
				return nil, fmt.Errorf("bundle object limit exceeded")
			}
			nodes[id] = node
		}
		return map[string]any{"ref": id}, nil
	}
	child, err := walk(root)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(nodes))
	for id := range nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	objects := []any{}
	for _, id := range ids {
		objects = append(objects, nodes[id])
	}
	bundle := map[string]any{"protocol": scvBundleProtocol, "root_digest": child["ref"], "objects": objects}
	bundle["digest"], err = SCVDigest(bundle)
	if err != nil {
		return nil, err
	}
	raw, err := SCVCanonical(bundle)
	if err != nil {
		return nil, err
	}
	parsed, err := scvBundleObject(raw)
	if err != nil {
		return nil, err
	}
	if _, _, err = scvBundleExpand(parsed); err != nil {
		return nil, err
	}
	return raw, nil
}

type scvBundleSize struct{ bytes, values, depth int }
type scvBundleNode struct {
	value    map[string]any
	state    int
	size     scvBundleSize
	expanded any
}

func scvBundleAdd(total *int, added, limit int) error {
	if added < 0 || *total > limit-added {
		return fmt.Errorf("bundle expansion budget exceeded")
	}
	*total += added
	return nil
}

// All aggregate sizes are computed before materialization. Every repeated edge
// contributes its complete subtree size to the selected logical root.
func scvBundleExpand(bundle map[string]any) (json.RawMessage, map[string]any, error) {
	if !scvCorpusFields(bundle, "protocol", "root_digest", "objects", "digest") || bundle["protocol"] != scvBundleProtocol {
		return nil, nil, fmt.Errorf("invalid bundle fields")
	}
	if err := scvSeal(bundle, "digest"); err != nil {
		return nil, nil, err
	}
	root, ok := bundle["root_digest"].(string)
	if !ok || !taggedSHA256(root) {
		return nil, nil, fmt.Errorf("invalid bundle root")
	}
	objects, ok := bundle["objects"].([]any)
	if !ok || len(objects) == 0 || len(objects) > scvBundleMaxObjects {
		return nil, nil, fmt.Errorf("invalid bundle object count")
	}
	nodes := map[string]*scvBundleNode{}
	refs := 0
	last := ""
	childCheck := func(raw any) error {
		child, ok := raw.(map[string]any)
		if !ok || len(child) != 1 {
			return fmt.Errorf("invalid bundle child")
		}
		if ref, exists := child["ref"]; exists {
			id, ok := ref.(string)
			if !ok || !taggedSHA256(id) {
				return fmt.Errorf("invalid bundle reference")
			}
			refs++
			if refs > scvBundleMaxReferences {
				return fmt.Errorf("bundle reference limit exceeded")
			}
			return nil
		}
		scalar, exists := child["scalar"]
		if !exists {
			return fmt.Errorf("invalid bundle child tag")
		}
		switch scalar.(type) {
		case nil, bool, int64, string:
			return nil
		default:
			return fmt.Errorf("bundle scalar cannot contain a container or noninteger")
		}
	}
	for _, raw := range objects {
		node, ok := raw.(map[string]any)
		if !ok {
			return nil, nil, fmt.Errorf("invalid bundle node")
		}
		id, ok := node["digest"].(string)
		if !ok || !taggedSHA256(id) || id <= last {
			return nil, nil, fmt.Errorf("bundle object identities must be unique and sorted")
		}
		last = id
		switch node["kind"] {
		case "object":
			if !scvCorpusFields(node, "digest", "kind", "members") {
				return nil, nil, fmt.Errorf("invalid object node fields")
			}
			members, ok := node["members"].(map[string]any)
			if !ok {
				return nil, nil, fmt.Errorf("invalid object node members")
			}
			for _, child := range members {
				if err := childCheck(child); err != nil {
					return nil, nil, err
				}
			}
		case "array":
			if !scvCorpusFields(node, "digest", "kind", "items") {
				return nil, nil, fmt.Errorf("invalid array node fields")
			}
			items, ok := node["items"].([]any)
			if !ok {
				return nil, nil, fmt.Errorf("invalid array node items")
			}
			for _, child := range items {
				if err := childCheck(child); err != nil {
					return nil, nil, err
				}
			}
		default:
			return nil, nil, fmt.Errorf("unsupported bundle node kind")
		}
		nodes[id] = &scvBundleNode{value: node}
	}
	if len(nodes)+refs > scvBundleMaxSteps {
		return nil, nil, fmt.Errorf("bundle traversal budget exceeded")
	}
	if nodes[root] == nil || nodes[root].value["kind"] != "object" {
		return nil, nil, fmt.Errorf("bundle root must resolve to object")
	}
	materialized, visited := 0, 0
	var measure func(string, int) (scvBundleSize, error)
	var measureChild func(any, int) (scvBundleSize, error)
	measureChild = func(raw any, depth int) (scvBundleSize, error) {
		child := raw.(map[string]any)
		if ref, ok := child["ref"]; ok {
			return measure(ref.(string), depth)
		}
		scalar, err := SCVCanonical(child["scalar"])
		if err != nil {
			return scvBundleSize{}, err
		}
		return scvBundleSize{bytes: len(scalar), values: 1}, nil
	}
	measure = func(id string, pathDepth int) (scvBundleSize, error) {
		if pathDepth > maxJSONDepth {
			return scvBundleSize{}, fmt.Errorf("bundle expansion depth exceeded")
		}
		node := nodes[id]
		if node == nil {
			return scvBundleSize{}, fmt.Errorf("missing bundle node")
		}
		if node.state == 1 {
			return scvBundleSize{}, fmt.Errorf("cyclic bundle reference")
		}
		if node.state == 2 {
			if pathDepth+node.size.depth > maxJSONDepth {
				return scvBundleSize{}, fmt.Errorf("bundle expansion depth exceeded")
			}
			return node.size, nil
		}
		node.state = 1
		size := scvBundleSize{bytes: 2, values: 1}
		count := 0
		addChild := func(key *string, child any) error {
			if count > 0 {
				if err := scvBundleAdd(&size.bytes, 1, maxResponseBytes); err != nil {
					return err
				}
			}
			count++
			if key != nil {
				encoded, err := SCVCanonical(*key)
				if err != nil {
					return err
				}
				if err = scvBundleAdd(&size.bytes, len(encoded)+1, maxResponseBytes); err != nil {
					return err
				}
				if err = scvBundleAdd(&size.values, 1, maxJSONValues); err != nil {
					return err
				}
			}
			sub, err := measureChild(child, pathDepth+1)
			if err != nil {
				return err
			}
			if err = scvBundleAdd(&size.bytes, sub.bytes, maxResponseBytes); err != nil {
				return err
			}
			if err = scvBundleAdd(&size.values, sub.values, maxJSONValues); err != nil {
				return err
			}
			if sub.depth+1 > size.depth {
				size.depth = sub.depth + 1
			}
			if size.depth > maxJSONDepth || pathDepth+size.depth > maxJSONDepth {
				return fmt.Errorf("bundle expansion depth exceeded")
			}
			return nil
		}
		if node.value["kind"] == "object" {
			members := node.value["members"].(map[string]any)
			for _, key := range scvBundleKeys(members) {
				if err := addChild(&key, members[key]); err != nil {
					return scvBundleSize{}, err
				}
			}
		} else {
			for _, child := range node.value["items"].([]any) {
				if err := addChild(nil, child); err != nil {
					return scvBundleSize{}, err
				}
			}
		}
		if err := scvBundleAdd(&materialized, size.bytes, scvBundleMaterializedBytes); err != nil {
			return scvBundleSize{}, err
		}
		visited++
		node.size = size
		node.state = 2
		return size, nil
	}
	rootSize, err := measure(root, 0)
	if err != nil {
		return nil, nil, err
	}
	if visited != len(nodes) {
		return nil, nil, fmt.Errorf("unreachable bundle node")
	}
	var expand func(string) (any, error)
	var expandChild func(any) (any, error)
	expandChild = func(raw any) (any, error) {
		child := raw.(map[string]any)
		if ref, ok := child["ref"]; ok {
			return expand(ref.(string))
		}
		return child["scalar"], nil
	}
	expand = func(id string) (any, error) {
		node := nodes[id]
		if node.expanded != nil {
			return node.expanded, nil
		}
		var value any
		if node.value["kind"] == "object" {
			out := map[string]any{}
			for key, child := range node.value["members"].(map[string]any) {
				v, err := expandChild(child)
				if err != nil {
					return nil, err
				}
				out[key] = v
			}
			value = out
		} else {
			out := []any{}
			for _, child := range node.value["items"].([]any) {
				v, err := expandChild(child)
				if err != nil {
					return nil, err
				}
				out = append(out, v)
			}
			value = out
		}
		digest, err := SCVDigest(value)
		if err != nil || digest != id {
			return nil, fmt.Errorf("bundle expanded content digest mismatch")
		}
		node.expanded = value
		return value, nil
	}
	value, err := expand(root)
	if err != nil {
		return nil, nil, err
	}
	raw, err := SCVCanonical(value)
	if err != nil {
		return nil, nil, err
	}
	if len(raw) != rootSize.bytes {
		return nil, nil, fmt.Errorf("bundle expanded byte accounting mismatch")
	}
	metrics := map[string]any{"object_count": len(nodes), "reference_count": refs, "traversal_steps": len(nodes) + refs, "expanded_bytes": rootSize.bytes, "expanded_values": rootSize.values, "expanded_depth": rootSize.depth, "materialized_bytes": materialized}
	return raw, metrics, nil
}

// SCVBundleDecode verifies complete closure and returns canonical logical bytes.
// It makes no claim about the logical composition's semantics or its producer.
func SCVBundleDecode(bundle json.RawMessage) (json.RawMessage, error) {
	parsed, err := scvBundleObject(bundle)
	if err != nil {
		return nil, err
	}
	raw, _, err := scvBundleExpand(parsed)
	return raw, err
}

func validateSCVBundleResult(operation string, input, result map[string]any) error {
	// Normalize only this new path; native parsing canonicalizes integral -0.
	inBytes, err := SCVCanonical(input)
	if err != nil {
		return err
	}
	input, err = scvBundleObject(inBytes)
	if err != nil {
		return err
	}
	outBytes, err := SCVCanonical(result)
	if err != nil {
		return err
	}
	result, err = scvBundleObject(outBytes)
	if err != nil {
		return err
	}
	if !scvCorpusFields(input, "operation", "owner", "bundle") {
		return fmt.Errorf("invalid bundle invocation fields")
	}
	logical, ok := input["operation"].(string)
	if !ok || !scvBundleOperation(logical) {
		return fmt.Errorf("unsupported bundled operation")
	}
	owner, ok := input["owner"].(map[string]any)
	if !ok || !scvCorpusFields(owner, "domain", "version") {
		return fmt.Errorf("invalid bundle owner")
	}
	domain, ok := owner["domain"].(string)
	if !ok || !scvCoverageAdmits(domain, domain) || owner["version"] != "0.9.0-dev" {
		return fmt.Errorf("unsupported exact bundle owner")
	}
	if err = scvSeal(result, "digest"); err != nil {
		return err
	}
	inputDigest, err := SCVDigest(input)
	if err != nil {
		return err
	}
	if !scvEqual(result["owner"], owner) || result["domain"] != domain || result["operation"] != logical || result["input_digest"] != inputDigest {
		return fmt.Errorf("bundle result input or owner correspondence mismatch")
	}
	bundle, ok := input["bundle"].(map[string]any)
	if !ok {
		return fmt.Errorf("missing input bundle")
	}
	expanded, metrics, err := scvBundleExpand(bundle)
	if err != nil {
		return err
	}
	if operation == "bundle_inspect" {
		if !scvCorpusFields(result, "protocol", "domain", "operation", "owner", "input_digest", "root_digest", "metrics", "validation", "limitations", "digest") || result["protocol"] != "symphony.scv.bundle-inspection.v1" || result["validation"] != "transport_only" || result["root_digest"] != bundle["root_digest"] || !scvEqual(result["metrics"], metrics) || !scvEqual(result["limitations"], scvBundleInspectionLimitations) {
			return fmt.Errorf("bundle inspection attribution mismatch")
		}
		return nil
	}
	if operation != "composition_bundle_evaluate" || !scvCorpusFields(result, "protocol", "domain", "operation", "owner", "input_digest", "input_root_digest", "input_metrics", "result_bundle", "native_result_digest", "result_metrics", "validation", "limitations", "digest") || result["protocol"] != "symphony.scv.composition-bundle-evaluation.v1" || result["validation"] != "owner_evaluated" || result["input_root_digest"] != bundle["root_digest"] || !scvEqual(result["input_metrics"], metrics) || !scvEqual(result["limitations"], scvBundleEvaluationLimitations) {
		return fmt.Errorf("bundle evaluation attribution mismatch")
	}
	output, ok := result["result_bundle"].(map[string]any)
	if !ok {
		return fmt.Errorf("missing result bundle")
	}
	expandedResult, resultMetrics, err := scvBundleExpand(output)
	if err != nil {
		return err
	}
	native, err := scvBundleObject(expandedResult)
	if err != nil {
		return err
	}
	if native["domain"] != domain || !scvEqual(result["result_metrics"], resultMetrics) || result["native_result_digest"] != native["digest"] {
		return fmt.Errorf("bundled native result identity mismatch")
	}
	return ValidateSCVResult(logical, expanded, expandedResult)
}

// Preserve the original wire escape spelling until strict bundle decoding. The
// generic JSON decoder otherwise replaces unpaired UTF-16 escapes before the
// new consumer could distinguish them from a literal replacement character.
func validateSCVBundleWireResult(operation string, input, result []byte) error {
	payload, err := scvBundleObject(input)
	if err != nil {
		return err
	}
	value, err := scvBundleObject(result)
	if err != nil {
		return err
	}
	return validateSCVBundleResult(operation, payload, value)
}
