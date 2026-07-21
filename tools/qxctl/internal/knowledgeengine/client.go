package knowledgeengine

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	processProtocol  = "symphony.knowledge.engine-process.v1"
	receiptProtocol  = "symphony.knowledge.install-receipt.v1"
	moduleID         = "skvi-engine"
	engineID         = "symphony-skvi"
	sclvModuleID     = "sclv-engine"
	sclvEngineID     = "symphony-sclv"
	maxReceiptBytes  = 256 * 1024
	maxRequestBytes  = 1024 * 1024
	maxResponseBytes = 4 * 1024 * 1024
	maxJSONDepth     = 64
	maxJSONValues    = 16384
	maxStringBytes   = 65536
	operationTimeout = 5 * time.Second
)

type engineSpec struct {
	label         string
	moduleID      string
	engineID      string
	expectedFiles func(string) map[string]struct{}
}

var skviSpec = engineSpec{
	label: "SKVI", moduleID: moduleID, engineID: engineID, expectedFiles: expectedFiles,
}

var sclvSpec = engineSpec{
	label: "SCLV", moduleID: sclvModuleID, engineID: sclvEngineID, expectedFiles: expectedSCLVFiles,
}

type receipt struct {
	Protocol        string   `json:"protocol"`
	ModuleID        string   `json:"module_id"`
	Version         string   `json:"version"`
	InstallScope    string   `json:"install_scope"`
	PrefixMode      string   `json:"prefix_mode"`
	State           string   `json:"state"`
	Active          bool     `json:"active"`
	DefaultReceptor *string  `json:"default_receptor"`
	Files           []string `json:"files"`
}

type processRequest struct {
	Protocol       string          `json:"protocol"`
	RequestID      string          `json:"request_id"`
	CorrelationID  string          `json:"correlation_id"`
	Operation      string          `json:"operation"`
	TargetEngine   string          `json:"target_engine"`
	DeadlineUnixMS int64           `json:"deadline_unix_ms"`
	Payload        json.RawMessage `json:"payload"`
}

type Response struct {
	Protocol       string          `json:"protocol"`
	RequestID      string          `json:"request_id"`
	CorrelationID  string          `json:"correlation_id"`
	Operation      string          `json:"operation"`
	EngineID       string          `json:"engine_id"`
	EngineVersion  string          `json:"engine_version"`
	Outcome        string          `json:"outcome"`
	Result         json.RawMessage `json:"result"`
	Error          *ProcessError   `json:"error"`
	ResponseDigest string          `json:"response_digest"`
}

type ProcessError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *ProcessError) Error() string {
	return fmt.Sprintf("engine rejected request (%s): %s", e.Code, e.Message)
}

func Invoke(ctx context.Context, prefix, version, repositoryRoot, operation string, payload []byte) (Response, error) {
	return invoke(ctx, skviSpec, prefix, version, repositoryRoot, operation, payload)
}

func InvokeSCLV(ctx context.Context, prefix, version, repositoryRoot, operation string, payload []byte) (Response, error) {
	return invoke(ctx, sclvSpec, prefix, version, repositoryRoot, operation, payload)
}

