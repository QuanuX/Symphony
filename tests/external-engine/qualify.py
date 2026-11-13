#!/usr/bin/env python3
"""Offline qualification example; no product Python dependency or live provider."""
import argparse
import collections
import copy
import fractions
import hashlib
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import time

ID = "beta-empirical-price-example"
PRICES = [100, 95, 105, 96, 104, 97, 103, 98, 102, 99, 101, 100]
START = 1609160400000000000

def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode()

def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def fraction(value):
    return {"numerator": str(value.numerator), "denominator": str(value.denominator)}

def oracle(prices, version):
    counts = collections.Counter(prices)
    return {"protocol": "quanux.beta.empirical-price-result.v1", "engine_id": ID,
            "engine_version": version, "sample_count": str(len(prices)),
            "support": [{"price_nanos": str(p), "weight": fraction(fractions.Fraction(n, len(prices)))}
                        for p, n in sorted(counts.items())],
            "mean_price_nanos": fraction(fractions.Fraction(sum(prices), len(prices)))}

def request(operation, payload, target="symphony-sbv"):
    return {"protocol": "symphony.knowledge.engine-process.v2", "request_id": "beta-" + operation,
            "correlation_id": "beta-qualification", "operation": operation, "target_engine": target,
            "deadline_unix_ms": None, "payload": payload}

def unavailable(reason):
    return {"status": "unavailable", "reason": reason, "value": None}

def outcome(expected, index, version, provider_sha):
    ordinal = 3 + index * 2
    return {"protocol": "symphony.sbv.model-outcome.v1", "signal_id": "signal-" + str(index),
            "model_id": "native_provider", "model_version": "1", "status": "available", "reason": "",
            "support": expected["support"], "measure": "probability", "conditioning": "scenario",
            "calibration": "not_verified", "calibration_reference": "", "evidence_origin": "native_provider",
            "evidence_reference": "sha256:" + hashlib.sha256(canonical(expected)).hexdigest(),
            "producer": {"id": ID, "version": version, "artifact_sha256": provider_sha,
                         "reproducibility": "deterministic_declared"},
            "horizon_end_ns": str(START + ordinal * 100 + 200), "horizon_within_observed_span": True,
            "price_scale": "1e-9", "price_unit": "price_nanos",
            "fill_probability": unavailable("Empirical prices do not estimate execution probability"),
            "no_fill_probability": unavailable("execution probability unavailable"),
            "fill_quantity": unavailable("not modeled by this contract"),
            "fill_time": unavailable("not modeled by this contract"),
            "actual_execution": unavailable("no actual execution supplied")}

