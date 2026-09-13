#!/usr/bin/env python3
"""Build one relocated-source DO pack with independent finite conformance cases.

No network, installed package mutation, native seal implementation, or expectation
derived from extractor outputs. Original package/capture bytes remain untouched.
"""
import argparse
import copy
import hashlib
import json
import pathlib
import subprocess
import time

HERE = pathlib.Path(__file__).resolve().parent
AUTHOR = "Symphony increment 12 independently authored source expectation"


def read(path):
    return json.loads(path.read_text())


def digest(data):
    return "sha256:" + hashlib.sha256(data).hexdigest()


def write(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2, ensure_ascii=False) + "\n")


def claim(claim_id, subject, predicate, scope, value):
    return {"claim_id": claim_id, "subject": subject, "predicate": predicate,
            "scope": scope, "statement_kind": "documented_fact", "dependencies": [],
            "value": {"type": "boolean" if isinstance(value, bool) else "string",
                      "value": value, "unit": None}}


# Authored after reading the actual body and independent review. These are not
# loaded from a profile, historical expected-claims table, or native result.
VALKEY_SCOPE = {"provider": "do", "service": "app-platform",
                "component": "development-database", "engine": "valkey"}
LIMITS_EXPECTED = [
    claim("do-app-platform-amd64", "do.app-platform.container", "image.architecture",
          {"provider": "do", "service": "app-platform", "operating-system": "linux"}, "amd64"),
    claim("do-app-platform-local-storage", "do.app-platform.container", "local-storage.persistent",
          {"provider": "do", "service": "app-platform", "storage": "host-local-filesystem"}, False),
    claim("do-limits-dev-valkey", "do.app-platform.development-database", "engine.valkey.available",
          VALKEY_SCOPE, False),
]
LABS_EXPECTED = [claim("do-guide-dev-valkey", "do.app-platform.development-database",
                       "engine.valkey.available", VALKEY_SCOPE, True)]


def projected(claims):
    keys = ("claim_id", "subject", "predicate", "scope", "statement_kind", "value", "dependencies")
    return sorted(({k: c[k] for k in keys} for c in claims), key=lambda c: c["claim_id"])


def snapshot(old_package, fresh_capture):
    files = [old_package / "pack.json", old_package / "prepare-input.json", fresh_capture]
    return [{"path": str(p), "bytes": p.stat().st_size, "sha256": digest(p.read_bytes())} for p in files]


