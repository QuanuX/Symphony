# SBV selected local M3 gate and remote qualification

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
