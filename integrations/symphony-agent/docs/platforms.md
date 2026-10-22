# Platform qualification — 22 October 2026

Eleven distinct platform candidates were built and exercised on standard GitHub-hosted runners. These are Symphony plugin/bridge builds; this project does not build or distribute OpenAI's Codex desktop application. The runtime source remains the frozen C++26 Alpha implementation with MCP 2025-11-25. Windows checkouts differ only by recorded CRLF line endings in some core files.

| Target | Observed result |
| --- | --- |
| Ubuntu 22.04, 24.04, 26.04 — x86-64 and ARM64 | All six native builds passed four suites without skips. Each exact ZIP then passed 20 MCP/Unix-socket cases in a fresh matching native container without a compiler. |
| Apple Silicon — macOS 15 and 26 | Both native ARM64 builds passed four suites without skips; no Rosetta. Deployment target 14.0 is a build setting, not macOS 14 execution evidence. |
| Windows Server 2025 → WSL 2 — Ubuntu 22.04, 24.04, 26.04 x86-64 | All three passed four guest suites without skips, plus a cold Windows `wsl.exe` MCP initialization, tool listing and fixture read. |
| DGX OS 6/7 x86-64, enterprise ARM64, DGX Spark | Hardware/vendor-OS qualification pending. Generic Ubuntu ARM64, GPU VMs and containers do not close these lanes. |

Each four-suite release run covers 44 transport checks, five hash checks, 20 MCP cases and four adapter cases. Ubuntu 26.04 on both CPU architectures also passed ASan/UBSan runs for the instrumented transport/MCP targets. No real Symphony account, provider or native-owner mutation was exercised in this matrix. See the machine-readable [platform results and archive hashes](platform-results.json).

Evidence: [native builds](https://github.com/QuanuX/Symphony/actions/runs/39854747330), [corrected WSL runs](https://github.com/QuanuX/Symphony/actions/runs/39855798088), [clean package execution](https://github.com/QuanuX/Symphony/actions/runs/39857103515). The initial overall build run is red because an unnecessary conversion of already-version-2 WSL distributions failed; all eight native jobs passed. The dedicated WSL rerun removed that conversion and passed. An earlier package run stopped before testing because its evidence parent directory was absent; the corrected run passed all six packages. Failed runs remain visible.

The workflow is manually dispatched by suite. Read-only repository permissions, pinned Actions, separate unprivileged Linux fixture users and no real connection credentials are used. Artifacts expire after 14 days in GitHub; the review supplement preserves exact downloaded ZIPs and checksums. The Linux provisioner is for disposable Ubuntu runners only, never a DGX appliance.

[Windows setup](windows-wsl2.md) supplies explicit Windows-to-WSL adapters. Actual Codex/Claude/Antigravity UI, native-owner authority/recovery, clean desktop installation/uninstall, macOS minimum-version execution, signing/notarization, applicable privacy/terms and OpenAI local-MCP eligibility remain release gates. No merge, GitHub Release or marketplace submission has occurred.

The dated Intel-only verification and website submission documents shipped inside the tested ZIPs are preserved as build-time snapshots. This document and its linked results supersede their platform-pending statements only; they do not close the other public review gates.
