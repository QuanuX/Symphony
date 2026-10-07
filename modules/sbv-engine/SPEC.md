# SBV engine specification

## Version and ownership

Contract v1; package `sbv-engine` 0.19.0-dev; engine `symphony-sbv`; vector `sbv`. C++26 owns computations, result sealing and artifact queries. Go qxctl verifies the exact installation and projects native evidence. OWNER-INTERFACE.json is the exact operation declaration; generated C++/Go metadata must match it. This release is experimental.

The engine uses knowledge-vector-engine-cpp 0.2.0-dev and sqav-databento-dbn-cpp 0.5.0-dev, with that adapter's exact static dependencies. It performs no provider calls. Signal/replay semantics belong to SBV; acquisition remains SQAV, metadata SQMV, batch ownership SQFV, transformation SQTV, physical stores SQPV and recipient delivery SQDV. Its portable local JSON result is a derived artifact, not a new market-data persistence service. SQPV/SQDV integrations are not wired in this release.

## Administrative boundary

One engine-process.v2 request on stdin, one digest-bound response on stdout. Request 1 MiB, response 4 MiB, JSON depth 64, values 32768 and string 65536-byte limits remain unchanged. qxctl imposes no operation deadline. The user may select a timeout or absolute deadline; caller cancellation retains process-group cleanup. Artifact files have a separate 128 MiB / 4,000,000-value bound; large artifacts never travel in one administrative response. All native operation inputs reject unknown fields. All numeric experiment input/output values use canonical decimal strings or exact rationals; no floating JSON values.

`capabilities`, `run`, `evaluate`, `catalogue`, `compose`, `compose_joint`, `economics`, `book`, `liquidity`, `allocation_economics`, `result_select`, `backend_plan`, `live_plan`, `result_inspect`, `result_query` are implemented. Every operation has an exact `symphony.sbv.<hyphenated-operation>-input.v1` request and `symphony.sbv.<hyphenated-operation>.v1` response. Installed schemas/templates describe their shapes. qxctl also exposes schema, template and complete result export.

Run, evaluate, compose, compose_joint, economics, book, liquidity and allocation_economics create a single new file selected explicitly by the caller. They require target-host filesystem permission, not canonical-source apply permission. They do not update canonical knowledge, bind an identity or submit an order. The existing process contract's permission-backed-mutation classification describes this narrow output effect. These writes are non-idempotent: an existing path is refused even for identical data. Reads are idempotent at a supplied digest. If publication or the response outcome becomes uncertain, inspect the destination before retrying; never overwrite it automatically.

Paths must be absolute and component-safe. Reads and parent traversal reject symlinks. The parent must already exist. Output uses a mode-0600 exclusive staging file, fsync, atomic no-replace hard-link publication and parent fsync; cleanup removes only its staging name. Interrupted pre-publication work can leave a private `.sbv-*.tmp` file; it is not a committed result. No engine-generated path is executed.

## Historical run

One uncompressed DBNv1 or DBNv3 MBO file, one instrument, with no application-imposed source-byte, metadata-byte or event-count ceiling. Optional constraints are selected by the user (see Dataset resource authority). The caller supplies the exact source SHA256 and dataset; both are verified. Availability order is nondecreasing provider ts_recv with original source ordinal as tie-break. Unknown receive timestamps or mixed instruments are rejected; input is not silently sorted. Event timestamp, receive timestamp, sequence, flags and all other decoded MBO fields remain separate. The DBN undefined price sentinel INT64_MAX (9223372036854775807) is retained verbatim in replay but excluded from trade predicates. Provider timestamp sentinels remain original values; ts_recv must be known for this ordered mode.

The initial criteria catalogue contains spaced positive-size trades and direction of current trade relative to the preceding observed trade. The user selects minimum size, spacing, direction and a cap of 1–4096 signals. Zero spacing is admitted. Direction uses past/current observations only. Selection stops admitting signals at the cap; the reported additional eligible count uses the final admitted signal as spacing anchor. This count is diagnostic, not the count of a hypothetical uncapped strategy. The closed signal array has its own canonical SHA256 and cannot depend on later execution/replay choices.

Pass two operates on the immutable census. A worker pool of 1–64 std::jthreads writes disjoint preallocated output slots. Reductions retain census order. Worker errors are captured, joined and propagated; no partial result is published. Source bounds, window arithmetic, price arithmetic and deadlines are checked.

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

