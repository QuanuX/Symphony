package supervision

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	ssiagpaths "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
)

const testTOPSID = "018f0c3a-7b2d-7e11-8c12-0242ac120002"

func TestUserDescriptorIsBoundedAndIndependent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	spec := Spec{Scope: ssiagpaths.ScopeUser, TOPSID: testTOPSID, Binary: filepath.Join(home, ".local", "bin", "symphony-ssiag"), UID: uint32(os.Geteuid()), GID: uint32(os.Getegid())}
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
	if strings.Contains(strings.ToLower(text), "stav") {
		t.Fatalf("SSIAG supervisor is coupled to STAV:\n%s", text)
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

func TestSystemdDescriptorUsesConfiguredIdentityAndNoSTAVCoupling(t *testing.T) {
	content, err := renderSystemd(renderData{TOPSID: testTOPSID, Binary: "/usr/local/bin/symphony-ssiag", Scope: "system", UID: 412, GID: 413, System: true})
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, required := range []string{"User=412", "Group=413", "StartLimitBurst=5", "TimeoutStopSec=10s", "ExecStart=\"/usr/local/bin/symphony-ssiag\"", "/run/symphony/" + testTOPSID + "/ssiag"} {
		if !strings.Contains(text, required) {
			t.Fatalf("systemd descriptor omits %q:\n%s", required, text)
		}
	}
	if strings.Contains(strings.ToLower(text), "stav") || strings.Contains(text, "Requires=") || strings.Contains(text, "Wants=") {
		t.Fatalf("systemd SSIAG descriptor gained bootstrap coupling:\n%s", text)
	}
}
