#!/usr/bin/python3 -I
"""Create an encrypted systemd credential for an empty node vault exactly once."""
import ctypes
import os
from pathlib import Path
import pwd
import resource
import stat
import subprocess
import sys
import tempfile


def seal(uid, key):
    if os.geteuid() != 0 or os.getuid() != 0 or uid <= 0 or key not in ("host", "tpm2"):
        raise ValueError()
    home = Path(pwd.getpwuid(uid).pw_dir)
    if not home.is_absolute() or home.resolve(strict=True) != home or home.stat().st_uid != uid:
        raise ValueError()
    existing = home / ".local/share/keyrings"
    if os.path.lexists(existing) and (existing.resolve(strict=True) != existing or any(existing.iterdir())):
        raise ValueError()
    resource.setrlimit(resource.RLIMIT_CORE, (0, 0))
    if ctypes.CDLL(None, use_errno=True).prctl(4, 0, 0, 0, 0) != 0:  # PR_SET_DUMPABLE
        raise ValueError()
    os.umask(0o077)
    directory = Path("/etc/credstore.encrypted")
    directory.mkdir(mode=0o700, exist_ok=True)
    s = directory.lstat()
    if directory.resolve(strict=True) != directory or not stat.S_ISDIR(s.st_mode) or s.st_uid or s.st_mode & 0o022:
        raise ValueError()
    dest = directory / f"symphony-node-keyring-{uid}.cred"
    if os.path.lexists(dest):
        raise ValueError()
    args = ["/usr/bin/systemd-creds", "encrypt", f"--with-key={key}", "--name=vault-unlock"]
    if key == "tpm2":
        args.extend(["--tpm2-device=auto", "--tpm2-pcrs=7"])
    args.extend(["-", "-"])
    # systemd-creds writes ciphertext directly. The random vault password
    # travels only over stdin and is never an argument or environment value.
    with tempfile.NamedTemporaryFile(dir=directory, prefix=".node-keyring-", mode="w+b") as ciphertext:
        password = bytearray(os.getrandom(32).hex(), "ascii")
        if len(password) != 64:
            raise ValueError()
        try:
            subprocess.run(args, input=password, stdout=ciphertext, stderr=subprocess.DEVNULL,
                           env={"PATH": "/usr/bin:/bin"}, timeout=40, check=True)
        finally:
            password[:] = bytes(len(password))
        ciphertext.flush()
        if os.fstat(ciphertext.fileno()).st_size == 0:
            raise ValueError()
        os.fsync(ciphertext.fileno())
        os.link(ciphertext.name, dest)  # Fails if another enrollment won.
        parent = os.open(directory, os.O_DIRECTORY | os.O_RDONLY | os.O_CLOEXEC)
        try:
            os.fsync(parent)
        finally:
            os.close(parent)


if __name__ == "__main__":
    try:
        if len(sys.argv) != 3 or not sys.argv[1].isascii() or not sys.argv[1].isdigit() or str(int(sys.argv[1])) != sys.argv[1]:
            raise ValueError()
        seal(int(sys.argv[1]), sys.argv[2])
        print("Sealed local vault unlock credential")
    except (KeyError, OSError, ValueError, subprocess.SubprocessError):
        print("Vault seal refused or unavailable", file=sys.stderr)
        sys.exit(1)