The caller supplies an ordered selection of zero to 4096 distinct signal IDs. Each binds a version-1 `linear_price_pnl` transform, an interpretation mode and explicit nonexecution treatment. The engine retains this selection separately from the original census; it never regenerates signals. Each transform declares a reference price, positive integer `price_unit_nanos`, signed rational `position_units`, signed rational `value_per_price_unit`, signed `cost_per_outcome`, nonzero signed `return_basis`, and attributable `pnl_unit` label. A negative cost can express a rebate; a negative position can express a short. Labels do not verify currency or instrument metadata.

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

`book` consumes an exact DBN source and the digest of a closed `run` or `evaluate` census artifact. Caller-selected signal IDs preserve their existing coordinates and never rerun signal criteria. The caller chooses independent replay durations, observed-event retention, frame cadence (`signals` or `signals_and_event_ends`), display depth 1–64, an explicit frame bound 1–4096, checkpoint emission, and anomaly policy. Exceeding the frame bound rejects; there is no implicit thinning. Empty selection is valid and emits no checkpoint. This exact release has ten native operations and thirteen qxctl leaves.

The `databento_mbo_orders_strict_v1` profile accepts one publisher, instrument and channel in one ordered source file. DBN decoding stays with the exact existing SQAV adapter. SBV interprets decoded actions using full order-ID state and ordered aggregate price levels. Add inserts; modify replaces the order; cancel subtracts its stated size and removes an exhausted order; reset clears all orders. Trades, fill reports and no-op records do not change quantities. Prices and quantities remain exact integers. Display depth limits exported frames, never the reconstructed state or full checkpoint. Locked and crossed observations are classified explicitly, without altering the supplied feed.

An initial fragment has **unknown state**, not an empty initialized book. A clean source reset or supplied compatible checkpoint is required. Snapshot records following a reset remain incomplete until an instrument event-end (`F_LAST`), including a later nonsnapshot event end when necessary. Frames at intermediate records are unavailable even if orders have been accumulated. Snapshot flags without an opening reset invalidate previously initialized state. Empty reset-plus-event-end is a valid initialized empty book.

`on_anomaly` is explicitly `reject` or `invalidate_until_reset`. Gaps (`F_MAYBE_BAD_BOOK`), top-of-book/MBP records, publisher-specific flags with unimplemented dataset semantics, unknown actions, duplicate adds, absent modified/canceled orders, invalid sides/prices/sizes and cancel mismatches follow this policy. Invalidation clears internal state and keeps later frames unavailable until a clean reset. An unknown initial state is expected missingness, independently of anomaly policy. Quantitative A/M/C interpretation is withheld while state is unknown. Reserved flag bit 1 is ignored according to the provider contract. Inaccurate receive timestamps remain flagged; ts_recv ordering is a selected reconstruction clock, not proof of exchange availability. No sequence-plus-one rule is applied to filtered instrument records. Unsupported feed semantics require a distinct admitted profile; this profile does not restrict user-authored alternatives.

Portable `book_frames` separate reconstructed levels from raw `replay.events`. Each frame carries source ordinal, timestamp, associated selected signal IDs, readiness, reason, event-boundary status, timestamp warning and initialization lineage. Windows retain their requested limits, observed-span flags and unproven completeness. Hydration cursor bounds are separate from visible windows. All retained frames and raw window events are self-contained for viewing. Mechanically rebuilding the full book from them still requires the referenced full source/checkpoint when hydration records were outside the retained windows. No complete-source, queue-priority or execution-likelihood claim follows from a ready frame.

Optional `book_checkpoint` contains every remaining order, exact source/dataset/publisher/instrument/channel binding, the next source ordinal and the preceding receive timestamp. Emission requires initialized state at a complete event boundary. Resume verifies the artifact content digest, profile, scope, ordered unique IDs, order values, cursor and source event boundary. The cursor must precede every requested window; a future checkpoint cannot hydrate an earlier signal or pre-window. Checkpoints only resume within the same exact source in this release. Integrity is verified, but supplied checkpoint authorship/state truth is not authenticated. Checkpoint provenance remains explicitly attributed.

