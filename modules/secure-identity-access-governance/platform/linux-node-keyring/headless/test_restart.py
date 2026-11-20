"""Qualify unattended GNOME Keyring restart in a fresh Linux --network none container."""

import contextlib
import errno
import os
from pathlib import Path
import pwd
import secrets
import struct
import subprocess
import sys
import tempfile
import time
import uuid


def main():
    network = Path("/sys/class/net")
    active = {entry.name for entry in network.iterdir() if entry.is_dir() and
              int((entry / "flags").read_text(), 16) & 1}
    if (os.geteuid() != 0 or not Path("/.dockerenv").is_file() or active != {"lo"} or
            len(Path("/proc/net/route").read_text().splitlines()) != 1):
        raise SystemExit("run in a disposable root QA container with --network none")
    if subprocess.check_output(["stat", "-f", "-c", "%T", "/dev/shm"]).strip() != b"tmpfs":
        raise SystemExit("a private tmpfs is required for test unlock input")
    launcher, importer, consumer = map(os.path.abspath, sys.argv[1:])
    name = "symphonybootqa"
    subprocess.run(["useradd", "--create-home", "--shell", "/bin/sh", "--password", "*", name], check=True)
    account = pwd.getpwnam(name)
    uid, gid = account.pw_uid, account.pw_gid
    home = Path(account.pw_dir)
    home.chmod(0o700)
    runtime = Path(f"/run/symphony-node-keyring-{uid}")
    runtime.mkdir(mode=0o700)
    os.chown(runtime, uid, gid)
    bus = runtime / "bus"
    identity = ["setpriv", f"--reuid={uid}", f"--regid={gid}", "--init-groups"]
    collection = "/org/freedesktop/secrets/collection/login"
    tops, reference, request = (str(uuid.uuid4()) for _ in range(3))
    value = b"\x00\xff" + secrets.token_bytes(42)
    unlock = secrets.token_hex(32).encode()
    with tempfile.TemporaryDirectory(dir="/dev/shm") as cdir, tempfile.TemporaryDirectory(dir="/run") as workspace:
        os.chown(cdir, uid, gid)
        credential = Path(cdir) / "vault-unlock"
        credential.write_bytes(unlock)
        os.chown(credential, uid, gid)
        credential.chmod(0o400)
        work = Path(workspace)
        work.chmod(0o755)
        state = work / "receipts"
        state.mkdir(mode=0o700)
        os.chown(state, uid, gid)
        env = {"HOME": str(home), "PATH": "/usr/bin:/bin", "USER": name, "LOGNAME": name,
               "CREDENTIALS_DIRECTORY": cdir}
        command = [*identity, sys.executable, "-I", "-B", launcher]

        def call(cmd, *, data=None, extra=None, timeout=12):
            result = subprocess.run(cmd, input=data or b"", capture_output=True, timeout=timeout,
                                    env=env | (extra or {}))
            if any(secret in stream for stream in (result.stdout, result.stderr) for secret in (value, unlock)):
                raise RuntimeError("test value in process output")
            return result

        def require_failure(mode):
            result = call([*command, mode, str(uid)])
            assert result.returncode != 0, f"unexpected {mode} success"
            assert not bus.exists(), "bus left accessible after refused startup"

        def ready():
            if not bus.is_socket():
                return False
            result = call([*identity, "gdbus", "call", "--address", "unix:path=" + str(bus),
                           "--dest", "org.freedesktop.secrets", "--object-path", collection,
                           "--method", "org.freedesktop.DBus.Properties.Get", "org.freedesktop.Secret.Collection", "Locked"])
            return result.returncode == 0 and result.stdout.strip() == b"(<false>,)"

        @contextlib.contextmanager
        def service():
            daemon = subprocess.Popen([*command, "serve", str(uid)], env=env,
                                      stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            try:
                cutoff = time.monotonic() + 10
                while not ready():
                    if daemon.poll() is not None or time.monotonic() >= cutoff:
                        raise RuntimeError("unattended keyring failed to unlock")
                    time.sleep(0.05)
                yield
            finally:
                if daemon.poll() is None:
                    daemon.terminate()
                    try:
                        daemon.wait(timeout=5)
                    except subprocess.TimeoutExpired:
                        daemon.kill()
                        daemon.wait()
                        raise RuntimeError("headless service failed to stop")
            assert not bus.exists(), "private bus remained after service shutdown"

        require_failure("serve")
        enrollment = call([*command, "enroll", str(uid)])
        assert enrollment.returncode == 0, "first node-local enrollment failed"
        require_failure("enroll")

        with service():
            frame = f"SNKI1 {request} {tops} {reference} {len(value)}\n".encode() + value
            reply = call([*identity, importer, collection, tops, reference, str(state), str(bus)],
                         data=frame, extra={"SSH_ORIGINAL_COMMAND": "symphony-keyring-import-v1"})
            assert reply.returncode == 0, "native import refused in service session"
            tag, rid, gen = reply.stdout.decode("ascii").strip().split()
            assert tag == "STORED" and rid == request and str(uuid.UUID(gen)) == gen

        keyring = home / ".local/share/keyrings/login.keyring"
        assert keyring.is_file() and value not in keyring.read_bytes()
        # Simulate a fresh root-delivered systemd credential with the wrong
        # bytes, then a missing credential. Neither may create another vault.
        credential.write_bytes(secrets.token_hex(32).encode())
        require_failure("serve")
        credential.unlink()
        require_failure("serve")
        credential.write_bytes(unlock)
        credential.chmod(0o400)
        # systemd preserves root ownership and grants only the unit's UID a
        # POSIX ACL when the backing filesystem supports it. Earlier enrollment
        # exercised the alternate service-owner-only 0400 representation.
        undefined = 0xffffffff
        acl = [(1, 4, undefined), (2, 4, uid), (4, 0, undefined), (16, 4, undefined), (32, 0, undefined)]
        encoded_acl = struct.pack("<I", 2) + b"".join(struct.pack("<HHI", *entry) for entry in acl)
        try:
            os.setxattr(credential, "system.posix_acl_access", encoded_acl)
        except OSError as exc:
            if exc.errno != errno.ENOTSUP:
                raise
            # Some tmpfs kernels (including LinuxKit) lack POSIX ACLs.
            # systemd's fallback is a service-owned file in a read-only mount.
            os.chown(credential, uid, gid)
            print("Fixture tmpfs uses the service-owner credential representation")

        # Test the actual kernel ACL on a filesystem with ACL support even
        # when the runner's tmpfs lacks ACLs. This public marker never opens
        # any vault; the real fixture's unlock remains exclusively in tmpfs.
        with tempfile.TemporaryDirectory(dir="/var/tmp") as sample:
            os.chmod(sample, 0o755)
            marker = Path(sample) / "vault-unlock"
            marker.write_bytes(b"x" * 64)
            marker.chmod(0o400)
            os.setxattr(marker, "system.posix_acl_access", encoded_acl)
            inspect = [*identity, sys.executable, "-I", "-B", "-c",
                       "import os,runpy,sys; h=runpy.run_path(sys.argv[1]); "
                       "h['open_unlock'](os.getuid()).close()", launcher]
            assert call(inspect, extra={"CREDENTIALS_DIRECTORY": sample}).returncode == 0
            # The service UID can still read the file, but a second group
            # can now read it as well, so the bootstrap must refuse.
            acl[2] = (4, 4, undefined)
            unsafe = struct.pack("<I", 2) + b"".join(struct.pack("<HHI", *entry) for entry in acl)
            os.setxattr(marker, "system.posix_acl_access", unsafe)
            assert call(inspect, extra={"CREDENTIALS_DIRECTORY": sample}).returncode != 0

        binding = work / "binding"
        binding.write_text(f"SYMPHONY-NODE-KEYRING-1\nuid={uid}\nbus_socket={bus}\ncollection={collection}\n"
                           f"tops={tops}\nreference={reference}\ngeneration={gen}\nmaximum_bytes=64\n")
        binding.chmod(0o644)
        assert call([*identity, consumer, "--expect-refusal", "unavailable", str(binding)]).returncode == 0
        with service():
            result = call([*identity, consumer, str(binding)], data=value)
            assert result.returncode == 0, "installed native consumer could not use the pre-restart generation"
    print("PASS unattended service restart, persistent generation, missing/wrong unlock refusal, and offline native use")


if __name__ == "__main__":
    main()
