package main

import (
	"strings"
	"testing"
)

func TestFoundationLifecycleCLIRejectsUnratifiedMutationShortcuts(t *testing.T) {
	root, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	for _, component := range []string{"ssiag", "stav"} {
		for _, surface := range []string{"enrollment", "supervisor"} {
			for _, leaf := range []string{"status", "plan", "apply", "apply-status", "recover"} {
				command, _, err := root.Find([]string{component, surface, leaf})
				if err != nil || command == nil {
					t.Fatalf("missing %s %s %s command: %v", component, surface, leaf, err)
				}
				for _, prohibited := range []string{"purge", "force", "no-stop", "serve"} {
					if command.Flags().Lookup(prohibited) != nil {
						t.Fatalf("%s %s %s exposes prohibited --%s", component, surface, leaf, prohibited)
					}
				}
				for _, requiredSurface := range []string{"prefix", "version", "tops-id", "scope", "json"} {
					if command.Flags().Lookup(requiredSurface) == nil {
						t.Fatalf("%s %s %s lacks --%s", component, surface, leaf, requiredSurface)
					}
				}
			}
		}
	}
}

func TestFoundationLifecycleInputValidationPrecedesAdapterInvocation(t *testing.T) {
	prefix := t.TempDir()
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"ssiag", "enrollment", "plan", "--prefix", prefix, "--tops-id", ssiagTestTOPSID}, "--operation-id, --expected-state-digest, and --desired-state are required"},
		{[]string{"stav", "supervisor", "plan", "--prefix", prefix, "--tops-id", stavTestTOPSID, "--operation-id", "op", "--expected-state-digest", "absent", "--desired-state", "enrolled"}, "supervisor --desired-state"},
		{[]string{"ssiag", "enrollment", "apply", "--prefix", prefix, "--tops-id", ssiagTestTOPSID}, "--plan and --expected-attempt-digest are required"},
		{[]string{"stav", "supervisor", "recover", "--prefix", prefix, "--tops-id", stavTestTOPSID, "--operation-id", "op"}, "exactly one of --expected-attempt-digest or --discover is required"},
		{[]string{"stav", "supervisor", "status", "--prefix", prefix, "--tops-id", "INVALID"}, "invalid foundational lifecycle target"},
	}
	for _, test := range tests {
		err := executeCommand(test.args)
		if err == nil || !strings.Contains(err.Error(), test.want) {
			t.Errorf("%v error = %v, want %q", test.args, err, test.want)
		}
	}
}

func TestFoundationLifecyclePlanIdentityRules(t *testing.T) {
	base := foundationLifecycleOptions{
		scope: "user", desiredState: "enrolled", topsName: "Desk", auditMode: "ordinary",
	}
	if _, err := foundationLifecycleIntent("ssiag", "enrollment", base); err != nil {
		t.Fatalf("valid user SSIAG intent rejected: %v", err)
	}
	base.serviceUID, base.serviceGID = "501", "20"
	if _, err := foundationLifecycleIntent("ssiag", "enrollment", base); err == nil || !strings.Contains(err.Error(), "rejects UID/GID overrides") {
		t.Fatalf("user identity override accepted: %v", err)
	}
	base.scope, base.serviceUID, base.serviceGID = "system", "", ""
	if _, err := foundationLifecycleIntent("ssiag", "enrollment", base); err == nil || !strings.Contains(err.Error(), "requires --service-uid") {
		t.Fatalf("system SSIAG enrollment without identity accepted: %v", err)
	}
}
