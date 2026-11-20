"""Run inside a disposable Linux container with a private, root-owned /run."""

import concurrent.futures
import os
from pathlib import Path
import stat
import subprocess
import sys
import tempfile
import uuid


helper = os.path.abspath(sys.argv[1])
old, first, second, third = (str(uuid.uuid4()) for _ in range(4))


def activate(path, expected, new):
    result = subprocess.run([helper, str(path), expected, new], capture_output=True, text=True, timeout=5)
    assert not result.stderr
    return result.returncode, result.stdout.strip()


def main():
    if os.getuid() != 0:
        raise RuntimeError("run the local administration tests as root in a disposable container")
    with tempfile.TemporaryDirectory(prefix="symphony-activate-", dir="/run") as tmp:
        root = Path(tmp)
        target = root / "venue.binding"
        doc = (
            "SYMPHONY-NODE-KEYRING-1\nuid=1000\nbus_socket=/run/user/1000/bus\n"
            "collection=/org/freedesktop/secrets/collection/login\n"
            f"tops={uuid.uuid4()}\nreference={uuid.uuid4()}\ngeneration={old}\nmaximum_bytes=4096\n"
        )
        assert activate(target, old, first) == (1, "REFUSED")
        target.write_text(doc)
        target.chmod(0o640)
        assert activate(target, old.upper(), first) == (1, "REFUSED")
        assert activate(target, second, first) == (3, "STALE")
        linked = root / "link"
        linked.symlink_to(target.name)
        assert activate(linked, old, first) == (1, "REFUSED")
        hard = root / "hard"
        hard.hardlink_to(target)
        assert activate(target, old, first) == (1, "REFUSED")
        hard.unlink()
        root.chmod(0o770)
        assert activate(target, old, first) == (1, "REFUSED")
        root.chmod(0o700)
        target.chmod(0o666)
        assert activate(target, old, first) == (1, "REFUSED")
        target.chmod(0o640)
        target.write_text(doc.replace("maximum_bytes=4096", "maximum_bytes=4097"))
        assert activate(target, old, first) == (1, "REFUSED")
        target.write_text(doc)
        try:
            os.setxattr(target, "user.example", b"retain")
        except OSError:
            pass
        else:
            assert activate(target, old, first) == (1, "REFUSED")
            assert os.getxattr(target, "user.example") == b"retain"
            os.removexattr(target, "user.example")
        with concurrent.futures.ThreadPoolExecutor(max_workers=2) as threads:
            futures = {g: threads.submit(activate, target, old, g) for g in (first, second)}
            results = {g: future.result() for g, future in futures.items()}
        winner, = [g for g, outcome in results.items() if outcome == (0, "ACTIVE " + g)]
        loser = second if winner == first else first
        assert results[loser] == (3, "STALE")
        assert stat.S_IMODE(target.stat().st_mode) == 0o640
        assert target.read_text() == doc.replace(f"generation={old}", f"generation={winner}")
        assert activate(target, old, winner) == (0, "ALREADY_ACTIVE " + winner)
        assert activate(target, old, third) == (3, "STALE")
        assert activate(target, winner, third) == (0, "ACTIVE " + third)
        assert not list(root.glob("*.tmp"))
    print("PASS owner-only generation activation, concurrent CAS and retry")


if __name__ == "__main__":
    main()
