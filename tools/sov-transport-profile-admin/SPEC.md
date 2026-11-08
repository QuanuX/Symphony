# Protected transport profile administrator 0.1.0-dev

This independent Go owner implements I2 local profile administration. `qxctl sov transport profile {list,inspect,schema,template,plan,apply,status,recover}` presents its closed controls. Direct SSH v1 is unchanged. Endpoint v2, carrier admission, connections, sessions and Cloudflare provider operations belong to later increments and are not implemented here.

`contract/types.go`, `codec.go` and `validate.go` own wire semantics. `scripts/generate_contract.go` projects closed JSON schemas and the independent qxctl reader without an unpublished Go module dependency. `schemas.json` documents shapes; runtime validators additionally enforce exact operation shape, state, generations, digest and authority. Schema/template discovery uses compiled owner data and requires no elevation or host store. The template is a synthetic create-plan request, not an admitted installation.

## Identity and input

Every request is at most 8 MiB, contains all declared members (including explicit nulls), and rejects duplicate keys, aliases, unknown members, malformed Unicode and noncanonical numeric values. Host, TOPS and profile UUIDs are canonical lowercase; request and operation UUIDs use v4 or v7. UID/GID and generation counters use decimal strings. Root and the POSIX ownership sentinel are excluded from permitted caller pairs. No raw command, executable path, SSH option, configuration include, private key or credential is accepted.

Digests remove only their own `digest` member, hash protocol bytes, NUL, then normalized Go JSON with sorted object keys. Arrays retain order, Go escaping remains effective, nulls are explicit. This is not RFC 8785. The literal independent golden vector is checked by producer and consumer. Existing knowledge receipt-v2 intrinsic digests use their existing undomained encoding instead.

Store scope is fixed-root/host UUID/TOPS UUID. Profile names remain reserved after removal; replacement retains the name and UUID. Create expects absence and publishes disabled generation 1. Replace, enable, disable, remove and policy changes bind exact previous profile/policy generations and snapshot digest. Each change increments both counters without uint64 wrap. Removal publishes a terminal tombstone. Retained generations remain identifiable in history. Plans last five minutes and bind the exact protected administrator binary SHA256. Settle pending operations before updating that binary; recovery requires restoring the exact admitted binary if it changed.

## Authority and publication

The only production process arguments are `query`, `administer`, `schema`, `template`, and `--version`. Query runs as a non-root user. Apply/recover require effective UID 0, the exact installed executable path, and verified protected ancestry. qxctl itself refuses root invocation. It invokes only `/usr/bin/sudo -n -- <fixed-helper> administer` with a closed request on stdin, controlled environment and no arbitrary forwarding. `-n` does not prompt; the administrator may first establish sudo authorization using their normal host policy. Successful sudo does not prove a fresh password prompt. Submitted actor data and SUDO_* variables grant no authority.

All store directories, the stable LOCK file and records must be root-owned, nonsymlinked and non-writable by group/others; regular files must have one link. Descriptor-relative no-follow access prevents user path substitution. Queries/plans hold a shared lock; mutations hold a nonblocking exclusive lock. The lock is provisioned once and never replaced. History and current state contain a coherent profile/policy snapshot. Atomic writes use same-directory exclusive staging, file sync, rename and directory sync. Unreferenced staging files are inert; an administrator may inspect/remove them during separate maintenance while the store is quiescent.

Publication order: persist prepared intent; commit authenticated intent audit; retain history; atomically publish current snapshot; commit authenticated outcome audit; retain settled attempt; clear pending intent. Any interruption retains pending evidence. New mutation refuses until explicit recovery binds operation ID, plan digest and the currently observed pending digest. Complete rechecks current state and package trust, then completes or acknowledges the same publication. Abort is allowed only while state is still the prior snapshot. Audit request UUIDs are allocated and persisted before dispatch; explicit recovery reuses them after lost replies. No automatic mutation retry occurs. An archived attempt cannot be overwritten with different content.

The helper has a 30-second process deadline; qxctl bounds its invocation and output. Interrupted output is unknown and requires status inspection. Status/list/inspect expose only safe profile data, exact plan/state references and intent/outcome candidate digests. Internal journals retain complete validated receipts. A missing or malformed required audit never permits publication. I3 must treat any pending journal as an admission block and hold the shared lock through its launch decision; I2 does not implement or claim connection admission.

## SSIAG/STAV relationship

The new `/v1/transport-profile/audit` endpoint accepts only an actual kernel-root Unix peer on a system-mode SSIAG with the exact TOPS. The client uses only the fixed protected system socket and independently verifies its kernel UID 0; it uses no proxy, redirects or external network. This explicit root host-administration relationship does not make SSIAG local-v1 user decisions transferable and does not grant target operation authority.

SSIAG submits these declared STAV producer pairs through its existing authenticated producer channel:

- `symphony.sov.transport.profile.intent` / `symphony.sov.transport.profile.prepare`, allowed.
- `symphony.sov.transport.profile.outcome` / `symphony.sov.transport.profile.observe`, succeeded with published or aborted reason.

The exact producer grant must admit those pairs. Profile administration neither installs nor broadens that grant. Request and candidate bind host/TOPS/profile, operation/plan, prior and desired state, phase, resolution and observed state. SSIAG and the helper independently require the exact committed STAV receipt, request UUID, TOPS and candidate digest. Root authority is the host boundary; this mechanism does not constrain a malicious root administrator.

## Package prerequisite

Enable and enabled replacement/policy mutation require the exact protected package trees for carrier, SSH and proxy. Disabled draft creation permits unresolved package references, but does not admit execution. Paths derive only from fixed package root, package UUID and wrapper version. The manifest pins role/platform, executable, complete file sizes/digests/modes, receipt file bytes, source receipt component/version/entry point, and qualification evidence path/digest. Its digest is domain-separated. Wrapper version and selected upstream receipt version remain distinct.

Every runtime/dependency file must appear in the existing `symphony.knowledge.install-receipt.v2` and the transport manifest; the receipt itself and separately reviewed qualification evidence are manifest-owned. The receipt self-digest, exact identity, platform and selected executable entry point are checked. The carrier receipt must declare all three selected v2 protocols; an existing v1-only carrier cannot qualify. Extra files, links, writable ancestors, drift and missing evidence refuse enable/recovery. Qualification evidence is administrator-reviewed, byte-pinned content; possession of a file or hash does not prove a live provider test. System libraries outside the submitted tree must be declared by the selected platform qualification; undeclared runtime dependencies are not qualified.

Installation is a separate administrator operation described in INSTALL.md. The helper does not install, execute or probe any selected package and has no subprocess imports. The first profile's future execution contract is Cloudflare Access user-session authentication as an explicitly permitted non-root UID/GID. I3/I5 must independently reverify the full package tree, identity and generations immediately before launch. No supplied I2 manifest establishes an I3-compatible carrier or a qualified Cloudflare session.

## Evidence and limits

Focused tests cover strict controls, independent digests, generation/CAS/ABA/tombstone behavior, protected paths and locks, forged/missing audit, interrupted publication, explicit recovery, package closure and drift, kernel-peer IPC, qxctl sudo argv and complete discovery. The optional installed-process fixture runs only on an explicitly selected disposable root host; it uses a simulated append sink and makes no deployed SSIAG/STAV or live Cloudflare claim. SSH v1 regression tests remain required. Full SCV/SHV and integrated/live I6 gates remain deferred.
