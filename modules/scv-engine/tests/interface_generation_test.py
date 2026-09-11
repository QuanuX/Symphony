#!/usr/bin/env python3
"""Independent historical/interface-authoring regression checks."""
import copy
import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

sys.dont_write_bytecode = True
ROOT = Path(__file__).resolve().parents[3]
SPEC = importlib.util.spec_from_file_location("scv_interface_generator", ROOT / "modules/scv-engine/tools/generate_interface.py")
gen = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(gen)


class InterfaceGeneration(unittest.TestCase):
    def setUp(self):
        self.manifest = gen.read_json(ROOT / gen.MANIFEST)
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.write(gen.MANIFEST, self.manifest)
        self.write(gen.HISTORY, gen.read_json(ROOT / gen.HISTORY))
        self.catalog = {"protocol": "symphony.scv.schema-catalog.v1", "engine_version": self.manifest["current_release"], "entries": []}
        definitions = {}
        for op in self.manifest["operations"]:
            for field, kind in (("input_protocol", "input"), ("output_protocol", "output")):
                protocol = op[field]
                definitions[protocol] = {"type": "object"}
                self.catalog["entries"].append({"protocol": protocol, "kind": kind, "operations": [op["name"]], "file": "fixture.schema.json", "fragment": "#/$defs/" + protocol})
        self.write(self.manifest["schema_catalog"], self.catalog)
        self.write("knowledge/scv/schemas/v1/fixture.schema.json", {"$defs": definitions})

    def write(self, name, value):
        path = self.root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps(value))

    def test_historical_projection_is_frozen(self):
        gen.check_history(self.manifest, ROOT)
        self.assertEqual([len(gen.projection(self.manifest, f"0.{n}.0-dev")["operations"]) for n in range(1, 7)], [13, 17, 20, 21, 22, 26])
        self.manifest["operations"][1]["output_protocol"] = "symphony.scv.changed.v1"
        with self.assertRaisesRegex(gen.Invalid, "frozen interface changed"):
            gen.check_history(self.manifest, ROOT)

    def test_future_domain_does_not_change_old_admission_or_definition(self):
        original = copy.deepcopy(self.manifest)
        next_release = copy.deepcopy(self.manifest["releases"][-1])
        next_release["version"] = "0.7.0-dev"
        self.manifest["releases"].append(next_release)
        self.manifest["current_release"] = "0.7.0-dev"
        self.manifest["domains"].append({"name": "scev-new-provider", "introduced_in": "0.7.0-dev"})
        gen.validate(self.manifest)
        for release in original["releases"]:
            version = release["version"]
            self.assertEqual(gen.projection(original, version), gen.projection(self.manifest, version))
            self.assertEqual(gen.definition_digest(gen.release_manifest(original, version)), gen.definition_digest(gen.release_manifest(self.manifest, version)))
        self.assertIn("scev-new-provider", gen.projection(self.manifest, "0.7.0-dev")["domains"])

    def test_complete_generation_is_deterministic(self):
        first = gen.render(self.manifest, self.root)
        self.assertEqual(first, gen.render(self.manifest, self.root))
        self.assertEqual(set(first), {gen.CPP_OUTPUT, gen.GO_OUTPUT, gen.CMAKE_OUTPUT})
        self.assertIn(b"OWNER-INTERFACE.json", first[gen.CMAKE_OUTPUT])
        self.assertIn(b"fixture.schema.json", first[gen.CMAKE_OUTPUT])

    def test_check_detects_output_drift(self):
        command = [sys.executable, str(ROOT / "modules/scv-engine/tools/generate_interface.py"), "--root", str(self.root)]
        self.assertEqual(subprocess.run(command, capture_output=True).returncode, 0)
        self.assertEqual(subprocess.run(command + ["--check"], capture_output=True).returncode, 0)
        path = self.root / gen.GO_OUTPUT
        path.write_bytes(path.read_bytes() + b"// drift\n")
        result = subprocess.run(command + ["--check"], capture_output=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn(b"generated interface drift", result.stderr)

    def test_malformed_manifest_rejected(self):
        mutations = [
            lambda m: m.update(unknown=True),
            lambda m: m.update(current_release="0.7.0-dev"),
            lambda m: m["domains"].append(copy.deepcopy(m["domains"][0])),
            lambda m: m["domains"][0].update(introduced_in="unknown"),
            lambda m: m["operations"].append(copy.deepcopy(m["operations"][0])),
            lambda m: m["operations"][0].update(handler="run_shell"),
            lambda m: m["operations"][0].update(mutability="unrestricted"),
            lambda m: m["operations"][0].update(expected_state=1),
            lambda m: m["operations"][0].update(interactions=[{}]),
            lambda m: m["operations"][-1]["artifact"].update(introduced_in="0.1.0-dev"),
            lambda m: m["releases"][0].update(companions=[{}]),
        ]
        for mutation in mutations:
            with self.subTest(mutation=mutations.index(mutation)):
                value = copy.deepcopy(self.manifest)
                mutation(value)
                with self.assertRaises(gen.Invalid):
                    gen.validate(value)

    def test_malformed_catalog_and_references_rejected(self):
        mutations = [
            lambda c: c.update(engine_version="0.5.0-dev"),
            lambda c: c["entries"].pop(0),
            lambda c: c["entries"].append(copy.deepcopy(c["entries"][0])),
            lambda c: c["entries"].append(4),
            lambda c: c["entries"][0].update(kind="input" if c["entries"][0]["kind"] == "output" else "output"),
            lambda c: c["entries"][0].update(file="../escape.schema.json"),
            lambda c: c["entries"][0].update(fragment="#/$defs/missing"),
        ]
        for i, mutation in enumerate(mutations):
            with self.subTest(mutation=i):
                value = copy.deepcopy(self.catalog)
                mutation(value)
                self.write(self.manifest["schema_catalog"], value)
                with self.assertRaises(gen.Invalid):
                    gen.schema_inventory(self.manifest, self.root)
        self.write(self.manifest["schema_catalog"], self.catalog)
        path = "knowledge/scv/schemas/v1/fixture.schema.json"
        doc = gen.read_json(self.root / path)
        for ref in ("https://example.invalid/schema", "../escape.schema.json", "#/$defs/missing", "#/$defs/~2invalid"):
            with self.subTest(reference=ref):
                value = copy.deepcopy(doc)
                value["$ref"] = ref
                self.write(path, value)
                with self.assertRaises(gen.Invalid):
                    gen.schema_inventory(self.manifest, self.root)

    def test_duplicate_json_keys_rejected(self):
        path = self.root / "duplicate.json"
        path.write_text('{"key":1,"key":2}')
        with self.assertRaises(gen.Invalid):
            gen.read_json(path)

    def test_checked_in_metadata_has_no_drift(self):
        for name, expected in gen.render(self.manifest, ROOT, metadata_only=True).items():
            self.assertEqual((ROOT / name).read_bytes(), expected, name)

    def test_artifact_schema_admissions_match_owner_interface(self):
        definitions = gen.read_json(ROOT / "knowledge/scv/schemas/v1/scv-artifact.schema.json")["$defs"]
        artifacts = [op for op in self.manifest["operations"] if op["artifact"]]
        operations = [op["name"] for op in artifacts]
        kinds = [op["artifact"]["kind"] for op in artifacts]
        self.assertEqual(definitions["ImportInput"]["properties"]["operation"]["enum"], operations)
        self.assertEqual(definitions["Record"]["properties"]["operation"]["enum"], operations)
        self.assertEqual(definitions["Record"]["properties"]["kind"]["enum"], kinds)
        catalog = gen.read_json(ROOT / self.manifest["schema_catalog"])
        entries = {entry["protocol"]: entry for entry in catalog["entries"]}
        versions = [release["version"] for release in self.manifest["releases"]]
        def reference(protocol):
            entry = entries[protocol]
            return entry["file"] + entry["fragment"]
        self.assertEqual(len(definitions["Record"]["allOf"]), len(artifacts))
        self.assertEqual(len(definitions["ImportInput"]["allOf"]), len(artifacts))
        for op, imported, recorded in zip(artifacts, definitions["ImportInput"]["allOf"], definitions["Record"]["allOf"]):
            for branch in (imported, recorded):
                self.assertEqual(branch["if"]["properties"]["operation"]["const"], op["name"])
                self.assertEqual(branch["then"]["properties"]["input"], {"$ref": reference(op["input_protocol"])})
            self.assertEqual(imported["then"]["properties"]["result"], {"anyOf": [{"type": "null"}, {"$ref": reference(op["output_protocol"])}]})
            properties = recorded["then"]["properties"]
            self.assertEqual(properties["artifact"], {"$ref": reference(op["output_protocol"])})
            self.assertEqual(properties["kind"]["const"], op["artifact"]["kind"])
            admitted = [version for version in versions if gen.projection(self.manifest, version)["operations"] and any(x["name"] == op["name"] and x["artifact_kind"] for x in gen.projection(self.manifest, version)["operations"])]
            self.assertEqual(properties["installation"]["properties"]["Version"]["enum"], admitted)

    def test_checked_in_complete_inventory_has_no_drift(self):
        # This is deliberately a full check, not the incomplete bootstrap mode.
        for name, expected in gen.render(self.manifest, ROOT).items():
            self.assertEqual((ROOT / name).read_bytes(), expected, name)


if __name__ == "__main__":
    unittest.main()
