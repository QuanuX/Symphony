#!/usr/bin/env python3
"""Real, isolated SSIAG/STAV source-adoption specimen; no host service changes."""

import argparse
import datetime
import hashlib
import json
import os
from pathlib import Path
import signal
import stat
import subprocess
import tempfile
import time
import uuid

def now():
    return datetime.datetime.now(datetime.timezone.utc).isoformat()


def digest(raw):
    return "sha256:" + hashlib.sha256(raw).hexdigest()


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode()


def write_json(path, value):
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
    raw = (json.dumps(value, indent=2, sort_keys=True, ensure_ascii=False) + "\n").encode()
    temporary = path.with_name(path.name + ".writing")
    with open(temporary, "wb") as handle:
        os.chmod(temporary, 0o600)
        handle.write(raw)
        handle.flush()
        os.fsync(handle.fileno())
    os.replace(temporary, path)


def read_json(path):
    return json.loads(Path(path).read_text())


class Harness:
    def __init__(self, output, *, repo=None, cli=None, prefix=None):
        self.output = Path(output).resolve()
        self.state_path = self.output / "RUNTIME.json"
        self.state = read_json(self.state_path) if self.state_path.exists() else None
        chosen = {"repository": repo, "qxctl": cli, "scv_prefix": prefix}
        for key, explicit in chosen.items():
            stored = self.state.get(key) if self.state else None
            if explicit is None and stored is None:
                raise RuntimeError(f"explicit {key} selection is required for a fresh specimen")
            value = Path(explicit or stored).resolve()
            if stored and explicit and str(value) != stored:
                raise RuntimeError(f"retained {key} selection differs from the explicit argument")
            setattr(self, {"repository": "repo", "qxctl": "cli", "scv_prefix": "prefix"}[key], value)

    def save(self):
        write_json(self.state_path, self.state)

    def environment(self):
        environment = dict(os.environ)
        for name in list(environment):
            if name.startswith("SYMPHONY_SSIAG_") or name.startswith("SYMPHONY_STAV_"):
                environment.pop(name)
        environment.update(self.state["environment"])
        environment["GOCACHE"] = os.environ.get("GOCACHE", str(self.output / "build/go-cache"))
        return environment

    def command(self, label, argv, *, cwd=None, expected_success=True, timeout=30):
        log_path = self.output / "COMMANDS.json"
        commands = read_json(log_path) if log_path.exists() else []
        index = len(commands) + 1
        stem = f"{index:03d}-{label}"
        started = now()
        completed = subprocess.run([str(x) for x in argv], cwd=cwd or self.repo,
                                   env=self.environment(), capture_output=True, timeout=timeout)
        stdout, stderr = self.output / (stem + "-stdout.json"), self.output / (stem + "-stderr.txt")
        stdout.write_bytes(completed.stdout)
        stderr.write_bytes(completed.stderr)
        os.chmod(stdout, 0o600)
        os.chmod(stderr, 0o600)
        commands.append({"label": label, "argv": [str(x) for x in argv], "cwd": str(cwd or self.repo),
                         "started_at": started, "completed_at": now(), "exit_code": completed.returncode,
                         "expected_success": expected_success, "stdout": str(stdout), "stderr": str(stderr),
                         "stdout_digest": digest(completed.stdout), "stderr_digest": digest(completed.stderr)})
        write_json(log_path, commands)
        if (completed.returncode == 0) != expected_success:
            raise RuntimeError(f"{label}: unexpected exit {completed.returncode}; see {stderr}")
        if not expected_success:
            return {"stdout": str(stdout), "stderr": str(stderr), "exit_code": completed.returncode}
        try:
            return json.loads(completed.stdout)
        except json.JSONDecodeError:
            return completed.stdout.decode()

    def qx(self, label, args, **kwargs):
        if hashlib.sha256(self.cli.read_bytes()).hexdigest() != self.state["qxctl_sha256"]:
            raise RuntimeError("selected qxctl identity changed")
        return self.command(label, [self.cli, *args], **kwargs)

    def setup(self):
        if self.output.exists():
            raise RuntimeError("setup requires a fresh output directory; use start for a retained setup")
        self.output.mkdir(parents=True, mode=0o700)
        runtime = Path(tempfile.mkdtemp(prefix="s12.", dir="/private/tmp" if Path("/private/tmp").is_dir() else "/tmp"))
        os.chmod(runtime, 0o700)
        tops = str(uuid.uuid4())
        config = self.output / "config"
        state = self.output / "state"
        prefix = self.output / "installed"
        for path in [config, state, prefix, self.output / "build", self.output / "source-state"]:
            path.mkdir(mode=0o700)
        ssiag_config = config / "symphony" / tops / "ssiag/config.json"
        stav_config = config / "symphony" / tops / "stav/append-authority.json"
        ssiag_socket = runtime / "symphony" / tops / "ssiag/ssiag.sock"
        stav_socket = runtime / "symphony" / tops / "stav/append.sock"
        ledger = state / "symphony" / tops / "stav/ledger-v1.stavlog"
        uid, gid = os.geteuid(), os.getegid()
        self.state = {
            "protocol": "local.scv.authority-runtime.v1", "created_at": now(), "phase": "initializing",
            "tops_id": tops, "domain": "schv-do", "source_id": "do-app-platform-limits", "locator_id": "docs",
            "source_root": str(self.output / "source-state"), "scv_prefix": str(self.prefix), "scv_version": "0.10.0-dev",
            "repository": str(self.repo), "qxctl": str(self.cli), "qxctl_sha256": hashlib.sha256(self.cli.read_bytes()).hexdigest(),
            "uid": uid, "gid": gid, "subject_id": "scv-maintenance-owner",
            "environment": {"XDG_CONFIG_HOME": str(config), "XDG_STATE_HOME": str(state), "XDG_RUNTIME_DIR": str(runtime),
                            "SYMPHONY_STAV_CONFIG": str(stav_config)},
            "paths": {"prefix": str(prefix), "ssiag_config": str(ssiag_config), "stav_config": str(stav_config),
                      "ssiag_socket": str(ssiag_socket), "stav_socket": str(stav_socket), "ledger": str(ledger)},
            "services": {}, "process_history": [],
            "scope": "Fresh same-UID diagnostic services using real kernel-peer authentication and real STAV storage. No existing host service, account, provider, keyring, grant or source state is changed. This is not distinct-account isolation or production supervision.",
        }
        self.save()
        # No provider can be invoked, and source permissions initially remain absent.
        write_json(ssiag_config, {
            "schema": "symphony.ssiag.config.v1", "mode": "user", "tops": {"id": tops, "name": "SCV source maintenance specimen"},
            "listen": {"network": "unix", "address": str(ssiag_socket)},
            "authentication": {"mechanism": "unix_peer_credentials",
                "service": {"id": "symphony.ssiag.service", "kind": "symphony.identity.service", "uid": uid, "gid": gid},
                "subjects": [{"id": self.state["subject_id"], "kind": "symphony.identity.operator", "uid": uid, "gid": gid}]},
            "authorization": {"default_effect": "deny", "max_capability_seconds": 60, "grants": []}, "providers": [],
        })
        write_json(stav_config, {
            "schema": "symphony.stav.append-authority.config.v1", "mode": "user", "tops_id": tops,
            "listen": {"network": "unix", "address": str(stav_socket)},
            "ledger": {"durability": "fsync-before-receipt", "max_bytes": 1048576, "path": str(ledger),
                       "recovery": "preserve-incomplete-tail", "retention": "preserve_all", "rotation": "disabled"},
            "authentication": {"mechanism": "kernel-peer-credentials",
                "authority": {"uid": uid, "gid": gid, "subject": {"id": "stav-authority", "kind": "symphony.identity.service"}},
                "producers": [{"uid": uid, "gid": gid, "subject": {"id": "ssiag-service", "kind": "symphony.identity.service"},
                               "producer": {"id": "ssiag", "kind": "symphony.stav.producer"},
                               "permissions": [{"event_class": "symphony.ssiag.policy.decision", "operation_id": "symphony.ssiag.authorize"}]}],
                "readers": [{"uid": uid, "gid": gid, "subject": {"id": "scv-maintenance-reader", "kind": "symphony.identity.operator"},
                             "classifications": ["administrative_metadata"]}]},
        })
        for path in [ssiag_config.parent, stav_config.parent, ssiag_socket.parent, stav_socket.parent, ledger.parent]:
            path.mkdir(parents=True, exist_ok=True, mode=0o700)
            os.chmod(path, 0o700)
        modules = {
            "ssiag": ("secure-identity-access-governance", "symphony-ssiag"),
            "stav": ("stav-append-authority", "symphony-stav-append-authority"),
        }
        for name, (module, executable) in modules.items():
            binary = self.output / "build" / executable
            self.command("build-" + name, ["go", "build", "-o", binary, "./cmd/" + executable],
                         cwd=self.repo / "modules" / module, timeout=120)
            install_args = ["package", "install"] if name == "ssiag" else ["install", "--scope", "user"]
            self.command("install-" + name, [binary, *install_args, "--prefix", prefix, "--version", "dev"])
            installed = prefix / "libexec/symphony" / module / "dev" / executable
            receipt = prefix / "share/symphony/receipts" / module / "dev/install-receipt.json"
            value = read_json(receipt)
            assert value["version"] == "dev" and value["protocol"] == "symphony.knowledge.install-receipt.v2"
            assert binary.read_bytes() == installed.read_bytes()
            self.state["services"][name] = {"binary": str(installed), "binary_digest": digest(installed.read_bytes()),
                "receipt": str(receipt), "receipt_file_digest": digest(receipt.read_bytes()), "receipt_digest": value["receipt_digest"],
                "version": "dev", "pid": None}
            self.save()
        self.state["source_commit"] = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=self.repo, text=True).strip()
        self.state["phase"] = "configured"
        self.save()
        self.start()

    def process_matches(self, service):
        pid = service.get("pid")
        if not pid:
            return False
        observed = subprocess.run(["/bin/ps", "-p", str(pid), "-o", "command="], capture_output=True, text=True)
        command = observed.stdout.strip()
        return observed.returncode == 0 and command.startswith(service["binary"] + " serve ") and self.state["tops_id"] in command

    def start_service(self, name):
        service = self.state["services"][name]
        if self.process_matches(service):
            return
        binary = Path(service["binary"])
        if digest(binary.read_bytes()) != service["binary_digest"]:
            raise RuntimeError("service executable changed")
        attempt = 1 + sum(x["service"] == name for x in self.state["process_history"])
        stdout = self.output / f"{name}-serve-{attempt}-stdout.txt"
        stderr = self.output / f"{name}-serve-{attempt}-stderr.txt"
        argv = [str(binary), "serve", "--scope", "user", "--tops-id", self.state["tops_id"],
                "--config", self.state["paths"][name + "_config"]]
        with open(stdout, "wb") as out, open(stderr, "wb") as err:
            process = subprocess.Popen(argv, env=self.environment(), cwd=self.repo,
                                       stdin=subprocess.DEVNULL, stdout=out, stderr=err, start_new_session=True)
        service["pid"] = process.pid
        self.state["process_history"].append({"service": name, "pid": process.pid, "argv": argv,
            "started_at": now(), "stdout": str(stdout), "stderr": str(stderr), "stopped_at": None})
        self.save()
        socket = Path(self.state["paths"][name + "_socket"])
        for _ in range(100):
            if process.poll() is not None:
                raise RuntimeError(f"{name} exited {process.returncode}; inspect {stderr}")
            if socket.exists() and stat.S_ISSOCK(socket.stat().st_mode):
                return
            time.sleep(0.05)
        raise RuntimeError(f"{name} did not create its selected socket")

    def start(self):
        for name in ["stav", "ssiag"]:
            self.start_service(name)
        tops = self.state["tops_id"]
        self.qx("ready-stav", ["stav", "status", "--scope", "user", "--tops-id", tops, "--json"])
        self.qx("ready-ssiag", ["ssiag", "status", "--scope", "user", "--tops-id", tops, "--json"])
        self.state["phase"] = "ready"
        self.save()
        print(json.dumps({"phase": "ready", "runtime": str(self.state_path), "tops_id": tops}))

    def stop_service(self, name):
        service = self.state["services"][name]
        if not self.process_matches(service):
            if Path(self.state["paths"][name + "_socket"]).exists():
                raise RuntimeError("refusing to stop unverified PID with a remaining selected socket")
            return
        os.kill(service["pid"], signal.SIGTERM)
        for _ in range(100):
            if not self.process_matches(service):
                break
            time.sleep(0.05)
        else:
            raise RuntimeError("owned service did not stop; no force-kill performed")
        for event in reversed(self.state["process_history"]):
            if event["service"] == name and event["pid"] == service["pid"] and event["stopped_at"] is None:
                event["stopped_at"] = now()
                break
        service["pid"] = None
        self.save()

    def stop(self, audit_label="final"):
        audit_error = None
        try:
            self.audit(audit_label)
        except Exception as error:
            # Shutdown still acts only on the recorded, identity-checked PIDs.
            # An unavailable final read must not leave our diagnostic services.
            audit_error = str(error)
            write_json(self.output / "FINAL_AUDIT_UNAVAILABLE.json", {"observed_at": now(), "error": audit_error})
        for name in ["ssiag", "stav"]:
            self.stop_service(name)
        self.state["phase"] = "stopped"
        self.state["final_audit_error"] = audit_error
        self.save()
        print(json.dumps({"phase": "stopped", "runtime": str(self.state_path)}))

    def legacy_recovery(self, client):
        """Recover the actual first specimen's pending v1 intent, not a fixture."""
        if self.state["phase"] != "stopped" or (self.output / "ADOPTION.json").exists():
            raise RuntimeError("legacy recovery requires the stopped, unsuccessful original specimen")
        if client is None:
            raise RuntimeError("legacy recovery requires an explicit corrected --cli")
        client = Path(client).resolve()
        client_digest = digest(client.read_bytes())
        retained = self.output / "legacy-recovery"
        retained.mkdir(mode=0o700)  # Refuse accidental repetition or overwrite.
        before_runtime = self.state_path.read_bytes()
        (retained / "RUNTIME-before.json").write_bytes(before_runtime)
        documents = list(Path(self.state["source_root"]).rglob("state.json"))
        if len(documents) != 1:
            raise RuntimeError("legacy specimen source document is ambiguous")
        document_path = documents[0]
        before_bytes = document_path.read_bytes()
        (retained / "source-state-before.json").write_bytes(before_bytes)
        before = json.loads(before_bytes)
        assert before["protocol"] == "symphony.qxctl.scv-source-store.v1" and before["state_digest"] is None
        operation = "inc12-do-onboard"
        attempt = before["operations"][operation]
        assert attempt["status"] == "prepared" and "correlation_id" not in attempt
        try:
            self.start()
            ledger_before = Path(self.state["paths"]["ledger"]).read_bytes()
            args = [client, "scv", "source", "recover", "--domain", self.state["domain"],
                    "--prefix", self.state["scv_prefix"], "--version", self.state["scv_version"],
                    "--tops-id", self.state["tops_id"], "--state-root", self.state["source_root"],
                    "--source-id", self.state["source_id"], "--operation-id", operation, "--json"]
            result = self.command("legacy-source-recover", args)
            assert digest(client.read_bytes()) == client_digest
            after_bytes = document_path.read_bytes()
            (retained / "source-state-after.json").write_bytes(after_bytes)
            after = json.loads(after_bytes)
            assert after["protocol"] == "symphony.qxctl.scv-source-store.v2"
            assert after["operations"][operation]["intent"] == attempt["intent"]
            assert result["attempt"]["status"] == "committed"
            assert result["state_digest"] == attempt["intent"]["transition"]["state_digest"]
            correlation = after["operations"][operation]["correlation_id"]
            assert str(uuid.UUID(correlation)) == correlation
            decision = result["attempt"]["authorization"]
            assert decision["correlation_id"] == correlation
            page = self.audit("legacy-recovery")
            events = [entry for entry in page["entries"] if entry["projection"]["request_id"] == decision["request_id"]]
            assert len(events) == 1 and events[0]["projection"]["outcome"] == "allowed"
            assert events[0]["projection"]["correlation_id"] == correlation
            assert (retained / "source-state-before.json").read_bytes() == before_bytes
            write_json(retained / "VERIFICATION.json", {
                "protocol": "local.scv.real-legacy-source-recovery.v1", "status": "verified", "completed_at": now(),
                "original_cli": self.state["qxctl"], "original_cli_digest": "sha256:" + self.state["qxctl_sha256"],
                "corrected_cli": str(client), "corrected_cli_digest": client_digest,
                "original_runtime_digest": digest(before_runtime), "original_state_digest": digest(before_bytes),
                "result_state_digest": digest(after_bytes), "ledger_before_digest": digest(ledger_before),
                "operation_id": operation, "correlation_id": correlation, "original_intent_unchanged": True,
                "source_generation": after["source"]["generation"], "source_digest": after["state_digest"],
                "real_source_authorization_event": events[0], "source_write_receipt": None,
                "scope": "A genuine previously failed v1 pending operation recovered through the exact original native installation and real isolated SSIAG/STAV; copied before bytes remain immutable."})
        finally:
            self.stop(audit_label="legacy-final")

    def source(self, label, operation, *, value=None, operation_id=None, expected_success=True):
        args = ["scv", "source", operation, "--domain", self.state["domain"], "--prefix", self.state["scv_prefix"],
                "--version", self.state["scv_version"], "--tops-id", self.state["tops_id"],
                "--state-root", self.state["source_root"], "--source-id", self.state["source_id"], "--json"]
        if value is not None:
            path = self.output / (label + "-input.json")
            write_json(path, value)
            args.extend(["--input", path])
        if operation_id:
            args.extend(["--operation-id", operation_id])
        return self.qx(label, args, expected_success=expected_success)

    def audit(self, label):
        tops = self.state["tops_id"]
        query = self.qx(label + "-audit-query", ["stav", "query", "--scope", "user", "--tops-id", tops, "--limit", "100", "--json"])
        verify = self.qx(label + "-audit-verify", ["stav", "verify", "--scope", "user", "--tops-id", tops, "--json"])
        if verify["result"]["state"] != "verified":
            raise RuntimeError("STAV chain verification failed")
        write_json(self.output / (label + "-audit.json"), {"query": query, "verification": verify,
            "ledger_digest": digest(Path(self.state["paths"]["ledger"]).read_bytes())})
        return query

    def adoption(self, source_input):
        if (self.output / "ADOPTION.json").exists():
            raise RuntimeError("adoption already recorded; no implicit replay or overwrite")
        selected = read_json(source_input)
        initial, desired = selected["current"], selected["desired"]
        assert initial["source_id"] == desired["source_id"] == self.state["source_id"]
        old_desired = {key: value for key, value in initial.items() if key not in ["protocol", "digest", "generation", "predecessor_digest"]}
        # These deliberately exercise SCV's caller-selected opaque operation IDs.
        # The authenticated adapter must bind them to STAV's UUID correlation
        # field without silently restricting the caller to UUID operations.
        ids = {name: "inc12-do-" + name for name in ["onboard", "relocate", "stale", "permission"]}
        self.state["operation_ids"] = ids
        self.save()
        onboard_id, relocate_id, stale_id = (ids[name] for name in ["onboard", "relocate", "stale"])
        onboard = self.source("onboard-proposal", "propose", value={"operation_id": onboard_id, "desired": old_desired, "reason": "Explicit isolated adoption of the previously retained official source declaration"})
        assert onboard["source"] == initial
        self.source("onboard-denied", "apply", value=onboard, expected_success=False)
        denied = self.source("denied-source-status", "status", operation_id=onboard_id)
        assert denied["state_digest"] is None and denied["attempt"]["status"] == "prepared"
        denied_audit = self.audit("denied")
        denied_events = denied_audit["entries"]
        assert len(denied_events) == 1 and denied_events[0]["projection"]["outcome"] == "denied"
        tops = self.state["tops_id"]
        policy = self.qx("policy-status", ["ssiag", "policy", "status", "--scope", "user", "--tops-id", tops, "--json"])
        resource = "symphony.scv.source:" + hashlib.sha256(canonical({"tops_id": tops, "domain": self.state["domain"], "source_id": self.state["source_id"]})).hexdigest()
        grants = {"default_effect": "deny", "max_capability_seconds": 60, "grants": [
            {"id": "scv-maintenance-" + kind, "subject_id": self.state["subject_id"], "authority_basis": "granted_permission",
             "operation": "symphony.scv.source." + kind, "resource": resource, "audience": "qxctl", "scope": "tops:" + tops}
            for kind in ["onboard", "relocate"]]}
        policy_input = self.output / "exact-source-grants.json"
        write_json(policy_input, grants)
        proposal = self.qx("policy-propose", ["ssiag", "policy", "propose", "--scope", "user", "--tops-id", tops,
            "--operation-id", ids["permission"], "--expected-policy-digest", policy["policy_digest"],
            "--authority-basis", "host_owner", "--input", policy_input, "--json"])
        proposal_path = self.output / "policy-proposal.json"
        write_json(proposal_path, proposal)
        self.qx("policy-apply", ["ssiag", "policy", "apply", "--scope", "user", "--tops-id", tops, "--input", proposal_path, "--json"])
        first = self.source("onboard-recover", "recover", operation_id=onboard_id)
        assert first["state_digest"] == initial["digest"] and first["attempt"]["status"] == "committed"
        relocation = self.source("relocation-proposal", "propose", value={"operation_id": relocate_id, "desired": desired, "reason": selected["reason"]})
        stale = self.source("stale-proposal", "propose", value={"operation_id": stale_id, "desired": desired, "reason": selected["reason"]})
        assert relocation["change_kind"] == "relocate" and relocation["expected_state_digest"] == initial["digest"]
        self.audit("before-outage")
        self.stop_service("stav")
        self.source("relocation-audit-unavailable", "apply", value=relocation, expected_success=False)
        pending = self.source("pending-source-status", "status", operation_id=relocate_id)
        assert pending["state_digest"] == initial["digest"] and pending["attempt"]["status"] == "prepared"
        self.start_service("stav")
        adopted = self.source("relocation-recover", "recover", operation_id=relocate_id)
        assert adopted["state_digest"] == relocation["source"]["digest"] and adopted["attempt"]["status"] == "committed"
        ledger_before_replay = Path(self.state["paths"]["ledger"]).read_bytes()
        replay = self.source("relocation-committed-replay", "recover", operation_id=relocate_id)
        assert replay == adopted and Path(self.state["paths"]["ledger"]).read_bytes() == ledger_before_replay
        self.source("stale-plan-rejected", "apply", value=stale, expected_success=False)
        final = self.source("adopted-source-status", "status", operation_id=relocate_id)
        assert final["state_digest"] == relocation["source"]["digest"]
        assert final["owner_result"]["source"] == relocation["source"]
        assert Path(self.state["paths"]["ledger"]).read_bytes() == ledger_before_replay
        ledger = self.audit("adoption")
        assert len(ledger["entries"]) == 4
        source_events = [entry for entry in ledger["entries"]
                         if entry["projection"]["target"]["id"] == resource]
        onboard_correlation = first["attempt"]["authorization"]["correlation_id"]
        relocate_correlation = adopted["attempt"]["authorization"]["correlation_id"]
        assert onboard_correlation != relocate_correlation
        assert str(uuid.UUID(onboard_correlation)) == onboard_correlation
        assert str(uuid.UUID(relocate_correlation)) == relocate_correlation
        assert [(entry["projection"]["correlation_id"], entry["projection"]["outcome"])
                for entry in source_events] == [(onboard_correlation, "denied"), (onboard_correlation, "allowed"), (relocate_correlation, "allowed")]
        for entry in source_events:
            projection = entry["projection"]
            assert entry["verification_state"] == "verified"
            assert projection["actor"]["id"] == self.state["subject_id"]
            assert projection["event_class"] == "symphony.ssiag.policy.decision"
            assert projection["operation_id"] == "symphony.ssiag.authorize"
        for result in [first, adopted]:
            decision = result["attempt"]["authorization"]
            matches = [entry for entry in source_events
                       if entry["projection"]["request_id"] == decision["request_id"]]
            assert len(matches) == 1 and matches[0]["projection"]["outcome"] == "allowed"
            assert matches[0]["projection"]["correlation_id"] == decision["correlation_id"]
            assert decision["target"]["resource"] == resource
        adopted_path = self.output / "ADOPTED_SOURCE.json"
        write_json(adopted_path, final["owner_result"]["source"])
        result = {"protocol": "local.scv.real-source-adoption.v1", "status": "verified", "completed_at": now(),
            "runtime": str(self.state_path), "source_input": str(Path(source_input).resolve()), "source_input_digest": digest(Path(source_input).read_bytes()),
            "initial_source_digest": initial["digest"], "adopted_source_digest": final["state_digest"], "adopted_source": str(adopted_path),
            "source_resource": resource, "onboard_operation_id": onboard_id, "relocate_operation_id": relocate_id,
            "onboard_audit_correlation_id": onboard_correlation, "relocate_audit_correlation_id": relocate_correlation,
            "denied_kept_prepared_intent_and_absent_head": True, "grants_applied_through_real_qxctl_ssiag_policy": True,
            "stav_unavailable_kept_prepared_intent_and_initial_head": True, "recovery_committed_successor": True,
            "committed_retry_and_stale_rejection_added_no_audit_event_or_write": True,
            "real_stav_query": ledger, "authorization_audit": "real_ssiag_policy_decision_committed_to_real_stav",
            "source_authorization_events": source_events,
            "source_write_receipt": None, "source_write_evidence": "exact protected source journal transition and committed state",
            "continuity_semantics": "The authored declaration and selected grant do not independently establish publisher authority or content sufficiency."}
        write_json(self.output / "ADOPTION.json", result)
        self.state["phase"] = "adopted"
        self.state["adopted_source"] = str(adopted_path)
        self.state["adopted_source_digest"] = final["state_digest"]
        self.save()
        print(json.dumps({"status": "verified", "adopted_source": str(adopted_path), "runtime": str(self.state_path)}))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("action", choices=["setup", "start", "adoption", "audit", "stop", "legacy-recovery"])
    parser.add_argument("--out", type=Path, required=True)
    parser.add_argument("--repo", type=Path, required=True)
    parser.add_argument("--cli", type=Path, required=True)
    parser.add_argument("--prefix", type=Path, required=True)
    parser.add_argument("--source-input", "--source-plan", type=Path)
    args = parser.parse_args()
    if args.action == "adoption" and args.source_input is None:
        parser.error("adoption requires --source-plan pointing to the selected source-plan input")
    harness = Harness(args.out, repo=args.repo, cli=None if args.action == "legacy-recovery" else args.cli, prefix=args.prefix)
    try:
        if args.action == "setup": harness.setup()
        elif args.action == "start": harness.start()
        elif args.action == "adoption": harness.adoption(args.source_input)
        elif args.action == "audit": harness.audit("inspection")
        elif args.action == "stop": harness.stop()
        elif args.action == "legacy-recovery": harness.legacy_recovery(args.cli)
    except Exception as error:
        if harness.output.exists():
            write_json(harness.output / ("FAILURE-" + datetime.datetime.now().strftime("%Y%m%dT%H%M%S") + ".json"),
                       {"action": args.action, "observed_at": now(), "error": str(error), "runtime": str(harness.state_path)})
        raise


if __name__ == "__main__":
    main()