class Campaign:
    def __init__(self, args):
        self.source = Path(__file__).resolve().parent
        self.sdk = Path(args.sdk_prefix).resolve(strict=True)
        self.work = Path(args.work)
        if not self.work.is_absolute() or self.work.exists() or self.work.is_symlink():
            raise ValueError("new absolute work directory required")
        self.work = self.work.parent.resolve(strict=True) / self.work.name
        if self.work.is_relative_to(self.sdk) or self.work.is_relative_to(self.source):
            raise ValueError("work must be outside SDK and source")
        self.work.mkdir(mode=0o700)
        for name in ["logs", "outputs", "home", "tmp", "example-install"]:
            (self.work / name).mkdir(mode=0o700)
        self.env = dict(os.environ, HOME=str(self.work / "home"), TMPDIR=str(self.work / "tmp"))
        for key in ["CPATH", "CPLUS_INCLUDE_PATH", "C_INCLUDE_PATH", "LIBRARY_PATH", "CMAKE_PREFIX_PATH",
                    "CMAKE_TOOLCHAIN_FILE", "DYLD_LIBRARY_PATH", "DYLD_FALLBACK_LIBRARY_PATH", "LD_LIBRARY_PATH"]:
            self.env.pop(key, None)
        self.cxx = Path(args.cxx).absolute()
        # Preserve clang++ argv[0]: resolving its symlink to clang loses the
        # C++ driver mode and omits the standard C++ runtime at link time.
        if not self.cxx.is_file() or not os.access(self.cxx, os.X_OK):
            raise ValueError("executable C++ driver required")
        self.checks = []
        self.serial = 0
        self.sdk_before = self.verify_sdk()
        self.project = self.work / "source"
        self.project.mkdir(mode=0o700)
        self.source_files = ["CMakeLists.txt", "calculation.hpp", "engine.cpp", "provider.cpp", "sdk-client.cpp",
                             "provider-probe.cpp", "fixture.cpp", "qualify.py", "README.md",
                             "fixtures/public_fixture.hpp", "fixtures/ATTRIBUTION.md", "fixtures/LICENSE.Apache-2.0"]
        for name in self.source_files:
            out = self.project / name
            out.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(self.source / name, out)
        self.sentinel = self.work / "example-install/user-research.txt"
        self.sentinel.write_text("User-owned unrelated research; preserve unchanged.\n")
        self.sentinel_sha = digest(self.sentinel)

    def save(self, path, value):
        with path.open("xb") as output:
            output.write(value if isinstance(value, bytes) else json.dumps(value, indent=2).encode() + b"\n")

    def check(self, condition, label):
        if not condition:
            raise AssertionError(label)
        self.checks.append(label)

    def run(self, label, command, value=None, success=True):
        self.serial += 1
        stem = self.work / "logs" / f"{self.serial:03d}-{label}"
        if value is not None:
            self.save(Path(str(stem) + ".input.json"), value)
        completed = subprocess.run([str(v) for v in command], input=None if value is None else canonical(value),
                                   capture_output=True, env=self.env, cwd=self.work, timeout=600)
        self.save(Path(str(stem) + ".stdout"), completed.stdout)
        self.save(Path(str(stem) + ".stderr"), completed.stderr)
        self.check((completed.returncode == 0) == success, label + ": expected process status")
        return json.loads(completed.stdout) if value is not None and success else completed

    def verify_sdk(self):
        receipts = {}
        for component, version in [("knowledge-vector-engine-cpp", "0.2.0-dev"), ("sbv-engine", "0.33.0-dev")]:
            path = self.sdk / "share/symphony/receipts" / component / version / "install-receipt.json"
            if path.is_symlink() or not path.is_file():
                raise ValueError("complete public receipt required")
            data = json.loads(path.read_text())
            expected = data.pop("receipt_digest")
            if expected != "sha256:" + hashlib.sha256(canonical(data)).hexdigest():
                raise ValueError("receipt digest mismatch")
            if data["component_id"] != component or data["version"] != version:
                raise ValueError("exact SDK identity required")
            for entry in data["files"]:
                relative = Path(entry["path"])
                if relative.is_absolute() or ".." in relative.parts:
                    raise ValueError("unsafe SDK receipt path")
                owned = self.sdk / relative
                for parent in [owned, *owned.parents]:
                    if parent == self.sdk:
                        break
                    if parent.is_symlink():
                        raise ValueError("symlinked SDK receipt member")
                if not owned.is_file() or owned.stat().st_size != entry["size"] or "sha256:" + digest(owned) != entry["digest"]:
                    raise ValueError("SDK owned byte mismatch")
            receipts[str(path.relative_to(self.sdk))] = digest(path)
        return receipts

    def build(self, version):
        build = self.work / ("build-" + version)
        self.run("configure-" + version, ["cmake", "-S", self.project, "-B", build,
                    "-DCMAKE_CXX_COMPILER=" + str(self.cxx), "-DCMAKE_BUILD_TYPE=Release",
                    "-DBETA_SDK_PREFIX=" + str(self.sdk), "-DBETA_EXAMPLE_VERSION=" + version,
                    "-DCMAKE_INSTALL_PREFIX=" + str(self.work / "example-install")])
        self.run("build-" + version, ["cmake", "--build", build, "--parallel", "2"])
        self.run("install-" + version, ["cmake", "--install", build])
        commands = json.loads((build / "compile_commands.json").read_text())
        self.check(all(Path(row["file"]).is_relative_to(self.project) for row in commands), "independent copied source only: " + version)
        # NO_DEFAULT_PATH imports and actual generated dependency/link records
        # remain attached. The final fresh-host run must also lack the checkout.
        for path in list(build.rglob("*.o.d")) + list(build.rglob("link.txt")):
            text = path.read_text()
            forbidden = str(self.source).replace(" ", "\\ ")
            self.check(str(self.source) not in text and forbidden not in text, "no original-source dependency: " + str(path.relative_to(build)))
        installed = [Path(p) for p in (build / "install_manifest.txt").read_text().splitlines()]
        self.check(len(installed) == 2 and all(p.is_relative_to(self.work / "example-install") for p in installed), "independent exact example install inventory: " + version)
        inventory = {str(p): digest(p) for p in installed}
        self.save(self.work / "outputs" / ("installed-" + version + ".json"), inventory)
        self.run("abi-probe-" + version, [build / "beta-provider-probe", build / "beta-empirical-provider.so"])
        return build, inventory

    def direct(self, binary, prices, version, label):
        payload = {"protocol": "quanux.beta.empirical-price-input.v1", "prices_nanos": [str(p) for p in prices]}
        response = self.run(label, [binary], request("empirical_distribution", payload, ID))
        expected = oracle(prices, version)
        self.check(response["result"] == expected and response["outcome"] == "ok", label + ": complete exact oracle")
        self.save(self.work / "outputs" / (label + "-oracle.json"), expected)
        return response

    def remove(self, inventory):
        # Example-only removal: preflight every owned byte before any deletion.
        for path, sha in inventory.items():
            p = Path(path)
            if p.is_symlink() or not p.is_file() or digest(p) != sha:
                raise ValueError("example removal refused changed owned bytes")
        for path in inventory:
            Path(path).unlink()

    def execute(self):
        old, old_files = self.build("0.1.0")
        new, new_files = self.build("0.1.1")
        for version, build in [("0.1.0", old), ("0.1.1", new)]:
            for i, prices in enumerate([[100, 95, 105, 95], [-3, -3, 2, 4], [10**15] * 64, [7]]):
                self.direct(build / "beta-empirical-price", prices, version, version + "-direct-" + str(i))
        self.run("missing-sdk", ["cmake", "-S", self.project, "-B", self.work / "missing-sdk",
                 "-DBETA_SDK_PREFIX=" + str(self.work / "home")], success=False)
        wrong_sdk = self.work / "wrong-sdk-copy"
        shutil.copytree(self.sdk, wrong_sdk)
        cfg = wrong_sdk / "lib/cmake/SymphonySbvProvider/0.33.0-dev/SymphonySbvProviderConfig.cmake"
        cfg.write_text(cfg.read_text().replace('set(SymphonySbvProvider_VERSION "0.33.0-dev")', 'set(SymphonySbvProvider_VERSION "0.32.0-dev")'))
        self.run("wrong-sdk", ["cmake", "-S", self.project, "-B", self.work / "wrong-sdk-build",
                 "-DBETA_SDK_PREFIX=" + str(wrong_sdk)], success=False)
        binary = old / "beta-empirical-price"
        valid = request("empirical_distribution", {"protocol": "quanux.beta.empirical-price-input.v1", "prices_nanos": ["1"]}, ID)
        for label, prices in [("empty", []), ("oversize", ["1"] * 65), ("overflow", [str(10**15 + 1)]),
                              ("noncanonical", ["01"]), ("numeric", [1])]:
            bad = copy.deepcopy(valid); bad["payload"]["prices_nanos"] = prices
            self.run("direct-refuse-" + label, [binary], bad, success=False)
        for label, change in [("schema", lambda r: r["payload"].update(protocol="other")),
                              ("extra", lambda r: r["payload"].update(extra=True)),
                              ("target", lambda r: r.update(target_engine="other")),
                              ("deadline", lambda r: r.update(deadline_unix_ms="1"))]:
            bad = copy.deepcopy(valid); change(bad); self.run("direct-refuse-" + label, [binary], bad, success=False)
        stalled = subprocess.Popen([str(binary)], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                   cwd=self.work, env=self.env)
        time.sleep(0.05)
        self.check(stalled.poll() is None, "bounded interruption reached waiting process")
        stalled.send_signal(signal.SIGTERM); out, err = stalled.communicate(timeout=5)
        self.save(self.work / "outputs/interruption.stdout", out)
        self.check(stalled.returncode != 0 and not out, "interrupted input does not fabricate completion")
        source = self.work / "outputs/synthetic.dbn"
        self.run("synthetic-dbn", [old / "beta-fixture", source])
        templates = json.loads((self.sdk / "share/symphony/schemas/sbv-engine/0.33.0-dev/admin.templates.json").read_text())["templates"]
        run = copy.deepcopy(templates["run"])
        run.update(source_path=str(source), source_sha256=digest(source), output_path=str(self.work / "outputs/census.json"),
                   criteria={"rule": "spaced_trades", "spacing_ns": "200", "min_trade_size": "1", "direction": "any", "max_signals": "4"},
                   execution={"model": "none", "horizon_ns": "200", "price_offsets_nanos": [], "probability_numerator": "0", "probability_denominator": "1"},
                   replay={"before_ns": "1000", "after_ns": "200", "retain_events": True}, studies=["signal_summary"], workers="1")
        self.run("sdk-census", [old / "beta-sdk-client"], request("run", run))
        census = json.loads(Path(run["output_path"]).read_text())
        self.check([s["source_ordinal"] for s in census["sections"]["signals"]["data"]] == ["3", "5", "7", "9"], "independent census order oracle")
        for version, build in [("0.1.0", old), ("0.1.1", new)]:
            library = build / "beta-empirical-provider.so"
            provider = {"protocol": "symphony.sbv.native-provider-selection.v1", "library": {"path": str(library), "expected_sha256": digest(library)},
                        "id": ID, "version": version, "role": "model", "input_profile": "symphony.sbv.provider-databento-mbo-event.v1",
                        "concurrency": "serialized_instance", "parameters": {"version": "1", "window": "4"},
                        "dependencies": {"capture": "uncaptured", "description": "Exact installed public SDK example and platform runtime", "artifacts": []}, "extensions": {}}
            evaluation = copy.deepcopy(templates["evaluate"])
            evaluation.update(source_path=str(source), source_sha256=digest(source), output_path=str(self.work / "outputs" / ("evaluation-" + version + ".json")),
                              census={"path": run["output_path"], "expected_sha256": census["content_sha256"], "pointer": ""},
                              model={"protocol": "symphony.sbv.model-selection.v1", "id": "native_provider", "version": "1", "horizon_ns": "200",
                                     "parameters": {"provider": provider, "measure": "probability", "conditioning": "scenario", "calibration_reference": "", "on_unavailable": "reject"}},
                              replay=run["replay"], studies=["signal_summary", "model_summary"], workers="1")
            self.run("sdk-evaluation-" + version, [build / "beta-sdk-client"], request("evaluate", evaluation))
            result = json.loads(Path(evaluation["output_path"]).read_text())
            expected = []
            for i in range(4):
                through = i * 2 + 1
                prices = [p * 10**9 for p in PRICES[max(0, through - 4):through]]
                computed = oracle(prices, version)
                self.direct(build / "beta-empirical-price", prices, version, version + "-causal-" + str(i))
                expected.append(outcome(computed, i, version, digest(library)))
            self.save(self.work / "outputs" / ("full-execution-oracle-" + version + ".json"), expected)
            self.check(result["sections"]["execution"]["data"] == expected, "complete native-provider outcome oracle: " + version)
            inspected = self.run("complete-inspect-" + version, [build / "beta-sdk-client"], request("result_inspect", {"protocol": "symphony.sbv.result-inspect-input.v1", "path": evaluation["output_path"], "expected_sha256": result["content_sha256"]}))["result"]
            exported = Path(evaluation["output_path"]).read_bytes()
            self.check(inspected["expected_digest_verified"] and inspected["file_sha256"] == hashlib.sha256(exported).hexdigest() and inspected["bytes"] == str(len(exported)) and json.loads(exported) == result, "complete native-validated result read: " + version)
            before = digest(Path(evaluation["output_path"]))
            # Treat the first committed response as lost; inspect the artifact and
            # deliberately prove same-output retry refuses rather than replaces.
            self.run("lost-reply-no-overwrite-" + version, [build / "beta-sdk-client"], request("evaluate", evaluation), success=False)
            self.check(digest(Path(evaluation["output_path"])) == before, "lost-reply retained bytes unchanged: " + version)
            for label in ["library-digest", "identity", "config-version", "source-digest", "deadline"]:
                bad = copy.deepcopy(evaluation); output = self.work / "outputs" / (version + "-refuse-" + label + ".json"); bad["output_path"] = str(output)
                selected = bad["model"]["parameters"]["provider"]
                if label == "library-digest": selected["library"]["expected_sha256"] = "0" * 64
                if label == "identity": selected["version"] = "9.9.9"
                if label == "config-version": selected["parameters"]["version"] = "2"
                if label == "source-digest": bad["source_sha256"] = "0" * 64
                message = request("evaluate", bad)
                if label == "deadline": message["deadline_unix_ms"] = "1"
                self.run(version + "-refuse-" + label, [build / "beta-sdk-client"], message, success=False)
                self.check(not output.exists(), "refused publication: " + version + ":" + label)
        changed = Path(next(iter(new_files))); prior = changed.read_bytes(); changed.write_bytes(prior + b"tamper")
        try:
            self.remove(new_files)
        except ValueError:
            self.check(all(Path(p).exists() for p in new_files), "removal refusal preflights entire ownership set")
        else:
            raise AssertionError("changed owned bytes removed")
        changed.write_bytes(prior)
        self.remove(new_files)
        old_binary = self.work / "example-install/bin/beta-empirical-price/0.1.0/beta-empirical-price"
        self.direct(old_binary, [1, 3, 4], "0.1.0", "compatible-rollback")
        self.check(digest(self.sentinel) == self.sentinel_sha, "upgrade and rollback preserve unrelated user file")
        self.remove(old_files)
        self.check(digest(self.sentinel) == self.sentinel_sha and all(not Path(p).exists() for p in old_files), "removal preserves unrelated user file")
        self.check(self.verify_sdk() == self.sdk_before, "original complete SDK receipts and bytes unchanged")
        self.run("loader-closure", ["otool", "-L", old / "beta-empirical-provider.so", old / "beta-sdk-client"] if os.uname().sysname == "Darwin" else ["ldd", old / "beta-empirical-provider.so"])
        self.save(self.work / "evidence.json", {"protocol": "symphony.public-beta.external-engine-campaign.v1", "scope": "private_installed_sdk_fixture",
                  "engine_id": ID, "versions": ["0.1.0", "0.1.1"], "checks": self.checks, "check_count": len(self.checks),
                  "sdk_receipts": self.sdk_before, "example_source_sha256": {name: digest(self.project / name) for name in self.source_files},
                  "evidence_files_sha256": {str(p.relative_to(self.work)): digest(p) for folder in [self.work / "logs", self.work / "outputs"] for p in sorted(folder.iterdir()) if p.is_file()},
                  "complete_exact_math_oracle": True, "final_download_qualified": False, "independent_author_review": False,
                  "documentation_only_agent_attempt": "pending independently assigned reviewer", "administrative_stall_experiment": "prepared in README; not executed",
                  "release_acceptance_complete": False, "publication_authorized": False})
        print(json.dumps({"result": "passed_private_installed_sdk_fixture", "checks": len(self.checks), "release_acceptance_complete": False}, indent=2))

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--sdk-prefix", required=True)
    parser.add_argument("--work", required=True)
    parser.add_argument("--cxx", required=True)
    args = parser.parse_args()
    Campaign(args).execute()

if __name__ == "__main__":
    main()