func invoke(ctx context.Context, spec engineSpec, prefix, version, repositoryRoot, operation string, payload []byte) (Response, error) {
	if !safeVersion(version) {
		return Response{}, fmt.Errorf("invalid %s engine version", spec.label)
	}
	if !safeToken(operation, 64) {
		return Response{}, fmt.Errorf("invalid %s operation", spec.label)
	}
	if err := validateJSONObject(payload, maxRequestBytes); err != nil {
		return Response{}, fmt.Errorf("invalid %s operation payload: %w", spec.label, err)
	}

	binary, err := resolveInstalledFor(spec, prefix, version)
	if err != nil {
		return Response{}, err
	}
	repositoryRoot, err = canonicalDirectory(repositoryRoot, "repository root")
	if err != nil {
		return Response{}, err
	}

	requestID, err := randomToken()
	if err != nil {
		return Response{}, fmt.Errorf("generate %s request identity: %w", spec.label, err)
	}
	request := processRequest{
		Protocol:       processProtocol,
		RequestID:      requestID,
		CorrelationID:  requestID,
		Operation:      operation,
		TargetEngine:   spec.engineID,
		DeadlineUnixMS: time.Now().Add(operationTimeout).UnixMilli(),
		Payload:        json.RawMessage(payload),
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		return Response{}, fmt.Errorf("encode %s request: %w", spec.label, err)
	}
	if len(encoded) > maxRequestBytes {
		return Response{}, fmt.Errorf("encoded %s request exceeds %d bytes", spec.label, maxRequestBytes)
	}

	childContext, cancel := context.WithTimeout(ctx, operationTimeout+time.Second)
	defer cancel()
	command := exec.CommandContext(childContext, binary)
	command.Dir = repositoryRoot
	command.Env = []string{}
	command.Stdin = bytes.NewReader(encoded)
	stdout := &boundedBuffer{limit: maxResponseBytes}
	stderr := &boundedBuffer{limit: 64 * 1024}
	command.Stdout = stdout
	command.Stderr = stderr
	runErr := command.Run()
	if childContext.Err() != nil {
		return Response{}, fmt.Errorf("%s engine exceeded its hard process deadline: %w", spec.label, childContext.Err())
	}
	if stdout.exceeded {
		return Response{}, fmt.Errorf("%s engine response exceeds %d bytes", spec.label, maxResponseBytes)
	}
	if stderr.exceeded {
		return Response{}, fmt.Errorf("%s engine diagnostic output exceeds 65536 bytes", spec.label)
	}

	response, responseErr := validateResponseFor(
		spec, stdout.Bytes(), requestID, operation, version)
	if responseErr != nil {
		if runErr != nil {
			return Response{}, fmt.Errorf("%s engine process failed and returned an invalid response: %w", spec.label, responseErr)
		}
		return Response{}, responseErr
	}
	if response.Outcome == "error" {
		if runErr == nil {
			return Response{}, fmt.Errorf("%s engine returned an error outcome with a successful process status", spec.label)
		}
		var exitError *exec.ExitError
		if !errors.As(runErr, &exitError) || exitError.ExitCode() < 2 || exitError.ExitCode() > 5 {
			return Response{}, fmt.Errorf("%s engine returned an error outcome with an invalid process status", spec.label)
		}
		return response, response.Error
	}
	if runErr != nil {
		var exitError *exec.ExitError
		if errors.As(runErr, &exitError) {
			return Response{}, fmt.Errorf("%s engine returned success with process status %d", spec.label, exitError.ExitCode())
		}
		return Response{}, fmt.Errorf("execute %s engine: %w", spec.label, runErr)
	}
	if len(stderr.Bytes()) != 0 {
		return Response{}, fmt.Errorf("%s engine emitted diagnostics during a successful operation", spec.label)
	}
	return response, nil
}

func ReadPayload(path string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("--input is required for this knowledge-engine operation")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve knowledge-engine input: %w", err)
	}
	requested, err := os.Lstat(abs)
	if err != nil || requested.Mode()&os.ModeSymlink != 0 || !requested.Mode().IsRegular() {
		return nil, fmt.Errorf("knowledge-engine input must be a no-follow regular file")
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("canonicalize knowledge-engine input: %w", err)
	}
	resolvedInfo, err := os.Lstat(resolved)
	if err != nil || resolvedInfo.Mode()&os.ModeSymlink != 0 || !resolvedInfo.Mode().IsRegular() ||
		!os.SameFile(requested, resolvedInfo) {
		return nil, fmt.Errorf("knowledge-engine input changed during canonicalization")
	}
	root, relative, err := splitAbsolutePath(resolved)
	if err != nil {
		return nil, fmt.Errorf("resolve knowledge-engine input root: %w", err)
	}
	data, err := readNoFollowRelative(root, relative, maxRequestBytes)
	if err != nil {
		return nil, fmt.Errorf("read no-follow knowledge-engine input: %w", err)
	}
	if err := validateJSONObject(data, maxRequestBytes); err != nil {
		return nil, fmt.Errorf("invalid knowledge-engine input: %w", err)
	}
	return data, nil
}

