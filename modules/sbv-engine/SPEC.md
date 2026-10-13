# SBV engine specification

## Version and ownership

Contract v1; package `sbv-engine` 0.32.0-dev; engine `symphony-sbv`; vector `sbv`. C++26 owns computations, result sealing and artifact queries. Go qxctl verifies the exact installation and projects native evidence. OWNER-INTERFACE.json is the exact operation declaration; generated C++/Go metadata must match it. This release is experimental.

The engine uses knowledge-vector-engine-cpp 0.2.0-dev and sqav-databento-dbn-cpp 0.5.0-dev, with that adapter's exact static dependencies. It performs no data-provider network calls. Signal/replay semantics belong to SBV; acquisition remains SQAV, metadata SQMV, batch ownership SQFV, transformation SQTV, physical stores SQPV and recipient delivery SQDV. Its portable local JSON result is a derived artifact, not a new market-data persistence service. An optional offline retained-source route composes SQAV capture 0.2.0-dev, SQMV 0.2.0-dev, SQFV 0.3.0-dev, SQPV 0.2.0-dev and SQDV 0.3.0-dev under the selected profile described below.

## Current operation/profile matrix (0.25)

`qxctl sbv capabilities` and the identical native/SDK operation expose `operation_profiles`: independent read/write support, working-memory scope and retained domain/format limits for every declared operation. Nullable dataset/resource/deadline choices belong to the user. No worker, job, aggregate dataset or partitioned-result policy ceiling is imposed. Host allocation, exact arithmetic and explicitly selected formats remain finite. The matrix below supersedes historical incremental scope wording in this document; it does not confer a cluster/GPU runtime or a competitive speed claim.

| Operations / profile | Reads | Writes | Working state and limits |
| --- | --- | --- | --- |
| `run`, `generate_census` / source_execution | file / optional exact retained-source / resident decoded DBN | legacy or partitioned, independently selected | decoded events, census/ID and trade indexes, replay intervals, provider state; per-worker readers/spools. No aggregate dataset/census or worker policy maximum; caller limits, host/uint64 and selected format/model domains apply. |
| `evaluate` / legacy_evaluation | inline census or legacy complete-result reference | legacy only in this profile | whole legacy artifact and model results. 4096 external census rows; legacy artifact/control framing. Select partitioned output for bundle census/outcomes and uncapped aggregate census. |
| `evaluate` / partitioned_evaluation | inline, legacy or bundle census; inline or bundle external outcomes | partitioned | trade/ID indexes, provider instances, worker readers/spools. No aggregate census maximum; each model retains its named support/numeric domain. |
| `economics` / legacy_economics | legacy complete-result selection | legacy | whole artifact/results. 4096 signals/selections; legacy artifact framing. Select partitioned output for scalable instructions and bundle sources. |
| `economics` / partitioned_economics | legacy or bundle result; inline/apply/referenced selections | partitioned | census validation/ID indexes, one selected model/economic row plus reader cache. No aggregate signal/selection maximum; typed model support and exact numeric domains apply. |
| `compose_economics` / economic_composition | legacy or bundle economic result selections | legacy or partitioned independently | selected supports, exact terminal/prefix maps and selected joint paths. Caller nullable max_paths/max_terminal_atoms; exact rational overflow rejects. Inline component/joint instructions use control framing. |
| `book`, `liquidity`, `allocation_economics` / execution_chain | legacy or bundle result; inline or referenced signal/order/valuation choices | legacy or partitioned independently | book order/level state and window indexes; liquidity frame map and typed intents; allocation lineage/indexes. No aggregate census/frame/order/checkpoint maximum. Caller nullable frame maximum. Named book depth 1..64; displayed-depth quantity 1..1e9 and at most 64 fills. |
| `analyze`, `resample`, `split` / research_rows | legacy or bundle row selection | legacy or partitioned independently | typed observations, exact sorts, bootstrap values, split intervals/ID indexes and plans. No aggregate observation/replicate/draw/fold/retained-row policy maximum. Exact arithmetic, caller parameters and host memory apply; no automatic spill. |
| `fit`, `predict` / exact_linear_models | legacy or bundle rows and model selections | legacy or partitioned independently | dense rows times features plus features squared; exact rational solver and ID sets. No row/feature policy maximum; numeric overflow/rank rules explicit, no automatic approximation/spill. Feature descriptors use control framing. |
| `compare`, `research_history` / research_ledger | inline or referenced candidates/objectives/history; legacy or bundle results | legacy or partitioned independently | candidate/objective vectors and worst-case quadratic dominance graph; observation occurrence index. No aggregate candidate/objective/source-byte/history maximum; typed IDs/units/pointers and exact numeric domains remain. |
| `experiment` / local_experiment | inline trials or referenced bundle table copied to caller-selected owned plan store | legacy or partitioned aggregate independently | per-trial control requests, receipts, dependency graph and stable ready waves. No trial/worker policy maximum; optional worker budget. Inline plan and each child request/claim/receipt use bounded control/legacy journal formats. Local execution only. |
| `result_select` / legacy_direct_selection | legacy array | control page | whole artifact, selected row indexes and sort. 65536 source rows; 16 filters; 8 sort keys; 64 columns; page 1..256. Use saved_view profile for larger aggregate views. |
| `result_select` / saved_view | legacy or bundle array for prepare; immutable bundle view for page | partitioned prepare; control page | ordered prepare uses caller-selected memory_index, optional max_index_rows; page reads selected range. No aggregate row/filter/order/column policy maximum beyond request framing; pointers <=4096 bytes. Page <=512000 bytes /16000 values; oversized rows remain accessible by bundle query/export. |
| `compose` / legacy_direct_probability_composition | inline support/measure | legacy | exact enumerated paths/distributions. support <=64; steps 1..16; enumerated paths <=65536; exact int64 rationals. Broader economic composition is a separate selectable operation. |
| `compose_joint` / legacy_direct_joint_composition | inline joint paths | legacy | exact supplied paths/distributions. paths 1..4096; steps 1..16; exact int64 rationals. These are named legacy operation profiles, not limits on compose_economics. |
| `backend_plan` / capability_only_backend_tensor | inline explicit declarations | control result | descriptor metadata only. tensor rank <=16, nonnegative strides, exact dtype/extent/ready state. CPU worker request has no policy maximum; CUDA/tensor runtime unavailable, no silent fallback or reservation. |
| `live_plan` / disconnected_live_fixture | inline synthetic fixture; no live connection | control result | selected ring and pending horizons. event_cap 1..65536; pending_cap 1..128; fixture <=512 events and <=128 signals; event bytes <=1e9; UINT64_MAX time sentinel excluded. Wider/live runtime deferred. |
| `dataset_load`, `dataset_inspect`, `dataset_execute`, `dataset_release`, `dataset_jobs`, `dataset_cancel`, `dataset_configure`, `dataset_stop` / resident_dataset | file or optional exact retained-source; local handle | control lifecycle receipts; execute child legacy or partitioned | original load bytes plus decoded allocation; admitted jobs and worker state. No dataset/job/worker policy maximum; nullable caller budgets and deadlines. Load-buffer budget is not total RSS; reject excess admission without implicit queue. Execute: run/generate_census/evaluate/book. |
| `source_retain`, `source_export` / sqv-local-retained-capture.v1 | selected SQAV/SQMV/SQFV/SQPV/SQDV owner route | legacy source receipts/original byte export | owner capture/batch/store/delivery state, separately scoped from SBV load buffer. Exact source_owner_profile publishes owner limits, including 64 MiB DBN/capture and 1048576 records; ordinary file/RAM route has no such policy maximum. |
| `bundle_import`, `bundle_inspect`, `bundle_query`, `bundle_verify`, `bundle_export` / partitioned_result_v1 | canonical JSON import or pinned bundle | partitioned import; bounded query/receipt; streaming JSON/node NDJSON export | page readers/cache and bounded traversal; caller-selected page size/fanout. No aggregate bytes/nodes maximum; depth <=64, string/key <=65536 UTF-8 bytes; checked uint64/offset/hash representation. page_bytes>=512/fanout>=2; scalar must fit selected layout. |
| `result_inspect`, `result_query` / legacy_result_access | legacy artifact | control receipt/page | whole bounded legacy artifact. 128 MiB and 4000000 JSON values; depth64/string65536. Bundle operations provide scalable independent access. |
| `capabilities`, `catalogue`, `provider_inspect` / control_discovery | bounded request and optional exact provider selection | control result | catalogue or selected provider metadata. Process request 1 MiB / response 4 MiB; depth64/32768 values/string65536. Provider code is caller-selected native execution; no sandbox or guaranteed preemption. |

## Native partitioned execution in 0.23

`run`, `generate_census`, `evaluate`, `economics` and `compose_economics` accept an explicit `output` instead of `output_path`: `{kind:"partitioned",bundle_path,workspace_path,write_options:{page_bytes,index_fanout}}`. The final bundle and private scratch workspace must be fresh, disjoint caller-selected paths. The engine retains scratch on success and failure; it never automatically retries, resumes, rolls back or deletes it. `dataset_execute` supports the same output choice for resident run, generation and evaluation. The child omits both output forms. The wrapper binds its own request digest and preserves a failed child's original receipt.

The existing operation protocol returns a compact result receipt containing the result digest, summary and exact `storage.reference`. Use that reference with bundle inspect/query/verify/export to view every field in qxctl or the native SDK. `qxctl sbv template --operation OP --variant partitioned` selects the complete alternative template; omission preserves the existing default. A domain `recovery_required` is not a successful backtest even when SDK transport succeeded. Its bounded `cause` classifies the failure and `recovery` identifies the workspace/publication phase and any final writer evidence. The small control receipt is preflighted before final publication; large financial arrays are never embedded in it.

Built-in `criteria.max_signals` is null for no selection cap or a caller-selected positive uint64. This applies to both output forms. Capped spacing and additional-eligible diagnostics preserve the last admitted anchor; an uncapped census reports zero candidates excluded by a cap. Native run identity remains the canonical signal-array hash, imported external identity remains the complete original declaration hash, and native provider identity remains the complete declaration including completion evidence. Changing model, replay or storage representation does not regenerate census membership or silently rebase that identity.

Partitioned evaluation accepts the existing inline declaration and legacy complete-result reference, or a bundle selection `{kind:"bundle",reference,selector,read_options}` selecting a complete run/generation/evaluation result or external declaration. The selector is a JSON pointer or stable node ID. Full outer logical closure is verified before financial admission; the captured reference distinguishes outer, selected value and selected result hashes. The external-outcomes model may supply `outcomes_source` instead of inline `outcomes`, selecting a typed array in a bundle. All identities/domains are admitted before model evaluation. Unordered supplied rows join by exact signal ID. Descendants retain input arrays and their captured correspondence without reopening deleted ancestors; deterministic supplied support/likelihood must agree with saved external model outcomes. Native and observed-model facts are not recomputed merely from an attribution string.

Workers receive fixed contiguous census ranges, independent reader caches and private row spools; outputs merge in census order. Native providers retain their selected serialized/per-worker/shared-reentrant topology. Stateful provider behavior may depend on that topology and partition assignment; output-order determinism is not a claim of worker-count-independent provider state. Replay windows retain each signal's independently selected before/after boundaries. Sorted overlapping ordinal intervals share observed event storage without dropping per-signal window metadata or relabeling observed events as executions.

Partitioned economics accepts explicit four-field per-signal selections or `{kind:"apply",signals,economics}`. Signals may select all, an ordinal range with optional count, inline IDs, or referenced IDs; IDs declare supplied or census order. `referenced_rows` selects complete heterogeneous economic instructions. Zero selections and zero studies are valid. Expanded selections and source arrays are retained and hashed. Exact existing rational economic arithmetic is shared with the file route, including separate availability and caller interpretation policies.

Composition still requires explicit component order, source outcome pointer, conditioning, dependence, conversion, path retention, studies and nullable caller limits. Bundle sources and current results exported/embedded in legacy JSON preserve the same financial interpretation. Original census, full source context and replay are copied into partitioned composition descendants. Exact path enumeration and terminal/prefix rational maps are unchanged; paths and distribution arrays are written incrementally. No automatic independence, normalization, rounding, pruning, account-feasibility assertion or approximation is introduced.

These routes impose no aggregate census/result byte or node policy quota. Process requests/receipts, typed row domains, portable depth/scalar representation, checked arithmetic and selected page layout remain explicit contracts. Dataset residency, observed-trade vectors, exact ID indices, disjoint replay intervals, selected composition supports and exact terminal/prefix maps consume memory proportional to their content. Row spools require scratch disk and extra verification/copy passes. This is not a constant-memory, terabyte-scale, CUDA-runtime or fastest-engine claim. The 0.24 resource and consumer migrations are described in the current matrix; the legacy output route retains its named artifact profile.

## Partitioned result storage and access in 0.22

`bundle_import`, `bundle_inspect`, `bundle_query`, `bundle_verify` and `bundle_export` expose an explicit `symphony.sbv.partitioned-result.v1` format through the native process/SDK and `qxctl sbv bundle`. The source reference pins both exact manifest bytes and the canonical logical result body. Its self-contained hierarchy has content-addressed `symphony.sbv.result-page.v1` pages and bounded index fanout; there is no flat aggregate page inventory, aggregate byte/node quota, provider call, required storage service or default deadline. The 0.23 native routes described above use this format directly. Other existing producers and consumers retain their own format/capacity contracts until separately migrated.

The logical result retains version-1 exact strings, booleans, null, arrays and objects, depth at most 64 and keys/strings at most 65,536 UTF-8 bytes. Numbers remain exact strings. Object keys are sorted uniquely in native compact UTF-8 serialization order; HTML characters and U+2028/U+2029 remain literal, while JSON controls are escaped. `content_sha256` hashes the body excluding that member; physical page layout does not change it. Every exact manifest/page digest is separately verified when accessed. Hashes establish correspondence, not authorship. Checked uint64/host file offsets, allocation outcomes and SHA-256 length representation remain finite physical/format boundaries, not policy quotas.

Import reads a selected canonical-key-order result JSON file incrementally and verifies its exact selected file SHA-256 plus embedded logical SHA-256 before publishing the manifest last. Whitespace is admitted; object key order remains canonical and duplicates/numeric literals reject. The caller selects `page_bytes` (at least 512) and `index_fanout` (at least two); a single descriptor that cannot fit rejects with explicit layout evidence, without truncation. Small subtrees may be stored inline; large containers stream through bounded hierarchical pages. Reader `max_page_bytes:null` imposes no extra caller page budget; `cache_bytes` is a selected cache budget and zero disables caching. This cache budget concerns retained encoded page bytes, not total process RSS or allocator overhead.

Inspection verifies the pinned manifest and reports `manifest_only`; it does not assert that untouched pages exist. Queries verify the accessed index/page paths and report `accessed_pages`. Explicit verify and complete export stream the logical closure, validate structure/counts, recompute logical identity and report `full_logical_closure`. Corruption or missing dependencies reject required reads. No previously reported full verification bypasses verification of newly read bytes.

Queries select an RFC6901 pointer or a stable preorder VALUE `node_id`; root is zero and the virtual digest is one, and keys are immediate edge metadata rather than value nodes. Node IDs reach all admitted fields even when a complete escaped pointer exceeds the control string bound. Containers return a typed empty value plus immediate child count; scalars return their exact value. Pages are in canonical immediate-child order. A versioned cursor binds the exact reference, selector, selected node, order and access profile; it is data rather than an authorization token. Row and byte budgets may change between pages without changing the logical query. The byte budget includes conservative process-envelope headroom and cursor/choice echo; counted-value framing also applies. A response either advances, completes, or refuses the selected insufficient budget. No empty incomplete continuation is returned. The fixed process framing applies to control messages only.

Native export writes a fresh selected file with no replacement. JSON is the complete logical result followed by LF. NDJSON uses `symphony.sbv.node-stream.v1`: one begin record, preorder value-node records with bounded node/parent/edge addresses, then an exact completion record after full verification. Legacy pointer-stream-v1 remains unchanged. Logical counted nodes include keys; logical value nodes exclude keys; exported value nodes add the virtual digest. These quantities are distinct.

`qxctl sbv bundle export --input REQUEST --format json|text|ndjson` streams the native output using bounded buffers. `--receipt-only` renders the complete bounded native receipt instead. The request explicitly chooses the retained output path and native format. JSON/text require native JSON; NDJSON requires native NDJSON. A mismatch refuses before dispatch rather than rewriting the request. The wrapper verifies byte count and physical file SHA-256 while withholding the final JSON brace/LF or NDJSON completion line; it releases completion only after checks pass. Text escapes unsafe/non-ASCII content incrementally. A failed stream may have delivered a prefix; it must not claim completion or append a second stdout error document. A successful write establishes downstream writer acceptance, not durable delivery. Cancellation is checked between reads/writes and before completion; a blocking writer that does not support deadlines cannot be preempted safely by the generic streaming helper.

Publication failures after new storage is created return a closed `recovery_required` domain result, binding the canonical request SHA-256, known paths, actual phase and known publication/durability state. Import recovery includes a nullable pinned reference when available; export recovery includes a nullable expected binary receipt when available. Automatic retry and rollback are false. Successful process/SDK delivery of this evidence does not mean the operation completed; qxctl renders all evidence then exits 5. No existing caller-owned destination is overwritten or removed. Cooperative local filesystem checks are not hostile same-user isolation or a guarantee of recovery after process/transport death.

## Administrative boundary

One engine-process.v2 request on stdin, one digest-bound response on stdout. Request 1 MiB, response 4 MiB, JSON depth 64, values 32768 and string 65536-byte limits remain unchanged. qxctl imposes no operation deadline. The user may select a timeout or absolute deadline; caller cancellation retains process-group cleanup. Legacy artifact files have a separate 128 MiB / 4,000,000-value bound; large artifacts never travel in one administrative response. All native operation inputs reject unknown fields. All numeric experiment input/output values use canonical decimal strings or exact rationals; no floating JSON values.

`capabilities`, `run`, `compose`, `result_inspect`, `result_query`, `evaluate`, `catalogue`, `compose_joint`, `economics`, `book`, `liquidity`, `allocation_economics`, `result_select`, `backend_plan`, `live_plan`, `analyze`, `compare`, `resample`, `experiment`, `split`, `dataset_load`, `dataset_inspect`, `dataset_execute`, `dataset_release`, `fit`, `predict`, `research_history`, `compose_economics`, `generate_census`, `provider_inspect`, `source_retain`, `source_export` are implemented. Every operation has an exact `symphony.sbv.<hyphenated-operation>-input.v1` request and `symphony.sbv.<hyphenated-operation>.v1` response. Installed schemas/templates describe their shapes. qxctl also exposes schema, template and complete result export.

