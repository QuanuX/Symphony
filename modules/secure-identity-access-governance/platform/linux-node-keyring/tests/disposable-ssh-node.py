"""Run as root only in a fresh --network none QA container (sshd on loopback)."""
import argparse
import contextlib
import os
import pathlib
import pwd
import secrets
import socket
import subprocess
import tempfile
import time
import uuid


def run(*cmd, **kw):
    return subprocess.run(cmd, check=True, timeout=35, **kw)


def wait_for(check, processes):
    deadline = time.monotonic() + 8
    while time.monotonic() < deadline:
        if any(p.poll() is not None for p in processes):
            raise RuntimeError("fixture daemon exited during startup")
        if check():
            return
        time.sleep(0.05)
    raise RuntimeError("fixture daemon did not become ready")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("prefix", type=pathlib.Path)
    parser.add_argument("installed_consumer", type=pathlib.Path)
    args = parser.parse_args()
    active = {p.parent.name for p in pathlib.Path("/sys/class/net").glob("*/flags") if int(p.read_text(), 16) & 1}
    if os.geteuid() != 0 or not pathlib.Path("/.dockerenv").is_file() or active != {"lo"}:
        raise SystemExit("Use a fresh root QA container with --network none")
    tools = args.prefix / "libexec/symphony/ssiag-node-keyring/0.1.0-dev"
    importer = tools / "symphony-node-keyring-import"
    sender = tools / "symphony-node-keyring-send"
    collection = "/org/freedesktop/secrets/collection/login"
    user = "nodefixture"
    run("useradd", "--create-home", "--shell", "/bin/sh", "--password", "*", user)
    account = pwd.getpwnam(user)
    identity = ["setpriv", f"--reuid={account.pw_uid}", f"--regid={account.pw_gid}", "--init-groups"]
    tops, reference, request = (str(uuid.uuid4()) for _ in range(3))
    with tempfile.TemporaryDirectory(prefix="node-keyring-", dir="/run") as tmp, contextlib.ExitStack() as stack:
        root = pathlib.Path(tmp)
        root.chmod(0o755)
        private = root / "session"
        private.mkdir(mode=0o700)
        os.chown(private, account.pw_uid, account.pw_gid)
        state = private / "receipts"
        state.mkdir(mode=0o700)
        os.chown(state, account.pw_uid, account.pw_gid)
        bus = private / "bus"
        address = "unix:path=" + str(bus)
        env = {"HOME": account.pw_dir, "USER": user, "LOGNAME": user, "PATH": "/usr/bin:/bin",
               "DBUS_SESSION_BUS_ADDRESS": address, "XDG_RUNTIME_DIR": str(private)}

        def daemon(cmd, **kwargs):
            p = subprocess.Popen(cmd, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, **kwargs)
            def stop():
                if p.poll() is None:
                    p.terminate()
                    try:
                        p.wait(timeout=3)
                    except subprocess.TimeoutExpired:
                        p.kill()
                        p.wait()
            stack.callback(stop)
            return p

        bus_config = root / "bus.conf"
        bus_config.write_text(f'<busconfig><type>session</type><listen>{address}</listen><auth>EXTERNAL</auth>'
                              '<policy context="default"><allow send_destination="*"/><allow receive_sender="*"/>'
                              '<allow own="*"/></policy></busconfig>')
        dbus = daemon([*identity, "dbus-daemon", "--nofork", "--config-file=" + str(bus_config)], env=env)
        wait_for(bus.is_socket, [dbus])
        keyring = daemon([*identity, "gnome-keyring-daemon", "--foreground", "--unlock", "--components=secrets"],
                         stdin=subprocess.PIPE, env=env)
        keyring.stdin.write(secrets.token_hex(32).encode())
        keyring.stdin.close()
        run(*identity, "gdbus", "wait", "--session", "--timeout", "8", "org.freedesktop.secrets", env=env)

        host_key, client_key, known = (root / n for n in ("hostkey", "clientkey", "pins"))
        for key in (host_key, client_key):
            run("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", key)
        with socket.socket() as probe:
            probe.bind(("127.0.0.1", 0))
            port = probe.getsockname()[1]
        public = host_key.with_suffix(".pub").read_text().split()
        known.write_text(f"[127.0.0.1]:{port} {public[0]} {public[1]}\n")
        authorized = root / "authorized_keys"
        forced = f"{importer} {collection} {tops} {reference} {state} {bus}"
        authorized.write_text(f'restrict,command="{forced}" ' + client_key.with_suffix(".pub").read_text())
        cfg = root / "sshd_config"
        cfg.write_text(f"Port {port}\nListenAddress 127.0.0.1\nHostKey {host_key}\nPidFile {root}/sshd.pid\n"
                       f"AuthorizedKeysFile {authorized}\nAllowUsers {user}\nDisableForwarding yes\nPermitTTY no\n"
                       "PasswordAuthentication no\nKbdInteractiveAuthentication no\nPermitUserRC no\nUsePAM no\n")
        pathlib.Path("/run/sshd").mkdir(mode=0o755, exist_ok=True)
        sshd = daemon(["/usr/sbin/sshd", "-D", "-e", "-f", str(cfg)])
        def listening():
            with socket.socket() as s:
                return s.connect_ex(("127.0.0.1", port)) == 0
        wait_for(listening, [dbus, keyring, sshd])

        value = b"\x00\xff" + secrets.token_bytes(32)
        def send(data, rid=request, slot=reference, pins=known):
            cmd = [sender, "--host", "127.0.0.1", "--user", user, "--port", str(port),
                   "--identity", client_key, "--known-hosts", pins, "--tops", tops,
                   "--reference", slot, "--request", rid, "--length", str(len(data))]
            result = subprocess.run(cmd, input=data, capture_output=True, timeout=35)
            if data in result.stdout or data in result.stderr:
                raise RuntimeError("credential appeared in output")
            return result

        bad = root / "wrong-pins"
        bad.write_text("")
        assert send(value, pins=bad).returncode != 0 and not list(state.iterdir())
        assert send(value, slot=str(uuid.uuid4())).returncode != 0 and not list(state.iterdir())
        delivered = send(value)
        if delivered.returncode:
            raise RuntimeError(f"fixture import failed with exit {delivered.returncode}")
        status, rid, generation = delivered.stdout.decode("ascii").split()
        assert status == "STORED" and rid == request and str(uuid.UUID(generation)) == generation
        previous = send(secrets.token_bytes(32))
        assert previous.returncode == 3 and previous.stdout == f"ALREADY_STORED {request} {generation}\n".encode()
        # No provisioning connection is available when the native module authenticates.
        sshd.terminate()
        sshd.wait(timeout=3)
        binding = root / "active"
        binding.write_text(f"SYMPHONY-NODE-KEYRING-1\nuid={account.pw_uid}\nbus_socket={bus}\n"
                           f"collection={collection}\ntops={tops}\nreference={reference}\n"
                           f"generation={generation}\nmaximum_bytes=64\n")
        binding.chmod(0o644)
        run(*identity, args.installed_consumer, binding, input=value, env=env)
    print("PASS pinned SSH import, wrong host and slot refusal, replay, and offline installed C++ consumer")


if __name__ == "__main__":
    main()
