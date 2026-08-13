package foundationlifecycle

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	maxRequestBytes  = 1 << 20
	maxResponseBytes = 4 << 20
	maxJSONDepth     = 64
	maxJSONValues    = 65536
)

var safeTokenPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]+$`)

func DecodeCommand(reader io.Reader) (Command, error) {
	raw, err := io.ReadAll(io.LimitReader(reader, maxRequestBytes+1))
	if err != nil {
		return Command{}, fmt.Errorf("read foundation lifecycle command: %w", err)
	}
	if len(raw) == 0 || len(raw) > maxRequestBytes {
		return Command{}, fmt.Errorf("foundation lifecycle command exceeds its bound")
	}
	if err := validateJSONShape(raw); err != nil {
		return Command{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var command Command
	if err := decoder.Decode(&command); err != nil {
		return Command{}, fmt.Errorf("decode foundation lifecycle command: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return Command{}, fmt.Errorf("foundation lifecycle command must contain one JSON value")
	}
	if err := command.validate(); err != nil {
		return Command{}, err
	}
	return command, nil
}

func EncodeResult(writer io.Writer, result Result) error {
	encoded, err := json.Marshal(result)
	if err != nil {
		return err
	}
	if len(encoded)+1 > maxResponseBytes {
		return fmt.Errorf("foundation lifecycle result exceeds its bound")
	}
	_, err = writer.Write(append(encoded, '\n'))
	return err
}

func (command Command) validate() error {
	if command.Protocol != CommandProtocol || command.FormatVersion != 1 || command.Component != "ssiag" ||
		!oneOf(command.Operation, "observe", "plan", "apply", "apply_status", "recover") ||
		!oneOf(command.Surface, "enrollment", "supervisor") || !oneOf(command.Scope, "user", "system") {
		return fmt.Errorf("foundation lifecycle command identity is invalid")
	}
	if !validTOPSID(command.TOPSID) || !validSTSC(command.RequestedAt) || !validSTSC(command.DeadlineAt) {
		return fmt.Errorf("foundation lifecycle command identity or time is invalid")
	}
	requested, _ := time.Parse(time.RFC3339, command.RequestedAt)
	deadline, _ := time.Parse(time.RFC3339, command.DeadlineAt)
	if !deadline.After(requested) || deadline.Sub(requested) > time.Minute {
		return fmt.Errorf("foundation lifecycle command deadline is invalid")
	}
	for _, value := range []*string{command.OperationID, command.ExpectedStateDigest, command.ExpectedAttemptDigest} {
		if value != nil && *value != "absent" && !validTokenOrDigest(*value) {
			return fmt.Errorf("foundation lifecycle command expected state or operation identity is invalid")
		}
	}
	for _, value := range []*string{command.RequestID, command.CorrelationID} {
		if value != nil && !validTOPSID(*value) {
			return fmt.Errorf("foundation lifecycle command request identity is invalid")
		}
	}
	switch command.Operation {
	case "observe", "apply_status":
		if command.Intent != nil || command.Plan != nil || command.Discover {
			return fmt.Errorf("read-only foundation lifecycle command has mutation fields")
		}
	case "plan":
		if command.Intent == nil || command.Plan != nil || command.Discover || command.OperationID == nil || command.RequestID == nil || command.CorrelationID == nil || command.ExpectedStateDigest == nil {
			return fmt.Errorf("foundation lifecycle plan command is incomplete")
		}
		if err := command.Intent.validate(command.Surface); err != nil {
			return err
		}
	case "apply":
		if command.Plan == nil || command.Intent != nil || command.Discover || command.OperationID == nil || command.ExpectedAttemptDigest == nil {
			return fmt.Errorf("foundation lifecycle apply command is incomplete")
		}
		if err := command.Plan.validate(); err != nil {
			return err
		}
	case "recover":
		if command.OperationID == nil || command.Intent != nil || command.Plan != nil || command.Discover == (command.ExpectedAttemptDigest != nil) {
			return fmt.Errorf("foundation lifecycle recovery command is invalid")
		}
	}
	return nil
}

func (intent Intent) validate(surface string) error {
	if !oneOf(intent.AuditMode, "ordinary", "audit_deferred") || intent.TTLSeconds == 0 || intent.TTLSeconds > 600 {
		return fmt.Errorf("foundation lifecycle intent is invalid")
	}
	if (intent.ServiceUID == nil) != (intent.ServiceGID == nil) || intent.AuthorityUID != nil || intent.AuthorityGID != nil {
		return fmt.Errorf("SSIAG lifecycle intent identity is invalid")
	}
	if surface == "enrollment" {
		if !oneOf(intent.DesiredState, "enrolled", "unenrolled_preserved") || intent.DesiredState == "enrolled" && (intent.TOPSName == nil || *intent.TOPSName == "") {
			return fmt.Errorf("SSIAG enrollment lifecycle intent is invalid")
		}
	} else if !oneOf(intent.DesiredState, "native_running", "native_installed_stopped", "absent_stopped") || intent.TOPSName != nil || intent.ServiceUID != nil {
		return fmt.Errorf("SSIAG supervisor lifecycle intent is invalid")
	}
	return nil
}

func (plan Plan) validate() error {
	if plan.Protocol != PlanProtocol || plan.FormatVersion != 1 || plan.Component != "ssiag" ||
		!oneOf(plan.Surface, "enrollment", "supervisor") || !oneOf(plan.Scope, "user", "system") || !validTOPSID(plan.TOPSID) ||
		!validToken(plan.OperationID) || !validTOPSID(plan.RequestID) || !validTOPSID(plan.CorrelationID) ||
		!validDigestOrAbsent(plan.ExpectedStateDigest) || !validSTSC(plan.CreatedAt) || !validSTSC(plan.ExpiresAt) || !validDigest(plan.PlanDigest) {
		return fmt.Errorf("foundation lifecycle plan is invalid")
	}
	intent := Intent{DesiredState: plan.DesiredState, TOPSName: plan.TOPSName, ServiceUID: plan.ServiceUID, ServiceGID: plan.ServiceGID, AuthorityUID: plan.AuthorityUID, AuthorityGID: plan.AuthorityGID, AuditMode: plan.AuditMode, TTLSeconds: 1}
	if err := intent.validate(plan.Surface); err != nil {
		return err
	}
	want, err := digestWithout(plan, "plan_digest")
	if err != nil || want != plan.PlanDigest {
		return fmt.Errorf("foundation lifecycle plan digest mismatch")
	}
	return nil
}

func validateJSONShape(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	depth, values := 0, 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("foundation lifecycle command is invalid JSON: %w", err)
		}
		values++
		if values > maxJSONValues {
			return fmt.Errorf("foundation lifecycle command has too many JSON values")
		}
		if delimiter, ok := token.(json.Delim); ok {
			switch delimiter {
			case '{', '[':
				depth++
				if depth > maxJSONDepth {
					return fmt.Errorf("foundation lifecycle command nesting exceeds its bound")
				}
			case '}', ']':
				depth--
			}
		}
	}
	return nil
}

func digestWithout(value any, field string) (string, error) {
	return digestWithoutFields(value, field)
}

func digestWithoutFields(value any, fields ...string) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&object); err != nil {
		return "", err
	}
	for _, field := range fields {
		delete(object, field)
	}
	return digestValue(object)
}

func digestValue(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func digestFile(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("refusing unsafe lifecycle evidence %s", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func canonicalExecutable() (string, error) {
	value, err := os.Executable()
	if err != nil {
		return "", err
	}
	value, err = filepath.EvalSymlinks(value)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(value) {
		return "", fmt.Errorf("foundation lifecycle executable is not absolute")
	}
	return filepath.Clean(value), nil
}

func validTOPSID(value string) bool {
	if len(value) != 36 || strings.ToLower(value) != value || value == "00000000-0000-0000-0000-000000000000" {
		return false
	}
	for index, char := range value {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if char != '-' {
				return false
			}
		} else if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
			return false
		}
	}
	return value[14] >= '1' && value[14] <= '8' && strings.Contains("89ab", value[19:20])
}

func validSTSC(value string) bool {
	parsed, err := time.Parse(time.RFC3339, value)
	return err == nil && parsed.Nanosecond() == 0 && parsed.UTC().Format(time.RFC3339) == value
}

func validToken(value string) bool {
	return len(value) >= 1 && len(value) <= 256 && safeTokenPattern.MatchString(value)
}
func validDigest(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}
func validDigestOrAbsent(value string) bool { return value == "absent" || validDigest(value) }
func validTokenOrDigest(value string) bool  { return validToken(value) || validDigest(value) }
func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
