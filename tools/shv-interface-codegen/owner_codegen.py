"""Data-driven mechanical interface authoring; never generates semantic admission."""

import argparse
import hashlib
import json
import re
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
REG_KEYS = {
    "protocol",
    "declaration",
    "history",
    "history_digest",
    "go_title",
    "go_stem",
    "cpp_output",
    "go_output",
    "cmake_output",
    "embedded_compatibility_version",
    "embedded_macro",
}
DEC_KEYS = {
    "protocol",
    "module_id",
    "engine_id",
    "namespace",
    "current_version",
    "releases",
    "contract_versions",
    "operations",
    "schemas",
    "companions",
    "embedded_dependencies",
}
OP_KEYS = {
    "engine_operation_id",
    "operation_name",
    "availability",
    "feature_ids",
    "administrative_interactions",
    "administration_disposition",
    "input_protocol",
    "output_protocol",
    "mutability",
    "idempotency",
    "expected_state_required",
    "authorization_requirement",
    "recovery_operation_id",
    "direct_invocation",
    "thermal_path",
}


def require(ok, message):
    if not ok:
        raise ValueError(message)


def read(path):
    def pairs(values):
        result = {}
        for key, value in values:
            require(key not in result, "duplicate JSON key: " + key)
            result[key] = value
        return result

    return json.loads(path.read_text(), object_pairs_hook=pairs)


def digest(value):
    return (
        "sha256:"
        + hashlib.sha256(
            json.dumps(value, sort_keys=True, separators=(",", ":")).encode()
        ).hexdigest()
    )


def local(root, value, exists=True):
    require(
        isinstance(value, str) and re.fullmatch(r"[A-Za-z0-9_./-]+", value),
        "invalid local path",
    )
    p = Path(value)
    require(
        not p.is_absolute() and ".." not in p.parts and p.as_posix() == value,
        "escaping path",
    )
    q = root / p
    require(q.resolve().is_relative_to(root.resolve()), "escaping link")
    require(
        not any(x.is_symlink() for x in [q, *q.parents] if x.is_relative_to(root)),
        "symlink path",
    )
    if exists:
        require(q.is_file(), "missing regular file: " + value)
    return q


def load(root, registration):
    r = read(registration)
    require(
        set(r) == REG_KEYS
        and r["protocol"] == "symphony.shv.interface-registration.v1",
        "registration shape",
    )
    d = read(local(root, r["declaration"]))
    require(
        set(d) == DEC_KEYS and d["protocol"] == "symphony.shv.owner-interface.v2",
        "declaration shape",
    )
    for key in ["module_id", "engine_id"]:
        require(
            re.fullmatch(r"[a-z][a-z0-9-]*", d[key]) is not None, "owner identifier"
        )
    require(
        re.fullmatch(r"[A-Za-z_]\w*(::[A-Za-z_]\w*)*", d["namespace"]) is not None,
        "C++ namespace",
    )
    for key in ["go_title", "go_stem", "embedded_macro"]:
        require(re.fullmatch(r"[A-Za-z_]\w*", r[key]) is not None, "output identifier")
    hp = local(root, r["history"])
    require(
        "sha256:" + hashlib.sha256(hp.read_bytes()).hexdigest() == r["history_digest"],
        "frozen history changed",
    )
    hist = read(hp)["descriptors"]
    current = d["current_version"]
    require(
        re.fullmatch(r"\d+\.\d+\.\d+(?:-dev)?", current) is not None
        and current not in hist,
        "exact new release required",
    )
    require(
        isinstance(d["operations"], list)
        and all(isinstance(o, dict) for o in d["operations"]),
        "operation list required",
    )
    require(
        isinstance(d["contract_versions"], list)
        and all(isinstance(v, str) and v for v in d["contract_versions"]),
        "contract references required",
    )
    ops = d["operations"]
    names = [o["operation_name"] for o in ops]
    require(bool(ops) and len(names) == len(set(names)), "unique operations required")
    require(
        len({o["engine_operation_id"] for o in ops}) == len(ops),
        "duplicate operation identity",
    )
    for op in ops:
        require(set(op) == OP_KEYS, "operation fields differ")
        require(type(op["expected_state_required"]) is bool, "expected-state flag")
        for key in ["feature_ids", "administrative_interactions"]:
            require(
                isinstance(op[key], list) and all(isinstance(v, str) for v in op[key]),
                "operation list",
            )
        for key in OP_KEYS - {
            "feature_ids",
            "administrative_interactions",
            "expected_state_required",
            "recovery_operation_id",
        }:
            require(isinstance(op[key], str), "operation string")
        require(
            op["recovery_operation_id"] is None
            or isinstance(op["recovery_operation_id"], str),
            "recovery identity",
        )
    require(
        set(d["releases"]) == set(hist) | {current} and d["releases"][current] == names,
        "release map differs",
    )
    for version, desc in hist.items():
        require(
            desc["module_id"] == d["module_id"]
            and desc["engine_id"] == d["engine_id"]
            and desc["engine_version"] == version,
            "history owner differs",
        )
        require(
            d["releases"][version] == [o["operation_name"] for o in desc["operations"]],
            "historical release changed",
        )
    if hist:
        last = list(hist.values())[-1]
        require(
            ops == last["operations"]
            and d["contract_versions"] == last["contract_versions"],
            "metadata migration changes semantics",
        )
    require(
        r["embedded_compatibility_version"] is None
        or r["embedded_compatibility_version"] in hist,
        "unproved embedded version",
    )
    if r["embedded_compatibility_version"]:
        require(
            hist[r["embedded_compatibility_version"]]["operations"] == ops,
            "embedded operations differ",
        )
    require(isinstance(d["embedded_dependencies"], list), "dependency list")
    for dep in d["embedded_dependencies"]:
        require(
            set(dep) == {"engine_id", "version"}
            and all(isinstance(x, str) and x for x in dep.values()),
            "dependency identity",
        )
    require(
        len(d["schemas"]) == len(set(d["schemas"]))
        and len(d["companions"]) == len(set(d["companions"])),
        "duplicate resources",
    )
    for path in d["schemas"] + d["companions"]:
        local(root, path)
    for path in d["schemas"]:
        doc = read(local(root, path))

        def visit(x):
            if isinstance(x, dict):
                if "$ref" in x:
                    ref = x["$ref"]
                    require(ref.startswith("#/"), "nonlocal schema reference")
                    target = doc
                    for part in ref[2:].split("/"):
                        target = target[part.replace("~1", "/").replace("~0", "~")]
                for v in x.values():
                    visit(v)
            elif isinstance(x, list):
                for v in x:
                    visit(v)

        visit(doc)
    outputs = [r[k] for k in ["cpp_output", "go_output", "cmake_output"]]
    require(
        len(set(outputs)) == 3
        and not set(outputs)
        & set([r["declaration"], r["history"], *d["schemas"], *d["companions"]]),
        "output collision",
    )
    for path in outputs:
        local(root, path, False)
    return r, d


