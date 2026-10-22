# Connection and discovery

## Connection choices

The host starts `symphony-mcp` as an MCP stdio process. A connection uses either an operator-selected configuration file (`--config` or the runtime's `SYMPHONY_MCP_CONFIG` fallback) or `--connect-socket` to an already running local bridge. The optional daemon is started explicitly with `--config FILE --listen SOCKET`.

The socket belongs to this bridge. It is not a generic pre-existing Symphony administration endpoint, a replacement for native SSIAG/STAV IPC, or a new authority service. Its intended boundary is local, same-user IPC on supported POSIX hosts. Do not expose the socket over a network or substitute a remote endpoint without a separately supported transport and authorization design.

Client registration examples are provided by the package's adapter generator. A missing binary, configuration, socket, or qxctl installation is setup work. Preserve an existing configuration and report the exact missing input; do not invent an installation from repository contents. Never collect credentials in chat.

## Discovery sequence

1. Call `symphony_connection`. Record the selected configuration's installation identity and available connection details without exposing private paths unnecessarily.
2. Call `symphony_discover` with a focused query such as `engines`, `ssiag`, `stav`, `scv`, `shv`, `snv`, `sqv`, `sbv`, or `sov`. Queries are search terms, not executable commands. Use `offset` and `limit` to page results when needed.
3. Call `symphony_describe` with a returned `command_id`. Inspect the accepted flags and positionals, mutability, owner, and schema references. Registry grammar is not a complete requiredness/default schema: resolve required native inputs from the owner's schemas, templates, and current evidence before constructing a call.
4. Inspect exact installation/binding status when the operation depends on an installed engine. Discovery alone does not establish runtime readiness.

The optional `mutability` discovery filter selects the registry's actual classification. Do not infer effects from a friendly command name. In particular, a command called `plan`, `verify`, or `recover` is not necessarily read-only.

Hosts that expose MCP resources can also read `symphony://connection` and `symphony://usage`. The `symphony-admin` MCP prompt accepts a `goal` string and supplies the same discovery-first workflow. These optional host conveniences do not change access or operation semantics; the tools remain the direct interface.

The source baseline distinguishes `read_only`, `permission_backed_mutation`, `proposal_only`, `evidence_only`, and `prohibited`. The bridge runs only the operations admitted by its configuration and runtime. `symphony_read` is for `read_only` commands. Other permitted classifications use preparation/application even when their owner-specific effects are limited to local artifacts. A prohibited command must remain unavailable.

## Arguments and result interpretation

Supply structured `flags` and `positionals` using the tool schema and command description. Flag keys are bare long names without `--`. Value-taking flags accept strings or arrays of strings for repetition; represent numbers as strings to preserve precision. Valueless boolean flags accept booleans. Positionals are strings. Omit `json`: the bridge requests JSON itself when supported. Do not construct shell syntax, pipelines, redirects, scripts, or a `qxctl ...` command string in an argument. The bridge executes argv directly; shell-like characters inside a value are data and do not establish a supported argument value.

The connection pins the qxctl executable and working directory. It does not implicitly bind every command's TOPS identity, scope, installation prefix, or version. Resolve those exact targets through the user's request and the owner's schemas and evidence, then pass the command's supported arguments. A different command-selected target does not by itself require a new profile. Selecting a different executable or working directory uses the documented connection setup workflow. Do not invent missing target choices or native defaults.

The bridge supplies no subprocess stdin payload; qxctl receives an empty stdin stream. An owner protocol requiring a stdin document needs a supported input-file route or an explicit bridge extension. Do not claim that a registry entry or accepted argument grammar guarantees every invocation form is available.

Distinguish these observations:

| Observation | What it establishes |
| --- | --- |
| MCP connection succeeds | The client can exchange protocol messages with this bridge. |
| Command is discovered | The selected qxctl registry advertises that command. |
| Repository status/inventory/doctor succeeds | The reported repository or structural checks succeeded within their documented scope. |
| Exact installation/binding verifies | The reported files, version, receipt, or binding passed the owner's checks. |
| Native service status succeeds | That particular service returned the reported observation at that time. |
| Operation receipt/status confirms completion | The exact recorded attempt reached the owner's reported state. |

## When no operation matches

Try the owner's actual terminology and inspect its registered capabilities. Do not claim an absent operation is available because a white paper names it. If the current bridge cannot expose a needed command, report the concrete gap. User-authored modules and alternative adapters are valid extensions; this Alpha's current registry support is not a universal limit on Symphony.
