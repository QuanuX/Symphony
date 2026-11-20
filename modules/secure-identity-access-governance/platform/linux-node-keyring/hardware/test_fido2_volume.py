"""Exercise the startup policy without opening a block device or touching a token."""

from copy import deepcopy
import importlib.util
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location("fido2_volume", Path(__file__).with_name("fido2_volume.py"))
volume = importlib.util.module_from_spec(spec)
spec.loader.exec_module(volume)

TOKEN = {"type": "systemd-fido2", "keyslots": ["1"], "fido2-rp": "io.systemd.cryptsetup",
         "fido2-clientPin-required": True, "fido2-up-required": True,
         "fido2-salt": "synthetic-salt", "fido2-credential": "synthetic-id"}
HEADER = {"keyslots": {"1": {"type": "luks2"}}, "tokens": {"0": TOKEN}}
MOUNT = "13 10 253:8 / /var/lib/symphony/nodevault rw - ext4 /dev/mapper/symphonynodevault rw"


class VolumePolicy(unittest.TestCase):
    def test_enrolled_and_second_enrolled_token(self):
        volume.check_slots(HEADER)
        header = deepcopy(HEADER)
        header["keyslots"]["2"] = {"type": "luks2"}
        header["tokens"]["1"] = {**TOKEN, "keyslots": ["2"]}
        volume.check_slots(header)

    def test_extra_keyslot(self):
        header = deepcopy(HEADER)
        header["keyslots"]["3"] = {"type": "luks2"}
        with self.assertRaises(volume.Refused):
            volume.check_slots(header)

    def test_wrong_or_weaker_enrollment(self):
        for field, value in [("type", "systemd-tpm2"), ("fido2-up-required", False),
                             ("fido2-clientPin-required", "true"), ("fido2-rp", "other"),
                             ("keyslots", []), ("keyslots", ["2"]), ("fido2-credential", "")]:
            with self.subTest(field=field), self.assertRaises(volume.Refused):
                volume.check_slots({"keyslots": HEADER["keyslots"], "tokens": {"0": {**TOKEN, field: value}}})

    def test_dedicated_mount_required(self):
        self.assertEqual(volume.mount_device(MOUNT), "253:8")
        for entry in ["", MOUNT.replace("nodevault rw", "elsewhere rw"),
                      MOUNT.replace(" / /var", " /subdir /var"), MOUNT.replace("ext4", "tmpfs"),
                      MOUNT + "\n" + MOUNT, MOUNT + "\n" + MOUNT.replace("nodevault rw", "nodevault/.local rw")]:
            with self.subTest(entry=entry), self.assertRaises(volume.Refused):
                volume.mount_device(entry)


if __name__ == "__main__":
    unittest.main()