Run, evaluate, compose, compose_joint, economics, book, liquidity and allocation_economics create a single new file selected explicitly by the caller. They require target-host filesystem permission, not canonical-source apply permission. They do not update canonical knowledge, bind an identity or submit an order. The existing process contract's permission-backed-mutation classification describes this narrow output effect. These writes are non-idempotent: an existing path is refused even for identical data. Reads are idempotent at a supplied digest. If publication or the response outcome becomes uncertain, inspect the destination before retrying; never overwrite it automatically.

Paths must be absolute and component-safe. Reads and parent traversal reject symlinks. The parent must already exist. Output uses a mode-0600 exclusive staging file, fsync, atomic no-replace hard-link publication and parent fsync; cleanup removes only its staging name. Interrupted pre-publication work can leave a private `.sbv-*.tmp` file; it is not a committed result. Result paths are never executed. The separately selected native-provider profile below explicitly loads a verified private library stage.

## Historical run

One uncompressed DBNv1 or DBNv3 MBO file, one instrument, with no application-imposed source-byte, metadata-byte or event-count ceiling. Optional constraints are selected by the user (see Dataset resource authority). The caller supplies the exact source SHA256 and dataset; both are verified. Availability order is nondecreasing provider ts_recv with original source ordinal as tie-break. Unknown receive timestamps or mixed instruments are rejected; input is not silently sorted. Event timestamp, receive timestamp, sequence, flags and all other decoded MBO fields remain separate. The DBN undefined price sentinel INT64_MAX (9223372036854775807) is retained verbatim in replay but excluded from trade predicates. Provider timestamp sentinels remain original values; ts_recv must be known for this ordered mode.

The initial criteria catalogue contains spaced positive-size trades and direction of current trade relative to the preceding observed trade. The user selects minimum size, spacing, direction and a positive uint64 selection cap or null for no cap. Zero spacing is admitted. Direction uses past/current observations only. Selection stops admitting signals at the cap; the reported additional eligible count uses the final admitted signal as spacing anchor. This count is diagnostic, not the count of a hypothetical uncapped strategy. The closed signal array has its own canonical SHA256 and cannot depend on later execution/replay choices.

Pass two operates on the immutable census. A caller-selected worker pool of std::jthreads writes disjoint preallocated output slots. Reductions retain census order. Worker errors are captured, joined and propagated; no partial result is published. Source bounds, window arithmetic, price arithmetic and deadlines are checked.

The user selects `none`, `touch_observation` or `user_probability`. An explicit list of distinct signed nanounit price offsets defines targets around each anchor (up to 64, including zero if selected). Positive/zero targets record the first later positive-size trade at or above target; negative targets use at or below. Only records after the signal ordinal and at/before the horizon in availability time qualify. These are trade-touch observations, never broker fills. Touch, quantity, fill time and conditional fill price are distinct fields. Quantity, fill time and conditional fill price remain unavailable. A user probability is a supplied rational assumption with its exact complementary no-fill mass; it is not calibrated from touches. Model-none requires an empty target profile and unused probability 0/1. No queue/latency/fee/portfolio model is implied.

Selected studies are `signal_summary` and `forward_markout`; zero studies is valid. Forward markout is last observed post-signal trade at/before horizon minus anchor in price nanounits, not an execution return. It is unavailable when the horizon exceeds the observed span, no qualifying trade exists, or subtraction overflows. Unknown study names fail explicitly rather than being silently ignored. User-defined result data may be retained in the extensions object, distinctly attributed as supplied data, without being executed or promoted to a native study.

Replay before/after durations are independent nonnegative nanoseconds, including zero. Start saturates at epoch with a disclosed flag; end overflow rejects. Windows include both requested time boundaries and use end-exclusive source ordinals. Equal-time post-signal ordinals remain future observations despite sharing a timestamp. Retention selects the union of windows and stores each decoded observed event once, keyed by source_ordinal. With retention disabled, window manifests remain but events are explicitly not retained. The original DBN file is unchanged. Replay remains usable after removal of that source when events were retained. Pre-window length is not book hydration. Span inclusion, provider cap, book initialization and exchange completeness remain distinct; no complete-book claim is made. Live operation is unavailable; capabilities carries a conceptual lifecycle only.

## Exact scenario composition

The user explicitly supplies return support, probability measure, steps and dependence. No implicit weighting or normalization. Returns must be at least -1; nonnegative weights must sum exactly to one. Inputs have signed numerators in ±1e9 and denominators 1..1e9. Native reduced int64 rational arithmetic checks addition/multiplication overflow; an unrepresentable exact answer fails rather than rounding. Up to 64 support points, 16 steps and 65,536 retained paths, additionally bounded by artifact size/structure.

Economics is sequential multiplication from wealth one. `independent` enumerates each draw sequence with product mass; `same_draw` repeats one draw across all steps with its original mass. Full paths and a separately aggregated terminal distribution are retained. Two independent equiprobable ±1% steps yield wealth 9801/10000 (mass 1/4), 9999/10000 (mass 1/2), 10201/10000 (mass 1/4). Mixed paths alone do not constitute the distribution. All scenario measures remain user assumptions; no calibration claim.

## Portable results and queries

`symphony.sbv.result.v1` contains protocol, origin, status, sections and content_sha256. Required sections: summary, signals, execution, distributions, studies, replay, comparisons, search, resources, diagnostics, choices, provenance. Additional sections are allowed. Each section has status, reason and data; nonavailable/partial sections require a reason. Scalar values are exact strings; unavailable metrics have explicit status/reason/value wrappers. Container booleans remain booleans. The hash omits content_sha256 and covers compact lexically key-sorted native JSON, not file bytes. Native inspections also return a separate exact file SHA256. Source paths or digest possession grant no external data rights.

Result inspection requires the expected content SHA256. Result query selects any JSON Pointer; object/array targets enumerate immediate children, scalars return their value. Container nodes carry empty shells plus child counts so traversal never silently truncates a nested value. A page has at most 256 nodes and a 512,000-byte node budget. Its cursor binds snapshot, pointer, page size and offset. Changed files, snapshots, query parameters or malformed cursors fail. Digests detect change; they do not prove who authored an artifact. Inspection verifies format/integrity, not recomputation or authorship of imported bytes.

qxctl export first obtains native inspection, then reads no-follow bytes and verifies the exact file hash/length before rendering. JSON preserves all fields. Text safely escapes terminal controls, non-ASCII directional controls and DEL. NDJSON emits begin, one JSON Pointer node per container/scalar, then end with complete status and total node count. Empty objects/arrays survive. No end frame means incomplete output; consumers must reject incomplete streams. The retained content hash is itself included in the stream. A GUI or external AI/ML pipeline can consume the same result contract; no frontend-owned financial recomputation is represented as native evidence.

## External models and joint paths

### External census and model evaluation in 0.2

`evaluate` reads the same exact user-constrained Databento source as `run`, and admits `symphony.sbv.external-census.v1`. Signal identifiers remain exactly user supplied, unique within the census and ordered by nondecreasing source ordinal; multiple signals may share an ordinal. The available timestamp must match that source record's receive timestamp. Anchor prices are explicit signed nanounits, not inferred order instructions. The DBN undefined-price sentinel is rejected as an anchor. Context references are inert attributable strings, never executed or fetched.

The declared causal prefix is an end-exclusive source ordinal. `causal_declared` bounds it at the signal cursor; `retrospective` permits later evidence within the source. Producer id/version, optional artifact hash and `deterministic_declared`, `nondeterministic` or `uncaptured` reproducibility remain visible. Reference validity does not prove external code was causal or deterministic. The census digest covers its complete canonical declaration including source identity and producer, before model admission. Changing a follow-up model cannot change that digest. Changing original source bytes intentionally changes source/census identity.

`AdmittedModel` is a compileable immutable C++ source interface with bounded `ModelFrame` and `ObservedTrade` inputs, concurrent `evaluate`, explicit horizon and owned selection. Borrowed trade spans must remain alive and immutable until the synchronous call returns. Two model versions are implemented by this new interface and discoverable through `qxctl sbv catalogue`. The catalogue also lists the three original run profiles and all four currently supported study names, each with its permitted operation names:

- `observed_trade_levels` version 1 selects the nearest requested number of distinct observed positive-size trade prices on each side of the anchor in the later ordinal/inclusive availability horizon. `levels_per_side` is 1–32; anchor inclusion is a separate boolean. The anchor must itself equal a defined observed trade at its source cursor. This is retrospective trade support, not an initialized book or executable liquidity. The five-per-side, anchor-excluded profile has ten conditional prices, each with explicitly assumed mass 1/10. No execution likelihood follows from that support. Missing support, a non-trade anchor or a horizon beyond the observed span follows the user's `unavailable` or whole-operation `reject` policy. Observed candidates on each side remain inspectable even when the conditional support is unavailable; they carry no invented probability. Candidate counts stop at the requested per-side count and exclude the anchor. No missing mass is renormalized. Span inclusion still does not establish exchange completeness.
- `external_outcomes` version 1 admits one outcome per exact census signal id. Its measure is explicitly `probability`, nonnegative `scenario_weight`, or `signed_coefficient`. Exact rational input numerators are ±1e9 and denominators 1..1e9; probability-sum overflow rejects. Non-probability coefficients are not subjected to an unrequested aggregate calculation. Probability support must sum exactly to one. Scenario weights and signed coefficients are never normalized or labeled probabilities. Each support has 1–64 distinct defined prices; supplied support order is retained. Any supported measure may separately supply a probability of execution in [0,1]; its exact complement is recorded. This separate likelihood never turns scenario weights or signed coefficients into price probabilities, and no joint mass is inferred. The user independently labels the support conditioning as execution or scenario. Otherwise execution probability remains unavailable with the supplied reason. Quantity, timing and actual execution stay unavailable. Producer/calibration/evidence references remain external claims; SBV does not verify their calibration. Non-deterministic producers are admitted with that limitation visible.

Every model outcome uses `symphony.sbv.model-outcome.v1`. `qxctl sbv schema --operation evaluate` exposes census/model-selection/outcome and excursion schemas in `$defs`. These definitions are tested against actual installed native output. External code may compute whatever models it chooses; this admission surface states only the domains this native consumer can presently preserve and validate.

Evaluation studies are independently selected: `signal_summary` counts the admitted census; `model_summary` counts available/unavailable model outcomes without imposing probability averaging; `path_excursion` records minimum/maximum observed post-signal price changes and the first source cursor/time attaining each extremum. It is a price path study, not fill-based MAE/MFE or portfolio P&L. A clipped horizon is labeled `truncated_observed_span`; no trade or arithmetic overflow yields explicit unavailability. Empty selection is valid. Replay shares source events across independent signal windows, including external signals with identical source cursors; zero signals starts zero follow-up workers.

### Explicit joint paths in 0.2

`compose_joint` / `qxctl sbv compose-joint` consumes only the caller's listed paths, each with an exact probability and ordered return sequence. It permits 1–4096 unique path identities and 1–16 equal-length steps. Probabilities must sum exactly to one; returns must remain in the selected nonnegative multiplicative-wealth domain. Native arithmetic retains each trajectory and aggregates identical terminal atoms. It never expands marginal support into absent paths. Two equally weighted mixed ±1% paths yield only 9999/10000 with total mass one. Admission verifies the scenario arithmetic, not market feasibility, shared liquidity or capital availability.

### Release and remaining scope

0.2.0-dev is a new exact experimental installation. Its qxctl wrapper does not silently admit 0.1 installations; retain the earlier matching binary for that package. Version-1 portable result artifacts remain readable through the new reader. No global registry or identity service is required to use either installation.

This release does not implement arbitrary executable strategy loading, queue/hydration/latency simulation, calibrated execution distributions, portfolio accounting, cross-validation or the full study catalogue, optimization, comparison execution, market-data acquisition, remote/distributed scheduling, SDK installation, CUDA/tensor runtime integration or live execution. External census and model values are admitted as attributed data; they are not executed plugin code. TensorView defines a source-level owner/device/shape/stride/stream/completion seam; capability status remains unavailable. Further implementations must preserve user choices and independently owned contracts rather than imposing a universal research policy.

`result_inspect` additionally admits an empty expected_sha256 for recovery/discovery of a possibly published destination. It reports expected_digest_verified=false in that case; this verifies internal integrity, not an externally known snapshot or authorship. Queries and exports still require a known digest. `qxctl sbv schema --operation result_export` returns the receipt-owned artifact and NDJSON schemas; operation schemas expose input/output plus shared recursive definitions. The common descriptor uses the exact platform v2 field and limit vocabulary, without owner-specific additions to its top level.

## User-selected economic transforms and studies in 0.3

`economics` consumes an immutable `symphony.sbv.result.v1` evaluation artifact at an exact expected content digest. It verifies the file, result hash, the full declared external-census hash, its exact signal array and one-to-one model/signal correspondence. Selected model rows must use `symphony.sbv.model-outcome.v1` and defined nanounit prices. Additional attributable source fields are retained, not executed. These checks establish integrity and compatible numeric domains; they do not authenticate the producer, re-run a strategy, reread DBN or verify calibration. The preceding 0.2 evaluation artifacts are supported. Legacy `run` touch observations are not interchangeable with typed price support.

The legacy economics profile accepts zero to 4096 distinct signal IDs; partitioned economics also accepts scalable apply/referenced selections. Each binds a version-1 `linear_price_pnl` transform, an interpretation mode and explicit nonexecution treatment. The engine retains this selection separately from the original census; it never regenerates signals. Each transform declares a reference price, positive integer `price_unit_nanos`, signed rational `position_units`, signed rational `value_per_price_unit`, signed `cost_per_outcome`, nonzero signed `return_basis`, and attributable `pnl_unit` label. A negative cost can express a rebate; a negative position can express a short. Labels do not verify currency or instrument metadata.

```
gross_pnl = (price_nanos - reference_price_nanos) / price_unit_nanos
            * position_units * value_per_price_unit
net_pnl   = gross_pnl - cost_per_outcome
return    = net_pnl / return_basis
```

All original support points, weights, gross P&L, cost, net P&L and returns remain inspectable. This linear transform is a selected scenario interpretation; it does not assert that a modeled support price is an actual fill or settlement. Quantity is the user's scenario position, not modeled fill quantity. Nonlinear, inverse, quanto and option economics require separately implemented transforms. Returns below -1 and negative return bases are admitted here; the separate multiplicative-wealth composer retains its own domain.

- `support_only` preserves the source measure and conditioning, without multiplying by execution likelihood. `nonexecution_pnl` must be null because this mode does not use it.
- `execution_mixture` requires a probability price measure conditional on execution and a separately supplied likelihood with an exact complementary nonexecution probability. Each price probability is multiplied by the execution likelihood; an additional atom carries the nonexecution probability and the caller's explicit `nonexecution_pnl / return_basis`. Nonexecution P&L is already net; price-outcome cost is not charged to it implicitly. Zero-mass atoms remain visible. This mixture is still an attributed assumption, not calibrated evidence.

`on_incompatible` is `unavailable` or `reject`. Missing model support, missing execution likelihood, a nonprobability mixture, or a probability-only study on another measure follows this policy. Under `unavailable`, compatible transformed support is preserved, the incompatible atoms/studies carry exact reasons, and the result is partial. Under `reject`, no result is published. Malformed input, mismatched source identity, invalid probability or unrepresentable arithmetic rejects independently of this policy. No domain is silently normalized or relabeled.

Three optional, versioned study descriptors are exposed by the catalogue and installed schemas. Every study is per selected signal; empty study selection is valid:

- `weighted_return_sum`: exact sum of weight times return. Probability, scenario weights and signed coefficients are accepted without normalization. The result is labeled a weighted sum; nonprobability measures are not called expectations.
- `return_moments`: exact probability mean and population variance, without sample correction or annualization. Requires a probability measure. Zero-mass contributions are mathematically absent.
- `return_quantiles`: caller supplies 1–32 exact levels in [0,1], with order and duplicate levels preserved. The selected rule sorts returns exactly, ignores zero-mass support, and picks the first cumulative mass greater than or equal to the level. Level zero selects the minimum positive-mass return. No interpolation or quantile approximation is substituted.

Study descriptors include required input protocol, units, measure domains, parameters, grouping, missingness, exact method, backend, atom bound and output schema. Results include the signal identity, resolved parameters, measure/conditioning, atom count, method and an explicit unavailable estimation-uncertainty field. No sample-size or denominator claim is invented for user-defined coefficients. No cross-signal dependence, shared capital, trade pairing, portfolio/account curve, drawdown, Sharpe ratio, execution calibration or economic feasibility is inferred.

Input transform/quantile rationals keep the existing numerator ±1e9 and denominator 1..1e9 bound. Admitted artifact weights and calculated outputs use reduced signed int64 rationals (excluding INT64_MIN numerators). Every arithmetic intermediate must be representable in the selected profile; overflow rejects, with no float rounding or hidden sampling. Price subtraction is checked before rational conversion. Exact ordering uses a wide integer product. There are at most 65 source prices and 66 mixture atoms per signal. Numeric restrictions are implementation contracts, not universal constraints on user-authored transforms.

The new result adds typed `economics` rows and typed study results to the portable contract. It copies the entire original replay section verbatim, including windows for source signals outside the chosen economic subset, so retained replay remains usable without the source file. `source_context` preserves the complete original census, original choices and original provenance; selected model/signal rows are retained separately. All fields remain accessible through existing qxctl inspect/query/export, including unknown future sections. The result file remains subject to the existing size/structure bounds and exclusive publication semantics.

0.3.0-dev is a separate exact installation with twelve qxctl leaves and nine native operations. Earlier binaries/installations remain separately usable; this client does not silently admit an older installation. Full account/portfolio simulation, initialized books, acquisition/lifecycle integration, installed SDK/backend contracts and milestone/release closure remain open.


## Source-bound order-book continuation (0.4)

`book` consumes an exact DBN source and the digest of a closed `run` or `evaluate` census artifact. Caller-selected signal IDs preserve their existing coordinates and never rerun signal criteria. The caller chooses independent replay durations, observed-event retention, frame cadence (`signals` or `signals_and_event_ends`), display depth 1–64, a positive caller-selected frame bound or null, checkpoint emission, and anomaly policy. Exceeding the frame bound rejects; there is no implicit thinning. Empty selection is valid and emits no checkpoint. The current owner interface enumerates all operations.

The `databento_mbo_orders_strict_v1` profile accepts one publisher, instrument and channel in one ordered source file. DBN decoding stays with the exact existing SQAV adapter. SBV interprets decoded actions using full order-ID state and ordered aggregate price levels. Add inserts; modify replaces the order; cancel subtracts its stated size and removes an exhausted order; reset clears all orders. Trades, fill reports and no-op records do not change quantities. Prices and quantities remain exact integers. Display depth limits exported frames, never the reconstructed state or full checkpoint. Locked and crossed observations are classified explicitly, without altering the supplied feed.

