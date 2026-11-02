# L21 security triage and extraction evidence

This is a dated review of completed analyses, not a security clearance. The [machine-readable snapshot](L21-SNAPSHOT.json) records the observed alert states, exact analysis IDs and source commits, category matrix, source hashes and raw evidence hashes. It was assembled on 2026-11-02 UTC from GitHub's read-only API and completed job logs. No alert was dismissed, severity changed, query suppressed, or scan cancelled.

## Findings and decisions

The current default-branch inventory contains **eight fixed alerts and one open alert**. A fixed alert's `most_recent_instance.commit_sha` identifies its last reported instance; it is not itself the fixing commit. Preserve `fixed_at` and source evidence separately.

| Alert | Observed state | Reviewed source boundary and decision |
| --- | --- | --- |
| [1](https://github.com/QuanuX/Symphony/security/code-scanning/1), [2](https://github.com/QuanuX/Symphony/security/code-scanning/2), [7](https://github.com/QuanuX/Symphony/security/code-scanning/7), [8](https://github.com/QuanuX/Symphony/security/code-scanning/8) | Fixed | Retain L02 ownership closure. The [authority validator](../../modules/stav-append-authority/internal/config/config.go) excludes the all-ones sentinel before mutation. The [directory helper](../../modules/stav-append-authority/internal/lifecycle/enrollment.go) repeats admission, checks representability before conversion to host `int`, and verifies actual UID/GID/type after `Chown`. [Focused tests](../../modules/stav-append-authority/internal/lifecycle/ownership_test.go) preserve zero, ordinary and maximum usable identities and reject no-op ownership mismatch. Machine intent/plan admission remains independently checked by [qxctl](../../tools/qxctl/internal/foundationlifecycle/client.go). |
| [3](https://github.com/QuanuX/Symphony/security/code-scanning/3) | Fixed | [Resident request expansion](../../tools/qxctl/internal/knowledgeengine/sbv_partitioned.go) allocates `len(child)` rather than adding an unnecessary capacity hint. The prior 1 MiB / 32,768-value admission limited reachability; no reachable allocation overflow was established under that contract. This is bounded-admission reasoning and focused hardening, not a claim that arbitrary Go allocations are safe. |
| [4](https://github.com/QuanuX/Symphony/security/code-scanning/4) | Fixed | The non-installed [feedback provider fixture](../../modules/sbv-engine/tests/feedback_fixture.cpp) now consumes a caller-owned private append descriptor, validates its kind/owner/mode/access flags, and never opens an environment-selected path. [Fixture tests](../../modules/sbv-engine/tests/feedback_settlement_test.cpp) cover malformed, closed, read-only and missing descriptors. The original finding was test-only; that scope did not excuse retaining the avoidable path opening. |
| [5](https://github.com/QuanuX/Symphony/security/code-scanning/5) | Fixed | The old shared-temporary unsafe root was replaced by a deliberately unsafe leaf under a private parent. This did not erase the negative test; the remaining literal is tracked separately as alert 9. |
| [6](https://github.com/QuanuX/Symphony/security/code-scanning/6) | Fixed | The [capture SDK consumer](../../modules/sqav-capture-cpp/tests/sdk-consumer/main.cpp) creates its synthetic export exclusively, without following symlinks, with mode 0600. Its checks retain exact bytes/mode under umask 000 and refusal of existing files and live/dangling symlinks. |
| [9](https://github.com/QuanuX/Symphony/security/code-scanning/9) | **Open**, high severity, classified by GitHub as test | Retain and review the intentional negative-permission fixture as described below. This review neither dismisses the alert nor claims it is fixed. |

The original L05 reasoning, regression target names and historical states remain in [L05-TRIAGE.md](L05-TRIAGE.md). They are not overwritten by this later snapshot.

### Separate decision for the intentional negative test

At reviewed source `33a69a74ca2f061627c841fe7f0e7c10f29219d2`, [lifecycle_test.cpp](../../modules/knowledge-session-coordinator/tests/lifecycle_test.cpp) creates the parent with `mkdtemp` (lines 55–64), then creates `unsafe-state` beneath that retained parent and visibly sets only that leaf to 0777 (lines 879–900). It asserts the parent is 0700, the leaf is actually 0777, the production owner returns `lifecycle_journal.state_directory_unsafe`, and the refused leaf remains empty. The [production checks](../../modules/knowledge-session-coordinator/src/lifecycle_journal.cpp) reject wrong ownership or group/other-write permission on state roots and child directories (lines 561–595).

Decision: **retain the explicit negative test and its literal permission**, with its private-parent assertions and production refusal. Another host user cannot traverse the 0700 parent to reach the leaf. This is not an installed runtime path or a permission relaxation. No production exploit was established from this fixture boundary. GitHub alert 9 remains open and visible; this assessment is not an alert dismissal, a blanket risk acceptance, or a waiver for another 0777 path. A moved fixture, changed parent ownership/mode, removed refusal assertion, or altered production admission invalidates this reasoning and requires a fresh review.

## Uploaded analysis binding

All six configured categories were present for each of the observed PRs. These uploads bind their **synthetic merge commits**, not merely similarly named branches or a later head. Exact category/ID/result metadata is in the snapshot.

| Ref | Head / analysis target |
| --- | --- |
| Main | Head `c15eda9b0092f123e86d4be577fe6d00ba68e1b1`; latest six uploads bind `692349ca4791f2f392b9b3ea72543061b6f4f19e`. The only intervening file is `knowledge/sclv/CHANGELOG.md`; there is no new exact-head upload claim. |
| PR 175 | Head `7ee16ad6e309c3fa3c9b34b6839337b1eba0b155`; merge `3c93ec43aac7b332cf85c0541cf08c3077b089d5` |
| PR 176 | Head `ded20079a8deb3966075415e0d7dfb184220214f`; merge `2f1be85ff80073f84fba1e7b9e7083f26d756168` |
| PR 177 | Head `3cdbdfbfb9feb03c83a8a48cf42b023ad2ae077e`; merge `2717d570f358272736d4aa210603ea8305b7c058` |
| PR 178 | Head `cc1cdf94f042b3c3813190c65045a4c8db144542`; merge `320edcc367389f039d490b661427b49d15fb07a7` |
| PR 179 | Head `33a69a74ca2f061627c841fe7f0e7c10f29219d2`; merge `57c23ccd452d794471ed96266a9a9bc9b5b1de7e` |

PR 179's [CodeQL run 41351436975](https://github.com/QuanuX/Symphony/actions/runs/41351436975) completed successfully in all six jobs. The PR SARIF includes `codeql-action/pr-diff-range`; its zero result counts cannot establish closure of unchanged default-branch alert 9. Main's C++ analyses report one result in each operating-system category. API `error` and `warning` fields were empty; these fields are not a complete extraction-diagnostic inventory.

## Measured coverage and remaining limits

The completed PR 179 [Linux C++ job 124982267235](https://github.com/QuanuX/Symphony/actions/runs/41351436975/job/124982267235) and [macOS Intel C++ job 124982267443](https://github.com/QuanuX/Symphony/actions/runs/41351436975/job/124982267443) each reported **575 of 588 C++ files and 8 of 8 C files**. Both used CodeQL 2.27.3 / `codeql/cpp-queries` 1.9.2 and source-only extraction. Thirteen C++ files were outside each reported scanned count. The downloaded public SARIF and job summary do not name those files, so this review does not invent their identities or reasons. The logs explicitly reserve detailed file coverage for default/protected branches.

Source-only extraction infers flags and include paths; dependency-heavy builds and missing macro/header context can reduce accuracy. It does not reproduce the installed SDK matrix. See [GitHub's compiled-language build guidance](https://docs.github.com/en/code-security/reference/code-scanning/codeql/build-options-for-compiled-languages).

The [published CodeQL language support matrix](https://codeql.github.com/docs/codeql-overview/supported-languages-and-frameworks/) checked for this review still lists C++23 as beta and does not list C++26; C++20 modules are unsupported. Changing Symphony's standard or assuming a traced build removes this limitation is not authorized here. The two native analysis runners are amd64; required Apple Silicon/Ubuntu ARM runtime qualification is separate from CodeQL extraction. Static analysis cannot establish absence of vulnerabilities.

## Durable collection and next check

The workflow now retains original analyzer SARIF, the action's post-processed upload SARIF, and a source/category-bound summary per job for 14 days. This uses the pinned action's documented [`output` and `post-processed-sarif-path` inputs](https://github.com/github/codeql-action/blob/9b3254085494b55a0f7c4c2b3d2d52cc712ce370/analyze/action.yml). The [collector](collect_evidence.py) keeps diagnostic notifications and metric properties without treating absent values or zero findings as full coverage. Missing original SARIF or contradictory source provenance fails collection. Raw SARIF remains available to inspect diagnostics not modeled by the summary. Preserve the selected release evidence outside the expiring Actions artifact before its 14-day retention ends; the committed snapshot keeps IDs and content hashes, not a permanent copy of every raw scan.

After the next completed default/protected-branch analysis, inspect those retained originals and the tool status page to identify the 13 uncounted files and decide whether each is an intentional exclusion, unavailable dependency/generated source, or an extractor limitation. Recheck alerts and all six categories at the final source commit. The new evidence-retention steps and process-helper correction need fresh hosted results; this change does not wait for or assert them.

Read-only refresh examples (preserve complete pagination and retrieval time):

```sh
gh api --paginate --slurp 'repos/QuanuX/Symphony/code-scanning/alerts?per_page=100'
gh api --paginate --slurp 'repos/QuanuX/Symphony/code-scanning/analyses?per_page=100'
gh api --paginate --slurp 'repos/QuanuX/Symphony/code-scanning/alerts/9/instances?per_page=100'
```

Match each current PR head to its current merge SHA, then match all six analysis categories to that SHA. Do not combine stale categories from different commits into an apparently complete matrix. Preserve unchanged open alerts even when PR-diff results are empty. No command in this procedure changes alert disposition.
