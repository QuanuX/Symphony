package knowledgelifecycle

import (
	"reflect"
	"testing"
)

func TestNativeOwnerBindingsAndClosedArguments(t *testing.T) {
	for _, bindings := range [][]string{{"unknown=/tmp/tool"}, {"stav-append-authority=relative"}, {"stav-append-authority=/"}, {"stav-append-authority=/tmp/../tool"}, {"stav-append-authority=/tmp/tool", "accordare-stav-producer=/tmp/tool"}, {"stav-append-authority=/tmp/tool\n"}} {
		if ValidateNativeOwners(bindings) == nil {
			t.Fatalf("unsafe owner bindings accepted: %q", bindings)
		}
	}
	if err := ValidateNativeOwners([]string{"stav-append-authority=/tmp/one", "stav-append-authority=/tmp/two"}); err != nil {
		t.Fatal(err)
	}
	want := []string{"package", "uninstall", "--prefix", "/exact/root", "--version", "0.1.0-dev", "--lifecycle-root-fd", "3", "--lifecycle-lock-fd", "4", "--lifecycle-proof-fd", "5"}
	if got := nativeOwnerArguments("uninstall", "secure-identity-access-governance", "/exact/root", "0.1.0-dev"); !reflect.DeepEqual(got, want) {
		t.Fatalf("argv drift: %q", got)
	}
}
