#!/usr/bin/env python3
"""Private local qualification of real Go owner packages; no provider/service work."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

OWNERS = [
    ("secure-identity-access-governance", "symphony-ssiag", "ssiag-receipt-v2", ["package"]),
    ("stav-append-authority", "symphony-stav-append-authority", "stav-receipt-v2", []),
    ("accordare-stav-producer", "symphony-accordare-stav-producer", "accordare-receipt-v2", []),
]


def digest(path):
    with path.open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def write(path, value):
    data = (json.dumps(value, indent=2) + "\n").encode() if not isinstance(value, bytes) else value
    path.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
    with path.open("xb") as stream:
        stream.write(data)
    path.chmod(0o600)


def qualify(build_root, work, delivery_tool):
    build_root = build_root.resolve(strict=True)
    delivery_tool = delivery_tool.resolve(strict=True)
    # Inputs are explicitly operator-selected local builds, not downloads.
    if not work.is_absolute() or work.exists() or work.is_symlink():
        raise ValueError("work must be a new absolute directory")
    work = work.parent.resolve(strict=True) / work.name
    if work.is_relative_to(build_root):
        raise ValueError("work must be outside candidate build inputs")
    work.mkdir(mode=0o700)
    root = work / "delivery"
    root.mkdir(mode=0o700)
    home = work / "private-home"
    home.mkdir(mode=0o700)
    logs = work / "logs"
    logs.mkdir(mode=0o700)
    env = {"PATH": os.environ.get("PATH", "/usr/bin:/bin"), "HOME": str(home),
           "XDG_CONFIG_HOME": str(home / "config"), "XDG_STATE_HOME": str(home / "state"),
           "XDG_RUNTIME_DIR": str(home / "runtime"), "LANG": "C"}
    commands = []

    def run(label, args, expected=0):
        output = subprocess.run([str(x) for x in args], env=env, capture_output=True, timeout=120)
        if len(output.stdout) + len(output.stderr) > 16 << 20:
            raise ValueError("unexpected output volume")
        log = logs / (label + ".log")
        write(log, output.stdout + output.stderr)
        commands.append({"label": label, "arguments": [str(x) for x in args],
                         "exit_code": output.returncode, "log_sha256": digest(log)})
        if output.returncode != expected:
            raise ValueError("qualification step failed: " + label)
        return output.stdout

    records = []
    for component, binary, adapter, leading in OWNERS:
        record = json.loads((build_root / component / "owner-tool-build.json").read_bytes())
        identity = json.loads((build_root / "identities" / (component + ".json")).read_bytes())
        tool = build_root / component / "bin/inspect-owner"
        executable = build_root / component / "bin" / binary
        if (record["component_id"] != component or record["verification_adapter"] != adapter
                or record["tool_sha256"] != digest(tool)
                or identity["executable_sha256"] != digest(executable)
                or record["source"]["commit"] != identity["source_commit"]
                or record["source"]["tree"] != identity["source_tree"]
                or record["version"] != identity["version"]
                or record["source_archive_sha256"] != digest(build_root / component / "source.tar")):
            raise ValueError("local build binding mismatch")
        records.append((component, binary, adapter, leading, record, executable))
    source = records[0][4]["source"]
    target = (records[0][4]["os"], records[0][4]["architecture"])
    if any(r[4]["source"] != source or (r[4]["os"], r[4]["architecture"]) != target for r in records):
        raise ValueError("mixed build source or host")
    profile = {"profile_id": "local-owner-" + "-".join(target),
               "os": target[0], "architecture": target[1]}
    choices, artifacts, files = [], [], []
    for name, text in [
        ("install.txt", "Fixture: existing explicit-prefix package install; no enrollment."),
        ("remove.txt", "Fixture: owner uninstall on a disposable copy; preserve sentinel."),
        ("notices.txt", "Local qualification only; actual redistribution notice review remains pending."),
    ]:
        relative = "metadata/" + name
        write(root / relative, text.encode())
        files.append(relative)
    prefix = root / "payload"
    prefix.mkdir(mode=0o700)
    for component, binary, adapter, leading, record, executable in records:
        version = record["version"]
        run(component + "-install", [executable, *leading, "install", "--prefix", prefix, "--version", version])
        primary = f"payload/libexec/symphony/{component}/{version}/{binary}"
        receipt = f"payload/share/symphony/receipts/{component}/{version}/install-receipt.json"
        build = f"metadata/{component}-build.json"
        material = f"metadata/{component}-source.tar"
        shutil.copyfile(build_root / "identities" / (component + ".json"), root / build)
        shutil.copyfile(build_root / component / "source.tar", root / material)
        (root / build).chmod(0o600)
        (root / material).chmod(0o600)
        choice = {"id": component, "package_id": component, "kind": "go",
                  "verification_adapter": adapter, "component_versions": {component: version},
                  "dependencies": (["secure-identity-access-governance", "stav-append-authority"]
                                   if component == "accordare-stav-producer" else [])}
        choices.append(choice)
        artifacts.append({"selection": choice, "source": source, "vendor": None,
                          "file_paths": [primary, receipt], "primary_path": primary,
                          "payload_inventory_path": receipt, "manifest_path": None,
                          "signature_path": None, "artifact_signer_policy_sha256": None,
                          "receipt_paths": [receipt], "install_recipe_path": "metadata/install.txt",
                          "remove_recipe_path": "metadata/remove.txt", "notice_paths": ["metadata/notices.txt"],
                          "source_material_paths": [material], "build_evidence_paths": [build]})
        files.extend([primary, receipt, build, material])
    selection = {"protocol": "symphony.release.delivery-selection.v1", "source": source, "profile": profile,
                 "capability_manifest_sha256": hashlib.sha256(b"local three-owner fixture only").hexdigest(),
                 "dependency_map_sha256": digest(build_root / "mvp-dependencies.json"),
                 "identity_catalog_sha256": digest(build_root / "mvp-release-identities.json"), "artifacts": choices}
    write(work / "selection.json", selection)
    write(work / "definition.json", {"set_id": "local-go-owner-qualification", "file_paths": files, "artifacts": artifacts})
    with tempfile.TemporaryDirectory(prefix="fixture-key-", dir=work) as keydir:
        key = Path(keydir) / "key"
        run("fixture-key", ["ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", key])
        write(work / "fixture-allowed-signers",
              b'fixture namespaces="symphony-delivery-set" ' + key.with_suffix(".pub").read_bytes())
        common = [delivery_tool, "-root", root, "-selection", work / "selection.json",
                  "-allowed-signers", work / "fixture-allowed-signers", "-profile", profile["profile_id"], "-signer", "fixture"]
        run("prepare", [*common, "-mode", "prepare", "-definition", work / "definition.json"])
        run("fixture-sign", ["ssh-keygen", "-Y", "sign", "-f", key, "-n", "symphony-delivery-set", root / "delivery-set.json"])
        reports = []
        for component, binary, adapter, leading, record, executable in records:
            report = work / (component + "-owner.json")
            run(component + "-owner", [*common, "-mode", "inspect-owner", "-artifact", component,
                                      "-owner-tool", build_root / component / "bin/inspect-owner",
                                      "-owner-build", build_root / component / "owner-tool-build.json", "-report", report])
            proof = json.loads(report.read_bytes())
            if (not proof["selected_artifact_owner_verified"] or proof["fresh_host_verified"]
                    or proof["release_acceptance_complete"] or proof["publication_authorized"]):
                raise ValueError("overclaimed owner result")
            reports.append({"component_id": component, "report_sha256": digest(report),
                            "tool_build_sha256": digest(build_root / component / "owner-tool-build.json")})
        # Reject drift through the complete signed set before calling any owner.
        receipt = root / artifacts[0]["receipt_paths"][0]
        original = receipt.read_bytes()
        receipt.write_bytes(b"tampered receipt")
        run("tampered-receipt", [*common, "-mode", "verify"], expected=1)
        receipt.write_bytes(original)
        run("restored-set", [*common, "-mode", "verify"])

    disposable = work / "removal-copy"
    shutil.copytree(prefix, disposable)
    write(disposable / "user-sentinel", b"preserve unrelated user data")
    before = digest(disposable / "user-sentinel")
    for component, binary, adapter, leading, record, executable in reversed(records):
        run(component + "-remove", [executable, *leading, "uninstall", "--prefix", disposable, "--version", record["version"]])
    remaining = sorted(str(p.relative_to(disposable)) for p in disposable.rglob("*") if p.is_file())
    if remaining != ["user-sentinel"] or digest(disposable / "user-sentinel") != before or list(home.iterdir()):
        raise ValueError("removal or no-activation boundary failed")
    result = {"protocol": "symphony.release.go-owner-qualification.v1", "status": "passed_local_source_qualification",
              "source": source, "profile": profile, "delivery_index_sha256": digest(root / "delivery-set.json"),
              "delivery_tool_sha256": digest(delivery_tool), "owner_reports": reports,
              "commands": commands, "sentinel_preserved": True, "operational_state_created": False,
              "fixture_signing_key_destroyed": True, "skips": 0, "fresh_host_qualified": False,
              "provider_qualified": False, "notice_review_complete": False,
              "release_acceptance_complete": False, "publication_authorized": False}
    write(work / "evidence.json", result)
    return result


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--build-root", type=Path, required=True)
    parser.add_argument("--work", type=Path, required=True)
    parser.add_argument("--delivery-tool", type=Path, required=True)
    args = parser.parse_args()
    try:
        result = qualify(args.build_root, args.work, args.delivery_tool)
        print(json.dumps({"status": result["status"], "owner_reports": len(result["owner_reports"]),
                          "release_acceptance_complete": False, "publication_authorized": False}))
    except (ValueError, OSError, subprocess.SubprocessError, KeyError) as error:
        raise SystemExit(f"Local owner qualification failed; retain work/logs: {error}")
