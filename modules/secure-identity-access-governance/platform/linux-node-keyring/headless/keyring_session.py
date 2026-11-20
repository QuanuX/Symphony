#!/usr/bin/python3 -I
"""Run a dedicated local D-Bus and GNOME Keyring with a systemd-decrypted credential."""
import contextlib
import ctypes
import fcntl
import os
from pathlib import Path
import pwd
import resource
import signal
import socket
import stat
import struct
import subprocess
import sys
import tempfile
import time


class Refused(Exception):
    pass


def require(condition):
    if not condition:
        raise Refused()


def private_directory(path, uid):
    s = path.lstat()
    require(stat.S_ISDIR(s.st_mode) and s.st_uid == uid and (s.st_mode & 0o077) == 0)


def configure_home(uid, initialize):
    home = Path(pwd.getpwuid(uid).pw_dir)
    require(home.is_absolute() and home.resolve(strict=True) == home and home != Path("/"))
    private_directory(home, uid)
    directory = home
    for component in (".local", "share", "keyrings"):
        directory /= component
        if initialize:
            directory.mkdir(mode=0o700, exist_ok=True)
        s = directory.lstat()
        require(stat.S_ISDIR(s.st_mode) and s.st_uid == uid and (s.st_mode & 0o022) == 0)
    private_directory(directory, uid)
    login = directory / "login.keyring"
    if initialize:
        require(not any(directory.iterdir()))
    else:
        check_vault(login, uid)
    return home, login


def check_vault(login, uid):
    s = login.lstat()
    require(stat.S_ISREG(s.st_mode) and s.st_uid == uid and s.st_nlink == 1 and
            (s.st_mode & 0o077) == 0 and s.st_size > 0)
    # Empty-password GNOME keyrings can be saved as plaintext keyfiles.
    with login.open("rb") as file:
        require(file.read(16) == b"GnomeKeyring\n\r\x00\n")


def private_credential(fd, s, uid):
    # systemd commonly retains root ownership and grants the unit UID a named
    # read ACL. The group bits in st_mode then reflect the POSIX ACL mask.
    if stat.S_IMODE(s.st_mode) == 0o400:
        return s.st_uid in (0, uid)
    if s.st_uid != 0 or stat.S_IMODE(s.st_mode) != 0o440:
        return False
    acl = os.getxattr(fd, "system.posix_acl_access")
    if len(acl) != 44 or acl[:4] != struct.pack("<I", 2):
        return False
    undefined = 0xffffffff
    entries = list(struct.iter_unpack("<HHI", acl[4:]))
    return entries == [(1, 4, undefined), (2, 4, uid), (4, 0, undefined),
                       (16, 4, undefined), (32, 0, undefined)]


def open_unlock(uid):
    directory = Path(os.environ["CREDENTIALS_DIRECTORY"])
    require(directory.is_absolute() and directory.resolve(strict=True) == directory)
    directory_fd = os.open(directory, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW | os.O_CLOEXEC)
    try:
        owner = os.fstat(directory_fd)
        require(owner.st_uid in (0, uid) and (owner.st_mode & 0o022) == 0)
        file_fd = os.open("vault-unlock", os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK | os.O_CLOEXEC,
                          dir_fd=directory_fd)
    finally:
        os.close(directory_fd)
    file = os.fdopen(file_fd, "rb", buffering=0)
    try:
        s = os.fstat(file_fd)
        require(stat.S_ISREG(s.st_mode) and s.st_nlink == 1 and s.st_size == 64 and
                private_credential(file_fd, s, uid))
        # The credential never enters this Python process's memory.
        return file
    except BaseException:
        file.close()
        raise


def launch(stack, args, env, stdin=subprocess.DEVNULL):
    proc = subprocess.Popen(args, stdin=stdin, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                            env=env, close_fds=True)

    def finish():
        if proc.poll() is None:
            proc.terminate()
            try:
                proc.wait(timeout=3)
            except subprocess.TimeoutExpired:
                proc.kill()
                proc.wait()

    stack.callback(finish)
    return proc


