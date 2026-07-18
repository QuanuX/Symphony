package supervision

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	texttemplate "text/template"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/config"
	ssiagpaths "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
)

const (
	launchdPrefix = "io.github.quanux.symphony.ssiag."
	systemdPrefix = "symphony-ssiag@"
)

type Spec struct {
	Scope  ssiagpaths.Scope
	TOPSID string
	Binary string
	UID    uint32
	GID    uint32
}

type Record struct {
	Manager        string
	Scope          ssiagpaths.Scope
	TOPSID         string
	Name           string
	Descriptor     string
	DescriptorHash string
	Changed        bool
	Domain         string
}

type renderData struct {
	Label     string
	Unit      string
	TOPSID    string
	Binary    string
	Scope     string
	UID       uint32
	GID       uint32
	UserName  string
	GroupName string
	System    bool
}

func Install(spec Spec, force bool) (Record, error) {
	record, content, err := render(spec)
	if err != nil {
		return Record{}, err
	}
	if err := ensureDirectory(filepath.Dir(record.Descriptor)); err != nil {
		return Record{}, err
	}
	info, err := os.Lstat(record.Descriptor)
	if err == nil {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return Record{}, fmt.Errorf("refusing unsafe supervisor descriptor %s", record.Descriptor)
		}
		existing, readErr := os.ReadFile(record.Descriptor)
		if readErr != nil {
			return Record{}, fmt.Errorf("read supervisor descriptor: %w", readErr)
		}
		if bytes.Equal(existing, content) {
			return record, nil
		}
		if !force {
			return Record{}, fmt.Errorf("supervisor descriptor differs; use --force to replace it")
		}
	} else if !os.IsNotExist(err) {
		return Record{}, fmt.Errorf("inspect supervisor descriptor: %w", err)
	}
	if err := writeAtomic(record.Descriptor, content, 0o644); err != nil {
		return Record{}, err
	}
	record.Changed = true
	return record, nil
}