One state worker processes the ordered hydration prefix; separate experiments may run concurrently. This does not alter existing parallel census/model evaluation. No live request, calibrated queue/fill model, source acquisition lifecycle, distributed scheduler or CUDA execution is added. `catalogue`, exact installed schema/template, terminal inspect/query and lossless export expose the entire new capability and payload. The portable result protocol remains v1; older artifacts remain readable through generic result operations.


## Displayed-depth execution scenarios (0.5)

`liquidity` consumes a content-digest-bound strict `book` result and 0–4096 explicit order intents. Each intent names a source signal and retained frame ordinal at or after that signal, buy/sell side, source quantity 1–1e9, optional limit price (null means market), IOC or FOK behavior, per-level participation in [0,1], and supplied or unavailable activation probability. The profile is `displayed_depth_sweep` version 1. Actual source/book integrity is preserved; artifact hashes do not authenticate authorship or establish market truth.

The selected frame is explicit retrospective evidence. Its receive-time delay from the signal is reported, but is not an inferred arrival book, a continuous latency simulation or a calibrated time-to-fill distribution. Unknown frames, unavailable books, insufficient retained depth and excluded locked/crossed or flagged-time frames follow the caller's `on_unavailable` selection. Malformed identities, prices, quantities, source digests, ordering and shapes reject independently. A ready frame must be at an event boundary, with positive ordered unique levels and consistent total/retained level counts. A frame earlier than its signal is rejected.

The model walks opposite-side displayed levels in best-price order and respects the supplied limit. At each level the per-order participation cap is `floor(original_displayed_quantity * max_level_participation)`; selected quantity is bounded by that cap, remaining order quantity and physically remaining modeled depth. No rounding up, hidden liquidity, replenishment, queue priority, venue routing or historical response to a hypothetical order is invented. Source quantities have no inferred contract multiplier or currency conversion. Signed prices are supported except the DBN undefined sentinel.

IOC takes the selected available quantity and cancels the remainder. FOK applies all fills only if the full requested quantity is supported; otherwise it emits zero fills and consumes no modeled depth. `depth_policy=require_sufficient` leaves an unresolved remaining quantity unavailable if omitted levels might satisfy it. An explicit limit boundary or zero participation can prove that omitted levels are irrelevant. `visible_only` instead authorizes a scenario restricted to retained levels, with that coverage label. This choice never establishes complete market depth.

`liquidity_mode=independent` evaluates each intent against an independent immutable frame copy using 1–64 workers and stable input-order output. Those outcomes do not share feasible aggregate liquidity. `shared_snapshot` requires every intent to name one exact frame; it executes sequentially in caller order and deducts successful conditional fills from that one shared snapshot. This scenario is conditioned on all supplied intents being active. Supplied activation probabilities must therefore be 1; unavailable probability declarations remain valid. Fractional activation requires stochastic joint branching, which this profile does not implement. An unavailable earlier shared intent leaves later dependent outcomes unavailable; it is not treated as zero consumption. Failed FOK orders are resolved zero-consumption outcomes. There is no cross-frame shared market evolution or shared-capital model.

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

`result_select` / `qxctl sbv result select --input ...` is a read-only native array view over an exact result digest. The user chooses the source array pointer (up to 65,536 rows), up to sixteen conjunctive predicates, eight ordered sort keys, sixty-four named column pointers, page size 1–256 and a continuation cursor. Columns omitted by selecting an empty list return full rows. Integer strings and reduced rational comparisons use exact signed 128-bit domains; rational ordering uses continued fractions, avoiding overflowing cross products. Text uses bytewise UTF-8 ordering, booleans false then true. No numerical type or missing-value meaning is inferred.

Predicates select eq/ne/lt/le/gt/ge and an explicit missing policy include/exclude/reject. Missing means absent pointer; present null/wrong-domain values reject. Every predicate value is validated even when another predicate excludes the row. Sort keys validate all matched rows, with explicit first/last/reject missing placement independent of direction. Ties preserve original source index. Projected missing fields carry unavailable status; full values retain all source fields. Each page discloses source count, matched count, offset, rows and completeness. Its cursor binds all selections, exact path/snapshot and page size. Response pages stop before 512,000 row bytes or 16,000 row values; an individually oversized row requires narrower projection and fails explicitly. Selection never rewrites the source or computes a financial aggregate. qxctl matches native Unicode canonicalization when checking query identities, including U+2028/U+2029 and literal backslash-u text.

