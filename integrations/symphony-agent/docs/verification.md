# Alpha verification — 22 October 2026

Candidate: `0.1.0-alpha.1`. Fresh local verification on macOS 26.7.1, x86_64, AppleClang 21.0.0 in checked C++26 draft mode; CMake 4.4.3. This is adapter verification, not fresh whole-Symphony qualification or approval by an agent host.

## Passed

| Boundary | Evidence |
|---|---|
| Native process and IPC | 44 checks: exact argv/environment, no shell interpretation, bounded combined output, deadline and process-group cleanup, graceful owner termination, forced termination, OS randomness, private socket modes, same-UID peers, existing-path refusal and inode-safe cleanup |
| Bounded hashing | Five checks: SHA-256 known vectors, size limit, FIFO and directory rejection |
| MCP process integration | 20 cases against the actual C++ binary: handshake, discovery, grammar, argv, environment, denied mutation, preview/apply, single-use/session binding, changed inputs/executable, FIFO and symlink/parent-path handling, uncertain exits/timeouts/output limits, invalid JSON/RPC, resources/prompts and real socket relay; no skips in the final permitted run |
| Client adapters | Four cases: JSON/TOML roundtrip, arguments remain data, invalid paths rejected, exclusive private output without overwrite. Later timeout-only adjustment verified with a focused TOML assertion |
| Sanitizers | The 20 MCP cases pass ASan/UBSan, including local IPC; the 44 native transport checks also pass ASan/UBSan |
| Skill | Bundled skill validator passes; independent behavioral review covers already-authorized mutation, response-loss recovery, repository-health interpretation and prompt injection; two wording findings corrected |
| Packaging schemas | Portable `plugin.json` and `mcp.json` pass official Agent Plugins 1.0.0 schemas with Draft202012Validator |
| Real qxctl | Fresh offline Go 1.26.5 build from Symphony commit `53d6a5a3bbdfac209b5885a690a8847f0144759e`; live manifest, discovery, describe, repository status and inventory pass through the release C++ bridge |
| Native runtime linkage | The macOS x86_64 executable links only system libc++ and libSystem; no build-machine shared library dependency |

The source-current qxctl manifest contains **461 records**; **459 are routable** through this Alpha. The hidden/prohibited `stav.append` command is excluded. `sov.target.invoke` is excluded because it requires a stdin control payload; this bridge deliberately provides `/dev/null` stdin. The existing controller-side remote commands remain discoverable under their own contracts. `routable` means the adapter can construct the registered invocation; it does not mean all 461/459 operations or engine installations were executed.

The system-PATH qxctl encountered during development was older and lacked command discovery. Verification used a separate current-source binary; no existing installation was replaced. The live source checkout remained clean and unchanged.

## Remaining qualification

- Real Codex desktop/plugin installation and end-to-end agent workflow, plus Claude and Antigravity host checks. Their configuration formats are implemented and tested as data; actual host behavior has not been certified.
- Native owner-authorized state changes through an isolated installed Symphony test environment. Mutation boundary/recovery tests currently use synthetic qxctl fixtures; real integration here is read-only.
- Linux runtime, Apple Silicon runtime, independent-UID rejection and system-scope delegation. The source includes Linux transport and SHA implementation; the supplied built candidate is macOS x86_64 only.
- Current MCP 2026-07-28 semantics, Windows transport and hosted HTTPS/OAuth. None is advertised as implemented.
- Clean-machine install/uninstall, platform signing/notarization as applicable, reviewer demonstration and public-directory acceptance.

Full SCV/SHV/SBV and historical installation suites were not rerun: those engines were unchanged. Existing engine test reports remain dated evidence. A fixture result does not establish real provider connectivity, deployed SSIAG/STAV authority, live trading, WAN behavior or general service health.

Public release requirements are tracked separately in [submission.md](submission.md). Private machine-specific logs and connection profiles are intentionally outside the distributable archive.
