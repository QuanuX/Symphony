#!/usr/bin/env python3
"""Offline, focused .10 qxctl acceptance over two real DO limits captures.

Uses a new private output/store, never acquires URLs or adopts source state.
The deliberately failed capture is synthetic. Query-time boundary cases are
explicit simulations; corpus imports use qxctl's actual local clock.
"""

import argparse
import copy
import datetime as dt
import hashlib
import json
from pathlib import Path
import subprocess
import sys
import time
import uuid


VERSION = "0.10.0-dev"
MAX_AGE = 86400
MEMBER = "limits"
CLAIM = "maintenance-host-local-storage"
QUOTE = "The host instances running App Platform containers do not provide persistent data storage."
CAPTURE_FIELDS = (
    "source", "locator_id", "resolved_uri", "redirects", "observed_at",
    "upstream_revision", "media_type", "body", "completeness", "issues",
)


def utc(value=None):
    return (value or dt.datetime.now(dt.timezone.utc)).replace(microsecond=0).strftime("%Y-%m-%dT%H:%M:%SZ")


def timestamp(value):
    return dt.datetime.strptime(value, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=dt.timezone.utc)


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def encoded(value):
    return (json.dumps(value, ensure_ascii=False, indent=2, allow_nan=False) + "\n").encode()


def load(path):
    def unique(pairs):
        value = {}
        for key, item in pairs:
            if key in value:
                raise ValueError(f"duplicate JSON key: {key}")
            value[key] = item
        return value
    return json.loads(Path(path).read_bytes(), object_pairs_hook=unique)


def desired(source):
    return {key: copy.deepcopy(value) for key, value in source.items()
            if key not in ("protocol", "generation", "predecessor_digest", "digest")}


def source_tuple(capture):
    source = capture["source"]
    return [source["family_id"], source["provider_id"], source["source_id"], capture["locator_id"]]