`backend_plan` is disconnected planning, not resource reservation or execution. The caller supplies requested backend/workers/memory, declared limits, dependency-safe independent or serial-state scheduling, reject/reduce policy and optional explicit CPU fallback. CPU has a 64-worker implementation ceiling; serial-state planning grants at most one. CUDA/tensor runtime is unavailable. No automatic fallback or device copy occurs. Resolved fields are a proposed CPU plan when admitted; applied is always null and reservation false. Declared limits are caller assumptions, not SHV/SNV observation or grants. Existing calculations retain their separately reported actual scheduling; a plan does not silently install per-run memory enforcement.

An optional tensor descriptor declares dtype, device, storage bytes, byte offset, shape, nonnegative byte strides, owner/stream IDs and completion state. Rank is at most sixteen; dtype width, alignment, empty/scalar shapes, reachable storage extent, arithmetic overflow, selected memory limit, device match and ready completion are checked. Zero strides allow broadcast views. Negative strides and unknown dtypes are outside this profile. Metadata admission cannot prove foreign pointer validity, ownership or device completion. Pending, failed, cancelled, incompatible or oversized buffers remain unavailable for the proposed plan.

The receipt installs the independent header-only C++26 `symphony/sbv/interop.hpp` v1 contract and CMake target `Symphony::SbvInterop`. It needs no vendor or engine JSON headers. Buffer descriptors, backend requests/plans, shared lifetime leases and a completion interface expose ready/pending/failed/cancelled state, explicit monotonic-deadline wait and cancellation request. Host access requires a ready CPU lease. Adapters retain ownership and supply the completion implementation; the core does not fabricate asynchronous completion or initialize a GPU runtime. This is a source contract, not a stable cross-toolchain binary ABI. The full engine/model SDK remains distinct.

`live_plan` simulates an explicit ordered offline event fixture (up to 512 records/128 signals). Inputs include independent pre/post nanoseconds, event/byte/pending-window capacities, selected overload policies and a caller-declared closed-through watermark. Pre-trigger and pending windows share retained event indices. Expiry cannot remove events needed by a pending window; selected overflow may drop oldest/newest events or reject. Pending overflow may reject or skip the signal. Equal-time source indices remain distinct, windows include both time boundaries, and only windows closed by the watermark finish. Output exposes completed/loss/pending windows, retained/expected fixture counts, every dropped event and skipped signal, and actual simulated peaks. Fixture completeness is not market completeness. The planner has no subscription, socket, entitlement probe, spill implementation or activation path; `can_activate=false` and provider requests zero.

All three operations are read-only, preserve extensions where admitted, return bounded process-v1 payloads, and have installed qxctl schemas/templates. The release has eighteen SBV leaves and fifteen native operations. Runtime GPU integration, live capture, resource reservation and full SDK packaging remain separate work.

## Source-series studies and supplied-trial comparison — experimental 0.8

`analyze` binds one immutable result digest and an array pointer, with explicit value/weight pointers, series role, unit, ordering attribution, missing treatment and optional study selection. At most 65,536 observations and five study families are admitted. Equal probability is assigned over retained observations; supplied probability, scenario and signed-coefficient weights are preserved without normalization. Missing fields can be excluded or rejected; a present malformed number rejects. The canonical source is unchanged and retained signal, execution, replay and book sections remain accessible.

`series_summary` computes count, weight total, exact weighted sum and retained-observation extrema. `series_moments` requires total probability one; the user selects population variance or the equal-weight sample estimator. `series_quantiles` uses an inverse CDF on positive mass, q=0 minimum and first cumulative mass >= q. `equity_drawdown` treats caller-declared equity in retained source order, computes peak-relative and absolute drawdown and first recovery of the maximum absolute drawdown. Relative conventions are positive peaks only or absolute peak denominator; zero remains unavailable. No account feasibility, cashflow adjustment, elapsed duration or return sampling is inferred.

`return_ratios` requires caller-declared per-period returns, a constant benchmark, positive periods-per-year and explicit `binary64_roundtrip`. Mean differential return, selected variance and downside second moment remain exact. Standard deviation and square-root time scaling use IEEE 754 binary64 under nearest rounding; outputs retain max-digits round-trip decimal and 16-hex-digit bit identity. Sharpe is mean differential return divided by the selected standard deviation, times sqrt(periods). Sortino uses the square root of the second moment of negative benchmark differences over all probability mass. Zero denominators are unavailable. Scaling records the user's convention; it establishes neither serial independence nor predictive validity. Exact-rational-only selection leaves square-root ratios unavailable or rejects according to `on_incompatible`. Checked signed 128-bit intermediate overflow rejects before persistence. Selected operations may complete with unavailable study values; inspect per-study status.

