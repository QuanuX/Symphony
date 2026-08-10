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

The procedure removes only files named by this version's receipt model. Canonical knowledge, reconciliation journals, authenticated-session journals, lifecycle boot journals, protected lifecycle profiles, other versions, user files, and containing directories are preserved. qxctl can bind an exact installed version and administer reconciliation, authenticated sessions, report-only lifecycle planning, and durable lifecycle boot/status/recovery through that bound executable. qxctl collects and journals report-only lifecycle evidence but does not install, upgrade, roll back, activate, dock, or uninstall packages through the lifecycle surface; receipt-owned install/uninstall remain explicit CMake operations.
