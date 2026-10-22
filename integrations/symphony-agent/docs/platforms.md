# Platform qualification

The C++ runtime supports POSIX macOS and Linux. Build candidates are specific to OS series and CPU architecture. Windows hosts use a WSL2 Linux executable through `wsl.exe`; no native Windows binary or named-pipe transport is implemented.

The new CI workflow runs bridge fixture tests on Ubuntu 22.04/24.04/26.04 x86-64 and ARM64, macOS 15/26 Apple Silicon, and attempts actual WSL2 tests from Windows Server 2025. A job is evidence only after its recorded tests pass without required skips. Native Ubuntu 26.04 jobs include ASan/UBSan. Read-only repository permissions, pinned Actions and unprivileged fixture execution are used. No real Symphony account, provider, service or mutation is exercised.

Local container tests have passed on Ubuntu 24.04 and 26.04 x86-64. An Apple Silicon binary has cross-compiled on Intel macOS but has not yet executed on Apple Silicon. Additional local/hosted runs are in progress; no unobserved result is claimed here. The original verification document is a dated Intel macOS baseline.

DGX OS 6/7, enterprise ARM64 and DGX Spark need separate hardware/vendor-OS evidence. Generic GPU clouds, Ubuntu containers and ARM emulation do not certify those installations. The provisioner in `ci/` is for disposable Ubuntu runners only; do not run it on a DGX appliance.

Public signing/notarization, minimum-macOS execution, installed qxctl authority, Codex/Claude/Antigravity UI integration, clean-machine install/uninstall and OpenAI local-MCP eligibility remain separate release gates. No platform artifact is an approved public release merely because it compiles or passes fixtures.