def outputs(r, d):
    q = lambda x: json.dumps(x, ensure_ascii=True)
    version = "inline constexpr auto version = " + q(d["current_version"]) + ";\n"
    if r["embedded_compatibility_version"]:
        version = (
            "#ifdef "
            + r["embedded_macro"]
            + "\ninline constexpr auto version = "
            + q(r["embedded_compatibility_version"])
            + ";\n#else\n"
            + version
            + "#endif\n"
        )
    cpp = (
        '// Generated by owner_codegen.py; do not edit.\n#pragma once\n#include "symphony/knowledge/engine/operation.hpp"\nnamespace '
        + d["namespace"]
        + " {\nnamespace engine = symphony::knowledge::engine;\n"
        + version
        + "inline std::vector<engine::OperationSpec> interface_operations() { return {\n"
    )
    for o in d["operations"]:
        values = [
            q(o[k]) for k in ["engine_operation_id", "operation_name", "availability"]
        ] + ["false", "true"]
        values += [
            "{" + ",".join(q(v) for v in o[k]) + "}"
            for k in ["feature_ids", "administrative_interactions"]
        ]
        values += [
            q(o[k])
            for k in [
                "administration_disposition",
                "input_protocol",
                "output_protocol",
                "mutability",
                "idempotency",
            ]
        ]
        values += [
            str(o["expected_state_required"]).lower(),
            q(o["authorization_requirement"]),
            q(o["recovery_operation_id"] or ""),
            q(o["direct_invocation"]),
            q(o["thermal_path"]),
        ]
        cpp += "  {" + ", ".join(values) + "},\n"
    cpp += "}; }\n}\n"
    t = r["go_title"]
    go = (
        "// Code generated by owner_codegen.py for "
        + d["module_id"]
        + "; DO NOT EDIT.\npackage knowledgeengine\nconst SHV"
        + t
        + "InterfaceVersion = "
        + q(d["current_version"])
        + "\nconst shv"
        + t
        + "InterfaceDigest = "
        + q(digest(d))
        + "\nvar shv"
        + t
        + "InterfaceOutputs = map[string]string{\n"
    )
    go += (
        "".join(
            q(o["operation_name"]) + ": " + q(o["output_protocol"]) + ",\n"
            for o in d["operations"]
        )
        + "}\nvar shv"
        + t
        + "InterfaceAdmission = map[string]map[string]bool{\n"
    )
    go += (
        "".join(
            q(v) + ": {" + ", ".join(q(n) + ": true" for n in names) + "},\n"
            for v, names in d["releases"].items()
        )
        + "}\n"
    )
    go = subprocess.check_output(["gofmt"], input=go.encode()).decode()
    cm = (
        "# Generated by owner_codegen.py for "
        + d["module_id"]
        + '; do not edit.\nset(SHV_OWNER_VERSION "'
        + d["current_version"].removesuffix("-dev")
        + '")\n'
    )
    for key, var in [
        ("schemas", "SHV_OWNER_SCHEMA_FILES"),
        ("companions", "SHV_OWNER_COMPANIONS"),
    ]:
        cm += (
            "set("
            + var
            + "\n"
            + "".join(' "${SYMPHONY_REPOSITORY_ROOT}/' + p + '"\n' for p in d[key])
            + ")\n"
        )
    return dict(
        zip([r["cpp_output"], r["go_output"], r["cmake_output"]], [cpp, go, cm])
    )


def emit(root, r, d, check=False):
    projected = outputs(r, d)
    # Preflight every destination before any write. Existing output is replaced
    # only when it bears this generator's marker; unrelated files are never adopted.
    for path, body in projected.items():
        p = local(root, path, False)
        if check:
            require(p.is_file() and p.read_text() == body, "generated drift: " + path)
        elif p.exists():
            require(
                p.read_text().splitlines()[0] == body.splitlines()[0],
                "destination belongs to another author",
            )
    if not check:
        for path, body in projected.items():
            p = local(root, path, False)
            p.parent.mkdir(parents=True, exist_ok=True)
            p.write_text(body)


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--registration", type=Path, required=True)
    p.add_argument("--root", type=Path, default=ROOT)
    p.add_argument("--check", action="store_true")
    a = p.parse_args()
    root = a.root.resolve()
    r, d = load(root, a.registration.resolve())
    emit(root, r, d, a.check)
    print(d["module_id"], "checked" if a.check else "generated")


if __name__ == "__main__":
    main()
