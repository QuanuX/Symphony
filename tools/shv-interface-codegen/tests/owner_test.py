import argparse, copy, hashlib, importlib.util, json, subprocess, tempfile, unittest
from pathlib import Path

H = Path(__file__).resolve().parents[1]
R = H.parents[1]
s = importlib.util.spec_from_file_location("owner_codegen", H / "owner_codegen.py")
g = importlib.util.module_from_spec(s)
s.loader.exec_module(g)
p = argparse.ArgumentParser()
p.add_argument("--registration", type=Path, required=True)
p.add_argument("--engine", required=True)
a, rest = p.parse_known_args()


class OwnerTests(unittest.TestCase):
    def test_frozen_descriptor(self):
        r, d = g.load(R, a.registration)
        hist = g.read(R / r["history"])["descriptors"]
        expected = copy.deepcopy(list(hist.values())[-1])
        expected["engine_version"] = d["current_version"]
        expected.pop("descriptor_digest")
        expected["descriptor_digest"] = g.digest(expected)
        self.assertEqual(
            json.loads(subprocess.check_output([a.engine, "--descriptor"])), expected
        )

    def test_generated_parity(self):
        r, d = g.load(R, a.registration)
        g.emit(R, r, d, True)
        self.assertEqual(g.outputs(r, d), g.outputs(r, copy.deepcopy(d)))

    def test_future_owner_and_collision(self):
        r, d = g.load(R, a.registration)
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            d = copy.deepcopy(d)
            d.update(
                module_id="user-sensor-engine",
                engine_id="user-sensor",
                namespace="user::sensor",
                current_version="1.0.0",
                releases={"1.0.0": [o["operation_name"] for o in d["operations"]]},
                schemas=[],
                companions=[],
                embedded_dependencies=[],
            )
            (root / "declaration.json").write_text(json.dumps(d))
            (root / "history.json").write_text('{"descriptors":{}}')
            r.update(
                declaration="declaration.json",
                history="history.json",
                history_digest="sha256:"
                + hashlib.sha256((root / "history.json").read_bytes()).hexdigest(),
                go_title="UserSensor",
                go_stem="user_sensor",
                cpp_output="out/sensor.hpp",
                go_output="out/sensor.go",
                cmake_output="out/sensor.cmake",
                embedded_compatibility_version=None,
                embedded_macro="USER_SENSOR_PREVIOUS",
            )
            (root / "registration.json").write_text(json.dumps(r))
            reg, decl = g.load(root, root / "registration.json")
            g.emit(root, reg, decl)
            g.emit(root, reg, decl, True)
            other = copy.deepcopy(decl)
            other.update(module_id="other-sensor-engine", engine_id="other-sensor")
            before = {path: (root / path).read_bytes() for path in g.outputs(reg, decl)}
            with self.assertRaises(ValueError):
                g.emit(root, reg, other)
            self.assertEqual(
                before, {path: (root / path).read_bytes() for path in before}
            )
            (root / "out/sensor.go").write_text("caller-owned")
            with self.assertRaises(ValueError):
                g.emit(root, reg, decl)
            self.assertEqual((root / "out/sensor.go").read_text(), "caller-owned")

    def test_registration_rejections(self):
        r, d = g.load(R, a.registration)
        for name, mutate in [
            ("path", lambda x: x.update(cpp_output="../escape")),
            ("symbol", lambda x: x.update(go_title="bad;code")),
            ("collision", lambda x: x.update(cpp_output=x["declaration"])),
            ("history", lambda x: x.update(history_digest="sha256:" + "0" * 64)),
            ("embedded", lambda x: x.update(embedded_compatibility_version="latest")),
        ]:
            with self.subTest(name=name), tempfile.TemporaryDirectory() as td:
                bad = copy.deepcopy(r)
                mutate(bad)
                p = Path(td) / "reg.json"
                p.write_text(json.dumps(bad))
                with self.assertRaises(ValueError):
                    g.load(R, p)


if __name__ == "__main__":
    unittest.main(argv=["owner-test", *rest])
