#!/usr/bin/env python3
"""Process-level MCP tests. Python is a test dependency, not a runtime dependency.

Usage: python3 tests/test_mcp.py /absolute/path/to/symphony-mcp
The fake qxctl is confined to a private temporary directory. No installed
Symphony state, network endpoint, service, or credentials are used.
"""

import hashlib
import json
import os
from pathlib import Path
import select
import subprocess
import sys
import tempfile
import time
import unittest


if len(sys.argv) < 2:
    raise SystemExit("Pass the built symphony-mcp executable as the first argument")
MCP = str(Path(sys.argv[1]).resolve())
REGISTRY_PROTOCOL = "symphony.qxctl.command-registry.v1"
PREFIX = "qxcmd:symphony:"


def digest(data):
    return "sha256:" + hashlib.sha256(data).hexdigest()


def record(name, grammar, mutability="read_only", **extra):
    result = {
        "command_id": PREFIX + name,
        "status": "experimental",
        "introduced_in": "0.1.0-dev",
        "deprecated_in": None,
        "replacement_ids": [],
        "grammar": "qxctl " + grammar,
        "aliases": [],
        "visibility": "public",
        "feature_bindings": [],
        "infrastructure_purpose": "isolated test fixture",
        "backend_operation_ids": [],
        "mutability": mutability,
        "authority_mode": "none" if mutability == "read_only" else "target_host_permission",
        "target_scope": "local",
        "input_protocols": [],
        "output_protocols": [],
        "result_validation_protocols": [],
        "recovery_command_id": None if mutability == "read_only" else PREFIX + "status",
        "noninteractive": True,
        "json_output": True,
    }
    result.update(extra)
    return result


RECORDS = [
    record("status", "status [--json] [--mode VALUE] [--quiet]"),
    record("module.inspect", "module inspect <module-name> [--json] [--tag VALUE...]"),
    record("numbers", "numbers [--json] [--limit UINT64] [--tag VALUE...] [--ttl DURATION]"),
    record("plain", "plain", json_output=False),
    record("mutate", "mutate [--input VALUE] [--json] [--mode VALUE]", "permission_backed_mutation"),
    record("evidence", "evidence [--json]", "evidence_only"),
    record("proposal", "proposal [--json]", "proposal_only"),
    record("hidden", "hidden", "prohibited", visibility="hidden", json_output=False),
]


FAKE_SOURCE = r'''#!PYTHON_EXECUTABLE
import hashlib, json, os, sys, time
from pathlib import Path
records = RECORDS_LITERAL
args = sys.argv[1:]
with open(os.environ["FIXTURE_LOG"], "a", encoding="utf-8") as log:
    log.write(json.dumps(args) + "\n")
if args == ["commands", "manifest", "--json"]:
    value = {
        "protocol": "symphony.qxctl.command-registry.v1", "format_version": 1,
        "registry_kind": "observed", "client_id": "qxctl", "client_version": "fixture-1",
        "client_trust": "unreceipted", "receipt_digest": None,
        "executable_digest": "sha256:" + hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
        "commands": records,
    }
    value["registry_digest"] = "sha256:" + hashlib.sha256(json.dumps(value, sort_keys=True, separators=(",", ":")).encode()).hexdigest()
    print(json.dumps(value))
    raise SystemExit(0)
mode = next((arg.split("=", 1)[1] for arg in args if arg.startswith("--mode=")), "ok")
if mode == "timeout":
    time.sleep(2)
if mode == "overflow":
    sys.stdout.write("X" * 32768)
    sys.stdout.flush()
    raise SystemExit(0)
if mode == "failure":
    print(json.dumps({"outcome": "owner_partial", "argv": args}))
    print("fixture diagnostic", file=sys.stderr)
    raise SystemExit(5)
print(json.dumps({"argv": args, "cwd": os.getcwd(), "fixture_marker": os.environ.get("FIXTURE_MARKER"), "ambient_secret_present": "SYMPHONY_TEST_AMBIENT_SECRET" in os.environ}))
'''