func Uninstall(spec Spec, force, stop bool) (Record, error) {
	record, content, err := render(spec)
	if err != nil {
		return Record{}, err
	}
	info, err := os.Lstat(record.Descriptor)
	if os.IsNotExist(err) {
		return record, nil
	}
	if err != nil {
		return Record{}, fmt.Errorf("inspect supervisor descriptor: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return Record{}, fmt.Errorf("refusing unsafe supervisor descriptor %s", record.Descriptor)
	}
	existing, err := os.ReadFile(record.Descriptor)
	if err != nil {
		return Record{}, fmt.Errorf("read supervisor descriptor: %w", err)
	}
	if !bytes.Equal(existing, content) && !force {
		return Record{}, fmt.Errorf("supervisor descriptor differs; use --force to remove it")
	}
	if stop {
		if err := Stop(record); err != nil {
			return Record{}, fmt.Errorf("stop supervised SSIAG service: %w", err)
		}
	}
	if err := os.Remove(record.Descriptor); err != nil {
		return Record{}, fmt.Errorf("remove supervisor descriptor: %w", err)
	}
	if err := syncDirectory(filepath.Dir(record.Descriptor)); err != nil {
		return Record{}, err
	}
	record.Changed = true
	if stop && record.Manager == "systemd" {
		if err := run("systemctl", systemctlArgs(record.Scope, "daemon-reload")...); err != nil {
			return Record{}, err
		}
	}
	return record, nil
}

func Start(record Record) error {
	switch record.Manager {
	case "launchd":
		target := record.Domain + "/" + record.Name
		if commandOK("launchctl", "print", target) {
			return run("launchctl", "kickstart", "-k", target)
		}
		return run("launchctl", "bootstrap", record.Domain, record.Descriptor)
	case "systemd":
		args := systemctlArgs(record.Scope, "daemon-reload")
		if err := run("systemctl", args...); err != nil {
			return err
		}
		args = systemctlArgs(record.Scope, "enable", "--now", record.Name)
		return run("systemctl", args...)
	default:
		return fmt.Errorf("unsupported supervisor manager %q", record.Manager)
	}
}

func Stop(record Record) error {
	switch record.Manager {
	case "launchd":
		target := record.Domain + "/" + record.Name
		if !commandOK("launchctl", "print", target) {
			return nil
		}
		return run("launchctl", "bootout", target)
	case "systemd":
		args := systemctlArgs(record.Scope, "disable", "--now", record.Name)
		return run("systemctl", args...)
	default:
		return fmt.Errorf("unsupported supervisor manager %q", record.Manager)
	}
}

func render(spec Spec) (Record, []byte, error) {
	if err := ssiagpaths.ValidateTOPSID(spec.TOPSID); err != nil {
		return Record{}, nil, err
	}
	if !filepath.IsAbs(spec.Binary) {
		return Record{}, nil, fmt.Errorf("supervised binary path must be absolute")
	}
	if spec.Scope == ssiagpaths.ScopeUser && (spec.UID != uint32(os.Geteuid()) || spec.GID != uint32(os.Getegid())) {
		return Record{}, nil, fmt.Errorf("user supervisor identity does not match the invoking effective UID/GID")
	}
	if spec.Scope == ssiagpaths.ScopeSystem && os.Geteuid() != 0 {
		return Record{}, nil, fmt.Errorf("system supervisor installation requires administrator privileges")
	}

	data := renderData{TOPSID: spec.TOPSID, Binary: spec.Binary, Scope: string(spec.Scope), UID: spec.UID, GID: spec.GID, System: spec.Scope == ssiagpaths.ScopeSystem}
	var err error
	if data.System {
		data.UserName, data.GroupName, err = resolveNames(spec.UID, spec.GID)
		if err != nil {
			return Record{}, nil, err
		}
	}
	var record Record
	var content []byte
	switch runtime.GOOS {
	case "darwin":
		data.Label = launchdPrefix + spec.TOPSID
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return Record{}, nil, homeErr
		}
		directory := filepath.Join(home, "Library", "LaunchAgents")
		domain := "gui/" + strconv.Itoa(os.Geteuid())
		if data.System {
			directory = "/Library/LaunchDaemons"
			domain = "system"
		}
		record = Record{Manager: "launchd", Scope: spec.Scope, TOPSID: spec.TOPSID, Name: data.Label, Descriptor: filepath.Join(directory, data.Label+".plist"), Domain: domain}
		content, err = renderLaunchd(data)
	case "linux":
		data.Unit = systemdPrefix + spec.TOPSID + ".service"
		directory := "/etc/systemd/system"
		if spec.Scope == ssiagpaths.ScopeUser {
			data.Binary = "%h/.local/bin/symphony-ssiag"
			home, homeErr := os.UserHomeDir()
			if homeErr != nil {
				return Record{}, nil, homeErr
			}
			configHome := os.Getenv("XDG_CONFIG_HOME")
			if configHome == "" {
				configHome = filepath.Join(home, ".config")
			}
			directory = filepath.Join(configHome, "systemd", "user")
		}
		record = Record{Manager: "systemd", Scope: spec.Scope, TOPSID: spec.TOPSID, Name: data.Unit, Descriptor: filepath.Join(directory, data.Unit)}
		content, err = renderSystemd(data)
	default:
		return Record{}, nil, fmt.Errorf("native SSIAG supervision is unsupported on %s", runtime.GOOS)
	}
	if err != nil {
		return Record{}, nil, err
	}
	digest := sha256.Sum256(content)
	record.DescriptorHash = hex.EncodeToString(digest[:])
	return record, content, nil
}