func resolveInstalled(prefix, version string) (string, error) {
	return resolveInstalledFor(skviSpec, prefix, version)
}

func resolveInstalledFor(spec engineSpec, prefix, version string) (string, error) {
	if !safeVersion(version) {
		return "", fmt.Errorf("invalid %s engine version", spec.label)
	}
	prefix, err := canonicalDirectory(prefix, "installation prefix")
	if err != nil {
		return "", err
	}
	receiptRelative := filepath.ToSlash(filepath.Join(
		"share", "symphony", "receipts", spec.moduleID, version, "install-receipt.json"))
	binaryRelative := filepath.ToSlash(filepath.Join(
		"libexec", "symphony", spec.moduleID, version, spec.engineID))
	receiptBytes, err := readNoFollowRelative(prefix, receiptRelative, maxReceiptBytes)
	if err != nil {
		return "", fmt.Errorf("validate %s install receipt: %w", spec.label, err)
	}
	if err := validateJSONObject(receiptBytes, maxReceiptBytes); err != nil {
		return "", fmt.Errorf("invalid %s install receipt JSON: %w", spec.label, err)
	}
	if err := requireExactFields(receiptBytes, []string{
		"protocol", "module_id", "version", "install_scope", "prefix_mode",
		"state", "active", "default_receptor", "files",
	}); err != nil {
		return "", fmt.Errorf("invalid %s install receipt fields: %w", spec.label, err)
	}
	var installed receipt
	if err := decodeExact(receiptBytes, &installed); err != nil {
		return "", fmt.Errorf("decode %s install receipt: %w", spec.label, err)
	}
	if installed.Protocol != receiptProtocol || installed.ModuleID != spec.moduleID || installed.Version != version {
		return "", fmt.Errorf("%s install receipt identity mismatch", spec.label)
	}
	if installed.InstallScope != "prefix" || installed.PrefixMode != "installation_prefix" ||
		installed.State != "installed_undocked" || installed.Active || installed.DefaultReceptor != nil {
		return "", fmt.Errorf("%s install receipt lifecycle state is not the inactive undocked contract", spec.label)
	}
	expected := spec.expectedFiles(version)
	if len(installed.Files) != len(expected) {
		return "", fmt.Errorf("%s install receipt file set has %d entries, want %d", spec.label, len(installed.Files), len(expected))
	}
	seen := make(map[string]struct{}, len(installed.Files))
	for _, relative := range installed.Files {
		if !safeRelativePath(relative) {
			return "", fmt.Errorf("%s install receipt contains an unsafe path", spec.label)
		}
		if _, duplicate := seen[relative]; duplicate {
			return "", fmt.Errorf("%s install receipt contains a duplicate path", spec.label)
		}
		seen[relative] = struct{}{}
		if _, ok := expected[relative]; !ok {
			return "", fmt.Errorf("%s install receipt contains an unexpected path: %s", spec.label, relative)
		}
		if err := validateNoFollowRelative(prefix, relative, maxInstalledFileBytes(relative)); err != nil {
			return "", fmt.Errorf("validate receipt-owned file %s: %w", relative, err)
		}
	}
	for relative := range expected {
		if _, ok := seen[relative]; !ok {
			return "", fmt.Errorf("%s install receipt is missing required path: %s", spec.label, relative)
		}
	}
	binary := filepath.Join(prefix, filepath.FromSlash(binaryRelative))
	info, err := os.Lstat(binary)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("%s installed engine is not a no-follow executable regular file", spec.label)
	}
	return binary, nil
}

func expectedFiles(version string) map[string]struct{} {
	base := "share/doc/symphony/skvi-engine/" + version + "/"
	license := "share/licenses/symphony-skvi-engine/" + version + "/"
	paths := []string{
		"libexec/symphony/skvi-engine/" + version + "/symphony-skvi",
		"share/symphony/receipts/skvi-engine/" + version + "/install-receipt.json",
		base + "INTENT.md",
		base + "MANIFEST.md",
		base + "INSTALL.md",
		base + "SKILL.md",
		base + "SPEC.md",
		license + "LICENSE-AGPL-3.0",
		license + "nlohmann-json-LICENSE.MIT",
	}
	result := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		result[path] = struct{}{}
	}
	return result
}

