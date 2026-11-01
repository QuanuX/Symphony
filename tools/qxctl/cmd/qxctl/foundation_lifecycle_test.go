package main

import (
	"math"
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

func TestFoundationLifecycleOwnershipIdentityBounds(t *testing.T) {
	for _, test := range []struct {
		name, uid, gid   string
		wantUID, wantGID uint32
		wantError        string
	}{
		{"root", "0", "0", 0, 0, ""},
		{"ordinary", "501", "20", 501, 20, ""},
		{"highest usable UID", "4294967294", "20", math.MaxUint32 - 1, 20, ""},
		{"highest usable GID", "501", "4294967294", 501, math.MaxUint32 - 1, ""},
		{"reserved UID", "4294967295", "20", 0, 0, "invalid --service-uid: reserved ownership sentinel"},
		{"reserved GID", "501", "4294967295", 0, 0, "invalid --service-gid: reserved ownership sentinel"},
		{"overflow UID", "4294967296", "20", 0, 0, "invalid --service-uid"},
		{"overflow GID", "501", "4294967296", 0, 0, "invalid --service-gid"},
		{"negative UID", "-1", "20", 0, 0, "invalid --service-uid"},
		{"negative GID", "501", "-1", 0, 0, "invalid --service-gid"},
		{"missing GID", "501", "", 0, 0, "must be supplied together"},
	} {
		t.Run(test.name, func(t *testing.T) {
			uid, gid, err := pairedUint32(test.uid, test.gid, "service")
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) || uid != nil || gid != nil {
					t.Fatalf("invalid identity returned uid=%v gid=%v err=%v; want %q", uid, gid, err, test.wantError)
				}
				return
			}
			if err != nil || uid == nil || gid == nil || *uid != test.wantUID || *gid != test.wantGID {
				t.Fatalf("usable identity rejected or changed: uid=%v gid=%v err=%v", uid, gid, err)
			}
		})
	}
}

func TestFoundationLifecycleCLIRejectsReservedOwnershipBeforeInvocation(t *testing.T) {
	prefix := t.TempDir() // No adapter exists, so a dispatch would return a different error.
	for _, component := range []string{"ssiag", "stav"} {
		label := "service"
		if component == "stav" {
			label = "authority"
		}
		for _, field := range []string{"uid", "gid"} {
			t.Run(component+"/"+field, func(t *testing.T) {
				uid, gid := "501", "20"
				if field == "uid" {
					uid = "4294967295"
				} else {
					gid = "4294967295"
				}
				args := []string{component, "enrollment", "plan", "--prefix", prefix,
					"--tops-id", ssiagTestTOPSID, "--scope", "system", "--operation-id", "ownership-check",
					"--expected-state-digest", "absent", "--desired-state", "enrolled",
					"--" + label + "-uid", uid, "--" + label + "-gid", gid}
				if component == "ssiag" {
					args = append(args, "--tops-name", "Desk")
				}
				want := "invalid --" + label + "-" + field + ": reserved ownership sentinel"
				if err := executeCommand(args); err == nil || !strings.Contains(err.Error(), want) {
					t.Fatalf("CLI did not reject ownership before dispatch: %v; want %q", err, want)
				}
			})
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