class Session:
    def __init__(self, profile=None, extra=None):
        argv = [MCP]
        if profile is not None:
            argv += ["--config", str(profile)]
        argv += extra or []
        self.process = subprocess.Popen(argv, stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                        stderr=subprocess.PIPE, bufsize=0)
        self.sequence = 0
        self.pending = b""

    def send_raw(self, value):
        self.process.stdin.write(value.encode() + b"\n")
        self.process.stdin.flush()

    def send(self, value):
        self.send_raw(json.dumps(value, separators=(",", ":")))

    def receive(self, timeout=8):
        deadline = time.monotonic() + timeout
        while b"\n" not in self.pending:
            remaining = deadline - time.monotonic()
            if remaining <= 0 or not select.select([self.process.stdout], [], [], remaining)[0]:
                raise AssertionError("MCP response deadline exceeded")
            data = os.read(self.process.stdout.fileno(), 65536)
            if not data:
                error = self.process.stderr.read().decode(errors="replace")
                raise AssertionError("MCP closed before a response: " + error)
            self.pending += data
        line, self.pending = self.pending.split(b"\n", 1)
        return json.loads(line)

    def rpc(self, method, params=None):
        self.sequence += 1
        value = {"jsonrpc": "2.0", "id": self.sequence, "method": method}
        if params is not None:
            value["params"] = params
        self.send(value)
        response = self.receive()
        if response.get("id") != self.sequence:
            raise AssertionError("Unexpected response ID, possibly a notification response: " + repr(response))
        return response

    def initialize(self):
        response = self.rpc("initialize", {
            "protocolVersion": "2025-11-25", "capabilities": {},
            "clientInfo": {"name": "isolated-test", "version": "1"},
        })
        if "result" not in response:
            raise AssertionError("Initialization failed: " + repr(response))
        self.send({"jsonrpc": "2.0", "method": "notifications/initialized"})
        return response["result"]

    def tool(self, name, arguments=None):
        response = self.rpc("tools/call", {"name": name, "arguments": arguments or {}})
        if "result" not in response:
            raise AssertionError("Tool response was not a result: " + repr(response))
        return response["result"]

    def close(self):
        if self.process.poll() is None:
            self.process.stdin.close()
            try:
                self.process.wait(timeout=3)
            except subprocess.TimeoutExpired:
                self.process.terminate()
                try:
                    self.process.wait(timeout=3)
                except subprocess.TimeoutExpired:
                    self.process.kill()
                    self.process.wait(timeout=3)
        for stream in (self.process.stdin, self.process.stdout, self.process.stderr):
            if stream and not stream.closed:
                stream.close()


