# Installed SDK distribution and removal

Run after a completed `tests/mvp-dependencies` group and its selected native
campaign. The original installed prefix is read-only to this harness:

```sh
sh tests/native-sdk-distribution/run.sh knowledge /absolute/mvp-installed \
  /absolute/new-sdk-work /absolute/snv-qualification
sh tests/native-sdk-distribution/run.sh quantitative /absolute/mvp-installed \
  /absolute/new-sdk-work /absolute/sbv-launch-boundary
sh tests/native-sdk-distribution/run.sh cloud /absolute/mvp-installed \
  /absolute/new-sdk-work
```

`bootstrap` is also supported for the foundation-only focused regression. Select
Go 1.26.5, CMake 3.30+ and the same C++26 compiler used for the installed graph.
The existing four-target matrix supplies fresh dependency prefixes in exact
runtime DAG order. This gate consumes those final artifacts and completed
consumer logs; it does not rebuild the production graph or count test-only
cycles as runtime dependencies.

`manifest.json` accounts for all 23 native SDK producers. The inventory refuses
new/unclassified exports or shipped consumer directories. Existing MVP gates
already copy and execute all 14 ordinary SDK consumers, including the native
source adapters whose examples are outside CTest. Their copied source bytes and
successful configure/build/run steps are required here. The special SBV C ABI,
C++ SDK/interop and loader gates remain required; the five-owner SNV/observer
campaign is bound to the exact selected receipt bytes. A small additional copied
consumer imports the foundation and, where selected, neutral SNV common SDK. It
includes the public manifest/operation headers and calls actual installed code.

The shipped Go SDK `examples/roundtrip` is copied with its independent module and
built with `GOWORK=off`. In the quantitative group it consumes L20's exact qxctl,
native installation and retained synthetic result/source. Its two explicitly
supplied probabilities are synthetic declarations. Its native evaluation,
streaming, cancellation, experiment retry and selection checks make no live
provider request or market calibration claim.

The pinned DuckDB connector consumers retain their separate four-platform
workflow; their frozen embedded SDK identities are intentional. IBKR SDK 10.45
object conformance requires separately supplied licensed material and is an
explicit external exclusion. The public IBKR SDK consumers and fixture session
checks remain mandatory. An unavailable licensed SDK is never counted as run.

For each selected package, this gate verifies the final receipt and copies the
configured uninstaller, strict receipt support and expected-identity helper into
a detached directory. The configured owned-file list must exactly equal the
receipt; missing/duplicate public headers cannot disappear from removal coverage.
Each command has a two-minute bound. A private supervisor retains the Unix
process-group identity until command completion and output draining; cancellation
and an early-exited command with inherited open pipes both receive bounded group
cleanup before that supervisor is reaped.

Only a disposable copy of the prefix is mutated. The gate exercises changed
owned bytes, SDK headers and CMake exports, an invalid receipt digest and a
self-resealed wrong package identity. Every refusal must preserve all remaining
bytes and modes. It then simulates an interrupted removal by deleting one owned
file, confirms consumer receipt admission refuses the incomplete package, and
runs the configured resumable uninstaller. Reverse runtime DAG removal and an
exact retry must preserve every unrelated package and caller sentinel. The
original installation is checked unchanged at the end.

This is the existing unmanaged-prefix CMake ownership contract. It is neither
an authenticated shared-root reclamation request nor publisher-signature proof;
L10/L11's package/lifecycle gates retain those distinct responsibilities. No
service, provider, broker or production installation is activated.

Retain `evidence.json`, `metadata/`, `logs/` and the source consumer's small
`source-roundtrip-output/report.json`. Evidence includes exact original file
hashes/modes, projection/coverage identity, completed assertions and remaining
unrelated files. Failure writes `status: failed`; an absent success file is not
completion. Do not upload copied libraries, executables, caches or private
synthetic working stores. Fresh four-target execution must pass before this
profile is reported complete; a focused local bootstrap pass is narrower.
