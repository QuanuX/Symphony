#!/usr/bin/env python3
"""Retain observations from original CodeQL SARIF; never declare security clearance."""
import argparse
import hashlib
import json
import pathlib
import re


def summarize(root, commit, category):
    if not re.fullmatch(r"[0-9a-f]{40}", commit):
        raise ValueError("an exact source commit is required")
    paths = sorted(root.glob("*.sarif"))
    if not paths:
        raise ValueError("original analyzer SARIF is absent")
    files = []
    for path in paths:
        if path.is_symlink() or not path.is_file() or path.stat().st_size > 256 * 1024 * 1024:
            raise ValueError("SARIF must be a bounded regular file")
        raw = path.read_bytes()
        value = json.loads(raw)
        if value.get("version") != "2.1.0" or not value.get("runs"):
            raise ValueError("invalid or empty SARIF document")
        runs = []
        for run in value["runs"]:
            automation_id = run.get("automationDetails", {}).get("id")
            if automation_id and automation_id.rstrip("/") != category.rstrip("/"):
                raise ValueError("SARIF category differs from the selected job category")
            provenance = run.get("versionControlProvenance", [])
            if any(p.get("revisionId") not in (None, commit) for p in provenance):
                raise ValueError("SARIF provenance differs from the checked-out commit")
            invocations = run.get("invocations", [])
            notifications = []
            for invocation in invocations:
                for field in ("toolExecutionNotifications", "toolConfigurationNotifications"):
                    notifications.extend(invocation.get(field, []))
            runs.append({
                "tool": run.get("tool", {}),
                "automation_details": run.get("automationDetails", {}),
                "version_control_provenance": provenance,
                "provenance_present_in_original_sarif": bool(provenance),
                "result_count": len(run.get("results", [])),
                "artifact_count": len(run.get("artifacts", [])),
                "invocations": invocations,
                "diagnostic_notification_count": len(notifications),
                "diagnostic_notifications": notifications,
                "properties": run.get("properties", {}),
                "coverage_completeness": "not_asserted",
            })
        files.append({"file": path.name, "bytes": len(raw),
                      "sha256": hashlib.sha256(raw).hexdigest(), "runs": runs})
    return {"format_version": 1, "source_commit": commit, "category": category,
            "status": "evidence_collected", "security_clearance": "not_asserted",
            "original_sarif": files,
            "limits": ["A successful analysis is not zero vulnerabilities.",
                       "No missing diagnostic or metric is interpreted as complete coverage.",
                       "PR filtering does not close an unchanged default-branch alert.",
                       "C++26 is outside the published CodeQL language support matrix."]}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--sarif-root", required=True, type=pathlib.Path)
    parser.add_argument("--output", required=True, type=pathlib.Path)
    parser.add_argument("--commit", required=True)
    parser.add_argument("--category", required=True)
    parser.add_argument("--ref", required=True)
    parser.add_argument("--language", required=True)
    parser.add_argument("--runner", required=True)
    parser.add_argument("--build-mode", required=True)
    args = parser.parse_args()
    result = summarize(args.sarif_root, args.commit, args.category)
    result.update({"ref": args.ref, "language": args.language,
                   "runner": args.runner, "build_mode": args.build_mode})
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
    print("Collected original CodeQL evidence; security and full coverage are not asserted")


if __name__ == "__main__":
    main()