class MCPIntegration(unittest.TestCase):
    def setUp(self):
        # A short canonical directory works within both Darwin and Linux sun_path.
        tmp_base = "/private/tmp" if Path("/private/tmp").is_dir() else "/tmp"
        self.temp = tempfile.TemporaryDirectory(prefix="s-mcp-", dir=tmp_base)
        self.root = Path(self.temp.name).resolve()
        self.root.chmod(0o700)
        self.addCleanup(self.temp.cleanup)
        self.log = self.root / "argv.jsonl"
        self.fake = self.root / "qxctl-fixture"
        source = FAKE_SOURCE.replace("PYTHON_EXECUTABLE", sys.executable).replace("RECORDS_LITERAL", repr(RECORDS))
        self.fake.write_text(source)
        self.fake.chmod(0o700)
        manifest = json.loads(subprocess.check_output([str(self.fake), "commands", "manifest", "--json"],
                                                      env={"FIXTURE_LOG": str(self.log)}))
        self.config = {
            "schema": "symphony.agent.connection.v1", "qxctl": str(self.fake),
            "qxctl_sha256": digest(self.fake.read_bytes()), "working_directory": str(self.root),
            "registry_digest": manifest["registry_digest"],
            "environment": {"FIXTURE_LOG": str(self.log), "FIXTURE_MARKER": "explicit"},
            "inherit_environment": [], "timeout_ms": 1000, "max_output_bytes": 4096,
            "enabled_nonread_commands": [PREFIX + "mutate", PREFIX + "evidence", PREFIX + "proposal"],
        }
        self.profile = self.root / "connection.json"
        self.write_profile()
        self.log.write_text("")

    def write_profile(self):
        self.profile.write_text(json.dumps(self.config))
        self.profile.chmod(0o600)

    def session(self, initialized=True):
        session = Session(self.profile)
        self.addCleanup(session.close)
        if initialized:
            session.initialize()
        return session

    def calls(self):
        return [row for row in (json.loads(line) for line in self.log.read_text().splitlines())
                if row != ["commands", "manifest", "--json"]]

    def content(self, result):
        self.assertFalse(result.get("isError", False), result)
        return result["structuredContent"]

    def rejected(self, result):
        self.assertTrue(result.get("isError"), result)

    def test_initialization_and_notifications(self):
        session = self.session(initialized=False)
        self.assertIn("error", session.rpc("tools/list"))
        initial = session.initialize()
        self.assertEqual(initial["protocolVersion"], "2025-11-25")
        self.assertEqual(set(initial["capabilities"]), {"tools", "resources", "prompts"})
        session.send({"jsonrpc": "2.0", "method": "notifications/unrecognized", "params": {}})
        self.assertEqual(session.rpc("ping")["result"], {})
        tools = session.rpc("tools/list")["result"]["tools"]
        self.assertEqual(len(tools), 6)
        self.assertTrue(next(t for t in tools if t["name"] == "symphony_read")["annotations"]["readOnlyHint"])
        self.assertFalse(next(t for t in tools if t["name"] == "symphony_apply")["annotations"]["readOnlyHint"])
        self.assertIn("error", session.rpc("initialize", {}))
        self.assertEqual(self.calls(), [])

    def test_discovery_pagination_and_exact_describe(self):
        session = self.session()
        connection = self.content(session.tool("symphony_connection"))
        self.assertEqual(connection["registered_commands"], len(RECORDS))
        self.assertEqual(connection["service_health"], "not_probed")
        first = self.content(session.tool("symphony_discover", {"limit": 2}))
        second = self.content(session.tool("symphony_discover", {"offset": first["next_offset"], "limit": 2}))
        self.assertEqual(len(first["commands"]), 2)
        self.assertTrue({x["command_id"] for x in first["commands"]}.isdisjoint(x["command_id"] for x in second["commands"]))
        found = self.content(session.tool("symphony_discover", {"query": "module inspect"}))
        self.assertEqual([x["command_id"] for x in found["commands"]], [PREFIX + "module.inspect"])
        description = self.content(session.tool("symphony_describe", {"command_id": PREFIX + "module.inspect"}))
        self.assertEqual(description["positionals"], ["module-name"])
        self.assertEqual(description["registry_record"]["authority_mode"], "none")
        self.rejected(session.tool("symphony_describe", {"command_id": PREFIX + "hidden"}))
        self.assertEqual(self.calls(), [])

    def test_read_argv_positionals_repeats_and_no_shell(self):
        session = self.session()
        hostile = "--mode=timeout; $(touch should-not-exist)"
        result = self.content(session.tool("symphony_read", {
            "command_id": PREFIX + "module.inspect", "positionals": [hostile],
            "flags": {"tag": ["first", "a=b; $HOME"]},
        }))
        self.assertEqual(result["parsed_stdout"]["argv"],
                         ["module", "inspect", "--tag=first", "--tag=a=b; $HOME", "--json", "--", hostile])
        self.assertEqual(result["parsed_stdout"]["cwd"], str(self.root))
        self.assertEqual(result["parsed_stdout"]["fixture_marker"], "explicit")
        self.assertFalse((self.root / "should-not-exist").exists())
        self.assertEqual(result["semantic_success"], "not_asserted")
        self.assertFalse(result["retry_automatically"])
        plain = self.content(session.tool("symphony_read", {"command_id": PREFIX + "plain"}))
        self.assertEqual(plain["parsed_stdout"]["argv"], ["plain"])

    def test_live_registry_placeholder_families(self):
        session = self.session()
        result = self.content(session.tool("symphony_read", {
            "command_id": PREFIX + "numbers", "flags": {"limit": "18446744073709551615", "ttl": "5m", "tag": ["one", "two"]},
        }))
        self.assertEqual(result["parsed_stdout"]["argv"], ["numbers", "--limit=18446744073709551615", "--tag=one", "--tag=two", "--ttl=5m", "--json"])

    def test_environment_is_explicit_and_ambient_values_are_not_forwarded(self):
        name = "SYMPHONY_TEST_AMBIENT_SECRET"
        prior = os.environ.get(name)
        os.environ[name] = "synthetic-marker-not-a-credential"
        try:
            session = self.session()
            result = self.content(session.tool("symphony_read", {"command_id": PREFIX + "status"}))
        finally:
            if prior is None:
                os.environ.pop(name, None)
            else:
                os.environ[name] = prior
        self.assertFalse(result["parsed_stdout"]["ambient_secret_present"])
        self.assertEqual(result["parsed_stdout"]["fixture_marker"], "explicit")

    def test_arguments_cannot_select_other_commands(self):
        session = self.session()
        cases = [
            {"command_id": PREFIX + "status", "positionals": ["mutate"]},
            {"command_id": PREFIX + "status", "flags": {"json": False}},
            {"command_id": PREFIX + "status", "flags": {"unknown": "x"}},
            {"command_id": PREFIX + "status", "flags": {"quiet": "false"}},
            {"command_id": PREFIX + "status", "flags": {"mode": 123}},
            {"command_id": PREFIX + "module.inspect", "positionals": []},
            {"command_id": PREFIX + "status", "flags": {"mode": "bad\u0000value"}},
            {"command_id": PREFIX + "status", "unknown": "field"},
        ]
        for case in cases:
            with self.subTest(case=case):
                self.rejected(session.tool("symphony_read", case))
        self.assertEqual(self.calls(), [])

    def test_read_denies_all_nonread_classes(self):
        session = self.session()
        for name in ("mutate", "evidence", "proposal"):
            with self.subTest(name=name):
                self.rejected(session.tool("symphony_read", {"command_id": PREFIX + name}))
        self.assertEqual(self.calls(), [])

    def test_prepare_apply_is_single_use_and_not_owner_authority(self):
        session = self.session()
        preview = self.content(session.tool("symphony_prepare", {"command_id": PREFIX + "mutate", "flags": {"mode": "ok"}}))
        self.assertFalse(preview["authorization_granted"])
        self.assertFalse(preview["owner_plan_executed"])
        self.assertEqual(self.calls(), [])
        result = self.content(session.tool("symphony_apply", {"plan_id": preview["plan_id"]}))
        self.assertEqual(result["parsed_stdout"]["argv"], preview["argv"])
        self.rejected(session.tool("symphony_apply", {"plan_id": preview["plan_id"]}))
        self.assertEqual(len(self.calls()), 1)

    def test_plan_is_session_bound(self):
        first = self.session()
        preview = self.content(first.tool("symphony_prepare", {"command_id": PREFIX + "mutate"}))
        second = self.session()
        self.rejected(second.tool("symphony_apply", {"plan_id": preview["plan_id"]}))
        self.assertEqual(self.calls(), [])

    def test_disabled_nonread_not_dispatchable(self):
        self.config["enabled_nonread_commands"] = []
        self.write_profile()
        session = self.session()
        self.rejected(session.tool("symphony_prepare", {"command_id": PREFIX + "mutate"}))
        self.assertEqual(self.calls(), [])

    def test_changed_direct_input_consumes_plan_without_dispatch(self):
        source = self.root / "request.json"
        source.write_text('{"revision":1}')
        session = self.session()
        preview = self.content(session.tool("symphony_prepare", {"command_id": PREFIX + "mutate", "flags": {"input": str(source)}}))
        self.assertEqual(preview["direct_file_sha256"][str(source)], digest(source.read_bytes()))
        source.write_text('{"revision":2}')
        self.rejected(session.tool("symphony_apply", {"plan_id": preview["plan_id"]}))
        self.rejected(session.tool("symphony_apply", {"plan_id": preview["plan_id"]}))
        self.assertEqual(self.calls(), [])

    def test_changed_executable_rejected_before_dispatch(self):
        session = self.session()
        self.fake.write_text(self.fake.read_text() + "\n# changed binary\n")
        self.rejected(session.tool("symphony_read", {"command_id": PREFIX + "status"}))
        self.assertEqual(self.calls(), [])

    def test_prepared_input_replaced_by_fifo_is_rejected_without_hang(self):
        source = self.root / "request.json"
        source.write_text('{"revision":1}')
        session = self.session()
        preview = self.content(session.tool("symphony_prepare", {"command_id": PREFIX + "mutate", "flags": {"input": str(source)}}))
        source.unlink()
        os.mkfifo(source, 0o600)
        self.rejected(session.tool("symphony_apply", {"plan_id": preview["plan_id"]}))
        self.assertEqual(self.calls(), [])

    def test_preview_preserves_symlink_parent_resolution(self):
        outer = self.root / "outer"
        (outer / "child").mkdir(parents=True)
        (self.root / "link").symlink_to(outer / "child", target_is_directory=True)
        actual = outer / "request.json"
        actual.write_text('{"revision":1}')
        (self.root / "request.json").write_text('{"revision":1}')
        original_spelling = str(self.root) + "/link/../request.json"
        session = self.session()
        preview = self.content(session.tool("symphony_prepare", {"command_id": PREFIX + "mutate", "flags": {"input": original_spelling}}))
        actual.write_text('{"revision":2}')
        self.rejected(session.tool("symphony_apply", {"plan_id": preview["plan_id"]}))
        self.assertEqual(self.calls(), [])

    def test_nonread_timeout_limit_and_nonzero_preserve_uncertainty(self):
        # Leave time for interpreter launch and descriptor isolation; the
        # timeout fixture then deliberately sleeps beyond this bound.
        self.config["timeout_ms"] = 1000
        self.config["max_output_bytes"] = 1024
        self.write_profile()
        session = self.session()
        for mode in ("timeout", "overflow", "failure"):
            with self.subTest(mode=mode):
                preview = self.content(session.tool("symphony_prepare", {"command_id": PREFIX + "mutate", "flags": {"mode": mode}}))
                raw = session.tool("symphony_apply", {"plan_id": preview["plan_id"]})
                self.assertTrue(raw["isError"])
                result = raw["structuredContent"]
                self.assertEqual(result["transport_outcome"], "unknown_requires_owner_reconciliation")
                self.assertFalse(result["retry_automatically"])
                self.assertEqual(result["recovery_command_id"], PREFIX + "status")
                self.assertLessEqual(len(result["stdout"].encode()) + len(result["stderr"].encode()), 1024)
                if mode == "timeout":
                    self.assertTrue(result["timed_out"])
                elif mode == "overflow":
                    self.assertTrue(result["output_limited"])
                    self.assertNotIn("parsed_stdout", result)
                else:
                    self.assertEqual(result["exit_code"], 5)
                    self.assertEqual(result["parsed_stdout"]["outcome"], "owner_partial")
                self.rejected(session.tool("symphony_apply", {"plan_id": preview["plan_id"]}))
        self.assertEqual(len(self.calls()), 3)

    def test_malformed_duplicate_unknown_fields_then_continuation(self):
        session = self.session()
        for value in ('{"broken":', '{"jsonrpc":"2.0","id":90,"id":91,"method":"ping"}'):
            session.send_raw(value)
            self.assertEqual(session.receive()["error"]["code"], -32700)
        session.send({"jsonrpc": "2.0", "id": 92, "method": "ping", "unexpected": True})
        self.assertEqual(session.receive()["error"]["code"], -32600)
        self.assertEqual(session.rpc("not/a/method")["error"]["code"], -32601)
        self.assertEqual(session.rpc("ping")["result"], {})
        self.rejected(session.tool("not_a_tool"))
        self.rejected(session.tool("symphony_connection", {"unexpected": True}))
        self.assertEqual(self.calls(), [])

    def test_prompt_and_resources(self):
        session = self.session()
        resources = session.rpc("resources/list")["result"]["resources"]
        self.assertEqual({item["uri"] for item in resources}, {"symphony://connection", "symphony://usage"})
        connection = session.rpc("resources/read", {"uri": "symphony://connection"})["result"]["contents"][0]
        self.assertEqual(json.loads(connection["text"])["qxctl_sha256"], self.config["qxctl_sha256"])
        self.assertIn("error", session.rpc("resources/read", {"uri": "file:///etc/passwd"}))
        prompts = session.rpc("prompts/list")["result"]["prompts"]
        self.assertEqual([p["name"] for p in prompts], ["symphony-admin"])
        response = session.rpc("prompts/get", {"name": "symphony-admin", "arguments": {"goal": "inspect exact installation"}})
        self.assertIn("inspect exact installation", response["result"]["messages"][0]["content"]["text"])
        self.assertEqual(self.calls(), [])

    def test_invalid_rpc_shape_does_not_echo_unsafe_id_or_break_session(self):
        session = self.session()
        for value in ([1, 2], {"jsonrpc": "2.0", "id": {"unsafe": "id"}, "method": "ping"},
                      {"jsonrpc": 2, "id": 20, "method": "ping"}):
            with self.subTest(value=value):
                session.send(value)
                result = session.receive()
                self.assertEqual(result["error"]["code"], -32600)
                self.assertTrue(result["id"] is None or isinstance(result["id"], (int, str)))
        self.assertEqual(session.rpc("ping")["result"], {})

    def test_startup_rejects_untrusted_profiles(self):
        original = dict(self.config)
        for kind in ("binary_digest", "registry_digest", "permissions", "unknown", "duplicate"):
            with self.subTest(kind=kind):
                self.config = dict(original)
                if kind == "binary_digest":
                    self.config["qxctl_sha256"] = "sha256:" + "0" * 64
                if kind == "registry_digest":
                    self.config["registry_digest"] = "sha256:" + "0" * 64
                if kind == "unknown":
                    self.config["unexpected"] = True
                self.write_profile()
                if kind == "permissions":
                    self.profile.chmod(0o644)
                if kind == "duplicate":
                    value = self.profile.read_text()
                    self.profile.write_text('{"schema":"duplicate",' + value[1:])
                result = subprocess.run([MCP, "--config", str(self.profile)], input=b"", capture_output=True, timeout=5)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(result.stdout, b"")
                self.assertNotEqual(result.stderr, b"")
        self.assertEqual(self.calls(), [])

    def test_optional_private_unix_socket_roundtrip(self):
        socket_path = self.root / "mcp.sock"
        listener = subprocess.Popen([MCP, "--config", str(self.profile), "--listen", str(socket_path)],
                                    stdin=subprocess.DEVNULL, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        def close_listener():
            if listener.poll() is None:
                listener.terminate()
                try:
                    listener.wait(timeout=4)
                except subprocess.TimeoutExpired:
                    listener.kill()
                    listener.wait(timeout=4)
            listener.stdout.close()
            listener.stderr.close()
        self.addCleanup(close_listener)
        deadline = time.monotonic() + 5
        while not socket_path.exists() and listener.poll() is None and time.monotonic() < deadline:
            time.sleep(0.02)
        if listener.poll() is not None:
            message = listener.stderr.read().decode(errors="replace")
            if "Operation not permitted" in message or "EPERM" in message or "local socket bind failed (errno 1)" in message:
                self.skipTest("Explicit EPERM: sandbox forbids Unix-socket binding; rerun with socket permission")
            self.fail("Listener stopped before readiness: " + message)
        self.assertTrue(socket_path.exists(), "Listener readiness timed out")
        self.assertEqual(socket_path.stat().st_mode & 0o777, 0o600)
        session = Session(extra=["--connect-socket", str(socket_path)])
        self.addCleanup(session.close)
        session.initialize()
        result = self.content(session.tool("symphony_read", {"command_id": PREFIX + "status"}))
        self.assertEqual(result["parsed_stdout"]["argv"], ["status", "--json"])
        session.close()
        listener.terminate()
        listener.wait(timeout=4)
        self.assertFalse(socket_path.exists(), "Owned socket should be removed on listener shutdown")


if __name__ == "__main__":
    unittest.main(argv=[sys.argv[0]], verbosity=2)
