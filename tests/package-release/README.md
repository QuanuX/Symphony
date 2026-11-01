# Package candidate and lifecycle regressions

Run from the checkout with CMake 3.30 or newer, a native C compiler, Git,
OpenSSH (`ssh-keygen` with SSH signature support), and `strip`:

```sh
umask 077
cmake -DWORK=/absolute/new/private-work -P tests/package-release/run.cmake
```

`WORK` must be a new directory outside the checkout. macOS also uses the selected
Apple toolchain's `codesign` with an ad-hoc identity. Linux and macOS on amd64 and
arm64 are supported fixture targets. No system installation or provider is used.

The harness builds two versions of a tiny executable and shared library. It
checks finalization before immutable receipt hashing, side-by-side installation,
relocation and native execution, immutable reinstall refusal, exact-version
removal and retry, preservation of foreign files, and managed-root ownership
fences. All forbidden finalizer path control bytes and CMake delimiters are
tested with real files and the expected admission diagnostic, before any
installation payload is written. An interrupted finalizer must leave no sealed
receipt; recovery installs into a fresh private stage and leaves the interrupted
stage unchanged for inspection.

Candidate tests use a small committed **fixture repository** as declared source
provenance and a disposable SSH key created under the private work directory.
They reject receipt/platform OS and architecture mismatches before candidate
output, verify the detached signature and bound archive/evidence bytes, extract
the verified archive into a fresh directory, relocate it, verify the unpacked
receipt/file inventory, and execute both relocated probes. Negative cases cover
the wrong signer or profile, a changed signed manifest, changed evidence/archive
bytes, tampered unpacked files, and validly signed but incompatible target
claims. Download-byte verification and unpacked verification produce separate
records (`downloaded_bytes` and `caller_supplied_payload`); the latter does not
claim how the caller obtained that directory. The harness separately records
actual extraction and execution. Successful verification alone does not imply
execution.

Malformed descriptors are signed directly with the fixture key, bypassing the
normal signing helper. Verification must reject source-field injection, invalid
boolean/size/hash types and values, unsupported target names, duplicate file or
component identities, and unknown fields before writing a verification record.
The normal signing helper must also reject a malformed descriptor before signing.

`logs/qualification.json` lists each successful assertion, including expected
refusals. Individual logs retain failures at their source. Inspect a failed
workspace; retry with a new `WORK`, rather than reusing partial staging or
candidate output. Test keys, archives and binaries remain outside `logs/`.

The `package-release.yml` workflow runs this harness on Ubuntu 24.04 amd64,
Ubuntu 26.04 amd64, Ubuntu 24.04 arm64 and macOS 15 arm64. It also runs 12 focused
qxctl tests with Go 1.26.5, the race detector and independent public module
resolution: runtime compare-and-swap/retry, receipt-v2 install/uninstall and
rollback evidence, ownership/reconciliation fences, and receipt/file tampering.
Any skipped selected test fails qualification. Only logs and verification
records are uploaded, never the fixture keys or candidate payloads.

These tests qualify packaging mechanics using synthetic provenance. They do not
qualify a complete product bundle, publisher identity, public download, archive
reproducibility, Developer ID signature, notarization, supported-host deployment
or release approval. Hosted results remain pending until reviewed for the exact
source tree.
