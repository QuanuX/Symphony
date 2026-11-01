# SQAV attempt administration install

Build `modules/sqav-attempt-engine` with CMake 3.30+, a C++26 compiler and the exact native dependencies listed in SPEC. Install into an explicitly selected prefix. The engine, contracts, resources and licenses are receipt-v2 owned. Use the existing qxctl lifecycle receipt inspection. Target `uninstall-sqav-attempt-engine` performs guarded receipt-v2 removal. No global installation or automatic version selection is performed.

## Platform selection

Runtime support covers macOS local APFS and Ubuntu LTS Linux x86_64 and ARM64/aarch64. The macOS local-APFS gate remains enforced. DGX Spark uses the Linux ARM64 implementation; qualification of physical Spark hardware and its exact supported DGX OS image, kernel, drivers and runtime remains separate and pending. Generic Ubuntu ARM builds or tests do not qualify Spark or GPU behavior. Native Windows is unsupported.

Select the Linux filesystem and physical absolute retained-data root explicitly. There is no filesystem-name allowlist. The selected deployment must preserve the shared reader's no-follow path, shared-lock, coherent stat/inventory, ownership, permissions, link-count, bounds and identity-fence requirements. Failed checks refuse observation. Successful syscalls do not certify the underlying mount's topology, persistence or power-loss durability; retain separate deployment evidence for those claims. See SPEC for the exact read-only and dirty-state behavior.

## Build and verification

```sh
cmake -S modules/sqav-attempt-engine -B BUILD -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=PREFIX
cmake --build BUILD
cmake --install BUILD
```

`SYMPHONY_SQV_ADMIN_INSTALLED=ON` selects exact installed dependencies through CMAKE_PREFIX_PATH. The default builds the pinned source dependencies. Configure `tests/sqv-administration` to run the shared focused native gate and generate real writer fixtures for the installed qxctl tests. Test-only writers are absent from production inspection executables. See `tests/sqv-administration/README.md`.

Qualify each selected OS/architecture and filesystem using the exact source revision, compiler/dependency versions and installation receipt in a fresh prefix. Exercise the shared reader tests and the installed qxctl routes with real native writer fixtures, including active-writer exclusion, changed-identity refusal, finite bounds and dirty-state behavior. Record the actual host, kernel, filesystem and outcomes; these instructions do not assert that qualification has already passed. Production observation remains read-only and uses no provider credentials.
