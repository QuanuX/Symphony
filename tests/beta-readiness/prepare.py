#!/usr/bin/env python3
"""Prepare a source-bound, explicitly incomplete W12 review; never deploy or publish."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys

INPUTS = ["docs/public-beta/package-plan.json", "docs/public-beta/acceptance-mapping.json",
          "docs/public-beta/capability-groups.json", "docs/public-beta/owners.json",
          "docs/public-beta/execution/authorization.json", "cmake/mvp-dependencies.json",
          "cmake/mvp-release-identities.json", "cmake/mvp-platform-profiles.json",
          "tests/launch-acceptance/plan.json"]
FLAGS = {"release_acceptance_complete": False, "publication_authorized": False}

def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def read(path):
    if path.is_symlink() or not path.is_file() or path.stat().st_size > 16 * 1024 * 1024:
        raise ValueError("bounded regular source input required: " + str(path))
    return json.loads(path.read_text())

def unique(rows, key):
    result = {r[key]: r for r in rows}
    if len(result) != len(rows):
        raise ValueError("duplicate " + key)
    return result

def construction(package, dependency, identities, plan):
    """Reconcile approved engineering recipes against current owner build facts."""
    recipes = unique(package["package_recipes"], "id")
    native = unique(dependency["native_components"], "path")
    go_owners = unique(identities["go_artifacts"], "path")
    result = {}
    for profile in plan["required_profiles"]:
        group = recipes["PKG-CORE-" + profile]
        selected = group["content_recipe_ids"]
        if len(set(selected)) != len(selected):
            raise ValueError("duplicate selected recipe")
        selected_native, selected_go = {}, []
        for recipe_id in selected:
            r = recipes[recipe_id]
            if profile not in r["candidate_profiles"]:
                raise ValueError("recipe not declared on profile")
            if r["kind"] == "native_receipt_unit":
                if len(r["source_roots"]) != 1:
                    raise ValueError("native recipe root mismatch")
                root = r["source_roots"][0]
                n = native[root]
                if (r["proposed_package_version"] != n["package_release_version"] or
                    r["project_version"] != n["project_version"] or
                    r["installed_options"] != n["installed_options"] or
                    r["build_targets"] != n["artifact_targets"] or not n["installable"]):
                    raise ValueError("recipe drift from current native owner: " + recipe_id)
                required = {"PKG-NATIVE-" + native[p]["component_id"] for p in n["dependency_paths_in_build_order"]}
                if not required <= set(selected):
                    raise ValueError("missing native dependency closure")
                selected_native[root] = {"recipe_id": recipe_id, "component_id": n["component_id"],
                    "source_root": root, "version": n["package_release_version"],
                    "installed_options": n["installed_options"], "build_targets": n["artifact_targets"],
                    "dependency_source_roots": n["dependency_paths_in_build_order"],
                    "external_dependency_refs": r["external_dependency_refs"],
                    "recipe_template": package["build_templates"][r["build_template"]]}
            elif r["kind"] == "go_command":
                if len(r["source_roots"]) != 1:
                    raise ValueError("Go recipe root mismatch")
                g = go_owners[r["source_roots"][0]]
                if (r["proposed_package_version"] != g["package_version"] or
                    any(r[k] != g[k] for k in ["binary_name", "main_package", "version_symbol", "source_runtime_version"])):
                    raise ValueError("recipe drift from current Go owner")
                selected_go.append({k: r[k] for k in ["id", "source_roots", "binary_name", "main_package",
                    "proposed_package_version", "version_symbol", "build_template", "installation_contract"]})
            else:
                raise ValueError("unsupported core recipe kind")
        if len(selected_native) != 51 or len(selected_go) != 6:
            raise ValueError("approved 51 native / 6 Go engineering selection changed")
        order = [p for p in dependency["native_build_order"] if p in selected_native]
        if len(order) != len(set(order)) or set(order) != set(selected_native):
            raise ValueError("build order omits or duplicates selected unit")
        seen = set()
        for p in order:
            if not set(native[p]["dependency_paths_in_build_order"]) <= seen:
                raise ValueError("non-topological native build order")
            seen.add(p)
        result[profile] = {"native_build_order": [selected_native[p] for p in order], "go_commands": selected_go,
            "optional_recipe_ids": group["separately_selectable_recipe_ids"], "final_stage": None,
            "finalizer_and_loader_policy": None, "toolchain_and_dependency_inputs": None,
            "payload_review": None, "candidate_identity": None, "retrieved_anaconda_identity": None}
    return {"protocol": "symphony.public-beta.construction-worklist.v1", "scope": "source_plan_only",
        "profiles": result, "go_build_templates": {k: package["build_templates"][k] for k in ["GO-IDENTITIES", "QXCTL-REPRODUCIBLE"]},
        "external_dependency_requirements": dependency["external_dependencies"],
        "supplemental_cloudflare": recipes["PKG-CF-INTEL"], "source_material": recipes["PKG-SOURCE"],
        "construction_note": "No final stage or build is implied. Select exact inputs, finalize before receipt sealing, review actual notices/runtime/source closure, then sign with independent policy.", **FLAGS}

def workorders(mapping, plan):
    if (len(plan["required_profiles"]) != 4 or len(set(plan["required_profiles"])) != 4 or
        len(plan["required_scenarios"]) != 9 or len(set(plan["required_scenarios"])) != 9):
        raise ValueError("four unique profiles and nine unique scenarios required")
    if (mapping["core_cell_count"] != 36 or mapping["paper_criterion_count"] != 58 or
        set(mapping["required_profiles"]) != set(plan["required_profiles"]) or
        set(mapping["required_scenarios"]) != set(plan["required_scenarios"])):
        raise ValueError("acceptance scope drift")
    campaigns = unique(mapping["campaigns"], "scenario")
    if set(campaigns) != set(plan["required_scenarios"]):
        raise ValueError("missing scenario work order")
    criteria = unique(mapping["paper_criteria"], "criterion_id")
    extra = mapping["cloudflare_operation_cells"] + mapping["additional_campaigns"]
    unique(extra, "id")
    if len(criteria) != 58 or len(extra) != 31:
        raise ValueError("missing retained criteria or supplemental work")
    pending = {"final_artifact_bindings": None, "operator": None, "independent_reviewer": None,
               "observed_result": None, "evidence": [], "acceptance_status": "pending"}
    return {"protocol": "symphony.public-beta.campaign-workorders.v1",
        "core": [{"id": profile + "::" + scenario, "profile": profile,
                  "source_campaign": campaigns[scenario], **pending}
                 for profile in plan["required_profiles"] for scenario in plan["required_scenarios"]],
        "paper": [{"id": r["criterion_id"], "source_criterion": r, **pending} for r in criteria.values()],
        "supplemental": [{"id": r["id"], "source_requirement": r, **pending} for r in extra], **FLAGS}

def source(repo):
    def git(*args):
        return subprocess.check_output(["git", "-C", str(repo), *args], text=True).strip()
    if git("status", "--porcelain", "--untracked-files=all"):
        raise ValueError("clean committed source required; retain changes before preparing review")
    return {"commit": git("rev-parse", "HEAD"), "tree": git("rev-parse", "HEAD^{tree}")}

def prepare(args):
    repo = Path(args.repo).resolve(strict=True)
    initial = source(repo)
    values = {p: read(repo / p) for p in INPUTS}
    hashes = {p: sha(repo / p) for p in INPUTS}
    auth = values[INPUTS[4]]
    if (auth["state"] != "implementation_authorized" or not auth["feature_freeze_in_force"] or
        auth["new_product_features_authorized"] or auth["publication_authorized"]):
        raise ValueError("expected frozen engineering authorization")
    plan = values[INPUTS[-1]]
    builds = construction(values[INPUTS[0]], values[INPUTS[5]], values[INPUTS[6]], plan)
    campaigns = workorders(values[INPUTS[1]], plan)
    work = Path(args.work)
    if not work.is_absolute() or work.exists() or work.is_symlink():
        raise ValueError("new absolute work directory required")
    work = work.parent.resolve(strict=True) / work.name
    if work.is_relative_to(repo):
        raise ValueError("review directory must be outside source")
    go = Path(args.go).absolute()
    env = dict(os.environ, GOWORK="off", GOENV="off", GOTOOLCHAIN="local", GOPROXY="off", GOSUMDB="off")
    if not subprocess.check_output([str(go), "version"], env=env, text=True).startswith("go version go1.26.5 "):
        raise ValueError("exact Go 1.26.5 required")
    os.umask(0o077)
    work.mkdir(mode=0o700)
    (work / "logs").mkdir()
    (work / "tools").mkdir()
    env.update(GOCACHE=str(work / "go-cache"), GOMODCACHE=str(work / "go-modules"))
    def write(name, obj):
        with (work / name).open("x") as f:
            json.dump(obj, f, indent=2); f.write("\n")
    def run(label, command, expected=0):
        with (work / "logs" / (label + ".log")).open("xb") as log:
            r = subprocess.run([str(x) for x in command], cwd=repo, env=env, stdout=log, stderr=subprocess.STDOUT)
        if r.returncode != expected:
            raise ValueError(label + ": unexpected exit " + str(r.returncode))
    for check in ["mvp-dependencies", "release-qualification"]:
        run(check, ["cmake", "-P", repo / "tests" / check / "verify.cmake"])
    prefix = repo / "tests/launch-acceptance"
    for binary, files in {"acceptance-gates": ["acceptance-gates.go", "acceptance-gates-main.go"],
        "delivery-acceptance": ["delivery-set.go", "acceptance-gates.go", "delivery-acceptance.go"]}.items():
        run("build-" + binary, [go, "build", "-trimpath", "-o", work / "tools" / binary, *[prefix / f for f in files]])
    core = work / "tools/acceptance-gates"
    companion = work / "tools/delivery-acceptance"
    for name, value in [("core-plan.json", plan), ("acceptance-mapping.json", values[INPUTS[1]]),
                        ("construction.json", builds), ("workorders.json", campaigns)]:
        write(name, value)
    plan_path, mapping_path = work / "core-plan.json", work / "acceptance-mapping.json"
    run("init-core", [core, "-mode", "init", "-plan", plan_path, "-packet", work / "core-packet.json"])
    run("init-requirements", [companion, "-mode", "init-requirements", "-plan", plan_path,
        "-mapping", mapping_path, "-requirements", work / "requirements.json"])
    base = [companion, "-plan", plan_path, "-mapping", mapping_path, "-requirements", work / "requirements.json",
            "-core-packet", work / "core-packet.json", "-packet", work / "companion-packet.json"]
    run("init-companion", [*base, "-mode", "init"])
    write("targets.json", {"protocol": "symphony.release.delivery-targets.v1", "profiles": {p: None for p in plan["required_profiles"]}})
    run("check-core", [core, "-mode", "check", "-plan", plan_path, "-packet", work / "core-packet.json", "-report", work / "core-report.json"], 2)
    run("check-companion", [*base, "-mode", "check", "-targets", work / "targets.json", "-report", work / "companion-report.json"], 2)
    final = source(repo)
    if final != initial or hashes != {p: sha(repo / p) for p in INPUTS}:
        raise ValueError("source changed during preparation")
    write("review.json", {"protocol": "symphony.public-beta.w12-review.v1", "source": initial,
        "state": "incomplete_no_go_for_public_beta", "human_release_decision": None,
        "distribution_channel": auth["distribution_channel"], "end_W12_inputs": auth["deferred_to_end_of_W12"],
        "scope": "local engineering and qualification preparation; no final candidate or operational acceptance",
        "source_inputs_sha256": hashes, "go_driver_sha256": sha(go),
        "files_sha256": {str(p.relative_to(work)): sha(p) for p in sorted(work.rglob("*"))
            if p.is_file() and p.parts[-2] in [work.name, "logs", "tools"]},
        "counts": {"profiles": 4, "native_units_per_core": 51, "go_commands_per_core": 6,
                   "core_cells": len(campaigns["core"]), "paper_criteria": len(campaigns["paper"]),
                   "supplemental": len(campaigns["supplemental"])}, **FLAGS})
    print(json.dumps({"review": str(work / "review.json"), "state": "incomplete_no_go_for_public_beta", **FLAGS}, indent=2))
    return 2

def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--repo", required=True); p.add_argument("--work", required=True); p.add_argument("--go", required=True)
    try:
        return prepare(p.parse_args())
    except (ValueError, KeyError, OSError, subprocess.SubprocessError) as error:
        print("beta review refused: " + str(error), file=sys.stderr)
        return 1

if __name__ == "__main__":
    sys.exit(main())