An initial fragment has **unknown state**, not an empty initialized book. A clean source reset or supplied compatible checkpoint is required. Snapshot records following a reset remain incomplete until an instrument event-end (`F_LAST`), including a later nonsnapshot event end when necessary. Frames at intermediate records are unavailable even if orders have been accumulated. Snapshot flags without an opening reset invalidate previously initialized state. Empty reset-plus-event-end is a valid initialized empty book.

`on_anomaly` is explicitly `reject` or `invalidate_until_reset`. Gaps (`F_MAYBE_BAD_BOOK`), top-of-book/MBP records, publisher-specific flags with unimplemented dataset semantics, unknown actions, duplicate adds, absent modified/canceled orders, invalid sides/prices/sizes and cancel mismatches follow this policy. Invalidation clears internal state and keeps later frames unavailable until a clean reset. An unknown initial state is expected missingness, independently of anomaly policy. Quantitative A/M/C interpretation is withheld while state is unknown. Reserved flag bit 1 is ignored according to the provider contract. Inaccurate receive timestamps remain flagged; ts_recv ordering is a selected reconstruction clock, not proof of exchange availability. No sequence-plus-one rule is applied to filtered instrument records. Unsupported feed semantics require a distinct admitted profile; this profile does not restrict user-authored alternatives.

Portable `book_frames` separate reconstructed levels from raw `replay.events`. Each frame carries source ordinal, timestamp, associated selected signal IDs, readiness, reason, event-boundary status, timestamp warning and initialization lineage. Windows retain their requested limits, observed-span flags and unproven completeness. Hydration cursor bounds are separate from visible windows. All retained frames and raw window events are self-contained for viewing. Mechanically rebuilding the full book from them still requires the referenced full source/checkpoint when hydration records were outside the retained windows. No complete-source, queue-priority or execution-likelihood claim follows from a ready frame.

Optional `book_checkpoint` contains every remaining order, exact source/dataset/publisher/instrument/channel binding, the next source ordinal and the preceding receive timestamp. Emission requires initialized state at a complete event boundary. Resume verifies the artifact content digest, profile, scope, ordered unique IDs, order values, cursor and source event boundary. The cursor must precede every requested window; a future checkpoint cannot hydrate an earlier signal or pre-window. Checkpoints only resume within the same exact source in this release. Integrity is verified, but supplied checkpoint authorship/state truth is not authenticated. Checkpoint provenance remains explicitly attributed.

One state worker processes the ordered hydration prefix; separate experiments may run concurrently. This does not alter existing parallel census/model evaluation. No live request, calibrated queue/fill model, source acquisition lifecycle, distributed scheduler or CUDA execution is added. `catalogue`, exact installed schema/template, terminal inspect/query and lossless export expose the entire new capability and payload. The portable result protocol remains v1; older artifacts remain readable through generic result operations.


## Displayed-depth execution scenarios (0.5)

`liquidity` consumes a content-digest-bound strict `book` result and inline or referenced order intents. Each intent names a source signal and retained frame ordinal at or after that signal, buy/sell side, source quantity 1–1e9, optional limit price (null means market), IOC or FOK behavior, per-level participation in [0,1], and supplied or unavailable activation probability. The profile is `displayed_depth_sweep` version 1. Actual source/book integrity is preserved; artifact hashes do not authenticate authorship or establish market truth.

The selected frame is explicit retrospective evidence. Its receive-time delay from the signal is reported, but is not an inferred arrival book, a continuous latency simulation or a calibrated time-to-fill distribution. Unknown frames, unavailable books, insufficient retained depth and excluded locked/crossed or flagged-time frames follow the caller's `on_unavailable` selection. Malformed identities, prices, quantities, source digests, ordering and shapes reject independently. A ready frame must be at an event boundary, with positive ordered unique levels and consistent total/retained level counts. A frame earlier than its signal is rejected.

The model walks opposite-side displayed levels in best-price order and respects the supplied limit. At each level the per-order participation cap is `floor(original_displayed_quantity * max_level_participation)`; selected quantity is bounded by that cap, remaining order quantity and physically remaining modeled depth. No rounding up, hidden liquidity, replenishment, queue priority, venue routing or historical response to a hypothetical order is invented. Source quantities have no inferred contract multiplier or currency conversion. Signed prices are supported except the DBN undefined sentinel.

IOC takes the selected available quantity and cancels the remainder. FOK applies all fills only if the full requested quantity is supported; otherwise it emits zero fills and consumes no modeled depth. `depth_policy=require_sufficient` leaves an unresolved remaining quantity unavailable if omitted levels might satisfy it. An explicit limit boundary or zero participation can prove that omitted levels are irrelevant. `visible_only` instead authorizes a scenario restricted to retained levels, with that coverage label. This choice never establishes complete market depth.

`liquidity_mode=independent` evaluates each intent against an independent immutable frame copy using caller-selected workers and stable input-order output. Those outcomes do not share feasible aggregate liquidity. `shared_snapshot` requires every intent to name one exact frame; it executes sequentially in caller order and deducts successful conditional fills from that one shared snapshot. This scenario is conditioned on all supplied intents being active. Supplied activation probabilities must therefore be 1; unavailable probability declarations remain valid. Fractional activation requires stochastic joint branching, which this profile does not implement. An unavailable earlier shared intent leaves later dependent outcomes unavailable; it is not treated as zero consumption. Failed FOK orders are resolved zero-consumption outcomes. There is no cross-frame shared market evolution or shared-capital model.

The conditional outcome contains per-level allocations, filled/unfilled quantities, disposition, exact notional in price-nanounits times source quantity, and rational VWAP. Activation probability is optional. When supplied as p, an independent scenario puts mass p on its conditional filled quantity and 1-p on zero. If the conditional quantity is zero, the quantity distribution collapses to zero with mass one. Any-fill probability is p only when conditional quantity is positive; complete-fill probability is p only when it equals the requested quantity; no-fill probability is its exact complement. These fields are `scenario_assumption`, never empirical/calibrated predictions. With activation unavailable, conditional price/quantity still remain available but probability fields remain unavailable. Shared-snapshot probabilities describe the explicitly all-active scenario only.

Two optional studies are selectable individually or together, and empty study selection is valid. `liquidity_summary` reports available/unavailable and conditional full/partial/no-fill counts, without treating them as estimated probabilities. `fill_quality` reports filled/requested quantity and side-adjusted VWAP minus the source signal anchor: buy uses VWAP-anchor, sell uses anchor-VWAP. No fill makes slippage unavailable, not zero. These are execution-scenario studies without fees, future markout, account P&L or risk metrics. Native exact signed 128-bit arithmetic holds price-times-quantity sums and side-adjusted differences; reduced rational outputs are decimal strings, without float conversion or int64 truncation. Input participation and activation use the existing bounded exact rational profile.

Every result copies the original book frames, checkpoint and entire replay section, including unselected source windows. `source_context` retains original signals, choices and provenance. All outcome, probability, quantity-mass and study fields are typed in installed schema definitions and accessible through terminal result inspect/query/export after source removal. Existing per-signal economics remains a separate model-artifact transform: this release does not silently convert fill allocations into random price support or a portfolio P&L curve. A later quantity-aware economic contract must define that composition explicitly.

0.5.0-dev has eleven native operations and fourteen qxctl leaves. It preserves the original portable-result protocol and separately installed earlier artifacts. Real initialized historical validation, full venue/session/correction semantics, calibrated likelihood, queue/latency/impact models, stochastic shared-liquidity branching, SDK/backend integration and M1 release gates remain open.


## Quantity-aware allocation economics in 0.6

`allocation_economics` / `qxctl sbv allocation-economics` consumes a digest-bound displayed-depth liquidity result. The new `filled_quantity_markout` transform values its actual modeled fill allocations. The earlier per-signal `economics` transform keeps its separate random-price-support meaning. Neither contract implies broker execution or account simulation.

Selections contain unique order IDs and a valuation: mark price, mark availability timestamp, mark evidence description, positive price unit, signed rational value per price unit, signed activation/filled-order/per-filled-unit costs, nonzero signed return basis and P&L unit. All conventions are caller choices. Negative costs denote rebates. Marks must be defined int64 prices and their known timestamps must be at or after the selected frame. The supplied evidence is retained as an unverified claim; a timestamp check does not independently establish the mark's provenance or market causality. Mark selection never changes the original signal census.

For filled quantity Q, allocated notional N=sum(fill price * fill quantity), mark M, side S=+1 for buy and -1 for sell, price unit U and value multiplier V: gross=S*(M*Q-N)/U*V. Conditional cost=activation_cost+(Q>0 ? filled_order_cost : 0)+Q*per_filled_unit_cost. Net=gross-cost; return=net/return_basis. Unfilled requested units contribute no markout or per-filled-unit costs. Active zero-fill and canceled FOK orders still incur the selected activation cost. Markout is hypothetical valuation, not a liquidation execution or realized P&L.

`conditional_only` requires null inactive P&L and does not create probability mass. `activation_mixture` requires explicit inactive P&L and uses only the source order's supplied activation probability. Its two labeled branches retain active and inactive costs/values separately, including when both have zero fills. Missing activation produces an unavailable mixture while preserving available conditional economics. Users select `on_unavailable=unavailable` or `reject`; rejection leaves no artifact. Source-unavailable allocations remain unavailable even with zero supplied activation, since the selected conditional result is unresolved.

The source order, outcome identities, original order choices, quantities, fill totals, notional, VWAP, IOC/FOK disposition and conditioning are checked. The entire source execution array is retained even when selecting fewer orders. Shared snapshots remain conditioned on all orders being active in original input order; no independent cross-order probabilities or portfolio distribution are inferred. Source hashes check identity, not authenticity. The transform validates economic consistency without rerunning the book/execution simulation.

Zero, one or both studies may be selected. `allocation_costs` exposes the three conditional cost components and total. `activation_moments` reports exact per-order expected net markout/return and variances for the selected activation mixture. Missing or unselected mixtures remain explicitly unavailable to that study. These are assumption-derived moments, not calibration or account-risk estimates. No aggregate is formed across potentially conflicting liquidity, currencies, capital or time bases.

Inputs use existing bounded rationals (numerator ±1e9, denominator 1..1e9), int64 prices and source quantities up to 1e9. Intermediate and output arithmetic uses checked, reduced signed 128-bit rationals, with the most-negative int128 excluded. Cross cancellation precedes multiplication. Any unrepresentable intermediate rejects the whole request before publication, with no floating conversion or silent precision loss. Selecting an optional variance can exceed these bounds when the conditional transform is representable; choosing fewer studies remains valid. This is bounded exact arithmetic, not arbitrary precision.

Output `symphony.sbv.allocation-economic-outcome.v1` exposes conditional economics and optional mixtures; schemas also define typed valuations and studies. Original signals, complete execution allocations, raw replay, book frames and checkpoint are copied without alteration. Nested source choices/provenance remain in source_context, with immediate source and census digests in provenance. All values use ordinary inspect/query/export. Computation is sequential in stable selection order over immutable allocations; CPU concurrency remains available in independent signal and liquidity passes. Provider/live requests and new spend are zero.

0.6.0-dev is an exact experimental installation with twelve native operations and fifteen qxctl leaves. Earlier matching installations and clients remain separate; v1 result artifacts remain readable. Calibrated execution, initialized real historical validation, account ledgers, joint shared activation, liquidation, funding, FX, nonlinear derivatives and portfolio feasibility remain open extensions. No M1 completion or performance ranking is claimed.


## Typed selection and disconnected contracts in 0.7

The legacy direct `result_select` / `qxctl sbv result select --input ...` profile is a read-only native array view over an exact result digest. The user chooses the source array pointer (up to 65,536 rows), up to sixteen conjunctive predicates, eight ordered sort keys, sixty-four named column pointers, page size 1–256 and a continuation cursor. Columns omitted by selecting an empty list return full rows. Integer strings and reduced rational comparisons use exact signed 128-bit domains; rational ordering uses continued fractions, avoiding overflowing cross products. Text uses bytewise UTF-8 ordering, booleans false then true. No numerical type or missing-value meaning is inferred.

Predicates select eq/ne/lt/le/gt/ge and an explicit missing policy include/exclude/reject. Missing means absent pointer; present null/wrong-domain values reject. Every predicate value is validated even when another predicate excludes the row. Sort keys validate all matched rows, with explicit first/last/reject missing placement independent of direction. Ties preserve original source index. Projected missing fields carry unavailable status; full values retain all source fields. Each page discloses source count, matched count, offset, rows and completeness. Its cursor binds all selections, exact path/snapshot and page size. Response pages stop before 512,000 row bytes or 16,000 row values; an individually oversized row requires narrower projection and fails explicitly. Selection never rewrites the source or computes a financial aggregate. qxctl matches native Unicode canonicalization when checking query identities, including U+2028/U+2029 and literal backslash-u text.

`backend_plan` is disconnected planning, not resource reservation or execution. The caller supplies requested backend/workers/memory, declared limits, dependency-safe independent or serial-state scheduling, reject/reduce policy and optional explicit CPU fallback. CPU has a 64-worker implementation ceiling; serial-state planning grants at most one. CUDA/tensor runtime is unavailable. No automatic fallback or device copy occurs. Resolved fields are a proposed CPU plan when admitted; applied is always null and reservation false. Declared limits are caller assumptions, not SHV/SNV observation or grants. Existing calculations retain their separately reported actual scheduling; a plan does not silently install per-run memory enforcement.

An optional tensor descriptor declares dtype, device, storage bytes, byte offset, shape, nonnegative byte strides, owner/stream IDs and completion state. Rank is at most sixteen; dtype width, alignment, empty/scalar shapes, reachable storage extent, arithmetic overflow, selected memory limit, device match and ready completion are checked. Zero strides allow broadcast views. Negative strides and unknown dtypes are outside this profile. Metadata admission cannot prove foreign pointer validity, ownership or device completion. Pending, failed, cancelled, incompatible or oversized buffers remain unavailable for the proposed plan.

The receipt installs the independent header-only C++26 `symphony/sbv/interop.hpp` v1 contract and CMake target `Symphony::SbvInterop`. It needs no vendor or engine JSON headers. Buffer descriptors, backend requests/plans, shared lifetime leases and a completion interface expose ready/pending/failed/cancelled state, explicit monotonic-deadline wait and cancellation request. Host access requires a ready CPU lease. Adapters retain ownership and supply the completion implementation; the core does not fabricate asynchronous completion or initialize a GPU runtime. This is a source contract, not a stable cross-toolchain binary ABI. The full engine/model SDK remains distinct.

`live_plan` simulates an explicit ordered offline event fixture (up to 512 records/128 signals). Inputs include independent pre/post nanoseconds, event/byte/pending-window capacities, selected overload policies and a caller-declared closed-through watermark. Pre-trigger and pending windows share retained event indices. Expiry cannot remove events needed by a pending window; selected overflow may drop oldest/newest events or reject. Pending overflow may reject or skip the signal. Equal-time source indices remain distinct, windows include both time boundaries, and only windows closed by the watermark finish. Output exposes completed/loss/pending windows, retained/expected fixture counts, every dropped event and skipped signal, and actual simulated peaks. Fixture completeness is not market completeness. The planner has no subscription, socket, entitlement probe, spill implementation or activation path; `can_activate=false` and provider requests zero.

All three operations are read-only, preserve extensions where admitted, return bounded process-v1 payloads, and have installed qxctl schemas/templates. The release has eighteen SBV leaves and fifteen native operations. Runtime GPU integration, live capture, resource reservation and full SDK packaging remain separate work.

## Source-series studies and supplied-trial comparison — experimental 0.8

`analyze` binds one immutable result digest and an array pointer, with explicit value/weight pointers, series role, unit, ordering attribution, missing treatment and optional study selection. Observation count is caller/host constrained; five study families are implemented. Equal probability is assigned over retained observations; supplied probability, scenario and signed-coefficient weights are preserved without normalization. Missing fields can be excluded or rejected; a present malformed number rejects. The canonical source is unchanged and retained signal, execution, replay and book sections remain accessible.

`series_summary` computes count, weight total, exact weighted sum and retained-observation extrema. `series_moments` requires total probability one; the user selects population variance or the equal-weight sample estimator. `series_quantiles` uses an inverse CDF on positive mass, q=0 minimum and first cumulative mass >= q. `equity_drawdown` treats caller-declared equity in retained source order, computes peak-relative and absolute drawdown and first recovery of the maximum absolute drawdown. Relative conventions are positive peaks only or absolute peak denominator; zero remains unavailable. No account feasibility, cashflow adjustment, elapsed duration or return sampling is inferred.

`return_ratios` requires caller-declared per-period returns, a constant benchmark, positive periods-per-year and explicit `binary64_roundtrip`. Mean differential return, selected variance and downside second moment remain exact. Standard deviation and square-root time scaling use IEEE 754 binary64 under nearest rounding; outputs retain max-digits round-trip decimal and 16-hex-digit bit identity. Sharpe is mean differential return divided by the selected standard deviation, times sqrt(periods). Sortino uses the square root of the second moment of negative benchmark differences over all probability mass. Zero denominators are unavailable. Scaling records the user's convention; it establishes neither serial independence nor predictive validity. Exact-rational-only selection leaves square-root ratios unavailable or rejects according to `on_incompatible`. Checked signed 128-bit intermediate overflow rejects before persistence. Selected operations may complete with unavailable study values; inspect per-study status.

`compare` binds inline or referenced supplied trial records and exact numeric objectives to source artifact digests. Completed artifacts are read and validated; failed/pruned records retain caller reasons, parameters and lineage. Objective pointers, units, directions, positive scales and signed weights are user inputs. Weighted utility is sum(direction sign * value / scale * weight), sorted descending with stable input ties and competition ranks. Pareto fronts use maximize/minimize directions on raw values and ignore weights, including zero/negative weights. Front zero is nondominated; ties do not dominate. Missing fields exclude whole candidates or reject; malformed present values always reject. There is no aggregate source-byte policy ceiling; individual legacy sources retain their format bound. Empty method selection still records the ledger and extracted objective values. This is comparison of already supplied trials, not an executed optimizer, validated holdout protocol or significance test.

Both operations persist the same immutable result envelope, disclose choices/provenance/resources and support complete qxctl inspect/query/select/export and future GUI access. Current bounds are declared implementation limits, not restrictions on independently authored studies or optimizers. Source schema definitions include normalized series observations, study status rows and comparison candidates; study meanings are defined above and exposed by catalogue cards.

## Bootstrap and local trial execution — experimental 0.9

