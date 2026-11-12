# I5 Cloudflare Access user sessions

Status: implemented local owner and fixtures; live Access SSH qualification remains open.
The November 7 Cloudflare/transport implementation contract assigns this optional
capability to SOV. SCV provider administration, SSIAG credentials, protected local
profile administration, transport reachability, and target authority stay distinct.
This contract adds no behavior to SSH v1 and no new remote operation to SSH v2.

## Ownership and entry points

The independently installed C++26 `symphony-cloudflare-access` entry point in
`sov-cloudflare-adapter` owns private session bytes and cloudflared execution.
`--describe` returns its installed nonsecret schema projection. `--session` accepts
only `symphony.sov.transport-session-native-request.v1` from the protected carrier.
The connection arguments are the existing fixed I3 ProxyCommand protocol.
No provider CLI passthrough, shell fragment, raw token, token digest, or token
output is exposed through qxctl. Both provider output streams are discarded during
login; only the SSH byte stream is forwarded during connection.

`qxctl sov transport session schema|template|inspect|plan|authenticate|renew|clear`
uses complete text/JSON projections. The template is synthetic and deliberately
expired. Author an endpoint through the existing `ssh-endpoint --transport-profile`
path; select a fresh deadline no more than five minutes ahead. Controls and replies
are closed, duplicate-free JSON capped at 65,536 bytes. Full endpoint/profile
bindings and protocol shapes are in `cloudflare/access-sessions.json`.

Every request carries `protocol`, UUID-v4 `request_id`, `operation`, `endpoint`,
`intent`, `plan_digest`, and canonical decimal `deadline_unix_ms` (Unix milliseconds).
Inspect uses null intent/plan. Plan carries `{action,identity}`: action is
`authenticate`, `renew`, or `clear`; clear identity is null, other actions require
exact Access audience, issuer and subject. Execute carries the returned plan digest
and null intent. The profile's application is independent of the SSH destination,
SSH user and pinned host key. Audience/issuer/subject are independently selected
Access identity constraints, never inferred from that SSH user.

## Privilege and dispatch

Profile/policy mutation remains the sudo-gated I2 path. qxctl, carrier, native
session owner, cloudflared and SSH run with the caller's ordinary UID/GID. Set-ID
and root execution are refused. Authentication and renewal hold the existing
protected profile lock through native process start and require the current
profile, policy, package digests and caller admission. The native owner independently
checks its kernel parent against the protected carrier digest; connections check
the exact OpenSSH parent. Caller-supplied executable paths are not accepted.

Inspect and clear (including clear planning) may use a retained endpoint snapshot
after disable/removal; protected package verification still applies. Native private
ownership enforces the actual UID. This exception grants no connection admission,
profile write, provider-resource authority, or ability to read another user's cache.
An unavailable/replaced package remains a refusal requiring administrator repair.

## Session state, plans and recovery

The caller selects an existing mode-0700 directory with safe, nonsymlink ancestors.
The owner serializes it with a private nonblocking lock. `state.json` and
`readiness.json` contain nonsecret bindings and expiry; `token` is private mode0600.
No credential bytes or credential digests appear in controls, results or journals.
Plan digests bind the planning request UUID, UID, helper/runtime, application, endpoint/profile, selected
identity, previous nonsecret state and expiry. An attempt records dispatch intent
before login, consumes its exact plan once, and records completion only afterward.
Changed state, expired plans and replay fail closed. Lost mutating replies are
`session.outcome_unknown`: inspect before authoring a fresh plan; never auto-replay.

Authentication and renewal invoke exactly `cloudflared access login --quiet
https://<application>` in a fresh private HOME. This explicit operation can open
the user's browser; qxctl itself has no prompt, reads no login URL, and never obtains
a token. Headless browser completion is not implemented. Renewal retains the same
selected identity. Failed login leaves any previous usable session in place.
Successful replacement invalidates readiness first, so interruption cannot expose
an apparently ready partial replacement. Local inspection checks JWT claim shape,
identity and expiry, not JWT signature or current provider policy. Every reply
therefore retains `provider_authorization=not_proven`.

Clear deletes the owner's token/state/readiness and inactive scratch caches.
Active connections retain their isolated snapshot until exit; clear does not kill
an admitted stream or revoke provider/browser sessions. Disabled profiles block new
connections. Orphaned scratch caches from killed helpers are private and reclaimed
by clear. Active scratch leases prevent cleanup from deleting another running
helper's state. Plan/attempt evidence is retained for local recovery.

## No implicit login on a connection

The selected cloudflared release is exactly 2026.10.7. Current admission is limited
to the C0 macOS-amd64 artifact; other runtime artifacts require separate qualification.
The release has no noninteractive Access-SSH flag. The wrapper copies only the
selected application token into a fresh mode0500 HOME/cache with a mode0400 token;
no organization token or ambient proxy/credential environment is inherited. The
pinned upstream obtains a cache lock before browser login and cannot replace a
rejected token in this snapshot. Failures return to the caller; they do not start
renewal, use another route, or relax host-key checking. Local readiness missing or
expired is I3 `transport_profile.interaction_required`. A later provider rejection
is an unsuccessful/uncertain transport response, not evidence of target execution.
The actual pinned-binary and live Access gates must prove this release-specific
behavior before marking the transport qualified.

Connection lifetime follows the I3 caller deadline/process-group cleanup, not JWT
expiry after a stream has already been admitted. Authentication deadlines bound
child lifetime. Core dumps are disabled at the helper entry point. Private caches
are not a defense against hostile same-UID/root processes, filesystem snapshots or
a compromised approved executable.

Exact implementation references: [token handling](https://github.com/cloudflare/cloudflared/blob/2026.10.7/token/token.go),
[cache paths](https://github.com/cloudflare/cloudflared/blob/2026.10.7/token/path.go),
[Access CLI](https://github.com/cloudflare/cloudflared/blob/2026.10.7/cmd/cloudflared/access/cmd.go).
The nine existing bounded remote operations and target-local SSIAG/STAV/SBV
semantics remain unchanged. Fixture success is not live transport acceptance.
