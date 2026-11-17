package supervision

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/packageinstall"
	ssiagpaths "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
)

const testTOPSID = "018f0c3a-7b2d-7e11-8c12-0242ac120002"

func TestUserDescriptorIsBoundedAndIndependent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	source, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := packageinstall.Install(source, filepath.Join(home, "prefix"), "dev")
	if err != nil {
		t.Fatal(err)
	}
	spec := Spec{Scope: ssiagpaths.ScopeUser, TOPSID: testTOPSID, Binary: pkg.Binary, UID: uint32(os.Geteuid()), GID: uint32(os.Getegid())}
	record, content, err := render(spec)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, required := range []string{spec.Binary, "--supervised", testTOPSID, "on-failure"} {
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

func TestRetirementFencesSupervisorInstallAndStart(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	source, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := packageinstall.Install(source, filepath.Join(home, "prefix"), "dev")
	if err != nil {
		t.Fatal(err)
	}
	spec := Spec{Scope: ssiagpaths.ScopeUser, TOPSID: testTOPSID, Binary: pkg.Binary, UID: uint32(os.Geteuid()), GID: uint32(os.Getegid())}
	record, err := Install(spec, false)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(record.Descriptor)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := packageinstall.Uninstall(pkg.Prefix, "dev"); err == nil {
		t.Fatal("descriptor retention did not block package reclamation")
	}
	if _, err := Install(spec, true); err == nil {
		t.Fatal("retirement admitted supervisor descriptor mutation")
	}
	if err := Start(record); err == nil || !strings.Contains(err.Error(), "admission is closed") {
		t.Fatalf("retirement reached native start: %v", err)
	}
	after, err := os.ReadFile(record.Descriptor)
	if err != nil || string(before) != string(after) {
		t.Fatal("blocked start changed descriptor")
	}
	if _, err := Uninstall(spec, false, false); err != nil {
		t.Fatalf("retirement blocked native descriptor cleanup: %v", err)
	}
}
