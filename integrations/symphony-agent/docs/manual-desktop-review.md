# Manual desktop installation and recording

Symphony is headless. The recorded interface is the host conversation, plugin installation controls and actual MCP tool results. Native Computer Use is not a Symphony dependency; manual desktop operation can supply the required GUI evidence.

## Updated client evidence

The exact revision 2 Intel candidate was rechecked after the desktop app update to 26.1016.62342 (21477), with bundled CLI 0.163.0-alpha.11.2. [Eleven focused checks](updated-client-results.json) passed: package/native executable identity, native extraction, declared privacy without invented terms/video links, isolated installation, two skills and onboarding, missing-profile failure, private read-only setup, six-tool connection, pinned qxctl and health boundaries, command discovery and a real native template read. No owner write or model turn ran in this increment; broad platform/model campaigns were not repeated.

The archive remains `symphony-0.1.0-alpha.1-macos-x86_64-codex-review-20261023.2.zip`, SHA-256 `f0422b475181ba8f0c2a4aaa68f17383d0f8e146f45d49acff415841a7be9985`. No package or executable bytes changed. Initial harness probes are preserved separately; a digest assertion was corrected to include the native `sha256:` prefix. Pre-install metadata may have a null onboarding field; the installed setup skill was checked independently.

## Record the actual UI

1. Extract the exact candidate to a private review workspace. Add it to a local marketplace with installation policy `AVAILABLE`, following [OpenAI's packaging instructions](https://developers.openai.com/plugins/build/plugins). Register the marketplace with `codex plugin marketplace add /absolute/path/to/review-workspace`. Registration is not installation.
2. Open the desktop Plugins directory and select the review marketplace. Refresh or restart the desktop app if the new source is not visible. Start screen recording before clicking Install. Capture any actual errors or reconnect steps.
3. Install Symphony and use its setup skill in a fresh local conversation. Select the pinned reviewer qxctl and read-only connection. Verify the real six-tool connection and run the supplied five positive and three negative [review cases](reviewer-setup.md).
4. Exercise installation during an existing GUI conversation and desktop disable/uninstall/reinstall separately. Check existing and fresh conversations independently. Keep retained profiles and native owner records distinct; [API lifecycle observations](desktop-lifecycle.md) do not establish GUI behavior.
5. After the refusal scenarios, explicitly authorize cleanup of the disposable owner configuration. Select a cleanup-only connection enabling `qxcmd:symphony:sov.target.remove`, inspect the current digest, prepare/apply the exact removal, verify absence, and restore read-only. If the configuration is already absent, skip removal. This later explicit cleanup is separate from the negative case's request to bypass a disabled command.

Preserve original clips. Replay them for legibility and private collateral, record observations and timestamps, and hash the approved review copy. Only declare `extensions.com.openai.review.demo_recording_url` after a real reviewer-accessible URL is approved and verified. A local recording, scripted transcript or expected output is not a verified public demo URL.

## Remaining requirements

The local marketplace is staged for the recording, but the real GUI installation and video remain pending. This locally produced candidate also does not qualify the browser-download/quarantine path. Its unsigned Intel binary and prior Gatekeeper rejection remain disclosed. Applicable terms, OpenAI local-MCP organization approval and identity, and native distribution/signing arrangements remain open. NVIDIA qualification stays deferred. See [submission status](submission.md).