func expectedSCLVFiles(version string) map[string]struct{} {
	base := "share/doc/symphony/sclv-engine/" + version + "/"
	license := "share/licenses/symphony-sclv-engine/" + version + "/"
	paths := []string{
		"libexec/symphony/sclv-engine/" + version + "/symphony-sclv",
		"libexec/symphony/sclv-engine/" + version + "/symphony-sclv-evidence-local-git",
		"libexec/symphony/sclv-engine/" + version + "/symphony-sclv-evidence-airgap",
		"share/symphony/receipts/sclv-engine/" + version + "/install-receipt.json",
		base + "INTENT.md",
		base + "MANIFEST.md",
		base + "INSTALL.md",
		base + "SKILL.md",
		base + "SPEC.md",
		license + "LICENSE-AGPL-3.0",
		license + "nlohmann-json-LICENSE.MIT",
	}
	result := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		result[path] = struct{}{}
	}
	return result
}

func maxInstalledFileBytes(relative string) int64 {
	if strings.HasPrefix(relative, "libexec/") {
		return 64 * 1024 * 1024
	}
	return 4 * 1024 * 1024
}

func canonicalDirectory(path, label string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("%s is required", label)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", label, err)
	}
	requested, err := os.Lstat(abs)
	if err != nil || requested.Mode()&os.ModeSymlink != 0 || !requested.IsDir() {
		return "", fmt.Errorf("%s must identify a directory rather than a symlink", label)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("canonicalize %s: %w", label, err)
	}
	resolvedInfo, err := os.Lstat(resolved)
	if err != nil || resolvedInfo.Mode()&os.ModeSymlink != 0 || !resolvedInfo.IsDir() ||
		!os.SameFile(requested, resolvedInfo) {
		return "", fmt.Errorf("%s changed during canonicalization", label)
	}
	return filepath.Clean(resolved), nil
}

func splitAbsolutePath(path string) (string, string, error) {
	clean := filepath.Clean(path)
	volume := filepath.VolumeName(clean)
	root := volume + string(os.PathSeparator)
	relative, err := filepath.Rel(root, clean)
	if err != nil || relative == "." {
		return "", "", fmt.Errorf("path must identify a descendant of its filesystem root")
	}
	relative = filepath.ToSlash(relative)
	if !safeRelativePath(relative) {
		return "", "", fmt.Errorf("path has unsafe components")
	}
	return root, relative, nil
}

func readNoFollowRelative(root, relative string, maxBytes int64) ([]byte, error) {
	file, err := openNoFollowRelative(root, relative, maxBytes)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("file exceeds %d bytes", maxBytes)
	}
	return data, nil
}

func validateNoFollowRelative(root, relative string, maxBytes int64) error {
	file, err := openNoFollowRelative(root, relative, maxBytes)
	if err != nil {
		return err
	}
	return file.Close()
}

