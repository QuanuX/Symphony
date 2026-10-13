# SOV remote handoff planning v1

## Status and ownership

R6.2b1 implements a read-only local contract through `qxctl sov remote
plan|assess|schema|template`. SOV owns the handoff meaning; qxctl owns its Go
presentation. SBV continues to own calculation admission, cancellation, journal
state and results. This contract is a preparation increment under Duncan's
authorization to proceed locally while no remote host is available.

No adapter, remote receiver, provisioning, resource reservation, authentication,
SSIAG grant, transfer or remote execution is implemented by these commands.
The remote-operation arrangements in `REMOTE-QXCTL.md` remain architectural
possibilities. An output plan is not an executable authorization or evidence of
an installed transport. Future independently installed adapters must close their
own exact executable contracts; they are not absorbed into qxctl.

## Commands and portable results

`qxctl sov remote template --operation plan --json` supplies a synthetic example.
Extract its `template` member, select all fields, then pass the file to
`qxctl sov remote plan --input <file> --json`. Schema discovery uses the same
`--operation plan|assess` selection. The schema response contains request and
result shape fragments, with required closed members derived from the actual Go
types. Cross-field checks below and SBV's native observation contract additionally
apply. Examples contain zero digests and placeholder identities; acceptance as a
plan proves neither that a target exists nor that those identities are correct.

Default text enumerates every field and escapes terminal controls. `--json` or
`--format json` emits one complete portable document, without banners. A GUI or
external agent consumes exactly that document. The SBV-specific NDJSON stream
protocol is not reused for these bounded SOV documents. Invalid input returns an
error with no input contents in diagnostics and no dispatched operation.

All fields, including nulls and empty collections, are explicit and required.
Unknown or duplicate members, case aliases, non-UTF-8 input and trailing values
are rejected. The shared qxctl control-document parser bounds document size,
depth, strings and value count; this contract's document bound is 8 MiB. These are
administrative message bounds, not limits on datasets, RAM or calculation time.

## Selection and plan identity

The v1 selection binds:

| Field | Meaning |
|---|---|
| `operation_id` | Caller-selected stable orchestration identity; this planner stores no uniqueness registry. |
| `target` | Exact selected TOPS, Node, Habitat references and identity evidence digest. No discovery, SNV resolution or implicit `::` grammar. |
| `transport` | Controller, target or staged arrangement; IPC or scoped CLI family; exact adapter ID/version/digest and opaque endpoint reference. No URI or shell interpolation. |
| `runtime` | Exact qxctl version/executable digest and SBV version/installation digest. This profile supports SBV 0.33.0-dev job observations; no newest-version substitution. |
| `job` | Canonical absolute target POSIX journal directory, native job ID and immutable native plan digest. The path is not opened or resolved on the controller. |
| `resources` | Requested threads, memory bytes, device references, absolute deadline and optional resource-observation digest. No reservation or applied-resource claim. |
| `authority` | Selected operation/resource/audience/scope and optional policy digest that a future target admission must verify. No subject assertion, credential or transferable capability. |
| `extensions` | Caller string metadata retained in the plan; it is not interpreted as executable controls. |

Resource quantities are arbitrary-length canonical positive decimal strings;
null means no user constraint. No calculation deadline, RAM limit or dataset-size
ceiling is imposed. The selected worker must later report actual requested,
granted and applied resources and any inability to honor a choice. These requests
do not silently rewrite resource controls already sealed in the native job plan.
Only safe references belong in the document; all supplied metadata is returned
to its caller and must not contain credentials or secret payloads.

Plan identity is SHA-256 of the UTF-8 bytes
`symphony.sov.remote-plan.v1` followed by a NUL byte followed by Go
`encoding/json.Marshal` of the typed selection normalized to JSON objects (all
object keys sorted; default HTML/Unicode separator escaping; no trailing newline).
Input whitespace and object order are immaterial. Array order, every selection,
explicit null and extension content are bound. This is a versioned Go encoding,
not RFC 8785. Consumers may preserve the full document and ask `plan` to recompute
identity rather than reproduce its encoding. Output formatting and object order
do not alter this identity.

Plans report `planned_not_dispatched`; `remote_operational`,
`authority_verified`, `resources_reserved`, `automatic_retry` and
`checkpoint_resume_supported` are always false. The requirements array preserves
the remaining target, authority, resource and access obligations.

## Supplied-observation assessment

`assess` accepts the original selection, expected SOV plan digest, a transport
outcome (`not_dispatched`, `response_lost`, `response_received`), and either null
observation/target digest or a supplied exact `symphony.sbv.job-inspect.v1`
observation plus the selected target digest. `not_dispatched` cannot contain an
observation; `response_received` must contain one. Other operation responses must
first be followed by a fresh native inspect, not relabeled as inspect.

SBV's existing validator checks its full status shape, engine version, job ID,
directory, plan digest, generation, phase and journal reference bindings. The
assessment returns the complete observation without dropping fields. It does not
read journal files, rehash results, authenticate an endpoint, discover current
state, compare a sequence of generations or verify freshness. A supplied target
digest is an identity comparison, not proof of origin. Every assessment therefore
reports `supplied_unverified` and `authority_verified: false`. It never confers
dispatch/retry permission.

| Supplied situation | Recommended next observation/action |
|---|---|
| Not dispatched, no observation | Qualify target and authority. |
| Response lost, with or without an earlier valid observation | Inspect the same job. Earlier state cannot settle a later ambiguous dispatch. |
| Execution lock reported held | Observe the same job; a phase label alone does not prove worker exit. |
| Completed/partial, lock not held, native outcome reference present | Inspect the bound result and its exact availability/completeness. |
| Queued, no claim, no cancellation | Qualify target and authority; never automatically resubmit. |
| Durable claim before state replacement, lost worker, unknown, failed or cancelled | Reconcile the same job using its native generation-checked operation. |

Automatic retry is always false. Cancellation must use the separate native
generation/request-ID/reason contract. Closing a transport does not prove
cancellation or termination. A target's irreversible side effect cannot be
rolled back by the controller. Native checkpoint/resume is not supported in this
selected profile; an unknown admitted job is not silently executed again.

## Remaining transport qualification

R6.2b2 must implement an independently installed selected SOV adapter/target
dispatch boundary. Target-local SSIAG endpoint authentication, non-transferable
operation-bound decisions, current policy and safe STAV audit requirements must
be enforced there where protected authority applies. A controller's local SSIAG
decision cannot be replayed as remote authority. SSH, a permanent service, or a
general remote shell is not selected by this planner.

Qualification must bind exact versions/identities and exercise ambiguity before
and after admission, worker/controller loss, cancellation versus termination,
stale observations/cursors, result access/transfer identity, permission expiry and
revocation, resource refusal, and repeated reconciliation without duplicate
effects. Existing local filesystem jobs do not prove shared-storage or cluster
locking semantics. No host is selected at this increment; full R6.2b and M3 remain
open.