class Campaign:
    def __init__(self, args):
        self.args = args
        self.out = args.out.resolve()
        self.out.mkdir(parents=True, exist_ok=False, mode=0o700)
        self.tops = str(uuid.uuid4())
        self.commands, self.assertions = [], []
        self.summary = {
            "status": "running", "started_at": utc(), "native_version": VERSION,
            "domain": args.domain, "tops_id": self.tops, "network_requests": 0,
            "synthetic_failure": "Authored fixture only; no live request or provider outage is asserted.",
            "recovery_scope": "Completed-job exact replay; no interrupted-write recovery is claimed.",
            "qxctl": str(args.qxctl.resolve()), "qxctl_sha256": sha(args.qxctl.read_bytes()),
            "prefix": str(args.prefix.resolve()), "script_sha256": sha(Path(__file__).read_bytes()),
            "old_capture_file_sha256": sha(args.old_capture.read_bytes()),
            "fresh_capture_file_sha256": sha(args.fresh_capture.read_bytes()),
        }
        self.save()

    def save(self):
        self.summary["commands"] = len(self.commands)
        self.summary["assertions"] = len(self.assertions)
        for name, value in (("COMMANDS.json", self.commands), ("ASSERTIONS.json", self.assertions), ("SUMMARY.json", self.summary)):
            (self.out / name).write_bytes(encoded(value))

    def check(self, label, condition):
        self.assertions.append({"assertion": label, "passed": bool(condition)})
        self.save()
        if not condition:
            raise AssertionError(label)

    def call(self, name, command, value=None, *, corpus=False, error=None):
        input_path = self.out / f"{name}-input.json"
        if value is not None:
            input_path.write_bytes(encoded(value))
        argv = [str(self.args.qxctl.resolve()), "scv", *command,
                "--domain", self.args.domain, "--version", VERSION,
                "--prefix", str(self.args.prefix.resolve()), "--json"]
        if value is not None:
            argv += ["--input", str(input_path)]
        if corpus:
            argv += ["--corpus-root", str(self.out / "corpus-store"), "--tops-id", self.tops]
        start = time.monotonic()
        result = subprocess.run(argv, cwd=self.out, capture_output=True, timeout=180, check=False)
        stdout, stderr = self.out / f"{name}-stdout.json", self.out / f"{name}-stderr.txt"
        stdout.write_bytes(result.stdout)
        stderr.write_bytes(result.stderr)
        self.commands.append({
            "name": name, "argv": argv, "cwd": str(self.out), "exit_code": result.returncode,
            "expected_error": error, "elapsed_seconds": round(time.monotonic() - start, 6),
            "input": input_path.name if value is not None else None,
            "stdout": stdout.name, "stderr": stderr.name,
            "input_file_digest": sha(input_path.read_bytes()) if value is not None else None,
            "stdout_file_digest": sha(result.stdout), "stderr_file_digest": sha(result.stderr),
            "input_bytes": input_path.stat().st_size if value is not None else 0,
            "stdout_bytes": len(result.stdout),
        })
        self.save()
        parsed = load(stdout)
        if error is None:
            self.check(name + ": successful owner/adapter execution", result.returncode == 0)
        else:
            self.check(name + ": published rejection", result.returncode != 0 and
                       parsed.get("protocol") == "symphony.qxctl.error.v1" and
                       parsed.get("error", {}).get("code") == error)
        return parsed

    def imported(self, name, capture, previous=None, corpus_id=None, error=None):
        return self.call(name, ["corpus", "import"], {
            "operation_id": name, "corpus_id": corpus_id or name,
            "previous_snapshot_digest": previous,
            "members": [{"member_id": MEMBER, "capture": capture}],
        }, corpus=True, error=error)

    def query(self, name, snapshot, selection, query_time):
        return self.call(name, ["corpus", "query"], {
            "snapshot_digest": snapshot, "member_ids": [MEMBER], "selection": selection,
            "query_time": query_time, "max_age_seconds": MAX_AGE,
        }, corpus=True)

    def member(self, name, result, capture, freshness, source_matches, reasons):
        self.check(name + ": exactly selected member", len(result["members"]) == 1 and result["members"][0]["member_id"] == MEMBER)
        member = result["members"][0]
        selected = member["selected"]
        self.check(name + ": preserved capture/source/time", selected["capture_digest"] == capture["digest"] and
                   selected["source_digest"] == capture["source"]["digest"] and selected["observed_at"] == capture["observed_at"])
        self.check(name + ": independent disposition and freshness", member["status"] == capture["completeness"] and
                   member["freshness"] == freshness and member["source_revision_matches_latest"] is source_matches)
        self.check(name + ": exact reasons", member["reasons"] == reasons)

    def graph(self, name, capture, claim=True):
        assertions = []
        if claim:
            self.check(name + ": independently selected literal anchor exists", QUOTE in capture["body"])
            assertions = [{
                "claim_id": CLAIM, "subject": "do.app-platform.container", "predicate": "local-storage.persistent",
                "value": {"type": "boolean", "value": False, "unit": None},
                "scope": {"provider": "do", "service": "app-platform", "storage": "host-local-filesystem"},
                "statement_kind": "documented_fact", "evidence": [{"capture_digest": capture["digest"], "quote": QUOTE}],
                "dependencies": [], "valid_from": None, "valid_until": None,
            }]
        knowledge = self.call(name + "-interpret", ["interpret"], {
            "captures": [capture], "claims": assertions, "interpreter_version": "source-maintenance-acceptance-v1",
            "selection_policy": {"policy_id": "source-maintenance-one-day", "max_age_seconds": MAX_AGE,
                                 "partial_capture": "exclude", "allowed_statement_kinds": ["documented_fact"]},
        })
        return self.call(name + "-graph", ["graph"], {"knowledge": [knowledge]})

    def run(self):
        old, fresh = load(self.args.old_capture), load(self.args.fresh_capture)
        now = utc()
        self.check("source1/source2 exact selected chain", source_tuple(old) == source_tuple(fresh) ==
                   ["schv", "do", "do-app-platform-limits", "docs"] and old["source"]["generation"] == 1 and
                   fresh["source"]["generation"] == 2 and fresh["source"]["predecessor_digest"] == old["source"]["digest"])
        self.check("both supplied captures are complete and nonfuture", all(c["completeness"] == "complete" and
                   timestamp(c["observed_at"]) <= timestamp(now) for c in (old, fresh)))
        self.check("historical capture is actually expired at campaign time", timestamp(now) > timestamp(old["observed_at"]) + dt.timedelta(seconds=MAX_AGE))
        self.call("owner-inspect", ["inspect"])
        for name, capture in (("old", old), ("fresh", fresh)):
            replay = self.call(name + "-capture-replay", ["capture-import"], {key: capture[key] for key in CAPTURE_FIELDS})
            self.check(name + ": native offline import reproduces exact capture", replay == capture)

        failure = self.call("synthetic-failure-import", ["capture-import"], {
            "source": fresh["source"], "locator_id": fresh["locator_id"], "resolved_uri": fresh["resolved_uri"],
            "redirects": [], "observed_at": utc(), "upstream_revision": None, "media_type": fresh["media_type"],
            "body": "", "completeness": "failed",
            "issues": ["Synthetic offline acceptance failure; no live request was made and no provider outage is asserted."],
        })
        self.check("synthetic failure preserves source2 and has no body", failure["source"] == fresh["source"] and
                   failure["completeness"] == "failed" and failure["byte_size"] == 0 and failure["body"] == "")
        failed_time = failure["observed_at"]
        boundary = utc(timestamp(fresh["observed_at"]) + dt.timedelta(seconds=MAX_AGE))
        expired = utc(timestamp(fresh["observed_at"]) + dt.timedelta(seconds=MAX_AGE + 1))
        self.summary["query_times"] = {"historical_expiry_actual_time": failed_time,
                                        "fresh_inclusive_boundary_simulation": boundary,
                                        "fresh_first_expired_second_simulation": expired}
        self.summary["identities"] = {"source_tuple": source_tuple(fresh), "old_source": old["source"]["digest"],
                                       "fresh_source": fresh["source"]["digest"], "old_capture": old["digest"],
                                       "fresh_capture": fresh["digest"], "synthetic_failure": failure["digest"]}
        previous_reasons = ["latest_attempt_failed", "selected_previous_complete"]
        snapshots = {}
        for name, capture in (("historical", old), ("fresh", fresh)):
            first = self.imported(name + "-base", capture, corpus_id=name)
            second = self.imported(name + "-failed", failure, first["snapshot_digest"], name)
            recovered = self.call(name + "-recover", ["corpus", "recover"], {"operation_id": name + "-failed"}, corpus=True)
            self.check(name + ": completed recovery returns exact original result", recovered == second)
            inspected = self.call(name + "-inspect", ["corpus", "inspect"], {"snapshot_digest": second["snapshot_digest"]}, corpus=True)
            self.check(name + ": retained snapshot reconstructs exactly", inspected == second["snapshot"])
            self.check(name + ": stable member tuple and previous complete", inspected["generation"] == 2 and
                       inspected["parent_digest"] == first["snapshot_digest"] and len(inspected["members"]) == 1 and
                       inspected["members"][0]["member_id"] == MEMBER and
                       inspected["members"][0]["last_complete"] == first["snapshot"]["members"][0]["last_complete"])
            self.check(name + ": actual snapshot clock did not rewrite capture age", timestamp(failure["observed_at"]) <=
                       timestamp(inspected["snapshot_time"]) <= timestamp(utc()))
            latest = self.query(name + "-latest-failed", second["snapshot_digest"], "latest_attempt", failed_time)
            self.member(name + " latest", latest, failure, "current", True, ["latest_attempt_failed"])
            when = failed_time if name == "historical" else boundary
            selected = self.query(name + "-last-complete", second["snapshot_digest"], "last_complete", when)
            reasons = previous_reasons + (["source_revision_differs_from_latest_attempt", "maximum_age_exceeded"] if name == "historical" else [])
            self.member(name + " retained", selected, capture, "expired" if name == "historical" else "current", name == "fresh", reasons)
            exported = self.call(name + "-export", ["corpus", "export"], {
                "snapshot_digest": second["snapshot_digest"], "member_ids": [MEMBER], "selection": "last_complete",
                "query_time": when, "max_age_seconds": MAX_AGE,
            }, corpus=True)
            self.check(name + ": exported old evidence was not relabeled", exported["query"] == selected and exported["captures"] == [capture])
            difference = self.call(name + "-diff", ["corpus", "diff"], {
                "before_snapshot_digest": first["snapshot_digest"], "after_snapshot_digest": second["snapshot_digest"],
            }, corpus=True)
            self.check(name + ": independently expected refresh axes", difference["added_member_ids"] == [] and
                       difference["removed_member_ids"] == [] and difference["affected_member_ids"] == [MEMBER] and
                       difference["changes"] == [{"member_id": MEMBER, "body_changed": True,
                           "source_changed": name == "historical", "observation_changed": True,
                           "coverage_changed": True, "last_complete_changed": False}])
            snapshots[name] = second
        later = self.query("fresh-expired", snapshots["fresh"]["snapshot_digest"], "last_complete", expired)
        self.member("fresh later", later, fresh, "expired", True, previous_reasons + ["maximum_age_exceeded"])

        graph = self.graph("fresh-evidence", fresh)
        for name, query_time, expected in (("inclusive", boundary, "supported"), ("expired", expired, "unsupported")):
            finding = self.call("evidence-" + name, ["query"], {"graph": graph, "query_time": query_time, "claim_ids": [CLAIM]})
            self.check(name + ": same graph evidence ages without changed caller claim", finding["graph_digest"] == graph["digest"] and
                       len(finding["findings"]) == 1 and finding["findings"][0]["status"] == expected and
                       finding["findings"][0]["claim"]["value"] == {"type": "boolean", "unit": None, "value": False})
        empty_graph = self.graph("failed-evidence", failure, claim=False)
        empty = self.call("failed-evidence-query", ["query"], {"graph": empty_graph, "query_time": failed_time, "claim_ids": [CLAIM]})
        self.check("failed current acquisition supplies no positive assertion", empty["findings"] == [])

        for field in ("source_id", "provider_id"):
            replacement = desired(fresh["source"])
            replacement[field] += "-replacement"
            rejected = self.call("reject-" + field, ["source-plan"], {
                "operation_id": "reject-" + field, "current": fresh["source"], "desired": replacement,
                "reason": "Synthetic negative: identity replacement cannot reuse an existing source chain.",
            }, error="engine_rejected")
            self.check(field + ": chain identity rejection", rejected["error"]["engine_code"] == "scv.identity_change")
        relocated = desired(fresh["source"])
        next(locator for locator in relocated["locators"] if locator["locator_id"] == fresh["locator_id"])["locator_id"] += "-other"
        plan = self.call("other-locator-plan", ["source-plan"], {
            "operation_id": "other-locator", "current": fresh["source"], "desired": relocated,
            "reason": "Synthetic negative: a different locator ID is a different corpus tuple, even at the same URI.",
        })
        alternate = {key: failure[key] for key in CAPTURE_FIELDS}
        alternate.update(source=plan["source"], locator_id=fresh["locator_id"] + "-other")
        alternate = self.call("other-locator-capture", ["capture-import"], alternate)
        self.imported("reject-locator-rebinding", alternate, snapshots["fresh"]["snapshot_digest"], "fresh", error="command_failed")
        unchanged = self.call("post-rejection-inspect", ["corpus", "inspect"], {"snapshot_digest": snapshots["fresh"]["snapshot_digest"]}, corpus=True)
        self.check("rejected member rebinding preserves the prior immutable snapshot", unchanged == snapshots["fresh"]["snapshot"])

        if self.args.authority_result:
            raw = self.args.authority_result.read_bytes()
            (self.out / "supplied-authority-result.json").write_bytes(raw)
            authority = load(self.args.authority_result)
            self.check("supplied protected-state report selects the exact fresh source", authority.get("protocol") ==
                       "symphony.qxctl.scv-source-result.v1" and authority.get("state_digest") == fresh["source"]["digest"] and
                       authority.get("source_id") == fresh["source"]["source_id"])
            self.summary["authority_comparison"] = {"file_sha256": sha(raw), "state_digest": authority["state_digest"],
                "scope": "Compared supplied report only; this offline runner does not replay SSIAG authority or query protected state."}
        else:
            self.summary["authority_comparison"] = None
        self.summary.update(status="pass", completed_at=utc(), successes=sum(c["exit_code"] == 0 for c in self.commands),
                            expected_rejections=sum(c["expected_error"] is not None for c in self.commands))
        self.save()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for flag in ("qxctl", "prefix", "old-capture", "fresh-capture", "out"):
        parser.add_argument("--" + flag, type=Path, required=True)
    parser.add_argument("--authority-result", type=Path)
    parser.add_argument("--domain", default="scv", help="exact installed parent domain; default scv permits identity-replacement rejection")
    args = parser.parse_args()
    if args.domain not in ("scv", "schv"):
        parser.error("use scv or schv so replacement-provider tests reach the identity boundary")
    campaign = Campaign(args)
    try:
        campaign.run()
    except Exception as error:
        campaign.summary.update(status="fail", completed_at=utc(), error=f"{type(error).__name__}: {error}")
        campaign.save()
        print(campaign.summary["error"], file=sys.stderr)
        return 1
    print(json.dumps(campaign.summary, indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
