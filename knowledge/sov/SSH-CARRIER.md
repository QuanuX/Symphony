# Optional scoped SSH carrier v1

This is the first optional network carrier under WP16, extending the unchanged
local handoff and native job contracts. The caller chooses this carrier; SSH is
not a platform requirement. Its selected arrangement is a controller and an
already installed target. It provisions neither a host nor an SSH service.

## Complete qxctl control

`qxctl sov remote template|schema --operation ssh_endpoint|ssh_invoke --json`
exposes every field. `qxctl sov remote ssh-endpoint --input FILE --json` validates
and returns the complete endpoint, public-key hashes, endpoint digest and reference
without connecting. Put that reference in the existing selection's
`transport.endpoint_ref`, select `scoped_cli`, adapter ID
`symphony.sov.ssh-adapter`, version `0.1.0-dev`, and its exact executable digest.
Use existing `remote plan` to compute the complete selection digest.
`qxctl sov remote ssh-invoke --input FILE --json` performs one explicit invocation.
Human text and JSON expose the same values, including all returned native fields.

The input envelope contains `protocol: symphony.sov.ssh-invoke-input.v1`, the
endpoint, and the complete unchanged `remote-invoke-input.v1` invocation. Unknown,
duplicate, missing and case-aliased members are refused, including omitted nulls.
The endpoint contains exact local SSH executable path/digest and execution mode, host, decimal port,
target login user, one raw host public key, one raw client public key, absolute
SSH-agent socket, optional connection timeout, and target launcher path/digest.
Private keys/passwords are not control fields. Supported raw public-key forms are
Ed25519, RSA and ECDSA P-256/P-384/P-521, subject to the exact selected OpenSSH
client's support. Certificates, security-key providers, proxies and jump chains
are outside this initial profile; another compatible carrier can add them.

All fields are caller authored and visible. Changing any endpoint field changes
its digest and requires an explicit new plan. The endpoint digest is SHA-256 over
`symphony.sov.ssh-endpoint.v1`, NUL, and sorted-key Go JSON of the typed endpoint;
the reference is `sov:ssh:sha256:<digest>`. This is versioned encoding, not a claim
of RFC 8785. The invocation digest retains its existing v1 definition.

## Authentication and execution

The independently installed `sov-ssh-adapter` accepts one envelope on stdin. It
uses the caller-selected `ssh_execution` mode. `stage_copy` stages and hashes the
client before execution. `protected_system_path` hashes the original client and
requires its entire absolute path to be free of symlinks, owned by root and not
group/other writable. This explicit host-administrator trust profile supports
system-signed clients that refuse relocation, such as macOS OpenSSH. It detects
drift but does not claim isolation from a root administrator replacing a file.
There is no automatic fallback between execution modes.
Public identity and known-host files are private, per-invocation temporary files.
No ambient user/system SSH configuration, connection multiplexing, default
identities, password prompting, agent forwarding, port forwarding, proxy command,
local command or interactive terminal is enabled. Only the selected agent-held
client key is offered. Host authentication uses the supplied raw host key, strict
checking, an isolated known-host file and no DNS or additional host-key authority.
Public-key hashes describe the SSH wire key blob, not its text/comment spelling.

The fixed remote command invokes the selected preinstalled `sov-stdio-adapter`
with `--self-sha256` and the existing exact target qxctl/configuration arguments.
Every argument is independently POSIX-shell quoted; user input cannot supply a
general remote command. The target requires a compatible POSIX command shell.
The launcher checks its own digest to detect installation drift, then stages and
hashes target qxctl before execution. qxctl verifies its runtime, native receipt,
target configuration, job/artifact bindings and selected authority before native
dispatch. The authenticated host and installed launcher are trusted execution
boundaries. A self-check is not hardware attestation or proof against a malicious
host administrator; an endpoint digest is not network-origin proof on its own.

