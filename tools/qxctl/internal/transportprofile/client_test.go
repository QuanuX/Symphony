//go:build darwin || linux

package transportprofile

import (
	"context"
	w "github.com/QuanuX/Symphony/tools/qxctl/internal/transportprofile/wire"
	"reflect"
	"testing"
)

func TestFixedInvocationAndClosedInput(t *testing.T) {
	for _, op := range []string{"apply", "recover"} {
		binary, args := invocation(op)
		if binary != "/usr/bin/sudo" || !reflect.DeepEqual(args, []string{"-n", "--", HelperPath(), "administer"}) {
			t.Fatal("open privilege command", binary, args)
		}
	}
	for _, op := range []string{"list", "inspect", "plan", "status"} {
		binary, args := invocation(op)
		if binary != HelperPath() || !reflect.DeepEqual(args, []string{"query"}) {
			t.Fatal("read elevated", op)
		}
	}
	if _, e := protectedBinary(t.TempDir() + "/helper"); e == nil {
		t.Fatal("user installation trusted")
	}
	input := w.Template()
	raw, _ := w.Canonical(input)
	if _, e := Run(context.Background(), "apply", raw); e == nil {
		t.Fatal("operation substitution admitted")
	}
}
func TestIndependentProfileConsumerGolden(t *testing.T) {
	object := map[string]any{"generation": "18446744073709551615", "a": "<", "b": []any{"x", nil}, "digest": "excluded"}
	// Literal preimage is evaluated independently, not through producer Digest.
	got := w.Digest("symphony.sov.transport-profile-test.v1", object)
	if got != "sha256:1f3474b87588c7328eb3813fcf60deca43fa08856ba9744adbbe6fc3f4702177" {
		t.Fatal(got)
	}
	input := w.Template()
	raw, _ := w.Canonical(input)
	var decoded w.Request
	if w.Decode(raw, &decoded) != nil || w.ValidateRequest(decoded) != nil {
		t.Fatal("wire mirror mismatch")
	}
	var invalid w.Request
	if w.Decode([]byte(`{"protocol":"one","protocol":"two"}`), &invalid) == nil {
		t.Fatal("duplicate accepted")
	}
}
func TestProfileResultBindingAndCompleteEvidence(t *testing.T) {
	input := w.Template()
	r := w.Result{Protocol: w.ResultProtocol, RequestID: input.RequestID, Operation: input.Operation, Scope: input.Scope, Profiles: []w.Snapshot{}, Completed: false, Code: "transport_profile.recovery_required", NextAction: "recover"}
	if w.ValidateResult(r, input) != nil {
		t.Fatal("safe refusal invalid")
	}
	r.RequestID = "55555555-5555-4555-8555-555555555555"
	if w.ValidateResult(r, input) == nil {
		t.Fatal("unrelated response accepted")
	}
}
