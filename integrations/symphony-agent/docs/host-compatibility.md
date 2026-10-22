# Host compatibility

This Alpha shares one native bridge and one administration skill across clients. Host adapters only register the process and select its configuration or local socket. They do not translate command meaning, manufacture authority, or create separate domain APIs.

Protocol target: **MCP 2025-11-25**. This is an explicit compatibility baseline, not a claim to implement the current 2026-07-28 protocol. The latter changes request metadata and stateless operation semantics. The bridge must pass its own tests and real client qualification before a host is advertised as tested. [Baseline lifecycle](https://modelcontextprotocol.io/specification/2025-11-25/basic/lifecycle), [current protocol](https://modelcontextprotocol.io/specification/2026-07-28/basic).

## Connection modes

Direct mode starts the installed binary with `--config /absolute/path/to/connection.json`. The runtime also supports `SYMPHONY_MCP_CONFIG` when no explicit configuration argument is supplied. Generate the connection file with the package's `scripts/configure_connection.py`; it binds a selected qxctl executable and working directory and defaults to read-only dispatch.

IPC mode explicitly starts a bridge with `--config FILE --listen SOCKET`, then starts each MCP stdio relay with `--connect-socket SOCKET`. This socket is the plugin's optional local same-user endpoint. It is not a pre-existing general Symphony IPC API. Native owner IPC remains behind qxctl. The intended Alpha hosts are supported POSIX systems; native Windows transport is not supplied by these adapters.

The MCP-facing transport is UTF-8 newline-delimited JSON-RPC. stdout carries only protocol messages; diagnostics use stderr. This lets a C++ process serve the same protocol as a server written in another language. [MCP stdio contract](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports).

The bridge's subprocess interface supplies structured argv and empty stdin. It does not supply stdin control documents. Grammar routing is not complete native input validation, and owner schemas remain authoritative. Connection limits and preview semantics are documented in [Connection profile](connection-profile.md). Socket clients are served serially; this Alpha does not provide concurrent operation cancellation or streamed native output.

## Client packages

| Client | Delivered adapter | Documented format | Qualification |
| --- | --- | --- | --- |
| Codex / compatible OpenAI desktop host | Root `plugin.json`, `mcp.json`, shared `skills/` | Portable Agent Plugins 1.0.0 with stdio | Real host installation pending |
| Codex standalone configuration | `scripts/configure_clients.py --client codex` | `[mcp_servers.symphony]` TOML | Real host connection pending |
| Claude Code | Root `.claude-plugin/plugin.json`, `.mcp.json`, shared `skills/` | Claude plugin and local stdio configuration | Real host connection pending |
| Antigravity | Separate `adapters/antigravity/` manifest and generated MCP JSON | `plugin.json`, `mcp_config.json`, shared `skills/` | Real host connection pending |
| Other MCP hosts | `--client generic` JSON fragment | Local command/args profile, subject to host support | Host-specific qualification required |

### Codex

The portable package discovers `skills/` and `mcp.json` at the root. It launches `./bin/symphony-mcp`, with a connection file at `${PLUGIN_DATA}/connection.json`. An operator must create that file before use; package installation does not silently select an installation. A direct generated adapter can instead select an existing absolute connection path.

Portable `command` accepts a bare executable or `./` plugin-relative path. `${PLUGIN_ROOT}` and `${PLUGIN_DATA}` expansion applies to args, environment values, and cwd, not command. The root manifest uses `extensions.com.openai`; a redundant legacy Codex manifest is intentionally omitted. [OpenAI package guide](https://developers.openai.com/plugins/build/plugins), [portable MCP rules](https://agent-plugins.org/plugin-authors/mcp-servers).

Standalone generated TOML is a fragment to merge into user `~/.codex/config.toml` or a trusted project's `.codex/config.toml`. It sets `default_tools_approval_mode = "writes"` and 70-second host timeouts to leave margin beyond the bridge's maximum 60-second subprocess timeout and cleanup. This is host policy, not native Symphony authority. The generator does not edit an existing host configuration. [Codex MCP](https://learn.chatgpt.com/docs/extend/mcp?surface=cli).

### Claude Code

The root Claude manifest and `.mcp.json` use `${CLAUDE_PLUGIN_ROOT}` for the binary and `${CLAUDE_PLUGIN_DATA}/connection.json` for operator configuration. Those are Claude-specific variables. For a standalone connection, generate JSON with an absolute executable and configuration path, then merge it into the intended `.mcp.json` or register its entry with Claude's MCP manager. A Claude Code adapter does not establish Claude Desktop or cloud compatibility. [Claude MCP](https://code.claude.com/docs/en/mcp), [Claude plugin manifest](https://code.claude.com/docs/en/plugins-reference).

### Antigravity

Antigravity uses `mcpServers` with `command` and `args` for local stdio. Its current documented config paths are `~/.gemini/config/mcp_config.json` and workspace `.agents/mcp_config.json`. Generate the MCP file using the installed binary's absolute path; the adapter deliberately assumes no undocumented plugin-root variable.

Its plugin manifest is separate because its schema differs from Agent Plugins. To assemble an Antigravity bundle, copy `adapters/antigravity/plugin.json.example` as root `plugin.json`, the generated JSON as root `mcp_config.json`, and the shared skill into `skills/`. Select the same installed binary. Do not install the portable OpenAI manifest in its place. [Antigravity MCP](https://antigravity.google/docs/mcp), [Antigravity plugins](https://antigravity.google/docs/plugins).

## Generate a connection fragment

```sh
python3 scripts/configure_clients.py --client codex \
  --binary /absolute/path/to/bin/symphony-mcp \
  --config /absolute/path/to/connection.json

python3 scripts/configure_clients.py --client antigravity \
  --binary /absolute/path/to/bin/symphony-mcp \
  --connect-socket /absolute/path/to/symphony.sock
```

Use `--output FILE` to create a new file exclusively with private permissions. It never overwrites or merges an existing file. Generated JSON has a local schema in `adapters/stdio-config.schema.json`; this describes this generator's narrow output, not every option the hosts support. Arguments remain individual strings; no shell command is generated.

## Qualification before release

For each named host, record the exact host version, platform, binary digest, configuration mode, and actual results for connection/discovery/read, an enabled disposable mutation, native refusal, unknown-outcome recovery, and reconnect. Test from a clean installation. A protocol test harness is useful evidence but is not a substitute for these real host checks.

No remote HTTPS/OAuth MCP endpoint is configured by this package. Existing bounded Symphony remote-owner commands may be discoverable through local qxctl; that is distinct from exposing this MCP bridge remotely. Client adapter support and a provider or broker integration must also be qualified separately.

Official host documentation was inspected on 2026-10-22. Recheck before public submission. The core uses C++; the official MCP SDK inventory currently does not list a C++ SDK, so this implementation must maintain its own conformance evidence. [Official SDK inventory](https://modelcontextprotocol.io/docs/2026-07-28/sdk).