`origin_verification: selected_adapter_authenticated_ssh_host` is emitted only
after the selected carrier reports successful SSH completion and the controller
validates endpoint/key/invocation and full native target-result bindings. It is
the exact selected adapter's authenticated-channel evidence, not independent
attestation by qxctl. Missing or invalid replies keep `not_verified`.
Connecting successfully does not identify a human, authorize arbitrary SBV work,
or create a transferable SSIAG grant. The server maps the authenticated key to
the selected target OS account; target-local SSIAG, when selected, still derives
the subject from the local kernel peer. The existing target authorizer uses user
scope. System scope and additional remote-principal delegation are not added here.

## Operation, lifecycle and recovery scope

The nine existing job/result/bundle operations are carried unchanged. Job creation,
source preparation, target configuration and native resource/resident controls
remain explicit target-side qxctl operations. This increment adds no generic
remote shell, remote configuration mutation, data upload, provisioning or resource
broker. Target configure/inspect/remove and expected-digest replacement remain
available. Removal rejects future admissions; running work uses native job_cancel.
Endpoint selections are stateless input documents, so replacing a document changes
the next explicit invocation; there is no hidden carrier configuration store.

Before adapter start, an execution failure is `not_dispatched`. Once started, any
failed SSH authentication, connection loss, nonzero exit, oversized/malformed or
mismatched response is conservatively `response_lost`. The controller cannot
prove how far a selected external adapter progressed from its exit code alone.
A complete valid response is `response_received`, including native error results.
The CLI preserves its complete result with exit 5 for uncertain or native-error
outcomes. No automatic retry occurs. Inspect and reconcile the same native job;
do not re-create it to hide an ambiguous admitted execution.

There is no default calculation or connection timeout. A caller may select the
connection/handshake timeout and independently the existing calculation deadline,
or neither. OpenSSH represents positive connection seconds through a signed
31-bit value; unrepresentable values are refused rather than capped. SIGINT/SIGTERM
cancels the local process group, with a 250 ms pipe/process cleanup bound after
cancellation or exit. Disconnection does not establish remote cancellation or
termination. Native observation resolves what actually happened.

The shared 8 MiB bound applies to a whole control frame, including its envelope,
not total dataset, replay, result or RAM size. Query pages must fit the complete
frame; all artifacts remain available through caller-selected pages. Agent socket
paths are canonical absolute paths; this profile refuses newline, percent and
dollar expansion syntax instead of interpreting ambient substitutions.

## Qualification and remaining work

Focused tests cover exact binding, hostile command-path quoting, explicit SSH
options, parser boundaries, lost/mismatched responses, cancellation and complete
qxctl presentation. Real OpenSSH loopback tests, when recorded, establish only
that host/client/runtime profile. Two-host connectivity, deployed SSIAG/STAV
allow/expiry/revocation, remote access/hydration and network fault qualification
require separate positive gates. Cluster/NUMA/terabyte, shared-storage/power-loss,
checkpoint resume and peer-performance claims remain open.

The 13 October 2026 selected gate now supplies those positive carrier and authority
checks for macOS to an Ubuntu Linux guest with a distinct LinuxKit kernel on the
same physical Mac. Real same-UID user-scope SSIAG/STAV, all nine operations, complete
typed account/Databento replay access, cooperative cancellation, killed worker,
controller/SSH loss and response loss after completion pass without duplicate
execution. Real expiry/policy/audit checks use the macOS service profile; Linux
verifies fresh grants, denial and revocation. See the exact scope, retained failed
fixtures and exclusions in [M3 qualification](../../modules/sbv-engine/M3-QUALIFICATION.md).
The temporary SSH port was bound only to loopback. This does not qualify WAN,
separate physical hosts or distinct-principal system-scope authority.

OpenSSH option semantics were checked against its primary
[ssh manual](https://man.openbsd.org/ssh) and
[ssh_config manual](https://man.openbsd.org/ssh_config), and the actual selected
client's `-G` interpretation. No newest-version substitution is implied.
