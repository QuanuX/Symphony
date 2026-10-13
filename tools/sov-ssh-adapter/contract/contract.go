// Package contract defines the optional one-shot SSH carrier's public wire types.
// It contains no credential loading, network code, or scientific interpretation.
package contract

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

const Version = "0.1.0-dev"
const AdapterID = "symphony.sov.ssh-adapter"
const EndpointProtocol = "symphony.sov.ssh-endpoint.v1"
const InputProtocol = "symphony.sov.ssh-invoke-input.v1"
const ReplyProtocol = "symphony.sov.ssh-carrier-result.v1"
const MaxDocumentBytes int64 = 8 << 20

type Executable struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type Endpoint struct {
	Protocol              string     `json:"protocol"`
	SSH                   Executable `json:"ssh"`
	SSHExecution          string     `json:"ssh_execution"`
	Host                  string     `json:"host"`
	Port                  string     `json:"port"`
	User                  string     `json:"user"`
	HostPublicKey         string     `json:"host_public_key"`
	ClientPublicKey       string     `json:"client_public_key"`
	AgentSocket           string     `json:"agent_socket"`
	ConnectTimeoutSeconds *string    `json:"connect_timeout_seconds"`
	TargetLauncher        Executable `json:"target_launcher"`
}
type Input struct {
	Protocol   string          `json:"protocol"`
	Endpoint   Endpoint        `json:"endpoint"`
	Invocation json.RawMessage `json:"invocation"`
}
type Reply struct {
	Protocol         string          `json:"protocol"`
	EndpointSHA256   string          `json:"endpoint_sha256"`
	InvocationSHA256 string          `json:"invocation_sha256"`
	HostKeySHA256    string          `json:"host_key_sha256"`
	ClientKeySHA256  string          `json:"client_key_sha256"`
	TargetResult     json.RawMessage `json:"target_result"`
}

var digestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
var hostPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
var userPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]*\$?$`)
var positive = regexp.MustCompile(`^[1-9][0-9]*$`)

func Digest(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func PublicKeyDigest(s string) (string, error) {
	parts := strings.Split(s, " ")
	if len(parts) != 2 {
		return "", fmt.Errorf("expected one public key without options or comment")
	}
	switch parts[0] {
	case "ssh-ed25519", "ssh-rsa", "ecdsa-sha2-nistp256", "ecdsa-sha2-nistp384", "ecdsa-sha2-nistp521":
	default:
		return "", fmt.Errorf("unsupported raw public key type")
	}
	b, err := base64.StdEncoding.Strict().DecodeString(parts[1])
	if err != nil || len(b) < 5 || base64.StdEncoding.EncodeToString(b) != parts[1] {
		return "", fmt.Errorf("invalid public key encoding")
	}
	n := uint64(binary.BigEndian.Uint32(b[:4]))
	if n >= uint64(len(b)-4) || string(b[4:4+n]) != parts[0] {
		return "", fmt.Errorf("public key type mismatch")
	}
	return Digest(b), nil
}
func absolute(s string) bool {
	return path.IsAbs(s) && path.Clean(s) == s && s != "/" && !strings.ContainsRune(s, 0)
}
func ValidExecutable(e Executable) bool {
	return absolute(e.Path) && digestPattern.MatchString(e.SHA256)
}
func Validate(e Endpoint) error {
	if e.SSHExecution != "stage_copy" && e.SSHExecution != "protected_system_path" {
		return fmt.Errorf("invalid SSH executable execution selection")
	}
	if e.Protocol != EndpointProtocol || !ValidExecutable(e.SSH) || !ValidExecutable(e.TargetLauncher) || !absolute(e.AgentSocket) || strings.ContainsAny(e.AgentSocket, "\r\n%$") || !userPattern.MatchString(e.User) {
		return fmt.Errorf("invalid SSH endpoint identity")
	}
	if !hostPattern.MatchString(e.Host) && net.ParseIP(e.Host) == nil {
		return fmt.Errorf("invalid SSH host")
	}
	port, err := strconv.ParseUint(e.Port, 10, 16)
	if err != nil || port == 0 || !positive.MatchString(e.Port) {
		return fmt.Errorf("invalid SSH port")
	}
	if _, err = PublicKeyDigest(e.HostPublicKey); err != nil {
		return err
	}
	if _, err = PublicKeyDigest(e.ClientPublicKey); err != nil {
		return err
	}
	if e.ConnectTimeoutSeconds != nil {
		v, err := strconv.ParseUint(*e.ConnectTimeoutSeconds, 10, 31)
		if err != nil || v == 0 || !positive.MatchString(*e.ConnectTimeoutSeconds) {
			return fmt.Errorf("connection timeout is not representable by OpenSSH")
		}
	}
	return nil
}
func Canonical(raw []byte) ([]byte, error) {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var v any
	if err := d.Decode(&v); err != nil {
		return nil, err
	}
	return json.Marshal(v)
}
func EndpointDigest(e Endpoint) string {
	b, _ := json.Marshal(e)
	b, _ = Canonical(b)
	return Digest(append([]byte("symphony.sov.ssh-endpoint.v1\x00"), b...))
}
func EndpointRef(e Endpoint) string { return "sov:ssh:sha256:" + EndpointDigest(e) }

// CheckJSON rejects duplicate keys, excessive nesting and trailing documents.
// The bound is for control frames, never aggregate dataset/result bytes.
func CheckJSON(raw []byte) error {
	if !utf8.Valid(raw) {
		return fmt.Errorf("control input is not UTF-8")
	}
	if int64(len(raw)) > MaxDocumentBytes {
		return fmt.Errorf("control frame too large")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var value func(int) error
	value = func(depth int) error {
		if depth > 64 {
			return fmt.Errorf("control nesting too deep")
		}
		t, err := d.Token()
		if err != nil {
			return err
		}
		if delim, ok := t.(json.Delim); ok {
			switch delim {
			case '{':
				seen := map[string]bool{}
				for d.More() {
					k, err := d.Token()
					if err != nil {
						return err
					}
					key, ok := k.(string)
					if !ok || seen[key] {
						return fmt.Errorf("duplicate control member")
					}
					seen[key] = true
					if err = value(depth + 1); err != nil {
						return err
					}
				}
			case '[':
				for d.More() {
					if err = value(depth + 1); err != nil {
						return err
					}
				}
			default:
				return fmt.Errorf("unexpected delimiter")
			}
			_, err = d.Token()
			return err
		}
		return nil
	}
	if err := value(0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return fmt.Errorf("trailing control value")
	}
	return nil
}
func DecodeInput(raw []byte) (Input, error) {
	var in Input
	if err := CheckJSON(raw); err != nil {
		return in, err
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&in); err != nil {
		return in, err
	}
	encoded, _ := json.Marshal(in)
	var original, typed any
	_ = json.Unmarshal(raw, &original)
	_ = json.Unmarshal(encoded, &typed)
	if !sameShape(original, typed) {
		return in, fmt.Errorf("missing or aliased carrier member")
	}
	if in.Protocol != InputProtocol {
		return in, fmt.Errorf("wrong SSH invocation protocol")
	}
	return in, Validate(in.Endpoint)
}

func sameShape(a, b any) bool {
	m, ok := b.(map[string]any)
	if !ok {
		return reflect.TypeOf(a) == reflect.TypeOf(b)
	}
	n, ok := a.(map[string]any)
	if !ok || len(n) != len(m) {
		return false
	}
	for k, v := range m {
		w, ok := n[k]
		if !ok || !sameShape(w, v) {
			return false
		}
	}
	return true
}

// TargetCommand reads only transport identity. Native request admission remains
// entirely with qxctl and the selected C++ engine on the target.
func TargetCommand(in Input) (string, string, error) {
	var inv struct {
		Protocol  string `json:"protocol"`
		Selection struct {
			Transport struct {
				Arrangement    string `json:"arrangement"`
				Family         string `json:"family"`
				AdapterID      string `json:"adapter_id"`
				AdapterVersion string `json:"adapter_version"`
				EndpointRef    string `json:"endpoint_ref"`
			} `json:"transport"`
			Runtime struct {
				SHA256 string `json:"qxctl_sha256"`
			} `json:"runtime"`
		} `json:"selection"`
		Endpoint struct {
			Target        Executable `json:"target_qxctl"`
			Configuration string     `json:"configuration_path"`
			SHA256        string     `json:"configuration_sha256"`
		} `json:"endpoint"`
	}
	if err := json.Unmarshal(in.Invocation, &inv); err != nil {
		return "", "", err
	}
	t := inv.Selection.Transport
	e := inv.Endpoint
	if inv.Protocol != "symphony.sov.remote-invoke-input.v1" || t.Arrangement != "controller" || t.Family != "scoped_cli" || t.AdapterID != AdapterID || t.AdapterVersion != Version || t.EndpointRef != EndpointRef(in.Endpoint) || !ValidExecutable(e.Target) || e.Target.SHA256 != inv.Selection.Runtime.SHA256 || !absolute(e.Configuration) || !digestPattern.MatchString(e.SHA256) {
		return "", "", fmt.Errorf("SSH invocation binding mismatch")
	}
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }
	args := []string{in.Endpoint.TargetLauncher.Path, "--self-sha256", in.Endpoint.TargetLauncher.SHA256, "--target", e.Target.Path, "--target-sha256", e.Target.SHA256, "--configuration", e.Configuration, "--configuration-sha256", e.SHA256}
	for i := range args {
		args[i] = quote(args[i])
	}
	b, err := Canonical(in.Invocation)
	if err != nil {
		return "", "", err
	}
	return "exec " + strings.Join(args, " "), Digest(append([]byte("symphony.sov.invocation.v1\x00"), b...)), nil
}
