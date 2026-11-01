# L05 focused alert triage

At the read-only GitHub inventory on 2026-11-01 at 08:43 UTC, alerts
[1](https://github.com/QuanuX/Symphony/security/code-scanning/1),
[2](https://github.com/QuanuX/Symphony/security/code-scanning/2),
[7](https://github.com/QuanuX/Symphony/security/code-scanning/7), and
[8](https://github.com/QuanuX/Symphony/security/code-scanning/8) were fixed;
alerts 3–6 remained open. No alerts were dismissed. The four open instances
referred to main commit `b53f3c075dd1e2e013ea73583e1eb1500750570f`; source review
started from `4499b25170fc958d9f15f2c8d9446878060d3b78`.
This is a dated triage record, not a claim about a later scan's state.

| Alert / rule | Source scope and boundary | Scoped change |
| --- | --- | --- |
| [3](https://github.com/QuanuX/Symphony/security/code-scanning/3), `go/allocation-size-overflow` | [`sbvExpandedResidentRequest`](../../tools/qxctl/internal/knowledgeengine/sbv_partitioned.go) adds one parent output to a copied child map. Production entry paths first admit at most 1 MiB of JSON and 32,768 keys/values; no reachable integer overflow was established under those bounds. | Remove the unnecessary `+1` capacity hint. Retain the fresh map, exact output selection and existing admission bounds. |
| [4](https://github.com/QuanuX/Symphony/security/code-scanning/4), `cpp/path-injection` | [`feedback_fixture.cpp`](../../modules/sbv-engine/tests/feedback_fixture.cpp) is an independently linked, non-installed `BUILD_TESTING` provider fixture. Its environment-selected lifetime log was controlled by the same test caller; no production privilege boundary was established. | The test opens an exclusive, no-follow, mode-0600 log in its private temporary directory and passes a descriptor. The fixture strictly parses the descriptor, rejects standard/malformed/closed/read-only descriptors, verifies a private regular file owned by the caller with append/write flags, and writes without opening a pathname. |
| [5](https://github.com/QuanuX/Symphony/security/code-scanning/5), `cpp/world-writable-file-creation` | [`lifecycle_test.cpp`](../../modules/knowledge-session-coordinator/tests/lifecycle_test.cpp) deliberately creates mode 0777 to verify production refusal. Previously the world-writable fixture itself was a top-level shared-temporary directory. Production already rejects group/other writable state roots. | Nest the deliberately unsafe leaf under a retained mode-0700 parent. Assert both modes, preserve `lifecycle_journal.state_directory_unsafe`, and verify refusal leaves the leaf empty. |
| [6](https://github.com/QuanuX/Symphony/security/code-scanning/6), `cpp/world-writable-file-creation` | The non-installed [`capture SDK consumer`](../../modules/sqav-capture-cpp/tests/sdk-consumer/main.cpp) optionally exports synthetic fixture bytes using `fopen(wb)`, whose creation permissions depend on umask and which can clobber an existing path. | Use exclusive, no-follow, close-on-exec creation with mode 0600. Existing files and symlinks are refused without modifying their targets. Check writes and close. |

The initial focused Go verification used Go 1.26.5, `GOWORK=off`,
`GOTOOLCHAIN=local`, a fresh build cache and `-mod=readonly -race -count=1`.
Three top-level tests and ten subtests passed with zero skips or failures:
`TestSBVPartitionedResidentRecoveryIdentities`,
`TestSBVExpandedResidentOutputSelection`, and
`TestValidateJSONObjectHonorsSharedValueBoundary`.
These cover resident identity, both output forms, ambiguous selection, map
isolation, and the exact shared JSON value-count boundary.

Native regression coverage is attached to existing targets:

- `sbv-feedback-settlement-contracts` retains lifetime sequence, cancellation,
  and bad-ABI checks and adds malformed, read-only, closed and missing log
  descriptor cases. Its target is `sbv-feedback-settlement-tests`, with both
  feedback fixture DSOs as dependencies.
- `knowledge-lifecycle-planner-tests` checks the private parent, unsafe leaf,
  exact refusal code and absence of journal writes.
- The independently copied capture SDK consumer now runs three groups. Its
  secure-export group checks correct bytes, mode 0600 under umask 000, and
  refusal of existing files plus live and dangling symlinks.

These native cases must be compiled and executed by the installed-dependency
qualification gate; source inspection or syntax-only compilation does not
establish their runtime results. Consult the specific workflow run and platform
artifacts for that evidence.

Alert 5's literal `chmod(..., 0777)` remains intentionally visible. The private
parent removes the shared-temp exposure while preserving the negative
admission case. A query may still flag the deliberate leaf permission;
retaining that test is not an instruction to suppress or dismiss its alert.
Any subsequent disposition requires a separate scoped review of this boundary.
Fresh uploaded analyses, including the default branch after merge, must be
checked before reporting closure. No rule suppression, query obfuscation or
alert dismissal is part of this change.