def is_unlocked(env):
    result = subprocess.run(
        ["/usr/bin/gdbus", "call", "--session", "--timeout", "1", "--dest", "org.freedesktop.secrets",
         "--object-path", "/org/freedesktop/secrets/collection/login",
         "--method", "org.freedesktop.DBus.Properties.Get", "org.freedesktop.Secret.Collection", "Locked"],
        stdin=subprocess.DEVNULL, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, env=env, timeout=2)
    return result.returncode == 0 and result.stdout.strip() == b"(<false>,)"


def notify_ready():
    dest = os.environ.get("NOTIFY_SOCKET")
    if dest:
        address = "\0" + dest[1:] if dest.startswith("@") else dest
        with socket.socket(socket.AF_UNIX, socket.SOCK_DGRAM) as sock:
            sock.sendto(b"READY=1", address)


def run(mode, uid):
    require(uid > 0 and os.getuid() == uid and os.geteuid() == uid and os.getgid() == os.getegid())
    require(set(os.getgroups()).issubset({os.getgid()}))
    resource.setrlimit(resource.RLIMIT_CORE, (0, 0))
    require(ctypes.CDLL(None, use_errno=True).prctl(4, 0, 0, 0, 0) == 0)  # PR_SET_DUMPABLE
    os.umask(0o077)
    runtime = Path(f"/run/symphony-node-keyring-{uid}")
    require(runtime.resolve(strict=True) == runtime)
    private_directory(runtime, uid)
    bus = runtime / "bus"
    address = f"unix:path={bus}"
    with contextlib.ExitStack() as stack:
        lockfd = os.open(runtime / "session.lock", os.O_RDONLY | os.O_CLOEXEC | os.O_NOFOLLOW | os.O_CREAT, 0o600)
        stack.callback(os.close, lockfd)
        fcntl.flock(lockfd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        require(not os.path.lexists(bus))
        home, login = configure_home(uid, mode == "enroll")
        env = {"HOME": str(home), "PATH": "/usr/bin:/bin", "XDG_DATA_HOME": str(home / ".local/share"),
               "XDG_RUNTIME_DIR": str(runtime), "DBUS_SESSION_BUS_ADDRESS": address}
        tmp = Path(stack.enter_context(tempfile.TemporaryDirectory(dir=runtime)))
        conf = tmp / "bus.conf"
        conf.write_text(f"<busconfig><type>session</type><listen>{address}</listen><auth>EXTERNAL</auth>"
                        '<policy context="default"><allow send_destination="*"/><allow receive_sender="*"/>'
                        '<allow own="*"/></policy></busconfig>')
        with open_unlock(uid) as password:
            daemon = launch(stack, ["/usr/bin/dbus-daemon", "--nofork", f"--config-file={conf}"], env)
            cutoff = time.monotonic() + 8
            while not bus.is_socket():
                require(daemon.poll() is None and time.monotonic() < cutoff)
                time.sleep(.05)
            keyring = launch(stack, ["/usr/bin/gnome-keyring-daemon", "--foreground", "--unlock",
                                     "--components=secrets", f"--control-directory={tmp}"], env, password)
        while not is_unlocked(env):
            require(keyring.poll() is None and daemon.poll() is None and time.monotonic() < cutoff)
            time.sleep(.08)
        check_vault(login, uid)
        if mode == "serve":
            notify_ready()
            while daemon.poll() is None and keyring.poll() is None:
                time.sleep(.25)
            raise Refused()


def main():
    require(len(sys.argv) == 3 and sys.argv[1] in ("enroll", "serve") and
            sys.argv[2].isascii() and sys.argv[2].isdigit() and str(int(sys.argv[2])) == sys.argv[2])
    def stop(_number, _frame):
        raise SystemExit(0)
    signal.signal(signal.SIGTERM, stop)
    run(sys.argv[1], int(sys.argv[2]))


if __name__ == "__main__":
    try:
        main()
    except (Refused, OSError, ValueError, KeyError, subprocess.SubprocessError):
        print("Keyring session refused or unavailable", file=sys.stderr)
        sys.exit(1)
