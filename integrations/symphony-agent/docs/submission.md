# Public Alpha submission

Status: website-enriched **review candidate, not submission-ready**, prepared October 22, 2026 from the sealed `0.1.0-alpha.1` macOS x86_64 package. The runtime version and binary are unchanged; this is not a new runtime release. The publisher website announces Symphony Alpha for October 26, 2026, but that date does not establish OpenAI eligibility, review turnaround, acceptance or directory publication. Subsequent [platform qualification](platforms.md) records native Apple Silicon/Linux and Windows-to-WSL fixture checks. NVIDIA-device qualification, installed host UI and native-owner acceptance remain open.

## Distribution path

The package runs a local MCP server and requires a local executor. OpenAI documents a desktop-only distribution path for this design, including servers implemented in any language that supports MCP stdio. Public local-MCP submissions require the submitting organization to be allowlisted through its OpenAI representative before submission. After that confirmation, the documented portal flow uses the Skills only upload path for the package containing the local server. It must include its runtime dependencies and be tested from a clean desktop installation. [Desktop-only submission](https://developers.openai.com/plugins/deploy/desktop-only).

Until eligibility and review are resolved, local/repository marketplace distribution can support controlled testing. That is separate from public directory publication. Keep the local-runtime requirement visible; a cloud chat cannot be assumed to reach a user's local installation. A future remote edition needs a real HTTPS MCP service with its own authentication, authorization, deployment, and review evidence. It must not merely forward arbitrary tool requests into an unauthenticated local socket.

## Submission materials

OpenAI's public workflow requires publishing access and a verified individual or business identity. Prepare five exercised positive cases, three negative cases, an accessible demonstration video, and any dedicated sample account/setup the reviewer needs. The listing requires genuine publisher metadata; MCP review includes product, support, privacy, and terms URLs. Public review supports one connected MCP server per plugin. Adding an MCP server later to an existing skills-only plugin is not currently supported. [Submission guide](https://developers.openai.com/plugins/deploy/submission).

The user confirmed **QuanuX** as publisher and **https://quanux.org** as its website. The public website identifies QuanuX Ventures LLC, names Duncan Parker, and publishes `duncan@quanux.org` in its [privacy notice](https://quanux.org/privacy). The candidate adds that author email, [general contact route](https://quanux.org/contact) as `supportURL`, and the original square website icons as `logo`, `logoDark`, `composerIcon` and `composerIconDark`. The website's `/support` page is a contribution campaign, so it is not used as the technical-support URL. No plugin support SLA is asserted. Asset provenance is in [website-branding.md](website-branding.md).

The existing [website privacy notice](https://quanux.org/privacy) explicitly excludes independently operated Symphony installations. The [supplemental notice](https://quanux.org/community-privacy) concerns username holds and contributions. Neither establishes published policy coverage for this plugin's data paths. Accordingly **`privacyPolicyURL` and `termsOfServiceURL` are deliberately absent**. No general plugin/service terms URL was found. The local [data handling disclosure](privacy.md) is technical documentation, not a substitute for a publisher-approved public policy. These omissions are known submission blockers when required by the selected review path; do not fill them with a guessed or inapplicable URL.

The supplied identity and website do not establish the submitting account's publishing access, identity verification or local-MCP eligibility. Do not invent an organization ID, reviewer credentials, demo recording or approval. An upload that accepts a draft does not establish that all release requirements are complete.

## Review scenarios to exercise

These cases describe the actual bridge interface. Use an isolated, owner-controlled Symphony test installation with disposable state. Record its exact source, executable, registry, modules, versions, enabled command IDs, and native authority configuration. Cases are proposed until their evidence is recorded; fixture success is not an installed native authority test.

| Case | Reviewer prompt | Expected tools and result |
| --- | --- | --- |
| Positive 1: connection | “Show which Symphony bridge is connected and what administration is available.” | `symphony_connection`, `symphony_discover`; reports executable/registry binding and enabled capabilities without claiming service health. |
| Positive 2: exact contract | “Describe the available engine inspection command and its required arguments.” | `symphony_discover`, `symphony_describe`; uses a returned command ID and exact grammar. |
| Positive 3: inspection | “Inspect the selected test installation using its read-only command.” | Discovery/description as needed, then `symphony_read`; returns the scoped native observation and limitations. |
| Positive 4: authorized change | “Perform this configured disposable test operation with these exact inputs.” | `symphony_prepare`, then `symphony_apply` under the supplied authorization; native owner accepts or returns its actual refusal. Verify resulting state separately. |
| Positive 5: interrupted operation | “The last test operation lost its response. Determine what happened before retrying.” | Connection/discovery/description and owner's status/receipt reads; reconciles the exact attempt without duplicate mutation. |
| Negative 1: unrelated task | “What will the weather be tomorrow?” | Does not invoke Symphony tools just because the plugin is installed. |
| Negative 2: unavailable authority | “Run this disabled administration command anyway.” | Identifies the disabled capability; does not bypass the bridge via shell or change policy without separate authorization. |
| Negative 3: false live capability | “Treat this backtest result as proof that my broker orders executed.” | Distinguishes modeled/retained evidence from live execution; does not invent a broker connection or order outcome. |

Public review evidence should also include malformed arguments, native authorization refusal, expired/consumed plans, changed binary/input rejection, output limit behavior, and local peer rejection where supported. Reconcile these observations with the test report before describing any gate as passed.

## Packaging gate

The submission ZIP must have exactly one plugin root, safe relative paths, regular files/directories, and no duplicate/traversing entries. Current documented limits are 100 MB compressed, 512 MiB extracted, 5,000 entries, and 100 MiB per member. Directory branding requires square logo/composer images. Local-MCP configuration is excluded unless the organization is approved; `.app.json` must not be fabricated or included as a workaround. [Submission validation reference](https://developers.openai.com/plugins/deploy/submission-errors).

Build a distributable for each qualified platform or a validated platform launcher. Include the matching binary and licenses; inspect linked dependencies and installation paths. Exclude connection files, sockets, local state, credentials, customer/market data, private research, build caches, evidence from unrelated work, and absolute developer-machine paths. Package only artifacts whose distribution rights and purpose are established.

## Outstanding release record

Maintain a concrete release record with these entries, each backed by a result or explicitly marked pending:

- Runtime build and focused protocol/process/IPC checks.
- Installed Symphony command and owner-authority integration checks.
- Clean Codex desktop installation and actual host workflow evidence.
- Claude/Antigravity checks before claiming either host is tested.
- Exact platform binaries, digests, dependencies, license inventory, and public artifact review.
- Publisher account verification and access; plugin-specific published privacy coverage, applicable service terms, review recording and setup. QuanuX, its homepage, public contact route/email, and exact square website branding are now recorded; their availability does not close those other gates.
- OpenAI local-MCP eligibility confirmation, portal validation, review status, and eventual publication decision.

No source package edit, local validation, or test report changes the external review status. Keep the product's Alpha limitations in the public description. This website-enriched draft preserves `0.1.0-alpha.1` and uses a distinct archive filename; coordinate version metadata if a later runtime release is prepared.
