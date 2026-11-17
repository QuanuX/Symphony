package packageinstall

import (
	"path/filepath"
	"testing"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/packageretirement"
)

func TestInstallPinsOtherSourceWithoutSelfLeaseConflict(t *testing.T) {
	source, prefix := setupPackageTest(t)
	installed, err := Install(source, prefix, compiledVersion())
	if err != nil {
		t.Fatal(err)
	}
	self, err := acquireInstallSourceUse(installed.Binary, installed.Binary)
	if err != nil || self != nil {
		t.Fatalf("same-package install acquired conflicting source custody: %v", err)
	}
	use, err := acquireInstallSourceUse(installed.Binary, filepath.Join(filepath.Dir(prefix), "another-binary"))
	if err != nil || use == nil {
		t.Fatalf("different source package was not pinned: %v", err)
	}
	defer use.Close()
	id, _, err := packageretirement.Inspect(installed.Prefix, ComponentID, compiledVersion())
	if err != nil {
		t.Fatal(err)
	}
	if maintenance, err := packageretirement.PrepareInstall(installed.Prefix, id, false); err == nil {
		maintenance.Close()
		t.Fatal("source package accepted exclusive maintenance during another installation")
	}
	if err := use.Close(); err != nil {
		t.Fatal(err)
	}
	maintenance, err := packageretirement.PrepareInstall(installed.Prefix, id, false)
	if err != nil {
		t.Fatalf("completed source use retained its lease: %v", err)
	}
	maintenance.Close()
}