`resample` selects IID, moving-block or circular-block uniform row resampling from an exact immutable source array. Uniform sampling is explicit; source weights are never silently imported. Missing or malformed source values reject (select the source observations before calling). The user selects seed, sample length, block length, replica count, workers, optional source-index retention and zero or more studies. Source/sample/replica/retained-index counts and workers have no separate policy maxima; caller parameters, checked arithmetic and host resources apply. IID requires block one; moving starts range from zero to N-block inclusive; circular starts cover all N rows and wrap. Blocks concatenate, with the last truncated to the sample length.

SplitMix64 has fixed unsigned 64-bit arithmetic. Replica r initializes a mixer at seed + r*0xd1342543de82ef95 modulo 2^64 and initializes its own generator with the mixer's next word. Rejection-modulo sampling discards words below (-N mod 2^64) mod N before reducing modulo N. Separate per-replica generator state and ordinal output are unaffected by worker scheduling; this is not a proof of statistical independence between streams. Means, population moments of replica means and empirical inverse-CDF quantiles use checked exact rationals. The optional index tape permits separate reconstruction. Empirical percentiles do not establish confidence coverage, stationarity, serial independence or predictive returns. External bootstrap protocols remain possible through source artifacts and extension data.

`experiment` runs caller-supplied native SBV producer requests in an existing private local directory. This is local SBV artifact orchestration; no SOV remote/job authority, network endpoint, shell/plugin execution or broker operation is inferred. Ready legacy trials omit output_path and resolve `<id>.result.json` inside the directory; partitioned trials select explicit output and result_read_options. Pruned trials retain the caller's reason and null request/operation. Parameters and lineage remain supplied claims. IDs use ASCII letters/digits/underscore/hyphen, at most 80 characters. The engine checks the outer*maximum-child worker product against an optional caller budget and requires one outer worker for deterministic ordered stop-on-failure. It applies those selected worker counts, not a host resource reservation.

One process-held, nonblocking private flock serializes a directory. Plan, claim, completion and result files use immutable no-replace publication and fsync. A retry must select a new summary output path and the exact same plan/version (summary path is excluded from plan identity). Committed trials are reused only after claim/request/plan identity and child content/file hashes verify. A prior claim without completion remains **ambiguous** and is neither rerun nor adopted, even if a result is present. This also covers a child reporting failure after its result appeared. Unstarted trials can run on a later identical invocation; failed/pruned/ambiguous records are retained. An ordered stop policy prevents later claims. An interrupted experiment may have committed child artifacts without its summary; every journal is itself a normal SBV result and remains inspectable/exportable. A new separately identified experiment is the explicit retry path after reviewing ambiguity.

Concurrent producers with identical content use distinct staging names derived from destination and content, preventing false collision across different output files. Destination publication still never replaces an existing file. Worker exceptions and deadlines can leave durable claims; this is disclosed uncertainty rather than evidence that no side effect occurred. Caller-owned private directories are the current operational scope, not a shared multi-tenant or remote filesystem guarantee.

## Independently installed SDK — experimental 0.10

The versioned native shared library exposes `symphony_sbv_sdk_abi_v1`, `symphony_sbv_sdk_version_v1`, `symphony_sbv_sdk_process_v1` and `symphony_sbv_sdk_release_v1`. It accepts the same bounded engine-process.v2 request envelope, native operations, deadlines and exact payload conventions as qxctl. It returns the same digested response envelope with native status. No C++ exceptions cross the C ABI. Request bytes are borrowed until return; each response is a separate native allocation with length and convenience NUL termination, released exactly once by the originating library. Null release is valid. Invalid ABI arguments return 64; allocation/boundary failure returns 70 without a response. As with process failures, a caller must inspect output paths after uncertain writes. Concurrent calls are permitted; artifact and experiment exclusivity rules still apply.

The exact installed `SymphonySbvSdkConfig.cmake` exports `Symphony::SbvSdk`. Its public C header has no C++/JSON/vendor dependency. `sdk.hpp` is an optional RAII byte-buffer wrapper. A separately packaged external standard-library Python consumer in the implementation evidence loads the same installed ABI, verifies local receipt/file identities and response correspondence, and reproduces native results without recomputing finance. It is deliberately outside the Python-free native repository and installation. Its receipt checks are local integrity checks, not an authenticated distribution signature or protection against a concurrent hostile owner of the installation. External consumer languages do not become engine dependencies.

JSON transfer currently copies bytes and preserves rational/integer strings. The SDK does not pretend to be zero-copy Arrow/DLPack, a vendor model runtime, a CUDA implementation or an isolation boundary for untrusted native pointers. Those optional adapters can compose with the independent interop lifetime/stream contract and native external-outcome admission. All twenty-seven owner operations remain available through qxctl; capabilities reports the installed SDK contract and bindings. The shared library adds a language boundary rather than a separate quantitative meaning.


## Temporal partitions — experimental 0.11

`split` consumes a digest-bound SBV result array and explicit relative JSON Pointers for a unique string sample ID, label start/end and optional whole-observation availability. All timestamps are canonical uint64 nanosecond strings in the caller-declared shared clock. Labels are closed intervals `[start,end]`; a zero-duration label is a point. Equal endpoints overlap. Missing/malformed fields reject; no label horizon, censoring, availability or feature lookback is invented. Supplied availability remains an assertion, not verified causal lineage.

Choose `explicit` fold windows or native `expanding`/`rolling` schedules. Explicit folds name train/test half-open windows and a fit cutoff. Generated schedules start with the supplied train start, first train end and first fit cutoff; each step adds `step_ns` to train end and fit cutoff, and to train start only for rolling mode. Test start is train end plus `gap_ns`; test end adds `test_duration_ns`. IDs are `fold-0`, etc. Time duration is independent of observation density. `past_only` requires train end and cutoff no later than test start; `unrestricted` admits retrospective/custom designs. This is a per-fold constraint, not a claim that the ordered collection is causally independent.

Membership uses label start in `[window_start,window_end)`, preserving source row order, including ties. Test membership takes precedence if explicit windows overlap. Optional `label_overlap` purging excludes a training label intersecting the union of selected test labels expanded by `purge_before_ns`/`purge_after_ns`. Gaps in that union are not filled. `none` requires zero padding. `embargo_ns` independently excludes train starts in `(maximum selected test label end, maximum end+duration]`; zero disables it. An empty test selection has no purge/embargo domain. Padding/embargo/schedule uint64 overflow or underflow rejects, never clamps.

Choose `by_fit_cutoff` to exclude candidates whose supplied whole-observation availability is greater than the cutoff; equality is admitted. It requires a pointer. `ignore` allows absent availability and retains late observations; diagnostics disclose both. Purging, embargo and availability exclusions accumulate all applicable reasons. Raw overlap/late candidate counts and retained overlap/late counts remain visible even when filtering is disabled. An empty retained train/test fold is unavailable; it is never replaced or silently discarded.

Output retains normalized observation coordinates, each resolved fold window, train/test source indices, exclusion reasons, exact test/purge interval unions and embargo endpoints. Index-only output binds the source digest/pointer; selected `retain_rows` embeds unchanged source rows for direct native analyze/resample or external fitting. No weights are normalized or recalculated. All nodes are inspectable/queryable/selectable/exportable through qxctl and the installed SDK. The source artifact and original replay remain untouched and linked by digest.

For explicit/rolling/expanding plans, source/fold/retained-row counts have no separate policy maxima; legacy output alone retains the 128 MiB/JSON-value format limit. An explicit zero-fold plan and empty source array are supported. This implementation uses one CPU worker, interval-union sorting and binary search; independent split trials may use the existing experiment runner. Oversized retention rejects without silently switching to index-only output. The current release has twenty-seven native operations and thirty qxctl leaves.

The operation computes partitions, not fitted models, predicted scores, enforced holdout authority, nested validation or stitched combinatorial performance paths, multiple-testing correction or independence. Repeated test selection remains in the fold records. Arbitrary external protocols and user-defined clocks/conventions remain possible through other implementations and exact source artifacts.

## Combinatorial temporal partitions (0.15)

The additive `split` plan kind `combinatorial` selects k-of-N time groups as
held-out groups, with the complementary groups as training candidates. Supply
`groups` in chronological order, each with a unique `id` and uint64 nanosecond
`start_ns`/`end_ns`. Groups are nonempty, disjoint half-open windows; adjacent
windows and gaps are permitted. At least two groups and `0 < test_group_count < N`
are required. Labels are assigned by start time even if they cross group ends.
Rows in gaps or outside the groups remain visible in the observation table and
source artifact; they are not silently assigned to a nearby group. Row order is
preserved. No observation weights or user fields are changed.

Supply a common `fit_cutoff_ns` and the existing user-selected chronology,
availability, purging, padding and embargo policies. `past_only` requires every
training window end and the cutoff to be no later than the first held-out
window start in every requested fold; ordinary symmetric combinations therefore
use `unrestricted`. No policy is silently changed and no fold is silently skipped.
A fold with an empty retained training or test selection is explicitly unavailable.

Purging uses the union of actual test label intervals across all selected groups,
with the chosen padding. Embargo applies separately to each populated test group:
`(maximum label end in that group, maximum end + embargo_ns]`. It can therefore
exclude rows following an earlier held-out group even when another held-out group
occurs later. Empty groups create no embargo. Group-level embargo records preserve
their origin, including overlapping embargo intervals. Native membership checks
use their merged union. The existing `embargo_interval` is null for this profile;
`embargo_intervals` is the explicit per-group list. All timestamp overflow and
underflow reject before artifact publication. Ignoring availability, disabling
purging or setting zero embargo remains a user choice; overlap/late diagnostics
remain visible.

Combinations are ordered lexicographically by input group indices. `fold_offset`
is a canonical nonnegative decimal string of arbitrary precision. `fold_count`
is a canonical uint64 decimal page size, including zero for discovery, or null
to materialize all remaining folds when that page fits the host representation.
Exact total counts and offsets are not limited to uint64. Direct unranking skips
whole combination subtrees; it does not enumerate earlier folds. Nonzero pages
past the end or extending beyond the remaining space reject. Offset equal to the
total with a zero/null page returns an empty terminal page. Offset beyond the
total always rejects. `next_fold_offset`, `complete_space`, the total number of
combinations and full-space test appearances per group describe the selection.
A completed page is not presented as completion of the entire combination space.
No implicit deadline, random subset, sampling, fold cap or row-fold workload cap
is imposed on this new plan mode. The user selects the page and deadline.

The split planner retains typed source/index state in memory. The legacy output
profile alone retains its 128 MiB artifact and JSON
value representation bounds apply. These are distinct from the unlimited-size
local DBN dataset feed. Page sizes must be host-addressable and the selected output
must fit the artifact representation; a large source/retention-format release
remains separate work. No silent truncation, spill or index-only substitution
occurs. Index-only output avoids copying retained rows across folds.

Results use `combinatorial_intervals_v1`, with exact source/digest/pointer and
choices. `groups` records source membership plus selected-page test/training
candidate use counts. `folds[*].windows` binds `combination_rank`, train/test group
indices and cutoff; group window definitions are in `groups`. All exclusions,
retained rows, diagnostics and counts are available through the same qxctl
`sbv split`, `result inspect/query/select/export`, schema, and installed SDK
surfaces. The ordinary expanding template remains available; the installed
split schema describes this alternative plan fully. Existing temporal plans
retain their selection and embargo semantics.

These are combinatorial partitions with optionally selected purging. They do not
fit a model, assemble independent out-of-sample performance paths, compute a
multiple-testing estimator, orchestrate nested fits or prove an untouched holdout.
Repeated test use is disclosed rather than prohibited. External fitting/search
pipelines can consume exact fold indices or retained rows and preserve the
native result digest as their lineage reference.

## Resident datasets (0.12)

`dataset_load` validates the exact source path, SHA256 and dataset through the
selected SQAV DBN decoder once, then retains immutable decoded MBO events in a
native companion process. `dataset_execute` invokes `run`, `evaluate` or `book`
against that shared allocation; its nested original request omits `output_path`,
which is supplied by the wrapper. Exact source identity must match the load.
The original file can subsequently disappear; the resident snapshot remains
unchanged. Signals, models, reconstructed books and replay are computed afresh
from those immutable events. This is an execution cache, not SQPV persistence,
SQDV delivery or a new provider binding.

`dataset_inspect` exposes identity, event count, decoded bytes, accounted load
buffer bytes, selected budget, residency, decode/read counters, active jobs and
workers, completed/failed jobs and idle expiry. `dataset_release` refuses while
jobs are active; on success it stops admission and frees the event allocation.
Release removes its endpoint before attempting the acknowledgement, whose
wait follows the user-selected request deadline, or has no time limit. Release
reports historical allocation sizes, not remaining allocated memory.
No endpoint/PID kill or implicit file fallback occurs. A failed transport after
submission has uncertain outcome: inspect the requested result before retrying.
Memory is volatile; a crash requires an explicit reload with a fresh instance.

The caller supplies an existing no-symlink mode-0700 private directory and a
fresh 32-hex instance id. Use a short directory to satisfy Unix socket path
limits. Each instance uses a mode-0600 local Unix-domain socket, same-user peer
checks, bounded length-framed messages and exact version/instance checks.
The host sends an authenticated readiness frame before accepting a request. A
permanent exclusive `.claim` prevents reusing an incarnation after release,
expiry, startup failure or crash. Claims and crash-stale endpoints remain in the
caller-owned directory for inspection; choose a new id rather than replacing
an existing endpoint. Multiple datasets use separate independently owned hosts.
`network_listener: false` means no TCP/network endpoint; this local IPC endpoint
is explicitly part of this release. No live feed or provider connection is made.

When non-null, `memory_budget_bytes` bounds accounted simultaneous source-string capacity plus
decoded event-vector capacity during load; raw DBN bytes are discarded after
validation/decoding. It is **not** a process RSS cap or a budget for results,
allocator overhead, OS pages, metadata or job working sets. `pageable` selects
ordinary RAM (the OS may page it); `locked` requires successful `mlock` of the
decoded event span and fails explicitly if unavailable. OS locking rounds to
pages; this is not CUDA-pinned memory. Source admission requires one ordered
instrument; book additionally requires one publisher/channel. These format and
model contracts are separate from dataset volume. See the 0.13 resource controls below.

The caller selects 1–16 simultaneous jobs and a 1–64 aggregate inner worker
budget. An over-capacity submission is rejected without execution; there is no
hidden retry/oversubscription. Run/evaluate use caller-selected workers; book
uses one. Completed threads are reclaimed; accepted jobs share the allocation
without copying or IPC transport of event bytes. `idle_timeout_ms` is either
zero (the default: explicit release/crash) or a positive uint64 millisecond duration; active jobs prevent expiry.
Status and attempted submissions count as activity. Wire requests have the
normal bounded request size and a user-selected deadline or null (none). Disconnects do
not promise cancellation of accepted work.

The ordinary engine-process.v2 boundary remains one request/response. The
companion's `--resident-worker` entrypoint uses internal resident-wire.v1 on a
private socketpair for startup, then local IPC; it is not a stream extension to
engine-process.v2. `posix_spawn` starts the exact companion beside the current
executable, or under the same installed prefix as the SDK. The SDK needs that
companion installed; it owns no global cache inside the embedding application.
The freshly executed, single-threaded companion reparents the resident host;
the embedding SDK neither forks itself nor retains a background reaper thread.
Receipt-verified qxctl and the installed SDK can use the same handles. Native
`experiment` admits `dataset_execute` trials, preserves handle identity in its
immutable plan, and accounts nested workers. Caller policies decide scheduling;
no optimization of parameters or metrics is inferred.

All four operations are exposed as `qxctl sbv dataset load|inspect|execute|release`
with the same explicit prefix/version/input and text/JSON/NDJSON choices.
Schema/template discovery uses operation ids `dataset_load`, `dataset_inspect`,
`dataset_execute`, `dataset_release`. Execute's child schema is the selected
run/evaluate/book schema minus output_path; its ordinary output is the same
portable SBV artifact, with explicit `resources.data.dataset_feed` evidence.
The result's quantitative sections do not change merely because data was
preloaded. No claim of fastest performance, device integration, NUMA placement,
cross-host handles or restart-persistent RAM is made.


## Dataset resource authority (0.13)

SBV imposes no dataset-volume policy ceiling. File-fed `run`, `evaluate` and
`book`, and resident `dataset_load`, accept `dataset_limits` with exactly
`max_source_bytes`, `max_source_events`, and `max_metadata_bytes`. Each value is
null (no user limit) or a positive canonical uint64 decimal string. Omission
means all three values are null. `memory_budget_bytes` is likewise null or a
positive canonical uint64 decimal string; it remains required for resident load
and is optional for file-fed producers, where omission means null. Templates
select null throughout. Zero, numeric JSON values, noncanonical decimal strings,
and unrepresentable explicit limits fail. Terabyte-scale limits are accepted;
there is no smaller engine-selected maximum or automatic budget substitution.

Limits apply before admission and decoded allocation where possible. The memory
budget accounts the source string capacity including its NUL and decoded event
vector capacity simultaneously; it is not total RSS or a job working-set cap.
File loading reserves the observed file size instead of repeatedly growing the
buffer. SHA256 hashes directly from that buffer with two constant-size padding
blocks, without making another complete file copy. Allocation failures and
checked arithmetic/address-space/container representation bounds still apply.
The selected DBN format itself has finite field widths. No dataset is silently
truncated, sampled, spilled, split or relocated to satisfy a selected constraint.

The additive local reader contract is
`symphony.sqav.databento.dataset-user-limits.v1`, provided by
`FileView::inspect_dataset` / `DatasetLimits` in the selected 0.5.0-dev native
adapter. The old `FileView::inspect` / `Limits` bounded inspection and paid
historical capture/download contracts remain unchanged; no provider version,
acquisition budget, or live integration is selected by this local reader change.

Resident inspect/release evidence reports the original nullable budget and all
three selected constraints. A job may additionally select tighter admission
constraints; they are checked against the complete loaded snapshot, even if the
job only examines a small window. Omitting job controls does not modify the
original resident allocation or its load policy. Dataset feed evidence records
the reader contract and source-load controls. Capabilities advertises null
`max_source_bytes` and `max_source_events` and `dataset_limit_authority: user`.
All controls and evidence use the same qxctl, SDK and portable JSON surfaces.

This removes dataset-size ceilings; it does not implement pooled cluster RAM,
distributed loading, NUMA placement, streaming or GPU memory placement. The
current local execution host must fit the dataset and complete the operation
within a deadline only if the user selects one. Result retention,
strategy profiles, concurrency and administrative-message contracts have their
own documented limits. Removing the dataset ceilings alone is not evidence of
terabyte-scale throughput or completion. Large-host/distributed placement still requires its own implementation and measurement. Long-running synchronous requests are supported with no default deadline.


## User-controlled deadlines (0.14)

