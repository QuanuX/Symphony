#!/usr/bin/env python3
"""Offline exact-owner maintenance/reassessment acceptance over retained DO captures.

No acquisition, protected selection, provider operation or implicit version choice.
Expected typed meanings are authored independently from the three cited limit
sentences. A matching quote is not proof of universal provider behavior.
"""
import argparse
import datetime as dt
import hashlib
import json
from pathlib import Path
import subprocess

VERSION = "0.10.0-dev"
EXPECTED_CLAIMS = {
    "do-app-platform-amd64": {
        "claim_id": "do-app-platform-amd64", "subject": "do.app-platform.container",
        "predicate": "image.architecture",
        "scope": {"provider": "do", "service": "app-platform", "operating-system": "linux"},
        "statement_kind": "documented_fact", "dependencies": [],
        "valid_from": None, "valid_until": None,
        "value": {"type": "string", "value": "amd64", "unit": None}},
    "do-app-platform-local-storage": {
        "claim_id": "do-app-platform-local-storage", "subject": "do.app-platform.container",
        "predicate": "local-storage.persistent",
        "scope": {"provider": "do", "service": "app-platform", "storage": "host-local-filesystem"},
        "statement_kind": "documented_fact", "dependencies": [],
        "valid_from": None, "valid_until": None,
        "value": {"type": "boolean", "value": False, "unit": None}},
    "do-limits-dev-valkey": {
        "claim_id": "do-limits-dev-valkey", "subject": "do.app-platform.development-database",
        "predicate": "engine.valkey.available",
        "scope": {"provider": "do", "service": "app-platform", "component": "development-database", "engine": "valkey"},
        "statement_kind": "documented_fact", "dependencies": [],
        "valid_from": None, "valid_until": None,
        "value": {"type": "boolean", "value": False, "unit": None}},
}
# Only this explicitly authored oracle supplies expected semantics. Null claim
# validity bounds do not bypass the independently selected capture-age policy.
EXPECTED = {claim_id: value["value"] for claim_id, value in EXPECTED_CLAIMS.items()}
SEMANTIC_FIELDS = ("claim_id", "subject", "predicate", "scope", "statement_kind",
                   "value", "dependencies", "valid_from", "valid_until")


def read(path):
    return json.loads(path.read_bytes())


def digest(raw):
    return "sha256:" + hashlib.sha256(raw).hexdigest()


def write(path, value):
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2) + "\n")


def stamp(value):
    return value.isoformat().replace("+00:00", "Z")


