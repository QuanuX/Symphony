#!/usr/bin/env python3
"""Generate mechanical SCV interface projections; never provider semantics.

Normal builds consume checked-in projections. Python is needed only to author
or check them, and the generator never runs as part of an engine invocation.
"""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path, PurePosixPath
import re
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[3]
MANIFEST = "knowledge/scv/OWNER-INTERFACE.json"
CPP_OUTPUT = "modules/scv-engine/src/interface.generated.inc"
GO_OUTPUT = "tools/qxctl/internal/knowledgeengine/scv_interface_generated.go"
CMAKE_OUTPUT = "cmake/ScvInterface.generated.cmake"
HISTORY = "modules/scv-engine/tests/fixtures/interface-history.v1.json"


class Invalid(ValueError):
    pass


def object_pairs(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise Invalid(f"duplicate JSON key: {key}")
        result[key] = value
    return result


def read_json(path):
    raw = Path(path).read_bytes()
    if len(raw) > 1048576:
        raise Invalid("interface input exceeds 1 MiB")
    return json.loads(raw, object_pairs_hook=object_pairs)


def exact(value, fields, label):
    if not isinstance(value, dict) or set(value) != set(fields.split()):
        raise Invalid(f"unexpected {label} fields")


def word(value, pattern, label):
    if not isinstance(value, str) or not re.fullmatch(pattern, value):
        raise Invalid(f"invalid {label}")
    return value


def unique(values, label):
    if not isinstance(values, list) or not all(isinstance(x, str) for x in values) or len(values) != len(set(values)):
        raise Invalid(f"invalid or duplicate {label}")


def validate(manifest):
    exact(manifest, "protocol current_release domains releases operations schema_catalog", "manifest")
    if manifest["protocol"] != "symphony.scv.owner-interface.v1":
        raise Invalid("unsupported owner interface protocol")
    domains = manifest["domains"]
    if not isinstance(domains, list) or not 1 <= len(domains) <= 64:
        raise Invalid("invalid domain inventory")
    for domain in domains:
        exact(domain,"name introduced_in","domain")
        word(domain["name"], r"(?:scv|schv|scev|(?:schv|scev)-[a-z0-9][a-z0-9-]{0,63})", "domain")
    unique([domain["name"] for domain in domains], "domain")
    releases = manifest["releases"]
    if not isinstance(releases, list) or not 1 <= len(releases) <= 64:
        raise Invalid("invalid release inventory")
    versions = []
    for release in releases:
        exact(release, "version surfaces companions contract_versions", "release")
        version = word(release["version"], r"[0-9]+\.[0-9]+\.[0-9]+-dev", "exact release")
        versions.append(version)
        for field in ("surfaces", "companions", "contract_versions"):
            if not isinstance(release[field], list) or not all(isinstance(x, str) for x in release[field]):
                raise Invalid(f"invalid release {field}")
            unique(release[field], field)
        if any(surface not in ("corpus", "workflow", "artifact", "schema", "interface", "composition_workflow") for surface in release["surfaces"]):
            raise Invalid("unknown adapter surface")
        for companion in release["companions"]:
            word(companion, r"[A-Z][A-Z0-9-]*\.md", "owner companion")
        if not release["companions"]:
            raise Invalid("release lacks owner companions")
        expected = ["knowledge/SPEC.md@v1"] + ["knowledge/scv/" + name + "@v1" for name in release["companions"]]
        if release["contract_versions"] != expected:
            raise Invalid("contract inventory differs from ordered owner companions")
    unique(versions, "release")
    if any(domain["introduced_in"] not in versions for domain in domains):
        raise Invalid("domain names an undeclared release")
    if manifest["current_release"] != versions[-1]:
        raise Invalid("current release must select the last declared exact release")
    operations = manifest["operations"]
    if not isinstance(operations, list) or not 1 <= len(operations) <= 128:
        raise Invalid("invalid operation inventory")
    names, kinds = [], []
    for operation in operations:
        exact(operation, "name introduced_in input_protocol output_protocol interactions mutability expected_state handler artifact", "operation")
        name = word(operation["name"], r"[a-z][a-z0-9]*(?:_[a-z0-9]+)*", "operation name")
        names.append(name)
        if operation["introduced_in"] not in versions:
            raise Invalid("operation names an undeclared release")
        for field in ("input_protocol", "output_protocol"):
            word(operation[field], r"symphony\.[a-z0-9.-]+\.v[1-9][0-9]*", "operation protocol")
        if operation["input_protocol"] != "symphony.scv." + name.replace("_", "-") + "-input.v1":
            raise Invalid("operation input protocol identity mismatch")
        interactions = operation["interactions"]
        if not isinstance(interactions, list) or not interactions or not all(isinstance(x, str) for x in interactions):
            raise Invalid("missing operation interactions")
        unique(interactions, "interaction")
        if any(x not in ("inspect", "discover", "propose", "apply", "recover", "invoke", "query", "validate") for x in interactions):
            raise Invalid("unknown operation interaction")
        if operation["mutability"] not in ("read_only", "proposal_only", "evidence_only") or type(operation["expected_state"]) is not bool:
            raise Invalid("operation broadens mutation authority")
        if operation["handler"] not in ("inspect", "handle_source", "handle_knowledge", "handle_corpus", "handle_interpretation", "handle_coverage", "handle_pack", "handle_composition"):
            raise Invalid("unknown compiled handler")
        artifact = operation["artifact"]
        if artifact is not None:
            exact(artifact, "kind introduced_in", "artifact admission")
            kinds.append(word(artifact["kind"], r"[a-z][a-z0-9_]*", "artifact kind"))
            if artifact["introduced_in"] not in versions or versions.index(artifact["introduced_in"]) < versions.index(operation["introduced_in"]):
                raise Invalid("artifact admission precedes native operation")
            admitted = releases[versions.index(artifact["introduced_in"])]
            if "artifact" not in admitted["surfaces"]:
                raise Invalid("artifact admitted without artifact adapter")
    unique(names, "operation")
    unique(kinds, "artifact kind")
    if names[0] != "inspect" or operations[0]["handler"] != "inspect":
        raise Invalid("inspect must remain the descriptor operation")
    catalog = manifest["schema_catalog"]
    if catalog != "knowledge/scv/schemas/v1/schema-catalog.json":
        raise Invalid("schema catalog must use the owned SCV resource location")
    return manifest


def projection(manifest, version):
    versions = [release["version"] for release in manifest["releases"]]
    if version not in versions:
        raise Invalid("undeclared exact release")
    ordinal = versions.index(version)
    release = manifest["releases"][ordinal]
    operations = []
    for operation in manifest["operations"]:
        if versions.index(operation["introduced_in"]) <= ordinal:
            projected = {key: value for key, value in operation.items() if key not in ("introduced_in", "artifact")}
            artifact = operation["artifact"]
            projected["artifact_kind"] = artifact["kind"] if artifact and versions.index(artifact["introduced_in"]) <= ordinal else None
            operations.append(projected)
    return {"version": version, "domains": [domain["name"] for domain in manifest["domains"] if versions.index(domain["introduced_in"])<=ordinal], "surfaces": release["surfaces"],
            "companions": release["companions"], "contract_versions": release["contract_versions"], "operations": operations}


def release_manifest(manifest, version):
    versions = [release["version"] for release in manifest["releases"]]
    ordinal = versions.index(version)
    result = json.loads(json.dumps(manifest))
    result["current_release"] = version
    result["releases"] = result["releases"][:ordinal + 1]
    result["operations"] = [op for op in result["operations"] if versions.index(op["introduced_in"]) <= ordinal]
    result["domains"] = [domain for domain in result["domains"] if versions.index(domain["introduced_in"])<=ordinal]
    # Later retention is not retroactively part of an earlier release's declaration.
    for op in result["operations"]:
        if op["artifact"] and versions.index(op["artifact"]["introduced_in"]) > ordinal:
            op["artifact"] = None
    return result


def definition_digest(manifest):
    return "sha256:" + hashlib.sha256(json.dumps(manifest, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode()).hexdigest()


def check_history(manifest, root):
    history = read_json(root / HISTORY)
    exact(history, "protocol releases", "frozen history")
    if history["protocol"] != "symphony.scv.owner-interface-history.v1":
        raise Invalid("invalid frozen interface history")
    for previous in history["releases"]:
        if projection(manifest, previous["version"]) != previous:
            raise Invalid(f"frozen interface changed: {previous['version']}")


def schema_inventory(manifest, root):
    catalog_path = root / manifest["schema_catalog"]
    catalog = read_json(catalog_path)
    if not isinstance(catalog, dict) or catalog.get("protocol") != "symphony.scv.schema-catalog.v1" or catalog.get("engine_version") != manifest["current_release"]:
        raise Invalid("catalog does not identify the current exact release")
    entries = catalog.get("entries")
    if not isinstance(entries, list) or not entries:
        raise Invalid("empty schema catalog")
    indexed, files = {}, {catalog_path.name}
    for entry in entries:
        if not isinstance(entry, dict) or not isinstance(entry.get("operations"), list) or not all(isinstance(x, str) for x in entry["operations"]):
            raise Invalid("invalid catalog entry")
        protocol = entry.get("protocol")
        if not isinstance(protocol, str) or protocol in indexed:
            raise Invalid("duplicate or missing catalog protocol")
        indexed[protocol] = entry
        files.add(schema_name(entry.get("file")))
    for operation in projection(manifest, manifest["current_release"])["operations"]:
        for key, kind in (("input_protocol", "input"), ("output_protocol", "output")):
            entry = indexed.get(operation[key])
            if entry is None or entry.get("kind") != kind or operation["name"] not in entry.get("operations", []):
                raise Invalid(f"catalog does not bind {operation['name']} {kind} protocol")
    # Follow schema-local references without URLs or arbitrary filesystem reads.
    documents = {}
    def document(name):
        if name not in documents:
            documents[name] = read_json(catalog_path.parent / schema_name(name))
        return documents[name]
    def resolve(name, fragment):
        value = document(name)
        if fragment == "" or fragment == "#":
            return
        if not isinstance(fragment, str) or not fragment.startswith("#/"):
            raise Invalid("invalid schema fragment")
        for token in fragment[2:].split("/"):
            if re.search(r"~(?![01])", token):
                raise Invalid("invalid schema pointer escape")
            token = token.replace("~1", "/").replace("~0", "~")
            if not isinstance(value, dict) or token not in value:
                raise Invalid("unresolvable schema fragment")
            value = value[token]
        if not isinstance(value, (dict, bool)):
            raise Invalid("schema fragment does not select a schema")
    for entry in entries:
        resolve(entry["file"], entry.get("fragment"))
    pending, seen = list(files - {catalog_path.name}), set()
    while pending:
        name = pending.pop()
        if name in seen:
            continue
        seen.add(name)
        if len(seen) > 64:
            raise Invalid("schema reference closure exceeds 64 files")
        doc = document(name)
        def walk(value):
            if isinstance(value, dict):
                if "$ref" in value:
                    ref = value["$ref"]
                    if not isinstance(ref, str):
                        raise Invalid("invalid schema reference")
                    file_part, marker, fragment = ref.partition("#")
                    target = schema_name(file_part) if file_part else name
                    resolve(target, "#" + fragment if marker else "")
                    if file_part:
                        pending.append(target)
                for child in value.values():
                    walk(child)
            elif isinstance(value, list):
                for child in value:
                    walk(child)
        walk(doc)
    files |= seen
    available = {path.name for path in catalog_path.parent.glob("*.schema.json")}
    if available != files - {catalog_path.name}:
        raise Invalid("schema inventory contains an unreferenced or absent schema file")
    return sorted(files)


def schema_name(name):
    if not isinstance(name, str) or not re.fullmatch(r"[a-z0-9-]+\.schema\.json", name) or PurePosixPath(name).name != name:
        raise Invalid("schema reference escapes owned schema directory")
    return name


def quoted(value):
    return json.dumps(value, ensure_ascii=False)


def render(manifest, root, metadata_only=False):
    validate(manifest)
    check_history(manifest, root)
    current = projection(manifest, manifest["current_release"])
    fingerprint = definition_digest(manifest)
    banner = "Generated by modules/scv-engine/tools/generate_interface.py; DO NOT EDIT."
    cpp = [f"// {banner}", f"// Owner interface SHA-256: {fingerprint}", "const std::vector<Operation>& registry() {", "    static const std::vector<Operation> operations{"]
    for op in current["operations"]:
        interactions = "{" + ", ".join(quoted(x) for x in op["interactions"]) + "}"
        cpp.append("        {" + ", ".join((quoted(op["name"]), quoted(op["input_protocol"]), quoted(op["output_protocol"]), interactions, quoted(op["mutability"]), str(op["expected_state"]).lower(), op["handler"])) + "},")
    cpp += ["    };", "    return operations;", "}", "Json interface_contract_versions() {", "    return Json::array({" + ", ".join(quoted(x) for x in current["contract_versions"]) + "});", "}", ""]
    go = [f"// Code generated by modules/scv-engine/tools/generate_interface.py; DO NOT EDIT.", f"// Owner interface SHA-256: {fingerprint}", "package knowledgeengine", "", "type SCVOperationInterface struct {", "InputProtocol string", "OutputProtocol string", "Interactions []string", "Mutability string", "ExpectedState bool", "}", "type scvReleaseInterface struct {", "operations map[string]bool", "artifacts map[string]bool", "surfaces map[string]bool", "domains map[string]bool", "companions []string", "}", "var scvInterfaceDomains = []string{" + ",".join(quoted(d) for d in current["domains"]) + "}", "var scvInterfaceOperations = map[string]SCVOperationInterface{"]
    for op in manifest["operations"]:
        go.append(quoted(op["name"]) + ": {InputProtocol:" + quoted(op["input_protocol"]) + ",OutputProtocol:" + quoted(op["output_protocol"]) + ",Interactions:[]string{" + ",".join(quoted(x) for x in op["interactions"]) + "},Mutability:" + quoted(op["mutability"]) + ",ExpectedState:" + str(op["expected_state"]).lower() + "},")
    go += ["}", "var scvInterfaceArtifacts = map[string]struct{kind, minimumRelease string}{"]
    for op in manifest["operations"]:
        if op["artifact"]:
            go.append(quoted(op["name"]) + ": {" + quoted(op["artifact"]["kind"]) + "," + quoted(op["artifact"]["introduced_in"]) + "},")
    go += ["}", "var scvInterfaceReleases = map[string]scvReleaseInterface{"]
    for release in manifest["releases"]:
        p = projection(manifest, release["version"])
        go.append(quoted(release["version"]) + ": {operations:map[string]bool{" + ",".join(quoted(op["name"]) + ":true" for op in p["operations"]) + "},artifacts:map[string]bool{" + ",".join(quoted(op["name"]) + ":true" for op in p["operations"] if op["artifact_kind"]) + "},surfaces:map[string]bool{" + ",".join(quoted(s) + ":true" for s in release["surfaces"]) + "},domains:map[string]bool{" + ",".join(quoted(d) + ":true" for d in p["domains"]) + "},companions:[]string{" + ",".join(quoted(x) for x in release["companions"]) + "}},")
    go += ["}", """
// Interface admission is exact and generated; validation behavior remains independent.
func SCVDomains() []string { return append([]string(nil), scvInterfaceDomains...) }
func SCVOperationMetadata(operation string) (SCVOperationInterface, bool) {
 value,ok:=scvInterfaceOperations[operation]; value.Interactions=append([]string(nil),value.Interactions...); return value,ok
}
func SCVResultProtocol(operation string) (string,bool) { value,ok:=scvInterfaceOperations[operation];return value.OutputProtocol,ok }
func SCVOperationSupported(version,operation string) bool { return scvInterfaceReleases[version].operations[operation] }
func SCVSupports(version,surface string) bool { return scvInterfaceReleases[version].surfaces[surface] }
func SCVDomainSupported(version,domain string) bool { return scvInterfaceReleases[version].domains[domain] }
func SCVArtifactKind(operation string) string { return scvInterfaceArtifacts[operation].kind }
func SCVArtifactMinimumRelease(operation string) string { return scvInterfaceArtifacts[operation].minimumRelease }
func SCVArtifactSupported(version,operation string) bool { return scvInterfaceReleases[version].artifacts[operation] }
func SCVOperationInteraction(operation string) string { value:=scvInterfaceOperations[operation];if len(value.Interactions)==0{return ""};return value.Interactions[0] }
func scvInterfaceOperationCount(version string) int { return len(scvInterfaceReleases[version].operations) }
func scvInterfaceCompanions(version string) []string { return append([]string(nil),scvInterfaceReleases[version].companions...) }
"""]
    go += ["func SCVArtifactOperations() []string { return []string{" + ",".join(quoted(op["name"]) for op in manifest["operations"] if op["artifact"]) + "} }"]
    go += ["var scvInterfaceDefinitionDigests = map[string]string{"]
    go += [quoted(release["version"]) + ":" + quoted(definition_digest(release_manifest(manifest,release["version"]))) + "," for release in manifest["releases"] if "interface" in release["surfaces"]]
    go += ["}"]
    formatted = subprocess.run(["gofmt"], input="\n".join(go).encode(), stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=True).stdout
    output = {CPP_OUTPUT: "\n".join(cpp).encode(), GO_OUTPUT: formatted}
    if not metadata_only:
        files = schema_inventory(manifest, root)
        cmake = [f"# {banner}", f"# Owner interface SHA-256: {fingerprint}", "set(SCV_INTERFACE_VERSION " + quoted(manifest["current_release"]) + ")", "set(SCV_INTERFACE_DOMAINS " + " ".join(current["domains"]) + ")", "set(SCV_INTERFACE_SCHEMA_FILES"]
        cmake += ['    "${SYMPHONY_REPOSITORY_ROOT}/knowledge/scv/schemas/v1/' + name + '"' for name in files]
        cmake += [")", "set(SCV_INTERFACE_COMPANION_FILES"]
        cmake += ['    "${SYMPHONY_REPOSITORY_ROOT}/knowledge/scv/' + name + '"' for name in current["companions"]]
        cmake += [")", 'set(SCV_INTERFACE_MANIFEST "${SYMPHONY_REPOSITORY_ROOT}/' + MANIFEST + '")', ""]
        output[CMAKE_OUTPUT] = "\n".join(cmake).encode()
    return output


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="reject missing or changed checked-in projections")
    parser.add_argument("--metadata-only", action="store_true", help="authoring bootstrap only: omit catalog validation/CMake inventory; not a complete drift check")
    parser.add_argument("--root", type=Path, default=ROOT)
    args = parser.parse_args()
    root = args.root.resolve()
    try:
        output = render(read_json(root / MANIFEST), root, args.metadata_only)
        for name, expected in output.items():
            path = root / name
            if args.check:
                if not path.is_file() or path.read_bytes() != expected:
                    raise Invalid("generated interface drift: " + name)
            else:
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_bytes(expected)
        print(json.dumps({"result": "checked" if args.check else "generated", "complete": not args.metadata_only, "files": list(output)}, sort_keys=True))
    except (Invalid, OSError, ValueError, subprocess.CalledProcessError) as error:
        print("SCV interface generation rejected: " + str(error), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
