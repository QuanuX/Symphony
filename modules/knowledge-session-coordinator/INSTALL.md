# Knowledge Session Coordinator Installation

## Requirements

- CMake 3.25 or newer
- a C++26-capable compiler
- a single-configuration CMake generator when building the foundation from this monorepo
- the checked-out Symphony monorepo, or a previously installed compatible `SymphonyKnowledgeVectorEngine` CMake package

## Build and Test from the Monorepo

```bash
cmake -S modules/knowledge-session-coordinator \
  -B build/knowledge-session-coordinator \
  -DBUILD_TESTING=ON
cmake --build build/knowledge-session-coordinator
ctest --test-dir build/knowledge-session-coordinator --output-on-failure
```

## Build Against an Installed Foundation

```bash
cmake -S modules/knowledge-session-coordinator \
  -B build/knowledge-session-coordinator-installed \
  -DBUILD_TESTING=ON \
  -DSYMPHONY_KVE_USE_INSTALLED=ON \
  -DCMAKE_PREFIX_PATH=/foundation/prefix
```

## Install

```bash
cmake --install build/knowledge-session-coordinator --prefix /chosen/prefix
```

The executable is installed at:

```text
libexec/symphony/knowledge-session-coordinator/0.1.0-dev/symphony-knowledge-session
```

No unversioned alias is created, no version is activated, and no Maestro receptor is contacted. Direct invocation remains available through the exact installed path.

## Uninstall

```bash
cmake -DINSTALL_PREFIX=/chosen/prefix \
  -P build/knowledge-session-coordinator/uninstall.cmake
```

The procedure removes only files named by this version's receipt model. Canonical knowledge, reconciliation journals, authenticated-session journals, persistent SSFV-maintenance journals, report/apply lifecycle journals, content-addressed applied evidence, protected lifecycle profiles/runtime state, other versions, user files, and containing directories are preserved. qxctl can bind an exact installed version and administer reconciliation, authenticated sessions, SSFV maintenance, report-only lifecycle planning/journaling, and separately authorized apply coordination through that bound executable. The coordinator never mutates its own package. This package now emits an immutable receipt v2. qxctl may install a candidate side by side and select it only after the candidate reproduces the exact prepared lifecycle journal; the invoking coordinator remains the finalizer on the uninterrupted path, while the newly selected coordinator can replay the same active action after a crash. Superseded versions are retained unless a separate exact removal action is authorized.

A repeated install of this exact module/version fails before any owned-file install rule. Install another version side by side or run the receipt-verified uninstaller first. The build-local uninstaller validates the configured ownership set and every remaining file's recorded size and SHA-256, removes the receipt last, and treats already-missing owned files only as idempotent retry evidence.