This release selects `symphony.knowledge.engine-process.v2` for the executable,
SDK request JSON and installation entry point. Its seven request fields and
response digest/identity binding follow v1, but `deadline_unix_ms` is required
and is either null (no deadline) or a canonical positive decimal string for an
absolute Unix-millisecond deadline. Finite values use signed int64 representation
below INT64_MAX; INT64_MAX is the internal no-deadline sentinel. Expired, malformed
or unrepresentable deadlines fail. There is no maximum future window or default
duration. Protocol v1 is not silently reinterpreted; earlier installed releases
remain preserved. The C SDK ABI remains v1, independently of the JSON protocol.

Every qxctl SBV engine leaf, including result export and dataset operations,
accepts `--timeout none` (default; `0` is an alias), a positive Go duration such as
`--timeout 30m`, or `--deadline-unix-ms <decimal>`. Absolute and relative options
are mutually exclusive. Relative-duration parsing has its representation limits;
the absolute form handles longer representable deadlines. Parent caller contexts
can also set a deadline or cancel. No default qxctl timer is inserted into v2.
The CLI catches Ctrl-C and cancels its child process group. This is cancellation
of that invocation; accepted work in a separate resident host may continue.

The selected absolute time is propagated through native work, experiment trials,
resident startup, IPC, dataset decoding and job execution. Resident-wire.v2 uses
null/string deadline_ms with the same meaning. Startup, handshake and release
acknowledgements have no independent fixed timeout. A client that never finishes
its frame can therefore occupy its caller-owned host indefinitely when no
deadline is selected. The user owns that availability choice. Poll intervals
are scheduling cadence, not operation timeouts. qxctl's process-group teardown
and pipe cleanup are lifecycle cleanup, not a running-job time limit.

`idle_timeout_ms` is a separate selected residency lifetime: zero is the default,
positive canonical uint64 durations are accepted without the former one-second
minimum or one-day maximum. Status preserves the user's exact value. Active jobs
prevent idle expiry. This policy does not reset a request's selected deadline.

Capabilities and the native descriptor advertise no default/max deadline and
user authority. Finite native deadlines are cooperative checks, not a promise of
preempting every blocked syscall or every compute instruction. qxctl enforces its
caller deadline on the invoked process; SDK callers own their outer scheduling.
A deadline/cancellation near artifact publication leaves an uncertain outcome:
inspect the output before retrying. No deadline means work may remain active
until completion, failure, explicit caller action or system termination.


## Native linear fitting and prediction (0.16)

`fit` and `predict` are optional native producers. Each is fully available through
qxctl, the installed SDK and local experiment trials. They use immutable result
artifacts, the existing exact source/digest/pointer boundary, and user-selected
or absent deadlines. No external runtime, live source or model service is needed.
This baseline does not replace external models or make a linear estimator a
platform-wide requirement.

Input columns declare JSON Pointers, `integer` or `rational` values, and explicit
unit strings. Features also have unique IDs and an ordered list. Observation IDs
are unique nonempty strings in a caller-defined `identity_namespace`. A supplied
target has its own pointer/type/unit. Missing feature, target or weight fields
follow `reject` or `exclude`; present malformed numbers always reject, including
in a row missing a different selected field. Missing/duplicate/malformed IDs
always reject. Null is a present malformed numeric value, not a missing field.
No scaling, centering, imputation or feature selection happens implicitly.

Fit selects an intercept, `uniform` weights (exactly one per retained row), or
`supplied` exact rational weights. Supplied weights are never normalized.
`nonnegative` rejects negative weights; `signed` explicitly admits them. The
regularization object supplies one nonnegative quadratic penalty per feature in
feature order and an intercept penalty; no intercept requires a zero intercept
penalty. Zero penalties select the unregularized calculation. All these fields
are mandatory choices; there is no hidden regularizer or optimizer stop policy.

For design matrix X (intercept first when selected), target y, diagonal weights W
and selected diagonal penalties L, native code solves `(X^T W X + L) beta = X^T W y`.
The corresponding objective is `sum(w_i * (prediction_i - y_i)^2) +
sum(lambda_j * beta_j^2)`, without division by sample count. With nonnegative
weights/penalties and a unique solution this is the quadratic minimizer. Signed
weights define stationary equations and an algebraic objective; a unique solution
may be a maximum or saddle. Neither kind supplies a calibrated probability.

The numerical profile is checked exact signed-128-bit rational arithmetic, with
canonical string numerators/positive denominators. Deterministic Gauss-Jordan
elimination chooses the first nonzero row in each column. No pivot tolerance,
pseudoinverse, precision downgrade or silent column removal is applied. The
reported rank belongs to the regularized normal system, not necessarily the raw
design matrix. Empty retained training data, inconsistent equations and
non-unique solutions follow `on_unsolved: reject|unavailable`. The unavailable
choice persists a normal result with an unavailable model and explicit reason;
no fabricated coefficients are emitted. Overflow always rejects before publication.
A zero-feature intercept-only model is supported, as is a user-selected
zero-feature/no-intercept zero predictor on nonempty training input.

Available models use `symphony.sbv.linear-model.v1`, with method
`penalized_linear_normal_equations_v1`, declared feature order/units and target,
intercept, coefficients (intercept first), penalties, weights and exact training
source digest/pointer/namespace/retained IDs. Fit diagnostics retain residual,
absolute-error and squared-error weighted sums, algebraic means, penalty and
objective values. Zero weight total leaves means unavailable. Optional normalized
training prediction rows contain the original source index, ID, target,
prediction, residual and weight; original feature/source rows remain bound by
digest. Coefficients have target/feature units; intercept has target units.

`predict` consumes a digest-bound model at an explicit pointer, validates its
supported shape and numerical values, and evaluates it in C++. This validation
does not authenticate authorship or recompute a supplied model's fit. Prediction
feature mappings may be reordered or use different source pointers/numeric
representations: matching IDs and unit strings resolve them into trained order.
Missing/extra IDs or mismatched units reject. Target is optional; when supplied,
its unit must match the model target unit. Targets never enter the prediction
formula. There is no automatic clipping of predictions to probabilities or any
other domain.

Prediction weights have the same explicit semantics and affect error studies,
not predicted values. Select no studies or `regression_errors` v1. The latter
reports count, algebraic weight total, weighted residual/absolute/squared-error
sums and means divided by algebraic weight total. Signed-weight absolute/squared
sums or their means may be negative; they are not probability MAE/MSE. Zero total
weight leaves means unavailable; absent targets make the selected study
unavailable while prediction rows remain useful. These diagnostics do not claim
predictive validity or independent observations.

The arbitrary nonempty `purpose` string is preserved as user context, not enforced
holdout authority. When training and prediction identity namespaces match, the
result lists retained prediction IDs also present in the model's retained
training IDs, with exact prediction source indices and reuse count. Different
namespaces yield unknown overlap (null list/count), never a fabricated zero.
Same namespace and no matched IDs establishes only no matching declared IDs;
it does not rule out shared events, feature lookbacks, labels or prior selection.
Exploratory reuse is allowed and visible. Repeated calls do not create a global
access ledger or claim that a holdout remains untouched.

These operations can be composed with retained train/test rows from `split`,
including combinatorial pages; their outputs also feed `analyze`, `compare` and
external pipelines. Local `experiment` admits both operations with already
resolved input/model references; 0.17 adds explicit dependency binding below. Automatic nested fitting,
stitched performance paths and cross-run holdout-access history remain separate
work. Existing operation identity and immutable journal rules still apply.

There are no new fixed row/feature policy caps. Materialized inputs/outputs remain
subject to the existing 128 MiB/four-million-value result representation, request
limits and host address space. Matrix storage is quadratic and elimination cubic
in selected parameter count; exact rational intermediates may overflow even when
the final mathematical value would fit. Users select data representation and
resources or another implementation; there is no silent approximation. This
first baseline uses one native worker per fit/prediction; independent trials may
run through the existing threaded experiment runner. It is not a GPU, large-dense
solver or competitive-performance claim.

## Explicit local trial dependencies (0.17)

`experiment` optionally accepts `depends_on` and `bindings` on each ready trial.
Omitted arrays mean no dependencies. Each dependency names a unique other trial
in the same plan, including trials later in the input array. All IDs, edges,
binding destinations and acyclicity are validated before creating the journal.
Pruned trials keep null operation/request and have no dependencies or bindings.
A ready trial may depend on a pruned trial; it becomes blocked.

Each binding has `target_pointer`, `trial_id` and `result_pointer`. The named
trial must also appear in `depends_on`. The destination is an existing null slot
in the consumer request, addressed by RFC6901; the request root, assigned output
path, existing values and duplicate/overlapping destinations cannot be replaced.
The source pointer is RFC6901 within the parent's complete result (empty selects
the root). Before the consumer is claimed, the engine checks the parent's receipt,
content hash and exact file bytes and verifies the source selection exists. It
replaces the null slot with `{path, expected_sha256, pointer}` referencing that
immutable result. It does not copy the selected data or infer compatible units,
feature meanings or an estimator. The selected producer validates its request as
usual. A selected null value is present; its consumer decides whether it is usable.
These reference bindings support native split → fit → predict compositions and
other producers whose input accepts this reference object. Arbitrary expression
evaluation, scalar substitution and implicit translation of other input shapes
are outside this release.

Scheduling uses stable topological waves. At each wave, ready graph nodes are
ordered by the caller's trial-array position; up to `workers` execute concurrently.
A wave joins before the next starts, so dependencies always precede consumers.
The ledger retains the caller's original order. `planned_waves` records the
schedule, `wave_count` its length and `actual_outer_workers` the maximum admitted
wave concurrency (not measured utilization). With `on_failure=stop`, one worker
is required; the first noncompleted executed node stops remaining claims in this
stable topological order. Continue mode executes independent branches. The prior
trial/worker policy maxima have been removed in 0.24; per-child framing and caller budgets remain explicit.

A dependency in failed, pruned, ambiguous or blocked state blocks its dependents.
An absent or invalid selection also blocks before claim. Blocked rows have no
request hash, journal or result; dependency receipts and any already resolved
bindings remain inspectable. They are observations, not claimed/failed runs.
Unexpected journal files for a blocked trial reject reconciliation. Descendant
blocking propagates transitively; independent work continues under continue mode.
A completed dependency means its native operation committed an artifact, not that
all its studies/model fields are available or statistically valid. Result status
is retained in its receipt. No automatic quality threshold or holdout authority
is imposed.

Every claim records the full resolved request, its hash, dependency states/receipts
and resolved bindings. Completion receipts retain plan/request correspondence.
On repeated invocation the same graph is resolved and every completed ancestor is
verified before dependent claims/receipts can be reused. No completed trial is
reexecuted. An incomplete claim remains ambiguous; its descendants are not run.
A separately identified experiment is the explicit retry path. Version/plan
changes still require a separate directory. This is local artifact composition,
not distributed scheduling, nested cross-validation orchestration, automatic
search-space expansion, optimizer selection or a cross-run holdout-access ledger.
No default deadline or dataset capacity policy was added. All graph records,
claims and results use the existing qxctl/SDK result contract for terminal and
future external/GUI consumers.

## Supplied research usage and selection history (0.18)

`research_history` (`qxctl sbv research-history`) creates an immutable history
result from caller-selected fit/prediction and comparison artifacts. It accepts
ordered `entries` and independently ordered `selections`, explicit duplicate
counting/unavailable-identity policies, observation retention, and the caller's
ordering/coverage descriptions. No filesystem discovery, authenticated chronology,
complete access capture, automatic holdout classification or promotion gate is
implied. All twenty-seven native operations are exposed through thirty qxctl leaves.

Each entry specifies a unique nonempty `id`, `kind` (`fit` or `predict`), a source
reference, arbitrary nonempty `role` and extension object. References contain
`path`, `expected_sha256` and RFC6901 `pointer`: the path/digest bind the complete
outer SBV result; the pointer selects either its root or a nested complete SBV
result. Both content digests are verified, and outer file bytes/digest/selection
identity are retained. The selected choices protocol must match the declared kind.
Native-style source profiles are structurally checked; hashes do not authenticate
that a particular executable produced them, and estimates are not recomputed.

An available fit contributes its model's retained training IDs. This works when
training predictions were not retained. Model, choices and source provenance must
agree on namespace/source coordinates and the training count. An unavailable fit
has unknown IDs, not zero IDs: `on_unavailable=retain` records null counts and
partial history status; `reject` refuses publication. The engine does not reread
original observations to infer unrecorded usage. Predictions contribute retained
prediction-row IDs; target availability must agree across choices, summary and
rows. IDs are unique within an entry. Excluded rows are not counted, and retained
zero-weight rows are counted. A prediction's inherited model training reference
is preserved as attribution, not counted as an additional fit event.

Identity comparison is exact on `(identity_namespace, observation_id)`. Different
namespaces are not compared; they are not presumed causally disjoint. Entry order
comes from the supplied array, not source timestamps. Each counted entry reports
its number of distinct earlier overlapping entries, reused observations,
previously fitted observations, previously predicted observations and observations
whose targets were previously supplied. These are observation counts, not weight
mass, exposure probabilities, actual physical reads or evidence of human viewing.
A targetless prediction still counts as prediction use, but not target exposure.
All counts are exact integer strings. Roles such as `holdout` remain user labels.

`duplicate_artifacts=collapse` counts only the first entry for a selected result
content digest; `count_entries` counts every supplied entry. All records retain
`duplicate_of`, and collapsed records retain their available observation count
with null reuse counters. A repeated listing does not prove repeated execution.
Totals distinguish entries, counted/collapsed/unresolved entries, observation
occurrences, unique/reused namespaced identities and namespaces with counted IDs.
Unknown identities prevent any complete-coverage interpretation of those totals.

`retain_observations=true` includes per-entry reused IDs/prior entry IDs and a
lexicographically ordered per-identity occurrence ledger. False omits those two
expanded projections while retaining the same aggregate counts. It does not
redact source model-training references, selection records or artifact paths.
The original result artifacts remain independently accessible through qxctl.

A selection specifies unique `id`, comparison source reference,
`chosen_candidate_ids`, nonempty `reason` and extensions. Candidate choices must
be unique and present in that supplied comparison; an empty or multiple selection
is allowed. The record retains every comparison candidate, objectives, methods,
weighted order and Pareto fronts, plus exact chosen rows. Selecting a lower-ranked,
excluded, failed or pruned candidate is allowed and disclosed. This records a
caller decision against immutable comparison evidence; it does not calculate a
new ranking, claim execution of failed/pruned candidates, infer a link to usage
entries or authorize deployment. The two input arrays have separate caller orders.

The operation is a local, single-worker result transform. No new entry, observation
or read-total policy cap is imposed; request/artifact representation, host address
space and checked byte-accounting bounds still apply. No default deadline is
introduced. Supplied references can bind history as a dependent native experiment
trial; ordinary immutable claim/receipt and no-repeat reconciliation apply. The
same result is inspectable/queryable/exportable through qxctl and the installed
SDK for future GUI and external research consumers. Automatic nested fitting,
complete cross-run access capture and causal leakage inference remain separate
unimplemented capabilities.

## Retained census and source-bound economic continuity (0.19)

`run` and `evaluate` now retain `sections.census.data` using
`symphony.sbv.census-evidence.v1`. Native identity remains the digest of the exact
native signal array, with criteria and selection context retained separately.
External identity remains the digest of the complete original external census
declaration, including producer, causal mode and original signal fields. Model
selection cannot convert one identity domain into the other.

`evaluate.census` accepts either the original inline external declaration or
`{path, expected_sha256, pointer}` referencing an entire root or embedded sealed
run/evaluate result. The outer and selected seals, duplicate declarations,
source digest, dataset, instrument, source ordinals and timestamps are checked.
Native anchors and previous observed trade prices must match the decoded source.
The normalized evidence is self-contained: another evaluation can consume it
without reopening the first result's ancestors. Legacy native run and inline
external evaluate results without the new section have strict adapters; an
invalid present census section never falls back to legacy interpretation.
Hashes establish content identity, not authorship or proof of external causality.

`economics` accepts its legacy `path` plus `expected_sha256`, or the alternative
`source` reference triple. Mixing those forms rejects. Both select complete
model evaluation results; legacy touch observations are not reinterpreted as
price-support models. Original census evidence, selected model/signal rows and
full source replay remain retained. Source references record both the enclosing
file and the selected result identity. `book` shares census/source validation.

`compose_economics` (`qxctl sbv compose-economics`) consumes immutable economics
results. Each component selects a source result, economic row pointer, signal
identity and conditioning; no numerical support needs to be copied into a new
request. Admission checks the retained census, selected model and transform
correspondence, and exact economic arithmetic. Original assumptions, replay
references and separate content/file identities remain inspectable.

The user chooses heterogeneous independent draws, or explicit joint paths with
one source atom index per component and a probability for the whole path.
Independent masses are products. Joint masses are used once, never multiplied
again by source weights. `require_source_match` requires exact induced marginals;
`support_only` explicitly selects a new joint probability measure while preserving
original source measures and weights. No missing mass is normalized and no
unavailable component is silently dropped. The user selects whole-distribution
unavailability or rejection for unavailable/incompatible outcomes.

Economic state evolves in declared component order by multiplicative returns or
additive P&L with explicit unit conversions. Initial state, unit and signed versus
nonnegative state domain are user choices. Terminal distributions are retained;
full paths and prefix distributions are optional. Zero-probability paths and
atoms remain present. Zero studies is valid; selected `terminal_moments` computes
exact mean/population variance and `terminal_quantiles` selects inverse-CDF
quantiles over positive mass. These describe outcome distributions, not parameter
confidence or calibrated execution. Compounding does not establish shared
liquidity, capital, reinvestment, FX correctness or feasible account execution.

Checked signed int128 rationals reject unrepresentable intermediates without
rounding. Optional `max_paths` and `max_terminal_atoms` are caller limits; null
adds no policy limit. Host/address-space, numeric and existing result-artifact
representation bounds remain explicit. This increment does not resolve the
separate scalable-result work package. No default deadline is introduced.

Experiment bindings can now connect run → evaluate → economics →
compose_economics using retained references, with the existing digest-bound
journal/resume contract. Installed schemas/templates, catalogue, complete
terminal text/JSON/NDJSON and the calculation SDK expose the same data. This
release connects the core workflow; milestone acceptance remains a separate gate.

## Independently compiled native strategies and models (0.20)

`generate_census` is a separate first pass. It accepts the same verified Databento file or resident RAM dataset, an explicit `native-provider-selection.v1`, and a fresh output path. `provider_inspect` loads the selected trusted library and returns its descriptor and retained identity without creating a strategy/model instance. Inspection may execute platform library initializers; it is not inert file parsing. Its host effect is an inspection response, while selected trusted code may have other effects. It is declared non-idempotent and must not be blindly retried. Both operations and the `evaluate.model.id=native_provider` alternative are available through installed qxctl and the C/C++ invocation SDK.