func openNoFollowRelative(root, relative string, maxBytes int64) (*os.File, error) {
	if !safeRelativePath(relative) {
		return nil, fmt.Errorf("unsafe relative path")
	}
	components := strings.Split(relative, "/")
	file, err := openRelativeNoFollow(root, components)
	if err != nil {
		return nil, fmt.Errorf("no-follow open failed: %w", err)
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxBytes {
		_ = file.Close()
		if err == nil && info.Size() > maxBytes {
			return nil, fmt.Errorf("file exceeds %d bytes", maxBytes)
		}
		return nil, fmt.Errorf("no-follow target is not a regular file")
	}
	return file, nil
}

func validateResponse(data []byte, requestID, operation, version string) (Response, error) {
	return validateResponseFor(skviSpec, data, requestID, operation, version)
}

func validateResponseFor(spec engineSpec, data []byte, requestID, operation, version string) (Response, error) {
	if err := validateJSONObject(data, maxResponseBytes); err != nil {
		return Response{}, fmt.Errorf("invalid %s engine response: %w", spec.label, err)
	}
	if err := requireExactFields(data, []string{
		"protocol", "request_id", "correlation_id", "operation", "engine_id",
		"engine_version", "outcome", "result", "error", "response_digest",
	}); err != nil {
		return Response{}, fmt.Errorf("invalid %s engine response fields: %w", spec.label, err)
	}
	var response Response
	if err := decodeExact(data, &response); err != nil {
		return Response{}, fmt.Errorf("decode %s engine response: %w", spec.label, err)
	}
	if response.Protocol != processProtocol || response.RequestID != requestID ||
		response.CorrelationID != requestID || response.Operation != operation ||
		response.EngineID != spec.engineID || response.EngineVersion != version {
		return Response{}, fmt.Errorf("%s engine response identity mismatch", spec.label)
	}
	if !taggedDigest(response.ResponseDigest) {
		return Response{}, fmt.Errorf("%s engine response digest has invalid syntax", spec.label)
	}
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&object); err != nil {
		return Response{}, fmt.Errorf("decode %s response digest input: %w", spec.label, err)
	}
	delete(object, "response_digest")
	canonical, err := marshalCanonical(object)
	if err != nil {
		return Response{}, fmt.Errorf("canonicalize %s response: %w", spec.label, err)
	}
	digest := sha256.Sum256(canonical)
	if response.ResponseDigest != "sha256:"+hex.EncodeToString(digest[:]) {
		return Response{}, fmt.Errorf("%s engine response digest mismatch", spec.label)
	}
	switch response.Outcome {
	case "ok":
		if response.Error != nil || bytes.Equal(response.Result, []byte("null")) || len(response.Result) == 0 {
			return Response{}, fmt.Errorf("%s success response has invalid result/error state", spec.label)
		}
		if err := validateJSONObject(response.Result, maxResponseBytes); err != nil {
			return Response{}, fmt.Errorf("%s success result is invalid: %w", spec.label, err)
		}
	case "error":
		if response.Error == nil || !bytes.Equal(response.Result, []byte("null")) ||
			!safeToken(response.Error.Code, 128) || len(response.Error.Message) == 0 || len(response.Error.Message) > 512 {
			return Response{}, fmt.Errorf("%s error response has invalid result/error state", spec.label)
		}
	default:
		return Response{}, fmt.Errorf("%s engine response has unknown outcome", spec.label)
	}
	return response, nil
}

func marshalCanonical(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buffer.Bytes(), []byte("\n")), nil
}

func decodeExact(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := ensureEOF(decoder); err != nil {
		return err
	}
	return nil
}

func requireExactFields(data []byte, fields []string) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		return err
	}
	if len(object) != len(fields) {
		return fmt.Errorf("field count is %d, want %d", len(object), len(fields))
	}
	for _, field := range fields {
		if _, ok := object[field]; !ok {
			return fmt.Errorf("required field %q is missing", field)
		}
	}
	return nil
}

