from pathlib import Path
import argparse, copy, hashlib, importlib.util, json, shutil, subprocess, tempfile, unittest

H = Path(__file__).resolve().parents[1]
R = H.parents[1]
s = importlib.util.spec_from_file_location("generator", H / "generate.py")
g = importlib.util.module_from_spec(s)
s.loader.exec_module(g)
p = argparse.ArgumentParser()
p.add_argument("--owner", required=True)
p.add_argument("--engine", required=True)
a, rest = p.parse_known_args()
M = R / "modules" / a.owner


class InterfaceTests(unittest.TestCase):
    def test_frozen_native_parity(self):
        d = g.read(M / "OWNER-INTERFACE.json")
        hist = g.read(M / "tests/fixtures/interface-history.v1.json")["descriptors"]
        expected = copy.deepcopy(list(hist.values())[-1])
        expected["engine_version"] = d["current_version"]
        expected.pop("descriptor_digest")
        expected["descriptor_digest"] = (
            "sha256:"
            + hashlib.sha256(
                json.dumps(expected, sort_keys=True, separators=(",", ":")).encode()
            ).hexdigest()
        )
        self.assertEqual(
            json.loads(subprocess.check_output([a.engine, "--descriptor"])), expected
        )

    def test_deterministic_projection_and_drift(self):
        d = g.validate(g.read(M / "OWNER-INTERFACE.json"))
        self.assertEqual(g.outputs(d), g.outputs(copy.deepcopy(d)))
        g.emit(d, check=True)
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            g.emit(d, root)
            path = next(iter(g.outputs(d)))
            (root / path).write_text("changed")
            with self.assertRaises(AssertionError):
                g.emit(d, root, check=True)

    def test_declaration_rejections(self):
        original = g.read(M / "OWNER-INTERFACE.json")
        old = list(original["releases"])[0]
        mutations = {
            "extra": lambda d: d.update(extra=True),
            "missing": lambda d: d.pop("schemas"),
            "owner": lambda d: d.update(module_id="unknown"),
            "namespace": lambda d: d.update(namespace="foreign"),
            "release": lambda d: d["releases"].update(latest=[]),
            "old_admission": lambda d: d["releases"][old].append("unknown"),
            "duplicate": lambda d: d["operations"].append(d["operations"][0]),
            "mutation": lambda d: d["operations"][0].update(mutability="write"),
            "protocol": lambda d: d["operations"][0].update(output_protocol="invented"),
            "interaction": lambda d: d["operations"][0].update(
                administrative_interactions=["mutate"]
            ),
            "reader": lambda d: d["embedded_dependencies"].append(
                {"engine_id": "foreign", "version": "latest"}
            ),
            "escape": lambda d: d["companions"].append("../escape"),
            "duplicate_schema": lambda d: d["schemas"].append(d["schemas"][0]),
            "missing_schema": lambda d: d["schemas"].pop(),
        }
        for name, mutate in mutations.items():
            with self.subTest(name=name):
                d = copy.deepcopy(original)
                mutate(d)
                with self.assertRaises((AssertionError, KeyError)):
                    g.validate(d)

    def test_duplicate_keys_and_schema_refs(self):
        with tempfile.TemporaryDirectory() as td:
            p = Path(td) / "dup.json"
            p.write_text('{"a":1,"a":2}')
            with self.assertRaises(ValueError):
                g.read(p)
        d = g.read(M / "OWNER-INTERFACE.json")
        for ref in ["../../outside.json", "#/$defs/Absent"]:
            with self.subTest(ref=ref), tempfile.TemporaryDirectory() as td:
                root = Path(td)
                for path in [
                    "modules/" + a.owner + "/tests/fixtures/interface-history.v1.json",
                    *d["schemas"],
                    *d["companions"],
                ]:
                    p = root / path
                    p.parent.mkdir(parents=True, exist_ok=True)
                    shutil.copyfile(R / path, p)
                p = root / d["schemas"][0]
                v = g.read(p)
                v["$ref"] = ref
                p.write_text(json.dumps(v))
                with self.assertRaises((AssertionError, KeyError)):
                    g.validate(d, root)


if __name__ == "__main__":
    unittest.main(argv=["interface-test", *rest])
