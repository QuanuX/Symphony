# Public Alpha submission — 23 October 2026

Status: **review candidate; external submission requirements remain open**. QuanuX is the publisher and https://quanux.org is the user-confirmed website. This is a metadata, skill and reviewer-setup revision of runtime `0.1.0-alpha.1`, not a new runtime release or OpenAI approval. No merge, release or submission has occurred.

## Completed preparation

The [platform record](platforms.md) preserves eleven native/WSL candidates and six clean Linux package checks. [Codex acceptance](codex-acceptance.md) adds actual installed-plugin/client checks, live model scenarios, private IPC reconnect and disposable native SOV configuration recovery. Five positive and three negative cases are included under `extensions.com.openai.review.test_cases`; the first unrelated-task failure and its successful correction/retest remain disclosed. The package offers `symphony-setup` onboarding and a [disposable reviewer setup](reviewer-setup.md).

NVIDIA DGX/Spark qualification is a **later milestone**, not a prerequisite for the initial Codex submission. Generic Ubuntu ARM64 does not certify those devices. Claude and Antigravity adapter files remain available for later actual-host qualification. Neither adapter presence nor a native binary build is a GUI acceptance claim.

## Distribution and publisher requirements

The local C++ MCP server uses the documented desktop-only distribution path. Public local-MCP submissions require the submitting organization to be allowlisted by its OpenAI representative. After confirmation, the documented portal flow selects Skills only and uploads the package containing its local server. Submission access and identity verification are separate prerequisites. [Desktop-only guidance](https://developers.openai.com/plugins/deploy/desktop-only), [submission guide](https://developers.openai.com/plugins/deploy/submission), checked 23 October 2026.

QuanuX, its public contact email `duncan@quanux.org`, contact route https://quanux.org/contact and original square website branding are recorded. `/support` is a contribution campaign and is not presented as technical support. [Asset provenance](website-branding.md) remains preserved.

The public https://quanux.org/privacy page was refreshed on 23 October after the publisher's deployment completed. Its notice, last updated 22 October 2026, explicitly covers the official local Symphony plugin, independent installations, connected AI applications, local retention/removal and separately supplied support information. The manifest now declares `privacyPolicyURL`. No applicable general plugin/service terms URL or genuine demo URL has been verified, so those fields remain absent. The [technical data disclosure](privacy.md) describes implementation behavior; verification of published coverage does not establish legal certification or OpenAI approval.

## Remaining submission checks

- Confirm submitting account/business identity and local-MCP allowlisting with OpenAI; confirm public OS/architecture distribution, WSL launch arrangements and native signing requirements.
- Publish or supply an applicable publisher-approved terms URL. Published plugin privacy coverage is verified and included.
- Exercise fresh desktop GUI installation/setup, installation during an existing GUI conversation and download/quarantine/uninstall behavior on the intended supported desktop platforms. App-server/CLI checks do not close those GUI checks.
- Record a genuine reviewer-accessible demonstration video using the supplied cases and disposable setup. No separate service credentials are required for the local owner-filesystem configuration scenarios; reviewers need their authorized host account and local qxctl.
- Select final exact artifacts, validate them in the approved portal and obtain publisher authorization to submit. Portal acceptance of a draft does not imply review approval.

The five positive and three negative cases and accessible recording are review materials described by OpenAI. Local deterministic transcripts and model-case summaries are supporting evidence, not a substitute for the video. Broader native-engine/SSIAG/provider workflows require their own qualification before corresponding claims.

[Desktop lifecycle preparation](desktop-lifecycle.md) adds focused existing-thread, disable/re-enable, uninstall/reinstall and macOS extraction checks. It records active-thread disable limitations, retained connection profiles and an unsigned Intel binary rejected by Gatekeeper assessment. Genuine GUI/download checks and the recording remain open; the prepared recording workspace does not close those requirements.

## Packaging

Build a package for each selected platform with its matching binary. The tested WSL binaries run inside Linux guests; the explicit Windows-to-WSL launcher needs host-specific configuration and is not a native Windows executable. Do not upload an outer evidence bundle as a plugin. The upload ZIP has one plugin root, `plugin.json`, `mcp.json`, skills, assets, source/dependency licenses and the selected executable.

Current documented limits are 100 MB compressed, 512 MiB extracted, 5,000 entries and 100 MiB/member. Exclude profiles, sockets, credentials, customer data, private transcripts and developer-machine paths. No `.app.json`, remote service or hook is fabricated to avoid local-MCP eligibility. [Validation reference](https://developers.openai.com/plugins/deploy/submission-errors).

The announced Symphony Alpha date does not establish OpenAI review timing, acceptance or publication. Keep the runtime's synchronous bounds, serial optional IPC and exact protocol baseline visible; preserve native owner authority and unknown-outcome recovery.
