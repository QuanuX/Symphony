#!/usr/bin/env python3
"""Qualify one trusted local qxctl or plain SOV executable in a fixture-signed set."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile


def digest(path):
    with path.open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def write(path, data):
    path.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
    with path.open("xb") as stream:
        stream.write(data if isinstance(data, bytes) else (json.dumps(data, indent=2) + "\n").encode())
    path.chmod(0o600)


def qualify(args):
    build = json.loads(args.identity.read_bytes())
    if build["component_id"] not in ("qxctl", "sov-stdio-adapter", "sov-ssh-adapter"):
        raise ValueError("unsupported executable")
    if digest(args.binary) != build["executable_sha256"]:
        raise ValueError("binary does not match the selected local build")
    if not args.work.is_absolute() or args.work.exists() or args.work.is_symlink():
        raise ValueError("new absolute work required")
    work = args.work.parent.resolve(strict=True) / args.work.name
    work.mkdir(mode=0o700)
    root = work / "delivery"
    root.mkdir(mode=0o700)
    logdir = work / "logs"
    logdir.mkdir(mode=0o700)
    component = build["component_id"]
    source = {"commit": build["source_commit"], "tree": build["source_tree"]}
    os_name = "macos" if build["goos"] == "darwin" else build["goos"]
    profile = {"profile_id": "local-executable-" + os_name + "-" + build["goarch"],
               "os": os_name, "architecture": build["goarch"]}
    primary = "payload/" + component
    (root / "payload").mkdir(mode=0o700)
    shutil.copyfile(args.binary, root / primary)
    (root / primary).chmod(0o700)
    inventory = "payload/immutable-files.json"
    write(root / inventory, {"protocol": "symphony.release.immutable-files.v1", "files": [
        {"path": primary, "sha256": digest(root / primary), "size": (root / primary).stat().st_size, "mode": 0o700}]})
    identity = "metadata/build-identity.json"
    write(root / identity, args.identity.read_bytes())
    metadata = "metadata/fixture.json"
    write(root / metadata, {"scope": "local executable fixture", "source": source,
                           "notice_review_complete": False, "install_contract": "explicit selected executable path"})
    choice = {"id": component, "package_id": component, "kind": "go",
              "verification_adapter": "qxctl-files-v1" if component == "qxctl" else "sov-executable-v1",
              "component_versions": {component: build["version"]}, "dependencies": []}
    artifact = {"selection": choice, "source": source, "vendor": None,
                "file_paths": [primary, inventory], "primary_path": primary, "payload_inventory_path": inventory,
                "manifest_path": None, "signature_path": None, "artifact_signer_policy_sha256": None,
                "receipt_paths": [], "install_recipe_path": metadata, "remove_recipe_path": metadata,
                "notice_paths": [metadata], "source_material_paths": [metadata], "build_evidence_paths": [identity]}
    write(work / "selection.json", {"protocol": "symphony.release.delivery-selection.v1", "source": source,
          "profile": profile, "capability_manifest_sha256": digest(root / metadata),
          "dependency_map_sha256": build["dependency_map_sha256"],
          "identity_catalog_sha256": build["release_identity_catalog_sha256"], "artifacts": [choice]})
    write(work / "definition.json", {"set_id": "local-executable-fixture", "file_paths": [primary, inventory, metadata, identity],
                                    "artifacts": [artifact]})

    def run(name, command, expected=0):
        result = subprocess.run([str(v) for v in command], capture_output=True, timeout=90)
        write(logdir / (name + ".log"), result.stdout + result.stderr)
        if result.returncode != expected:
            raise ValueError("executable qualification failed: " + name)

    with tempfile.TemporaryDirectory(prefix="fixture-key-", dir=work) as keydir:
        key = Path(keydir) / "key"
        run("fixture-key", ["ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", key])
        write(work / "allowed-signers", b'fixture namespaces="symphony-delivery-set" ' + key.with_suffix(".pub").read_bytes())
        common = [args.delivery_tool.resolve(strict=True), "-root", root, "-selection", work / "selection.json",
                  "-allowed-signers", work / "allowed-signers", "-profile", profile["profile_id"], "-signer", "fixture"]
        run("prepare", [*common, "-mode", "prepare", "-definition", work / "definition.json"])
        run("sign", ["ssh-keygen", "-Y", "sign", "-f", key, "-n", "symphony-delivery-set", root / "delivery-set.json"])
        inspect = [*common, "-mode", "inspect-executable", "-artifact", component]
        run("inspect", [*inspect, "-report", work / "executable-owner.json"])
        proof = json.loads((work / "executable-owner.json").read_bytes())
        if (not proof["selected_artifact_owner_verified"] or proof["fresh_host_verified"]
                or proof["release_acceptance_complete"] or proof["publication_authorized"]):
            raise ValueError("overclaimed executable qualification")
        run("reused-output", [*inspect, "-report", work / "executable-owner.json"], expected=1)
        with (root / primary).open("ab") as stream:
            stream.write(b"tampered")
        run("tampered-executable", inspect, expected=1)
        shutil.copyfile(args.binary, root / primary)
        (root / primary).chmod(0o700)
        run("restored-set", [*common, "-mode", "verify"])
    write(work / "evidence.json", {"protocol": "symphony.release.executable-qualification.v1",
          "status": "passed_local_source_qualification", "source": source, "profile": profile,
          "component_id": component, "executable_sha256": build["executable_sha256"],
          "owner_report_sha256": digest(work / "executable-owner.json"),
          "cases": ["signed_identity_and_version", "reused_output_refusal", "tamper_refusal", "restored_set"],
          "fixture_key_destroyed": True, "fresh_host_qualified": False,
          "protected_profile_qualified": False, "release_acceptance_complete": False, "publication_authorized": False})
    print(component + ": real executable identity and tamper checks passed; final-host acceptance pending")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    for option in ("binary", "identity", "delivery-tool", "work"):
        parser.add_argument("--" + option, type=Path, required=True)
    try:
        qualify(parser.parse_args())
    except (ValueError, OSError, subprocess.SubprocessError, KeyError) as error:
        raise SystemExit(f"Executable fixture failed; retain work/logs: {error}")
