//go:build linux && symphony_node_keyring_fixture

package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func sshChecks(prefix, consumer string) {
	disposableNode()
	a := addAccount("nodefixture")
	root := checked(os.MkdirTemp("/run", "node-keyring-"))
	defer os.RemoveAll(root)
	must(os.Chmod(root, 0755))
	private := filepath.Join(root, "session")
	a.mkdir(private)
	state := filepath.Join(private, "receipts")
	a.mkdir(state)
	bus := filepath.Join(private, "bus")
	address := "unix:path=" + bus
	value := append([]byte{0, 255}, randomBytes(32)...)
	unlock := fmt.Appendf(nil, "%x", randomBytes(32))
	p := process{env: append(a.environment(), "DBUS_SESSION_BUS_ADDRESS="+address, "XDG_RUNTIME_DIR="+private),
		identity: a.credentials(), protected: [][]byte{value, unlock}}
	config := filepath.Join(root, "bus.conf")
	xml := fmt.Sprintf(`<busconfig><type>session</type><listen>%s</listen><auth>EXTERNAL</auth><policy context="default"><allow send_destination="*"/><allow receive_sender="*"/><allow own="*"/></policy></busconfig>`, address)
	must(os.WriteFile(config, []byte(xml), 0644))
	dbus := p.start(nil, "dbus-daemon", "--nofork", "--config-file="+config)
	defer dbus.stop()
	waitFor(func() bool { return isSocket(bus) }, dbus)
	vault := p.start(unlock, "gnome-keyring-daemon", "--foreground", "--unlock", "--components=secrets")
	defer vault.stop()
	p.success(nil, "gdbus", "wait", "--session", "--timeout", "8", "org.freedesktop.secrets")

	hostKey := filepath.Join(root, "hostkey")
	clientKey := filepath.Join(root, "clientkey")
	pins := filepath.Join(root, "pins")
	system := process{protected: p.protected}
	for _, key := range []string{hostKey, clientKey} {
		system.success(nil, "ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", key)
	}
	probe := checked(net.Listen("tcp4", "127.0.0.1:0"))
	endpoint := probe.Addr().String()
	port := strconv.Itoa(probe.Addr().(*net.TCPAddr).Port)
	must(probe.Close())
	pub := strings.Fields(string(checked(os.ReadFile(hostKey + ".pub"))))
	must(os.WriteFile(pins, []byte("[127.0.0.1]:"+port+" "+pub[0]+" "+pub[1]+"\n"), 0600))
	tools := filepath.Join(prefix, "libexec/symphony/ssiag-node-keyring/0.1.1-dev")
	tops, ref, request := uuid(), uuid(), uuid()
	forced := strings.Join([]string{filepath.Join(tools, "symphony-node-keyring-import"), collection, tops, ref, state, bus}, " ")
	keys := filepath.Join(root, "authorized_keys")
	must(os.WriteFile(keys, append([]byte(`restrict,command="`+forced+`" `), checked(os.ReadFile(clientKey+".pub"))...), 0644))
	cfg := filepath.Join(root, "sshd_config")
	sshdConfig := fmt.Sprintf("Port %s\nListenAddress 127.0.0.1\nHostKey %s\nPidFile %s/sshd.pid\nAuthorizedKeysFile %s\nAllowUsers %s\nDisableForwarding yes\nPermitTTY no\nPasswordAuthentication no\nKbdInteractiveAuthentication no\nPermitUserRC no\nUsePAM no\n", port, hostKey, root, keys, a.name)
	must(os.WriteFile(cfg, []byte(sshdConfig), 0600))
	must(os.MkdirAll("/run/sshd", 0755))
	sshd := system.start(nil, "/usr/sbin/sshd", "-D", "-e", "-f", cfg)
	defer sshd.stop()
	waitFor(func() bool {
		conn, err := net.DialTimeout("tcp4", endpoint, 100*time.Millisecond)
		if err != nil {
			return false
		}
		conn.Close()
		return true
	}, dbus, vault, sshd)

	send := func(data []byte, slot, known string) result {
		client := process{protected: [][]byte{value, data}}
		return client.run(data, filepath.Join(tools, "symphony-node-keyring-send"),
			"--host", "127.0.0.1", "--user", a.name, "--port", port,
			"--identity", clientKey, "--known-hosts", known, "--tops", tops,
			"--reference", slot, "--request", request, "--length", strconv.Itoa(len(data)))
	}
	badPins := filepath.Join(root, "wrong-pins")
	must(os.WriteFile(badPins, nil, 0600))
	require(send(value, ref, badPins).code != 0 && len(checked(os.ReadDir(state))) == 0, "unpinned SSH host accepted")
	require(send(value, uuid(), pins).code != 0 && len(checked(os.ReadDir(state))) == 0, "wrong slot accepted")
	gen := generation(send(value, ref, pins), request)
	expectReply(send(randomBytes(32), ref, pins), 3, "ALREADY_STORED "+request+" "+gen+"\n")
	sshd.stop()
	selection := filepath.Join(root, "active")
	binding(selection, a, bus, tops, ref, gen)
	p.success(value, consumer, selection)
	fmt.Println("PASS pinned SSH import, wrong host and slot refusal, replay, and offline installed C++ consumer")
}