The installed plain C `symphony/sbv/provider.h` defines ABI version 1, opaque instances, explicit-length borrowed views, fixed-width MBO fields, function tables and provider-owned output buffers. Export `symphony_sbv_provider_api_v1`. The separate header-only `Symphony::SbvProvider` target has no engine or vendor dependency. Native authors compile for the host platform and may choose their own dependencies. There is no mandatory provider registry or Python runtime.

The caller selects library path and exact SHA256, provider ID/version, role, input profile, concurrency, parameters and dependency disclosure. The host verifies source bytes, copies them into a private directory, verifies the staged bytes again and loads with local symbol visibility. This first POSIX profile uses `/private/tmp` on Apple and `/tmp` elsewhere; a no-execute filesystem can prevent loading. Loader-relative dependencies resolve from the staged location. Other dependencies follow the platform loader, and separately verified declared artifacts do not prove which dynamic libraries were loaded. Native initializers, callbacks and any provider I/O remain trusted user code. No sandbox, crash isolation, authentication or safe preemption is claimed.

Descriptor bytes and their SHA256 are retained verbatim as `descriptor_json`, allowing numeric JSON Schema keywords while the surrounding result keeps its exact numeric-string contract. Retained descriptor strings share the existing 65,536-byte result-string representation; this is distinct from raw library/dataset size. Parameters must be an object; their semantic validation belongs to the provider. Provider-selection parameters and extensions retain nested objects, arrays, strings, booleans and null; numeric values use strings under the existing portable-result contract. Raw JSON numbers anywhere in those selected objects reject before opening the library or invoking native code, rather than after backtest execution. A descriptor's configuration schema is discoverable documentation, not a claim of generic host JSON Schema validation. Selected artifact digests and reproducibility declarations are evidence of the caller's composition, not proof that undisclosed dependencies or mutable external state are absent.

One strategy instance receives every original MBO event in source order. All decoded fields, including event/receive clocks, sequence, flags, order ID, publisher, channel, `ts_in_delta` and optional `ts_out`, remain distinct. Its active prefix reader permits only `[0,current_ordinal+1)`. Returned candidates contain ID, anchor and context reference; the host supplies source ordinal, receive-time availability and causal end. Multiple candidates at the same event preserve provider order. IDs must be unique; undefined anchor prices reject. Finish returns diagnostics/extensions, never retroactive signals. This operation adds no policy signal-count limit; current artifact/address/arithmetic representations remain separate constraints pending scalable-result work.

The host seals `native-provider-census.v1` with exact source, selection, descriptor evidence, signals and completion data. Normalized census evidence has kind `native_provider`, identity domain `native_provider_declaration` and mode `native_provider_prefix`. These describe the host's exposed inputs; a trusted library can access other resources, so the engine does not attest to complete causal isolation. A retained generated census feeds evaluate/book through the same source-bound admission as other census kinds, and evaluated results feed economics and source-bound composition. The original provider library is not needed merely to read or reevaluate an already retained census with another model.

A native model receives the complete admitted signal, causal MBO prefix and later source ordinals through the caller-selected inclusive receive-time horizon. Equal-time events after the signal remain follow-up. Raw MBO does not establish initial book state or source completeness. Candidate support, exact weights, independent execution probability and evidence reference pass through native typed-model admission. Probability, scenario weights and signed coefficients keep their existing distinct semantics. No renormalization or calibration claim is introduced. `UNAVAILABLE` follows the caller's `on_unavailable` choice; errors and malformed candidates reject publication. Full attributed error JSON is retained in the unavailable reason when selected.

Strategies use `serialized_instance`. Models may declare and select `serialized_instance`, `per_worker_instances` or `shared_reentrant_instance`. Serialized evaluation follows census order with one worker; parallel modes retain census output order but use dynamic assignment and may invoke concurrently. Per-worker instances do not promise stable worker assignment, and provider hidden/global state can make results depend on scheduling. Requested/applied workers, selected topology and instance counts are retained. Zero signals create zero model instances. A declaration of reentrancy is the provider's responsibility, not a host proof of thread safety.

Contexts, event readers, input frames and host callbacks are valid only during the current synchronous invocation on its invoking thread. Providers copy data they retain and must never call an expired context. Active range/thread checks do not make stale pointers safe. Provider-owned output/error buffers have one provider release callback; the host copies and releases each published buffer on success and rejection, destroys instances after callbacks finish, and unloads last. Exceptions must not cross the C ABI. A conforming failed create cleans up and returns no instance. If a failing callback nevertheless publishes a handle, the host defensively destroys it before reporting failure.

No deadline exists unless the caller selects one. Providers may poll cancellation and deadline state during an active callback; the host also checks between callbacks. A noncooperative callback cannot be safely interrupted inside the SDK process. qxctl caller cancellation retains existing child-process cleanup semantics; it cannot undo provider side effects. No implicit retry or model fallback is performed. The source corpus, replay selections and all retained authoring evidence remain accessible through the same result inspect/query/export contract for terminals, future GUIs and external AI/ML consumers.

## Optional offline retained sources (0.21)

`source_retain` imports a caller-selected exact uncompressed Databento DBN file into the existing owners. The user supplies source SHA-256/dataset, actual capture description and acquisition-time evidence, access-evidence references, all owner limits, exact create/open store choices and a new result path. SBV calls the SQAV capture/metadata factories, prepares an actual SQFV batch, appends to SQPV and retains the actual receipt. Original DBN bytes and every MBO field remain unchanged. Declared acquisition/access evidence is not a new provider authorization or inferred download timestamp. Exact open/retry preserves original metadata/options/sequence; duplicate append is explicit.

The discoverable `source_owner_profile` selects `sqv-local-retained-capture.v1` on local macOS APFS. Its maximums belong only to this optional composed route: 64 MiB capture/DBN, 1,048,576 DBN records, and the exact owner limits published in capabilities. Root/partition/delivery fields are at most 4096 bytes for SQDV compatibility. These are existing owner-profile constraints, not global SBV dataset limits. Ordinary file/RAM input retains nullable caller-selected volume and memory bounds. A wider store/provider route requires its own capability contract; it must not silently change this one. No default deadline is introduced.

Run, evaluate, book, generate_census and dataset_load select exactly one source: the complete original file tuple or `retained_source` (exact result reference plus explicit delivery choices). The latter reconstructs the canonical SQMV manifest from retained typed evidence, reopens actual storage and verifies receipt, batch/capture identity and original bytes. The historical original pathname is an identity label and need not exist. Actual retention must contain the recorded prefix; same-prefix snapshots must match exactly. After legitimate growth, `historical_within_current_prefix` records the limited historical check and the actual current confirmed snapshot.

The adapter borrows the delivered original span until Dataset owns the complete verified MBO decode and has satisfied selected memory, residency and deadline checks. Only then does `processed_ack:dataset_admitted` acknowledge that admission; `none` is also available. Failures before successful Dataset admission and destructors do not acknowledge processing. Later job, resident-startup or publication failures do not undo a completed admission acknowledgement. Explicit before/after states, actual receipt, view reference, selected acknowledgement, released handles and checkpoint evidence are retained. This profile accepts only `checkpoint:null`; it reports no durable checkpoint, exactly-once processing, source authorship or destination-commit guarantee. The load-buffer budget covers original-byte span plus decoded event allocation, excluding owner capture/batch/store/delivery allocations; it is not process RSS.

RAM loads retain the same admission evidence. Jobs bind either the exact retained selection or the matching original tuple and perform zero new original reads/decodes. Descendant economics, economic composition, liquidity and research transforms preserve optional dataset-feed evidence. Pure result consumers validate its correspondence without reopening unavailable source artifacts or stores; that does not authenticate original bytes or source authorship again.

`source_export` selects unchanged `original`, `capture` or canonical `manifest` bytes with explicit binary and result-receipt destinations. It requires `processed_ack:none`, records exact byte count/SHA-256 and source/capture/manifest/batch/retention identities, and publishes without replacement. Destinations inside the selected store are excluded by opened no-follow directory ancestry (device/inode), including filesystem case/Unicode aliases. No binary is automatically imported into another system.

Source-retain/export can return a closed `status:recovery_required` payload when persistence is confirmed but result publication fails, or an owner outcome is uncertain. It contains the canonical request digest, exact destinations, fixed code/stage and typed reopen or expected-binary evidence. Confirmed retention includes the actual receipt; uncertainty never invents one. Binary confirmation distinguishes binary-publication from receipt-publication failure. `automatic_retry` and `rollback_performed` are always false. Native process/SDK success here means the evidence was returned; callers must inspect domain status. qxctl validates request correspondence, renders the entire payload in text/JSON/NDJSON and exits 5 without a second generic error. The caller controls reconciliation and any retry. Unknown/pre-mutation failures retain the existing generic error contract.

Every operation and result is available through the same native dispatch, exact installed SDK and qxctl discovery/result projection. These changes do not perform HTTP/live acquisition, imply entitlement, calibrate fills, or qualify underlying owner crash/power-loss behavior. Full programme acceptance and large derived-result scalability remain separate gates.

## Resource admission — experimental 0.24

Worker requests are positive exact uint64 counts with no engine policy maximum. Applied workers depend on eligible tasks and the selected provider's topology; serialized state remains serialized. A request is not an instruction to allocate that many threads before task admission. Thread-launch failure stops scheduling, joins started workers and preserves the operation's existing failure/recovery semantics.

Resident `max_concurrent_jobs` and `worker_budget` accept positive user limits or null. Null selects no policy ceiling; uint64 accounting and host allocation remain real constraints. Slots grow only when a job is admitted. Capacity is rejected without an implicit work queue; the Unix listen backlog is transport buffering, not a job limit. Resident worker accounting reserves requested job workers, not measured operating-system threads or total RSS. Dataset memory budgets cover declared load buffers, not all derived caches, indexes or process RSS.

Experiment workers/trial count no longer have 16/64 policy maxima. Optional `worker_budget` limits the exact checked outer-request × maximum-inner-request product; null/absence has no selected policy ceiling. Inline plans retain the control-frame profile; referenced plans and partitioned aggregates are also implemented below. Ordered stop-on-failure still requires one outer worker.

Backend planning removes the hidden 64-worker clamp and accepts nullable declared worker/memory limits. The installed C++ `BackendRequest` layout remains unchanged; additive `BackendRequestV2` and `plan_backend_v2` express optional limits. Both entry points preserve selected reject/reduce policy and explicit serial dependency constraints. No reservation or CUDA/tensor execution is implied.

This resource increment does not remove remaining legacy scientific-consumer limits. Those remain separately identified in the R3.3 operation/profile audit and are not global limits on partitioned SBV results.

## Partitioned experiment children — experimental 0.24

The current experiment contract accepts explicit partitioned outputs for run, generate_census, evaluate, economics, compose_economics and dataset_execute. Each such trial supplies `result_read_options` (`max_page_bytes` nullable, `cache_bytes` exact unsigned decimal). Legacy trials still omit output_path and use their deterministic journal filename. The partitioned template demonstrates forward-compatible result bindings through run → evaluate → economics.

Before claims, destination parents must exist and pass no-follow traversal. Filesystem-resolved aliases, nested partitioned destinations and collisions with journal files are rejected. A private sibling probe checks not-yet-created case/Unicode aliases using the destination filesystem's actual rules; only that owned probe is removed. Cooperating callers must keep parent directories stable.

Completed children are reused only after operation, requested output, exact file or manifest identity, complete logical closure, status and summary verification. Bindings retain logical selections and caller read options without loading aggregate arrays into control requests. Original market files, released RAM handles and model ancestors are unnecessary for completed-child reconciliation. Missing selections block dependents without a claim.

A child recovery_required payload is retained verbatim and bound to its exact resolved request. Its trial state is recovery_required, the experiment is partial, and dependents remain unclaimed. Re-observation retains recovery without retry or adopting an appeared manifest. Claim-without-completion remains ambiguous. The later referenced-plan increment below supplies an owned plan table and partitioned aggregate; per-child journal records remain bounded.

## Typed views and research consumers — experimental 0.24

`result_select` preserves the legacy inline page request and adds two explicit variants. The `partitioned` template supplies `source` (legacy result reference or tagged bundle result), `pointer`, filters, ordering, columns, `ordering_storage:memory_index`, nullable `max_index_rows`, and partitioned `output`. Preparation verifies the whole source, filters typed rows, orders selected keys with original source index as the exact tie break, and writes a self-contained immutable view. An unordered selection streams rows directly. Ordered selection holds typed sort keys and pinned row handles in memory; `max_index_rows` constrains that index only. There is no aggregate source-row ceiling or automatic spill algorithm.

The `view_page` template supplies the exact prepared-view `view` reference, read options, positive page `limit`, and string cursor. Each continuation verifies the pinned manifest and accessed page/subtree closure, reports actual file/cache counters, and reads only the selected range. It does not reopen, filter or sort the source. The cursor binds the full paging request excluding the cursor itself, including the view's physical and logical hashes. Changing page size or read options starts a new cursor sequence. Control pages retain their byte/value envelope; a large retained row is always reachable through bundle query/export. Page verification is explicitly narrower than `bundle_verify`'s full closure. Source authorship is never inferred from hashes.

`analyze`, `resample`, and `split` independently select legacy `path`/`expected_sha256` or a `source` result reference, and legacy `output_path` or partitioned `output`. Their partitioned templates expose every choice through qxctl and the identical SDK protocol. Full source closure precedes typed row admission. The legacy JSON representation still has its explicit 128 MiB/four-million-value envelope. There is no separate engine row-count ceiling. Output never silently changes representation, drops observations, truncates indices or changes weighting.

Analysis retains a typed sample vector and, when selected, a sorted quantile copy; series, exclusions and drawdown curves stream into result pages. Bootstrap retains a source-value vector, one exact mean per replica and lightweight pinned index handles; per-worker index arrays and final replica rows stream. Completed replicas release their warm decoded-page caches, and final copying opens one selected reader at a time. The SplitMix64 replica seed and draw-index sequences are unchanged. Temporal splits retain source interval/index metadata, the selected fold plan and per-fold interval unions; result indices, exclusions, groups, folds and retained rows stream. No source 65,536-row, 1,024-replicate, 65,536-sample/index, draw-product 1,048,576, 128-fold, 131,072 row-fold or 64 MiB retained-split policy remains. Exact checked numeric and host-addressability limits still reject unrepresentable work. These are CPU algorithms; typed input/index vectors, selected plan metadata, sort keys and interval unions remain memory resident. Their memory descriptions are algorithm scope, not a process RSS reservation.

Partitioned outputs optionally accept `scratch_read_options:{max_page_bytes,cache_bytes}`. Omission preserves unlimited additional page size and cache zero. This is the maximum canonical-page-byte cache **per active independent scratch reader**, not total RSS; parsed JSON, typed indexes and allocator overhead are separate. Multiple simultaneously retained arrays/workers can each own a reader. Caller-supplied source read options remain independent. Cache selection does not weaken page hashes, full-closure admission, immutable publication or recovery evidence.

The current matrix covers the implemented 0.25 read/write profiles. Typed view preparation writes partitioned results and its page form is a selected-range reader. Experiments support producer children, referenced plans and partitioned aggregate ledgers. The following sections describe execution, model and experiment consumer details. Selected R3.3/R4 acceptance belongs to the sealed 0.24 qualification packet. The separate 0.25 control-hardening checks do not confer a cluster/GPU runtime or comparative fastest-engine claim.

### 0.24 execution-chain transport increment

`book` reads either a legacy sealed census/checkpoint reference or an exact bundle selection. Its signal selection accepts the original ID array or `all`, `range`, `ids`, and `referenced_ids` forms. `frames.maximum` is a caller-selected positive count or null. No separate census/frame/checkpoint-order policy ceiling remains. Display depth 1..64 remains part of the named `databento_mbo_orders_strict_v1` profile; full checkpoint state is independent of display depth. The current state maps and signal/window indexes reside in host memory.

`liquidity` and `allocation_economics` each independently accept a legacy result or a full-closure bundle result, and legacy or partitioned output. Order/valuation selections accept inline arrays or `{kind: referenced_rows, source: <bundle value source>}`. Selected input rows and their exact reference evidence are retained. Allocation can validate retained order declarations without reopening the original order bundle. Liquidity keeps typed frame snapshots, order choices and an ID index in memory; allocation keeps typed allocation metadata and owned source-row handles. Emitted outcomes and study rows stream through private pages. Shared-snapshot execution remains serial and all-active; independent workers retain stable input order. Displayed-depth-sweep v1's quantity 1,000,000,000 and depth/fill 64 bounds remain explicit finite model choices.

Partitioned `book` is admitted by resident dataset execution. Experiments accept partitioned children for all three operations with explicit result read options. The engine/SDK control frame still bounds each inline request; referenced arrays carry larger selections without enlarging that frame. A selected scratch cache applies to each active reader, not total process RSS. Complete-profile qualification and installation remain required at R4.

### 0.24 model, comparison and research-history transport

`fit` and `predict` accept their legacy value references or full-closure bundle selections independently of legacy/partitioned output. Bundle selectors can be pointers or exact source-bound node IDs. For a node-ID selection the convenience `source_pointer` is null and `source_selection` retains the exact admitted selector and source identity. This does not assert that an empty pointer selected the rows. Training IDs, exclusions, predictions and overlap records stream into owned result pages. The solver still retains typed samples and a dense exact normal system in host memory: O(rows × features + features²), plus IDs and output scratch. Exact checked 128-bit rational arithmetic and rank/unsolved policy are unchanged. Prediction retains the selected model's training evidence; it need not reopen that model's training data.

`compare` accepts inline or referenced candidate/objective arrays. A completed candidate can retain its legacy path/hash identity or select an exact result with `source`; a failed/pruned candidate supplies null identity and its reason. Full source closure precedes objective selection. No 128-candidate, 16-objective or 256 MiB aggregate comparison-source policy remains. Weighted ordering retains stable competition ranks. Pareto comparison retains typed objective vectors and a dominance graph (quadratic in eligible candidate count in the worst case). These are disclosed CPU/memory costs, not hidden sampling. Ranking, provenance, objectives and lineage arrays are independently queryable in the resulting bundle.

`research_history` accepts inline or referenced entry/selection arrays and legacy or bundle fit/predict/comparison sources. It records exactly the supplied order, duplicate policy, namespaced observation reuse, selected comparison candidates and user labels. Retained IDs, usage entries, comparison context and selected evidence are copied into owned output pages, usable after ancestors are removed. The observation-occurrence index remains resident in memory. No complete access history, causal independence or promotion authority is inferred. For comparison/history calls using bundle references, `source_bytes_read` is null because these operations do not collect aggregate physical I/O; `legacy_source_bytes_read` and its scope identify the measured subset rather than reporting logical bytes as disk reads. All four operations are partitioned experiment children with explicit result readers.


