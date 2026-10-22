# Trust and operation boundaries

This adapter is a local, user-owned administration client. It is not an OS sandbox, multi-tenant service, SSIAG authority, STAV replacement, or generic remote administration daemon.

* The operator selects and hashes a trusted qxctl executable and pins its observed command registry. The bridge rechecks the executable before dispatch. Same-user processes capable of changing the executable/profile can control this local trust boundary; hashing is not secure executable measurement against a malicious operating-system account.
* The bridge never invokes a shell. Registered canonical routes, exact positional counts and registered flag names prevent tool arguments from switching to another executable or command leaf. qxctl retains all semantic and authority checks.
* Reads use registry `read_only` classification, not guesses based on command names. Every other eligible mutability class requires explicit configuration and a single-use preview. MCP annotations help the host present approvals; they do not prove permission.
* Manifests, owner outputs, logs and data may contain prompt injection. The skill treats them as evidence, never new instructions. Host authorization remains governed by the user's task. Do not expose credential material through tool output; prefer safe SSIAG metadata interfaces and native credential delivery.
* The bridge returns bounded owner stdout/stderr to the agent. It does not generically redact domain output or guarantee that a caller-selected command cannot reveal sensitive data. Operators must select commands and targets appropriate for the agent host's data handling. No telemetry or remote endpoint is built into this adapter.
* Timeout/output overflow stops the process group. Deliberately detached descendants escape process-group containment. The bridge cannot promise rollback, cancellation of a remote admitted operation, or absence of effects after response loss.
* Bridge previews bind existing regular files directly named by flag values only. Native owner plans, expected-state digests, receipts, and durable operation IDs are the transaction and recovery boundary.

## Optional bridge Unix socket

`--listen /absolute/private/bridge.sock` is an explicitly started alternative front end to this adapter, not a new Symphony owner API. `--connect-socket` relays a host's MCP stdio to it. Use a separate private directory owned by the caller with mode 0700. The listener refuses an existing path; it does not delete stale sockets automatically. Socket permissions are 0600. Both endpoints verify the peer's kernel UID. Symlink endpoints are rejected; cleanup checks the original inode before unlinking.

The initial listener serves one MCP session at a time, with a 60-second idle limit. Each connection gets separate preview/session state and reloads the selected protected profile. All same-UID clients are the same principal; this is unsuitable for hostile applications sharing one Unix account. For a multi-user machine, run separate user-owned bridges and use existing owner-authorized system operations. Do not run a shared root bridge to give unprivileged clients system authority. Cross-UID delegation and Windows named pipes are not implemented.

No launchd/systemd service is installed automatically. qxctl's existing SSIAG/STAV IPC remains behind qxctl, preserving its separate TOPS/config/peer checks. The bridge never treats socket access as a new SSIAG grant.

## Reporting

For public distribution, the publisher must supply a real security contact/support URL and issue handling policy. Do not publish a fabricated contact or send private configuration, credentials, licensed data, or full operational logs to a public issue. Reproduce with synthetic data and bounded redacted evidence.