class Run:
    def __init__(self, args):
        self.args, self.out, self.commands = args, args.out.resolve(), []
        if self.out.exists():
            raise ValueError("--out must name a new immutable evidence directory")
        self.out.mkdir(parents=True)
        self.originals = snapshot(args.old_package, args.fresh_capture)
        write(self.out / "ORIGINALS_BEFORE.json", self.originals)

    def call(self, name, route, value, expected_error=False):
        inp = self.out / (name + "-input.json")
        write(inp, value)
        assert inp.stat().st_size < 1048576, "Do not expand the process input bound"
        argv = [str(self.args.qxctl), "scv", *route.split(), "--domain", "schv-do",
                "--version", "0.10.0-dev", "--prefix", str(self.args.prefix),
                "--input", str(inp), "--json"]
        started = time.time()
        r = subprocess.run(argv, capture_output=True, timeout=40)
        stdout, stderr = self.out / (name + "-stdout.json"), self.out / (name + "-stderr.txt")
        stdout.write_bytes(r.stdout)
        stderr.write_bytes(r.stderr)
        self.commands.append({"name": name, "argv": argv, "exit_code": r.returncode,
                              "elapsed_seconds": round(time.time() - started, 6),
                              "expected_error": expected_error, "input_file": inp.name,
                              "stdout_file": stdout.name, "stderr_file": stderr.name,
                              "input_bytes": inp.stat().st_size, "stdout_bytes": len(r.stdout),
                              "input_sha256": digest(inp.read_bytes()), "stdout_sha256": digest(r.stdout),
                              "stderr_sha256": digest(r.stderr)})
        write(self.out / "COMMANDS.json", self.commands)
        if expected_error:
            assert r.returncode != 0, "Fixture substitution must not be accepted"
            value = json.loads(r.stdout)
            # qxctl intentionally exposes the stable owner error code, not the
            # native diagnostic prose. The exact substituted input is retained.
            assert value["protocol"] == "symphony.qxctl.error.v1"
            assert value["error"]["code"] == "engine_rejected"
            assert value["error"]["engine_code"] == "pack.invalid"
            return value
        assert r.returncode == 0, (name, r.stdout[:2000], r.stderr[:2000])
        return json.loads(r.stdout)

    def fixture(self, fixture_id, captures, selection_policy, expectations, rationale):
        bindings, extractions = [], []
        for cap, expected in zip(captures, expectations):
            profile_id = "coverage-" + cap["source"]["source_id"]
            bindings.append({"profile_id": profile_id, "capture_digest": cap["digest"]})
            extractions.extend({"profile_id": profile_id, "capture_digest": cap["digest"],
                                "rule_id": c["claim_id"], "claim_id": c["claim_id"],
                                "status": "matched", "reasons": []} for c in expected)
        data = {"fixture_id": fixture_id, "captures": captures, "bindings": bindings,
                "selection_policy": selection_policy}
        manifest = {"fixture_id": fixture_id, "label": fixture_id, "authored_by": AUTHOR,
                    "rationale": rationale, "input_digest": None,
                    "expected_claims": [c for rows in expectations for c in rows],
                    "expected_extractions": extractions}
        return manifest, data

    def execute(self):
        old_prepare = read(self.args.old_package / "prepare-input.json")
        old_pack = read(self.args.old_package / "pack.json")
        captures = {}
        for fixture in old_prepare["fixtures"]:
            for cap in fixture["captures"]:
                source_id = cap["source"]["source_id"]
                assert source_id not in captures or captures[source_id] == cap, "Ambiguous old source capture"
                captures[source_id] = cap
        old_capture = captures["do-app-platform-limits"]
        fresh = read(self.args.fresh_capture)
        labs = captures["do-app-platform-designer"]
        all_fixture = next(f for f in old_prepare["fixtures"] if f["fixture_id"] == "do-all-retained-source-semantics")
        policy = copy.deepcopy(all_fixture["selection_policy"])
        assert fresh["source"]["generation"] == 2 and fresh["source"]["predecessor_digest"] == old_capture["source"]["digest"]
        assert fresh["body"] == old_capture["body"] and fresh["body_digest"] == old_capture["body_digest"]
        assert fresh["completeness"] == "complete" and fresh["byte_size"] == 16774
        assert fresh["source"]["locators"][0]["uri"].endswith("/limits/index.html.md")
        oracle = {"authored_by": AUTHOR, "method": "Manually authored typed meanings after reading exact source bodies; independent reviewer agrees. No native/profile values used as an oracle.",
                  "limits_capture_digest": fresh["digest"], "limits_expected_claims": LIMITS_EXPECTED,
                  "labs_capture_digest": labs["digest"], "labs_expected_claims": LABS_EXPECTED,
                  "qualifications": ["Only App Platform Linux image architecture; not all DigitalOcean compute.",
                                     "Host-local filesystem is ephemeral; Spaces/managed databases are separate persistence options.",
                                     "Development PostgreSQL-only is separate from managed Valkey availability.",
                                     "Historical Labs table says Valkey development DB yes; opposing source facts remain separate.",
                                     "One-day policy preserved; fixture pass does not freshen historical Labs evidence or evaluate a graph at current time."]}
        write(self.out / "SEMANTIC_EXPECTATIONS.json", oracle)
        cases = [
            ("do-all-retained-source-semantics", [labs, fresh], [LABS_EXPECTED, LIMITS_EXPECTED], "New limits declaration/capture with unchanged historical Labs evidence; contradictory development-Valkey facts remain explicit."),
            ("do-product-limits-only", [fresh], [LIMITS_EXPECTED], "Source generation 2 with unchanged three-rule mapping; host-local persistence, Linux architecture and development/managed database distinction stay scoped."),
            ("do-labs-guide-only", [labs], [LABS_EXPECTED], "Unchanged exact historical Labs statement; a fixture pass does not renew its observation or reconcile it with product limits."),
        ]
        manifests, detached = [], []
        for name, captures, expected, rationale in cases:
            manifest, data = self.fixture(name, captures, copy.deepcopy(policy), expected, rationale)
            manifests.append(manifest)
            detached.append(data)
            write(self.out / "package/fixtures" / (name + ".json"), data)
        draft = copy.deepcopy(old_prepare["pack"])
        draft["pack_version"] = "2026-09-13.1"
        draft["authored_by"] = "Symphony increment 12 source-maintenance package author"
        draft["provenance"] = ["Derived separately from historical pack " + old_pack["digest"] + "; originals preserved.",
                               "One fresh candidate capture of the already-established official Markdown destination; body unchanged. Protected source adoption is separate.",
                               "Limits source generation 2; Labs declaration and historical capture unchanged; no provider/API upgrade.",
                               "Exact sealed profile version 1 objects unchanged. Fixture inputs match their selected source declarations.",
                               "Independent expectation file " + digest((self.out / "SEMANTIC_EXPECTATIONS.json").read_bytes()),
                               "Authored provenance and conformance do not authenticate publishers, refresh historical evidence or prove runtime compatibility."]
        desired = {k: v for k, v in fresh["source"].items() if k not in ("protocol", "digest", "generation", "predecessor_digest")}
        found = False
        for i, source in enumerate(draft["provider"]["sources"]):
            if source["source_id"] == fresh["source"]["source_id"]:
                draft["provider"]["sources"][i], found = desired, True
        assert found
        draft["fixtures"] = manifests
        assert draft["profiles"] == old_pack["profiles"]
        prepare = {"pack": draft, "fixtures": detached}
        write(self.out / "package/prepare-input.json", prepare)
        pack = self.call("prepare", "provider pack prepare", prepare)
        write(self.out / "package/pack.json", pack)
        selection = {k: v for k, v in detached[0].items() if k != "fixture_id"}
        selection["fixtures"] = detached
        write(self.out / "package/evaluate-selection.json", selection)
        evaluation_input = {"pack": pack, **selection}
        write(self.out / "package/evaluate-input.json", evaluation_input)
        evaluation = self.call("evaluate", "provider pack evaluate", evaluation_input)
        write(self.out / "package/evaluation.json", evaluation)
        assert evaluation["conformance"] == {"passed": 3, "failed": 0, "not_run": 0}
        assert projected(evaluation["knowledge"]["claims"]) == projected(LABS_EXPECTED + LIMITS_EXPECTED)
        assert all(e["status"] == "matched" and e["reasons"] == [] for e in evaluation["extractions"])
        old_fixture = next(f for f in old_prepare["fixtures"] if f["fixture_id"] == "do-product-limits-only")
        substituted = {"pack": pack, **{k: v for k, v in selection.items() if k != "fixtures"}, "fixtures": [old_fixture]}
        rejection = self.call("reject-old-fixture-identity", "provider pack evaluate", substituted, expected_error=True)
        # A separately labeled diagnostic binds the old fixture honestly, rather
        # than falsifying its digest, so the source-declaration failure is visible.
        negative = copy.deepcopy(draft)
        negative["pack_version"] = "2026-09-13.1-negative-old-source"
        bad_manifest, bad_fixture = self.fixture("do-old-generation1-fixture-diagnostic", [old_capture], copy.deepcopy(policy), [LIMITS_EXPECTED], "Deliberately mismatched generation-1 desired source: expected facts remain unchanged; native conformance must fail with source_declaration_differs.")
        negative["fixtures"] = [bad_manifest]
        negative_pack = self.call("diagnostic-prepare", "provider pack prepare", {"pack": negative, "fixtures": [bad_fixture]})
        negative_input = {"pack": negative_pack, **{k: v for k, v in selection.items() if k != "fixtures"}, "fixtures": [bad_fixture]}
        failed = self.call("diagnostic-evaluate", "provider pack evaluate", negative_input)
        assert failed["conformance"] == {"passed": 0, "failed": 1, "not_run": 0}
        case = failed["fixture_results"][0]
        assert case["actual_claims"] == [] and len(case["actual_extractions"]) == 3
        assert all(e["status"] == "unresolved" and e["reasons"] == ["source_declaration_differs"] for e in case["actual_extractions"])
        assert case["difference_axes"] == {"claims": True, "extractions": True}
        for profile in old_pack["profiles"]:
            write(self.out / "package/profiles" / (profile["profile_id"] + ".json"), profile)
        write(self.out / "package/captures/do-app-platform-limits.json", fresh)
        write(self.out / "package/captures/do-app-platform-designer.json", labs)
        assert snapshot(self.args.old_package, self.args.fresh_capture) == self.originals, "Original evidence changed"
        write(self.out / "ORIGINALS_AFTER.json", snapshot(self.args.old_package, self.args.fresh_capture))
        summary = {"status": "passed", "command_count": len(self.commands), "successful_commands": 4,
                   "expected_command_rejections": 1, "cli": str(self.args.qxctl), "cli_sha256": digest(self.args.qxctl.read_bytes()),
                   "prefix": str(self.args.prefix), "domain": "schv-do", "engine_version": "0.10.0-dev",
                   "pack_id": pack["pack_id"], "pack_version": pack["pack_version"], "pack_digest": pack["digest"],
                   "evaluation_digest": evaluation["digest"], "conformance": evaluation["conformance"],
                   "production_matched_rules": len(evaluation["extractions"]), "limits_matched_rules": 3,
                   "profile_digests": [p["digest"] for p in pack["profiles"]],
                   "source_generation": 2, "source_digest": fresh["source"]["digest"], "capture_digest": fresh["digest"],
                   "body_digest": fresh["body_digest"], "body_bytes": fresh["byte_size"],
                   "old_fixture_identity_rejected": True, "diagnostic_conformance": failed["conformance"],
                   "diagnostic_generation1_unresolved_rules": 3, "diagnostic_result_digest": failed["digest"],
                   "original_files_preserved": len(self.originals), "network_acquisitions": 0,
                   "maximum_input_file_bytes": max(c["input_bytes"] for c in self.commands),
                   "maximum_output_file_bytes": max(c["stdout_bytes"] for c in self.commands),
                   "scope": "Mapping/conformance only. No source-head adoption or graph freshness/runtime compatibility claim."}
        write(self.out / "SUMMARY.json", summary)
        print(json.dumps(summary, indent=2))


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--qxctl", type=pathlib.Path, required=True)
    parser.add_argument("--prefix", type=pathlib.Path, required=True)
    parser.add_argument("--old-package", type=pathlib.Path, required=True, help="Directory containing immutable pack.json and prepare-input.json")
    parser.add_argument("--fresh-capture", type=pathlib.Path, required=True, help="Exact retained complete generation-2 DO limits capture JSON")
    parser.add_argument("--out", type=pathlib.Path, required=True)
    args = parser.parse_args()
    args.qxctl, args.prefix = args.qxctl.resolve(), args.prefix.resolve()
    args.old_package, args.fresh_capture = args.old_package.resolve(), args.fresh_capture.resolve()
    Run(args).execute()
