# SBV selected M3 release qualification

**Current decision — 13 October 2026:** the selected M3 release gate is complete,
including the optional authenticated macOS-to-Linux guest SSH profile described
below. Earlier local-only and remote-pending entries remain dated evidence. This
decision qualifies the selected R6 implementation; it does not claim that every
optional capability in the original 96-case register is implemented.

The resumable [follow-on note](FOLLOW-ON.md) records remaining deployment,
authority, scale, device, data, model, visualization and comparison work, with
prerequisites and evidence needed before claiming each capability.

The selected local R6 profile is qualified on macOS with native 0.33.0-dev,
the optional Go adapter 0.1.7-dev, and the SOV process adapter 0.1.0-dev.
The private workspace packet `implementation/sbv-m3-qualification` records
exact source, build, test, result, timing and original-case provenance.
It is evidence for this selected profile, not full original-register acceptance.

The gate resolves 55 native suites, four selected ASan/UBSan suites, installed
qxctl/native checks, command discovery, Go race checks, independent C/C++
consumers and complete terminal exports. The original broad runs and their
focused corrections are retained; this is not a claim that every initial run
passed. The independent reviewer reproduced and verified the transport
cancellation, malformed-reply and FIFO regressions on final binaries.

Native scientific code remains unchanged from 0.33. New SOV operations expose
exact configuration, invocation, inspection and removal through qxctl. The
independently installed adapter runs the selected target process. Native jobs
preserve one admitted attempt across lost replies and explicit reconciliation.
See `knowledge/sov/PROCESS-EXECUTION.md` for the precise authority, outcome,
resource and cancellation contract.

## Reproduce on a second host

Use the exact published branch commit and a clean private checkout. Record the
commit, OS, compiler, CMake, Go, selected environment, commands, exit codes and
full logs. Select a compiler supporting the repository's C++26 requirements,
CMake 3.30 or newer, and the exact Go version in `go.work`. Do not silently
downgrade the language standard or replace source-selected dependencies.

From the repository root, select new absolute private build/install paths:

```sh
cmake -S modules/sbv-engine -B "$SBV_BUILD" -DCMAKE_BUILD_TYPE=Release -DBUILD_TESTING=ON -DCMAKE_INSTALL_PREFIX="$SBV_PREFIX"
cmake --build "$SBV_BUILD" --parallel "$SBV_BUILD_WORKERS"
cmake --install "$SBV_BUILD"
ctest --test-dir "$SBV_BUILD" --output-on-failure
go build -o "$SBV_QXCTL" ./tools/qxctl/cmd/qxctl
go build -o "$SBV_ADAPTER" ./tools/sov-stdio-adapter
go test -race ./tools/qxctl/internal/sovremote ./tools/sov-stdio-adapter ./libraries/sbv-go/...
go test ./tools/qxctl/cmd/qxctl -run 'SBV|SOV|CommandManifest|CommandRegistry|ValidateSessionAuthorization'
go vet ./tools/qxctl/internal/sovremote ./tools/sov-stdio-adapter ./libraries/sbv-go/...
```

Set `SYMPHONY_SBV_TEST_PREFIX` and `SYMPHONY_SBV_RESOURCE_PREFIX` to the installed
prefix before installed Go boundary tests. Inspect fixture requirements in
`tools/qxctl/internal/knowledgeengine/sbv*_test.go`; optional fixture tests can
skip without those inputs. Record each skip as unqualified, not passed. Native
test targets generate their own mechanical fixtures. Private retained Databento
files are not committed and must not be inferred to exist on the second host.
Do not copy licensed data to a service without the applicable transfer rights.
No fresh provider acquisition or live connection belongs to this campaign.

The private local gate tested real staged controller → adapter → target →
installed native processes, including a dropped reply after completion,
duplicate submission, target refusal, running cancellation, all nine selected
read/lifecycle operations, and full result-node equality. Reproduce those
boundaries using schema/template discovery and caller-selected private paths.
Record exact executable and installation hashes; never reuse the local Mac
binary as a Linux build. Both deadlines and resource limits are caller choices.

## What a second-host pass establishes

A remote build and test pass establishes that exact host/toolchain profile.
Running all controller/target processes on that host still tests a local
process transport. Cross-host SOV qualification additionally needs an actual
selected network carrier, authenticated target identity, configured target
authority, allowed/denied/expired/revoked decisions, dropped/late replies,
cancellation and target-loss reconciliation. Shared-storage durability, cluster
resource placement and hard remote termination each require their own positive
evidence. The stock adapter does not claim those capabilities.

## Remaining scope

The current gate does not establish calibrated market fill probabilities,
general exchange matching, nonlinear derivative margin/liquidation, device
runtime integration, borrowed Arrow/DLPack buffers, NUMA/terabyte allocation,
cold-cache results or competitive speed superiority. CUDA/tensor seams remain
capability contracts. The fresh primary competitor review records nine products
and version/edition limits; no peer binaries were benchmarked. These distinctions
preserve the original plan and user authority over future models and deployment.

## Portable fixture roots and Linux storage

Native fixtures and the installed Go resident-control check select canonical
`/tmp` by default. Set `SYMPHONY_TEST_TMPDIR` to an existing private scratch parent
on the filesystem being qualified; retain the selected path and mount properties.
The short default avoids incidental Unix-domain socket path exhaustion. Supplied
fixture/output arguments still take precedence where supported. Run this source
revision without proot or LD_PRELOAD path rewriting. Do not reuse prior fixture
products or same-version install receipts as evidence for a changed binary.