`compare` binds up to 128 supplied trial records and 16 exact numeric objectives to source artifact digests. Completed artifacts are read and validated; failed/pruned records retain caller reasons, parameters and lineage. Objective pointers, units, directions, positive scales and signed weights are user inputs. Weighted utility is sum(direction sign * value / scale * weight), sorted descending with stable input ties and competition ranks. Pareto fronts use maximize/minimize directions on raw values and ignore weights, including zero/negative weights. Front zero is nondominated; ties do not dominate. Missing fields exclude whole candidates or reject; malformed present values always reject. Sources are bounded to 256 MiB aggregate reads. Empty method selection still records the ledger and extracted objective values. This is comparison of already supplied trials, not an executed optimizer, validated holdout protocol or significance test.

Both operations persist the same immutable result envelope, disclose choices/provenance/resources and support complete qxctl inspect/query/select/export and future GUI access. Current bounds are declared implementation limits, not restrictions on independently authored studies or optimizers. Source schema definitions include normalized series observations, study status rows and comparison candidates; study meanings are defined above and exposed by catalogue cards.

## Bootstrap and local trial execution — experimental 0.9

`resample` selects IID, moving-block or circular-block uniform row resampling from an exact immutable source array. Uniform sampling is explicit; source weights are never silently imported. Missing or malformed source values reject (select the source observations before calling). The user selects seed, sample length, block length, replica count, workers, optional source-index retention and zero or more studies. Bounds: 65,536 source rows/sample values, 1,024 replicas, 1,048,576 total sampled values, 65,536 retained indices, and 64 workers. IID requires block one; moving starts range from zero to N-block inclusive; circular starts cover all N rows and wrap. Blocks concatenate, with the last truncated to the sample length.

SplitMix64 has fixed unsigned 64-bit arithmetic. Replica r initializes a mixer at seed + r*0xd1342543de82ef95 modulo 2^64 and initializes its own generator with the mixer's next word. Rejection-modulo sampling discards words below (-N mod 2^64) mod N before reducing modulo N. Separate per-replica generator state and ordinal output are unaffected by worker scheduling; this is not a proof of statistical independence between streams. Means, population moments of replica means and empirical inverse-CDF quantiles use checked exact rationals. The optional index tape permits separate reconstruction. Empirical percentiles do not establish confidence coverage, stationarity, serial independence or predictive returns. External bootstrap protocols remain possible through source artifacts and extension data.

`experiment` runs up to 64 caller-supplied native SBV producer requests in an existing private local directory. This is local SBV artifact orchestration; no SOV remote/job authority, network endpoint, shell/plugin execution or broker operation is inferred. Ready trials name an implemented producer and omit output_path; the engine resolves `<id>.result.json` inside the selected directory. Pruned trials retain the caller's reason and null request/operation. Parameters and lineage remain supplied claims. IDs use ASCII letters/digits/underscore/hyphen, at most 80 characters. The engine supports at most 16 outer workers, limits selected outer*maximum child workers to 64 and requires one outer worker for deterministic ordered stop-on-failure. It applies those selected worker counts, not a host resource reservation.

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

For explicit/rolling/expanding plans: 65,536 source rows, 128 folds, 131,072 row-fold pairs, 64 MiB cumulative canonical embedded row bytes and the existing 128 MiB artifact/JSON-value limit. An explicit zero-fold plan and empty source array are supported. This implementation uses one CPU worker, interval-union sorting and binary search; independent split trials may use the existing experiment runner. Oversized retention rejects without silently switching to index-only output. The current release has twenty-seven native operations and thirty qxctl leaves.

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

This is still an in-memory result-artifact consumer: the existing 65,536-row
source profile, 64 MiB embedded-row retention profile, 128 MiB artifact and JSON
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
hidden retry/oversubscription. Run/evaluate retain their own 1–64 workers; book
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
64-trial/16-outer-worker/64-worker-product operational profile is unchanged.

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
