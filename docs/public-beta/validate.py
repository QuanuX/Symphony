#!/usr/bin/env python3
"""Validate the frozen preparation dossier, never candidate/release acceptance.

Uses Python's standard library and local Git only. No network, builds, provider
calls, command execution from JSON, or changes to the repository are performed.
"""

import argparse
from collections import Counter
import hashlib
import json
from pathlib import Path
import subprocess
import sys


class InvalidDossier(Exception):
    pass


def require(condition, message):
    if not condition:
        raise InvalidDossier(message)


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        require(key not in result, f"duplicate JSON key: {key}")
        result[key] = value
    return result


def parse_json(data):
    return json.loads(data, object_pairs_hook=unique_object)


def sha(data):
    return hashlib.sha256(data).hexdigest()


def unique(values, label):
    values = list(values)
    require(len(values) == len(set(values)), f"duplicate {label}")
    return set(values)


def validate(repo, folder, historical=False):
    def git(*args):
        return subprocess.check_output(
            ["git", "-C", str(repo), *args], stderr=subprocess.PIPE
        )

    def load(name):
        return parse_json((folder / name).read_text())

    d = load("dossier.json")
    inventory = load("source-inventory.json")
    mapping = load("acceptance-mapping.json")
    paper = load("paper-criteria.json")
    owners = load("owners.json")
    capabilities = load("capability-groups.json")
    packages = load("package-plan.json")
    base = d["source_commit"]
    require(len(base) == 40 and all(c in "0123456789abcdef" for c in base), "invalid source commit")
    require(d["format_version"] == 1, "unsupported dossier format")
    require(d["state"] == "prepared_for_manual_review", "not a preparation snapshot")
    for flag in ("scope_approved", "implementation_continuation_approved", "release_acceptance_complete", "publication_authorized"):
        require(d[flag] is False, f"preparation must not assert {flag}")
    require(d["feature_freeze"]["in_force"] is True, "feature freeze must remain active")
    require(d["feature_freeze"]["new_features_authorized"] is False, "new feature authority is not granted")
    require(git("rev-parse", f"{base}^{{tree}}").decode().strip() == d["source_tree"], "source tree mismatch")
    git("merge-base", "--is-ancestor", base, "HEAD")
    require(d["source_inventory"]["path"] == "source-inventory.json", "unexpected inventory path")
    require(sha((folder / "source-inventory.json").read_bytes()) == d["source_inventory"]["sha256"], "inventory digest mismatch")
    require(inventory["source_commit"] == base and inventory["source_tree"] == d["source_tree"], "inventory source binding mismatch")
    for data in (mapping, owners, capabilities, packages):
        require(data["format_version"] == 1 and data["source_commit"] == base, "companion source/version mismatch")

    # Verify the historical snapshot AND that product files have not drifted.
    allowed_tracking_file = ".github/RELEASE-GATES.md"
    for path, digest in d["source_fingerprints"].items():
        require(not Path(path).is_absolute() and ".." not in Path(path).parts, "unsafe source fingerprint path")
        require(sha(git("show", f"{base}:{path}")) == digest, f"historical fingerprint mismatch: {path}")
        if not historical and path != allowed_tracking_file:
            require(sha((repo / path).read_bytes()) == digest, f"working source drift: {path}")
    changed = git("diff", "--name-only", base, "--").decode().splitlines()
    untracked = git("ls-files", "--others", "--exclude-standard").decode().splitlines()
    if not historical:
        for path in changed + untracked:
            require(path.startswith("docs/public-beta/") or path == allowed_tracking_file,
                    f"source outside dossier changed; regenerate/review source binding: {path}")

    def source_json(path):
        return parse_json(git("show", f"{base}:{path}"))

    source_map = source_json("cmake/mvp-dependencies.json")
    plan = source_json("tests/launch-acceptance/plan.json")
    cloudflare = source_json("tests/cloudflare-release/support-matrix.json")
    for key in ("component_roots", "native_components", "native_build_order", "go_modules", "swift_provider", "external_dependencies"):
        require(inventory[key] == source_map[key], f"inventory projection drift: {key}")
    for key, path in (("dependency_map_sha256", "cmake/mvp-dependencies.json"),
                      ("acceptance_plan_sha256", "tests/launch-acceptance/plan.json"),
                      ("platform_profile_catalog_sha256", "cmake/mvp-platform-profiles.json")):
        require(inventory[key] == sha(git("show", f"{base}:{path}")), f"inventory binding mismatch: {key}")
    require(inventory["platform_profile_catalog"] == source_json("cmake/mvp-platform-profiles.json"), "platform inventory drift")
    for key in ("selected_components", "promised_provider_operations", "reviewer", "reviewed_at"):
        require(inventory[key] is None, f"raw inventory approval changed: {key}")
    for data in (inventory, mapping):
        for flag in ("release_acceptance_complete", "publication_authorized"):
            require(data[flag] is False, f"companion asserts {flag}")
        for key in ("required_profiles", "required_scenarios", "report_fields"):
            require(data[key] == plan[key], f"core acceptance contract changed: {key}")
        require(set(data["required_inputs"]) == set(plan["required_inputs"]), "required input lost")
        require(all(v is None for v in data["required_inputs"].values()), "raw input supplied without review")

    role_ids = unique((r["id"] for r in owners["roles"]), "owner role")
    require({r["id"]: r["person"] for r in owners["roles"] if r["person"] is not None}
            == {"release_approver": "@quantDIY", "rollback_primary": "@quantDIY"}, "unexpected named assignment")
    decision_ids = unique((x["id"] for x in d["decisions"]), "decision")
    require(decision_ids == {f"D{i:02}" for i in range(1, 8)}, "decision coverage changed")
    for decision in d["decisions"]:
        require(decision["state"] == "pending_manual_review" and decision["approved_by"] is None and decision["approved_at"] is None, "preparation asserts decision approval")
        require(decision["decision_owner_role"] in role_ids, "unknown decision owner")
        for path in decision["references"]:
            require((folder / path).is_file(), f"missing decision reference: {path}")

    roots = {x["path"] for x in inventory["component_roots"]}
    unique((g["id"] for g in capabilities["groups"]), "capability group")
    assigned = Counter(p for group in capabilities["groups"] for p in group["component_roots"])
    require(set(assigned) == roots, "capability roots missing or unknown")
    require(all(n == 1 for n in assigned.values()), "component root assigned more than once")
    for group in capabilities["groups"]:
        for key in ("operations", "platform_claims", "implementation_state", "qualification_state", "proposed_selection", "freeze_disposition", "evidence_paths"):
            require(bool(group[key]), f"capability group lacks {key}: {group['id']}")
        if not group["acceptance_scenarios"]:
            require(group["proposed_selection"]["state"] in ("proposed_build_source_only", "proposed_no_binary_proposal_visible"), "runtime group lacks acceptance scenarios")
        for scenario in group["acceptance_scenarios"]:
            require(scenario in plan["required_scenarios"], "unknown capability scenario")
        require(group["proposed_selection"]["approval"] == "unapproved", "capability approval invented")
        for path in group["evidence_paths"]:
            require((repo / path).is_file(), f"missing capability evidence: {path}")
    require(capabilities["capability_selection_approved"] is False and capabilities["publication_authorized"] is False, "capability approval asserted")

    require(packages["approval_state"] == "pending_human_approval", "package selection approval invented")
    require(packages["publication_authorized"] is False and packages["real_build_execution_authorized_by_this_document"] is False, "package plan grants execution/publication")
    require(packages["candidate_source_commit"] is None and packages["candidate_source_tree"] is None, "candidate identity fabricated")
    recipes = packages["package_recipes"]
    recipe_ids = unique((r["id"] for r in recipes), "package recipe")
    input_ids = unique((i["id"] for i in packages["open_inputs"]), "package input")
    trust_ids = unique((t["id"] for t in packages["trust_requirements"]), "trust requirement")
    dispositions = packages["source_root_dispositions"]
    require(unique((r["source_root"] for r in dispositions), "package source root") == roots, "package root coverage mismatch")
    for row in dispositions:
        if row["proposed_recipe_id"] is not None:
            require(row["proposed_recipe_id"] in recipe_ids, "unknown root recipe")
    for item in packages["open_inputs"]:
        require(item["decision_id"] in decision_ids and item["state"] == "open" and item["value"] is None, "package input silently supplied")
    for requirement in packages["trust_requirements"]:
        require(set(requirement["input_ids"]) <= input_ids, "unknown trust input")
    for recipe in recipes:
        require(recipe["approval_state"] == "pending_human_approval" and recipe["supported_profiles"] == [], "recipe claims acceptance")
        require(set(recipe["source_roots"]) <= roots, "unknown recipe root")
        require(set(recipe["trust_requirements"]) <= trust_ids and set(recipe["open_input_ids"]) <= input_ids, "recipe trust/input mismatch")
        require(all(value is None for value in recipe["observed_artifacts"].values()), "fabricated candidate artifact")
        require(set(recipe.get("dependency_recipe_ids", [])) <= recipe_ids, "unknown dependency recipe")
        for field in ("content_recipe_ids", "separately_selectable_recipe_ids"):
            require(set(recipe.get(field, [])) <= recipe_ids, f"unknown recipe reference: {field}")
    by_recipe_id = {r["id"]: r for r in recipes}
    for recipe in recipes:
        if recipe["kind"] == "delivery_set_not_single_native_archive":
            contents = unique(recipe["content_recipe_ids"], "delivery content")
            for member in contents:
                require(set(by_recipe_id[member].get("dependency_recipe_ids", [])) <= contents, "delivery set omits native dependency closure")
    native_by_root = {n["path"]: n for n in inventory["native_components"] if n["installable"]}
    native_recipes = [r for r in recipes if r["kind"] == "native_receipt_unit"]
    require({r["source_roots"][0] for r in native_recipes} == set(native_by_root), "native receipt unit omitted or invented")
    for recipe in native_recipes:
        component = native_by_root[recipe["source_roots"][0]]
        require(recipe["proposed_package_version"] == component["package_release_version"] and recipe["sdk_exports"] == component["exported_packages"], "native version/SDK export drift")

    campaign_ids = unique((x["id"] for x in mapping["campaigns"]), "campaign")
    require({x["scenario"] for x in mapping["campaigns"]} == set(plan["required_scenarios"]), "campaign coverage mismatch")
    for campaign in mapping["campaigns"]:
        require(campaign["owner_role"] in role_ids, "unknown campaign owner")
        require(campaign["execution_status"] == "not_run" and campaign["acceptance_status"] == "pending", "prepared campaign asserts result")
        require(set(campaign["required_decisions"]) <= decision_ids, "unknown campaign decision")
        for path in campaign["existing_harnesses"]:
            require((repo / path).is_file(), f"missing harness reference: {path}")
    cells = mapping["core_cells"]
    unique((c["id"] for c in cells), "core cell")
    expected = {(p, s) for p in plan["required_profiles"] for s in plan["required_scenarios"]}
    actual = [(c["profile"], c["scenario"]) for c in cells]
    require(len(actual) == len(expected) == mapping["core_cell_count"] == 36 and set(actual) == expected, "36 core profile/scenario cells must be preserved")
    for cell in cells:
        require(cell["id"] == f"{cell['profile']}::{cell['scenario']}", "core cell ID mismatch")
        require(cell["mandatory"] is True and cell["candidate_sha256"] is None and cell["skips"] is None, "core cell fabricates qualification")

    required_criteria = {f"C{i:02}.{j}" for i in range(1, 14) for j in range(1, 5)} | {f"UE{i}" for i in range(1, 7)}
    paper_ids = unique((x["criterion_id"] for x in paper["criteria"]), "paper criterion")
    mapped_ids = unique((x["criterion_id"] for x in mapping["paper_criteria"]), "mapped criterion")
    require(paper_ids == mapped_ids == required_criteria and mapping["paper_criterion_count"] == 58, "58 criterion coverage mismatch")
    paper_by_id = {x["criterion_id"]: x for x in paper["criteria"]}
    for row in mapping["paper_criteria"]:
        require(row["criterion_specific_expected_outcome"] == paper_by_id[row["criterion_id"]]["paper_criterion_text"], "criterion text changed or omitted")
        require(row["owner_role"] in role_ids and set(row["campaign_ids"]) <= campaign_ids, "criterion owner/campaign mismatch")
        require(row["scope_exclusion_decision"] is None and row["applicability"] == "proposed_pending_manual_scope_review", "criterion silently waived")
    for category in paper["categories"]:
        require(category["current_accepted_score"] is None and category["current_score_status"] == "not_reassessed", "invented acceptance score")

    cf_cells = mapping["cloudflare_operation_cells"]
    unique((c["operation"] for c in cf_cells), "Cloudflare operation")
    cf_by_operation = {r["operation"]: r for r in cloudflare["resource_operations"]}
    require({c["operation"] for c in cf_cells} == set(cf_by_operation) and len(cf_cells) == 23, "Cloudflare operation coverage changed")
    for cell in cf_cells:
        source = cf_by_operation[cell["operation"]]
        require(cell["runtime_platform"] in cloudflare["admitted_vendor_runtime_platforms"], "unadmitted Cloudflare runtime")
        require(cell["http_method"] == source["method"] and cell["provider_path_template"] == source["path"], "Cloudflare operation binding drift")
        require(cell["implementation_at_baseline"] == source["implementation"] and cell["live_at_baseline"] == source["live"], "Cloudflare evidence status drift")
        require(cell["exact_host_profile"] is None and cell["host_profile_decision"] == "D02", "invented Cloudflare host admission")
    for row in cells + mapping["paper_criteria"] + cf_cells + mapping["additional_campaigns"]:
        require(row["execution_status"] == "not_run" and row["review_status"] == "not_accepted" and row["evidence"] == [], "prepared mapping asserts acceptance evidence")
        if "campaign_ids" in row:
            require(set(row["campaign_ids"]) <= campaign_ids, "unknown supplemental campaign")
    return {"status": "valid_historical_preparation_only" if historical else "valid_preparation_only", "source_commit": base,
            "current_product_source_verified": not historical,
            "component_roots": len(roots), "native_components": len(inventory["native_components"]),
            "capability_groups": len(capabilities["groups"]), "core_cells": len(cells),
            "paper_criteria": len(required_criteria), "cloudflare_operations": len(cf_cells),
            "scope_approved": False, "release_acceptance_complete": False, "publication_authorized": False}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dossier", type=Path, default=Path(__file__).resolve().parent,
                        help="alternate copy for validation; source always binds to this script's repository")
    parser.add_argument("--historical", action="store_true",
                        help="validate the preserved preparation against its Git source, not current implementation files")
    args = parser.parse_args()
    repo = Path(__file__).resolve().parents[2]
    try:
        result = validate(repo, args.dossier.resolve(), args.historical)
    except (InvalidDossier, OSError, ValueError, KeyError, TypeError, subprocess.CalledProcessError) as error:
        print(f"INVALID preparation dossier: {error}", file=sys.stderr)
        return 1
    print(json.dumps(result, indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
