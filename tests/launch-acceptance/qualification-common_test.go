package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func qRefuses(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected refusal")
		}
	}()
	fn()
}
func TestQualificationWritePreservesExistingEvidence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "evidence.json")
	qwrite(path, qObject{"publication_authorized": false})
	before := qdigest(path)
	qRefuses(t, func() { qwrite(path, qObject{"publication_authorized": true}) })
	if qdigest(path) != before {
		t.Fatal("evidence overwritten")
	}
}
func TestQualificationNewWorkRefusesInputOrAlias(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	qerr(os.Mkdir(source, 0700))
	link := filepath.Join(root, "source-alias")
	qerr(os.Symlink(source, link))
	for _, path := range []string{source, filepath.Join(source, "work"), filepath.Join(link, "work"), "relative-work"} {
		t.Run(path, func(t *testing.T) { qRefuses(t, func() { qnewwork(path, source) }) })
	}
	if qexists(filepath.Join(source, "work")) {
		t.Fatal("refusal created work")
	}
	sibling := qnewwork(filepath.Join(root, "source-sibling"), source)
	if !qexists(sibling) {
		t.Fatal("distinct sibling rejected")
	}
}
func TestQualificationCopyRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	qerr(os.Mkdir(source, 0700))
	outside := filepath.Join(root, "outside")
	qwrite(outside, []byte("retained"))
	qerr(os.Symlink(outside, filepath.Join(source, "link")))
	qRefuses(t, func() { qcopytree(source, filepath.Join(root, "copy")) })
	if string(qbytes(outside)) != "retained" {
		t.Fatal("source mutated")
	}
}
func TestQualificationReadRejectsTrailingJSON(t *testing.T) {
	for _, input := range []string{"{} {}", "[]", "null"} {
		t.Run(input, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "bad.json")
			qwrite(path, []byte(input))
			qRefuses(t, func() { qread(path) })
		})
	}
}
func TestQualificationReadRejectsInvalidUTF8(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid-utf8.json")
	qwrite(path, []byte("{\"value\":\"\xff\"}"))
	qRefuses(t, func() { qread(path) })
}
func TestQualificationRefusesOverclaimedProof(t *testing.T) {
	base := qObject{"selected_artifact_owner_verified": true, "fresh_host_verified": false, "release_acceptance_complete": false, "publication_authorized": false}
	qproof(base)
	for _, key := range []string{"selected_artifact_owner_verified", "fresh_host_verified", "release_acceptance_complete", "publication_authorized"} {
		t.Run(key, func(t *testing.T) {
			p := qObject{}
			for k, v := range base {
				p[k] = v
			}
			p[key] = !qb(base[key])
			qRefuses(t, func() { qproof(p) })
		})
	}
}
func TestQualificationCommandFailureRetainsLog(t *testing.T) {
	root := t.TempDir()
	r := qnewRunner(root, 5*time.Second, nil)
	qRefuses(t, func() { r.run("failure", []string{"/bin/sh", "-c", "echo retained-failure >&2; exit 7"}, 0) })
	if string(qbytes(filepath.Join(root, "logs/failure.log"))) != "retained-failure\n" || len(r.commands) != 1 || r.commands[0]["exit_code"] != 7 {
		t.Fatal("failure evidence missing")
	}
}
func TestQualificationCommandTimeoutRefuses(t *testing.T) {
	r := qnewRunner(t.TempDir(), time.Millisecond, nil)
	qRefuses(t, func() { r.run("timeout", []string{"/bin/sleep", "10"}, 0) })
}
func TestQualificationFixtureKeyCleanupOnFailure(t *testing.T) {
	root := t.TempDir()
	r := qnewRunner(root, time.Second, nil)
	var key string
	qRefuses(t, func() {
		var cleanup func()
		key, cleanup = qfixtureKey(root, r)
		defer cleanup()
		qwrite(key, []byte("disposable fixture"))
		panic("later qualification refusal")
	})
	if qexists(filepath.Dir(key)) {
		t.Fatal("fixture key retained after failure")
	}
}
