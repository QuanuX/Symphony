# Symphony agent plugin — 0.1.0-alpha.1

A C++26 MCP bridge, Codex skill/plugin, and reusable client adapters for direct administration of a selected Symphony installation. Publisher: **QuanuX**, [quanux.org](https://quanux.org).

The bridge discovers the selected qxctl's actual command registry, exposes exact command descriptions, runs reads, and previews/applies configured administration operations. It preserves qxctl as Symphony's operating interface and the native owners' identity, authority, receipts, versions and recovery rules. It adds no new vector or mandatory workload dependency. User modules and compositions remain open-ended.

This is a local Alpha candidate, not an OpenAI-approved directory release. Current compatibility is the explicitly pinned MCP **2025-11-25** protocol. See [verification](docs/verification.md), [host compatibility](docs/host-compatibility.md) and [public submission requirements](docs/submission.md).

## Build and connect

Requirements: macOS or Linux; a compiler accepting C++26 draft mode (`-std=c++2c`), CMake 3.30+, and Python 3.11+ for configuration/tests. Linux also requires OpenSSL 3 development files. nlohmann/json 3.12.0 is included with its license. No build-time dependency download occurs.

```sh
cmake -S . -B build -DCMAKE_BUILD_TYPE=Release
cmake --build build -j 4
ctest --test-dir build --output-on-failure
cmake --install build --prefix "$PWD"
```

Select a registry-capable qxctl installation and an existing working directory explicitly (use a Symphony repository for repository-inspection commands):

```sh
python3 scripts/configure_connection.py \
  --qxctl /absolute/installation/bin/qxctl \
  --working-directory /absolute/Symphony \
  --output /absolute/private/connection.json

python3 scripts/configure_clients.py --client codex \
  --binary /absolute/plugin/bin/symphony-mcp \
  --config /absolute/private/connection.json
```

The second command prints a client configuration fragment; review and add it to the intended host configuration. Use `--client claude`, `antigravity`, or `generic` for other MCP clients. These commands neither install into the host nor overwrite existing configuration. Full plugin installations instead read their connection profile from the host's documented plugin-data directory; see [host configuration](docs/host-compatibility.md).

Profiles default to read operations. Add `--enable EXACT_COMMAND_ID` when creating a profile to expose the specific non-read operations delegated to the agent. [Connection profiles](docs/connection-profile.md) describe pinning, environment forwarding, limits and replacement. Native authority still applies. An existing older qxctl without `commands manifest --json` is explicitly rejected; the adapter never upgrades it automatically.

## Agent interface

| Tool | Purpose |
|---|---|
| `symphony_connection` | Executable/registry binding, bridge limits and enabled capabilities; no service-health inference |
| `symphony_discover` | Paginated search through the actual installed registry |
| `symphony_describe` | Exact command grammar, flags, mutability, authority, owner protocols and recovery |
| `symphony_read` | Direct execution of a registered read-only operation |
| `symphony_prepare` | Five-minute, session-bound, single-use argument preview for an enabled non-read operation |
| `symphony_apply` | Execute that preview once, retaining native authority and conservative uncertainty |

The package also exposes connection/usage resources and the `symphony-admin` prompt. The [skill](skills/symphony-admin/SKILL.md) guides discovery, administration, identity/audit, knowledge, provider/hardware/node workflows, quantitative research and operation recovery through their actual owners. Availability depends on the selected installation; a command's presence is not proof of a deployed engine, credential, grant, provider connection or service health.

A transport preview is not a native owner plan. Native expected-state checks and durable operation IDs remain essential for their applicable operations. Output is bounded and preserves exit code/stdout/stderr; it never equates process exit 0 with semantic success. A timeout, lost response or output limit during mutation requires owner observation/reconciliation before another action.

## Optional local IPC

The ordinary MCP stdio transport is already local IPC. An explicitly started same-user Unix-socket front end is also supplied:

```sh
mkdir -m 700 /absolute/private/symphony-ipc
bin/symphony-mcp --config /absolute/private/connection.json \
  --listen /absolute/private/symphony-ipc/bridge.sock
```

In another process, launch `bin/symphony-mcp --connect-socket /absolute/private/symphony-ipc/bridge.sock` as the host's stdio server, or generate that adapter with `configure_clients.py --connect-socket ...`.

Both peers verify the kernel UID; the socket is 0600 in an owned 0700 directory. The bridge does not install a service automatically. It serves one local session at a time, with a 60-second idle limit. This is the bridge's transport, separate from qxctl's existing SSIAG/STAV transports. Windows named pipes, cross-user delegation, and a hosted HTTPS MCP endpoint are future adapters. See [security](docs/security.md) and [data handling](docs/privacy.md).

## Packaging and release

```sh
python3 scripts/package_release.py --binary build/symphony-mcp \
  --output /absolute/output/symphony-alpha-platform.zip
```

The packager includes source, the selected binary, skill, adapters and licenses; it excludes local profiles, build caches and private evidence. It emits an archive digest and never uploads. Build distinct qualified platform artifacts. A source-only archive omits `--binary` and requires a build before installation.

OpenAI's public local-MCP path currently requires partner allowlisting. Publisher support/privacy/terms URLs, branding, reviewer demonstration, real host qualification and release acceptance remain external gates. The configured C++ runtime and adapters can be tested locally before those gates close. Neither this package nor a successful test guarantees a public launch date.

Current target work and limits are described in [platform qualification](docs/platforms.md). The original verification report remains a dated baseline.
