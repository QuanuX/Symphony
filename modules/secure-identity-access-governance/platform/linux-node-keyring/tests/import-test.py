"""Exercise the import boundary against a deterministic C++ keyring fixture."""

import concurrent.futures
import os
from pathlib import Path
import signal
import socket
import subprocess
import sys
import tempfile
import uuid

receiver = os.path.abspath(sys.argv[1])
collection = "/org/freedesktop/secrets/collection/login"
tops = str(uuid.uuid4())
reference = str(uuid.uuid4())
generation = b"cfbaec86-99e4-4bfd-9c8f-d84968e85c1c"
env = {"SSH_ORIGINAL_COMMAND": "symphony-keyring-import-v1"}


def frame(request, secret, *, selected_tops=tops, selected_ref=reference, length=None):
    count = len(secret) if length is None else length
    return f"SNKI1 {request} {selected_tops} {selected_ref} {count}\n".encode() + secret


def invoke(args, payload, environment=env):
    proc = subprocess.run(args, input=payload, capture_output=True, env=environment, timeout=10)
    assert not proc.stderr, "receiver emitted unexpected diagnostics"
    return proc.returncode, proc.stdout


def response(status, request, issued=None):
    return b" ".join([status, request.encode()] + ([issued] if issued else [])) + b"\n"


def main():
    if os.getuid() == 0:
        raise RuntimeError("run under an ordinary test UID")
    with tempfile.TemporaryDirectory(prefix="symphony-import-", dir="/tmp") as fixture:
        root = Path(fixture)
        state = root / "state"
        state.mkdir(mode=0o700)
        with socket.socket(socket.AF_UNIX) as bus:
            bus.bind(str(root / "bus"))
            args = [receiver, collection, tops, reference, str(state), str(root / "bus")]
            rid = str(uuid.uuid4())
            # A valid frame must be closed by EOF. It must match the node-owned
            # slot, and an authorized remote command cannot contain arguments.
            good = frame(rid, b"\x01\x00\xff")
            for bad in (
                b"", b"SNKI2" + good[5:], good[:-1], good + b"trailing",
                frame(rid, b"", length=0), frame(rid, b"1", length="01"),
                frame(rid, b"1", length=4097), frame(rid.upper(), b"1"),
                frame(rid, b"1", selected_ref=str(uuid.uuid4())),
                frame(rid, b"1", selected_tops=str(uuid.uuid4())),
            ):
                assert invoke(args, bad) == (1, b"REFUSED\n")
            assert invoke(args, good, {}) == (1, b"REFUSED\n")
            assert invoke(args, good, {"SSH_ORIGINAL_COMMAND": "symphony-keyring-import-v1 extra"}) == (1, b"REFUSED\n")
            linked = root / "linked"
            linked.symlink_to(state)
            assert invoke([*args[:4], str(linked), args[-1]], good) == (1, b"REFUSED\n")
            state.chmod(0o750)
            assert invoke(args, good) == (1, b"REFUSED\n")
            state.chmod(0o700)
            assert not list(state.iterdir())

            generated = b"\x01test-only-value\x00\xff" + os.urandom(32)
            assert invoke(args, frame(rid, generated)) == (0, response(b"STORED", rid, generation))
            # A different payload under the same ID must not be reported as a
            # new success. If this payload reached the backend it would SIGKILL.
            assert invoke(args, frame(rid, b"\x04")) == (3, response(b"ALREADY_STORED", rid, generation))
            assert generated not in (state / rid / "result").read_bytes()

            for selector, code, outcome, replay, replay_code in [
                (b"\x02", 1, b"REFUSED", b"ALREADY_REFUSED", 3),
                (b"\x03", 2, b"UNCERTAIN", b"UNCERTAIN", 2),
            ]:
                request = str(uuid.uuid4())
                gen = generation if code == 2 else None
                assert invoke(args, frame(request, selector)) == (code, response(outcome, request, gen))
                assert invoke(args, frame(request, b"\x04")) == (replay_code, response(replay, request, gen))
            request = str(uuid.uuid4())
            assert invoke(args, frame(request, b"\x04")) == (-signal.SIGKILL, b"")
            assert invoke(args, frame(request, b"\x01")) == (2, response(b"UNCERTAIN", request))
            assert not list((state / request).iterdir())
            # Receipts are scoped even if an administrator accidentally reuses
            # the state path for another reference or local bus.
            other_ref = str(uuid.uuid4())
            assert invoke([*args[:3], other_ref, *args[4:]], frame(rid, b"X", selected_ref=other_ref)) == (1, b"REFUSED\n")
            with socket.socket(socket.AF_UNIX) as other:
                other.bind(str(root / "other"))
                assert invoke([*args[:-1], str(root / "other")], good) == (1, b"REFUSED\n")
            # Reserve via mkdir on the same durable state concurrently. The loser
            # can see a committed receipt or the winner's pending reservation.
            concurrent_id = str(uuid.uuid4())
            with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
                outcomes = list(pool.map(lambda _: invoke(args, frame(concurrent_id, b"X")), range(2)))
            assert outcomes.count((0, response(b"STORED", concurrent_id, generation))) == 1
            assert any(r in outcomes for r in [(3, response(b"ALREADY_STORED", concurrent_id, generation)),
                                               (2, response(b"UNCERTAIN", concurrent_id))])
            # Even accidental copies of a record into another request directory
            # cannot make the new request appear to have committed.
            other = str(uuid.uuid4())
            (state / other).mkdir(mode=0o700)
            (state / other / "result").write_bytes((state / rid / "result").read_bytes())
            (state / other / "result").chmod(0o600)
            assert invoke(args, frame(other, b"X")) == (1, b"REFUSED\n")
            request = str(uuid.uuid4())
            assert invoke(args, frame(request, b"A" * 4096)) == (0, response(b"STORED", request, generation))
    print("PASS scoped import framing, prior outcomes, filesystem boundary and crash recovery")


if __name__ == "__main__":
    main()
