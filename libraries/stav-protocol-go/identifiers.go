package stavprotocol

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	uuidPattern       = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-([0-9a-f])[0-9a-f]{3}-([0-9a-f])[0-9a-f]{3}-[0-9a-f]{12}$`)
	registeredPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*(\.[a-z][a-z0-9-]*)+$`)
	timestampPattern  = regexp.MustCompile(`^[0-9]{4}-(0[1-9]|1[0-2])-([0-2][0-9]|3[01])T([01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9]\.[0-9]{9}Z$`)
	digestPattern     = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
)

const timestampLayout = "2006-01-02T15:04:05.000000000Z"

// ValidateTOPSID validates the canonical stable TOPS identifier. Versions 1
// through 8 are accepted; nil and non-RFC variants are rejected.
func ValidateTOPSID(id string) error {
	m := uuidPattern.FindStringSubmatch(id)
	if m == nil || m[1] < "1" || m[1] > "8" || !strings.Contains("89ab", m[2]) {
		return fmt.Errorf("stav: invalid TOPS ID")
	}
	if id == "00000000-0000-0000-0000-000000000000" {
		return fmt.Errorf("stav: nil TOPS ID is prohibited")
	}
	return nil
}

// ValidateRequestUUID accepts canonical RFC 9562 UUIDv4 and UUIDv7 values.
func ValidateRequestUUID(id string) error {
	return validateUUIDVersion(id, "47")
}

// ValidateEventUUID accepts only an authority-generated RFC 9562 UUIDv4.
func ValidateEventUUID(id string) error {
	return validateUUIDVersion(id, "4")
}

func validateUUIDVersion(id, versions string) error {
	m := uuidPattern.FindStringSubmatch(id)
	if m == nil || !strings.Contains(versions, m[1]) || !strings.Contains("89ab", m[2]) {
		return fmt.Errorf("stav: invalid UUID")
	}
	return nil
}

// GenerateUUIDv4 returns a canonical UUIDv4 generated from crypto/rand.
func GenerateUUIDv4() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("stav: generate UUIDv4: %w", err)
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// FormatTimestamp returns the exact STAV v1 UTC nanosecond timestamp form.
func FormatTimestamp(t time.Time) string {
	return t.UTC().Format(timestampLayout)
}

func validateRegisteredIdentifier(s string) error {
	if len(s) < 3 || len(s) > 128 || !registeredPattern.MatchString(s) {
		return fmt.Errorf("stav: invalid registered identifier")
	}
	return nil
}

func validateOpaqueReference(s string) error {
	if len(s) < 1 || len(s) > 256 {
		return fmt.Errorf("stav: invalid opaque reference")
	}
	for i, c := range []byte(s) {
		ok := c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || strings.ContainsRune("._:@-", rune(c))
		if !ok || i == 0 && !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
			return fmt.Errorf("stav: invalid opaque reference")
		}
	}
	return nil
}

func validateDigest(s string) error {
	if !digestPattern.MatchString(s) {
		return fmt.Errorf("stav: invalid digest")
	}
	return nil
}

func validateTimestamp(s string) error {
	if !timestampPattern.MatchString(s) {
		return fmt.Errorf("stav: invalid timestamp")
	}
	if _, err := time.Parse(timestampLayout, s); err != nil {
		return fmt.Errorf("stav: invalid timestamp")
	}
	return nil
}