func validateJSONObject(data []byte, maxBytes int64) error {
	if len(data) == 0 || int64(len(data)) > maxBytes {
		return fmt.Errorf("JSON byte bound violated")
	}
	if !utf8.Valid(data) {
		return fmt.Errorf("JSON is not valid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	count := 0
	rootObject, err := validateJSONValue(decoder, 0, &count)
	if err != nil {
		return err
	}
	if !rootObject {
		return fmt.Errorf("JSON root must be an object")
	}
	return ensureEOF(decoder)
}

func validateJSONValue(decoder *json.Decoder, depth int, count *int) (bool, error) {
	if depth > maxJSONDepth {
		return false, fmt.Errorf("JSON depth exceeds %d", maxJSONDepth)
	}
	token, err := decoder.Token()
	if err != nil {
		return false, err
	}
	(*count)++
	if *count > maxJSONValues {
		return false, fmt.Errorf("JSON value count exceeds %d", maxJSONValues)
	}
	switch value := token.(type) {
	case json.Delim:
		switch value {
		case '{':
			seen := make(map[string]struct{})
			for decoder.More() {
				keyToken, keyErr := decoder.Token()
				if keyErr != nil {
					return false, keyErr
				}
				key, ok := keyToken.(string)
				if !ok || len(key) > maxStringBytes {
					return false, fmt.Errorf("JSON object key is invalid or too large")
				}
				if _, duplicate := seen[key]; duplicate {
					return false, fmt.Errorf("JSON object contains duplicate key %q", key)
				}
				seen[key] = struct{}{}
				(*count)++
				if _, valueErr := validateJSONValue(decoder, depth+1, count); valueErr != nil {
					return false, valueErr
				}
			}
			end, endErr := decoder.Token()
			if endErr != nil || end != json.Delim('}') {
				return false, fmt.Errorf("JSON object is not closed")
			}
			return true, nil
		case '[':
			for decoder.More() {
				if _, valueErr := validateJSONValue(decoder, depth+1, count); valueErr != nil {
					return false, valueErr
				}
			}
			end, endErr := decoder.Token()
			if endErr != nil || end != json.Delim(']') {
				return false, fmt.Errorf("JSON array is not closed")
			}
			return false, nil
		default:
			return false, fmt.Errorf("unexpected JSON delimiter")
		}
	case string:
		if len(value) > maxStringBytes {
			return false, fmt.Errorf("JSON string exceeds %d bytes", maxStringBytes)
		}
	case json.Number:
		text := value.String()
		if strings.ContainsAny(text, ".eE") {
			return false, fmt.Errorf("floating-point JSON values are prohibited")
		}
		integer, numberErr := strconv.ParseInt(text, 10, 64)
		if numberErr != nil || integer < -9007199254740991 || integer > 9007199254740991 {
			return false, fmt.Errorf("JSON integer exceeds interoperable range")
		}
	case bool, nil:
	default:
		return false, fmt.Errorf("unsupported JSON token")
	}
	return false, nil
}

func ensureEOF(decoder *json.Decoder) error {
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("JSON contains trailing data")
		}
		return err
	}
	return nil
}

func safeRelativePath(value string) bool {
	if value == "" || len(value) > 4096 || strings.HasPrefix(value, "/") ||
		strings.Contains(value, "\\") || strings.Contains(value, "//") || strings.ContainsRune(value, '\x00') {
		return false
	}
	for _, component := range strings.Split(value, "/") {
		if component == "" || component == "." || component == ".." {
			return false
		}
		for _, character := range []byte(component) {
			if character < 0x20 || character == 0x7f {
				return false
			}
		}
	}
	return true
}

func safeToken(value string, limit int) bool {
	if value == "" || len(value) > limit {
		return false
	}
	for _, character := range []byte(value) {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') || character == '.' || character == '_' ||
			character == ':' || character == '-' {
			continue
		}
		return false
	}
	return true
}

func safeVersion(value string) bool {
	if value == "" || len(value) > 64 || value == "." || value == ".." {
		return false
	}
	for _, character := range []byte(value) {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') || character == '.' || character == '+' ||
			character == '-' {
			continue
		}
		return false
	}
	return true
}

func taggedDigest(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for _, character := range value[7:] {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func randomToken() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return "qxctl:" + hex.EncodeToString(value[:]), nil
}

type boundedBuffer struct {
	buffer   bytes.Buffer
	limit    int64
	exceeded bool
}

func (b *boundedBuffer) Write(data []byte) (int, error) {
	if b.exceeded {
		return len(data), nil
	}
	remaining := b.limit - int64(b.buffer.Len())
	if remaining <= 0 {
		b.exceeded = true
		return len(data), nil
	}
	if int64(len(data)) > remaining {
		_, _ = b.buffer.Write(data[:remaining])
		b.exceeded = true
		return len(data), nil
	}
	return b.buffer.Write(data)
}

func (b *boundedBuffer) Bytes() []byte { return b.buffer.Bytes() }