def instant(value):
    return dt.datetime.fromisoformat(value.replace("Z", "+00:00"))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for key in ["qxctl", "prefix", "old-capture", "fresh-capture", "insufficient-capture", "profile", "out"]:
        parser.add_argument("--" + key, required=True, type=Path)
    args = parser.parse_args()
    out = args.out.resolve()
    out.mkdir(parents=True, exist_ok=False, mode=0o700)
    commands = []

    def call(name, route, value):
        request = out / (name + "-input.json")
        write(request, value)
        argv = [str(args.qxctl.resolve()), "scv", *route.split(), "--domain", "scv",
                "--version", VERSION, "--prefix", str(args.prefix.resolve()),
                "--input", str(request), "--json"]
        run = subprocess.run(argv, cwd=out, capture_output=True, timeout=120)
        (out / (name + "-stdout.json")).write_bytes(run.stdout)
        (out / (name + "-stderr.txt")).write_bytes(run.stderr)
        commands.append({"name": name, "argv": argv, "exit_code": run.returncode,
                         "input_file_digest": digest(request.read_bytes()),
                         "stdout_file_digest": digest(run.stdout), "stderr_file_digest": digest(run.stderr)})
        write(out / "COMMANDS.json", commands)
        assert run.returncode == 0, (name, run.stdout[:2000], run.stderr[:2000])
        return json.loads(run.stdout)

    captures = {"old": read(args.old_capture), "fresh": read(args.fresh_capture),
                "insufficient": read(args.insufficient_capture)}
    profile = read(args.profile)
    assert profile["profile_id"] == "coverage-do-app-platform-limits"
    assert profile["profile_version"] == "1"
    assert all(c["completeness"] == "complete" for c in captures.values())
    assert captures["old"]["body"] == captures["fresh"]["body"]
    assert captures["old"]["source"]["digest"] != captures["fresh"]["source"]["digest"]
    assert captures["insufficient"]["source"]["digest"] != captures["fresh"]["source"]["digest"]
    policy = {"policy_id": "maintained-provider-reference-policy-v1", "max_age_seconds": 86400,
              "allowed_statement_kinds": ["documented_fact", "requirement", "recommendation"],
              "partial_capture": "exclude"}
    interpretations = {}
    for label, capture in captures.items():
        interpretations[label] = call(label + "-interpret", "provider interpret", {
            "captures": [capture], "profiles": [profile], "bindings": [{
                "profile_digest": profile["digest"], "capture_digest": capture["digest"]}],
            "selection_policy": policy})
        claims = interpretations[label]["knowledge"]["claims"]
        semantics = {c["claim_id"]: {key: c[key] for key in SEMANTIC_FIELDS} for c in claims}
        if label == "insufficient":
            assert semantics == {}, semantics
            assert len(interpretations[label]["extractions"]) == 3
            assert all(e["status"] == "unresolved" for e in interpretations[label]["extractions"])
        else:
            assert semantics == EXPECTED_CLAIMS, semantics
            assert all(e["status"] == "matched" for e in interpretations[label]["extractions"])

    query_time = stamp(max(instant(c["observed_at"]) for c in captures.values()) + dt.timedelta(seconds=1))
    assert (instant(query_time) - instant(captures["old"]["observed_at"])).total_seconds() > 86400
    assert 0 <= (instant(query_time) - instant(captures["fresh"]["observed_at"])).total_seconds() <= 86400
    scope = {"provider": "do", "service": "app-platform", "storage": "host-local-filesystem"}
    resolution = {"kind": "caller_decision", "reference": "example:host-local-storage-requirement",
                  "description": "Caller selects a disposable host-local filesystem for this hypothetical recipe; this is not a platform preference."}
    requirement = {"requirement_id": "host-local-persistence", "importance": "required", "operator": "eq",
                   "right": {"kind": "literal", "value": {"type": "boolean", "value": False, "unit": None}},
                   "resolution": resolution}
    persistent = {**requirement, "right": {"kind": "literal", "value": {"type": "boolean", "value": True, "unit": None}}}
    base = {"interpretations": [], "additional_knowledge": [], "provider_packs": [],
            "query_time": query_time, "requirements": [requirement], "slots": [{
                "slot_id": "caller-selected-app-platform", "allowed_provider_ids": ["do"], "recipes": [{
                    "recipe_id": "hypothetical-host-local-workload", "provider_id": "do", "bindings": [{
                        "requirement_id": "host-local-persistence", "claim": {"claim_id": "do-app-platform-local-storage",
                            "subject": "do.app-platform.container", "scope": scope}}],
                    "prerequisites": [], "requires_interfaces": [], "supplies_interfaces": [], "guarantee_changes": [],
                    "implementation": {"status": "unimplemented", "reference": None},
                    "resolution": {"kind": "adapter", "reference": "example:unimplemented-host-local-workload",
                                   "description": "Authored recipe only; no runtime, account, network, deployment or payload verification."}}]}],
            "allowed_guarantee_changes": [], "counterfactuals": [{"counterfactual_id": "caller-requires-persistence", "requirements": [persistent]}],
            "bounds": {"max_candidates": 1}}
    compositions = {}
    statuses = {}
    for label, interpretation in interpretations.items():
        result = call(label + "-explore", "composition explore", {**base, "interpretations": [interpretation]})
        compositions[label] = result
        statuses[label] = {s["scenario_id"]: s["candidates"][0]["status"] for s in result["scenarios"]}
        for scenario in result["scenarios"]:
            candidate = scenario["candidates"][0]
            assert candidate["implementation_status"] == "unimplemented"
            assert any(o["kind"] == "implementation_gap" for o in candidate["obligations"])
        expected = {"baseline": "satisfied", "caller-requires-persistence": "contradicted"} if label == "fresh" else {
            "baseline": "unresolved", "caller-requires-persistence": "unresolved"}
        assert statuses[label] == expected, (label, statuses[label])
    reassessments = {}
    for label, before, after in [("refresh", "old", "fresh"), ("insufficient", "fresh", "insufficient")]:
        result = call(label + "-reassess", "composition reassess", {"before": compositions[before], "after": compositions[after]})
        reassessments[label] = result
        assert result["change_axes"] == {"evidence": True, "policy": False, "requirements": False,
                                           "query_time": False, "recipes": False, "selections": False, "bounds": False}, result["change_axes"]
    expiry_time = stamp(instant(captures["fresh"]["observed_at"]) + dt.timedelta(seconds=86401))
    expired = call("expired-explore", "composition explore", {**base, "query_time": expiry_time, "interpretations": [interpretations["fresh"]]})
    assert all(s["candidates"][0]["status"] == "unresolved" for s in expired["scenarios"])
    expiry = call("expiry-reassess", "composition reassess", {"before": compositions["fresh"], "after": expired})
    assert expiry["change_axes"] == {"evidence": False, "policy": False, "requirements": False,
                                     "query_time": True, "recipes": False, "selections": False, "bounds": False}
    summary = {"format": "local.scv.source-maintenance-semantics.v1", "passed": True,
               "native_version": VERSION, "qxctl_file_digest": digest(args.qxctl.read_bytes()),
               "input_files": {str(p.resolve()): digest(p.read_bytes()) for p in [args.old_capture, args.fresh_capture, args.insufficient_capture, args.profile]},
               "capture_digests": {k: v["digest"] for k, v in captures.items()}, "profile_digest": profile["digest"],
               "independent_expected_values": EXPECTED, "independent_expected_claims": EXPECTED_CLAIMS,
               "query_time": query_time, "expiry_query_time": expiry_time,
               "selection_policy": policy, "statuses": statuses,
               "composition_digests": {k: v["digest"] for k, v in compositions.items()},
               "reassessment_digests": {k: v["digest"] for k, v in reassessments.items()},
               "expiry_reassessment_digest": expiry["digest"], "commands": len(commands),
               "scope": "Retained-document and hypothetical caller-recipe acceptance only; no publisher authentication, protected selection, runtime proof or provider-wide exclusion."}
    write(out / "SUMMARY.json", summary)
    print(json.dumps({"passed": True, "commands": len(commands), "statuses": statuses, "out": str(out)}, indent=2))


if __name__ == "__main__":
    main()
