package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOwnerPortableExportExcludesHostGuard(t *testing.T) {
	root := t.TempDir()
	native := filepath.Join(root, "native")
	portable := filepath.Join(root, "portable")
	qerr(os.Mkdir(native, 0700))
	qerr(os.Mkdir(portable, 0700))
	o := qOwner{component: "fixture-owner", binary: "fixture", record: qObject{"version": "0.1.0-dev"}}
	binary := "libexec/symphony/fixture-owner/0.1.0-dev/fixture"
	receipt := "share/symphony/receipts/fixture-owner/0.1.0-dev/install-receipt.json"
	qwrite(qjoin(native, binary), []byte("bounded owner payload"))
	qwrite(qjoin(native, receipt), qObject{"protocol": "symphony.knowledge.install-receipt.v2", "component_id": o.component, "version": "0.1.0-dev", "files": []any{qObject{"path": binary, "kind": "executable", "size": len("bounded owner payload"), "digest": "sha256:" + qdigest(qjoin(native, binary))}}, "provides_capabilities": []any{"symphony.package-retirement.v1"}})
	qwrite(qjoin(native, ".symphony-package-retirement/fixture-owner/0.1.0-dev/state.json"), []byte("nonportable host identity"))
	qExportOwnerPayload(o, native, portable)
	files := qfiles(portable)
	if len(files) != 2 || files[0] != binary || files[1] != receipt {
		t.Fatalf("host custody was exported: %v", files)
	}
}
func TestOwnerRemovalRequiresRetainedTombstone(t *testing.T) {
	prefix := t.TempDir()
	o := qOwner{component: "fixture-owner", record: qObject{"version": "1"}}
	qwrite(qjoin(prefix, "user-sentinel"), []byte("preserve"))
	qwrite(qjoin(prefix, ".symphony-lifecycle.lock"), nilBytes())
	base := ".symphony-package-retirement/fixture-owner/1/"
	for _, name := range []string{"admission.lock", "use.lock"} {
		qwrite(qjoin(prefix, base, name), nilBytes())
	}
	qwrite(qjoin(prefix, base, "state.json"), qObject{"protocol": "symphony.package-retirement-state.v1", "phase": "retired", "identity": qObject{"owner": "fixture-owner", "version": "1"}})
	qVerifyRetiredOwners(prefix, []qOwner{o})
	qerr(os.Remove(qjoin(prefix, base, "use.lock")))
	defer func() {
		if recover() == nil {
			t.Fatal("missing stable guard was accepted as successful retirement")
		}
	}()
	qVerifyRetiredOwners(prefix, []qOwner{o})
}
func nilBytes() []byte { return []byte{} }
