# Protected SSH profile carrier v2

I3 adds an explicit profile-aware controller transport. The original endpoint,
invocation and reply v1 contract, defaults and proxy prohibition remain exact.
`0.1.0-dev` is still the development binary identity; v2 eligibility requires the
exact installed bytes and a receipt declaring all three v2 protocols. Version
text alone does not admit an old binary. Public Go v0.1.0-dev dependencies remain
unchanged; generated independent consumers carry new contracts.

## Planning and discovery

`qxctl sov remote template|schema --operation ssh_profile_selection` describes
planning input: the original direct SSH target/key/agent fields, local host/TOPS
scope, application hostname, private session and dispatch directories. Invoke
`qxctl sov remote ssh-endpoint --transport-profile NAME --input selection.json`.
Name resolution is scoped, protected and unprivileged. The result seals the
complete coherent profile/policy snapshot. `ssh_endpoint_v2` and `ssh_invoke_v2`
expose full endpoint and invocation shapes. Omission of the flag preserves v1.
The flag is not accepted on invocation. No planning call connects or elevates.

The v2 endpoint reference hashes the canonical complete endpoint with domain
`symphony.sov.ssh-endpoint.v2` followed by NUL, and is represented as
`sov:ssh:v2:sha256:<hex>`. Existing remote-plan/invocation v1 binds that opaque
reference. Target launcher, host key, public client identity, agent and exact
remote runtime bindings remain effective. qxctl validates the complete native
result independently after the carrier reports authenticated SSH transport.

## Protected admission and execution

Both qxctl and the actual carrier read the fixed I2 store; bypassing qxctl does
not bypass admission. The carrier rejects root, resolves the protected package
closure, requires its own installed pathname and digest, and matches the sealed
snapshot to enabled current state and the exact kernel UID/GID. Supplementary
groups and SUDO_* grant no profile authority. Any pending journal, corruption,
package drift or generation change refuses connection. The stable shared lock
covers fresh trust admission, session-readiness check, audit, durable private
dispatch intent, child Start and launch observation. It is released before Wait;
disable prevents later admissions without claiming to revoke established work.

The generated OpenSSH ProxyCommand is one fixed Symphony proxy-wrapper contract:
`<protected-wrapper> --application <host> --session-directory <private-dir>
--session-ref <reference>`. OpenSSH adds its own `exec` prefix. Arguments are individually POSIX quoted; OpenSSH `%`
expansions and control characters are refused. No caller command, shell fragment,
config Include, ProxyJump, alternate host key or ambient SSH config is admitted.
The wrapper runs as the invoking user in the same process group, with controlled
PATH/locale and the selected session directory as HOME. Output and cancellation
are bounded. qxctl allows the carrier up to the declared five-second cleanup
bound to stop its own SSH/proxy group before forcing carrier termination. Direct carrier calls also honor the selected invocation deadline.
The supplied profile store is never written by connections.

## Audit, private journal and session boundary

The new system SSIAG endpoint `/v1/transport-connection/audit` accepts a bounded,
closed request from an authenticated non-root kernel peer in the exact TOPS.
Its producer grant must admit event `symphony.sov.transport.connection.intent`
and operation `symphony.sov.transport.connection.prepare`. The receipt binds
caller identity, endpoint and invocation digests; it records intent, never grants
profile or target authority. The carrier authenticates a root Unix server at the
fixed system socket and independently validates the committed STAV receipt.
There is no proxy, redirect or automatic POST retry. Installation must grant the
permitted user socket access without making socket ancestry writable.

The existing private journal directory must be owned by the caller and mode 0700.
A fresh request UUID creates one exclusive mode-0600 JSONL record, fsynced with its
directory before audit/dispatch. Stages distinguish prepared, committed audit,
dispatch intent, observed start and received/lost response. Results store digests,
not target response bytes. Existing request IDs refuse automatic replay; after an
uncertain outcome inspect the retained record and reconcile the original remote
job before explicitly creating a new request. A crash after dispatch intent is
uncertain. Loss or local cancellation does not prove remote cancellation.

`readiness.json` in the private session directory is a closed nonsecret
`symphony.sov.proxy-readiness.v1` record with application, session_ref,
helper_digest and RFC3339 expires_at. It is only local readiness metadata, not
proof of provider authentication. A qualified wrapper must independently verify
actual session state and refuse expiry/denial without starting a browser/login.
Missing or expired readiness returns `transport_profile.interaction_required`; explicitly set up or renew the selected session before a new invocation. Other admission failures return a safe fixed reason, never vendor error text.
I5 owns the exact Cloudflare wrapper, session setup/renewal/removal and live
qualification. I3 does not ship a qualified Cloudflare session implementation.

## Validation boundary

Focused tests cover strict parsing, v1 separation, independently consumed
bindings, root/principal refusal, pending/disabled/stale state, package drift,
lock exclusion, private readiness, durable replay refusal, generated proxy
quoting and authenticated SSIAG intent. The optional disposable Linux installed
fixture exercises an actual selected OpenSSH client/server and controlled proxy
under UID/GID 1001 with synthetic keys and a simulated committed audit sink.
This is not deployed SSIAG/STAV, Cloudflare, WAN or multi-physical-host evidence.

Local evidence, November 8, 2026: carrier and SSIAG boundary race tests, protected
store lock/recovery tests, qxctl v1/v2 bindings and full refusal presentation,
shared independently computed endpoint digest, generated-consumer checks and
repository/dependency/semantic validators passed. The actual Linux amd64
OpenSSH fixture also covers session paths with spaces and apostrophes, hostile
ambient SSH configuration, wrong host/client keys, pending and re-enabled
generations, unavailable audit/readiness, replay and cancellation cleanup.
macOS received local Go/quoting checks; no installed macOS proxy or provider
qualification is claimed. CI retains the same bounded Linux fixture. Broad
SCV/SHV and live-provider campaigns remain deferred to their milestone gates.

The optional I5 native Access helper and explicit session controls are assigned by
`knowledge/sov/CLOUDFLARE-ACCESS-SESSIONS.md`. They preserve this carrier contract
and its target authority; local implementation is not live Access qualification.