### Referenced experiment plans and partitioned aggregate ledgers (0.24)

`experiment` accepts inline trials or a `referenced_rows` bundle selection. Referenced trials require explicit `plan_output` and `plan_read_options`: the engine verifies the complete source closure and publishes an owned immutable trial table before creating any child claim. Its receipt becomes part of the sealed journal plan. Resume verifies that owned table against the original selection and exact choices; it does not reopen the original trial-table source. Caller-selected aggregate output may independently be legacy or partitioned. Partitioned aggregate results retain the complete trial plan, selection identity, ledger and stable waves, remaining readable after the journal and plan store are removed.

Scheduling uses dependency counts and reverse edges, with caller-order ties within topological waves. Trial records, child control requests, dependency graph and observed receipts occupy host memory; the referenced-plan route is not a constant-memory scheduler or distributed execution claim. Per-child requests, claims and receipts retain their existing bounded control/legacy journal profiles. Saved-view preparation is also admitted as a partitioned child.

An incomplete owned-plan publication is retained in `experiment-plan-recovery.json` when that record can be written. No child is claimed before a committed plan. Existing recovery or an uncommitted appeared destination requires explicit caller reconciliation; no automatic adoption, cleanup or retry occurs. A deadline/process failure can prevent writing the recovery record, leaving the private workspace as uncommitted evidence. Exclusive journal locking revalidates the admitted plan before scheduling. All output/workspace/journal paths use the existing alias/overlap admission.


### Caller-defined executable study formulas (0.24)

`analyze.study_definitions` admits descriptors with protocol `symphony.sbv.exact-series-formula.v1`, caller-selected `id`, `version`, `unit`, `description` and `expression`. Select their IDs in `studies` alongside any builtin studies; empty selection executes none. Descriptors need no platform registry or CLI changes. Duplicate IDs or collision with a builtin meaning reject. Get a complete example with `qxctl sbv template --operation analyze --variant formula`. Definitions, exact outputs, declared units/authorship and unavailable reasons travel in the same legacy/partitioned result contract for qxctl, SDK and future GUI consumers.

The C++26 interpreter supports exact rational `constant`, aggregate `count`, row `row_value`/`row_weight`, reductions `sum`/`minimum`/`maximum`, unary `negate`/`absolute`, and binary `add`/`subtract`/`multiply`/`divide`/`min`/`max`. Binary nodes name `left` and `right`; unary/reduction nodes name `value`; constants carry an exact rational `value`. Rows are the admitted retained sample vector after explicit missingness/weight selection. Row variables are legal inside one reduction; nested reductions and aggregate count inside a reduction are outside this named profile. The published recursive expression schema and native validator define exact operator fields; qxctl performs bounded structural preflight before native semantic admission. No expression text is executed as code.

Supplied weights are used only when the formula selects them. An empty sum is exactly zero; empty extrema and division by zero are unavailable or reject according to `on_incompatible`. No finite stand-in or hidden normalization is used. Thus users can explicitly define profit factor, weighted sums, centered moments or other compatible formulas without adopting a mandatory metric catalogue. Units and economic interpretation are caller declarations, not dimensional inference or portfolio validation. Arithmetic uses checked signed-128-bit rationals; overflow rejects, never changes methods. Formula traversal checks the user's deadline. Current control/result depth and scalar representation apply; there is no additional study-count or operation-time policy ceiling. This scalar formula profile does not claim arbitrary Python execution, native study-plugin ABI or a tensor runtime.


## Resident control hardening (0.25)

`dataset_execute` accepts an optional `job_id` (1–128 ASCII letters, digits,
dot, dash, underscore). Omission uses SHA256 of the canonical outer request.
An admitted ID is never reused in that resident incarnation, including after
failure. Admission failures do not consume an ID. A disconnected caller does
not cancel accepted work. Preserve the selected ID before submission and use
`dataset_jobs` to reconcile an uncertain response; never infer non-execution
from a disconnected or terminated qxctl process.

`dataset_jobs` lists admission-ordered job records with `after_sequence` and a
caller-selected positive `limit`; control frame byte/value budgets may shorten
a page, with `more` and `next_after_sequence` exposing continuation. A selected
`job_id` requires `after_sequence: "0"` and `limit: "1"`. Records expose request
hash, operation, requested workers, output selection, cancellation flag and
state (`running`, `completed`, `failed`, `cancelled`, `recovery_required`).
They describe live observations, not an immutable cross-page snapshot. Records
remain in host memory for that incarnation; they are not durable job storage.
No total job count policy cap is imposed. Output selection is a requested
location, not proof that a complete artifact exists there.

`dataset_cancel` requests cooperative cancellation of one admitted job. Native
worker threads and provider cancellation contexts inherit it. Repeating the
request is safe; a terminal job keeps its outcome. A concurrent publication
may complete or require recovery. Cancellation does not roll back files or
claims. `failed_jobs` includes failed, cancelled and recovery-required jobs.
A provider that does not cooperate requires explicit whole-host stop.

`dataset_configure` uses `expected_generation` (initially `"0"`) to atomically
replace `max_concurrent_jobs`, `worker_budget`, and `idle_timeout_ms` without
reloading source bytes. Null concurrency/worker limits remain unbounded by
policy; idle timeout `"0"` disables expiry. `on_active_over_limit: "reject"`
refuses settings below active use, while `"drain"` admits them and preserves
running work; new admissions wait for capacity by explicit caller retry.
The returned generation increments exactly once. Source identity, memory mode
and already-admitted data are immutable for that incarnation; selecting a new
load creates a new incarnation. Completed research artifacts remain immutable;
users revise their inputs and produce new artifacts.

`dataset_release` still refuses active work. `dataset_stop` explicitly stops
that exact authenticated resident incarnation, including noncooperative native
providers. Its best-effort receipt says `termination_requested`, with active
counts and `results_may_be_incomplete: true`; it does not claim rollback,
completed publication or durable job recovery. Lost acknowledgements are
uncertain outcomes. The process exits without native destructor cleanup and
the OS reclaims its RAM. Inspect output destinations before further work.

Control connections are independent; a stalled frame does not block inspection,
configuration, cancellation or stop. There is no imposed connection/job deadline
or connection-count policy cap. Host memory/thread exhaustion remains possible
under caller-selected unbounded use. qxctl SIGINT/SIGTERM cancels its foreground
native process group; resident work has the explicit separate controls above.
All four operations, exact schemas/templates and complete JSON/NDJSON/text
results are available through qxctl and the same native SDK dispatch.

## Queue replay scenarios (0.26, R5.1)

`queue_scenarios` / `qxctl sbv queue-scenarios` consumes an exact admitted
`databento_mbo_orders_strict_v1` book result. It preserves the source census,
book frames and replay and produces independent hypothetical passive orders.
It does not change the strategy census or historically observed quantities.
Every request field is exposed in the installed schema/template and the same
native SDK operation. Inline or bundle-referenced orders, latency atoms and
execution labels are supported; legacy and partitioned output are independent
choices. Completed artifacts are immutable; changing assumptions creates a new
artifact with the original source identity retained.

Each order chooses side, integer limit price/quantity, horizon measured from
signal availability, explicit queue-ahead quantity or the displayed arrival
queue, and activation probability. Exact finite latency atoms carry unique IDs,
nonnegative delays and probabilities summing to one. There is no normalization,
random sampling or hidden seed. Alternative externally generated latency tapes
use this same interface. Existing native-provider and external-outcome contracts
remain alternate model-authoring routes; this built-in queue profile is not a
universal contract for those providers.

Arrival occurs after every retained event at the same `ts_recv`, and requires
an initialized complete frame at the last such source ordinal. Passive orders
that cross the arrival book are unavailable in this profile; the separately
selectable sweep model handles aggressive snapshot orders. A missing queue
level beyond retained depth is unavailable unless explicit queue-ahead quantity
is supplied. Contiguous replay from the signal through the horizon (including
the next retained event when present) is required. An initialized frame is still
an interpretation of its source, not proof of live-market completeness.

Exactly one service action (`T` or `F`) and an explicit same/opposite/any side
rule determine which exact-price events provide hypothetical service volume.
FIFO serves the queue ahead before the order. Pro-rata gives the order
`min(remaining, floor(service * remaining / (ahead + remaining)))`; the
remainder serves queue ahead up to its remaining size. Unused service is
retained. Fractional C advancement and A replenishment are floored per event.
These are user assumptions: a nonzero C advancement can overlap trade-related
cancellations; the engine does not infer a venue's priority or cancellation
attribution. Modifications are either explicitly ignored or make the scenario
unavailable. Reset either makes the scenario unavailable or cancels the
remaining hypothetical order while preserving prior modeled fills. Gap,
snapshot and unsupported flags produce unavailable evidence; receive-time
flags follow the selected policy.

The scenario table retains latency mass, arrival/horizon, frame coordinate,
first/last fill time, individual fills and queue/service conservation terms.
Each row is conditional on the selected order being active. The execution table
combines exact latency mass with the selected activation probability into
any/full/no-fill probabilities and expected filled quantity. Any nonzero
unavailable latency mass makes these summaries unavailable, without dropping
or renormalizing it. No-fill after a selected horizon or arrival after horizon
is distinct from missing evidence. Zero-weight rows remain inspectable.

`execution_labels` is optional. Caller-declared provenance, domain, permission,
training identities, partition, filled quantity and horizon accompany every
selected label table. An exact Brier score for any-fill is computed only for
selected held-out observations with matching domains/horizons, valid quantities,
available predictions and no declared training overlap (label/order/signal
identity). Training rows are indexed before scoring so row order cannot hide
reuse. Invalid evidence follows the caller's reject/unavailable choice. Synthetic
labels remain synthetic; external permission, authenticity and completeness of
training history remain declarations. Neither admission nor a Brier score
establishes learned calibration, independence or deployment applicability.

Orders run independently across selected CPU workers and merge in order/tape
order. No aggregate order/latency/dataset policy ceiling or default deadline is
added. Quantities/times use checked uint64 representation; probabilities and
moments use checked signed 128-bit exact rationals. Representation overflow
fails explicitly. The retained replay/frame/census indexes and order/latency
vectors reside in memory; each scenario's fill trace is materialized before
spooling. This is not a bounded-RSS or terabyte/cluster qualification claim.

This increment does not extend the snapshot-specific `allocation_economics`
consumer to queue distributions. Portfolio/shared-capital and queue economics
remain a named integration for the stateful/account lane; exported typed fills
and exact masses are available to external consumers immediately. M2/M3 remain
open under `M2-M3-READINESS.json`.


## Independent Go adapter (R5.2, source module v0.1.0-dev)

`libraries/sbv-go` is a separately consumable Go 1.26.5 standard-library module
that delegates the complete current 42-operation surface to an exact selected
qxctl/native 0.26 installation. Its copied streaming and row interfaces preserve
integer/rational strings, source identity, feature order and unavailable/null
evidence; completion requires source-bound full export and a matching logical
hash. Native result_select owns time/column filtering; external outcomes and
experiment requests retain native admission and provenance. Every mutation uses
the existing qxctl surface. The adapter owns no native installation or service.
Its [contract](../../libraries/sbv-go/IMPLEMENTATION.md) states memory, lifetime,
cancellation and trust boundaries. Native runtime/schema/ABI bytes are unchanged
by this adapter release. The previous external Python verification consumer
remains a historical isolated example, not a first-party dependency. Arrow,
DLPack, borrowed memory, vendor runtimes and adaptive search are not claimed.


## Nested temporal partitions (0.27)

`qxctl sbv split` adds `plan.kind: nested`, discoverable with the `nested`
and `nested_partitioned` template variants. `outer` is an explicit, rolling
or expanding temporal plan. `inner` contains exactly one selection for each
outer declared fold ID: `outer_fold_id`, a temporal `plan`, explicit `population`
and explicit `policies`. Empty inner plans and an empty outer plan are valid.
Combinatorial nesting and recursively nested plans are outside this named
profile; the existing combinatorial splitter remains independently available.

Outer folds use the request's chronology, purge, padding, embargo and
availability choices. Each inner selection supplies its own six policy fields
with the same exact semantics. `population: outer_train` restricts both inner
training and validation to **retained** parent training indices, after outer
exclusions. `population: source` explicitly permits source-wide exploration.
It may reuse parent holdouts or reintroduce excluded rows, and does not require
an approval or promotion gate. Inputs are not reweighted. Label/availability
pointers remain caller-selected; visual replay windows are never implicitly
used for purging. Unknown prior selection history remains unknown.

Results flatten each outer fold followed by its inner folds. Stable IDs are
`outer-I` and `outer-I/inner-J`; `windows.id` and `nesting.declared_id` preserve
caller names. Names must be unique within their sibling plan, and may repeat
across different outer folds. `nesting` records role, parent, population,
resolved policies, eligible row count, retained train/test rows outside the
parent training population, and retained train/test rows in the parent test
set. Outer-only counts that have no parent are null. Indices always refer to
the original selected source array, never to a renumbered inner population.
Empty folds remain unavailable with retained reasons.

Legacy and partitioned results expose every field through ordinary query/export
and SDK contracts. With `retain_rows`, existing `experiment` dependency bindings
can feed each fold into native `fit` and `predict`; the caller chooses the
graph, target, study list and purpose. This splitter does not choose a winning
model, refit it automatically, estimate overfitting confidence, or claim an
untouched holdout. Adaptive proposal journaling and automatic model-selection
orchestration remain R5.3 work.

The CPU implementation retains interval/index metadata, the scheduled folds
and one parent membership mask. It streams partitioned result arrays and
reuses that mask after each parent's inner folds. There is no policy maximum
on observations or fold counts and no default deadline. Host addressability,
exact uint64 timestamp arithmetic, and selected legacy/control framing still
apply. Explicit schema fold/row count annotations inherited from earlier
split releases are removed from the active split-plan and temporal-fold
contracts; they are not user resource policies.


## Adaptive revisions and automatic selection/refit (0.28)

`qxctl sbv experiment` template variant `adaptive` opts into append-only local
revisions. `revision.index` is a canonical uint64 decimal, beginning at `0`;
`previous_plan_sha256` is null at zero and the exact preceding plan content
hash thereafter. Each revision has its own `experiment-plan-N.json` and plan
publication recovery record. Fixed plans retain their existing filenames and
must use a different directory. A superseded revision cannot execute; observe
the latest plan through ordinary result inspection/query/export. An exact retry
of the current revision uses a fresh summary destination and reuses verified
receipts. A conflicting same-index plan, stale predecessor or noncontiguous
revision rejects before child work. The directory lock serializes publication
and execution; it is not distributed coordination.

Every revision supplies the cumulative trial table, inline or referenced.
Earlier rows must remain an exact prefix. Each inherited trial retains its
original admission plan hash in `admission_origins`; claims, resolved requests,
bindings, receipts and result identities are checked against that origin.
New trials have new unique IDs. Referenced tables are copied into a separate
caller-selected owned plan store for each revision; resumption does not require
the original external table. Worker count, failure policy, caller worker budget
and descriptive extensions can change in a new revision. Existing claims and
scientific requests cannot be rewritten. A failed or ambiguous trial is not
automatically retried; users can append a separately identified retry with
explicit lineage. Prior outcomes remain visible.

Optional cumulative `proposal_events` retain immutable `event_id`, `proposal_id`,
`state`, nullable `trial_id`, `reason`, `parameters` and `lineage`. A proposal
starts once as `proposed`, followed by at most one `admitted`, `rejected` or
`pruned` decision. Admitted events identify a unique ready trial. Rejected and
pruned decisions require a reason and no trial. Pending proposals may remain
pending across revisions. Trial states and claims remain the execution evidence;
a claim establishes admission/running intent, not current process liveness.
Proposal metadata is caller-declared, including any retrospective association.
`proposal_states` projects decisions; all events and rejected choices remain
in the durable plan. The engine does not infer an optimizer, approval authority,
search budget or statistical confidence from these records.

`qxctl sbv compare` template variant `selection` adds an optional explicit
`selection` policy. Choose an already selected `method` (`weighted_sum` or
`pareto`) and `tie_break`: `input_order`, lexicographic `candidate_id`, or
`require_unique`. The best set is weighted competition rank one or Pareto front
zero. Empty eligibility or a nonunique set under `require_unique` returns
`selection.status: unavailable` with the reason and best-set size, without
candidate parameters. A selected decision retains candidate ID, original input
index, parameters, lineage, source evidence and policy. Objective directions,
exact weights/scales and units remain caller-owned. No selection is performed
when this optional field is absent.

Experiment bindings optionally set `mode: value`; omitted mode remains
`reference`. Value mode copies the exact selected JSON into the existing null
request slot and records both an immutable source reference and the copied
value's SHA256 in the claim. It does not coerce rationals or integers. Missing
selections block the dependent before claiming it. Protocol, worker and output
controls cannot be rebound after admission. Per-child native validation still
applies. Copied values use the child control/claim representation limits; large
data should remain reference-bound.

A caller-authored DAG can therefore split nested populations, fit candidates on
inner training rows, predict inner validation rows, compare chosen objectives,
copy selected regularization or other parameters into an outer refit, then
predict the outer holdout automatically. The engine executes that entire graph
without an external winner-selection step. Users choose candidate generation,
fold aggregation, objectives, pruning, tie rules, holdout reuse and whether to
append another revision. No hidden fitting, promotion, holdout gate, default
deadline or trial quota is added. Scheduler metadata, proposal events and
admission-origin maps are resident; plan/claim control formats remain bounded.
Partitioned result/table arrays do not become an aggregate administrative frame.
This profile does not implement a built-in adaptive proposal algorithm, recursive
optimizer, arbitrary external-code dispatch or remote SOV execution.

Native 0.28 and optional Go transport 0.1.2 expose the same 42 operations and all
retained fields to qxctl text/JSON/NDJSON and future GUI/ML consumers. The C ABI
and portable result protocol remain unchanged; older installed engine/client
pairs and sealed evidence remain separate. R5.3c and the broad R5.4 M2 gate
remain open.


## Binary predictive-quality studies (0.29)

`qxctl sbv analyze` exposes `binary` and `binary_partitioned` template variants.
Select any subset of `binary_confusion`, `binary_probability`, `binary_ranking`,
combine them with other series/formula studies, or select no studies. Numeric
work remains native C++26. Existing result inspect/query/select/export surfaces
retain every field for terminal, GUI and external ML consumers. Native 0.29 and
optional Go transport 0.1.3 retain the same 42 operations and SDK ABI.

