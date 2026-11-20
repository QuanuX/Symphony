"""Read-only startup check for the optional Symphony GNOME/FIDO2 volume profile."""

import json
import os
from pathlib import Path
import pwd
import re
import stat
import subprocess
import sys
from uuid import UUID


HOME = Path("/var/lib/symphony/nodevault")
UUID_FILE = Path("/etc/symphony/node-keyring/luks-uuid")
MAPPER = "symphonynodevault"


class Refused(Exception):
    """A fixed, nonsecret reason suitable for the system journal."""


def require(condition, reason):
    if not condition:
        raise Refused(reason)


def check_slots(header):
    slots, tokens = header.get("keyslots"), header.get("tokens")
    require(isinstance(slots, dict) and slots and isinstance(tokens, dict), "missing enrollment")
    covered = set()
    for token in tokens.values():
        require(isinstance(token, dict) and token.get("type") == "systemd-fido2",
                "unexpected unlock method")
        require(token.get("fido2-clientPin-required") is True
                and token.get("fido2-up-required") is True
                and token.get("fido2-rp") == "io.systemd.cryptsetup"
                and bool(token.get("fido2-salt")) and bool(token.get("fido2-credential")),
                "FIDO2 enrollment requires PIN and user presence")
        binding = token.get("keyslots")
        require(isinstance(binding, list) and len(binding) == 1
                and isinstance(binding[0], str) and binding[0] in slots
                and binding[0] not in covered, "unexpected slot binding")
        covered.add(binding[0])
    require(covered == slots.keys(), "extra unlock slot")


def mount_device(mountinfo):
    devices = []
    for line in mountinfo.splitlines():
        fields, filesystem = line.split(" - ", 1)
        mount_id, parent, device, root, path, *options = fields.split()
        require(not path.startswith(str(HOME) + "/"), "unexpected nested mount")
        if path == str(HOME):
            require(root == "/" and filesystem.split()[0] == "ext4", "unexpected filesystem")
            require(re.fullmatch(r"[0-9]+:[0-9]+", device), "invalid mount device")
            devices.append(device)
    require(len(devices) == 1, "dedicated volume is not mounted")
    return devices[0]


def read_uuid():
    for directory in UUID_FILE.parents:
        info = directory.lstat()
        require(stat.S_ISDIR(info.st_mode) and info.st_uid == 0 and not info.st_mode & 0o022,
                "untrusted configuration directory")
    with os.fdopen(os.open(UUID_FILE, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK), "r") as config:
        info = os.fstat(config.fileno())
        require(stat.S_ISREG(info.st_mode) and info.st_uid == 0 and not info.st_mode & 0o022
                and info.st_size <= 37, "untrusted volume selection")
        value = config.read(38).removesuffix("\n")
    require(str(UUID(value)) == value, "invalid volume selection")
    return value


def check_volume(uid):
    require(sys.platform == "linux" and os.geteuid() == 0, "Linux system manager required")
    require(0 < uid < 4294967295 and pwd.getpwuid(uid).pw_dir == str(HOME),
            "unexpected account home")
    volume_uuid = read_uuid()
    mounted = mount_device(Path("/proc/self/mountinfo").read_text())
    mapped = Path("/sys/dev/block") / mounted
    expected = f"CRYPT-LUKS2-{UUID(volume_uuid).hex}-{MAPPER}"
    require((mapped / "dm/uuid").read_text().strip() == expected, "wrong mapped volume")
    require((mapped / "dm/name").read_text().strip() == MAPPER, "wrong mapping name")

    # Resolve the actual device below the mounted mapper. Checking an arbitrary
    # enrolled device elsewhere in /dev would say nothing about this keyring.
    backings = list((mapped / "slaves").iterdir())
    require(len(backings) == 1, "unexpected mapper topology")
    raw_device = (backings[0] / "dev").read_text().strip()
    require(re.fullmatch(r"[0-9]+:[0-9]+", raw_device), "invalid backing device")
    raw_path = Path("/dev/block") / raw_device
    raw_stat = raw_path.stat()
    require(stat.S_ISBLK(raw_stat.st_mode) and raw_stat.st_rdev ==
            (Path("/dev/disk/by-uuid") / volume_uuid).stat().st_rdev, "wrong backing volume")

    # This fixed profile expects a pre-created private home and GNOME's data
    # directory on the same filesystem, with no symlink or nested mount.
    device = os.makedev(*map(int, mounted.split(":")))
    for directory in (HOME, HOME / ".local", HOME / ".local/share", HOME / ".local/share/keyrings"):
        info = directory.lstat()
        require(stat.S_ISDIR(info.st_mode) and info.st_uid == uid and info.st_dev == device
                and not info.st_mode & 0o022, "unexpected keyring directory")
        if directory == HOME or directory.name == "keyrings":
            require(info.st_mode & 0o077 == 0, "private directories required")

    header = subprocess.run(
        ["/usr/sbin/cryptsetup", "luksDump", "--dump-json-metadata", str(raw_path)],
        check=True, stdin=subprocess.DEVNULL, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
        timeout=8, env={"PATH": "/usr/sbin:/usr/bin", "LC_ALL": "C"})
    check_slots(json.loads(header.stdout))


def main():
    try:
        require(len(sys.argv) == 2 and re.fullmatch(r"[1-9][0-9]{0,9}", sys.argv[1]),
                "expected a dedicated numeric UID")
        check_volume(int(sys.argv[1]))
    except Refused as failure:
        print("Hardware volume preflight refused:", failure, file=sys.stderr)
        return 1
    except (OSError, ValueError, KeyError, TypeError, AttributeError, OverflowError,
            subprocess.SubprocessError):
        print("Hardware volume preflight refused: local evidence unavailable", file=sys.stderr)
        return 1
    print("Hardware volume preflight passed")
    return 0


if __name__ == "__main__":
    sys.exit(main())
