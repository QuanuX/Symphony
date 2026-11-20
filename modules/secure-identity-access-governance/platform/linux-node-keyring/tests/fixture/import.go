//go:build linux && symphony_node_keyring_fixture

package main

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func expectReply(r result, code int, text string) {
	require(r.code == code && r.stdout == text && r.stderr == "", "unexpected import response status or body")
}

func importChecks(receiver string) {
	require(os.Getuid() != 0, "import fixture requires a non-root service UID")
	root := checked(os.MkdirTemp("/tmp", "symphony-import-"))
	defer os.RemoveAll(root)
	state := filepath.Join(root, "state")
	must(os.Mkdir(state, 0700))
	bus := filepath.Join(root, "bus")
	sock := checked(net.Listen("unix", bus))
	defer sock.Close()
	tops, ref, rid := uuid(), uuid(), uuid()
	const issued = "cfbaec86-99e4-4bfd-9c8f-d84968e85c1c"
	p := process{env: []string{"SSH_ORIGINAL_COMMAND=symphony-keyring-import-v1"}}
	args := []string{receiver, collection, tops, ref, state, bus}
	send := func(id string, value []byte) result { return p.run(frame(id, tops, ref, value), args...) }
	good := frame(rid, tops, ref, []byte{1, 0, 255})
	invalidLength := func(length string) []byte {
		return fmt.Appendf(nil, "SNKI1 %s %s %s %s\n1", rid, tops, ref, length)
	}
	for _, bad := range [][]byte{
		nil, append([]byte("SNKI2"), good[5:]...), good[:len(good)-1], append(bytes.Clone(good), []byte("trailing")...),
		frame(rid, tops, ref, nil), invalidLength("01"), invalidLength("4097"),
		frame(strings.ToUpper("a"+rid[1:]), tops, ref, []byte{1}), frame(rid, tops, uuid(), []byte{1}), frame(rid, uuid(), ref, []byte{1}),
	} {
		expectReply(p.run(bad, args...), 1, "REFUSED\n")
	}
	for _, env := range [][]string{{}, {"SSH_ORIGINAL_COMMAND=symphony-keyring-import-v1 extra"}} {
		expectReply((process{env: env}).run(good, args...), 1, "REFUSED\n")
	}
	linked := filepath.Join(root, "linked")
	must(os.Symlink(state, linked))
	otherArgs := append([]string{}, args...)
	otherArgs[4] = linked
	expectReply(p.run(good, otherArgs...), 1, "REFUSED\n")
	must(os.Chmod(state, 0750))
	expectReply(p.run(good, args...), 1, "REFUSED\n")
	must(os.Chmod(state, 0700))
	require(len(checked(os.ReadDir(state))) == 0, "refused requests changed state")

	payload := append([]byte{1, 0, 255}, randomBytes(32)...)
	p.protected = [][]byte{payload}
	expectReply(send(rid, payload), 0, "STORED "+rid+" "+issued+"\n")
	expectReply(send(rid, []byte{4}), 3, "ALREADY_STORED "+rid+" "+issued+"\n")
	receipt := checked(os.ReadFile(filepath.Join(state, rid, "result")))
	require(!bytes.Contains(receipt, payload), "credential in import metadata")
	for _, test := range []struct {
		command          byte
		code, replayCode int
		outcome, replay  string
	}{
		{2, 1, 3, "REFUSED", "ALREADY_REFUSED"}, {3, 2, 2, "UNCERTAIN", "UNCERTAIN"},
	} {
		req := uuid()
		suffix := " " + req
		if test.command == 3 {
			suffix += " " + issued
		}
		expectReply(send(req, []byte{test.command}), test.code, test.outcome+suffix+"\n")
		expectReply(send(req, []byte{4}), test.replayCode, test.replay+suffix+"\n")
	}
	interrupted := uuid()
	expectReply(send(interrupted, []byte{4}), -int(syscall.SIGKILL), "")
	expectReply(send(interrupted, []byte{1}), 2, "UNCERTAIN "+interrupted+"\n")
	require(len(checked(os.ReadDir(filepath.Join(state, interrupted)))) == 0, "interrupted request gained a result")
	otherArgs = append([]string{}, args...)
	otherArgs[3] = uuid()
	expectReply(p.run(frame(rid, tops, otherArgs[3], []byte("X")), otherArgs...), 1, "REFUSED\n")
	otherArgs = append([]string{}, args...)
	otherArgs[5] = filepath.Join(root, "other")
	otherBus := checked(net.Listen("unix", otherArgs[5]))
	defer otherBus.Close()
	expectReply(p.run(good, otherArgs...), 1, "REFUSED\n")

	// Exactly one racing writer may commit. The other may see the durable
	// reservation or the winner's completed outcome.
	race := uuid()
	results := make(chan result, 2)
	for i := 0; i < 2; i++ {
		go func() { results <- send(race, []byte("X")) }()
	}
	first, second := <-results, <-results
	if first.code != 0 {
		first, second = second, first
	}
	expectReply(first, 0, "STORED "+race+" "+issued+"\n")
	if second.code == 2 {
		expectReply(second, 2, "UNCERTAIN "+race+"\n")
	} else {
		expectReply(second, 3, "ALREADY_STORED "+race+" "+issued+"\n")
	}
	copyID := uuid()
	must(os.Mkdir(filepath.Join(state, copyID), 0700))
	must(os.WriteFile(filepath.Join(state, copyID, "result"), receipt, 0600))
	expectReply(send(copyID, []byte("X")), 1, "REFUSED\n")
	full := uuid()
	expectReply(send(full, bytes.Repeat([]byte("A"), 4096)), 0, "STORED "+full+" "+issued+"\n")
	fmt.Println("PASS scoped import framing, prior outcomes, filesystem boundary and crash recovery")
}
