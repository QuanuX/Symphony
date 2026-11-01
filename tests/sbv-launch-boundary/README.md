# SBV installed MVP boundary qualification

This harness exercises installed SBV `0.33.0-dev` through independently built
qxctl and the production SOV stdio adapter. It selects a same-host, same-UID,
private `owner_filesystem` target. Its actual transport is local process IPC;
there is no SSH connection, network host, SSIAG grant, live provider, GPU, or
checkpoint resume in this profile. The earlier selected M3 SSH qualification
remains dated evidence in `modules/sbv-engine/M3-QUALIFICATION.md`.

Run after installing the quantitative dependency group:

```sh
sh tests/sbv-launch-boundary/run.sh /absolute/installed-prefix /absolute/new-work
```

The destination must be new and outside the source checkout. Select the exact Go
version from `tools/qxctl/go.mod`, CMake 3.31 or later, and the profile's C++26
compiler. The script uses fresh private module/build caches, `GOWORK=off`, and
copies both Go module roots out of the checkout. Dependencies resolve through
Go's public proxy/checksum database. An optional explicit
`SBV_QUALIFICATION_MODULE_SEED` is copied into the new module cache for offline
local qualification; the caller's cache is never modified. Ambient private/insecure proxy and build
settings are cleared, authentication is disabled, and both copied modules must
pass `go mod verify` before and after the isolated readonly builds. The environment
and verification logs are retained as build provenance. Native execution
uses the selected prefix and receipt-owned interfaces. The tiny data generator
and feedback provider compile outside the checkout using installed public
headers; they do not link SBV implementation sources.

The mandatory cases are:

1. Generate exactly 1,200 bytes: the attributed public Databento DBN v3 header and
   record layout with 15 deliberately synthetic events for instrument 5482.
   Select two spaced trades, ordinals 3 and 5, with one worker. A two-signal cap
   leaves additional eligible trades: the result must retain that partial scope.
2. Run the native census and two `observed_trade_levels` evaluations with explicit
   one-level/anchor choices. Census membership, identity, order and evidence
   remain unchanged, including after removing the original ancestor result.
   Reject a bare signal-array reference without publishing an output. Compare
   complete native result JSON with CLI export using exact JSON numbers.
3. Cancel queued work without executing it. Execute a small installed account
   template through real controller, adapter, target and native processes. A
   test-only carrier drops its actual completed reply. The controller must
   report `response_lost`, never retry, and then observe the original receipt.
   An explicit target-owned artifact projection then permits exact remote result
   inspection and query parity. Duplicate execution and reconciliation preserve
   generation and journal bytes;
   late cancellation cannot rewrite completion.
4. Execute the installed account template with a public-ABI feedback fixture
   blocked at a known event. Actual cooperative cancellation must end in
   `cancelled` with no result. In a separate invocation, SIGKILL only that blocked
   native worker, identified by its private fixture marker. Reconciliation must
   preserve `unknown` without result adoption or automatic re-execution.
5. Inject an assertion failure while another native fixture is blocked. Cleanup
   must release the barrier, propagate cancellation through the isolated process
   chain, reap the controller, and leave no blocked native worker.

`evidence.json` reports each assertion, binary hashes, receipt digest, data size,
actual host architecture and transport/authority limits. `outputs/` contains the
exact synthetic requests and results, source commit and working-diff observation,
fixture attribution, and durable journal evidence. `logs/` contains bounded
process observations. Failure exits nonzero; there is no optional/skipped case.
The 20/25/45/60-second bounds are harness deadlines, not SBV policy limits.
Keep failed evidence; do not interpret an absent success file as a pass.

The quantitative CI matrix runs this after the installed native suites on
macOS 15 ARM64, Ubuntu 24.04 AMD64 and ARM64, and Ubuntu 26.04 AMD64. A local Intel
macOS run is development evidence only. A hosted ARM result satisfies the user's
current Ubuntu ARM qualification choice; it makes no assertion about a physical
DGX Spark image, driver, GPU or filesystem. Success is the recorded small CPU
profile, not large-scale, WAN, independent-principal or performance acceptance.