Linux SQPV uses renameat2(RENAME_NOREPLACE) and the existing file/directory fsync
and flock contracts. Run its owner crash/recovery/collision tests on the selected
filesystem, then SBV source owner/dataset and installed recovery checks. The
qxctl-visible source-owner profile must identify the actual host platform.
A passing container overlayfs or tmpfs profile does not certify power loss, reboot
persistence, underlying physical locality or a cross-host/network carrier.


## Selected Linux portability gate — 13 October 2026

Debian 13 amd64 with Clang19.1.7, CMake3.31.6, Go1.26.5 and private libcurl8.7.1
qualifies the Linux portability revision. The exact source implementation is
`49702801b4d0ef21f2efc86b0caaadcda1132862`; the private workspace packet
`implementation/sbv-linux-portability` binds the baseline, two applied patches,
source hashes, fresh installation, commands and raw result provenance.

The 55 native suites resolve through the initial broad run and focused corrections:
three socket fixtures pass with a shorter caller-selected scratch path, and the
administration contract passes after canonical SQPV registry/profile digest refresh.
Four ASan/UBSan/leak suites pass without shims. Eight selected installed Go checks
pass with zero skips, including owner evidence, three recovery cases, resident
control and partitioned source. Installed qxctl reports
`linux_caller_selected_filesystem` and verifies source retention/export, file and
retained runs, and result inspection/query/export. SQPV passes three overlayfs gates
and two tmpfs gates, with actual process-crash and exclusive-collision evidence.
No proot, preload or path/ownership emulation is used. Initial failures are retained.

The final administration rerun uses the unchanged native test executable and a new
follow-up evidence directory; the earlier cloud archive remains sealed. This is
selected second-host qualification. It does not close the remaining original M3
carrier, deployed authority, cluster, device, durability or peer-performance lanes.

## Selected cross-OS SSH and authority gate — 13 October 2026

Runtime source `af4cd24d05b9881aa1496abe805b2b109d33d508` qualifies the optional
SOV SSH carrier from macOS into an isolated Ubuntu 26.04 Linux guest. Linux uses
Clang 19/C++26, Go 1.26.5 and exact libcurl 8.7.1. The guest has a separate
LinuxKit kernel but shares physical hardware with the Mac; TCP reaches its sshd
through a loopback-only Docker port mapping. Docker provisions fixtures and
injects faults. Every claimed remote job/result operation traverses pinned SSH
and target qxctl. This establishes the selected virtualized cross-OS deployment,
not a separate physical machine, WAN, private cloud route or cluster.

Real target-local SSIAG and STAV services qualify same-UID, user-scope authority.
The Linux guest uses its isolated root account; this is not system-scope or
distinct-principal service isolation. macOS tests cover real-service grant,
wrong-resource/default denial, exact policy pinning, revocation, audit outage
and expired decisions. Expiry is deterministic: a test-only Unix-socket relay
delays a genuine SSIAG allow response beyond its actual expiry. Linux verifies
fresh grants, default denial and revocation over the selected SSH route.

All nine selected lifecycle/result/bundle operations pass. qxctl authors jobs,
target configuration and SSIAG policy; remote qxctl retrieves all 899 account
nodes and all 374 nodes of a retained Databento replay with exact JSON types.
An independent Python reader checks every value, child count and pointer. The
macOS account output is identical to Linux. The Databento outputs differ only
in two declared source paths and the content digest binding those paths; all
scientific, signal, execution and replay values agree. A caller-selected one-signal
cap correctly produces `completed_partial`, with the omitted population visible.
The replay includes observed DBN events and explicit unavailable execution fields;
it is not a calibrated fill-probability claim.

Queued and running cancellation, stale generations/cursors, removed or missing
artifacts, killed native worker, killed controller, killed SSH connection and
response loss after target completion are covered. Re-observation preserves the
original claim/result identity and never automatically re-executes admitted work.
Controller loss and connection loss remain distinct tests. An unknown job outcome
is not checkpoint/resume, and disconnection does not imply remote termination.

The resolved gate combines passing focused follow-ups with retained earlier
attempts: initial fixtures mishandled sshd environment, a timing-only expiry test,
and the expected partial status of capped data. These were harness corrections;
no production runtime fix was required. The earlier 55-suite native gates,
four sanitizer suites and carrier/installed checks remain attributed to their
exact source. Unchanged broad suites were not rerun. Independent final review
found no actionable issue in this selected profile.

Private evidence: `workspace:implementation/sbv-m3-remote-final/`, particularly
`CHECKS.json`, `CROSS_OS_PARITY.json`, `FINAL_REVIEW.json`, `MILESTONE_DECISION.json`
and the sealed command/output and native-job journals. The temporary container,
services and private keys were removed; exact Linux installed binaries were
retained locally. No provider/live request or new historical-data spend occurred.

R6.1, R6.2 and R6.3 are complete within this selected M3 release. Physical-host/WAN
deployment, distinct-principal authority, cluster placement, shared-storage/power-loss,
NUMA/terabyte and GPU execution, checkpoint recovery and equal-fidelity installed
peer benchmarks remain explicit follow-on qualification or implementation work.
Existing CPU/RAM measurements and cross-OS comparisons support their stated
profiles only; no fastest-engine or competitor-speed ranking is asserted.
Source/data placement remains an owner-selected prerequisite managed through
target qxctl and the chosen provisioning route. No aggregate dataset/RAM limit
or mandatory job deadline has been added. This closes the selected release gate;
it does not close the platform's evolving capability backlog.
