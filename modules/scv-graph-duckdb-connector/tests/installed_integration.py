#!/usr/bin/env python3
"""Focused receipt-backed graph-index acceptance. No network or selected-head writes.

Uses the caller's retained native graph; evaluation times are explicit simulations.
Prepared/reopened recovery is separate from native crash-durability tests.
"""
import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import time
import uuid


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False, allow_nan=False).encode()


def digest(value):
    return "sha256:" + hashlib.sha256(canonical(value)).hexdigest()


def seal_valid(value):
    assert value["digest"] == digest({k: v for k, v in value.items() if k != "digest"})


class Campaign:
    def __init__(self, args):
        self.args = args
        self.out = args.out.resolve()
        self.out.mkdir(parents=True, exist_ok=False, mode=0o700)
        self.root = self.out / "index"
        self.root.mkdir(mode=0o700)
        self.tops = str(uuid.uuid4())
        self.namespace = "research-a"
        self.time = "2026-09-13T05:00:00Z"
        self.graph = json.loads(args.graph.read_text())
        self.calls = []
        self.assertions = []
        self.summary = {"status": "running", "qxctl": str(args.qxctl.resolve()),
                        "qxctl_sha256": hashlib.sha256(args.qxctl.read_bytes()).hexdigest(),
                        "graph_digest": self.graph["digest"], "tops_id": self.tops,
                        "network_requests": 0, "selected_head_mutations": 0,
                        "recovery_scope": "Committed native prepare, process exit, new-process qxctl recovery; crash tests are separate.",
                        "query_times": "Explicit simulated valid/expired times against retained evidence, not refreshed retrievals."}
        self.save()

    def save(self):
        for name, value in [("SUMMARY.json", {**self.summary, "calls": len(self.calls), "assertions": len(self.assertions)}),
                            ("COMMANDS.json", self.calls), ("ASSERTIONS.json", self.assertions)]:
            (self.out / name).write_text(json.dumps(value, indent=2, ensure_ascii=False) + "\n")

    def check(self, condition, explanation):
        assert condition, explanation
        self.assertions.append(explanation)
        self.save()

    def execute(self, name, command, *, payload=None, cwd=None, ok=True):
        stdin = None if payload is None else canonical(payload)
        p = subprocess.run(command, input=stdin, capture_output=True, cwd=cwd, timeout=45)
        (self.out / (name + "-stdout.json")).write_bytes(p.stdout)
        (self.out / (name + "-stderr.txt")).write_bytes(p.stderr)
        self.calls.append({"name": name, "command": list(map(str, command)), "cwd": str(cwd) if cwd else None,
                           "exit_code": p.returncode, "expected_success": ok})
        self.save()
        assert (p.returncode == 0) == ok, (name, p.returncode, p.stdout[:1200], p.stderr[:800])
        value = json.loads(p.stdout)
        if ok:
            if value.get("protocol") == "symphony.knowledge.engine-process.v1":
                return value["result"]
            seal_valid(value)
        return value

    def qx(self, name, leaf, value=None, *, namespace=None, tops=None, root=None, owner=None, ok=True):
        command = [str(self.args.qxctl.resolve()), "scv", "graph-index", leaf,
                   "--connector-prefix", str(self.args.connector_prefix.resolve()),
                   "--connector-version", "0.1.0-dev", "--json"]
        if leaf != "inspect":
            command += ["--index-root", str(root or self.root), "--tops-id", tops or self.tops,
                        "--namespace", namespace or self.namespace]
        if value is not None:
            path = self.out / (name + "-input.json")
            path.write_bytes(canonical(value))
            command += ["--input", str(path)]
        if leaf in ("status", "recover"):
            command += ["--operation-id", name.split("__", 1)[-1]]
        if leaf == "import":
            command += ["--domain", "scv", "--prefix", str(owner or self.args.owner_prefix.resolve()),
                        "--version", "0.10.0-dev"]
        return self.execute(name, command, ok=ok)

    def query(self, name, snapshot, *, kind="nodes", filters=None, cursor=None, limit=2, query_time=None, **kwargs):
        return self.qx(name, "query", {"snapshot_digest": snapshot, "kind": kind,
                       "filters": filters or {}, "cursor": cursor, "limit": limit,
                       "query_time": query_time or self.time}, **kwargs)

    def native(self, name, operation, payload, installation, ok=True):
        request = {"protocol": "symphony.knowledge.engine-process.v1", "request_id": str(uuid.uuid4()),
                   "correlation_id": str(uuid.uuid4()), "operation": operation,
                   "target_engine": installation["EngineID"], "deadline_unix_ms": int(time.time() * 1000) + 30000,
                   "payload": payload}
        (self.out / (name + "-request.json")).write_bytes(canonical(request))
        return self.execute(name, [installation["ExecutablePath"]], payload=request, cwd=self.root, ok=ok)

    def clone_owner(self):
        dest = self.out / "original-owner-copy"
        dest.mkdir(mode=0o700)
        relative = Path("share/symphony/receipts/scv-engine/0.10.0-dev/install-receipt.json")
        source = self.args.owner_prefix.resolve()
        receipt = json.loads((source / relative).read_text())
        for item in [x["path"] for x in receipt["files"]] + [str(relative)]:
            target = dest / item
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(source / item, target)
        return dest

    def test_installed_graph_index_recovery_and_retrieval(self):
        self.qx("inspect", "inspect")
        source = {"operation_id": "initial-index", "graph": self.graph, "query_time": self.time}
        first = self.qx("import-initial", "import", source)
        record = first["connector_result"]
        snapshot = record["intent"]["snapshot"]
        sd = snapshot["digest"]
        self.check(snapshot["backend"] == "duckdb" and snapshot["graph"] == self.graph,
                   "Default backend is reported as DuckDB and complete native graph is preserved")
        self.check(record["state"] == "committed" and record["index_verified"], "Import reports only complete committed index")
        self.check(first["owner_evaluation"]["graph_digest"] == self.graph["digest"], "Import binds native semantic evaluation to retained graph")
        retry = self.qx("import-retry", "import", source)
        self.check(retry["connector_result"] == record, "Exact operation retry preserves logical record and snapshot")
        conflict = copy.deepcopy(source)
        conflict["query_time"] = "2026-09-13T05:00:01Z"
        self.qx("operation-reuse-conflict", "import", conflict, ok=False)
        status = self.qx("status__initial-index", "status")
        self.check(status["owner_evaluation"] is None, "Status does not execute semantic owner")
        p1 = self.query("nodes-page-1", sd)["connector_result"]
        p2 = self.query("nodes-page-2", sd, cursor=p1["next_cursor"])["connector_result"]
        expected = sorted(self.graph["native_nodes"], key=lambda x: x["node_id"].encode())
        self.check([r["value"] for r in p1["rows"] + p2["rows"]] == expected[:4], "Native indexed pagination equals original native node values and byte order")
        self.check(p1["matched_count"] == len(expected), "Indexed query reports complete matched count before pagination")
        self.query("cursor-other-query", sd, filters={"kind": expected[0]["kind"]}, cursor=p1["next_cursor"], ok=False)
        badcursor = {"query_digest": p1["next_cursor"]["query_digest"], "after_key": "absent"}
        self.query("cursor-missing-key", sd, cursor=badcursor, ok=False)
        lastcursor = {"query_digest": p1["next_cursor"]["query_digest"], "after_key": expected[-2]["node_id"]}
        last = self.query("nodes-last-page", sd, cursor=lastcursor)["connector_result"]
        self.check(len(last["rows"]) == 1 and last["next_cursor"] is None, "Last page has explicit null continuation")
        claim = self.graph["claims"][0]
        claims = self.query("claim-exact-scope", sd, kind="claims", filters={"scope": claim["scope"]})
        self.check(claims["connector_result"]["rows"][0]["value"] == claim, "Claim retrieval preserves exact scope, supports and dependencies")
        empty = self.query("claim-absence", sd, kind="claims", filters={"subject": "caller-absent-subject"})
        self.check(empty["connector_result"]["matched_count"] == 0 and empty["owner_evaluation"] == first["owner_evaluation"],
                   "Structural absence does not replace complete native semantic evaluation")
        edge = self.graph["native_edges"][0]
        edges = self.query("edge-exact-filter", sd, kind="edges", filters=edge, limit=128)["connector_result"]
        self.check(edges["rows"] == [{"key": digest(edge), "value": edge}], "Structural edge is preserved with canonical identity")
        expired = self.query("explicit-expired-evidence", sd, kind="claims", query_time="2026-09-15T05:00:00Z")
        self.check(expired["connector_result"]["snapshot"] == snapshot and expired["owner_evaluation"] != first["owner_evaluation"],
                   "Explicit later evaluation changes native freshness result without renewing or rewriting snapshot")
        exported = self.qx("export-original", "export", {"snapshot_digest": sd, "query_time": self.time})
        self.check(exported["connector_result"]["snapshot"] == snapshot, "Export retains exact graph and validating installations")
        self.query("namespace-hidden", sd, namespace="research-b", ok=False)
        self.qx("tops-hidden__initial-index", "status", tops=str(uuid.uuid4()), ok=False)
        other = self.qx("namespace-import", "import", source, namespace="research-b")["connector_result"]["intent"]["snapshot"]
        self.check(other["digest"] != sd and other["graph"] == snapshot["graph"], "Namespace changes index identity without changing native graph identity")
        prepared_payload = {"tops_id": self.tops, "namespace": "pending-recovery", "operation_id": "pending-index",
                            "graph": self.graph, "owner": snapshot["owner"], "connector": snapshot["connector"], "query_time": self.time}
        prepared = self.native("native-prepare-and-exit", "prepare", prepared_payload, snapshot["connector"])
        self.check(prepared["state"] == "prepared" and not prepared["index_verified"], "Durable prepare exists independently of published snapshot")
        self.query("unpublished-snapshot", prepared["snapshot_digest"], namespace="pending-recovery", ok=False)
        pending = self.qx("pending-status__pending-index", "status", namespace="pending-recovery")
        self.check(pending["connector_result"] == prepared, "New process observes exact durable prepared intent")
        recovered = self.qx("recover__pending-index", "recover", namespace="pending-recovery")
        self.check(recovered["connector_result"]["state"] == "committed" and recovered["connector_result"]["intent"] == prepared["intent"],
                   "qxctl recovery replays retained owner and commits original intent exactly")
        rerecovered = self.qx("recover-again__pending-index", "recover", namespace="pending-recovery")
        self.check(rerecovered["connector_result"] == recovered["connector_result"], "Repeated recovery produces no replacement snapshot")
        owner = self.clone_owner()
        owned = self.qx("owner-copy-import", "import", {**source, "operation_id": "owner-copy"}, owner=owner)
        copied_snapshot = owned["connector_result"]["intent"]["snapshot"]
        executable = Path(copied_snapshot["owner"]["ExecutablePath"])
        offline = executable.with_name(executable.name + ".offline")
        executable.rename(offline)
        try:
            self.qx("missing-owner-status__owner-copy", "status")
            self.query("missing-owner-query", copied_snapshot["digest"], ok=False)
            self.qx("missing-owner-export", "export", {"snapshot_digest": copied_snapshot["digest"], "query_time": self.time}, ok=False)
            self.qx("missing-owner-recover__owner-copy", "recover", ok=False)
        finally:
            offline.rename(executable)
        self.check(True, "Missing exact owner permits index status but blocks semantic query, export and recovery")
        self.summary.update(status="passed", snapshot_digest=sd, snapshot_counts=record["counts"],
                            original_owner=snapshot["owner"], connector=snapshot["connector"])
        self.save()


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--qxctl", type=Path, required=True)
    p.add_argument("--connector-prefix", type=Path, required=True)
    p.add_argument("--owner-prefix", type=Path, required=True)
    p.add_argument("--graph", type=Path, required=True)
    p.add_argument("--out", type=Path, required=True)
    campaign = Campaign(p.parse_args())
    try:
        campaign.test_installed_graph_index_recovery_and_retrieval()
    except Exception:
        campaign.summary["status"] = "failed"
        campaign.save()
        raise
    print(json.dumps({"status": "passed", "calls": len(campaign.calls), "assertions": len(campaign.assertions)}))


if __name__ == "__main__":
    main()