func renderLaunchd(data renderData) ([]byte, error) {
	const source = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	  <key>Label</key><string>{{xml .Label}}</string>
  <key>ProgramArguments</key>
  <array>
	    <string>{{xml .Binary}}</string><string>serve</string><string>--supervised</string>
	    <string>--scope</string><string>{{xml .Scope}}</string><string>--tops-id</string><string>{{xml .TOPSID}}</string>
  </array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><dict><key>SuccessfulExit</key><false/></dict>
  <key>ThrottleInterval</key><integer>10</integer>
  <key>ProcessType</key><string>Background</string>
  <key>Umask</key><integer>63</integer>
	  {{if .System}}<key>UserName</key><string>{{xml .UserName}}</string>
	  <key>GroupName</key><string>{{xml .GroupName}}</string>{{end}}
</dict>
</plist>
`
	tmpl, err := texttemplate.New("launchd").Funcs(texttemplate.FuncMap{"xml": escapeXML}).Parse(source)
	if err != nil {
		return nil, err
	}
	var output bytes.Buffer
	if err := tmpl.Execute(&output, data); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func escapeXML(value string) string {
	var output bytes.Buffer
	_ = xml.EscapeText(&output, []byte(value))
	return output.String()
}

func renderSystemd(data renderData) ([]byte, error) {
	const source = `[Unit]
Description=Symphony SSIAG for TOPS {{.TOPSID}}
After=local-fs.target
StartLimitIntervalSec=60
StartLimitBurst=5

[Service]
Type=simple
{{if .System}}User={{.UID}}
Group={{.GID}}
{{end}}ExecStart="{{.Binary}}" serve --supervised --scope {{.Scope}} --tops-id {{.TOPSID}}
Restart=on-failure
RestartSec=5s
TimeoutStopSec=10s
KillSignal=SIGTERM
UMask=0077
NoNewPrivileges=true
PrivateTmp=true
{{if .System}}ProtectSystem=strict
ProtectHome=true
ReadOnlyPaths=/etc/symphony/{{.TOPSID}}/ssiag
ReadWritePaths=/var/lib/symphony/{{.TOPSID}}/ssiag /run/symphony/{{.TOPSID}}/ssiag
{{end}}
[Install]
WantedBy={{if .System}}multi-user.target{{else}}default.target{{end}}
`
	tmpl, err := texttemplate.New("systemd").Parse(source)
	if err != nil {
		return nil, err
	}
	var output bytes.Buffer
	if err := tmpl.Execute(&output, data); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func SpecFromConfig(scope ssiagpaths.Scope, topsID string, binary string, cfg config.Config) (Spec, error) {
	if cfg.Authentication == nil || cfg.Authentication.Service == nil || cfg.Authentication.Service.UID == nil || cfg.Authentication.Service.GID == nil {
		return Spec{}, fmt.Errorf("configuration lacks canonical SSIAG service identity")
	}
	if cfg.TOPS.ID != topsID || cfg.Mode != string(scope) {
		return Spec{}, fmt.Errorf("configuration does not match selected TOPS and scope")
	}
	return Spec{Scope: scope, TOPSID: topsID, Binary: binary, UID: *cfg.Authentication.Service.UID, GID: *cfg.Authentication.Service.GID}, nil
}

func resolveNames(uid, gid uint32) (string, string, error) {
	account, err := user.LookupId(strconv.FormatUint(uint64(uid), 10))
	if err != nil {
		return "", "", fmt.Errorf("configured service UID %d is not provisioned: %w", uid, err)
	}
	group, err := user.LookupGroupId(strconv.FormatUint(uint64(gid), 10))
	if err != nil {
		return "", "", fmt.Errorf("configured service GID %d is not provisioned: %w", gid, err)
	}
	return account.Username, group.Name, nil
}

func systemctlArgs(scope ssiagpaths.Scope, args ...string) []string {
	if scope == ssiagpaths.ScopeUser {
		return append([]string{"--user"}, args...)
	}
	return args
}

func run(name string, args ...string) error {
	output, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("%s: %s", name, message)
	}
	return nil
}

func commandOK(name string, args ...string) bool {
	return exec.Command(name, args...).Run() == nil
}

func ensureDirectory(path string) error {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		return fmt.Errorf("unsafe supervisor directory %q", path)
	}
	if path == string(filepath.Separator) {
		return nil
	}
	parent := filepath.Dir(path)
	if parent != path {
		if err := ensureDirectory(parent); err != nil {
			return err
		}
	}
	info, err := os.Lstat(path)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 && permittedSystemAlias(path) {
			return nil
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsafe supervisor directory component %s", path)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	if err := os.Mkdir(path, 0o755); err != nil {
		return fmt.Errorf("create supervisor directory: %w", err)
	}
	return nil
}

func permittedSystemAlias(path string) bool {
	expected := map[string]string{"/etc": "/private/etc", "/tmp": "/private/tmp", "/var": "/private/var"}
	want, ok := expected[path]
	if !ok {
		return false
	}
	resolved, err := filepath.EvalSymlinks(path)
	return err == nil && resolved == want
}

func writeAtomic(path string, content []byte, mode os.FileMode) error {
	temp, err := os.CreateTemp(filepath.Dir(path), ".symphony-ssiag-supervisor-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(mode); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(content); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
