# Connection profiles

Create one profile per explicitly selected qxctl installation and working directory; use a repository context for repository-inspection commands. The bridge does not search PATH or select the newest installed version. `scripts/configure_connection.py` reads the selected executable's live manifest, hashes the executable, and writes a new private file. It does not install software or alter Symphony state.

```sh
python3 scripts/configure_connection.py \
  --qxctl /absolute/installation/bin/qxctl \
  --working-directory /absolute/Symphony \
  --output /absolute/private/connection.json
```

The default profile exposes reads. To enable a non-read operation, add `--enable EXACT_COMMAND_ID` for each selected command. Obtain IDs from the registry. `proposal_only` and `evidence_only` are non-read operations because they may persist artifacts. An enabled command is transport permission, not an SSIAG grant, acceptance of a native plan, or user authorization for every invocation. Configure host approval policy to match the operator's intended delegation. Existing authorization can suffice; the skill does not impose a new human-only ritual.

The script refuses to overwrite an existing profile. Create a replacement at a new path, review it, and explicitly select it. The executable and registry are pinned; upgrading requires selecting the replacement. `--inherit ENV_NAME` explicitly forwards additional required environment names. Avoid broad ambient credential forwarding. The bridge normally forwards HOME/TMPDIR and sets a fixed system PATH and C.UTF-8 locale. Credentials remain with native owner mechanisms. This profile is not a secret vault.

## Format

All paths are absolute. The profile must be an owned regular file with no group/other permissions; symlink profiles are rejected. Unknown fields are rejected.

| Field | Meaning |
|---|---|
| `schema` | `symphony.agent.connection.v1` |
| `qxctl` | Exact executable, canonicalized at startup |
| `qxctl_sha256` | `sha256:` plus 64 lowercase hex digits |
| `registry_digest` | Exact observed registry digest from that executable |
| `working_directory` | Repository context for commands that inspect source; not an installation-health assertion |
| `enabled_nonread_commands` | Exact permitted non-read command IDs; defaults to empty |
| `inherit_environment` | Explicit names forwarded from launcher; defaults HOME/TMPDIR |
| `environment` | Explicit string values; never supplied through an MCP call |
| `timeout_ms` | 100–60,000, default 30,000 |
| `max_output_bytes` | Combined stdout/stderr bound, 1,024–8,388,608, default 524,288 |

The manifest probe has an independent 8 MiB limit and at least a five-second startup budget. Process termination allows up to 500 ms for qxctl to perform signal cleanup before force-killing its process group. The stdin-only `sov.target.invoke` command, unsupported future grammar, and hidden/prohibited/retired/interactive commands are excluded and reported by `symphony_connection`. The first Alpha understands literal canonical routes, fixed `<positional>` arguments, boolean flags, and VALUE, VALUE..., UINT64 and DURATION flags. The registry is not a complete input schema: qxctl and native owners validate required flags, exact numeric formats, versions, grants and state.

MCP flag values use strings (including integer quantities), string arrays for repetition, or booleans for boolean flags. The bridge constructs `--flag=value`, forces JSON where the registry advertises it, and puts fixed-count positional arguments after `--`. It never accepts an executable, shell string, working directory, or environment through a tool call.

## Preview and uncertainty

`symphony_prepare` retains exact argv and SHA-256 hashes of existing regular files directly named by flag values up to 64 MiB each. It does **not** invoke a native plan, reserve resources, snapshot nested dependencies, freeze directories, or grant authority. Previews expire after five minutes, are limited to 128 per session, and are lost when the session ends. Input hashes are a drift check, not an atomic transaction; native owners must validate their own referenced state and expected digests.

`symphony_apply` consumes the preview before dispatch and checks direct file hashes and the executable pin again. A failed, timed-out or output-limited non-read result conservatively reports unknown outcome and the registry's recovery command. A successful process exit is recorded without fabricating semantic success. Inspect owner results/receipts and reconcile the original operation ID. Do not create a new operation or replay a mutation merely because a response was lost. A bridge restart does not deduplicate owner operations: native durable operation IDs remain authoritative.

Long-running work should use an owner's submit/status/reconcile operations where available. The bridge is synchronous, bounded to 60 seconds per subprocess, and does not implement concurrent cancellation or large-data streaming. Read result pages using owner query/selection operations; an output-limited response is incomplete and must not be treated as complete evidence.
