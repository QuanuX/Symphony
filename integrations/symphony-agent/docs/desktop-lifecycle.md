# Desktop preparation and client lifecycle evidence — 23 October 2026

This increment exercised Codex 0.163.0-alpha.10.2's actual app-server on macOS 26.7.1 x86_64. It adds 17 focused lifecycle/extraction checks, with the limitations below. It does **not** establish native desktop GUI installation or a recorded walkthrough. The exact tested candidate remains `symphony-0.1.0-alpha.1-macos-x86_64-codex-review-20261023.2.zip`, SHA-256 `f0422b475181ba8f0c2a4aaa68f17383d0f8e146f45d49acff415841a7be9985`, packaged at commit `299281b207621a18c6e58626d324d728ae27a1d7`. This later documentation increment does not change that archive or its executable.

## Observed lifecycle

- macOS `ditto` extracted the executable with its execution permission intact; no chmod was needed for this check.
- A thread started before installation discovered the plugin and its two skills/onboarding. The missing profile failed closed. Adding a private read-only profile and reloading connected the same thread with six tools.
- Changing plugin enablement to false, calling plugin reconciliation, and reloading MCP did **not** revoke the already connected thread: its read-only connection tool remained callable. Plugin metadata correctly reported disabled. A new thread in the same process had no server; restarting the process also respected disable and rejected tool calls. Re-enable plus restart restored the connection. This tests a configuration/API sequence, not the desktop GUI disable control.
- App-server uninstall removed the plugin from subsequent thread inventory but retained its mode-0600 connection profile. Reinstall reused that profile and connected read-only. Therefore a reinstall is not necessarily a clean first-run setup. The GUI reset/uninstall path must be tested separately; do not promise it deletes local profile data.
- Independent native SOV configuration bytes survived disable and uninstall. Exact-digest native removal then cleaned up only the disposable owner configuration. Test plugin registrations were removed; retained profiles/transcripts are private evidence.

Until GUI behavior is verified, do not rely on a configuration toggle plus MCP reload to revoke an already active thread. End active sessions and restart the host, then verify the server is absent. Recheck private configuration on reinstall rather than assuming a blank state. These are observed host lifecycle limits; no bridge runtime or authority policy was changed to conceal them.

## macOS distribution checks

The Intel candidate is unsigned. `codesign --verify` reported no signature and `spctl --assess --type execute` rejected it with “no usable signature.” The locally produced ZIP had no quarantine attribute. Successful local execution does not establish a clean downloaded-app path. Browser download, quarantine prompts, signing/notarization and final distributor requirements remain open. No protection or quarantine was removed during testing.

## Demo status

A private recording workspace contains the exact candidate, pinned reviewer qxctl, read-only and one-command write profiles, all five positive/three negative prompts, a shot list and an unfilled GUI evidence worksheet. Preparation creates no owner configuration and installs nothing into the user's main desktop profile. An actual desktop recording and reviewer-accessible URL remain outstanding. Native UI controls are unavailable to the current automation environment; app-server transcripts are supporting evidence, not a video substitute.

The original probes are preserved, including the unsupported hot-disable assumption and the incorrect expectation that uninstall would erase the profile. The final record reports actual retention and restart behavior rather than turning those observations into deletion or GUI-success claims. NVIDIA remains deferred; applicable terms remain a publisher-confirmed release requirement.
