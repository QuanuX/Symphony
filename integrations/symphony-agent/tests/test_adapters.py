"""Client fragment behavior; no host installation or runtime is required."""

import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import tomllib
import unittest


ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / "scripts" / "configure_clients.py"


class AdapterTests(unittest.TestCase):
    def invoke(self, client, *extra):
        return subprocess.run(
            [sys.executable, str(SCRIPT), "--client", client, *extra],
            text=True, capture_output=True, check=False,
        )

    def test_json_profiles_preserve_arguments_as_data(self):
        binary = '/opt/Alpha Build/quo"te/$literal/symphony-mcp'
        config = '/tmp/input $(literal)/connection.json'
        for client in ("claude", "antigravity", "generic"):
            with self.subTest(client=client):
                result = self.invoke(client, "--binary", binary, "--config", config)
                self.assertEqual(result.returncode, 0, result.stderr)
                server = json.loads(result.stdout)["mcpServers"]["symphony"]
                self.assertEqual(server["command"], binary)
                self.assertEqual(server["args"], ["--config", config])
                self.assertEqual(server.get("type"), "stdio" if client == "claude" else None)

    def test_toml_roundtrip_and_socket_selection(self):
        binary = '/opt/Native "Build"/symphony-mcp'
        socket = "/tmp/alpha IPC/bridge.sock"
        result = self.invoke("codex", "--binary", binary, "--connect-socket", socket)
        self.assertEqual(result.returncode, 0, result.stderr)
        server = tomllib.loads(result.stdout)["mcp_servers"]["symphony"]
        self.assertEqual(server["command"], binary)
        self.assertEqual(server["args"], ["--connect-socket", socket])
        self.assertEqual(server["default_tools_approval_mode"], "writes")

    def test_reject_ambiguous_or_unsafe_paths(self):
        for path in ("relative/binary", "/", "/a/../b", "/a\nb", "/a\x00b"):
            # NUL cannot be passed through the OS argv boundary.
            if "\x00" in path:
                continue
            with self.subTest(path=path):
                result = self.invoke("generic", "--binary", path, "--config", "/tmp/config.json")
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(result.stdout, "")
        result = self.invoke("generic", "--binary", "/opt/bridge", "--config", "/tmp/config", "--connect-socket", "/tmp/socket")
        self.assertNotEqual(result.returncode, 0)

    def test_exclusive_private_output_and_no_overwrite(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "client.json"
            args = ("--binary", "/opt/bridge", "--config", "/tmp/config", "--output", str(output))
            result = self.invoke("generic", *args)
            self.assertEqual(result.returncode, 0, result.stderr)
            original = output.read_bytes()
            self.assertEqual(output.stat().st_mode & 0o777, 0o600)
            repeated = self.invoke("generic", *args)
            self.assertNotEqual(repeated.returncode, 0)
            self.assertEqual(output.read_bytes(), original)
            link = Path(directory) / "link.json"
            os.symlink(output, link)
            rejected = self.invoke("generic", *args[:-1], str(link))
            self.assertNotEqual(rejected.returncode, 0)
            self.assertEqual(output.read_bytes(), original)


if __name__ == "__main__":
    unittest.main()
