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

	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/config"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/paths"
)

const launchdPrefix = "io.github.quanux.symphony.accordare-stav-producer."
const systemdPrefix = "symphony-accordare-stav-producer@"

type Spec struct {
	Scope          paths.Scope
	TOPSID, Binary string
	UID, GID       uint32
}
type Record struct {
	Manager                                  string
	Scope                                    paths.Scope
	TOPSID, Name, Descriptor, Digest, Domain string
	Changed                                  bool
}

func SpecFromConfig(scope paths.Scope, topsID, binary string, cfg config.Config) (Spec, error) {
	if cfg.TOPSID != topsID || cfg.Mode != string(scope) || !filepath.IsAbs(binary) {
		return Spec{}, fmt.Errorf("supervisor evidence does not match enrollment")
	}
	if cfg.Identity.UID > uint64(^uint32(0)) || cfg.Identity.GID > uint64(^uint32(0)) {
		return Spec{}, fmt.Errorf("supervisor identity exceeds platform range")
	}
	return Spec{Scope: scope, TOPSID: topsID, Binary: filepath.Clean(binary), UID: uint32(cfg.Identity.UID), GID: uint32(cfg.Identity.GID)}, nil
}

func Install(spec Spec, force, start bool) (Record, error) {
	record, content, err := render(spec)
	if err != nil {
		return Record{}, err
	}
	if err := ensureDirectory(filepath.Dir(record.Descriptor)); err != nil {
		return Record{}, err
	}
	if info, statErr := os.Lstat(record.Descriptor); statErr == nil {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return Record{}, fmt.Errorf("supervisor descriptor is unsafe")
		}
		existing, err := os.ReadFile(record.Descriptor)
		if err != nil {
			return Record{}, err
		}
		if bytes.Equal(existing, content) {
			if start {
				err = Start(record)
			}
			return record, err
		}
		if !force {
			return Record{}, fmt.Errorf("supervisor descriptor differs; use --force to replace it")
		}
	} else if !os.IsNotExist(statErr) {
		return Record{}, statErr
	}
	if err := writeAtomic(record.Descriptor, content); err != nil {
		return Record{}, err
	}
	record.Changed = true
	if start {
		err = Start(record)
	}
	return record, err
}

func Uninstall(spec Spec, force, stop bool) (Record, error) {
	record, expected, err := render(spec)
	if err != nil {
		return Record{}, err
	}
	info, statErr := os.Lstat(record.Descriptor)
	if os.IsNotExist(statErr) {
		return record, nil
	}
	if statErr != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return Record{}, fmt.Errorf("supervisor descriptor is unsafe")
	}
	existing, err := os.ReadFile(record.Descriptor)
	if err != nil {
		return Record{}, err
	}
	if !bytes.Equal(existing, expected) && !force {
		return Record{}, fmt.Errorf("supervisor descriptor differs; use --force to remove it")
	}
	if stop {
		if err := Stop(record); err != nil {
			return Record{}, err
		}
	}
	if err := os.Remove(record.Descriptor); err != nil {
		return Record{}, err
	}
	record.Changed = true
	if record.Manager == "systemd" {
		_ = run("systemctl", systemctlArgs(record.Scope, "daemon-reload")...)
	}
	return record, syncDirectory(filepath.Dir(record.Descriptor))
}

func Start(record Record) error {
	if record.Manager == "launchd" {
		target := record.Domain + "/" + record.Name
		if exec.Command("launchctl", "print", target).Run() == nil {
			return run("launchctl", "kickstart", "-k", target)
		}
		return run("launchctl", "bootstrap", record.Domain, record.Descriptor)
	}
	if err := run("systemctl", systemctlArgs(record.Scope, "daemon-reload")...); err != nil {
		return err
	}
	return run("systemctl", systemctlArgs(record.Scope, "enable", "--now", record.Name)...)
}

func Stop(record Record) error {
	if record.Manager == "launchd" {
		target := record.Domain + "/" + record.Name
		if exec.Command("launchctl", "print", target).Run() != nil {
			return nil
		}
		return run("launchctl", "bootout", target)
	}
	return run("systemctl", systemctlArgs(record.Scope, "disable", "--now", record.Name)...)
}

func Render(spec Spec) (Record, []byte, error) { return render(spec) }

