# I6 integrated qualification — November 12, 2026

**Gate open.** This increment runs the integrated local checks and adds a bounded,
repeatable evidence report. It does not declare I5 live acceptance or M1 complete.
The original November 7 implementation/acceptance contract remains the target.
I7 project deployment and CF-11 are outside this first milestone.

The source starts from I5 `08a8f16d` and I4 `e37e6b64`. Integration commit
`73829980` resolves only the verification-document conflict, retaining both
records; runtime code is identical to I5's already repaired head. Both PRs remain
unmerged. CodeQL success is source-security evidence, not live acceptance.

`tests/cloudflare-release/support-matrix.json` binds the current owner and all 23 resource profiles,
exact selected vendor versions, the sole admitted vendor runtime platform and
all seven mandatory requirements. Linux/macOS source builds do not admit a new
vendor artifact. No mandatory action is removed from the milestone.

## Corrected rotation finding

Earlier verification records described rotation as absent from cf 1.0.0-beta.28.
That statement was too broad: the literal command `tunnels token rotate` is absent,
but the retained C0 schema for **`tunnels edit`** exposes PATCH and `tunnel-secret`.
Cloudflare documents rotation through the same PATCH and a fresh tunnel secret:
[Remote tunnel permissions](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/configure-tunnels/remote-tunnel-permissions/).
The documentation also distinguishes rejecting new connections with the old token
from terminating already connected replicas. No CLI upgrade or alternate API
backend is needed merely to identify the existing primitive.

The native adapter currently admits only `name` for `tunnels.edit` and treats its
output as a closed nonsecret projection. Rotation therefore **remains unsupported
by Symphony**, pending a dedicated private mutation-body/result/recovery path.
Do not add `tunnel_secret` to ordinary qxctl JSON, plan previews, arguments or
journals. The implementation must bind the reviewed tunnel/credential generation,
generate or admit the new secret only within native credential authority, deliver
it privately to the exact CLI child, import returned token bytes through SSIAG,
and prove the postcondition without replaying a PATCH after a lost response.
Test secret sentinels, preflight denial, exactly one mutation attempt, import loss,
old/new connector admission and continued existing connections separately.

This is a documented implementation gap, not an upstream impossibility or a
scope waiver. The historical C0 captures and earlier evidence remain unchanged.

## Local gate and remaining execution

See `tests/cloudflare-release/README.md` and its manifest for exact commands.
The reducer reports missing tests, skips, failures and artifact drift. It never
promotes fixture results into live acceptance or grants release authority.
Source integration uses the repository's existing module workspace; standalone
SDK/module qualification remains a separate release boundary.

The host's selected candidate is macOS amd64. The full native administration and
Access suites, native credential provider, installed receipt/schema/refusal,
profile/carrier/SSIAG/Go consumers, generated contracts, pinned CLI offline
probes, and affected projections are recorded individually in the verification
record. The disposable-root profile fixture, Linux installed OpenSSH fixture and
installed SBV target authority must not be counted as passes when unavailable.

## Live preflight and exact next actions

Read-only Cloudflare inventory on November 12 confirmed the existing `quanux.dev`
zone is active. Exact-name queries found no `symphony-i5` tunnel, no Access app
for `symphony-i5.quanux.dev` and no DNS record at that hostname. No resources or
cost-bearing services were created. Local `sudo -n -l` still requires administrator
authentication. Connected Cloudflare account access is not an SSIAG credential
provisioned to the native adapter and cannot substitute for native qualification.

1. Complete the private native rotation operation described above; retain the
   selected cf/Node/cloudflared versions and requalify only affected boundaries.
2. Provision exact protected carrier/helper/runtime packages and the profile
   administrator on a selected disposable host. Provision real controller and
   target SSIAG/STAV permission and audit services. Keep profile publication sudo
   gated and all provider, session, connector and SSH execution non-root.
3. Create only the isolated selected Tunnel/Access/DNS composition through the
   admitted qxctl/native path. Record resource IDs and before/after observations in
   the private run ledger. Bind a real SSH user, host key, client agent and target
   launcher; do not substitute invented host keys or fixture authority.
4. Complete explicit browser user-session authentication and LIVE-01: all nine
   remote operations, host-key mismatch, Access denial/expiry, local SSIAG denial,
   required audit failure, profile revocation, connector loss, lost reply and
   observation-only recovery. Record macOS-controller/same-host Linux-guest
   topology honestly; it is not a separate-physical-host or WAN claim.
5. Verify old/new token behavior, explicit retirement and independently observed
   cleanup. Record every residual resource or retained credential reference.
   Re-run the integrated gate against the exact final candidates, then close the
   canonical SCLV record after completed ratified work. No canonical closure is
   appended while these required cases remain open.

Local evidence is retained under workspace `implementation/cloudflare-i6/evidence/`.
Private operational credentials, Access assertions and provider response bodies
are excluded from repository evidence. No public release or deployment is made.
