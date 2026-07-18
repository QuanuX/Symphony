package supervision

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	stavpaths "github.com/QuanuX/Symphony/modules/stav-append-authority/internal/paths"
)

const testTOPSID = "123e4567-e89b-42d3-a456-426614174000"

func TestUserDescriptorIsBoundedAndIndependent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	spec := Spec{Scope: stavpaths.ScopeUser, TOPSID: testTOPSID, Binary: filepath.Join(home, ".local", "bin", stavpaths.BinaryName), UID: uint32(os.Geteuid()), GID: uint32(os.Getegid())}
	record, content, err := render(spec)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, required := range []string{"--supervised", testTOPSID, "on-failure"} {
		if !strings.Contains(text, required) && !(runtime.GOOS == "darwin" && required == "on-failure") {
			t.Fatalf("descriptor omits %q:\n%s", required, text)
		}
	}
	if strings.Contains(strings.ToLower(text), "ssiag") {
		t.Fatalf("STAV supervisor is coupled to SSIAG:\n%s", text)
	}
	if !strings.Contains(record.Name, testTOPSID) || record.DescriptorHash == "" {
		t.Fatalf("descriptor identity is not TOPS-bound: %+v", record)
	}
	installed, err := Install(spec, false)
	if err != nil {
		t.Fatal(err)
	}
	if !installed.Changed {
		t.Fatal("first descriptor install reported no change")
	}
	idempotent, err := Install(spec, false)
	if err != nil {
		t.Fatal(err)
	}
	if idempotent.Changed {
		t.Fatal("identical descriptor install was not idempotent")
	}
}

func TestSystemdDescriptorUsesConfiguredIdentityAndNoSSIAGCoupling(t *testing.T) {
	content, err := renderSystemd(renderData{TOPSID: testTOPSID, Binary: "/usr/local/bin/symphony-stav-append-authority", Scope: "system", UID: 414, GID: 415, System: true})
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, required := range []string{"User=414", "Group=415", "StartLimitBurst=5", "TimeoutStopSec=10s", "ExecStart=\"/usr/local/bin/symphony-stav-append-authority\"", "/run/symphony/" + testTOPSID + "/stav"} {
		if !strings.Contains(text, required) {
			t.Fatalf("systemd descriptor omits %q:\n%s", required, text)
		}
	}
	if strings.Contains(strings.ToLower(text), "ssiag") || strings.Contains(text, "Requires=") || strings.Contains(text, "Wants=") {
		t.Fatalf("systemd STAV descriptor gained bootstrap coupling:\n%s", text)
	}
}
