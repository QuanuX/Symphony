#!/usr/bin/env python3
"""Exercise the native delivery adapter with the existing package-release fixture."""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess

SOURCES = [
    "tests/launch-acceptance/candidate-host.go",
    "cmake/release/VerifyCandidate.cmake", "cmake/release/Support.cmake",
    "cmake/release/StrictJSON.cmake", "cmake/SymphonyReceiptV2Support.cmake",
]


def digest(path):
    with path.open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def write(path, value):
    path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
    with path.open("x") as stream:
        stream.write(json.dumps(value, indent=2) + "\n")
    path.chmod(0o600)


def main(args):
    paths = {k: v.resolve(strict=True) for k, v in vars(args).items() if k != "work"}
    work = args.work
    if not work.is_absolute() or work.exists() or work.is_symlink():
        raise ValueError("new absolute work required")
    work = work.parent.resolve(strict=True) / work.name
    if any(work.is_relative_to(paths[k]) for k in ("repository", "candidate")):
        raise ValueError("work must be outside source and candidate")
    work.mkdir(mode=0o700)
    root = work / "delivery"
    root.mkdir(mode=0o700)
    shutil.copytree(paths["candidate"], root / "native")
    manifest = json.loads((root / "native/candidate.json").read_bytes())
    versions = {c["component_id"]: c["version"] for c in manifest["components"]}
    if len(versions) != len(manifest["components"]):
        raise ValueError("select one version per component in each native artifact")
    profile = {k: manifest["platform"][k] for k in ("profile_id", "os", "architecture")}
    choice = {"id": "native-fixture", "package_id": "native-fixture", "kind": "native",
              "verification_adapter": "native-candidate-v1", "component_versions": versions, "dependencies": []}
    files = ["native/" + p.name for p in sorted(paths["candidate"].iterdir())]
    metadata = "metadata/fixture-provenance.json"
    write(root / metadata, {"scope": "synthetic native package fixture", "source": manifest["source"],
                            "notice_review_complete": False})
    artifact = {"selection": choice, "source": manifest["source"], "vendor": None, "file_paths": files,
                "primary_path": "native/candidate.tar", "payload_inventory_path": "native/candidate.json",
                "manifest_path": "native/candidate.json", "signature_path": "native/candidate.json.sig",
                "artifact_signer_policy_sha256": digest(paths["policy"]), "receipt_paths": [],
                "install_recipe_path": metadata, "remove_recipe_path": metadata, "notice_paths": [metadata],
                "source_material_paths": [metadata], "build_evidence_paths": ["native/build-evidence.json"]}
    selection = {"protocol": "symphony.release.delivery-selection.v1", "source": manifest["source"], "profile": profile,
                 "capability_manifest_sha256": digest(root / metadata),
                 "dependency_map_sha256": manifest["dependency_map_sha256"],
                 "identity_catalog_sha256": manifest["identity_catalog_sha256"], "artifacts": [choice]}
    write(work / "selection.json", selection)
    write(work / "definition.json", {"set_id": "native-fixture", "file_paths": files + [metadata], "artifacts": [artifact]})
    record = {"protocol": "symphony.release.native-verifier-tool.v1", "tool_sha256": digest(paths["candidate_host"]),
              "go_version": "go1.26.5", "source_files_sha256": {p: digest(paths["repository"] / p) for p in SOURCES}}
    write(work / "native-tool.json", record)
    logs = work / "logs"
    logs.mkdir(mode=0o700)

    def run(name, command, expected=0):
        result = subprocess.run([str(v) for v in command], capture_output=True, timeout=720)
        log = logs / (name + ".log")
        log.write_bytes(result.stdout + result.stderr)
        log.chmod(0o600)
        if result.returncode != expected:
            raise ValueError("native adapter step failed: " + name)
        return result

    common = [paths["delivery_tool"], "-root", root, "-selection", work / "selection.json",
              "-allowed-signers", paths["policy"], "-profile", profile["profile_id"], "-signer", "fixture-publisher"]
    run("prepare", [*common, "-mode", "prepare", "-definition", work / "definition.json"])
    run("sign-index", ["ssh-keygen", "-Y", "sign", "-f", paths["key"], "-n", "symphony-delivery-set", root / "delivery-set.json"])
    inspect = [*common, "-mode", "inspect-native", "-artifact", "native-fixture",
               "-owner-tool", paths["candidate_host"], "-owner-build", work / "native-tool.json",
               "-owner-repository", paths["repository"], "-native-allowed-signers", paths["policy"],
               "-native-signer", "fixture-publisher"]
    run("inspect", [*inspect, "-owner-work", work / "extracted", "-report", work / "native-owner.json"])
    proof = json.loads((work / "native-owner.json").read_bytes())
    if (not proof["selected_artifact_owner_verified"] or proof["fresh_host_verified"]
            or proof["release_acceptance_complete"] or proof["publication_authorized"]
            or proof["artifact_sha256"] != manifest["artifact"]["sha256"]):
        raise ValueError("native proof mismatch or overclaim")
    run("reused-work", [*inspect, "-owner-work", work / "extracted"], expected=1)
    bad_policy = work / "changed-native-policy"
    bad_policy.write_bytes(paths["policy"].read_bytes() + b"\n")
    run("wrong-native-policy", [*inspect, "-native-allowed-signers", bad_policy,
                              "-owner-work", work / "must-not-extract"], expected=1)
    if (work / "must-not-extract").exists():
        raise ValueError("native policy mismatch performed extraction")
    write(work / "evidence.json", {"protocol": "symphony.release.native-delivery-qualification.v1",
          "status": "passed_fixture_only", "source": manifest["source"], "profile": profile,
          "native_owner_report_sha256": digest(work / "native-owner.json"),
          "tool_record_sha256": digest(work / "native-tool.json"),
          "cases": ["signed_native_receipts", "reused_work_refusal", "independent_native_policy_refusal"],
          "fresh_host_qualified": False, "release_acceptance_complete": False, "publication_authorized": False})
    print("Native delivery adapter passed fixture extraction and refusal checks; acceptance remains pending")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    for option in ("repository", "candidate", "candidate-host", "delivery-tool", "policy", "key", "work"):
        parser.add_argument("--" + option, type=Path, required=True)
    try:
        main(parser.parse_args())
    except (ValueError, OSError, subprocess.SubprocessError, KeyError) as error:
        raise SystemExit(f"Native delivery fixture failed; retain work/logs: {error}")