func render(spec Spec) (Record, []byte, error) {
	if spec.Scope == paths.ScopeUser && (spec.UID != uint32(os.Geteuid()) || spec.GID != uint32(os.Getegid())) {
		return Record{}, nil, fmt.Errorf("user supervisor identity differs from effective identity")
	}
	if spec.Scope == paths.ScopeSystem && os.Geteuid() != 0 {
		return Record{}, nil, fmt.Errorf("system supervisor mutation requires administrator privileges")
	}
	data := struct {
		Label, Unit, TOPSID, Binary, Scope, UserName, GroupName string
		UID, GID                                                uint32
		System                                                  bool
	}{TOPSID: spec.TOPSID, Binary: spec.Binary, Scope: string(spec.Scope), UID: spec.UID, GID: spec.GID, System: spec.Scope == paths.ScopeSystem}
	if data.System {
		account, err := user.LookupId(strconv.FormatUint(uint64(spec.UID), 10))
		if err != nil {
			return Record{}, nil, fmt.Errorf("configured UID is not provisioned: %w", err)
		}
		group, err := user.LookupGroupId(strconv.FormatUint(uint64(spec.GID), 10))
		if err != nil {
			return Record{}, nil, fmt.Errorf("configured GID is not provisioned: %w", err)
		}
		data.UserName, data.GroupName = account.Username, group.Name
	}
	var record Record
	var source string
	switch runtime.GOOS {
	case "darwin":
		data.Label = launchdPrefix + spec.TOPSID
		home, err := os.UserHomeDir()
		if err != nil {
			return Record{}, nil, err
		}
		dir, domain := filepath.Join(home, "Library", "LaunchAgents"), "gui/"+strconv.Itoa(os.Geteuid())
		if data.System {
			dir, domain = "/Library/LaunchDaemons", "system"
		}
		record = Record{Manager: "launchd", Scope: spec.Scope, TOPSID: spec.TOPSID, Name: data.Label, Descriptor: filepath.Join(dir, data.Label+".plist"), Domain: domain}
		source = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict><key>Label</key><string>{{xml .Label}}</string><key>ProgramArguments</key><array><string>{{xml .Binary}}</string><string>serve</string><string>--supervised</string><string>--scope</string><string>{{xml .Scope}}</string><string>--tops-id</string><string>{{xml .TOPSID}}</string></array><key>RunAtLoad</key><true/><key>KeepAlive</key><dict><key>SuccessfulExit</key><false/></dict><key>ThrottleInterval</key><integer>10</integer><key>ProcessType</key><string>Background</string><key>Umask</key><integer>63</integer>{{if .System}}<key>UserName</key><string>{{xml .UserName}}</string><key>GroupName</key><string>{{xml .GroupName}}</string>{{end}}</dict></plist>
`
	case "linux":
		data.Unit = systemdPrefix + spec.TOPSID + ".service"
		dir := "/etc/systemd/system"
		if spec.Scope == paths.ScopeUser {
			home, err := os.UserHomeDir()
			if err != nil {
				return Record{}, nil, err
			}
			base := os.Getenv("XDG_CONFIG_HOME")
			if base == "" {
				base = filepath.Join(home, ".config")
			}
			dir = filepath.Join(base, "systemd", "user")
		}
		record = Record{Manager: "systemd", Scope: spec.Scope, TOPSID: spec.TOPSID, Name: data.Unit, Descriptor: filepath.Join(dir, data.Unit)}
		source = `[Unit]
Description=Symphony Accordare STAV producer for TOPS {{.TOPSID}}
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

[Install]
WantedBy={{if .System}}multi-user.target{{else}}default.target{{end}}
`
	default:
		return Record{}, nil, fmt.Errorf("native Accordare supervision is unsupported on %s", runtime.GOOS)
	}
	tmpl, err := texttemplate.New("descriptor").Funcs(texttemplate.FuncMap{"xml": func(value string) string {
		var output bytes.Buffer
		_ = xml.EscapeText(&output, []byte(value))
		return output.String()
	}}).Parse(source)
	if err != nil {
		return Record{}, nil, err
	}
	var output bytes.Buffer
	if err := tmpl.Execute(&output, data); err != nil {
		return Record{}, nil, err
	}
	sum := sha256.Sum256(output.Bytes())
	record.Digest = "sha256:" + hex.EncodeToString(sum[:])
	return record, output.Bytes(), nil
}

func systemctlArgs(scope paths.Scope, args ...string) []string {
	if scope == paths.ScopeUser {
		return append([]string{"--user"}, args...)
	}
	return args
}
func run(name string, args ...string) error {
	output, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", name, strings.TrimSpace(string(output)))
	}
	return nil
}
func ensureDirectory(path string) error {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		return fmt.Errorf("unsafe supervisor directory")
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
			return fmt.Errorf("unsafe supervisor directory component")
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	return os.Mkdir(path, 0o755)
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
func writeAtomic(path string, content []byte) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".accordare-supervisor-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err := temporary.Chmod(0o644); err != nil {
		return err
	}
	if _, err := temporary.Write(content); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
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