The optional `binary` object is required exactly when at least one binary study
is selected. Its fields are explicit: `label_pointer`, `label_type` (integer or
rational), distinct `positive_label`/`negative_label` rationals, `unknown_label`
(exclude/reject), `weight_normalization` (by_total/require_unit_mass), `thresholds`,
`bin_edges`, `tie_credit`, `retain_rows` and nonempty `label_description`. Existing
`value_pointer`/`value_type` select the score, and weighting/weight_pointer select
weights. Larger scores indicate the selected positive class. Label mapping and
meaning are caller declarations, not inferred execution truth. Missing labels
follow the existing missing policy. Unknown labels follow the explicit binary
policy; malformed numeric labels reject. Binary label exclusions do not remove
observations from separately selected nonbinary studies.

`binary_input` retains source indices, selected mapping, exclusions, positive
and negative weight totals and optional labeled rows. Input weights stay intact.
`by_total` explicitly divides metric sums by retained binary weight; require_unit_mass
requires that retained mass equal one. Signed-coefficient weighting is unavailable
(or rejects under on_incompatible), including when its current values happen to
be nonnegative. Empty or zero-weight populations are unavailable. Zero denominators
within otherwise available studies retain status/reason/null rather than a finite
substitute. Unknown prior selection history is explicit; no overfitting or calibrated
execution-likelihood claim follows from these descriptive statistics.

Confusion thresholds are a nonempty array of `{value, positive_when}` with ge or
gt equality. Each retains weighted TP/FP/TN/FN, precision, recall, specificity,
false-positive rate, accuracy, F1 and balanced accuracy. Bin edges must be strictly
increasing and cover exactly [0,1]. Bins are left closed/right open, except the
final upper endpoint is included. Probability studies retain binary Brier mean
squared error, its weighted sum, expected/maximum empirical bin gap, bin counts,
masses, means and observed positive fractions. Empty weighted bins stay visible
and unavailable. Every retained score must lie in [0,1], including zero-weight
rows; no clipping occurs. Out-of-range scores invalidate this probability study
only and remain usable for classification/ranking.

Ranking groups exactly equal scores, descends by score, skips zero-mass groups,
and retains the full ROC/precision-recall curve. ROC AUC is weighted pair
concordance with one-half tie credit. Separately named pairwise_concordance uses
the caller's tie_credit in [0,1]. Average precision is the sum of recall increment
times precision at each tied group, not trapezoidal PR area. One-class populations
retain undefined rates; AP requires positive-class mass, while ROC AUC requires
both classes. Thresholds/bin_edges must be empty when their study is unselected,
and tie_credit must be null without ranking. Inputs, thresholds and bins have no
additional policy count ceiling; selected control-frame formats still apply.

Exact checked signed-128-bit rational arithmetic rejects unrepresentable values
before result publication. No floating conversion is used for these studies even
when other selected studies request binary64 estimates. The per-study descriptor
records the exact profile. Typed observations and sorted groups use O(n) resident
memory, sorting O(n log n); bins and thresholds add their selected storage. Output
rows/curves stream into partitioned storage. Labels require an additional indexed
source pass. This is not out-of-core sorting, a cluster/RAM scale qualification or
a speed-superiority claim. User deadlines and host allocations remain authoritative.

Log loss, multiclass studies, calibration fitting, distribution shift, uncertainty
intervals and further catalogue proposals remain separate selectable extensions.
This increment qualifies the named binary subset only. R5.3 acceptance is reconciled
against original case wording in the accompanying packet; the broad M2 gate is R5.4.


## M2 measurement profile (R5.4)

The private `sbv-performance-tests` target measures the existing native run
profile; it is test infrastructure, not a new installed operation or SDK ABI.
Its JSON configuration chooses source requests, file/in-process RAM feeds,
legacy/partitioned storage, workers, warmups, repetitions, page/cache options,
output directories and a portable SBV result report. No observer is installed
by normal native-process, SDK or resident-worker entry points. Inactive probes
perform no clock sampling and do not change requests, calculations or results.

A stack-scoped per-thread recorder reports exclusive wall time for source
read/identity admission, DBN decode/validation/allocation, compute including
scratch I/O and envelope work, and final publication including validation,
hashing and durable output. Parallel compute is critical-path wall time, not
summed worker CPU time. Source-owner admission completion belongs to decode
validation; it is not represented as pure record conversion. The residual from
observed producer wall time remains separately visible. Full logical-result
materialization/hash verification is measured separately and included in the
complete producer-plus-reader total. RAM preload is reported separately, so
amortized totals must include it. Ordinary installed qxctl operations/export
supply independent process-level measurements and scientific parity evidence.

Benchmark reports preserve each sample, warmup status, scientific/result hashes,
resource disclosures, process cumulative peak RSS, selected parameters and host
context. Scientific equality compares complete summary/signals/execution/
distributions/studies/replay/census sections, with representation differences
qualified explicitly. Source-level observations and execution assumptions do not
become calibrated market claims. Actual host allocation, selected dataset/control
formats and user resource/deadline choices remain authoritative. Cache eviction,
NUMA, clusters, terabyte RAM and GPU execution require their own real evidence.

Graceful resident release sends its entire retained-source acknowledgement using
the caller-selected deadline (including none). It does not treat its own host
shutdown as cancellation of that response. Force-stop retains its nonblocking
best-effort acknowledgement because it must terminate even with a stalled reader.
A lost/partial lifecycle receipt still requires reconciliation, never automatic
retry. The R5.4 retained-source lifecycle regression covers the larger response.


## Exact supplied-event account simulation (0.30, R6.1a)

`qxctl sbv account-simulate` adds an exact C++26 cash-notional account model.
Every choice is in its installed schema and templates (`feedback`, `partitioned`),
with the same operation available through the C/C++ SDK and optional Go 0.1.4
transport. Native 0.30 exposes 43 operations; qxctl has 46 SBV leaves. All ledger,
signal, snapshot, input and branch evidence is portable through existing result
query/export or bundle query/export. This is the first R6.1 increment, not full
feedback-provider, exchange-matching or multiasset-economic qualification.

`cash_units` and `instruments` explicitly define account balances and instrument
cash units, positive exact price multipliers and `cash_notional` settlement.
Initial cash, positions and optional marks are exact signed-128 rational maps;
every declared cash/position key must appear. Prices, quantities, balances and
fees may be signed (negative fees are rebates); zero fill quantity is invalid.
The user supplies minimum/maximum constraints for each cash and position key;
null means no bound. Initial state must satisfy selected bounds. A user can
select no bounds to model debt or short positions. `on_constraint` selects
atomic event rejection or request abortion. No clipping, forced liquidation,
borrowing rate, implicit deadline or aggregate event-count limit is imposed.

Each named stream supplies chronological event rows inline or through a verified
bundle selection. Event IDs are unique within a stream. The stable merge order
is increasing time, then the caller's stream-array order, then source ordinal;
all tied events from an earlier stream precede a later stream's tied events.
Decreasing time or duplicate IDs reject. A referenced array preserves its exact
source and selected-value identity. Supplied timestamps describe the simulation;
they do not authenticate causal availability in an external market source.

A `fill_intent` declares instrument, signed quantity, price, fee and guards. Its
cash delta is `-(quantity * price * multiplier + fee)` and position delta is
quantity. Fees are in that instrument's cash unit. These supplied conditional
fills are not observed executions or calibrated likelihoods. `cashflow` declares
an exact signed amount in one unit. `fx` supplies positive debit and credit legs
in distinct units, without deriving a rate or adding fees. `mark` changes the
selected valuation price only; fills never silently update marks. Terminal equity
is cash plus marked positions, separately per unit; nonzero unmarked positions
make that unit's equity unavailable. No cross-currency sum is manufactured.

`fixed_intents` requires empty guards and preserves every supplied fill intent
in its own account-intent census, including constraint-rejected intents.
`stateful_feedback` evaluates all declared cash/position comparison guards against
the account state immediately before each intent. A failed guard suppresses that
signal and execution; a later cashflow or execution can change future membership.
An emitted but constraint-rejected intent remains in the census. Both modes create
new immutable account results and identify their mode and model; neither rewrites
or claims the original market fixed-census invariant. Account intent rows carry
`symphony.sbv.account-intent.v1` and source occurrences. Guard suppression, constraint
rejection and application are distinct ledger dispositions. No candidate output
or tentative balance is silently committed after a rejected event.

Every ledger row retains the exact input event, stream occurrence, before/after
cash and positions, applied deltas and constraint findings. `retain_snapshots`
selects initial/after-each-event state snapshots; terminal state is always retained.
An explicit initial-state reference can select a prior snapshot using a legacy
or bundle result. Its instrument/currency model digest must match; branches cannot
move earlier than its time watermark. This starts a new caller-selected branch,
not automatic resume/deduplication of its event tape. Equal-time branch events are
permitted and their new stream order is explicit. Users can provide a new inline
state/model to experiment with alternative definitions without changing the parent.

Account state transitions are serial because they share balances. Independent
experiments can run concurrently through `experiment`; this operation does not
pretend instrument partitioning preserves shared capital. Memory holds account
state, per-event candidate/before/after copies, one row per stream, stream metadata
and per-stream seen-ID sets. Partitioned output spools ledger/signals/snapshots;
legacy output retains them in memory and keeps its named format bounds. No
out-of-core identity set, cluster, CUDA or throughput superiority is claimed.
Arithmetic overflow rejects, never rounds; malformed events and cancellation do
not publish a complete financial result. Existing partitioned recovery preserves
scratch and typed uncertain-publication evidence.

R6.1b remains open for native execution-result coupling and competing market
liquidity, richer feedback providers and selected instrument economics. Futures
margin, option exercise, multileg atomicity and automatic corporate-action/FX
interpretation are not implied by cash-notional accounting. Explicit supplied
cashflows remain available for caller-authored scenarios with disclosed semantics.


## Account-coupled depth execution and target feedback (0.31, R6.1b)

`account_simulate` optionally accepts `liquidity_pools`. Each names a pool,
account instrument and exact legacy/bundle value selection of a retained
`symphony.sbv.book-frame.v1` reconstructed frame. This may be selected from book,
liquidity or another result retaining the exact frame. Original source/reference,
frame status, availability, depth and caller instrument mapping are retained.
Mapping is caller attribution, not authenticated instrument/exchange identity.
Two pool IDs cannot alias the same selected frame in the same outer result;
different source objects are not automatically reconciled into one global market.
Users assign competing orders to the same pool. Each pool starts once from its
selected displayed depth and never implicitly replenishes.

A `liquidity_order` executes the existing native `displayed_depth_sweep` v1 model
against the pool remaining after prior committed account events. Event time must
equal that snapshot's availability; this specific model is an ordered, single-frame
scenario, not a multi-time exchange simulator. Tied orders follow caller stream
order and source ordinal. The caller selects IOC/FOK, limit price in source nanos,
per-level participation, insufficient-depth/market-state/receive-flag policies,
fees, quantity rule, guards and unavailable behavior. The existing bounded retained
frame format and numeric representations remain named contracts, not dataset caps.
An order quantity is an exact signed integer with uint64 magnitude. This account
path does not impose the standalone legacy liquidity request's 1e9 quantity bound.
Buy is positive; sell is negative. Nanos convert to cash-unit price exactly by
1e9, then the instrument multiplier applies. Fees are `fee_fixed + abs(filled) *
fee_per_unit`, in the instrument cash unit, assessed only when quantity fills;
negative fees represent rebates. No-fill and FOK cancellation charge zero here.
Other fee conventions remain explicit supplied events or future model choices.

The tentative execution and cash/position transition are one atomic account event.
A capital/position rejection rolls back consumed depth as well as balances. Its
retained candidate outcome has `committed:false` and the account disposition;
candidate fills must not be counted as committed fills. FOK failure and guard
suppression consume no depth. `on_unavailable:abort` rejects the calculation;
`skip_event` retains an unavailable outcome, applies nothing, continues with the
known prior state and marks the result partial. Skipping unknown execution is a
caller-selected scenario assumption, not proof of a no-fill. Conditional sweep
probabilities remain assumptions, never calibrated execution likelihood.

`quantity_rule` chooses a signed `fixed` quantity, or `target_position` in
`stateful_feedback` mode. A target resolves to target minus prior simulated
position after earlier accepted events. An already-reached target emits no signal
and retains `target_reached`; failed guards retain `guard_false`. Fixed mode
cannot use target rules or nonempty feedback guards. Emitted signals keep their
own mode and input occurrence identity; original market censuses remain immutable.
The rule is a shipped declarative feedback model. Arbitrary runtime strategy or
ML-provider callbacks are not claimed by this release.

With pools, snapshots use `symphony.sbv.account-state.v2`, preserving exact
remaining bid/ask quantities per level. Pool-free snapshots remain v1. An explicit
branch uses the same instrument/pool model and restores remaining capacity,
including exhausted levels. It cannot silently start a fresh pool on resume.
Initial inline account state initializes fresh pools; choosing that is a new
scenario. Model mismatches, missing pool keys, excess capacity and inconsistent
depth reject. Branches are still explicit initial-state runs, not automatic event
deduplication. Remaining capacity, source frames, candidate execution, commitment,
resolved quantities, dispositions and all other values use existing qxctl
query/export and shared SDK result surfaces. `shared_liquidity`, `target_position`
and `position_split` templates expose the controls; references must be replaced
with caller-owned artifacts. Native 0.31 and optional Go 0.1.5 are exact selections.

`position_split` is an explicit positive ratio supplied at the chosen event time.
It multiplies the position and divides an existing mark, preserving exact marked
notional with no rounding, cash-in-lieu, tax or automatic data action. Caller bounds
apply atomically. A split on an instrument with a retained pool is unsupported in
this model because unchanged source units would be invalid; select a new branch
with explicitly adjusted pool evidence instead. Cash-notional settlement remains
the shipped instrument model; futures margin, option exercise and atomic multileg
settlement are not supplied by this operation.

Accounts remain serial; independent experiments may run concurrently. Pool state
is retained in memory and copied with tentative account state; snapshot retention
includes that depth state. No hidden spill, automatic parallelism across dependent
orders, global dataset cap, deadline, device runtime or speed claim is introduced.
This release verifies selected account/shared-snapshot execution. Parent R6.1 and
M3 remain open for richer provider/instrument selection and subsequent gates.


## Native account feedback and linear variation settlement (0.32, R6.1c)

The optional `feedback` selection on `account_simulate` requires
`mode:stateful_feedback` and `on_unavailable:abort|skip_trigger`. It uses the
existing digest-pinned native loader with role `feedback`, input profile
`symphony.sbv.account-feedback-call.v1` and `serialized_instance` concurrency.
The separately installed `symphony/sbv/feedback.h` exports
`symphony_sbv_feedback_api_v1`; the existing strategy/model ABI table is unchanged.
A provider implements descriptor, create, decide and destroy. Required capability
bits must be zero in v1. Descriptor, configuration, library/dependency hashes,
loading policy and reproducibility declarations are retained as provider evidence.
`provider_inspect` accepts this selection without creating an account instance.
Configuration schema is provider-declared and validated by that provider, not
interpreted as a host authorization policy.

A caller places `feedback_trigger` events in the ordinary ordered streams, with
an explicit `context` object. At that event the host passes only its occurrence,
time, caller context, current cash/positions/marks/basis and prior state time,
explicit memory, and a bounded summary of the preceding processed event. It does
not supply future stream rows, whole-run inputs or future depth frames. The
summary carries resolved event, account disposition, cash/position deltas and
execution outcome. Context/configuration and native code remain user authority:
this is a host information boundary, not a sandbox, audit of external I/O or proof
that a user-authored provider has no future information. Input numbers, returned
memory and extensions use the same portable exact-string result convention.

`feedback-decision.v1` returns an ordinary account `action` or null, explicit
object `memory` and object `extensions`. An action must keep trigger ID/time and
cannot recursively trigger feedback. Null retains `provider_no_action`. Successful
memory is retained even when a selected account bound rejects the proposed trade;
the following call receives that rejection. Unavailable callbacks abort unless
skip_trigger was selected; skipping changes no balances or memory, retains the
provider error and marks the result partial. Failed/malformed/cancelled callbacks
do not publish a completed account result. Borrowed inputs and cancellation tokens
last only for the synchronous callback; owned output/error buffers release once
before instance destruction and library unload. Cancellation/deadlines follow the
existing cooperative provider rules; native code is trusted, and the host cannot
preempt a noncooperating callback. No deadline is selected by default.

Continuation state belongs in returned memory. Each account invocation creates a
new provider instance; independent experiment trials create independent instances.
Hidden process state or nondeterministic/external dependencies cannot establish
portable branch equivalence. v3 snapshots preserve basis and, when selected,
feedback memory and the preceding event summary, together with remaining pool
capacity. Model hashes bind provider configuration and settlement selection.
Referenced branches require matching model/state coverage. Inline initial state
is a caller-authored scenario; optional memory/previous summary can seed it.
Stream source ordinals are local to each invocation, so a branch that reindexes
streams must account for that in a provider that uses ordinals. Event ID/time and
explicit returned state are available for stable continuation. Branching does
not automatically deduplicate replayed events.

`linear_variation` adds an exact linear settlement model per instrument, selected
alongside `cash_notional`. Inline state supplies a `basis` entry for every such
instrument. Opening/increasing a position charges fees, with no notional debit;
basis is the signed-quantity weighted average. Closing quantity realizes signed
closed quantity times (trade price minus basis) times the positive multiplier.
Partial closes retain basis; flattening clears it; reversals use the new trade
price for the residual position. Depth sweeps apply each fill in price order,
which matters when a sweep crosses flat, then assess the selected fill fees once.
A caller-timed `settle` event realizes position times (settlement price minus
basis) times multiplier, resets non-flat basis and updates the mark. Terminal
equity adds only unrealized variation to cash. Negative prices and rational bases
remain exact. `position_split` also divides basis; stale pools remain disallowed.
All basis, mark, cash, position and pool updates roll back together on financial
constraint rejection. Explicit successful provider memory does not roll back.

This is linear variation accounting, not exchange margin, liquidation, funding
schedules, inverse/nonlinear payoff, option exercise, tax or corporate-action
inference. Callers choose settlement events, bounds or no bounds, and can supply
other explicit cashflows/FX events. Unsupported settlement names fail admission.
The host imposes no dataset-size policy or hidden deadline. The serial account
path preserves event dependencies; independent accounts can run concurrently.

`native_feedback`, `linear_variation` and provider-inspect `feedback` templates
expose all new controls through qxctl. The entire resolved action, callback input,
decision/error, memory, financial state and provider evidence use existing terminal
query/export and portable SDK surfaces, including partitioned results. Native
0.32 and optional Go transport 0.1.6 are exact package selections. Retained market
signals used as triggers establish source lineage, not calibration of the fixture
strategy or truth of supplied execution. R6.1 reconciliation records this selected
local scope; M3 remote/scale/device/peer qualification remains separate.
