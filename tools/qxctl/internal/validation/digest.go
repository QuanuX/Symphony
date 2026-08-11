package validation

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	digestPattern  = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	tokenPattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,255}$`)
	topsPattern    = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	versionPattern = regexp.MustCompile(`^[0-9A-Za-z.+-]{1,64}$`)
)

func canonicalJSON(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "")
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buffer.Bytes(), []byte("\n")), nil
}

func digestValue(value any) (string, error) {
	encoded, err := canonicalJSON(value)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func objectWithout(value any, fields ...string) (map[string]any, error) {
	encoded, err := canonicalJSON(value)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var object map[string]any
	if err := decoder.Decode(&object); err != nil {
		return nil, err
	}
	for _, field := range fields {
		delete(object, field)
	}
	return object, nil
}

func taggedDigest(value string) bool { return digestPattern.MatchString(value) }
func safeToken(value string) bool    { return tokenPattern.MatchString(value) }
func validTOPSID(value string) bool  { return topsPattern.MatchString(value) }
func safeVersion(value string) bool  { return versionPattern.MatchString(value) }

func exactUTCSeconds(value string) bool {
	parsed, err := time.Parse("2006-01-02T15:04:05Z", value)
	return err == nil && parsed.Format("2006-01-02T15:04:05Z") == value
}

func uniqueSorted(values []string) ([]string, error) {
	result := append([]string(nil), values...)
	sort.Strings(result)
	for index, value := range result {
		if !taggedDigest(value) {
			return nil, fmt.Errorf("digest collection contains invalid value")
		}
		if index > 0 && result[index-1] == value {
			return nil, fmt.Errorf("digest collection contains a duplicate")
		}
	}
	return result, nil
}

func stringPointer(value string) *string { return &value }

func sameStrings(left, right []string) bool {
	return strings.Join(left, "\x00") == strings.Join(right, "\x00")
}
