# MVP finish-line gates

This is the working release checklist after L27–L30 integration, not publication approval. The [30-task register](launch-tasks.json) keeps the original completion criteria. The [candidate acceptance plan](../tests/launch-acceptance/plan.json) supplies required profiles and scenarios. No additional numbered sprint is implied.

## Public beta feature freeze — November 12, 2026

The frozen feature baseline is main `51d4806a60bceca1f47513f90206155d9775d510`.
New product/provider features are deferred. Existing-behavior stabilization fixes,
focused regression and qualification, credential-status documentation, packaging
and release preparation may continue. The broader gates below remain an inventory
of unresolved work, not authorization to expand the frozen feature set. A retained
promise that needs missing implementation requires an explicit scope/freeze
decision; the freeze neither completes that work nor silently removes the promise.
This records project direction, not enforced GitHub branch rules or release approval.

The [source-bound Public Beta dossier](../tests/public-beta/README.md) prepares
W01/W06 against the PR #196 main revision and supplies the full W01–W12 plan for
manual review. Its proposed selections, pending trust/owner inputs and acceptance
mapping do not authorize new features or establish final-candidate acceptance.

## Verified source baseline

PRs [#185](https://github.com/QuanuX/Symphony/pull/185) and [#186](https://github.com/QuanuX/Symphony/pull/186) subsequently passed all eight workflows and 48 checks each. They merged as `4e697a80950b5a9146f29ec80f061eee5b3ba811` and `cbb201920bed3447502bf72254602bf6ac5978cf`; each merge retained its tested source tree. The earlier L27–L30 baseline is retained below.

[PR #184](https://github.com/QuanuX/Symphony/pull/184), source `1084e155d3557bb5e56c619c5cea33874b5c9f29`, merged as `15bd46c4da63fcad96ada9cf2f3d8941b109fe5f`. Its merge tree equals the tested source tree, `8a2958b6680b33bd7726019959df5f9634ad274f`. All eight workflows completed successfully:

| Workflow | Exact-source run |
| --- | --- |
| MVP dependency map / installed SDKs | [41599795910](https://github.com/QuanuX/Symphony/actions/runs/41599795910) |
| SQV retained inspection | [41599795992](https://github.com/QuanuX/Symphony/actions/runs/41599795992) |
| Release functional qualification | [41599796017](https://github.com/QuanuX/Symphony/actions/runs/41599796017) |
| CodeQL | [41599795953](https://github.com/QuanuX/Symphony/actions/runs/41599795953) |
| Package candidate and lifecycle qualification | [41599795932](https://github.com/QuanuX/Symphony/actions/runs/41599795932) |
| qxctl standalone build | [41599795960](https://github.com/QuanuX/Symphony/actions/runs/41599795960) |
| Ownership regression | [41599795968](https://github.com/QuanuX/Symphony/actions/runs/41599795968) |
| Native DuckDB connectors | [41599795933](https://github.com/QuanuX/Symphony/actions/runs/41599795933) |

These results qualify the selected source paths and declared fixtures on their recorded profiles. They do not qualify a later source revision automatically, establish live provider success, or turn test signing keys into publisher authority. New CI runs may be reviewed later; a pending run is never recorded as a pass.

## Packaging and post-packaging qualification

The user has assigned live-provider qualification to users **after packaging**. Preparing and verifying a candidate therefore does not require live test profiles or completed live-provider results. Before offering it for testing, record the shipped scope, implementation limits, build/package evidence, runtime/notice review and selected trust policy. Users then exercise supported live operations with their own authorized profiles and return redacted evidence. Unimplemented native credential/provider paths remain unavailable and must be labelled; this sequencing decision does not mark them implemented.

The responsibilities below cover both candidate preparation and subsequent user qualification; they are not a requirement to finish live testing before building packages.

## Remaining responsibilities

| Gate | Concrete work and evidence | Current state / owner |
| --- | --- | --- |
| **1. Select the exact MVP scope** | Select shipped components, independent versions, supported operations and required profiles from the current dependency map. Record optional components, proposal-only exclusions and every promised credential/provider operation. Bind the selection to a source commit/tree and map digest. | Selection/approval pending. @quantDIY is release approver; module maintainers remain unassigned. |
| **2. Complete native credential integration** | For the general L14 paths, wire permission-backed lease lifecycle/audit into the durable admission snapshot; implement native recipient/provider admission and protected delivery; finish Apple data-protection Keychain and Linux Secret Service operations; connect native consumers to their admitted owner. | General L14 **implementation remains**, not just unavailable credentials; see [CREDENTIAL-QUALIFICATION.md](../modules/secure-identity-access-governance/CREDENTIAL-QUALIFICATION.md). The separate [Cloudflare private-store route](../modules/ssiag-provider-private-store/SPEC.md) is implemented and locally fixture-qualified; its actual deployment/live qualification remains open and does not close these general paths. |
| **3. Qualify promised provider operations** | Use authorized profiles to prove bounded success, refusal, expiry/revocation, cancellation, disconnect, uncertain outcome and reconciliation on each selected target. Complete the concrete TWS vendor bridge and SDK 10.45 provenance/conformance. News remains vendor-neutral ingress with authorized content/correction retention. | User-led after packaging. Profiles, authorized content and TWS SDK/provenance remain unavailable here. Fixture passes do not establish live success; missing implementation remains explicitly unavailable. |
| **4. Close the actual artifact payload** | Select binary/SDK/optional-runtime contents; resolve recursive loader/TLS dependencies and exact notices/source/build material; review optional PDFium, DuckDB and vendor material for the chosen payload. Retain publisher decision with exact artifact identities. | [Collector](../tests/dependency-notices/README.md) and [publisher worksheet](../cmake/notices/PUBLISHER-REVIEW.md) prepared; final payload and publisher review pending. |
| **5. Establish candidate identity and trust** | Build selected immutable packages; bind source/recipe/receipt/file hashes, target profile and independent signer policy; retain exact candidate URL/digest. Exercise wrong-signer, tamper and profile refusals. | [Candidate tooling](../tests/package-release/README.md) prepared. Production signer policy, candidate URLs and publisher authorization absent. Anaconda/mirror setup remains later distribution work. |
| **6. Run final-download acceptance** | Use fresh hosts and the exact approved downloads for every profile/scenario in the [acceptance plan](../tests/launch-acceptance/plan.json). Prove loader closure without build prefixes, bounded research, knowledge workflows, protected administration, all promised providers, recovery, upgrade/rollback and data-preserving removal. | Run after verified candidate packages are available; users supply the live-provider results. Packaging itself need not wait for those results. Source/cache runs cannot replace final-download evidence. Ubuntu ARM64 suffices for this sprint; physical Spark/DGX OS/GPU claims stay excluded. |
| **7. Close operational ownership** | Name a backup, module maintainers, release milestone and acceptance reviewer. Verify support-ticket privacy, routing/access and response ownership through an authorized test. Record supported account model and retention/recovery limits. | @quantDIY owns approval and rollback. [SECURITY.md](../SECURITY.md) selects support.quanux.org tickets; operational intake verification and other assignments remain pending. No ticket is sent by this preparation. |
| **8. Approve publication** | Review exact capability selection, artifact trust, runtime/notices, all required target/scenario results, provider evidence and rollback plan. Record an explicit publisher decision. | Not authorized by source checks or this checklist. |

Packaging/trust and operational ownership work can proceed alongside implementation. Live acceptance follows packaging and needs the real candidate and actual supported operations. Do not silently reduce the promised provider scope to declare the MVP complete. Distinct-account administration, atomic authorization-to-effect, live database backup, power-loss durability and cross-version migration remain limited to evidence explicitly established in the [operator runbooks](../tests/launch-operations/README.md).

## Prepare an exact-source scope review

From a clean, committed checkout, with CMake 3.30+ and Git, select a new output file in an existing private directory **outside** the checkout:

```sh
umask 077
review=$(mktemp -d "${TMPDIR:-/tmp}/symphony-release-review.XXXXXX")
cmake -DRELEASE_REVIEW_OUTPUT="$review/review.json" \
  -P tests/launch-acceptance/prepare-review.cmake
```

The read-only helper verifies the dependency and test maps, binds the current Git commit/tree and map digest, and copies the current component roots, native versions/dependencies, Go identities, Swift provider and platform catalog into one review packet. It includes the required acceptance inputs/scenarios and leaves every approval/input unfilled. It refuses a dirty checkout, an existing output, or an output inside the repository. Use a private, quiescent checkout/output directory; this is not a hostile-filesystem or concurrent-writer boundary.

The packet is an inventory for **selection**, not an instruction to ship every root. Build support, proposal-only modules and platform-specific components remain distinguishable. It does not build packages, download artifacts, access credentials, test a provider, sign anything, select a publisher, or infer completion. Complete and review the actual capability manifest separately; the helper's `release_acceptance_complete=false` and `publication_authorized=false` are intentional.

## Current implementation increment

The durable SSIAG admission snapshot now preserves exact-owner revisions and grants across restart and fences uncertain persistence. Permission-backed issuance/revocation and native delivery for the general L14 dispatcher remain separate work; the later Cloudflare-specific native route does not close those gaps. The [candidate-host helper](../tests/launch-acceptance/README.md#verified-extraction-on-the-selected-host) now verifies and bounds extraction before receipt re-verification; its report explicitly leaves fresh-host/provider/release acceptance false. Neither increment supplies production signing/access-group policy, provider profiles, licensed SDK material or final candidate approvals.

With live inputs still unavailable, the [runnable evidence gates](../tests/launch-acceptance/README.md#runnable-evidence-gates-while-live-inputs-are-unavailable) prepare the full target/scenario packet and return an explicit incomplete result. They reject mismatched or tampered attachments and exclude fixture, failed or skipped outcomes from structural completeness. Even a complete packet requires independent review and never authorizes publication automatically.
